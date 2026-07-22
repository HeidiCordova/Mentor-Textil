-- ============================================================
-- fix_turno_dia_permissions.sql
-- Fix permissions for turno_dia table in each plant database
-- Run against each mentor_planta_X database
--
-- Usage (example for planta 14):
--   psql -h 152.53.253.59 -U postgres -d mentor_planta_14 -f fix_turno_dia_permissions.sql
-- ============================================================

-- Grant all necessary permissions on turno_dia table to mentor user
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.turno_dia TO mentor;

-- Grant usage and update permissions on the sequence
GRANT USAGE, SELECT ON SEQUENCE public.turno_dia_id_seq TO mentor;

-- Verify permissions
\dp public.turno_dia
