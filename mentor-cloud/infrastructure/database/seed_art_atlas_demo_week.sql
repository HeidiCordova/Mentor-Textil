-- ============================================================
-- Seed: OEE sintético para demo — Art Atlas S.A.
-- Base de datos destino : mentor_planta_14
-- Período cubierto      : 2026-04-08 08:05 a 2026-04-11 11:00 (Lima, UTC-5)
-- Ventana de snapshot   : 1800 s (30 min) — pasa filtro >= 300 del dashboard
-- Líneas                : linea_11 (Maquina01) | linea_12 (linea3)
--                         linea_13 (linea4)    | linea_14 (linea1)
-- Productos textiles    : TXT-01 a TXT-05
--
-- VERIFICAR device_id antes de ejecutar (en mentor_cloud):
--   SELECT device_id, linea_id FROM config.dispositivos WHERE planta_id=14;
-- Si difiere de '1', cambiar v_device_id abajo.
--
-- Ejecución:
--   psql "host=152.53.253.59 port=5434 dbname=mentor_planta_14 user=<user>" \
--        -f seed_art_atlas_demo_week.sql
--
-- Idempotente: ON CONFLICT DO NOTHING en oee_snapshots (device_id, hora)
--              y run_id determinístico via md5 en production_runs.
-- ============================================================

DO $$
DECLARE
    v_empresa_id  INT  := 8;
    v_planta_id   INT  := 14;
    v_device_id   TEXT := '1';
    v_interval    INT  := 1800;
    v_start       TIMESTAMPTZ := '2026-04-08 08:05:00-05:00';
    v_end         TIMESTAMPTZ := '2026-04-11 11:00:00-05:00';

    schemas      TEXT[]    := ARRAY['linea_11','linea_12','linea_13','linea_14'];
    linea_ids    INT[]     := ARRAY[11, 12, 13, 14];
    phases       FLOAT[]   := ARRAY[0.0, 1.05, 2.09, 3.14];
    -- Producto por línea (TXT-05 queda en catálogo, sin run activo)
    skus         TEXT[]    := ARRAY['TXT-01','TXT-02','TXT-03','TXT-04'];
    run_nombres  TEXT[]    := ARRAY['Manga','Cuerpo de prenda','Cuello','Puño'];

    i    INT;
    sch  TEXT;
    lid  INT;
    ph   FLOAT;
    sku  TEXT;
    rnom TEXT;
