# Paradas por movimiento v1

Esta versión separa dos responsabilidades que el flujo heredado mezclaba:

```text
FSM textil (idle/beige_in/en_prenda/cooldown) -> eventos CORTE -> conteo
ROI de movimiento                              -> tiempos de parada
```

El piloto también refleja un `AVANCE_ESTIMADO_PCT`, calculado en Python con
tiempo activo y limitado a 99 % hasta un `CORTE` durable. No es una medición de
longitud ni se persiste todavía en Modbus/SaveDB. Su contrato, precisión y
límites están en [`../../../docs/09_AVANCE_ESTIMADO_PRENDA.md`](../../../docs/09_AVANCE_ESTIMADO_PRENDA.md).

## Contrato de `/status`

Node-RED solo procesa una observación cuando el detector entrega:

```json
{
  "fsm_state": "en_prenda",
  "presence_motion": true,
  "motion_ready": true,
  "motion_fresh": true,
  "motion_age_s": 0.08,
  "micro_stop_max_s": 120,
  "progress_state": "active",
  "progress_estimated_pct": 25.0,
  "progress_valid": true
}
```

- `fsm_state`: etapa del ciclo textil; no decide paradas.
- `presence_motion`: movimiento físico estabilizado en el ROI.
- `motion_ready`: la ventana lenta ya tiene suficientes frames.
- `motion_fresh`: la muestra procede de una captura reciente y válida.
- `micro_stop_max_s`: umbral efectivo configurado; es la única fuente para
  clasificar microparada frente a parada no asignada.
- `progress_*`: estimación calculada por el detector; Node-RED solo la refleja
  en el debug de Producción durante esta versión.

Si falta un campo, la cámara está calentando, la muestra está vencida o el
ROI es inválido, Node-RED congela la continuidad. Esa ausencia de evidencia no
se convierte en una parada.

## Clasificación

Con movimiento válido en `false`, Node-RED acumula un único periodo pendiente:

```text
duración < 3 s                         -> se descarta
3 s <= duración < micro_stop_max_s     -> MICROPARADA al reanudarse
duración >= micro_stop_max_s           -> PARADA_NO_ASIGNADA
```

Al cruzar el umbral, todo el periodo pendiente pasa una sola vez a
`T_PARADA_NO_ASIGNADA`. A partir de ahí solo se agrega cada delta nuevo. Cuando
vuelve el movimiento, una PNA no vuelve a registrarse como microparada.

El umbral ya no está duplicado como el literal heredado de 210 s dentro de
Node-RED. El detector lo obtiene de `oee.micro_stop_max_s` y lo publica en cada
respuesta. Los defaults actuales del repositorio son 120 s, pero el valor
efectivo depende de la configuración de la línea.

## Precisión y latch

Node-RED consulta `/status` cada 5 s. Por eso sus acumuladores tienen una
resolución aproximada de cinco segundos y una pausa completamente contenida
entre dos consultas puede no observarse. El detector Python, que procesa los
frames, sigue siendo el responsable de los registros formales de parada.

`presence_motion` usa una ventana lenta y un latch configurable para tolerar
tejido de movimiento muy lento. El tiempo del umbral empieza cuando esa señal
estable cambia a `false`, no necesariamente en el primer frame inmóvil. La
ventana, sensibilidad y hold deben calibrarse en planta.

## Corte de versión

El hot-deploy conserva los acumulados globales `T_*`, pero reinicia una sola
vez `prod_tiempo_idle_s` y `prod_pna_activa`. Esto impide que un candidato
creado por la semántica antigua de `fsm_state=idle` se reclasifique después del
cambio.

## Orden obligatorio de despliegue

1. Desplegar primero `vision-event-detector`.
2. Verificar `/status` hasta observar `motion_ready=true`,
   `motion_fresh=true` y un `micro_stop_max_s` correcto.
3. Ejecutar el `prepare` del hot-deploy Node-RED y revisar el diff.
4. Ejecutar `deploy` y luego `status`/`validate`.
5. Validar en planta: pausa corta, cruce del umbral y parada durante
   `fsm_state=en_prenda`.

El parche Node-RED es deliberadamente fail-closed. Si se despliega antes que
el detector nuevo, los tiempos `T_*` se congelan hasta que el contrato de
movimiento esté disponible; no inventa paradas.
