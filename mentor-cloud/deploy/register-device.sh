#!/bin/bash
# register-device.sh — Registra un nuevo Jetson en la BD cloud
# Uso: ./register-device.sh
# Requiere: psql y DATABASE_URL exportado (o ajusta las vars abajo).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="$SCRIPT_DIR/../infrastructure/database/register_device.sql"

# ── Parámetros (editar por dispositivo) ─────────────────────────
DEVICE_ID="${DEVICE_ID:-}"
DEVICE_NOMBRE="${DEVICE_NOMBRE:-}"
EMPRESA_ID="${EMPRESA_ID:-0}"
PLANTA_ID="${PLANTA_ID:-0}"
LINEA_ID="${LINEA_ID:-0}"
DATABASE_URL="${DATABASE_URL:-}"

# ── Validaciones ─────────────────────────────────────────────────
if [[ -z "$DEVICE_ID" ]]; then
    read -rp "device_id (ej: jetson-orin-planta01-l1): " DEVICE_ID
fi
if [[ -z "$DEVICE_NOMBRE" ]]; then
    read -rp "nombre del dispositivo: " DEVICE_NOMBRE
fi
if [[ "$EMPRESA_ID" == "0" ]]; then
    read -rp "empresa_id (número): " EMPRESA_ID
fi
if [[ "$PLANTA_ID" == "0" ]]; then
    read -rp "planta_id (número): " PLANTA_ID
fi
if [[ "$LINEA_ID" == "0" ]]; then
    read -rp "linea_id (número): " LINEA_ID
fi
if [[ -z "$DATABASE_URL" ]]; then
    read -rp "DATABASE_URL (postgresql://user:pass@host:port/db): " DATABASE_URL
fi

# Generar API key única (32 bytes hex)
API_KEY=$(openssl rand -hex 32)

echo ""
echo "──────────────────────────────────────────"
echo "  Registrando dispositivo en cloud DB"
echo "──────────────────────────────────────────"
echo "  device_id    : $DEVICE_ID"
echo "  nombre        : $DEVICE_NOMBRE"
echo "  empresa_id    : $EMPRESA_ID"
echo "  planta_id     : $PLANTA_ID"
echo "  linea_id      : $LINEA_ID"
echo "  api_key       : ${API_KEY:0:8}...  (completa guardada abajo)"
echo "──────────────────────────────────────────"
read -rp "¿Confirmar? (s/N): " CONFIRM
if [[ "${CONFIRM,,}" != "s" ]]; then
    echo "Cancelado."
    exit 0
fi

psql "$DATABASE_URL" \
    -v "device_id='$DEVICE_ID'" \
    -v "device_nombre='$DEVICE_NOMBRE'" \
    -v "empresa_id=$EMPRESA_ID" \
    -v "planta_id=$PLANTA_ID" \
    -v "linea_id=$LINEA_ID" \
    -v "api_key='$API_KEY'" \
    -f "$SQL_FILE"

echo ""
echo "✓ Dispositivo registrado correctamente."
echo ""
echo "  ┌─────────────────────────────────────────────────────────────────┐"
echo "  │  Copia estos valores en el .env del Jetson:                     │"
echo "  │                                                                  │"
echo "  │  DEVICE_ID=$DEVICE_ID"
echo "  │  EMPRESA_ID=$EMPRESA_ID"
echo "  │  LINEA_ID=$LINEA_ID"
echo "  │  CLOUD_API_KEY=$API_KEY"
echo "  └─────────────────────────────────────────────────────────────────┘"
