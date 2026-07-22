#!/bin/bash
# =======================================================================
# build-deb.sh
# Genera el paquete mentor-edge_VERSION_arm64.deb listo para instalar
# en un Jetson Orin con `sudo apt install ./mentor-edge_VERSION_arm64.deb`.
#
# Qué incluye el .deb:
#   - Binarios arm64 pre-compilados de todos los servicios Go
#   - dist/ pre-compilado de ui-local (Vue/Vite — UI de administración, :8080)
#   - dist/ pre-compilado de tablet-app (Vue — UI operadores, :8090)
#   - SQL de inicialización de DB (init.sql, linea_template.sql, etc.)
#   - docker-compose.yml con rutas absolutas al .deb instalado
#   - Dockerfiles slim (solo COPY del binario, sin compilar en el Jetson)
#   - Servicio systemd mentor-edge.service
#   - CLI mentor-edge-setup para configuración inicial guiada
#
# Prerequisitos en la máquina de build (Linux x86_64):
#   - Go 1.22+  (GOARCH=arm64 cross-compile)
#   - Node.js 18+ y npm
#   - dpkg-deb (apt install dpkg)
#
# Uso:
#   ./scripts/build-deb.sh [--version 1.2.0] [--use-existing-bins] [--skip-ui] [--skip-tablet]
#
# Resultado:
#   dist/mentor-edge_VERSION_arm64.deb
# =======================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
EDGE_ROOT="$REPO_ROOT/mentor-edge"

# ── Defaults ─────────────────────────────────────────────────────────
VERSION="1.0.0"
USE_EXISTING_BINS=false
SKIP_UI=false
SKIP_TABLET=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)           VERSION="$2"; shift 2 ;;
    --use-existing-bins) USE_EXISTING_BINS=true; shift ;;
    --skip-ui)           SKIP_UI=true; shift ;;
    --skip-tablet)       SKIP_TABLET=true; shift ;;
    *) echo "Argumento desconocido: $1"; exit 1 ;;
  esac
done

PACKAGE="mentor-edge"
ARCH="arm64"
DEB_NAME="${PACKAGE}_${VERSION}_${ARCH}"
BUILD_DIR="$REPO_ROOT/dist/deb_build/${DEB_NAME}"
OUTPUT_DIR="$REPO_ROOT/dist"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         Mentor Edge — build-deb.sh                        ║"
echo "╠═══════════════════════════════════════════════════════════╣"
printf "  %-20s %s\n" "Versión:"    "$VERSION"
printf "  %-20s %s\n" "Paquete:"    "${DEB_NAME}.deb"
printf "  %-20s %s\n" "Directorio:" "$BUILD_DIR"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# ── Crear árbol del paquete ───────────────────────────────────────────
rm -rf "$BUILD_DIR"
mkdir -p \
  "$BUILD_DIR/DEBIAN" \
  "$BUILD_DIR/opt/mentor-edge/compose" \
  "$BUILD_DIR/opt/mentor-edge/database" \
  "$BUILD_DIR/opt/mentor-edge/services/edge-gateway/bin" \
  "$BUILD_DIR/opt/mentor-edge/services/resiliencia/bin" \
  "$BUILD_DIR/opt/mentor-edge/services/enviador/bin" \
  "$BUILD_DIR/opt/mentor-edge/services/edge-config-service/bin" \
  "$BUILD_DIR/opt/mentor-edge/services/ui-local/dist" \
  "$BUILD_DIR/opt/mentor-edge/services/tablet-app/dist" \
  "$BUILD_DIR/opt/mentor-edge/services/vision-event-detector" \
  "$BUILD_DIR/opt/mentor-edge/services/yolo-counter" \
  "$BUILD_DIR/etc/mentor-edge" \
  "$BUILD_DIR/usr/bin" \
  "$BUILD_DIR/lib/systemd/system" \
  "$OUTPUT_DIR"

