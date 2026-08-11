#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

usage() {
  cat <<'EOF'
Uso:
  ./detector-image.sh build
  ./detector-image.sh deploy STATE_FILE
  ./detector-image.sh rollback STATE_FILE
  ./detector-image.sh status STATE_FILE

Variables opcionales:
  COMPOSE_FILE          Ruta al docker-compose.jetson-orin.yml.
  SERVICE               Servicio Compose (vision-event-detector).
  CANDIDATE_REPOSITORY  Repositorio local para la imagen candidata.
  STATE_DIR             Directorio donde guardar el manifiesto de rollback.
  HEALTH_TIMEOUT_S      Espera máxima del detector nuevo (por defecto 240).
EOF
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "falta el comando requerido: $1"
}

absolute_existing_file() {
  local path="$1"
  test -f "$path" || die "no existe el archivo: $path"
  readlink -f "$path"
}

compose_container_id() {
  docker compose -f "$COMPOSE_FILE" ps --all -q "$SERVICE"
}

write_state() {
  local file="$1"
  {
    printf 'STATE_VERSION=%q\n' "1"
    printf 'STAMP=%q\n' "$STAMP"
    printf 'COMPOSE_FILE=%q\n' "$COMPOSE_FILE"
    printf 'SERVICE=%q\n' "$SERVICE"
    printf 'CONTAINER_NAME=%q\n' "$CONTAINER_NAME"
    printf 'TARGET_REF=%q\n' "$TARGET_REF"
    printf 'OLD_ID=%q\n' "$OLD_ID"
    printf 'OLD_ROLLBACK_REF=%q\n' "$OLD_ROLLBACK_REF"
    printf 'BASE_REF=%q\n' "$BASE_REF"
    printf 'CANDIDATE_REF=%q\n' "$CANDIDATE_REF"
    printf 'CANDIDATE_ID=%q\n' "$CANDIDATE_ID"
    printf 'SOURCE_MANIFEST_SHA256=%q\n' "$SOURCE_MANIFEST_SHA256"
  } > "$file"
  chmod 0600 "$file"
}

load_state() {
  local file
  file="$(absolute_existing_file "$1")"

  # This is a local, mode-0600 file produced by write_state().
  # shellcheck disable=SC1090
  source "$file"

  test "${STATE_VERSION:-}" = "1" || die "versión de estado no soportada"
  for name in COMPOSE_FILE SERVICE CONTAINER_NAME TARGET_REF OLD_ID \
    OLD_ROLLBACK_REF BASE_REF CANDIDATE_REF CANDIDATE_ID \
    SOURCE_MANIFEST_SHA256; do
    test -n "${!name:-}" || die "estado incompleto: falta $name"
  done

  COMPOSE_FILE="$(absolute_existing_file "$COMPOSE_FILE")"
  STATE_FILE="$file"
}

verify_bundle() {
  test -f "$BUNDLE_DIR/SHA256SUMS" ||
    die "falta SHA256SUMS en $BUNDLE_DIR"
  (
    cd "$BUNDLE_DIR"
    sha256sum --strict -c SHA256SUMS
  )
}

verify_linux_arm64_image() {
  local image="$1"
  local os arch
  os="$(docker image inspect "$image" --format '{{.Os}}')"
  arch="$(docker image inspect "$image" --format '{{.Architecture}}')"
  test "$os" = "linux" || die "$image no es Linux: $os"
  test "$arch" = "arm64" || die "$image no es ARM64: $arch"
}

restore_old_stopped() {
  set +e
  printf '\nRestaurando %s y dejándolo DETENIDO...\n' "$OLD_ROLLBACK_REF" >&2
  docker compose -f "$COMPOSE_FILE" stop "$SERVICE" >/dev/null 2>&1
  docker image tag "$OLD_ID" "$TARGET_REF"
  docker compose -f "$COMPOSE_FILE" up \
    --no-start --no-build --no-deps --force-recreate "$SERVICE"
  local rc=$?
  local restored_id
  restored_id="$(compose_container_id)"
  if test -n "$restored_id"; then
    docker stop "$restored_id" >/dev/null 2>&1
  fi
  if test "$rc" -eq 0; then
    printf 'ROLLBACK_OK: imagen anterior restaurada; detector detenido.\n' >&2
  else
    printf 'ROLLBACK_INCOMPLETO: revisar manualmente %s.\n' "$STATE_FILE" >&2
  fi
  return "$rc"
}

