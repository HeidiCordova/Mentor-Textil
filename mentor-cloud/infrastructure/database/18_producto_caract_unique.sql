-- Migracion 18: Agrega UNIQUE(producto_id, variable_id) a producto_caracteristicas
-- en todos los schemas linea_* existentes y elimina filas duplicadas previas.
-- Ejecutar en cada base de datos de planta (mentor_planta_XX).

DO $$
DECLARE
    s TEXT;
    cname TEXT;
BEGIN
    FOR s IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^linea_[0-9]+'
        ORDER BY schema_name
    LOOP
        -- Verificar que la tabla existe en este schema
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_schema = s
              AND table_name = 'producto_caracteristicas'
        ) THEN
            CONTINUE;
        END IF;

        -- Eliminar filas duplicadas (conservar la de menor id)
        EXECUTE format(
            'DELETE FROM %I.producto_caracteristicas
             WHERE id NOT IN (
                 SELECT MIN(id)
                 FROM %I.producto_caracteristicas
                 GROUP BY producto_id, variable_id
             )',
            s, s
        );

        -- Nombre del constraint (max 63 chars en postgres)
        cname := 'uq_prod_caract_' || s;

        -- Agregar UNIQUE si no existe ninguno sobre esa tabla
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint c
            JOIN pg_class t    ON t.oid = c.conrelid
            JOIN pg_namespace n ON n.oid = t.relnamespace
            WHERE n.nspname = s
              AND t.relname  = 'producto_caracteristicas'
              AND c.contype  = 'u'
        ) THEN
            EXECUTE format(
                'ALTER TABLE %I.producto_caracteristicas
                 ADD CONSTRAINT %I UNIQUE (producto_id, variable_id)',
                s, cname
            );
            RAISE NOTICE 'Constraint % anadido en schema %', cname, s;
        ELSE
            RAISE NOTICE 'Constraint UNIQUE ya existe en %.producto_caracteristicas -- omitido', s;
        END IF;
    END LOOP;
END $$;
