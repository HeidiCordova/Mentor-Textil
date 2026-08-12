# Avance estimado de una prenda por tiempo activo

## Objetivo

El detector publica un avance **estimado** de la prenda actual sin convertir
el tiempo de una parada en producción. La fuente de verdad es Python, porque
allí se observan todos los frames y se confirma el evento durable `CORTE`.
Node-RED no vuelve a integrar segundos: solamente refleja el resultado en el
debug de `Producción Art Atlas` durante el piloto.

No es una medición de longitud tejida. `presence_motion` proviene de visión y
está estabilizado para evitar falsos cambios; no equivale a un encoder, PLC o
sensor eléctrico del motor.

## Fórmula

```text
ideal_cycle_s = 1 / velocidad_us

avance_estimado =
    segundos_activos_observados / ideal_cycle_s * 100

avance_publicado = min(99, avance_estimado)
```

El 100 % no sale de la fórmula. Solo se publica cuando la salida durable
acepta un `CORTE` confirmado por la cámara.

Ejemplo para una prenda ideal de 20 minutos:

```text
ideal_cycle_s = 1200 s
velocidad_us  = 1 / 1200 = 0.000833333 u/s = 3 u/h
```

Con 300 s activos el avance es 25 %. Una detención de 300 s no modifica el
valor. Al reanudarse, 300 s activos adicionales producen 50 %.

## Fuente del tiempo ideal

El detector consulta al `edge-gateway`, siempre con el `linea_id` del
detector:

1. `GET /edge/vision/count?linea_id=...` para obtener la corrida y el producto
   activos mediante el mismo criterio temporal autoritativo del contador.
2. `GET /edge/catalogs/velocidad-nominal?linea_id=...` para obtener
   `velocidad_us` del producto.

El tiempo ideal queda congelado cuando la FSM cambia de `beige_in` a
`en_prenda`. Una edición de velocidad se aplica a la siguiente prenda, no a
la que ya está en proceso. `factor_conv` no participa en esta versión porque
el cálculo OEE cloud vigente tampoco define su uso.

La frontera no reutiliza una caché anterior: abre un ciclo pendiente y solicita
un contexto autoritativo nuevo en segundo plano. Una respuesta que ya estaba en
vuelo antes de la frontera se descarta. El avance empieza en cero cuando llega
la primera respuesta posterior válida y no rellena el tiempo que no observó.

## Estados

| Estado | Significado |
|---|---|
| `unavailable` | No existe producto activo o velocidad nominal válida. |
| `waiting_cycle` | Producto listo; espera una frontera real de nueva prenda. |
| `active` | Ciclo válido y `presence_motion=true`; suma tiempo. |
| `paused` | Ciclo válido y `presence_motion=false`; congela el avance. |
| `observation_gap` | Warm-up, cámara/ROI inválidos, muestra vieja o salto temporal; no suma. |
| `completed` | `CORTE` durable aceptado; mantiene 100 % hasta la siguiente prenda. |
| `invalidated` | Cambió la corrida/producto o se reinició la FSM durante el ciclo. |

Cuando no se conoce el ciclo, `progress_estimated_pct` es `null`. El valor
`0` se reserva para una prenda cuyo inicio sí fue confirmado.

## Contrato `/status`

Campos principales:

```json
{
  "progress_version": "active-time-v1",
  "progress_signal": "presence_motion_latched",
  "progress_state": "active",
  "progress_estimated_pct": 25.0,
  "progress_valid": true,
  "progress_observable": true,
  "active_cycle_s": 300.0,
  "ideal_cycle_s": 1200.0,
  "progress_cycle_id": "...",
  "progress_run_id": "...",
  "progress_product_id": 17,
  "progress_reason": null,
  "progress_completion_event_id": null,
  "progress_completion_context_valid": null,
  "progress_last_completion_event_id": "corte-durable-anterior",
  "progress_last_completion_cycle_id": "...",
  "progress_last_completion_context_valid": true
}
```

Una muestra inválida o un hueco mayor de 2 s rompe la continuidad monotónica.
Al recuperarse, la primera muestra solo vuelve a fijar el punto de observación;
no añade el periodo desconocido.

Si llega un `CORTE` mientras producto o velocidad todavía no pudieron
resolverse, el cierre físico sí se publica como 100 %, pero con identidad e
ideal en `null` y `progress_completion_context_valid=false`. Así nunca se
atribuye la prenda nueva al producto anterior. Los campos `progress_last_*`
conservan la identidad del último cierre para que una consulta de Node-RED cada
5 s no pierda un 100 % breve entre dos prendas.

## Relación con Node-RED y la base de datos

Durante esta prueba, `Producción Art Atlas` muestra:

- `avance_estimado_pct`;
- `avance_estimado_valido`;
- `estado_avance`;
- `segundos_activos_prenda`;
- `tiempo_ideal_prenda_segundos`;
- producto, corrida y motivo de invalidez.
- identificador del último `CORTE` y validez de su contexto.

No crea un segundo cálculo, no modifica `CONTEO_1` y no escribe todavía una
variable Modbus o una quinta columna en `SaveDB`. El receptor externo y varios
canarios locales validan hoy un `head` de cuatro variables. Persistir el gauge
requiere primero versionar ese contrato, definir agregación `last` (nunca
`sum`) y acompañar el porcentaje con validez para no guardar un cero inventado.

## Límite de precisión

La configuración visual actual compara aproximadamente 30 s de historia y
mantiene la presencia hasta unos 60 s adicionales. Por ello una parada física
puede tardar hasta cerca de 90 s en hacer caer `presence_motion`. Para una
prenda ideal de 1200 s, el sesgo máximo teórico por una detención puede llegar
a 7.5 puntos porcentuales.

Si se necesita avance físico exacto, la entrada correcta es un encoder, señal
PLC o sensor de corriente del motor. Esta versión debe presentarse siempre
como `AVANCE_ESTIMADO_PCT`.
