"""
Tests funcionales contra la camara RTSP en vivo.
Se ejecutan dentro del contenedor, con acceso real al stream.
"""
import sys
import time
import cv2
import numpy as np
import urllib.request
import json

RTSP_URL   = "rtsp://admin2026:admin2026@192.168.1.188/stream1"
HEALTH_URL = "http://127.0.0.1:8001/health"
CAL_URL    = "http://127.0.0.1:8001/calibrate"
ROI        = (120, 60, 120 + 320, 60 + 200)   # (x1,y1,x2,y2)
DY_THRESHOLD  = 5.0
FLOW_GATE     = 0.5


def separator(title):
    print(f"\n{'='*60}")
    print(f"  {title}")
    print('='*60)


def result(name, passed, detail=""):
    mark = "PASS" if passed else "FAIL"
    line = f"  [{mark}] {name}"
    if detail:
        line += f"  ->  {detail}"
    print(line)
    return passed


# ---------------------------------------------------------------
# T01: Conectividad HTTP con el servicio
# ---------------------------------------------------------------
separator("T01  Health endpoint")
try:
    resp = urllib.request.urlopen(HEALTH_URL, timeout=5)
    data = json.loads(resp.read().decode())
    result("GET /health HTTP 200",       resp.status == 200)
    result("status == ok",               data.get("status") == "ok",   str(data.get("status")))
    result("camera == true",             data.get("camera") is True)
except Exception as e:
    result("GET /health reachable",      False, str(e))

# ---------------------------------------------------------------
# T02: Calibrate endpoint
# ---------------------------------------------------------------
separator("T02  Calibrate endpoint")
try:
    resp = urllib.request.urlopen(CAL_URL, timeout=5)
    data = json.loads(resp.read().decode())
    result("GET /calibrate HTTP 200",    resp.status == 200)
    result("has 'active' field",         "active"   in data, str(data))
    result("has 'progress' field",       "progress" in data)
except Exception as e:
    result("GET /calibrate reachable",   False, str(e))

# ---------------------------------------------------------------
# T03: Captura de frames RTSP
# ---------------------------------------------------------------
separator("T03  Captura RTSP")
cap = cv2.VideoCapture(RTSP_URL, cv2.CAP_FFMPEG)
cap.set(cv2.CAP_PROP_BUFFERSIZE, 1)
frames = []
for i in range(5):
    ok, frame = cap.read()
    if ok:
        frames.append(frame)
    time.sleep(0.04)
cap.release()

result("VideoCapture abre stream",       len(frames) > 0, f"{len(frames)}/5 frames")
if frames:
    f = frames[0]
    result("Frame tiene 3 canales BGR",  len(f.shape) == 3 and f.shape[2] == 3, str(f.shape))
    result("Frame dimension >= 240p",    f.shape[0] >= 240 and f.shape[1] >= 320,
           f"{f.shape[1]}x{f.shape[0]}")
    result("Frame no es negro puro",     np.mean(f) > 5.0, f"mean pixel={np.mean(f):.1f}")

# ---------------------------------------------------------------
# T04: Extraccion ROI
# ---------------------------------------------------------------
separator("T04  Extraccion ROI")
if frames:
    frame  = frames[0]
    x1,y1,x2,y2 = ROI
    h, w = frame.shape[:2]
    roi_valid = (x1 < w and x2 <= w and y1 < h and y2 <= h)
    result("ROI cabe dentro del frame",  roi_valid, f"frame={w}x{h}  roi={x1},{y1},{x2},{y2}")
    if roi_valid:
        crop = frame[y1:y2, x1:x2]
        result("Crop tiene contenido",   crop.size > 0 and np.mean(crop) > 2.0,
               f"shape={crop.shape} mean={np.mean(crop):.1f}")

# ---------------------------------------------------------------
# T05: EdgeSignal
# ---------------------------------------------------------------
separator("T05  EdgeSignal")
if frames:
    edge_scores = []
    for f in frames:
        crop = f[ROI[1]:ROI[3], ROI[0]:ROI[2]]
        gray  = cv2.cvtColor(crop, cv2.COLOR_BGR2GRAY)
        edges = cv2.Canny(gray, 50, 150)
        score = float(np.sum(edges > 0) / edges.size)
        edge_scores.append(score)
    avg = np.mean(edge_scores)
    result("EdgeSignal retorna float [0,1]",  0 <= avg <= 1,    f"avg={avg:.4f}")
    result("EdgeSignal no es 0 constante",    avg > 0.001,      f"avg={avg:.4f}")
    result("EdgeSignal no es 1 constante",    avg < 0.99,       f"avg={avg:.4f}")

