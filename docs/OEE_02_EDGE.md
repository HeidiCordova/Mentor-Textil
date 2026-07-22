# Mentor Edge — Documentación Detallada del Edge (OEE/Producción)

## Plataforma

- **Hardware**: NVIDIA Jetson Orin
- **SO**: JetPack (Ubuntu-based)
- **Runtime**: Docker Compose
- **Base de datos**: PostgreSQL 14 Alpine
- **Red**: Todos los servicios en `network_mode: host` o red bridge interna

---

## Servicios Edge

El edge ejecuta **7 servicios** Docker orquestados por `docker-compose.jetson-orin.yml`:

```
┌─────────────────────────────────────────────────────────────┐
│                     JETSON ORIN                              │
│                                                              │
│   ┌───────────────────┐     ┌──────────────────┐            │
│   │ vision-event-      │     │  yolo-counter    │            │
│   │ detector :8001     │     │  :8006           │            │
│   │ Python · GPU       │     │  Python · GPU    │            │
│   └────────┬───────────┘     └───────┬──────────┘            │
│            │                         │                       │
│            ▼                         ▼                       │
│   ┌──────────────────────────────────────────┐               │
│   │        resiliencia :8002 (Go)            │               │
│   └────────────────┬─────────────────────────┘               │
│                    │                                          │
│         ┌──────────┼──────────┐                              │
│         ▼          ▼          ▼                              │
│   ┌──────────┐ ┌──────────┐ ┌──────────────────┐           │
│   │ enviador │ │ gateway  │ │ config-service   │           │
│   │ :8003    │ │ :8005    │ │ :8004            │           │
│   └──────────┘ └────┬─────┘ └──────────────────┘           │
│                     │                                        │
│              ┌──────┴──────┐                                 │
│              ▼             ▼                                 │
│         ┌─────────┐ ┌──────────┐                            │
│         │ Tablet  │ │ ui-local │                            │
│         │ (SSE)   │ │ :8080    │                            │
│         └─────────┘ └──────────┘                            │
│                                                              │
│   ┌──────────────────────────────────────┐                  │
│   │       PostgreSQL 14 :5432             │                  │
│   │  config · linea_1 · linea_2 · ...    │                  │
│   └──────────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. vision-event-detector (Python, :8001)

### Propósito
Procesa el stream de video RTSP, detecta eventos de producción (paso de prenda = CORTE), rastrea paradas automáticamente y genera OEE snapshots periódicos.

### Arquitectura Hexagonal

```
app/
├── adapters/
│   ├── config_client.py         → HTTP client a config-service :8004
│   ├── http_event_adapter.py    → POST eventos a resiliencia :8002
│   ├── opencv_adapter.py        → RTSP via OpenCV
│   ├── gstreamer_adapter.py     → RTSP via GStreamer NVDEC (opcional)
│   ├── cv_image_processor.py    → Transformaciones de imagen
│   └── gateway_stop_client.py   → POST stops a gateway :8005
├── application/
│   └── detector_service.py      → Orquestación: frame → señales → FSM → output
├── domain/
│   ├── roi/roi_manager.py            → Extracción de región de interés
│   ├── signals/signal_extractors.py  → EdgeSignal, HistogramSignal, FlowSignal, BeigeSignal
│   ├── fusion/fusion_engine.py       → Combina 4 señales → score 0-1
│   ├── fsm/event_fsm.py             → Máquina de estados finitos (4 estados)
│   ├── oee/oee_aggregator.py        → Ventanas de agregación OEE
│   ├── presence/presence_detector.py → Detección de presencia continua
│   ├── stop_tracker/stop_tracker.py  → Rastreo de paradas en tiempo real
│   ├── calibration/calibrator.py     → Calibración automática de umbrales
│   └── watchdog/watchdog.py          → Monitoreo de salud de cámara
└── ports/
    ├── frame_input.py       → Interface para entrada de frames
    ├── config_port.py       → Interface para configuración
    ├── event_output.py      → Interface para emisión de eventos
    └── image_processor.py   → Interface para procesamiento de imagen
