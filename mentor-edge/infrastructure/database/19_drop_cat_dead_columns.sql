-- =============================================================
-- Migración 19: Eliminar columnas muertas de sync_categoria_paradas
-- El cloud (fuente de verdad) no tiene descripcion_parada, maquina,
-- parte_maquina, area_responsable en config.cat_programada / cat_no_programada.
-- Esas columnas nunca se poblaron y se eliminan para mantener el esquema
-- del edge idéntico a lo que el cloud envía.
-- Ejecutar por cada schema de línea activo (linea_1, linea_2, etc.)
-- =============================================================

-- Para cada schema de línea (ejecutar sustituyendo SCHEMA por linea_1, linea_2...):
-- ALTER TABLE SCHEMA.sync_categoria_paradas DROP COLUMN IF EXISTS descripcion_parada;
-- ALTER TABLE SCHEMA.sync_categoria_paradas DROP COLUMN IF EXISTS maquina;
-- ALTER TABLE SCHEMA.sync_categoria_paradas DROP COLUMN IF EXISTS parte_maquina;
-- ALTER TABLE SCHEMA.sync_categoria_paradas DROP COLUMN IF EXISTS area_responsable;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^linea_[0-9]+$'
    LOOP
        EXECUTE format('ALTER TABLE %I.sync_categoria_paradas DROP COLUMN IF EXISTS descripcion_parada', r.schema_name);
        EXECUTE format('ALTER TABLE %I.sync_categoria_paradas DROP COLUMN IF EXISTS maquina', r.schema_name);
        EXECUTE format('ALTER TABLE %I.sync_categoria_paradas DROP COLUMN IF EXISTS parte_maquina', r.schema_name);
        EXECUTE format('ALTER TABLE %I.sync_categoria_paradas DROP COLUMN IF EXISTS area_responsable', r.schema_name);
        RAISE NOTICE 'Dropped dead columns from %.sync_categoria_paradas', r.schema_name;
    END LOOP;
END;
$$;

-- También limpiar shared si existe
ALTER TABLE IF EXISTS shared.sync_categoria_paradas DROP COLUMN IF EXISTS descripcion_parada;
ALTER TABLE IF EXISTS shared.sync_categoria_paradas DROP COLUMN IF EXISTS maquina;
ALTER TABLE IF EXISTS shared.sync_categoria_paradas DROP COLUMN IF EXISTS parte_maquina;
ALTER TABLE IF EXISTS shared.sync_categoria_paradas DROP COLUMN IF EXISTS area_responsable;
