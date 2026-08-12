import json
import threading
import time
import unittest
from types import SimpleNamespace

from app.application.detector_service import _HealthHandler
from app.domain.progress import ActiveCycleProgress


class _CaptureHandler:
    @staticmethod
    def status(detector):
        handler = object.__new__(_HealthHandler)
        handler.detector_ref = detector
        captured = {}

        def capture(status_code, body):
            captured["status"] = status_code
            captured["body"] = json.loads(body)

        handler._json = capture
        handler._send_status()
        return captured


def _detector(*, sample_age_s=0.1, sample_valid=True):
    fsm = SimpleNamespace(
        _state=SimpleNamespace(value="en_prenda"),
        diagnostics={"state": "en_prenda"},
    )
    tracker = SimpleNamespace(
        state=SimpleNamespace(value="producing"),
        active_stop_id=None,
        idle_duration_s=0.0,
        micro_stop_max_s=120.0,
    )
    presence = SimpleNamespace(is_warmed_up=True)
    progress = ActiveCycleProgress()
    progress.start_cycle(
        1200.0,
        cycle_id="cycle-1",
        run_id="run-1",
        product_id=17,
        sku="SKU-17",
    )
    progress.tick(False)
    return SimpleNamespace(
        _lock=threading.RLock(),
        _last_detecting=True,
        _last_presence_motion=False,
        _last_tracker_producing=False,
        _last_motion_sample_valid=sample_valid,
        _last_motion_sample_at=time.monotonic() - sample_age_s,
        _last_fusion_score=0.25,
        _last_beige_ratio=0.7,
        _last_motion_score=0.0,
        _presence_detector=presence,
        _progress=progress,
        _stop_tracker=tracker,
        fsm=fsm,
    )


class StatusMovementContractTests(unittest.TestCase):
    def test_status_separates_textile_state_from_physical_motion(self):
        result = _CaptureHandler.status(_detector())
        self.assertEqual(result["status"], 200)
        body = result["body"]
        self.assertEqual(body["fsm_state"], "en_prenda")
        self.assertFalse(body["presence_motion"])
        self.assertTrue(body["motion_ready"])
        self.assertTrue(body["motion_fresh"])
        self.assertEqual(body["micro_stop_max_s"], 120.0)
        self.assertEqual(body["progress_state"], "paused")
        self.assertEqual(body["progress_estimated_pct"], 0.0)
        self.assertTrue(body["progress_valid"])
        self.assertEqual(body["ideal_cycle_s"], 1200.0)
        self.assertEqual(body["progress_product_id"], 17)

    def test_stale_or_invalid_sample_is_not_a_stop_observation(self):
        stale = _CaptureHandler.status(_detector(sample_age_s=3.0))
        invalid = _CaptureHandler.status(_detector(sample_valid=False))
        self.assertFalse(stale["body"]["motion_fresh"])
        self.assertGreater(stale["body"]["motion_age_s"], 2.0)
        self.assertFalse(invalid["body"]["motion_fresh"])


if __name__ == "__main__":
    unittest.main()
