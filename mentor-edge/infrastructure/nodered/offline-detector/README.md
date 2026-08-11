# Detector textil offline para Jetson

Este paquete superpone el código local de `vision-event-detector` sobre la
imagen ARM64 que ya existe en el Jetson. No ejecuta `apt`, `pip`, `pull` ni
requiere DNS. Durante el build, Docker usa `--network=none` y ejecuta las
pruebas del caché de imagen, FSM, cola durable y calibración persistente.

El procedimiento está separado en dos acciones deliberadas:

1. `build` congela la imagen anterior, construye y prueba la candidata. No
   detiene, recrea ni inicia el contenedor.
2. `deploy` solo acepta un detector previamente detenido. Reetiqueta la
   candidata, recrea el servicio y espera salud. Ante cualquier fallo restaura
   la imagen anterior y la deja detenida.

No se modifica el Compose ni se aplican migraciones desde este paquete.

## 1. Crear y copiar el paquete

Desde `F:\Mentor-Textil` en PowerShell:

```powershell
$jetson = "orin@192.168.1.130"
$stage = "/home/orin/mentor-textile-pilot-v4"

.\mentor-edge\infrastructure\nodered\offline-detector\New-OfflineDetectorBundle.ps1 `
  -OutputArchive .\detector-offline-v5.tgz

scp .\detector-offline-v5.tgz "${jetson}:${stage}/"
scp .\detector-offline-v5.tgz.sha256 "${jetson}:${stage}/"
```

## 2. Verificar, extraer y construir sin cambiar el contenedor

En el Jetson:

```bash
set -Eeuo pipefail

stage=/home/orin/mentor-textile-pilot-v4
unpack="$stage/unpack-$(date -u +%Y%m%dT%H%M%SZ)"

cd "$stage"
sha256sum -c detector-offline-v5.tgz.sha256
mkdir -p "$unpack"
tar -xzf detector-offline-v5.tgz -C "$unpack"

bundle="$unpack/detector-offline-v5"
chmod 0755 "$bundle/detector-image.sh"

"$bundle/detector-image.sh" build
```

Guardar el `STATE_FILE` mostrado al final. Un build correcto termina con
`BUILD_OK`; la imagen activa y el estado del contenedor siguen intactos.

## 3. Preparar persistencia y Compose

Con el detector todavía detenido, respaldar PostgreSQL y el Compose. Luego
aplicar la tabla de calibración, normalizar los triggers de conteo y agregar el
volumen durable del detector:

```bash
set -Eeuo pipefail

repo=/home/orin/Mentoredge
compose="$repo/mentor-edge/infrastructure/docker/docker-compose.jetson-orin.yml"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="/home/orin/mentor-backups/$stamp"
mkdir -p "$backup"

cp -- "$compose" "$backup/docker-compose.jetson-orin.yml"
docker exec docker-postgres-1 pg_dump -U mentor -d mentor_edge -Fc \
  > "$backup/mentor_edge.dump"
test -s "$backup/mentor_edge.dump"

if git -C "$repo" apply --check "$bundle/compose-detector-v5.patch"; then
  git -C "$repo" apply "$bundle/compose-detector-v5.patch"
elif git -C "$repo" apply --reverse --check "$bundle/compose-detector-v5.patch"; then
  echo "Compose v5 ya estaba aplicado"
else
  echo "El Compose no coincide con el parche esperado" >&2
  exit 1
fi

docker exec -i docker-postgres-1 \
  psql -v ON_ERROR_STOP=1 -1 -U mentor -d mentor_edge \
  < "$bundle/30_detector_calibration.sql"

docker exec -i docker-postgres-1 \
  psql -v ON_ERROR_STOP=1 -1 -U mentor -d mentor_edge \
  < "$bundle/33_vision_detections_existing_lines.sql"

docker compose -f "$compose" config --quiet
```

La segunda migración conserva el `counter_epoch` existente al repetirse y deja
un único trigger extractor canónico por línea.

## 4. Corte coordinado y despliegue

Detener la producción y el detector antes de modificar el epoch/contador.
Después de completar el respaldo, la migración o reset coordinado que
corresponda, y mientras el detector siga detenido:

```bash
state=/ruta/impresa/detector-v5-AAAAMMDDTHHMMSSZ.state

/ruta/al/bundle/detector-image.sh status "$state"
/ruta/al/bundle/detector-image.sh deploy "$state"
```

El resultado válido es `DEPLOY_OK`, `HEALTH=healthy` y el `IMAGE_ID` candidato.
El script comprueba que el contenedor no haya cambiado desde `build`; si cambió
o está ejecutándose, se detiene sin hacer el corte.

## 5. Rollback recuperable

Si `deploy` falla, el rollback es automático. La imagen anterior queda
restaurada bajo la referencia que usa Compose, pero el detector queda detenido
para evitar reactivar falsos conteos.

Para volver manualmente a la imagen anterior:

```bash
/ruta/al/bundle/detector-image.sh rollback "$state"
```

No se eliminan imágenes, volúmenes, eventos ni el bundle. La candidata fallida,
la etiqueta `pre-v5-*` y el archivo de estado quedan disponibles para auditoría.
No iniciar la imagen anterior mientras siga vigente el defecto de conteo.
