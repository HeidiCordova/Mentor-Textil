-- Add the structured CORTE history required by GET /vision/count to every
-- already-provisioned production schema.
--
-- IMPORTANT: run this migration with vision-event-detector stopped. Trigger
-- replacement, index creation and the historical backfill take table locks.
-- Run it immediately after an exact five-minute boundary with production
-- stopped. The initial baseline is intentionally zero; verify the latest
-- legacy L1_CONTEO_1 snapshot is still zero before executing this file.
--
-- Safe to rerun:
--   * tables, columns and indexes are created only when missing;
--   * every trigger calling extract_vision_detection is discovered and
--     removed before one canonical trigger is installed;
--   * historical rows are deduplicated by detection_id;
--   * malformed legacy signal values are stored as NULL, not allowed to abort
--     the whole migration.

DO $migration$
DECLARE
    target RECORD;
    extraction_trigger RECORD;
    has_detection_unique BOOLEAN;
    extraction_trigger_count INTEGER;
    canonical_trigger_count INTEGER;
    migrated_schemas INTEGER := 0;
    cutover_epoch TIMESTAMPTZ :=
        date_trunc('hour', CURRENT_TIMESTAMP)
        + ((EXTRACT(MINUTE FROM CURRENT_TIMESTAMP)::INTEGER / 5)
           * INTERVAL '5 minutes');
    cutover_baseline BIGINT := 0;
