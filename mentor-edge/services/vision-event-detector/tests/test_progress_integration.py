import json
import logging
import threading
import time
import unittest
from unittest.mock import patch

from app.application.detector_service import (
    DetectorService,
    _select_active_product_progress_context,
)
from app.domain.fsm.textile_separator_fsm import State
from app.domain.fusion.fusion_engine import SignalValues
from app.domain.progress import ActiveCycleProgress, ProgressState
from app.ports.event_output import EventOutputError


def _active(run_id="run-17", product_id=17, sku="SKU-17"):
    result = {
        "status": "active",
        "run_id": run_id,
        "sku": sku,
        "linea_id": 1,
    }
    if product_id is not None:
        result["producto_id"] = product_id
    return result


def _nominal(velocity_us=1 / 1200, rows=None):
    return {
        "data": rows if rows is not None else [
            {
                "producto_id": 17,
                "sku": "SKU-17",
                "velocidad_us": velocity_us,
                "factor_conv": 99,
            }
        ]
    }


class _JSONResponse:
    def __init__(self, value):
        self._body = json.dumps(value).encode()

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def read(self):
        return self._body


class ActiveProductContextTests(unittest.TestCase):
    def test_authoritative_active_run_uses_oee_velocity_formula(self):
        context, reason = _select_active_product_progress_context(
            _active(),
            _nominal(),
        )

        self.assertIsNone(reason)
        self.assertEqual(context["run_id"], "run-17")
        self.assertEqual(context["product_id"], 17)
        self.assertAlmostEqual(context["ideal_cycle_s"], 1200.0)
        # factor_conv is intentionally ignored to stay aligned with cloud OEE.
        self.assertAlmostEqual(context["velocity_us"], 1 / 1200)

    def test_missing_run_and_invalid_or_extreme_velocity_are_unavailable(self):
        context, reason = _select_active_product_progress_context(
            {"status": "no_active_run"},
            _nominal(),
        )
        self.assertIsNone(context)
        self.assertEqual(reason, "active_production_run_missing")

        context, reason = _select_active_product_progress_context(
            _active(),
            _nominal(0),
        )
        self.assertIsNone(context)
        self.assertEqual(reason, "nominal_velocity_invalid")

        context, reason = _select_active_product_progress_context(
            _active(),
            _nominal(5e-324),
        )
        self.assertIsNone(context)
        self.assertEqual(reason, "nominal_cycle_time_invalid")

    def test_product_id_is_exclusive_and_sku_is_only_a_fallback(self):
        rows = [
            {
                "producto_id": 18,
                "sku": "STALE-SKU",
                "velocidad_us": 1 / 600,
            },
            {
                "producto_id": 17,
                "sku": "CANONICAL-SKU",
                "velocidad_us": 1 / 1200,
            },
        ]
        context, reason = _select_active_product_progress_context(
            _active(product_id=17, sku="STALE-SKU"),
            _nominal(rows=rows),
        )
        self.assertIsNone(reason)
        self.assertEqual(context["product_id"], 17)
        self.assertEqual(context["ideal_cycle_s"], 1200.0)

        context, reason = _select_active_product_progress_context(
            _active(product_id=None, sku="STALE-SKU"),
            _nominal(rows=rows),
        )
        self.assertIsNone(reason)
        self.assertEqual(context["product_id"], 18)

    def test_gateway_requests_are_line_scoped_and_use_authoritative_active_api(self):
        service = object.__new__(DetectorService)
        service._gateway_url = "http://edge-gateway:8005"
        service.line_id = "7"
        service._logger = logging.getLogger("test.progress.fetch")
        responses = [_JSONResponse(_active()), _JSONResponse(_nominal())]

        with patch(
            "app.application.detector_service.urllib.request.urlopen",
            side_effect=responses,
        ) as urlopen:
            identity, context, reason, authoritative = (
                service._fetch_active_product_progress_context()
            )

        self.assertTrue(authoritative)
        self.assertIsNone(reason)
        self.assertEqual(identity["run_id"], "run-17")
        self.assertEqual(context["ideal_cycle_s"], 1200.0)
        urls = [call.args[0] for call in urlopen.call_args_list]
        self.assertIn("/edge/vision/count?linea_id=7", urls[0])
        self.assertIn("/edge/catalogs/velocidad-nominal?linea_id=7", urls[1])

    def test_catalog_timeout_preserves_authoritative_run_identity(self):
        service = object.__new__(DetectorService)
        service._gateway_url = "http://edge-gateway:8005"
        service.line_id = "1"
        service._logger = logging.getLogger("test.progress.fetch")

        with patch(
            "app.application.detector_service.urllib.request.urlopen",
            side_effect=[_JSONResponse(_active(run_id="run-18")), TimeoutError()],
        ):
            identity, context, reason, authoritative = (
                service._fetch_active_product_progress_context()
            )

        self.assertTrue(authoritative)
        self.assertEqual(identity["run_id"], "run-18")
        self.assertIsNone(context)
        self.assertEqual(reason, "nominal_context_lookup_failed")


