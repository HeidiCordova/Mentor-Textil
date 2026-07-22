-- ============================================================
-- Migración 16: Agregar stop_id a parada_programada y
-- parada_no_programada para idempotencia en sincronización
-- desde el Jetson edge.
--
-- Aplica sobre TODAS las BDs de planta (mentor_planta_*) en
-- todos sus schemas linea_N.
-- Ejecutar por planta:
--   psql -U mentor -d mentor_planta_13 -f 16_paradas_stop_id.sql
-- ============================================================

DO $$
DECLARE
    r RECORD;
    tbl TEXT;
BEGIN
    FOR r IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
        ORDER BY schema_name
    LOOP
        -- parada_programada
        tbl := r.schema_name || '.parada_programada';
        IF EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = r.schema_name AND table_name = 'parada_programada'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.parada_programada
                 ADD COLUMN IF NOT EXISTS stop_id UUID;
                 CREATE UNIQUE INDEX IF NOT EXISTS uq_%I_pp_stop_id
                 ON %I.parada_programada(stop_id) WHERE stop_id IS NOT NULL;',
                r.schema_name, r.schema_name, r.schema_name
            );
            RAISE NOTICE 'parada_programada: stop_id agregado en %', r.schema_name;
        END IF;

        -- parada_no_programada
        IF EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = r.schema_name AND table_name = 'parada_no_programada'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.parada_no_programada
                 ADD COLUMN IF NOT EXISTS stop_id UUID;
                 CREATE UNIQUE INDEX IF NOT EXISTS uq_%I_pnp_stop_id
                 ON %I.parada_no_programada(stop_id) WHERE stop_id IS NOT NULL;',
                r.schema_name, r.schema_name, r.schema_name
            );
            RAISE NOTICE 'parada_no_programada: stop_id agregado en %', r.schema_name;
        END IF;
    END LOOP;
END $$;
