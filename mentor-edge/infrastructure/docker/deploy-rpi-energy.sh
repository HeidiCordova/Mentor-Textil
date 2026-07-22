#!/usr/bin/env bash
# deploy-rpi-energy.sh
# Despliega el stack de energia en la Raspberry Pi.
#
# Estrategia (sin buildx ni QEMU):
#   1. Compila el binario Go (energy-sender) para ARM64 en el host.
#   2. Construye el front-end (npm run build) en el host.
#   3. Transfiere todos los artefactos a la RPi via SCP.
#   4. Construye las imagenes Docker en la RPi (nativo aarch64).
#   5. Lanza el stack con docker compose.
#
# Uso:
#   chmod +x deploy-rpi-energy.sh
#   ./deploy-rpi-energy.sh
#
# Variables opcionales de entorno:
#   RPI_HOST       IP o hostname (default: 192.168.100.30)
#   RPI_USER       usuario SSH (default: py)
#   RPI_PASS       contrasena SSH (default: 12345678)
#   RPI_DEPLOY_DIR directorio destino en la RPi (default: /home/py/mentor-energy)
#   SKIP_BUILD     si = 1 no compila Go ni npm (reutiliza artefactos previos)

set -euo pipefail

RPI_HOST="${RPI_HOST:-192.168.100.30}"
RPI_USER="${RPI_USER:-py}"
RPI_PASS="${RPI_PASS:-12345678}"
RPI_DEPLOY_DIR="${RPI_DEPLOY_DIR:-/home/py/mentor-energy}"
SKIP_BUILD="${SKIP_BUILD:-0}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

SVC="$PROJECT_ROOT/services"
INFRA_DB="$PROJECT_ROOT/infrastructure/database"

SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
SCP_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"

ssh_cmd()  { sshpass -p "$RPI_PASS" ssh  $SSH_OPTS "${RPI_USER}@${RPI_HOST}" "$@"; }
scp_file() { sshpass -p "$RPI_PASS" scp  $SCP_OPTS "$1" "${RPI_USER}@${RPI_HOST}:$2"; }
scp_dir()  { sshpass -p "$RPI_PASS" scp  $SCP_OPTS -r "$1" "${RPI_USER}@${RPI_HOST}:$2"; }

log()  { echo "[deploy] $*"; }
fail() { echo "[ERROR] $*" >&2; exit 1; }

# ---------------------------------------------------------------------------- #
log "Verificando herramientas locales..."
command -v sshpass >/dev/null || fail "sshpass no encontrado. Instalar: sudo apt install sshpass"
command -v go      >/dev/null || fail "go no encontrado"
command -v npm     >/dev/null || fail "npm no encontrado"

# ---------------------------------------------------------------------------- #
log "Verificando conexion SSH con la Raspberry Pi (${RPI_HOST})..."
ssh_cmd "echo ok" >/dev/null || fail "No se pudo conectar a ${RPI_USER}@${RPI_HOST}"
log "Conexion OK"

# ---------------------------------------------------------------------------- #
if [ "$SKIP_BUILD" != "1" ]; then
    log "Compilando energy-sender para ARM64..."
    cd "$SVC/energy-sender"
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
        go build -ldflags="-s -w" -o energy-sender-arm64 .
    log "  Binario: $SVC/energy-sender/energy-sender-arm64"

    log "Construyendo front-end ui-energy..."
    cd "$SVC/ui-energy"
    npm ci --silent
    npm run build --silent
    log "  Dist: $SVC/ui-energy/dist"
else
    log "SKIP_BUILD=1 — omitiendo compilacion local"
    [ -f "$SVC/energy-sender/energy-sender-arm64" ] || \
        fail "energy-sender-arm64 no existe. Ejecutar sin SKIP_BUILD=1 primero."
    [ -d "$SVC/ui-energy/dist" ] || \
        fail "ui-energy/dist no existe. Ejecutar sin SKIP_BUILD=1 primero."
fi

# ---------------------------------------------------------------------------- #
log "Preparando directorio en la RPi..."
ssh_cmd "mkdir -p $RPI_DEPLOY_DIR/{mc60-reader,energy-sender,ui-energy,database}"

