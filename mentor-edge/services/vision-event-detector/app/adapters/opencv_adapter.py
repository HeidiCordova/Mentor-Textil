import cv2
import time
import logging
import threading
import numpy as np
from typing import Optional
from ..ports.frame_input import FrameInput

logger = logging.getLogger('detector.camera')


class OpenCVAdapter(FrameInput):

    MAX_RECONNECT_DELAY = 30.0
    INITIAL_RECONNECT_DELAY = 0.5
    MAX_NULL_FRAMES = 15

    def __init__(self, camera_url: str, reconnect_attempts: int = 10):
        self.camera_url = camera_url
        self.reconnect_attempts = reconnect_attempts
        self._cap: Optional[cv2.VideoCapture] = None
        self._consecutive_failures = 0
        self._frame: Optional[np.ndarray] = None
        self._lock = threading.Lock()
        self._running = False
        self._thread: Optional[threading.Thread] = None
        self._connect()

    def _apply_rtsp_opts(self) -> None:
        if self._cap is None:
            return
        is_rtsp = isinstance(self.camera_url, str) and self.camera_url.lower().startswith('rtsp')
        if is_rtsp:
            self._cap.set(cv2.CAP_PROP_BUFFERSIZE, 1)

    def _connect(self) -> bool:
        self._stop_reader()
        if self._cap is not None:
            self._cap.release()
            self._cap = None

        try:
            src = int(self.camera_url) if self.camera_url.isdigit() else self.camera_url
        except (ValueError, AttributeError):
            src = self.camera_url

        self._cap = cv2.VideoCapture(src)
        self._apply_rtsp_opts()

        if self._cap.isOpened():
            self._consecutive_failures = 0
            self._start_reader()
            logger.info('Camera connected: %s', self.camera_url)
            return True

        logger.warning('Camera open failed: %s', self.camera_url)
        return False

    def _start_reader(self) -> None:
        self._running = True
        self._thread = threading.Thread(target=self._read_loop, daemon=True)
        self._thread.start()

    def _stop_reader(self) -> None:
        self._running = False
        if self._thread is not None:
            self._thread.join(timeout=3.0)
            self._thread = None

    def _read_loop(self) -> None:
        while self._running and self._cap is not None and self._cap.isOpened():
            ret, frame = self._cap.read()
            if ret:
                with self._lock:
                    self._frame = frame
                    self._consecutive_failures = 0
            else:
                with self._lock:
                    self._consecutive_failures += 1
                time.sleep(0.005)

    def get_frame(self) -> Optional[np.ndarray]:
        with self._lock:
            frame = self._frame
            self._frame = None
        return frame

    def is_connected(self) -> bool:
        if self._cap is None or not self._cap.isOpened():
            return False
        with self._lock:
            return self._consecutive_failures < self.MAX_NULL_FRAMES

    def reconnect(self) -> bool:
        self._stop_reader()
        if self._cap is not None:
            self._cap.release()
            self._cap = None

        delay = self.INITIAL_RECONNECT_DELAY
        for attempt in range(1, self.reconnect_attempts + 1):
            logger.info(
                'Reconnect attempt %d/%d (delay %.1fs)',
                attempt, self.reconnect_attempts, delay
            )
            time.sleep(delay)
            if self._connect():
                return True
            delay = min(delay * 2.0, self.MAX_RECONNECT_DELAY)

        logger.error(
            'All %d reconnect attempts exhausted for %s',
            self.reconnect_attempts, self.camera_url
        )
        return False

    def update_url(self, url: str) -> bool:
        self._stop_reader()
        if self._cap is not None:
            self._cap.release()
            self._cap = None
        self.camera_url = url
        return self._connect()

    def __del__(self):
        self._stop_reader()
        if self._cap is not None:
            self._cap.release()
