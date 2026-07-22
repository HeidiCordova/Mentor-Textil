#!/bin/bash
# =======================================================================
# publish-deb.sh
# Sube el .deb al servidor cloud y actualiza el repositorio apt.
#
# Después de ejecutarlo, en cualquier Jetson nuevo:
#
#   # Una sola vez — agregar el repo:
#   curl -fsSL http://152.53.253.59:9090/install.sh | sudo bash
#
#   # Instalar / actualizar:
#   sudo apt-get update && sudo apt-get install -y mentor-edge
#
# El repo corre en un contenedor nginx en puerto 9090 del servidor cloud.
# No requiere GPG — usa [trusted=yes] porque el tráfico es LAN/VPN privado.
#
# Uso:
#   ./scripts/publish-deb.sh [--version 1.0.0] [--build] [--server IP] [--port 9090]
#
# Con --build compila el .deb antes de subir.
# =======================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# ── Defaults ──────────────────────────────────────────────────────────
VERSION="1.0.0"
SERVER_IP="152.53.253.59"
SERVER_USER="root"
SERVER_PORT="22"
APT_PORT="9090"
DO_BUILD=false
SERVER_PASS=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)    VERSION="$2";     shift 2 ;;
    --server)     SERVER_IP="$2";   shift 2 ;;
    --user)       SERVER_USER="$2"; shift 2 ;;
    --ssh-port)   SERVER_PORT="$2"; shift 2 ;;
    --apt-port)   APT_PORT="$2";    shift 2 ;;
    --pass)       SERVER_PASS="$2"; shift 2 ;;
    --build)      DO_BUILD=true;    shift ;;
    *) echo "Argumento desconocido: $1"; exit 1 ;;
  esac
done

DEB_FILE="$REPO_ROOT/dist/mentor-edge_${VERSION}_arm64.deb"

# Comando SSH base
SSH_OPTS="-o StrictHostKeyChecking=no -p $SERVER_PORT"
if [[ -n "$SERVER_PASS" ]]; then
  _ssh() { sshpass -p "$SERVER_PASS" ssh $SSH_OPTS "$SERVER_USER@$SERVER_IP" "$@"; }
  _scp() { sshpass -p "$SERVER_PASS" scp -P "$SERVER_PORT" -o StrictHostKeyChecking=no "$@"; }
else
  _ssh() { ssh $SSH_OPTS "$SERVER_USER@$SERVER_IP" "$@"; }
  _scp() { scp -P "$SERVER_PORT" -o StrictHostKeyChecking=no "$@"; }
fi

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         Mentor Edge — publish-deb.sh                      ║"
echo "╠═══════════════════════════════════════════════════════════╣"
printf "  %-20s %s\n" "Servidor:"      "$SERVER_IP:$SERVER_PORT"
printf "  %-20s %s\n" "Repo apt URL:"  "http://$SERVER_IP:$APT_PORT"
printf "  %-20s %s\n" "Versión .deb:"  "$VERSION"
printf "  %-20s %s\n" "Archivo:"       "$DEB_FILE"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# ── PASO 1: Build del .deb si se indicó ──────────────────────────────
if [[ "$DO_BUILD" == "true" ]]; then
  echo "▶ [1/4] Compilando .deb..."
  bash "$SCRIPT_DIR/build-deb.sh" --version "$VERSION" --use-existing-bins --skip-ui --skip-tablet
  echo "  ✓ .deb compilado"
else
  echo "▶ [1/4] Saltando build (usar --build para recompilar)"
fi

# Verificar que el .deb existe
if [[ ! -f "$DEB_FILE" ]]; then
  echo ""
  echo "ERROR: No se encuentra $DEB_FILE"
  echo "  Ejecuta primero:  ./scripts/build-deb.sh --version $VERSION --use-existing-bins"
  exit 1
fi

DEB_SIZE=$(du -sh "$DEB_FILE" | cut -f1)
echo "  Archivo: $DEB_FILE ($DEB_SIZE)"

# ── PASO 2: Subir .deb al servidor ───────────────────────────────────
echo ""
echo "▶ [2/4] Subiendo .deb al servidor..."

_ssh "mkdir -p /opt/apt-mentor-edge/pool"
_scp "$DEB_FILE" "$SERVER_USER@$SERVER_IP:/opt/apt-mentor-edge/pool/mentor-edge_${VERSION}_arm64.deb"
echo "  ✓ .deb subido a /opt/apt-mentor-edge/pool/"

# ── PASO 3: Generar metadatos del repo apt en el servidor ─────────────
echo ""
echo "▶ [3/4] Generando metadatos del repositorio apt..."

_ssh bash -s << 'REMOTE'
set -euo pipefail
REPO_DIR="/opt/apt-mentor-edge"
POOL_DIR="$REPO_DIR/pool"

# Instalar herramientas si no están
if ! command -v dpkg-scanpackages &>/dev/null; then
  apt-get install -y --no-install-recommends dpkg-dev 2>/dev/null || true
fi

