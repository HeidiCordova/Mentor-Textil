import time
from typing import Optional


class Watchdog:

    def __init__(self, timeout_seconds: float = 10.0):
        self.timeout = timeout_seconds
        self._last_frame_time: Optional[float] = None
        self._error_count = 0

    def update(self) -> None:
        self._last_frame_time = time.monotonic()

    def check(self) -> bool:
        if self._last_frame_time is None:
            return True
        elapsed = time.monotonic() - self._last_frame_time
        if elapsed > self.timeout:
            self._error_count += 1
            return False
        return True

    def reset(self) -> None:
        self._last_frame_time = None
        self._error_count = 0

    @property
    def error_count(self) -> int:
        return self._error_count
