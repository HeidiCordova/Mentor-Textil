-- Migración: agregar columna mode a config.lineas
-- Esta columna registra el modo de operación de la línea (textil/botellas)
-- y es sincronizada automáticamente desde el edge al cloud.
ALTER TABLE config.lineas ADD COLUMN IF NOT EXISTS mode VARCHAR(20) NOT NULL DEFAULT 'botellas';
