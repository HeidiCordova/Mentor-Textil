# Release reversible del Sender externo

`sender-release-canary.js` cambia solo dos nodos del candidato validado:

- `b94f3dfa216658f3`: Sender principal;
- `90544e7288574184`: Interval exclusivo del Sender.

No modifica el contador oficial, SaveDB, credenciales ni el firewall. La
versión v2 exige seleccionar explícitamente el perfil instalado:

- `legacy`: hashes del módulo 0.0.42 original;
- `hardened`: hashes `f64477...` de `sender.js` y `5dddf9...` de
  `sender.html`.

Si módulo JS/HTML, topología, revisión, credenciales, runtime o `flows.json`
cambiaron, aborta.

## Modos

| Modo | Sender | Intervalo | Uso |
|---|---:|---:|---|
| `disabled` | apagado | 300 s | rollback seguro |
| `canary` | encendido | 30 s | confirmar el primer ACK |
| `drain` | encendido | 10 s | drenar la cola histórica |
| `steady` | encendido | 60 s | operación normal del perfil seleccionado |

La transición `canary -> drain` exige `--confirm-canary`. Esta confirmación
solo debe darse después de comprobar en MariaDB que las identidades exactas
enviadas cambiaron a `status=2, restful=1`.

## Orden obligatorio del orquestador host

1. Verificar que el host Jetson alcanza `52.11.253.25:1883`.
2. Mantener presente la regla `mentor-textile-v4-*`.
3. Ejecutar `prepare`.
4. Aplicar `canary` mientras el firewall todavía bloquea al contenedor.
5. Verificar hash del runtime, `flows.json` y blob cifrado.
6. Quitar únicamente la regla exacta respaldada por el piloto.
7. Validar ACK de `device+time` concretos y reducción de pendientes.
8. Aplicar `drain --confirm-canary`.
9. Al llegar a cero pendientes, aplicar `steady`.

Ante cualquier fallo:

1. reinsertar primero la regla exacta del firewall;
2. aplicar después el modo `disabled`;
3. verificar que el runtime volvió al hash del candidato oficial.

No usar el `rollback` del piloto v4 para detener el Sender: aquel rollback
también restauraría la lógica anterior de conteo.

## Ejemplo dentro del contenedor

El wrapper host debe copiar el helper al contenedor y establecer un directorio
privado único:

```bash
node /tmp/sender-release-canary-v2.js \
  --action prepare \
  --sender-profile hardened \
  --state-dir /data/mentor-sender-release/sender-release-YYYYMMDDTHHMMSSZ

node /data/mentor-sender-release/sender-release-*/helper.prepared.js \
  --action apply \
  --mode canary \
  --sender-profile hardened \
  --state-dir /data/mentor-sender-release/sender-release-YYYYMMDDTHHMMSSZ

node /data/mentor-sender-release/sender-release-*/helper.prepared.js \
  --action apply \
  --mode drain \
  --confirm-canary \
  --sender-profile hardened \
  --state-dir /data/mentor-sender-release/sender-release-YYYYMMDDTHHMMSSZ
```

Los globs del ejemplo son ilustrativos; el orquestador real debe usar el path
exacto guardado en su pointer, nunca resolver varios estados automáticamente.
