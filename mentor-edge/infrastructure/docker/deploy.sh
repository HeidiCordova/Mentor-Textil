#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_DIR="$SCRIPT_DIR"
COMPOSE_FILE="$COMPOSE_DIR/docker-compose.jetson-orin.yml"
CALIBRATION_MIGRATION="$PROJECT_ROOT/infrastructure/database/30_detector_calibration.sql"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_requirements() {
    local missing=0

    if ! command -v docker &> /dev/null; then
        log_error "docker no encontrado"
        missing=1
    fi

    if ! docker compose version &> /dev/null; then
        log_error "docker compose no encontrado"
        missing=1
    fi

    if [ "$(uname -m)" != "aarch64" ]; then
        log_warn "Arquitectura detectada: $(uname -m) - Este despliegue esta optimizado para Jetson Orin (aarch64)"
    fi

    if command -v nvidia-smi &> /dev/null; then
        log_info "GPU NVIDIA detectada"
        nvidia-smi --query-gpu=name,memory.total --format=csv,noheader 2>/dev/null || true
    else
        log_warn "nvidia-smi no disponible"
    fi

    if [ $missing -eq 1 ]; then
        log_error "Faltan dependencias"
        exit 1
    fi
}

setup_env() {
    if [ ! -f "$COMPOSE_DIR/.env" ]; then
        if [ -f "$COMPOSE_DIR/.env.example" ]; then
            cp "$COMPOSE_DIR/.env.example" "$COMPOSE_DIR/.env"
            log_warn "Archivo .env creado desde .env.example - Actualizar credenciales antes de continuar"
            exit 1
        else
            log_error "No se encontro .env.example"
            exit 1
        fi
    fi
}

cmd_migrate() {
    if [ ! -f "$CALIBRATION_MIGRATION" ]; then
        log_error "No se encontro la migracion: $CALIBRATION_MIGRATION"
        exit 1
    fi

    log_info "Aplicando migraciones idempotentes de PostgreSQL..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" up -d postgres

    local ready=0
    for _ in $(seq 1 30); do
        if docker compose -f "$COMPOSE_FILE" exec -T postgres \
            sh -c 'pg_isready --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"' \
            >/dev/null 2>&1; then
            ready=1
            break
        fi
        sleep 2
    done

    if [ "$ready" -ne 1 ]; then
        log_error "PostgreSQL no estuvo disponible para ejecutar migraciones"
        exit 1
    fi

    docker compose -f "$COMPOSE_FILE" exec -T postgres \
        sh -c 'psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"' \
        < "$CALIBRATION_MIGRATION"
    log_info "Migracion detector_calibration aplicada"
}

