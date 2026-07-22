-- Migration 09: dedicated table for vision detection events.
-- Populated automatically via trigger on events_buffer (event_type = 'CORTE').
-- Purpose: structured, indexed audit trail for every garment detection,
--          separating camera events from the generic event bus.

CREATE TABLE IF NOT EXISTS vision_detections (
    id             SERIAL       PRIMARY KEY,
    detection_id   UUID         NOT NULL,
    device_id      VARCHAR(64)  NOT NULL,
    detected_at    TIMESTAMPTZ  NOT NULL,
    line_code      VARCHAR(64),
    confidence     REAL,
    signal_edge    REAL,
    signal_color   REAL,
    signal_flow    REAL,
    signal_beige   REAL,
    roi_id         VARCHAR(32),
    fsm_state      VARCHAR(32),

    CONSTRAINT vision_detections_detection_id_key UNIQUE (detection_id)
);

CREATE INDEX idx_vd_device_ts  ON vision_detections (device_id, detected_at DESC);
CREATE INDEX idx_vd_line_ts    ON vision_detections (line_code, detected_at DESC);
CREATE INDEX idx_vd_confidence ON vision_detections (confidence);

ALTER TABLE vision_detections SET (autovacuum_vacuum_scale_factor = 0.05);


CREATE OR REPLACE FUNCTION extract_vision_detection()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.event_type <> 'CORTE' THEN
        RETURN NEW;
    END IF;

    INSERT INTO vision_detections (
        detection_id,
        device_id,
        detected_at,
        line_code,
        confidence,
        signal_edge,
        signal_color,
        signal_flow,
        signal_beige,
        roi_id
    )
    VALUES (
        NEW.event_id,
        NEW.device_id,
        NEW.timestamp,
        NEW.payload ->> 'line_code',
        (NEW.payload ->> 'confidence')::REAL,
        (NEW.payload -> 'signals' ->> 'edge')::REAL,
        (NEW.payload -> 'signals' ->> 'color')::REAL,
        (NEW.payload -> 'signals' ->> 'flow')::REAL,
        (NEW.payload -> 'signals' ->> 'beige')::REAL,
        NEW.payload ->> 'roi_id'
    )
    ON CONFLICT (detection_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER trg_extract_vision_detection
AFTER INSERT ON events_buffer
FOR EACH ROW
EXECUTE FUNCTION extract_vision_detection();
