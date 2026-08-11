# Piloto Node-RED v4 en caliente

Este paquete cambia la procedencia de `L1_CONTEO_1` y migra la clasificación
de tiempos de parada desde `fsm_state=idle` hacia el contrato de movimiento
físico del detector:

```text
PostgreSQL /vision/counter
  -> Node-RED
  -> Modbus 8-9
  -> MariaDB cada 5 minutos
```

No reinicia ni recrea el contenedor Node-RED. El despliegue usa Admin API v2,
el `rev` actual y `Node-RED-Deployment-Type: nodes`. Conserva el blob cifrado
de credenciales completo.

El detector nuevo debe desplegarse primero. Antes de aplicar el candidato,
`/status` debe entregar `presence_motion`, `motion_ready`, `motion_fresh` y
`micro_stop_max_s`. Ver `MOTION_STOPS_V1.md`.

## Acciones

En el Jetson, dentro del directorio extraido:

```bash
chmod 700 nodered-pilot-v4.sh

./nodered-pilot-v4.sh prepare
./nodered-pilot-v4.sh deploy
./nodered-pilot-v4.sh status
./nodered-pilot-v4.sh validate
```

No reiniciar Jetson, Docker ni Node-RED entre `prepare` y `deploy`: durante
esa ventana el aislamiento de red es una regla `iptables` deliberadamente
temporal. Tras `deploy`, el `Sender` tambien queda deshabilitado en los flows.

Si falla una comprobacion despues del deploy:

```bash
./nodered-pilot-v4.sh rollback
```

El rollback restaura la logica previa de conteo, pero mantiene el `Sender`
externo deshabilitado. El respaldo original completo se conserva para una
restauracion posterior y controlada.

## Barreras incluidas

- respaldo de flows, credenciales, settings, MariaDB y tablas de contador;
- rechazo si existen duplicados o falta `UNIQUE(device,time)`;
- tres indices aditivos e idempotentes;
- aislamiento exacto del broker externo `52.11.253.25:1883`;
- `Sender` externo deshabilitado dentro del candidato para sobrevivir reinicios;
- parche conservador que no modifica ni reinicia `SaveDB`;
- pruebas locales y sobre los flows reales;
- `rev` optimista: un cambio concurrente produce abort, nunca force deploy;
- comparacion de ID, `StartedAt` y reinicios del contenedor;
- correlacion por frontera `B` entre API, Modbus, `mqtt_lecturas` y
  `mqtt_snapshot`;
- validacion del `mqtt_snapshot` mutable contra el outbox exacto de su propia
  frontera actual, aunque ya haya avanzado despues de `B`;
- rollback en caliente solo si el runtime aun coincide con el candidato.

## Limite

Es un piloto aislado. `SaveDB` y `Sender` siguen siendo las versiones legacy.
El `Sender` queda deshabilitado y el firewall queda puesto incluso despues de
un resultado correcto, por lo que este paquete no valida entrega al receptor
externo ni drena su backlog.

No quitar la regla del firewall ni reactivar el egreso como parte de este
piloto. Eso requiere primero endurecer `SaveDB`/`Sender` y decidir el tratamiento
de la cola historica.
