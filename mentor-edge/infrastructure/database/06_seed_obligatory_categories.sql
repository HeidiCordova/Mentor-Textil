-- Migration 06: Seed obligatory stop categories (REFRIGERIO, CAPACITACION, MANTENIMIENTO)
-- These are independent root nodes, NOT sub-categories of PROGRAMADA/NO_PROGRAMADA.
-- Must be present for every linea. Run once per linea_id present in the system.
--
-- Applied: 2025 — edge DB (docker-postgres-1 @ 192.168.100.31)

INSERT INTO sync_categoria_paradas
  (id, nombre, codigo, padre_id, empresa_id, linea_id, orden, tipo_parada,
   descripcion_parada, maquina, parte_maquina, area_responsable)
VALUES
  (1001, 'Refrigerio',    'REFRIGERIO',    NULL, NULL,  9, 15, 'REFRIGERIO',    '', '', '', ''),
  (1002, 'Capacitación Obligatoria',  'CAPACITACION',  NULL, NULL,  9, 16, 'CAPACITACION',  '', '', '', ''),
  (1003, 'Mantenimiento Planificado', 'MANTENIMIENTO', NULL, NULL,  9, 17, 'MANTENIMIENTO', '', '', '', ''),
  (1004, 'Refrigerio',               'REFRIGERIO',    NULL, NULL, 10, 15, 'REFRIGERIO',    '', '', '', ''),
  (1005, 'Capacitación Obligatoria',  'CAPACITACION',  NULL, NULL, 10, 16, 'CAPACITACION',  '', '', '', ''),
  (1006, 'Mantenimiento Planificado', 'MANTENIMIENTO', NULL, NULL, 10, 17, 'MANTENIMIENTO', '', '', '', '')
ON CONFLICT (id) DO UPDATE SET
  nombre      = EXCLUDED.nombre,
  tipo_parada = EXCLUDED.tipo_parada,
  orden       = EXCLUDED.orden;