# ---------------------------------------------------------------------------- #
log "Transfiriendo mc60-reader..."
scp_file "$SVC/mc60-reader/main.py"          "$RPI_DEPLOY_DIR/mc60-reader/"
scp_file "$SVC/mc60-reader/configurator.py"  "$RPI_DEPLOY_DIR/mc60-reader/"
scp_file "$SVC/mc60-reader/requirements.txt" "$RPI_DEPLOY_DIR/mc60-reader/"
scp_file "$SVC/mc60-reader/Dockerfile"       "$RPI_DEPLOY_DIR/mc60-reader/"

log "Transfiriendo energy-sender..."
scp_file "$SVC/energy-sender/energy-sender-arm64"  "$RPI_DEPLOY_DIR/energy-sender/"
scp_file "$SVC/energy-sender/Dockerfile.prebuilt"  "$RPI_DEPLOY_DIR/energy-sender/Dockerfile"

log "Transfiriendo ui-energy..."
scp_dir  "$SVC/ui-energy/dist"          "$RPI_DEPLOY_DIR/ui-energy/"
scp_file "$SVC/ui-energy/nginx.conf"    "$RPI_DEPLOY_DIR/ui-energy/"
scp_file "$SVC/ui-energy/Dockerfile.prebuilt" "$RPI_DEPLOY_DIR/ui-energy/Dockerfile"

log "Transfiriendo migraciones SQL..."
for f in 26_energy_local.sql 27_energy_meters.sql 28_energy_meter_index.sql 29_energy_config_audit.sql; do
    [ -f "$INFRA_DB/$f" ] && scp_file "$INFRA_DB/$f" "$RPI_DEPLOY_DIR/database/" || true
done

log "Transfiriendo docker-compose y .env..."
scp_file "$SCRIPT_DIR/docker-compose.raspberry-energy.yml" "$RPI_DEPLOY_DIR/docker-compose.yml"
# Copiar .env si no existe en la RPi para no pisar configuracion existente
ssh_cmd "test -f $RPI_DEPLOY_DIR/.env && echo 'env existente, no se sobreescribe' || cp $RPI_DEPLOY_DIR/../.env.rpi $RPI_DEPLOY_DIR/.env 2>/dev/null || true"
if ! ssh_cmd "test -f $RPI_DEPLOY_DIR/.env"; then
    scp_file "$SCRIPT_DIR/.env.rpi" "$RPI_DEPLOY_DIR/.env"
    log "  .env copiado desde .env.rpi — REVISAR DEVICE_ID y MODBUS_SERIAL_DEV"
fi

# ---------------------------------------------------------------------------- #
log "Construyendo imagenes Docker en la RPi..."
ssh_cmd "
    set -e
    cd $RPI_DEPLOY_DIR

    echo '[rpi] Construyendo mc60-reader...'
    docker build -t mentor-mc60-reader:latest mc60-reader/

    echo '[rpi] Construyendo energy-sender...'
    chmod +x energy-sender/energy-sender-arm64
    docker build -t mentor-energy-sender:arm64 energy-sender/

    echo '[rpi] Construyendo ui-energy...'
    docker build -t mentor-ui-energy:arm64 ui-energy/

    echo '[rpi] Imagenes construidas:'
    docker images | grep mentor
"

# ---------------------------------------------------------------------------- #
log "Lanzando el stack..."
ssh_cmd "
    set -e
    cd $RPI_DEPLOY_DIR
    docker compose --env-file .env pull postgres 2>/dev/null || true
    docker compose --env-file .env up -d --no-build
    sleep 5
    docker compose ps
"

log ""
log "Deploy completado."
log "  UI:            http://${RPI_HOST}:8087"
log "  API:           http://${RPI_HOST}:8086/health"
log "  Logs:          sshpass -p '${RPI_PASS}' ssh ${RPI_USER}@${RPI_HOST} 'cd ${RPI_DEPLOY_DIR} && docker compose logs -f'"
log ""
log "Si es el primer deploy, verificar .env en la RPi:"
log "  DEVICE_ID       = identificador del dispositivo"
log "  MODBUS_SERIAL_DEV = puerto RS-485 (ej: /dev/ttyUSB0)"
