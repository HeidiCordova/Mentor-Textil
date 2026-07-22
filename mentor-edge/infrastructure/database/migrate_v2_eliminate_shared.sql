-- =============================================================
-- migrate_v2_eliminate_shared.sql
-- Migra datos de shared.* → linea_3.* (tablas ya existen en linea_3)
-- Luego elimina el schema shared.
--
-- PRE-REQUISITO: ejecutar linea_template.sql para linea_3 primero:
--   sed 's/{schema}/linea_3/g' linea_template.sql | psql -U mentor -d mentor_edge
--
-- EJECUTAR CON:
--   docker exec -i docker-postgres-1 psql -U mentor -d mentor_edge < migrate_v2_eliminate_shared.sql
-- =============================================================

BEGIN;

-- ─── 1. Copiar datos de shared → linea_3 ───────────────────────────
-- Columnas explícitas para tablas con diferente orden entre schemas.

-- line_config (diferente orden: tablet antes/después de created_at)
INSERT INTO linea_3.line_config (id, device_id, config_version, roi, thresholds, fsm, mode, camera, oee, cloud, tablet, created_at, updated_at)
  SELECT id, device_id, config_version, roi, thresholds, fsm, mode, camera, oee, cloud, tablet, created_at, updated_at
  FROM shared.line_config
  ON CONFLICT (device_id) DO NOTHING;

-- commands_buffer (diferente orden: status antes/después de issued_by)
INSERT INTO linea_3.commands_buffer (id, command_id, device_id, command_type, payload, status, issued_by, issued_at, idempotency_key, result, error_message, applied_at, created_at)
  SELECT id, command_id, device_id, command_type, payload, status, issued_by, issued_at, idempotency_key, result, error_message, applied_at, created_at
  FROM shared.commands_buffer
  ON CONFLICT (id) DO NOTHING;

-- audit_log (mismo orden)
INSERT INTO linea_3.audit_log
  SELECT * FROM shared.audit_log
  ON CONFLICT (id) DO NOTHING;

-- health_logs (mismo orden)
INSERT INTO linea_3.health_logs
  SELECT * FROM shared.health_logs
  ON CONFLICT (id) DO NOTHING;

-- calibration_history (mismo orden)
INSERT INTO linea_3.calibration_history
  SELECT * FROM shared.calibration_history
  ON CONFLICT (id) DO NOTHING;

-- sync_categoria_paradas (diferente orden: synced_at movido al final)
INSERT INTO linea_3.sync_categoria_paradas (id, nombre, codigo, padre_id, empresa_id, linea_id, orden, tipo_parada, descripcion_parada, maquina, parte_maquina, area_responsable, synced_at)
  SELECT id, nombre, codigo, padre_id, empresa_id, linea_id, orden, tipo_parada, descripcion_parada, maquina, parte_maquina, area_responsable, synced_at
  FROM shared.sync_categoria_paradas
  ON CONFLICT (id) DO NOTHING;

-- sync_productos (diferente orden)
INSERT INTO linea_3.sync_productos (id, codigo, nombre, empresa_id, activo, linea_id, velocidad_us, factor_conv, synced_at)
  SELECT id, codigo, nombre, empresa_id, activo, linea_id, velocidad_us, factor_conv, synced_at
  FROM shared.sync_productos
  ON CONFLICT (id) DO NOTHING;

-- sync_turnos (mismo orden)
INSERT INTO linea_3.sync_turnos
  SELECT * FROM shared.sync_turnos
  ON CONFLICT (id) DO NOTHING;

-- sync_usuarios (diferente orden)
INSERT INTO linea_3.sync_usuarios (id, username, email, nombre, rol_id, rol, empresa_id, activo, synced_at)
  SELECT id, username, email, nombre, rol_id, rol, empresa_id, activo, synced_at
  FROM shared.sync_usuarios
  ON CONFLICT (id) DO NOTHING;

-- sync_variables (diferente orden)
INSERT INTO linea_3.sync_variables (id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo, synced_at)
  SELECT id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo, synced_at
  FROM shared.sync_variables
  ON CONFLICT (id) DO NOTHING;

-- sync_linea_producto_vars (mismo orden)
INSERT INTO linea_3.sync_linea_producto_vars
  SELECT * FROM shared.sync_linea_producto_vars
  ON CONFLICT (id) DO NOTHING;

-- sync_producto_caracteristicas (mismo orden)
INSERT INTO linea_3.sync_producto_caracteristicas
  SELECT * FROM shared.sync_producto_caracteristicas
  ON CONFLICT (id) DO NOTHING;

-- sync_plantas (diferente orden)
INSERT INTO linea_3.sync_plantas (id, nombre, empresa_id, empresa_nombre, activo, synced_at)
  SELECT id, nombre, empresa_id, empresa_nombre, activo, synced_at
  FROM shared.sync_plantas
  ON CONFLICT (id) DO NOTHING;

-- sync_lineas (mismo orden)
INSERT INTO linea_3.sync_lineas
  SELECT * FROM shared.sync_lineas
  ON CONFLICT (id) DO NOTHING;

-- sync_velocidad_nominal (mismo orden)
INSERT INTO linea_3.sync_velocidad_nominal
  SELECT * FROM shared.sync_velocidad_nominal
  ON CONFLICT (linea_id, producto_id) DO NOTHING;

-- ─── 2. Actualizar secuencias ──────────────────────────────────────
-- Asegurar que los serial/sequences apuntan al último ID

DO $$
DECLARE
    max_id BIGINT;
BEGIN
    SELECT COALESCE(MAX(id), 0) INTO max_id FROM linea_3.line_config;
    PERFORM setval(pg_get_serial_sequence('linea_3.line_config', 'id'), GREATEST(max_id, 1));

    SELECT COALESCE(MAX(id), 0) INTO max_id FROM linea_3.commands_buffer;
    PERFORM setval(pg_get_serial_sequence('linea_3.commands_buffer', 'id'), GREATEST(max_id, 1));

    SELECT COALESCE(MAX(id), 0) INTO max_id FROM linea_3.audit_log;
    PERFORM setval(pg_get_serial_sequence('linea_3.audit_log', 'id'), GREATEST(max_id, 1));

    SELECT COALESCE(MAX(id), 0) INTO max_id FROM linea_3.health_logs;
    PERFORM setval(pg_get_serial_sequence('linea_3.health_logs', 'id'), GREATEST(max_id, 1));

    SELECT COALESCE(MAX(id), 0) INTO max_id FROM linea_3.calibration_history;
    PERFORM setval(pg_get_serial_sequence('linea_3.calibration_history', 'id'), GREATEST(max_id, 1));
END $$;

-- ─── 3. Eliminar schema shared ────────────────────────────────────
DROP SCHEMA IF EXISTS shared CASCADE;

-- ─── 4. Verificar ─────────────────────────────────────────────────
DO $$
DECLARE
    tbl_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO tbl_count
    FROM information_schema.tables
    WHERE table_schema = 'linea_3';
    RAISE NOTICE 'linea_3 tiene % tablas', tbl_count;

    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'shared') THEN
        RAISE WARNING 'shared schema aún existe!';
    ELSE
        RAISE NOTICE 'shared schema eliminado correctamente';
    END IF;
END $$;

COMMIT;
