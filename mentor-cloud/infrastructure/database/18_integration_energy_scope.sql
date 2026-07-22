-- ─────────────────────────────────────────────────────────────────────────────
-- 18_integration_energy_scope.sql
-- Agrega scopes energy:read y produccion:read a la API de integración
-- ─────────────────────────────────────────────────────────────────────────────

-- Nuevas claves creadas a partir de ahora incluirán los scopes de energía
ALTER TABLE integration.api_keys
    ALTER COLUMN scopes
    SET DEFAULT ARRAY['oee:read','snapshots:read','paradas:read','energy:read','produccion:read'];

-- Índices de performance para las queries de energía por rango temporal
CREATE INDEX IF NOT EXISTS idx_energy_snap_empresa_hora
    ON energy.snapshots (empresa_id, hora DESC);

CREATE INDEX IF NOT EXISTS idx_energy_snap_meter_hora
    ON energy.snapshots (meter_id, hora DESC);

CREATE INDEX IF NOT EXISTS idx_energy_snap_device_hora
    ON energy.snapshots (device_id, hora DESC);
