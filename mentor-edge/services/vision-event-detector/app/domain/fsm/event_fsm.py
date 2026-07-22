# Architecture: Hexagonal
# Domain layer: no imports from adapters

from enum import Enum
from dataclasses import dataclass, field
from typing import Optional


class State(Enum):
    IDLE       = "idle"       # Esperando primer beige (calibración)
    BEIGE_IN   = "beige_in"   # Separador beige visible, esperando prenda
    EN_PRENDA  = "en_prenda"  # Prenda pasando, esperando siguiente beige
    COOLDOWN   = "cooldown"   # Pausa breve post-CORTE


@dataclass
class FSMConfig:
    # Calibración en IDLE
    calibration_frames:  int   = 20    # frames de beige sostenido para calibrar
    # Detección de beige
    beige_threshold:     float = 0.25  # beige_ratio mínimo para considerar "hay beige"
    beige_confirm_frames: int  = 10    # frames consecutivos de beige para confirmar separador
    beige_exit_frames:   int  = 8     # frames consecutivos sin beige para confirmar prenda iniciada
    # Post-CORTE
    cooldown_frames:     int   = 6
    min_rearm_s:         float = 2.0   # mínimo segundos entre CORTEs


class EventFSM:
    def __init__(self, config: FSMConfig):
        import time as _time
        self.config = config
        self._state             = State.IDLE
        self._beige_streak      = 0   # frames consecutivos con beige
        self._no_beige_streak   = 0   # frames consecutivos sin beige
        self._calibration_count = 0   # frames acumulados en calibración IDLE
        self._cooldown_counter  = 0
        self._last_corte_time   = -999.0
        self._time              = _time

    def process(self, fusion_score: float, beige_ratio: float) -> Optional[str]:
        """
        Retorna: None | 'CALIBRATE' | 'CORTE'

        Lógica de conteo:
          Panel: [BEIGE][PRENDA1][BEIGE][PRENDA2][BEIGE]...[PRENDAN][BEIGE]
          Total prendas = total separadores beige - 1
          Cada vez que reaparece beige después de una prenda → CORTE (1 prenda contada)
        """
        has_beige = beige_ratio >= self.config.beige_threshold

        if self._state == State.IDLE:
            return self._handle_idle(has_beige)
        if self._state == State.BEIGE_IN:
            return self._handle_beige_in(has_beige)
        if self._state == State.EN_PRENDA:
            return self._handle_en_prenda(has_beige)
        if self._state == State.COOLDOWN:
            return self._handle_cooldown()
        return None

    def _handle_idle(self, has_beige: bool) -> Optional[str]:
        """Acumula frames de beige para calibración dinámica."""
        if has_beige:
            self._calibration_count += 1
            if self._calibration_count >= self.config.calibration_frames:
                # Beige calibrado — transicionar a BEIGE_IN y pedir calibración al servicio
                self._state = State.BEIGE_IN
                self._beige_streak = 0
                self._no_beige_streak = 0
                self._calibration_count = 0
                return 'CALIBRATE'
        else:
            self._calibration_count = 0
        return None

    def _handle_beige_in(self, has_beige: bool) -> Optional[str]:
        """Beige visible. Espera que desaparezca para confirmar inicio de prenda."""
        if not has_beige:
            self._no_beige_streak += 1
            self._beige_streak = 0
            if self._no_beige_streak >= self.config.beige_exit_frames:
                # La prenda empezó a pasar
                self._state = State.EN_PRENDA
                self._no_beige_streak = 0
        else:
            self._beige_streak += 1
            self._no_beige_streak = 0
        return None

    def _handle_en_prenda(self, has_beige: bool) -> Optional[str]:
        """Prenda pasando. Espera que reaparezca el beige separador → CORTE."""
        if has_beige:
            self._beige_streak += 1
            self._no_beige_streak = 0
            if self._beige_streak >= self.config.beige_confirm_frames:
                # Siguiente separador beige confirmado → 1 prenda completada
                now = self._time.time()
                if (now - self._last_corte_time) < self.config.min_rearm_s:
                    # Demasiado rápido — volver a BEIGE_IN sin emitir
                    self._state = State.BEIGE_IN
                    self._beige_streak = 0
                    return None
                self._last_corte_time = now
                self._state = State.COOLDOWN
                self._cooldown_counter = 0
                self._beige_streak = 0
                return 'CORTE'
        else:
            self._no_beige_streak += 1
            self._beige_streak = 0
        return None

    def _handle_cooldown(self) -> Optional[str]:
        self._cooldown_counter += 1
        if self._cooldown_counter >= self.config.cooldown_frames:
            # Volver a BEIGE_IN porque el separador sigue siendo visible post-CORTE
            self._state = State.BEIGE_IN
            self._cooldown_counter = 0
            self._beige_streak = 0
            self._no_beige_streak = 0
        return None

    def reset(self) -> None:
        self._state             = State.IDLE
        self._beige_streak      = 0
        self._no_beige_streak   = 0
        self._calibration_count = 0
        self._cooldown_counter  = 0
        self._last_corte_time   = -999.0

    @property
    def state(self) -> State:
        return self._state
