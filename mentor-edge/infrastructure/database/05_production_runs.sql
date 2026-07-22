CREATE TABLE IF NOT EXISTS production_runs (
    id          SERIAL PRIMARY KEY,
    run_id      UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    device_id   VARCHAR(64) NOT NULL,
    linea_id    INTEGER,
    producto_id INTEGER,
    sku         VARCHAR(64),
    nombre      TEXT,
    started_at  TIMESTAMPTZ NOT NULL,
    ended_at    TIMESTAMPTZ,
    synced      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prod_runs_device_time ON production_runs (device_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_prod_runs_unsynced    ON production_runs (synced, started_at) WHERE synced = FALSE;
CREATE INDEX IF NOT EXISTS idx_prod_runs_linea       ON production_runs (linea_id, started_at DESC) WHERE linea_id IS NOT NULL;
