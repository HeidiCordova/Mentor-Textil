# Conteo textil acumulativo para Node-RED

La migración de paradas basada en movimiento está documentada en
[`MOTION_STOPS_V1.md`](MOTION_STOPS_V1.md). Forma parte del mismo parche
idempotente para que el nodo `Producción Art Atlas` no vuelva a clasificar
`fsm_state=idle` como parada.

Estado: **backend y flujo preparados para un piloto controlado; envío externo
todavía bloqueado**.

Las muestras reales de `mqtt_lecturas` confirman el contrato del receptor:
cada cinco minutos se guarda una fotografía de acumulados. Por ejemplo,
`L1_T_DISPONIBLE` aumentó de `20293` a `23321`, mientras
`L1_T_PARADA_NO_ASIGNADA` permaneció en `3208`. Por compatibilidad,
`L1_CONTEO_1` también debe ser acumulativo.

## Fuente de verdad

Una prenda se cuenta cuando la FSM emite un único evento `CORTE` al reaparecer
el separador:

```text
idle -> beige_in -> en_prenda -> cooldown -> beige_in
                                 |
                               CORTE
```

El contador técnico se define así:

```text
C(B) = C0 + cantidad de CORTE en [B0, B)
```

- `B0` es `counter_epoch`, la frontera técnica del despliegue.
- `C0` es `counter_baseline`; para este primer corte se usa `0`, pero debe
  verificarse justo antes de migrar.
- Ni calibrar, cambiar producto, reiniciar Node-RED, reiniciar el Jetson ni
  desplegar servicios cambia `B0` o reinicia el contador.
- Un reset exige un procedimiento explícito y coordinado con el receptor.

El producto no forma parte de este contador bruto. El receptor asigna producto
por tiempo y calcula la producción como diferencia entre dos acumulados. Por
eso un cambio de producto debe hacerse después de cerrar una frontera de cinco
minutos, o aceptarse que ese intervalo no podrá dividirse exactamente.

## Recorrido del dato

```text
FSM CORTE
  -> SQLite durable del detector
  -> POST /events en resiliencia
  -> PostgreSQL linea_N.vision_detections
  -> vision_counter_state
  -> GET /vision/counter?linea_id=N&until=B
  -> Node-RED escribe TOTAL en Modbus 8-9
  -> Generic In -> SaveDB -> mqtt_lecturas
  -> Sender existente
```

El primer `GET` de cada frontera, permitido desde `B+10 s`, congela el
resultado en `vision_counter_snapshots`. Repetir el mismo `B` devuelve
exactamente el mismo total. Crear por primera vez una frontera anterior a otra
ya congelada se rechaza.

Node-RED actúa como adaptador:

- no reconoce estados de la FSM;
- no incrementa ni suma pulsos;
- no calcula deltas de cinco minutos;
- no decide cuándo reiniciar;
- valida línea, epoch, frontera, `as_of`, estado y rango `UINT32`;
- escribe el total en registros Modbus `8-9`;
- conserva el mismo `B` durante los reintentos HTTP `425`;
- deja intacto el `Sender`.

## Archivos

- `deploy-textile-counter-api.patch`: API acumulativa de `resiliencia`.
- `../database/33_vision_detections_existing_lines.sql`: historial estructurado,
  estado, snapshots e invariantes para líneas existentes.
- `patch-official-textile-count.js`: parche idempotente del flujo auditado.
- `custom-nodes/.../save-db.js`: inserción transaccional e idempotente.
- `custom-nodes/.../sender.js`: versión endurecida, reservada para otra etapa.
- `production-hardening/`: migración fail-closed del contexto vivo a disco y
  actualización del Sender manteniéndolo desactivado.
- `sql/preflight_mentor_outbox.sql`: auditoría MariaDB de solo lectura.
- `sql/01_mentor_outbox_idempotency.sql`: índices de outbox, no aplicados aún.
- `REMOVE_YOLO_JETSON.md`: retiro recuperable del servicio y volumen de
  botellas del host.

## Validación local

Desde `F:\Mentor-Textil`:

```powershell
node .\mentor-edge\infrastructure\nodered\custom-nodes\node-red-contrib-services-mentor\test\save-db.test.js

node --test --test-isolation=none `
  .\mentor-edge\infrastructure\nodered\custom-nodes\node-red-contrib-services-mentor\test\sender.test.js

node .\mentor-edge\infrastructure\nodered\patch-official-textile-count.test.js

python -m unittest discover `
  -s .\mentor-edge\services\vision-event-detector\tests `
  -p "test_durable_event_output.py" -v

python -m unittest discover `
  -s .\mentor-edge\services\vision-event-detector\tests `
  -p "test_calibration_persistence.py" -v
```

Resultados actuales:

- SaveDB: `24/24`.
- Sender: `10/10`.
- parche Node-RED v4: `31/31`.
- cola durable del detector: `10/10`.
- calibración persistente: `10/10`.
- dry-run sobre el `flows.json` real: 16 cambios, hash sin modificación.

## Límites y puertas de seguridad

- `B+10 s` no es un watermark. Un `CORTE` generado antes de `B` pero entregado
  después se reflejará en el siguiente acumulado. El total converge, pero la
  atribución temporal puede desplazarse.
- La cola SQLite es acotada a 10 000 eventos. Un fallo de persistencia ahora
  detiene visiblemente el detector, pero disco lleno no permite prometer cero
  pérdida; se necesita monitoreo antes de producción.
- El detector bloquea el arranque si el reloj del sistema todavía está antes
  de 2024.
- Los acumulados `T_*` históricos de Node-RED siguen en context global. En el
  Jetson auditado `contextStorage` está comentado, por lo que pueden reiniciarse
  al recrear Node-RED.
- Solo `425` tiene reintento inmediato. Una caída de red o `5xx` omite esa fila;
  el siguiente snapshot recupera el total acumulado, no la fila faltante.
- Existen aproximadamente 476 alarmas y 80 filas de producción pendientes.
  No se reemplaza ni acelera el `Sender` durante el piloto.

Por esos dos últimos puntos, el backend puede probarse con una prenda, pero el
flujo no debe conectarse todavía al receptor externo. El procedimiento seguro
está en [`DEPLOY_JETSON.md`](./DEPLOY_JETSON.md).
