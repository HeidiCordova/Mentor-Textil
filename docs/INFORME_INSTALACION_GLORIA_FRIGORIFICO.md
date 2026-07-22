# INFORME TECNICO DE INSTALACION
## Sistema de Monitoreo de Energia Electrica MentorEdge
### Punto de Medicion: Tanque de Frigorificos — Gloria S.A.

---

**Referencia:** ME-ENE-Gloria-001
**Fecha de instalacion:** 24 de abril de 2026
**Elaborado por:** Equipo Tecnico MentorEdge
**Version:** 1.0

---

## 1. RESUMEN EJECUTIVO

Se realizó la instalacion del sistema de monitoreo de energia electrica MentorEdge en las instalaciones de Gloria S.A., especificamente en el tablero electrico principal del sistema de refrigeracion industrial (tanque de frigorificos). El sistema opera con el medidor multifuncion MEATROL MC60, comunicado via protocolo Modbus RTU al dispositivo edge (NVIDIA Jetson), el cual transmite datos de forma continua hacia la plataforma cloud MentorML para su analisis, visualizacion y generacion de predicciones de consumo.

La instalacion fue completada satisfactoriamente. El sistema se encuentra en produccion, registrando variables electricas con un intervalo de muestreo de 30 segundos y transmitiendo en tiempo real a la plataforma MentorML.

---

## 2. DESCRIPCION DEL PUNTO DE MEDICION

| Campo | Detalle |
|---|---|
| Cliente | Gloria S.A. |
| Planta | Planta de Produccion Gloria |
| Ubicacion del tablero | Sala de compresores — Tanque de Frigorificos |
| Alimentacion del circuito | Trifasico 380 V / 60 Hz |
| Carga monitoreada | Sistema de compresion de refrigeracion industrial |
| Criticidad del punto | Alta — operacion continua 24/7 |

El tanque de frigorificos es un activo de alta criticidad operativa. El sistema de compresion opera de forma continua y representa una fraccion significativa del consumo electrico total de la planta. La medicion en este punto permite detectar desviaciones de consumo, degradacion de equipos y oportunidades de optimizacion energetica.

---

## 3. EQUIPAMIENTO INSTALADO

### 3.1 Medidor de Energia

| Parametro | Valor |
|---|---|
| Fabricante | MEATROL |
| Modelo | MC60 |
| Tipo | Multifuncion trifasico |
| Protocolo de comunicacion | Modbus RTU (RS-485) |
| Modbus Unit ID asignado | 1 |
| Precision de medicion | Clase 0.5S |
| Rango de voltaje | 57.7 V — 400 V (fase-neutro) |
| Rango de corriente | 1 A — 6 A (entrada TC) |
| Frecuencia de red | 50 / 60 Hz |

### 3.2 Dispositivo Edge

| Parametro | Valor |
|---|---|
| Hardware | NVIDIA Jetson |
| Sistema operativo | Linux (Ubuntu 20.04 LTS) |
| Servicio de envio | energy-sender (systemd, arranque automatico) |
| Base de datos local | PostgreSQL (persistencia ante perdida de red) |
| Interfaz de configuracion | Web local (accesible desde LAN de planta) |
| Intervalo de muestreo configurado | 30 segundos |
| Protocolo de transmision cloud | HTTPS + API Key |

### 3.3 Plataforma Cloud

| Parametro | Valor |
|---|---|
| Plataforma | MentorML Cloud |
| Servicio receptor | energy-ingest |
| Servicio de analisis | cloud-analytics |
| Almacenamiento | PostgreSQL — schema energy |
| Dashboard | Modulo Monitor de Energia — MentorML Web |

---

## 4. IMAGEN DE INSTALACION EN TABLERO

> **[IMAGEN 1 — Instalacion del medidor MEATROL MC60 en tablero electrico, Sala de Compresores, Tanque de Frigorificos, Gloria S.A.]**
>
> _Insertar imagen de la instalacion fisica del medidor en el tablero._

