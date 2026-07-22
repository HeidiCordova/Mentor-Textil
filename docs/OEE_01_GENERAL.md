# Mentor Edge — Sistema OEE de Producción

## Visión General

Mentor Edge es una plataforma distribuida de monitoreo industrial que mide **OEE (Overall Equipment Effectiveness)** en tiempo real. Opera en dos niveles:

- **Edge** (Jetson Orin): Detección por visión artificial, cálculo local de OEE, almacenamiento offline-first.
- **Cloud** (VPS Linux): Gestión multi-tenant, dashboards analíticos, configuración centralizada, integraciones externas.
- **Tablet** (Vue.js + Capacitor): Interfaz de operador para control de línea, justificación de paradas y seguimiento de producción.

Los tres niveles se comunican mediante API REST con autenticación por API Key (edge→cloud), JWT (usuario→cloud) y SSE (Server-Sent Events) para actualizaciones en tiempo real.

---

## Principios de Diseño

| Principio | Descripción |
|-----------|-------------|
| **Offline-First** | El edge opera autónomamente; la sincronización con cloud es diferida y resiliente |
| **Arquitectura Hexagonal** | Dominio desacoplado de infraestructura en cada servicio (puertos/adaptadores) |
| **Event-Driven** | Flujo asíncrono de eventos desde la cámara hasta el cloud |
| **Multi-Tenant** | Cloud aísla datos por empresa_id; schemas por línea (`linea_{id}`) |
| **Zero-Trust** | API Key para dispositivos, JWT para usuarios, Internal Key para comunicación entre servicios |
| **Hot-Reload** | Configuración cambiadable en caliente sin reiniciar servicios |

---

## Arquitectura de Alto Nivel

```
┌──────────────────────────────────────────────────────────────────────┐
│                        JETSON ORIN (Edge)                            │
│                                                                      │
│  Cámara RTSP                                                         │
│      │                                                               │
│      ▼                                                               │
│  ┌─────────────────────┐    ┌──────────────────┐                     │
│  │ vision-event-detector│    │   yolo-counter   │                     │
│  │  (Python :8001)      │    │  (Python :8006)  │                     │
│  │  FSM + Fusion Engine │    │  YOLO + TensorRT │                     │
│  └────────┬─────────────┘    └───────┬──────────┘                     │
│           │ CORTE + OEE              │ Conteos                        │
│           ▼                          ▼                                │
│  ┌──────────────────────────────────────┐                             │
│  │          resiliencia (Go :8002)       │                             │
│  │   Buffer BD · Dedup · 6 meses ret.   │                             │
│  └────────────────┬─────────────────────┘                             │
│                   │                                                   │
│           ┌───────┴────────┐                                          │
│           ▼                ▼                                          │
│  ┌─────────────┐  ┌──────────────────┐  ┌───────────────────┐        │
│  │  enviador   │  │  edge-gateway    │  │ edge-config-service│        │
│  │ (Go :8003)  │  │  (Go :8005)      │  │ (Go :8004)         │        │
│  │ Retry exp.  │  │  API unificada   │  │ CRUD config        │        │
│  │ 6 goroutines│  │  SSE broker      │  │ Versionamiento     │        │
│  └──────┬──────┘  └───────┬──────────┘  └────────────────────┘        │
│         │                 │                                           │
│         │            ┌────┴────┐                                      │
│         │            ▼         ▼                                      │
│         │      ┌─────────┐  ┌──────────┐                              │
│         │      │ Tablet  │  │ ui-local │                              │
│         │      │ (SSE)   │  │ (:8080)  │                              │
│         │      └─────────┘  └──────────┘                              │
└─────────┼────────────────────────────────────────────────────────────┘
          │ HTTPS (API Key)
          ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          CLOUD (VPS)                                 │
│                                                                      │
│  ┌──────────────────────────────────────────────┐                    │
│  │           cloud-gateway (Go :8888)            │                    │
│  │  Router · SSE Hub · Rate Limiting · Audit     │                    │
│  └───┬──────┬──────┬──────┬──────┬──────┬───────┘                    │
│      │      │      │      │      │      │                            │
│      ▼      ▼      ▼      ▼      ▼      ▼                           │
│  identity ingest config analytics integration frontend               │
│  (:8081) (:8082) (:8083)  (:8084)  (:8085)    (:80)                 │
│                                                                      │
│  PostgreSQL 16 (schemas: identity, config, ingest, analytics, ...)   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Inventario de Servicios

### Edge (Jetson Orin)

| Servicio | Puerto | Lenguaje | Responsabilidad |
|----------|--------|----------|-----------------|
| vision-event-detector | 8001 | Python | Detección de prendas por visión (FSM 4 estados), generación de OEE snapshots |
| resiliencia | 8002 | Go | Buffer local con dedup, retención 6 meses, purga automática |
| enviador | 8003 | Go | Sincronización con cloud, retry exponencial, heartbeat |
| edge-config-service | 8004 | Go | CRUD de configuración por línea, auto-versionamiento |
| edge-gateway | 8005 | Go | API REST unificada, SSE broker, proxy a servicios internos |
| yolo-counter | 8006 | Python | Conteo por YOLO+TensorRT, correlación de marca |
| ui-local | 8080 | Vue.js/Nginx | Interfaz local para tablets |
| PostgreSQL | 5432 | PostgreSQL 14 | BD local con schemas `config` + `linea_{id}` |

### Cloud (VPS)

| Servicio | Puerto | Lenguaje | Responsabilidad |
|----------|--------|----------|-----------------|
| cloud-identity | 8081 | Go (Gin) | Autenticación, JWT, usuarios, roles, empresas |
| cloud-ingest | 8082 | Go (Gin) | Ingesta de OEE snapshots, paradas, production runs |
| cloud-config | 8083 | Go (Gin) | Plantas, líneas, dispositivos, variables, turnos, productos, categorías |
| cloud-analytics | 8084 | Go (Gin) | Dashboards, Pareto, reportes, consultas OEE/paradas |
| cloud-integration | 8085 | Go (Gin) | API keys externas, consultas OEE para terceros |
| cloud-gateway | 8888 | Go (net/http) | Router central, SSE hub, rate limiting, audit log |
| cloud-frontend | 80 | Vue 3/Nginx | SPA de administración |
| PostgreSQL | 5432 | PostgreSQL 16 | BD multi-tenant (master + DBs por planta) |

---

## Flujo de Datos Principal

### 1. Detección → Buffer → Cloud

```
Cámara RTSP → vision-event-detector (FSM + Fusion)
                    │
                    ├─ Evento CORTE (detección de prenda)
                    ├─ OEE Snapshot (cada 300s)
                    └─ Stop automático (si parada detectada)
                    │
                    ▼
              resiliencia (buffer local)
                    │
                    ▼
              enviador (retry exponencial)
                    │
                    ▼ HTTPS
              cloud-ingest → PostgreSQL (linea_X.oee_snapshots)
