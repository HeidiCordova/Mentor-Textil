-- =============================================================
-- Migración 06: Índices de rendimiento para escala (50+ Jetsons)
-- Aplica sobre: ingest.oee_snapshots, ingest.raw_events
-- Idempotente: usa IF NOT EXISTS
-- =============================================================

-- ── oee_snapshots ────────────────────────────────────────────
-- Queries por línea + fecha (más frecuente en dashboard cloud)
CREATE INDEX IF NOT EXISTS idx_ingest_oee_linea
    ON ingest.oee_snapshots(linea_id, fecha DESC);

-- Queries por planta + fecha (vista agrupada por planta)
CREATE INDEX IF NOT EXISTS idx_ingest_oee_planta
    ON ingest.oee_snapshots(planta_id, fecha DESC);

-- Queries multi-tenant: empresa → linea → fecha
-- Cubre el 90% de las queries del dashboard analytics
CREATE INDEX IF NOT EXISTS idx_ingest_oee_empresa_linea
    ON ingest.oee_snapshots(empresa_id, linea_id, fecha DESC);

-- Queries por hora exacta (series temporales intra-día)
CREATE INDEX IF NOT EXISTS idx_ingest_oee_hora
    ON ingest.oee_snapshots(hora DESC);

-- ── raw_events ───────────────────────────────────────────────
-- Queries de log por empresa + tiempo
CREATE INDEX IF NOT EXISTS idx_ingest_raw_empresa_ts
    ON ingest.raw_events(empresa_id, timestamp_edge DESC);

-- Queries de auditoría por linea + tipo de evento
CREATE INDEX IF NOT EXISTS idx_ingest_raw_linea_type
    ON ingest.raw_events(linea_id, event_type, timestamp_edge DESC);