```

### Máquina de Estados Finitos (FSM)

```
    ┌─────────┐  score > high_threshold
    │  IDLE   │────────────────────────────┐
    └────▲────┘                            ▼
         │                          ┌─────────────┐
         │ score < low_threshold    │  DETECTING   │
         │                          └──────┬──────┘
         │                                 │ n_frames confirmados
         │                                 ▼
         │                          ┌─────────────┐
         │                          │  WAIT_EXIT   │ ← Emite evento CORTE
         │                          └──────┬──────┘
         │                                 │ exit_frames bajo umbral
         │                                 ▼
         │                          ┌─────────────┐
         └──────────────────────────│  COOLDOWN    │
              cooldown completado   └─────────────┘
```

**Parámetros configurables**:
| Parámetro | Default | Descripción |
|-----------|---------|-------------|
| high_threshold | 0.7 | Score mínimo para iniciar detección |
| low_threshold | 0.3 | Score máximo para cancelar/finalizar |
| n_frames | 3 | Frames consecutivos para confirmar detección |
| exit_frames | 5 | Frames consecutivos bajo umbral para salir |
| cooldown | 8 | Frames de anti-rebote post-detección |
| max_wait_exit_frames | 750 | Timeout máximo en WAIT_EXIT |

### Motor de Fusión (4 Señales)

| Señal | Algoritmo | Qué Detecta |
|-------|-----------|-------------|
| **EdgeSignal** | Canny edge detection | Densidad de bordes en ROI (aparición de objeto) |
| **HistogramSignal** | Correlación chi-square HSV | Cambio de color respecto a referencia |
| **FlowSignal** | Optical flow (magnitud) | Movimiento vertical en ROI |
| **BeigeSignal** | HSV range [15°-35°, S:30-130, V>120] | Presencia de prenda textil (color beige) |

**Fusión**: Promedio ponderado configurable → score final 0-1

### Stop Tracker

El detector rastrea paradas automáticamente:

```
producing → sin detección por micro_stop_max_s → MICROPARADA
producing → sin detección por stop_max_s       → PARADA_NO_ASIGNADA
```

- **Modo textil**: microparada < 120s, parada > 120s
- **Modo botellas**: microparada < 210s, parada > 210s
- Las paradas se registran via `POST /stops` al gateway con `source: "detector"`

### OEE Aggregator

Genera un snapshot columnar cada `snapshot_interval_s` (default 300s para botellas, 1800s para textil):

```json
{
  "code": "LINEA_1",
  "time": 1711270000000,
  "device_id": "jetson-orin-01",
  "interval_s": 300,
  "v": 5,
  "head": ["T_DISPONIBLE", "T_MICROPARADA", "T_PARADA_NO_ASIGNADA", ...],
  "data": [300000, 15000, 0, ...]
}
```

Se envía a resiliencia como evento tipo `OEE_SNAPSHOT`.

### Endpoints HTTP

| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/health` | Estado del servicio y cámara |
| GET | `/status` | Estado detallado: FSM, presencia, stop_tracker, idle_duration |
| GET | `/frame` | Último frame JPEG procesado |
| GET | `/stream` | MJPEG stream continuo (frames con ROI overlay) |
| GET | `/ws/stream` | WebSocket de frames binarios (para tablet) |
| POST | `/calibrate` | Iniciar calibración de referencia de color |
| GET | `/calibrate` | Progreso de calibración actual |

### Configuración (Variables de Entorno)

| Variable | Default | Descripción |
|----------|---------|-------------|
| `LINEA_ID` | auto-detect | ID de línea a monitorear |
| `DEVICE_ID` | auto-detect | ID del dispositivo Jetson |
| `CONFIG_SERVICE_URL` | http://localhost:8004 | URL del config service |
| `RESILIENCIA_URL` | http://localhost:8002 | URL del buffer |
| `GATEWAY_URL` | http://localhost:8005 | URL del gateway |
| `CAMERA_URL` | desde DB | URL RTSP de la cámara |
| `FRAME_BACKEND` | opencv | opencv \| gstreamer |
| `FRAME_SKIP` | 2 | Procesar 1 de cada N frames |

---

