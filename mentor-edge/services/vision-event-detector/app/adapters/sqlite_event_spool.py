"""Crash-safe, bounded spool for outbound detector events.

The spool is intentionally transport-agnostic.  Once ``enqueue`` returns, the
event has been committed to SQLite and may safely be delivered by a background
worker.  Rows are deleted only after the remote transport acknowledges them.
"""

from __future__ import annotations

import json
import os
import sqlite3
import threading
import time
from dataclasses import dataclass
from typing import Any, Dict, Optional


class EventSpoolError(RuntimeError):
    """Base class for local spool failures."""


class InvalidEventError(EventSpoolError):
    """Raised when an event cannot be stored safely."""


class EventConflictError(EventSpoolError):
    """Raised when an event_id is reused with different content."""


class SpoolFullError(EventSpoolError):
    """Raised when a configured spool limit has been reached."""


@dataclass(frozen=True)
class SpoolRecord:
    event_id: str
    event_type: str
    event_json: str
    attempts: int

    def event(self) -> Dict[str, Any]:
        value = json.loads(self.event_json)
        if not isinstance(value, dict):
            raise InvalidEventError(
                f"Stored event {self.event_id!r} is not a JSON object"
            )
        return value


class SQLiteEventSpool:
    """Persistent FIFO spool backed by one SQLite connection.

    ``max_events`` and ``max_event_bytes`` are hard logical limits.  Existing
    events are never evicted to make room: rejecting a new event is safer and
    observable, while silent eviction would destroy production history.
    """

    def __init__(
        self,
        path: str,
        *,
        max_events: int = 10_000,
        max_event_bytes: int = 65_536,
        busy_timeout_s: float = 5.0,
        journal_size_limit_bytes: int = 16 * 1024 * 1024,
    ) -> None:
        if not path or not path.strip():
            raise ValueError("Spool path must not be empty")
        if max_events <= 0:
            raise ValueError("max_events must be greater than zero")
        if max_event_bytes <= 0:
            raise ValueError("max_event_bytes must be greater than zero")
        if busy_timeout_s <= 0:
            raise ValueError("busy_timeout_s must be greater than zero")

        self.path = os.path.abspath(path)
        self.max_events = int(max_events)
        self.max_event_bytes = int(max_event_bytes)
        self._lock = threading.Lock()
        self._closed = False

        parent = os.path.dirname(self.path)
        if parent:
            os.makedirs(parent, exist_ok=True)

        self._connection = sqlite3.connect(
            self.path,
            timeout=float(busy_timeout_s),
            isolation_level=None,
            check_same_thread=False,
        )
        try:
            with self._lock:
                self._connection.execute(
                    f"PRAGMA busy_timeout = {int(busy_timeout_s * 1000)}"
                )
                self._connection.execute("PRAGMA journal_mode = WAL")
                self._connection.execute("PRAGMA synchronous = FULL")
                self._connection.execute("PRAGMA wal_autocheckpoint = 1000")
                self._connection.execute(
                    f"PRAGMA journal_size_limit = {int(journal_size_limit_bytes)}"
                )
                self._connection.executescript(
                    """
                    CREATE TABLE IF NOT EXISTS event_spool (
                        event_id TEXT PRIMARY KEY,
                        event_type TEXT NOT NULL,
                        event_json TEXT NOT NULL,
                        created_at REAL NOT NULL,
                        next_attempt_at REAL NOT NULL,
                        attempts INTEGER NOT NULL DEFAULT 0,
                        last_error TEXT,
                        dead INTEGER NOT NULL DEFAULT 0
                            CHECK (dead IN (0, 1))
                    );

                    CREATE INDEX IF NOT EXISTS idx_event_spool_due
                    ON event_spool(dead, next_attempt_at, created_at);
                    """
                )
        except Exception:
            self._connection.close()
            raise

    @staticmethod
    def _serialize(event: Dict[str, Any]) -> tuple[str, str, str]:
        if not isinstance(event, dict):
            raise InvalidEventError("Event must be a dictionary")

        event_id = event.get("event_id")
        event_type = event.get("event_type")
        if not isinstance(event_id, str) or not event_id.strip():
            raise InvalidEventError("Event must contain a non-empty string event_id")
        if len(event_id) > 255:
            raise InvalidEventError("event_id exceeds 255 characters")
        if not isinstance(event_type, str) or not event_type.strip():
            raise InvalidEventError("Event must contain a non-empty string event_type")
        if len(event_type) > 255:
            raise InvalidEventError("event_type exceeds 255 characters")

        try:
            event_json = json.dumps(
                event,
                ensure_ascii=False,
                allow_nan=False,
                sort_keys=True,
                separators=(",", ":"),
            )
        except (TypeError, ValueError) as exc:
            raise InvalidEventError(f"Event is not valid JSON: {exc}") from exc
        return event_id, event_type, event_json

    def _ensure_open(self) -> None:
        if self._closed:
            raise EventSpoolError("Event spool is closed")

    def enqueue(self, event: Dict[str, Any]) -> bool:
        """Commit an event.

        Returns ``True`` when a row was inserted and ``False`` when the exact
        same event was already pending.  Reusing an event_id with different
        content raises ``EventConflictError``.
        """

        event_id, event_type, event_json = self._serialize(event)
        event_bytes = len(event_json.encode("utf-8"))
        if event_bytes > self.max_event_bytes:
            raise SpoolFullError(
                f"Event {event_id!r} is {event_bytes} bytes; "
                f"limit is {self.max_event_bytes}"
            )

        now = time.time()
        with self._lock:
            self._ensure_open()
            connection = self._connection
            connection.execute("BEGIN IMMEDIATE")
            try:
                existing = connection.execute(
                    "SELECT event_json FROM event_spool WHERE event_id = ?",
                    (event_id,),
                ).fetchone()
                if existing is not None:
                    if existing[0] != event_json:
                        raise EventConflictError(
                            f"event_id {event_id!r} already exists with different content"
                        )
                    connection.execute("COMMIT")
                    return False

                current_count = connection.execute(
                    "SELECT COUNT(*) FROM event_spool"
                ).fetchone()[0]
                if current_count >= self.max_events:
                    raise SpoolFullError(
                        f"Event spool contains {current_count} rows; "
                        f"limit is {self.max_events}"
                    )

                connection.execute(
                    """
                    INSERT INTO event_spool (
                        event_id,
                        event_type,
                        event_json,
                        created_at,
                        next_attempt_at
                    ) VALUES (?, ?, ?, ?, ?)
                    """,
                    (event_id, event_type, event_json, now, now),
                )
                connection.execute("COMMIT")
                return True
            except Exception:
                connection.execute("ROLLBACK")
                raise

    def next_due(self, now: Optional[float] = None) -> Optional[SpoolRecord]:
        due_at = time.time() if now is None else float(now)
        with self._lock:
            self._ensure_open()
            row = self._connection.execute(
                """
                SELECT event_id, event_type, event_json, attempts
                FROM event_spool
                WHERE dead = 0 AND next_attempt_at <= ?
                ORDER BY created_at ASC, rowid ASC
                LIMIT 1
                """,
                (due_at,),
            ).fetchone()
        if row is None:
            return None
        return SpoolRecord(
            event_id=row[0],
            event_type=row[1],
            event_json=row[2],
            attempts=int(row[3]),
        )

    def seconds_until_next(self, now: Optional[float] = None) -> Optional[float]:
        current = time.time() if now is None else float(now)
        with self._lock:
            self._ensure_open()
            row = self._connection.execute(
                "SELECT MIN(next_attempt_at) FROM event_spool WHERE dead = 0"
            ).fetchone()
        if row is None or row[0] is None:
            return None
        return max(0.0, float(row[0]) - current)

    def ack(self, event_id: str) -> None:
        with self._lock:
            self._ensure_open()
            self._connection.execute(
                "DELETE FROM event_spool WHERE event_id = ?",
                (event_id,),
            )

    def reschedule(
        self,
        event_id: str,
        *,
        attempts: int,
        next_attempt_at: float,
        last_error: str,
    ) -> None:
        with self._lock:
            self._ensure_open()
            self._connection.execute(
                """
                UPDATE event_spool
                SET attempts = ?, next_attempt_at = ?, last_error = ?
                WHERE event_id = ?
                """,
                (
                    int(attempts),
                    float(next_attempt_at),
                    str(last_error)[:1024],
                    event_id,
                ),
            )

    def mark_dead(self, event_id: str, *, attempts: int, last_error: str) -> None:
        with self._lock:
            self._ensure_open()
            self._connection.execute(
                """
                UPDATE event_spool
                SET attempts = ?, last_error = ?, dead = 1
                WHERE event_id = ?
                """,
                (int(attempts), str(last_error)[:1024], event_id),
            )

    def pending_count(self, *, include_dead: bool = True) -> int:
        where = "" if include_dead else " WHERE dead = 0"
        with self._lock:
            self._ensure_open()
            return int(
                self._connection.execute(
                    f"SELECT COUNT(*) FROM event_spool{where}"
                ).fetchone()[0]
            )

    def dead_count(self) -> int:
        with self._lock:
            self._ensure_open()
            return int(
                self._connection.execute(
                    "SELECT COUNT(*) FROM event_spool WHERE dead = 1"
                ).fetchone()[0]
            )

    def close(self) -> None:
        with self._lock:
            if self._closed:
                return
            self._connection.close()
            self._closed = True
