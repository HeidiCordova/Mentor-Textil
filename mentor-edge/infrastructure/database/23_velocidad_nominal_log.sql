-- Agrega la tabla de historial de cambios de velocidad nominal (origen edge/tablet).

DO $$
DECLARE
    s TEXT;
BEGIN
    FOR s IN
        SELECT schema_name FROM information_schema.schemata
        WHERE schema_name ~ '^linea_\d+$'
    LOOP
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.velocidad_nominal_log (
                id                    BIGSERIAL PRIMARY KEY,
                producto_id           INT NOT NULL,
                sku                   TEXT,
                velocidad_us_anterior DOUBLE PRECISION,
                velocidad_us_nueva    DOUBLE PRECISION NOT NULL,
                factor_conv_anterior  INT,
                factor_conv_nueva     INT NOT NULL DEFAULT 1,
                motivo                TEXT,
                usuario               TEXT,
                origen                TEXT NOT NULL DEFAULT ''edge'',
                cambiado_en           TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )', s);

        EXECUTE format('
            CREATE INDEX IF NOT EXISTS %I ON %I.velocidad_nominal_log (producto_id, cambiado_en DESC)',
            'idx_' || s || '_vnlog_prod', s);

        EXECUTE format('
            CREATE INDEX IF NOT EXISTS %I ON %I.velocidad_nominal_log (cambiado_en DESC)',
            'idx_' || s || '_vnlog_ts', s);
    END LOOP;
END $$;
