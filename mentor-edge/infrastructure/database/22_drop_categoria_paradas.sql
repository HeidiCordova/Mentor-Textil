-- =============================================================
-- Migration 22: Drop legacy categoria_paradas
--   Migrate data → cat_programada / cat_no_programada, then DROP.
--   Applies to ALL linea_N schemas.
-- =============================================================

DO $$
DECLARE
    sch TEXT;
BEGIN
    FOR sch IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
    LOOP
        -- Skip if categoria_paradas doesn't exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = sch AND table_name = 'categoria_paradas'
        ) THEN
            CONTINUE;
        END IF;

        -- ── 1. Migrate NO_PROGRAMADA ──
        EXECUTE format('
            INSERT INTO %I.cat_no_programada (id, codigo, nombre, padre_id, orden, activo, creado_en)
            SELECT id,
                   COALESCE(NULLIF(codigo,''''), ''cat-'' || id),
                   nombre,
                   NULL,
                   orden,
                   activo,
                   synced_at
            FROM %I.categoria_paradas
            WHERE UPPER(COALESCE(tipo_parada,'''')) LIKE ''%%NO%%PROGRAMADA%%''
            ON CONFLICT DO NOTHING
        ', sch, sch);

        EXECUTE format('
            UPDATE %I.cat_no_programada cp
            SET padre_id = orig.padre_id
            FROM %I.categoria_paradas orig
            WHERE cp.id = orig.id
              AND orig.padre_id IS NOT NULL
              AND EXISTS (SELECT 1 FROM %I.cat_no_programada WHERE id = orig.padre_id)
        ', sch, sch, sch);

        -- ── 2. Migrate PROGRAMADA (everything else) ──
        EXECUTE format('
            INSERT INTO %I.cat_programada (id, codigo, nombre, padre_id, orden, activo, creado_en)
            SELECT cp.id,
                   COALESCE(NULLIF(cp.codigo,''''), ''cat-'' || cp.id),
                   cp.nombre,
                   NULL,
                   cp.orden,
                   cp.activo,
                   cp.synced_at
            FROM %I.categoria_paradas cp
            WHERE NOT EXISTS (SELECT 1 FROM %I.cat_no_programada WHERE id = cp.id)
            ON CONFLICT DO NOTHING
        ', sch, sch, sch);

        EXECUTE format('
            UPDATE %I.cat_programada cp
            SET padre_id = orig.padre_id
            FROM %I.categoria_paradas orig
            WHERE cp.id = orig.id
              AND orig.padre_id IS NOT NULL
              AND EXISTS (SELECT 1 FROM %I.cat_programada WHERE id = orig.padre_id)
        ', sch, sch, sch);

        -- ── 3. Reset sequences ──
        EXECUTE format(
            'SELECT setval(pg_get_serial_sequence(%L, ''id''), COALESCE((SELECT MAX(id) FROM %I.cat_programada), 0) + 1, false)',
            sch || '.cat_programada', sch);
        EXECUTE format(
            'SELECT setval(pg_get_serial_sequence(%L, ''id''), COALESCE((SELECT MAX(id) FROM %I.cat_no_programada), 0) + 1, false)',
            sch || '.cat_no_programada', sch);

        -- ── 4. Drop legacy table ──
        EXECUTE format('DROP TABLE IF EXISTS %I.categoria_paradas CASCADE', sch);

        RAISE NOTICE 'Schema % — categoria_paradas migrated and dropped', sch;
    END LOOP;
END
$$;
