-- Tabla de sesiones del Laboratorio OEE.
-- Uso exclusivo del módulo de simulación; no afecta datos productivos.

CREATE TABLE IF NOT EXISTS public.oee_lab_sessions (
    id         SERIAL PRIMARY KEY,
    nombre     VARCHAR(200) NOT NULL,
    linea_id   INTEGER,
    inputs     JSONB NOT NULL DEFAULT '{}',
    results    JSONB NOT NULL DEFAULT '{}',
    notas      TEXT,
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oee_lab_sessions_linea    ON public.oee_lab_sessions(linea_id);
CREATE INDEX IF NOT EXISTS idx_oee_lab_sessions_created  ON public.oee_lab_sessions(created_at DESC);
