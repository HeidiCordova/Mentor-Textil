import unittest

from app.domain.fsm.textile_separator_fsm import EventFSM, FSMConfig, State


class _Clock:
    def __init__(self):
        self.now = 1000.0

    def monotonic(self):
        return self.now

    def advance(self, seconds):
        self.now += float(seconds)


def _config(**overrides):
    values = dict(
        calibration_frames=2,
        beige_confirm_frames=2,
        beige_confirm_s=0.0,
        beige_exit_frames=2,
        garment_evidence_frames=2,
        activity_window_s=5.0,
        garment_score_threshold=0.2,
        garment_motion_threshold=0.02,
        slow_motion_guard_frames=3,
        min_garment_s=1.0,
        rearm_beige_frames=3,
        rearm_beige_s=0.0,
        min_rearm_s=1.0,
    )
    values.update(overrides)
    return FSMConfig(**values)


class EventFSMTests(unittest.TestCase):
    def setUp(self):
        self.clock = _Clock()
        self.fsm = EventFSM(_config(), monotonic=self.clock.monotonic)
        self.assertIsNone(self.fsm.process(0.0, 0.9))
        self.assertIsNone(self.fsm.process(0.0, 0.9))
        self.assertEqual(self.fsm.state, State.BEIGE_IN)

    def _valid_garment(self):
        self.assertIsNone(self.fsm.process(0.3, 0.1, 0.03))
        self.clock.advance(1.1)
        self.assertIsNone(self.fsm.process(0.3, 0.1, 0.03))
        self.assertEqual(self.fsm.state, State.EN_PRENDA)
        self.assertIsNone(self.fsm.process(0.0, 0.9, 0.0))
        return self.fsm.process(0.0, 0.9, 0.0)

    def _rearm(self):
        self.clock.advance(1.1)
        for _ in range(3):
            self.assertIsNone(self.fsm.process(0.0, 0.9, 0.0))
        self.assertEqual(self.fsm.state, State.BEIGE_IN)

    def test_valid_garment_emits_exactly_one_corte(self):
        self.assertEqual(self._valid_garment(), "CORTE")
        self.assertEqual(self.fsm.state, State.COOLDOWN)
        metadata = self.fsm.last_event_metadata
        self.assertGreaterEqual(metadata["garment_duration_s"], 1.0)
        self.assertGreaterEqual(metadata["evidence_hits"], 2)
        self.assertEqual(metadata["evidence_sources"], ["fast_fusion"])

    def test_beige_dropout_without_independent_evidence_never_counts(self):
        events = []
        for _ in range(4):
            events.extend([
                self.fsm.process(0.0, 0.1, 0.005),
                self.fsm.process(0.0, 0.1, 0.005),
                self.fsm.process(0.0, 0.99, 0.005),
                self.fsm.process(0.0, 0.99, 0.005),
            ])
            self.clock.advance(10)
        self.assertNotIn("CORTE", events)
        for _ in range(3):
            self.fsm.process(0.0, 0.99, 0.0)
        self.assertEqual(self.fsm.state, State.BEIGE_IN)

    def test_one_evidence_spike_is_not_a_garment(self):
        self.fsm.process(0.3, 0.1, 0.0)
        self.fsm.process(0.0, 0.1, 0.0)
        self.clock.advance(2)
        self.fsm.process(0.0, 0.9, 0.0)
        event = self.fsm.process(0.0, 0.9, 0.0)
        self.assertIsNone(event)
        self.assertEqual(self.fsm.state, State.COOLDOWN)
        self.fsm.process(0.0, 0.9, 0.0)
        self.assertEqual(self.fsm.state, State.BEIGE_IN)

    def test_cooldown_requires_continuous_stable_beige(self):
        self.assertEqual(self._valid_garment(), "CORTE")
        self.clock.advance(2)
        for beige in (0.1, 0.9, 0.9, 0.1, 0.9, 0.9):
            self.fsm.process(0.0, beige, 0.0)
        self.assertEqual(self.fsm.state, State.COOLDOWN)
        self.fsm.process(0.0, 0.9, 0.0)
        self.assertEqual(self.fsm.state, State.BEIGE_IN)

    def test_second_real_garment_counts_once_after_rearm(self):
        self.assertEqual(self._valid_garment(), "CORTE")
        self._rearm()
        self.assertEqual(self._valid_garment(), "CORTE")

    def test_too_short_low_high_flash_is_rejected(self):
        strict = EventFSM(
            _config(min_garment_s=3.0),
            monotonic=self.clock.monotonic,
        )
        strict.process(0.0, 0.9)
        strict.process(0.0, 0.9)
        strict.process(0.3, 0.1, 0.03)
        strict.process(0.3, 0.1, 0.03)
        self.clock.advance(0.5)
        strict.process(0.0, 0.9, 0.0)
        event = strict.process(0.0, 0.9, 0.0)
        self.assertIsNone(event)
        self.assertEqual(strict.state, State.COOLDOWN)

    def test_activity_hits_must_cluster_inside_window(self):
        for _ in range(2):
            self.fsm.process(0.3, 0.1, 0.0)
            self.clock.advance(10)
            self.fsm.process(0.0, 0.1, 0.0)
            self.clock.advance(10)

        self.assertEqual(self.fsm.state, State.EN_PRENDA)
        self.fsm.process(0.0, 0.9, 0.0)
        self.assertIsNone(self.fsm.process(0.0, 0.9, 0.0))
        self.assertEqual(
            self.fsm.diagnostics["last_transition"]["rejection_reason"],
            "missing_clustered_activity",
        )

    def test_slow_motion_is_ignored_until_candidate_window_is_fresh(self):
        slow = EventFSM(
            _config(
                garment_evidence_frames=2,
                slow_motion_guard_frames=3,
            ),
            monotonic=self.clock.monotonic,
        )
        slow.process(0.0, 0.9)
        slow.process(0.0, 0.9)

        for _ in range(3):
            slow.process(
                0.0,
                0.1,
                0.03,
                fast_activity=False,
                slow_activity=True,
            )
        self.assertFalse(slow.diagnostics["candidate_verified"])

        slow.process(
            0.0,
            0.1,
            0.03,
            fast_activity=False,
            slow_activity=True,
        )
        slow.process(
            0.0,
            0.1,
            0.03,
            fast_activity=False,
            slow_activity=True,
        )
        self.assertTrue(slow.diagnostics["candidate_verified"])

    def test_final_high_activity_cannot_validate_retroactively(self):
        self.fsm.process(0.0, 0.1, 0.0)
        self.clock.advance(1.1)
        self.fsm.process(0.0, 0.1, 0.0)
        self.assertEqual(self.fsm.state, State.EN_PRENDA)

        self.fsm.process(0.9, 0.9, 0.9)
        self.assertIsNone(self.fsm.process(0.9, 0.9, 0.9))
        self.assertEqual(
            self.fsm.diagnostics["last_transition"]["rejection_reason"],
            "missing_clustered_activity",
        )

    def test_minimum_event_interval_does_not_delay_next_arm(self):
        guarded = EventFSM(
            _config(min_rearm_s=300.0),
            monotonic=self.clock.monotonic,
        )
        guarded.process(0.0, 0.9)
        guarded.process(0.0, 0.9)

        def cycle():
            guarded.process(0.3, 0.1, 0.0)
            self.clock.advance(1.1)
            guarded.process(0.3, 0.1, 0.0)
            guarded.process(0.0, 0.9, 0.0)
            return guarded.process(0.0, 0.9, 0.0)

        self.assertEqual(cycle(), "CORTE")
        guarded.process(0.0, 0.9, 0.0)
        self.assertEqual(guarded.state, State.BEIGE_IN)

        self.clock.advance(69.0)
        self.assertIsNone(cycle())
        self.assertEqual(
            guarded.diagnostics["last_transition"]["rejection_reason"],
            "minimum_event_interval",
        )
        guarded.process(0.0, 0.9, 0.0)
        self.assertEqual(guarded.state, State.BEIGE_IN)

        self.clock.advance(240.0)
        self.assertEqual(cycle(), "CORTE")

    def test_invalid_hysteresis_is_rejected(self):
        with self.assertRaises(ValueError):
            EventFSM(_config(beige_low=0.7, beige_high=0.5))


if __name__ == "__main__":
    unittest.main()
