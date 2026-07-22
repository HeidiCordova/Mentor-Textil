# Persistencia de calibracion HSV en Jetson

La referencia HSV se almacena en `public.detector_calibration` dentro del
PostgreSQL local del Jetson. El volumen Docker `pgdata` conserva los datos
frente a reinicios del proceso, contenedor y equipo.

## Instalacion nueva

`docker-compose.jetson-orin.yml` monta `30_detector_calibration.sql` en
`/docker-entrypoint-initdb.d`. PostgreSQL crea la tabla automaticamente cuando
inicializa un volumen `pgdata` nuevo.

## Jetson existente

Los scripts de `docker-entrypoint-initdb.d` no vuelven a ejecutarse sobre un
volumen existente. La migracion debe aplicarse antes de reiniciar el detector:

```bash
cd mentor-edge/infrastructure/docker
./deploy.sh migrate
docker compose -f docker-compose.jetson-orin.yml build vision-event-detector
docker compose -f docker-compose.jetson-orin.yml up -d vision-event-detector
```

La migracion es idempotente y no elimina tablas, calibraciones ni eventos.
`./deploy.sh deploy` tambien ejecuta la migracion automaticamente.

## Verificacion

```bash
docker compose -f docker-compose.jetson-orin.yml exec -T postgres \
  psql -U mentor -d mentor_edge -c '\d+ public.detector_calibration'

curl -s http://127.0.0.1:8001/calibrate
```

Después de una calibracion valida, el endpoint debe informar
`state=ready_persisted` y un `calibration_id`. Para revisar la fila activa:

```bash
docker compose -f docker-compose.jetson-orin.yml exec -T postgres \
  psql -U mentor -d mentor_edge -c \
  "SELECT id, line_id, device_id, quality_score, calibrated_at, expires_at \
     FROM public.detector_calibration WHERE active = TRUE;"
```

## Reemplazo del Jetson

Antes de retirar el equipo, respaldar la base completa o al menos la tabla:

```bash
docker compose -f docker-compose.jetson-orin.yml exec -T postgres \
  pg_dump -U mentor -d mentor_edge -t public.detector_calibration \
  > detector_calibration_backup.sql
```

En el equipo nuevo se crea el stack, se ejecuta `./deploy.sh migrate` y se
restaura el respaldo. La consulta usa `line_id` como identidad operativa;
`device_id` es trazabilidad. Por ello, cambiar solamente el Jetson no obliga a
recalibrar si la linea, camara, ROI y parametros visuales siguen iguales.

Si cambia la camara, el ROI o una propiedad visual incluida en la huella de
configuracion, el detector no reutiliza la referencia y comienza una nueva
calibracion automaticamente.

## Operacion manual

```bash
curl -X POST http://127.0.0.1:8001/calibrate
curl http://127.0.0.1:8001/calibrate
```

La calibracion anterior permanece disponible hasta que la nueva alcanza la
calidad minima y se confirma la transaccion PostgreSQL.

No ejecutar `docker compose down -v` durante una actualizacion: la opcion `-v`
elimina `pgdata` y, con ello, las calibraciones y demas datos locales.
