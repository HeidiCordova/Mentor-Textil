# Architecture: Hexagonal
# Domain layer: no imports from adapters

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional, Tuple
import numpy as np

@dataclass
class ROI:
    x: int
    y: int
    width: int
    height: int
    bottom_margin: int = 0

    def extract(self, frame: np.ndarray) -> np.ndarray:
        h, w = frame.shape[:2]
        x0 = max(0, self.x)
        y0 = max(0, self.y)
        x1 = min(w, self.x + self.width)
        y1 = min(h, self.y + self.height - max(0, self.bottom_margin))
        if x1 <= x0 or y1 <= y0:
            # ROI inválido o fuera del frame: devuelve recorte vacío seguro.
            return frame[0:0, 0:0]
        return frame[y0:y1, x0:x1]

    def contains(self, point: Tuple[int, int]) -> bool:
        px, py = point
        return (self.x <= px < self.x + self.width and 
                self.y <= py < self.y + self.height)
    
    def to_dict(self) -> dict:
        return {
            'x': self.x,
            'y': self.y,
            'width': self.width,
            'height': self.height
        }

class ROIManager:
    def __init__(self, roi: ROI):
        self._roi = roi
        self._active = True
    
    @property
    def roi(self) -> ROI:
        return self._roi
    
    def update(self, roi: ROI) -> None:
        self._roi = roi
    
    def activate(self) -> None:
        self._active = True
    
    def deactivate(self) -> None:
        self._active = False
    
    @property
    def active(self) -> bool:
        return self._active
