-- =============================================================
-- Migración: public.* → shared.* + linea_3.*
-- Ejecutar UNA VEZ con los servicios detenidos.
-- =============================================================

BEGIN;

-- ─── 1. Crear schemas ──────────────────────────────────────────────
CREATE SCHEMA IF NOT EXISTS shared;
CREATE SCHEMA IF NOT EXISTS linea_3;

-- ─── 2. Mover tablas SHARED (device-wide) ──────────────────────────
ALTER TABLE public.line_config             SET SCHEMA shared;
ALTER TABLE public.commands_buffer         SET SCHEMA shared;
ALTER TABLE public.audit_log               SET SCHEMA shared;
ALTER TABLE public.health_logs             SET SCHEMA shared;
ALTER TABLE public.calibration_history     SET SCHEMA shared;
ALTER TABLE public.sync_categoria_paradas  SET SCHEMA shared;
ALTER TABLE public.sync_productos          SET SCHEMA shared;
ALTER TABLE public.sync_turnos             SET SCHEMA shared;
ALTER TABLE public.sync_usuarios           SET SCHEMA shared;
ALTER TABLE public.sync_variables          SET SCHEMA shared;
ALTER TABLE public.sync_linea_producto_vars      SET SCHEMA shared;
ALTER TABLE public.sync_producto_caracteristicas SET SCHEMA shared;
ALTER TABLE public.sync_plantas            SET SCHEMA shared;
ALTER TABLE public.sync_lineas             SET SCHEMA shared;
ALTER TABLE public.sync_velocidad_nominal  SET SCHEMA shared;

-- ─── 3. Mover tablas PER-LINE (linea_3) ───────────────────────────
ALTER TABLE public.events_buffer           SET SCHEMA linea_3;
ALTER TABLE public.stops                   SET SCHEMA linea_3;
ALTER TABLE public.production_runs         SET SCHEMA linea_3;
ALTER TABLE public.vision_detections       SET SCHEMA linea_3;

-- ─── 4. Mover funciones/triggers al schema correcto ────────────────
-- El trigger de line_config se mueve automáticamente con la tabla.
-- Mover la función update_config_timestamp al schema shared.
CREATE OR REPLACE FUNCTION shared.update_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.config_version = OLD.config_version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Re-crear trigger apuntando a la función en shared
DROP TRIGGER IF EXISTS config_update_trigger ON shared.line_config;
CREATE TRIGGER config_update_trigger
BEFORE UPDATE ON shared.line_config
FOR EACH ROW
EXECUTE FUNCTION shared.update_config_timestamp();

-- Trigger de duración de stops (recalcula duration_ms al cerrar)
CREATE OR REPLACE FUNCTION linea_3.calc_stop_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.ended_at IS NOT NULL AND OLD.ended_at IS NULL THEN
        NEW.duration_ms = EXTRACT(EPOCH FROM (NEW.ended_at - NEW.started_at)) * 1000;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_stop_duration ON linea_3.stops;
CREATE TRIGGER trg_stop_duration
BEFORE UPDATE ON linea_3.stops
FOR EACH ROW
EXECUTE FUNCTION linea_3.calc_stop_duration();

-- ─── 5. Verificación ──────────────────────────────────────────────
DO $$
DECLARE
    cnt_public INT;
BEGIN
    SELECT COUNT(*) INTO cnt_public
    FROM pg_tables WHERE schemaname = 'public' AND tablename NOT LIKE 'pg_%';
    
    IF cnt_public > 0 THEN
        RAISE NOTICE 'ATENCION: Quedan % tabla(s) en public', cnt_public;
    ELSE
        RAISE NOTICE 'OK: public vacío, migración completa';
    END IF;
END $$;

COMMIT;
