import unittest

import numpy as np

from app.adapters.cv_image_processor import CVImageProcessor
from app.domain.signals.signal_extractors import MeshSignal


def _solid(value: int = 180, size: int = 200) -> np.ndarray:
    """Tela solida: superficie plana sin huecos → casi sin bordes."""
    return np.full((size, size, 3), value, dtype=np.uint8)


def _mesh(size: int = 200, step: int = 6, hole: int = 3) -> np.ndarray:
    """Separador de malla: fondo claro con una rejilla regular de vacios
    oscuros (los huecos del dralon) → densidad de bordes alta."""
    img = np.full((size, size, 3), 200, dtype=np.uint8)
    for y in range(0, size, step):
        for x in range(0, size, step):
            img[y:y + hole, x:x + hole] = 30  # hueco oscuro
    return img


class MeshSignalTests(unittest.TestCase):
    def setUp(self):
        self.processor = CVImageProcessor()
        self.signal = MeshSignal()
        self.signal.set_image_processor(self.processor)

    def test_mesh_scores_high(self):
        score = self.signal.compute(None, _mesh())
        self.assertGreater(score, 0.5, f"la malla deberia puntuar alto, dio {score}")

    def test_solid_scores_low(self):
        score = self.signal.compute(None, _solid())
        self.assertLess(score, 0.1, f"la tela solida deberia puntuar bajo, dio {score}")

    def test_mesh_separates_from_solid_regardless_of_color(self):
        # Mismo patron de malla pero sobre fondo oscuro (otro "tono"):
        # la textura de huecos sigue disparando alto.
        dark_mesh = _mesh()
        dark_mesh[dark_mesh[:, :, 0] == 200] = 90  # baja el brillo del fondo
        mesh_score = self.signal.compute(None, dark_mesh)
        solid_score = self.signal.compute(None, _solid(90))
        self.assertGreater(mesh_score, solid_score + 0.4)

    def test_score_is_bounded(self):
        for frame in (_solid(), _mesh(), _solid(0), _solid(255)):
            score = self.signal.compute(None, frame)
            self.assertGreaterEqual(score, 0.0)
            self.assertLessEqual(score, 1.0)

    def test_edge_ref_controls_normalization(self):
        loose = MeshSignal(edge_ref=0.5)
        loose.set_image_processor(self.processor)
        tight = MeshSignal(edge_ref=0.01)
        tight.set_image_processor(self.processor)
        frame = _mesh()
        # Un edge_ref menor normaliza mas agresivo → score mayor o igual.
        self.assertGreaterEqual(tight.compute(None, frame), loose.compute(None, frame))


if __name__ == '__main__':
    unittest.main()
