import math
import unittest

from app.domain.progress.active_cycle_progress import (
    ActiveCycleProgress,
    ProgressState,
)


class _Clock:
    def __init__(self) -> None:
        self.now = 1000.0

    def monotonic(self) -> float:
        return self.now

    def advance(self, seconds: float) -> None:
        self.now += seconds


class ActiveCycleProgressTests(unittest.TestCase):
    def setUp(self) -> None:
        self.clock = _Clock()
        self.progress = ActiveCycleProgress(
            clock=self.clock.monotonic,
            max_observation_gap_s=5.0,
        )

    def _start(self, ideal_s: float = 1200.0) -> None:
        self.progress.start_cycle(
            ideal_s,
            cycle_id="cycle-1",
            run_id="run-1",
            product_id=17,
            sku="SKU-17",
        )

    def _observe(self, active: bool, seconds: float = 0.0) -> None:
        self.clock.advance(seconds)
        self.progress.tick(active)

    def test_starts_unavailable_with_null_percentage(self) -> None:
        snap = self.progress.snapshot()
        self.assertEqual(snap["progress_state"], "unavailable")
        self.assertIsNone(snap["progress_estimated_pct"])
        self.assertFalse(snap["progress_valid"])

    def test_waiting_context_is_not_a_false_zero_cycle(self) -> None:
        self.progress.wait_for_cycle(1200, run_id="run-1", product_id=17)
        snap = self.progress.snapshot()
        self.assertEqual(snap["progress_state"], "waiting_cycle")
        self.assertIsNone(snap["progress_estimated_pct"])
        self.assertEqual(snap["ideal_cycle_s"], 1200.0)

    def test_start_cycle_validates_nominal_duration(self) -> None:
        for invalid in (None, True, 0, -1, math.inf, -math.inf, math.nan, "x"):
            with self.subTest(invalid=invalid):
                with self.assertRaises(ValueError):
                    self.progress.start_cycle(invalid)  # type: ignore[arg-type]

    def test_cycle_boundary_starts_at_exactly_zero_and_freezes_identity(self) -> None:
        self._start()
        snap = self.progress.snapshot()
        self.assertEqual(snap["progress_estimated_pct"], 0.0)
        self.assertEqual(snap["ideal_cycle_s"], 1200.0)
        self.assertEqual(snap["progress_run_id"], "run-1")
        self.assertEqual(snap["progress_product_id"], 17)

    def test_active_time_produces_expected_percentage(self) -> None:
        self._start()
        self._observe(True)
        for _ in range(60):
            self._observe(True, 5.0)
        self.assertEqual(self.progress.active_cycle_s, 300.0)
        self.assertEqual(self.progress.progress_estimated_pct, 25.0)
        self.assertEqual(self.progress.state, ProgressState.ACTIVE)

    def test_pause_freezes_and_resume_never_counts_stopped_gap(self) -> None:
        self._start()
        self._observe(True)
        self._observe(False, 5.0)
        active_before_pause = self.progress.active_cycle_s
        self._observe(False, 5.0)
        self._observe(True, 5.0)
        self.assertEqual(self.progress.active_cycle_s, active_before_pause)
        self._observe(True, 5.0)
        self.assertEqual(self.progress.active_cycle_s, active_before_pause + 5.0)

    def test_invalid_sample_and_large_gap_break_continuity(self) -> None:
        self._start(100.0)
        self._observe(True)
        self._observe(True, 5.0)
        self.clock.advance(1.0)
        self.progress.tick(True, sample_valid=False)
        self._observe(True, 4.0)
        self._observe(True, 6.0)
        self.assertEqual(self.progress.active_cycle_s, 5.0)
        self.assertEqual(self.progress.state, ProgressState.OBSERVATION_GAP)
        self._observe(True, 1.0)
        self.assertEqual(self.progress.active_cycle_s, 5.0)

    def test_clock_regression_breaks_anchor_without_later_overcount(self) -> None:
        self._start(100.0)
        self._observe(True)
        self._observe(True, 5.0)
        self._observe(True, -1.0)
        self._observe(True, 2.0)
        self._observe(True, 2.0)
        self.assertEqual(self.progress.active_cycle_s, 7.0)

    def test_estimate_caps_at_99_until_durable_corte(self) -> None:
        self._start(10.0)
        self._observe(True)
        for _ in range(5):
            self._observe(True, 5.0)
        self.assertEqual(self.progress.progress_estimated_pct, 99.0)
        self.assertTrue(self.progress.complete("event-corte-1"))
        self.assertEqual(self.progress.progress_estimated_pct, 100.0)
        self.assertEqual(self.progress.state, ProgressState.COMPLETED)
        self.assertTrue(
            self.progress.snapshot()["progress_completion_context_valid"]
        )

    def test_completion_is_retained_and_idempotent(self) -> None:
        self._start(100.0)
        self.assertTrue(self.progress.complete("event-corte-1"))
        self.clock.advance(500.0)
        self.progress.tick(True)
        self.assertEqual(self.progress.progress_estimated_pct, 100.0)
        self.assertFalse(self.progress.complete("event-corte-1"))

    def test_next_cycle_resets_retained_completion(self) -> None:
        self._start(100.0)
        self.progress.complete("event-corte-1")
        self.progress.start_cycle(200.0, cycle_id="cycle-2", run_id="run-1")
        self.assertEqual(self.progress.progress_estimated_pct, 0.0)
        self.assertEqual(self.progress.active_cycle_s, 0.0)
        self.assertEqual(self.progress.snapshot()["progress_cycle_id"], "cycle-2")
        self.assertIsNone(self.progress.snapshot()["progress_completion_event_id"])
        self.assertEqual(
            self.progress.snapshot()["progress_last_completion_event_id"],
            "event-corte-1",
        )
        self.assertEqual(
            self.progress.snapshot()["progress_last_completion_cycle_id"],
            "cycle-1",
        )

    def test_invalidation_publishes_null_not_a_misleading_number(self) -> None:
        self._start()
        self._observe(True)
        self._observe(True, 5.0)
        self.progress.invalidate("production_run_changed")
        snap = self.progress.snapshot()
        self.assertEqual(snap["progress_state"], "invalidated")
        self.assertEqual(snap["active_cycle_s"], 5.0)
        self.assertIsNone(snap["progress_estimated_pct"])
        self.assertFalse(snap["progress_valid"])

    def test_durable_corte_is_true_completion_even_after_estimate_invalidates(self) -> None:
        self._start()
        self.progress.invalidate("active_production_run_changed")

        self.assertTrue(self.progress.complete("event-corte-after-gap"))
        self.assertEqual(self.progress.state, ProgressState.COMPLETED)
        self.assertEqual(self.progress.progress_estimated_pct, 100.0)
        self.assertTrue(self.progress.valid)
        snap = self.progress.snapshot()
        self.assertIsNone(snap["progress_run_id"])
        self.assertIsNone(snap["progress_product_id"])
        self.assertIsNone(snap["ideal_cycle_s"])
        self.assertFalse(snap["progress_completion_context_valid"])

    def test_unresolved_boundary_never_borrows_previous_garment_identity(self) -> None:
        self._start()
        self.progress.complete("event-old")

        self.progress.begin_unresolved_cycle("cycle-new")
        pending = self.progress.snapshot()
        self.assertEqual(pending["progress_cycle_id"], "cycle-new")
        self.assertIsNone(pending["progress_run_id"])
        self.assertIsNone(pending["progress_product_id"])
        self.assertIsNone(pending["ideal_cycle_s"])
        self.assertIsNone(pending["progress_estimated_pct"])

        self.assertTrue(self.progress.complete("event-new"))
        completed = self.progress.snapshot()
        self.assertEqual(completed["progress_estimated_pct"], 100.0)
        self.assertEqual(completed["progress_cycle_id"], "cycle-new")
        self.assertEqual(
            completed["progress_last_completion_cycle_id"],
            "cycle-new",
        )
        self.assertFalse(completed["progress_completion_context_valid"])

    def test_completion_requires_event_identity(self) -> None:
        self._start()
        with self.assertRaises(ValueError):
            self.progress.complete("")


if __name__ == "__main__":
    unittest.main()
