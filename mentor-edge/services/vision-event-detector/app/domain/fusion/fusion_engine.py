from dataclasses import dataclass
from typing import Dict

# Pesos para detección textil: con dy_threshold bajo (1.0) el flow funciona,
# color (histograma) detecta bien el producto, beige puede ser 0 si el
# producto no es crema → no depender de beige.
TEXTILE_WEIGHTS: Dict[str, float] = {
    'edge':  0.25,
    'color': 0.30,
    'flow':  0.35,
    'beige': 0.10,
}

# Umbral mínimo de flow para que la fusión produzca score > 0.
# Configurable via thresholds.flow desde la UI/config-service.
_DEFAULT_FLOW_GATE: float = 0.5


@dataclass
class SignalValues:
    edge: float
    color: float
    flow: float
    beige: float = 0.0


class FusionEngine:
    def __init__(self, weights: Dict[str, float] = None, flow_gate: float = _DEFAULT_FLOW_GATE):
        self.weights   = dict(weights) if weights else dict(TEXTILE_WEIGHTS)
        self.flow_gate = flow_gate

    def fuse(self, signals: SignalValues) -> float:
        if signals.flow < self.flow_gate:
            return 0.0
        score = (
            signals.edge  * self.weights['edge']  +
            signals.color * self.weights['color'] +
            signals.flow  * self.weights['flow']  +
            signals.beige * self.weights['beige']
        )
        return min(1.0, max(0.0, score))

    def update_weights(self, weights: Dict[str, float]) -> None:
        total = sum(weights.values())
        if abs(total - 1.0) > 0.01:
            raise ValueError('Weights must sum to 1.0')
        self.weights = dict(weights)