BEGIN
    -- Catálogo de productos textiles en cada schema de línea
    FOR i IN 1..4 LOOP
        sch := schemas[i];
        EXECUTE format('
            INSERT INTO %I.productos (codigo, nombre, activo) VALUES
                (''TXT-01'', ''Manga'',            true),
                (''TXT-02'', ''Cuerpo de prenda'', true),
                (''TXT-03'', ''Cuello'',           true),
                (''TXT-04'', ''Puño'',             true),
                (''TXT-05'', ''Pieza completa'',   true)
            ON CONFLICT (codigo) DO NOTHING', sch);
    END LOOP;

    -- Production run por línea — run_id determinístico para idempotencia
    FOR i IN 1..4 LOOP
        sch  := schemas[i];
        lid  := linea_ids[i];
        sku  := skus[i];
        rnom := run_nombres[i];

        EXECUTE format('
            INSERT INTO %I.production_runs
                (run_id, device_id, linea_id, planta_id, empresa_id, sku, nombre,
                 started_at, ended_at)
            VALUES
                (md5(''artAtlas-demo-'' || %L)::uuid,
                 $1, $2, $3, $4, $5, $6, $7, $8)
            ON CONFLICT (device_id, run_id) DO NOTHING',
            sch, sch)
        USING v_device_id, lid, v_planta_id, v_empresa_id,
              sku, rnom,
              v_start, v_end + INTERVAL '30 minutes';
    END LOOP;

    -- OEE snapshots + raw_events via generate_series
    FOR i IN 1..4 LOOP
        sch := schemas[i];
        lid := linea_ids[i];
        ph  := phases[i];

        -- oee_snapshots
        EXECUTE format($sql$
            INSERT INTO %I.oee_snapshots
                (device_id, linea_id, planta_id, empresa_id,
                 turno, fecha, hora,
                 disponibilidad, rendimiento, calidad, oee,
                 produccion, energia_kwh, interval_s)
            SELECT
                $1, $2, $3, $4,
                CASE
                    WHEN EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') >= 6
                     AND EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') < 14 THEN 'MANANA'
                    WHEN EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') >= 14
                     AND EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') < 22 THEN 'TARDE'
                    ELSE 'NOCHE'
                END,
                (ts AT TIME ZONE 'America/Lima')::date,
                ts,
                CASE
                    WHEN EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') IN (11,19)
                     AND EXTRACT(MINUTE FROM ts AT TIME ZONE 'America/Lima') < 30
                        THEN GREATEST(10.0,
                                22.0 + 5.0 * sin(EXTRACT(EPOCH FROM ts)::float / 900.0))
                    ELSE
                        GREATEST(62.0, LEAST(97.0,
                            85.0
                            + 8.0 * sin(EXTRACT(EPOCH FROM ts)::float / 7200.0  + $5)
                            + 4.0 * sin(EXTRACT(EPOCH FROM ts)::float / 28800.0 + $5 * 0.5)
                        ))
                END,
                GREATEST(55.0, LEAST(95.0,
                    80.0
                    + 7.0 * sin(EXTRACT(EPOCH FROM ts)::float / 10800.0 + $5 + 1.57)
                    + 3.0 * sin(EXTRACT(EPOCH FROM ts)::float / 43200.0)
                )),
                GREATEST(93.0, LEAST(99.5,
                    97.0 + 1.5 * sin(EXTRACT(EPOCH FROM ts)::float / 21600.0 + $5 * 2.0)
                )),
                ROUND((
                    GREATEST(62.0, LEAST(97.0,
                        85.0 + 8.0 * sin(EXTRACT(EPOCH FROM ts)::float / 7200.0 + $5)
                        + 4.0 * sin(EXTRACT(EPOCH FROM ts)::float / 28800.0 + $5 * 0.5)
                    )) *
                    GREATEST(55.0, LEAST(95.0,
                        80.0 + 7.0 * sin(EXTRACT(EPOCH FROM ts)::float / 10800.0 + $5 + 1.57)
                        + 3.0 * sin(EXTRACT(EPOCH FROM ts)::float / 43200.0)
                    )) *
                    GREATEST(93.0, LEAST(99.5,
                        97.0 + 1.5 * sin(EXTRACT(EPOCH FROM ts)::float / 21600.0 + $5 * 2.0)
                    )) / 10000.0
                )::numeric, 2),
                (GREATEST(62.0, LEAST(97.0,
                    85.0 + 8.0 * sin(EXTRACT(EPOCH FROM ts)::float / 7200.0 + $5)
                )) *
                GREATEST(55.0, LEAST(95.0,
                    80.0 + 7.0 * sin(EXTRACT(EPOCH FROM ts)::float / 10800.0 + $5 + 1.57)
                )) / 100.0 * 0.40)::int,
                ROUND(((GREATEST(62.0, LEAST(97.0,
                    85.0 + 8.0 * sin(EXTRACT(EPOCH FROM ts)::float / 7200.0 + $5)
                )) *
                GREATEST(55.0, LEAST(95.0,
                    80.0 + 7.0 * sin(EXTRACT(EPOCH FROM ts)::float / 10800.0 + $5 + 1.57)
                )) / 100.0 * 0.40) * 0.15)::numeric, 3),
                $6
            FROM generate_series($7::timestamptz, $8::timestamptz, '1800 seconds'::interval) AS ts
            ON CONFLICT (device_id, hora) DO NOTHING
        $sql$, sch)
        USING v_device_id, lid, v_planta_id, v_empresa_id, ph, v_interval, v_start, v_end;

        -- raw_events (event_type='oee') — leídos por "Datos Recibidos" en la UI cloud
        -- payload sigue la estructura de OEERecord: head[] + data[] + campos auxiliares
        EXECUTE format($raw$
            INSERT INTO %I.raw_events
                (device_id, empresa_id, planta_id, linea_id,
                 event_type, payload, timestamp_edge)
            SELECT
                $1, $2, $3, $4,
                'oee',
                jsonb_build_object(
                    'interval_s', $5,
                    'turno', CASE
                        WHEN EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') >= 6
                         AND EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') < 14 THEN 'MANANA'
                        WHEN EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') >= 14
                         AND EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') < 22 THEN 'TARDE'
                        ELSE 'NOCHE'
                    END,
                    'head', jsonb_build_array(
                        'CONTEO_1','CONTEO_2','T_DISPONIBLE',
                        'T_MICROPARADA','T_PARADA_NO_ASIGNADA',
                        'MARCA','SABOR','TAMANIO','MATERIAL','DESTINO',
                        'T_REFRIGERIO','T_CAPACITACION_OBLIGATORIA',
                        'T_MANTENIMIENTO_PLANIFICADO',
                        'T_PARADA_PROGRAMADA','T_PARADA_NO_PROGRAMADA',
                        'TIPO_PARADA_PROGRAMADA','TIPO_PARADA_NO_PROGRAMADA','MERMA'
                    ),
                    'data', jsonb_build_array(
                        ((GREATEST(62.0, LEAST(97.0,
                            85.0 + 8.0 * sin(EXTRACT(EPOCH FROM ts)::float / 7200.0 + $6)
                        )) *
                        GREATEST(55.0, LEAST(95.0,
                            80.0 + 7.0 * sin(EXTRACT(EPOCH FROM ts)::float / 10800.0 + $6 + 1.57)
                        )) / 100.0 * 0.40)::int)::text,
                        ((GREATEST(62.0, LEAST(97.0,
                            85.0 + 8.0 * sin(EXTRACT(EPOCH FROM ts)::float / 7200.0 + $6)
                        )) *
                        GREATEST(55.0, LEAST(95.0,
                            80.0 + 7.0 * sin(EXTRACT(EPOCH FROM ts)::float / 10800.0 + $6 + 1.57)
                        )) / 100.0 * 0.38)::int)::text,
                        $5::text,
                        ROUND((1800 * 0.03 * (1 + 0.5 * sin(EXTRACT(EPOCH FROM ts)::float / 3600.0)))::numeric)::text,
                        '0',
                        'ArtAtlas', 'N/A', 'Std', 'Tela', 'Lima',
                        CASE
                            WHEN EXTRACT(HOUR FROM ts AT TIME ZONE 'America/Lima') IN (11,19)
                             AND EXTRACT(MINUTE FROM ts AT TIME ZONE 'America/Lima') < 30
                            THEN '1800' ELSE '0'
                        END,
                        '0', '0',
                        '0', '0',
                        '', '',
                        '0'
                    )
                ),
                ts
            FROM generate_series($7::timestamptz, $8::timestamptz, '1800 seconds'::interval) AS ts
            ON CONFLICT (device_id, timestamp_edge) DO NOTHING
        $raw$, sch)
        USING v_device_id, v_empresa_id, v_planta_id, lid,
              v_interval, ph, v_start, v_end;
    END LOOP;

    RAISE NOTICE 'Seed OEE completado: 2026-04-08 08:05 — 2026-04-11 11:00 (Lima)';
END;
$$;

-- Índice único necesario para ON CONFLICT (creado si no existe en el tenant)
CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora ON linea_11.oee_snapshots(device_id, hora);
CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora ON linea_12.oee_snapshots(device_id, hora);
CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora ON linea_13.oee_snapshots(device_id, hora);
CREATE UNIQUE INDEX IF NOT EXISTS uq_oee_snapshots_device_hora ON linea_14.oee_snapshots(device_id, hora);

REFRESH MATERIALIZED VIEW CONCURRENTLY linea_11.mv_produccion_mensual;
REFRESH MATERIALIZED VIEW CONCURRENTLY linea_12.mv_produccion_mensual;
REFRESH MATERIALIZED VIEW CONCURRENTLY linea_13.mv_produccion_mensual;
REFRESH MATERIALIZED VIEW CONCURRENTLY linea_14.mv_produccion_mensual;
