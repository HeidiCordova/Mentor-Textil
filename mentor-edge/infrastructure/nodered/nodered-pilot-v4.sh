#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
POINTER="$SCRIPT_DIR/pilot-state.env"
NR_CONTAINER="mentor-nodered"
PG_CONTAINER="docker-postgres-1"
BROKER_IP="52.11.253.25"
BROKER_PORT="1883"
STATE_ROOT="/data/mentor-hot-deploy"
BACKUP_ROOT="/home/orin/mentor-backups"
LOCK_FILE="/home/orin/.mentor-nodered-v4.lock"
PATCH="$SCRIPT_DIR/patch-official-textile-count.js"
PATCH_TEST="$SCRIPT_DIR/patch-official-textile-count.test.js"
HOT="$SCRIPT_DIR/hot-deploy-official-counter.js"
HOT_TEST="$SCRIPT_DIR/hot-deploy-official-counter.test.js"
MIGRATION="$SCRIPT_DIR/01_mentor_outbox_idempotency.sql"

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Uso:
  ./nodered-pilot-v4.sh prepare
  ./nodered-pilot-v4.sh deploy
  ./nodered-pilot-v4.sh status
  ./nodered-pilot-v4.sh validate
  ./nodered-pilot-v4.sh rollback

prepare:
  respalda, bloquea solo 52.11.253.25:1883, crea indices y prepara candidato.
deploy:
  espera una frontera segura y hace hot deploy API v2 tipo nodes.
validate:
  correlaciona API, Modbus y la fila MariaDB de la primera frontera oficial.
rollback:
  restaura la logica anterior, pero mantiene aislado el Sender externo.
  No borra filas ni indices.

El firewall permanece bloqueado despues de todas las acciones.
USAGE
}

acquire_global_lock() {
  command -v flock >/dev/null 2>&1 ||
    die "falta flock; no es seguro ejecutar el piloto"
  (
    umask 077
    touch "$LOCK_FILE"
  )
  exec 9<>"$LOCK_FILE"
  flock -n 9 ||
    die "otra accion del piloto Node-RED ya esta en ejecucion"
}

verify_bundle() {
  for file in \
    "$PATCH" "$PATCH_TEST" "$HOT" "$HOT_TEST" "$MIGRATION" \
    "$SCRIPT_DIR/SHA256SUMS"; do
    test -s "$file" || die "falta artefacto: $file"
  done
  (
    cd "$SCRIPT_DIR"
    sha256sum -c SHA256SUMS
  )
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
  test ! -e "$temporary" || die "temporal pointer ya existe"
  (
    umask 077
    {
      printf 'STATE_NAME=%s\n' "$STATE_NAME"
      printf 'STATE_DIR=%s\n' "$STATE_DIR"
      printf 'BACKUP=%s\n' "$BACKUP"
      printf 'NR_IP=%s\n' "$NR_IP"
      printf 'RULE_COMMENT=%s\n' "$RULE_COMMENT"
      printf 'CONTAINER_ID=%s\n' "$CONTAINER_ID"
      printf 'STARTED_AT=%s\n' "$STARTED_AT"
      printf 'RESTARTS=%s\n' "$RESTARTS"
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
  NR_IP="$(pointer_value NR_IP)"
  RULE_COMMENT="$(pointer_value RULE_COMMENT)"
  CONTAINER_ID="$(pointer_value CONTAINER_ID)"
  STARTED_AT="$(pointer_value STARTED_AT)"
  RESTARTS="$(pointer_value RESTARTS)"

  [[ "$STATE_NAME" =~ ^nodered-v4-[0-9]{8}T[0-9]{6}Z$ ]] ||
    die "STATE_NAME invalido"
  test "$STATE_DIR" = "$STATE_ROOT/$STATE_NAME" ||
    die "STATE_DIR invalido"
  [[ "$BACKUP" =~ ^/home/orin/mentor-backups/nodered-v4-[0-9]{8}T[0-9]{6}Z$ ]] ||
    die "BACKUP invalido"
  [[ "$NR_IP" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] ||
    die "NR_IP invalido"
  [[ "$RULE_COMMENT" =~ ^mentor-textile-v4-[0-9]{8}T[0-9]{6}Z$ ]] ||
    die "RULE_COMMENT invalido"
  [[ "$CONTAINER_ID" =~ ^[0-9a-f]{64}$ ]] ||
    die "CONTAINER_ID invalido"
  [[ "$RESTARTS" =~ ^[0-9]+$ ]] || die "RESTARTS invalido"
}

firewall_rule() {
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

require_network_identity() {
  local current_ips
  mapfile -t current_ips < <(
    docker inspect "$NR_CONTAINER" \
      --format '{{range .NetworkSettings.Networks}}{{println .IPAddress}}{{end}}' |
    sed '/^$/d'
  )
  test "${#current_ips[@]}" -eq 1 ||
    die "Node-RED ya no tiene exactamente una IP"
  test "${current_ips[0]}" = "$NR_IP" ||
    die "IP Node-RED cambio: esperada=$NR_IP actual=${current_ips[0]}"
}

require_firewall() {
  local packets_before packets_after
  require_network_identity
  firewall_rule
  sudo iptables -C DOCKER-USER "${FIREWALL_RULE[@]}" ||
    die "regla de aislamiento ausente"
  packets_before="$(
    sudo iptables -vxnL DOCKER-USER |
      awk -v marker="$RULE_COMMENT" '
        index($0, marker) { print $1; found++ }
        END { if (found != 1) exit 1 }
      '
  )"
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
    die "el contenedor logro conectar o la prueba de firewall fallo"
  fi
  packets_after="$(
    sudo iptables -vxnL DOCKER-USER |
      awk -v marker="$RULE_COMMENT" '
        index($0, marker) { print $1; found++ }
        END { if (found != 1) exit 1 }
      '
  )"
  test "$packets_after" -gt "$packets_before" ||
    die "la prueba TCP no atraveso la regla de aislamiento"
}

current_identity() {
  docker inspect "$NR_CONTAINER" \
    --format '{{.Id}}|{{.State.StartedAt}}|{{.RestartCount}}'
}

require_identity() {
  local expected="$CONTAINER_ID|$STARTED_AT|$RESTARTS"
  local actual
  actual="$(current_identity)"
  test "$actual" = "$expected" ||
    die "Node-RED fue recreado/reiniciado: esperado=$expected actual=$actual"
}

require_indexes() {
  local dbcli actual expected
  dbcli="$(command -v mariadb || command -v mysql || true)"
  test -n "$dbcli" || die "no existe cliente MariaDB"
  actual="$(
    sudo "$dbcli" --database=mentor --batch --skip-column-names \
      --execute="
        SELECT CONCAT(
          table_name,'|',index_name,'|',non_unique,'|',
          GROUP_CONCAT(column_name ORDER BY seq_in_index)
        )
        FROM information_schema.statistics
        WHERE table_schema=DATABASE()
          AND index_name IN (
            'mqtt_snapshot_device_uq',
            'mqtt_lecturas_mqtt_pending_idx',
            'mqtt_lecturas_rest_pending_idx'
          )
        GROUP BY table_name,index_name,non_unique
        ORDER BY table_name,index_name;
      "
  )"
  expected="$(
    cat <<'EOF'
mqtt_lecturas|mqtt_lecturas_mqtt_pending_idx|1|mentor_id,status,device,time
mqtt_lecturas|mqtt_lecturas_rest_pending_idx|1|mentor_id,restful,device,time
mqtt_snapshot|mqtt_snapshot_device_uq|0|device
EOF
  )"
  test "$actual" = "$expected" ||
    die "indices ausentes o con definicion inesperada: $actual"
}

