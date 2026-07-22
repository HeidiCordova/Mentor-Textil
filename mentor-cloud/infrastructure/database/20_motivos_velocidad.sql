-- Agrega la tabla de motivos predefinidos para cambios de velocidad nominal.
-- Las nuevas líneas la reciben a través de 12_linea_template.sql.

DO $$
DECLARE
    s TEXT;
BEGIN
    FOR s IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name ~ '^linea_\d+$'
    LOOP
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.motivos_velocidad (
                id        SERIAL PRIMARY KEY,
                texto     VARCHAR(255) NOT NULL,
                activo    BOOLEAN NOT NULL DEFAULT TRUE,
                orden     INT NOT NULL DEFAULT 0,
                creado_en TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', s);

        -- Insertar valores por defecto si la tabla está vacía
        EXECUTE format('
            INSERT INTO %I.motivos_velocidad (texto, orden)
            SELECT v.texto, v.orden
            FROM (VALUES
                (''Ajuste de velocidad de línea'',   10),
                (''Cambio de formato / producto'',   20),
                (''Optimización de rendimiento'',    30),
                (''Corrección por calidad'',         40),
                (''Instrucción de mantenimiento'',   50),
                (''Orden de supervisión'',           60),
                (''Calibración de equipo'',          70),
                (''Otro'',                           80)
            ) AS v(texto, orden)
            WHERE NOT EXISTS (SELECT 1 FROM %I.motivos_velocidad LIMIT 1)',
            s, s);
    END LOOP;
END $$;
