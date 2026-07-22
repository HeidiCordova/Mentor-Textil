from datetime import datetime
from typing import Any, Mapping, Optional

import numpy as np
import psycopg2
from psycopg2 import Binary
from psycopg2.extras import Json, RealDictCursor

from ..domain.calibration.codec import deserialize_histogram, serialize_histogram
from ..domain.calibration.model import StoredCalibration
from ..ports.calibration_repository import CalibrationRepository


class PostgresCalibrationRepository(CalibrationRepository):
    def __init__(self, dsn: str, connect_timeout_s: int = 5):
        # An empty DSN is valid: libpq then reads PGHOST, PGPORT, PGDATABASE,
        # PGUSER and PGPASSWORD. This avoids constructing a URL with secrets.
        self._dsn = dsn or ""
        self._connect_timeout_s = max(1, int(connect_timeout_s))

    def _connect(self):
        return psycopg2.connect(
            self._dsn,
            connect_timeout=self._connect_timeout_s,
            application_name="vision-event-detector",
        )

    def load_active(
        self,
        *,
        line_id: str,
        config_fingerprint: str,
        algorithm_version: str,
    ) -> Optional[StoredCalibration]:
        conn = self._connect()
        try:
            with conn:
                with conn.cursor(cursor_factory=RealDictCursor) as cursor:
                    cursor.execute(
                        """
                        UPDATE public.detector_calibration
                           SET active = FALSE,
                               deactivated_at = CURRENT_TIMESTAMP,
                               deactivation_reason = 'expired'
                         WHERE line_id = %s
                           AND active = TRUE
                           AND expires_at <= CURRENT_TIMESTAMP
                        """,
                        (line_id,),
                    )
                    cursor.execute(
                        """
                        SELECT id, line_id, device_id, config_fingerprint,
                               algorithm_version, histogram_data,
                               histogram_dtype, histogram_shape, samples_used,
                               quality_score, calibrated_at, expires_at, metadata
                          FROM public.detector_calibration
                         WHERE line_id = %s
                           AND config_fingerprint = %s
                           AND algorithm_version = %s
                           AND histogram_format = 'npy-v1'
                           AND active = TRUE
                           AND expires_at > CURRENT_TIMESTAMP
                         ORDER BY calibrated_at DESC
                         LIMIT 1
                        """,
                        (line_id, config_fingerprint, algorithm_version),
                    )
                    row = cursor.fetchone()
        finally:
            conn.close()

        if row is None:
            return None

        histogram = deserialize_histogram(
            row["histogram_data"],
            row["histogram_dtype"],
            row["histogram_shape"],
        )
        return StoredCalibration(
            calibration_id=int(row["id"]),
            line_id=row["line_id"],
            device_id=row["device_id"],
            config_fingerprint=row["config_fingerprint"].strip(),
            algorithm_version=row["algorithm_version"],
            histogram=histogram,
            samples_used=int(row["samples_used"]),
            quality_score=float(row["quality_score"]),
            calibrated_at=row["calibrated_at"],
            expires_at=row["expires_at"],
            metadata=row["metadata"] or {},
        )

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
        if samples_used <= 0:
            raise ValueError("samples_used must be greater than zero")
        if not 0.0 <= quality_score <= 1.0:
            raise ValueError("quality_score must be between zero and one")
        if expires_at.tzinfo is None or expires_at.utcoffset() is None:
            raise ValueError("expires_at must be timezone-aware")
        if thumbnail_jpeg is not None and len(thumbnail_jpeg) > 262144:
            raise ValueError("thumbnail cannot exceed 256 KiB")

        payload, dtype_name, shape = serialize_histogram(histogram)
        conn = self._connect()
        try:
            with conn:
                with conn.cursor() as cursor:
                    # The update and insert are one transaction. A failed insert
                    # restores the previous active calibration automatically.
                    cursor.execute(
                        """
                        UPDATE public.detector_calibration
                           SET active = FALSE,
                               deactivated_at = CURRENT_TIMESTAMP,
                               deactivation_reason = 'replaced'
                         WHERE line_id = %s
                           AND active = TRUE
                        """,
                        (line_id,),
                    )
                    cursor.execute(
                        """
                        INSERT INTO public.detector_calibration (
                            line_id, device_id, calibrated_lot_code,
                            config_fingerprint, algorithm_version,
                            algorithm_parameters, histogram_data,
                            histogram_format, histogram_dtype, histogram_shape,
                            samples_used, quality_score, expires_at, active,
                            reference_thumbnail_jpeg, metadata
                        ) VALUES (
                            %s, %s, %s, %s, %s, %s,
                            %s, 'npy-v1', %s, %s, %s, %s, %s, TRUE, %s, %s
                        )
                        RETURNING id
                        """,
                        (
                            line_id,
                            device_id,
                            calibrated_lot_code,
                            config_fingerprint,
                            algorithm_version,
                            Json(dict(algorithm_parameters)),
                            Binary(payload),
                            dtype_name,
                            shape,
                            samples_used,
                            quality_score,
                            expires_at,
                            Binary(thumbnail_jpeg) if thumbnail_jpeg else None,
                            Json(dict(metadata or {})),
                        ),
                    )
                    calibration_id = int(cursor.fetchone()[0])
        finally:
            conn.close()
        return calibration_id
