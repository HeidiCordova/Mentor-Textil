-- Normaliza instalaciones existentes al unico modo soportado: textil.
ALTER TABLE config.lineas
  ADD COLUMN IF NOT EXISTS mode VARCHAR(20) NOT NULL DEFAULT 'textil';
ALTER TABLE config.lineas ALTER COLUMN mode SET DEFAULT 'textil';
UPDATE config.lineas SET mode = 'textil' WHERE mode IS DISTINCT FROM 'textil';
ALTER TABLE config.lineas DROP CONSTRAINT IF EXISTS lineas_mode_textil_check;
ALTER TABLE config.lineas
  ADD CONSTRAINT lineas_mode_textil_check CHECK (mode = 'textil');

-- Retira variables heredadas exclusivas del flujo eliminado.
DELETE FROM config.variables
WHERE clave IN ('CONTEO_2', 'CONTEO_2_DELTA', 'SABOR');
