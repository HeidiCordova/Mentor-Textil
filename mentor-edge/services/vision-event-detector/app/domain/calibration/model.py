from dataclasses import dataclass
from datetime import datetime
from typing import Any, Mapping, Optional

import numpy as np


HISTOGRAM_ALGORITHM_VERSION = "hsv-h180-l2-correlation-v1"
HISTOGRAM_ALGORITHM_PARAMETERS = {
    "color_space": "HSV",
    "channels": [0],
    "bins": [180],
    "ranges": [0, 180],
    "normalization": "L2",
    "comparison_method": "CORREL",
}


@dataclass(frozen=True)
class StoredCalibration:
    calibration_id: int
    line_id: str
    device_id: str
    config_fingerprint: str
    algorithm_version: str
    histogram: np.ndarray
    samples_used: int
    quality_score: float
    calibrated_at: datetime
    expires_at: datetime
    metadata: Mapping[str, Any]


@dataclass(frozen=True)
class CalibrationResult:
    histogram: np.ndarray
    samples_used: int
    quality_score: float
    reference_frame: Optional[np.ndarray] = None
