#!/bin/bash
# =======================================================================
# deploy-new-jetson.sh
# Despliegue completo de Mentor Edge en un Jetson nuevo desde cero.
#
# Pasos automatizados:
#   1. Registrar el dispositivo en cloud DB (gateway.device_registry)
#   2. Sincronizar el repositorio al Jetson vía rsync
#   3. Crear el schema de la línea (linea_template.sql) en la DB local
#   4. Escribir el .env con todas las variables
#   5. Build y arranque de servicios Docker
#   6. Configurar la línea vía API (edge-config-service)
#   7. Verificación final
#
# Uso:
#   ./scripts/deploy-new-jetson.sh \
#     --jetson-ip    192.168.100.31 \
#     --device-id    1 \
#     --empresa-id   8 \
#     --planta-id    14 \
#     --cloud-linea-id 14 \
#     --cloud-url    http://152.53.253.59:8888 \
#     --cloud-db-url "postgresql://mentor:mentor2026@152.53.253.59:5432/mentor_cloud" \
#     --line-name    "LINEA1 — Confección" \
#     --camera-url   "rtsp://user:pass@192.168.100.32:554/stream2" \
#     [--jetson-user orin] [--jetson-pass 123456] [--tz America/Lima]
#
# Ejemplo mínimo (Art Atlas):
#   ./scripts/deploy-new-jetson.sh \
#     --jetson-ip 192.168.100.31 --device-id 1 \
#     --empresa-id 8 --planta-id 14 --cloud-linea-id 14 \
#     --cloud-url http://152.53.253.59:8888 \
#     --cloud-db-url "postgresql://mentor:mentor2026@152.53.253.59:5432/mentor_cloud" \
#     --line-name "LINEA1" --camera-url "rtsp://..."
# =======================================================================
set -euo pipefail

# ── Valores por defecto ───────────────────────────────────────────────
JETSON_USER="orin"
JETSON_PASS="123456"
TZ="America/Lima"
LOCAL_LINEA_ID=""   # se infiere como cloud-linea-id si no se especifica
LINE_NAME="LINEA1"
CAMERA_URL=""

# ── Parser de argumentos ──────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --jetson-ip)       JETSON_IP="$2";       shift 2 ;;
    --device-id)       DEVICE_ID="$2";       shift 2 ;;
    --empresa-id)      EMPRESA_ID="$2";      shift 2 ;;
    --planta-id)       PLANTA_ID="$2";       shift 2 ;;
    --cloud-linea-id)  CLOUD_LINEA_ID="$2";  shift 2 ;;
    --local-linea-id)  LOCAL_LINEA_ID="$2";  shift 2 ;;
    --cloud-url)       CLOUD_URL="$2";       shift 2 ;;
    --cloud-db-url)    CLOUD_DB_URL="$2";    shift 2 ;;
    --line-name)       LINE_NAME="$2";       shift 2 ;;
    --camera-url)      CAMERA_URL="$2";      shift 2 ;;
    --jetson-user)     JETSON_USER="$2";     shift 2 ;;
    --jetson-pass)     JETSON_PASS="$2";     shift 2 ;;
    --tz)              TZ="$2";              shift 2 ;;
    *) echo "Argumento desconocido: $1"; exit 1 ;;
  esac
done

# Validaciones
: "${JETSON_IP:?Falta --jetson-ip}"
: "${DEVICE_ID:?Falta --device-id}"
: "${EMPRESA_ID:?Falta --empresa-id}"
: "${PLANTA_ID:?Falta --planta-id}"
: "${CLOUD_LINEA_ID:?Falta --cloud-linea-id}"
: "${CLOUD_URL:?Falta --cloud-url}"
: "${CLOUD_DB_URL:?Falta --cloud-db-url}"

# El linea_id local del schema (linea_N) coincide con cloud-linea-id por defecto
# para evitar confusión. Puede sobreescribirse con --local-linea-id si hay conflicto.
LOCAL_LINEA_ID="${LOCAL_LINEA_ID:-$CLOUD_LINEA_ID}"

REMOTE_DIR="/home/$JETSON_USER/mentor-edge"
COMPOSE_FILE="infrastructure/docker/docker-compose.jetson-orin.yml"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

SSH_CMD="sshpass -p '$JETSON_PASS' ssh -o StrictHostKeyChecking=no $JETSON_USER@$JETSON_IP"

