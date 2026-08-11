#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
COMPOSE_DIR="$PROJECT_ROOT/infrastructure/docker"

if [ ! -f "$COMPOSE_DIR/.env" ]; then
    echo "Archivo .env no encontrado. Copiando .env.example..."
    cp "$COMPOSE_DIR/.env.example" "$COMPOSE_DIR/.env"
    echo "Editar $COMPOSE_DIR/.env con valores de produccion antes de continuar."
    exit 1
fi

echo "Construyendo imagenes..."
docker compose -f "$COMPOSE_DIR/docker-compose.cloud.yml" --env-file "$COMPOSE_DIR/.env" build --no-cache

echo "Aplicando migraciones textile-only..."
bash "$PROJECT_ROOT/deploy/apply-textile-migrations.sh"

echo "Levantando servicios..."
docker compose -f "$COMPOSE_DIR/docker-compose.cloud.yml" --env-file "$COMPOSE_DIR/.env" up -d

echo "Esperando que los servicios esten saludables..."
sleep 10

SERVICES=("cloud-identity:8081" "cloud-ingest:8082" "cloud-config:8083" "cloud-analytics:8084" "cloud-integration:8085")
ALL_OK=true

for svc in "${SERVICES[@]}"; do
    NAME="${svc%%:*}"
    PORT="${svc##*:}"
    HTTP_CODE=$(docker compose -f "$COMPOSE_DIR/docker-compose.cloud.yml" exec -T "$NAME" wget -qO- -S "http://localhost:$PORT/health" 2>&1 | grep -c '"ok"' || true)
    if [ "$HTTP_CODE" -ge 1 ]; then
        echo "  $NAME: OK"
    else
        echo "  $NAME: SIN RESPUESTA"
        ALL_OK=false
    fi
done

if [ "$ALL_OK" = true ]; then
    echo "Todos los servicios operativos."
else
    echo "Algunos servicios no responden. Revisar logs:"
    echo "  docker compose -f $COMPOSE_DIR/docker-compose.cloud.yml logs --tail=50"
fi
