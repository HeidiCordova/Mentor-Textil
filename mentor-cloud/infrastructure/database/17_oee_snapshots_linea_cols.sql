-- ══════════════════════════════════════════════════════════════════════
-- 17_oee_snapshots_linea_cols.sql
-- Agrega las columnas linea_id / planta_id / empresa_id a:
--   · {schema}.oee_snapshots
--   · {schema}.production_runs
-- en todos los schemas linea_* del tenant.
--
-- Cuándo ejecutar: UNA VEZ por base de datos de planta (mentor_planta_N).
-- Es idempotente — ignora columnas que ya existen.
--
-- Ejemplo (planta 14 en producción):
--   psql "host=... dbname=mentor_planta_14 ..." -f 17_oee_snapshots_linea_cols.sql
-- ══════════════════════════════════════════════════════════════════════

DO $$
DECLARE
    r RECORD;
    col TEXT;
    tbl TEXT;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
        ORDER BY schema_name
    LOOP
        -- ── oee_snapshots ───────────────────────────────────────────
        FOREACH col IN ARRAY ARRAY['linea_id','planta_id','empresa_id'] LOOP
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_schema = r.schema_name
                  AND table_name   = 'oee_snapshots'
                  AND column_name  = col
            ) THEN
                EXECUTE format('ALTER TABLE %I.oee_snapshots ADD COLUMN %I INT',
                               r.schema_name, col);
                RAISE NOTICE '% agregado a %.oee_snapshots', col, r.schema_name;
            END IF;
        END LOOP;

        -- ── production_runs ─────────────────────────────────────────
        FOREACH col IN ARRAY ARRAY['linea_id','planta_id','empresa_id'] LOOP
            IF NOT EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_schema = r.schema_name
                  AND table_name   = 'production_runs'
                  AND column_name  = col
            ) THEN
                EXECUTE format('ALTER TABLE %I.production_runs ADD COLUMN %I INT',
                               r.schema_name, col);
                RAISE NOTICE '% agregado a %.production_runs', col, r.schema_name;
            END IF;
        END LOOP;

        -- ── oee_snapshots.data: cambiar BIGINT[] → TEXT[] ───────────
        -- data puede contener strings vacíos o valores float (energia_kwh),
        -- BIGINT[] rechaza esos valores con "invalid input syntax for type bigint".
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = r.schema_name
              AND table_name   = 'oee_snapshots'
              AND column_name  = 'data'
              AND udt_name     = '_int8'   -- _int8 = BIGINT[]
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.oee_snapshots ALTER COLUMN data TYPE TEXT[] USING data::TEXT[]',
                r.schema_name);
            RAISE NOTICE 'data convertido BIGINT[]→TEXT[] en %.oee_snapshots', r.schema_name;
        END IF;

    END LOOP;
    RAISE NOTICE 'Migración 17 completada.';
END;
$$;
