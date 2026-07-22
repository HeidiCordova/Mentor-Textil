"""Automatic stop detection state machine.

Observes a boolean ``is_producing`` signal every tick and decides when to
open / close stops in the edge-gateway.

Classification:
    idle < micro_stop_max_s  →  MICROPARADA  (created retroactively when production resumes)
    idle >= micro_stop_max_s →  PARADA_NO_ASIGNADA  (opened live, closed when production resumes)

The tracker emits *actions* (pure data) instead of making HTTP calls itself,
keeping this module free of I/O and fully unit-testable.
"""

import time
from dataclasses import dataclass
from datetime import datetime, timezone
from enum import Enum
from typing import List, Optional


class _State(Enum):
    PRODUCING = "producing"
    IDLE_WAIT = "idle_wait"
    STOP_OPEN = "stop_open"


@dataclass
class StopAction:
    """An action the caller must execute against the edge-gateway."""
    kind: str               # "ABRIR_PARADA" | "CERRAR_PARADA" | "CREAR_MICROPARADA"
    started_at: datetime
    ended_at: Optional[datetime] = None   # set for CERRAR / MICROPARADA
    stop_id: Optional[str] = None         # set for CERRAR


class StopTracker:
    """Pure-domain state machine — no I/O.

    Call :meth:`update` on every processed frame.  It returns a (possibly
    empty) list of :class:`StopAction` that the caller must dispatch.

    Parameters
    ----------
    micro_stop_max_s:
        Seconds of idle time before a micro-stop becomes a formal stop.
    idle_debounce_ticks:
        Consecutive idle ticks required to leave PRODUCING (entry debounce).
    resume_debounce_ticks:
        Consecutive producing ticks required to close an open stop (exit
        debounce).  Prevents a single noisy frame from fragmenting a long
        stop into two records.  Matches JS ``_MOTION_RESET_THR = 12``.
    brief_gap_s:
        Idle periods shorter than this (seconds) are silently discarded and
        never recorded as micro-stops.  Matches JS ``_LOG_BRIEF_S = 3``.
    """

    def __init__(
        self,
        micro_stop_max_s: float = 120.0,
        idle_debounce_ticks: int = 3,
        resume_debounce_ticks: int = 6,   # 6 ticks × ~80ms = ~480ms (antes 12 × 80ms = 960ms)
        brief_gap_s: float = 3.0,
    ) -> None:
        self._micro_stop_max_s = micro_stop_max_s
        self._idle_debounce_ticks = max(1, idle_debounce_ticks)
        self._resume_debounce_ticks = max(1, resume_debounce_ticks)
        self._brief_gap_s = brief_gap_s

        self._state = _State.PRODUCING
        self._idle_streak = 0           # consecutive idle ticks (entry debounce)
        self._resume_streak = 0         # consecutive producing ticks (exit debounce)
        self._idle_start_mono: float = 0.0
        self._idle_start_wall: Optional[datetime] = None
        self._first_idle_wall: Optional[datetime] = None  # wall time of FIRST idle tick
        self._active_stop_id: Optional[str] = None

    # ── public API ──────────────────────────────────────────────

    def update(self, is_producing: bool) -> List[StopAction]:
        """Feed one observation; return zero or more actions."""
        if self._state == _State.PRODUCING:
            return self._in_producing(is_producing)
        if self._state == _State.IDLE_WAIT:
            return self._in_idle_wait(is_producing)
        if self._state == _State.STOP_OPEN:
            return self._in_stop_open(is_producing)
        return []

    def set_open_stop_id(self, stop_id: Optional[str]) -> None:
        """Called by the integration layer after the gateway returns stop_id."""
        self._active_stop_id = stop_id

    def update_threshold(self, micro_stop_max_s: float) -> None:
        self._micro_stop_max_s = micro_stop_max_s

    # ── read-only introspection (for /status) ───────────────────

    @property
    def state(self) -> _State:
        return self._state

    @property
    def active_stop_id(self) -> Optional[str]:
        return self._active_stop_id

    @property
    def idle_duration_s(self) -> float:
        if self._state == _State.PRODUCING:
            return 0.0
        return time.monotonic() - self._idle_start_mono

    # ── state handlers ──────────────────────────────────────────

    def _in_producing(self, is_producing: bool) -> List[StopAction]:
        if is_producing:
            self._idle_streak = 0
            self._first_idle_wall = None
            return []
        self._idle_streak += 1
        if self._idle_streak == 1:
            # Record wall time of the very first idle tick (before debounce elapses)
            self._first_idle_wall = datetime.now(timezone.utc)
        if self._idle_streak >= self._idle_debounce_ticks:
            self._state = _State.IDLE_WAIT
            self._idle_start_mono = time.monotonic()
            # Use first-idle wall time so started_at is not shifted by debounce delay
            self._idle_start_wall = self._first_idle_wall or datetime.now(timezone.utc)
        return []

    def _in_idle_wait(self, is_producing: bool) -> List[StopAction]:
        if is_producing:
            # Exit debounce: igual que STOP_OPEN, exige resume_debounce_ticks frames
            # consecutivos antes de cerrar el estado.
            # Sin esto, un solo frame de movimiento (mano pasando por el ROI) hace
            # que stop_tracker_state vuelva a 'producing' y el log parpadee.
            self._resume_streak += 1
            if self._resume_streak < self._resume_debounce_ticks:
                return []
            # Producción confirmada → cerrar idle_wait
            self._state = _State.PRODUCING
            self._idle_streak = 0
            self._resume_streak = 0
            elapsed = time.monotonic() - self._idle_start_mono
            actions: List[StopAction] = []
            # Brief gap: skip micro-stops shorter than brief_gap_s (matches JS _LOG_BRIEF_S)
            if self._idle_start_wall is not None and elapsed >= self._brief_gap_s:
                actions.append(StopAction(
                    kind="CREAR_MICROPARADA",
                    started_at=self._idle_start_wall,
                    ended_at=datetime.now(timezone.utc),
                ))
            self._idle_start_wall = None
            self._first_idle_wall = None
            return actions
        else:
            # Frame idle → resetear racha de producción
            self._resume_streak = 0

        elapsed = time.monotonic() - self._idle_start_mono
        if elapsed >= self._micro_stop_max_s:
            # Threshold crossed → open a real stop
            self._state = _State.STOP_OPEN
            self._resume_streak = 0
            return [StopAction(
                kind="ABRIR_PARADA",
                started_at=self._idle_start_wall or datetime.now(timezone.utc),
            )]
        return []

    def _in_stop_open(self, is_producing: bool) -> List[StopAction]:
        if not is_producing:
            self._resume_streak = 0
            return []
        # Exit debounce: require resume_debounce_ticks consecutive producing ticks
        # before closing the stop (prevents a single noisy frame from fragmenting it)
        self._resume_streak += 1
        if self._resume_streak < self._resume_debounce_ticks:
            return []
        # Production confirmed resumed → close the open stop
        self._state = _State.PRODUCING
        self._idle_streak = 0
        self._resume_streak = 0
        self._first_idle_wall = None
        actions: List[StopAction] = []
        if self._active_stop_id:
            actions.append(StopAction(
                kind="CERRAR_PARADA",
                started_at=self._idle_start_wall or datetime.now(timezone.utc),
                ended_at=datetime.now(timezone.utc),
                stop_id=self._active_stop_id,
            ))
        self._active_stop_id = None
        self._idle_start_wall = None
        return actions
