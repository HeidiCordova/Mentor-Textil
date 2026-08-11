#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
POINTER="$SCRIPT_DIR/hardening-state.env"
NR_CONTAINER="mentor-nodered"
NR_ADMIN_URL="http://127.0.0.1:1880"
BROKER_IP="52.11.253.25"
BROKER_PORT="1883"
EXPECTED_FLOW_HASH="f65c87507232c7bdd4cc4aca7a0beec607b54cd60e704d65ff634569ff5e9e88"
LEGACY_SENDER_JS_HASH="3d9b929beaf2f3b195e008442938e7f05b76a78bb5893e1d13a91cd8fcb58a7b"
LEGACY_SENDER_HTML_HASH="d17cbaf2f926d3ba897827236298aad8b6076e6378575cfba2ba92da8e4c0e01"
LEGACY_PACKAGE_HASH="67afd41a991056692b91a77b5058a6eec73acdb15f3c4c89d9afd975194ed5ea"
HARDENED_SENDER_JS_HASH="f64477cd45698afff49e2209fa8cbc051aeb752a22680107a42fdafc6deb771a"
HARDENED_SENDER_HTML_HASH="5dddf9c7d0e39c9e78968653497bdb0c9d1dc285297cb40dcfc009d06aebd178"
MODULE_DIR="/data/node_modules/node-red-contrib-services-mentor/services"
STATE_ROOT="/data/mentor-runtime-hardening"
CONTEXT_BASE="/data/mentor-context"
BASE_SETTINGS="/data/settings.mentor-base.js"
BACKUP_ROOT="/home/orin/mentor-backups"
LOCK_FILE="/home/orin/.mentor-nodered-runtime-hardening.lock"
CONTEXT_TOOL="$SCRIPT_DIR/runtime-context.js"
CONTEXT_TEST="$SCRIPT_DIR/runtime-context.test.js"
ARTIFACT_TEST="$SCRIPT_DIR/artifact-contract.test.js"
SETTINGS_WRAPPER="$SCRIPT_DIR/settings-persistent-context.js"
SENDER_JS="$SCRIPT_DIR/payload/services/sender.js"
SENDER_HTML="$SCRIPT_DIR/payload/services/sender.html"
SENDER_TEST="$SCRIPT_DIR/payload/test/sender.test.js"
RELEASE_TOOL="$SCRIPT_DIR/release/sender-release-canary-v2.js"

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Uso:
  ./deploy-runtime-hardening.sh preflight
  ./deploy-runtime-hardening.sh prepare
  ./deploy-runtime-hardening.sh activate
  ./deploy-runtime-hardening.sh prove
  ./deploy-runtime-hardening.sh status
  ./deploy-runtime-hardening.sh abort-prepare
  ./deploy-runtime-hardening.sh rollback-sender

Orden normal:
  preflight -> prepare -> activate -> prove

Garantías:
  - nunca habilita Sender ni modifica flows.json;
  - exige los dos Sender desactivados y el firewall externo activo;
  - captura global/flow/node context antes del primer reinicio;
  - conserva T_* y L1_CONTEO_1 sin usar epoch ni conteos hardcodeados;
  - instala localfilesystem con flushInterval=5 s;
  - actualiza Sender en disco, pero la habilitación cloud es otra fase.

abort-prepare restaura settings y módulo si todavía no hubo reinicio.
rollback-sender restaura solo el módulo anterior y conserva contexto persistente.
USAGE
}

acquire_lock() {
  command -v flock >/dev/null 2>&1 ||
    die "falta flock"
  (
    umask 077
    touch "$LOCK_FILE"
  )
  exec 9<>"$LOCK_FILE"
  flock -n 9 ||
    die "otra acción de hardening Node-RED está en ejecución"
}

sha_file_in_container() {
  docker exec "$NR_CONTAINER" sha256sum "$1" |
    awk '{print $1}'
}

verify_bundle() {
  for file in \
    "$CONTEXT_TOOL" "$CONTEXT_TEST" "$ARTIFACT_TEST" "$SETTINGS_WRAPPER" \
    "$SENDER_JS" "$SENDER_HTML" "$SENDER_TEST" "$RELEASE_TOOL" \
    "$SCRIPT_DIR/SHA256SUMS"; do
    test -s "$file" || die "falta artefacto: $file"
  done
  (
    cd "$SCRIPT_DIR"
    sha256sum -c SHA256SUMS
  )
  test "$(sha256sum "$SENDER_JS" | awk '{print $1}')" = \
    "$HARDENED_SENDER_JS_HASH" ||
    die "hash interno sender.js no coincide"
  test "$(sha256sum "$SENDER_HTML" | awk '{print $1}')" = \
    "$HARDENED_SENDER_HTML_HASH" ||
    die "hash interno sender.html no coincide"
}

