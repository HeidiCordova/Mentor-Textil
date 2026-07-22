import io
import json
import logging
import threading
from http.server import HTTPServer, ThreadingHTTPServer, BaseHTTPRequestHandler
from typing import Optional

import cv2
import numpy as np

# ── Rangos HSV de colores de tapa (OpenCV: H 0-179, S 0-255, V 0-255) ──────────
# Cada color tiene uno o dos rangos (rojo cruza el 0/180).
_CAP_HSV_RANGES = [
    ('rojo',     [(  0, 100,  60), ( 10, 255, 255)], [(165, 100,  60), (179, 255, 255)]),
    ('naranja',  [( 11,  80,  60), ( 25, 255, 255)], None),
    ('amarillo', [( 26,  80,  60), ( 34, 255, 255)], None),
    ('verde',    [( 35,  50,  40), ( 80, 255, 255)], None),
    ('celeste',  [( 81,  50,  40), (110, 255, 255)], None),
    ('azul',     [(111,  60,  40), (130, 255, 255)], None),
    ('violeta',  [(131,  40,  40), (150, 255, 255)], None),
    ('rosado',   [(151,  40,  40), (164, 255, 255)], None),
]


def _detect_cap_color(frame_bgr, x1: int, y1: int, x2: int, y2: int) -> str:
    """Detecta el color dominante en la franja superior (tapa) de un bounding box."""
    cap_h = max(1, int((y2 - y1) * 0.30))   # top 30% = zona de tapa
    roi = frame_bgr[y1:y1 + cap_h, x1:x2]
    if roi.size == 0:
        return 'desconocido'
    hsv = cv2.cvtColor(roi, cv2.COLOR_BGR2HSV)
    best_color, best_px = 'desconocido', 0
    for name, r1, r2 in _CAP_HSV_RANGES:
        mask = cv2.inRange(hsv, np.array(r1[0]), np.array(r1[1]))
        if r2 is not None:
            mask |= cv2.inRange(hsv, np.array(r2[0]), np.array(r2[1]))
        px = int(mask.sum() // 255)
        if px > best_px:
            best_px, best_color = px, name
    # Si no alcanza al menos 8% del roi → sin color definido
    total = roi.shape[0] * roi.shape[1]
    if best_px < total * 0.08:
        return 'desconocido'
    return best_color

logger = logging.getLogger('yolo.http')

_service_ref = None


def set_service(svc):
    global _service_ref
    _service_ref = svc


class Handler(BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):
        pass

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()

    def do_POST(self):
        if self.path == '/lab/preview':
            svc = _service_ref
            try:
                length = int(self.headers.get('Content-Length', 0))
                body = json.loads(self.rfile.read(length)) if length else {}
                if svc:
                    svc.set_lab_line(
                        line_y=body.get('counting_line_y'),
                        line_x=body.get('counting_line_x'),
                        direction=body.get('counting_direction'),
                        cap_colors=body.get('cap_colors'),
                    )
                self._json_response({'ok': True})
            except Exception as e:
                self._json_response({'error': str(e)}, 400)
        else:
            self.send_error(404)

    def do_GET(self):
        if self.path == '/health':
            self._json_response({'service': 'yolo-counter', 'status': 'ok'})
        elif self.path == '/status':
            self._status()
        elif self.path == '/lab/snapshot':
            self._lab_snapshot()
        elif self.path.startswith('/lab/stream'):
            self._lab_stream()
        else:
            self.send_error(404)

    def _json_response(self, data: dict, code: int = 200):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(body)

    def _status(self):
        svc = _service_ref
        if svc is None:
            self._json_response({'error': 'not initialized'}, 503)
            return
        self._json_response({
            'model': svc._yolo.model_name,
            'total_count': svc.total_count,
            'line_y_ratio': svc._counter.line_y_ratio,
            'line_x_ratio': svc._counter.line_x_ratio,
            'direction': svc._counter._direction,
            'frame_skip': svc._frame_skip,
        })

    def _lab_snapshot(self):
        svc = _service_ref
        if svc is None:
            self.send_error(503)
            return
        frame, detections = svc.get_lab_snapshot()
        if frame is None:
            self.send_error(204)
            return
        annotated = self._annotate(frame, detections)
        _, buf = cv2.imencode('.jpg', annotated, [cv2.IMWRITE_JPEG_QUALITY, 80])
        data = buf.tobytes()
        self.send_response(200)
        self.send_header('Content-Type', 'image/jpeg')
        self.send_header('Content-Length', str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _lab_stream(self):
        import time
        svc = _service_ref
        if svc is None:
            self.send_error(503)
            return
        self.send_response(200)
        self.send_header('Content-Type', 'multipart/x-mixed-replace; boundary=frame')
        self.send_header('Cache-Control', 'no-cache')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        svc.register_lab_viewer()  # activa el buffer de frames
        try:
            while True:
                frame, detections = svc.get_lab_snapshot()
                if frame is not None:
                    annotated = self._annotate(frame, detections)
                    _, buf = cv2.imencode('.jpg', annotated, [cv2.IMWRITE_JPEG_QUALITY, 65])
                    data = buf.tobytes()
                    self.wfile.write(b'--frame\r\n')
                    self.wfile.write(b'Content-Type: image/jpeg\r\n')
                    self.wfile.write(f'Content-Length: {len(data)}\r\n\r\n'.encode())
                    self.wfile.write(data)
                    self.wfile.write(b'\r\n')
                time.sleep(0.1)  # ~10 fps máximo
        except (BrokenPipeError, ConnectionResetError):
            pass
        finally:
            svc.unregister_lab_viewer()  # libera el buffer cuando se desconecta

    def _annotate(self, frame, detections):
        import supervision as sv
        annotated = frame.copy()
        svc = _service_ref
        cap_colors: dict = svc._cap_colors if svc else {}

        if detections is not None and len(detections) > 0:
            box_annotator = sv.BoxAnnotator(thickness=2)
            annotated = box_annotator.annotate(annotated, detections)

            # Etiquetar siempre con color detectado o marca (ayuda a depurar)
            for bbox in detections.xyxy:
                x1, y1, x2, y2 = int(bbox[0]), int(bbox[1]), int(bbox[2]), int(bbox[3])
                color_name = _detect_cap_color(frame, x1, y1, x2, y2)
                brand = cap_colors.get(color_name, '') if cap_colors else ''
                # Si hay marca → mostrarla en cian; si no → mostrar color en gris
                if brand:
                    label = brand
                    text_color = (0, 220, 255)   # cian
                elif color_name != 'desconocido':
                    label = f'[{color_name}]'
                    text_color = (160, 160, 160)  # gris
                else:
                    continue
                (tw, th), _ = cv2.getTextSize(label, cv2.FONT_HERSHEY_SIMPLEX, 0.6, 2)
                lx = x1
                ly = y1 - 8 if y1 > th + 12 else y2 + th + 8
                cv2.rectangle(annotated, (lx, ly - th - 4), (lx + tw + 8, ly + 4), (0, 0, 0), -1)
                cv2.putText(annotated, label, (lx + 4, ly),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.6, text_color, 2)

        h, w = annotated.shape[:2]
        if svc:
            counter = svc._counter
            if counter.is_horizontal_dir:
                x = int(w * counter.line_x_ratio)
                cv2.line(annotated, (x, 0), (x, h), (0, 255, 0), 2)
            else:
                y = int(h * counter.line_y_ratio)
                cv2.line(annotated, (0, y), (w, y), (0, 255, 0), 2)
            cv2.putText(
                annotated,
                f'Count: {svc.total_count}',
                (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 1.0, (0, 255, 0), 2,
            )
        return annotated


def start_http_server(port: int = 8006):
    server = ThreadingHTTPServer(('0.0.0.0', port), Handler)
    t = threading.Thread(target=server.serve_forever, daemon=True)
    t.start()
    logger.info('HTTP server on :%d', port)
    return server