# Generar Packages + Packages.gz (índice del repo)
cd "$REPO_DIR"
dpkg-scanpackages --multiversion pool/ > Packages 2>/dev/null
gzip -9 -k -f Packages

# Generar Release (metadatos del repo)
PKGS_MD5=$(md5sum Packages | awk '{print $1}')
PKGS_SHA=$(sha256sum Packages | awk '{print $1}')
PKGS_GZ_MD5=$(md5sum Packages.gz | awk '{print $1}')
PKGS_GZ_SHA=$(sha256sum Packages.gz | awk '{print $1}')
PKGS_SIZE=$(stat -c%s Packages)
PKGS_GZ_SIZE=$(stat -c%s Packages.gz)

cat > Release <<RELEASEEOF
Origin: Mentor Edge
Label: Mentor Edge
Suite: stable
Codename: stable
Architectures: arm64
Components: main
Description: Repositorio privado de Mentor Edge
Date: $(date -u '+%a, %d %b %Y %H:%M:%S UTC')
MD5Sum:
 $PKGS_MD5 $PKGS_SIZE Packages
 $PKGS_GZ_MD5 $PKGS_GZ_SIZE Packages.gz
SHA256:
 $PKGS_SHA $PKGS_SIZE Packages
 $PKGS_GZ_SHA $PKGS_GZ_SIZE Packages.gz
RELEASEEOF

echo "  Packages:    $(wc -l < Packages) líneas"
echo "  Release:     generado"

# Generar install.sh — one-liner para configurar el repo y opcionalmente instalar
cat > "$REPO_DIR/install.sh" <<INSTALLEOF
#!/bin/bash
# Mentor Edge — Instalador de un click para Jetson
# Uso: curl -fsSL http://$(curl -s ifconfig.me 2>/dev/null || echo "SERVER_IP"):APT_PORT/install.sh | sudo bash
set -euo pipefail

APT_REPO_URL="\${MENTOR_EDGE_REPO:-http://$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}'):APT_PORT}"

echo "Configurando repositorio Mentor Edge..."
echo "deb [trusted=yes arch=arm64] \$APT_REPO_URL ./" \
  > /etc/apt/sources.list.d/mentor-edge.list

apt-get update -o Dir::Etc::sourcelist="sources.list.d/mentor-edge.list" \
               -o Dir::Etc::sourceparts="-" \
               -o APT::Get::List-Cleanup="0" 2>/dev/null

echo ""
echo "✓ Repositorio configurado. Para instalar/actualizar:"
echo "  sudo apt-get install -y mentor-edge"
echo ""
echo "  O instalar ahora: apt-get install -y mentor-edge"
INSTALLEOF
chmod +x "$REPO_DIR/install.sh"
REMOTE

# Inyectar la IP y puerto reales en install.sh (el heredoc los dejó como placeholders)
_ssh bash -s << REMOTE
  sed -i "s|APT_PORT|${APT_PORT}|g" /opt/apt-mentor-edge/install.sh
REMOTE

echo "  ✓ Packages, Packages.gz y Release generados"
echo "  ✓ install.sh generado"

# ── PASO 4: Levantar/actualizar contenedor nginx ──────────────────────
echo ""
echo "▶ [4/4] Actualizando contenedor nginx del repo apt..."

_ssh bash -s << REMOTE
set -euo pipefail

# Si ya existe un contenedor apt-mentor-edge, solo hacer reload
if docker ps -a --format '{{.Names}}' | grep -q '^apt-mentor-edge$'; then
  echo "  Recargando nginx existente..."
  docker exec apt-mentor-edge nginx -s reload 2>/dev/null || \
    docker restart apt-mentor-edge
  echo "  ✓ nginx recargado"
else
  echo "  Creando contenedor apt-mentor-edge (nginx)..."
  docker run -d \
    --name apt-mentor-edge \
    --restart unless-stopped \
    -p ${APT_PORT}:80 \
    -v /opt/apt-mentor-edge:/usr/share/nginx/html:ro \
    nginx:stable-alpine
  echo "  ✓ Contenedor creado en puerto ${APT_PORT}"
fi
REMOTE

# ── Resumen ───────────────────────────────────────────────────────────
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║  ✅  Repositorio apt publicado                            ║"
echo "╠═══════════════════════════════════════════════════════════╣"
echo ""
echo "  En cada Jetson nuevo, ejecutar UNA VEZ:"
echo ""
echo "  curl -fsSL http://${SERVER_IP}:${APT_PORT}/install.sh | sudo bash"
echo ""
echo "  Luego instalar / actualizar:"
echo ""
echo "  sudo apt-get install -y mentor-edge"
echo "  sudo apt-get upgrade    mentor-edge   # en próximas versiones"
echo ""
echo "╠═══════════════════════════════════════════════════════════╣"
echo "  También descarga directa sin repo apt:"
echo "  wget http://${SERVER_IP}:${APT_PORT}/pool/mentor-edge_${VERSION}_arm64.deb"
echo "  sudo apt install ./mentor-edge_${VERSION}_arm64.deb"
echo "╚═══════════════════════════════════════════════════════════╝"
