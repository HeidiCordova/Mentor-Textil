CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- -------------------------------------------------------
-- SCHEMA: identity
-- -------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS identity;

CREATE TABLE identity.roles (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR(50) NOT NULL UNIQUE,
    descripcion TEXT,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.empresas (
    id             SERIAL PRIMARY KEY,
    nombre         VARCHAR(200) NOT NULL,
    ruc            VARCHAR(20) NOT NULL UNIQUE,
    direccion      TEXT,
    telefono       VARCHAR(30),
    email          VARCHAR(150),
    responsable    VARCHAR(150),
    estado         BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.usuarios (
    id             SERIAL PRIMARY KEY,
    username       VARCHAR(100) NOT NULL UNIQUE,
    email          VARCHAR(150) NOT NULL UNIQUE,
    nombre         VARCHAR(150) NOT NULL,
    password_hash  TEXT NOT NULL,
    rol_id         INT REFERENCES identity.roles(id) ON DELETE SET NULL,
    empresa_id     INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    activo         BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    usuario_id INT NOT NULL REFERENCES identity.usuarios(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revocado   BOOLEAN NOT NULL DEFAULT FALSE,
    creado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.api_keys (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nombre     VARCHAR(100) NOT NULL,
    key_hash   TEXT NOT NULL UNIQUE,
    empresa_id INT REFERENCES identity.empresas(id) ON DELETE CASCADE,
    activo     BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identity_refresh_usuario ON identity.refresh_tokens(usuario_id);
CREATE INDEX idx_identity_apikeys_hash    ON identity.api_keys(key_hash);

-- -------------------------------------------------------
-- SCHEMA: config
-- -------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS config;

CREATE TABLE config.plantas (
    id             SERIAL PRIMARY KEY,
    nombre         VARCHAR(200) NOT NULL,
    empresa_id     INT NOT NULL REFERENCES identity.empresas(id) ON DELETE CASCADE,
    lineas         INT NOT NULL DEFAULT 0,
    activo         BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE config.lineas (
    id         SERIAL PRIMARY KEY,
    nombre     VARCHAR(200) NOT NULL,
    planta_id  INT NOT NULL REFERENCES config.plantas(id) ON DELETE CASCADE,
    mode       VARCHAR(20) NOT NULL DEFAULT 'textil'
               CONSTRAINT lineas_mode_textil_check CHECK (mode = 'textil'),
    activo     BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE config.dispositivos (
    id             SERIAL PRIMARY KEY,
    device_id      VARCHAR(100) NOT NULL UNIQUE,
    nombre         VARCHAR(200) NOT NULL,
    linea_id       INT REFERENCES config.lineas(id) ON DELETE SET NULL,
    planta_id      INT REFERENCES config.plantas(id) ON DELETE SET NULL,
    empresa_id     INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    estado         VARCHAR(30) NOT NULL DEFAULT 'offline',
    ultimo_ping    TIMESTAMPTZ,
    activo         BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE config.variables (
    id              SERIAL PRIMARY KEY,
    nombre          VARCHAR(200) NOT NULL,
    clave           VARCHAR(100) NOT NULL,
    valor           TEXT NOT NULL,
    tipo            VARCHAR(30) NOT NULL DEFAULT 'string',
    dispositivo_id  INT REFERENCES config.dispositivos(id) ON DELETE CASCADE,
    planta_id       INT REFERENCES config.plantas(id) ON DELETE SET NULL,
    empresa_id      INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    activo          BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(clave, dispositivo_id)
);

CREATE TABLE config.device_configs (
    id             SERIAL PRIMARY KEY,
    dispositivo_id INT NOT NULL REFERENCES config.dispositivos(id) ON DELETE CASCADE,
    version        INT NOT NULL DEFAULT 1,
    payload        JSONB NOT NULL DEFAULT '{}',
    aplicado       BOOLEAN NOT NULL DEFAULT FALSE,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE config.turnos (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio TIME NOT NULL,
    hora_fin    TIME NOT NULL,
    planta_id   INT REFERENCES config.plantas(id) ON DELETE CASCADE,
    activo      BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE config.productos (
    id         SERIAL PRIMARY KEY,
    codigo     VARCHAR(50) NOT NULL UNIQUE,
    nombre     VARCHAR(200) NOT NULL,
    empresa_id INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    activo     BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE config.categoria_paradas (
    id                 BIGSERIAL PRIMARY KEY,
    nombre             VARCHAR(200) NOT NULL,
    codigo             TEXT,
    padre_id           INT REFERENCES config.categoria_paradas(id) ON DELETE SET NULL,
    empresa_id         INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    linea_id           INT REFERENCES config.lineas(id) ON DELETE CASCADE,
    orden              INT NOT NULL DEFAULT 0,
    tipo_parada        TEXT,
    descripcion_parada TEXT NOT NULL DEFAULT '',
    maquina            TEXT NOT NULL DEFAULT '',
    parte_maquina      TEXT NOT NULL DEFAULT '',
    area_responsable   TEXT NOT NULL DEFAULT '',
    creado_en          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_config_dispositivos_device ON config.dispositivos(device_id);
CREATE INDEX idx_config_device_configs_pend ON config.device_configs(dispositivo_id, aplicado);
CREATE INDEX idx_config_variables_device    ON config.variables(dispositivo_id);
CREATE INDEX idx_config_plantas_empresa     ON config.plantas(empresa_id);

-- -------------------------------------------------------
-- SEED
-- -------------------------------------------------------
INSERT INTO identity.roles (nombre, descripcion) VALUES
    ('ADMIN',      'Administrador del sistema'),
    ('SUPERVISOR', 'Supervisor de planta'),
    ('OPERATOR',   'Operador de linea'),
    ('VIEWER',     'Solo lectura')
ON CONFLICT (nombre) DO NOTHING;

INSERT INTO identity.empresas (nombre, ruc, direccion, email, responsable)
VALUES ('Art Atlas', '20456879901', 'Av. Textil 123, Lima', 'admin@artatlas.pe', 'Admin Textil')
ON CONFLICT (ruc) DO NOTHING;

INSERT INTO config.plantas (nombre, empresa_id, lineas) VALUES
    ('Planta Textil Lima',  1, 3),
    ('Planta Textil Norte', 1, 2)
ON CONFLICT DO NOTHING;

INSERT INTO config.lineas (nombre, planta_id) VALUES
    ('Linea Corte',    1),
    ('Linea Costura',  1),
    ('Linea Acabado',  1),
    ('Linea Tenido',   2),
    ('Linea Empaque',  2)
ON CONFLICT DO NOTHING;

INSERT INTO config.turnos (nombre, hora_inicio, hora_fin, planta_id) VALUES
    ('Turno 1', '06:00', '14:00', 1),
    ('Turno 2', '14:00', '22:00', 1),
    ('Turno 3', '22:00', '06:00', 1),
    ('Turno 1', '06:00', '18:00', 2),
    ('Turno 2', '18:00', '06:00', 2)
ON CONFLICT DO NOTHING;

INSERT INTO config.productos (codigo, nombre, empresa_id) VALUES
    ('TEX-POLO',   'Polo de algodon', 1),
    ('TEX-CAMISA', 'Camisa manga larga', 1),
    ('TEX-PANT',   'Pantalon de drill', 1)
ON CONFLICT (codigo) DO NOTHING;

INSERT INTO identity.usuarios (username, email, nombre, password_hash, rol_id, empresa_id)
VALUES (
    'admin',
    'admin@mentormonitor.com',
    'Administrador',
    crypt('Admin1234!', gen_salt('bf', 12)),
    (SELECT id FROM identity.roles WHERE nombre = 'ADMIN'),
    (SELECT id FROM identity.empresas WHERE ruc = '20456879901')
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO identity.usuarios (username, email, nombre, password_hash, rol_id, empresa_id)
VALUES (
    'supervisor.textil',
    'supervisor@artatlas.pe',
    'Supervisor Textil',
    crypt('Super1234!', gen_salt('bf', 12)),
    (SELECT id FROM identity.roles WHERE nombre = 'SUPERVISOR'),
    1
)
ON CONFLICT (email) DO NOTHING;

-- -------------------------------------------------------
-- SCHEMA: gateway
-- -------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS gateway;

CREATE TABLE gateway.audit_log (
    id         BIGSERIAL PRIMARY KEY,
    ts         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    method     TEXT NOT NULL,
    path       TEXT NOT NULL,
    status     INT  NOT NULL,
    latency_ms INT  NOT NULL,
    ip         TEXT,
    user_id    BIGINT,
    empresa_id BIGINT,
    device_id  TEXT
);

CREATE TABLE gateway.commands (
    id           BIGSERIAL PRIMARY KEY,
    device_id    TEXT   NOT NULL,
    empresa_id   BIGINT,
    type         TEXT   NOT NULL,
    payload      JSONB  NOT NULL DEFAULT '{}',
    status       TEXT   NOT NULL DEFAULT 'pending',
    issued_by    BIGINT,
    fail_reason  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    acked_at     TIMESTAMPTZ
);

CREATE TABLE gateway.device_registry (
    id            BIGSERIAL PRIMARY KEY,
    device_id     TEXT    NOT NULL UNIQUE,
    empresa_id    BIGINT  REFERENCES identity.empresas(id) ON DELETE SET NULL,
    planta_id     BIGINT,
    linea_id      BIGINT,
    api_key       TEXT    NOT NULL UNIQUE,
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active        BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_gw_commands_device_status ON gateway.commands (device_id, status);
CREATE INDEX idx_gw_audit_ts               ON gateway.audit_log (ts DESC);
CREATE INDEX idx_gw_device_apikey          ON gateway.device_registry (api_key);

-- -------------------------------------------------------
-- Canvas OEE: Fórmulas visuales (scope: system / planta / dispositivo)
-- -------------------------------------------------------
CREATE TABLE IF NOT EXISTS config.canvas_oee (
    id             SERIAL PRIMARY KEY,
    scope          TEXT NOT NULL DEFAULT 'system' CHECK (scope IN ('system','planta','dispositivo')),
    planta_id      INT REFERENCES config.plantas(id) ON DELETE CASCADE,
    dispositivo_id INT REFERENCES config.dispositivos(id) ON DELETE CASCADE,
    nombre         TEXT NOT NULL DEFAULT 'Fórmula OEE',
    grafo          JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
    activo         BOOLEAN DEFAULT true,
    created_at     TIMESTAMPTZ DEFAULT now(),
    updated_at     TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT canvas_oee_unique_scope UNIQUE NULLS NOT DISTINCT (scope, planta_id, dispositivo_id)
);

CREATE TABLE IF NOT EXISTS config.turno_dia (
    id          SERIAL PRIMARY KEY,
    planta_id   INT NOT NULL REFERENCES config.plantas(id) ON DELETE CASCADE,
    linea_id    INT REFERENCES config.lineas(id) ON DELETE CASCADE,
    dia_semana  SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6),
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio TIME NOT NULL,
    hora_fin    TIME NOT NULL,
    color       VARCHAR(20) NOT NULL DEFAULT '#6366f1',
    activo      BOOLEAN NOT NULL DEFAULT TRUE,
    renovacion_semanal BOOLEAN NOT NULL DEFAULT TRUE,
    vigente_desde DATE NOT NULL DEFAULT CURRENT_DATE,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_turno_dia_scope ON config.turno_dia (planta_id, linea_id, vigente_desde);

-- Variables por línea que se usan como columnas dinámicas de producto
CREATE TABLE IF NOT EXISTS config.linea_producto_vars (
    id          SERIAL PRIMARY KEY,
    linea_id    INT NOT NULL REFERENCES config.lineas(id) ON DELETE CASCADE,
    variable_id INT NOT NULL REFERENCES config.variables(id) ON DELETE CASCADE,
    nombre_col  VARCHAR(100) NOT NULL,
    orden       INT NOT NULL DEFAULT 0,
    UNIQUE(linea_id, variable_id)
);
CREATE INDEX IF NOT EXISTS idx_lpv_linea ON config.linea_producto_vars (linea_id);

-- Valores de características por producto + línea + variable
CREATE TABLE IF NOT EXISTS config.producto_caracteristicas (
    id          SERIAL PRIMARY KEY,
    producto_id INT NOT NULL REFERENCES config.productos(id) ON DELETE CASCADE,
    linea_id    INT NOT NULL REFERENCES config.lineas(id) ON DELETE CASCADE,
    variable_id INT NOT NULL REFERENCES config.variables(id) ON DELETE CASCADE,
    valor       TEXT NOT NULL DEFAULT '',
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(producto_id, linea_id, variable_id)
);
CREATE INDEX IF NOT EXISTS idx_pc_linea_prod ON config.producto_caracteristicas (linea_id, producto_id);

-- Catálogo de valores permitidos por línea + variable (configura dropdowns en Excel)
CREATE TABLE IF NOT EXISTS config.linea_var_catalogo (
    id          SERIAL PRIMARY KEY,
    linea_id    INT NOT NULL REFERENCES config.lineas(id) ON DELETE CASCADE,
    variable_id INT NOT NULL REFERENCES config.variables(id) ON DELETE CASCADE,
    valor       VARCHAR(200) NOT NULL,
    orden       INT NOT NULL DEFAULT 0,
    UNIQUE(linea_id, variable_id, valor)
);
CREATE INDEX IF NOT EXISTS idx_lvc_linea_var ON config.linea_var_catalogo (linea_id, variable_id);

CREATE TABLE IF NOT EXISTS config.linea_productos (
    linea_id    INT NOT NULL REFERENCES config.lineas(id) ON DELETE CASCADE,
    producto_id INT NOT NULL REFERENCES config.productos(id) ON DELETE CASCADE,
    PRIMARY KEY (linea_id, producto_id)
);

CREATE TABLE IF NOT EXISTS config.velocidad_nominal (
    id              SERIAL PRIMARY KEY,
    linea_id        INT NOT NULL REFERENCES config.lineas(id) ON DELETE CASCADE,
    producto_id     INT NOT NULL REFERENCES config.productos(id) ON DELETE CASCADE,
    velocidad_us    DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv     INT NOT NULL DEFAULT 1,
    UNIQUE(linea_id, producto_id)
);

-- -------------------------------------------------------
-- SCHEMA: integration
-- -------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS integration;

CREATE TABLE IF NOT EXISTS integration.api_keys (
    id          SERIAL PRIMARY KEY,
    empresa_id  INT NOT NULL REFERENCES identity.empresas(id) ON DELETE CASCADE,
    nombre      VARCHAR(100) NOT NULL,
    key_prefix  VARCHAR(12) NOT NULL,
    key_hash    TEXT NOT NULL,
    scopes      TEXT[] NOT NULL DEFAULT ARRAY['oee:read','snapshots:read','paradas:read'],
    activo      BOOLEAN NOT NULL DEFAULT true,
    creado_en   TIMESTAMPTZ NOT NULL DEFAULT now(),
    ultimo_uso  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS api_keys_prefix_idx ON integration.api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS api_keys_empresa_idx ON integration.api_keys(empresa_id);

-- Indice unico para upsert estable del arbol de paradas por linea
CREATE UNIQUE INDEX IF NOT EXISTS uq_cat_paradas_linea_codigo
  ON config.categoria_paradas (linea_id, codigo)
  WHERE codigo IS NOT NULL AND codigo <> '';
