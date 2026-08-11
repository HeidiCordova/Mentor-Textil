-- ============================================================
-- Migración 10: Índices compuestos para queries de dashboard
-- Complementa 06_performance_indexes.sql
-- Idempotente: usa IF NOT EXISTS
-- ============================================================

-- Cubre: WHERE linea_id=X AND fecha=Y AND interval_s >= 1800
-- GetStats, GetCharts, GetReports filtran siempre por interval_s
CREATE INDEX IF NOT EXISTS idx_ingest_oee_linea_fecha_interval
    ON ingest.oee_snapshots(linea_id, fecha DESC, interval_s);

-- Cubre: GROUP BY turno en análisis de turno
CREATE INDEX IF NOT EXISTS idx_ingest_oee_turno_fecha
    ON ingest.oee_snapshots(linea_id, turno, fecha DESC);

-- Cubre: paradas abiertas (fin IS NULL) — query muy frecuente
CREATE INDEX IF NOT EXISTS idx_ingest_paradas_abiertas
    ON ingest.paradas(linea_id, inicio DESC)
    WHERE fin IS NULL;

-- Cubre: historial de paradas por rango de fechas
CREATE INDEX IF NOT EXISTS idx_ingest_paradas_inicio
    ON ingest.paradas(linea_id, inicio DESC);

-- Cubre: production_runs activos (ended_at IS NULL)
CREATE INDEX IF NOT EXISTS idx_ingest_runs_activos
    ON ingest.production_runs(linea_id, started_at DESC)
    WHERE ended_at IS NULL;

-- Cubre: historial de production_runs
CREATE INDEX IF NOT EXISTS idx_ingest_runs_started
    ON ingest.production_runs(linea_id, started_at DESC);

-- Cubre: device_registry lookup por api_key (ruta crítica de cada request de edge)
CREATE UNIQUE INDEX IF NOT EXISTS idx_gateway_registry_apikey
    ON gateway.device_registry(api_key)
    WHERE active = true;

-- Cubre: refresh_tokens lookup por hash (ruta crítica de /auth/refresh)
CREATE INDEX IF NOT EXISTS idx_identity_refresh_hash
    ON identity.refresh_tokens(token_hash)
    WHERE revocado = false;
