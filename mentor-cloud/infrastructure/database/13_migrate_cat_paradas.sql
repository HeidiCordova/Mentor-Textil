-- ============================================================
-- Migración: Nueva estructura de categorías y paradas
-- Ejecutar en: mentor_cloud
-- Efecto: crea tablas nuevas y migra datos existentes
--         NO elimina tablas antiguas (backward compatible)
-- ============================================================

BEGIN;

-- ── 1. Crear config.cat_programada ──────────────────────────

CREATE TABLE IF NOT EXISTS config.cat_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES config.cat_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    empresa_id INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    linea_id   INT REFERENCES config.lineas(id) ON DELETE SET NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cat_prog_codigo
    ON config.cat_programada(codigo)
    WHERE codigo IS NOT NULL AND codigo <> '';
CREATE INDEX IF NOT EXISTS idx_cat_prog_padre
    ON config.cat_programada(padre_id);
CREATE INDEX IF NOT EXISTS idx_cat_prog_empresa
    ON config.cat_programada(empresa_id);
CREATE INDEX IF NOT EXISTS idx_cat_prog_linea
    ON config.cat_programada(linea_id);

-- ── 2. Crear config.cat_no_programada ───────────────────────

CREATE TABLE IF NOT EXISTS config.cat_no_programada (
    id        SERIAL PRIMARY KEY,
    codigo    TEXT NOT NULL,
    nombre    VARCHAR(200) NOT NULL,
    padre_id  INT REFERENCES config.cat_no_programada(id) ON DELETE SET NULL,
    orden     INT NOT NULL DEFAULT 0,
    activo    BOOLEAN NOT NULL DEFAULT TRUE,
    empresa_id INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    linea_id   INT REFERENCES config.lineas(id) ON DELETE SET NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cat_noprog_codigo
    ON config.cat_no_programada(codigo)
    WHERE codigo IS NOT NULL AND codigo <> '';
CREATE INDEX IF NOT EXISTS idx_cat_noprog_padre
    ON config.cat_no_programada(padre_id);
CREATE INDEX IF NOT EXISTS idx_cat_noprog_empresa
    ON config.cat_no_programada(empresa_id);
CREATE INDEX IF NOT EXISTS idx_cat_noprog_linea
    ON config.cat_no_programada(linea_id);

-- ── 3. Migrar datos desde config.categoria_paradas ──────────
-- Función auxiliar: obtiene el tipo_parada del nodo raíz
CREATE OR REPLACE FUNCTION config._tipo_raiz(p_id INT)
RETURNS TEXT AS $$
DECLARE
    v_tipo   TEXT;
    v_padre  INT;
    v_actual INT := p_id;
BEGIN
    LOOP
        SELECT tipo_parada, padre_id
        INTO v_tipo, v_padre
        FROM config.categoria_paradas
        WHERE id = v_actual;

        -- Si tiene tipo propio claro, lo usa
        IF v_tipo IN ('PROGRAMADA', 'NO_PROGRAMADA') THEN
            RETURN v_tipo;
        END IF;
        -- Si no tiene padre, no se puede determinar
        IF v_padre IS NULL THEN
            RETURN NULL;
        END IF;
        v_actual := v_padre;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Migrar PROGRAMADAS (inserción en 2 pasadas: primero sin padre_id, luego resuelve)
-- Pasada 1: insertar solo raíces programadas
INSERT INTO config.cat_programada (id, codigo, nombre, padre_id, orden, activo, empresa_id, linea_id, creado_en)
SELECT
    c.id,
    COALESCE(NULLIF(c.codigo,''), 'cat-prog-' || c.id),
    c.nombre,
    NULL,  -- se actualizará en pasada 2
    c.orden,
    TRUE,
    c.empresa_id,
    c.linea_id,
    c.creado_en
FROM config.categoria_paradas c
WHERE config._tipo_raiz(c.id) = 'PROGRAMADA'
ON CONFLICT DO NOTHING;

-- Pasada 2: resolver padre_id dentro de cat_programada
UPDATE config.cat_programada cp
SET padre_id = orig.padre_id
FROM config.categoria_paradas orig
WHERE cp.id = orig.id
  AND orig.padre_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM config.cat_programada WHERE id = orig.padre_id);

-- Migrar NO PROGRAMADAS
INSERT INTO config.cat_no_programada (id, codigo, nombre, padre_id, orden, activo, empresa_id, linea_id, creado_en)
SELECT
    c.id,
    COALESCE(NULLIF(c.codigo,''), 'cat-noprog-' || c.id),
    c.nombre,
    NULL,
    c.orden,
    TRUE,
    c.empresa_id,
    c.linea_id,
    c.creado_en
FROM config.categoria_paradas c
WHERE config._tipo_raiz(c.id) = 'NO_PROGRAMADA'
ON CONFLICT DO NOTHING;

UPDATE config.cat_no_programada cp
SET padre_id = orig.padre_id
FROM config.categoria_paradas orig
WHERE cp.id = orig.id
  AND orig.padre_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM config.cat_no_programada WHERE id = orig.padre_id);

-- ── 4. Crear analytics.parada_programada ────────────────────
-- Reemplaza analytics.paradas para el nuevo flujo
-- Edge llena tiempos, operador clasifica

