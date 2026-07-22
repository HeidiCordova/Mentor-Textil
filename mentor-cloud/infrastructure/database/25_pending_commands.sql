-- ============================================================
-- Tabla pending_commands: comandos encolados desde cloud hacia edge.
-- El Enviador la consulta en cada ciclo para aplicar comandos pendientes
-- incluso si la conexión SSE estaba caída al momento de la justificación.
-- ============================================================

-- Crear en cada schema linea_* existente
DO $$
DECLARE
    s TEXT;
BEGIN
    FOR s IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
    LOOP
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.pending_commands (
                id          BIGSERIAL PRIMARY KEY,
                device_id   VARCHAR(100) NOT NULL,
                command     TEXT NOT NULL,
                payload     JSONB NOT NULL DEFAULT ''{}''::jsonb,
                applied     BOOLEAN NOT NULL DEFAULT FALSE,
                applied_at  TIMESTAMPTZ,
                created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )
        ', s);
        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_pc_device_applied
            ON %I.pending_commands (device_id, applied)
            WHERE applied = FALSE
        ', s);
    END LOOP;
END$$;

-- Legacy (schema vacío): crear en el schema analytics
CREATE TABLE IF NOT EXISTS analytics.pending_commands (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR(100) NOT NULL,
    command     TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    applied     BOOLEAN NOT NULL DEFAULT FALSE,
    applied_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytics_pc_device_applied
ON analytics.pending_commands (device_id, applied)
WHERE applied = FALSE;