## 2. yolo-counter (Python, :8006)

### Propósito
Cuenta productos usando YOLO + TensorRT en GPU NVIDIA. Detecta objetos cruzando una línea de conteo configurable y opcionalmente identifica marca por color de tapa.

### Arquitectura

```
app/
├── adapters/
│   ├── config_client.py        → Polling de config cada 5s
│   ├── http_event_adapter.py   → POST conteos a resiliencia
│   └── opencv_adapter.py       → RTSP camera client
├── application/
│   └── counter_service.py      → frame → YOLO → LineCounter → output
├── domain/
│   ├── yolo_engine.py          → DetectionEngine con caché de modelos
│   ├── line_counter.py         → Lógica de conteo por cruce de línea
│   ├── roi_manager.py          → Extracción ROI
│   └── count_aggregator.py     → Snapshots OEE periódicos
└── http_server.py              → Lab MJPEG stream
```

### Algoritmo de Conteo

1. Lee frame cada `frame_skip` iteraciones (default 2 → ~15 fps)
2. YOLO detecta objetos en ROI con modelo configurado
3. `LineCounter` rastrea centroide de cada detección frame a frame
4. Si el centroide cruza la línea de conteo → incrementa contador
5. Dirección configurable: `top_to_bottom` | `left_to_right`
6. Correlación de marca opcional via `cap_colors` (color de tapa → SKU)

### Configuración YOLO

| Parámetro | Default | Descripción |
|-----------|---------|-------------|
| model_name | yolo26s | Modelo: yolo26s, yolo26m, yolo26l, custom |
| confidence | 0.45 | Umbral de confianza mínima |
| use_tensorrt | true | Aceleración GPU NVIDIA |
| counting_line_y | 0.5 | Posición Y de línea de conteo (ratio 0-1) |
| counting_direction | top_to_bottom | Dirección de conteo |
| cap_colors | {} | Mapa color→marca ({"rojo": "LOA", "celeste": "CIELO"}) |
| assigned_linea_id | null | Línea asignada (null = modo espera) |

### Modo Espera
Si no hay `assigned_linea_id`, el counter arranca sin abrir cámara y sin contar. Espera a que un admin asigne una línea via config.

### Endpoints HTTP

| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/health` | Estado del servicio |
| GET | `/lab/stream` | MJPEG stream con bounding boxes dibujadas |
| GET | `/lab/detections` | JSON con últimas detecciones |
| POST | `/lab/line?y=0.5&direction=top_to_bottom` | Ajustar línea de conteo (preview, no persiste) |

---

## 3. resiliencia (Go, :8002)

### Propósito
Buffer local en BD con tolerancia industrial a fallos. Garantiza que ningún evento de producción se pierda aunque el cloud esté caído por días.

### Arquitectura Hexagonal

```
internal/
├── adapters/
│   ├── http_server.go      → REST API
│   └── postgres_repo.go    → Read/write events_buffer
├── application/
│   └── buffer_service.go   → Lógica de buffer, dedup, mantenimiento
├── domain/
│   ├── event_buffer.go     → EventBuffer struct
│   ├── dedup_policy.go     → InMemoryDedup (últimos 10k event_id)
│   └── queue_policy.go     → Aceptar/rechazar según condiciones
└── ports/
    ├── event_storage.go    → Interface EventStorage
    └── dedup.go            → Interface Dedup
