# Optimización de CPU — vision-event-detector (Jetson Orin Nano Super)

**Fecha:** Marzo 2026  
**Dispositivo:** NVIDIA Jetson Orin Nano Super 8GB  
**Sistema:** JetPack 6 / L4T R36.4 / CUDA 12.6 / aarch64

---

## Contexto

El servicio `vision-event-detector` procesa streams de video RTSP de cámaras IP (1080p@25fps) en tiempo real para detectar eventos de producción (corte, inicio, pausa). Cada instancia del servicio atiende una cámara.

**Problema inicial:** Con la imagen base `ubuntu:22.04` + `pip install opencv-python`, el servicio consumía **106% CPU por cámara** (sobre una base de 100% = 1 núcleo). Esto hacía inviable conectar más de 2 cámaras simultáneas en el Jetson Orin Nano Super (6 núcleos × 100% = 600% total disponible).

**Objetivo:** Bajar a menos de 25% CPU por cámara para soportar al menos 5 cámaras simultáneas.

---

## Arquitectura del servicio

El servicio calcula 4 señales visuales por frame procesado:

| Signal | Operación | Coste relativo |
|---|---|---|
| `FlowSignal` | Flujo óptico denso (movimiento vertical) | Alto |
| `EdgeSignal` | Densidad de bordes (Canny) | Medio |
| `HistogramSignal` | Correlación de histograma HSV | Bajo |
| `BeigeSignal` | Cobertura de color beige (inRange HSV) | Bajo |

La fusión de señales alimenta una máquina de estados (`EventFSM`) que emite eventos al sistema.

---

## Optimizaciones aplicadas

### 1. Reemplazo de OpenCV pip → OpenCV NVIDIA (con GStreamer + NVDEC)

**Problema:** `pip install opencv-python` instala una build genérica sin soporte de hardware. El decode H264 del stream RTSP se hacía en CPU pura.

**Solución:** Instalar `libopencv-python 4.8.0` desde el repositorio APT oficial de NVIDIA para Jetson (`repo.download.nvidia.com/jetson/common r36.4`). Esta build incluye:
- **GStreamer** con plugin NVDEC (decode H264 en chip dedicado)
- Soporte nativo para formatos de memoria Jetson (NVMM)

**Cambio en Dockerfile:**
```dockerfile
RUN echo "deb [trusted=yes] https://repo.download.nvidia.com/jetson/common r36.4 main" \
    > /etc/apt/sources.list.d/nvidia-l4t.list \
    && apt-get update \
    && apt-get install -y libopencv libopencv-python libegl1
```

**Restricción añadida en `requirements.txt`:**
```
numpy>=1.24,<2
```
La build 4.8.0 de NVIDIA fue compilada con NumPy 1.x; NumPy 2.x es incompatible.

**Resultado:** 106% → **67% CPU** (−37%)

---

### 2. VPI OFA — Optical Flow Accelerator (chip dedicado Jetson Orin)

**Problema:** `cv2.calcOpticalFlowFarneback()` es la operación más costosa del pipeline — corre en CPU pura y procesa cada pixel del ROI en cada frame.

**Solución:** El Jetson Orin incluye un chip físico dedicado llamado **OFA (Optical Flow Accelerator)** accesible via NVIDIA VPI (Vision Programming Interface) v3.2.4. Al ejecutarse en silicio dedicado, consume **0% CPU** para el cálculo del flujo óptico.

**Requisitos técnicos del OFA:**
- Imágenes en formato block-linear (`Y8_ER_BL`), mínimo 64×64 píxeles
- Conversión desde pitch-linear (numpy array) via chip VIC
- API: `vpi.optflow_dense(prev_bl, cur_bl, backend=vpi.Backend.OFA, quality=vpi.OptFlowQuality.LOW)`

**Paquetes añadidos al Dockerfile:**
```dockerfile
python3.10-vpi3
libegl1
```

**Implementación en `signal_extractors.py`:**
- Detección automática de OFA al iniciar (`_init_vpi()`) con fallback a Farneback CPU
- Buffers VPI (`_prev_bl`, `_cur_bl`) **pre-asignados** en el primer frame — evita `malloc` por frame
- Swap de punteros al final de cada frame: `self._prev_bl, self._cur_bl = self._cur_bl, self._prev_bl` — O(1), cero copia

**Benchmark del chip OFA aislado:**
- Latencia: ~6.5ms por frame
- CPU: ~0%
- Throughput máximo: ~155fps

**Resultado:** 67% → **42% CPU** (−25%)  
**Resultado post-buffer pre-asignado:** 42% → **30.8% CPU** (−11%)

---

### 3. Caché de conversiones de color por frame

**Problema:** Por cada frame procesado se llamaban 4 veces a funciones de conversión de color en `CVImageProcessor`:
- `cvtColor(BGR2GRAY)` × 2 — `EdgeSignal` y `FlowSignal` reciben el mismo `roi_frame`
- `cvtColor(BGR2HSV)` × 2 — `HistogramSignal` y `BeigeSignal` reciben el mismo `roi_frame`

**Solución:** Caché por `id(frame)` en `CVImageProcessor`. Si el frame ya fue convertido en este tick, retorna el array cacheado sin recalcular.

```python
def to_grayscale(self, frame: np.ndarray) -> np.ndarray:
    fid = id(frame)
    if self._gray_cache_id != fid:
        self._gray_cache_id = fid
        self._gray_cache = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
    return self._gray_cache
```

Adicionalmente: arrays de umbrales de `beige_ratio` (`_beige_lo`, `_beige_hi`) movidos a `__init__` para evitar `np.array()` por frame.

**Resultado:** ~1-2% CPU adicional eliminado

---

### 4. GStreamer `capture_every` — reducción de memcpy en callback

