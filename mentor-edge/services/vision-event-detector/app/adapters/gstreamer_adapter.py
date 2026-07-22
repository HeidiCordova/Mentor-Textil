"""GStreamerAdapter: captura RTSP con NVDEC GPU en Jetson Orin."""

import logging
import threading
import time
from typing import Optional

import numpy as np

logger = logging.getLogger('detector.camera.gst')

_GST_AVAILABLE = False
try:
    import gi
    gi.require_version('Gst', '1.0')
    from gi.repository import Gst
    Gst.init(None)
    _GST_AVAILABLE = True
except Exception as _gst_err:
    logger.warning('[GStreamer] PyGObject no disponible (%s)', _gst_err)

from ..ports.frame_input import FrameInput


class GStreamerAdapter(FrameInput):

    PLAY_TIMEOUT_S       = 12
    MAX_RECONNECT_DELAY  = 30.0
    INIT_RECONNECT_DELAY = 1.0

    def __init__(self, camera_url: str, reconnect_attempts: int = 10,
                 capture_every: int = 1):
        """
        capture_every: copiar solo 1 de cada N frames del callback de GStreamer.
        Debe coincidir con frame_skip del detector para evitar memcpy innecesaria.
        Ejemplo: capture_every=2 -> 12.5fps copiados en vez de 25fps = -50% memcpy.
        """
        if not _GST_AVAILABLE:
            raise RuntimeError('GStreamer/PyGObject no disponible en este entorno')

        self.camera_url        = camera_url
        self.reconnect_attempts = reconnect_attempts
        self._capture_every    = max(1, int(capture_every))
        self._capture_counter  = 0

        self._frame:  Optional[np.ndarray] = None
        self._lock    = threading.Lock()
        self._pipeline = None
        self._connected = False
        self._consecutive_failures = 0

        if not self._connect():
            raise RuntimeError(
                f'[GStreamer] No se pudo iniciar el pipeline para: {camera_url}'
            )

    def _pipeline_str(self, url: str) -> str:
        return (
            f'rtspsrc location="{url}" latency=100 protocols=udp '
            f'retry=5 timeout=5000000 do-retransmission=false '
            f'! rtph264depay '
            f'! h264parse '
            f'! nvv4l2decoder enable-max-performance=true '
            f'! nvvidconv '
            f'! video/x-raw,format=BGRx '
            f'! videoconvert '
            f'! video/x-raw,format=BGR '
            f'! appsink name=sink emit-signals=true '
            f'max-buffers=1 drop=true sync=false'
        )

    # ------------------------------------------------------------------
    # Callbacks del pipeline
    # ------------------------------------------------------------------

    def _on_new_sample(self, sink) -> int:
        """Callback desde el thread de streaming de GStreamer."""
        # Saltar frames a nivel de captura para evitar memcpy innecesaria
        self._capture_counter += 1
        if self._capture_counter % self._capture_every != 0:
            sink.emit('pull-sample')  # descartar el sample para liberar el buffer
            return Gst.FlowReturn.OK

        sample = sink.emit('pull-sample')
        if sample is None:
            return Gst.FlowReturn.ERROR  # noqa

        buf  = sample.get_buffer()
        caps = sample.get_caps()
        st   = caps.get_structure(0)
        w    = st.get_value('width')
        h    = st.get_value('height')

        ok, mi = buf.map(Gst.MapFlags.READ)  # noqa
        if ok:
            try:
                frame = np.frombuffer(mi.data, dtype=np.uint8).reshape(h, w, 3).copy()
                with self._lock:
                    self._frame = frame
                    self._consecutive_failures = 0
            except Exception as exc:
                logger.warning('[GStreamer] frame reshape error: %s', exc)
            finally:
                buf.unmap(mi)

        return Gst.FlowReturn.OK

    def _poll_bus(self) -> None:
        if self._pipeline is None:
            return
        bus = self._pipeline.get_bus()
        while True:
            msg = bus.pop()
            if msg is None:
                break
            if msg.type == Gst.MessageType.ERROR:
                err, dbg = msg.parse_error()
                logger.error('[GStreamer] Bus ERROR: %s | %s', err.message, dbg)
                with self._lock:
                    self._connected = False
            elif msg.type == Gst.MessageType.EOS:
                logger.warning('[GStreamer] EOS recibido')
                with self._lock:
                    self._connected = False

    def _cleanup(self) -> None:
        if self._pipeline is not None:
            try:
                self._pipeline.set_state(Gst.State.NULL)
            except Exception:
                pass
            self._pipeline = None
        self._connected = False

    def _connect(self) -> bool:
        self._cleanup()
        try:
            pstr = self._pipeline_str(self.camera_url)
            logger.debug('[GStreamer] Pipeline: %s', pstr)
            pipe = Gst.parse_launch(pstr)

            sink = pipe.get_by_name('sink')
            sink.connect('new-sample', self._on_new_sample)

            pipe.set_state(Gst.State.PLAYING)

            deadline = time.monotonic() + self.PLAY_TIMEOUT_S
            while time.monotonic() < deadline:
                ret, state, _ = pipe.get_state(100 * Gst.MSECOND)
                if (ret == Gst.StateChangeReturn.SUCCESS
                        and state == Gst.State.PLAYING):
                    self._pipeline = pipe
                    self._connected = True
                    logger.info(
                        '[GStreamer] NVDEC pipeline PLAYING: %s', self.camera_url
                    )
                    return True
                if ret == Gst.StateChangeReturn.FAILURE:
                    raise RuntimeError('Pipeline falló al iniciar (StateChangeReturn.FAILURE)')
                time.sleep(0.1)

            raise RuntimeError(f'Pipeline no alcanzó PLAYING en {self.PLAY_TIMEOUT_S}s')

        except Exception as exc:
            logger.error('[GStreamer] _connect falló: %s', exc)
            self._cleanup()
            return False

    def get_frame(self) -> Optional[np.ndarray]:
        self._poll_bus()
        if not self._connected:
            return None
        with self._lock:
            frame = self._frame
            self._frame = None
        if frame is None:
            self._consecutive_failures += 1
        return frame

    def is_connected(self) -> bool:
        self._poll_bus()
        return self._connected

    def reconnect(self) -> bool:
        delay = self.INIT_RECONNECT_DELAY
        for attempt in range(1, self.reconnect_attempts + 1):
            logger.info(
                '[GStreamer] Reconexión %d/%d (espera %.1fs)',
                attempt, self.reconnect_attempts, delay
            )
            time.sleep(delay)
            if self._connect():
                return True
            delay = min(delay * 2.0, self.MAX_RECONNECT_DELAY)
        logger.error(
            '[GStreamer] Todos los intentos de reconexión agotados: %s',
            self.camera_url
        )
        return False

    def update_url(self, url: str) -> bool:
        self.camera_url = url
        return self._connect()

    def __del__(self):
        self._cleanup()
