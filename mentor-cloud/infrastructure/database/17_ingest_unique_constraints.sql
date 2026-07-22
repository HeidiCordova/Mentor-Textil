-- Unique constraints for idempotent ingest (at-least-once delivery from edge).

-- Schema ingest (central)
CREATE UNIQUE INDEX IF NOT EXISTS uq_raw_events_device_ts
    ON ingest.raw_events(device_id, timestamp_edge);
CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora
    ON ingest.oee_snapshots(device_id, hora);

-- Per-linea schemas (if tables exist)
DO $$
DECLARE
    s TEXT;
BEGIN
    FOR s IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name LIKE 'linea_%'
    LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = s AND table_name = 'raw_events'
        ) THEN
            EXECUTE format(
                'CREATE UNIQUE INDEX IF NOT EXISTS uq_raw_events_device_ts ON %I.raw_events(device_id, timestamp_edge)',
                s
            );
        END IF;

        IF EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = s AND table_name = 'oee_snapshots'
        ) THEN
            EXECUTE format(
                'CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora ON %I.oee_snapshots(device_id, hora)',
                s
            );
        END IF;
    END LOOP;
END $$;
