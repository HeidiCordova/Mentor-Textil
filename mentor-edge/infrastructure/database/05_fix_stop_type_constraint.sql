-- 05_fix_stop_type_constraint.sql
-- Amplía el CHECK de stop_type para incluir los tipos que Go ya acepta
-- pero la BD rechazaba: REFRIGERIO, CAPACITACION, MANTENIMIENTO

ALTER TABLE stops
  DROP CONSTRAINT IF EXISTS chk_stop_type;

ALTER TABLE stops
  ADD CONSTRAINT chk_stop_type CHECK (stop_type IN (
    'MICROPARADA', 'PARADA_NO_ASIGNADA', 'PROGRAMADA', 'NO_PROGRAMADA',
    'MECANICA', 'ELECTRICA', 'CAMBIO_FORMATO',
    'FALTA_MATERIAL', 'CALIDAD', 'REFRIGERIO',
    'CAPACITACION', 'MANTENIMIENTO', 'OTRA'
  ));
