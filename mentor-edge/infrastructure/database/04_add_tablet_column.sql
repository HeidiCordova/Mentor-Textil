-- 04_add_tablet_column.sql
-- Migración: agrega columna tablet a line_config si no existe
-- Ejecutar una sola vez en instancias existentes.

ALTER TABLE line_config
  ADD COLUMN IF NOT EXISTS tablet JSONB;