# ════════════════════════════════════════════════════════════════════
# PASO 1: Compilar binarios Go para arm64
# ════════════════════════════════════════════════════════════════════
compile_go_service() {
  local SERVICE="$1"
  local BIN_OUT="$2"
  local SRC_DIR="$EDGE_ROOT/services/$SERVICE"

  if [[ "$USE_EXISTING_BINS" == "true" ]]; then
    # Buscar binario arm64 existente (con o sin sufijo -arm64)
    local EXISTING
    EXISTING=$(find "$SRC_DIR" -maxdepth 1 -name "*arm64*" -not -name "*.bak" | head -1)
    if [[ -z "$EXISTING" ]]; then
      # Fallback: binario sin sufijo si es aarch64 en su metadata ELF
      local FALLBACK
      FALLBACK=$(find "$SRC_DIR" -maxdepth 1 -type f -executable ! -name "*.sh" ! -name "*.go" ! -name "go.*" | head -1)
      if [[ -n "$FALLBACK" ]] && file "$FALLBACK" | grep -q 'ARM aarch64'; then
        EXISTING="$FALLBACK"
      fi
    fi
    if [[ -n "$EXISTING" ]]; then
      echo "  [existing] $SERVICE → $(basename "$EXISTING")"
      cp "$EXISTING" "$BIN_OUT"
      chmod +x "$BIN_OUT"
      return
    fi
  fi

  echo "  [compile]  $SERVICE (GOOS=linux GOARCH=arm64)..."
  cd "$SRC_DIR"
  GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w -X main.Version=$VERSION" \
    -o "$BIN_OUT" ./cmd/main.go
  chmod +x "$BIN_OUT"
  echo "  ✓ $SERVICE → $(du -sh "$BIN_OUT" | cut -f1)"
}

echo "▶ [1/6] Compilando servicios Go para arm64..."
compile_go_service "edge-gateway"       "$BUILD_DIR/opt/mentor-edge/services/edge-gateway/bin/edge-gateway"
compile_go_service "resiliencia"        "$BUILD_DIR/opt/mentor-edge/services/resiliencia/bin/resiliencia"
compile_go_service "enviador"           "$BUILD_DIR/opt/mentor-edge/services/enviador/bin/enviador"
compile_go_service "edge-config-service" "$BUILD_DIR/opt/mentor-edge/services/edge-config-service/bin/edge-config-service"

# ════════════════════════════════════════════════════════════════════
# PASO 2: Build del UI local (Vue/Vite)
# ════════════════════════════════════════════════════════════════════
echo ""
echo "▶ [2/6] Compilando ui-local (Vue/Vite)..."

UI_DIR="$EDGE_ROOT/services/ui-local"
if [[ "$SKIP_UI" == "true" && -d "$UI_DIR/dist" ]]; then
  echo "  [existing] Usando dist/ existente"
  cp -r "$UI_DIR/dist/." "$BUILD_DIR/opt/mentor-edge/services/ui-local/dist/"
else
  cd "$UI_DIR"
  npm ci --silent
  npm run build
  cp -r "$UI_DIR/dist/." "$BUILD_DIR/opt/mentor-edge/services/ui-local/dist/"
  echo "  ✓ ui-local dist copiado"
fi
cp "$UI_DIR/nginx.conf" "$BUILD_DIR/opt/mentor-edge/services/ui-local/"

# ════════════════════════════════════════════════════════════════════
# PASO 2b: Tablet App (Vue — UI operadores servida desde el Jetson en :8090)
# El dist/ de la tablet SÍ debe viajar en el .deb: solo pesa ~364KB
# y es lo que los operadores ven en sus tablets conectándose al Jetson.
# ════════════════════════════════════════════════════════════════════
TABLET_DIR="$REPO_ROOT/mentor-apps/mentor-tablet-app"
if [[ "$SKIP_TABLET" == "true" && -d "$TABLET_DIR/dist" ]]; then
  echo "  [existing] Usando dist/ de tablet existente"
  cp -r "$TABLET_DIR/dist/." "$BUILD_DIR/opt/mentor-edge/services/tablet-app/dist/"
elif [[ -d "$TABLET_DIR" ]]; then
  if [[ ! -d "$TABLET_DIR/dist" ]]; then
    echo "  [build]    Compilando tablet-app..."
    cd "$TABLET_DIR"
    npm ci --silent
    npm run build
  fi
  cp -r "$TABLET_DIR/dist/." "$BUILD_DIR/opt/mentor-edge/services/tablet-app/dist/"
  echo "  ✓ tablet-app dist copiado ($(du -sh "$BUILD_DIR/opt/mentor-edge/services/tablet-app/dist" | cut -f1))"
