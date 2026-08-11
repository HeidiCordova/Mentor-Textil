-- Migración: agregar columna mode a config.lineas
-- Esta columna registra el modo de operación textil de la línea.
-- y es sincronizada automáticamente desde el edge al cloud.
ALTER TABLE config.lineas ADD COLUMN IF NOT EXISTS mode VARCHAR(20) NOT NULL DEFAULT 'textil';
ALTER TABLE config.lineas ALTER COLUMN mode SET DEFAULT 'textil';
UPDATE config.lineas SET mode = 'textil' WHERE mode <> 'textil';