La imagen muestra la ubicacion del medidor MEATROL MC60 dentro del tablero electrico del sistema de compresion. Se puede apreciar:

- Montaje del medidor en riel DIN dentro del tablero.
- Conexion de transformadores de corriente (TC) en las tres fases de alimentacion (L1, L2, L3).
- Cableado RS-485 (par trenzado) hacia el dispositivo edge (Jetson) ubicado en gabinete adjunto.
- Conexiones de tension directamente desde la barra de bornes del tablero.
- Correcta identificacion del cableado mediante etiquetado.

---

## 5. VARIABLES MONITOREADAS

El medidor MEATROL MC60 reporta en cada ciclo de lectura (30 s) el conjunto completo de variables electricas detallado a continuacion. Todas las variables son almacenadas en la plataforma MentorML y disponibles para consulta historica y analisis.

### 5.1 Corriente (A)

| Variable | Descripcion |
|---|---|
| I_A | Corriente fase A |
| I_B | Corriente fase B |
| I_C | Corriente fase C |
| I_AVG | Corriente promedio de las tres fases |

### 5.2 Voltaje (V)

| Variable | Descripcion |
|---|---|
| V_A | Tension fase A — neutro |
| V_B | Tension fase B — neutro |
| V_C | Tension fase C — neutro |
| V_AVG | Voltaje promedio fase-neutro |
| V_AB | Tension de linea A-B |
| V_BC | Tension de linea B-C |
| V_AC | Tension de linea A-C |

### 5.3 Potencia

| Variable | Unidad | Descripcion |
|---|---|---|
| Potencia Activa | kW | Trabajo util consumido |
| Potencia Reactiva | kVAR | Energia magnetizante (inductiva/capacitiva) |
| Potencia Aparente | kVA | Potencia total demandada |
| Factor de Potencia | adimensional (0–1) | Eficiencia de uso de la energia |

### 5.4 Frecuencia

| Variable | Unidad | Descripcion |
|---|---|---|
| Frecuencia | Hz | Frecuencia de la red electrica |

### 5.5 Energia Acumulada

| Variable | Unidad | Descripcion |
|---|---|---|
| Energia Activa | kWh | Acumulado del contador de energia util |
| Energia Reactiva | kVARh | Acumulado de energia reactiva |
| Energia Aparente | kVAh | Acumulado de energia aparente |

### 5.6 Calidad de Energia — Distorsion Armonica Total (THD)

| Variable | Descripcion |
|---|---|
| THD_IA | THD de corriente fase A |
| THD_IB | THD de corriente fase B |
| THD_IC | THD de corriente fase C |
| THD_UA | THD de voltaje fase A |
| THD_UB | THD de voltaje fase B |
| THD_UC | THD de voltaje fase C |

### 5.7 Consumo por Intervalo (calculado en plataforma MentorML)

La plataforma cloud calcula automaticamente, por cada snapshot recibido, el delta de energia respecto al registro anterior del mismo medidor mediante funcion de ventana (`LAG` con `PARTITION BY meter_id`). Esto permite conocer el consumo exacto en cada intervalo de 30 segundos sin depender de contadores externos.

| Variable calculada | Descripcion |
|---|---|
| Consumo Activa (kWh) | Delta de energia activa en el intervalo |
| Consumo Reactiva (kVARh) | Delta de energia reactiva en el intervalo |
| Consumo Aparente (kVAh) | Delta de energia aparente en el intervalo |

Los deltas se anulan automaticamente cuando se detecta un reinicio del contador del medidor, evitando valores erroneos en el historico.

---

## 6. ARQUITECTURA DEL SISTEMA

