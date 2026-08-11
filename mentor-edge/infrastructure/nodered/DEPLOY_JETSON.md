# Piloto controlado en Jetson

Este procedimiento valida primero PostgreSQL, `resiliencia` y una prenda.
**No aplica todavía el parche de Node-RED, no reemplaza el Sender y no drena la
cola MariaDB.**

## 1. Preparar artefactos en Windows

Desde PowerShell en `F:\Mentor-Textil`:

```powershell
$jetson = "orin@192.168.1.13"
$stage = "/home/orin/mentor-textile-pilot-v4"

ssh $jetson "mkdir -p $stage"

scp `
  .\mentor-edge\infrastructure\nodered\deploy-textile-counter-api.patch `
  "${jetson}:${stage}/"

scp `
  .\mentor-edge\infrastructure\database\33_vision_detections_existing_lines.sql `
  "${jetson}:${stage}/"
```

## 2. Verificar continuidad heredada

Detener físicamente la producción antes del corte. No es necesario detener
Node-RED todavía.

En el Jetson:

```bash
dbcli="$(command -v mariadb || command -v mysql)"

sudo "$dbcli" --database=mentor --batch --raw <<'SQL'
SELECT
    FROM_UNIXTIME(CAST(time AS UNSIGNED) / 1000) AS fecha,
    time,
    JSON_EXTRACT(content, '$.head') AS campos,
    JSON_EXTRACT(content, '$.data') AS valores,
    JSON_EXTRACT(content, '$.data[3]') AS ultimo_conteo
FROM mqtt_lecturas
WHERE device = 'ART_ATLAS_MAQUINA_1_PRODUCCION'
ORDER BY CAST(time AS UNSIGNED) DESC
LIMIT 1;
SQL
```

Continuar únicamente si:

- `campos[3]` es `L1_CONTEO_1`;
- `ultimo_conteo` sigue siendo `0`;
- no está pasando ninguna prenda durante la intervención.

Si el último valor ya no es cero, detenerse: la migración actual establece
`counter_baseline=0` y habría que preparar un baseline distinto.

## 3. Validar el parche sin modificar el repo

```bash
repo=/home/orin/Mentoredge
stage=/home/orin/mentor-textile-pilot-v4

git -C "$repo" status --short -- \
  mentor-edge/services/resiliencia

git -C "$repo" apply --check \
  "$stage/deploy-textile-counter-api.patch"
```

Si `git apply --check` falla, detenerse. El Jetson ya contiene cambios locales;
no usar `reset`, `checkout`, `clean`, `--force` ni aplicar solo algunos hunks.

## 4. Respaldar y compilar sin reemplazar contenedores

```bash
set -euo pipefail

repo=/home/orin/Mentoredge
stage=/home/orin/mentor-textile-pilot-v4
compose="$repo/mentor-edge/infrastructure/docker/docker-compose.jetson-orin.yml"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="/home/orin/mentor-backups/$stamp"

mkdir -p "$backup"

tar -C "$repo" -czf "$backup/resiliencia-source.tgz" \
  mentor-edge/services/resiliencia

docker exec docker-postgres-1 \
  pg_dump -U mentor -d mentor_edge -Fc \
  > "$backup/mentor_edge.dump"

test -s "$backup/resiliencia-source.tgz"
test -s "$backup/mentor_edge.dump"

git -C "$repo" apply \
  "$stage/deploy-textile-counter-api.patch"

docker compose -f "$compose" build resiliencia
```

El `build` es la validación de compilación Go/ARM64 y no reemplaza el contenedor
actual. Si falla, no continuar.

## 5. Migrar con el detector detenido

Ejecutar este paso apenas pasada una frontera exacta de cinco minutos y sin
producción. La migración usa esa frontera como `counter_epoch`.

```bash
set -euo pipefail

repo=/home/orin/Mentoredge
stage=/home/orin/mentor-textile-pilot-v4
compose="$repo/mentor-edge/infrastructure/docker/docker-compose.jetson-orin.yml"

docker compose -f "$compose" stop vision-event-detector

if ! docker exec -i docker-postgres-1 \
       psql -v ON_ERROR_STOP=1 -U mentor -d mentor_edge \
       < "$stage/33_vision_detections_existing_lines.sql"; then
  echo "Migración fallida; PostgreSQL revirtió la sentencia"
  docker compose -f "$compose" start vision-event-detector
  exit 1
fi

docker compose -f "$compose" up -d --no-deps resiliencia

for intento in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:8002/health >/dev/null; then
    break
  fi
  if [ "$intento" -eq 30 ]; then
    echo "resiliencia no respondió; no reiniciar el detector"
    exit 1
  fi
  sleep 2
done

docker compose -f "$compose" start vision-event-detector
```

## 6. Verificar estado y API

Esperar al menos 10 segundos después de la siguiente frontera:

```bash
b_epoch=$(( $(date -u +%s) / 300 * 300 ))
until="$(date -u -d "@$b_epoch" +%Y-%m-%dT%H:%M:%SZ)"

curl -fsSG http://127.0.0.1:8002/vision/counter \
  --data-urlencode "linea_id=1" \
  --data-urlencode "until=$until" \
  | python3 -m json.tool

docker exec docker-postgres-1 \
  psql -U mentor -d mentor_edge -x -c \
  "SELECT counter_name, counter_epoch, counter_baseline,
          counter_value, updated_at
     FROM linea_1.vision_counter_state;
   SELECT counter_until, counter_value, created_at
     FROM linea_1.vision_counter_snapshots
    ORDER BY counter_until DESC
    LIMIT 5;"
```

El primer resultado esperado es `count: 0`, `event_type: "CORTE"` y un
`counter_epoch` que no cambiará al reiniciar servicios.

## 7. Prueba física de una prenda

Pasar exactamente una prenda y esperar a que reaparezca el separador. Primero
comprobar el evento:

```bash
docker exec docker-postgres-1 \
  psql -U mentor -d mentor_edge -x -c \
  "SELECT detection_id, detected_at, line_code, fsm_state
     FROM linea_1.vision_detections
    ORDER BY detected_at DESC
    LIMIT 5;"
```

Después de la siguiente frontera más 10 segundos:

```bash
b_epoch=$(( $(date -u +%s) / 300 * 300 ))
until="$(date -u -d "@$b_epoch" +%Y-%m-%dT%H:%M:%SZ)"

curl -fsSG http://127.0.0.1:8002/vision/counter \
  --data-urlencode "linea_id=1" \
  --data-urlencode "until=$until" \
  | python3 -m json.tool
```

El total debe aumentar exactamente en uno. Repetir el mismo `curl` debe
devolver el mismo `count`, `counter_epoch`, `until` y `as_of`.

## Punto de parada obligatorio

No ejecutar aún:

- `patch-official-textile-count.js --apply`;
- reemplazo del `Sender`;
- cambio de intervalos;
- limpieza o reenvío de `mqtt_lecturas`;
- recreación del detector sin montar el volumen SQLite;
- reset del contador por producto, calibración o reinicio.

Antes de activar Node-RED hay que hacer durables los acumulados `T_*`, decidir
la cola pendiente y probar el flujo sin salida al broker externo.