cmd_pull() {
    if ! git -C "$PROJECT_ROOT" rev-parse --is-inside-work-tree &>/dev/null; then
        log_error "$PROJECT_ROOT no es un repositorio git"
        log_warn "Ejecuta: git clone https://<TOKEN>@github.com/alonsosss/Mentoredge.git ~/mentor-edge"
        exit 1
    fi

    log_info "Actualizando desde origin/main..."
    git -C "$PROJECT_ROOT" fetch origin main

    local BEFORE
    BEFORE=$(git -C "$PROJECT_ROOT" rev-parse HEAD)

    git -C "$PROJECT_ROOT" merge --ff-only origin/main

    local AFTER
    AFTER=$(git -C "$PROJECT_ROOT" rev-parse HEAD)

    log_info "Codigo actualizado a $(git -C "$PROJECT_ROOT" log --oneline -1)"

    if [ "$BEFORE" = "$AFTER" ]; then
        log_info "Sin cambios nuevos. No se requiere rebuild."
        return 0
    fi

    # Detectar que servicios cambiaron para reconstruir solo los necesarios
    local CHANGED_SERVICES=()
    local CHANGED_FILES
    CHANGED_FILES=$(git -C "$PROJECT_ROOT" diff --name-only "$BEFORE" "$AFTER")

    if echo "$CHANGED_FILES" | grep -q "infrastructure/database/30_detector_calibration.sql"; then
        cmd_migrate
    fi

    for svc in ui-local edge-gateway enviador resiliencia edge-config-service vision-event-detector; do
        if echo "$CHANGED_FILES" | grep -q "services/$svc/"; then
            CHANGED_SERVICES+=("$svc")
        fi
    done

    if [ ${#CHANGED_SERVICES[@]} -eq 0 ]; then
        log_info "Solo cambiaron archivos de infraestructura/docs. Revisa manualmente si necesitas rebuild."
        return 0
    fi

    log_info "Servicios con cambios: ${CHANGED_SERVICES[*]}"
    log_info "Reconstruyendo imagenes afectadas..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" build --no-cache "${CHANGED_SERVICES[@]}"
    docker compose -f "$COMPOSE_FILE" up -d "${CHANGED_SERVICES[@]}"
    log_info "Servicios actualizados y reiniciados."
    cmd_status
}

cmd_deploy() {
    local services=("$@")
    log_info "=== DEPLOY COMPLETO ==="
    cmd_pull
    cmd_migrate
    if [ ${#services[@]} -eq 0 ]; then
        cmd_build
        cmd_up
    else
        cmd_build "${services[@]}"
        cd "$COMPOSE_DIR"
        docker compose -f "$COMPOSE_FILE" up -d "${services[@]}"
        log_info "Servicios actualizados: ${services[*]}"
        cmd_status
    fi
    log_info "=== DEPLOY COMPLETADO ==="
}

cmd_build() {
    log_info "Construyendo imagenes para $(uname -m)..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" build --no-cache "$@"
    log_info "Build completado"
}

cmd_tablet_build() {
    local TABLET_DIR="$PROJECT_ROOT/../mentor-apps/mentor-tablet-app"
    if [ ! -d "$TABLET_DIR" ]; then
        log_error "No se encontro mentor-tablet-app en $TABLET_DIR"
        exit 1
    fi
    log_info "Construyendo tablet app desde $TABLET_DIR..."
    cd "$TABLET_DIR"
    npm ci --prefer-offline 2>/dev/null || npm install
    npm run build
    export TABLET_DIST_PATH="$TABLET_DIR/dist"
    log_info "Tablet app lista en $TABLET_DIST_PATH"
}

cmd_up() {
    log_info "Iniciando servicios..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" up -d "$@"
    log_info "Servicios iniciados"
    cmd_status
}

cmd_down() {
    log_info "Deteniendo servicios..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" down "$@"
    log_info "Servicios detenidos"
}

cmd_status() {
    cd "$COMPOSE_DIR"
    echo ""
    docker compose -f "$COMPOSE_FILE" ps
    echo ""
}

cmd_logs() {
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" logs -f --tail=100 "$@"
}

cmd_restart() {
    log_info "Reiniciando servicios..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" restart "$@"
}

cmd_health() {
    echo ""
    log_info "Verificando salud de servicios..."
    echo ""

    local services=("resiliencia:8002" "enviador:8003" "edge-config-service:8004")

    for svc in "${services[@]}"; do
        IFS=':' read -r name port <<< "$svc"
        if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then
            echo -e "  ${GREEN}OK${NC}  $name (:$port)"
        else
            echo -e "  ${RED}FAIL${NC}  $name (:$port)"
        fi
    done

    if curl -sf "http://localhost:8080/" > /dev/null 2>&1; then
        echo -e "  ${GREEN}OK${NC}  ui-local (:8080)"
    else
        echo -e "  ${RED}FAIL${NC}  ui-local (:8080)"
    fi

    if curl -sf "http://localhost:8090/" > /dev/null 2>&1; then
        echo -e "  ${GREEN}OK${NC}  tablet-app (:8090)"
    else
        echo -e "  ${RED}FAIL${NC}  tablet-app (:8090)"
    fi

    echo ""
}

cmd_clean() {
    log_warn "Eliminando contenedores, imagenes y volumenes..."
    cd "$COMPOSE_DIR"
    docker compose -f "$COMPOSE_FILE" down -v --rmi local
    log_info "Limpieza completada"
}

usage() {
    echo "Mentor Edge Textil - Despliegue Jetson Orin"
    echo ""
    echo "Uso: $0 <comando> [opciones]"
    echo ""
    echo "Comandos:"
    echo "  pull          git pull + rebuild automatico de servicios con cambios"
    echo "  deploy        pull + build + up (rebuild forzado de todo)"
    echo "  build         Construir imagenes Docker (microservicios)"
    echo "  migrate       Aplicar migraciones PostgreSQL sin borrar pgdata"
    echo "  tablet-build  Compilar la tablet app (genera dist/)"
    echo "  up            Iniciar todos los servicios"
    echo "  down          Detener todos los servicios"
    echo "  restart       Reiniciar servicios"
    echo "  status        Estado de los servicios"
    echo "  logs          Ver logs (seguir)"
    echo "  health        Verificar salud de servicios"
    echo "  clean         Eliminar contenedores, imagenes y datos"
    echo ""
}

main() {
    check_requirements
    setup_env

    case "${1:-}" in
        pull)          cmd_pull ;;
        deploy)        shift; cmd_deploy "$@" ;;
        build)         shift; cmd_build "$@" ;;
        migrate)       cmd_migrate ;;
        tablet-build)  cmd_tablet_build ;;
        up)            shift; cmd_up "$@" ;;
        down)          shift; cmd_down "$@" ;;
        restart)       shift; cmd_restart "$@" ;;
        status)        cmd_status ;;
        logs)          shift; cmd_logs "$@" ;;
        health)        cmd_health ;;
        clean)         cmd_clean ;;
        *)             usage ;;
    esac
}

main "$@"
