from abc import ABC, abstractmethod
import numpy as np


class ImageProcessor(ABC):

    @abstractmethod
    def to_grayscale(self, frame: np.ndarray) -> np.ndarray:
        pass

    @abstractmethod
    def detect_edges(self, gray: np.ndarray, low: int, high: int) -> np.ndarray:
        pass

    @abstractmethod
    def to_hsv(self, frame: np.ndarray) -> np.ndarray:
        pass

    @abstractmethod
    def calc_histogram(self, hsv: np.ndarray) -> np.ndarray:
        pass

    @abstractmethod
    def compare_histograms(self, h1: np.ndarray, h2: np.ndarray) -> float:
        pass

    @abstractmethod
    def calc_optical_flow(self, prev_gray: np.ndarray, gray: np.ndarray) -> np.ndarray:
        pass

    @abstractmethod
    def edges_density(self, edges: np.ndarray) -> float:
        """Fraccion de pixeles activos sobre el total."""
        pass

    @abstractmethod
    def vertical_mean(self, flow: np.ndarray) -> float:
        """Media del componente vertical del flujo optico."""
        pass

    @abstractmethod
    def beige_ratio(self, frame: np.ndarray) -> float:
        """Fraccion de pixeles beige/crema sobre el total del frame."""
        pass
