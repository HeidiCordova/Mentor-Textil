-- Migration 07: production_runs sync table
-- Recibe los production_runs sincronizados desde los dispositivos edge.
-- La velocidad_nominal se une desde config.linea_productos para calcular Rendimiento.

CREATE TABLE IF NOT EXISTS analytics.production_runs (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID NOT NULL,
    device_id       VARCHAR(100) NOT NULL,
    linea_id        INT REFERENCES config.lineas(id) ON DELETE SET NULL,
    planta_id       INT,
    empresa_id      INT REFERENCES identity.empresas(id) ON DELETE SET NULL,
    producto_id     INT,
    sku             VARCHAR(64),
    nombre          TEXT,
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    creado_en       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actualizado_en  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_production_run_id UNIQUE (device_id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_prod_runs_device     ON analytics.production_runs(device_id);
CREATE INDEX IF NOT EXISTS idx_prod_runs_linea_ts   ON analytics.production_runs(linea_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_prod_runs_empresa_ts ON analytics.production_runs(empresa_id, started_at DESC);
