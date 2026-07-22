-- Migration: Seed 3 obligatory root stop categories for all lineas that are missing them.
-- Lineas 2, 6, 9, 10 were missing: Refrigerio, Capacitación Obligatoria, Mantenimiento Planificado.
-- These are independent root nodes (padre_id IS NULL) that must exist for every linea.
-- empresa_id is set from the planta → empresa join.
-- Run idempotently: the INSERT checks for existing tipo_parada roots per linea.

-- Step 1: Insert missing obligatory roots
INSERT INTO config.categoria_paradas
  (nombre, codigo, padre_id, empresa_id, linea_id, orden, tipo_parada,
   descripcion_parada, maquina, parte_maquina, area_responsable)
SELECT
  ob.nombre,
  ob.codigo,
  NULL,
  (SELECT p.empresa_id FROM config.lineas lx JOIN config.plantas p ON p.id = lx.planta_id WHERE lx.id = l.linea_id),
  l.linea_id,
  15 + ob.off,
  ob.tipo_parada,
  '', '', '', ''
FROM
  (SELECT DISTINCT linea_id FROM config.categoria_paradas WHERE linea_id IS NOT NULL) l
  CROSS JOIN (
    VALUES
      ('Refrigerio',               'REFRIGERIO', 'REFRIGERIO',    0),
      ('Capacitación Obligatoria', 'CAPACITACION', 'CAPACITACION',  1),
      ('Mantenimiento Planificado','MANTENIMIENTO', 'MANTENIMIENTO', 2)
  ) ob(nombre, codigo, tipo_parada, off)
WHERE NOT EXISTS (
  SELECT 1 FROM config.categoria_paradas cp
  WHERE cp.linea_id = l.linea_id
    AND cp.tipo_parada = ob.tipo_parada
    AND cp.padre_id IS NULL
)
ON CONFLICT DO NOTHING;

-- Step 2: Ensure empresa_id is populated for all inserted rows
UPDATE config.categoria_paradas cp
SET empresa_id = (
  SELECT p.empresa_id
  FROM config.lineas l JOIN config.plantas p ON p.id = l.planta_id
  WHERE l.id = cp.linea_id
)
WHERE cp.empresa_id IS NULL
  AND cp.linea_id IS NOT NULL
  AND cp.tipo_parada IN ('REFRIGERIO', 'CAPACITACION', 'MANTENIMIENTO')
  AND cp.padre_id IS NULL;