# ---------------------------------------------------------------
# T06: FlowSignal — sentido CORRECTO (positivo = cae)
# ---------------------------------------------------------------
separator("T06  FlowSignal (signo corregido)")
if len(frames) >= 2:
    flow_values = []
    prev = cv2.cvtColor(frames[0][ROI[1]:ROI[3], ROI[0]:ROI[2]], cv2.COLOR_BGR2GRAY)
    for f in frames[1:]:
        curr = cv2.cvtColor(f[ROI[1]:ROI[3], ROI[0]:ROI[2]], cv2.COLOR_BGR2GRAY)
        flow = cv2.calcOpticalFlowFarneback(prev, curr, None, 0.5, 3, 15, 3, 5, 1.2, 0)
        vy   = float(np.mean(flow[..., 1]))
        flow_values.append(vy)
        prev = curr
    avg_vy = np.mean(flow_values)
    result("FlowSignal computa sin error",    True,             f"vy_mean={avg_vy:.4f}")
    result("vy es float no NaN",              not np.isnan(avg_vy))
    # Gate: cuando ROI es estatico el flow debe ser ~0
    gate_ok = abs(avg_vy) < DY_THRESHOLD
    result("ROI estatico -> vy < dy_threshold (flow gate OK)", gate_ok,
           f"|vy|={abs(avg_vy):.3f}  threshold={DY_THRESHOLD}")
    if not gate_ok:
        # Hay movimiento real — verificar que la senal detecta
        flow_signal = 1.0 if avg_vy > DY_THRESHOLD else 0.0
        result("Si hay movimiento positivo, flow=1.0", flow_signal == 1.0,
               f"avg_vy={avg_vy:.3f}")

# ---------------------------------------------------------------
# T07: HistogramSignal (color) — referencia volatil fija
# ---------------------------------------------------------------
separator("T07  HistogramSignal")
if len(frames) >= 2:
    ref_frame   = frames[0][ROI[1]:ROI[3], ROI[0]:ROI[2]]
    test_frame  = frames[-1][ROI[1]:ROI[3], ROI[0]:ROI[2]]
    h_ref  = cv2.calcHist([cv2.cvtColor(ref_frame, cv2.COLOR_BGR2HSV)],
                           [0], None, [180], [0,180])
    h_test = cv2.calcHist([cv2.cvtColor(test_frame, cv2.COLOR_BGR2HSV)],
                           [0], None, [180], [0,180])
    cv2.normalize(h_ref,  h_ref)
    cv2.normalize(h_test, h_test)
    corr  = float(cv2.compareHist(h_ref.flatten(), h_test.flatten(), cv2.HISTCMP_CORREL))
    color_score = 1.0 - corr
    result("Histograma computa sin error",        True,             f"corr={corr:.4f}")
    result("color_score en [0,1]",                0 <= color_score <= 1, f"{color_score:.4f}")
    # Entre frames consecutivos del mismo video la correlacion debe ser alta
    result("Frames consecutivos alta correlacion", corr > 0.5,      f"corr={corr:.4f}")

# ---------------------------------------------------------------
# T08: FusionEngine — gate bloqueante
# ---------------------------------------------------------------
separator("T08  FusionEngine flow gate")
W = {'edge': 0.30, 'color': 0.25, 'flow': 0.35, 'beige': 0.10}
GATE = 0.5

def fuse(edge, color, flow, beige):
    if flow < GATE:
        return 0.0
    s = edge*W['edge'] + color*W['color'] + flow*W['flow'] + beige*W['beige']
    return min(1.0, max(0.0, s))

r_gate   = fuse(1.0, 1.0, 0.0, 1.0)
r_nogate = fuse(1.0, 1.0, 1.0, 1.0)
r_mid    = fuse(0.5, 0.5, 1.0, 0.5)
result("flow=0 -> score=0 (gate)",            r_gate   == 0.0, f"score={r_gate}")
result("flow=1 + todas=1 -> score=1.0",       r_nogate == 1.0, f"score={r_nogate}")
result("Pesos suman 1.0",                     abs(sum(W.values()) - 1.0) < 0.001)
result("Fusion mid produce 0 < score <= 1",   0 < r_mid <= 1,  f"score={r_mid:.3f}")

# ---------------------------------------------------------------
# T09: FSM — estados y transiciones
# ---------------------------------------------------------------
separator("T09  FSM estados y transiciones")

