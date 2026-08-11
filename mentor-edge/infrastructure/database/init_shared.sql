-- =============================================================
-- mentor_edge — Schema SHARED
-- Datos del dispositivo: comandos, salud, auditoría, configuración
-- de línea y catálogos sincronizados desde el cloud.
-- Se ejecuta UNA VEZ al levantar por primera vez el Jetson.
-- =============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE SCHEMA IF NOT EXISTS shared;

-- ─── Configuración de línea (una fila por DEVICE_ID) ─────────────────

CREATE TABLE IF NOT EXISTS shared.line_config (
    id              SERIAL PRIMARY KEY,
    device_id       VARCHAR(64) UNIQUE NOT NULL,
    config_version  INTEGER NOT NULL DEFAULT 1,
    roi             JSONB NOT NULL DEFAULT '{}',
    thresholds      JSONB NOT NULL DEFAULT '{}',
    fsm             JSONB NOT NULL DEFAULT '{}',
    mode            VARCHAR(16) NOT NULL DEFAULT 'textil',
    camera          JSONB,
    oee             JSONB NOT NULL DEFAULT '{"line_name":"","micro_stop_max_s":120,"stop_max_s":86400,"snapshot_interval_s":1800,"vel_unit":"uh","vel_nominal_us":0.008333333}',
    cloud           JSONB NOT NULL DEFAULT '{"sync_interval_s":300}',
    tablet          JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION shared.update_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.config_version = OLD.config_version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER config_update_trigger
BEFORE UPDATE ON shared.line_config
FOR EACH ROW
EXECUTE FUNCTION shared.update_config_timestamp();

-- ─── Comandos recibidos del cloud ─────────────────────────────────────

CREATE TABLE IF NOT EXISTS shared.commands_buffer (
    id              SERIAL PRIMARY KEY,
    command_id      UUID UNIQUE NOT NULL,
    device_id       VARCHAR(64) NOT NULL,
    command_type    VARCHAR(64) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    idempotency_key VARCHAR(255),
    issued_by       VARCHAR(128),
    issued_at       TIMESTAMPTZ DEFAULT NOW(),
    executed_at     TIMESTAMPTZ,
    fail_reason     TEXT,

    CONSTRAINT chk_cmd_status CHECK (status IN ('pending','running','done','failed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cmd_idempotency
    ON shared.commands_buffer (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_commands_device
    ON shared.commands_buffer (device_id, issued_at DESC);

-- ─── Health logs ──────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS shared.health_logs (
    id        SERIAL PRIMARY KEY,
    service   VARCHAR(64) NOT NULL,
    status    VARCHAR(16) NOT NULL,
    metrics   JSONB,
    errors    JSONB,
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_health_svc_ts ON shared.health_logs (service, timestamp);

-- ─── Calibration history ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS shared.calibration_history (
    id               SERIAL PRIMARY KEY,
    device_id        VARCHAR(64) NOT NULL,
    calibration_type VARCHAR(32) NOT NULL,
    parameters       JSONB NOT NULL,
    result           JSONB,
    success          BOOLEAN,
    timestamp        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_calibration_dev_ts
    ON shared.calibration_history (device_id, timestamp);

-- ─── Audit log ────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS shared.audit_log (
    id          BIGSERIAL  PRIMARY KEY,
    event_type  VARCHAR(64) NOT NULL,
    device_id   VARCHAR(64),
    user_id     VARCHAR(128),
    payload     JSONB,
    timestamp   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_device_ts
    ON shared.audit_log (device_id, timestamp DESC);
ALTER TABLE shared.audit_log SET (autovacuum_vacuum_scale_factor = 0.1);

-- ─── Catálogos — Réplica de tablas cloud + columnas de contexto sync ──

-- Categorías de paradas programadas (cloud: cat_programada)
CREATE TABLE IF NOT EXISTS shared.cat_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES shared.cat_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_shared_cat_prog_codigo
    ON shared.cat_programada(codigo);
CREATE INDEX IF NOT EXISTS idx_shared_cat_prog_padre
    ON shared.cat_programada(padre_id);

-- Categorías de paradas no programadas (cloud: cat_no_programada)
CREATE TABLE IF NOT EXISTS shared.cat_no_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES shared.cat_no_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_shared_cat_noprog_codigo
    ON shared.cat_no_programada(codigo);
CREATE INDEX IF NOT EXISTS idx_shared_cat_noprog_padre
    ON shared.cat_no_programada(padre_id);

-- Productos (cloud: productos + velocidad_nominal denormalizado)
CREATE TABLE IF NOT EXISTS shared.productos (
    id           INTEGER      PRIMARY KEY,
    codigo       VARCHAR(50)  NOT NULL,
    nombre       VARCHAR(200) NOT NULL,
    activo       BOOLEAN      NOT NULL DEFAULT TRUE,
    creado_en    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    empresa_id   INTEGER,
    linea_id     INTEGER,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER      NOT NULL DEFAULT 1,
    synced_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Turnos (cloud: turnos)
CREATE TABLE IF NOT EXISTS shared.turnos (
    id          INTEGER      PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio VARCHAR(20)  NOT NULL,
    hora_fin    VARCHAR(20)  NOT NULL,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    creado_en   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    planta_id   INTEGER,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Usuarios (cloud: identity.usuarios)
CREATE TABLE IF NOT EXISTS shared.usuarios (
    id            INTEGER      PRIMARY KEY,
    username      VARCHAR(100) NOT NULL,
    email         VARCHAR(150) NOT NULL DEFAULT '',
    nombre        VARCHAR(200) NOT NULL,
    apellido      VARCHAR(100) NOT NULL DEFAULT '',
    password_hash TEXT         NOT NULL DEFAULT '',
    rol_id        INTEGER,
    rol           VARCHAR(50)  NOT NULL DEFAULT 'operador',
    empresa_id    INTEGER,
    activo        BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Variables de configuración OEE (cloud: config.variables)
CREATE TABLE IF NOT EXISTS shared.variables (
    id             INTEGER      PRIMARY KEY,
    nombre         VARCHAR(200) NOT NULL,
    clave          VARCHAR(100) NOT NULL,
    valor          TEXT,
    tipo           VARCHAR(50),
    dispositivo_id INTEGER,
    planta_id      INTEGER,
    empresa_id     INTEGER,
    activo         BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Variables de producto (cloud: {schema}.variables)
CREATE TABLE IF NOT EXISTS shared.variables_producto (
    id       SERIAL PRIMARY KEY,
    nombre   VARCHAR(200) NOT NULL,
    unidad   VARCHAR(50),
    tipo     VARCHAR(50) NOT NULL DEFAULT 'numeric',
    activo   BOOLEAN NOT NULL DEFAULT TRUE
);

-- Turno por día de semana (cloud: turno_dia)
CREATE TABLE IF NOT EXISTS shared.turno_dia (
    id                 SERIAL PRIMARY KEY,
    linea_id           INT,
    dia_semana         SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6),
    nombre             VARCHAR(100) NOT NULL,
    hora_inicio        TIME NOT NULL,
    hora_fin           TIME NOT NULL,
    color              VARCHAR(20) NOT NULL DEFAULT '#6366f1',
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    renovacion_semanal BOOLEAN NOT NULL DEFAULT TRUE,
    vigente_desde      DATE NOT NULL DEFAULT CURRENT_DATE,
    creado_en          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Velocidad nominal (cloud: velocidad_nominal)
CREATE TABLE IF NOT EXISTS shared.velocidad_nominal (
    id           SERIAL  PRIMARY KEY,
    producto_id  INTEGER NOT NULL,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER NOT NULL DEFAULT 1,
    linea_id     INTEGER NOT NULL,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (linea_id, producto_id)
);

-- Linea producto vars (cloud: linea_producto_vars)
CREATE TABLE IF NOT EXISTS shared.linea_producto_vars (
    id          INTEGER PRIMARY KEY,
    variable_id INTEGER NOT NULL,
    nombre_col  VARCHAR(100) NOT NULL,
    orden       INTEGER NOT NULL DEFAULT 0,
    linea_id    INTEGER NOT NULL,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(linea_id, variable_id)
);

-- Producto características (cloud: producto_caracteristicas)
CREATE TABLE IF NOT EXISTS shared.producto_caracteristicas (
    id          INTEGER PRIMARY KEY,
    producto_id INTEGER NOT NULL,
    variable_id INTEGER NOT NULL,
    valor       TEXT    NOT NULL DEFAULT '',
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linea_id    INTEGER NOT NULL,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(producto_id, linea_id, variable_id)
);

-- Catálogo de valores dropdown (cloud: linea_var_catalogo)
CREATE TABLE IF NOT EXISTS shared.linea_var_catalogo (
    id          SERIAL PRIMARY KEY,
    variable_id INT NOT NULL,
    valor       VARCHAR(200) NOT NULL,
    orden       INT NOT NULL DEFAULT 0,
    UNIQUE(variable_id, valor)
);

-- Canvas OEE (cloud: canvas_oee)
CREATE TABLE IF NOT EXISTS shared.canvas_oee (
    id         SERIAL PRIMARY KEY,
    nombre     TEXT NOT NULL DEFAULT 'Formula OEE',
    grafo      JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
    activo     BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Plantas (cloud: config.plantas)
CREATE TABLE IF NOT EXISTS shared.plantas (
    id             INTEGER      PRIMARY KEY,
    nombre         VARCHAR(200) NOT NULL,
    empresa_id     INTEGER      NOT NULL,
    empresa_nombre VARCHAR(200) NOT NULL DEFAULT '',
    activo         BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Líneas (cloud: config.lineas)
CREATE TABLE IF NOT EXISTS shared.lineas (
    id        INTEGER      PRIMARY KEY,
    nombre    VARCHAR(200) NOT NULL,
    planta_id INTEGER      NOT NULL,
    tipo      VARCHAR(50)  NOT NULL DEFAULT '',
    subtipo   VARCHAR(50)  NOT NULL DEFAULT '',
    activo    BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_prod_empresa ON shared.productos (empresa_id);
CREATE INDEX IF NOT EXISTS idx_prod_linea   ON shared.productos (linea_id);
CREATE INDEX IF NOT EXISTS idx_turnos_activo ON shared.turnos (activo);
CREATE INDEX IF NOT EXISTS idx_usuarios_empresa ON shared.usuarios (empresa_id);
CREATE INDEX IF NOT EXISTS idx_variables_empresa ON shared.variables (empresa_id);
CREATE INDEX IF NOT EXISTS idx_variables_disp ON shared.variables (dispositivo_id);
CREATE INDEX IF NOT EXISTS idx_lpv_linea    ON shared.linea_producto_vars (linea_id);
CREATE INDEX IF NOT EXISTS idx_pc_linea     ON shared.producto_caracteristicas (linea_id, producto_id);
CREATE INDEX IF NOT EXISTS idx_plantas_empresa ON shared.plantas (empresa_id);
CREATE INDEX IF NOT EXISTS idx_lineas_planta ON shared.lineas (planta_id);
CREATE INDEX IF NOT EXISTS idx_vn_linea     ON shared.velocidad_nominal (linea_id);
CREATE INDEX IF NOT EXISTS idx_turno_dia_scope ON shared.turno_dia (linea_id, vigente_desde);
