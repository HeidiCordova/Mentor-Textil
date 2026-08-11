# Hardening de runtime Node-RED

Este paquete resuelve dos riesgos antes de habilitar el envío cloud:

1. migra el contexto vivo de Node-RED desde memoria a `localfilesystem`;
2. instala el `Sender` endurecido, pero lo mantiene desactivado.

No modifica `flows.json`, no habilita ningún `Sender`, no publica filas y no
usa un conteo ni un `counter_epoch` hardcodeado.

## Qué preserva

Antes del primer reinicio se captura mediante la API administrativa:

- todo contexto global, flow y node que sea primitivo;
- `L1_t_disponible`;
- `L1_t_microparada`;
- `L1_t_parada_no_asignada`;
- `L1_conteo_1`;
- el contexto local de `Producción Art Atlas`.

La migración aborta si encuentra objetos, buffers, arreglos u otro valor que la
API pudiera truncar. La única excepción es
`mentor_textile_count_v4_samples`: se acepta con un esquema estricto y acotado
porque evita duplicar una frontera en curso. Cualquier otro valor complejo
exige revisión manual, no una conversión silenciosa.

El almacén queda en:

```text
/data/mentor-context/
```

con `flushInterval: 5`. Un cierre ordenado también fuerza un flush. La acción
`prove` realiza un segundo reinicio y demuestra que los acumulados no
retroceden.

## Puertas de seguridad

El paquete exige exactamente el estado auditado:

- hash canónico de runtime y `flows.json`:
  `f65c87507232c7bdd4cc4aca7a0beec607b54cd60e704d65ff634569ff5e9e88`;
- 137 nodos;
- Sender principal `b94f3dfa216658f3` desactivado;
- Sender secundario `48f1dea3eb373643` desactivado;
- broker `85b6c2e7.5ca2f` en `52.11.253.25:1883`;
- Interval `90544e7288574184` a 300 segundos;
- módulo Sender legacy con el hash auditado;
- regla `mentor-textile-v4-*` en `DOCKER-USER`;
- conectividad del host al broker y bloqueo efectivo desde el contenedor.

Si cualquiera cambia, se detiene antes de reiniciar.

## Crear el paquete en Windows

Desde `F:\Mentor-Textil`:

```powershell
& .\mentor-edge\infrastructure\nodered\production-hardening\New-ProductionHardeningBundle.ps1
```

Se generan:

```text
nodered-production-hardening-v1.tgz
nodered-production-hardening-v1.tgz.sha256
```

## Copiar al Jetson

```powershell
$jetson = "orin@192.168.1.130"
$stage = "/home/orin/mentor-nodered-hardening-v1"

ssh $jetson "mkdir -p '$stage'"

scp .\nodered-production-hardening-v1.tgz `
  .\nodered-production-hardening-v1.tgz.sha256 `
  "${jetson}:${stage}/"
```

## Extraer y validar

Ya conectado por SSH:

```bash
set -Eeuo pipefail

stage=/home/orin/mentor-nodered-hardening-v1
cd "$stage"

sha256sum -c nodered-production-hardening-v1.tgz.sha256
tar -xzf nodered-production-hardening-v1.tgz

bundle="$stage/nodered-production-hardening"
cd "$bundle"
sha256sum -c SHA256SUMS
chmod 700 deploy-runtime-hardening.sh
```

## Secuencia de despliegue

Primero, solo lectura:

```bash
sudo -v
./deploy-runtime-hardening.sh preflight
```

Después se crea respaldo, se siembra el contexto y se dejan los archivos
preparados, todavía sin reiniciar:

```bash
./deploy-runtime-hardening.sh prepare
```

`prepare` deja un seed inicial. Si ocurre un reinicio inesperado antes de la
siguiente acción, el contexto no arranca vacío.

Para activar el almacén persistente y cargar el módulo nuevo:

```bash
./deploy-runtime-hardening.sh activate
```

Esta acción vuelve a capturar el contexto justo antes de `docker restart`,
espera `healthy` y correlaciona contexto y registros Modbus.

Finalmente, demostrar persistencia con un segundo reinicio:

```bash
./deploy-runtime-hardening.sh prove
./deploy-runtime-hardening.sh status
```

El resultado final correcto contiene:

```text
PERSISTENCE_PROVED_OK
Dos reinicios validados. Sender continúa desactivado y aislado.
```

Solo después de eso debe ejecutarse el procedimiento independiente de
habilitación/canary del Sender.

El bundle incluye `release/sender-release-canary-v2.js`. Tras `prove`, crear
un estado nuevo; no reutilizar el estado v1 que registró el hash legacy:

```bash
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
release_state="/data/mentor-sender-release/hardened-$stamp"

docker cp \
  release/sender-release-canary-v2.js \
  mentor-nodered:/tmp/sender-release-canary-v2.js

docker exec mentor-nodered \
  node /tmp/sender-release-canary-v2.js \
    --action prepare \
    --sender-profile hardened \
    --state-dir "$release_state"
```

El nuevo estado vuelve a exigir baseline `disabled`, hashes JS/HTML
endurecidos, Interval auditado, igualdad runtime/disco y el mismo blob cifrado.
Los modos posteriores conservan la secuencia `canary -> drain -> steady`; las
instrucciones están en `release/README.md`.

## Lo que este paquete no resuelve

El `Sender` endurecido usa `batchSize=1` por dispositivo cuando el nodo
existente no define otro valor. Con el Interval actual de 300 segundos procesa
como máximo una fila por dispositivo en cada disparo. Esto limita el impacto,
pero el backlog auditado de 721 filas no convergerá rápidamente y las alarmas
pueden crecer si entran más rápido.

La habilitación cloud necesita una fase separada con canary, ritmo explícito,
monitoreo de ACK y un plan acotado para drenar backlog. No se debe convertir
esta migración de contexto en un envío masivo.

Además, el broker auditado continúa en MQTT `1883` sin TLS. Los ACK y la
idempotencia mejoran la entrega, pero no cifran ni autentican el transporte.
TLS, VPN o una red privada siguen siendo una deuda antes de llamarlo un canal
de producción seguro.

## Rollback

Antes del primer reinicio:

```bash
./deploy-runtime-hardening.sh abort-prepare
```

Después de activar:

```bash
./deploy-runtime-hardening.sh rollback-sender
```

El rollback posterior restaura únicamente `sender.js` y `sender.html`. Conserva
`localfilesystem`, porque volver automáticamente a contexto en memoria
reintroduciría el riesgo de perder `T_*`.

Los respaldos se guardan en:

```text
/home/orin/mentor-backups/nodered-hardening-<UTC>/
```

Ningún rollback habilita el Sender ni elimina la regla firewall.