else
  echo "  ⚠ mentor-apps/mentor-tablet-app no encontrado — tablet-app omitida"
fi
# nginx config de la tablet (la que tiene la config de SPA routing)
cp "$TABLET_DIR/nginx.conf" "$BUILD_DIR/opt/mentor-edge/services/tablet-app/" 2>/dev/null || \
  cp "$EDGE_ROOT/infrastructure/docker/nginx-tablet.conf" "$BUILD_DIR/opt/mentor-edge/services/tablet-app/nginx.conf"

# ════════════════════════════════════════════════════════════════════
# PASO 3: Copiar archivos de configuración e infraestructura
# ════════════════════════════════════════════════════════════════════
echo ""
echo "▶ [3/6] Copiando infraestructura y SQL..."

# Database
cp "$EDGE_ROOT/infrastructure/database/init.sql"           "$BUILD_DIR/opt/mentor-edge/database/"
cp "$EDGE_ROOT/infrastructure/database/linea_template.sql" "$BUILD_DIR/opt/mentor-edge/database/"
cp "$EDGE_ROOT/infrastructure/database/init_line.sh"       "$BUILD_DIR/opt/mentor-edge/database/"
# Migraciones (copiar todas en orden)
for f in "$EDGE_ROOT/infrastructure/database"/[0-9]*.sql; do
  cp "$f" "$BUILD_DIR/opt/mentor-edge/database/" 2>/dev/null || true
done

# vision-event-detector (Python — se construye en el Jetson, necesita GPU base image)
cp -r "$EDGE_ROOT/services/vision-event-detector/." \
  "$BUILD_DIR/opt/mentor-edge/services/vision-event-detector/"

# yolo-counter (systemd nativo — NO va en Docker)
cp "$EDGE_ROOT/services/yolo-counter/install.sh"        "$BUILD_DIR/opt/mentor-edge/services/yolo-counter/"
cp "$EDGE_ROOT/services/yolo-counter/yolo-counter.service" "$BUILD_DIR/opt/mentor-edge/services/yolo-counter/"
cp "$EDGE_ROOT/services/yolo-counter/requirements.txt"  "$BUILD_DIR/opt/mentor-edge/services/yolo-counter/"

echo "  ✓ SQL e infraestructura copiados"

# ════════════════════════════════════════════════════════════════════
# PASO 4: Generar Dockerfiles slim (solo COPY del binario)
# ════════════════════════════════════════════════════════════════════
echo ""
echo "▶ [4/6] Generando Dockerfiles slim y docker-compose.yml..."

