-- =============================================================
-- mentor_edge — Template UNIFICADO de schema por línea
-- El placeholder {schema} se reemplaza con linea_{id}.
-- Ejemplo: linea_3 → todas las tablas en linea_3.*
--
-- Invocar desde init_line.sh:
--   sed 's/{schema}/linea_3/g' linea_template.sql | psql ...
--
-- Contiene TODAS las tablas del edge: buffer, stops, runs,
-- config, comandos, auditoría, health y catálogos sync.
-- =============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS {schema};

-- ═══════════════════════════════════════════════════════════════
-- 1. CONFIGURACIÓN DE LÍNEA
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.line_config (
    id              SERIAL PRIMARY KEY,
    device_id       VARCHAR(64) UNIQUE NOT NULL,
    config_version  INTEGER NOT NULL DEFAULT 1,
    roi             JSONB NOT NULL DEFAULT '{}',
    thresholds      JSONB NOT NULL DEFAULT '{}',
    fsm             JSONB NOT NULL DEFAULT '{}',
    mode            VARCHAR(16) NOT NULL DEFAULT 'textil',
    camera          JSONB,
    oee             JSONB NOT NULL DEFAULT '{"line_name":"","micro_stop_max_s":210,"stop_max_s":300,"snapshot_interval_s":300}',
    cloud           JSONB NOT NULL DEFAULT '{"sync_interval_s":300}',
    tablet          JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION {schema}.update_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.config_version = OLD.config_version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS config_update_trigger ON {schema}.line_config;
CREATE TRIGGER config_update_trigger
BEFORE UPDATE ON {schema}.line_config
FOR EACH ROW
EXECUTE FUNCTION {schema}.update_config_timestamp();

-- ═══════════════════════════════════════════════════════════════
-- 2. COMANDOS RECIBIDOS DEL CLOUD
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.commands_buffer (
    id              SERIAL PRIMARY KEY,
    command_id      UUID UNIQUE NOT NULL,
    device_id       VARCHAR(64) NOT NULL,
    command_type    VARCHAR(64) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(16) NOT NULL DEFAULT 'RECEIVED',
    issued_by       VARCHAR(128),
    issued_at       TIMESTAMPTZ DEFAULT NOW(),
    idempotency_key VARCHAR(255),
    result          JSONB,
    error_message   TEXT,
    applied_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT chk_{schema}_cmd_status CHECK (status IN ('RECEIVED','APPLIED','FAILED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_{schema}_cmd_idempotency
    ON {schema}.commands_buffer (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_{schema}_commands_device
    ON {schema}.commands_buffer (device_id, issued_at DESC);

-- ═══════════════════════════════════════════════════════════════
-- 3. AUDITORÍA
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.audit_log (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(64),
    actor       VARCHAR(128),
    action      VARCHAR(64) NOT NULL,
    resource    VARCHAR(64),
    resource_id VARCHAR(128),
    payload     JSONB,
    result      VARCHAR(32),
    timestamp   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_audit_device_ts
    ON {schema}.audit_log (device_id, timestamp DESC);
ALTER TABLE {schema}.audit_log SET (autovacuum_vacuum_scale_factor = 0.1);

-- ═══════════════════════════════════════════════════════════════
-- 4. HEALTH & CALIBRACIÓN
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.health_logs (
    id        SERIAL PRIMARY KEY,
    service   VARCHAR(64) NOT NULL,
    status    VARCHAR(16) NOT NULL,
    metrics   JSONB,
    errors    JSONB,
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_health_svc_ts
    ON {schema}.health_logs (service, timestamp);

CREATE TABLE IF NOT EXISTS {schema}.calibration_history (
    id               SERIAL PRIMARY KEY,
    device_id        VARCHAR(64) NOT NULL,
    calibration_type VARCHAR(32) NOT NULL,
    parameters       JSONB NOT NULL,
    result           JSONB,
    success          BOOLEAN,
    timestamp        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_calibration_dev_ts
    ON {schema}.calibration_history (device_id, timestamp);

-- ═══════════════════════════════════════════════════════════════
-- 5. BUFFER DE EVENTOS DE PRODUCCIÓN
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.events_buffer (
    id          BIGSERIAL  PRIMARY KEY,
    event_id    UUID UNIQUE NOT NULL,
    device_id   VARCHAR(100),
    event_type  VARCHAR(32) NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    synced      BOOLEAN DEFAULT FALSE,
    dead        BOOLEAN DEFAULT FALSE,
    retry_count INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    synced_at   TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '6 months')
);

CREATE INDEX IF NOT EXISTS idx_{schema}_events_pending
    ON {schema}.events_buffer (timestamp ASC)
    WHERE synced = false AND dead = false;
CREATE INDEX IF NOT EXISTS idx_{schema}_events_synced
    ON {schema}.events_buffer (synced, timestamp);
CREATE INDEX IF NOT EXISTS idx_{schema}_events_expiry
    ON {schema}.events_buffer (expires_at)
    WHERE synced = true OR dead = true;

ALTER TABLE {schema}.events_buffer SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE {schema}.events_buffer SET (autovacuum_analyze_scale_factor = 0.02);

-- ═══════════════════════════════════════════════════════════════
-- 6. PARADAS (STOPS)
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.stops (
    id              BIGSERIAL  PRIMARY KEY,
    stop_id         UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    device_id       VARCHAR(64) NOT NULL DEFAULT '',
    stop_type       VARCHAR(32) NOT NULL DEFAULT 'PARADA_NO_ASIGNADA',
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    duration_ms     INTEGER,
    justified       BOOLEAN NOT NULL DEFAULT FALSE,
    reason          TEXT,
    category        VARCHAR(64),
    categoria_id    INTEGER,
    justified_by    VARCHAR(128),
    justified_at    TIMESTAMPTZ,
    source          VARCHAR(32) NOT NULL DEFAULT 'detector',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    synced          BOOLEAN DEFAULT FALSE,
    synced_at       TIMESTAMPTZ,

    CONSTRAINT chk_{schema}_stop_type CHECK (stop_type IN (
        'MICROPARADA', 'PARADA_NO_ASIGNADA', 'PROGRAMADA', 'NO_PROGRAMADA',
        'MECANICA', 'ELECTRICA', 'CAMBIO_FORMATO',
        'FALTA_MATERIAL', 'CALIDAD', 'REFRIGERIO',
        'CAPACITACION', 'MANTENIMIENTO', 'OTRA'
    ))
);

CREATE INDEX IF NOT EXISTS idx_{schema}_stops_ts
    ON {schema}.stops (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_stops_open
    ON {schema}.stops (started_at)
    WHERE ended_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_{schema}_stops_unjustified
    ON {schema}.stops (started_at DESC)
    WHERE justified = false AND ended_at IS NOT NULL;

ALTER TABLE {schema}.stops SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE {schema}.stops SET (autovacuum_analyze_scale_factor = 0.02);

CREATE OR REPLACE FUNCTION {schema}.update_stop_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.ended_at IS NOT NULL AND OLD.ended_at IS NULL THEN
        NEW.duration_ms = EXTRACT(EPOCH FROM (NEW.ended_at - NEW.started_at)) * 1000;
    END IF;
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_{schema}_stop_duration ON {schema}.stops;
CREATE TRIGGER trg_{schema}_stop_duration
BEFORE UPDATE ON {schema}.stops
FOR EACH ROW
EXECUTE FUNCTION {schema}.update_stop_duration();

-- ═══════════════════════════════════════════════════════════════
-- 7. PRODUCTION RUNS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.production_runs (
    id              BIGSERIAL  PRIMARY KEY,
    run_id          UUID NOT NULL DEFAULT uuid_generate_v4(),
    device_id       VARCHAR(100) NOT NULL DEFAULT '',
    linea_id        INTEGER,
    producto_id     INTEGER,
    sku             VARCHAR(64),
    nombre          TEXT,
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    synced          BOOLEAN DEFAULT FALSE,
    synced_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_run_time
    ON {schema}.production_runs (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_run_pending
    ON {schema}.production_runs (synced)
    WHERE synced = false;

-- ═══════════════════════════════════════════════════════════════
-- 8. OEE SNAPSHOTS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.oee_snapshots (
    id             BIGSERIAL PRIMARY KEY,
    device_id      VARCHAR(100),
    turno          VARCHAR(50),
    fecha          DATE        NOT NULL,
    hora           TIMESTAMPTZ NOT NULL,
    disponibilidad NUMERIC(5,2),
    rendimiento    NUMERIC(5,2),
    calidad        NUMERIC(5,2),
    oee            NUMERIC(5,2),
    produccion     INT DEFAULT 0,
    energia_kwh    NUMERIC(10,3) DEFAULT 0,
    code           VARCHAR(50),
    interval_s     INT,
    head           TEXT[],
    data           BIGINT[],
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_oee_fecha ON {schema}.oee_snapshots (fecha DESC, interval_s);
CREATE INDEX IF NOT EXISTS idx_{schema}_oee_hora  ON {schema}.oee_snapshots (hora DESC);

-- ═══════════════════════════════════════════════════════════════
-- 9. VISION DETECTIONS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.vision_detections (
    id           BIGSERIAL   PRIMARY KEY,
    detection_id UUID        NOT NULL,
    detected_at  TIMESTAMPTZ NOT NULL,
    line_code    VARCHAR(64),
    confidence   REAL,
    signal_edge  REAL,
    signal_color REAL,
    signal_flow  REAL,
    signal_beige REAL,
    roi_id       VARCHAR(32),
    CONSTRAINT {schema}_vision_det_uq UNIQUE (detection_id)
);

CREATE INDEX IF NOT EXISTS idx_{schema}_vd_ts
    ON {schema}.vision_detections (detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_vd_conf
    ON {schema}.vision_detections (confidence);

ALTER TABLE {schema}.vision_detections SET (autovacuum_vacuum_scale_factor = 0.05);

CREATE OR REPLACE FUNCTION {schema}.extract_vision_detection()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.event_type <> 'CORTE' THEN
        RETURN NEW;
    END IF;
    INSERT INTO {schema}.vision_detections (
        detection_id, detected_at, line_code,
        confidence, signal_edge, signal_color, signal_flow, signal_beige, roi_id
    ) VALUES (
        NEW.event_id, NEW.timestamp,
        NEW.payload ->> 'line_code',
        (NEW.payload ->> 'confidence')::REAL,
        (NEW.payload -> 'signals' ->> 'edge')::REAL,
        (NEW.payload -> 'signals' ->> 'color')::REAL,
        (NEW.payload -> 'signals' ->> 'flow')::REAL,
        (NEW.payload -> 'signals' ->> 'beige')::REAL,
        NEW.payload ->> 'roi_id'
    )
    ON CONFLICT (detection_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_{schema}_extract_vision ON {schema}.events_buffer;
CREATE TRIGGER trg_{schema}_extract_vision
AFTER INSERT ON {schema}.events_buffer
FOR EACH ROW EXECUTE FUNCTION {schema}.extract_vision_detection();

-- ═══════════════════════════════════════════════════════════════
-- 10. CATÁLOGOS — Réplica exacta de tablas cloud (12_linea_template)
--     + columnas de contexto sync (synced_at, empresa_id, linea_id)
-- ═══════════════════════════════════════════════════════════════

-- ── Categorías de paradas (cloud: cat_programada + cat_no_programada) ──

CREATE TABLE IF NOT EXISTS {schema}.cat_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES {schema}.cat_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_{schema}_cat_prog_codigo
    ON {schema}.cat_programada(codigo);
CREATE INDEX IF NOT EXISTS idx_{schema}_cat_prog_padre
    ON {schema}.cat_programada(padre_id);

CREATE TABLE IF NOT EXISTS {schema}.cat_no_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES {schema}.cat_no_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_{schema}_cat_noprog_codigo
    ON {schema}.cat_no_programada(codigo);
CREATE INDEX IF NOT EXISTS idx_{schema}_cat_noprog_padre
    ON {schema}.cat_no_programada(padre_id);

-- ── Paradas programadas y no programadas (cloud: parada_programada / parada_no_programada) ──

CREATE TABLE IF NOT EXISTS {schema}.parada_programada (
    id             BIGSERIAL PRIMARY KEY,
    stop_id        UUID,
    device_id      VARCHAR(100) NOT NULL,
    inicio         TIMESTAMPTZ NOT NULL,
    fin            TIMESTAMPTZ,
    duracion_min   NUMERIC(10,2),
    categoria_id   INT REFERENCES {schema}.cat_programada(id) ON DELETE SET NULL,
    operador_id    INT,
    turno          VARCHAR(100),
    maquina        TEXT NOT NULL DEFAULT '',
    parte_maquina  TEXT NOT NULL DEFAULT '',
    descripcion    TEXT NOT NULL DEFAULT '',
    asignado       BOOLEAN NOT NULL DEFAULT FALSE,
    asignado_en    TIMESTAMPTZ,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_pp_device_inicio
    ON {schema}.parada_programada(device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_pp_pendiente
    ON {schema}.parada_programada(inicio DESC) WHERE asignado = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS uq_{schema}_pp_stop_id
    ON {schema}.parada_programada(stop_id) WHERE stop_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS {schema}.parada_no_programada (
    id             BIGSERIAL PRIMARY KEY,
    stop_id        UUID,
    device_id      VARCHAR(100) NOT NULL,
    inicio         TIMESTAMPTZ NOT NULL,
    fin            TIMESTAMPTZ,
    duracion_min   NUMERIC(10,2),
    categoria_id   INT REFERENCES {schema}.cat_no_programada(id) ON DELETE SET NULL,
    operador_id    INT,
    turno          VARCHAR(100),
    maquina        TEXT NOT NULL DEFAULT '',
    parte_maquina  TEXT NOT NULL DEFAULT '',
    descripcion    TEXT NOT NULL DEFAULT '',
    asignado       BOOLEAN NOT NULL DEFAULT FALSE,
    asignado_en    TIMESTAMPTZ,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_pnp_device_inicio
    ON {schema}.parada_no_programada(device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_pnp_pendiente
    ON {schema}.parada_no_programada(inicio DESC) WHERE asignado = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS uq_{schema}_pnp_stop_id
    ON {schema}.parada_no_programada(stop_id) WHERE stop_id IS NOT NULL;

-- ── Alarmas (cloud: alarmas) ──

CREATE TABLE IF NOT EXISTS {schema}.alarmas (
    id               BIGSERIAL PRIMARY KEY,
    device_id        VARCHAR(100) NOT NULL,
    tipo             VARCHAR(50) NOT NULL,
    mensaje          TEXT NOT NULL,
    severidad        VARCHAR(20) NOT NULL DEFAULT 'info',
    activa           BOOLEAN NOT NULL DEFAULT TRUE,
    timestamp_evento TIMESTAMPTZ NOT NULL,
    reconocido_en    TIMESTAMPTZ,
    creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_alarmas_activa ON {schema}.alarmas(activa) WHERE activa;

-- ── Productos (cloud: productos) ──

CREATE TABLE IF NOT EXISTS {schema}.productos (
    id           INTEGER      PRIMARY KEY,
    codigo       VARCHAR(50)  NOT NULL,
    nombre       VARCHAR(200) NOT NULL,
    activo       BOOLEAN      NOT NULL DEFAULT TRUE,
    creado_en    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- Columnas de contexto sync
    empresa_id   INTEGER,
    linea_id     INTEGER,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER      NOT NULL DEFAULT 1,
    synced_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_prod_empresa ON {schema}.productos (empresa_id);
CREATE INDEX IF NOT EXISTS idx_{schema}_prod_linea   ON {schema}.productos (linea_id);

-- ── Variables de producto (cloud: variables) ──

CREATE TABLE IF NOT EXISTS {schema}.variables_producto (
    id       SERIAL PRIMARY KEY,
    nombre   VARCHAR(200) NOT NULL,
    unidad   VARCHAR(50),
    tipo     VARCHAR(50) NOT NULL DEFAULT 'numeric',
    activo   BOOLEAN NOT NULL DEFAULT TRUE
);

-- ── Variables de configuración OEE (cloud: config.variables) ──

CREATE TABLE IF NOT EXISTS {schema}.variables (
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

CREATE INDEX IF NOT EXISTS idx_{schema}_variables_empresa ON {schema}.variables (empresa_id);
CREATE INDEX IF NOT EXISTS idx_{schema}_variables_disp    ON {schema}.variables (dispositivo_id);

-- ── Turnos (cloud: turnos) ──

CREATE TABLE IF NOT EXISTS {schema}.turnos (
    id          INTEGER      PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio VARCHAR(20)  NOT NULL,
    hora_fin    VARCHAR(20)  NOT NULL,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    creado_en   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- Columnas de contexto sync
    planta_id   INTEGER,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_turnos_activo ON {schema}.turnos (activo);

-- ── Turno por día de semana (cloud: turno_dia) ──

CREATE TABLE IF NOT EXISTS {schema}.turno_dia (
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

CREATE INDEX IF NOT EXISTS idx_{schema}_turno_dia_scope   ON {schema}.turno_dia (linea_id, vigente_desde);
CREATE INDEX IF NOT EXISTS idx_{schema}_turno_dia_vigente ON {schema}.turno_dia (vigente_desde);

-- ── Usuarios (cloud: identity.usuarios — tabla master) ──

CREATE TABLE IF NOT EXISTS {schema}.usuarios (
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

CREATE INDEX IF NOT EXISTS idx_{schema}_usuarios_empresa ON {schema}.usuarios (empresa_id);

-- ── Velocidad nominal (cloud: velocidad_nominal) ──

CREATE TABLE IF NOT EXISTS {schema}.velocidad_nominal (
    id           SERIAL  PRIMARY KEY,
    producto_id  INTEGER NOT NULL,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER NOT NULL DEFAULT 1,
    -- Columna de contexto sync
    linea_id     INTEGER NOT NULL,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (linea_id, producto_id)
);

CREATE INDEX IF NOT EXISTS idx_{schema}_vn_linea ON {schema}.velocidad_nominal (linea_id);

CREATE TABLE IF NOT EXISTS {schema}.velocidad_nominal_log (
    id                    BIGSERIAL PRIMARY KEY,
    producto_id           INT NOT NULL,
    sku                   TEXT,
    velocidad_us_anterior DOUBLE PRECISION,
    velocidad_us_nueva    DOUBLE PRECISION NOT NULL,
    factor_conv_anterior  INT,
    factor_conv_nueva     INT NOT NULL DEFAULT 1,
    motivo                TEXT,
    usuario               TEXT,
    origen                TEXT NOT NULL DEFAULT 'edge',
    cambiado_en           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_vnlog_prod ON {schema}.velocidad_nominal_log (producto_id, cambiado_en DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_vnlog_ts   ON {schema}.velocidad_nominal_log (cambiado_en DESC);

-- ── Linea producto vars (cloud: linea_producto_vars) ──

CREATE TABLE IF NOT EXISTS {schema}.linea_producto_vars (
    id          INTEGER PRIMARY KEY,
    variable_id INTEGER NOT NULL,
    nombre_col  VARCHAR(100) NOT NULL,
    orden       INTEGER NOT NULL DEFAULT 0,
    -- Columna de contexto sync
    linea_id    INTEGER NOT NULL,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(linea_id, variable_id)
);

CREATE INDEX IF NOT EXISTS idx_{schema}_lpv_linea ON {schema}.linea_producto_vars (linea_id);

-- ── Producto características (cloud: producto_caracteristicas) ──

CREATE TABLE IF NOT EXISTS {schema}.producto_caracteristicas (
    id          INTEGER PRIMARY KEY,
    producto_id INTEGER NOT NULL,
    variable_id INTEGER NOT NULL,
    valor       TEXT    NOT NULL DEFAULT '',
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Columna de contexto sync
    linea_id    INTEGER NOT NULL,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(producto_id, linea_id, variable_id)
);

CREATE INDEX IF NOT EXISTS idx_{schema}_pc_linea ON {schema}.producto_caracteristicas (linea_id, producto_id);

-- ── Catálogo de valores dropdown (cloud: linea_var_catalogo) ──

CREATE TABLE IF NOT EXISTS {schema}.linea_var_catalogo (
    id          SERIAL PRIMARY KEY,
    variable_id INT NOT NULL,
    valor       VARCHAR(200) NOT NULL,
    orden       INT NOT NULL DEFAULT 0,
    UNIQUE(variable_id, valor)
);

-- ── Canvas OEE (cloud: canvas_oee) ──

CREATE TABLE IF NOT EXISTS {schema}.canvas_oee (
    id         SERIAL PRIMARY KEY,
    nombre     TEXT NOT NULL DEFAULT 'Formula OEE',
    grafo      JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
    activo     BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- ── Plantas (cloud: config.plantas — tabla master) ──

CREATE TABLE IF NOT EXISTS {schema}.plantas (
    id             INTEGER      PRIMARY KEY,
    nombre         VARCHAR(200) NOT NULL,
    empresa_id     INTEGER      NOT NULL,
    empresa_nombre VARCHAR(200) NOT NULL DEFAULT '',
    activo         BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_plantas_empresa ON {schema}.plantas (empresa_id);

-- ── Líneas (cloud: config.lineas — tabla master) ──

CREATE TABLE IF NOT EXISTS {schema}.lineas (
    id        INTEGER      PRIMARY KEY,
    nombre    VARCHAR(200) NOT NULL,
    planta_id INTEGER      NOT NULL,
    tipo      VARCHAR(50)  NOT NULL DEFAULT '',
    subtipo   VARCHAR(50)  NOT NULL DEFAULT '',
    activo    BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_lineas_planta ON {schema}.lineas (planta_id);

-- ═══════════════════════════════════════════════════════════════
-- 11. FUNCIONES DE MANTENIMIENTO DEL BUFFER
-- ═══════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION {schema}.purge_expired_events() RETURNS INTEGER AS $$
DECLARE removed INTEGER;
BEGIN
    DELETE FROM {schema}.events_buffer
    WHERE expires_at < NOW() AND (synced = true OR dead = true);
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION {schema}.mark_stale_events_dead(max_age_hours INTEGER DEFAULT 48)
RETURNS INTEGER AS $$
DECLARE affected INTEGER;
BEGIN
    UPDATE {schema}.events_buffer
    SET dead = true
    WHERE synced = false AND dead = false
      AND created_at < (NOW() - make_interval(hours => max_age_hours));
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION {schema}.emergency_purge_synced(keep_count INTEGER DEFAULT 10000)
RETURNS INTEGER AS $$
DECLARE removed INTEGER; total INTEGER;
BEGIN
    SELECT COUNT(*) INTO total FROM {schema}.events_buffer WHERE synced = true;
    IF total <= keep_count THEN RETURN 0; END IF;
    DELETE FROM {schema}.events_buffer
    WHERE id IN (
        SELECT id FROM {schema}.events_buffer
        WHERE synced = true ORDER BY timestamp ASC
        LIMIT (total - keep_count)
    );
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$ LANGUAGE plpgsql;

-- ═══════════════════════════════════════════════════════════════
-- MOTIVOS PREDEFINIDOS PARA CAMBIOS DE VELOCIDAD NOMINAL
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.motivos_velocidad (
    id        SERIAL PRIMARY KEY,
    texto     VARCHAR(255) NOT NULL,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    orden     INT NOT NULL DEFAULT 0,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO {schema}.motivos_velocidad (texto, orden)
SELECT v.texto, v.orden
FROM (VALUES
    ('Ajuste de velocidad de línea',   10),
    ('Cambio de formato / producto',   20),
    ('Optimización de rendimiento',    30),
    ('Corrección por calidad',         40),
    ('Instrucción de mantenimiento',   50),
    ('Orden de supervisión',           60),
    ('Calibración de equipo',          70),
    ('Otro',                           80)
) AS v(texto, orden)
WHERE NOT EXISTS (SELECT 1 FROM {schema}.motivos_velocidad LIMIT 1);
