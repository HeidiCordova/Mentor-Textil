from abc import ABC, abstractmethod
from typing import Optional
import numpy as np


class FrameInput(ABC):

    @abstractmethod
    def get_frame(self) -> Optional[np.ndarray]:
        pass

    @abstractmethod
    def is_connected(self) -> bool:
        pass

    @abstractmethod
    def reconnect(self) -> bool:
        pass

    @abstractmethod
    def update_url(self, url: str) -> bool:
        pass
