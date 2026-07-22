-- ============================================================
-- register_device.sql — Onboarding de un nuevo dispositivo Jetson
-- ============================================================
-- Ejecutar UNA VEZ en la BD cloud (mentor_cloud) por cada Jetson nuevo.
-- Requiere que la empresa, planta y línea ya existan.
--
-- Uso:
--   psql "$DATABASE_URL" \
--     -v device_id="'jetson-orin-planta01-l1'" \
--     -v device_nombre="'Jetson Orin Planta01 L1'" \
--     -v empresa_id=7 \
--     -v planta_id=13 \
--     -v linea_id=10 \
--     -v api_key="'$(openssl rand -hex 32)'" \
--     -f register_device.sql
--
-- O sustituye los valores directamente abajo (más simple):
-- ============================================================

BEGIN;

-- ── 1. Registrar dispositivo en gateway.device_registry ──────────
--    api_key debe ser única por dispositivo y coincidir con CLOUD_API_KEY en .env
INSERT INTO gateway.device_registry (device_id, empresa_id, planta_id, linea_id, api_key, active)
VALUES (
    :device_id,
    :empresa_id,
    :planta_id,
    :linea_id,
    :api_key,
    true
)
ON CONFLICT (device_id) DO UPDATE
    SET empresa_id = EXCLUDED.empresa_id,
        planta_id  = EXCLUDED.planta_id,
        linea_id   = EXCLUDED.linea_id,
        api_key    = EXCLUDED.api_key,
        active     = true;

-- ── 2. Crear entrada en config.dispositivos ───────────────────────
INSERT INTO config.dispositivos (device_id, nombre, empresa_id, planta_id, linea_id)
VALUES (:device_id, :device_nombre, :empresa_id, :planta_id, :linea_id)
ON CONFLICT (device_id) DO UPDATE
    SET nombre     = EXCLUDED.nombre,
        empresa_id = EXCLUDED.empresa_id,
        planta_id  = EXCLUDED.planta_id,
        linea_id   = EXCLUDED.linea_id;

-- ── 3. Crear las 11 variables OEE para el dispositivo ─────────────
--    Sin estas filas, UpdateVariablesFromHeadData es un no-op y el
--    dashboard de Variables no muestra lecturas en vivo.
DO $$
DECLARE
    v_dispositivo_id INTEGER;
BEGIN
    SELECT id INTO v_dispositivo_id
    FROM config.dispositivos
    WHERE device_id = :device_id;

    IF v_dispositivo_id IS NULL THEN
        RAISE EXCEPTION 'dispositivo % no encontrado tras INSERT', :device_id;
    END IF;

    INSERT INTO config.variables (nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id)
    VALUES
        ('Tiempo Disponible',            'T_DISPONIBLE',               '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Microparada',           'T_MICROPARADA',              '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Parada No Asignada',    'T_PARADA_NO_ASIGNADA',       '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Conteo de Cortes',             'CONTEO_1',                   '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Parada Programada',     'T_PARADA_PROGRAMADA',        '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Parada No Programada',  'T_PARADA_NO_PROGRAMADA',     '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Refrigerio',            'T_REFRIGERIO',               '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Capacitación',          'T_CAPACITACION_OBLIGATORIA', '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Tiempo Mantenimiento',         'T_MANTENIMIENTO_PLANIFICADO','0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Merma',                        'MERMA',                      '0', 'integer', v_dispositivo_id, :planta_id, :empresa_id),
        ('Intervalo OEE (s)',            'OEE_INTERVAL_S',             '300', 'integer', v_dispositivo_id, :planta_id, :empresa_id)
    ON CONFLICT (clave, dispositivo_id) DO NOTHING;

    RAISE NOTICE 'Dispositivo % registrado OK (dispositivo_id=%)', :device_id, v_dispositivo_id;
END;
$$;

COMMIT;
