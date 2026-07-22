-- Migration 05: Fix device scope for jetson-orin-textil
-- Connects OEE time variables to the cloud DB pipeline.
--
-- Problems solved:
--  1. gateway.device_registry had planta_id=NULL, linea_id=NULL for jetson devices
--  2. config.dispositivos had no entry for 'jetson-orin-textil'
--  3. config.variables had empresa_id=NULL for OEE variables of empresa 7 devices
--  4. No OEE variables existed for jetson-orin-textil (UpdateVariablesFromHeadData was a no-op)
--
-- This migration is idempotent (safe to run multiple times).

BEGIN;

-- Fix 1: Assign planta_id and linea_id in gateway.device_registry
UPDATE gateway.device_registry
SET planta_id = 13,
    linea_id  = 10
WHERE device_id IN ('jetson-orin-textil', 'orin')
  AND (planta_id IS DISTINCT FROM 13 OR linea_id IS DISTINCT FROM 10);

-- Fix 2: Ensure jetson-orin-textil exists in config.dispositivos
INSERT INTO config.dispositivos (device_id, nombre, empresa_id, planta_id, linea_id)
VALUES ('jetson-orin-textil', 'Jetson Orin Textil', 7, 13, 10)
ON CONFLICT (device_id) DO UPDATE
    SET empresa_id = 7,
        planta_id  = 13,
        linea_id   = 10,
        nombre     = COALESCE(EXCLUDED.nombre, config.dispositivos.nombre);

-- Fix 3: Propagate empresa_id=7 to all variables of empresa_id=7 devices
UPDATE config.variables v
SET empresa_id = 7
FROM config.dispositivos d
WHERE v.dispositivo_id = d.id
  AND d.empresa_id = 7
  AND v.empresa_id IS NULL;

-- Fix 4: Insert the 11 OEE variables for jetson-orin-textil (if missing)
DO $$
DECLARE
    v_dispositivo_id INTEGER;
BEGIN
    SELECT id INTO v_dispositivo_id
    FROM config.dispositivos
    WHERE device_id = 'jetson-orin-textil';

    IF v_dispositivo_id IS NULL THEN
        RAISE EXCEPTION 'jetson-orin-textil not found in config.dispositivos after insert';
    END IF;

    INSERT INTO config.variables (nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id)
    VALUES
        ('Tiempo Disponible',           'T_DISPONIBLE',                '0', 'OEE', v_dispositivo_id, 13, 7),
        ('Tiempo Microparada',          'T_MICROPARADA',               '0', 'OEE', v_dispositivo_id, 13, 7),
        ('Tiempo Parada No Asignada',   'T_PARADA_NO_ASIGNADA',        '0', 'OEE', v_dispositivo_id, 13, 7),
        ('Conteo 1',                    'CONTEO_1',                    '0', 'OEE', v_dispositivo_id, 13, 7),
        ('Conteo 2',                    'CONTEO_2',                    '0', 'OEE', v_dispositivo_id, 13, 7),
        ('T Capacitacion Obligatoria',  'T_CAPACITACION_OBLIGATORIA',  '0', 'OEE', v_dispositivo_id, 13, 7),
        ('T Mantenimiento Planificado', 'T_MANTENIMIENTO_PLANIFICADO', '0', 'OEE', v_dispositivo_id, 13, 7),
        ('T Parada No Programada',      'T_PARADA_NO_PROGRAMADA',      '0', 'OEE', v_dispositivo_id, 13, 7),
        ('T Parada Programada',         'T_PARADA_PROGRAMADA',         '0', 'OEE', v_dispositivo_id, 13, 7),
        ('T Refrigerio',                'T_REFRIGERIO',                '0', 'OEE', v_dispositivo_id, 13, 7),
        ('Merma',                       'MERMA',                       '0', 'OEE', v_dispositivo_id, 13, 7)
    ON CONFLICT (clave, dispositivo_id) DO NOTHING;
END;
$$;

COMMIT;

-- Verify
SELECT d.device_id, v.clave, v.valor, v.empresa_id, v.tipo
FROM config.variables v
JOIN config.dispositivos d ON d.id = v.dispositivo_id
WHERE d.device_id = 'jetson-orin-textil'
ORDER BY v.clave;
