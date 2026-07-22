-- ============================================================
-- 15_planta_turno_dia.sql
-- Ejecutar contra cada mentor_planta_X (NO contra mentor_cloud).
-- Crea public.turno_dia para almacenar el schedule de turnos
-- por planta; migra los datos existentes desde config.turno_dia
-- del cloud y los inserta en la BD de cada planta.
--
-- Uso (ejemplo planta 12):
--   psql -h mentor-cloud-postgres -U mentor -d mentor_planta_12 -f 15_planta_turno_dia.sql
-- ============================================================

CREATE TABLE IF NOT EXISTS public.turno_dia (
    id                 SERIAL PRIMARY KEY,
    linea_id           INT,
    dia_semana         SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6),
    nombre             VARCHAR(100) NOT NULL,
    hora_inicio        TIME NOT NULL,
    hora_fin           TIME NOT NULL,
    color              VARCHAR(20) NOT NULL DEFAULT '#6366f1',
    activo             BOOLEAN NOT NULL DEFAULT TRUE,
    renovacion_semanal BOOLEAN NOT NULL DEFAULT TRUE,
    vigente_desde      DATE NOT NULL DEFAULT CURRENT_DATE,
    creado_en          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pub_turno_dia_scope   ON public.turno_dia (linea_id, vigente_desde);
CREATE INDEX IF NOT EXISTS idx_pub_turno_dia_vigente ON public.turno_dia (vigente_desde);
