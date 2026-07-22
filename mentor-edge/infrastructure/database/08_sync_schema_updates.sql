-- 08_sync_schema_updates.sql
-- Migración: agrega columnas faltantes a sync_categoria_paradas y sync_productos
-- en instancias Jetson desplegadas antes de que el init.sql fuera actualizado.
-- Seguro ejecutar múltiples veces (ADD COLUMN IF NOT EXISTS).

-- sync_categoria_paradas: columnas de scope y metadatos de parada
ALTER TABLE sync_categoria_paradas
  ADD COLUMN IF NOT EXISTS linea_id            INTEGER,
  ADD COLUMN IF NOT EXISTS orden               INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS tipo_parada         VARCHAR(30),
  ADD COLUMN IF NOT EXISTS descripcion_parada  VARCHAR(300) DEFAULT '',
  ADD COLUMN IF NOT EXISTS maquina             VARCHAR(200) DEFAULT '',
  ADD COLUMN IF NOT EXISTS parte_maquina       VARCHAR(200) DEFAULT '',
  ADD COLUMN IF NOT EXISTS area_responsable    VARCHAR(200) DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_sync_cat_linea ON sync_categoria_paradas (linea_id);

-- sync_productos: scope por línea y parámetros de velocidad
ALTER TABLE sync_productos
  ADD COLUMN IF NOT EXISTS linea_id    INTEGER,
  ADD COLUMN IF NOT EXISTS velocidad_us DOUBLE PRECISION NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS factor_conv  INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_sync_prod_linea ON sync_productos (linea_id);
