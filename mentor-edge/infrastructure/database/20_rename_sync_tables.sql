-- =============================================================
-- Migration 20: Rename sync_* tables to match cloud names
-- Edge tables now mirror cloud table names exactly.
-- Applies to ALL linea_N schemas + shared schema.
-- =============================================================

DO $$
DECLARE
    sch TEXT;
    renames TEXT[][] := ARRAY[
        ['sync_categoria_paradas',      'categoria_paradas'],
        ['sync_productos',              'productos'],
        ['sync_turnos',                 'turnos'],
        ['sync_usuarios',              'usuarios'],
        ['sync_variables',             'variables'],
        ['sync_linea_producto_vars',   'linea_producto_vars'],
        ['sync_producto_caracteristicas','producto_caracteristicas'],
        ['sync_plantas',               'plantas'],
        ['sync_lineas',                'lineas'],
        ['sync_velocidad_nominal',     'velocidad_nominal']
    ];
    r TEXT[];
BEGIN
    -- Iterate over all schemas: shared + linea_*
    FOR sch IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name = 'shared' OR schema_name LIKE 'linea_%'
    LOOP
        FOREACH r SLICE 1 IN ARRAY renames LOOP
            IF EXISTS (
                SELECT 1 FROM information_schema.tables
                WHERE table_schema = sch AND table_name = r[1]
            ) THEN
                EXECUTE format('ALTER TABLE %I.%I RENAME TO %I', sch, r[1], r[2]);
                RAISE NOTICE 'Renamed %.% → %.%', sch, r[1], sch, r[2];
            END IF;
        END LOOP;

        -- Add activo column to categoria_paradas if missing (older schemas)
        IF EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = sch AND table_name = 'categoria_paradas'
        ) AND NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = sch AND table_name = 'categoria_paradas' AND column_name = 'activo'
        ) THEN
            EXECUTE format('ALTER TABLE %I.categoria_paradas ADD COLUMN activo BOOLEAN NOT NULL DEFAULT TRUE', sch);
            RAISE NOTICE 'Added activo to %.categoria_paradas', sch;
        END IF;
    END LOOP;
END;
$$;