```

### Tabla: `linea_{id}.events_buffer`

| Columna | Tipo | Descripción |
|---------|------|-------------|
| event_id | UUID UNIQUE | Identificador único del evento |
| device_id | VARCHAR | Dispositivo origen |
| event_type | VARCHAR | CORTE, OEE_SNAPSHOT, etc. |
| timestamp | TIMESTAMPTZ | Momento del evento |
| payload | JSONB | Datos del evento |
| synced | BOOLEAN | ¿Sincronizado con cloud? |
| dead | BOOLEAN | ¿Agotados los reintentos? |
| retry_count | INT | Número de intentos de sync |
| expires_at | TIMESTAMPTZ | Auto-purga (NOW + 6 meses) |

**Índices optimizados**:
- `idx_events_pending`: WHERE synced=false AND dead=false
- `idx_events_synced`: (synced, timestamp)
- `idx_events_device_ts`: (device_id, timestamp)
- `idx_events_expiry`: WHERE synced=true OR dead=true

### Políticas

**Deduplicación**: Hash set en memoria con últimos 10,000 event_id. Rechaza duplicados silenciosamente.

**Queue Policy**: Puede rechazar eventos si hay >100k pendientes (protección contra overflow).

**Mantenimiento automático** (cada hora):
1. Purga eventos expirados (> 6 meses)
2. Marca como dead eventos stale (> 48h sin sync)
3. Emergency purge si disco > 5GB (mantiene 10k más recientes)
4. Log de estadísticas

### Endpoints HTTP

| Método | Path | Descripción |
|--------|------|-------------|
| POST | `/events` | Guardar evento (dedup + queue policy) |
| GET | `/events/summary` | {total, pending, synced, dead, disk_bytes} |
| GET | `/events/recent?limit=50&since=...` | Últimos eventos (incluye synced) |
| GET | `/events/pending?limit=500` | Eventos por enviar al cloud |
| POST | `/events/purge` | Purga manual (uso excepcional) |
| GET | `/health` | Estado del servicio |

---

## 4. enviador (Go, :8003)

### Propósito
Sincroniza eventos del buffer local con el cloud. Gestiona 6 goroutines paralelas con retry exponencial y heartbeat.

### Arquitectura Hexagonal

```
internal/
├── adapters/
│   ├── health_server.go       → GET /health
│   ├── http_cloud_client.go   → POST a cloud con retry
│   └── postgres_reader.go     → SELECT eventos, config
├── application/
│   └── sender_service.go      → Orquestación del sync
├── domain/
│   ├── retry_policy.go        → Exponencial: 1s, 2s, 4s, 8s... max 64s
│   └── sync_policy.go         → Batch size + poll interval configurable
└── ports/
    ├── event_storage.go       → Interface EventStorage
    └── cloud_client.go        → Interface CloudClient
```

### 6 Goroutines

| Goroutine | Intervalo | Descripción |
|-----------|-----------|-------------|
| **Main sync** | configurable (default 300s) | Fetch batch → enviar → marcar synced/dead |
| **Config refresh** | 30s | Recarga sync_interval_s, cloud_url, cloud_api_key |
| **Commands** | 3s | Poll y ejecutar comandos pendientes del cloud |
| **Stops sync** | 3s | Sincronizar paradas justificadas |
| **Production runs sync** | 3s | Sincronizar corridas de producción |
| **Heartbeat** | 60s | `POST /heartbeat` a cloud, auto-sync device_id/empresa_id |

### Retry Policy

```
Intento 1: espera 1s
Intento 2: espera 2s
Intento 3: espera 4s
Intento 4: espera 8s
Intento 5: espera 16s
Intento 6: espera 32s
Intento 7: espera 64s
Intento 8: espera 64s (cap)
> 8 intentos: marca evento como dead (no reintenta más)
```

### Health Server

`GET /health` responde `OK` **solo si el último heartbeat al cloud fue exitoso**. Esto permite que herramientas de monitoreo detecten desconexión del cloud.

### Comunicación con Cloud

```
POST /api/v1/edge/oee                  → Batch de OEE records
POST /api/v1/edge/stops-sync           → Sincroniza paradas
POST /api/v1/edge/production-runs-sync → Sincroniza production runs
POST /api/v1/edge/heartbeat            → Latido + scope resolution
GET  /api/v1/edge/pending-commands     → Poll comandos cloud→edge
POST /api/v1/edge/pending-commands/ack → ACK de ejecución
```

Headers requeridos: `X-API-Key`, `X-Device-ID`, `X-Linea-ID`

---

## 5. edge-gateway (Go, :8005)

### Propósito
API REST unificada que actúa como **punto de entrada único**. Tablet y cloud **nunca** llaman a servicios internos directamente. Soporta múltiples líneas en un solo proceso.

### Arquitectura Hexagonal

```
internal/
├── adapters/
│   ├── http_server.go           → Router y handlers (32+ endpoints)
│   ├── postgres_*_repo.go       → 10+ repos por tabla
│   ├── service_clients.go       → HTTP clients a servicios internos
│   ├── sse_broker.go            → Server-Sent Events para UI
│   ├── config_client.go         → Proxy a edge-config-service
│   ├── camera_push.go           → Push RTSP a cloud
│   ├── system_metrics.go        → CPU, memoria, disco
│   └── provision_handler.go     → Setup de nueva línea
├── application/
│   ├── gateway_service.go       → Orquestación de stops, runs, config
│   ├── command_service.go       → Aplicar comandos idempotentes
│   └── status_service.go        → Health agregado de todos los servicios
├── domain/
│   ├── stop.go                  → Stop struct, enums, validaciones
│   ├── production_run.go        → ProductionRun struct
│   ├── command.go               → Command struct con idempotency_key
│   ├── audit.go                 → AuditEntry struct
│   └── event.go                 → Event struct para buffer
└── ports/
    ├── stop_repo.go             → Interface StopRepository
    ├── production_run_repo.go   → Interface ProductionRunRepository
    ├── command_repo.go          → Interface CommandRepository
    ├── config_client.go         → Interface ConfigClient
    ├── buffer_client.go         → Interface BufferClient
    ├── detector_client.go       → Interface DetectorClient
    └── sse_broker.go            → Interface SSEBroker