```

### 2. Operador → Tablet → Edge → Cloud

```
Operador interactúa con Tablet
    │
    ├─ Justifica parada: POST /edge/stops/{id}/justify
    ├─ Crea production run: POST /edge/production-runs
    ├─ Edita config: PUT /edge/config
    │
    ▼
edge-gateway → BD local (linea_{id}.stops, production_runs)
    │
    ├─ SSE broadcast a tablets conectadas
    └─ enviador sincroniza al cloud
```

### 3. Cloud → Edge (Comandos)

```
Admin en cloud crea comando (ej: APPLY_STOP, SYNC_CATALOG)
    │
    ▼
cloud-analytics → pending_commands (BD cloud)
    │
    ▼
Edge poll: GET /api/v1/edge/pending-commands
    │
    ▼
Edge ejecuta → ACK: POST /api/v1/edge/pending-commands/ack
```

---

## Cálculo OEE

El sistema calcula OEE según la fórmula estándar de manufactura:

```
OEE = Disponibilidad × Rendimiento × Calidad
```

### Variables de Entrada (Snapshot cada 300s)

| Variable | Descripción | Unidad |
|----------|-------------|--------|
| T_DISPONIBLE | Tiempo total de la ventana | ms |
| T_MICROPARADA | Tiempo en microparadas (<120s textil, <210s botellas) | ms |
| T_PARADA_NO_ASIGNADA | Tiempo en paradas sin categorizar | ms |
| T_PARADA_PROGRAMADA | Parada planificada (mantenimiento) | ms |
| T_REFRIGERIO | Tiempo de descanso obligatorio | ms |
| T_CAPACITACION_OBLIGATORIA | Tiempo de capacitación | ms |
| T_MANTENIMIENTO_PLANIFICADO | Mantenimiento planificado | ms |
| T_PARADA_NO_PROGRAMADA | Parada no planificada | ms |
| CONTEO_1 | Piezas producidas | unidades |
| MERMA | Piezas defectuosas | unidades |

### Fórmulas

```
Parada_Obligatoria = T_PARADA_PROGRAMADA + T_REFRIGERIO +
                     T_CAPACITACION_OBLIGATORIA + T_MANTENIMIENTO_PLANIFICADO

