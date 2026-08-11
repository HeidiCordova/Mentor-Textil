-- Mentor textile Node-RED outbox hardening (MariaDB 10.6+).
--
-- Safety:
--   * This migration never deletes or rewrites readings.
--   * mqtt_lecturas already needs UNIQUE(device,time).
--   * The unique snapshot index deliberately fails if duplicate devices exist;
--     run the documented preflight before applying it.

CREATE UNIQUE INDEX IF NOT EXISTS mqtt_snapshot_device_uq
    ON mqtt_snapshot (device);

CREATE INDEX IF NOT EXISTS mqtt_lecturas_mqtt_pending_idx
    ON mqtt_lecturas (mentor_id, status, device, time);

CREATE INDEX IF NOT EXISTS mqtt_lecturas_rest_pending_idx
    ON mqtt_lecturas (mentor_id, restful, device, time);
