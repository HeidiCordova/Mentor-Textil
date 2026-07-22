CREATE UNIQUE INDEX IF NOT EXISTS idx_prod_one_open_per_device
    ON production_runs (device_id, COALESCE(linea_id, -1))
    WHERE ended_at IS NULL;
