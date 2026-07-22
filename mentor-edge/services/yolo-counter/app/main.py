import os
import logging
import requests
from .adapters.config_client import ConfigClient
from .adapters.http_event_adapter import HTTPEventAdapter
from .adapters.opencv_adapter import OpenCVAdapter
from .application.counter_service import CounterService
from .http_server import start_http_server, set_service

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(name)s] %(levelname)s %(message)s',
)
logger = logging.getLogger('yolo.main')


def main():
    config_service_url = os.getenv('CONFIG_SERVICE_URL', 'http://edge-config-service:8004')
    resiliencia_url = os.getenv('RESILIENCIA_URL', 'http://resiliencia:8002')

    linea_id = os.getenv('LINEA_ID', '')
    if not linea_id:
        # Leer la línea asignada EXPLÍCITAMENTE desde la configuración de sistema en BD.
        # Si no hay ninguna asignada, yolo-counter arranca sin cámara (modo espera).
        # NO se hace fallback automático a otras líneas: la cámara no debe abrirse
        # si el usuario no ha configurado y activado una línea en la UI.
        try:
            resp = requests.get(f"{config_service_url}/config/system", timeout=5)
            if resp.status_code == 200:
                assigned = resp.json().get('yolo_assigned_linea')
                if assigned and int(assigned) > 0:
                    linea_id = str(assigned)
        except Exception:
            pass

    if not linea_id:
        logger.warning('No hay linea_id configurado (yolo_assigned_linea ausente). '
                       'Arrancando en modo espera — sin conexión a cámara.')
        # Arrancar solo el HTTP server para que la UI pueda asignar la línea luego.
        # El servicio no procesará frames hasta que se reinicie con la línea asignada.
        start_http_server(port=int(os.getenv('PORT', '8006')))
        import time
        while True:
            time.sleep(60)
            logger.info('yolo-counter en espera — asigna una línea en Configuración > Dispositivos')

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

    logger.info('linea_id=%s device_id=%s', linea_id, device_id)

    config_port = ConfigClient(config_service_url, linea_id)
    cfg = config_port.get_config()

    # Si la línea asignada no tiene config en BD → no conectar cámara.
    # El usuario debe configurar la línea correctamente en la UI antes de que
    # yolo-counter use la cámara. No se aceptan fallbacks automáticos.
    if not cfg:
        logger.error('No hay configuración para linea_id=%s. '
                     'Configura la línea en la UI y reinicia yolo-counter.', linea_id)

    camera_url = os.getenv('CAMERA_URL', '')
    line_code = os.getenv('LINE_CODE', 'MENTOR_DEFAULT_L1')
    oee_interval = 300.0
    frame_skip = 2

    if cfg:
        cam = cfg.get('camera') or {}
        camera_url = cam.get('url', camera_url)
        frame_skip = int(cam.get('frame_skip', frame_skip))
        oee = cfg.get('oee') or {}
        line_code = oee.get('line_name', line_code)
        oee_interval = float(oee.get('snapshot_interval_s', oee_interval))

    if not camera_url:
        logger.error('linea_id=%s no tiene URL de cámara configurada. '
                     'Configura la cámara en la UI y reinicia yolo-counter.', linea_id)
        start_http_server(port=int(os.getenv('PORT', '8006')))
        import time
        while True:
            time.sleep(60)
            logger.info('yolo-counter en espera — configura la cámara de la línea %s', linea_id)

    yolo_config = cfg.get('yolo', {}) if cfg else {}
    if not yolo_config:
        yolo_config = {
            'model_name': os.getenv('YOLO_MODEL', 'yolo26s'),
            'confidence': float(os.getenv('YOLO_CONFIDENCE', '0.45')),
            'use_tensorrt': os.getenv('YOLO_TENSORRT', 'true').lower() == 'true',
            'counting_line_y': float(os.getenv('YOLO_LINE_Y', '0.5')),
            'counting_direction': os.getenv('YOLO_DIRECTION', 'top_to_bottom'),
        }

    frame_input = OpenCVAdapter(camera_url)
    event_output = HTTPEventAdapter(resiliencia_url, linea_id=linea_id)

    service = CounterService(
        frame_input=frame_input,
        config_port=config_port,
        event_output=event_output,
        device_id=device_id,
        line_code=line_code,
        oee_interval=oee_interval,
        frame_skip=frame_skip,
        yolo_config=yolo_config,
    )

    set_service(service)
    start_http_server(port=int(os.getenv('PORT', '8006')))

    print(f"yolo-counter started: device={device_id} line={line_code} model={yolo_config.get('model_name', 'yolo26s')}")
    service.run()


if __name__ == '__main__':
    main()