```

### Endpoints Completos

#### Status y Health
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/health` | {service, status, uptime, device_id, deps{}} |
| GET | `/edge/status` | {status, cloud_url, device_id, cloud_connected} |

#### Configuración (proxy a config-service :8004)
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/config?linea_id=X` | Config completa de línea |
| PUT | `/edge/config?linea_id=X` | Actualizar config (ROI, thresholds, FSM, etc.) |
| GET | `/edge/config/system` | Defaults globales |
| GET | `/edge/config/version?linea_id=X` | Versión actual |

#### Paradas (Stops)
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/stops` | Lista de paradas con filtros |
| POST | `/edge/stops` | Crear parada manual (source: operator) |
| GET | `/edge/stops/{stop_id}` | Parada individual |
| PUT | `/edge/stops/{stop_id}` | Actualizar parada |
| POST | `/edge/stops/{stop_id}/justify` | Justificar parada |
| DELETE | `/edge/stops/{stop_id}` | Eliminar parada |
| GET | `/edge/stops/summary` | Resumen por tipo de parada |

#### Production Runs
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/production-runs` | Lista de corridas |
| POST | `/edge/production-runs` | Crear/upsert corrida |
| GET | `/edge/production-runs/{run_id}` | Corrida individual |
| PUT | `/edge/production-runs/{run_id}` | Actualizar corrida |
| DELETE | `/edge/production-runs/{run_id}` | Eliminar corrida |

#### Eventos y Buffer
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/events/recent?limit=200` | Últimos eventos de detección |
| GET | `/edge/events/pending` | Eventos por sincronizar |
| GET | `/edge/buffer/summary` | {total, pending, synced, dead} |

#### Catálogos
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/catalogs/stop-categories` | Categorías de parada (árbol jerárquico) |
| GET | `/edge/catalogs/products` | Productos (SKU, nombre) |
| GET | `/edge/catalogs/velocidad-nominal` | Velocidades nominales por producto |
| PUT | `/edge/catalogs/velocidad-nominal` | Actualizar velocidad nominal |
| GET | `/edge/catalogs/sync/turnos` | Turnos de trabajo |

#### Comandos
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/commands` | Comandos pendientes |
| POST | `/edge/commands` | Registrar comando (requiere idempotency_key) |

#### Streaming y Cámara
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/camera/stream` | MJPEG stream (proxy a detector :8001) |
| GET | `/edge/camera/health` | Health check de conexión de cámara |

#### Autenticación
| Método | Path | Descripción |
|--------|------|-------------|
| POST | `/edge/auth/login` | Login con operador local |
| GET | `/edge/auth/operators` | Lista de operadores configurados |

#### Calibración
| Método | Path | Descripción |
|--------|------|-------------|
| POST | `/edge/calibration/start?linea_id=X` | Iniciar calibración |
| GET | `/edge/calibration/status` | Estado actual de calibración |

#### Variables y Turno
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/variables` | Variables OEE del sistema |
| GET | `/edge/current-turno` | Turno actual según configuración |

