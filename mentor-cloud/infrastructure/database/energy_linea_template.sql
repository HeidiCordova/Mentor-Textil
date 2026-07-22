-- ============================================================
-- Template: Schema de energía por línea dentro de BD de planta
-- Placeholder {schema} se reemplaza programáticamente (ej: linea_16)
-- Para líneas de tipo "Energía" — medidores MEATROL MC60 y similares
-- ============================================================

CREATE SCHEMA IF NOT EXISTS {schema};

CREATE TABLE IF NOT EXISTS {schema}.snapshots (
    id                  BIGSERIAL PRIMARY KEY,
    device_id           VARCHAR(100) NOT NULL,
    meter_id            VARCHAR(100) NOT NULL,
    hora                TIMESTAMPTZ NOT NULL,
    interval_s          INT NOT NULL DEFAULT 60,
    head                TEXT[],
    data                TEXT[],
    corriente_a         NUMERIC(10,3) NOT NULL DEFAULT 0,
    corriente_b         NUMERIC(10,3) NOT NULL DEFAULT 0,
    corriente_c         NUMERIC(10,3) NOT NULL DEFAULT 0,
    corriente_avg       NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_a           NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_b           NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_c           NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_avg         NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_ab          NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_bc          NUMERIC(10,3) NOT NULL DEFAULT 0,
    voltaje_ac          NUMERIC(10,3) NOT NULL DEFAULT 0,
    potencia_activa     NUMERIC(14,3) NOT NULL DEFAULT 0,
    potencia_reactiva   NUMERIC(14,3) NOT NULL DEFAULT 0,
    potencia_aparente   NUMERIC(14,3) NOT NULL DEFAULT 0,
    factor_potencia     NUMERIC(5,3) NOT NULL DEFAULT 0,
    frecuencia_hz       NUMERIC(6,3) NOT NULL DEFAULT 0,
    energia_activa      NUMERIC(14,3) NOT NULL DEFAULT 0,
    energia_reactiva    NUMERIC(14,3) NOT NULL DEFAULT 0,
    energia_aparente    NUMERIC(14,3) NOT NULL DEFAULT 0,
    thd_ia              NUMERIC(5,2) NOT NULL DEFAULT 0,
    thd_ib              NUMERIC(5,2) NOT NULL DEFAULT 0,
    thd_ic              NUMERIC(5,2) NOT NULL DEFAULT 0,
    thd_ua              NUMERIC(5,2) NOT NULL DEFAULT 0,
    thd_ub              NUMERIC(5,2) NOT NULL DEFAULT 0,
    thd_uc              NUMERIC(5,2) NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, meter_id, hora)
);

CREATE TABLE IF NOT EXISTS {schema}.meters (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    meter_id    VARCHAR(100) NOT NULL,
    nombre      VARCHAR(200),
    ubicacion   VARCHAR(200),
    activo      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ,
    UNIQUE (device_id, meter_id)
);

CREATE TABLE IF NOT EXISTS {schema}.device_sync_log (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    batch_size  INT NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'ok',
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_snap_device ON {schema}.snapshots (device_id, meter_id, hora DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_snap_hora   ON {schema}.snapshots (hora DESC);