```
┌─────────────────────────────────────────────────────────┐
│  TABLERO ELECTRICO — SALA DE COMPRESORES                │
│                                                         │
│   L1 ──[TC]──┐                                          │
│   L2 ──[TC]──┼──► MEATROL MC60                         │
│   L3 ──[TC]──┘         │                               │
│              Voltaje ───┘   RS-485 (Modbus RTU)         │
└─────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │  NVIDIA Jetson      │
                    │  energy-sender      │
                    │  PostgreSQL local   │
                    │  Interfaz web local │
                    └─────────┬──────────┘
                              │ HTTPS + API Key
                    ┌─────────▼──────────────────────┐
                    │  MentorML Cloud                 │
                    │  energy-ingest (receptor)       │
                    │  cloud-analytics (analisis)     │
                    │  PostgreSQL cloud (historico)   │
                    └─────────┬──────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │  Dashboard Web      │
                    │  Monitor Energia    │
                    │  MentorML          │
                    └────────────────────┘
```

### Flujo de datos

1. El medidor MC60 acumula lecturas y las expone via Modbus RTU cada ciclo.
2. El servicio `energy-sender` en el Jetson lee el medidor, almacena el snapshot localmente y lo encola para envio.
3. Cada 30 segundos (configurable) el batch de snapshots pendientes se transmite al endpoint `energy-ingest` en la nube mediante HTTPS autenticado con API Key.
4. El servicio `energy-ingest` valida, parsea y persiste los datos en el schema `energy` de PostgreSQL cloud.
5. El servicio `cloud-analytics` expone los datos procesados al dashboard web para visualizacion en tiempo real y analisis historico.
6. Si el Jetson pierde conectividad, los registros se acumulan en la base de datos local y se sincronizan automaticamente al recuperar la red, garantizando cero perdida de datos.

---

## 7. PROCEDIMIENTO DE INSTALACION

### 7.1 Preparacion

- Revision del diagrama unifilar del tablero del sistema de refrigeracion.
- Identificacion de las tres fases de alimentacion al compresor principal.
- Verificacion de espacio disponible en riel DIN dentro del tablero.
- Seleccion de la relacion de transformacion de los TC segun la corriente nominal del compresor.

### 7.2 Montaje fisico del medidor

- Montaje del medidor MEATROL MC60 en riel DIN dentro del tablero electrico.
- Instalacion de transformadores de corriente (TC) en las fases L1, L2, L3.
- Conexion de las salidas de TC a las entradas de corriente del medidor.
- Conexion de las entradas de voltaje (L1, L2, L3, N) en la bornera del medidor.
- Verificacion de polaridad y secuencia de fases.

### 7.3 Comunicacion RS-485

- Tendido de cable RS-485 (par trenzado, apantallado) desde el medidor al gabinete del Jetson.
- Conexion en borneras A(+) y B(-) del medidor.
- Configuracion del Modbus Unit ID = 1 en el medidor via botonera frontal.
- Verificacion de terminacion de bus RS-485.

### 7.4 Configuracion del dispositivo edge

Parametros configurados en la interfaz web local del Jetson:

| Parametro | Descripcion |
|---|---|
| `device_id` | Identificador unico del dispositivo edge asignado |
| `meter_id_1` | Identificador del medidor MC60 |
| `meter_unit_id` | ID de unidad Modbus = 1 |
| `cloud_url` | Endpoint HTTPS del servicio energy-ingest en MentorML Cloud |
| `energy_api_key` | Clave de autenticacion del dispositivo hacia la nube |
| `send_interval_s` | Intervalo de envio = 30 segundos |

### 7.5 Verificacion y puesta en marcha

- Confirmacion de lectura de variables en la interfaz local del edge node.
- Verificacion de transmision exitosa al cloud (status HTTP 200 en logs del servicio).
- Confirmacion de recepcion y almacenamiento de snapshots en base de datos cloud.
- Validacion de visualizacion correcta en el dashboard web de MentorML — Modulo Monitor de Energia.

---

## 8. RESULTADOS DE VERIFICACION

