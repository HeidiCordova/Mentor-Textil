#!/usr/bin/env bash
# =============================================================
# init_line.sh — Crea el schema de una línea en mentor_edge
# Uso: ./init_line.sh <linea_id>
# Ejemplo: ./init_line.sh 3   → crea schema linea_3
# =============================================================
set -euo pipefail

LINEA_ID="${1:?Falta el id de la línea (ej: 3)}"
SCHEMA="linea_${LINEA_ID}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Variables de conexión (del .env o del entorno)
DB_HOST="${DB_HOST:-postgres-local}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-mentor}"
DB_PASSWORD="${DB_PASSWORD:-mentor123456}"
DB_NAME="${DB_NAME:-mentor_edge}"

export PGPASSWORD="$DB_PASSWORD"

echo "[init_line] Creando schema '${SCHEMA}' en ${DB_HOST}:${DB_PORT}/${DB_NAME}..."

sed "s/{schema}/${SCHEMA}/g" "${SCRIPT_DIR}/linea_template.sql" \
  | psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1

echo "[init_line] Schema '${SCHEMA}' creado correctamente ✓"
