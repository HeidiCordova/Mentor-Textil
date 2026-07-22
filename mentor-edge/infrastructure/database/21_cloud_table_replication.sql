-- =============================================================
-- Migration 21: Replicate cloud table structures on edge
--   1. Add creado_en to productos, turnos, producto_caracteristicas
--   2. Create new cloud tables: cat_programada, cat_no_programada,
--      parada_programada, parada_no_programada, alarmas,
--      variables_producto, turno_dia, linea_var_catalogo, canvas_oee
-- Applies to ALL linea_N schemas.
-- =============================================================

DO $$
DECLARE
    sch TEXT;
BEGIN
    FOR sch IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
    LOOP
        -- ────────────────────────────────────────────────
        -- 1. Add creado_en to existing tables
        -- ────────────────────────────────────────────────
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = sch AND table_name = 'productos' AND column_name = 'creado_en'
        ) THEN
            EXECUTE format('ALTER TABLE %I.productos ADD COLUMN creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()', sch);
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = sch AND table_name = 'turnos' AND column_name = 'creado_en'
        ) THEN
            EXECUTE format('ALTER TABLE %I.turnos ADD COLUMN creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()', sch);
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = sch AND table_name = 'producto_caracteristicas' AND column_name = 'creado_en'
        ) THEN
            EXECUTE format('ALTER TABLE %I.producto_caracteristicas ADD COLUMN creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()', sch);
        END IF;

        -- ────────────────────────────────────────────────
        -- 2. cat_programada (cloud: árbol de categorías programadas)
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.cat_programada (
                id        SERIAL PRIMARY KEY,
                codigo    TEXT NOT NULL,
                nombre    VARCHAR(200) NOT NULL,
                padre_id  INT REFERENCES %I.cat_programada(id) ON DELETE SET NULL,
                orden     INT NOT NULL DEFAULT 0,
                activo    BOOLEAN NOT NULL DEFAULT TRUE,
                creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', sch, sch);
        EXECUTE format('CREATE UNIQUE INDEX IF NOT EXISTS uq_%s_cat_prog_codigo ON %I.cat_programada(codigo)', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_cat_prog_padre ON %I.cat_programada(padre_id)', sch, sch);

        -- ────────────────────────────────────────────────
        -- 3. cat_no_programada (cloud: árbol de categorías no programadas)
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.cat_no_programada (
                id        SERIAL PRIMARY KEY,
                codigo    TEXT NOT NULL,
                nombre    VARCHAR(200) NOT NULL,
                padre_id  INT REFERENCES %I.cat_no_programada(id) ON DELETE SET NULL,
                orden     INT NOT NULL DEFAULT 0,
                activo    BOOLEAN NOT NULL DEFAULT TRUE,
                creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', sch, sch);
        EXECUTE format('CREATE UNIQUE INDEX IF NOT EXISTS uq_%s_cat_noprog_codigo ON %I.cat_no_programada(codigo)', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_cat_noprog_padre ON %I.cat_no_programada(padre_id)', sch, sch);

        -- ────────────────────────────────────────────────
        -- 4. parada_programada
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.parada_programada (
                id             BIGSERIAL PRIMARY KEY,
                stop_id        UUID,
                device_id      VARCHAR(100) NOT NULL,
                inicio         TIMESTAMPTZ NOT NULL,
                fin            TIMESTAMPTZ,
                duracion_min   NUMERIC(10,2),
                categoria_id   INT REFERENCES %I.cat_programada(id) ON DELETE SET NULL,
                operador_id    INT,
                turno          VARCHAR(100),
                maquina        TEXT NOT NULL DEFAULT '''',
                parte_maquina  TEXT NOT NULL DEFAULT '''',
                descripcion    TEXT NOT NULL DEFAULT '''',
                asignado       BOOLEAN NOT NULL DEFAULT FALSE,
                asignado_en    TIMESTAMPTZ,
                creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_pp_device_inicio ON %I.parada_programada(device_id, inicio DESC)', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_pp_pendiente ON %I.parada_programada(inicio DESC) WHERE asignado = FALSE', sch, sch);
        EXECUTE format('CREATE UNIQUE INDEX IF NOT EXISTS uq_%s_pp_stop_id ON %I.parada_programada(stop_id) WHERE stop_id IS NOT NULL', sch, sch);

        -- ────────────────────────────────────────────────
        -- 5. parada_no_programada
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.parada_no_programada (
                id             BIGSERIAL PRIMARY KEY,
                stop_id        UUID,
                device_id      VARCHAR(100) NOT NULL,
                inicio         TIMESTAMPTZ NOT NULL,
                fin            TIMESTAMPTZ,
                duracion_min   NUMERIC(10,2),
                categoria_id   INT REFERENCES %I.cat_no_programada(id) ON DELETE SET NULL,
                operador_id    INT,
                turno          VARCHAR(100),
                maquina        TEXT NOT NULL DEFAULT '''',
                parte_maquina  TEXT NOT NULL DEFAULT '''',
                descripcion    TEXT NOT NULL DEFAULT '''',
                asignado       BOOLEAN NOT NULL DEFAULT FALSE,
                asignado_en    TIMESTAMPTZ,
                creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_pnp_device_inicio ON %I.parada_no_programada(device_id, inicio DESC)', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_pnp_pendiente ON %I.parada_no_programada(inicio DESC) WHERE asignado = FALSE', sch, sch);
        EXECUTE format('CREATE UNIQUE INDEX IF NOT EXISTS uq_%s_pnp_stop_id ON %I.parada_no_programada(stop_id) WHERE stop_id IS NOT NULL', sch, sch);

        -- ────────────────────────────────────────────────
        -- 6. alarmas
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.alarmas (
                id               BIGSERIAL PRIMARY KEY,
                device_id        VARCHAR(100) NOT NULL,
                tipo             VARCHAR(50) NOT NULL,
                mensaje          TEXT NOT NULL,
                severidad        VARCHAR(20) NOT NULL DEFAULT ''info'',
                activa           BOOLEAN NOT NULL DEFAULT TRUE,
                timestamp_evento TIMESTAMPTZ NOT NULL,
                reconocido_en    TIMESTAMPTZ,
                creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_alarmas_activa ON %I.alarmas(activa) WHERE activa', sch, sch);

        -- ────────────────────────────────────────────────
        -- 7. variables_producto (cloud: {schema}.variables — tipos de atributo de producto)
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.variables_producto (
                id     SERIAL PRIMARY KEY,
                nombre VARCHAR(200) NOT NULL,
                unidad VARCHAR(50),
                tipo   VARCHAR(50) NOT NULL DEFAULT ''numeric'',
                activo BOOLEAN NOT NULL DEFAULT TRUE
            )', sch);

        -- ────────────────────────────────────────────────
        -- 8. turno_dia (cloud: turno_dia — horario por día de semana)
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.turno_dia (
                id                 SERIAL PRIMARY KEY,
                linea_id           INT,
                dia_semana         SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6),
                nombre             VARCHAR(100) NOT NULL,
                hora_inicio        TIME NOT NULL,
                hora_fin           TIME NOT NULL,
                color              VARCHAR(20) NOT NULL DEFAULT ''#6366f1'',
                activo             BOOLEAN NOT NULL DEFAULT TRUE,
                renovacion_semanal BOOLEAN NOT NULL DEFAULT TRUE,
                vigente_desde      DATE NOT NULL DEFAULT CURRENT_DATE,
                creado_en          TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_turno_dia_scope ON %I.turno_dia(linea_id, vigente_desde)', sch, sch);
        EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_turno_dia_vigente ON %I.turno_dia(vigente_desde)', sch, sch);

        -- ────────────────────────────────────────────────
        -- 9. linea_var_catalogo (cloud: catálogo dropdown por variable)
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.linea_var_catalogo (
                id          SERIAL PRIMARY KEY,
                variable_id INT NOT NULL,
                valor       VARCHAR(200) NOT NULL,
                orden       INT NOT NULL DEFAULT 0,
                UNIQUE(variable_id, valor)
            )', sch);

        -- ────────────────────────────────────────────────
        -- 10. canvas_oee (cloud: canvas OEE formula graph)
        -- ────────────────────────────────────────────────
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.canvas_oee (
                id         SERIAL PRIMARY KEY,
                nombre     TEXT NOT NULL DEFAULT ''Formula OEE'',
                grafo      JSONB NOT NULL DEFAULT ''{"nodes":[],"edges":[]}'',
                activo     BOOLEAN DEFAULT true,
                created_at TIMESTAMPTZ DEFAULT now(),
                updated_at TIMESTAMPTZ DEFAULT now()
            )', sch);

        RAISE NOTICE 'Schema % — cloud tables replicated', sch;
    END LOOP;
END
$$;
