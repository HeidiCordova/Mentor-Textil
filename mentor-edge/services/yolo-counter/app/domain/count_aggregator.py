import time
import logging
from typing import Dict, Any, Optional

logger = logging.getLogger('yolo.oee')


class CountAggregator:
    """Accumulates counts per OEE snapshot window aligned to wall-clock."""

    def __init__(self, interval_s: float = 300.0, device_id: str = '', line_code: str = ''):
        self._interval_s = interval_s
        self._device_id = device_id
        self._line_code = line_code
        self._count = 0
        self._window_start = self._align_now()

    def _align_now(self) -> float:
        now = time.time()
        return now - (now % self._interval_s)

    def add(self, n: int):
        self._count += n

    def should_emit(self) -> bool:
        return time.time() >= self._window_start + self._interval_s

    def emit(self) -> Optional[Dict[str, Any]]:
        if not self.should_emit():
            return None
        record = {
            'code': self._line_code,
            'time': int(self._window_start * 1000 + self._interval_s * 1000),
            'device_id': self._device_id,
            'interval_s': int(self._interval_s),
            'v': 1,
            'head': ['T_DISPONIBLE', 'T_MICROPARADA', 'T_PARADA_NO_ASIGNADA', 'CONTEO_1', 'CONTEO_2'],
            'data': [0, 0, 0, self._count, 0],
        }
        self._count = 0
        self._window_start = self._align_now()
        return record

    def update_config(self, interval_s: float = None, line_code: str = None, device_id: str = None):
        if interval_s is not None:
            self._interval_s = interval_s
        if line_code is not None:
            self._line_code = line_code
        if device_id is not None:
            self._device_id = device_id