**Problema:** El callback `_on_new_sample` de GStreamer copiaba **todos** los frames (25fps) a un buffer numpy, aunque el detector solo procesa 1 de cada N frames (`frame_skip`). A 1080p, esto representa ~150MB/s de copias de memoria innecesarias.

**Solución:** Parámetro `capture_every=N` en `GStreamerAdapter`. Cuando el contador no es múltiple de N, el sample se descarta sin copiar a numpy:

```python
self._capture_counter += 1
if self._capture_counter % self._capture_every != 0:
    sink.emit('pull-sample')  # libera el buffer sin copiar
    return Gst.FlowReturn.OK
```

En `main.py` se pasa `frame_skip` como `capture_every` al construir el adapter.

**Resultado:** 30.8% → **28.1% CPU** (−2.7%)

---

### 5. SIGNAL_SCALE — reducción de resolución del ROI

**Solución:** Variable de entorno `SIGNAL_SCALE` (default `0.75`) que reduce el ROI extraído antes de pasarlo a los signals. Con `0.75`, cada dimensión se reduce a 75% → 56% de píxeles totales → ~40% menos trabajo en `Canny` e `inRange`.

No afecta a VPI OFA (trabaja con imágenes redimensionadas igualmente).

**Configuración en docker-compose:**
```yaml
SIGNAL_SCALE: ${SIGNAL_SCALE:-0.75}
```

---

### 6. FRAME_SKIP: 2 → 3

**Cambio de configuración:** Aumentar `FRAME_SKIP` de 2 a 3 reduce la tasa efectiva de procesamiento de 12.5fps a 8.3fps.

**Impacto en detección:** A 8.3fps se muestrea un frame cada 120ms. Las prendas en líneas de confección tardan típicamente >300ms en cruzar el ROI, por lo que 8.3fps es suficiente para detección confiable.

**Límite inferior recomendado:** No bajar de `FRAME_SKIP=3` (8.3fps). Con `FRAME_SKIP=4` (6.25fps) se arriesga perder eventos cortos de <200ms.

**Resultado:** 28.1% → **24.3% CPU** (−3.8%)

---

## Resumen de progresión

| Etapa | CPU por cámara | Técnica aplicada |
|---|---|---|
| Baseline | 106% | `pip install opencv-python`, CPU pura |
| + NVDEC + frame_skip=2 | 67% | `libopencv-python` NVIDIA, decode H264 en chip |
| + VPI OFA | 42% | Flujo óptico en chip OFA, 0% CPU |
| + Buffer pre-asignado OFA | 30.8% | Swap de punteros, sin malloc por frame |
| + cache cvtColor + captura | 28.1% | Cache de conversiones + GStreamer capture_every |
| + SIGNAL_SCALE=0.75 | ~27% | ROI al 75%, menos píxeles en Canny/inRange |
| **+ FRAME_SKIP=3** | **24.3%** | **8.3fps efectivo, punto óptimo** |

**Reducción total: 106% → 24.3% = −77% de CPU por cámara**

---

## Capacidad del sistema

**Hardware:** Jetson Orin Nano Super — 6 núcleos ARM Cortex-A78AE = **600% CPU total**

El sistema operativo + servicios de soporte (PostgreSQL, edge-config-service, resiliencia, edge-gateway) consumen aproximadamente 100-150% en operación normal.

| Cámaras | CPU cámaras | CPU total estimado | % del sistema | Estado |
|---|---|---|---|---|
| 1 | 24.3% | ~150% | 25% | ✅ |
| 3 | 73% | ~200% | 33% | ✅ |
| 5 | 122% | ~270% | 45% | ✅ objetivo original |
| **8** | **194%** | **~340%** | **57%** | ✅ límite práctico |
| 10 | 243% | ~390% | 65% | ⚠️ margen ajustado |
| 12+ | 292%+ | ~440%+ | 73%+ | ❌ sin margen de seguridad |

> **Nota sobre la métrica:** `docker stats` reporta CPU como porcentaje de un solo núcleo (100% = 1 núcleo). El sistema total tiene 600 unidades (6 núcleos × 100%). Un valor de 194% para 8 cámaras equivale al 32% del sistema total, no al 194%.

**Recomendación de producción: 5-8 cámaras simultáneas** por Jetson Orin Nano Super.

---

## Variables de configuración

Configurables via variables de entorno en `docker-compose.jetson-orin.yml` o en el `.env` del Jetson:

| Variable | Default | Descripción |
|---|---|---|
| `FRAME_SKIP` | `3` | Procesar 1 de cada N frames. No bajar de 3. |
| `SIGNAL_SCALE` | `0.75` | Escala del ROI antes de signals (0.5–1.0). |
| `FRAME_BACKEND` | `gstreamer` | Backend de captura. Usar siempre `gstreamer` en Jetson. |
| `OEE_INTERVAL` | `300` | Intervalo en segundos para emitir métricas OEE. |

---

## Archivos modificados

| Archivo | Cambio |
|---|---|
| `Dockerfile` | Base NVIDIA apt, `libegl1`, `python3.10-vpi3` |
| `requirements.txt` | `numpy>=1.24,<2` |
| `app/adapters/gstreamer_adapter.py` | `capture_every` para saltar frames en callback |
| `app/adapters/cv_image_processor.py` | Caché de conversiones, arrays pre-asignados |
| `app/domain/signals/signal_extractors.py` | VPI OFA con buffers pre-asignados y swap |
| `app/application/detector_service.py` | `signal_scale`, log de backend extendido |
| `app/main.py` | `SIGNAL_SCALE` env var, `capture_every=frame_skip` |
| `infrastructure/docker/docker-compose.jetson-orin.yml` | Defaults `FRAME_SKIP=3`, `SIGNAL_SCALE=0.75` |
