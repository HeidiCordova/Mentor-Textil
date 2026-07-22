-- Mentor Edge Textil - PostgreSQL Schema
-- Target: Jetson Orin / ARM64

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE events_buffer (
    id          SERIAL PRIMARY KEY,
    event_id    UUID UNIQUE NOT NULL,
    device_id   VARCHAR(64) NOT NULL,
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

CREATE INDEX idx_events_pending ON events_buffer (timestamp ASC)
    WHERE synced = false AND dead = false;
CREATE INDEX idx_events_synced ON events_buffer (synced, timestamp);
CREATE INDEX idx_events_device_ts ON events_buffer (device_id, timestamp);
CREATE INDEX idx_events_expiry ON events_buffer (expires_at)
    WHERE synced = true OR dead = true;

ALTER TABLE events_buffer SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE events_buffer SET (autovacuum_analyze_scale_factor = 0.02);

CREATE TABLE line_config (
    id              SERIAL PRIMARY KEY,
    device_id       VARCHAR(64) UNIQUE NOT NULL,
    config_version  INTEGER NOT NULL DEFAULT 1,
    roi             JSONB NOT NULL,
    thresholds      JSONB NOT NULL,
    fsm             JSONB NOT NULL,
    mode            VARCHAR(16) NOT NULL DEFAULT 'textil',
    camera          JSONB,
    oee             JSONB NOT NULL DEFAULT '{"line_name":"","micro_stop_max_s":210,"stop_max_s":300,"snapshot_interval_s":300}',
    cloud           JSONB NOT NULL DEFAULT '{"sync_interval_s":300}',
    tablet          JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE health_logs (
    id        SERIAL PRIMARY KEY,
    service   VARCHAR(64) NOT NULL,
    status    VARCHAR(16) NOT NULL,
    metrics   JSONB,
    errors    JSONB,
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_health_svc_ts ON health_logs (service, timestamp);

CREATE TABLE calibration_history (
    id               SERIAL PRIMARY KEY,
    device_id        VARCHAR(64) NOT NULL,
    calibration_type VARCHAR(32) NOT NULL,
    parameters       JSONB NOT NULL,
    result           JSONB,
    success          BOOLEAN,
    timestamp        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_calibration_dev_ts ON calibration_history (device_id, timestamp);

CREATE OR REPLACE FUNCTION update_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.config_version = OLD.config_version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER config_update_trigger
BEFORE UPDATE ON line_config
FOR EACH ROW
EXECUTE FUNCTION update_config_timestamp();

-- ====================================================================
-- Buffer maintenance functions
-- ====================================================================

CREATE OR REPLACE FUNCTION purge_expired_events() RETURNS INTEGER AS $$
DECLARE
    removed INTEGER;
BEGIN
    DELETE FROM events_buffer
    WHERE expires_at < NOW()
      AND (synced = true OR dead = true);
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION mark_stale_events_dead(max_age_hours INTEGER DEFAULT 48) RETURNS INTEGER AS $$
DECLARE
    affected INTEGER;
BEGIN
    UPDATE events_buffer
    SET dead = true
    WHERE synced = false
      AND dead = false
      AND created_at < (NOW() - make_interval(hours => max_age_hours));
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION emergency_purge_synced(keep_count INTEGER DEFAULT 10000) RETURNS INTEGER AS $$
DECLARE
    removed INTEGER;
    total   INTEGER;
BEGIN
    SELECT COUNT(*) INTO total FROM events_buffer WHERE synced = true;
    IF total <= keep_count THEN
        RETURN 0;
    END IF;
    DELETE FROM events_buffer
    WHERE id IN (
        SELECT id FROM events_buffer
        WHERE synced = true
        ORDER BY timestamp ASC
        LIMIT (total - keep_count)
    );
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION resurrect_dead_events() RETURNS INTEGER AS $$
DECLARE
    affected INTEGER;
BEGIN
    UPDATE events_buffer
    SET dead = false, retry_count = 0
    WHERE dead = true;
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_buffer_stats()
RETURNS TABLE(
    total_count    BIGINT,
    pending_count  BIGINT,
    synced_count   BIGINT,
    dead_count     BIGINT,
    oldest_pending TIMESTAMPTZ,
    newest_pending TIMESTAMPTZ,
    disk_bytes     BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT,
        COUNT(*) FILTER (WHERE eb.synced = false AND eb.dead = false)::BIGINT,
        COUNT(*) FILTER (WHERE eb.synced = true)::BIGINT,
        COUNT(*) FILTER (WHERE eb.dead = true)::BIGINT,
        MIN(eb.timestamp) FILTER (WHERE eb.synced = false AND eb.dead = false),
        MAX(eb.timestamp) FILTER (WHERE eb.synced = false AND eb.dead = false),
        pg_total_relation_size('events_buffer')::BIGINT
    FROM events_buffer eb;
END;
$$ LANGUAGE plpgsql;

-- ====================================================================
-- Stops: paradas individuales detectadas o registradas manualmente
-- ====================================================================

CREATE TABLE stops (
    id              SERIAL PRIMARY KEY,
    stop_id         UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    device_id       VARCHAR(64) NOT NULL,
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

    CONSTRAINT chk_stop_type CHECK (stop_type IN (
        'MICROPARADA', 'PARADA_NO_ASIGNADA', 'PROGRAMADA', 'NO_PROGRAMADA',
        'MECANICA', 'ELECTRICA', 'CAMBIO_FORMATO',
        'FALTA_MATERIAL', 'CALIDAD', 'REFRIGERIO',
        'CAPACITACION', 'MANTENIMIENTO', 'OTRA'
    )),
    CONSTRAINT chk_source CHECK (source IN ('detector', 'operator', 'cloud', 'system')),
    CONSTRAINT chk_duration CHECK (duration_ms IS NULL OR duration_ms >= 0)
);

CREATE INDEX idx_stops_device_ts ON stops (device_id, started_at DESC);
CREATE INDEX idx_stops_open ON stops (device_id, started_at)
    WHERE ended_at IS NULL;
CREATE INDEX idx_stops_unjustified ON stops (device_id, started_at)
    WHERE justified = FALSE AND ended_at IS NOT NULL;
CREATE INDEX idx_stops_synced ON stops (synced, started_at)
    WHERE synced = FALSE;

ALTER TABLE stops SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE stops SET (autovacuum_analyze_scale_factor = 0.02);

CREATE OR REPLACE FUNCTION update_stop_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    IF NEW.ended_at IS NOT NULL AND OLD.ended_at IS NULL THEN
        NEW.duration_ms = EXTRACT(EPOCH FROM (NEW.ended_at - NEW.started_at)) * 1000;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stop_update_trigger
BEFORE UPDATE ON stops
FOR EACH ROW
EXECUTE FUNCTION update_stop_timestamp();

-- ====================================================================
-- Production runs: corridas de producción (orden de trabajo)
-- ====================================================================

CREATE TABLE IF NOT EXISTS production_runs (
    id          SERIAL PRIMARY KEY,
    run_id      UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
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

-- Un solo run abierto por dispositivo/línea a la vez
CREATE UNIQUE INDEX IF NOT EXISTS idx_prod_one_open_per_device
    ON production_runs (device_id, COALESCE(linea_id, -1))
    WHERE ended_at IS NULL;

-- ====================================================================
-- Commands buffer: command bus idempotente para cloud y tablet
-- ====================================================================

CREATE TABLE commands_buffer (
    id              SERIAL PRIMARY KEY,
    command_id      UUID UNIQUE NOT NULL DEFAULT uuid_generate_v4(),
    device_id       VARCHAR(64) NOT NULL,
    command_type    VARCHAR(64) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    issued_by       VARCHAR(128) NOT NULL,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    idempotency_key VARCHAR(256) UNIQUE NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'RECEIVED',
    result          JSONB,
    error_message   TEXT,
    applied_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT chk_command_status CHECK (status IN ('RECEIVED', 'APPLIED', 'FAILED')),
    CONSTRAINT chk_command_type CHECK (command_type IN (
        'CREAR_PARADA', 'MODIFICAR_PARADA', 'JUSTIFICAR_PARADA',
        'CERRAR_PARADA', 'ELIMINAR_PARADA',
        'ACTUALIZAR_CONFIG', 'INICIAR_CALIBRACION',
        'REINICIAR_PIPELINE', 'COMANDO_CUSTOM',
        'SYNC_CATALOG', 'SYNC_PRODUCTOS', 'SYNC_TURNOS',
        'SYNC_USUARIOS', 'SYNC_VARIABLES',
        'SYNC_LINEA_PRODUCTO_VARS', 'SYNC_PRODUCTO_CARACTERISTICAS',
        'SYNC_PLANTAS_LINEAS', 'SYNC_PARADAS', 'SYNC_VELOCIDAD_NOMINAL'
    ))
);

CREATE INDEX idx_commands_device ON commands_buffer (device_id, issued_at DESC);
CREATE INDEX idx_commands_status ON commands_buffer (status, issued_at)
    WHERE status = 'RECEIVED';
CREATE INDEX idx_commands_idemp ON commands_buffer (idempotency_key);

-- ====================================================================
-- Audit log: registro inmutable de acciones
-- ====================================================================

CREATE TABLE audit_log (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR(64) NOT NULL,
    actor       VARCHAR(128) NOT NULL,
    action      VARCHAR(64) NOT NULL,
    resource    VARCHAR(64) NOT NULL,
    resource_id VARCHAR(128),
    payload     JSONB,
    result      VARCHAR(16) NOT NULL DEFAULT 'OK',
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_audit_result CHECK (result IN ('OK', 'ERROR', 'DENIED'))
);

CREATE INDEX idx_audit_device_ts ON audit_log (device_id, timestamp DESC);
CREATE INDEX idx_audit_actor ON audit_log (actor, timestamp DESC);

ALTER TABLE audit_log SET (autovacuum_vacuum_scale_factor = 0.1);

-- ====================================================================
-- NOTIFY triggers for SSE real-time streaming
-- ====================================================================

CREATE OR REPLACE FUNCTION notify_event_created()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('event_created', json_build_object(
        'event_id', NEW.event_id,
        'device_id', NEW.device_id,
        'event_type', NEW.event_type,
        'timestamp', NEW.timestamp
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notify_event
AFTER INSERT ON events_buffer
FOR EACH ROW
EXECUTE FUNCTION notify_event_created();

CREATE OR REPLACE FUNCTION notify_stop_changed()
RETURNS TRIGGER AS $$
DECLARE
    op TEXT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        op := 'created';
    ELSE
        op := 'updated';
    END IF;
    PERFORM pg_notify('stop_changed', json_build_object(
        'op', op,
        'stop_id', NEW.stop_id,
        'device_id', NEW.device_id,
        'stop_type', NEW.stop_type,
        'started_at', NEW.started_at,
        'ended_at', NEW.ended_at,
        'justified', NEW.justified
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notify_stop
AFTER INSERT OR UPDATE ON stops
FOR EACH ROW
EXECUTE FUNCTION notify_stop_changed();

CREATE OR REPLACE FUNCTION notify_config_updated()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('config_updated', json_build_object(
        'device_id', NEW.device_id,
        'config_version', NEW.config_version,
        'updated_at', NEW.updated_at
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notify_config
AFTER UPDATE ON line_config
FOR EACH ROW
EXECUTE FUNCTION notify_config_updated();

CREATE OR REPLACE FUNCTION notify_command_changed()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('command_changed', json_build_object(
        'command_id', NEW.command_id,
        'device_id', NEW.device_id,
        'command_type', NEW.command_type,
        'status', NEW.status,
        'issued_by', NEW.issued_by
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notify_command
AFTER INSERT OR UPDATE ON commands_buffer
FOR EACH ROW
EXECUTE FUNCTION notify_command_changed();

-- ====================================================================
-- Stops maintenance functions
-- ====================================================================

CREATE OR REPLACE FUNCTION close_stale_stops(max_open_hours INTEGER DEFAULT 24) RETURNS INTEGER AS $$
DECLARE
    affected INTEGER;
BEGIN
    UPDATE stops
    SET ended_at = NOW()
    WHERE ended_at IS NULL
      AND started_at < (NOW() - make_interval(hours => max_open_hours));
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_stops_summary(p_device_id VARCHAR, p_hours INTEGER DEFAULT 24)
RETURNS TABLE(
    total_stops      BIGINT,
    open_stops       BIGINT,
    justified_stops  BIGINT,
    unjustified_stops BIGINT,
    total_downtime_ms BIGINT,
    by_type          JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT,
        COUNT(*) FILTER (WHERE s.ended_at IS NULL)::BIGINT,
        COUNT(*) FILTER (WHERE s.justified = TRUE)::BIGINT,
        COUNT(*) FILTER (WHERE s.justified = FALSE AND s.ended_at IS NOT NULL)::BIGINT,
        COALESCE(SUM(s.duration_ms) FILTER (WHERE s.duration_ms IS NOT NULL), 0)::BIGINT,
        COALESCE(jsonb_object_agg(s.stop_type, s.cnt) FILTER (WHERE s.stop_type IS NOT NULL), '{}'::jsonb)
    FROM (
        SELECT st.*, COUNT(*) OVER (PARTITION BY st.stop_type) as cnt
        FROM stops st
        WHERE st.device_id = p_device_id
          AND st.started_at >= (NOW() - make_interval(hours => p_hours))
    ) s;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION purge_old_commands(keep_days INTEGER DEFAULT 90) RETURNS INTEGER AS $$
DECLARE
    removed INTEGER;
BEGIN
    DELETE FROM commands_buffer
    WHERE issued_at < (NOW() - make_interval(days => keep_days))
      AND status IN ('APPLIED', 'FAILED');
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION purge_old_audit(keep_days INTEGER DEFAULT 180) RETURNS INTEGER AS $$
DECLARE
    removed INTEGER;
BEGIN
    DELETE FROM audit_log
    WHERE timestamp < (NOW() - make_interval(days => keep_days));
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$ LANGUAGE plpgsql;

-- ====================================================================
-- Seed data: se ejecuta desde seed.sh con las variables del .env
-- (ver 02_seed.sh)
-- ====================================================================

-- ====================================================================
-- Tablas replica para sincronizacion de catalogo desde el cloud
-- ====================================================================

CREATE TABLE IF NOT EXISTS sync_categoria_paradas (
    id                  BIGINT       PRIMARY KEY,
    nombre              VARCHAR(200) NOT NULL,
    codigo              TEXT         NOT NULL DEFAULT '',
    padre_id            BIGINT,
    empresa_id          INTEGER,
    linea_id            INTEGER,
    orden               INTEGER      NOT NULL DEFAULT 0,
    tipo_parada         VARCHAR(30),
    descripcion_parada  VARCHAR(300) DEFAULT '',
    maquina             VARCHAR(200) DEFAULT '',
    parte_maquina       VARCHAR(200) DEFAULT '',
    area_responsable    VARCHAR(200) DEFAULT '',
    synced_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_productos (
    id          INTEGER      PRIMARY KEY,
    codigo      VARCHAR(50)  NOT NULL,
    nombre      VARCHAR(200) NOT NULL,
    empresa_id  INTEGER,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    linea_id    INTEGER,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER      NOT NULL DEFAULT 1,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_turnos (
    id          INTEGER      PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio VARCHAR(20)  NOT NULL,
    hora_fin    VARCHAR(20)  NOT NULL,
    planta_id   INTEGER,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_usuarios (
    id          INTEGER      PRIMARY KEY,
    username    VARCHAR(100) NOT NULL,
    email       VARCHAR(150) NOT NULL DEFAULT '',
    nombre      VARCHAR(200) NOT NULL,
    rol_id      INTEGER,
    rol         VARCHAR(50)  NOT NULL DEFAULT 'operador',
    empresa_id  INTEGER,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_variables (
    id              INTEGER      PRIMARY KEY,
    nombre          VARCHAR(200) NOT NULL,
    clave           VARCHAR(100) NOT NULL,
    valor           TEXT,
    tipo            VARCHAR(50),
    dispositivo_id  INTEGER,
    planta_id       INTEGER,
    empresa_id      INTEGER,
    activo          BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_linea_producto_vars (
    id          INTEGER      PRIMARY KEY,
    linea_id    INTEGER      NOT NULL,
    variable_id INTEGER      NOT NULL,
    nombre_col  VARCHAR(100) NOT NULL,
    orden       INTEGER      NOT NULL DEFAULT 0,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(linea_id, variable_id)
);

CREATE TABLE IF NOT EXISTS sync_producto_caracteristicas (
    id          INTEGER      PRIMARY KEY,
    producto_id INTEGER      NOT NULL,
    linea_id    INTEGER      NOT NULL,
    variable_id INTEGER      NOT NULL,
    valor       TEXT         NOT NULL DEFAULT '',
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(producto_id, linea_id, variable_id)
);

-- Plantas y líneas sincronizadas desde el cloud (scoped por empresa)
CREATE TABLE IF NOT EXISTS sync_plantas (
    id              INTEGER      PRIMARY KEY,
    nombre          VARCHAR(200) NOT NULL,
    empresa_id      INTEGER      NOT NULL,
    empresa_nombre  VARCHAR(200) NOT NULL DEFAULT '',
    activo          BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync_lineas (
    id          INTEGER      PRIMARY KEY,
    nombre      VARCHAR(200) NOT NULL,
    planta_id   INTEGER      NOT NULL,
    tipo        VARCHAR(50)  NOT NULL DEFAULT '',
    subtipo     VARCHAR(50)  NOT NULL DEFAULT '',
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_cat_empresa ON sync_categoria_paradas (empresa_id);
CREATE INDEX IF NOT EXISTS idx_sync_cat_linea ON sync_categoria_paradas (linea_id);
CREATE INDEX IF NOT EXISTS idx_sync_prod_empresa ON sync_productos (empresa_id);
CREATE INDEX IF NOT EXISTS idx_sync_prod_linea ON sync_productos (linea_id);
CREATE INDEX IF NOT EXISTS idx_sync_turnos_activo ON sync_turnos (activo);
CREATE INDEX IF NOT EXISTS idx_sync_usuarios_empresa ON sync_usuarios (empresa_id);
CREATE INDEX IF NOT EXISTS idx_sync_variables_empresa ON sync_variables (empresa_id);
CREATE INDEX IF NOT EXISTS idx_sync_variables_dispositivo ON sync_variables (dispositivo_id);
CREATE INDEX IF NOT EXISTS idx_sync_lpv_linea ON sync_linea_producto_vars (linea_id);
CREATE INDEX IF NOT EXISTS idx_sync_pc_linea ON sync_producto_caracteristicas (linea_id, producto_id);
CREATE INDEX IF NOT EXISTS idx_sync_plantas_empresa ON sync_plantas (empresa_id);
CREATE INDEX IF NOT EXISTS idx_sync_lineas_planta ON sync_lineas (planta_id);

CREATE TABLE IF NOT EXISTS sync_velocidad_nominal (
    id          SERIAL       PRIMARY KEY,
    linea_id    INTEGER      NOT NULL,
    producto_id INTEGER      NOT NULL,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER     NOT NULL DEFAULT 1,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (linea_id, producto_id)
);
CREATE INDEX IF NOT EXISTS idx_sync_vn_linea ON sync_velocidad_nominal (linea_id);

-- ====================================================================
-- Vision detections: audit trail estructurado de cada corte detectado
-- Poblado automáticamente via trigger sobre events_buffer (event_type='CORTE')
-- ====================================================================

CREATE TABLE IF NOT EXISTS vision_detections (
    id             SERIAL       PRIMARY KEY,
    detection_id   UUID         NOT NULL,
    device_id      VARCHAR(64)  NOT NULL,
    detected_at    TIMESTAMPTZ  NOT NULL,
    line_code      VARCHAR(64),
    confidence     REAL,
    signal_edge    REAL,
    signal_color   REAL,
    signal_flow    REAL,
    signal_beige   REAL,
    roi_id         VARCHAR(32),

    CONSTRAINT vision_detections_detection_id_key UNIQUE (detection_id)
);

CREATE INDEX idx_vd_device_ts  ON vision_detections (device_id, detected_at DESC);
CREATE INDEX idx_vd_line_ts    ON vision_detections (line_code, detected_at DESC);
CREATE INDEX idx_vd_confidence ON vision_detections (confidence);

ALTER TABLE vision_detections SET (autovacuum_vacuum_scale_factor = 0.05);

CREATE OR REPLACE FUNCTION extract_vision_detection()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.event_type <> 'CORTE' THEN
        RETURN NEW;
    END IF;

    INSERT INTO vision_detections (
        detection_id, device_id, detected_at, line_code,
        confidence, signal_edge, signal_color, signal_flow, signal_beige, roi_id
    )
    VALUES (
        NEW.event_id,
        NEW.device_id,
        NEW.timestamp,
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

CREATE TRIGGER trg_extract_vision_detection
AFTER INSERT ON events_buffer
FOR EACH ROW
EXECUTE FUNCTION extract_vision_detection();