_ssh() { eval "sshpass -p '$JETSON_PASS' ssh -o StrictHostKeyChecking=no $JETSON_USER@$JETSON_IP $@"; }
_ssh_bash() {
  sshpass -p "$JETSON_PASS" ssh -o StrictHostKeyChecking=no "$JETSON_USER@$JETSON_IP" bash -s
}

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         Mentor Edge — Despliegue en Jetson nuevo               ║"
echo "╠════════════════════════════════════════════════════════════════╣"
printf "  %-18s %s\n" "Jetson IP:"      "$JETSON_IP"
printf "  %-18s %s\n" "Device ID:"      "$DEVICE_ID"
printf "  %-18s %s\n" "Empresa/Planta:" "$EMPRESA_ID / $PLANTA_ID"
printf "  %-18s %s (cloud) → linea_%s (local schema)\n" "Línea ID:" "$CLOUD_LINEA_ID" "$LOCAL_LINEA_ID"
printf "  %-18s %s\n" "Cloud URL:"      "$CLOUD_URL"
printf "  %-18s %s\n" "Nombre línea:"   "$LINE_NAME"
if [[ -n "$CAMERA_URL" ]]; then
  printf "  %-18s %s\n" "Cámara:" "$(echo "$CAMERA_URL" | sed 's|://[^:]*:[^@]*@|://****:****@|')"
fi
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
read -rp "¿Continuar? (s/N): " CONFIRM
[[ "${CONFIRM,,}" == "s" ]] || { echo "Cancelado."; exit 0; }

# ── PASO 1: Registrar dispositivo en cloud DB ─────────────────────────
echo ""
echo "▶ [1/7] Registrando dispositivo en cloud DB..."

API_KEY=$(openssl rand -hex 32)

psql "$CLOUD_DB_URL" \
  -v "device_id='$DEVICE_ID'" \
  -v "device_nombre='Jetson $DEVICE_ID'" \
  -v "empresa_id=$EMPRESA_ID" \
  -v "planta_id=$PLANTA_ID" \
  -v "linea_id=$CLOUD_LINEA_ID" \
  -v "api_key='$API_KEY'" \
  -f "$REPO_ROOT/mentor-cloud/infrastructure/database/register_device.sql"

echo "  ✓ Dispositivo registrado. API Key: ${API_KEY:0:8}..."

# ── PASO 2: Sincronizar repositorio al Jetson ─────────────────────────
echo ""
echo "▶ [2/7] Sincronizando repositorio mentor-edge al Jetson..."

sshpass -p "$JETSON_PASS" rsync -az --delete --progress \
  -e "ssh -o StrictHostKeyChecking=no" \
  --exclude=".git" \
  --exclude="node_modules" \
  --exclude="__pycache__" \
  --exclude="*.pyc" \
  "$REPO_ROOT/mentor-edge/" \
  "$JETSON_USER@$JETSON_IP:$REMOTE_DIR/"

echo "  ✓ Repositorio sincronizado en $REMOTE_DIR"

# ── PASO 3: Crear schema de línea en DB local del Jetson ──────────────
echo ""
echo "▶ [3/7] Creando schema linea_${LOCAL_LINEA_ID} en DB local del Jetson..."

# Primero levantamos solo postgres para poder ejecutar SQL
_ssh_bash <<REMOTE
set -euo pipefail
cd "$REMOTE_DIR/infrastructure/docker"

# Levantar solo postgres si no está corriendo
docker compose -f docker-compose.jetson-orin.yml up -d postgres
echo "  Esperando que postgres esté listo..."
for i in \$(seq 1 30); do
  docker compose -f docker-compose.jetson-orin.yml exec -T postgres \
    pg_isready -U mentor -d mentor_edge -q && break || sleep 2
done

# Verificar si el schema ya existe
SCHEMA_EXISTS=\$(docker compose -f docker-compose.jetson-orin.yml exec -T postgres \
  psql -U mentor -d mentor_edge -tAc \
  "SELECT COUNT(*) FROM pg_namespace WHERE nspname='linea_${LOCAL_LINEA_ID}';")

if [ "\$SCHEMA_EXISTS" = "1" ]; then
  echo "  ⚠ Schema linea_${LOCAL_LINEA_ID} ya existe — omitiendo creación"
else
  sed 's/{schema}/linea_${LOCAL_LINEA_ID}/g' ../database/linea_template.sql \
    | docker compose -f docker-compose.jetson-orin.yml exec -T postgres \
      psql -U mentor -d mentor_edge -v ON_ERROR_STOP=1
  echo "  ✓ Schema linea_${LOCAL_LINEA_ID} creado"
fi
REMOTE

# ── PASO 4: Escribir .env ─────────────────────────────────────────────
echo ""
echo "▶ [4/7] Escribiendo .env en el Jetson..."

_ssh_bash <<REMOTE
set -euo pipefail
mkdir -p "$REMOTE_DIR/infrastructure/docker"
cat > "$REMOTE_DIR/infrastructure/docker/.env" <<'ENVEOF'
# Generado automáticamente por deploy-new-jetson.sh — NO editar manualmente
CLOUD_URL=${CLOUD_URL}
CLOUD_API_KEY=${API_KEY}
TZ=${TZ}
ENVEOF
echo "  ✓ .env escrito"
REMOTE