container_identity() {
  docker inspect "$NR_CONTAINER" \
    --format '{{.Id}}|{{.State.StartedAt}}|{{.RestartCount}}'
}

container_ip() {
  local ips
  mapfile -t ips < <(
    docker inspect "$NR_CONTAINER" \
      --format '{{range .NetworkSettings.Networks}}{{println .IPAddress}}{{end}}' |
    sed '/^$/d'
  )
  test "${#ips[@]}" -eq 1 ||
    die "Node-RED no tiene exactamente una IP"
  [[ "${ips[0]}" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] ||
    die "IP Node-RED inválida: ${ips[0]}"
  printf '%s\n' "${ips[0]}"
}

require_healthy() {
  test "$(docker inspect "$NR_CONTAINER" --format '{{.State.Running}}')" = "true" ||
    die "Node-RED no está corriendo"
  test "$(docker inspect "$NR_CONTAINER" --format '{{.State.Health.Status}}')" = "healthy" ||
    die "Node-RED no está healthy"
}

wait_healthy() {
  local health=""
  for _ in $(seq 1 90); do
    health="$(
      docker inspect "$NR_CONTAINER" \
        --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' \
        2>/dev/null || true
    )"
    test "$health" = "healthy" && return
    sleep 2
  done
  die "Node-RED no llegó a healthy; estado=$health"
}

require_flow_hash() {
  local actual temporary
  temporary="/tmp/mentor-runtime-flow-hash.$$"
  copy_tool_to_container "$temporary"
  actual="$(
    docker exec "$NR_CONTAINER" \
      node "$temporary" flow-hash \
        --url "$NR_ADMIN_URL" \
        --disk /data/flows.json
  )"
  docker exec "$NR_CONTAINER" rm -f -- "$temporary"
  test "$actual" = "$EXPECTED_FLOW_HASH" ||
    die "hash canónico de flows cambió: esperado=$EXPECTED_FLOW_HASH actual=$actual"
}

discover_firewall() {
  NR_IP="$(container_ip)"
  local marker_lines
  marker_lines="$(
    sudo iptables -S DOCKER-USER |
      grep -F -- "-s $NR_IP/32" |
      grep -F -- "-d $BROKER_IP/32" |
      grep -F -- "--dport $BROKER_PORT" |
      grep -F -- "--comment mentor-textile-v4-" |
      grep -F -- "-j REJECT" || true
  )"
  test "$(printf '%s\n' "$marker_lines" | sed '/^$/d' | wc -l)" -eq 1 ||
    die "no existe exactamente una regla de aislamiento pilot para Node-RED"
  RULE_COMMENT="$(
    printf '%s\n' "$marker_lines" |
      sed -n 's/.*--comment \([^ ]*\).*/\1/p'
  )"
  [[ "$RULE_COMMENT" =~ ^mentor-textile-v4-[0-9]{8}T[0-9]{6}Z$ ]] ||
    die "comentario firewall inesperado: $RULE_COMMENT"
}

firewall_rule_array() {
  FIREWALL_RULE=(
    -s "$NR_IP/32"
    -d "$BROKER_IP/32"
    -p tcp
    --dport "$BROKER_PORT"
    -m comment
    --comment "$RULE_COMMENT"
    -j REJECT
    --reject-with tcp-reset
  )
}

firewall_packets() {
  sudo iptables -vxnL DOCKER-USER |
    awk -v marker="$RULE_COMMENT" '
      index($0, marker) { print $1; found++ }
      END { if (found != 1) exit 1 }
    '
}

require_firewall() {
  test "$(container_ip)" = "$NR_IP" ||
    die "IP Node-RED cambió; no se puede garantizar aislamiento"
  firewall_rule_array
  sudo iptables -C DOCKER-USER "${FIREWALL_RULE[@]}" ||
    die "regla firewall de aislamiento ausente"

  timeout 5 bash -c "</dev/tcp/$BROKER_IP/$BROKER_PORT" ||
    die "el host no alcanza el broker; no se puede distinguir firewall de caída de red"

  local before after
  before="$(firewall_packets)"
  if ! docker exec "$NR_CONTAINER" node -e "
    const net=require('net');
    const socket=net.createConnection({
      host:'$BROKER_IP',
      port:$BROKER_PORT
    });
    socket.setTimeout(3000,()=>socket.destroy(Error('timeout')));
    socket.on('connect',()=>process.exit(42));
    socket.on('error',()=>process.exit(0));
  "; then
    die "el contenedor conectó al broker o la prueba falló"
  fi
  after="$(firewall_packets)"
  test "$after" -gt "$before" ||
    die "la conexión bloqueada no atravesó la regla esperada"
}

copy_tool_to_container() {
  local target="$1"
  docker cp "$CONTEXT_TOOL" "$NR_CONTAINER:$target"
}