wait_healthy() {
  local deadline status running cid
  deadline=$((SECONDS + HEALTH_TIMEOUT_S))

  while (( SECONDS < deadline )); do
    cid="$(compose_container_id)"
    if test -z "$cid"; then
      sleep 2
      continue
    fi

    running="$(docker inspect "$cid" --format '{{.State.Running}}')"
    if test "$running" != "true"; then
      docker logs --tail 80 "$cid" >&2 || true
      return 1
    fi

    status="$(
      docker inspect "$cid" \
        --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
    )"
    if test "$status" = "healthy" &&
       curl -fsS http://127.0.0.1:8001/health >/dev/null; then
      return 0
    fi
    sleep 2
  done

  cid="$(compose_container_id)"
  test -z "$cid" || docker logs --tail 120 "$cid" >&2 || true
  return 1
}

build_candidate() {
  verify_bundle

  local cid old_os old_arch manifest_hash build_utc
  cid="$(compose_container_id)"
  test -n "$cid" || die "Compose no conoce el contenedor de $SERVICE"

  CONTAINER_NAME="$(docker inspect "$cid" --format '{{.Name}}')"
  CONTAINER_NAME="${CONTAINER_NAME#/}"
  TARGET_REF="$(docker inspect "$cid" --format '{{.Config.Image}}')"
  OLD_ID="$(docker inspect "$cid" --format '{{.Image}}')"
  test -n "$TARGET_REF" || die "no se pudo determinar la imagen de Compose"
  case "$TARGET_REF" in
    sha256:*|*@*) die "la referencia Compose no se puede reetiquetar: $TARGET_REF" ;;
  esac

  old_os="$(docker image inspect "$OLD_ID" --format '{{.Os}}')"
  old_arch="$(docker image inspect "$OLD_ID" --format '{{.Architecture}}')"
  test "$old_os" = "linux" || die "la imagen actual no es Linux"
  test "$old_arch" = "arm64" || die "la imagen actual no es ARM64"

  STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
  OLD_ROLLBACK_REF="${CANDIDATE_REPOSITORY}:pre-v5-${STAMP}"
  BASE_REF="${CANDIDATE_REPOSITORY}:base-${STAMP}"
  CANDIDATE_REF="${CANDIDATE_REPOSITORY}:v5-arm64-${STAMP}"
  manifest_hash="$(sha256sum "$BUNDLE_DIR/SHA256SUMS" | awk '{print $1}')"
  SOURCE_MANIFEST_SHA256="$manifest_hash"
  build_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  mkdir -p "$STATE_DIR"
  STATE_FILE="$STATE_DIR/detector-v5-${STAMP}.state"
  test ! -e "$STATE_FILE" || die "ya existe el estado: $STATE_FILE"

  docker image tag "$OLD_ID" "$OLD_ROLLBACK_REF"
  docker image tag "$OLD_ID" "$BASE_REF"

  printf 'Base congelada: %s (%s)\n' "$BASE_REF" "$OLD_ID"
  printf 'Construyendo sin red: %s\n' "$CANDIDATE_REF"

  DOCKER_BUILDKIT=1 docker build \
    --pull=false \
    --network=none \
    --build-arg "BASE_IMAGE=$BASE_REF" \
    --build-arg "SOURCE_MANIFEST_SHA256=$SOURCE_MANIFEST_SHA256" \
    --build-arg "BUILD_UTC=$build_utc" \
    --tag "$CANDIDATE_REF" \
    --file "$BUNDLE_DIR/Dockerfile" \
    "$BUNDLE_DIR"

  verify_linux_arm64_image "$CANDIDATE_REF"
  CANDIDATE_ID="$(docker image inspect "$CANDIDATE_REF" --format '{{.Id}}')"
  write_state "$STATE_FILE"

  printf '\nBUILD_OK\n'
  printf 'CANDIDATE=%s\n' "$CANDIDATE_REF"
  printf 'CANDIDATE_ID=%s\n' "$CANDIDATE_ID"
  printf 'ROLLBACK=%s\n' "$OLD_ROLLBACK_REF"
  printf 'STATE_FILE=%s\n' "$STATE_FILE"
  printf 'El contenedor no fue modificado ni iniciado.\n'
}