T_Disponible_Neto  = T_DISPONIBLE - Parada_Obligatoria
T_Operativo        = T_Disponible_Neto - T_PARADA_NO_PROGRAMADA - T_PARADA_NO_ASIGNADA

Disponibilidad = (T_Operativo / T_Disponible_Neto) × 100

T_Neto_Producción    = T_Operativo - T_MICROPARADA
T_Nominal_Producción = CONTEO_1 / velocidad_nominal_us
T_Pérdida_Velocidad  = T_Neto_Producción - T_Nominal_Producción

Rendimiento = ((T_Operativo - T_MICROPARADA - T_Pérdida_Velocidad) / T_Operativo) × 100

Calidad = ((CONTEO_1 - MERMA) / CONTEO_1) × 100

OEE = (Disponibilidad/100) × (Rendimiento/100) × (Calidad/100) × 100
```

### Interpretación

| Rango OEE | Clasificación |
|-----------|--------------|
| > 85% | Manufactura de clase mundial |
| 65% - 85% | Buen desempeño |
| < 65% | Requiere mejora |

---

## Modelo de Datos

### Edge: PostgreSQL 14

```
BD: mentor_edge

Schema config (compartido):
├── line_config      → 1 fila por device_id (ROI, thresholds, FSM, modo, cámara, OEE, cloud, yolo)

Schema linea_{id} (por línea):
├── stops            → Paradas (tipo, duración, justificación, fuente)
├── production_runs  → Corridas de producción (producto, SKU, inicio/fin)
├── events_buffer    → Buffer de eventos (synced, dead, retry_count, expires_at)
├── commands_buffer  → Comandos idempotentes (cloud→edge)
├── audit_log        → Auditoría (actor, acción, recurso, resultado)
├── vision_detections→ Detecciones individuales (confidence, señales)
├── oee_snapshots    → Snapshots OEE columnar (head/data)
├── stop_categories  → Categorías de parada (jerárquicas)
├── products         → Productos (SKU, nombre)
├── variables        → Variables OEE del sistema
├── velocidad_nominal→ Velocidades por producto
```

### Cloud: PostgreSQL 16

```
BD Master: mentor_planta_0
├── identity   → usuarios, empresas, roles, refresh_tokens, api_keys
├── config     → plantas, lineas, dispositivos, variables, turnos, productos,
│                categoria_paradas, canvas_oee, velocidad_nominal
├── gateway    → audit_log, commands, device_registry
├── integration→ api_keys

