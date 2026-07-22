#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────
# setup-deploy-key.sh
# Ejecutar UNA SOLA VEZ en tu máquina local para configurar CI/CD.
# Genera una clave SSH, la instala en el servidor, y muestra los
# valores que debes poner en GitHub Secrets.
# ─────────────────────────────────────────────────────────────────────
set -euo pipefail

SERVER_HOST="152.53.253.59"
SERVER_USER="asotoc"
SERVER_PASS="IpKmrJ0GXRgHe1ol"
KEY_FILE="$HOME/.ssh/mentor_cloud_deploy"

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║   Mentor Edge — Setup CI/CD Deploy Key   ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# 1. Generar par de claves si no existe
if [ ! -f "$KEY_FILE" ]; then
  echo "▶ Generando clave SSH..."
  ssh-keygen -t ed25519 -C "github-actions-deploy" -f "$KEY_FILE" -N ""
  echo "  ✓ Clave generada en $KEY_FILE"
else
  echo "  ℹ Clave ya existe en $KEY_FILE, reutilizando."
fi

# 2. Instalar clave pública en el servidor
echo ""
echo "▶ Instalando clave pública en el servidor $SERVER_HOST..."
sshpass -p "$SERVER_PASS" ssh-copy-id \
  -i "${KEY_FILE}.pub" \
  -o StrictHostKeyChecking=no \
  "${SERVER_USER}@${SERVER_HOST}"
echo "  ✓ Clave pública instalada en el servidor"

# 3. Verificar acceso sin contraseña
echo ""
echo "▶ Verificando acceso SSH sin contraseña..."
if ssh -i "$KEY_FILE" -o StrictHostKeyChecking=no -o BatchMode=yes \
    "${SERVER_USER}@${SERVER_HOST}" "echo OK" 2>/dev/null; then
  echo "  ✓ Acceso SSH verificado correctamente"
else
  echo "  ✗ Error: no se pudo conectar sin contraseña"
  exit 1
fi

# 4. Mostrar los GitHub Secrets necesarios
echo ""
echo "══════════════════════════════════════════════════════"
echo "  GITHUB SECRETS — añadir en:"
echo "  https://github.com/alonsosss/Mentoredge/settings/secrets/actions"
echo "══════════════════════════════════════════════════════"
echo ""
echo "  CLOUD_HOST            →  $SERVER_HOST"
echo "  CLOUD_USER            →  $SERVER_USER"
echo "  CLOUD_SSH_PRIVATE_KEY →  (contenido del archivo de abajo)"
echo ""
echo "──── CLOUD_SSH_PRIVATE_KEY ────────────────────────────"
cat "$KEY_FILE"
echo "───────────────────────────────────────────────────────"
echo ""
echo "  ✓ Copia TODO el contenido entre las líneas (incluye"
echo "    -----BEGIN y -----END) como valor del secret."
echo ""
echo "══════════════════════════════════════════════════════"
echo "  LISTO. Cada 'git push' a main desplegará automáticamente."
echo "══════════════════════════════════════════════════════"
