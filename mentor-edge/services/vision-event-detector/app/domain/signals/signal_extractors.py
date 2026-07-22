from abc import ABC, abstractmethod
from typing import Optional, TYPE_CHECKING
import numpy as np

if TYPE_CHECKING:
    from ...ports.image_processor import ImageProcessor

# ---------------------------------------------------------------------------
# VPI OFA — Optical Flow Accelerator (chip dedicado Jetson Orin, 0% CPU)
# ---------------------------------------------------------------------------
_vpi = None
_OFA_AVAILABLE = False

def _init_vpi() -> None:
    global _vpi, _OFA_AVAILABLE
    try:
        import vpi as _v
        # OFA requiere imágenes de al menos 32x32
        _bl = _v.Image((64, 64), _v.Format.Y8_ER).convert(_v.Format.Y8_ER_BL, backend=_v.Backend.VIC)
        _v.optflow_dense(_bl, _bl, backend=_v.Backend.OFA, quality=_v.OptFlowQuality.LOW)
        _v.Stream.default.sync()
        _vpi = _v
        _OFA_AVAILABLE = True
    except Exception:
        _OFA_AVAILABLE = False

_init_vpi()


def _gray_to_bl(gray: np.ndarray, out_bl=None):
    """Convierte array numpy uint8 a imagen VPI block-linear (requiere OFA).
    Si se pasa out_bl pre-asignado, reutiliza el buffer (evita malloc por frame).
    """
    h, w = gray.shape
    pl = _vpi.asimage(gray, _vpi.Format.Y8_ER)
    if out_bl is None:
        out_bl = _vpi.Image((w, h), _vpi.Format.Y8_ER_BL)
    pl.convert(out_bl, backend=_vpi.Backend.VIC)
    return out_bl


def _ofa_vertical_mean(prev_bl, cur_bl) -> float:
    """Calcula flujo óptico con OFA y devuelve la media vertical (eje Y)."""
    mv = _vpi.optflow_dense(prev_bl, cur_bl, backend=_vpi.Backend.OFA, quality=_vpi.OptFlowQuality.LOW)
    _vpi.Stream.default.sync()
    # mv es formato 2S16: cada valor es S10.5 fixed-point → dividir entre 32
    with mv.rlock_cpu() as data:
        vy = data[..., 1].astype(np.float32) / 32.0
    return float(np.mean(vy))


def _ofa_flow_max(prev_bl, cur_bl) -> float:
    """Calcula flujo óptico con OFA y devuelve max(|media_x|, |media_y|).
    Detecta movimiento en cualquier dirección (horizontal o vertical)."""
    mv = _vpi.optflow_dense(prev_bl, cur_bl, backend=_vpi.Backend.OFA, quality=_vpi.OptFlowQuality.LOW)
    _vpi.Stream.default.sync()
    with mv.rlock_cpu() as data:
        arr = data.astype(np.float32) / 32.0
        vx_mean = abs(float(np.mean(arr[..., 0])))
        vy_mean = abs(float(np.mean(arr[..., 1])))
    return max(vx_mean, vy_mean)


class Signal(ABC):

    @abstractmethod
    def compute(self, frame: np.ndarray, roi_frame: np.ndarray) -> float:
        pass

    @abstractmethod
    def reset(self) -> None:
        pass

    def set_image_processor(self, processor: 'ImageProcessor') -> None:
        self._processor = processor


class EdgeSignal(Signal):

    def __init__(self, threshold: float = 0.4):
        self.threshold = threshold
        self._baseline: Optional[float] = None
        self._processor: Optional['ImageProcessor'] = None

    def compute(self, frame: np.ndarray, roi_frame: np.ndarray) -> float:
        gray = self._processor.to_grayscale(roi_frame)
        edges = self._processor.detect_edges(gray, 50, 150)
        return self._processor.edges_density(edges)

    def reset(self) -> None:
        self._baseline = None


