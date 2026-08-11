import os
import tempfile
import threading
import time
import unittest

from app.adapters.durable_event_output import DurableEventOutput
from app.adapters.http_event_adapter import HTTPEventAdapter
from app.adapters.sqlite_event_spool import (
    EventConflictError,
    SQLiteEventSpool,
    SpoolFullError,
)
from app.ports.event_output import EventOutputError


def _event(event_id="event-1", event_type="CORTE", value=1):
    return {
        "event_id": event_id,
        "device_id": "jetson-test",
        "event_type": event_type,
        "timestamp": "2026-07-30T12:00:00+00:00",
        "payload": {"value": value},
    }


def _wait_until(predicate, timeout_s=2.0):
    deadline = time.monotonic() + timeout_s
    while time.monotonic() < deadline:
        if predicate():
            return True
        time.sleep(0.01)
    return predicate()


class _SequenceTransport:
    def __init__(self, results):
        self._results = list(results)
        self.calls = []
        self._lock = threading.Lock()

    def send_event(self, event):
        with self._lock:
            self.calls.append(event)
            if self._results:
                result = self._results.pop(0)
            else:
                result = True
        if isinstance(result, Exception):
            raise result
        return result


class _BlockingTransport:
    def __init__(self):
        self.started = threading.Event()
        self.release = threading.Event()
        self.calls = []

    def send_event(self, event):
        self.calls.append(event)
        self.started.set()
        self.release.wait(timeout=2.0)
        return True


class _FakeResponse:
    def __init__(self, status_code):
        self.status_code = status_code

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        return False


class _RecordingSession:
    def __init__(self, status_code):
        self.status_code = status_code
        self.calls = []
        self.closed = False

    def post(self, url, **kwargs):
        self.calls.append((url, kwargs))
        return _FakeResponse(self.status_code)

    def close(self):
        self.closed = True


class HTTPEventAdapterTests(unittest.TestCase):
    def test_sends_event_id_as_idempotency_key_and_accepts_any_2xx(self):
        adapter = HTTPEventAdapter(
            "http://resiliencia:8002/",
            linea_id="line 3",
            timeout=4,
        )
        adapter._session.close()
        session = _RecordingSession(202)
        adapter._session = session
        self.addCleanup(adapter.close)

        self.assertTrue(adapter.send_event(_event(event_id="stable-id")))

        url, kwargs = session.calls[0]
        self.assertEqual(url, "http://resiliencia:8002/events")
        self.assertEqual(kwargs["params"], {"linea_id": "line 3"})
        self.assertEqual(kwargs["headers"], {"Idempotency-Key": "stable-id"})
        self.assertEqual(kwargs["timeout"], 4.0)

    def test_non_2xx_is_not_acknowledged(self):
        adapter = HTTPEventAdapter("http://resiliencia:8002")
        adapter._session.close()
        adapter._session = _RecordingSession(503)
        self.addCleanup(adapter.close)

        self.assertFalse(adapter.send_event(_event()))


class SQLiteEventSpoolTests(unittest.TestCase):
    def setUp(self):
        self.tempdir = tempfile.TemporaryDirectory()
        self.addCleanup(self.tempdir.cleanup)
        self.path = os.path.join(self.tempdir.name, "events.sqlite3")

    def test_duplicate_id_is_idempotent_but_content_is_immutable(self):
        spool = SQLiteEventSpool(self.path)
        self.addCleanup(spool.close)

        self.assertTrue(spool.enqueue(_event()))
        self.assertFalse(spool.enqueue(_event()))
        self.assertEqual(spool.pending_count(), 1)

        with self.assertRaises(EventConflictError):
            spool.enqueue(_event(value=2))
        self.assertEqual(spool.pending_count(), 1)

    def test_limits_reject_new_event_without_evicting_existing_rows(self):
        spool = SQLiteEventSpool(
            self.path,
            max_events=1,
            max_event_bytes=1024,
        )
        self.addCleanup(spool.close)
        spool.enqueue(_event())

        with self.assertRaises(SpoolFullError):
            spool.enqueue(_event(event_id="event-2"))
        self.assertEqual(spool.pending_count(), 1)

        small_spool = SQLiteEventSpool(
            os.path.join(self.tempdir.name, "small.sqlite3"),
            max_event_bytes=100,
        )
        self.addCleanup(small_spool.close)
        with self.assertRaises(SpoolFullError):
            small_spool.enqueue(_event(value="x" * 500))
        self.assertEqual(small_spool.pending_count(), 0)


