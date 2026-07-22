from abc import ABC, abstractmethod
from datetime import datetime
from typing import Any, Mapping, Optional

import numpy as np

from ..domain.calibration.model import StoredCalibration


class CalibrationRepository(ABC):
    @abstractmethod
    def load_active(
        self,
        *,
        line_id: str,
        config_fingerprint: str,
        algorithm_version: str,
    ) -> Optional[StoredCalibration]:
        pass

    @abstractmethod
    def save(
        self,
        *,
        line_id: str,
        device_id: str,
        config_fingerprint: str,
        algorithm_version: str,
        algorithm_parameters: Mapping[str, Any],
        histogram: np.ndarray,
        samples_used: int,
        quality_score: float,
        expires_at: datetime,
        thumbnail_jpeg: bytes | None = None,
        metadata: Mapping[str, Any] | None = None,
        calibrated_lot_code: str | None = None,
    ) -> int:
        pass