class HistogramSignal(Signal):

    def __init__(self, threshold: float = 0.6):
        self.threshold = threshold
        self._reference_hist: Optional[np.ndarray] = None
        self._processor: Optional['ImageProcessor'] = None

    def compute(self, frame: np.ndarray, roi_frame: np.ndarray) -> float:
        hist = self.calculate_histogram(roi_frame)

        if self._reference_hist is None:
            # Calibration is an explicit lifecycle step. Never learn from an
            # arbitrary first production frame after a restart.
            return 0.0

        correlation = self.compare_histograms(self._reference_hist, hist)
        return 1.0 - correlation

    def calculate_histogram(self, roi_frame: np.ndarray) -> np.ndarray:
        hsv = self._processor.to_hsv(roi_frame)
        return self._processor.calc_histogram(hsv)

    def compare_histograms(self, first: np.ndarray, second: np.ndarray) -> float:
        return float(self._processor.compare_histograms(first, second))

    def learn_reference(self, roi_frame: np.ndarray) -> None:
        self.set_reference_histogram(self.calculate_histogram(roi_frame))

    def set_reference_histogram(self, histogram: np.ndarray) -> None:
        reference = np.ascontiguousarray(histogram, dtype=np.float32)
        if reference.shape != (180,):
            raise ValueError(f"expected HSV H histogram shape (180,), got {reference.shape}")
        if not np.isfinite(reference).all():
            raise ValueError("reference histogram is empty or invalid")
        self._reference_hist = reference.copy()

    def get_reference_histogram(self) -> np.ndarray:
        if self._reference_hist is None:
            raise RuntimeError("histogram reference is not calibrated")
        return self._reference_hist.copy()

    @property
    def has_reference(self) -> bool:
        return self._reference_hist is not None

    def reset(self) -> None:
        self._reference_hist = None


class FlowSignal(Signal):

    def __init__(self, dy_threshold: float = 5.0):
        # dy_threshold: velocidad vertical mínima (px/frame) para retornar 1.0
        # El gate de fusión (thresholds.flow) se gestiona en FusionEngine, no aquí.
        self.dy_threshold = dy_threshold
        self._prev_gray: Optional[np.ndarray] = None
        self._prev_bl = None  # imagen VPI block-linear reutilizada entre frames
        self._cur_bl  = None  # buffer pre-asignado para el frame actual
        self._processor: Optional['ImageProcessor'] = None

    def compute(self, frame: np.ndarray, roi_frame: np.ndarray) -> float:
        gray = self._processor.to_grayscale(roi_frame)

        if self._prev_gray is None:
            self._prev_gray = gray
            if _OFA_AVAILABLE:
                h, w = gray.shape
                if h < 16 or w < 16:
                    # OFA exige mínimo 16px; esperar hasta que el ROI sea válido.
                    self._prev_gray = None
                    return 0.0
                self._prev_bl = _gray_to_bl(gray)
                self._cur_bl  = _vpi.Image((w, h), _vpi.Format.Y8_ER_BL)
            return 0.0

        if _OFA_AVAILABLE:
            h, w = gray.shape
            # Si las dimensiones cambiaron (p.ej. cambio de ROI), reiniciar buffers.
            if (self._cur_bl is None or self._prev_bl is None
                    or h < 16 or w < 16
                    or self._cur_bl.size != (w, h)):
                self.reset()
                return 0.0
            try:
                # Reutiliza buffers pre-asignados (sin malloc por frame)
                _gray_to_bl(gray, self._cur_bl)
                flow_magnitude = _ofa_flow_max(self._prev_bl, self._cur_bl)
                # Intercambia prev ↔ cur para el siguiente frame
                self._prev_bl, self._cur_bl = self._cur_bl, self._prev_bl
            except Exception:
                # Cualquier error VPI (p.ej. dimensión inválida en runtime):
                # resetear buffers y esperar al siguiente frame.
                self.reset()
                return 0.0
        else:
            # Fallback CPU Farneback
            flow = self._processor.calc_optical_flow(self._prev_gray, gray)
            vx = float(np.mean(flow[..., 0]))
            vy = float(np.mean(flow[..., 1]))
            flow_magnitude = max(abs(vx), abs(vy))

        self._prev_gray = gray

        if flow_magnitude > self.dy_threshold:
            return 1.0
        return 0.0

    def reset(self) -> None:
        self._prev_gray = None
        self._prev_bl = None
        self._cur_bl  = None


class BeigeSignal(Signal):
    """Detecta cobertura de color beige/crema en el ROI via mascara HSV."""

    def __init__(self, threshold: float = 0.35):
        self.threshold = threshold
        self._processor: Optional['ImageProcessor'] = None

    def compute(self, frame: np.ndarray, roi_frame: np.ndarray) -> float:
        return self._processor.beige_ratio(roi_frame)

    def reset(self) -> None:
        pass
