# Retiro recuperable de yolo-counter en el Jetson

El servicio observado es exclusivamente de botellas/cajas y no participa en el
flujo textil. Estos comandos lo deshabilitan y mueven sus archivos a un
respaldo; no los borran directamente.

```bash
set -euo pipefail

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="/home/orin/mentor-backups/yolo-counter-$stamp"

sudo install -d -m 0755 "$backup"

if systemctl list-unit-files yolo-counter.service --no-legend \
     | grep -q '^yolo-counter.service'; then
  sudo systemctl disable --now yolo-counter.service
fi

if [ -e /etc/systemd/system/yolo-counter.service ]; then
  sudo mv /etc/systemd/system/yolo-counter.service "$backup/"
fi

if [ -d /opt/yolo-counter ]; then
  sudo mv /opt/yolo-counter "$backup/"
fi

sudo systemctl daemon-reload
sudo systemctl reset-failed yolo-counter.service 2>/dev/null || true

systemctl is-active yolo-counter.service 2>/dev/null || true
systemctl is-enabled yolo-counter.service 2>/dev/null || true
sudo find "$backup" -maxdepth 2 -mindepth 1 -printf '%p\n'
```

El volumen Docker antiguo se retira únicamente si ningún contenedor lo usa y
después de exportarlo:

```bash
set -euo pipefail

volume=docker_yolo_models
users="$(docker ps -aq --filter "volume=$volume")"

if [ -n "$users" ]; then
  echo "NO retirar $volume; lo usan estos contenedores:"
  docker ps -a --filter "volume=$volume" \
    --format 'table {{.ID}}\t{{.Names}}\t{{.Status}}'
  exit 1
fi

if docker volume inspect "$volume" >/dev/null 2>&1; then
  backup="$(find /home/orin/mentor-backups \
    -maxdepth 1 -type d -name 'yolo-counter-*' \
    -printf '%T@ %p\n' | sort -nr | head -n1 | cut -d' ' -f2-)"

  test -n "$backup"

  docker run --rm --entrypoint sh \
    -v "$volume:/source:ro" \
    -v "$backup:/backup" \
    nginx:alpine \
    -c 'tar -C /source -czf /backup/docker_yolo_models.tgz .'

  sudo test -s "$backup/docker_yolo_models.tgz"
  docker volume rm "$volume"
fi

docker volume ls
```

La columna PostgreSQL `config.line_config.yolo` se elimina aparte mediante
`31_textile_only_remove_yolo.sql`, pero solo después de compilar y desplegar la
versión textil de `edge-config-service`. Ejecutar primero esa migración con el
servicio antiguo rompería sus consultas.
