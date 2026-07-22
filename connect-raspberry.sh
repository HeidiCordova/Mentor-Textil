#!/usr/bin/env bash
# Conectar via SSH a la Raspberry Pi (energy-sender + mc60-reader)
#
# Uso:
#   RPI_IP=192.168.0.119 ./connect-raspberry.sh
#   ./connect-raspberry.sh 192.168.0.119
#   ./connect-raspberry.sh user@192.168.0.119
set -euo pipefail

# Resolución de destino: argumento posicional > variable de entorno > error
if [[ -n "${1:-}" ]]; then
  TARGET="$1"
  shift
elif [[ -n "${RPI_IP:-}" ]]; then
  TARGET="${RPI_USER:-py}@${RPI_IP}"
else
  echo "Uso: RPI_IP=<ip> ./connect-raspberry.sh  o  ./connect-raspberry.sh [user@]<ip> [comando...]" >&2
  exit 1
fi

# Si el target no contiene '@', antepone el usuario por defecto
[[ "$TARGET" == *@* ]] || TARGET="${RPI_USER:-py}@${TARGET}"

echo "Conectando a ${TARGET}..."
ssh -o StrictHostKeyChecking=no "${TARGET}" "$@"