official_counter_state() {
  docker exec "$PG_CONTAINER" \
    psql -X -U mentor -d mentor_edge -At -F '|' -c "
      SELECT
        EXTRACT(EPOCH FROM state.counter_epoch)::BIGINT,
        state.counter_value,
        (
          SELECT COUNT(*)
          FROM linea_1.vision_detections AS detection
          WHERE detection.detected_at >= state.counter_epoch
        )
      FROM linea_1.vision_counter_state AS state
      WHERE state.counter_name='CORTE_TOTAL';
    "
}

require_pilot_counter() {
  local value epoch count dets
  value="$(official_counter_state)"
  IFS='|' read -r epoch count dets <<<"$value"
  test "$epoch" = "1785457500" ||
    die "epoch del contador oficial inesperado (esperado 1785457500): $value"
  [[ "$count" =~ ^[0-9]+$ && "$dets" =~ ^[0-9]+$ ]] ||
    die "estado del contador oficial no numerico: $value"
  test "$count" = "$dets" ||
    die "contador oficial incoherente (counter_value != detecciones): $value"
}

finalize_prepare() {
  local dbcli existing_rule
  dbcli="$(command -v mariadb || command -v mysql || true)"
  test -n "$dbcli" || die "no existe cliente MariaDB"
  require_network_identity
  firewall_rule
  if sudo iptables -C DOCKER-USER "${FIREWALL_RULE[@]}" \
       >/dev/null 2>&1; then
    existing_rule="true"
  else
    existing_rule="false"
  fi
  if [ "$existing_rule" = "false" ]; then
    if sudo iptables -S DOCKER-USER |
         grep -F "mentor-textile-v4-" |
         grep -Fvq -- "$RULE_COMMENT"; then
      die "existe otra regla mentor-textile-v4; auditar manualmente"
    fi
    sudo iptables -I DOCKER-USER 1 "${FIREWALL_RULE[@]}"
  fi
  require_firewall
  sudo iptables -S DOCKER-USER |
    grep -F -- "$RULE_COMMENT" > "$BACKUP/iptables.block.rule"
  chmod 600 "$BACKUP/iptables.block.rule"

  sudo "$dbcli" --database=mentor --batch --raw < "$MIGRATION"
  require_indexes
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
    raw: registers,
  }, null, 2));
  socket.end();
});
NODE
  python3 -m json.tool "$destination" >/dev/null
}