deploy_candidate() {
  load_state "$1"
  verify_linux_arm64_image "$CANDIDATE_ID"

  local cid current_id running
  cid="$(compose_container_id)"
  test -n "$cid" || die "no existe el contenedor Compose de $SERVICE"
  current_id="$(docker inspect "$cid" --format '{{.Image}}')"
  running="$(docker inspect "$cid" --format '{{.State.Running}}')"

  test "$current_id" = "$OLD_ID" ||
    die "el contenedor cambió desde el build ($current_id != $OLD_ID)"
  test "$running" = "false" ||
    die "detén $SERVICE antes del corte; deploy no lo detendrá automáticamente"

  on_deploy_error() {
    local rc=$?
    trap - ERR
    printf 'El candidato no quedó sano (rc=%s).\n' "$rc" >&2
    restore_old_stopped || true
    exit "$rc"
  }
  trap on_deploy_error ERR

  docker image tag "$CANDIDATE_ID" "$TARGET_REF"
  docker compose -f "$COMPOSE_FILE" up -d \
    --no-build --no-deps --force-recreate "$SERVICE"

  cid="$(compose_container_id)"
  test -n "$cid"
  current_id="$(docker inspect "$cid" --format '{{.Image}}')"
  test "$current_id" = "$CANDIDATE_ID"
  wait_healthy

  trap - ERR
  printf '\nDEPLOY_OK\n'
  printf 'IMAGE_ID=%s\n' "$CANDIDATE_ID"
  printf 'CONTAINER=%s\n' "$CONTAINER_NAME"
  printf 'HEALTH=%s\n' "$(
    docker inspect "$cid" \
      --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
  )"
  printf 'ROLLBACK_STATE=%s\n' "$STATE_FILE"
}

rollback_candidate() {
  load_state "$1"
  restore_old_stopped
}

show_status() {
  load_state "$1"
  local cid
  cid="$(compose_container_id)"
  printf 'target_ref=%s\n' "$TARGET_REF"
  printf 'old_id=%s\n' "$OLD_ID"
  printf 'candidate_id=%s\n' "$CANDIDATE_ID"
  if test -n "$cid"; then
    docker inspect "$cid" \
      --format 'container={{.Name}} image_id={{.Image}} running={{.State.Running}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}'
  else
    printf 'container=missing\n'
  fi
}

need docker
need sha256sum
need readlink
need curl

BUNDLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-/home/orin/Mentoredge/mentor-edge/infrastructure/docker/docker-compose.jetson-orin.yml}"
SERVICE="${SERVICE:-vision-event-detector}"
CANDIDATE_REPOSITORY="${CANDIDATE_REPOSITORY:-mentor-vision-event-detector}"
STATE_DIR="${STATE_DIR:-$BUNDLE_DIR/deploy-state}"
HEALTH_TIMEOUT_S="${HEALTH_TIMEOUT_S:-240}"

case "${1:-}" in
  build)
    COMPOSE_FILE="$(absolute_existing_file "$COMPOSE_FILE")"
    build_candidate
    ;;
  deploy)
    test -n "${2:-}" || die "deploy requiere STATE_FILE"
    deploy_candidate "$2"
    ;;
  rollback)
    test -n "${2:-}" || die "rollback requiere STATE_FILE"
    rollback_candidate "$2"
    ;;
  status)
    test -n "${2:-}" || die "status requiere STATE_FILE"
    show_status "$2"
    ;;
  *)
    usage
    exit 2
    ;;
esac
