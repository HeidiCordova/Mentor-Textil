"""Estimated progress for one textile cycle.

The estimate advances from observed active-machine time and deliberately
stops at 99 percent.  Only a durable, verified ``CORTE`` may move it to
100 percent.  No wall-clock timestamp participates in the accumulation.
"""

from __future__ import annotations

import math
import time
from enum import Enum
from typing import Any, Callable, Dict, Optional


PROGRESS_VERSION = "active-time-v1"
PROGRESS_SIGNAL = "presence_motion_latched"


class ProgressState(str, Enum):
    """Externally visible state of the progress estimate."""

    UNAVAILABLE = "unavailable"
    WAITING_CYCLE = "waiting_cycle"
    ACTIVE = "active"
    PAUSED = "paused"
    OBSERVATION_GAP = "observation_gap"
    COMPLETED = "completed"
    INVALIDATED = "invalidated"


class ActiveCycleProgress:
    """Accumulate progress from valid, bounded monotonic observations.

    ``ideal_cycle_s`` and the production identity are frozen at the cycle
    boundary.  Invalid samples and observation gaps break continuity, so a
    camera outage can never be counted as productive time on recovery.
    """

    _PRE_COMPLETION_CAP_PCT = 99.0

    def __init__(
        self,
        clock: Callable[[], float] = time.monotonic,
        max_observation_gap_s: float = 2.0,
    ) -> None:
        max_gap = float(max_observation_gap_s)
        if not math.isfinite(max_gap) or max_gap <= 0:
            raise ValueError("max_observation_gap_s must be finite and greater than zero")

        self._clock = clock
        self._max_observation_gap_s = max_gap
        self._state = ProgressState.UNAVAILABLE
        self._reason: Optional[str] = "detector_startup"
        self._active_cycle_s = 0.0
        self._ideal_cycle_s: Optional[float] = None
        self._last_sample_at: Optional[float] = None
        self._last_sample_active = False
        self._cycle_id: Optional[str] = None
        self._run_id: Optional[str] = None
        self._product_id: Optional[int] = None
        self._sku: Optional[str] = None
        self._completion_event_id: Optional[str] = None
        self._completion_context_valid: Optional[bool] = None
        self._last_completion_event_id: Optional[str] = None
        self._last_completion_cycle_id: Optional[str] = None
        self._last_completion_context_valid: Optional[bool] = None

    @staticmethod
    def _validated_ideal(ideal_cycle_s: float) -> float:
        try:
            ideal_s = float(ideal_cycle_s)
        except (TypeError, ValueError) as exc:
            raise ValueError(
                "ideal_cycle_s must be a finite number greater than zero"
            ) from exc
        if (
            isinstance(ideal_cycle_s, bool)
            or not math.isfinite(ideal_s)
            or ideal_s <= 0
        ):
            raise ValueError("ideal_cycle_s must be a finite number greater than zero")
        return ideal_s

    def wait_for_cycle(
        self,
        ideal_cycle_s: float,
        *,
        run_id: Optional[str] = None,
        product_id: Optional[int] = None,
        sku: Optional[str] = None,
    ) -> None:
        """Publish a valid nominal context while waiting for a cycle boundary."""

        self._ideal_cycle_s = self._validated_ideal(ideal_cycle_s)
        self._state = ProgressState.WAITING_CYCLE
        self._reason = "awaiting_beige_to_garment_transition"
        self._active_cycle_s = 0.0
        self._clear_observation_anchor()
        self._set_identity(None, run_id, product_id, sku)
        self._completion_event_id = None
        self._completion_context_valid = None

    def set_unavailable(self, reason: str) -> None:
        """Clear an estimate when no trustworthy nominal context exists."""

        self._state = ProgressState.UNAVAILABLE
        self._reason = str(reason or "nominal_context_unavailable")
        self._active_cycle_s = 0.0
        self._ideal_cycle_s = None
        self._clear_observation_anchor()
        self._set_identity(None, None, None, None)
        self._completion_event_id = None
        self._completion_context_valid = None

    def begin_unresolved_cycle(self, cycle_id: str) -> None:
        """Open a new physical cycle without borrowing stale product context.

        The authoritative run and nominal speed are resolved asynchronously at
        the garment boundary.  Until that lookup completes, the cycle exists
        but its percentage and production identity are deliberately unknown.
        """

        if not isinstance(cycle_id, str) or not cycle_id.strip():
            raise ValueError("cycle_id is required for an unresolved cycle")
        self._state = ProgressState.INVALIDATED
        self._reason = "awaiting_boundary_progress_context"
        self._active_cycle_s = 0.0
        self._ideal_cycle_s = None
        self._clear_observation_anchor()
        self._set_identity(cycle_id.strip(), None, None, None)
        self._completion_event_id = None
        self._completion_context_valid = None

    def start_cycle(
        self,
        ideal_cycle_s: float,
        *,
        cycle_id: Optional[str] = None,
        run_id: Optional[str] = None,
        product_id: Optional[int] = None,
        sku: Optional[str] = None,
    ) -> None:
        """Start a confirmed textile cycle at exactly zero percent."""

        self._ideal_cycle_s = self._validated_ideal(ideal_cycle_s)
        self._state = ProgressState.OBSERVATION_GAP
        self._reason = "awaiting_first_motion_sample"
        self._active_cycle_s = 0.0
        self._clear_observation_anchor()
        self._set_identity(cycle_id, run_id, product_id, sku)
        self._completion_event_id = None
        self._completion_context_valid = None

    def tick(self, active: bool, sample_valid: bool = True) -> None:
        """Observe physical activity without ever counting an unobserved gap."""

        if not self.cycle_in_progress:
            self._clear_observation_anchor()
            return

        if not isinstance(sample_valid, bool) or not sample_valid:
            self._mark_gap("motion_sample_invalid")
            return
        if not isinstance(active, bool):
            self._mark_gap("motion_sample_invalid")
            return

        try:
            now = float(self._clock())
        except (TypeError, ValueError):
            self._mark_gap("monotonic_clock_invalid")
            return
        if not math.isfinite(now):
            self._mark_gap("monotonic_clock_invalid")
            return

        if self._last_sample_at is not None:
            delta_s = now - self._last_sample_at
            if delta_s < 0:
                self._mark_gap("monotonic_clock_regressed")
                return
            if delta_s > self._max_observation_gap_s:
                self._mark_gap("motion_observation_gap")
                return
            # The interval since the previous observation belongs to the
            # previous sample's state.  This counts up to the observed stop,
            # while a stopped -> active transition never counts the pause.
            if self._last_sample_active:
                self._active_cycle_s += delta_s

        self._last_sample_at = now
        self._last_sample_active = active
        self._state = ProgressState.ACTIVE if active else ProgressState.PAUSED
        self._reason = None if active else "presence_motion_false"

    def complete(self, event_id: str) -> bool:
        """Move a valid in-progress cycle to 100% after durable ``CORTE``."""

        if not isinstance(event_id, str) or not event_id.strip():
            raise ValueError("event_id is required to complete progress")
        if not self.cycle_in_progress and self._state is not ProgressState.INVALIDATED:
            return False
        context_valid = self.cycle_in_progress and self.valid
        if not context_valid:
            # CORTE proves physical completion, but an invalidated estimate
            # must never retain a run/product/ideal from stale context.
            self._active_cycle_s = 0.0
            self._ideal_cycle_s = None
            self._run_id = None
            self._product_id = None
            self._sku = None
        self._state = ProgressState.COMPLETED
        self._reason = None
        self._completion_event_id = event_id.strip()
        self._completion_context_valid = context_valid
        self._last_completion_event_id = self._completion_event_id
        self._last_completion_cycle_id = self._cycle_id
        self._last_completion_context_valid = context_valid
        self._clear_observation_anchor()
        return True

    def invalidate(self, reason: str) -> None:
        """Make a partial estimate explicitly unusable until a new boundary."""

        self._state = ProgressState.INVALIDATED
        self._reason = str(reason or "cycle_invalidated")
        self._clear_observation_anchor()
        self._completion_event_id = None
        self._completion_context_valid = None

    def _mark_gap(self, reason: str) -> None:
        self._state = ProgressState.OBSERVATION_GAP
        self._reason = reason
        self._clear_observation_anchor()

    def _clear_observation_anchor(self) -> None:
        self._last_sample_at = None
        self._last_sample_active = False

    def _set_identity(
        self,
        cycle_id: Optional[str],
        run_id: Optional[str],
        product_id: Optional[int],
        sku: Optional[str],
    ) -> None:
        self._cycle_id = str(cycle_id) if cycle_id is not None else None
        self._run_id = str(run_id) if run_id is not None else None
        self._product_id = int(product_id) if product_id is not None else None
        self._sku = str(sku) if sku is not None else None

    @property
    def state(self) -> ProgressState:
        return self._state

    @property
    def cycle_in_progress(self) -> bool:
        return self._state in {
            ProgressState.ACTIVE,
            ProgressState.PAUSED,
            ProgressState.OBSERVATION_GAP,
        }

    @property
    def valid(self) -> bool:
        if self._state is ProgressState.COMPLETED:
            return True
        return self._state in {
            ProgressState.ACTIVE,
            ProgressState.PAUSED,
            ProgressState.OBSERVATION_GAP,
        } and self._ideal_cycle_s is not None

    @property
    def observable(self) -> bool:
        return self._state in {ProgressState.ACTIVE, ProgressState.PAUSED}

    @property
    def active_cycle_s(self) -> float:
        return self._active_cycle_s

    @property
    def ideal_cycle_s(self) -> Optional[float]:
        return self._ideal_cycle_s

    @property
    def progress_estimated_pct(self) -> Optional[float]:
        if not self.valid:
            return None
        if self._state is ProgressState.COMPLETED:
            return 100.0
        assert self._ideal_cycle_s is not None
        estimated = (self._active_cycle_s / self._ideal_cycle_s) * 100.0
        return min(self._PRE_COMPLETION_CAP_PCT, max(0.0, estimated))

    def snapshot(self) -> Dict[str, Any]:
        """Return a status-ready, non-mutating view of the estimate."""

        return {
            "progress_version": PROGRESS_VERSION,
            "progress_signal": PROGRESS_SIGNAL,
            "progress_state": self.state.value,
            "progress_estimated_pct": self.progress_estimated_pct,
            "progress_valid": self.valid,
            "progress_observable": self.observable,
            "progress_reason": self._reason,
            "active_cycle_s": self.active_cycle_s,
            "ideal_cycle_s": self.ideal_cycle_s,
            "progress_cycle_id": self._cycle_id,
            "progress_run_id": self._run_id,
            "progress_product_id": self._product_id,
            "progress_sku": self._sku,
            "progress_completion_event_id": self._completion_event_id,
            "progress_completion_context_valid": self._completion_context_valid,
            "progress_last_completion_event_id": self._last_completion_event_id,
            "progress_last_completion_cycle_id": self._last_completion_cycle_id,
            "progress_last_completion_context_valid": (
                self._last_completion_context_valid
            ),
        }
