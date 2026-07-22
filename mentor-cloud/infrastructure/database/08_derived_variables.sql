-- -------------------------------------------------------
-- Migración 08: Variables derivadas (DELTA) para acumuladores
-- Crea automáticamente variantes "por intervalo" para variables
-- que acumulan indefinidamente (conteo, merma).
-- -------------------------------------------------------

INSERT INTO config.variables (nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo)
SELECT
  v.nombre || ' (por intervalo)',
  v.clave || '_DELTA',
  'DELTA:' || v.clave,
  'DERIVADA',
  v.dispositivo_id,
  v.planta_id,
  v.empresa_id,
  true
FROM config.variables v
WHERE v.clave IN ('CONTEO_UNITARIO_PRINCIPAL', 'MERMA', 'CONTEO_1', 'CONTEO_2')
  AND v.activo = true
ON CONFLICT (clave, dispositivo_id) DO NOTHING;
