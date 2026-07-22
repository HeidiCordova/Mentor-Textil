-- Tablas replica para sincronizacion de catalogo desde el cloud.
-- La PK es el ID del cloud (no auto-incremental) para permitir upsert deterministico.

CREATE TABLE sync_categoria_paradas (
    id          BIGINT       PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    codigo      TEXT         NOT NULL DEFAULT '',
    padre_id    BIGINT,
    empresa_id  INTEGER,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE sync_productos (
    id          INTEGER      PRIMARY KEY,
    codigo      VARCHAR(50)  NOT NULL,
    nombre      VARCHAR(100) NOT NULL,
    empresa_id  INTEGER,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE sync_turnos (
    id          INTEGER      PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    hora_inicio VARCHAR(20)  NOT NULL,
    hora_fin    VARCHAR(20)  NOT NULL,
    planta_id   INTEGER,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_cat_empresa ON sync_categoria_paradas (empresa_id);
CREATE INDEX idx_sync_prod_empresa ON sync_productos (empresa_id);
CREATE INDEX idx_sync_turnos_activo ON sync_turnos (activo);
