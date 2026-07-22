import time
import uuid
import logging
import threading
from datetime import datetime, timezone
from typing import Dict, Any, Optional

from ..ports.frame_input import FrameInput
from ..ports.config_port import ConfigPort
from ..ports.event_output import EventOutput
from ..domain.yolo_engine import YOLOEngine
from ..domain.roi_manager import ROIManager
from ..domain.line_counter import LineCounter
from ..domain.count_aggregator import CountAggregator

logger = logging.getLogger('yolo.service')

CONFIG_POLL_INTERVAL = 5.0
RECONNECT_WAIT = 3.0
# Pausa entre frames descartados por frame_skip (cede CPU sin perder conteo)
_SKIP_SLEEP = 0.004  # ~250 Hz max en skip, evita busy-loop


class CounterService:

    def __init__(
        self,
        frame_input: FrameInput,
        config_port: ConfigPort,
        event_output: EventOutput,
        device_id: str,
        line_code: str,
        oee_interval: float,
        frame_skip: int,
        yolo_config: Dict[str, Any],
    ):
        self._frame_input = frame_input
        self._config_port = config_port
        self._event_output = event_output
        self._device_id = device_id
        self._line_code = line_code
        self._frame_skip = max(frame_skip, 1)
        self._config_version = 0
        self._running = False
        self._frame_count = 0

        self._roi: Optional[ROIManager] = None
        self._yolo = YOLOEngine(
            model_name=yolo_config.get('model_name', 'yolo26s'),
            confidence=yolo_config.get('confidence', 0.45),
            classes=yolo_config.get('classes'),
            class_names=yolo_config.get('class_names'),
            use_tensorrt=yolo_config.get('use_tensorrt', True),
            imgsz=yolo_config.get('imgsz', 640),
        )
        self._counter = LineCounter(
            line_y_ratio=yolo_config.get('counting_line_y', 0.5),
            line_x_ratio=yolo_config.get('counting_line_x', 0.5),
            direction=yolo_config.get('counting_direction', 'top_to_bottom'),
        )
        # Mapeo color_tapa -> marca: {"rojo": "LOA", "celeste": "CIELO", ...}
        self._cap_colors: dict = yolo_config.get('cap_colors', {})
        self._aggregator = CountAggregator(
            interval_s=oee_interval,
            device_id=device_id,
            line_code=line_code,
        )

        self._lab_frame = None
        self._lab_detections = None
        self._lab_lock = threading.Lock()
        # Contador de clientes activos del stream MJPEG.
        # El lab frame solo se copia cuando hay al menos un viewer activo,
        # evitando la copia innecesaria de cada frame en producción.
        self._lab_viewer_count = 0

    def run(self):
        self._running = True
        last_config_check = 0.0
        logger.info('counter service started (model=%s, skip=%d)', self._yolo.model_name, self._frame_skip)

        while self._running:
            now = time.monotonic()
            if now - last_config_check > CONFIG_POLL_INTERVAL:
                self._check_config()
                last_config_check = now

            if not self._frame_input.is_connected():
                logger.warning('camera disconnected, reconnecting...')
                time.sleep(RECONNECT_WAIT)
                self._frame_input.reconnect()
                continue

            frame = self._frame_input.get_frame()
            if frame is None:
                time.sleep(0.01)
                continue

            self._frame_count += 1
            should_detect = (self._frame_count % self._frame_skip == 0)
            
            roi_frame = self._roi.extract(frame) if self._roi else frame
            h, w = roi_frame.shape[:2]
            
            if should_detect:
                # Ejecutar detección YOLO solo cada N frames
                detections = self._yolo.detect_to_sv(roi_frame)
            else:
                # En frames intermedios, usar detecciones vacías pero mantener el tracker activo
                # Esto permite que ByteTrack mantenga la continuidad de los IDs
                import supervision as sv
                detections = sv.Detections.empty()
                # Pequeña pausa para ceder CPU sin perder tracking
                time.sleep(_SKIP_SLEEP)
            
            # SIEMPRE actualizar el counter (incluso con detecciones vacías)
            # Esto mantiene el tracker activo y evita pérdida de IDs
            delta = self._counter.update(detections, h, w)

            # Solo copiar el frame al buffer lab si hay un cliente activo
            if self._lab_viewer_count > 0 and should_detect:
                with self._lab_lock:
                    self._lab_frame = roi_frame.copy()
                    self._lab_detections = detections

            if delta > 0:
                self._aggregator.add(delta)
                self._emit_count_event(delta, detections)

            oee_record = self._aggregator.emit()
            if oee_record:
                self._event_output.send_event(oee_record)
                logger.info('OEE snapshot: count=%d', oee_record['data'][3])

    def stop(self):
        self._running = False

    def get_lab_snapshot(self):
        with self._lab_lock:
            return self._lab_frame, self._lab_detections

    def register_lab_viewer(self):
        """Llamar al inicio de un cliente MJPEG para activar el buffer de frames."""
        self._lab_viewer_count += 1

    def unregister_lab_viewer(self):
        """Llamar cuando un cliente MJPEG se desconecta."""
        self._lab_viewer_count = max(0, self._lab_viewer_count - 1)
        if self._lab_viewer_count == 0:
            with self._lab_lock:
                self._lab_frame = None
                self._lab_detections = None

    def set_lab_line(self, line_y=None, line_x=None, direction=None, cap_colors=None):
        """Actualiza la línea de conteo en vivo para preview del Lab (no persiste en BD)."""
        self._counter.update_line(y_ratio=line_y, x_ratio=line_x, direction=direction)
        if cap_colors is not None:
            self._cap_colors = cap_colors

    @property
    def total_count(self) -> int:
        return self._counter.total_count

    def _emit_count_event(self, count: int, detections):
        event = {
            'event_id': str(uuid.uuid4()),
            'device_id': self._device_id,
            'event_type': 'cut_detected',
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'payload': {
                'confidence': float(detections.confidence.mean()) if len(detections) > 0 else 0.0,
                'roi_id': 'yolo_main',
                'signals': {
                    'edge': 0.0,
                    'color': 0.0,
                    'flow': 0.0,
                    'yolo': float(detections.confidence.mean()) if len(detections) > 0 else 0.0,
                },
            },
        }
        for _ in range(count):
            event['event_id'] = str(uuid.uuid4())
            self._event_output.send_event(event)

    def _check_config(self):
        new_ver = self._config_port.get_config_version()
        if new_ver <= self._config_version:
            return
        cfg = self._config_port.get_config()
        if not cfg:
            return
        self._config_version = new_ver
        logger.info('config updated to v%d', new_ver)
        self._apply_config(cfg)

    def _apply_config(self, cfg: Dict[str, Any]):
        roi_raw = cfg.get('roi', {})
        # El config service puede devolver roi como lista [x,y,w,h,margin] o como dict
        if isinstance(roi_raw, list) and len(roi_raw) >= 4:
            rx, ry, rw, rh = roi_raw[0], roi_raw[1], roi_raw[2], roi_raw[3]
        elif isinstance(roi_raw, dict):
            rx, ry, rw, rh = roi_raw.get('x', 0), roi_raw.get('y', 0), roi_raw.get('width', 0), roi_raw.get('height', 0)
        else:
            rx, ry, rw, rh = 0, 0, 0, 0
        if rw > 0 and rh > 0:
            if self._roi is None:
                self._roi = ROIManager(rx, ry, rw, rh)
            else:
                self._roi.update(rx, ry, rw, rh)

        cam = cfg.get('camera') or {}
        if cam.get('url'):
            self._frame_input.update_url(cam['url'])
        if cam.get('frame_skip'):
            self._frame_skip = max(int(cam['frame_skip']), 1)

        oee = cfg.get('oee') or {}
        self._aggregator.update_config(
            interval_s=oee.get('snapshot_interval_s'),
            line_code=oee.get('line_name'),
        )

        yolo = cfg.get('yolo') or {}
        if yolo:
            self._yolo.update_config(
                model_name=yolo.get('model_name'),
                confidence=yolo.get('confidence'),
                classes=yolo.get('classes'),
                class_names=yolo.get('class_names'),
            )
            if 'counting_line_y' in yolo or 'counting_line_x' in yolo or 'counting_direction' in yolo:
                self._counter.update_line(
                    y_ratio=yolo.get('counting_line_y'),
                    x_ratio=yolo.get('counting_line_x'),
                    direction=yolo.get('counting_direction'),
                )
            if 'cap_colors' in yolo:
                self._cap_colors = yolo.get('cap_colors') or {}
