-- 07_sync_velocidad_nominal.sql
-- Tabla de sincronización de velocidad nominal cloud → edge.
-- Permite que el edge gateway y el detector de OEE conozcan la
-- velocidad nominal configurada en cloud para cada producto + línea.

CREATE TABLE IF NOT EXISTS sync_velocidad_nominal (
    id          SERIAL PRIMARY KEY,
    linea_id    INTEGER NOT NULL,
    producto_id INTEGER NOT NULL,
    velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
    factor_conv  INTEGER NOT NULL DEFAULT 1,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (linea_id, producto_id)
);

-- Agrega SYNC_VELOCIDAD_NOMINAL a la constraint de tipos válidos en commands_buffer.
-- Si la constraint ya no existe o ya incluye el valor, el bloque DO lo maneja.
DO $$
BEGIN
  -- Verificar si la constraint ya incluye SYNC_VELOCIDAD_NOMINAL
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'chk_command_type'
      AND pg_catalog.pg_get_constraintdef(oid) LIKE '%SYNC_VELOCIDAD_NOMINAL%'
  ) THEN
    ALTER TABLE commands_buffer DROP CONSTRAINT IF EXISTS chk_command_type;
    ALTER TABLE commands_buffer ADD CONSTRAINT chk_command_type CHECK (command_type = ANY (ARRAY[
      'CREAR_PARADA', 'MODIFICAR_PARADA', 'JUSTIFICAR_PARADA', 'CERRAR_PARADA',
      'ELIMINAR_PARADA', 'ACTUALIZAR_CONFIG', 'INICIAR_CALIBRACION', 'REINICIAR_PIPELINE',
      'COMANDO_CUSTOM', 'SYNC_CATALOG', 'SYNC_PRODUCTOS', 'SYNC_TURNOS',
      'SYNC_USUARIOS', 'SYNC_VARIABLES', 'SYNC_LINEA_PRODUCTO_VARS',
      'SYNC_PRODUCTO_CARACTERISTICAS', 'SYNC_PLANTAS_LINEAS', 'SYNC_PARADAS',
      'SYNC_VELOCIDAD_NOMINAL'
    ]));
  END IF;
END $$;
