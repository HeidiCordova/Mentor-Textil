#!/bin/bash
# 02_seed.sh — Seed inicial de line_config con defaults.
# Valores de negocio (camera, oee, cloud, linea_id, empresa_id)
# se configuran despues via UI (http://<jetson>:8080).

set -e

DEVICE_ID="${DEVICE_ID:-1}"

echo "[seed] Insertando configuracion inicial para device_id='$DEVICE_ID'"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
     -c "INSERT INTO line_config (device_id, roi, thresholds, fsm, mode, camera, oee, cloud)
         VALUES (
             \$\$${DEVICE_ID}\$\$,
             '{\"x\": 120, \"y\": 60, \"width\": 320, \"height\": 200}',
             '{\"edge\": 0.4, \"color\": 0.6, \"flow\": 0.5, \"dy\": 5, \"beige\": 0.35, \"high\": 0.7, \"low\": 0.3}',
             '{\"n_frames\": 3, \"cooldown\": 8, \"exit_frames\": 5, \"max_wait_exit_frames\": 50}',
             'textil',
             '{\"url\": \"\", \"fps\": 25}',
             '{\"line_name\": \"\", \"micro_stop_max_s\": 120, \"stop_max_s\": 86400, \"snapshot_interval_s\": 1800, \"vel_unit\": \"uh\", \"vel_nominal_us\": 0.008333333}',
             '{\"sync_interval_s\": 300}'
         )
         ON CONFLICT (device_id) DO NOTHING;"

echo "[seed] line_config OK para '$DEVICE_ID'"
