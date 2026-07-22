import cv2
import logging
import numpy as np
from typing import Optional
from ..ports.frame_input import FrameInput

logger = logging.getLogger('yolo.camera')


class OpenCVAdapter(FrameInput):

    def __init__(self, url: str):
        self._url = url
        self._cap: Optional[cv2.VideoCapture] = None
        self._connect()

    def _connect(self):
        if self._cap is not None:
            self._cap.release()
        self._cap = cv2.VideoCapture(self._url, cv2.CAP_FFMPEG)
        if self._cap.isOpened():
            logger.info('camera connected: %s', self._url)
        else:
            logger.warning('camera open failed: %s', self._url)

    def get_frame(self) -> Optional[np.ndarray]:
        if self._cap is None or not self._cap.isOpened():
            return None
        ok, frame = self._cap.read()
        return frame if ok else None

    def is_connected(self) -> bool:
        return self._cap is not None and self._cap.isOpened()

    def reconnect(self) -> bool:
        self._connect()
        return self.is_connected()

    def update_url(self, url: str) -> bool:
        self._url = url
        self._connect()
        return self.is_connected()
