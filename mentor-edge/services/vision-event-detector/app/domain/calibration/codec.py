import io
from typing import Sequence

import numpy as np


def serialize_histogram(histogram: np.ndarray) -> tuple[bytes, str, list[int]]:
    """Serialize a numeric histogram as a non-pickle NPY payload."""
    array = np.ascontiguousarray(histogram, dtype=np.float32)
    if array.size == 0:
        raise ValueError("histogram cannot be empty")
    if not np.isfinite(array).all():
        raise ValueError("histogram contains non-finite values")

    buffer = io.BytesIO()
    np.save(buffer, array, allow_pickle=False)
    return buffer.getvalue(), array.dtype.name, list(array.shape)


def deserialize_histogram(
    payload: bytes | memoryview,
    stored_dtype: str,
    stored_shape: Sequence[int],
) -> np.ndarray:
    """Restore and validate a histogram persisted in BYTEA."""
    with io.BytesIO(bytes(payload)) as buffer:
        array = np.load(buffer, allow_pickle=False)

    if array.dtype.hasobject:
        raise ValueError("object arrays are not valid calibration histograms")
    if array.dtype.name != stored_dtype:
        raise ValueError("stored histogram dtype does not match NPY payload")
    if tuple(array.shape) != tuple(stored_shape):
        raise ValueError("stored histogram shape does not match NPY payload")
    if not np.isfinite(array).all():
        raise ValueError("stored histogram contains non-finite values")
    return np.ascontiguousarray(array, dtype=np.float32)
