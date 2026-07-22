-- Local energy backup on Raspberry Pi
-- Configuracion operacional en BB.DD. — sin .env de negocio, sin recargar Docker.

CREATE SCHEMA IF NOT EXISTS energy;

-- ─── Configuracion operacional ───────────────────────────────────────────────
-- Solo POSTGRES_URL y PORT viven en env (infra inmutable).
-- Todo lo demas se actualiza aqui en caliente; energy-sender recarga cada 60s.
CREATE TABLE IF NOT EXISTS energy.config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Valores por defecto — editar via psql
INSERT INTO energy.config (key, value) VALUES
    ('device_id',        ''),
    ('meter_id_1',       ''),
    ('meter_unit_id',    '1'),
    ('cloud_url',        'https://mentormonitor-ai.com'),
    ('energy_api_key',   ''),
    ('send_interval_s',  '30'),
    ('batch_size',       '50'),
    ('config_reload_s',  '60'),
    ('max_workers',      '4')
ON CONFLICT (key) DO NOTHING;

-- ─── Snapshots ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS energy.snapshots (
    id                  BIGSERIAL PRIMARY KEY,
    device_id           VARCHAR(100) NOT NULL,
    meter_id            VARCHAR(100) NOT NULL,
    hora                TIMESTAMPTZ NOT NULL,
    interval_s          INT NOT NULL DEFAULT 60,
    head                TEXT[],
    data                TEXT[],
    synced              BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, meter_id, hora)
);

CREATE INDEX IF NOT EXISTS idx_energy_snap_synced       ON energy.snapshots (synced) WHERE NOT synced;
CREATE INDEX IF NOT EXISTS idx_energy_snap_meter_synced ON energy.snapshots (meter_id, synced) WHERE NOT synced;
CREATE INDEX IF NOT EXISTS idx_energy_snap_hora         ON energy.snapshots (hora DESC);
