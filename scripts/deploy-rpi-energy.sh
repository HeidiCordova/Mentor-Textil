#!/usr/bin/env bash
# =============================================================================
# deploy-rpi-energy.sh
# Despliegue completo del stack de energía en un Raspberry Pi nuevo o de reemplazo.
#
# Stack: postgres (buffer local) + mc60-reader (Python/Modbus) +
#        energy-sender (Go/cloud-sync) + ui-energy (interfaz web local)
#
# Uso:
#   ./scripts/deploy-rpi-energy.sh \
#     --rpi-ip      <IP_DEL_DISPOSITIVO>  \
#     --rpi-user    py             \
#     --rpi-pass    12345678       \
#     --device-id   rpi-energy-01  \
#     --meter-id-1  mc60-01        \
#     --cloud-url   http://152.53.253.59:8888 \
#     --api-key     <ENERGY_API_KEY> \
#     [--tz         America/Lima]  \
#     [--serial-dev /dev/mc60-mbus]
#
# =============================================================================
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERR]${NC}  $*"; exit 1; }

# ── Valores por defecto ───────────────────────────────────────────────────────
RPI_USER="py"
RPI_PASS="12345678"
TZ="America/Lima"
DEVICE_ID="rpi-energy-01"
METER_ID_1="mc60-01"
METER_ID_2=""
METER_ID_3=""
SERIAL_DEV="/dev/mc60-mbus"
CLOUD_URL=""
API_KEY=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── Parser de argumentos ──────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --rpi-ip)     RPI_IP="$2";      shift 2 ;;
    --rpi-user)   RPI_USER="$2";    shift 2 ;;
    --rpi-pass)   RPI_PASS="$2";    shift 2 ;;
    --device-id)  DEVICE_ID="$2";   shift 2 ;;
    --meter-id-1) METER_ID_1="$2";  shift 2 ;;
    --meter-id-2) METER_ID_2="$2";  shift 2 ;;
    --meter-id-3) METER_ID_3="$2";  shift 2 ;;
    --serial-dev) SERIAL_DEV="$2";  shift 2 ;;
    --cloud-url)  CLOUD_URL="$2";   shift 2 ;;
    --api-key)    API_KEY="$2";     shift 2 ;;
    --tz)         TZ="$2";          shift 2 ;;
    *) error "Argumento desconocido: $1" ;;
  esac
done

[[ -z "${RPI_IP:-}"    ]] && error "Falta --rpi-ip"
[[ -z "${CLOUD_URL:-}" ]] && error "Falta --cloud-url"
[[ -z "${API_KEY:-}"   ]] && error "Falta --api-key"

# ── Helpers SSH/SCP ───────────────────────────────────────────────────────────
SSHOPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
ssh_cmd()  { sshpass -p "$RPI_PASS" ssh  $SSHOPTS "${RPI_USER}@${RPI_IP}" "$@"; }
scp_file() { sshpass -p "$RPI_PASS" scp  $SSHOPTS "$1" "${RPI_USER}@${RPI_IP}:$2"; }

# ── Verificar conectividad ────────────────────────────────────────────────────
info "Verificando conexión con $RPI_IP..."
ssh_cmd 'echo OK' > /dev/null || error "No se puede conectar a $RPI_IP"
success "RPi accesible"

# ── PASO 1: Regla udev para CH340/CH341 → /dev/mc60-mbus ─────────────────────
info "Configurando regla udev para CH340/CH341..."
ssh_cmd 'sudo tee /etc/udev/rules.d/99-mc60-serial.rules > /dev/null << EOF
SUBSYSTEM=="tty", ATTRS{idVendor}=="1a86", ATTRS{idProduct}=="7523", SYMLINK+="mc60-mbus", GROUP="dialout", MODE="0660"
SUBSYSTEM=="tty", ATTRS{idVendor}=="1a86", ATTRS{idProduct}=="55d4", SYMLINK+="mc60-mbus", GROUP="dialout", MODE="0660"
EOF
sudo udevadm control --reload-rules && sudo udevadm trigger'
sleep 2
DEVICE=$(ssh_cmd 'ls -la /dev/mc60-mbus 2>/dev/null || echo "MISSING"')
if echo "$DEVICE" | grep -q "MISSING"; then
  warn "CH340 no detectado aún — el symlink aparecerá al conectar el adaptador USB-RS485"
else
  success "CH340 detectado: $DEVICE"
fi

# ── PASO 2: Detener stack anterior si existe ──────────────────────────────────
info "Deteniendo stack anterior si existe..."
REMOTE_DIR="/home/${RPI_USER}/mentor-edge/infrastructure/docker"
ssh_cmd "cd $REMOTE_DIR 2>/dev/null && \
  docker compose -f docker-compose.raspberry-energy.yml down --remove-orphans 2>/dev/null || true"
