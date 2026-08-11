import unittest
from unittest.mock import patch

from app.domain.stop_tracker.stop_tracker import StopTracker


class _Clock:
    def __init__(self) -> None:
        self.now = 1000.0

    def monotonic(self) -> float:
        return self.now

    def advance(self, seconds: float) -> None:
        self.now += seconds


class StopTrackerContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.clock = _Clock()
        self.clock_patch = patch(
            "app.domain.stop_tracker.stop_tracker.time.monotonic",
            side_effect=self.clock.monotonic,
        )
        self.clock_patch.start()
        self.addCleanup(self.clock_patch.stop)

    def tracker(self, threshold: float = 10.0) -> StopTracker:
        return StopTracker(
            micro_stop_max_s=threshold,
            idle_debounce_ticks=1,
            resume_debounce_ticks=1,
            brief_gap_s=3.0,
        )

    def test_threshold_is_exposed_as_single_status_source(self) -> None:
        tracker = self.tracker(12.5)
        self.assertEqual(tracker.micro_stop_max_s, 12.5)
        tracker.update_threshold(20.0)
        self.assertEqual(tracker.micro_stop_max_s, 20.0)

    def test_gap_shorter_than_three_seconds_is_discarded(self) -> None:
        tracker = self.tracker()
        self.assertEqual(tracker.update(False), [])
        self.clock.advance(2.9)
        self.assertEqual(tracker.update(True), [])

    def test_three_seconds_is_a_microstop_when_motion_resumes(self) -> None:
        tracker = self.tracker()
        self.assertEqual(tracker.update(False), [])
        self.clock.advance(3.0)
        actions = tracker.update(True)
        self.assertEqual([action.kind for action in actions], ["CREAR_MICROPARADA"])

    def test_exact_threshold_opens_unassigned_stop_once(self) -> None:
        tracker = self.tracker(10.0)
        self.assertEqual(tracker.update(False), [])
        self.clock.advance(10.0)
        actions = tracker.update(False)
        self.assertEqual([action.kind for action in actions], ["ABRIR_PARADA"])
        self.assertEqual(tracker.update(False), [])

    def test_open_stop_closes_without_creating_microstop(self) -> None:
        tracker = self.tracker(10.0)
        tracker.update(False)
        self.clock.advance(10.0)
        opened = tracker.update(False)
        tracker.set_open_stop_id("stop-1")
        self.assertEqual(opened[0].kind, "ABRIR_PARADA")
        closed = tracker.update(True)
        self.assertEqual([action.kind for action in closed], ["CERRAR_PARADA"])
        self.assertEqual(closed[0].stop_id, "stop-1")


if __name__ == "__main__":
    unittest.main()
