CREATE TABLE IF NOT EXISTS energy.config_audit (
    id        SERIAL PRIMARY KEY,
    ts        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    command   VARCHAR(50)  NOT NULL,
    unit_id   INT,
    meter_id  VARCHAR(100),
    params    JSONB,
    result    VARCHAR(10),
    message   TEXT
);

CREATE INDEX IF NOT EXISTS idx_config_audit_ts      ON energy.config_audit (ts DESC);
CREATE INDEX IF NOT EXISTS idx_config_audit_meter   ON energy.config_audit (meter_id, ts DESC);
