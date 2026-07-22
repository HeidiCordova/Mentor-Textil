-- Energy monitoring schema (isolated from OEE/production data)

CREATE SCHEMA IF NOT EXISTS energy;

CREATE TABLE IF NOT EXISTS energy.meters (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    meter_id    VARCHAR(100) NOT NULL,
    nombre      VARCHAR(200),
    ubicacion   VARCHAR(200),
    planta_id   INT,
    empresa_id  INT,
    activo      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ,
    UNIQUE (device_id, meter_id)
);

CREATE TABLE IF NOT EXISTS energy.snapshots (
    id                  BIGSERIAL PRIMARY KEY,
    device_id           VARCHAR(100) NOT NULL,
    meter_id            VARCHAR(100) NOT NULL,
    planta_id           INT,
    empresa_id          INT,
    hora                TIMESTAMPTZ NOT NULL,
    interval_s          INT NOT NULL DEFAULT 60,
    head                TEXT[],
    data                TEXT[],
    corriente_a         NUMERIC(10,3),
    corriente_b         NUMERIC(10,3),
    corriente_c         NUMERIC(10,3),
    corriente_avg       NUMERIC(10,3),
    voltaje_a           NUMERIC(10,3),
    voltaje_b           NUMERIC(10,3),
    voltaje_c           NUMERIC(10,3),
    voltaje_avg         NUMERIC(10,3),
    voltaje_ab          NUMERIC(10,3),
    voltaje_bc          NUMERIC(10,3),
    voltaje_ac          NUMERIC(10,3),
    potencia_activa     NUMERIC(14,3),
    potencia_reactiva   NUMERIC(14,3),
    potencia_aparente   NUMERIC(14,3),
    factor_potencia     NUMERIC(5,3),
    frecuencia_hz       NUMERIC(6,3),
    energia_activa      NUMERIC(14,3),
    energia_reactiva    NUMERIC(14,3),
    energia_aparente    NUMERIC(14,3),
    thd_ia              NUMERIC(5,2),
    thd_ib              NUMERIC(5,2),
    thd_ic              NUMERIC(5,2),
    thd_ua              NUMERIC(5,2),
    thd_ub              NUMERIC(5,2),
    thd_uc              NUMERIC(5,2),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, meter_id, hora)
);

CREATE INDEX IF NOT EXISTS idx_energy_snap_empresa   ON energy.snapshots (empresa_id, hora DESC);
CREATE INDEX IF NOT EXISTS idx_energy_snap_planta    ON energy.snapshots (planta_id, hora DESC);
CREATE INDEX IF NOT EXISTS idx_energy_snap_device    ON energy.snapshots (device_id, meter_id, hora DESC);

CREATE TABLE IF NOT EXISTS energy.device_sync_log (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    batch_size  INT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'ok',
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
