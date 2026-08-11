#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLOUD_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$CLOUD_ROOT/infrastructure/docker"
COMPOSE_FILE="$COMPOSE_DIR/docker-compose.cloud.yml"
ENV_FILE="$COMPOSE_DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
    echo "Falta $ENV_FILE"
    exit 1
fi

compose=(
    docker compose
    -f "$COMPOSE_FILE"
    --env-file "$ENV_FILE"
)

"${compose[@]}" up -d postgres-cloud

ready=0
for _ in $(seq 1 30); do
    if "${compose[@]}" exec -T postgres-cloud sh -c \
        'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null 2>&1; then
        ready=1
        break
    fi
    sleep 2
done

if [[ "$ready" -ne 1 ]]; then
    echo "PostgreSQL cloud no quedo listo en 60 segundos."
    exit 1
fi

for migration in \
    "$CLOUD_ROOT/infrastructure/database/27_linea_mode.sql" \
    "$CLOUD_ROOT/infrastructure/database/30_textile_only_cleanup.sql"; do
    echo "Aplicando $(basename "$migration")..."
    "${compose[@]}" exec -T postgres-cloud sh -c \
        'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
        < "$migration"
done

echo "Migraciones textile-only aplicadas."
