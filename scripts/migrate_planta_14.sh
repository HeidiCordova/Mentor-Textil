#!/usr/bin/env bash
set -euo pipefail

# Migration script: Art Atlas (planta_id=14) -> mentor_planta_14 / linea_11
# Crea la BD dedicada, el schema linea_11, y migra datos desde mentor_master
# en lotes de 1000 filas para minimizar bloqueo.

MASTER_DB="${DATABASE_URL:-postgres://mentor:mentor@localhost:5432/mentor_master}"
PLANTA_ID=14
LINEA_ID=11
NEW_DB="mentor_planta_${PLANTA_ID}"
SCHEMA="linea_${LINEA_ID}"
BATCH_SIZE=1000

log() { echo "[$(date '+%H:%M:%S')] $*"; }

log "Creando BD $NEW_DB si no existe..."
psql "$MASTER_DB" -tc "SELECT 1 FROM pg_database WHERE datname='$NEW_DB'" | grep -q 1 || \
  psql "$MASTER_DB" -c "CREATE DATABASE $NEW_DB OWNER mentor"

NEW_DB_URL="${MASTER_DB%/*}/$NEW_DB"

log "Aplicando template de linea ($SCHEMA)..."
TEMPLATE_FILE="$(dirname "$0")/../mentor-cloud/infrastructure/database/12_linea_template.sql"
if [ ! -f "$TEMPLATE_FILE" ]; then
  echo "ERROR: No se encuentra $TEMPLATE_FILE" && exit 1
fi
sed "s/{schema}/$SCHEMA/g" "$TEMPLATE_FILE" | psql "$NEW_DB_URL"

log "Registrando en admin.planta_databases..."
psql "$MASTER_DB" -c "
INSERT INTO admin.planta_databases (planta_id, db_name, db_host, db_port, db_user, db_password_enc, instance_type, schemas, is_active)
VALUES ($PLANTA_ID, '$NEW_DB', 'localhost', 5432, 'mentor', '', 'vps', '{\"$SCHEMA\"}', true)
ON CONFLICT (planta_id) DO UPDATE SET schemas = EXCLUDED.schemas, is_active = true;
"

migrate_table() {
  local src_schema="$1" src_table="$2" dst_table="$3" col_list="$4"
  local total offset=0

  total=$(psql "$MASTER_DB" -tAc "SELECT COUNT(*) FROM ${src_schema}.${src_table} WHERE planta_id=$PLANTA_ID" 2>/dev/null || echo "0")
  if [ "$total" = "0" ]; then
    log "  $src_table: 0 filas, omitido"
    return
  fi

  log "  $src_table -> $dst_table: $total filas"

  while [ "$offset" -lt "$total" ]; do
    psql "$MASTER_DB" -tAc "
      COPY (
        SELECT $col_list FROM ${src_schema}.${src_table}
        WHERE planta_id=$PLANTA_ID
        ORDER BY id
        OFFSET $offset LIMIT $BATCH_SIZE
      ) TO STDOUT WITH CSV
    " | psql "$NEW_DB_URL" -c "COPY ${SCHEMA}.${dst_table}($col_list) FROM STDIN WITH CSV"
    offset=$((offset + BATCH_SIZE))
  done
}

migrate_table_nofilter() {
  local src_schema="$1" src_table="$2" dst_table="$3" col_list="$4" where_clause="$5"
  local total offset=0

  total=$(psql "$MASTER_DB" -tAc "SELECT COUNT(*) FROM ${src_schema}.${src_table} WHERE $where_clause" 2>/dev/null || echo "0")
  if [ "$total" = "0" ]; then
    log "  $src_table: 0 filas, omitido"
    return
  fi

  log "  $src_table -> $dst_table: $total filas"

  while [ "$offset" -lt "$total" ]; do
    psql "$MASTER_DB" -tAc "
      COPY (
        SELECT $col_list FROM ${src_schema}.${src_table}
        WHERE $where_clause
        ORDER BY id
        OFFSET $offset LIMIT $BATCH_SIZE
      ) TO STDOUT WITH CSV
    " | psql "$NEW_DB_URL" -c "COPY ${SCHEMA}.${dst_table}($col_list) FROM STDIN WITH CSV"
    offset=$((offset + BATCH_SIZE))
  done
}

log "Migrando datos de ingesta..."
migrate_table "ingest" "raw_events" "raw_events" \
  "device_id,event_type,payload,timestamp_edge,recibido_en,procesado"

migrate_table "ingest" "oee_snapshots" "oee_snapshots" \
  "device_id,turno,fecha,hora,disponibilidad,rendimiento,calidad,oee,produccion,energia_kwh,code,interval_s,head,data,creado_en"

log "Migrando datos analiticos..."
migrate_table "analytics" "paradas" "paradas" \
  "device_id,categoria_id,categoria_nombre,subcategoria_nombre,subcategoria_2_nombre,descripcion,inicio,fin,duracion_min,creado_en"

migrate_table "analytics" "alarmas" "alarmas" \
  "device_id,tipo,mensaje,severidad,activa,timestamp_evento,reconocido_en,creado_en"

log "Migrando production_runs..."
migrate_table "analytics" "production_runs" "production_runs" \
  "run_id,device_id,producto_id,sku,nombre,started_at,ended_at,creado_en,actualizado_en"

log "Refrescando vista materializada..."
psql "$NEW_DB_URL" -c "REFRESH MATERIALIZED VIEW ${SCHEMA}.mv_produccion_mensual" 2>/dev/null || true

log "Verificando conteos..."
for tbl in raw_events oee_snapshots paradas alarmas production_runs; do
  src_count=$(psql "$MASTER_DB" -tAc "
    SELECT COUNT(*) FROM (
      SELECT 1 FROM ingest.${tbl} WHERE planta_id=$PLANTA_ID
      UNION ALL
      SELECT 1 FROM analytics.${tbl} WHERE planta_id=$PLANTA_ID
    ) x" 2>/dev/null || echo "N/A")
  dst_count=$(psql "$NEW_DB_URL" -tAc "SELECT COUNT(*) FROM ${SCHEMA}.${tbl}" 2>/dev/null || echo "N/A")
  log "  $tbl: master=$src_count, nuevo=$dst_count"
done

log "Migracion de planta $PLANTA_ID completada."