BEGIN
    RAISE NOTICE
        'Vision counter cutover: epoch=%, baseline=%',
        cutover_epoch,
        cutover_baseline;

    FOR target IN
        SELECT s.schema_name
        FROM information_schema.schemata s
        WHERE s.schema_name ~ '^linea_[0-9]+$'
          AND EXISTS (
              SELECT 1
              FROM information_schema.tables t
              WHERE t.table_schema = s.schema_name
                AND t.table_name = 'events_buffer'
          )
        ORDER BY s.schema_name
    LOOP
        migrated_schemas := migrated_schemas + 1;
        RAISE NOTICE 'Migrating %.vision_detections', target.schema_name;

        -- Some early linea_template versions did not include device_id in the
        -- source table. Resiliencia writes that column, so normalize it before
        -- installing the extractor as well as before future event ingestion.
        EXECUTE format(
            'ALTER TABLE %I.events_buffer
             ADD COLUMN IF NOT EXISTS device_id VARCHAR(100)',
            target.schema_name
        );

        -- Covers newly provisioned schemas. The ALTER statements immediately
        -- below normalize the two known historical table variants.
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I.vision_detections (
                id           BIGSERIAL   PRIMARY KEY,
                detection_id UUID        NOT NULL,
                device_id    VARCHAR(100),
                detected_at  TIMESTAMPTZ NOT NULL,
                line_code    VARCHAR(64),
                confidence   REAL,
                signal_edge  REAL,
                signal_color REAL,
                signal_flow  REAL,
                signal_beige REAL,
                roi_id       VARCHAR(32),
                fsm_state    VARCHAR(32)
            )',
            target.schema_name
        );
        EXECUTE format(
            'ALTER TABLE %I.vision_detections
             ADD COLUMN IF NOT EXISTS device_id VARCHAR(100)',
            target.schema_name
        );
        EXECUTE format(
            'ALTER TABLE %I.vision_detections
             ALTER COLUMN device_id TYPE VARCHAR(100),
             ALTER COLUMN device_id DROP NOT NULL',
            target.schema_name
        );
        EXECUTE format(
            'ALTER TABLE %I.vision_detections
             ADD COLUMN IF NOT EXISTS fsm_state VARCHAR(32)',
            target.schema_name
        );

        -- Reuse an existing one-column UNIQUE constraint/index when either the
        -- legacy public schema or linea_template already supplied one.
        SELECT EXISTS (
            SELECT 1
            FROM pg_catalog.pg_index i
            JOIN pg_catalog.pg_class tbl
              ON tbl.oid = i.indrelid
            JOIN pg_catalog.pg_namespace ns
              ON ns.oid = tbl.relnamespace
            JOIN pg_catalog.pg_attribute attr
              ON attr.attrelid = tbl.oid
             AND attr.attnum = i.indkey[0]
            WHERE ns.nspname = target.schema_name
              AND tbl.relname = 'vision_detections'
              AND i.indisunique
              AND i.indisvalid
              AND i.indisready
              AND i.indpred IS NULL
              AND i.indnkeyatts = 1
              AND attr.attname = 'detection_id'
        )
        INTO has_detection_unique;

        IF NOT has_detection_unique THEN
            EXECUTE format(
                'CREATE UNIQUE INDEX %I
                 ON %I.vision_detections (detection_id)',
                'idx_' || target.schema_name || '_vd_detection',
                target.schema_name
            );
        END IF;

        EXECUTE format(
            'CREATE INDEX IF NOT EXISTS %I
             ON %I.vision_detections (detected_at DESC)',
            'idx_' || target.schema_name || '_vd_ts',
            target.schema_name
        );

        -- Raw-machine counter used by Node-RED. The first migration creates a
        -- stable epoch with value zero; reruns preserve both epoch and value.
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I.vision_counter_state (
                counter_name  VARCHAR(64) PRIMARY KEY,
                counter_epoch TIMESTAMPTZ NOT NULL,
                counter_baseline BIGINT   NOT NULL DEFAULT 0
                    CHECK (counter_baseline >= 0),
                counter_value BIGINT      NOT NULL DEFAULT 0
                    CHECK (counter_value >= 0),
                updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
            )',
            target.schema_name
        );
        EXECUTE format(
            'INSERT INTO %I.vision_counter_state (
                counter_name,
                counter_epoch,
                counter_baseline,
                counter_value
             )
             VALUES (''CORTE_TOTAL'', $1, $2, $2)
             ON CONFLICT (counter_name) DO NOTHING',
            target.schema_name
        )
        USING cutover_epoch, cutover_baseline;

        -- The first read for each five-minute boundary freezes its cumulative
        -- value. A retry of the same boundary therefore remains idempotent even
        -- if an older detector event arrives late.
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I.vision_counter_snapshots (
                counter_name  VARCHAR(64) NOT NULL,
                counter_epoch TIMESTAMPTZ NOT NULL,
                counter_until TIMESTAMPTZ NOT NULL,
                counter_value BIGINT      NOT NULL
                    CHECK (counter_value >= 0),
                state_updated_at TIMESTAMPTZ NOT NULL,
                created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY (counter_name, counter_epoch, counter_until)
            )',
            target.schema_name
        );

        EXECUTE format(
            $function$
            CREATE OR REPLACE FUNCTION %I.increment_vision_counter()
            RETURNS TRIGGER AS $body$
            DECLARE
                active_epoch TIMESTAMPTZ;
                affected_rows INTEGER;
            BEGIN
                SELECT counter_epoch
                INTO active_epoch
                FROM %I.vision_counter_state
                WHERE counter_name = 'CORTE_TOTAL';

                IF NOT FOUND THEN
                    RAISE EXCEPTION
                        'CORTE_TOTAL counter state is missing';
                END IF;

                IF NEW.detected_at < active_epoch THEN
                    RETURN NEW;
                END IF;

                UPDATE %I.vision_counter_state
                SET counter_value = counter_value + 1,
                    updated_at = CURRENT_TIMESTAMP
                WHERE counter_name = 'CORTE_TOTAL'
                  AND NEW.detected_at >= counter_epoch;

                GET DIAGNOSTICS affected_rows = ROW_COUNT;
                IF affected_rows <> 1 THEN
                    RAISE EXCEPTION
                        'CORTE_TOTAL counter state changed during increment';
                END IF;

                RETURN NEW;
            END;
            $body$ LANGUAGE plpgsql
            $function$,
            target.schema_name,
            target.schema_name,
            target.schema_name
        );
        EXECUTE format(
            'DROP TRIGGER IF EXISTS trg_increment_vision_counter
             ON %I.vision_detections',
            target.schema_name
        );
        EXECUTE format(
            'CREATE TRIGGER trg_increment_vision_counter
             AFTER INSERT ON %I.vision_detections
             FOR EACH ROW
             EXECUTE FUNCTION %I.increment_vision_counter()',
            target.schema_name,
            target.schema_name
        );

        -- A tolerant converter keeps one malformed historical JSON value from
        -- rolling back the migration and from breaking future CORTE inserts.
        EXECUTE format(
            $function$
            CREATE OR REPLACE FUNCTION %I.vision_try_real(value TEXT)
            RETURNS REAL AS $body$
            BEGIN
                IF value IS NULL
                   OR value !~ '^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)([eE][+-]?[0-9]+)?$'
                THEN
                    RETURN NULL;
                END IF;

                RETURN value::REAL;
            EXCEPTION
                WHEN invalid_text_representation OR numeric_value_out_of_range
                THEN RETURN NULL;
            END;
            $body$ LANGUAGE plpgsql IMMUTABLE
            $function$,
            target.schema_name
        );

        -- Installations have used schema-derived trigger names, and a copied
        -- trigger can retain the name of a different line. Discover triggers
        -- by the function they execute so every duplicate/orphan is removed,
        -- regardless of its name or the schema containing the old function.
        FOR extraction_trigger IN
            SELECT
                tg.tgname AS trigger_name,
                function_ns.nspname AS function_schema
            FROM pg_catalog.pg_trigger tg
            JOIN pg_catalog.pg_class source_table
              ON source_table.oid = tg.tgrelid
            JOIN pg_catalog.pg_namespace source_ns
              ON source_ns.oid = source_table.relnamespace
            JOIN pg_catalog.pg_proc trigger_function
              ON trigger_function.oid = tg.tgfoid
            JOIN pg_catalog.pg_namespace function_ns
              ON function_ns.oid = trigger_function.pronamespace
            WHERE NOT tg.tgisinternal
              AND source_ns.nspname = target.schema_name
              AND source_table.relname = 'events_buffer'
              AND trigger_function.proname = 'extract_vision_detection'
            ORDER BY tg.tgname
        LOOP
            RAISE NOTICE
                'Dropping %.% (function %.extract_vision_detection)',
                target.schema_name,
                extraction_trigger.trigger_name,
                extraction_trigger.function_schema;
            EXECUTE format(
                'DROP TRIGGER %I ON %I.events_buffer',
                extraction_trigger.trigger_name,
                target.schema_name
            );
        END LOOP;

        -- Also reserve the canonical name if a malformed installation used it
        -- for a different function.
        EXECUTE format(
            'DROP TRIGGER IF EXISTS trg_extract_vision_detection
             ON %I.events_buffer',
            target.schema_name
        );

        EXECUTE format(
            $function$
            CREATE OR REPLACE FUNCTION %I.extract_vision_detection()
            RETURNS TRIGGER AS $body$
            BEGIN
                IF NEW.event_type <> 'CORTE' THEN
                    RETURN NEW;
                END IF;

                INSERT INTO %I.vision_detections (
                    detection_id,
                    device_id,
                    detected_at,
                    line_code,
                    confidence,
                    signal_edge,
                    signal_color,
                    signal_flow,
                    signal_beige,
                    roi_id,
                    fsm_state
                )
                VALUES (
                    NEW.event_id,
                    to_jsonb(NEW) ->> 'device_id',
                    NEW.timestamp,
                    NEW.payload ->> 'line_code',
                    %I.vision_try_real(NEW.payload ->> 'confidence'),
                    %I.vision_try_real(NEW.payload -> 'signals' ->> 'edge'),
                    %I.vision_try_real(NEW.payload -> 'signals' ->> 'color'),
                    %I.vision_try_real(NEW.payload -> 'signals' ->> 'flow'),
                    %I.vision_try_real(NEW.payload -> 'signals' ->> 'beige'),
                    NEW.payload ->> 'roi_id',
                    NEW.payload ->> 'fsm_state'
                )
                ON CONFLICT (detection_id) DO NOTHING;

                RETURN NEW;
            END;
            $body$ LANGUAGE plpgsql
            $function$,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name
        );

        EXECUTE format(
            'CREATE TRIGGER trg_extract_vision_detection
             AFTER INSERT ON %I.events_buffer
             FOR EACH ROW
             EXECUTE FUNCTION %I.extract_vision_detection()',
            target.schema_name,
            target.schema_name
        );

        -- Fail atomically instead of leaving an installation that can extract
        -- one CORTE more than once. The sole extractor must be the enabled,
        -- row-level AFTER INSERT trigger backed by this line's function.
        SELECT
            COUNT(*),
            COUNT(*) FILTER (
                WHERE tg.tgname = 'trg_extract_vision_detection'
                  AND function_ns.nspname = target.schema_name
                  AND tg.tgenabled <> 'D'
                  AND tg.tgtype = 5
            )
        INTO extraction_trigger_count, canonical_trigger_count
        FROM pg_catalog.pg_trigger tg
        JOIN pg_catalog.pg_class source_table
          ON source_table.oid = tg.tgrelid
        JOIN pg_catalog.pg_namespace source_ns
          ON source_ns.oid = source_table.relnamespace
        JOIN pg_catalog.pg_proc trigger_function
          ON trigger_function.oid = tg.tgfoid
        JOIN pg_catalog.pg_namespace function_ns
          ON function_ns.oid = trigger_function.pronamespace
        WHERE NOT tg.tgisinternal
          AND source_ns.nspname = target.schema_name
          AND source_table.relname = 'events_buffer'
          AND trigger_function.proname = 'extract_vision_detection';

        IF extraction_trigger_count <> 1 OR canonical_trigger_count <> 1 THEN
            RAISE EXCEPTION
                'Expected one canonical extractor on %.events_buffer; found % extractor(s), % valid canonical',
                target.schema_name,
                extraction_trigger_count,
                canonical_trigger_count;
        END IF;

        -- Fill metadata added to linea_template-style tables before inserting
        -- any missing historical detections.
        EXECUTE format(
            'UPDATE %I.vision_detections vd
             SET device_id = COALESCE(
                     vd.device_id,
                     to_jsonb(eb) ->> ''device_id''
                 ),
                 fsm_state = COALESCE(vd.fsm_state, eb.payload ->> ''fsm_state'')
             FROM %I.events_buffer eb
             WHERE vd.detection_id = eb.event_id
               AND (vd.device_id IS NULL OR vd.fsm_state IS NULL)',
            target.schema_name,
            target.schema_name
        );

        EXECUTE format(
            $backfill$
            INSERT INTO %I.vision_detections (
                detection_id,
                device_id,
                detected_at,
                line_code,
                confidence,
                signal_edge,
                signal_color,
                signal_flow,
                signal_beige,
                roi_id,
                fsm_state
            )
            SELECT
                eb.event_id,
                to_jsonb(eb) ->> 'device_id',
                eb.timestamp,
                eb.payload ->> 'line_code',
                %I.vision_try_real(eb.payload ->> 'confidence'),
                %I.vision_try_real(eb.payload -> 'signals' ->> 'edge'),
                %I.vision_try_real(eb.payload -> 'signals' ->> 'color'),
                %I.vision_try_real(eb.payload -> 'signals' ->> 'flow'),
                %I.vision_try_real(eb.payload -> 'signals' ->> 'beige'),
                eb.payload ->> 'roi_id',
                eb.payload ->> 'fsm_state'
            FROM %I.events_buffer eb
            WHERE eb.event_type = 'CORTE'
            ON CONFLICT (detection_id) DO NOTHING
            $backfill$,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name,
            target.schema_name
        );

        -- Reconcile after backfill. This repairs any historical trigger gap
        -- without changing the stable counter epoch.
        EXECUTE format(
            'UPDATE %I.vision_counter_state state
             SET counter_value = state.counter_baseline + (
                     SELECT COUNT(*)
                     FROM %I.vision_detections detections
                     WHERE detections.detected_at >= state.counter_epoch
                 ),
                 updated_at = CURRENT_TIMESTAMP
             WHERE state.counter_name = ''CORTE_TOTAL''',
            target.schema_name,
            target.schema_name
        );

        -- Once a detection belongs to the active counter epoch, deleting it
        -- or changing its identity/timestamp would break the state invariant.
        -- Other metadata columns remain editable.
        EXECUTE format(
            $function$
            CREATE OR REPLACE FUNCTION %I.protect_vision_counter_history()
            RETURNS TRIGGER AS $body$
            DECLARE
                active_epoch TIMESTAMPTZ;
            BEGIN
                SELECT counter_epoch
                INTO active_epoch
                FROM %I.vision_counter_state
                WHERE counter_name = 'CORTE_TOTAL';

                IF NOT FOUND THEN
                    RAISE EXCEPTION 'CORTE_TOTAL counter state is missing';
                END IF;

                IF TG_OP = 'DELETE' THEN
                    IF OLD.detected_at >= active_epoch THEN
                        RAISE EXCEPTION
                            'active vision counter history is append-only';
                    END IF;
                    RETURN OLD;
                END IF;

                IF (
                    OLD.detection_id IS DISTINCT FROM NEW.detection_id
                    OR OLD.detected_at IS DISTINCT FROM NEW.detected_at
                ) AND (
                    OLD.detected_at >= active_epoch
                    OR NEW.detected_at >= active_epoch
                ) THEN
                    RAISE EXCEPTION
                        'active vision counter identity and timestamp are immutable';
                END IF;

                RETURN NEW;
            END;
            $body$ LANGUAGE plpgsql
            $function$,
            target.schema_name,
            target.schema_name
        );
        EXECUTE format(
            'DROP TRIGGER IF EXISTS trg_protect_vision_counter_history
             ON %I.vision_detections',
            target.schema_name
        );
        EXECUTE format(
            'CREATE TRIGGER trg_protect_vision_counter_history
             BEFORE UPDATE OF detection_id, detected_at OR DELETE
             ON %I.vision_detections
             FOR EACH ROW
             EXECUTE FUNCTION %I.protect_vision_counter_history()',
            target.schema_name,
            target.schema_name
        );
    END LOOP;

    IF migrated_schemas = 0 THEN
        RAISE EXCEPTION
            'No linea_<number> schema with events_buffer was found';
    END IF;
END;
$migration$;