class _FSM:
    N=3; COOL=8; HI=0.7; LO=0.3; EXIT=5; MAX_EXIT=50
    def __init__(self):
        self.state='IDLE'; self.cc=0; self.el=0; self.wt=0; self.cd=0
    def step(self, score):
        if self.state=='IDLE':
            if score>self.HI: self.state='DETECTING'; self.cc=1
            return None
        if self.state=='DETECTING':
            if score>self.HI:
                self.cc+=1
                if self.cc>=self.N:
                    self.state='WAIT_EXIT'; self.el=0; self.wt=0; self.cc=0
                    return 'CORTE'
            elif score<self.LO: self.state='IDLE'; self.cc=0
            return None
        if self.state=='WAIT_EXIT':
            self.wt+=1
            if score<self.LO: self.el+=1
            else: self.el=0
            if self.el>=self.EXIT or self.wt>=self.MAX_EXIT:
                self.state='COOLDOWN'; self.cd=0; self.el=0; self.wt=0
            return None
        if self.state=='COOLDOWN':
            self.cd+=1
            if self.cd>=self.COOL: self.state='IDLE'; self.cd=0
            return None

# Test 1: deteccion normal
fsm = _FSM()
events = []
for s in [0.8]*3: e=fsm.step(s); events.append(e)
result("N frames altos -> emite CORTE",       'CORTE' in events)
result("Despues de CORTE -> WAIT_EXIT",        fsm.state == 'WAIT_EXIT')

# Test 2: exit por frames bajos
for s in [0.1]*5: fsm.step(s)
result("5 frames bajos en WAIT_EXIT -> COOLDOWN", fsm.state == 'COOLDOWN')

# Test 3: cooldown -> IDLE
for _ in range(8): fsm.step(0.1)
result("8 frames cooldown -> vuelve a IDLE",  fsm.state == 'IDLE')

# Test 4: no doble conteo — objeto que se queda
fsm2 = _FSM()
ev2 = []
for s in [0.8]*3: e=fsm2.step(s); ev2.append(e)
# objeto se queda (score alto persistente en WAIT_EXIT)
for s in [0.85]*50: fsm2.step(s)
# timeout fuerza COOLDOWN
result("Objeto persistente -> max_wait_exit fuerza COOLDOWN", fsm2.state == 'COOLDOWN')
for _ in range(8): fsm2.step(0.1)
# segunda prenda
ev3 = []
for s in [0.8]*3: e=fsm2.step(s); ev3.append(e)
result("Segunda prenda detectada solo 1 vez",  ev2.count('CORTE')==1 and ev3.count('CORTE')==1,
       f"cortes1={ev2.count('CORTE')} cortes2={ev3.count('CORTE')}")

# Test 5: ruido breve no cuenta
fsm3 = _FSM()
ev_noise = []
for s in [0.8, 0.2, 0.8]: e=fsm3.step(s); ev_noise.append(e)
result("Ruido intercalado interrumpe conteo",  'CORTE' not in ev_noise)

# ---------------------------------------------------------------
# T10: DB trigger vision_detections
# ---------------------------------------------------------------
separator("T10  DB trigger vision_detections")
import subprocess, uuid
test_id = str(uuid.uuid4())
ts = time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
sql = f"""
INSERT INTO events_buffer (event_id, device_id, event_type, timestamp, payload)
VALUES ('{test_id}', 'test-device', 'CORTE', '{ts}',
  '{{"line_code":"TEST_L1","confidence":0.92,"roi_id":"default",
     "signals":{{"edge":0.5,"color":0.4,"flow":1.0,"beige":0.1}}}}');
SELECT detection_id, confidence, signal_flow FROM vision_detections
WHERE detection_id = '{test_id}';
"""
try:
    out = subprocess.check_output(
        ['psql', '-U', 'mentor', '-d', 'mentor_edge', '-c', sql],
        stderr=subprocess.STDOUT
    ).decode()
    result("INSERT CORTE -> trigger crea fila en vision_detections",
           test_id in out, out.strip().splitlines()[-1] if out else "sin salida")
    result("signal_flow copiado correctamente",
           '1' in out.split(test_id)[-1] if test_id in out else False)
except Exception as e:
    result("Trigger vision_detections ejecutable", False, str(e))

# ---------------------------------------------------------------
# Resumen
# ---------------------------------------------------------------
print("\n" + "="*60)
print("  Tests completados")
print("="*60)