| Verificacion | Resultado | Observacion |
|---|---|---|
| Lectura de variables en medidor MC60 | Conforme | Todas las variables dentro de rango esperado |
| Comunicacion Modbus RTU edge-medidor | Conforme | Sin errores de trama |
| Transmision HTTPS al cloud | Conforme | Latencia promedio < 500 ms |
| Recepcion en energy-ingest | Conforme | Snapshots almacenados correctamente |
| Calculo de consumo por intervalo | Conforme | Deltas coherentes con la carga instalada |
| Visualizacion en dashboard web | Conforme | Datos en tiempo real visibles en Monitor Energia |
| Resiliencia ante corte de red | Conforme | Acumulacion local y re-sincronizacion verificada |

---

## 9. OBSERVACIONES TECNICAS

1. **Equilibrio de fases:** Al momento de la verificacion, se observo equilibrio de corriente entre las tres fases del compresor dentro de margenes normales para motores de induccion de esta clase.

2. **Factor de potencia:** El sistema de refrigeracion opera con carga inductiva. Se recomienda revisar la necesidad de banco de condensadores una vez acumulados los primeros 7 dias de historico en la plataforma MentorML.

3. **Calidad de energia:** Los valores de THD seran monitoreados continuamente. La plataforma alertara si superan el 5 % en corriente o el 8 % en voltaje, umbrales tipicos para equipos de refrigeracion industrial.

4. **Acceso remoto:** El Jetson es accesible remotamente via SSH por el equipo tecnico MentorEdge para mantenimiento del servicio y actualizaciones de configuracion.

5. **Backup de datos:** Ante perdida de alimentacion del Jetson, los datos almacenados en PostgreSQL local son persistentes y se retransmiten al recuperar la operacion.

---

## ANEXO A — INFORME EN TIEMPO REAL Y PREDICCIONES DE CONSUMO

> **Nota:** Los datos de este anexo son generados automaticamente por la plataforma MentorML a partir del historico acumulado desde la puesta en marcha del sistema (24 de abril de 2026). Los valores se actualizan en tiempo real en el dashboard web del modulo Monitor de Energia.

---

### A.1 Lectura en Tiempo Real — Ultimo Snapshot Registrado

Los siguientes valores corresponden al ultimo snapshot procesado por la plataforma MentorML al momento de la generacion de este informe. Para consultar valores actualizados, acceder al dashboard web: **MentorML Cloud → Monitor de Energia → Tanque Frigorificos Gloria**.

| Variable | Valor registrado | Unidad |
|---|---|---|
| Corriente I_A | — | A |
| Corriente I_B | — | A |
| Corriente I_C | — | A |
| Corriente I_AVG | — | A |
| Voltaje V_A | — | V |
| Voltaje V_B | — | V |
| Voltaje V_C | — | V |
| Voltaje V_AVG | — | V |
| Potencia Activa | — | kW |
| Potencia Reactiva | — | kVAR |
| Potencia Aparente | — | kVA |
| Factor de Potencia | — | — |
| Frecuencia | — | Hz |
| Energia Activa acumulada | — | kWh |
| THD Corriente (promedio) | — | % |
| THD Voltaje (promedio) | — | % |

_Completar con captura de pantalla del dashboard MentorML o con los valores exportados directamente desde la API de cloud-analytics._

---

### A.2 Perfil de Consumo — Primeras 24 Horas de Operacion

El servicio `cloud-analytics` de MentorML calcula el consumo por intervalo en kWh para cada periodo de 30 segundos. La siguiente tabla muestra el perfil horario agregado de las primeras 24 horas de operacion continua del sistema de refrigeracion.

