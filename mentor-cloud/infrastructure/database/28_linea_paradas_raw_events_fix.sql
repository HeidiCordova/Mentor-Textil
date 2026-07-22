-- ============================================================
-- Migración 28: Correcciones al esquema por línea
--
-- 1. Agregar columns empresa_id, planta_id, linea_id a raw_events
--    (faltaban en 12_linea_template.sql; InsertRawEvent las inserta).
--
-- 2. Crear tabla {schema}.paradas en cada schema linea_* existente.
--    cloud-ingest (upsertParadasUnified) y cloud-analytics (ListStops,
--    JustifyStop, CreateStop, DeleteStop) leen/escriben en esta tabla
--    vía multitenancy.Tbl(schema, "analytics.paradas") → {schema}.paradas.
--
-- Aplicar por planta:
--   psql -U mentor -d mentor_planta_N -f 28_linea_paradas_raw_events_fix.sql
-- ============================================================

DO $$
DECLARE
    s TEXT;
BEGIN
    FOR s IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
        ORDER BY schema_name
    LOOP
        -- ── 1. raw_events: agregar columnas de scope si no existen ──────────
        EXECUTE format('ALTER TABLE %I.raw_events ADD COLUMN IF NOT EXISTS empresa_id INT', s);
        EXECUTE format('ALTER TABLE %I.raw_events ADD COLUMN IF NOT EXISTS planta_id  INT', s);
        EXECUTE format('ALTER TABLE %I.raw_events ADD COLUMN IF NOT EXISTS linea_id   INT', s);
        RAISE NOTICE 'raw_events: scope columns ensured in %', s;

        -- ── 2. paradas: crear tabla unificada si no existe ─────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.paradas (
                id                    BIGSERIAL PRIMARY KEY,
                device_id             VARCHAR(100) NOT NULL,
                linea_id              INT,
                planta_id             INT,
                empresa_id            INT,
                categoria_id          INT,
                categoria_nombre      TEXT,
                subcategoria_nombre   TEXT,
                subcategoria_2_nombre TEXT,
                descripcion           TEXT,
                stop_id               UUID,
                stop_type             VARCHAR(50),
                inicio                TIMESTAMPTZ NOT NULL,
                fin                   TIMESTAMPTZ,
                duracion_min          NUMERIC(10,2),
                justified             BOOLEAN NOT NULL DEFAULT FALSE,
                justified_by          TEXT,
                justified_at          TIMESTAMPTZ,
                reason                TEXT,
                source                TEXT,
                creado_en             TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', s);

        EXECUTE format('
            CREATE UNIQUE INDEX IF NOT EXISTS uq_paradas_device_stop_id
            ON %I.paradas (device_id, stop_id) WHERE stop_id IS NOT NULL', s);
        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_paradas_device_inicio
            ON %I.paradas (device_id, inicio DESC)', s);
        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_paradas_inicio
            ON %I.paradas (inicio DESC)', s);
        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_paradas_justified
            ON %I.paradas (justified) WHERE justified = FALSE', s);
        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_paradas_linea_inicio
            ON %I.paradas (linea_id, inicio DESC)', s);

        RAISE NOTICE 'paradas: table ensured in %', s;
    END LOOP;
END $$;