#### SSE (Real-time)
| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/edge/stream` | Server-Sent Events para actualización en tiempo real |

### Multi-Línea

El gateway soporta múltiples líneas en un solo proceso:

```go
// 1 mux HTTP, N contextos de línea
lines := map[int]*LineContext{}  // linea_id → context

// Resolución de línea en cada request:
// 1. Busca ?linea_id=X en parámetro
// 2. Si no: busca en cloud_linea_id mapping
// 3. Si no: usa defaultLineaID
```

### SSE Broker

Broadcast de eventos en tiempo real a todas las tablets conectadas:

- `event.created` → Nueva detección de prenda
- `stop.changed` → Parada creada, justificada o cerrada
- `stop_created`, `stop_closed`, `stop_deleted` → Eventos granulares
- `stops_synced` → Sincronización batch completada
- `catalogs_synced` → Catálogo actualizado desde cloud
- `production_runs_updated` → Corrida de producción modificada
- `config.updated` → Configuración cambiada

### Idempotencia

Todos los comandos requieren `idempotency_key`. El gateway verifica en `commands_buffer` si el comando ya fue ejecutado antes de procesarlo, evitando duplicaciones.

### Auditoría

Cada acción registra en `linea_{id}.audit_log`:
```json
{
  "actor": "operator:juan",
  "action": "justify_stop",
  "resource": "stop",
  "resource_id": "uuid-...",
  "payload": {"reason": "Cambio de formato", "categoria_id": 5},
  "result": "success",
  "timestamp": "2026-03-24T10:30:00Z"
}
```

---

## 6. edge-config-service (Go, :8004)

### Propósito
Autoridad central para la configuración de líneas. Evita que servicios lean configs inconsistentes.

### Arquitectura Hexagonal

```
internal/
├── adapters/
│   ├── http_server.go      → Endpoints REST
│   └── postgres_repo.go    → Read/write config.line_config
├── application/
│   └── config_service.go   → Lógica de negocio
├── domain/
│   ├── config_model.go     → Structs: LineConfig, ROI, Thresholds, FSM, etc.
│   └── errors.go
└── ports/
    └── storage.go          → Interface ConfigStorage
```

### Tabla: `config.line_config`

Una fila por `device_id` con campos JSONB:

| Campo | Tipo | Contenido |
|-------|------|-----------|
| device_id | VARCHAR(64) PK | Identificador del Jetson |
| config_version | INT | Auto-incrementado por trigger SQL |
| roi | JSONB | {x, y, width, height, bottom_margin} |
| thresholds | JSONB | {edge, color, flow, dy, beige, high, low} |
| fsm | JSONB | {n_frames, cooldown, exit_frames, max_wait_exit_frames} |
| mode | VARCHAR | textil \| botellas \| custom |
| camera | JSONB | {url, frame_backend, frame_skip, signal_scale} |
| oee | JSONB | {line_name, micro_stop_max_s, stop_max_s, snapshot_interval_s, vel_unit} |
| cloud | JSONB | {sync_interval_s, cloud_url, cloud_api_key} |
| tablet | JSONB | {config_url} |
| yolo | JSONB | {model_name, confidence, use_tensorrt, counting_line_y, counting_direction, cap_colors, assigned_linea_id} |

### Endpoints

| Método | Path | Descripción |
|--------|------|-------------|
| GET | `/config?linea_id=X` | Config completa de línea |
| PUT | `/config?linea_id=X` | Actualizar (parcial o completa) |
| GET | `/config/system` | Defaults globales (cloud, OEE, yolo) |
| PUT | `/config/system` | Actualizar defaults |
| GET | `/config/version?linea_id=X` | Solo el número de versión |
| GET | `/config/lines` | Array de linea_id configuradas |
| GET | `/config/device-id` | Device ID actual |
| PUT | `/config/device-id` | Actualizar device ID |
| POST | `/calibration/start?linea_id=X` | Proxy inicio calibración |
| GET | `/health` | Estado del servicio |

### Versionamiento Automático

```sql
-- Trigger SQL que incrementa config_version en cada UPDATE
CREATE OR REPLACE FUNCTION config.increment_version()
RETURNS TRIGGER AS $$
BEGIN
  NEW.config_version := OLD.config_version + 1;
  NEW.updated_at := NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