| Hora del dia | Consumo kWh | Potencia promedio kW | Observacion |
|---|---|---|---|
| 00:00 – 01:00 | — | — | — |
| 01:00 – 02:00 | — | — | — |
| 02:00 – 03:00 | — | — | — |
| 03:00 – 04:00 | — | — | — |
| 04:00 – 05:00 | — | — | — |
| 05:00 – 06:00 | — | — | — |
| 06:00 – 07:00 | — | — | — |
| 07:00 – 08:00 | — | — | — |
| 08:00 – 09:00 | — | — | — |
| 09:00 – 10:00 | — | — | — |
| 10:00 – 11:00 | — | — | — |
| 11:00 – 12:00 | — | — | — |
| 12:00 – 13:00 | — | — | — |
| 13:00 – 14:00 | — | — | — |
| 14:00 – 15:00 | — | — | — |
| 15:00 – 16:00 | — | — | — |
| 16:00 – 17:00 | — | — | — |
| 17:00 – 18:00 | — | — | — |
| 18:00 – 19:00 | — | — | — |
| 19:00 – 20:00 | — | — | — |
| 20:00 – 21:00 | — | — | — |
| 21:00 – 22:00 | — | — | — |
| 22:00 – 23:00 | — | — | — |
| 23:00 – 00:00 | — | — | — |
| **TOTAL DIA** | **—** | **—** | |

_Exportar desde MentorML → Monitor Energia → Exportar CSV y completar esta tabla._

---

### A.3 Prediccion de Consumo Mensual

La plataforma MentorML genera predicciones de consumo utilizando el historico de snapshots acumulados. Las proyecciones se basan en el patron de carga observado y asumen condiciones operativas similares a las del periodo de medicion.

#### Metodologia de prediccion (MentorML cloud-analytics)

- **Base de calculo:** Consumo acumulado por intervalo de 30 s (`consumo_activa` calculado via funcion `LAG` sobre `energy.snapshots`).
- **Agrupacion:** Promedio movil por hora del dia para capturar el patron circadiano del sistema de refrigeracion.
- **Proyeccion:** Extrapolacion del patron diario al horizonte mensual, con factor de ajuste por temperatura ambiental (a incorporar en version 2 del modelo).

#### Proyeccion de consumo

| Horizonte | Consumo Proyectado (kWh) | Costo estimado (S/. / kWh a tarifar) | Observacion |
|---|---|---|---|
| Dia siguiente | — | — | Basado en patron de las ultimas 24 h |
| Semana siguiente | — | — | Basado en patron de los ultimos 7 dias |
| Mes en curso | — | — | Basado en historico acumulado |
| Mes proximo | — | — | Proyeccion con patron estabilizado |

_Los valores de proyeccion seran calculados automaticamente por MentorML una vez el sistema acumule al menos 7 dias de historico continuo. Acceder al dashboard para visualizacion interactiva de tendencias._

---

### A.4 Indicadores de Eficiencia Energetica

Una vez acumulado el historico, la plataforma MentorML calculara automaticamente los siguientes indicadores de eficiencia del sistema de refrigeracion:

| Indicador | Formula | Referencia de industria |
|---|---|---|
| Factor de Potencia promedio | FP_AVG = P / S | > 0.92 (optimo para refrigeracion industrial) |
| THD de Corriente promedio | THD_I_AVG | < 5 % (norma IEEE 519) |
| Variacion de consumo diario | Desviacion estandar kWh/h | Indica estabilidad operativa |
| Consumo especifico | kWh / hora de operacion | Linea base para benchmarking futuro |

---

### A.5 Alertas Configuradas en MentorML

El modulo Monitor de Energia de MentorML genera alertas automaticas ante las siguientes condiciones detectadas en los snapshots del medidor MC60:

| Condicion | Umbral | Accion |
|---|---|---|
| Factor de potencia bajo | FP < 0.85 | Alerta en dashboard |
| Desequilibrio de corriente | Desviacion > 10 % entre fases | Alerta en dashboard |
| THD de corriente alto | THD_I > 5 % | Alerta en dashboard |
| Consumo anomalo | Desviacion > 2 sigma respecto a la media horaria | Alerta en dashboard |
| Sin datos del medidor | Sin snapshots por mas de 5 minutos | Alerta de conectividad |

---

_Informe generado por el equipo tecnico MentorEdge — 25 de abril de 2026_
_Para consultas: soporte@mentoredge.com_