# --- edge-gateway ---
cat > "$BUILD_DIR/opt/mentor-edge/services/edge-gateway/Dockerfile" <<'DOCKEREOF'
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates wget && rm -rf /var/lib/apt/lists/*
COPY bin/edge-gateway /app/edge-gateway
RUN chmod +x /app/edge-gateway
EXPOSE 8005
CMD ["/app/edge-gateway"]
DOCKEREOF

# --- resiliencia ---
cat > "$BUILD_DIR/opt/mentor-edge/services/resiliencia/Dockerfile" <<'DOCKEREOF'
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && rm -rf /var/lib/apt/lists/*
COPY bin/resiliencia /app/resiliencia
RUN chmod +x /app/resiliencia
EXPOSE 8002
CMD ["/app/resiliencia"]
DOCKEREOF

# --- enviador ---
cat > "$BUILD_DIR/opt/mentor-edge/services/enviador/Dockerfile" <<'DOCKEREOF'
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && rm -rf /var/lib/apt/lists/*
COPY bin/enviador /app/enviador
RUN chmod +x /app/enviador
EXPOSE 8003
CMD ["/app/enviador"]
DOCKEREOF

# --- edge-config-service ---
cat > "$BUILD_DIR/opt/mentor-edge/services/edge-config-service/Dockerfile" <<'DOCKEREOF'
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates && rm -rf /var/lib/apt/lists/*
COPY bin/edge-config-service /app/edge-config-service
RUN chmod +x /app/edge-config-service
EXPOSE 8004
CMD ["/app/edge-config-service"]
DOCKEREOF

# --- ui-local ---
cat > "$BUILD_DIR/opt/mentor-edge/services/ui-local/Dockerfile" <<'DOCKEREOF'
FROM nginx:stable-alpine
COPY dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
DOCKEREOF

# --- docker-compose.yml para la instalación .deb ---
cat > "$BUILD_DIR/opt/mentor-edge/compose/docker-compose.yml" <<'COMPOSEEOF'
# docker-compose gestionado por /lib/systemd/system/mentor-edge.service
# Configuración en /etc/mentor-edge/.env
# NO editar directamente — generado por build-deb.sh

services:

  postgres:
    image: postgres:14-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: mentor_edge
      POSTGRES_USER: mentor
      POSTGRES_PASSWORD: mentor
      TZ: ${TZ:-America/Lima}
      PGTZ: America/Lima
    volumes:
      - pgdata:/var/lib/postgresql/data
      - /opt/mentor-edge/database/init.sql:/docker-entrypoint-initdb.d/01_schema.sql:ro
      - /opt/mentor-edge/database/05_fix_stop_type_constraint.sql:/docker-entrypoint-initdb.d/03_fix_stop_type.sql:ro
      - /opt/mentor-edge/database/05_production_runs.sql:/docker-entrypoint-initdb.d/04_production_runs.sql:ro
      - /opt/mentor-edge/database/06_prod_runs_constraints.sql:/docker-entrypoint-initdb.d/05_prod_runs_idx.sql:ro
      - /opt/mentor-edge/database/09_vision_detections.sql:/docker-entrypoint-initdb.d/06_vision_detections.sql:ro
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mentor -d mentor_edge"]
      interval: 10s
      timeout: 3s
      retries: 5

  resiliencia:
    build:
      context: /opt/mentor-edge/services/resiliencia
      dockerfile: Dockerfile
    image: mentor-edge/resiliencia:installed
    restart: unless-stopped
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: mentor
      DB_PASSWORD: mentor
      DB_NAME: mentor_edge
      PORT: "8002"
      TZ: ${TZ:-America/Lima}
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8002:8002"

  enviador:
    build:
      context: /opt/mentor-edge/services/enviador
      dockerfile: Dockerfile
    image: mentor-edge/enviador:installed
    restart: unless-stopped
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: mentor
      DB_PASSWORD: mentor
      DB_NAME: mentor_edge
      HEALTH_PORT: "8003"
      TZ: ${TZ:-America/Lima}
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8003:8003"

  edge-config-service:
    build:
      context: /opt/mentor-edge/services/edge-config-service
      dockerfile: Dockerfile
    image: mentor-edge/edge-config-service:installed
    restart: unless-stopped
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: mentor
      DB_PASSWORD: mentor
      DB_NAME: mentor_edge
      PORT: "8004"
      TZ: ${TZ:-America/Lima}
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8004:8004"

  edge-gateway:
    build:
      context: /opt/mentor-edge/services/edge-gateway
      dockerfile: Dockerfile
    image: mentor-edge/edge-gateway:installed
    restart: unless-stopped
    extra_hosts:
      - "host.docker.internal:host-gateway"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: mentor
      DB_PASSWORD: mentor
      DB_NAME: mentor_edge
      PORT: "8005"
      CONFIG_SERVICE_URL: http://edge-config-service:8004
      RESILIENCIA_URL: http://resiliencia:8002
      DETECTOR_URL: http://host.docker.internal:8001
      ENVIADOR_URL: http://enviador:8003
      CLOUD_URL: ${CLOUD_URL}
      CLOUD_API_KEY: ${CLOUD_API_KEY}
      TZ: ${TZ:-America/Lima}
    volumes:
      - /opt/mentor-edge/database/linea_template.sql:/app/linea_template.sql:ro
      - /sys/devices:/sys/devices:ro
      - /sys/class:/sys/class:ro
    depends_on:
      postgres:
        condition: service_healthy
      resiliencia:
        condition: service_healthy
      edge-config-service:
        condition: service_healthy
    ports:
      - "8005:8005"
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8005/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3

  ui-local:
    build:
      context: /opt/mentor-edge/services/ui-local
      dockerfile: Dockerfile
    image: mentor-edge/ui-local:installed
    restart: unless-stopped
    extra_hosts:
      - "host.docker.internal:host-gateway"
    ports:
      - "8080:8080"
    depends_on:
      - resiliencia
      - edge-config-service

  tablet-app:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "8090:80"
    volumes:
      # dist bundled en el .deb — no depende de rutas externas
      - /opt/mentor-edge/services/tablet-app/dist:/usr/share/nginx/html:ro
      - /opt/mentor-edge/services/tablet-app/nginx.conf:/etc/nginx/conf.d/default.conf:ro

  vision-event-detector:
    build:
      context: /opt/mentor-edge/services/vision-event-detector
      dockerfile: Dockerfile
    image: mentor-edge/vision-event-detector:installed
    restart: unless-stopped
    runtime: nvidia
    network_mode: host
    environment:
      NVIDIA_VISIBLE_DEVICES: all
      CONFIG_SERVICE_URL: http://127.0.0.1:8004
      RESILIENCIA_URL: http://127.0.0.1:8002
      GATEWAY_URL: http://127.0.0.1:8005
      TZ: ${TZ:-America/Lima}
    depends_on:
      postgres:
        condition: service_healthy
      edge-config-service:
        condition: service_healthy
      resiliencia:
        condition: service_healthy
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]

volumes:
  pgdata:
COMPOSEEOF

# nginx-tablet.conf ya fue copiado junto al dist de la tablet en PASO 2b
# (ver /opt/mentor-edge/services/tablet-app/nginx.conf)

# .env.example
cat > "$BUILD_DIR/etc/mentor-edge/.env.example" <<'ENVEOF'
# ┌─────────────────────────────────────────────────────────────┐
# │  Mentor Edge — Configuración del Jetson                     │
# │  Copiar como /etc/mentor-edge/.env y completar los valores  │
# └─────────────────────────────────────────────────────────────┘

# URL del cloud gateway (OBLIGATORIO)
CLOUD_URL=http://TU_IP_CLOUD:8888

# API Key del dispositivo generada al registrar en cloud DB (OBLIGATORIO)
# Obtener ejecutando: sudo mentor-edge-setup
CLOUD_API_KEY=

# Zona horaria
TZ=America/Lima

# Nota: la tablet app está bundled en /opt/mentor-edge/services/tablet-app/dist/
ENVEOF

echo "  ✓ docker-compose.yml y Dockerfiles slim generados"

# ════════════════════════════════════════════════════════════════════
# PASO 5: Archivo systemd y CLI mentor-edge-setup
# ════════════════════════════════════════════════════════════════════
echo ""
echo "▶ [5/6] Generando systemd service y CLI..."

# systemd service
cat > "$BUILD_DIR/lib/systemd/system/mentor-edge.service" <<'UNITEOF'
[Unit]
Description=Mentor Edge — Stack de servicios Docker
Documentation=https://github.com/mentor-edge/docs
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
EnvironmentFile=/etc/mentor-edge/.env
WorkingDirectory=/opt/mentor-edge/compose
ExecStartPre=/bin/bash -c '[ -f /etc/mentor-edge/.env ] || { echo "ERROR: Falta /etc/mentor-edge/.env. Ejecuta: sudo mentor-edge-setup" && exit 1; }'
ExecStartPre=/bin/bash -c '[ -n "$CLOUD_API_KEY" ] || { echo "ERROR: CLOUD_API_KEY vacío en /etc/mentor-edge/.env. Ejecuta: sudo mentor-edge-setup" && exit 1; }'
ExecStart=/usr/bin/docker compose \
    -f /opt/mentor-edge/compose/docker-compose.yml \
    --env-file /etc/mentor-edge/.env \
    up -d --remove-orphans
ExecStop=/usr/bin/docker compose \
    -f /opt/mentor-edge/compose/docker-compose.yml \
    down
ExecReload=/usr/bin/docker compose \
    -f /opt/mentor-edge/compose/docker-compose.yml \
    --env-file /etc/mentor-edge/.env \
    up -d --remove-orphans
TimeoutStartSec=600
TimeoutStopSec=60
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
UNITEOF

# CLI mentor-edge-setup
cat > "$BUILD_DIR/usr/bin/mentor-edge-setup" <<'CLIEOF'
#!/bin/bash
# mentor-edge-setup — Configuración inicial guiada del Jetson
# Uso: sudo mentor-edge-setup [--linea-id N] [--cloud-url URL] [--api-key KEY]
set -euo pipefail

ENV_FILE="/etc/mentor-edge/.env"
COMPOSE_FILE="/opt/mentor-edge/compose/docker-compose.yml"
DB_INIT_SH="/opt/mentor-edge/database/init_line.sh"

echo "┌──────────────────────────────────────────────────────────┐"
echo "│  Mentor Edge — Asistente de configuración inicial        │"
echo "└──────────────────────────────────────────────────────────┘"
echo ""

# Leer parámetros o preguntar interactivamente
CLOUD_URL="${CLOUD_URL:-}"
CLOUD_API_KEY="${CLOUD_API_KEY:-}"
LINEA_ID="${1:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --linea-id)  LINEA_ID="$2"; shift 2 ;;
    --cloud-url) CLOUD_URL="$2"; shift 2 ;;
    --api-key)   CLOUD_API_KEY="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# Si .env ya tiene CLOUD_API_KEY, preguntar si reusar
if [[ -f "$ENV_FILE" ]] && grep -q "CLOUD_API_KEY=." "$ENV_FILE"; then
  echo "  ✓ Configuración existente encontrada en $ENV_FILE"
  echo "  Para reconfigurar desde cero, ejecuta: sudo rm $ENV_FILE && sudo mentor-edge-setup"
  echo ""
fi

[[ -z "$CLOUD_URL" ]]     && read -rp "Cloud URL (ej: http://1.2.3.4:8888): "  CLOUD_URL
[[ -z "$CLOUD_API_KEY" ]] && read -rp "API Key del dispositivo:              "  CLOUD_API_KEY
[[ -z "$LINEA_ID" ]]      && read -rp "Linea ID local (ej: 14):              "  LINEA_ID

# Escribir .env
cp "$ENV_FILE.example" "$ENV_FILE" 2>/dev/null || true
cat > "$ENV_FILE" <<ENV
CLOUD_URL=${CLOUD_URL}
CLOUD_API_KEY=${CLOUD_API_KEY}
TZ=America/Lima
ENV
echo "  ✓ /etc/mentor-edge/.env escrito"

# Iniciar solo postgres para crear schema
echo ""
echo "  Iniciando postgres para crear schema linea_${LINEA_ID}..."
docker compose -f "$COMPOSE_FILE" up -d postgres
for i in $(seq 1 30); do
  docker compose -f "$COMPOSE_FILE" exec -T postgres \
    pg_isready -U mentor -d mentor_edge -q && break || sleep 2
done

# Crear schema si no existe
SCHEMA_EXISTS=$(docker compose -f "$COMPOSE_FILE" exec -T postgres \
  psql -U mentor -d mentor_edge -tAc \
  "SELECT COUNT(*) FROM pg_namespace WHERE nspname='linea_${LINEA_ID}';")
if [ "$SCHEMA_EXISTS" = "1" ]; then
  echo "  ⚠ Schema linea_${LINEA_ID} ya existe — omitido"
else
  sed "s/{schema}/linea_${LINEA_ID}/g" /opt/mentor-edge/database/linea_template.sql \
    | docker compose -f "$COMPOSE_FILE" exec -T postgres \
      psql -U mentor -d mentor_edge -v ON_ERROR_STOP=1
  echo "  ✓ Schema linea_${LINEA_ID} creado"
fi

# Levantar servicios base (sin edge-gateway)
echo ""
echo "  Levantando servicios base..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d \
  postgres resiliencia enviador edge-config-service ui-local tablet-app

echo ""
echo "  Esperando que edge-config-service esté listo..."
for i in $(seq 1 30); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8004/config?linea_id=${LINEA_ID}" 2>/dev/null || true)
  [[ "$STATUS" == "200" || "$STATUS" == "404" ]] && break
  sleep 3
done

# Configurar la línea vía API
LINE_NAME="${LINE_NAME:-LINEA${LINEA_ID}}"
EMPRESA_ID="${EMPRESA_ID:-0}"
curl -s -X PUT "http://localhost:8004/config?linea_id=${LINEA_ID}" \
  -H "Content-Type: application/json" \
  -d "{\"linea_id\":${LINEA_ID},\"empresa_id\":${EMPRESA_ID},\"mode\":\"cloud\",\"oee\":{\"line_name\":\"${LINE_NAME}\"},\"cloud\":{\"cloud_url\":\"${CLOUD_URL}\",\"cloud_api_key\":\"${CLOUD_API_KEY}\"}}" \
  > /dev/null && echo "  ✓ Línea ${LINEA_ID} configurada" \
  || echo "  ⚠ API devolvió error — configura manualmente en http://localhost:8080/config/${LINEA_ID}"

# Levantar edge-gateway
echo ""
echo "  Levantando edge-gateway..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d edge-gateway

# Habilitar servicio systemd
systemctl daemon-reload
systemctl enable mentor-edge.service
echo "  ✓ mentor-edge.service habilitado para inicio automático"

echo ""
echo "┌──────────────────────────────────────────────────────────┐"
echo "│  ✅  Configuración completada                             │"
echo "├──────────────────────────────────────────────────────────┤"
printf "  %-20s %s\n" "UI local:"    "http://$(hostname -I | awk '{print $1}'):8080"
printf "  %-20s %s\n" "Tablet:"      "http://$(hostname -I | awk '{print $1}'):8090"
printf "  %-20s %s\n" "Gateway:"     "http://$(hostname -I | awk '{print $1}'):8005"
printf "  %-20s %s\n" "Logs:"        "journalctl -fu mentor-edge"
echo "└──────────────────────────────────────────────────────────┘"
echo ""
echo "  Para instalar yolo-counter (requiere JetPack/CUDA):"
echo "  sudo bash /opt/mentor-edge/services/yolo-counter/install.sh"
CLIEOF
chmod +x "$BUILD_DIR/usr/bin/mentor-edge-setup"

# ════════════════════════════════════════════════════════════════════
# PASO 6: Metadatos DEBIAN (control, postinst, prerm, conffiles)
# ════════════════════════════════════════════════════════════════════
echo ""
echo "▶ [6/6] Generando metadatos DEBIAN y empaquetando..."

# Calcular tamaño instalado en KB
INSTALLED_SIZE=$(du -sk "$BUILD_DIR" | cut -f1)

cat > "$BUILD_DIR/DEBIAN/control" <<CTRLEOF
Package: mentor-edge
Version: ${VERSION}
Architecture: arm64
Maintainer: Mentor Edge <soporte@mentoredge.io>
Installed-Size: ${INSTALLED_SIZE}
Depends: docker.io (>= 20.10) | docker-ce (>= 20.10), docker-compose-plugin (>= 2.0)
Recommends: postgresql-client-14
Description: Mentor Edge — Sistema de monitoreo OEE en tiempo real para Jetson
 Paquete completo de Mentor Edge para Jetson Orin (JetPack 6.x / L4T r36.x).
 Incluye: edge-gateway, resiliencia, enviador, edge-config-service, ui-local.
 .
 Después de instalar, ejecutar: sudo mentor-edge-setup
 .
 Servicios: http://JETSON_IP:8080 (UI), :8090 (tablet), :8005 (gateway)
CTRLEOF

# conffiles — archivos de config que apt NO sobreescribe en upgrades
cat > "$BUILD_DIR/DEBIAN/conffiles" <<'CFEOF'
/etc/mentor-edge/.env.example
CFEOF

# postinst — ejecutado DESPUÉS de instalar el paquete
cat > "$BUILD_DIR/DEBIAN/postinst" <<'POSTEOF'
#!/bin/bash
set -e

case "$1" in
  configure)
    echo ""
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║  Mentor Edge instalado correctamente                     ║"
    echo "╠══════════════════════════════════════════════════════════╣"

    # Verificar Docker
    if ! command -v docker &>/dev/null; then
      echo "  ⚠  Docker no encontrado. Instalar con:"
      echo "     sudo apt install docker.io docker-compose-plugin"
      echo "╚══════════════════════════════════════════════════════════╝"
      exit 0
    fi

    # Verificar docker compose plugin
    if ! docker compose version &>/dev/null; then
      echo "  ⚠  docker compose plugin no encontrado. Instalar con:"
      echo "     sudo apt install docker-compose-plugin"
      echo "╚══════════════════════════════════════════════════════════╝"
      exit 0
    fi

    # Construir imágenes Docker (rápido — binarios pre-compilados)
    echo "  Construyendo imágenes Docker (binarios pre-compilados)..."
    docker compose \
      -f /opt/mentor-edge/compose/docker-compose.yml \
      build \
      --parallel \
      postgres resiliencia enviador edge-config-service ui-local 2>&1 \
      | grep -E "^#|Successfully|ERROR" || true
    echo "  ✓ Imágenes construidas"

    # Habilitar systemd service (no iniciar hasta configurar .env)
    systemctl daemon-reload
    systemctl enable mentor-edge.service 2>/dev/null || true

    echo "  ✓ mentor-edge.service habilitado"
    echo "╠══════════════════════════════════════════════════════════╣"
    echo "  Próximo paso — configurar este Jetson:"
    echo ""
    echo "    sudo mentor-edge-setup"
    echo ""
    echo "  (necesitarás: Cloud URL, API Key y Linea ID)"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo ""
    ;;
esac

exit 0
POSTEOF
chmod 755 "$BUILD_DIR/DEBIAN/postinst"

# prerm — ejecutado ANTES de desinstalar
cat > "$BUILD_DIR/DEBIAN/prerm" <<'PRERMEOF'
#!/bin/bash
set -e

case "$1" in
  remove|purge)
    echo "  Deteniendo servicios Mentor Edge..."
    systemctl stop mentor-edge.service 2>/dev/null || true
    systemctl disable mentor-edge.service 2>/dev/null || true
    docker compose -f /opt/mentor-edge/compose/docker-compose.yml down 2>/dev/null || true
    echo "  ✓ Servicios detenidos"
    ;;
esac

exit 0
PRERMEOF
chmod 755 "$BUILD_DIR/DEBIAN/prerm"

# postrm — ejecutado DESPUÉS de desinstalar (solo con purge elimina datos)
cat > "$BUILD_DIR/DEBIAN/postrm" <<'POSTRMEOF'
#!/bin/bash
set -e

case "$1" in
  purge)
    echo "  Eliminando datos y configuración..."
    systemctl daemon-reload 2>/dev/null || true
    rm -rf /etc/mentor-edge
    # Los volúmenes Docker (pgdata) se preservan intencionalmente
    # Para eliminar datos: docker volume rm mentor-edge-compose_pgdata
    echo "  ✓ Configuración eliminada (base de datos preservada)"
    echo "  Para eliminar también la BD:"
    echo "    docker volume rm mentor-edge-compose_pgdata"
    ;;
esac

exit 0
POSTRMEOF
chmod 755 "$BUILD_DIR/DEBIAN/postrm"

# ════════════════════════════════════════════════════════════════════
# Empaquetar el .deb
# ════════════════════════════════════════════════════════════════════
echo ""
echo "  Empaquetando ${DEB_NAME}.deb..."
dpkg-deb --root-owner-group --build "$BUILD_DIR" "$OUTPUT_DIR/${DEB_NAME}.deb"

DEB_SIZE=$(du -sh "$OUTPUT_DIR/${DEB_NAME}.deb" | cut -f1)
echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  ✅  .deb generado exitosamente                          ║"
echo "╠══════════════════════════════════════════════════════════╣"
printf "  %-18s %s\n" "Archivo:"   "$OUTPUT_DIR/${DEB_NAME}.deb"
printf "  %-18s %s\n" "Tamaño:"    "$DEB_SIZE"
printf "  %-18s %s\n" "Versión:"   "$VERSION"
printf "  %-18s %s\n" "Arch:"      "$ARCH"
echo "╠══════════════════════════════════════════════════════════╣"
echo "  Instalar en el Jetson:"
echo ""
echo "    # Opción 1 — copiar y instalar"
echo "    scp dist/${DEB_NAME}.deb orin@JETSON_IP:/tmp/"
echo "    ssh orin@JETSON_IP 'sudo apt install /tmp/${DEB_NAME}.deb'"
echo ""
echo "    # Opción 2 — desde repo apt privado (si configurado)"
echo "    sudo apt install mentor-edge"
echo ""
echo "    # Luego en el Jetson:"
echo "    sudo mentor-edge-setup"
echo "╚══════════════════════════════════════════════════════════╝"
