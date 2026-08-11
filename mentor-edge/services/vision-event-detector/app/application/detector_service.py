import uuid
import time
import logging
import os
import threading
import urllib.request
import json
import unicodedata
import hashlib
import base64
import struct
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, Optional
from ..domain.roi.roi_manager import ROI, ROIManager
from ..domain.signals.signal_extractors import EdgeSignal, HistogramSignal, FlowSignal, BeigeSignal
from ..domain.fusion.fusion_engine import FusionEngine, SignalValues, TEXTILE_WEIGHTS
from ..domain.fsm.textile_separator_fsm import EventFSM, FSMConfig
from ..domain.calibration.calibrator import Calibrator
from ..domain.calibration.fingerprint import build_calibration_fingerprint
from ..domain.calibration.model import (
    CalibrationResult,
    HISTOGRAM_ALGORITHM_PARAMETERS,
    HISTOGRAM_ALGORITHM_VERSION,
)
from ..domain.watchdog.watchdog import Watchdog
from ..domain.oee.oee_aggregator import OEEAggregator
from ..domain.presence.presence_detector import PresenceDetector
from ..domain.stop_tracker import StopTracker, StopAction
from ..adapters.gateway_stop_client import GatewayStopClient
from ..ports.frame_input import FrameInput
from ..ports.config_port import ConfigPort
from ..ports.event_output import EventOutput, EventOutputError
from ..ports.image_processor import ImageProcessor
from ..ports.calibration_repository import CalibrationRepository


# Orden canónico fijo del head para todos los OEE_SNAPSHOT,
# independiente del dispositivo o línea.
# El device_id / code en el payload siguen siendo dinámicos por config.
_OEE_HEAD_KEYS = [
    'T_DISPONIBLE',
    'T_MICROPARADA',
    'T_PARADA_NO_ASIGNADA',
    'T_PARADA_MAYOR',
    'CONTEO_1',
    'MARCA',
    'TAMANIO',
    'MATERIAL',
    'DESTINO',
    'T_REFRIGERIO',
    'T_CAPACITACION_OBLIGATORIA',
    'T_MANTENIMIENTO_PLANIFICADO',
    'T_PARADA_PROGRAMADA',
    'T_PARADA_NO_PROGRAMADA',
    'TIPO_PARADA_PROGRAMADA',
    'TIPO_PARADA_NO_PROGRAMADA',
    'MERMA',
]

# Claves cuyo valor por defecto es cadena vacía (no numérico)
_OEE_STRING_KEYS = frozenset({
    'MARCA', 'TAMANIO', 'MATERIAL', 'DESTINO',
    'TIPO_PARADA_PROGRAMADA', 'TIPO_PARADA_NO_PROGRAMADA',
})

_OEE_PRODUCT_ATTR_KEYS = frozenset({
    'MARCA', 'TAMANIO', 'MATERIAL', 'DESTINO',
})

# ── OEE backfill on startup ────────────────────────────────
# Persists last emission wall-clock time so the detector can fill
# gaps with zero-valued snapshots after a power cycle / crash.
_OEE_STATE_FILE = os.getenv(
    'OEE_STATE_FILE',
    '/var/lib/mentor/last_oee_emit',
)
_BACKFILL_MAX_WINDOWS = 96   # cap: 96 × 30 min = 48 h


def _normalize_col_key(nombre: str) -> str:
    """Convierte un nombre de columna legible al clave canónica del sistema.

    Ejemplos:
        'Marca'   → 'MARCA'
        'Tamaño'  → 'TAMANIO'   (ñ→n, luego alias conocido)
        'TAMAÑO'  → 'TAMANIO'
        'Material'→ 'MATERIAL'
    """
    upper = nombre.upper().strip()
    # Eliminar diacríticos (á→A, ñ→N, etc.)
    nfd = unicodedata.normalize('NFD', upper)
    ascii_form = ''.join(c for c in nfd if unicodedata.category(c) != 'Mn')
    # Alias: "TAMANO" (sin ñ de "Tamaño") → "TAMANIO" (convención del sistema)
    if ascii_form == 'TAMANO':
        return 'TAMANIO'
    return ascii_form


