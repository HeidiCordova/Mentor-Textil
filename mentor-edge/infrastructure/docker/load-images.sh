#!/bin/sh
# load-images.sh — Carga imágenes Docker pre-compiladas para el stack de energía
# Usar cuando el RPi no tiene acceso a internet.
#
# Imágenes esperadas en /tmp/mentor-images.tar.gz:
#   postgres-arm64.tar         → postgres:15-alpine
#   energy-sender-arm64.tar    → mentor-energy-sender:arm64
#   ui-energy-arm64.tar        → mentor-ui-energy:arm64
#
# Nota: mentor-mc60-reader se construye desde código fuente con 'docker build'.
#
# Uso: sh load-images.sh
set -e

IMAGES_TAR="/tmp/mentor-images.tar.gz"

echo "[1/4] Verificando Docker..."
docker version > /dev/null || { echo "ERROR: Docker no disponible"; exit 1; }

echo "[2/4] Extrayendo archivos en /tmp..."
cd /tmp
tar -xzf "$IMAGES_TAR"
echo "  Archivos: $(ls /tmp/*.tar 2>/dev/null | tr '\n' ' ')"

echo "[3/4] Cargando imágenes..."
[ -f /tmp/postgres-arm64.tar ]      && docker load < /tmp/postgres-arm64.tar
[ -f /tmp/energy-sender-arm64.tar ] && docker load < /tmp/energy-sender-arm64.tar
[ -f /tmp/ui-energy-arm64.tar ]     && docker load < /tmp/ui-energy-arm64.tar

echo "[4/4] Imágenes disponibles:"
docker images | grep -E "postgres|energy-sender|ui-energy"

echo ""
echo "Siguiente paso — crear .env y lanzar el stack:"
echo "  cp .env.rpi .env.raspberry-energy"
echo "  docker compose -f docker-compose.raspberry-energy.yml --env-file .env.raspberry-energy up -d"