run_isolation_audit() {
  local temporary="/tmp/mentor-runtime-context-audit.$$"
  copy_tool_to_container "$temporary"
  docker exec "$NR_CONTAINER" \
    node "$temporary" audit --url "$NR_ADMIN_URL"
  docker exec "$NR_CONTAINER" rm -f -- "$temporary"
}

run_candidate_tests() {
  local temporary="/tmp/mentor-runtime-hardening-tests.$$"
  docker exec "$NR_CONTAINER" \
    mkdir -p \
      "$temporary/payload/services" \
      "$temporary/payload/test" \
      "$temporary/release"
  docker cp "$CONTEXT_TOOL" "$NR_CONTAINER:$temporary/runtime-context.js"
  docker cp "$CONTEXT_TEST" "$NR_CONTAINER:$temporary/runtime-context.test.js"
  docker cp "$ARTIFACT_TEST" "$NR_CONTAINER:$temporary/artifact-contract.test.js"
  docker cp "$SCRIPT_DIR/deploy-runtime-hardening.sh" \
    "$NR_CONTAINER:$temporary/deploy-runtime-hardening.sh"
  docker cp "$SENDER_JS" "$NR_CONTAINER:$temporary/payload/services/sender.js"
  docker cp "$SENDER_HTML" "$NR_CONTAINER:$temporary/payload/services/sender.html"
  docker cp "$SENDER_TEST" "$NR_CONTAINER:$temporary/payload/test/sender.test.js"
  docker cp "$SETTINGS_WRAPPER" \
    "$NR_CONTAINER:$temporary/settings-persistent-context.js"
  docker cp "$RELEASE_TOOL" \
    "$NR_CONTAINER:$temporary/release/sender-release-canary-v2.js"
  docker exec "$NR_CONTAINER" \
    node --check "$temporary/runtime-context.js"
  docker exec "$NR_CONTAINER" \
    node --check "$temporary/payload/services/sender.js"
  docker exec "$NR_CONTAINER" \
    node --check "$temporary/settings-persistent-context.js"
  docker exec "$NR_CONTAINER" \
    node --check "$temporary/release/sender-release-canary-v2.js"
  docker exec "$NR_CONTAINER" \
    node --test \
      "$temporary/runtime-context.test.js" \
      "$temporary/artifact-contract.test.js" \
      "$temporary/payload/test/sender.test.js"
  docker exec "$NR_CONTAINER" rm -rf -- "$temporary"
}

require_unconfigured_context() {
  docker exec "$NR_CONTAINER" node -e '
    const settings=require("/data/settings.js");
    if (
      Object.prototype.hasOwnProperty.call(settings,"contextStorage") &&
      settings.contextStorage !== undefined
    ) {
      console.error("contextStorage ya está configurado");
      process.exit(2);
    }
  ' || die "settings actual ya define contextStorage"
  test ! -e "$POINTER" ||
    die "ya existe $POINTER"
  docker exec "$NR_CONTAINER" test ! -e "$BASE_SETTINGS" ||
    die "$BASE_SETTINGS ya existe"
  docker exec "$NR_CONTAINER" test ! -e "$CONTEXT_BASE" ||
    die "$CONTEXT_BASE ya existe"
}

require_legacy_module() {
  local actual_js actual_html actual_package
  actual_js="$(sha_file_in_container "$MODULE_DIR/sender.js")"
  actual_html="$(sha_file_in_container "$MODULE_DIR/sender.html")"
  actual_package="$(
    sha_file_in_container \
      /data/node_modules/node-red-contrib-services-mentor/package.json
  )"
  test "$actual_js" = "$LEGACY_SENDER_JS_HASH" ||
    die "sender.js instalado no coincide con legacy auditado: $actual_js"
  test "$actual_html" = "$LEGACY_SENDER_HTML_HASH" ||
    die "sender.html instalado no coincide con legacy auditado: $actual_html"
  test "$actual_package" = "$LEGACY_PACKAGE_HASH" ||
    die "package.json instalado no coincide con 0.0.42 auditado: $actual_package"
}

preflight_common() {
  verify_bundle
  require_healthy
  require_flow_hash
  run_isolation_audit
  discover_firewall
  require_firewall
}

preflight_action() {
  preflight_common
  require_unconfigured_context
  require_legacy_module
  run_candidate_tests
  printf '\nPREFLIGHT_OK\nFLOW_HASH=%s\nFIREWALL=%s\n' \
    "$EXPECTED_FLOW_HASH" "$RULE_COMMENT"
  printf 'Sender continúa desactivado; no se modificaron flows, settings ni módulos.\n'
}

pointer_value() {
  local key="$1"
  awk -F= -v wanted="$key" '
    $1 == wanted {
      sub(/^[^=]*=/, "")
      print
      found=1
    }
    END { if (!found) exit 1 }
  ' "$POINTER"
}

