import numpy as np
from typing import Tuple


class ROIManager:

    def __init__(self, x: int, y: int, w: int, h: int):
        self.x = x
        self.y = y
        self.w = w
        self.h = h

    def extract(self, frame: np.ndarray) -> np.ndarray:
        return frame[self.y:self.y + self.h, self.x:self.x + self.w]

    def update(self, x: int, y: int, w: int, h: int):
        self.x, self.y, self.w, self.h = x, y, w, h

    @property
    def as_tuple(self) -> Tuple[int, int, int, int]:
        return (self.x, self.y, self.w, self.h)
