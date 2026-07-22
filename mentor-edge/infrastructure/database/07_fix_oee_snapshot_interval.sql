-- Migración: corregir snapshot_interval_s de 60 → 300 segundos en todos los dispositivos.
-- El intervalo OEE debe ser 300 s (5 minutos) para que T_DISPONIBLE refleje la ventana real.

UPDATE device_configs
SET oee = jsonb_set(oee, '{snapshot_interval_s}', '300')
WHERE (oee->>'snapshot_interval_s')::int = 60;
