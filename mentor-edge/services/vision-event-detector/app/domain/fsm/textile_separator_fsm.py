"""State machine for one-count-per-textile-separator cycles.

A ``CORTE`` is accepted only after this complete sequence:

1. a stable beige separator arms the detector;
2. a sustained non-beige candidate contains independent, clustered activity;
3. the following beige separator remains stable for the configured dwell time.

Activity is collected while the garment is visible and frozen when the closing
separator first appears. Flow is normally zero on the quiet separator frame
that closes a real garment, so final-frame activity is deliberately ignored.
"""

from collections import deque
from dataclasses import dataclass
from enum import Enum
import time
from typing import Any, Callable, Deque, Dict, Optional, Set, Tuple


FSM_VERSION = "textile-separator-v2"


class State(Enum):
    IDLE = "idle"
    BEIGE_IN = "beige_in"
    EN_PRENDA = "en_prenda"
    COOLDOWN = "cooldown"


@dataclass(slots=True)
class FSMConfig:
    calibration_frames: int = 20

    beige_high: float = 0.55
    beige_low: float = 0.30
    beige_confirm_frames: int = 10
    beige_confirm_s: float = 2.0
    beige_exit_frames: int = 8

    # N hits must occur inside one short window. Once reached, evidence is
    # latched for the rest of the potentially long textile cycle.
    garment_evidence_frames: int = 3
    activity_window_s: float = 5.0
    garment_score_threshold: float = 0.20
    garment_motion_threshold: float = 0.02
    slow_motion_guard_frames: int = 375
    min_garment_s: float = 30.0

    rearm_beige_frames: int = 25
    rearm_beige_s: float = 2.0
    # Kept under its legacy name for API compatibility. Semantically this is
    # the minimum interval between accepted events, not an arming delay.
    min_rearm_s: float = 300.0

    def validate(self) -> None:
        if not 0.0 <= self.beige_low < self.beige_high <= 1.0:
            raise ValueError("expected 0 <= beige_low < beige_high <= 1")
        for name in (
            "calibration_frames",
            "beige_confirm_frames",
            "beige_exit_frames",
            "garment_evidence_frames",
            "slow_motion_guard_frames",
            "rearm_beige_frames",
        ):
            if int(getattr(self, name)) <= 0:
                raise ValueError(f"{name} must be greater than zero")
        for name in (
            "beige_confirm_s",
            "activity_window_s",
            "garment_score_threshold",
            "garment_motion_threshold",
            "min_garment_s",
            "rearm_beige_s",
            "min_rearm_s",
        ):
            if float(getattr(self, name)) < 0.0:
                raise ValueError(f"{name} cannot be negative")