class _HealthHandler(BaseHTTPRequestHandler):
    detector_ref = None

    def do_GET(self):
        if self.path == '/health':
            self._send_health()
        elif self.path == '/calibrate':
            self._send_calibrate_status()
        elif self.path == '/status':
            self._send_status()
        elif self.path == '/frame':
            self._send_frame()
        elif self.path == '/stream':
            self._send_stream()
        elif self.path == '/ws/stream':
            self._handle_websocket()
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == '/calibrate':
            det = self.detector_ref
            if det:
                with det._lock:
                    det._start_calibration_locked('manual')
                self._json(200, '{"started":true}')
            else:
                self._json(503, '{"started":false}')
        elif self.path == '/reset-presence':
            # BeigeSignal es stateless: reset es no-op, respuesta OK siempre.
            self._json(200, '{"ok":true}')
        else:
            self.send_response(404)
            self.end_headers()

    def _send_health(self):
        det = self.detector_ref
        if det:
            with det._lock:
                wd_ok  = det.watchdog.check()
                cam_ok = det.frame_input.is_connected()
        else:
            wd_ok  = False
            cam_ok = False
        status = 'ok' if wd_ok else 'degraded'
        body = (
            '{"service":"vision-event-detector"'
            ',"status":"' + status + '"'
            ',"camera":' + ('true' if cam_ok else 'false') + '}'
        )
        self._json(200, body)

    def _send_status(self):
        det = self.detector_ref
        if det:
            with det._lock:
                detecting       = getattr(det, '_last_detecting', False)
                presence_motion = getattr(det, '_last_presence_motion', False)
                fusion_score    = getattr(det, '_last_fusion_score', 0.0)
                beige_ratio     = getattr(det, '_last_beige_ratio', 0.0)
                motion_score    = getattr(det, '_last_motion_score', 0.0)
                fsm_state       = det.fsm._state.value if hasattr(det.fsm, '_state') else 'idle'
                fsm_diagnostics = det.fsm.diagnostics
        else:
            detecting = False
            presence_motion = False
            fusion_score = 0.0
            beige_ratio  = 0.0
            motion_score = 0.0
            fsm_state = 'idle'
            fsm_diagnostics = {'state': 'idle'}
        import json as _json
        stop_tracker_state = 'producing'
        active_stop_id = None
        idle_duration_s = 0.0
        if det:
            with det._lock:
                stop_tracker_state = det._stop_tracker.state.value
                active_stop_id = det._stop_tracker.active_stop_id
                idle_duration_s = det._stop_tracker.idle_duration_s
        self._json(200, _json.dumps({
            'detecting': detecting,
            'presence_motion': presence_motion,
            'fusion_score': round(fusion_score, 4),
            'beige_ratio': round(beige_ratio, 4),
            'motion_score': round(motion_score, 4),
            'fsm_state': fsm_state,
            'fsm_diagnostics': fsm_diagnostics,
            'stop_tracker_state': stop_tracker_state,
            'active_stop_id': active_stop_id,
            'idle_duration_s': round(idle_duration_s, 1),
        }))

    def _send_calibrate_status(self):
        det = self.detector_ref
        if det:
            with det._lock:
                active   = det.calibrator.is_active
                progress = det.calibrator.progress
                samples_collected = det.calibrator.samples_collected
                samples_required = det.calibrator.required_samples
                state = det._calibration_state
                trigger = det._calibration_trigger
                quality = det._last_calibration_quality
                calibration_id = det._active_calibration_id
        else:
            active   = False
            progress = 0.0
            samples_collected = 0
            samples_required = 0
            state = 'unavailable'
            trigger = None
            quality = None
            calibration_id = None
        self._json(200, json.dumps({
            'active': active,
            'progress': round(progress, 4),
            'samples_collected': samples_collected,
            'samples_required': samples_required,
            'state': state,
            'trigger': trigger,
            'quality_score': quality,
            'calibration_id': calibration_id,
        }))

    def _send_stream(self):
        """MJPEG multipart stream del último frame procesado por el detector."""
        import time as _time
        self.send_response(200)
        self.send_header('Content-Type', 'multipart/x-mixed-replace; boundary=frame')
        self.send_header('Cache-Control', 'no-cache, no-store')
        self.send_header('Connection', 'keep-alive')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        det = self.detector_ref
        try:
            while True:
                jpeg = b''
                if det:
                    with det._lock:
                        jpeg = det._last_jpeg
                if jpeg:
                    header = (
                        b'--frame\r\n'
                        b'Content-Type: image/jpeg\r\n'
                        b'Content-Length: ' + str(len(jpeg)).encode() + b'\r\n\r\n'
                    )
                    self.wfile.write(header + jpeg + b'\r\n')
                    self.wfile.flush()
                _time.sleep(0.04)  # ~25 fps cap
        except (BrokenPipeError, ConnectionResetError, OSError):
            pass  # cliente desconectado

    def _send_frame(self):
        det = self.detector_ref
        jpeg = b''
        if det:
            with det._lock:
                jpeg = det._last_jpeg
        if not jpeg:
            self.send_response(503)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(b'{"error":"no frame available yet"}')
            return
        self.send_response(200)
        self.send_header('Content-Type', 'image/jpeg')
        self.send_header('Content-Length', str(len(jpeg)))
        self.send_header('Cache-Control', 'no-cache, no-store, must-revalidate')
        self.end_headers()
        self.wfile.write(jpeg)

    def _handle_websocket(self):
        """Upgrade HTTP → WebSocket (RFC 6455) y enviar frames JPEG binarios."""
        import time as _time
        import select as _select

        key = self.headers.get('Sec-WebSocket-Key', '').strip()
        if not key:
            self.send_response(400)
            self.end_headers()
            return

        # WebSocket handshake — RFC 6455 §4.2.2
        _WS_GUID = '258EAFA5-E914-47DA-95CA-C5AB0DC85B11'
        accept = base64.b64encode(
            hashlib.sha1((key + _WS_GUID).encode()).digest()
        ).decode()

        sock = self.request
        handshake = (
            b'HTTP/1.1 101 Switching Protocols\r\n'
            b'Upgrade: websocket\r\n'
            b'Connection: Upgrade\r\n'
            b'Sec-WebSocket-Accept: ' + accept.encode() + b'\r\n'
            b'\r\n'
        )
        sock.sendall(handshake)

        # Evitar que BaseHTTPRequestHandler intente leer más requests HTTP
        self.close_connection = True

        det = self.detector_ref
        sock.settimeout(None)
        last_jpeg = b''
        try:
            while True:
                # Verificar si el cliente envió algo (close frame, etc.)
                rlist, _, _ = _select.select([sock], [], [], 0)
                if rlist:
                    try:
                        data = sock.recv(1024)
                        if not data:
                            break
                        if len(data) >= 2 and (data[0] & 0x0F) == 0x8:
                            try:
                                sock.sendall(b'\x88\x00')
                            except OSError:
                                pass
                            break
                    except OSError:
                        break

                jpeg = b''
                if det:
                    with det._lock:
                        jpeg = det._last_jpeg
                # Solo enviar si hay un frame nuevo (evita re-enviar el mismo)
                if jpeg and jpeg is not last_jpeg:
                    self._ws_send_binary(sock, jpeg)
                    last_jpeg = jpeg
                _time.sleep(0.04)  # ~25 fps cap
        except (BrokenPipeError, ConnectionResetError, OSError):
            pass

    @staticmethod
    def _ws_send_binary(sock, data: bytes):
        """Enviar un frame WebSocket binario (opcode 0x02)."""
        length = len(data)
        header = bytearray()
        header.append(0x82)  # FIN + opcode binary
        if length < 126:
            header.append(length)
        elif length < 65536:
            header.append(126)
            header.extend(struct.pack('!H', length))
        else:
            header.append(127)
            header.extend(struct.pack('!Q', length))
        sock.sendall(bytes(header) + data)

    def _json(self, code: int, body: str) -> None:
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(body.encode())

    def log_message(self, format, *args):
        pass


