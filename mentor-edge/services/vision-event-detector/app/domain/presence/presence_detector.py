from collections import deque
from typing import Optional

import numpy as np


class PresenceDetector:
    """
    Detector de presencia por SLOW-WINDOW DIFF con latch asimétrico.

    Diseñado para máquinas textiles donde una prenda tarda ~40 min:
    a 12.5 fps el desplazamiento entre frames consecutivos es sub-pixel
    (≈ 0.016 px/frame), invisible para diff frame-a-frame.

    Solución: comparar frame[N] vs frame[N - window_frames] (ventana lenta).
    En 30 s (375 frames @ 12.5 fps) la tela habrá desplazado varios píxeles
    → diff detectable y confiable con pixel_threshold bajo (8).

    Latch asimétrico:
        motion event → present = True  (inmediato)
        exit_hold frames consecutivos sin movimiento → present = False
        (exit_hold largo = 750 frames ≈ 60 s para no cortar por vibración)

    Parámetros:
        pixel_threshold:  intensidad mínima de cambio por píxel (0-255). Default: 8.
        motion_area:      fracción mínima del ROI con píxeles cambiados. Default: 0.005.
        exit_hold:        frames consecutivos sin movimiento para salir. Default: 750.
        scale:            factor de downscale antes del diff (reduce memoria). Default: 0.25.
        window_frames:    ventana de comparación en frames. Default: 375 (≈ 30 s @ 12.5 fps).
    """

    def __init__(
        self,
        pixel_threshold: float = 8.0,
        motion_area: float = 0.005,
        exit_hold: int = 750,
        scale: float = 0.25,
        window_frames: int = 375,
    ) -> None:
        self._pixel_threshold = max(1.0, float(pixel_threshold))
        self._motion_area     = max(0.001, min(1.0, float(motion_area)))
        self._exit_hold       = max(1, int(exit_hold))
        self._scale           = max(0.1, min(1.0, float(scale)))
        self._window_frames   = max(1, int(window_frames))

        # Buffer circular: almacena los últimos window_frames+1 frames preprocesados.
        # frame_buffer[0] = frame más antiguo (= referencia de comparación)
        # frame_buffer[-1] = frame más reciente (appended justo ahora)
        self._frame_buffer: deque = deque(maxlen=self._window_frames + 1)
        self._is_present   = False
        self._still_count  = 0      # frames consecutivos sin movimiento
        self._motion_score = 0.0    # fracción de píxeles que cambiaron (0-1)

    # ─── API pública ─────────────────────────────────────────────────────────

    @property
    def is_present(self) -> bool:
        return self._is_present

    @property
    def motion_detected(self) -> bool:
        """Alias de is_present — compat con código existente."""
        return self._is_present

    @property
    def presence_score(self) -> float:
        """Fracción de píxeles que cambiaron respecto al frame anterior (0-1)."""
        return self._motion_score

    # Shims de compat con versiones anteriores (bg-subtraction / chroma)
    @property
    def is_learning(self) -> bool:
        return False

    @property
    def bg_progress(self) -> float:
        return 1.0

    @property
    def color_score(self) -> float:
        return self._motion_score

    def update(self, roi_frame: np.ndarray) -> bool:
        small = self._preprocess(roi_frame)

        # Acumular en buffer circular (maxlen = window_frames+1, auto-descarta el más viejo)
        self._frame_buffer.append(small)

        # Esperar hasta tener suficientes frames para la ventana completa
        if len(self._frame_buffer) <= self._window_frames:
            return self._is_present

        # ── Slow-window diff: actual vs frame de window_frames atrás ─────────
        # frame_buffer[0] es exactamente el frame de window_frames atrás
        # (deque maxlen garantiza que tiene window_frames+1 entradas cuando lleno)
        reference = self._frame_buffer[0]
        diff = np.abs(small.astype(np.int16) - reference.astype(np.int16))
        self._motion_score = float(np.mean(diff > self._pixel_threshold))

        motion = self._motion_score > self._motion_area

        # ── Latch asimétrico ──────────────────────────────────────────────────
        if motion:
            self._is_present  = True
            self._still_count = 0
        elif self._is_present:
            self._still_count += 1
            if self._still_count >= self._exit_hold:
                self._is_present  = False
                self._still_count = 0

        return self._is_present

    @property
    def motion_now(self) -> bool:
        """Movimiento RAW sin latch ni exit_hold — para stop_tracker.

        Devuelve True solo cuando el slow-window diff detecta movimiento EN ESTE
        frame, sin el latch asimétrico.  Esto hace que stop_tracker responda en
        ~0.24 s (3 idle_debounce_ticks) después de que la máquina para, en lugar
        de los ~90 s que impone exit_hold=750 + ventana de 30 s.

        El latch largo (is_present / motion_detected) se sigue usando para OEE:
        mantiene el bucket de producción activo entre los bursts de detección.
        """
        return (
            len(self._frame_buffer) > self._window_frames
            and self._motion_score > self._motion_area
        )

    @property
    def is_warmed_up(self) -> bool:
        """True cuando el buffer slow-window está lleno y puede detectar movimiento."""
        return len(self._frame_buffer) > self._window_frames

    def reset(self) -> None:
        """Reinicia el detector (útil al cambiar de ROI o parámetros)."""
        self._frame_buffer.clear()
        self._is_present   = False
        self._still_count  = 0
        self._motion_score = 0.0

    def reset_background(self) -> None:
        """Alias de reset() — compat con versión anterior."""
        self.reset()

    # ─── Setters configurables ───────────────────────────────────────────────

    @property
    def pixel_threshold(self) -> float:
        return self._pixel_threshold

    @pixel_threshold.setter
    def pixel_threshold(self, value: float) -> None:
        self._pixel_threshold = max(1.0, float(value))

    @property
    def threshold(self) -> float:         # compat alias
        return self._pixel_threshold

    @threshold.setter
    def threshold(self, value: float) -> None:
        self._pixel_threshold = max(1.0, float(value))

    @property
    def motion_area(self) -> float:
        return self._motion_area

    @motion_area.setter
    def motion_area(self, value: float) -> None:
        self._motion_area = max(0.001, min(1.0, float(value)))

    @property
    def area_threshold(self) -> float:    # compat alias
        return self._motion_area

    @area_threshold.setter
    def area_threshold(self, value: float) -> None:
        self._motion_area = max(0.001, min(1.0, float(value)))

    @property
    def exit_hold(self) -> int:
        return self._exit_hold

    @exit_hold.setter
    def exit_hold(self, value: int) -> None:
        self._exit_hold = max(1, int(value))

    @property
    def window_frames(self) -> int:
        return self._window_frames

    @window_frames.setter
    def window_frames(self, value: int) -> None:
        new_val = max(1, int(value))
        if new_val != self._window_frames:
            self._window_frames = new_val
            self._frame_buffer = deque(maxlen=new_val + 1)

    # No-ops compat (sat_threshold / reference_hold / bg_frames)
    @property
    def sat_threshold(self) -> float:
        return 0.0

    @sat_threshold.setter
    def sat_threshold(self, value: float) -> None:
        pass

    @property
    def reference_hold(self) -> int:
        return self._exit_hold

    @reference_hold.setter
    def reference_hold(self, value: int) -> None:
        self._exit_hold = max(1, int(value))

    # ─── helper ──────────────────────────────────────────────────────────────

    def _preprocess(self, roi_frame: np.ndarray) -> np.ndarray:
        gray = (np.mean(roi_frame, axis=2).astype(np.uint8)
                if roi_frame.ndim == 3 else roi_frame)
        if self._scale < 1.0:
            h, w = gray.shape
            bh = max(1, int(1.0 / self._scale))
            bw = max(1, int(1.0 / self._scale))
            nh = max(1, h // bh)
            nw = max(1, w // bw)
            gray = (gray[:nh * bh, :nw * bw]
                    .reshape(nh, bh, nw, bw)
                    .mean(axis=(1, 3))
                    .astype(np.uint8))
        return gray