class EventFSM:
    def __init__(
        self,
        config: FSMConfig,
        monotonic: Optional[Callable[[], float]] = None,
    ):
        config.validate()
        self.config = config
        self._monotonic = monotonic or time.monotonic
        self._last_event_metadata: Optional[Dict[str, Any]] = None
        self.reset()

    def process(
        self,
        fusion_score: float,
        beige_ratio: float,
        motion_score: float = 0.0,
        *,
        fast_activity: Optional[bool] = None,
        slow_activity: Optional[bool] = None,
    ) -> Optional[str]:
        now = float(self._monotonic())
        beige = max(0.0, min(1.0, float(beige_ratio)))
        score = max(0.0, float(fusion_score))
        motion = max(0.0, float(motion_score))
        fast = (
            score >= self.config.garment_score_threshold
            if fast_activity is None
            else bool(fast_activity)
        )
        slow = (
            motion >= self.config.garment_motion_threshold
            if slow_activity is None
            else bool(slow_activity)
        )

        if self._state == State.IDLE:
            return self._handle_idle(beige)
        if self._state == State.BEIGE_IN:
            return self._handle_beige_in(now, beige, score, motion, fast, slow)
        if self._state == State.EN_PRENDA:
            return self._handle_en_prenda(now, beige, score, motion, fast, slow)
        if self._state == State.COOLDOWN:
            return self._handle_cooldown(now, beige)
        return None

    def _handle_idle(self, beige: float) -> Optional[str]:
        if beige >= self.config.beige_high:
            self._calibration_count += 1
            if self._calibration_count >= self.config.calibration_frames:
                self._state = State.BEIGE_IN
                self._calibration_count = 0
                self._reset_candidate()
        else:
            self._calibration_count = 0
        return None

    def _handle_beige_in(
        self,
        now: float,
        beige: float,
        score: float,
        motion: float,
        fast: bool,
        slow: bool,
    ) -> Optional[str]:
        if beige >= self.config.beige_high:
            self._reset_candidate()
            return None

        if self._candidate_started_at is None:
            self._start_candidate(now, beige, score, motion)

        self._candidate_frames += 1
        self._observe_cycle(beige, score, motion)
        self._observe_activity(now, fast, slow)

        if beige < self.config.beige_low:
            self._no_beige_streak += 1
        else:
            # Mid-band frames keep the candidate/evidence but do not satisfy
            # the sustained exit requirement.
            self._no_beige_streak = 0

        if self._no_beige_streak >= self.config.beige_exit_frames:
            self._state = State.EN_PRENDA
            self._garment_started_at = self._candidate_started_at
            self._beige_streak = 0
            self._closing_started_at = None
        return None

    def _handle_en_prenda(
        self,
        now: float,
        beige: float,
        score: float,
        motion: float,
        fast: bool,
        slow: bool,
    ) -> Optional[str]:
        self._observe_cycle(beige, score, motion)

        if beige < self.config.beige_high:
            # A broken HIGH island is still part of this garment. Activity on
            # HIGH frames is never accepted, but low/mid activity may continue
            # validating the candidate.
            self._beige_streak = 0
            self._closing_started_at = None
            self._closing_evidence_valid = False
            self._closing_evidence_hits = 0
            self._closing_evidence_sources.clear()
            self._candidate_frames += 1
            self._observe_activity(now, fast, slow)
            return None

        if self._closing_started_at is None:
            self._closing_started_at = now
            self._closing_evidence_valid = self._candidate_verified
            self._closing_evidence_hits = self._verified_evidence_hits
            self._closing_evidence_sources = set(self._verified_evidence_sources)

        self._beige_streak += 1
        high_duration_s = max(0.0, now - self._closing_started_at)
        if (
            self._beige_streak < self.config.beige_confirm_frames
            or high_duration_s < self.config.beige_confirm_s
        ):
            return None

        garment_started_at = self._garment_started_at or now
        garment_duration_s = max(
            0.0,
            self._closing_started_at - garment_started_at,
        )
        rejection_reason: Optional[str] = None
        if garment_duration_s < self.config.min_garment_s:
            rejection_reason = "candidate_too_short"
        elif not self._closing_evidence_valid:
            rejection_reason = "missing_clustered_activity"
        elif (now - self._last_corte_time) < self.config.min_rearm_s:
            rejection_reason = "minimum_event_interval"

        emitted = rejection_reason is None
        metadata = self._build_cycle_metadata(
            garment_duration_s=garment_duration_s,
            high_duration_s=high_duration_s,
            emitted=emitted,
            rejection_reason=rejection_reason,
        )
        self._last_transition_metadata = metadata
        if emitted:
            self._last_corte_time = now
            self._last_event_metadata = metadata

        # Carry the already-observed HIGH into the post-cut lock. This rearms
        # on the current separator rather than the next one, avoiding a skipped
        # garment when the separator is shorter than min_rearm_s.
        self._begin_rearm(
            high_started_at=self._closing_started_at,
            high_frames=self._beige_streak,
        )
        return "CORTE" if emitted else None

    def _handle_cooldown(self, now: float, beige: float) -> Optional[str]:
        if beige >= self.config.beige_high:
            if self._stable_high_started_at is None:
                self._stable_high_started_at = now
            self._rearm_beige_streak += 1
        else:
            self._stable_high_started_at = None
            self._rearm_beige_streak = 0

        high_duration_s = (
            0.0
            if self._stable_high_started_at is None
            else max(0.0, now - self._stable_high_started_at)
        )
        if (
            self._rearm_beige_streak >= self.config.rearm_beige_frames
            and high_duration_s >= self.config.rearm_beige_s
        ):
            self._state = State.BEIGE_IN
            self._stable_high_started_at = None
            self._rearm_beige_streak = 0
            self._reset_candidate()
        return None

    def _start_candidate(
        self,
        now: float,
        beige: float,
        score: float,
        motion: float,
    ) -> None:
        self._reset_candidate()
        self._candidate_started_at = now
        self._cycle_min_beige = beige
        self._cycle_max_score = score
        self._cycle_max_motion = motion

    def _observe_activity(self, now: float, fast: bool, slow: bool) -> None:
        sources: Set[str] = set()
        if fast:
            sources.add("fast_fusion")
        # The slow-window comparison is fresh only after this candidate has
        # existed for a complete presence window.
        if slow and self._candidate_frames > self.config.slow_motion_guard_frames:
            sources.add("slow_window")
        if not sources:
            return

        self._activity_hits.append((now, sources))
        cutoff = now - self.config.activity_window_s
        while self._activity_hits and self._activity_hits[0][0] < cutoff:
            self._activity_hits.popleft()

        if len(self._activity_hits) >= self.config.garment_evidence_frames:
            self._candidate_verified = True
            self._verified_evidence_hits = len(self._activity_hits)
            for _, hit_sources in self._activity_hits:
                self._verified_evidence_sources.update(hit_sources)

    def _observe_cycle(self, beige: float, score: float, motion: float) -> None:
        self._cycle_min_beige = min(self._cycle_min_beige, beige)
        self._cycle_max_score = max(self._cycle_max_score, score)
        self._cycle_max_motion = max(self._cycle_max_motion, motion)

    def _build_cycle_metadata(
        self,
        *,
        garment_duration_s: float,
        high_duration_s: float,
        emitted: bool,
        rejection_reason: Optional[str],
    ) -> Dict[str, Any]:
        return {
            "fsm_version": FSM_VERSION,
            "garment_duration_s": round(garment_duration_s, 3),
            "separator_high_s": round(high_duration_s, 3),
            "candidate_frames": self._candidate_frames,
            "evidence_hits": self._closing_evidence_hits,
            "evidence_sources": sorted(self._closing_evidence_sources),
            "min_beige": round(self._cycle_min_beige, 6),
            "max_fusion": round(self._cycle_max_score, 6),
            "max_motion": round(self._cycle_max_motion, 6),
            "emitted": emitted,
            "rejection_reason": rejection_reason,
        }

    def _begin_rearm(self, *, high_started_at: float, high_frames: int) -> None:
        self._state = State.COOLDOWN
        self._stable_high_started_at = high_started_at
        self._rearm_beige_streak = high_frames
        self._beige_streak = 0
        self._closing_started_at = None

    def _reset_candidate(self) -> None:
        self._beige_streak = 0
        self._no_beige_streak = 0
        self._candidate_started_at: Optional[float] = None
        self._garment_started_at: Optional[float] = None
        self._closing_started_at: Optional[float] = None
        self._candidate_frames = 0
        self._activity_hits: Deque[Tuple[float, Set[str]]] = deque()
        self._candidate_verified = False
        self._verified_evidence_hits = 0
        self._verified_evidence_sources: Set[str] = set()
        self._closing_evidence_valid = False
        self._closing_evidence_hits = 0
        self._closing_evidence_sources: Set[str] = set()
        self._cycle_min_beige = 1.0
        self._cycle_max_score = 0.0
        self._cycle_max_motion = 0.0

    def reset(self) -> None:
        self._state = State.IDLE
        self._calibration_count = 0
        self._stable_high_started_at: Optional[float] = None
        self._rearm_beige_streak = 0
        self._last_corte_time = float("-inf")
        self._last_event_metadata = None
        self._last_transition_metadata: Optional[Dict[str, Any]] = None
        self._reset_candidate()

    @property
    def state(self) -> State:
        return self._state

    @property
    def last_event_metadata(self) -> Dict[str, Any]:
        return dict(self._last_event_metadata or {})

    @property
    def diagnostics(self) -> Dict[str, Any]:
        return {
            "fsm_version": FSM_VERSION,
            "state": self._state.value,
            "no_beige_streak": self._no_beige_streak,
            "beige_streak": self._beige_streak,
            "candidate_frames": self._candidate_frames,
            "candidate_verified": self._candidate_verified,
            "recent_activity_hits": len(self._activity_hits),
            "evidence_sources": sorted(self._verified_evidence_sources),
            "rearm_beige_streak": self._rearm_beige_streak,
            "candidate_active": self._candidate_started_at is not None,
            "max_fusion": round(self._cycle_max_score, 6),
            "max_motion": round(self._cycle_max_motion, 6),
            "min_beige": round(self._cycle_min_beige, 6),
            "last_transition": dict(self._last_transition_metadata or {}),
            "config": {
                "beige_high": self.config.beige_high,
                "beige_low": self.config.beige_low,
                "beige_confirm_frames": self.config.beige_confirm_frames,
                "beige_confirm_s": self.config.beige_confirm_s,
                "beige_exit_frames": self.config.beige_exit_frames,
                "garment_evidence_frames": self.config.garment_evidence_frames,
                "activity_window_s": self.config.activity_window_s,
                "garment_score_threshold": self.config.garment_score_threshold,
                "garment_motion_threshold": self.config.garment_motion_threshold,
                "slow_motion_guard_frames": self.config.slow_motion_guard_frames,
                "min_garment_s": self.config.min_garment_s,
                "rearm_beige_frames": self.config.rearm_beige_frames,
                "rearm_beige_s": self.config.rearm_beige_s,
                "min_event_interval_s": self.config.min_rearm_s,
            },
        }
