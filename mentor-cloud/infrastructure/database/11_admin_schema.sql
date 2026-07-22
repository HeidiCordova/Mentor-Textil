-- ============================================================
-- Migración 11: Schema admin y tabla planta_databases
-- Fuente de verdad para la localización de cada BD de planta.
-- Soporta instancias compartidas (VPS) y dedicadas (RDS).
-- ============================================================

CREATE SCHEMA IF NOT EXISTS admin;

CREATE TABLE IF NOT EXISTS admin.planta_databases (
    id               SERIAL      PRIMARY KEY,
    planta_id        INT         NOT NULL UNIQUE REFERENCES config.plantas(id),

    -- Credenciales y conexión
    db_name          TEXT        NOT NULL UNIQUE,
    pg_user          TEXT        NOT NULL,
    pg_password_enc  TEXT        NOT NULL,   -- AES-256-GCM, base64
    host             TEXT        NOT NULL DEFAULT 'localhost',
    port             INT         NOT NULL DEFAULT 5432,

    -- Tipo de instancia
    -- 'shared':    comparte el proceso Postgres del servidor (VPS)
    -- 'dedicated': instancia Postgres propia (RDS por planta)
    instance_type    TEXT        NOT NULL DEFAULT 'shared'
                                 CHECK (instance_type IN ('shared', 'dedicated')),

    -- Campos reservados para instancias dedicadas (RDS Option B).
    -- NULL mientras instance_type = 'shared'.
    rds_instance_id  TEXT,
    rds_region       TEXT,
    rds_arn          TEXT,
    rds_class        TEXT,

    -- Estado de provisioning
    provisioned      BOOLEAN     NOT NULL DEFAULT false,
    provisioned_at   TIMESTAMPTZ,
    lineas_creadas   INT[]       NOT NULL DEFAULT '{}',

    creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_plantadb_planta
    ON admin.planta_databases(planta_id);