class DetectorService:

    def __init__(
        self,
        frame_input: FrameInput,
        config_port: ConfigPort,
        event_output: EventOutput,
        image_processor: ImageProcessor,
        device_id: str,
        line_id: str = '1',
        line_code: str = 'MENTOR_DEFAULT_L1',
        oee_interval: float = 1800.0,
        micro_stop_max_s: float = 120.0,
        health_port: int = 8001,
        gateway_url: Optional[str] = None,
        frame_skip: int = 2,
        signal_scale: float = 1.0,
        calibration_repository: Optional[CalibrationRepository] = None,
        calibration_required_samples: int = 30,
        calibration_min_quality: float = 0.90,
        calibration_expiry_days: int = 90,
    ):
        self.frame_input = frame_input
        self.config_port = config_port
        self.event_output = event_output
        self.device_id = device_id
        self.line_id = str(line_id)
        self._line_code = line_code
        self._gateway_url = (gateway_url or '').rstrip('/')
        self._frame_skip = max(1, int(frame_skip))
        self._signal_scale = max(0.1, min(1.0, float(signal_scale)))
        self._frame_counter = 0

        self.roi_manager = ROIManager(ROI(120, 60, 320, 200))
        self._presence_roi_manager: Optional[ROIManager] = None
        # BeigeSignal dedicado al ROI de presencia: detecta el COLOR de la tela.
        # Si la cobertura beige >= threshold → tela presente. Sin latch, sin flujo.
        self._presence_beige_signal: Optional[BeigeSignal] = None
        # Presencia por movimiento temporal dentro del ROI (slow-window diff).
        # Compara frame actual vs frame de ~30 s atrás: detecta movimiento sub-pixel
        # de máquinas textiles donde 1 prenda ≅ 40 min (desplazamiento invisible frame-a-frame).
        self._presence_detector = PresenceDetector(
            pixel_threshold=8.0,   # intensidad de cambio mínima (0-255)
            motion_area=0.005,     # 0.5% del ROI debe cambiar para declarar movimiento
            exit_hold=750,         # ≥60 s sin movimiento para parar (≅ 750 frames @ 12.5 fps)
            scale=0.25,            # downscale 4x para reducir memoria del buffer
            window_frames=375,     # ventana de 30 s @ 12.5 fps
        )
        self._presence_beige_threshold: float = 0.05  # 5% del ROI cubierto de tela
        self.edge_signal = EdgeSignal()
        self.histogram_signal = HistogramSignal()
        self.flow_signal = FlowSignal()
        self.beige_signal = BeigeSignal()

        self._image_processor = image_processor
        for sig in [self.edge_signal, self.histogram_signal, self.flow_signal, self.beige_signal]:
            sig.set_image_processor(image_processor)

        self.fusion = FusionEngine()
        self.fsm = EventFSM(FSMConfig())
        self.calibrator = Calibrator(required_samples=calibration_required_samples)
        self._calibration_repository = calibration_repository
        self._calibration_min_quality = max(0.0, min(1.0, float(calibration_min_quality)))
        self._calibration_expiry_days = max(1, int(calibration_expiry_days))
        self._calibration_config_fingerprint = ''
        self._calibration_state = 'not_initialized'
        self._calibration_trigger: Optional[str] = None
        self._last_calibration_quality: Optional[float] = None
        self._active_calibration_id: Optional[int] = None
        self.watchdog = Watchdog()
        self.oee = OEEAggregator(
            snapshot_interval_s=oee_interval,
            micro_stop_max_s=micro_stop_max_s,
        )
        self._flow_production_threshold = 0.1

        self._stop_tracker = StopTracker(
            micro_stop_max_s=micro_stop_max_s,
            idle_debounce_ticks=3,
        )
        self._stop_client = GatewayStopClient(gateway_url or '', device_id)

        self._last_beige_ratio  = 0.0
        self._last_motion_score = 0.0
        self._last_jpeg: bytes = b''
        self._config_version = -1
        self._last_config_check = 0
        self._config_check_interval = 5.0
        self._health_port = health_port
        self._lock = threading.Lock()

        self._logger = logging.getLogger('detector')
        logging.basicConfig(level=logging.INFO, format='%(asctime)s %(name)s %(levelname)s %(message)s')

    def _start_health_server(self) -> None:
        _HealthHandler.detector_ref = self
        server = ThreadingHTTPServer(('0.0.0.0', self._health_port), _HealthHandler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

    def _log_compute_backend(self) -> None:
        import cv2
        from ..domain.signals.signal_extractors import _OFA_AVAILABLE
        cuda_devices = 0
        gst_support  = False
        try:
            if hasattr(cv2, 'cuda'):
                cuda_devices = cv2.cuda.getCudaEnabledDeviceCount()
            info = cv2.getBuildInformation()
            gst_support = 'GStreamer:                   YES' in info
        except Exception:
            pass
        self._logger.info(
            'Compute backend | OFA: %s | CUDA devices: %d | GStreamer: %s | frame_skip: %d | effective_fps: ~%.1f | signal_scale: %.2f',
            _OFA_AVAILABLE,
            cuda_devices,
            gst_support,
            self._frame_skip,
            25.0 / self._frame_skip,
            self._signal_scale,
        )

    # ── OEE state-file helpers ──────────────────────────────────

    def _start_calibration_locked(self, trigger: str) -> None:
        """Start sampling while preserving any currently valid reference."""
        self.calibrator.start()
        self.fsm.reset()
        self._calibration_state = 'calibrating'
        self._calibration_trigger = trigger
        self._logger.info(
            'HSV calibration started | trigger=%s samples=%d',
            trigger,
            self.calibrator.required_samples,
        )

    def _synchronize_calibration(self, config: Dict[str, Any]) -> None:
        """Load a compatible reference or start automatic calibration."""
        roi = self.roi_manager.roi
        roi_values = (roi.x, roi.y, roi.width, roi.height, roi.bottom_margin)
        camera_config = config.get('camera') or {}
        camera_url = camera_config.get('url') or getattr(self.frame_input, 'camera_url', '')
        fingerprint = build_calibration_fingerprint(
            roi=roi_values,
            camera_url=camera_url,
            camera_config=camera_config,
            signal_scale=self._signal_scale,
        )

        with self._lock:
            if (
                fingerprint == self._calibration_config_fingerprint
                and (self.histogram_signal.has_reference or self.calibrator.is_active)
            ):
                return

            previous_fingerprint = self._calibration_config_fingerprint
            self._calibration_config_fingerprint = fingerprint
            self.histogram_signal.reset()
            self.fsm.reset()
            self._active_calibration_id = None
            self._calibration_state = 'loading'

        stored = None
        if self._calibration_repository is not None:
            try:
                stored = self._calibration_repository.load_active(
                    line_id=self.line_id,
                    config_fingerprint=fingerprint,
                    algorithm_version=HISTOGRAM_ALGORITHM_VERSION,
                )
            except Exception:
                self._logger.exception(
                    'Could not load HSV calibration from PostgreSQL; calibrating in memory'
                )

        with self._lock:
            if fingerprint != self._calibration_config_fingerprint:
                return

            if stored is not None and stored.quality_score < self._calibration_min_quality:
                self._logger.warning(
                    'Stored HSV calibration %s rejected by current quality threshold '
                    '(%.4f < %.4f)',
                    stored.calibration_id,
                    stored.quality_score,
                    self._calibration_min_quality,
                )
                stored = None

            if stored is not None:
                try:
                    self.histogram_signal.set_reference_histogram(stored.histogram)
                except ValueError:
                    self._logger.exception(
                        'Stored HSV calibration %s is incompatible',
                        stored.calibration_id,
                    )
                else:
                    self._active_calibration_id = stored.calibration_id
                    self._last_calibration_quality = stored.quality_score
                    self._calibration_state = 'ready_persisted'
                    self._calibration_trigger = 'database'
                    self.fsm.reset()
                    self._logger.info(
                        'HSV calibration restored | id=%s quality=%.4f expires=%s',
                        stored.calibration_id,
                        stored.quality_score,
                        stored.expires_at.isoformat(),
                    )
                    return

            trigger = 'config_changed' if previous_fingerprint else 'startup_missing'
            self._start_calibration_locked(trigger)

    def _make_calibration_thumbnail(self, frame) -> Optional[bytes]:
        if frame is None:
            return None
        try:
            import cv2
            height, width = frame.shape[:2]
            scale = min(320 / max(1, width), 180 / max(1, height), 1.0)
            thumbnail = cv2.resize(
                frame,
                (max(1, int(width * scale)), max(1, int(height * scale))),
                interpolation=cv2.INTER_AREA,
            )
            ok, encoded = cv2.imencode(
                '.jpg',
                thumbnail,
                [cv2.IMWRITE_JPEG_QUALITY, 70],
            )
            if not ok:
                return None
            payload = encoded.tobytes()
            return payload if len(payload) <= 262144 else None
        except Exception:
            self._logger.debug('Could not create calibration thumbnail', exc_info=True)
            return None

    def _complete_calibration_locked(self, result: CalibrationResult) -> None:
        quality = float(result.quality_score)
        self._last_calibration_quality = quality

        if quality < self._calibration_min_quality:
            self._logger.warning(
                'HSV calibration rejected | quality=%.4f minimum=%.4f',
                quality,
                self._calibration_min_quality,
            )
            if self.histogram_signal.has_reference:
                self._calibration_state = 'rejected_using_previous'
                self._calibration_trigger = 'quality_rejected'
                self.fsm.reset()
            else:
                self._start_calibration_locked('quality_retry')
            return

        self.histogram_signal.set_reference_histogram(result.histogram)
        self.fsm.reset()
        self._active_calibration_id = None
        self._calibration_state = 'ready_memory'
        trigger = self._calibration_trigger or 'automatic'

        if self._calibration_repository is None:
            self._logger.warning(
                'HSV calibration accepted in memory but PostgreSQL is not configured'
            )
            return

        roi = self.roi_manager.roi
        try:
            calibration_id = self._calibration_repository.save(
                line_id=self.line_id,
                device_id=self.device_id,
                config_fingerprint=self._calibration_config_fingerprint,
                algorithm_version=HISTOGRAM_ALGORITHM_VERSION,
                algorithm_parameters=HISTOGRAM_ALGORITHM_PARAMETERS,
                histogram=result.histogram,
                samples_used=result.samples_used,
                quality_score=quality,
                expires_at=(
                    datetime.now(timezone.utc)
                    + timedelta(days=self._calibration_expiry_days)
                ),
                thumbnail_jpeg=self._make_calibration_thumbnail(result.reference_frame),
                metadata={
                    'trigger': trigger,
                    'line_code': self._line_code,
                    'roi': [roi.x, roi.y, roi.width, roi.height, roi.bottom_margin],
                    'signal_scale': self._signal_scale,
                },
            )
        except Exception:
            self._calibration_state = 'ready_memory_only'
            self._logger.exception(
                'HSV calibration accepted but could not be persisted to PostgreSQL'
            )
            return

        self._active_calibration_id = calibration_id
        self._calibration_state = 'ready_persisted'
        self._calibration_trigger = trigger
        self._logger.info(
            'HSV calibration persisted | id=%s quality=%.4f samples=%d',
            calibration_id,
            quality,
            result.samples_used,
        )

    def _save_last_oee_time(self) -> None:
        """Persist current UTC time as last OEE emission timestamp."""
        try:
            import os as _os
            tmp = _OEE_STATE_FILE + '.tmp'
            with open(tmp, 'w') as f:
                json.dump({'last_emit_utc': datetime.now(timezone.utc).isoformat()}, f)
            _os.replace(tmp, _OEE_STATE_FILE)   # atomic on POSIX
        except Exception as exc:
            self._logger.warning('Could not save OEE state file: %s', exc)

    def _load_last_oee_time(self) -> Optional[datetime]:
        """Load last OEE emission timestamp from state file."""
        try:
            with open(_OEE_STATE_FILE, 'r') as f:
                data = json.load(f)
            ts = data.get('last_emit_utc', '')
            if not ts:
                return None
            return datetime.fromisoformat(ts)
        except FileNotFoundError:
            return None
        except Exception as exc:
            self._logger.warning('Could not read OEE state file: %s', exc)
            return None

    # ── OEE backfill on startup ──────────────────────────────────

    def _backfill_missed_oee_windows(self) -> None:
        """Generate zero-filled OEE snapshots for time windows missed while
        the system was offline (power-off, crash, etc.).

        Reads the last emission time from the state file, computes how many
        snapshot-interval windows were missed, and emits a zero-valued
        OEE_SNAPSHOT for each one.  Product attributes (MARCA, MATERIAL…)
        are fetched once and applied to every backfilled record.
        """
        last_emit = self._load_last_oee_time()
        if last_emit is None:
            self._logger.info('No OEE state file — skipping backfill (first run?)')
            # Create the file now so next restart can backfill.
            self._save_last_oee_time()
            return

        now_utc = datetime.now(timezone.utc)
        interval_s = int(self.oee._snapshot_interval)
        if interval_s <= 0:
            interval_s = 1800

        gap_s = (now_utc - last_emit).total_seconds()
        if gap_s < interval_s * 1.5:
            self._logger.info(
                'OEE gap %.0fs < 1.5×interval %ds — no backfill needed',
                gap_s, interval_s,
            )
            return

        missed = int(gap_s / interval_s) - 1     # don't fill current window
        if missed <= 0:
            return
        missed = min(missed, _BACKFILL_MAX_WINDOWS)

        self._logger.info(
            'OEE backfill: last=%s  gap=%.1fh  windows=%d',
            last_emit.isoformat(), gap_s / 3600, missed,
        )

        # Fetch product attrs + turno once (best-effort).
        product_attrs = self._fetch_product_attrs()
        turno = self._fetch_current_turno()

        # Align to next interval boundary after last_emit.
        base_epoch = last_emit.timestamp()
        remainder = int(base_epoch) % interval_s
        next_boundary = base_epoch + (interval_s - remainder) if remainder else base_epoch + interval_s

        sent = 0
        for i in range(missed):
            window_epoch = next_boundary + i * interval_s
            window_dt = datetime.fromtimestamp(window_epoch, tz=timezone.utc)
            if window_dt >= now_utc:
                break

            values: Dict[str, Any] = {
                'T_DISPONIBLE': interval_s,
                'T_MICROPARADA': 0,
                'T_PARADA_NO_ASIGNADA': 0,
                'T_PARADA_MAYOR': 0,
                'CONTEO_1': 0,
                'MERMA': 0,
                'T_REFRIGERIO': 0,
                'T_CAPACITACION_OBLIGATORIA': 0,
                'T_MANTENIMIENTO_PLANIFICADO': 0,
                'T_PARADA_PROGRAMADA': 0,
                'T_PARADA_NO_PROGRAMADA': 0,
            }
            values.update(product_attrs)

            head = list(_OEE_HEAD_KEYS)
            data = [values.get(k, '' if k in _OEE_STRING_KEYS else 0) for k in head]

            event = {
                'event_id': str(uuid.uuid4()),
                'device_id': self.device_id,
                'event_type': 'OEE_SNAPSHOT',
                'timestamp': window_dt.isoformat(),
                'payload': {
                    'code': self._line_code,
                    'device_id': self.device_id,
                    'interval_s': interval_s,
                    'turno': turno,
                    'v': 1,
                    'head': head,
                    'data': data,
                },
            }
            try:
                if not self.event_output.send_event(event):
                    raise EventOutputError(
                        f"OEE backfill event {event['event_id']} was rejected"
                    )
                sent += 1
            except EventOutputError:
                raise
            except Exception as exc:
                self._logger.warning('Backfill window %d send failed: %s', i, exc)

        self._logger.info('OEE backfill complete: %d/%d windows sent', sent, missed)
        self._save_last_oee_time()

    def run(self) -> None:
        self._validate_system_clock()
        self._log_compute_backend()
        # Load configuration and persisted calibration before accepting frames.
        self._check_config_update(force=True)
        self._start_health_server()
        self._close_orphan_detector_stop()
        self._backfill_missed_oee_windows()
        null_streak = 0
        max_null_before_reconnect = 30

        while True:
            self._check_config_update()

            frame = self.frame_input.get_frame()
            if frame is not None:
                try:
                    import cv2 as _cv2
                    ok, buf = _cv2.imencode('.jpg', frame, [_cv2.IMWRITE_JPEG_QUALITY, 75])
                    if ok:
                        with self._lock:
                            self._last_jpeg = buf.tobytes()
                except Exception:
                    pass
            if frame is None:
                null_streak += 1
                if null_streak >= max_null_before_reconnect:
                    if not self.frame_input.reconnect():
                        time.sleep(1)
                    null_streak = 0
                else:
                    time.sleep(0.01)
                continue

            null_streak = 0
            self._frame_counter += 1
            if self._frame_counter % self._frame_skip != 0:
                continue

            try:
                with self._lock:
                    self.watchdog.update()

                    roi_frame = self.roi_manager.roi.extract(frame)
                    presence_source_frame = roi_frame

                    # Verificar que el ROI quede dentro del frame real de la cámara.
                    # Si la resolución de stream es menor que el ROI configurado,
                    # el recorte puede resultar < 16px (límite VPI OFA) → skip frame.
                    _rh, _rw = roi_frame.shape[:2]
                    if _rh < 16 or _rw < 16:
                        self.flow_signal.reset()
                        # Aun cuando el frame se descarta por ROI inválido,
                        # verificar si es momento de emitir el snapshot OEE
                        # (el timer es wall-clock; sin este check nunca emitiría).
                        if self.oee.should_emit():
                            self._emit_oee(self.oee.snapshot())
                        continue

                    # Reducir resolución del ROI para bajar CPU en los signals.
                    # Mínimo 16px en cada dimensión: VPI OFA exige height >= 16.
                    if self._signal_scale < 1.0:
                        import cv2 as _cv2
                        h, w = roi_frame.shape[:2]
                        new_w = max(16, int(w * self._signal_scale))
                        new_h = max(16, int(h * self._signal_scale))
                        roi_frame = _cv2.resize(
                            roi_frame,
                            (new_w, new_h),
                            interpolation=_cv2.INTER_LINEAR,
                        )

                    if self.calibrator.is_active:
                        self.calibrator.add_sample(roi_frame)
                        if self.calibrator.progress >= 1.0:
                            result = self.calibrator.finish(self.histogram_signal)
                            if result is not None:
                                self._complete_calibration_locked(result)
                        continue

                    signals = SignalValues(
                        edge=self.edge_signal.compute(frame, roi_frame),
                        color=self.histogram_signal.compute(frame, roi_frame),
                        flow=self.flow_signal.compute(frame, roi_frame),
                        beige=self.beige_signal.compute(frame, roi_frame)
                    )

                    # ── Presencia: BeigeSignal en el ROI de conteo ──
                    # Si hay color de tela (beige/crema) en el ROI → presente.
                    # Un único ROI para todo: conteo + presencia. Sin latch, sin flujo.
                    # Si hay ROI de presencia separado configurado, lo usa en su lugar.
                    if self._presence_roi_manager is not None and self._presence_beige_signal is not None:
                        pframe = self._presence_roi_manager.roi.extract(frame)
                        if pframe.size == 0:
                            coverage = 0.0
                            motion_present = False
                            motion_score = 0.0
                            motion_sample_fresh = False
                        else:
                            raw_coverage = self._presence_beige_signal.compute(frame, pframe)
                            # Normalizar: misma sensibilidad absoluta que el ROI de conteo.
                            # Si el ROI de presencia es 10x más grande, la misma tela cubre
                            # solo 1/10 de la fracción → amplificar para igualar la sensibilidad.
                            c_roi = self.roi_manager.roi
                            c_area = max(1, c_roi.width * (c_roi.height - c_roi.bottom_margin))
                            p_h, p_w = pframe.shape[:2]
                            coverage = min(1.0, raw_coverage * (p_h * p_w) / c_area)
                            motion_present = self._presence_detector.update(pframe)
                            motion_score = float(self._presence_detector.presence_score)
                            motion_sample_fresh = True
                    else:
                        coverage = signals.beige  # ROI de conteo como presencia
                        motion_present = self._presence_detector.update(presence_source_frame)
                        motion_score = float(self._presence_detector.presence_score)
                        motion_sample_fresh = True

                    # Producción activa: cuando el buffer slow-window está lleno, usar SOLO
                    # movimiento como señal (evita que tela estática en el ROI mantenga
                    # is_producing=True indefinidamente). Durante calentamiento (~30s iniciales)
                    # usar beige OR movimiento para no generar falsos idle_wait al arrancar.
                    if self._presence_detector.is_warmed_up:
                        is_producing = motion_present      # con latch → para OEE (estable)
                        # Para stop_tracker: motion_present OR fsm en WAIT_EXIT.
                        # - WAIT_EXIT: la FSM sabe que hay una pieza activa → produciendo siempre.
                        #   Si la tela va muy lenta, motion_present puede caer en 60s aunque
                        #   la máquina esté produciendo → falsa PARADA_NO_ASIGNADA.
                        # - Para paros reales durante WAIT_EXIT: max_wait_exit_frames=10000
                        #   (~13 min) actúa de timeout de seguridad → CORTE+IDLE → stop abre.
                        # - Fuera de WAIT_EXIT (IDLE/DETECTING/COOLDOWN): motion_present es la
                        #   señal estable (exit_hold=750 ≈ 60s) sin oscilaciones de motion_now.
                        fsm_tracking = (self.fsm._state.value == 'en_prenda')
                        is_producing_tracker = motion_present or fsm_tracking
                    else:
                        is_producing = (coverage >= self._presence_beige_threshold) or motion_present
                        # During warmup (~30 s) the slow-window buffer is empty, so
                        # motion_now = False always.  Feed True to stop_tracker to
                        # avoid creating a false MICROPARADA on every container restart.
                        is_producing_tracker = True
                    self._last_presence_motion = is_producing_tracker
                    self.oee.tick(is_producing)
                    stop_actions = self._stop_tracker.update(is_producing_tracker)
                    for action in stop_actions:
                        self._handle_stop_action(action)

                    self._last_beige_ratio  = round(float(signals.beige), 4)
                    self._last_motion_score = round(motion_score, 4)

                    fusion_score = self.fusion.fuse(signals)
                    self._last_fusion_score = fusion_score
                    self._last_detecting    = self.fsm._state.value != 'idle'

                    event_type = self.fsm.process(
                        fusion_score,
                        signals.beige,
                        motion_score,
                        fast_activity=(
                            signals.flow >= self.fusion.flow_gate
                            or fusion_score >= self.fsm.config.garment_score_threshold
                        ),
                        slow_activity=(
                            motion_sample_fresh
                            and self._presence_detector.is_warmed_up
                            and self._presence_detector.motion_now
                        ),
                    )
                    if event_type == 'CALIBRATE':
                        self._calibrate_beige_from_roi(roi_frame)
                    elif event_type:
                        self._emit_event(event_type, signals, fusion_score)
                        if event_type == 'CORTE':
                            self.oee.record_cut()

                    if self.oee.should_emit():
                        self._emit_oee(self.oee.snapshot())
            except EventOutputError:
                self._logger.critical(
                    'Durable event output rejected an event; stopping detector',
                    exc_info=True,
                )
                raise
            except Exception:
                self._logger.exception('Error processing frame')

            # El chequeo de emisión de OEE también se hace fuera del bloque try
            # para que se emita aunque los signals de frame fallen (p.ej. fallo VPI).
            # La segunda llamada es segura: should_emit() no resetea el timer;
            # snapshot() solo lee estado acumulado sin side-effects.
            if self.oee.should_emit():
                self._emit_oee(self.oee.snapshot())
    
    def _calibrate_beige_from_roi(self, roi_frame) -> None:
        """Aprende el rango HSV del beige real a partir del primer separador detectado."""
        import cv2
        import numpy as np
        try:
            hsv = cv2.cvtColor(roi_frame, cv2.COLOR_BGR2HSV)
            # Máscara con rango actual (amplio) para extraer solo píxeles beige
            mask = cv2.inRange(hsv, self._image_processor._beige_lo, self._image_processor._beige_hi)
            beige_pixels = hsv[mask > 0]
            if len(beige_pixels) < 50:
                return
            h_mean = int(np.mean(beige_pixels[:, 0]))
            s_mean = int(np.mean(beige_pixels[:, 1]))
            v_mean = int(np.mean(beige_pixels[:, 2]))
            # Rango ajustado ±10 en H, ±40 en S, V mínimo holgado
            self._image_processor.update_beige_range(
                h_min=max(0,   h_mean - 10),
                h_max=min(179, h_mean + 10),
                s_min=max(0,   s_mean - 40),
                s_max=min(255, s_mean + 40),
                v_min=max(0,   v_mean - 40),
            )
            self._logger.info(
                'Beige calibrado: H=%d±10 S=%d±40 V_min=%d', h_mean, s_mean, max(0, v_mean - 40)
            )
        except Exception:
            self._logger.warning('Fallo calibración beige — se mantiene rango por defecto')

    def _emit_event(self, event_type: str, signals: SignalValues, confidence: float) -> None:
        cycle = self.fsm.last_event_metadata
        event_confidence = max(
            float(confidence),
            float(cycle.get('max_fusion', 0.0)),
            float(cycle.get('max_motion', 0.0)),
        )
        event = {
            'event_id': str(uuid.uuid4()),
            'device_id': self.device_id,
            'event_type': event_type,
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'payload': {
                'line_code': self._line_code,
                'confidence': event_confidence,
                'roi_id': 'default',
                'fsm_state': self.fsm.state.value,
                'cycle': cycle,
                'signals': {
                    'edge': float(signals.edge),
                    'color': float(signals.color),
                    'flow': float(signals.flow),
                    'beige': float(signals.beige)
                }
            }
        }
        
        if not self.event_output.send_event(event):
            raise EventOutputError(
                f"Detector event {event['event_id']} was rejected"
            )

    def _emit_oee(self, oee_data: dict) -> None:
        # Acumular valores en dict para evitar duplicados al hacer merge.
        # Todos los valores de tiempo están en SEGUNDOS (unidad canónica del sistema).
        values: Dict[str, Any] = dict(zip(oee_data['head'], oee_data['data']))

        # Enriquecer con tiempos del edge-gateway (en segundos):
        #   • T_DISPONIBLE  → calculado desde sync_turnos (tiempo programado del turno)
        #   • T_PARADA_*    → duraciones de paradas justificadas y terminadas
        interval_s = int(self.oee._snapshot_interval)
        stop_durations = self._fetch_stop_durations(interval_s)
        for clave, seg in stop_durations.items():
            values[clave] = values.get(clave, 0) + seg

        # El aggregator de visión acumula TODO el tiempo idle de máquina en sus
        # tres buckets (T_MICROPARADA / T_PARADA_NO_ASIGNADA / T_PARADA_MAYOR),
        # independientemente de si ese tiempo será luego clasificado por la tablet.
        # El gateway SUMA encima los tiempos de paradas justificadas específicas
        # (T_REFRIGERIO, T_MANTENIMIENTO_PLANIFICADO, etc.), provocando doble
        # conteo: un mismo intervalo de idle aparece tanto en el bucket del
        # aggregator como en la clave de la parada clasificada.
        # Solución: deducir del aggregator exactamente los segundos que el
        # gateway ya contabilizó como paradas específicas.
        _GATEWAY_SPECIFIC_KEYS = frozenset({
            'T_REFRIGERIO', 'T_CAPACITACION_OBLIGATORIA',
            'T_MANTENIMIENTO_PLANIFICADO',
            'T_PARADA_PROGRAMADA', 'T_PARADA_NO_PROGRAMADA',
        })
        for key in _GATEWAY_SPECIFIC_KEYS:
            classified = stop_durations.get(key, 0)
            if classified <= 0:
                continue
            # Deducir de los buckets del aggregator en orden de mayor a menor.
            # Las paradas largas (> stop_max_s continuo) van a T_PARADA_MAYOR;
            # las medianas van a T_PARADA_NO_ASIGNADA; las cortas a T_MICROPARADA.
            for bucket in ('T_PARADA_MAYOR', 'T_PARADA_NO_ASIGNADA', 'T_MICROPARADA'):
                current = values.get(bucket, 0)
                deduct = min(current, classified)
                if deduct > 0:
                    values[bucket] = current - deduct
                    classified -= deduct
                if classified <= 0:
                    break

        # Enriquecer con MERMA actual de sync_variables (la tablet lo escribe)
        values['MERMA'] = self._fetch_merma()

        # Enriquecer con atributos textiles del producto activo.
        product_attrs = self._fetch_product_attrs()
        values.update(product_attrs)

        # Obtener el turno activo del gateway
        turno_nombre = self._fetch_current_turno()

        # Fallback: si el gateway no respondió y T_DISPONIBLE es 0,
        # usar el intervalo nominal para no reportar disponibilidad cero.
        if values.get('T_DISPONIBLE', 0) == 0:
            values['T_DISPONIBLE'] = interval_s

        # Emitir siempre en el orden canónico fijo _OEE_HEAD_KEYS.
        # Claves string usan '' como default; numéricas usan 0.
        head = list(_OEE_HEAD_KEYS)
        data = [values.get(k, '' if k in _OEE_STRING_KEYS else 0) for k in head]

        event = {
            'event_id': str(uuid.uuid4()),
            'device_id': self.device_id,
            'event_type': 'OEE_SNAPSHOT',
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'payload': {
                'code': self._line_code,       # dinámico: viene de config por línea
                'device_id': self.device_id,   # dinámico: viene de config por dispositivo
                'interval_s': interval_s,
                'turno': turno_nombre,
                'v': 1,
                'head': head,
                'data': data
            }
        }
        if not self.event_output.send_event(event):
            raise EventOutputError(
                f"OEE event {event['event_id']} was rejected"
            )
        self._save_last_oee_time()

    def _fetch_stop_durations(self, interval_s: int) -> Dict[str, int]:
        """Query edge-gateway for OEE time fields in **seconds**.

        Returns a dict like::

            {
                'T_DISPONIBLE':                   60,
                'T_PARADA_PROGRAMADA':            45,
                'T_PARADA_NO_PROGRAMADA':        120,
                'T_REFRIGERIO':                  900,
                'T_CAPACITACION_OBLIGATORIA':      0,
                'T_MANTENIMIENTO_PLANIFICADO':    60,
            }

        T_DISPONIBLE es calculado por el gateway a partir de sync_turnos.
        Returns an empty dict on error so the snapshot is still emitted.
        """
        if not self._gateway_url:
            return {}
        try:
            url = f'{self._gateway_url}/edge/stops/duration-by-type?interval_s={interval_s}'
            with urllib.request.urlopen(url, timeout=3) as resp:
                return json.loads(resp.read().decode())
        except Exception as exc:
            self._logger.warning('Could not fetch stop durations from gateway: %s', exc)
            return {}

    def _fetch_current_turno(self) -> str:
        """Query edge-gateway for the active turno name at the current time.

        Returns the nombre of the active sync_turnos entry, or '' if none is
        active or the gateway is unavailable.
        """
        if not self._gateway_url:
            return ''
        try:
            url = f'{self._gateway_url}/edge/current-turno'
            with urllib.request.urlopen(url, timeout=3) as resp:
                data = json.loads(resp.read().decode())
                return data.get('nombre', '') or ''
        except Exception as exc:
            self._logger.warning('Could not fetch current turno from gateway: %s', exc)
            return ''

    def _fetch_product_attrs(self) -> Dict[str, str]:
        """Query edge-gateway for the characteristics of the currently active product.

        Flow:
          1. GET /edge/production-runs?limit=200  → busca el run activo (ended_at is null)
          2. GET /edge/catalogs/product-characteristics → columns + values per product
          3. Match run by producto_id OR sku (codigo) → return {clave_canónica: valor}

        Returns empty dict on error or if no product is assigned.
        """
        if not self._gateway_url:
            return {}
        try:
            # 1. Active production run: el API ordena ASC, necesitamos el más
            # reciente SIN ended_at. Descargamos los últimos 200 y buscamos.
            url = f'{self._gateway_url}/edge/production-runs?limit=200'
            with urllib.request.urlopen(url, timeout=3) as resp:
                runs = json.loads(resp.read().decode())
            if not runs:
                return {}
            # Tomar el run activo más reciente (sin ended_at), último en lista ASC
            active = None
            for run in reversed(runs):
                if not run.get('ended_at'):
                    active = run
                    break
            if active is None:
                return {}
            producto_id = active.get('producto_id')
            run_sku = active.get('sku') or ''

            # Neither producto_id nor sku → no product assigned
            if not producto_id and not run_sku:
                return {}

            # 2. Product characteristics
            url2 = f'{self._gateway_url}/edge/catalogs/product-characteristics'
            with urllib.request.urlopen(url2, timeout=3) as resp:
                data = json.loads(resp.read().decode())

            # columnas: [{variable_id, nombre (=nombre_col), orden}]
            # Normalize nombre_col to canonical key (e.g. "Tamaño"→"TAMANIO")
            columnas: Dict[int, str] = {
                int(c['variable_id']): _normalize_col_key(c['nombre'])
                for c in data.get('columnas', [])
            }
            if not columnas:
                return {}

            # 3. Find values for the active product — match by producto_id OR sku/codigo
            for prod in data.get('productos', []):
                pid = int(prod.get('producto_id', -1))
                codigo = str(prod.get('codigo', ''))
                id_match = producto_id and pid == int(producto_id)
                sku_match = run_sku and codigo == run_sku
                if id_match or sku_match:
                    result: Dict[str, str] = {}
                    for var_id, clave in columnas.items():
                        # JSON encodes dict keys as strings
                        val = (prod['valores'].get(var_id)
                               or prod['valores'].get(str(var_id), ''))
                        if clave in _OEE_PRODUCT_ATTR_KEYS:
                            result[clave] = str(val) if val else ''
                    return result
            return {}
        except Exception as exc:
            self._logger.warning('Could not fetch product attributes from gateway: %s', exc)
            return {}

    def _fetch_merma(self) -> int:
        """Query edge-gateway for the current MERMA value stored in sync_variables.

        The tablet writes the scrap count to PATCH /edge/variables/MERMA.
        Returns integer units; defaults to 0 on error or if not yet set.
        """
        if not self._gateway_url:
            return 0
        try:
            url = f'{self._gateway_url}/edge/variables?clave=MERMA'
            with urllib.request.urlopen(url, timeout=3) as resp:
                records = json.loads(resp.read().decode())
                if records:
                    return int(records[0].get('valor') or 0)
                return 0
        except Exception as exc:
            self._logger.warning('Could not fetch MERMA from gateway: %s', exc)
            return 0

    # ── Automatic stop management ──────────────────────────────

    def _handle_stop_action(self, action: StopAction) -> None:
        if action.kind == "ABRIR_PARADA":
            stop_id = self._stop_client.create_stop(
                stop_type="PARADA_NO_ASIGNADA",
                started_at=action.started_at,
            )
            self._stop_tracker.set_open_stop_id(stop_id)
        elif action.kind == "CERRAR_PARADA":
            self._stop_client.close_stop(action.stop_id)
        elif action.kind == "CREAR_MICROPARADA":
            self._stop_client.create_stop(
                stop_type="MICROPARADA",
                started_at=action.started_at,
                ended_at=action.ended_at,
            )

    def _close_orphan_detector_stop(self) -> None:
        """On startup, close any detector-created stop left open from a
        previous session (crash, restart, etc.)."""
        open_id = self._stop_client.find_open_detector_stop()
        if open_id:
            self._stop_client.close_stop(open_id)
            self._logger.warning('Closed orphan detector stop %s on startup', open_id)

    def _check_config_update(self, force: bool = False) -> None:
        current_time = time.time()
        if not force and current_time - self._last_config_check < self._config_check_interval:
            return
        
        self._last_config_check = current_time
        
        version = self.config_port.get_config_version()
        if force or version != self._config_version:
            self._apply_config(self.config_port.get_config())
            self._config_version = version
    
    def _validate_system_clock(self) -> None:
        now = datetime.now(timezone.utc)
        boot_year = now.year
        if boot_year < 2024:
            self._logger.critical(
                'System clock appears incorrect (year=%d). '
                'Detector startup is blocked until NTP/systemd-timesyncd is active.',
                boot_year
            )
            raise RuntimeError(
                f'System clock is not valid for event timestamps: {now.isoformat()}'
            )
        else:
            self._logger.info('System clock OK: %s', now.isoformat())

    def _apply_config(self, config: Dict[str, Any]) -> None:
        with self._lock:
            self._apply_config_locked(config)
        self._synchronize_calibration(config)

    def _apply_config_locked(self, config: Dict[str, Any]) -> None:
        roi_data = config.get('roi', [])
        if len(roi_data) >= 4:
            bm = int(roi_data[4]) if len(roi_data) >= 5 else 0
            new_roi = ROI(int(roi_data[0]), int(roi_data[1]), int(roi_data[2]), int(roi_data[3]), bm)
            old_roi = self.roi_manager.roi
            if (new_roi.width, new_roi.height) != (old_roi.width, old_roi.height):
                # Reiniciar buffers VPI/OpenCV del flow_signal para evitar
                # error VPI_ERROR_INVALID_ARGUMENT por dimensiones obsoletas.
                self.flow_signal.reset()
                self._presence_detector.reset()
            self.roi_manager.update(new_roi)

        rp_data = config.get('roi_presencia')
        if rp_data:
            # roi_presencia puede llegar como dict {x,y,width,height} desde el Go model
            # o como lista [x,y,w,h] si se envió manualmente.
            if isinstance(rp_data, dict):
                rpx = rp_data.get('x', 0)
                rpy = rp_data.get('y', 0)
                rpw = rp_data.get('width', 0)
                rph = rp_data.get('height', 0)
            elif isinstance(rp_data, (list, tuple)) and len(rp_data) >= 4:
                rpx, rpy, rpw, rph = int(rp_data[0]), int(rp_data[1]), int(rp_data[2]), int(rp_data[3])
            else:
                rp_data = None

        if rp_data and rpw > 0 and rph > 0:
            new_roi = ROI(int(rpx), int(rpy), int(rpw), int(rph))
            if self._presence_roi_manager is None:
                self._presence_roi_manager = ROIManager(new_roi)
                # BeigeSignal para presencia: comparte processor con self.beige_signal
                self._presence_beige_signal = BeigeSignal(threshold=self._presence_beige_threshold)
                self._presence_beige_signal._processor = getattr(self.beige_signal, '_processor', None)
            else:
                self._presence_roi_manager.update(new_roi)
            self._presence_detector.reset()
        else:
            self._presence_roi_manager = None
            self._presence_beige_signal = None
            self._presence_detector.reset()

        # Restablecer el perfil textil antes de aplicar un override opcional.
        self.fusion.update_weights(TEXTILE_WEIGHTS)

        thresholds = config.get('thresholds', {})
        self.edge_signal.threshold        = thresholds.get('edge',  0.4)
        self.histogram_signal.threshold   = thresholds.get('color', 0.6)
        self.flow_signal.dy_threshold     = thresholds.get('dy',    5.0)
        self.beige_signal.threshold       = thresholds.get('beige', 0.35)
        # thresholds.flow actúa como gate de fusión: si flow < flow_gate → score=0
        self.fusion.flow_gate             = thresholds.get('flow',  0.5)

        # Pesos de fusión custom (override de los del modo si se envían).
        custom_weights = config.get('weights')
        if custom_weights and isinstance(custom_weights, dict):
            try:
                self.fusion.update_weights(custom_weights)
            except ValueError:
                pass  # pesos inválidos → se quedan los del modo

        # presencia_threshold: fracción del ROI cubierta de tela (beige) para declarar presente.
        # Rango 0.01–0.5 (1%–50%). Default: 0.05 (5%).
        # Valores > 1 son del formato antiguo (pixel-diff 1-80) → convertir dividiendo entre 100.
        pt = thresholds.get('presencia_threshold')
        if pt is not None:
            raw = float(pt)
            if raw > 1.0:
                raw = raw / 100.0  # convertir pixel-diff legacy a fracción
            self._presence_beige_threshold = max(0.01, min(0.9, raw))
            if self._presence_beige_signal:
                self._presence_beige_signal.threshold = self._presence_beige_threshold

        # Sensibilidad de micromovimiento del ROI de presencia.
        ps = thresholds.get('presencia_scale')
        if ps is not None:
            self._presence_detector.motion_area = float(ps)
        ph = thresholds.get('presencia_hold')
        if ph is not None:
            self._presence_detector.exit_hold = int(ph)
        ppw = thresholds.get('presencia_window')
        if ppw is not None:
            self._presence_detector.window_frames = int(ppw)
        ppt = thresholds.get('presencia_pixel_threshold')
        if ppt is not None:
            self._presence_detector.pixel_threshold = float(ppt)

        if thresholds.get('presencia_reset'):
            self._presence_detector.reset()

        # Rango HSV de beige configurable desde el Lab
        br = thresholds.get('beige_range')
        if br and isinstance(br, dict):
            proc = getattr(self.beige_signal, '_processor', None)
            if proc and hasattr(proc, 'update_beige_range'):
                proc.update_beige_range(
                    br.get('h_min', 15), br.get('h_max', 35),
                    br.get('s_min', 30), br.get('s_max', 130),
                    br.get('v_min', 120),
                )

        fsm_config = config.get('fsm', {})
        # Build and validate atomically: an invalid remote update cannot leave
        # a partially-mutated FSM running. Legacy values are accepted but
        # clamped to the safe textile pilot envelope.
        effective_fsm = FSMConfig(
            calibration_frames=max(
                1,
                int(fsm_config.get('calibration_frames', 20)),
            ),
            beige_high=float(fsm_config.get('beige_high', 0.55)),
            beige_low=float(fsm_config.get('beige_low', 0.30)),
            beige_confirm_frames=max(
                10,
                int(fsm_config.get(
                    'beige_confirm_frames',
                    fsm_config.get('n_frames', 10),
                )),
            ),
            beige_confirm_s=max(
                2.0,
                float(fsm_config.get('beige_confirm_s', 2.0)),
            ),
            beige_exit_frames=max(
                8,
                int(fsm_config.get(
                    'beige_exit_frames',
                    fsm_config.get('exit_frames', 8),
                )),
            ),
            garment_evidence_frames=max(
                3,
                int(fsm_config.get('garment_evidence_frames', 3)),
            ),
            activity_window_s=max(
                1.0,
                float(fsm_config.get('activity_window_s', 5.0)),
            ),
            garment_score_threshold=max(
                0.0,
                float(fsm_config.get('garment_score_threshold', 0.20)),
            ),
            garment_motion_threshold=max(
                0.0,
                float(fsm_config.get('garment_motion_threshold', 0.02)),
            ),
            slow_motion_guard_frames=max(
                1,
                int(fsm_config.get(
                    'slow_motion_guard_frames',
                    self._presence_detector.window_frames,
                )),
            ),
            min_garment_s=max(
                30.0,
                float(fsm_config.get('min_garment_s', 30.0)),
            ),
            rearm_beige_frames=max(
                25,
                int(fsm_config.get(
                    'rearm_beige_frames',
                    fsm_config.get('cooldown', 25),
                )),
            ),
            rearm_beige_s=max(
                2.0,
                float(fsm_config.get('rearm_beige_s', 2.0)),
            ),
            min_rearm_s=max(
                300.0,
                float(fsm_config.get('min_rearm_s', 300.0)),
            ),
        )
        effective_fsm.validate()
        self.fsm.config = effective_fsm
        # A config change in the middle of a candidate is fail-closed.
        self.fsm.reset()

        camera = config.get('camera')
        if camera and isinstance(camera, dict):
            new_url = camera.get('url', '')
            if new_url and new_url != self.frame_input.camera_url:
                self.frame_input.update_url(new_url)

        oee_cfg = config.get('oee', {})
        line_name = oee_cfg.get('line_name', '')
        if line_name:
            self._line_code = line_name
        micro_s = oee_cfg.get('micro_stop_max_s')
        snap_s = oee_cfg.get('snapshot_interval_s')
        stop_s = oee_cfg.get('stop_max_s')
        if micro_s is not None or snap_s is not None or stop_s is not None:
            self.oee.update_config(
                micro_stop_max_s=micro_s if micro_s is not None else self.oee._micro_stop_max_s,
                snapshot_interval_s=snap_s if snap_s is not None else self.oee._snapshot_interval,
                stop_max_s=stop_s,
            )
        if micro_s is not None:
            self._stop_tracker.update_threshold(micro_s)
