-- Mentor Edge textil-only cleanup.
--
-- IMPORTANT: run this migration only after deploying an edge-config-service
-- version that no longer reads or writes config.line_config.yolo.
-- The old service selects that column and would fail after it is dropped.

BEGIN;

SET LOCAL lock_timeout = '10s';

LOCK TABLE config.line_config IN ACCESS EXCLUSIVE MODE;

-- Preserve line/device and custom OEE configuration while converting any
-- legacy operating mode to the only supported mode.
UPDATE config.line_config
SET mode = 'textil'
WHERE mode IS DISTINCT FROM 'textil';

-- Some installations already stored mode='textil' while retaining the exact
-- legacy non-textile OEE preset. Replace only that known signature so custom
-- textile thresholds are preserved.
UPDATE config.line_config
SET oee = COALESCE(oee, '{}'::JSONB) || '{
      "micro_stop_max_s": 120,
      "stop_max_s": 86400,
      "snapshot_interval_s": 1800,
      "vel_unit": "uh",
      "vel_nominal_us": 0.008333333
    }'::JSONB
WHERE mode = 'textil'
  AND COALESCE(oee, '{}'::JSONB) @> '{
    "micro_stop_max_s": 210,
    "stop_max_s": 300,
    "snapshot_interval_s": 300
  }'::JSONB;

ALTER TABLE config.line_config
    ALTER COLUMN mode SET DEFAULT 'textil',
    ALTER COLUMN oee SET DEFAULT '{
      "line_name": "",
      "micro_stop_max_s": 120,
      "stop_max_s": 86400,
      "snapshot_interval_s": 1800,
      "vel_unit": "uh",
      "vel_nominal_us": 0.008333333
    }'::JSONB;

ALTER TABLE config.line_config
    DROP CONSTRAINT IF EXISTS chk_line_config_mode_textil_only;

ALTER TABLE config.line_config
    ADD CONSTRAINT chk_line_config_mode_textil_only
    CHECK (mode = 'textil');

ALTER TABLE config.line_config
    DROP COLUMN IF EXISTS yolo;

COMMIT;