assert_modbus_continuity() {
  local before="$1"
  local after="$2"
  local require_same_count="${3:-true}"
  python3 - "$before" "$after" "$require_same_count" <<'PY'
import json
import sys

before = json.load(open(sys.argv[1]))
after = json.load(open(sys.argv[2]))
same_count = sys.argv[3] == "true"

for key in ("disponible", "microparada", "pna"):
    assert after[key] >= before[key], (key, before[key], after[key])
if same_count:
    assert after["conteo"] == before["conteo"], (before, after)
print(json.dumps(after, indent=2))
print("MODBUS_CONTINUIDAD_OK")
PY
}

copy_runtime_state() {
  local label="$1"
  local destination="$BACKUP/$label"
  if [ -e "$destination" ]; then
    local source_hash destination_hash
    source_hash="$(
      docker exec "$NR_CONTAINER" \
        sha256sum "$STATE_DIR/state.json" |
      awk '{print $1}'
    )"
    destination_hash="$(
      sha256sum "$destination/state.json" |
      awk '{print $1}'
    )"
    test "$source_hash" = "$destination_hash" ||
      die "copia runtime existente difiere: $destination"
    return
  fi
  docker cp "$NR_CONTAINER:$STATE_DIR" "$destination"
  chmod -R go-rwx "$destination"
}

write_backup_checksums() {
  (
    cd "$BACKUP"
    find . -type f ! -name SHA256SUMS -print0 |
      sort -z |
      xargs -0 sha256sum
  ) > "$BACKUP/SHA256SUMS"
}

wait_until_epoch() {
  local target="$1"
  local now remaining delay
  while :; do
    now="$(date -u +%s)"
    if [ "$now" -ge "$target" ]; then
      break
    fi
    remaining="$((target - now))"
    delay=30
    if [ "$remaining" -lt "$delay" ]; then
      delay="$remaining"
    fi
    printf 'Faltan %s s...\n' "$remaining"
    sleep "$delay"
  done
}

