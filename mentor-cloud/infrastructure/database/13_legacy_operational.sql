-- ============================================================
-- LEGACY: Esquemas operacionales en mentor_master
-- 
-- Estas tablas existen por compatibilidad con plantas que aun
-- no han sido migradas a BD dedicada. Cuando schema="" en
-- multitenancy.Tbl(), el codigo Go resuelve al nombre
-- calificado original (ingest.*, analytics.*).
--
-- En nuevas instalaciones estos esquemas se crean vacios;
-- los datos operacionales viven en BD de planta bajo
-- esquemas linea_N (ver 12_linea_template.sql).
--
-- Eliminar este archivo una vez que todas las plantas
-- esten provisionadas con BD dedicada.
-- ============================================================

-- -------------------------------------------------------
-- SCHEMA: ingest (legacy)
-- -------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS ingest;

CREATE TABLE IF NOT EXISTS ingest.raw_events (
    id             BIGSERIAL PRIMARY KEY,
    device_id      VARCHAR(100) NOT NULL,
    empresa_id     INT,
    planta_id      INT,
    linea_id       INT,
    event_type     VARCHAR(50) NOT NULL,
    payload        JSONB NOT NULL,
    timestamp_edge TIMESTAMPTZ NOT NULL,
    recibido_en    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    procesado      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS ingest.oee_snapshots (
    id              BIGSERIAL PRIMARY KEY,
    device_id       VARCHAR(100) NOT NULL,
    linea_id        INT,
    planta_id       INT,
    empresa_id      INT,
    turno           VARCHAR(50),
    fecha           DATE NOT NULL,
    hora            TIMESTAMPTZ NOT NULL,
    disponibilidad  NUMERIC(5,2),
    rendimiento     NUMERIC(5,2),
    calidad         NUMERIC(5,2),
    oee             NUMERIC(5,2),
    produccion      INT DEFAULT 0,
    energia_kwh     NUMERIC(10,3) DEFAULT 0,
    code            VARCHAR(50),
    interval_s      INT,
    head            TEXT[],
    data            TEXT[],
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ingest.device_sync_log (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    batch_size  INT NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'ok',
    mensaje     TEXT,
    recibido_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingest_raw_device    ON ingest.raw_events(device_id);
CREATE INDEX IF NOT EXISTS idx_ingest_raw_type      ON ingest.raw_events(event_type);
CREATE INDEX IF NOT EXISTS idx_ingest_raw_ts        ON ingest.raw_events(timestamp_edge DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_raw_pending   ON ingest.raw_events(procesado) WHERE NOT procesado;
CREATE INDEX IF NOT EXISTS idx_ingest_oee_device    ON ingest.oee_snapshots(device_id);
CREATE INDEX IF NOT EXISTS idx_ingest_oee_fecha     ON ingest.oee_snapshots(fecha DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_oee_empresa   ON ingest.oee_snapshots(empresa_id, fecha DESC);

-- -------------------------------------------------------
-- SCHEMA: analytics (legacy)
-- -------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.paradas (
    id                   BIGSERIAL PRIMARY KEY,
    device_id            VARCHAR(100) NOT NULL,
    linea_id             INT,
    planta_id            INT,
    empresa_id           INT,
    categoria_id         INT,
    categoria_nombre      TEXT,
    subcategoria_nombre   TEXT,
    subcategoria_2_nombre TEXT,
    descripcion          TEXT,
    inicio               TIMESTAMPTZ NOT NULL,
    fin                  TIMESTAMPTZ,
    duracion_min         NUMERIC(10,2),
    creado_en            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.alarmas (
    id               BIGSERIAL PRIMARY KEY,
    device_id        VARCHAR(100) NOT NULL,
    linea_id         INT,
    planta_id        INT,
    empresa_id       INT,
    tipo             VARCHAR(50) NOT NULL,
    mensaje          TEXT NOT NULL,
    severidad        VARCHAR(20) NOT NULL DEFAULT 'info',
    activa           BOOLEAN NOT NULL DEFAULT TRUE,
    timestamp_evento TIMESTAMPTZ NOT NULL,
    reconocido_en    TIMESTAMPTZ,
    creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.compromisos (
    id             SERIAL PRIMARY KEY,
    empresa_id     INT,
    planta_id      INT,
    descripcion    TEXT NOT NULL,
    responsable    VARCHAR(150),
    fecha_limite   DATE,
    estado         VARCHAR(30) NOT NULL DEFAULT 'pendiente',
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.production_runs (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL,
    device_id       VARCHAR(100) NOT NULL,
    linea_id        INT,
    planta_id       INT,
    empresa_id      INT,
    producto_id     INT,
    sku             VARCHAR(64),
    nombre          TEXT,
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_production_run_id UNIQUE (device_id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_analytics_paradas_device  ON analytics.paradas(device_id);
CREATE INDEX IF NOT EXISTS idx_analytics_paradas_inicio  ON analytics.paradas(inicio DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_alarmas_activa  ON analytics.alarmas(activa) WHERE activa;
CREATE INDEX IF NOT EXISTS idx_analytics_alarmas_empresa ON analytics.alarmas(empresa_id);
CREATE INDEX IF NOT EXISTS idx_prod_runs_device          ON analytics.production_runs(device_id);
CREATE INDEX IF NOT EXISTS idx_prod_runs_linea_ts        ON analytics.production_runs(linea_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_prod_runs_empresa_ts      ON analytics.production_runs(empresa_id, started_at DESC);

-- Performance indexes (legacy)
CREATE INDEX IF NOT EXISTS idx_ingest_oee_linea          ON ingest.oee_snapshots(linea_id, fecha DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_oee_planta         ON ingest.oee_snapshots(planta_id, fecha DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_oee_empresa_linea  ON ingest.oee_snapshots(empresa_id, linea_id, fecha DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_oee_hora           ON ingest.oee_snapshots(hora DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_raw_empresa_ts     ON ingest.raw_events(empresa_id, timestamp_edge DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_raw_linea_type     ON ingest.raw_events(linea_id, event_type, timestamp_edge DESC);