CREATE TABLE IF NOT EXISTS analytics.parada_programada (
    id             BIGSERIAL PRIMARY KEY,
    stop_id        UUID,                          -- id del edge para deduplicación
    device_id      VARCHAR(100) NOT NULL,
    linea_id       INT REFERENCES config.lineas(id) ON DELETE SET NULL,
    planta_id      INT REFERENCES config.plantas(id) ON DELETE SET NULL,
    empresa_id     INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    inicio         TIMESTAMPTZ NOT NULL,
    fin            TIMESTAMPTZ,
    duracion_min   NUMERIC(10,2),
    -- asignación del operador
    categoria_id   INT REFERENCES config.cat_programada(id) ON DELETE SET NULL,
    operador_id    INT,
    turno          VARCHAR(100),
    maquina        TEXT NOT NULL DEFAULT '',
    parte_maquina  TEXT NOT NULL DEFAULT '',
    descripcion    TEXT NOT NULL DEFAULT '',
    -- control
    asignado       BOOLEAN NOT NULL DEFAULT FALSE,
    asignado_en    TIMESTAMPTZ,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_pp_device_stop
    ON analytics.parada_programada(device_id, stop_id)
    WHERE stop_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_app_device_inicio   ON analytics.parada_programada(device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_inicio          ON analytics.parada_programada(inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_pendiente       ON analytics.parada_programada(inicio DESC) WHERE asignado = FALSE;
CREATE INDEX IF NOT EXISTS idx_app_categoria       ON analytics.parada_programada(categoria_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_operador        ON analytics.parada_programada(operador_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_turno           ON analytics.parada_programada(turno, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_linea_inicio    ON analytics.parada_programada(linea_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_empresa_inicio  ON analytics.parada_programada(empresa_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_app_abierta         ON analytics.parada_programada(inicio DESC) WHERE fin IS NULL;

-- ── 5. Crear analytics.parada_no_programada ─────────────────

CREATE TABLE IF NOT EXISTS analytics.parada_no_programada (
    id             BIGSERIAL PRIMARY KEY,
    stop_id        UUID,
    device_id      VARCHAR(100) NOT NULL,
    linea_id       INT REFERENCES config.lineas(id) ON DELETE SET NULL,
    planta_id      INT REFERENCES config.plantas(id) ON DELETE SET NULL,
    empresa_id     INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    inicio         TIMESTAMPTZ NOT NULL,
    fin            TIMESTAMPTZ,
    duracion_min   NUMERIC(10,2),
    -- asignación del operador
    categoria_id   INT REFERENCES config.cat_no_programada(id) ON DELETE SET NULL,
    operador_id    INT,
    turno          VARCHAR(100),
    maquina        TEXT NOT NULL DEFAULT '',
    parte_maquina  TEXT NOT NULL DEFAULT '',
    descripcion    TEXT NOT NULL DEFAULT '',
    -- control
    asignado       BOOLEAN NOT NULL DEFAULT FALSE,
    asignado_en    TIMESTAMPTZ,
    creado_en      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_pnp_device_stop
    ON analytics.parada_no_programada(device_id, stop_id)
    WHERE stop_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_anp_device_inicio   ON analytics.parada_no_programada(device_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_inicio          ON analytics.parada_no_programada(inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_pendiente       ON analytics.parada_no_programada(inicio DESC) WHERE asignado = FALSE;
CREATE INDEX IF NOT EXISTS idx_anp_categoria       ON analytics.parada_no_programada(categoria_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_operador        ON analytics.parada_no_programada(operador_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_turno           ON analytics.parada_no_programada(turno, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_linea_inicio    ON analytics.parada_no_programada(linea_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_empresa_inicio  ON analytics.parada_no_programada(empresa_id, inicio DESC);
CREATE INDEX IF NOT EXISTS idx_anp_abierta         ON analytics.parada_no_programada(inicio DESC) WHERE fin IS NULL;

-- ── 6. Trigger duracion_min automático ──────────────────────

CREATE OR REPLACE FUNCTION analytics.fill_duracion_min_stop()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.fin IS NOT NULL AND NEW.inicio IS NOT NULL THEN
        NEW.duracion_min := ROUND(
            EXTRACT(EPOCH FROM (NEW.fin - NEW.inicio)) / 60.0, 2
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_fill_pp_duracion ON analytics.parada_programada;
CREATE TRIGGER trg_fill_pp_duracion
    BEFORE INSERT OR UPDATE ON analytics.parada_programada
    FOR EACH ROW EXECUTE FUNCTION analytics.fill_duracion_min_stop();

DROP TRIGGER IF EXISTS trg_fill_pnp_duracion ON analytics.parada_no_programada;
CREATE TRIGGER trg_fill_pnp_duracion
    BEFORE INSERT OR UPDATE ON analytics.parada_no_programada
    FOR EACH ROW EXECUTE FUNCTION analytics.fill_duracion_min_stop();

-- ── 7. Limpiar función temporal ─────────────────────────────
DROP FUNCTION IF EXISTS config._tipo_raiz(INT);

COMMIT;

-- ============================================================
-- NOTA: analytics.paradas y config.categoria_paradas se
-- conservan intactas para backward compatibility.
-- Eliminar en una versión futura una vez migrado el edge.
-- ============================================================
