import os
import logging
import requests
from .adapters.config_client import ConfigClient
from .adapters.http_event_adapter import HTTPEventAdapter
from .adapters.cv_image_processor import CVImageProcessor
from .adapters.postgres_calibration_repository import PostgresCalibrationRepository
from .application.detector_service import DetectorService

logger = logging.getLogger('detector.main')


def _build_frame_input(camera_url: str, backend: str):
    if backend == 'gstreamer':
        try:
            from .adapters.gstreamer_adapter import GStreamerAdapter
            # capture_every=frame_skip -> GStreamer solo hace memcpy en 1 de cada N frames
            # Reduce el uso de CPU del callback de captura a la mitad (o más)
            adapter = GStreamerAdapter(camera_url, capture_every=frame_skip)
            logger.info('[main] Backend: GStreamer (NVDEC)')
            return adapter
        except Exception as exc:
            logger.warning(
                '[main] GStreamer no disponible (%s) — usando OpenCV como fallback', exc
            )

    from .adapters.opencv_adapter import OpenCVAdapter as _OCV
    logger.info('[main] Backend: OpenCV (CPU)')
    return _OCV(camera_url)


def main():
    config_service_url = os.getenv('CONFIG_SERVICE_URL', 'http://edge-config-service:8004')
    resiliencia_url = os.getenv('RESILIENCIA_URL', 'http://resiliencia:8002')
    gateway_url = os.getenv('GATEWAY_URL', 'http://edge-gateway:8005')

    # LINEA_ID env var takes priority; fallback to auto-detect from config-service
    linea_id = os.getenv('LINEA_ID', '')
    if not linea_id:
        try:
            resp = requests.get(f"{config_service_url}/config/lines", timeout=5)
            if resp.status_code == 200:
                lines = resp.json().get('lines', [])
                if lines:
                    linea_id = str(lines[0])
        except Exception:
            pass
    if not linea_id:
        linea_id = '1'

    # Also fetch device_id for domain identification
    device_id = os.getenv('DEVICE_ID', '')
    if not device_id:
        try:
            resp = requests.get(f"{config_service_url}/config/device-id", timeout=5)
            if resp.status_code == 200:
                device_id = resp.json().get('device_id', '')
        except Exception:
            pass
    if not device_id:
        device_id = 'jetson-1'
    logger.info('[main] linea_id=%s device_id=%s', linea_id, device_id)

    # Defaults from env (overridden by DB config below)
    camera_url = os.getenv('CAMERA_URL', '')
    line_code = os.getenv('LINE_CODE', 'MENTOR_DEFAULT_L1')
    oee_interval = float(os.getenv('OEE_INTERVAL', '300'))
    micro_stop_max_s = float(os.getenv('MICRO_STOP_MAX_S', '120'))  # textil default; botellas usa 210
    frame_backend = os.getenv('FRAME_BACKEND', 'opencv').lower()
    frame_skip = int(os.getenv('FRAME_SKIP', '2'))
    signal_scale = float(os.getenv('SIGNAL_SCALE', '1.0'))
    calibration_samples = int(os.getenv('CALIBRATION_SAMPLES', '30'))
    calibration_min_quality = float(os.getenv('CALIBRATION_MIN_QUALITY', '0.90'))
    calibration_expiry_days = int(os.getenv('CALIBRATION_EXPIRY_DAYS', '90'))

    # Override env defaults with DB config (single source of truth)
    config_port = ConfigClient(config_service_url, linea_id)
    initial_cfg = config_port.get_config()
    if initial_cfg:
        cam = initial_cfg.get('camera') or {}
        if cam.get('url'):
            camera_url = cam['url']
        if cam.get('frame_backend'):
            frame_backend = cam['frame_backend']
        if cam.get('frame_skip'):
            frame_skip = int(cam['frame_skip'])
        if cam.get('signal_scale'):
            signal_scale = float(cam['signal_scale'])
        oee = initial_cfg.get('oee') or {}
        if oee.get('line_name'):
            line_code = oee['line_name']
        if oee.get('snapshot_interval_s'):
            oee_interval = float(oee['snapshot_interval_s'])
        if oee.get('micro_stop_max_s'):
            micro_stop_max_s = float(oee['micro_stop_max_s'])
        logger.info('[main] Config loaded from DB (version=%s)', initial_cfg.get('config_version'))

    frame_input = _build_frame_input(camera_url, frame_backend)
    event_output = HTTPEventAdapter(resiliencia_url, linea_id=linea_id)
    image_processor = CVImageProcessor()

    calibration_repository = None
    postgres_url = os.getenv('POSTGRES_URL', '').strip()
    if postgres_url or os.getenv('PGHOST'):
        try:
            calibration_repository = PostgresCalibrationRepository(postgres_url)
        except ValueError as exc:
            logger.error('[main] Invalid PostgreSQL calibration configuration: %s', exc)
    else:
        logger.warning(
            '[main] POSTGRES_URL not configured; HSV calibration will be memory-only'
        )

    detector = DetectorService(
        frame_input=frame_input,
        config_port=config_port,
        event_output=event_output,
        image_processor=image_processor,
        device_id=device_id,
        line_id=linea_id,
        line_code=line_code,
        oee_interval=oee_interval,
        micro_stop_max_s=micro_stop_max_s,
        gateway_url=gateway_url,
        frame_skip=frame_skip,
        signal_scale=signal_scale,
        calibration_repository=calibration_repository,
        calibration_required_samples=calibration_samples,
        calibration_min_quality=calibration_min_quality,
        calibration_expiry_days=calibration_expiry_days,
    )

    print(f"Starting vision-event-detector for device: {device_id} line: {line_code}")
    detector.run()


if __name__ == '__main__':
    main()
