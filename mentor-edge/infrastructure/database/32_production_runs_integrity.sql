-- Integridad de corridas para schemas linea_N existentes.
-- La migracion es atomica: si detecta datos ambiguos, aborta sin borrar ni
-- corregir filas silenciosamente.

DO $$
DECLARE
    rec RECORD;
    bad_value TEXT;
    constraint_name TEXT;
BEGIN
    FOR rec IN
        SELECT schema_name,
               substring(schema_name FROM '^linea_([0-9]+)$')::INTEGER AS linea_id
        FROM information_schema.schemata
        WHERE schema_name ~ '^linea_[0-9]+$'
        ORDER BY schema_name
    LOOP
        IF to_regclass(format('%I.production_runs', rec.schema_name)) IS NULL THEN
            CONTINUE;
        END IF;

        EXECUTE format(
            'SELECT run_id::text
               FROM %I.production_runs
              GROUP BY run_id
             HAVING COUNT(*) > 1
              LIMIT 1',
            rec.schema_name
        ) INTO bad_value;
        IF bad_value IS NOT NULL THEN
            RAISE EXCEPTION
                'No se puede migrar %.production_runs: run_id duplicado %',
                rec.schema_name, bad_value;
        END IF;

        bad_value := NULL;
        EXECUTE format(
            'SELECT device_id
               FROM %I.production_runs
              WHERE ended_at IS NULL
              GROUP BY device_id
             HAVING COUNT(*) > 1
              LIMIT 1',
            rec.schema_name
        ) INTO bad_value;
        IF bad_value IS NOT NULL THEN
            RAISE EXCEPTION
                'No se puede migrar %.production_runs: mas de una corrida abierta para device %',
                rec.schema_name, bad_value;
        END IF;

        bad_value := NULL;
        EXECUTE format(
            'SELECT run_id::text
               FROM %I.production_runs
              WHERE ended_at IS NOT NULL
                AND ended_at < started_at
              LIMIT 1',
            rec.schema_name
        ) INTO bad_value;
        IF bad_value IS NOT NULL THEN
            RAISE EXCEPTION
                'No se puede migrar %.production_runs: intervalo negativo en run_id %',
                rec.schema_name, bad_value;
        END IF;

        -- El schema local es la autoridad para linea_id. Esto repara filas
        -- legacy NULL o filas que recibieron accidentalmente el ID del cloud.
        EXECUTE format(
            'UPDATE %I.production_runs
                SET linea_id = $1,
                    updated_at = NOW()
              WHERE linea_id IS DISTINCT FROM $1',
            rec.schema_name
        ) USING rec.linea_id;

        EXECUTE format(
            'CREATE UNIQUE INDEX IF NOT EXISTS production_runs_run_id_uq
                ON %I.production_runs (run_id)',
            rec.schema_name
        );
        EXECUTE format(
            'CREATE UNIQUE INDEX IF NOT EXISTS production_runs_one_open_uq
                ON %I.production_runs (device_id)
             WHERE ended_at IS NULL',
            rec.schema_name
        );

        constraint_name := rec.schema_name || '_production_runs_time_ck';
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint c
            JOIN pg_namespace n ON n.oid = c.connamespace
            WHERE n.nspname = rec.schema_name
              AND c.conname = constraint_name
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.production_runs
                   ADD CONSTRAINT %I
                   CHECK (ended_at IS NULL OR ended_at >= started_at)',
                rec.schema_name,
                constraint_name
            );
        END IF;
    END LOOP;
END
$$;
