import cv2
import numpy as np
from ..ports.image_processor import ImageProcessor

# Use CUDA Farneback when available (Jetson CUDA OpenCV build).
# Falls back to CPU Farneback transparently.
_cuda_of: object = None

def _init_cuda_of() -> None:
    global _cuda_of
    try:
        if hasattr(cv2, 'cuda') and cv2.cuda.getCudaEnabledDeviceCount() > 0:
            _cuda_of = cv2.cuda.FarnebackOpticalFlow.create(
                numLevels=3,
                pyrScale=0.5,
                fastPyramids=True,
                winSize=13,
                numIters=3,
                polyN=5,
                polySigma=1.1,
                flags=0,
            )
    except Exception:
        _cuda_of = None

_init_cuda_of()


class CVImageProcessor(ImageProcessor):
    """Procesador de imagen con caché de conversiones por frame.

    Los 4 signals reciben el mismo roi_frame por tick. Sin caché se harían:
      - cvtColor(BGR2GRAY) x2 (EdgeSignal + FlowSignal)
      - cvtColor(BGR2HSV)  x2 (HistogramSignal + BeigeSignal)
    Cada caché conserva una referencia fuerte al frame para comparar su
    identidad sin depender de id(frame), que Python puede reutilizar.
    """

    def __init__(self) -> None:
        # La referencia fuerte evita que otro ndarray reutilice la identidad
        # del frame cacheado mientras su conversión siga vigente.
        self._gray_cache_frame: np.ndarray | None = None
        self._gray_cache: np.ndarray | None = None
        self._hsv_cache_frame: np.ndarray | None = None
        self._hsv_cache: np.ndarray | None = None
        # Arrays pre-asignados para beige_ratio (evita np.array() por frame)
        self._beige_lo = np.array([15, 30, 120], dtype=np.uint8)
        self._beige_hi = np.array([35, 130, 250], dtype=np.uint8)

    def update_beige_range(self, h_min: int, h_max: int, s_min: int, s_max: int, v_min: int) -> None:
        """Actualiza el rango HSV de deteccion beige en caliente."""
        self._beige_lo = np.array([h_min, s_min, v_min], dtype=np.uint8)
        self._beige_hi = np.array([h_max, s_max, 255],   dtype=np.uint8)

    def to_grayscale(self, frame: np.ndarray) -> np.ndarray:
        if self._gray_cache_frame is not frame:
            self._gray_cache = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
            self._gray_cache_frame = frame
        return self._gray_cache

    def to_hsv(self, frame: np.ndarray) -> np.ndarray:
        if self._hsv_cache_frame is not frame:
            self._hsv_cache = cv2.cvtColor(frame, cv2.COLOR_BGR2HSV)
            self._hsv_cache_frame = frame
        return self._hsv_cache

    def detect_edges(self, gray: np.ndarray, low: int, high: int) -> np.ndarray:
        return cv2.Canny(gray, low, high)

    def calc_histogram(self, hsv: np.ndarray) -> np.ndarray:
        hist = cv2.calcHist([hsv], [0], None, [180], [0, 180])
        return cv2.normalize(hist, hist).flatten()

    def compare_histograms(self, h1: np.ndarray, h2: np.ndarray) -> float:
        return cv2.compareHist(h1, h2, cv2.HISTCMP_CORREL)

    def calc_optical_flow(self, prev_gray: np.ndarray, gray: np.ndarray) -> np.ndarray:
        if _cuda_of is not None:
            try:
                gpu_prev = cv2.cuda_GpuMat()
                gpu_curr = cv2.cuda_GpuMat()
                gpu_prev.upload(prev_gray)
                gpu_curr.upload(gray)
                gpu_flow = _cuda_of.calc(gpu_prev, gpu_curr, None)
                return gpu_flow.download()
            except Exception:
                pass
        return cv2.calcOpticalFlowFarneback(
            prev_gray, gray, None,
            0.5, 3, 15, 3, 5, 1.2, 0,
        )

    def edges_density(self, edges: np.ndarray) -> float:
        return float(np.sum(edges > 0) / edges.size)

    def beige_ratio(self, frame: np.ndarray) -> float:
        # Reutiliza la conversión HSV ya cacheada si el frame es el mismo
        hsv = self.to_hsv(frame)
        mask = cv2.inRange(hsv, self._beige_lo, self._beige_hi)
        return float(np.count_nonzero(mask) / mask.size)

    def vertical_mean(self, flow: np.ndarray) -> float:
        return float(np.mean(flow[..., 1]))