BD Tenant: mentor_planta_{N}  (una por planta)
├── linea_{id} → oee_snapshots, paradas, production_runs, alarmas
```

---

## Comunicación Inter-Servicios

### Edge (HTTP interno)

| Origen | Destino | Endpoint | Propósito |
|--------|---------|----------|-----------|
| gateway | resiliencia :8002 | `/buffer/summary`, `/events/*` | Consultar eventos buffered |
| gateway | config-service :8004 | `/config`, `/config/system` | CRUD de configuración |
| gateway | detector :8001 | `/calibrate`, `/camera/stream` | Calibración y streaming |
| detector | resiliencia :8002 | `POST /events` | Guardar CORTE + OEE |
| detector | gateway :8005 | `POST /stops` | Registrar paradas automáticas |
| counter | resiliencia :8002 | `POST /events` | Guardar conteos YOLO |
| enviador | cloud :8888 | `POST /events/batch` | Sincronizar al cloud |
| tablet | gateway :8005 | HTTP + SSE | Toda interacción UI |

### Cloud (HTTP interno + proxy)

| Origen | Destino | Propósito |
|--------|---------|-----------|
| gateway :8888 | identity :8081 | Autenticación JWT |
| gateway :8888 | ingest :8082 | Ingesta de eventos edge |
| gateway :8888 | config :8083 | Gestión de configuración |
| gateway :8888 | analytics :8084 | Dashboards y paradas |
| gateway :8888 | integration :8085 | APIs externas |
| analytics :8084 | gateway :8888 | `POST /internal/notify` → SSE broadcast |

---

## Autenticación

| Contexto | Método | Detalles |
|----------|--------|---------|
| Edge → Cloud | API Key | Header `X-API-Key`, validado en gateway |
| Tablet → Edge | Token local | `POST /edge/auth/login` → token de sesión |
| Tablet → Cloud | JWT | `POST /api/auth/login` → access_token + refresh_token |
| Cloud Frontend → Cloud | JWT | Almacenado en sessionStorage, refresh automático |
| Servicio → Servicio (cloud) | Internal Key | Header `X-Internal-Key` |
| API Externa → Cloud | API Key por empresa | Header `X-API-Key`, scopes: oee:read, paradas:read |

---

## Contratos Compartidos (JSON Schemas)

Los contratos entre edge y cloud están definidos en `shared-contracts/`:

| Schema | Archivo | Campos Clave |
|--------|---------|-------------|
| OEE Record | `oee.schema.json` | code, time, device_id, interval_s, v, head[], data[] |
| Stop | `stop.schema.json` | stop_id, device_id, stop_type, started_at, ended_at, justified, source |
| Event | `event.schema.json` | event_id, device_id, event_type (CORTE), confidence, signals{} |
| Command | `command.schema.json` | command_id, device_id, command_type, payload, idempotency_key, status |
| Config | `config.schema.json` | (esquema de configuración de línea) |
| Health | `health.schema.json` | (esquema de health check) |

---

## Modos de Operación

| Modo | Microparada máx. | Snapshot interval | Uso |
|------|-------------------|-------------------|-----|
| **textil** | 120s | 1800s (30 min) | Líneas de confección textil |
| **botellas** | 210s | 300s (5 min) | Líneas de embotellado |

Cada modo predefine umbrales de FSM, intervalos de OEE y tiempos de clasificación de paradas.

---

## Despliegue

### Edge (Jetson Orin)

- **Docker Compose** (`docker-compose.jetson-orin.yml`)
- PostgreSQL 14 Alpine (512M)
- Servicios Python con acceso a GPU NVIDIA
- Servicios Go con límites de 128M
- Nginx como reverse proxy (:8080 → tablet)
- Health checks automáticos

### Cloud (VPS)

- **Docker Compose** con 6 microservicios Go + frontend Vue
- PostgreSQL 16 con schemas multi-tenant
- Nginx como reverse proxy + TLS
- Rate limiting por tipo de cliente (edge: 10/s, user: 20/s)

---

## Multi-Línea

Un solo Jetson puede monitorear múltiples líneas de producción:

- 1 `edge-config-service` compartido
- 1 `edge-gateway` con múltiples `LineContext`
- 1 `vision-event-detector` por línea (parámetro `LINEA_ID`)
- 1 `yolo-counter` asignado a una línea (configurable)
- 1 `resiliencia` y 1 `enviador` con goroutines por línea
- Schemas independientes: `linea_1`, `linea_2`, etc.

La tablet accede con `?linea_id=X` para seleccionar la línea.

---

## Resiliencia y Recuperación

| Escenario | Comportamiento |
|-----------|---------------|
| Cloud caído | Buffer local crece; enviador reintenta exponencialmente (1s→64s, 8 intentos) |
| Cloud extendido offline | Marca eventos stale tras 48h, dead tras 8 intentos; emergency purge si >5GB |
| Cloud reconecta | Sync automático; heartbeat restaura device_id/empresa_id |
| Cámara desconectada | Watchdog detecta; detector en modo espera; OEE sigue computando tiempos de parada |
| Config corrupta | Config versionada; rollback posible; defaults por modo |

---

## Hot-Reload de Configuración

El sistema soporta cambios en caliente sin reiniciar servicios:

1. Operador edita configuración en tablet → `PUT /edge/config`
2. `edge-config-service` actualiza BD → trigger SQL incrementa `config_version`
3. Detector y counter hacen polling cada **5 segundos** comparando `config_version`
4. Si version > local → `GET /config` completo → aplica dinámicamente
5. Parámetros reconfigurables: ROI, thresholds, FSM, modo, cámara, intervalo OEE, modelo YOLO, intervalo sync, URL cloud