write_pointer() {
  local temporary="$POINTER.tmp.$$"
  (
    umask 077
    {
      printf 'STATE_NAME=%s\n' "$STATE_NAME"
      printf 'STATE_DIR=%s\n' "$STATE_DIR"
      printf 'BACKUP=%s\n' "$BACKUP"
      printf 'PHASE=%s\n' "$PHASE"
      printf 'CONTAINER_ID=%s\n' "$CONTAINER_ID"
      printf 'STARTED_AT=%s\n' "$STARTED_AT"
      printf 'RESTARTS=%s\n' "$RESTARTS"
      printf 'NR_IP=%s\n' "$NR_IP"
      printf 'RULE_COMMENT=%s\n' "$RULE_COMMENT"
      printf 'ORIGINAL_SETTINGS_HASH=%s\n' "$ORIGINAL_SETTINGS_HASH"
      printf 'ORIGINAL_SENDER_HTML_HASH=%s\n' "$ORIGINAL_SENDER_HTML_HASH"
    } > "$temporary"
  )
  chmod 600 "$temporary"
  mv -- "$temporary" "$POINTER"
}

load_pointer() {
  test -s "$POINTER" || die "no existe $POINTER; ejecuta prepare"
  STATE_NAME="$(pointer_value STATE_NAME)"
  STATE_DIR="$(pointer_value STATE_DIR)"
  BACKUP="$(pointer_value BACKUP)"
  PHASE="$(pointer_value PHASE)"
  CONTAINER_ID="$(pointer_value CONTAINER_ID)"
  STARTED_AT="$(pointer_value STARTED_AT)"
  RESTARTS="$(pointer_value RESTARTS)"
  NR_IP="$(pointer_value NR_IP)"
  RULE_COMMENT="$(pointer_value RULE_COMMENT)"
  ORIGINAL_SETTINGS_HASH="$(pointer_value ORIGINAL_SETTINGS_HASH)"
  ORIGINAL_SENDER_HTML_HASH="$(pointer_value ORIGINAL_SENDER_HTML_HASH)"

  [[ "$STATE_NAME" =~ ^nodered-hardening-[0-9]{8}T[0-9]{6}Z$ ]] ||
    die "STATE_NAME inválido"
  test "$STATE_DIR" = "$STATE_ROOT/$STATE_NAME" ||
    die "STATE_DIR inválido"
  test "$BACKUP" = "$BACKUP_ROOT/$STATE_NAME" ||
    die "BACKUP inválido"
  [[ "$PHASE" =~ ^(prepared|activated|proved|aborted|sender_rolled_back)$ ]] ||
    die "PHASE inválida"
  [[ "$CONTAINER_ID" =~ ^[0-9a-f]{64}$ ]] ||
    die "CONTAINER_ID inválido"
  [[ "$RESTARTS" =~ ^[0-9]+$ ]] ||
    die "RESTARTS inválido"
  [[ "$NR_IP" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] ||
    die "NR_IP inválida"
  [[ "$RULE_COMMENT" =~ ^mentor-textile-v4-[0-9]{8}T[0-9]{6}Z$ ]] ||
    die "RULE_COMMENT inválido"
  [[ "$ORIGINAL_SETTINGS_HASH" =~ ^[0-9a-f]{64}$ ]] ||
    die "hash settings inválido"
  [[ "$ORIGINAL_SENDER_HTML_HASH" =~ ^[0-9a-f]{64}$ ]] ||
    die "hash sender.html legacy inválido"
}

require_identity() {
  local actual expected
  actual="$(container_identity)"
  expected="$CONTAINER_ID|$STARTED_AT|$RESTARTS"
  test "$actual" = "$expected" ||
    die "identidad Node-RED cambió: esperado=$expected actual=$actual"
}

backup_data_file() {
  local relative="$1"
  if docker exec "$NR_CONTAINER" test -e "/data/$relative"; then
    mkdir -p "$BACKUP/data/$(dirname "$relative")"
    docker cp "$NR_CONTAINER:/data/$relative" "$BACKUP/data/$relative"
  fi
}

write_backup_checksums() {
  (
    cd "$BACKUP"
    find . -type f ! -name SHA256SUMS -print0 |
      sort -z |
      xargs -0 sha256sum
  ) > "$BACKUP/SHA256SUMS"
}

copy_state_file() {
  local relative="$1"
  docker exec "$NR_CONTAINER" test -s "$STATE_DIR/$relative" ||
    die "estado runtime ausente: $relative"
  docker cp "$NR_CONTAINER:$STATE_DIR/$relative" "$BACKUP/$relative"
  chmod 600 "$BACKUP/$relative"
}

backup_context_seed() {
  local label="$1"
  local temporary="/tmp/$STATE_NAME.context.$label.tgz"
  docker exec "$NR_CONTAINER" \
    tar -C /data -czf "$temporary" mentor-context
  docker cp "$NR_CONTAINER:$temporary" \
    "$BACKUP/context-seed.$label.tgz"
  docker exec "$NR_CONTAINER" rm -f -- "$temporary"
  test -s "$BACKUP/context-seed.$label.tgz" ||
    die "respaldo de seed de contexto vacío"
  chmod 600 "$BACKUP/context-seed.$label.tgz"
}

capture_modbus() {
  local destination="$1"
  docker exec -i "$NR_CONTAINER" node > "$destination" <<'NODE'
const net = require("net");
const request = Buffer.alloc(12);
request.writeUInt16BE(0x4d54, 0);
request.writeUInt16BE(0, 2);
request.writeUInt16BE(6, 4);
request[6] = 1;
request[7] = 3;
request.writeUInt16BE(1, 8);
request.writeUInt16BE(9, 10);
let buffer = Buffer.alloc(0);
let finished = false;
const socket = net.createConnection(
  {host: "127.0.0.1", port: 10502},
  () => socket.write(request)
);
function fail(message) {
  if (finished) return;
  finished = true;
  socket.destroy();
  console.error(message instanceof Error ? message.stack : message);
  process.exitCode = 1;
}
socket.setTimeout(4000, () => fail("MODBUS_TIMEOUT"));
socket.on("error", fail);
socket.on("data", (chunk) => {
  buffer = Buffer.concat([buffer, chunk]);
  if (buffer.length < 6) return;
  const total = 6 + buffer.readUInt16BE(4);
  if (buffer.length < total || finished) return;
  if (
    buffer.readUInt16BE(0) !== 0x4d54 ||
    buffer.readUInt16BE(2) !== 0 ||
    buffer[7] !== 3 ||
    buffer[8] !== 18
  ) {
    fail(`MODBUS_RESPONSE_INVALID ${buffer.toString("hex")}`);
    return;
  }
  finished = true;
  const registers = [];
  for (let index = 0; index < 9; index += 1) {
    registers.push(buffer.readUInt16BE(9 + index * 2));
  }
  const uint32 = (index) =>
    registers[index] * 65536 + registers[index + 1];
  console.log(JSON.stringify({
    at: new Date().toISOString(),
    gpio: registers[0],
    disponible: uint32(1),
    microparada: uint32(3),
    pna: uint32(5),
    conteo: uint32(7),
    raw: registers
  }, null, 2));
  socket.end();
});
NODE
  python3 -m json.tool "$destination" >/dev/null
}

assert_modbus_continuity() {
  python3 - "$1" "$2" <<'PY'
import json
import sys

before = json.load(open(sys.argv[1]))
after = json.load(open(sys.argv[2]))
for key in ("disponible", "microparada", "pna", "conteo"):
    assert after[key] >= before[key], (key, before[key], after[key])
print(json.dumps({"before": before, "after": after}, indent=2))
print("MODBUS_CONTINUIDAD_OK")
PY
}

require_persistent_context_disk() {
  test "$(sha_file_in_container "$BASE_SETTINGS")" = \
    "$ORIGINAL_SETTINGS_HASH" ||
    die "settings base cambió"
  test "$(sha_file_in_container /data/settings.js)" = \
    "$(sha256sum "$SETTINGS_WRAPPER" | awk '{print $1}')" ||
    die "wrapper settings ausente"
  docker exec "$NR_CONTAINER" test -s "$CONTEXT_BASE/global/global.json" ||
    die "seed global de contexto ausente"
  docker exec "$NR_CONTAINER" node -e '
    const settings=require("/data/settings.js");
    const store=settings.contextStorage &&
      settings.contextStorage.default;
    const config=store && store.config;
    if (
      !store ||
      store.module !== "localfilesystem" ||
      !config ||
      config.dir !== "/data" ||
      config.base !== "mentor-context" ||
      config.cache !== true ||
      config.flushInterval !== 5
    ) {
      console.error(JSON.stringify(settings.contextStorage));
      process.exit(2);
    }
  ' || die "contextStorage persistente no coincide"
}

require_prepared_disk() {
  require_flow_hash
  test "$(sha_file_in_container "$MODULE_DIR/sender.js")" = \
    "$HARDENED_SENDER_JS_HASH" ||
    die "sender.js endurecido ausente"
  test "$(sha_file_in_container "$MODULE_DIR/sender.html")" = \
    "$HARDENED_SENDER_HTML_HASH" ||
    die "sender.html endurecido ausente"
  require_persistent_context_disk
}

prepare_action() {
  preflight_common
  require_unconfigured_context
  require_legacy_module
  run_candidate_tests

  local stamp
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  STATE_NAME="nodered-hardening-$stamp"
  STATE_DIR="$STATE_ROOT/$STATE_NAME"
  BACKUP="$BACKUP_ROOT/$STATE_NAME"
  PHASE="prepared"
  CONTAINER_ID="$(docker inspect "$NR_CONTAINER" --format '{{.Id}}')"
  STARTED_AT="$(docker inspect "$NR_CONTAINER" --format '{{.State.StartedAt}}')"
  RESTARTS="$(docker inspect "$NR_CONTAINER" --format '{{.RestartCount}}')"
  ORIGINAL_SETTINGS_HASH="$(sha_file_in_container /data/settings.js)"
  ORIGINAL_SENDER_HTML_HASH="$(sha_file_in_container "$MODULE_DIR/sender.html")"

  mkdir -- "$BACKUP"
  chmod 700 "$BACKUP"
  docker exec "$NR_CONTAINER" mkdir -p "$STATE_DIR"

  local mutation_started="false"
  local prepare_completed="false"
  rollback_failed_prepare() {
    local exit_status="$?"
    local current
    current="$(container_identity 2>/dev/null || true)"
    if test "$prepare_completed" != "true" &&
       test "$mutation_started" = "true" &&
       test "$current" = "$CONTAINER_ID|$STARTED_AT|$RESTARTS"; then
      docker cp "$BACKUP/data/settings.js" \
        "$NR_CONTAINER:/data/settings.js" >/dev/null 2>&1 || true
      docker cp "$BACKUP/data/node_modules/node-red-contrib-services-mentor/services/sender.js" \
        "$NR_CONTAINER:$MODULE_DIR/sender.js" >/dev/null 2>&1 || true
      docker cp "$BACKUP/data/node_modules/node-red-contrib-services-mentor/services/sender.html" \
        "$NR_CONTAINER:$MODULE_DIR/sender.html" >/dev/null 2>&1 || true
      docker exec "$NR_CONTAINER" rm -f -- \
        "$BASE_SETTINGS.$stamp.tmp" \
        "$MODULE_DIR/sender.js.$stamp.tmp" \
        "$MODULE_DIR/sender.html.$stamp.tmp" \
        "/data/settings.js.$stamp.tmp" >/dev/null 2>&1 || true
      docker exec "$NR_CONTAINER" rm -rf -- \
        "$BASE_SETTINGS" "$CONTEXT_BASE" "$STATE_DIR" >/dev/null 2>&1 || true
    fi
    return "$exit_status"
  }
  trap rollback_failed_prepare EXIT

  docker inspect "$NR_CONTAINER" > "$BACKUP/container.before.json"
  docker logs --tail 500 "$NR_CONTAINER" > "$BACKUP/logs.before.txt" 2>&1
  sudo iptables-save > "$BACKUP/iptables.before.rules"
  for relative in \
    settings.js \
    flows.json \
    flows_cred.json \
    .config.runtime.json \
    node_modules/node-red-contrib-services-mentor/package.json \
    node_modules/node-red-contrib-services-mentor/services/sender.js \
    node_modules/node-red-contrib-services-mentor/services/sender.html; do
    backup_data_file "$relative"
  done
  test -s "$BACKUP/data/settings.js" ||
    die "respaldo settings vacío"
  test -s "$BACKUP/data/node_modules/node-red-contrib-services-mentor/services/sender.js" ||
    die "respaldo sender.js vacío"

  copy_tool_to_container "$STATE_DIR/runtime-context.js"
  docker cp "$CONTEXT_TEST" "$NR_CONTAINER:$STATE_DIR/runtime-context.test.js"
  capture_modbus "$BACKUP/modbus.prepare.json"

  mutation_started="true"
  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/runtime-context.js" capture \
      --url "$NR_ADMIN_URL" \
      --output "$STATE_DIR/context.prepare.json" \
      --seed-base "$CONTEXT_BASE" |
    tee "$BACKUP/context.prepare.log"
  copy_state_file context.prepare.json
  backup_context_seed prepare

  docker exec "$NR_CONTAINER" \
    cp -- /data/settings.js "$BASE_SETTINGS.$stamp.tmp"
  docker exec "$NR_CONTAINER" \
    mv -- "$BASE_SETTINGS.$stamp.tmp" "$BASE_SETTINGS"
  docker cp "$SENDER_JS" \
    "$NR_CONTAINER:$MODULE_DIR/sender.js.$stamp.tmp"
  docker cp "$SENDER_HTML" \
    "$NR_CONTAINER:$MODULE_DIR/sender.html.$stamp.tmp"
  docker cp "$SETTINGS_WRAPPER" \
    "$NR_CONTAINER:/data/settings.js.$stamp.tmp"
  docker exec "$NR_CONTAINER" \
    mv -- "$MODULE_DIR/sender.js.$stamp.tmp" "$MODULE_DIR/sender.js"
  docker exec "$NR_CONTAINER" \
    mv -- "$MODULE_DIR/sender.html.$stamp.tmp" "$MODULE_DIR/sender.html"
  docker exec "$NR_CONTAINER" \
    mv -- "/data/settings.js.$stamp.tmp" /data/settings.js

  require_identity
  require_prepared_disk
  run_isolation_audit
  require_firewall
  write_backup_checksums
  write_pointer
  prepare_completed="true"
  trap - EXIT

  printf '\nPREPARE_OK\nSTATE=%s\nBACKUP=%s\n' "$STATE_NAME" "$BACKUP"
  printf 'No hubo reinicio. Sender sigue desactivado y bloqueado por firewall.\n'
  printf 'Siguiente paso controlado: ./deploy-runtime-hardening.sh activate\n'
}

validate_after_restart() {
  local baseline="$1"
  local label="$2"
  require_healthy
  require_flow_hash
  require_prepared_disk
  run_isolation_audit
  require_firewall
  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/runtime-context.js" validate \
      --url "$NR_ADMIN_URL" \
      --baseline "$baseline" \
      --output "$STATE_DIR/context.$label.after.json" |
    tee "$BACKUP/context.$label.validate.log"
  copy_state_file "context.$label.after.json"
  capture_modbus "$BACKUP/modbus.$label.after.json"
  assert_modbus_continuity \
    "$BACKUP/modbus.$label.before.json" \
    "$BACKUP/modbus.$label.after.json"
}

record_new_identity() {
  test "$(docker inspect "$NR_CONTAINER" --format '{{.Id}}')" = "$CONTAINER_ID" ||
    die "el contenedor fue recreado; solo se autorizó restart"
  test "$(container_ip)" = "$NR_IP" ||
    die "IP del contenedor cambió"
  STARTED_AT="$(docker inspect "$NR_CONTAINER" --format '{{.State.StartedAt}}')"
  RESTARTS="$(docker inspect "$NR_CONTAINER" --format '{{.RestartCount}}')"
}

activate_action() {
  load_pointer
  test "$PHASE" = "prepared" ||
    die "activate requiere PHASE=prepared; actual=$PHASE"
  require_identity
  require_healthy
  require_prepared_disk
  run_isolation_audit
  require_firewall

  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/runtime-context.js" capture \
      --url "$NR_ADMIN_URL" \
      --output "$STATE_DIR/context.activate.before.json" \
      --seed-base "$CONTEXT_BASE" \
      --replace |
    tee "$BACKUP/context.activate.capture.log"
  copy_state_file context.activate.before.json
  backup_context_seed activate
  capture_modbus "$BACKUP/modbus.activate.before.json"

  local restart_started
  restart_started="$(date -u --iso-8601=seconds)"
  docker restart -t 30 "$NR_CONTAINER" >/dev/null
  wait_healthy
  record_new_identity
  validate_after_restart "$STATE_DIR/context.activate.before.json" "activate"

  docker logs --since "$restart_started" "$NR_CONTAINER" \
    > "$BACKUP/logs.activate.txt" 2>&1
  if grep -Ei \
      'context.*(error|invalid|failed)|localfilesystem.*(error|invalid|failed)|uncaught exception|missing node module|failed to register' \
      "$BACKUP/logs.activate.txt"; then
    die "logs Node-RED contienen error de módulo/contexto"
  fi

  PHASE="activated"
  write_pointer
  write_backup_checksums
  printf '\nACTIVATE_OK\nSTATE=%s\nBACKUP=%s\n' "$STATE_NAME" "$BACKUP"
  printf 'Contexto persistente activo; Sender sigue desactivado y aislado.\n'
  printf 'Siguiente paso: ./deploy-runtime-hardening.sh prove\n'
}

prove_action() {
  load_pointer
  test "$PHASE" = "activated" ||
    die "prove requiere PHASE=activated; actual=$PHASE"
  require_identity
  require_healthy
  require_prepared_disk
  run_isolation_audit
  require_firewall

  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/runtime-context.js" capture \
      --url "$NR_ADMIN_URL" \
      --output "$STATE_DIR/context.prove.before.json" |
    tee "$BACKUP/context.prove.capture.log"
  copy_state_file context.prove.before.json
  capture_modbus "$BACKUP/modbus.prove.before.json"

  sleep 7
  local restart_started
  restart_started="$(date -u --iso-8601=seconds)"
  docker restart -t 30 "$NR_CONTAINER" >/dev/null
  wait_healthy
  record_new_identity
  validate_after_restart "$STATE_DIR/context.prove.before.json" "prove"

  docker logs --since "$restart_started" "$NR_CONTAINER" \
    > "$BACKUP/logs.prove.txt" 2>&1
  if grep -Ei \
      'context.*(error|invalid|failed)|localfilesystem.*(error|invalid|failed)|uncaught exception|missing node module|failed to register' \
      "$BACKUP/logs.prove.txt"; then
    die "logs Node-RED contienen error después de prueba de persistencia"
  fi

  PHASE="proved"
  write_pointer
  write_backup_checksums
  printf '\nPERSISTENCE_PROVED_OK\nSTATE=%s\nBACKUP=%s\n' \
    "$STATE_NAME" "$BACKUP"
  printf 'Dos reinicios validados. Sender continúa desactivado y aislado.\n'
}

status_action() {
  load_pointer
  printf 'PHASE=%s\n' "$PHASE"
  printf 'IDENTIDAD_ESPERADA=%s|%s|%s\n' \
    "$CONTAINER_ID" "$STARTED_AT" "$RESTARTS"
  printf 'IDENTIDAD_ACTUAL=%s\n' "$(container_identity)"
  if test "$PHASE" = "aborted"; then
    printf 'Preparación abortada; revisar BACKUP=%s\n' "$BACKUP"
    return
  fi
  require_identity
  require_healthy
  require_flow_hash
  run_isolation_audit
  require_firewall
  if test "$PHASE" = "sender_rolled_back"; then
    require_legacy_module
    require_persistent_context_disk
  else
    require_prepared_disk
  fi
  printf 'STATUS_OK\nBACKUP=%s\n' "$BACKUP"
}

restore_backup_file() {
  local relative="$1"
  test -s "$BACKUP/data/$relative" ||
    die "respaldo ausente: $relative"
  docker cp "$BACKUP/data/$relative" "$NR_CONTAINER:/data/$relative"
}

abort_prepare_action() {
  load_pointer
  test "$PHASE" = "prepared" ||
    die "abort-prepare solo admite PHASE=prepared"
  require_identity
  require_healthy
  require_firewall
  restore_backup_file settings.js
  restore_backup_file \
    node_modules/node-red-contrib-services-mentor/services/sender.js
  restore_backup_file \
    node_modules/node-red-contrib-services-mentor/services/sender.html
  docker exec "$NR_CONTAINER" rm -rf -- "$BASE_SETTINGS" "$CONTEXT_BASE"
  require_legacy_module
  test "$(sha_file_in_container /data/settings.js)" = "$ORIGINAL_SETTINGS_HASH" ||
    die "settings no se restauró"
  PHASE="aborted"
  write_pointer
  write_backup_checksums
  printf '\nABORT_PREPARE_OK\nBACKUP=%s\n' "$BACKUP"
  printf 'No hubo reinicio ni pérdida de contexto en memoria.\n'
}

rollback_sender_action() {
  load_pointer
  case "$PHASE" in
    activated|proved) ;;
    *) die "rollback-sender requiere PHASE=activated o proved" ;;
  esac
  require_identity
  require_healthy
  require_prepared_disk
  run_isolation_audit
  require_firewall

  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/runtime-context.js" capture \
      --url "$NR_ADMIN_URL" \
      --output "$STATE_DIR/context.rollback.before.json" |
    tee "$BACKUP/context.rollback.capture.log"
  copy_state_file context.rollback.before.json
  capture_modbus "$BACKUP/modbus.rollback.before.json"

  restore_backup_file \
    node_modules/node-red-contrib-services-mentor/services/sender.js
  restore_backup_file \
    node_modules/node-red-contrib-services-mentor/services/sender.html
  require_legacy_module

  docker restart -t 30 "$NR_CONTAINER" >/dev/null
  wait_healthy
  record_new_identity
  require_flow_hash
  run_isolation_audit
  require_firewall
  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/runtime-context.js" validate \
      --url "$NR_ADMIN_URL" \
      --baseline "$STATE_DIR/context.rollback.before.json" \
      --output "$STATE_DIR/context.rollback.after.json" |
    tee "$BACKUP/context.rollback.validate.log"
  copy_state_file context.rollback.after.json
  capture_modbus "$BACKUP/modbus.rollback.after.json"
  assert_modbus_continuity \
    "$BACKUP/modbus.rollback.before.json" \
    "$BACKUP/modbus.rollback.after.json"

  PHASE="sender_rolled_back"
  write_pointer
  write_backup_checksums
  printf '\nROLLBACK_SENDER_OK\nBACKUP=%s\n' "$BACKUP"
  printf 'Contexto persistente se conservó; Sender sigue desactivado y aislado.\n'
}

action="${1:-}"
case "$action" in
  preflight|prepare|activate|prove|status|abort-prepare|rollback-sender)
    acquire_lock
    ;;
  help|-h|--help|"")
    usage
    exit 0
    ;;
  *)
    usage >&2
    die "acción desconocida: $action"
    ;;
esac

case "$action" in
  preflight) preflight_action ;;
  prepare) prepare_action ;;
  activate) activate_action ;;
  prove) prove_action ;;
  status) status_action ;;
  abort-prepare) abort_prepare_action ;;
  rollback-sender) rollback_sender_action ;;
esac