ssh_cmd 'docker stop mc60-communication rpi-node-red 2>/dev/null || true'
ssh_cmd 'pm2 delete node-red 2>/dev/null || true; pm2 save --force 2>/dev/null || true'
success "Stack anterior detenido"

# ── PASO 3: Sincronizar archivos al RPi ───────────────────────────────────────
info "Sincronizando archivos al RPi..."
ssh_cmd "mkdir -p $REMOTE_DIR/database $REMOTE_DIR/../services/mc60-reader"

scp_file "$REPO_ROOT/mentor-edge/infrastructure/docker/docker-compose.raspberry-energy.yml" \
         "$REMOTE_DIR/"

for sql in 26_energy_local.sql 27_energy_meters.sql 28_energy_meter_index.sql 29_energy_config_audit.sql; do
  F="$REPO_ROOT/mentor-edge/infrastructure/docker/database/$sql"
  [[ -f "$F" ]] && scp_file "$F" "$REMOTE_DIR/database/" || warn "No encontrado: $F"
done

for f in Dockerfile main.py configurator.py requirements.txt; do
  F="$REPO_ROOT/mentor-edge/services/mc60-reader/$f"
  [[ -f "$F" ]] && scp_file "$F" "$REMOTE_DIR/../services/mc60-reader/" || warn "No encontrado: $F"
done

success "Archivos sincronizados"

# ── PASO 4: Escribir .env ─────────────────────────────────────────────────────
info "Escribiendo .env.raspberry-energy..."
ssh_cmd "cat > $REMOTE_DIR/.env.raspberry-energy << EOF
TZ=${TZ}
DEVICE_ID=${DEVICE_ID}
MODBUS_SERIAL_DEV=${SERIAL_DEV}
METER_ID_1=${METER_ID_1}
METER_UNIT_ID_1=1
METER_ID_2=${METER_ID_2}
METER_UNIT_ID_2=2
METER_ID_3=${METER_ID_3}
METER_UNIT_ID_3=3
EOF"
success ".env creado"

# ── PASO 5: Build de imagen mc60-reader en el RPi ─────────────────────────────
info "Construyendo imagen mentor-mc60-reader en el RPi (puede tardar ~3 min en primera vez)..."
MC60_SRC="/home/${RPI_USER}/mentor-edge/services/mc60-reader"
ssh_cmd "docker build -t mentor-mc60-reader:latest $MC60_SRC"
success "Imagen mentor-mc60-reader:latest construida"

# ── PASO 6: Arrancar el stack ─────────────────────────────────────────────────
info "Arrancando stack..."
ssh_cmd "cd $REMOTE_DIR && \
  docker compose -f docker-compose.raspberry-energy.yml --env-file .env.raspberry-energy up -d"
success "Servicios iniciados"

# ── PASO 7: Esperar healthcheck de postgres ───────────────────────────────────
info "Esperando que postgres esté healthy (máx 60s)..."
STATUS="none"
for i in $(seq 1 12); do
  STATUS=$(ssh_cmd 'docker inspect rpi-energy-postgres --format "{{.State.Health.Status}}" 2>/dev/null || echo "none"')
  [[ "$STATUS" == "healthy" ]] && break
  echo -n "."
  sleep 5
done
echo ""
[[ "$STATUS" == "healthy" ]] && success "Postgres healthy" || error "Postgres no llegó a healthy en 60s"

# ── PASO 8: Escribir config operacional en DB ─────────────────────────────────
info "Escribiendo config operacional en energy.config..."
ssh_cmd "docker exec rpi-energy-postgres psql -U mentor -d mentor_energy -c \"
INSERT INTO energy.config (key, value) VALUES
  ('device_id',        '${DEVICE_ID}'),
  ('cloud_url',        '${CLOUD_URL}'),
  ('energy_api_key',   '${API_KEY}'),
  ('send_interval_s',  '300'),
  ('batch_size',       '50')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
\"" && success "Config guardada en DB" || warn "No se pudo escribir config en DB"

ssh_cmd 'curl -s -X POST http://localhost:8086/config/reload > /dev/null 2>&1 || true'

# ── PASO 9: Validación final ──────────────────────────────────────────────────
echo ""
info "====== VALIDACIÓN FINAL ======"
ssh_cmd 'docker ps --format "{{.Names}}: {{.Status}}" | grep rpi-'
echo ""
info "Logs mc60-reader (últimas 5 líneas):"
ssh_cmd 'docker logs rpi-mc60-reader --tail 5 2>&1' || true
echo ""
info "Logs energy-sender (últimas 5 líneas):"
ssh_cmd 'docker logs rpi-energy-sender --tail 5 2>&1' || true
echo ""
success "Deploy completado en $RPI_IP"
