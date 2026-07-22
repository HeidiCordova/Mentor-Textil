from typing import List, Any, Optional
import numpy as np

from .model import CalibrationResult


class Calibrator:

    def __init__(self, required_samples: int = 30):
        if required_samples <= 0:
            raise ValueError("required_samples must be greater than zero")
        self._samples: List[np.ndarray] = []
        self._is_calibrating = False
        self._required_samples = int(required_samples)

    def start(self) -> None:
        self._samples.clear()
        self._is_calibrating = True

    def add_sample(self, roi_frame: np.ndarray) -> None:
        if not self._is_calibrating:
            return
        if len(self._samples) < self._required_samples:
            self._samples.append(roi_frame.copy())

    def finish(self, histogram_signal: Any) -> Optional[CalibrationResult]:
        if not self._is_calibrating:
            return None
        if len(self._samples) < self._required_samples:
            self._is_calibrating = False
            self._samples.clear()
            return None

        try:
            histograms = np.stack([
                histogram_signal.calculate_histogram(sample)
                for sample in self._samples
            ]).astype(np.float32, copy=False)

            # Median is robust against a few blurred, shadowed or partially
            # occluded samples. Normalize it exactly as the live histograms.
            reference = np.median(histograms, axis=0).astype(np.float32)
            norm = float(np.linalg.norm(reference))
            if not np.isfinite(norm) or norm <= 0.0:
                raise ValueError("calibration produced an invalid histogram")
            reference /= norm

            correlations = [
                histogram_signal.compare_histograms(reference, sample_hist)
                for sample_hist in histograms
            ]
            quality = float(np.mean(np.clip(correlations, 0.0, 1.0)))
            reference_frame = self._samples[len(self._samples) // 2].copy()

            return CalibrationResult(
                histogram=reference,
                samples_used=len(self._samples),
                quality_score=quality,
                reference_frame=reference_frame,
            )
        finally:
            self._samples.clear()
            self._is_calibrating = False
    
    @property
    def is_active(self) -> bool:
        return self._is_calibrating
    
    @property
    def progress(self) -> float:
        if not self._is_calibrating:
            return 0.0
        return len(self._samples) / self._required_samples

    @property
    def required_samples(self) -> int:
        return self._required_samples

    @property
    def samples_collected(self) -> int:
        return len(self._samples)
