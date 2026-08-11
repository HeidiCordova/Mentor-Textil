"""Durable, non-blocking outbound event adapter."""

from __future__ import annotations

import logging
import threading
import time
from typing import Any, Dict, Optional

from ..ports.event_output import EventOutput, EventOutputError
from .sqlite_event_spool import SQLiteEventSpool, SpoolRecord

logger = logging.getLogger("detector.event.durable")


class DurableEventOutput(EventOutput):
    """Queue events locally and deliver them from a background worker.

    ``send_event`` means "durably accepted", not "already delivered".  The
    remote service must treat ``event_id`` idempotently because a crash after a
    successful HTTP response but before local acknowledgement causes a replay.
    """

    def __init__(
        self,
        transport: EventOutput,
        *,
        spool_path: str,
        max_events: int = 10_000,
        max_event_bytes: int = 65_536,
        retry_initial_s: float = 1.0,
        retry_max_s: float = 60.0,
        max_attempts: int = 0,
        spool: Optional[SQLiteEventSpool] = None,
    ) -> None:
        if retry_initial_s <= 0:
            raise ValueError("retry_initial_s must be greater than zero")
        if retry_max_s < retry_initial_s:
            raise ValueError("retry_max_s must be >= retry_initial_s")
        if max_attempts < 0:
            raise ValueError("max_attempts must be zero or greater")

        self._transport = transport
        self._retry_initial_s = float(retry_initial_s)
        self._retry_max_s = float(retry_max_s)
        self._max_attempts = int(max_attempts)
        self._spool = spool or SQLiteEventSpool(
            spool_path,
            max_events=max_events,
            max_event_bytes=max_event_bytes,
        )
        self._wake = threading.Event()
        self._stop = threading.Event()
        self._closed = False
        self._resources_closed = False
        self._close_lock = threading.Lock()
        self._worker = threading.Thread(
            target=self._run,
            name="event-delivery-worker",
            daemon=True,
        )
        self._worker.start()

        pending = self._spool.pending_count(include_dead=False)
        if pending:
            logger.info("Replaying %d persisted outbound event(s)", pending)
        dead = self._spool.dead_count()
        if dead:
            logger.critical(
                "%d outbound event(s) remain in retained dead-letter state",
                dead,
            )

    @property
    def pending_count(self) -> int:
        return self._spool.pending_count()

    @property
    def dead_count(self) -> int:
        return self._spool.dead_count()

    def send_event(self, event: Dict[str, Any]) -> bool:
        with self._close_lock:
            if self._closed:
                logger.error("Rejected event because durable output is closed")
                return False
            try:
                self._spool.enqueue(event)
            except Exception as exc:
                logger.critical(
                    "Could not persist outbound event; event was not accepted",
                    exc_info=True,
                )
                raise EventOutputError(
                    "Could not persist outbound event safely"
                ) from exc

        self._wake.set()
        return True

    def _retry_delay(self, attempts: int) -> float:
        exponent = max(0, min(attempts - 1, 30))
        return min(self._retry_max_s, self._retry_initial_s * (2 ** exponent))

    def _record_failure(self, record: SpoolRecord, error: str) -> None:
        attempts = record.attempts + 1
        if self._max_attempts and attempts >= self._max_attempts:
            self._spool.mark_dead(
                record.event_id,
                attempts=attempts,
                last_error=error,
            )
            logger.critical(
                "Outbound event %s moved to retained dead-letter state after %d attempts",
                record.event_id,
                attempts,
            )
            return

        delay = self._retry_delay(attempts)
        self._spool.reschedule(
            record.event_id,
            attempts=attempts,
            next_attempt_at=time.time() + delay,
            last_error=error,
        )
        logger.warning(
            "Outbound event %s delivery failed; retry %d in %.1fs",
            record.event_id,
            attempts,
            delay,
        )

    def _deliver(self, record: SpoolRecord) -> None:
        try:
            event = record.event()
        except Exception as exc:
            self._spool.mark_dead(
                record.event_id,
                attempts=record.attempts,
                last_error=f"invalid persisted JSON: {exc}",
            )
            logger.critical(
                "Outbound event %s has invalid persisted JSON and was retained",
                record.event_id,
                exc_info=True,
            )
            return

        try:
            delivered = self._transport.send_event(event)
        except Exception as exc:
            self._record_failure(
                record,
                f"{type(exc).__name__}: {exc}",
            )
            return

        if delivered:
            self._spool.ack(record.event_id)
        else:
            self._record_failure(record, "transport returned false")

    def _run(self) -> None:
        while not self._stop.is_set():
            try:
                record = self._spool.next_due()
                if record is not None:
                    self._deliver(record)
                    continue

                delay = self._spool.seconds_until_next()
                wait_s = 30.0 if delay is None else min(30.0, delay)
                self._wake.wait(timeout=wait_s)
                self._wake.clear()
            except Exception:
                logger.exception("Outbound event worker failed; retrying")
                self._wake.wait(timeout=1.0)
                self._wake.clear()

    def close(self, timeout_s: float = 6.0) -> None:
        with self._close_lock:
            if self._resources_closed:
                return
            if not self._closed:
                self._closed = True
                self._stop.set()
                self._wake.set()
            self._worker.join(timeout=max(0.0, timeout_s))

            if self._worker.is_alive():
                logger.warning(
                    "Outbound event worker did not stop within %.1fs; "
                    "persisted events remain in the spool",
                    timeout_s,
                )
                return

            close_transport = getattr(self._transport, "close", None)
            try:
                if callable(close_transport):
                    close_transport()
            finally:
                self._spool.close()
                self._resources_closed = True
