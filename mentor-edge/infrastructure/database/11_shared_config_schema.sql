-- =============================================================
-- Migración: Schema compartido "config" para gestión de devices
--
-- Problema: edge-config-service dependía de DEVICE_ID env var
-- para seleccionar linea_{X}.line_config → requería restart.
--
-- Solución: config.line_config centralizada, el servicio la usa
-- directamente sin depender de DEVICE_ID.
-- =============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE SCHEMA IF NOT EXISTS config;

-- ═══════════════════════════════════════════════════════════════
-- 1. TABLA CENTRALIZADA DE CONFIGURACIÓN DE LÍNEAS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS config.line_config (
    id              SERIAL PRIMARY KEY,
    device_id       VARCHAR(100) UNIQUE NOT NULL,
    config_version  INTEGER NOT NULL DEFAULT 1,
    roi             JSONB NOT NULL DEFAULT '{"x":120,"y":60,"width":320,"height":200}',
    thresholds      JSONB NOT NULL DEFAULT '{"edge":0.4,"color":0.6,"flow":0.5,"dy":5.0,"beige":0.35,"high":0.7,"low":0.3}',
    fsm             JSONB NOT NULL DEFAULT '{"n_frames":3,"cooldown":8,"exit_frames":5,"max_wait_exit_frames":50}',
    mode            VARCHAR(30) NOT NULL DEFAULT 'textil',
    camera          JSONB,
    oee             JSONB NOT NULL DEFAULT '{"line_name":"","micro_stop_max_s":210,"stop_max_s":300,"snapshot_interval_s":300}',
    cloud           JSONB NOT NULL DEFAULT '{"sync_interval_s":300}',
    tablet          JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-incrementar config_version en cada UPDATE
CREATE OR REPLACE FUNCTION config.bump_config_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.config_version := OLD.config_version + 1;
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_config_bump_version ON config.line_config;
CREATE TRIGGER trg_config_bump_version
BEFORE UPDATE ON config.line_config
FOR EACH ROW
EXECUTE FUNCTION config.bump_config_version();

-- ═══════════════════════════════════════════════════════════════
-- 2. MIGRAR DATOS EXISTENTES DESDE linea_*.line_config
-- ═══════════════════════════════════════════════════════════════

DO $$
DECLARE
    schema_rec RECORD;
    dev_rec    RECORD;
BEGIN
    FOR schema_rec IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
    LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = schema_rec.schema_name
              AND table_name = 'line_config'
        ) THEN
            FOR dev_rec IN
                EXECUTE format(
                    'SELECT device_id, config_version, roi, thresholds, fsm, mode, camera, oee, cloud, tablet, created_at, updated_at FROM %I.line_config',
                    schema_rec.schema_name
                )
            LOOP
                INSERT INTO config.line_config
                    (device_id, config_version, roi, thresholds, fsm, mode, camera, oee, cloud, tablet, created_at, updated_at)
                VALUES
                    (dev_rec.device_id, dev_rec.config_version, dev_rec.roi, dev_rec.thresholds,
                     dev_rec.fsm, dev_rec.mode, dev_rec.camera, dev_rec.oee,
                     dev_rec.cloud, dev_rec.tablet, dev_rec.created_at, dev_rec.updated_at)
                ON CONFLICT (device_id) DO NOTHING;
            END LOOP;
        END IF;
    END LOOP;
END $$;
