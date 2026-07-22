-- ============================================================
-- Template: Schema por línea dentro de BD de planta
-- Placeholder {schema} se reemplaza programáticamente (ej: linea_11)
-- Las FK a tablas de mentor_master se omiten intencionalmente
-- ya que la aislación es física por BD/schema.
-- ============================================================

CREATE SCHEMA IF NOT EXISTS {schema};

-- ── Tablas a nivel de planta (public schema, idempotentes) ───

CREATE TABLE IF NOT EXISTS public.turno_dia (
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

CREATE INDEX IF NOT EXISTS idx_pub_turno_dia_scope   ON public.turno_dia (linea_id, vigente_desde);
CREATE INDEX IF NOT EXISTS idx_pub_turno_dia_vigente ON public.turno_dia (vigente_desde);

-- ── Tablas de ingesta ────────────────────────────────────────

CREATE TABLE IF NOT EXISTS {schema}.raw_events (
    id              BIGSERIAL PRIMARY KEY,
    device_id       VARCHAR(100) NOT NULL,
    empresa_id      INT,
    planta_id       INT,
    linea_id        INT,
    event_type      VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL,
    timestamp_edge  TIMESTAMPTZ NOT NULL,
    recibido_en     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    procesado       BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_raw_device ON {schema}.raw_events(device_id);
CREATE INDEX IF NOT EXISTS idx_raw_ts     ON {schema}.raw_events(timestamp_edge DESC);
CREATE INDEX IF NOT EXISTS idx_raw_pend   ON {schema}.raw_events(procesado) WHERE NOT procesado;
CREATE UNIQUE INDEX IF NOT EXISTS uq_raw_events_device_ts ON {schema}.raw_events(device_id, timestamp_edge);

CREATE TABLE IF NOT EXISTS {schema}.oee_snapshots (
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

CREATE INDEX IF NOT EXISTS idx_oee_fecha_interval ON {schema}.oee_snapshots(fecha DESC, interval_s);
CREATE INDEX IF NOT EXISTS idx_oee_hora           ON {schema}.oee_snapshots(hora DESC);
CREATE INDEX IF NOT EXISTS idx_oee_turno_fecha    ON {schema}.oee_snapshots(turno, fecha DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora ON {schema}.oee_snapshots(device_id, hora);

-- ── Tablas analíticas ────────────────────────────────────────

-- ── Categorías de paradas (árboles separados por tipo) ──────

-- Árbol para paradas PROGRAMADAS (cargado desde Excel)
CREATE TABLE IF NOT EXISTS {schema}.cat_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,           -- slug generado: "mant-limp-ext"
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES {schema}.cat_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cat_prog_codigo
    ON {schema}.cat_programada(codigo);
CREATE INDEX IF NOT EXISTS idx_cat_prog_padre
    ON {schema}.cat_programada(padre_id);

-- Árbol para paradas NO PROGRAMADAS (cargado desde Excel)
CREATE TABLE IF NOT EXISTS {schema}.cat_no_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,           -- slug generado: "meca-correa-a"
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES {schema}.cat_no_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cat_noprog_codigo
    ON {schema}.cat_no_programada(codigo);
CREATE INDEX IF NOT EXISTS idx_cat_noprog_padre
    ON {schema}.cat_no_programada(padre_id);

-- ── Registros de paradas ─────────────────────────────────────

-- Paradas PROGRAMADAS
-- Edge llena: device_id, inicio, fin, duracion_min
-- Operador completa: categoria_id, maquina, parte_maquina, descripcion, operador_id, turno
CREATE TABLE IF NOT EXISTS {schema}.parada_programada (
    id             BIGSERIAL PRIMARY KEY,
    stop_id        UUID,                          -- UUID del edge para idempotencia
    device_id      VARCHAR(100) NOT NULL,
    inicio         TIMESTAMPTZ NOT NULL,
    fin            TIMESTAMPTZ,
    duracion_min   NUMERIC(10,2),
    -- asignación del operador (NULL hasta que se clasifique)
    categoria_id   INT REFERENCES {schema}.cat_programada(id) ON DELETE SET NULL,
    operador_id    INT,
    turno          VARCHAR(100),
    maquina        TEXT NOT NULL DEFAULT '',
    parte_maquina  TEXT NOT NULL DEFAULT '',
    descripcion    TEXT NOT NULL DEFAULT '',
    -- control
    asignado       BOOLEAN NOT NULL DEFAULT FALSE,
    asignado_en    TIMESTAMPTZ,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pp_device_inicio
    ON {schema}.parada_programada(device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pp_inicio
    ON {schema}.parada_programada(inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pp_pendiente
    ON {schema}.parada_programada(inicio DESC) WHERE asignado = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS uq_pp_stop_id
    ON {schema}.parada_programada(stop_id) WHERE stop_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pp_categoria
    ON {schema}.parada_programada(categoria_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pp_operador
    ON {schema}.parada_programada(operador_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pp_turno
    ON {schema}.parada_programada(turno, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pp_abierta
    ON {schema}.parada_programada(inicio DESC) WHERE fin IS NULL;

-- Paradas NO PROGRAMADAS
-- Misma lógica: edge llena tiempos, operador clasifica
CREATE TABLE IF NOT EXISTS {schema}.parada_no_programada (
    id             BIGSERIAL PRIMARY KEY,
    stop_id        UUID,                          -- UUID del edge para idempotencia
    device_id      VARCHAR(100) NOT NULL,
    inicio         TIMESTAMPTZ NOT NULL,
    fin            TIMESTAMPTZ,
    duracion_min   NUMERIC(10,2),
    -- asignación del operador (NULL hasta que se clasifique)
    categoria_id   INT REFERENCES {schema}.cat_no_programada(id) ON DELETE SET NULL,
    operador_id    INT,
    turno          VARCHAR(100),
    maquina        TEXT NOT NULL DEFAULT '',
    parte_maquina  TEXT NOT NULL DEFAULT '',
    descripcion    TEXT NOT NULL DEFAULT '',
    -- control
    asignado       BOOLEAN NOT NULL DEFAULT FALSE,
    asignado_en    TIMESTAMPTZ,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pnp_device_inicio
    ON {schema}.parada_no_programada(device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pnp_inicio
    ON {schema}.parada_no_programada(inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pnp_pendiente
    ON {schema}.parada_no_programada(inicio DESC) WHERE asignado = FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS uq_pnp_stop_id
    ON {schema}.parada_no_programada(stop_id) WHERE stop_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pnp_categoria
    ON {schema}.parada_no_programada(categoria_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pnp_operador
    ON {schema}.parada_no_programada(operador_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pnp_turno
    ON {schema}.parada_no_programada(turno, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_pnp_abierta
    ON {schema}.parada_no_programada(inicio DESC) WHERE fin IS NULL;

-- Vista unificada de paradas (programadas + no programadas) para cloud-analytics.
-- Escrita por cloud-ingest (upsertParadasUnified) y leída por cloud-analytics (ListStops,
-- JustifyStop, CreateStop, DeleteStop via multitenancy.Tbl(schema, "analytics.paradas")).
CREATE TABLE IF NOT EXISTS {schema}.paradas (
    id                   BIGSERIAL PRIMARY KEY,
    device_id            VARCHAR(100) NOT NULL,
    linea_id             INT,
    planta_id            INT,
    empresa_id           INT,
    categoria_id         INT,
    categoria_nombre     TEXT,
    subcategoria_nombre  TEXT,
    subcategoria_2_nombre TEXT,
    descripcion          TEXT,
    stop_id              UUID,
    stop_type            VARCHAR(50),
    inicio               TIMESTAMPTZ NOT NULL,
    fin                  TIMESTAMPTZ,
    duracion_min         NUMERIC(10,2),
    justified            BOOLEAN NOT NULL DEFAULT FALSE,
    justified_by         TEXT,
    justified_at         TIMESTAMPTZ,
    reason               TEXT,
    source               TEXT,
    creado_en            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_paradas_device_stop_id
    ON {schema}.paradas (device_id, stop_id) WHERE stop_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_paradas_device_inicio
    ON {schema}.paradas (device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_paradas_inicio
    ON {schema}.paradas (inicio DESC);
CREATE INDEX IF NOT EXISTS idx_paradas_justified
    ON {schema}.paradas (justified) WHERE justified = FALSE;
CREATE INDEX IF NOT EXISTS idx_paradas_linea_inicio
    ON {schema}.paradas (linea_id, inicio DESC);

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

CREATE INDEX IF NOT EXISTS idx_alarmas_activa ON {schema}.alarmas(activa) WHERE activa;

CREATE TABLE IF NOT EXISTS {schema}.production_runs (
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
    CONSTRAINT uq_run_id UNIQUE (device_id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_runs_started ON {schema}.production_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_active  ON {schema}.production_runs(started_at DESC) WHERE ended_at IS NULL;

-- ── Tablas de configuración por línea ────────────────────────

CREATE TABLE IF NOT EXISTS {schema}.productos (
    id        SERIAL PRIMARY KEY,
    codigo    VARCHAR(50) NOT NULL UNIQUE,
    nombre    VARCHAR(200) NOT NULL,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS {schema}.variables (
    id       SERIAL PRIMARY KEY,
    nombre   VARCHAR(200) NOT NULL,
    unidad   VARCHAR(50),
    tipo     VARCHAR(50) NOT NULL DEFAULT 'numeric',
    activo   BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS {schema}.turnos (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio TIME NOT NULL,
    hora_fin    TIME NOT NULL,
    activo      BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS {schema}.turno_dia (
    id             SERIAL PRIMARY KEY,
    dia_semana     SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6),
    nombre         VARCHAR(100) NOT NULL,
    hora_inicio    TIME NOT NULL,
    hora_fin       TIME NOT NULL,
    color          VARCHAR(20) NOT NULL DEFAULT '#6366f1',
    activo         BOOLEAN NOT NULL DEFAULT TRUE,
    renovacion_semanal BOOLEAN NOT NULL DEFAULT TRUE,
    vigente_desde  DATE NOT NULL DEFAULT CURRENT_DATE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_turno_dia_vigente ON {schema}.turno_dia(vigente_desde);

CREATE TABLE IF NOT EXISTS {schema}.velocidad_nominal (
    id            SERIAL PRIMARY KEY,
    producto_id   INT NOT NULL REFERENCES {schema}.productos(id) ON DELETE CASCADE,
    velocidad_us  DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv   INT NOT NULL DEFAULT 1,
    UNIQUE(producto_id)
);

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
    origen                TEXT NOT NULL DEFAULT 'cloud',
    cambiado_en           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_{schema}_vnlog_prod ON {schema}.velocidad_nominal_log (producto_id, cambiado_en DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_vnlog_ts   ON {schema}.velocidad_nominal_log (cambiado_en DESC);

CREATE TABLE IF NOT EXISTS {schema}.linea_producto_vars (
    id          SERIAL PRIMARY KEY,
    variable_id INT NOT NULL REFERENCES {schema}.variables(id) ON DELETE CASCADE,
    nombre_col  VARCHAR(100) NOT NULL,
    orden       INT NOT NULL DEFAULT 0,
    UNIQUE(variable_id)
);

CREATE TABLE IF NOT EXISTS {schema}.producto_caracteristicas (
    id          SERIAL PRIMARY KEY,
    producto_id INT NOT NULL REFERENCES {schema}.productos(id) ON DELETE CASCADE,
    variable_id INT NOT NULL REFERENCES {schema}.variables(id) ON DELETE CASCADE,
    valor       TEXT NOT NULL DEFAULT '',
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(producto_id, variable_id)
);

CREATE TABLE IF NOT EXISTS {schema}.linea_var_catalogo (
    id          SERIAL PRIMARY KEY,
    variable_id INT NOT NULL REFERENCES {schema}.variables(id) ON DELETE CASCADE,
    valor       VARCHAR(200) NOT NULL,
    orden       INT NOT NULL DEFAULT 0,
    UNIQUE(variable_id, valor)
);

CREATE TABLE IF NOT EXISTS {schema}.canvas_oee (
    id         SERIAL PRIMARY KEY,
    nombre     TEXT NOT NULL DEFAULT 'Formula OEE',
    grafo      JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
    activo     BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- ── Vista materializada: producción mensual ──────────────────

CREATE MATERIALIZED VIEW IF NOT EXISTS {schema}.mv_produccion_mensual AS
SELECT DATE_TRUNC('month', fecha) AS mes,
       SUM(produccion)            AS total,
       AVG(oee)                   AS oee_promedio
FROM {schema}.oee_snapshots
WHERE interval_s >= 300
GROUP BY DATE_TRUNC('month', fecha);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_prod_mes ON {schema}.mv_produccion_mensual(mes);

-- ── Seed: variables estándar de características de producto ──────────────────

INSERT INTO {schema}.variables (nombre, unidad, tipo, activo) VALUES
  ('Material', '', 'catalogo', true),
  ('Destino',  '', 'catalogo', true),
  ('Marca',    '', 'catalogo', true),
  ('Sabor',    '', 'catalogo', true),
  ('Tamaño',   '', 'catalogo', true)
ON CONFLICT DO NOTHING;

-- Configurar las 5 columnas para esta línea
INSERT INTO {schema}.linea_producto_vars (variable_id, nombre_col, orden)
SELECT id, nombre, ROW_NUMBER() OVER (ORDER BY id) FROM {schema}.variables
ON CONFLICT (variable_id) DO NOTHING;

-- ── Motivos predefinidos para cambios de velocidad nominal ───────────────────

CREATE TABLE IF NOT EXISTS {schema}.motivos_velocidad (
    id        SERIAL PRIMARY KEY,
    texto     VARCHAR(255) NOT NULL,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    orden     INT NOT NULL DEFAULT 0,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO {schema}.motivos_velocidad (texto, orden) VALUES
    ('Ajuste de velocidad de línea',   10),
    ('Cambio de formato / producto',   20),
    ('Optimización de rendimiento',    30),
    ('Corrección por calidad',         40),
    ('Instrucción de mantenimiento',   50),
    ('Orden de supervisión',           60),
    ('Calibración de equipo',          70),
    ('Otro',                           80)
ON CONFLICT DO NOTHING;


-- ── Comandos pendientes para entrega confiable al edge ───────────────────────
-- El Enviador consulta esta tabla en cada ciclo y aplica comandos no entregados.

CREATE TABLE IF NOT EXISTS {schema}.pending_commands (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    command     TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    applied     BOOLEAN NOT NULL DEFAULT FALSE,
    applied_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pc_device_applied
ON {schema}.pending_commands (device_id, applied)
WHERE applied = FALSE;