class DurableEventOutputTests(unittest.TestCase):
    def setUp(self):
        self.tempdir = tempfile.TemporaryDirectory()
        self.addCleanup(self.tempdir.cleanup)
        self.path = os.path.join(self.tempdir.name, "events.sqlite3")
        self.outputs = []

    def tearDown(self):
        for output in reversed(self.outputs):
            output.close(timeout_s=2.1)

    def _output(self, transport, **kwargs):
        output = DurableEventOutput(
            transport,
            spool_path=self.path,
            retry_initial_s=kwargs.pop("retry_initial_s", 0.02),
            retry_max_s=kwargs.pop("retry_max_s", 0.05),
            **kwargs,
        )
        self.outputs.append(output)
        return output

    def test_send_event_does_not_wait_for_blocking_transport(self):
        transport = _BlockingTransport()
        output = self._output(transport)
        result = []

        caller = threading.Thread(
            target=lambda: result.append(output.send_event(_event()))
        )
        caller.start()
        caller.join(timeout=1.0)

        self.assertFalse(caller.is_alive(), "send_event waited for HTTP transport")
        self.assertEqual(result, [True])
        self.assertTrue(transport.started.wait(timeout=1.0))
        self.assertEqual(output.pending_count, 1)

        transport.release.set()
        self.assertTrue(_wait_until(lambda: output.pending_count == 0))

    def test_failed_delivery_is_retried_and_acknowledged(self):
        transport = _SequenceTransport([False, RuntimeError("offline"), True])
        output = self._output(transport)

        self.assertTrue(output.send_event(_event()))

        self.assertTrue(_wait_until(lambda: len(transport.calls) >= 3))
        self.assertTrue(_wait_until(lambda: output.pending_count == 0))
        self.assertEqual(len(transport.calls), 3)

    def test_persisted_event_is_replayed_on_startup(self):
        spool = SQLiteEventSpool(self.path)
        spool.enqueue(_event(event_id="persisted"))
        spool.close()

        transport = _SequenceTransport([True])
        output = self._output(transport)

        self.assertTrue(_wait_until(lambda: output.pending_count == 0))
        self.assertEqual(
            [event["event_id"] for event in transport.calls],
            ["persisted"],
        )

    def test_shared_output_protects_corte_and_oee_event_types(self):
        transport = _SequenceTransport([True, True])
        output = self._output(transport)

        self.assertTrue(output.send_event(_event("cut", "CORTE")))
        self.assertTrue(output.send_event(_event("oee", "OEE_SNAPSHOT")))

        self.assertTrue(_wait_until(lambda: output.pending_count == 0))
        self.assertEqual(
            {event["event_type"] for event in transport.calls},
            {"CORTE", "OEE_SNAPSHOT"},
        )

    def test_max_attempts_retains_dead_letter_instead_of_deleting_it(self):
        transport = _SequenceTransport([False, False])
        output = self._output(transport, max_attempts=2)

        self.assertTrue(output.send_event(_event()))
        self.assertTrue(_wait_until(lambda: output.dead_count == 1))
        self.assertEqual(output.pending_count, 1)
        self.assertEqual(len(transport.calls), 2)

    def test_spool_failure_is_fail_closed(self):
        transport = _BlockingTransport()
        output = self._output(transport, max_events=1)
        self.assertTrue(output.send_event(_event(event_id="already-full")))
        self.assertTrue(transport.started.wait(timeout=1.0))

        with self.assertRaises(EventOutputError):
            output.send_event(_event(event_id="must-not-be-lost"))

        self.assertEqual(output.pending_count, 1)
        transport.release.set()


if __name__ == "__main__":
    unittest.main()
