-- Migration 17: energy schema — medidores MEATROL MC60 via Modbus RTU
-- Aislado de OEE. Gestionado por el servicio energy-ingest (port 8086).

CREATE SCHEMA IF NOT EXISTS energy;

CREATE TABLE IF NOT EXISTS energy.meters (
    id          SERIAL PRIMARY KEY,
    device_id   TEXT        NOT NULL,
    meter_id    TEXT        NOT NULL,
    empresa_id  INTEGER,
    planta_id   INTEGER,
    nombre      TEXT,
    ubicacion   TEXT,
    activo      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ,
    CONSTRAINT uq_energy_meter UNIQUE (device_id, meter_id)
);

CREATE TABLE IF NOT EXISTS energy.snapshots (
    id                BIGSERIAL PRIMARY KEY,
    device_id         TEXT        NOT NULL,
    meter_id          TEXT        NOT NULL,
    hora              TIMESTAMPTZ NOT NULL,
    interval_s        INTEGER     NOT NULL DEFAULT 60,
    head              TEXT[],
    data              TEXT[],
    empresa_id        INTEGER,
    planta_id         INTEGER,
    corriente_a       DOUBLE PRECISION NOT NULL DEFAULT 0,
    corriente_b       DOUBLE PRECISION NOT NULL DEFAULT 0,
    corriente_c       DOUBLE PRECISION NOT NULL DEFAULT 0,
    corriente_avg     DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_a         DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_b         DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_c         DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_avg       DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_ab        DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_bc        DOUBLE PRECISION NOT NULL DEFAULT 0,
    voltaje_ac        DOUBLE PRECISION NOT NULL DEFAULT 0,
    potencia_activa   DOUBLE PRECISION NOT NULL DEFAULT 0,
    potencia_reactiva DOUBLE PRECISION NOT NULL DEFAULT 0,
    potencia_aparente DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_potencia   DOUBLE PRECISION NOT NULL DEFAULT 0,
    frecuencia_hz     DOUBLE PRECISION NOT NULL DEFAULT 0,
    energia_activa    DOUBLE PRECISION NOT NULL DEFAULT 0,
    energia_reactiva  DOUBLE PRECISION NOT NULL DEFAULT 0,
    energia_aparente  DOUBLE PRECISION NOT NULL DEFAULT 0,
    thd_ia            DOUBLE PRECISION NOT NULL DEFAULT 0,
    thd_ib            DOUBLE PRECISION NOT NULL DEFAULT 0,
    thd_ic            DOUBLE PRECISION NOT NULL DEFAULT 0,
    thd_ua            DOUBLE PRECISION NOT NULL DEFAULT 0,
    thd_ub            DOUBLE PRECISION NOT NULL DEFAULT 0,
    thd_uc            DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_energy_snapshot UNIQUE (device_id, meter_id, hora)
);

CREATE TABLE IF NOT EXISTS energy.device_sync_log (
    id         BIGSERIAL PRIMARY KEY,
    device_id  TEXT        NOT NULL,
    batch_size INTEGER     NOT NULL DEFAULT 0,
    status     TEXT        NOT NULL DEFAULT 'ok',
    error_msg  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_energy_snapshots_device_hora  ON energy.snapshots (device_id, hora DESC);
CREATE INDEX IF NOT EXISTS idx_energy_snapshots_meter_hora   ON energy.snapshots (meter_id, hora DESC);
CREATE INDEX IF NOT EXISTS idx_energy_snapshots_empresa_hora ON energy.snapshots (empresa_id, hora DESC);
CREATE INDEX IF NOT EXISTS idx_energy_snapshots_planta_hora  ON energy.snapshots (planta_id, hora DESC);

GRANT USAGE ON SCHEMA energy TO mentor;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA energy TO mentor;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA energy TO mentor;