Esto permite que detector y counter detecten cambios comparando solo un entero cada 5 segundos.

---

## Base de Datos Edge

### Schema `config` (compartido entre servicios)

```sql
-- Configuración de línea (1 fila por device_id)
config.line_config (
    device_id       VARCHAR(64) PRIMARY KEY,
    config_version  INT DEFAULT 1,
    roi             JSONB,
    thresholds      JSONB,
    fsm             JSONB,
    mode            VARCHAR(32),
    camera          JSONB,
    oee             JSONB,
    cloud           JSONB,
    tablet          JSONB,
    yolo            JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### Schema `linea_{id}` (uno por línea de producción)

```sql
-- Paradas de producción
stops (
    id              BIGSERIAL PRIMARY KEY,
    stop_id         UUID UNIQUE,
    device_id       VARCHAR(64),
    stop_type       VARCHAR(32),      -- MICROPARADA, PARADA_NO_ASIGNADA, PROGRAMADA, etc.
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,      -- NULL si abierta
    duration_ms     BIGINT,
    justified       BOOLEAN DEFAULT FALSE,
    reason          TEXT,
    category        VARCHAR(128),
    categoria_id    INT,
    justified_by    VARCHAR(128),
    justified_at    TIMESTAMPTZ,
    source          VARCHAR(32),      -- detector, operator, cloud, system
    synced          BOOLEAN DEFAULT FALSE,
    synced_at       TIMESTAMPTZ
);

-- Corridas de producción
production_runs (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID UNIQUE,
    device_id       VARCHAR(64),
    linea_id        INT,
    producto_id     INT,
    sku             VARCHAR(64),
    nombre          VARCHAR(256),
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    synced          BOOLEAN DEFAULT FALSE,
    synced_at       TIMESTAMPTZ
);

-- Buffer de eventos (corazón de resiliencia)
events_buffer (
    id              BIGSERIAL PRIMARY KEY,
    event_id        UUID UNIQUE,
    device_id       VARCHAR(64),
    event_type      VARCHAR(64),      -- CORTE, OEE_SNAPSHOT, COUNT
    timestamp       TIMESTAMPTZ,
    payload         JSONB,
    synced          BOOLEAN DEFAULT FALSE,
    dead            BOOLEAN DEFAULT FALSE,
    retry_count     INT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    expires_at      TIMESTAMPTZ DEFAULT NOW() + INTERVAL '6 months'
);

-- Comandos idempotentes (cloud → edge)
commands_buffer (
    id              SERIAL PRIMARY KEY,
    command_id      UUID UNIQUE,
    device_id       VARCHAR(64),
    command_type    VARCHAR(64),
    payload         JSONB,
    status          VARCHAR(16),      -- RECEIVED, APPLIED, FAILED
    issued_by       VARCHAR(128),
    issued_at       TIMESTAMPTZ,
    idempotency_key VARCHAR(256) UNIQUE,
    executed_at     TIMESTAMPTZ,
    fail_reason     TEXT
);

-- Auditoría completa
audit_log (
    id              BIGSERIAL PRIMARY KEY,
    device_id       VARCHAR(64),
    actor           VARCHAR(128),
    action          VARCHAR(64),
    resource        VARCHAR(64),
    resource_id     VARCHAR(128),
    payload         JSONB,
    result          VARCHAR(32),
    timestamp       TIMESTAMPTZ DEFAULT NOW()
);