class DetectorProgressBoundaryTests(unittest.TestCase):
    def _service(self):
        service = object.__new__(DetectorService)
        service._lock = threading.RLock()
        service._logger = logging.getLogger("test.progress")
        service._progress = ActiveCycleProgress(max_observation_gap_s=2.0)
        service._progress_active_identity = {
            "run_id": "run-17",
            "product_id": 17,
            "sku": "SKU-17",
        }
        service._progress_product_context = {
            "run_id": "run-17",
            "product_id": 17,
            "sku": "SKU-17",
            "velocity_us": 1 / 1200,
            "ideal_cycle_s": 1200.0,
        }
        service._progress_context_refreshed_at = time.monotonic()
        service._progress_context_failure_reason = None
        service._progress_refresh_wakeup = threading.Event()
        service._progress_pending_cycle_id = None
        service._progress_fetch_generation = 0
        service._progress_pending_after_generation = 0
        service.fsm = type("FSM", (), {"state": State.BEIGE_IN})()
        return service

    @staticmethod
    def _confirm_boundary_context(service):
        service.fsm.state = State.EN_PRENDA
        identity = dict(service._progress_active_identity)
        context = dict(service._progress_product_context)
        service._fetch_active_product_progress_context = lambda: (
            identity,
            context,
            None,
            True,
        )
        service._refresh_active_product_progress_context()

    def test_confirmed_garment_boundary_starts_zero_with_frozen_context(self):
        service = self._service()
        service._start_progress_cycle_locked()
        self.assertEqual(service._progress.state, ProgressState.INVALIDATED)
        self.assertEqual(
            service._progress.snapshot()["progress_reason"],
            "awaiting_boundary_progress_context",
        )
        self._confirm_boundary_context(service)

        snap = service._progress.snapshot()
        self.assertEqual(snap["progress_estimated_pct"], 0.0)
        self.assertEqual(snap["ideal_cycle_s"], 1200.0)
        self.assertEqual(snap["progress_run_id"], "run-17")
        self.assertEqual(snap["progress_product_id"], 17)

        service._progress_product_context["ideal_cycle_s"] = 600.0
        self.assertEqual(service._progress.ideal_cycle_s, 1200.0)

    def test_boundary_never_starts_from_a_stale_cached_product(self):
        service = self._service()
        service._progress_context_refreshed_at = 0.0
        service._start_progress_cycle_locked()
        self.assertEqual(service._progress.state, ProgressState.INVALIDATED)
        self.assertIsNone(service._progress.progress_estimated_pct)
        self.assertIsNotNone(service._progress_pending_cycle_id)

    def test_unresolved_boundary_cannot_reuse_previous_garment_identity(self):
        service = self._service()
        service._progress.start_cycle(
            900.0,
            cycle_id="old-cycle",
            run_id="old-run",
            product_id=99,
            sku="OLD-SKU",
        )
        service._progress.complete("old-corte")

        service._start_progress_cycle_locked()

        snap = service._progress.snapshot()
        self.assertEqual(
            snap["progress_cycle_id"],
            service._progress_pending_cycle_id,
        )
        self.assertIsNone(snap["progress_run_id"])
        self.assertIsNone(snap["progress_product_id"])
        self.assertIsNone(snap["progress_sku"])
        self.assertIsNone(snap["ideal_cycle_s"])

    def test_corte_while_boundary_context_is_missing_is_factual_but_unattributed(self):
        service = self._service()
        service._start_progress_cycle_locked()
        pending_cycle_id = service._progress_pending_cycle_id
        service.fsm.state = State.EN_PRENDA
        service._fetch_active_product_progress_context = lambda: (
            None,
            None,
            "active_production_run_missing",
            True,
        )

        service._refresh_active_product_progress_context()
        self.assertEqual(service._progress_pending_cycle_id, pending_cycle_id)
        self.assertTrue(service._progress.complete("corte-without-context"))

        snap = service._progress.snapshot()
        self.assertEqual(snap["progress_estimated_pct"], 100.0)
        self.assertEqual(snap["progress_cycle_id"], pending_cycle_id)
        self.assertIsNone(snap["progress_run_id"])
        self.assertIsNone(snap["progress_product_id"])
        self.assertFalse(snap["progress_completion_context_valid"])

    def test_response_started_before_boundary_cannot_satisfy_pending_cycle(self):
        service = self._service()
        identity = dict(service._progress_active_identity)
        context = dict(service._progress_product_context)
        fetch_started = threading.Event()
        release_fetch = threading.Event()

        def delayed_fetch():
            fetch_started.set()
            self.assertTrue(release_fetch.wait(timeout=2.0))
            return identity, context, None, True

        service._fetch_active_product_progress_context = delayed_fetch
        refresh = threading.Thread(
            target=service._refresh_active_product_progress_context,
        )
        refresh.start()
        self.assertTrue(fetch_started.wait(timeout=2.0))

        service.fsm.state = State.EN_PRENDA
        with service._lock:
            service._start_progress_cycle_locked()
        pending_cycle_id = service._progress_pending_cycle_id
        release_fetch.set()
        refresh.join(timeout=2.0)
        self.assertFalse(refresh.is_alive())

        self.assertEqual(service._progress_pending_cycle_id, pending_cycle_id)
        self.assertEqual(service._progress.state, ProgressState.INVALIDATED)

        service._fetch_active_product_progress_context = lambda: (
            identity,
            context,
            None,
            True,
        )
        service._refresh_active_product_progress_context()
        self.assertIsNone(service._progress_pending_cycle_id)
        self.assertEqual(service._progress.progress_estimated_pct, 0.0)

    def test_run_change_invalidates_even_when_nominal_catalog_times_out(self):
        service = self._service()
        service._start_progress_cycle_locked()
        self._confirm_boundary_context(service)
        new_identity = {
            "run_id": "run-18",
            "product_id": 18,
            "sku": "SKU-18",
        }
        service._fetch_active_product_progress_context = lambda: (
            new_identity,
            None,
            "nominal_context_lookup_failed",
            True,
        )

        service._refresh_active_product_progress_context()

        self.assertEqual(service._progress.state, ProgressState.INVALIDATED)
        self.assertEqual(
            service._progress.snapshot()["progress_reason"],
            "active_production_run_changed",
        )

    def test_nominal_edit_or_failure_does_not_rewrite_current_same_run(self):
        service = self._service()
        service._start_progress_cycle_locked()
        self._confirm_boundary_context(service)
        same_identity = dict(service._progress_active_identity)
        service._fetch_active_product_progress_context = lambda: (
            same_identity,
            None,
            "nominal_velocity_invalid",
            True,
        )

        service._refresh_active_product_progress_context()

        self.assertTrue(service._progress.cycle_in_progress)
        self.assertEqual(service._progress.ideal_cycle_s, 1200.0)
        self.assertIsNone(service._progress_product_context)

    def test_sku_only_run_is_not_invalidated_by_catalog_product_id_enrichment(self):
        service = self._service()
        service._progress_active_identity = {
            "run_id": "run-sku",
            "product_id": None,
            "sku": "SKU-18",
        }
        service._progress_product_context = {
            "run_id": "run-sku",
            "product_id": 18,
            "sku": "SKU-18",
            "velocity_us": 1 / 600,
            "ideal_cycle_s": 600.0,
        }
        service._start_progress_cycle_locked()
        self._confirm_boundary_context(service)
        self.assertTrue(service._progress.cycle_in_progress)

        # The next authoritative refresh still lacks producto_id on the run,
        # while the nominal catalog continues enriching the cycle with ID 18.
        self._confirm_boundary_context(service)
        self.assertTrue(service._progress.cycle_in_progress)
        self.assertNotEqual(service._progress.state, ProgressState.INVALIDATED)

    def test_closing_run_clears_retained_completion(self):
        service = self._service()
        service._start_progress_cycle_locked()
        self._confirm_boundary_context(service)
        service._progress.complete("corte-1")
        service._fetch_active_product_progress_context = lambda: (
            None,
            None,
            "active_production_run_missing",
            True,
        )

        service._refresh_active_product_progress_context()

        self.assertEqual(service._progress.state, ProgressState.UNAVAILABLE)
        self.assertIsNone(service._progress.progress_estimated_pct)

    def test_durable_output_acceptance_controls_one_hundred_percent(self):
        class Output:
            def __init__(self, accepted):
                self.accepted = accepted

            def send_event(self, _event):
                return self.accepted

        service = self._service()
        service._start_progress_cycle_locked()
        self._confirm_boundary_context(service)
        service.device_id = "device-1"
        service._line_code = "LINE-1"
        service.fsm.last_event_metadata = {}
        signals = SignalValues(edge=0.1, color=0.2, flow=0.3, beige=0.8)

        service.event_output = Output(True)
        event_id = service._emit_event("CORTE", signals, 0.9)
        service._progress.complete(event_id)
        self.assertEqual(service._progress.progress_estimated_pct, 100.0)

        service._progress.start_cycle(1200.0, run_id="run-17", product_id=17)
        service.event_output = Output(False)
        with self.assertRaises(EventOutputError):
            service._emit_event("CORTE", signals, 0.9)
        self.assertEqual(service._progress.progress_estimated_pct, 0.0)


if __name__ == "__main__":
    unittest.main()
