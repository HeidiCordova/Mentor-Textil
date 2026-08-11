import time


class OEEAggregator:
    """Accumulates OEE metrics within configurable time windows.

    Tracks micro-stops, unassigned stops, and production counts.
    All time values are in **seconds** (the canonical system unit).

    T_DISPONIBLE is NOT included in the snapshot — it is calculated
    from the shift schedule (sync_turnos) in the edge-gateway and
    injected by the detector service.

    Idle classification uses the *total continuous* idle duration across
    snapshot boundaries.  A period whose cumulative idle is shorter than
    ``micro_stop_max_s`` is a micro-stop; equal or longer is an
    unassigned stop.  Only the portion of idle that falls inside the
    current window is added to that window's counters, but the
    classification threshold is evaluated against the full continuous
    idle duration.
    """

    def __init__(
        self,
        snapshot_interval_s: float = 1800.0,
        micro_stop_max_s: float = 120.0,
        stop_max_s: float = 86400.0,
    ) -> None:
        self._snapshot_interval = snapshot_interval_s
        self._micro_stop_max_s: float = micro_stop_max_s
        self._stop_max_s: float = stop_max_s

        now = time.monotonic()
        self._window_start: float = now
        self._last_tick: float = now

        self._next_wall_emit: float = self._next_boundary(time.time())

        self._micro_stop_s: float = 0.0
        self._stop_s: float = 0.0
        self._parada_mayor_s: float = 0.0
        self._cut_count_1: int = 0

        self._continuous_idle_s: float = 0.0
        self._idle_in_window_s: float = 0.0
        self._was_producing: bool = False

    def tick(self, is_producing: bool) -> None:
        now = time.monotonic()
        delta_s = max(0.0, now - self._last_tick)
        self._last_tick = now

        if is_producing:
            if not self._was_producing and self._continuous_idle_s > 0:
                self._classify_and_flush()
        else:
            self._continuous_idle_s += delta_s
            self._idle_in_window_s += delta_s

        self._was_producing = is_producing

    def _classify_and_flush(self) -> None:
        if self._idle_in_window_s <= 0:
            self._continuous_idle_s = 0.0
            return
        if self._continuous_idle_s < self._micro_stop_max_s:
            self._micro_stop_s += self._idle_in_window_s
        elif self._continuous_idle_s < self._stop_max_s:
            self._stop_s += self._idle_in_window_s
        else:
            self._parada_mayor_s += self._idle_in_window_s
        self._idle_in_window_s = 0.0
        self._continuous_idle_s = 0.0

    def record_cut(self) -> None:
        self._cut_count_1 += 1

    def update_config(
        self,
        micro_stop_max_s: float,
        snapshot_interval_s: float,
        stop_max_s: float | None = None,
    ) -> None:
        old_interval = self._snapshot_interval
        self._micro_stop_max_s = micro_stop_max_s
        self._snapshot_interval = snapshot_interval_s
        if stop_max_s is not None:
            self._stop_max_s = stop_max_s
        if snapshot_interval_s != old_interval:
            self._next_wall_emit = self._next_boundary(time.time())

    def _next_boundary(self, wall_now: float) -> float:
        interval = int(self._snapshot_interval)
        if interval <= 0:
            interval = 1800
        remainder = int(wall_now) % interval
        return wall_now + (interval - remainder)

    def should_emit(self) -> bool:
        return time.time() >= self._next_wall_emit

    def snapshot(self) -> dict:
        """Return snapshot with time values in **seconds**.

        T_DISPONIBLE is intentionally absent — the edge-gateway provides
        it based on the shift schedule (sync_turnos).
        """
        pending_micro = 0.0
        pending_stop = 0.0
        pending_mayor = 0.0
        if self._idle_in_window_s > 0:
            if self._continuous_idle_s < self._micro_stop_max_s:
                pending_micro = self._idle_in_window_s
            elif self._continuous_idle_s < self._stop_max_s:
                pending_stop = self._idle_in_window_s
            else:
                pending_mayor = self._idle_in_window_s

        result = {
            "head": [
                "T_MICROPARADA",
                "T_PARADA_NO_ASIGNADA",
                "T_PARADA_MAYOR",
                "CONTEO_1",
            ],
            "data": [
                int(self._micro_stop_s + pending_micro),
                int(self._stop_s + pending_stop),
                int(self._parada_mayor_s + pending_mayor),
                self._cut_count_1,
            ],
        }

        self._window_start = time.monotonic()
        self._next_wall_emit = self._next_boundary(time.time())
        self._micro_stop_s = 0.0
        self._stop_s = 0.0
        self._parada_mayor_s = 0.0
        self._cut_count_1 = 0
        self._idle_in_window_s = 0.0

        return result
