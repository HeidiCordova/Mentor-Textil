-- Index for fast per-meter queries (used by views and energy-sender).
CREATE INDEX IF NOT EXISTS idx_energy_snap_meter_hora
    ON energy.snapshots (meter_id, hora DESC);