prepare_action() {
  verify_bundle
  if [ -s "$POINTER" ]; then
    load_pointer
    require_identity
    docker exec "$NR_CONTAINER" \
      node "$STATE_DIR/hot-deploy.before.js" \
        --action status \
        --state-dir "$STATE_DIR" |
      tee "$BACKUP/hot.status.resume-prepare.log"
    grep -Fq '"stateStatus": "prepared"' \
      "$BACKUP/hot.status.resume-prepare.log" ||
      die "el estado existente ya no esta en prepare"
    grep -Fq '"matches": "before"' \
      "$BACKUP/hot.status.resume-prepare.log" ||
      die "runtime cambio; prepare no puede reanudarse"
    finalize_prepare
    require_identity
    copy_runtime_state "runtime-state.prepared"
    write_backup_checksums
    printf '\nPREPARACION_REANUDADA_OK\nSTATE_NAME=%s\nBACKUP=%s\n' \
      "$STATE_NAME" "$BACKUP"
    printf 'El broker externo sigue BLOQUEADO. Aun no se desplegaron flows.\n'
    printf 'NO reinicies Jetson, Docker ni Node-RED antes de deploy.\n'
    return
  fi
  test ! -e "$POINTER" ||
    die "$POINTER existe pero esta vacio/invalido; auditar manualmente"
  test "$(docker inspect "$NR_CONTAINER" --format '{{.State.Running}}')" = "true" ||
    die "Node-RED no esta corriendo"
  test "$(docker inspect "$NR_CONTAINER" --format '{{.State.Health.Status}}')" = "healthy" ||
    die "Node-RED no esta healthy"

  local stamp
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  STATE_NAME="nodered-v4-$stamp"
  STATE_DIR="$STATE_ROOT/$STATE_NAME"
  BACKUP="$BACKUP_ROOT/$STATE_NAME"
  RULE_COMMENT="mentor-textile-v4-$stamp"
  CONTAINER_ID="$(docker inspect "$NR_CONTAINER" --format '{{.Id}}')"
  STARTED_AT="$(docker inspect "$NR_CONTAINER" --format '{{.State.StartedAt}}')"
  RESTARTS="$(docker inspect "$NR_CONTAINER" --format '{{.RestartCount}}')"

  mapfile -t nr_ips < <(
    docker inspect "$NR_CONTAINER" \
      --format '{{range .NetworkSettings.Networks}}{{println .IPAddress}}{{end}}' |
    sed '/^$/d'
  )
  test "${#nr_ips[@]}" -eq 1 ||
    die "se esperaba una sola IP de Node-RED"
  NR_IP="${nr_ips[0]}"
  [[ "$NR_IP" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] ||
    die "IP Node-RED invalida: $NR_IP"

  mkdir -- "$BACKUP" ||
    die "el respaldo ya existe o no pudo crearse: $BACKUP"
  chmod 700 "$BACKUP"
  docker inspect "$NR_CONTAINER" > "$BACKUP/container.before.json"
  docker cp "$NR_CONTAINER:/data/flows.json" \
    "$BACKUP/flows.disk.before.json"
  docker cp "$NR_CONTAINER:/data/flows_cred.json" \
    "$BACKUP/flows_cred.before.json"
  docker cp "$NR_CONTAINER:/data/settings.js" \
    "$BACKUP/settings.before.js"
  if docker exec "$NR_CONTAINER" test -s /data/.config.runtime.json; then
    docker cp "$NR_CONTAINER:/data/.config.runtime.json" \
      "$BACKUP/config.runtime.before.json"
  elif ! docker exec "$NR_CONTAINER" sh -lc \
    "grep -Eq '^[[:space:]]*credentialSecret[[:space:]]*:' /data/settings.js"; then
    die "no se encontro .config.runtime.json ni credentialSecret explicito"
  fi
  chmod 600 "$BACKUP"/*
  sudo iptables-save > "$BACKUP/iptables.before.rules"
  chmod 600 "$BACKUP/iptables.before.rules"

  local dbcli dumpcli
  dbcli="$(command -v mariadb || command -v mysql || true)"
  dumpcli="$(command -v mariadb-dump || command -v mysqldump || true)"
  test -n "$dbcli" || die "no existe cliente MariaDB"
  test -n "$dumpcli" || die "no existe dump MariaDB"
  sudo "$dumpcli" --single-transaction --quick \
    mentor mqtt_lecturas mqtt_snapshot \
    > "$BACKUP/mentor-outbox.before.sql"
  test -s "$BACKUP/mentor-outbox.before.sql" ||
    die "dump MariaDB vacio"
  chmod 600 "$BACKUP/mentor-outbox.before.sql"

  docker exec "$PG_CONTAINER" \
    pg_dump -U mentor -d mentor_edge -Fc \
    -t linea_1.vision_counter_state \
    -t linea_1.vision_counter_snapshots \
    -t linea_1.vision_detections \
    > "$BACKUP/mentor-edge-counter.before.dump"
  test -s "$BACKUP/mentor-edge-counter.before.dump" ||
    die "dump PostgreSQL vacio"
  docker exec -i "$PG_CONTAINER" \
    pg_restore --list < "$BACKUP/mentor-edge-counter.before.dump" \
    > "$BACKUP/mentor-edge-counter.before.list"
  local counter_state cp_epoch cp_count cp_dets
  counter_state="$(official_counter_state)"
  printf 'COUNTER_PREFLIGHT=%s\n' "$counter_state" |
    tee "$BACKUP/counter.preflight.txt"
  IFS='|' read -r cp_epoch cp_count cp_dets <<<"$counter_state"
  test "$cp_epoch" = "1785457500" ||
    die "epoch del contador oficial inesperado (esperado 1785457500): $counter_state"
  [[ "$cp_count" =~ ^[0-9]+$ && "$cp_dets" =~ ^[0-9]+$ ]] ||
    die "estado del contador oficial no numerico: $counter_state"
  test "$cp_count" = "$cp_dets" ||
    die "contador oficial incoherente (counter_value != detecciones): $counter_state"

  local duplicate_readings duplicate_snapshots reading_unique mentor_scope
  duplicate_readings="$(
    sudo "$dbcli" --database=mentor --batch --skip-column-names \
      --execute="
        SELECT COUNT(*)
        FROM (
          SELECT 1
          FROM mqtt_lecturas
          GROUP BY device,time
          HAVING COUNT(*) > 1
        ) AS duplicate_groups;
      "
  )"
  duplicate_snapshots="$(
    sudo "$dbcli" --database=mentor --batch --skip-column-names \
      --execute="
        SELECT COUNT(*)
        FROM (
          SELECT 1
          FROM mqtt_snapshot
          GROUP BY device
          HAVING COUNT(*) > 1
        ) AS duplicate_groups;
      "
  )"
  reading_unique="$(
    sudo "$dbcli" --database=mentor --batch --skip-column-names \
      --execute="
        SELECT COUNT(*)
        FROM (
          SELECT index_name
          FROM information_schema.statistics
          WHERE table_schema=DATABASE()
            AND table_name='mqtt_lecturas'
          GROUP BY index_name,non_unique
          HAVING non_unique=0
             AND GROUP_CONCAT(
               column_name ORDER BY seq_in_index
             )='device,time'
        ) AS matching_indexes;
      "
  )"
  mentor_scope="$(
    sudo "$dbcli" --database=mentor --batch --skip-column-names \
      --execute="
        SELECT CONCAT(
          COUNT(DISTINCT mentor_id),'|',
          MIN(mentor_id),'|',
          MAX(mentor_id)
        )
        FROM mqtt_lecturas;
      "
  )"
  test "$duplicate_readings" = "0" ||
    die "hay duplicados mqtt_lecturas(device,time)"
  test "$duplicate_snapshots" = "0" ||
    die "hay duplicados mqtt_snapshot(device)"
  test "$reading_unique" -ge 1 ||
    die "falta UNIQUE mqtt_lecturas(device,time)"
  test "$mentor_scope" = "1|478|478" ||
    die "mentor_id MariaDB ya no coincide con 478: $mentor_scope"

  local broker
  broker="$(
    docker exec "$NR_CONTAINER" node -e '
      const fs=require("fs");
      const flows=JSON.parse(fs.readFileSync("/data/flows.json","utf8"));
      const by=Object.fromEntries(flows.map((node)=>[node.id,node]));
      const sender=by["b94f3dfa216658f3"];
      if(!sender||sender.type!=="Sender") process.exit(2);
      const config=by[sender.mqtt];
      if(!config) process.exit(3);
      const save=by["b0a365a2d443baa2"];
      const alarms=by["b8aad5b1fb231930"];
      if(!save||!alarms||sender.mysql===save.mysql) process.exit(4);
      process.stdout.write(
        `${config.broker}|${config.port}|`+
        `${sender.mysql}|${save.mysql}|${alarms.mysql}`
      );
    '
  )"
  test "$broker" = \
    "$BROKER_IP|$BROKER_PORT|7ba0277c.2fa1b8|dfee56952fb9dd3b|7ba0277c.2fa1b8" ||
    die "broker Sender inesperado: $broker"

  capture_modbus "$BACKUP/modbus.prepare.json"

  local tmp_dir
  tmp_dir="/tmp/$STATE_NAME"
  docker exec "$NR_CONTAINER" mkdir -p "$tmp_dir"
  cleanup_prepare() {
    docker exec "$NR_CONTAINER" rm -rf -- "$tmp_dir" >/dev/null 2>&1 || true
  }
  trap cleanup_prepare EXIT
  for file in "$PATCH" "$PATCH_TEST" "$HOT" "$HOT_TEST"; do
    docker cp "$file" "$NR_CONTAINER:$tmp_dir/"
  done
  docker exec "$NR_CONTAINER" \
    cp -- /data/flows.json "$tmp_dir/nodered-flows.audit.json"

  docker exec \
    -e NODERED_AUDIT_PATH="$tmp_dir/nodered-flows.audit.json" \
    "$NR_CONTAINER" \
    node --test \
      "$tmp_dir/patch-official-textile-count.test.js" \
      "$tmp_dir/hot-deploy-official-counter.test.js" |
    tee "$BACKUP/tests.log"

  require_identity
  docker exec "$NR_CONTAINER" \
    node "$tmp_dir/hot-deploy-official-counter.js" \
      --action prepare \
      --state-dir "$STATE_DIR" \
      --patch "$tmp_dir/patch-official-textile-count.js" \
      --linea-id 1 |
    tee "$BACKUP/hot.prepare.log"

  require_identity
  write_pointer
  copy_runtime_state "runtime-state.prepared"
  trap - EXIT
  cleanup_prepare
  finalize_prepare
  require_identity
  write_backup_checksums

  printf '\nPREPARACION_OK\nSTATE_NAME=%s\nBACKUP=%s\n' \
    "$STATE_NAME" "$BACKUP"
  printf 'El broker externo sigue BLOQUEADO. Aun no se desplegaron flows.\n'
  printf 'NO reinicies Jetson, Docker ni Node-RED antes de deploy.\n'
}

deploy_action() {
  verify_bundle
  load_pointer
  require_identity
  require_firewall
  require_indexes
  require_pilot_counter
  docker exec "$NR_CONTAINER" test -s "$STATE_DIR/state.json" ||
    die "estado runtime ausente"

  local now deploy_at
  now="$(date -u +%s)"
  deploy_at="$(( ((now / 300) + 1) * 300 + 30 ))"
  printf 'No pases prendas durante el cambio.\n'
  printf 'Hot deploy programado: %s\n' \
    "$(date -u -d "@$deploy_at" +%Y-%m-%dT%H:%M:%SZ)"
  wait_until_epoch "$deploy_at"

  require_identity
  require_firewall
  require_pilot_counter
  capture_modbus "$BACKUP/modbus.predeploy.json"

  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/hot-deploy.before.js" \
      --action deploy \
      --state-dir "$STATE_DIR" \
      --patch "$STATE_DIR/patch.before.js" \
      --linea-id 1 |
    tee "$BACKUP/hot.deploy.log"

  sleep 8
  require_identity
  require_firewall
  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/hot-deploy.before.js" \
      --action status \
      --state-dir "$STATE_DIR" |
    tee "$BACKUP/hot.status.after-deploy.log"
  grep -Fq '"matches": "candidate"' \
    "$BACKUP/hot.status.after-deploy.log" ||
    die "runtime no coincide con candidato"

  capture_modbus "$BACKUP/modbus.postdeploy.json"
  assert_modbus_continuity \
    "$BACKUP/modbus.predeploy.json" \
    "$BACKUP/modbus.postdeploy.json" \
    true
  copy_runtime_state "runtime-state.deployed"
  write_backup_checksums

  printf '\nHOT_DEPLOY_OK\nSTATE_NAME=%s\nBACKUP=%s\n' \
    "$STATE_NAME" "$BACKUP"
  printf 'El broker externo sigue BLOQUEADO.\n'
}

status_action() {
  load_pointer
  require_firewall
  printf 'IDENTIDAD_ACTUAL=%s\n' "$(current_identity)"
  printf 'IDENTIDAD_ESPERADA=%s|%s|%s\n' \
    "$CONTAINER_ID" "$STARTED_AT" "$RESTARTS"
  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/hot-deploy.before.js" \
      --action status \
      --state-dir "$STATE_DIR"
}

validate_action() {
  load_pointer
  require_identity
  require_firewall
  printf 'VALIDATOR_VERSION=snapshot-current-v3-dynamic-baseline\n'
  local deployed_epoch deployed_iso
  deployed_iso="$(
    docker exec "$NR_CONTAINER" node -e "
      const fs=require('fs');
      const value=JSON.parse(
        fs.readFileSync('$STATE_DIR/state.json','utf8')
      );
      if(value.status!=='deployed'||typeof value.deployedAt!=='string')
        process.exit(2);
      process.stdout.write(value.deployedAt);
    "
  )"
  deployed_epoch="$(date -u -d "$deployed_iso" +%s)"
  [[ "$deployed_epoch" =~ ^[0-9]+$ ]] ||
    die "DEPLOYED_EPOCH invalido"

  local boundary ready b_ms until
  boundary="$(( ((deployed_epoch / 300) + 1) * 300 ))"
  ready="$((boundary + 60))"
  b_ms="${boundary}000"
  until="$(date -u -d "@$boundary" +%Y-%m-%dT%H:%M:%SZ)"
  printf 'Validando primera frontera oficial B=%s\n' "$until"
  wait_until_epoch "$ready"

  local pg_snapshot snapshot_rows snapshot_value snapshot_epoch
  pg_snapshot="$(
    docker exec "$PG_CONTAINER" \
      psql -X -U mentor -d mentor_edge -At -F '|' -c "
        SELECT
          COUNT(*),
          MIN(counter_value),
          EXTRACT(EPOCH FROM MIN(counter_epoch))::BIGINT
        FROM linea_1.vision_counter_snapshots
        WHERE counter_until=to_timestamp($boundary);
      "
  )"
  printf 'PG_SNAPSHOT_AUTOMATICO=%s\n' "$pg_snapshot"
  IFS='|' read -r snapshot_rows snapshot_value snapshot_epoch <<<"$pg_snapshot"
  test "$snapshot_rows" = "1" ||
    die "Node-RED no creo exactamente un snapshot para la frontera: $pg_snapshot"
  test "$snapshot_epoch" = "1785457500" ||
    die "epoch del snapshot inesperado (esperado 1785457500): $pg_snapshot"
  [[ "$snapshot_value" =~ ^[0-9]+$ ]] ||
    die "counter_value del snapshot no numerico: $pg_snapshot"

  local counter_file counter_retry_file expected_count
  counter_file="$BACKUP/counter.$boundary.first.json"
  counter_retry_file="$BACKUP/counter.$boundary.retry.json"
  curl -fsSG http://127.0.0.1:8002/vision/counter \
    --data-urlencode "linea_id=1" \
    --data-urlencode "until=$until" \
    -o "$counter_file"
  curl -fsSG http://127.0.0.1:8002/vision/counter \
    --data-urlencode "linea_id=1" \
    --data-urlencode "until=$until" \
    -o "$counter_retry_file"
  expected_count="$(
    python3 - "$counter_file" "$counter_retry_file" "$until" <<'PY'
import datetime as dt
import json
import sys

value = json.load(open(sys.argv[1]))
retry = json.load(open(sys.argv[2]))
until = sys.argv[3]
assert retry == value, (value, retry)
assert value["linea_id"] == 1, value
assert value["event_type"] == "CORTE", value
assert value["until"] == until, value
assert isinstance(value["count"], int) and value["count"] >= 0, value
epoch = dt.datetime.fromisoformat(
    value["counter_epoch"].replace("Z", "+00:00")
)
assert int(epoch.timestamp()) == 1785457500, value
as_of = dt.datetime.fromisoformat(value["as_of"].replace("Z", "+00:00"))
limit = dt.datetime.fromisoformat(until.replace("Z", "+00:00"))
assert as_of >= limit + dt.timedelta(seconds=10), value
updated = dt.datetime.fromisoformat(
    value["state_updated_at"].replace("Z", "+00:00")
)
assert epoch <= updated <= as_of, value
print(value["count"])
PY
  )"
  [[ "$expected_count" =~ ^[0-9]+$ ]] ||
    die "count API invalido"
  test "$snapshot_value" = "$expected_count" ||
    die "snapshot counter_value ($snapshot_value) != count API ($expected_count)"

  local dbcli rows
  dbcli="$(command -v mariadb || command -v mysql || true)"
  test -n "$dbcli" || die "no existe cliente MariaDB"
  rows="0"
  for _ in $(seq 1 20); do
    rows="$(
      sudo "$dbcli" --database=mentor --batch --skip-column-names \
        --execute="
          SELECT COUNT(*)
          FROM mqtt_lecturas
          WHERE device='ART_ATLAS_MAQUINA_1_PRODUCCION'
            AND time='$b_ms';
        "
    )"
    test "$rows" = "1" && break
    sleep 2
  done
  test "$rows" = "1" ||
    die "no existe exactamente una fila MariaDB para B=$b_ms"

  local outbox_meta snapshot_mentor snapshot_ms snapshot_status
  local snapshot_restful snapshot_hex outbox_mentor outbox_status
  local outbox_restful outbox_hex latest_outbox_ms latest_grid_ms
  local snapshot_capture snapshot_consistent
  outbox_meta="$(
    sudo "$dbcli" --database=mentor --batch --raw --skip-column-names \
      --execute="
        SELECT CONCAT(
          COUNT(*),'|',
          MIN(mentor_id),'|',
          MIN(time),'|',
          MIN(status),'|',
          MIN(restful)
        )
        FROM mqtt_lecturas
        WHERE device='ART_ATLAS_MAQUINA_1_PRODUCCION'
          AND time='$b_ms';
      "
  )"
  printf 'OUTBOX_META=%s\n' "$outbox_meta"
  test "$outbox_meta" = "1|478|$b_ms|0|0" ||
    die "metadata outbox no coincide o la fila salio del piloto"
  sudo "$dbcli" --database=mentor --batch --raw --skip-column-names \
    --execute="
      SELECT content
      FROM mqtt_lecturas
      WHERE device='ART_ATLAS_MAQUINA_1_PRODUCCION'
        AND time='$b_ms';
    " > "$BACKUP/outbox.$boundary.content.json"

  snapshot_capture="$BACKUP/snapshot.$boundary.capture.txt"
  snapshot_consistent="false"
  for _ in $(seq 1 20); do
    latest_grid_ms="$(( ($(date -u +%s) / 300) * 300 * 1000 ))"
    sudo "$dbcli" --database=mentor --batch --raw --skip-column-names \
      --execute="
        SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;
        START TRANSACTION WITH CONSISTENT SNAPSHOT;
        SELECT
          s.mentor_id,
          s.time,
          s.status,
          s.restful,
          o.mentor_id,
          o.status,
          o.restful,
          HEX(s.content),
          HEX(o.content),
          COALESCE((
            SELECT MAX(CAST(latest.time AS UNSIGNED))
            FROM mqtt_lecturas latest
            WHERE BINARY latest.device =
                  BINARY 'ART_ATLAS_MAQUINA_1_PRODUCCION'
              AND latest.mentor_id=478
              AND latest.time REGEXP '^[0-9]{13}$'
              AND CAST(latest.time AS UNSIGNED) <= $latest_grid_ms
              AND MOD(CAST(latest.time AS UNSIGNED),300000)=0
          ),0)
        FROM mqtt_snapshot s
        JOIN mqtt_lecturas o
          ON BINARY o.device=BINARY s.device
         AND BINARY o.time=BINARY s.time
        WHERE BINARY s.device =
              BINARY 'ART_ATLAS_MAQUINA_1_PRODUCCION';
        COMMIT;
      " > "$snapshot_capture"
    if test "$(wc -l < "$snapshot_capture")" -eq 1; then
      IFS=$'\t' read -r \
        snapshot_mentor snapshot_ms snapshot_status snapshot_restful \
        outbox_mentor outbox_status outbox_restful \
        snapshot_hex outbox_hex latest_outbox_ms < "$snapshot_capture"
      if [[ "$snapshot_ms" =~ ^[0-9]{13}$ ]] &&
         test "$snapshot_mentor|$snapshot_status|$snapshot_restful" = \
           "478|0|0" &&
         test "$outbox_mentor|$outbox_status|$outbox_restful" = \
           "478|0|0" &&
         test -n "$snapshot_hex" &&
         test "$snapshot_hex" = "$outbox_hex" &&
         test "$snapshot_ms" = "$latest_outbox_ms" &&
         test "$snapshot_ms" -ge "$b_ms" &&
         test "$snapshot_ms" -le "$latest_grid_ms" &&
         test "$(((snapshot_ms - b_ms) % 300000))" -eq 0; then
        snapshot_consistent="true"
        break
      fi
    fi
    sleep 2
  done

  test "$snapshot_consistent" = "true" ||
    die "mqtt_snapshot no converge con el ultimo outbox de 5 minutos"
  printf 'SNAPSHOT_META=1|%s|%s|%s|%s\n' \
    "$snapshot_mentor" "$snapshot_ms" "$snapshot_status" "$snapshot_restful"
  printf 'SNAPSHOT_OUTBOX_META=1|%s|%s|%s|%s\n' \
    "$outbox_mentor" "$snapshot_ms" "$outbox_status" "$outbox_restful"

  python3 - \
    "$BACKUP/outbox.$boundary.content.json" \
    "$snapshot_capture" \
    "$b_ms" \
    "$expected_count" \
    "$latest_grid_ms" <<'PY'
import json
from pathlib import Path
import sys

boundary_outbox = json.load(open(sys.argv[1]))
fields = Path(sys.argv[2]).read_text().rstrip("\n").split("\t")
assert len(fields) == 10, fields
(
    snapshot_mentor,
    snapshot_time,
    snapshot_status,
    snapshot_restful,
    outbox_mentor,
    outbox_status,
    outbox_restful,
    snapshot_hex,
    outbox_hex,
    latest_outbox_time,
) = fields
b_ms = int(sys.argv[3])
expected = int(sys.argv[4])
latest_grid_ms = int(sys.argv[5])
snapshot_ms = int(snapshot_time)

assert snapshot_mentor == outbox_mentor == "478", fields
assert snapshot_status == outbox_status == "0", fields
assert snapshot_restful == outbox_restful == "0", fields
assert snapshot_ms == int(latest_outbox_time), fields
assert b_ms <= snapshot_ms <= latest_grid_ms, fields
assert (snapshot_ms - b_ms) % 300000 == 0, fields
snapshot_raw = bytes.fromhex(snapshot_hex)
outbox_raw = bytes.fromhex(outbox_hex)
assert snapshot_raw == outbox_raw, fields
snapshot = json.loads(snapshot_raw.decode("utf-8"))
snapshot_outbox = json.loads(outbox_raw.decode("utf-8"))

expected_head = [
    "L1_T_DISPONIBLE",
    "L1_T_MICROPARADA",
    "L1_T_PARADA_NO_ASIGNADA",
    "L1_CONTEO_1",
]

def validate_envelope(value, expected_time):
    assert value["code"] == "ART_ATLAS_MAQUINA_1_PRODUCCION", value
    assert int(value["time"]) == expected_time, value
    assert value["head"] == expected_head, value
    assert isinstance(value["data"], list) and len(value["data"]) == 4, value
    assert all(
        type(item) in (int, float) and item >= 0
        for item in value["data"]
    ), value
    assert value["data"][3] == expected, value

validate_envelope(boundary_outbox, b_ms)
validate_envelope(snapshot_outbox, snapshot_ms)
validate_envelope(snapshot, snapshot_ms)
assert snapshot == snapshot_outbox, (snapshot_outbox, snapshot)
print(json.dumps({
    "boundary_outbox": boundary_outbox,
    "current_snapshot": snapshot,
}, indent=2))
print("OUTBOX_B_Y_SNAPSHOT_ACTUAL_EXACTOS")
PY

  capture_modbus "$BACKUP/modbus.validated.first.json"
  sleep 3
  capture_modbus "$BACKUP/modbus.validated.second.json"
  python3 - \
    "$BACKUP/modbus.predeploy.json" \
    "$BACKUP/modbus.validated.first.json" \
    "$BACKUP/modbus.validated.second.json" \
    "$expected_count" <<'PY'
import json
import sys

before = json.load(open(sys.argv[1]))
first = json.load(open(sys.argv[2]))
second = json.load(open(sys.argv[3]))
expected = int(sys.argv[4])
for key in ("disponible", "microparada", "pna"):
    assert first[key] >= before[key], (key, before[key], first[key])
    assert second[key] >= first[key], (key, first[key], second[key])
assert first["conteo"] == expected, (expected, first)
assert second["conteo"] == expected, (expected, second)
print(json.dumps(second, indent=2))
print("MODBUS_API_CORRELACION_OK")
PY

  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/hot-deploy.before.js" \
      --action status \
      --state-dir "$STATE_DIR" |
    tee "$BACKUP/hot.status.validated.log"
  grep -Fq '"matches": "candidate"' \
    "$BACKUP/hot.status.validated.log" ||
    die "runtime dejo de coincidir con candidato"

  docker logs --since "$deployed_iso" "$NR_CONTAINER" \
    > "$BACKUP/nodered.after-deploy.log" 2>&1
  if grep -Ei \
       'ER_DUP_ENTRY|Duplicate entry|Database not connected|uncaught exception|CONTADOR v4:.*(bloqueado|fallo|invalido|excedio)|SAVE_DB_|modbus.*(error|fail|timeout)' \
       "$BACKUP/nodered.after-deploy.log"; then
    die "logs Node-RED contienen error de persistencia/runtime"
  fi
  write_backup_checksums

  printf '\nPILOTO_NODE_RED_OK\nB=%s\nCOUNT=%s\nBACKUP=%s\n' \
    "$until" "$expected_count" "$BACKUP"
  printf 'El broker externo sigue BLOQUEADO; no se valido entrega cloud.\n'
}

rollback_action() {
  load_pointer
  require_identity
  require_firewall
  capture_modbus "$BACKUP/modbus.prerollback.json"

  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/hot-deploy.before.js" \
      --action rollback \
      --state-dir "$STATE_DIR" |
    tee "$BACKUP/hot.rollback.log"

  sleep 8
  require_identity
  require_firewall
  docker exec "$NR_CONTAINER" \
    node "$STATE_DIR/hot-deploy.before.js" \
      --action status \
      --state-dir "$STATE_DIR" |
    tee "$BACKUP/hot.status.after-rollback.log"
  grep -Fq '"matches": "safe_rollback"' \
    "$BACKUP/hot.status.after-rollback.log" ||
    die "runtime no coincide con rollback seguro"
  capture_modbus "$BACKUP/modbus.postrollback.json"
  assert_modbus_continuity \
    "$BACKUP/modbus.prerollback.json" \
    "$BACKUP/modbus.postrollback.json" \
    true
  copy_runtime_state "runtime-state.rolled-back"
  write_backup_checksums

  printf '\nROLLBACK_HOT_OK\nBACKUP=%s\n' "$BACKUP"
  printf 'No se borraron indices ni filas. El broker sigue BLOQUEADO.\n'
}

action="${1:-}"
case "$action" in
  prepare|deploy|status|validate|rollback)
    acquire_global_lock
    ;;
  help|-h|--help|"")
    usage
    exit 0
    ;;
  *)
    usage >&2
    die "accion desconocida: $action"
    ;;
esac

case "$action" in
  prepare) prepare_action ;;
  deploy) deploy_action ;;
  status) status_action ;;
  validate) validate_action ;;
  rollback) rollback_action ;;
esac