# ── PASO 5: Build y arranque (FASE 1 — sin edge-gateway) ─────────────
# edge-gateway hace fatal si no hay ninguna línea en config.line_config.
# Arrancamos todo lo demás primero, configuramos la línea vía API,
# y solo entonces levantamos edge-gateway para que encuentre la config.
echo ""
echo "▶ [5/7] Construyendo imágenes Docker (puede tardar ~10 min)..."

_ssh_bash <<REMOTE
set -euo pipefail
cd "$REMOTE_DIR/infrastructure/docker"
docker compose -f docker-compose.jetson-orin.yml build --parallel 2>&1 | tail -5
echo "  ✓ Imágenes construidas"

# Fase 1: todos los servicios EXCEPTO edge-gateway
# (edge-gateway necesita que config.line_config ya tenga la línea)
docker compose -f docker-compose.jetson-orin.yml up -d \
  postgres resiliencia enviador edge-config-service ui-local tablet-app
echo "  ✓ Servicios base levantados (edge-gateway pendiente hasta configurar línea)"
REMOTE

# ── PASO 6: Configurar línea vía API ─────────────────────────────────
echo ""
echo "▶ [6/7] Configurando línea vía edge-config-service API..."

# Esperar a que el edge-config-service esté listo
echo "  Esperando que edge-config-service responda en :8004..."
for i in $(seq 1 30); do
  HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "http://$JETSON_IP:8004/config?linea_id=$LOCAL_LINEA_ID" 2>/dev/null || true)
  [[ "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "404" ]] && break
  sleep 3
done

# PUT con configuración mínima de la línea
CAMERA_JSON="null"
if [[ -n "$CAMERA_URL" ]]; then
  CAMERA_JSON="{\"url\":\"$CAMERA_URL\",\"fps\":15,\"roi\":[0,0,1920,1080]}"
fi

curl -s -X PUT "http://$JETSON_IP:8004/config?linea_id=$LOCAL_LINEA_ID" \
  -H "Content-Type: application/json" \
  -d "{
    \"linea_id\": $LOCAL_LINEA_ID,
    \"empresa_id\": $EMPRESA_ID,
    \"cloud_linea_id\": $CLOUD_LINEA_ID,
    \"mode\": \"cloud\",
    \"oee\": {
      \"line_name\": \"$LINE_NAME\"
    },
    \"camera\": $CAMERA_JSON,
    \"cloud\": {
      \"cloud_url\": \"$CLOUD_URL\",
      \"cloud_api_key\": \"$API_KEY\"
    }
  }" | python3 -c "import json,sys; d=json.load(sys.stdin); print('  ✓ Línea configurada:', d.get('linea_id','?'))" \
  || { echo "  ✗ No se pudo configurar la línea vía API"; echo "    → hazlo manualmente en http://$JETSON_IP:8080/config/$LOCAL_LINEA_ID"; echo "    → Luego ejecuta: docker compose up -d edge-gateway"; exit 1; }

# ── Fase 2: levantar edge-gateway ahora que la línea ya está en DB ───
echo ""
echo "  ▶ Levantando edge-gateway (config.line_config ya tiene la línea)..."
_ssh_bash <<REMOTE
set -euo pipefail
cd "$REMOTE_DIR/infrastructure/docker"
docker compose -f docker-compose.jetson-orin.yml up -d edge-gateway
echo "  ✓ edge-gateway levantado"
REMOTE

# ── PASO 7: Verificación final ────────────────────────────────────────
echo ""
echo "▶ [7/7] Verificando estado de servicios..."
echo "  Esperando que edge-gateway levante (15 s)..."
sleep 15
_ssh_bash <<REMOTE
cd "$REMOTE_DIR/infrastructure/docker"
docker compose -f docker-compose.jetson-orin.yml ps --format "table {{.Service}}\t{{.Status}}"
REMOTE

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  ✅  Despliegue completado                                      ║"
echo "╠════════════════════════════════════════════════════════════════╣"
printf "  %-20s %s\n" "UI local:"          "http://$JETSON_IP:8080"
printf "  %-20s %s\n" "Tablet app:"        "http://$JETSON_IP:8090"
printf "  %-20s %s\n" "Gateway edge:"      "http://$JETSON_IP:8005"
printf "  %-20s %s\n" "Config API:"        "http://$JETSON_IP:8004"
printf "  %-20s %s\n" "Schema local:"      "linea_$LOCAL_LINEA_ID"
printf "  %-20s %s\n" "Cloud linea_id:"    "$CLOUD_LINEA_ID"
printf "  %-20s %s\n" "API Key (primeros 8):" "${API_KEY:0:8}..."
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "  Próximos pasos:"
echo "  1. Verificar cámara:  http://$JETSON_IP:8080  → Configuración → Línea $LOCAL_LINEA_ID"
echo "  2. Forzar SYNC desde cloud: ir a la UI cloud y usar el botón de sincronización"
echo "  3. Guardar la API Key completa en un lugar seguro:"
echo "     $API_KEY"
