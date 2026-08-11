"""False-positive-resistant state machine for textile separator counting.

A CORTE requires a full cycle:

1. stable beige separator;
2. sustained non-beige frames with independent motion/fusion evidence;
3. the next stable beige separator.

The second condition prevents camera/lighting dropouts from being interpreted as
garments merely because the beige ratio later returns to one.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Any, Dict, Optional


class State(Enum):
    IDLE = "idle"
    BEIGE_IN = "beige_in"
    EN_PRENDA = "en_prenda"
    COOLDOWN = "cooldown"


@dataclass
class FSMConfig:
    calibration_frames: int = 20

    beige_high: float = 0.55
    beige_low: float = 0.30
    beige_confirm_frames: int = 10
    beige_exit_frames: int = 8

    garment_evidence_frames: int = 3
    garment_score_threshold: float = 0.20
    garment_motion_threshold: float = 0.02
    min_garment_s: float = 1.0

    rearm_beige_frames: int = 25
    min_rearm_s: float = 300.0

    def validate(self) -> None:
        if not 0.0 <= self.beige_low < self.beige_high <= 1.0:
            raise ValueError("expected 0 <= beige_low < beige_high <= 1")
        for name in (
            "calibration_frames",
            "beige_confirm_frames",
            "beige_exit_frames",
            "garment_evidence_frames",
            "rearm_beige_frames",
        ):
            if int(getattr(self, name)) <= 0:
                raise ValueError(f"{name} must be greater than zero")
        for name in (
            "garment_score_threshold",
            "garment_motion_threshold",
            "min_garment_s",
            "min_rearm_s",
        ):
            if float(getattr(self, name)) < 0.0:
                raise ValueError(f"{name} cannot be negative")


class EventFSM:
    def __init__(self, config: FSMConfig):
        import time as _time

        config.validate()
        self.config = config
        self._time = _time
        self._last_event_metadata: Optional[Dict[str, Any]] = None
        self.reset()

    def process(
        self,
        fusion_score: float,
        beige_ratio: float,
        motion_score: float = 0.0,
    ) -> Optional[str]:
        now = float(self._time.time())
        beige = max(0.0, min(1.0, float(beige_ratio)))
        score = max(0.0, float(fusion_score))
        motion = max(0.0, float(motion_score))

        if self._state == State.IDLE:
            return self._handle_idle(beige)
        if self._state == State.BEIGE_IN:
            return self._handle_beige_in(now, beige, score, motion)
        if self._state == State.EN_PRENDA:
            return self._handle_en_prenda(now, beige, score, motion)
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
                return "CALIBRATE"
        else:
            self._calibration_count = 0
        return None

    def _handle_beige_in(
        self,
        now: float,
        beige: float,
        score: float,
        motion: float,
    ) -> Optional[str]:
        if beige >= self.config.beige_low:
            self._reset_candidate()
            return None

        if self._candidate_started_at is None:
            self._candidate_started_at = now
            self._cycle_min_beige = beige
            self._cycle_max_score = score
            self._cycle_max_motion = motion

        self._no_beige_streak += 1
        self._observe_cycle(beige, score, motion)
        if self._is_garment_evidence(score, motion):
            self._evidence_frames += 1

        if (
            self._no_beige_streak >= self.config.beige_exit_frames
            and self._evidence_frames >= self.config.garment_evidence_frames
        ):
            self._state = State.EN_PRENDA
            self._garment_started_at = self._candidate_started_at
            self._beige_streak = 0
            self._no_beige_streak = 0
        return None

    def _handle_en_prenda(
        self,
        now: float,
        beige: float,
        score: float,
        motion: float,
    ) -> Optional[str]:
        self._observe_cycle(beige, score, motion)
        if self._is_garment_evidence(score, motion):
            self._evidence_frames += 1

        if beige < self.config.beige_high:
            self._beige_streak = 0
            return None

        self._beige_streak += 1
        if self._beige_streak < self.config.beige_confirm_frames:
            return None

        garment_started_at = self._garment_started_at or now
        garment_duration_s = max(0.0, now - garment_started_at)
        if garment_duration_s < self.config.min_garment_s:
            self._begin_rearm(now)
            return None

        if (now - self._last_corte_time) < self.config.min_rearm_s:
            self._begin_rearm(now)
            return None

        self._last_corte_time = now
        self._last_event_metadata = {
            "garment_duration_s": round(garment_duration_s, 3),
            "evidence_frames": self._evidence_frames,
            "min_beige": round(self._cycle_min_beige, 6),
            "max_fusion": round(self._cycle_max_score, 6),
            "max_motion": round(self._cycle_max_motion, 6),
        }
        self._begin_rearm(now)
        return "CORTE"

    def _handle_cooldown(self, now: float, beige: float) -> Optional[str]:
        if beige >= self.config.beige_high:
            self._rearm_beige_streak += 1
        else:
            self._rearm_beige_streak = 0

        if (
            self._rearm_beige_streak >= self.config.rearm_beige_frames
            and now >= self._rearm_not_before
        ):
            self._state = State.BEIGE_IN
            self._reset_candidate()
            self._rearm_beige_streak = 0
        return None

    def _is_garment_evidence(self, score: float, motion: float) -> bool:
        return (
            score >= self.config.garment_score_threshold
            or motion >= self.config.garment_motion_threshold
        )

    def _observe_cycle(self, beige: float, score: float, motion: float) -> None:
        self._cycle_min_beige = min(self._cycle_min_beige, beige)
        self._cycle_max_score = max(self._cycle_max_score, score)
        self._cycle_max_motion = max(self._cycle_max_motion, motion)

    def _begin_rearm(self, now: float) -> None:
        self._state = State.COOLDOWN
        self._rearm_not_before = now + self.config.min_rearm_s
        self._rearm_beige_streak = 0
        self._beige_streak = 0
        self._no_beige_streak = 0
        self._candidate_started_at = None
        self._garment_started_at = None

    def _reset_candidate(self) -> None:
        self._beige_streak = 0
        self._no_beige_streak = 0
        self._candidate_started_at = None
        self._garment_started_at = None
        self._evidence_frames = 0
        self._cycle_min_beige = 1.0
        self._cycle_max_score = 0.0
        self._cycle_max_motion = 0.0

    def reset(self) -> None:
        self._state = State.IDLE
        self._calibration_count = 0
        self._rearm_beige_streak = 0
        self._rearm_not_before = 0.0
        self._last_corte_time = -999.0
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
            "state": self._state.value,
            "no_beige_streak": self._no_beige_streak,
            "beige_streak": self._beige_streak,
            "evidence_frames": self._evidence_frames,
            "rearm_beige_streak": self._rearm_beige_streak,
            "candidate_active": self._candidate_started_at is not None,
            "max_fusion": round(self._cycle_max_score, 6),
            "max_motion": round(self._cycle_max_motion, 6),
            "min_beige": round(self._cycle_min_beige, 6),
            "config": {
                "beige_high": self.config.beige_high,
                "beige_low": self.config.beige_low,
                "beige_confirm_frames": self.config.beige_confirm_frames,
                "beige_exit_frames": self.config.beige_exit_frames,
                "garment_evidence_frames": self.config.garment_evidence_frames,
                "garment_score_threshold": self.config.garment_score_threshold,
                "garment_motion_threshold": self.config.garment_motion_threshold,
                "min_garment_s": self.config.min_garment_s,
                "rearm_beige_frames": self.config.rearm_beige_frames,
                "min_rearm_s": self.config.min_rearm_s,
            },
        }