-- Detecciones de visión (historial detallado)
vision_detections (
    id              SERIAL PRIMARY KEY,
    detection_id    UUID UNIQUE,
    device_id       VARCHAR(64),
    detected_at     TIMESTAMPTZ,
    line_code       VARCHAR(32),
    confidence      REAL,
    signal_edge     REAL,
    signal_color    REAL,
    signal_flow     REAL,
    signal_beige    REAL,
    roi_id          VARCHAR(64),
    fsm_state       VARCHAR(32)
);

-- Snapshots OEE
oee_snapshots (...);

-- Catálogos sincronizados desde cloud
stop_categories (...);
products (...);
variables (...);
velocidad_nominal (...);
```

---

## Docker Compose (Jetson Orin)

```yaml
# docker-compose.jetson-orin.yml (esquema simplificado)
services:
  postgres:
    image: postgres:14-alpine
    ports: ["5432:5432"]
    mem_limit: 512m
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]

  vision-event-detector:
    build: ./services/vision-event-detector
    network_mode: host         # GPU access
    runtime: nvidia
    deploy:
      resources:
        reservations:
          devices:
            - capabilities: [gpu]

  yolo-counter:
    build: ./services/yolo-counter
    network_mode: host
    runtime: nvidia

  resiliencia:
    build: ./services/resiliencia
    ports: ["8002:8002"]
    mem_limit: 128m
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8002/health"]

  enviador:
    build: ./services/enviador
    ports: ["8003:8003"]
    mem_limit: 128m
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8003/health"]

  edge-config-service:
    build: ./services/edge-config-service
    ports: ["8004:8004"]
    mem_limit: 128m

  edge-gateway:
    build: ./services/edge-gateway
    ports: ["8005:8005"]
    mem_limit: 128m

  ui-local:
    build: ./services/ui-local
    ports: ["8080:80"]         # Nginx serving Vue.js SPA

  nginx:
    image: nginx:alpine
    ports: ["80:80"]           # Reverse proxy → ui-local + gateway
```

### Recursos del Sistema

| Servicio | Memoria | GPU | Health Check |
|----------|---------|-----|-------------|
| PostgreSQL | 512M | No | pg_isready |
| vision-event-detector | 2G | Sí (NVIDIA) | Implícito |
| yolo-counter | 2G | Sí (NVIDIA) | Implícito |
| resiliencia | 128M | No | wget /health |
| enviador | 128M | No | wget /health |
| edge-config-service | 128M | No | — |
| edge-gateway | 128M | No | — |
| ui-local | 64M | No | — |

---

## Flujos de Negocio Clave

### Detección de Prenda (CORTE)

1. Cámara RTSP → `vision-event-detector` lee frame
2. Extrae ROI → procesa con 4 señales → fusion score 0-1
3. FSM: IDLE → DETECTING (n_frames) → WAIT_EXIT → COOLDOWN
4. Genera evento CORTE con confidence y señales
5. `POST /events` a resiliencia → guarda en `events_buffer`
6. Enviador lee batch pendiente → `POST /api/v1/edge/oee` a cloud
7. Cloud confirma → resiliencia marca `synced=true`

### Justificación de Parada

1. Detector detecta idle > threshold → registra parada automática (`source: detector`)
2. Gateway SSE broadcast → tablet muestra parada abierta
3. Operador selecciona categoría del árbol jerárquico
4. Tablet `POST /edge/stops/{id}/justify` con {reason, categoria_id, justified_by}
5. Gateway actualiza BD → audita → SSE broadcast
6. Enviador sincroniza al cloud

### Hot-Reload de Configuración

1. Operador en tablet → `PUT /edge/config` → config-service
2. Trigger SQL incrementa `config_version`
3. Detector/counter polling cada 5s: `GET /config/version`
4. Si version > local → `GET /config` completo → aplica en caliente
5. Parámetros reconfigurables sin reiniciar: ROI, thresholds, FSM, modo, cámara, OEE interval, modelo YOLO

### Calibración Asistida

1. Operador remueve prenda del ROI
2. `POST /edge/calibration/start` → detector inicia captura de referencia
3. Captura 30 frames para referencia de color
4. Tablet pollea progreso via `GET /calibrate`
5. Al completar → guarda nuevos thresholds en BD → version++ → auto-reload
