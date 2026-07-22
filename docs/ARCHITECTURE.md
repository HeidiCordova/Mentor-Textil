# System Architecture

## Overview

Mentor Edge is a distributed industrial monitoring platform with two deployment tiers:

- **Edge** (Jetson devices): Real-time vision-based event detection, local OEE computation, offline-first buffering.
- **Cloud** (Docker on Linux VM/VPS): Multi-tenant management, analytics dashboards, centralized configuration, third-party integrations.

Both tiers communicate via a secure API gateway with SSE push and REST sync.

## Design Principles

1. **Offline-First**: Edge operates independently; cloud sync is deferred.
2. **Hexagonal Architecture**: Domain logic decoupled from infrastructure in every service.
3. **Event-Driven**: Asynchronous event flow from camera to cloud.
4. **Multi-Tenant**: Cloud schemas scoped by empresa_id; edge receives filtered config.
5. **Zero-Trust Between Tiers**: EDGE_API_KEY for device auth, JWT for users, INTERNAL_API_KEY for service-to-service.

---

## Cloud Architecture

### Service Map

| Service | Port | Language | Responsibility |
|---|---|---|---|
| cloud-gateway | 8888 | Go (net/http) | API gateway, SSE hub, rate limiting, audit log, scoped sync |
| cloud-identity | 8081 | Go (Gin) | Authentication, JWT, users, roles, empresas |
| cloud-ingest | 8082 | Go (Gin) | OEE snapshot ingestion, stops sync, production runs |
| cloud-config | 8083 | Go (Gin) | Plants, lines, devices, variables, shifts, products, categories |
| cloud-analytics | 8084 | Go (Gin) | Dashboards, Pareto, alarms, commitments, reports, OEE queries |
| cloud-integration | 8085 | Go (Gin) | Third-party API keys, external OEE/stops queries |
| cloud-frontend | 80 | Vue 3 / Nginx | SPA served through gateway catch-all |
| postgres-cloud | 5432 | PostgreSQL 16 | Shared database with schema-per-domain |

### Request Flow

```
Browser / Edge Device
        |
        v
  cloud-gateway :8888
   |  |  |  |  |  |
   |  |  |  |  |  +---> cloud-frontend :80     (catch-all)
   |  |  |  |  +------> cloud-integration :8085 (/api/integration, /api/v1/integration)
   |  |  |  +---------> cloud-analytics :8084   (/api/dashboard, /api/stops, /api/oee)
   |  |  +------------> cloud-config :8083      (/api/plantas, /api/lineas, /api/variables)
   |  +---------------> cloud-ingest :8082      (/api/v1/edge/oee, /api/v1/edge/stops-sync)
   +-------------------> cloud-identity :8081    (/api/auth, /api/usuarios)
```

### Authentication Matrix

| Path Pattern | Auth Type | Validated By |
|---|---|---|
| `/api/auth/*` | None | cloud-identity |
| `/api/v1/edge/*` | X-API-Key | gateway APIKey middleware |
| `/api/*` (remaining) | Bearer JWT | gateway JWT middleware |
| `/internal/*` | X-Internal-Key | gateway InternalKey middleware |
| `/api/v1/integration/*` | X-API-Key (per-empresa) | cloud-integration |
| `/*` (catch-all) | None | Serves frontend SPA |

### Database Schemas

```
mentor_cloud
  |-- identity    (roles, empresas, usuarios, refresh_tokens, api_keys)
  |-- config      (plantas, lineas, dispositivos, variables, turnos,
  |                productos, categoria_paradas, canvas_oee, turno_dia,
  |                velocidad_nominal, linea_productos, producto_caracteristicas)
  |-- ingest      (raw_events, oee_snapshots, device_sync_log)
  |-- analytics   (paradas, alarmas, compromisos, production_runs)
  |-- gateway     (audit_log, commands, device_registry)
  |-- integration (api_keys)
```

### SSE Real-Time Push

Edge devices connect to `GET /api/v1/edge/stream` and receive:

1. **Initial scoped sync**: All config filtered by empresa/planta/linea.
2. **Pending command replay**: Operational commands issued while device was offline.
3. **Live commands**: `SYNC_*` and operational commands pushed in real time.
4. **Heartbeat**: 30s keep-alive through proxies.

### Scoped Sync Resources

| Resource | Command Type | Source |
|---|---|---|
| categoria_paradas | SYNC_CATALOG | cloud-config |
| productos | SYNC_PRODUCTOS | cloud-config |
| turnos | SYNC_TURNOS | cloud-config |
| variables | SYNC_VARIABLES | cloud-config |
| usuarios | SYNC_USUARIOS | cloud-identity |
| plantas_lineas | SYNC_PLANTAS_LINEAS | cloud-config |
| velocidad_nominal | SYNC_VELOCIDAD_NOMINAL | cloud-config |
| paradas | SYNC_PARADAS | cloud-analytics |

### Rate Limiting

Token-bucket per client with periodic cleanup:

| Client Type | Capacity | Refill Rate |
|---|---|---|
| Edge device | 20 | 10/s |
| User (JWT) | 50 | 20/s |

Idle buckets are evicted after 30 minutes of inactivity.

---

## Edge Architecture

## Service Breakdown

### Vision Event Detector (Python)

Core vision processing service implementing hexagonal architecture.

**Domain Layer**:
- ROI Management
- Multi-modal signal extraction (Edge, Color, Flow, YOLO)
- Fusion engine with configurable weights
- Finite State Machine for event confirmation
- Automatic calibration
- Watchdog for pipeline health

**Ports**:
- FrameInput: Abstract camera interface
- ConfigPort: Configuration retrieval
- EventOutput: Event emission

**Adapters**:
- OpenCVAdapter: RTSP camera connection
- ConfigClient: HTTP client for config service
- HTTPEventAdapter: Event sender to resiliencia

**Signal Processing**:

1. **Edge Signal**: Canny edge detection with density measurement
2. **Histogram Signal**: HSV histogram correlation for color change detection
3. **Flow Signal**: Optical flow for vertical movement detection
4. **YOLO Signal**: Optional object detection (future enhancement)

**FSM States**:
- IDLE: Waiting for signal above high threshold
- DETECTING: Accumulating confirmation frames
- CONFIRMING: Event confirmed, emitting
- COOLDOWN: Anti-bounce period

### Resiliencia (Go)

Industrial-grade event buffer with guaranteed delivery.

**Features**:
- Event deduplication by UUID
- Temporal ordering preservation
- Configurable queue policies
- Sync state tracking
- Retry counter management

**Database Schema**:
```sql
events_buffer (
  event_id UUID UNIQUE,
  device_id VARCHAR,
  event_type VARCHAR,
  timestamp TIMESTAMPTZ,
  payload JSONB,
  synced BOOLEAN,
  retry_count INT
)
```

### Enviador (Go)

Cloud synchronization service with exponential backoff.

**Features**:
- Batch event transmission
- Exponential retry policy
- ACK confirmation
- Connection health monitoring
- Automatic recovery

**Retry Policy**:
- Initial delay: 2s
- Max delay: 5min
- Backoff factor: 2.0
- Max retries: 5

### Edge Config Service (Go)

Dynamic configuration manager with versioning.

**Features**:
- Real-time configuration updates
- Version tracking with auto-increment
- Validation rules enforcement
- Rollback capability
- Per-device configuration

**Validation Rules**:
- ROI dimensions > 0
- Thresholds ∈ [0, 1]
- FSM n_frames ∈ [1, 30]
- FSM cooldown ∈ [0, 60]
- Mode ∈ {textil, botellas}

### UI Local (Vue 3)

Modern web interface for monitoring and configuration.

**Views**:
- Dashboard: Real-time system status
- Configuration: Parameter adjustment and ROI editor
- Health: Detailed service diagnostics

**Technology Stack**:
- Vue 3 Composition API
- Tailwind CSS
- Axios for HTTP
- Vite for bundling

## Data Flow

```
Camera RTSP
    ↓
vision-event-detector
    ↓ (HTTP POST)
resiliencia
    ↓ (PostgreSQL)
events_buffer
    ↓ (polling)
enviador
    ↓ (HTTP POST batch)
Cloud API
```

## Communication Protocols

### Detector → Resiliencia

POST /events
```json
{
  "event_id": "uuid-v4",
  "device_id": "jetson-01",
  "event_type": "CORTE",
  "timestamp": "ISO8601",
  "payload": {
    "confidence": 0.92,
    "signals": {...}
  }
}
```

### Detector ← Config Service

GET /config?device_id=jetson-01
```json
{
  "roi": [x, y, w, h],
  "thresholds": {...},
  "fsm": {...},
  "config_version": 12
}
```

### Enviador → Cloud

POST /api/v1/edge/events
```json
{
  "device_id": "jetson-01",
  "events": [...]
}
```

Response:
```json
{
  "ack": true
}
```

## Deployment Architecture

```
                          CLOUD (Docker Compose on VPS)
 ┌─────────────────────────────────────────────────────────────┐
 │                                                             │
 │   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐  │
 │   │  cloud-      │   │  cloud-      │   │  cloud-      │  │
 │   │  identity    │   │  ingest      │   │  config      │  │
 │   │  :8081       │   │  :8082       │   │  :8083       │  │
 │   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘  │
 │          │                  │                   │          │
 │   ┌──────┴──────────────────┴───────────────────┴───────┐  │
 │   │                  cloud-gateway :8888                 │  │
 │   │         (API proxy, SSE hub, audit, CORS)           │  │
 │   └──────┬──────────────────┬───────────────────┬───────┘  │
 │          │                  │                   │          │
 │   ┌──────┴───────┐   ┌─────┴────────┐  ┌──────┴───────┐  │
 │   │  cloud-      │   │  cloud-      │  │  cloud-      │  │
 │   │  analytics   │   │  integration │  │  frontend    │  │
 │   │  :8084       │   │  :8085       │  │  :80 (nginx) │  │
 │   └──────────────┘   └──────────────┘  └──────────────┘  │
 │          │                                                 │
 │   ┌──────┴──────────────────────────────────────────────┐  │
 │   │              PostgreSQL 16  :5432                    │  │
 │   │   (identity | config | ingest | analytics |         │  │
 │   │    gateway | integration)                           │  │
 │   └─────────────────────────────────────────────────────┘  │
 └──────────────────────────────┬──────────────────────────────┘
                                │ HTTPS :8888
                                │
           ┌────────────────────┼────────────────────┐
           │                    │                    │
           ▼                    ▼                    ▼
    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
    │ Jetson #1   │     │ Jetson #2   │     │ Jetson #N   │
    │ (SSE + REST)│     │ (SSE + REST)│     │ (SSE + REST)│
    └─────────────┘     └─────────────┘     └─────────────┘
```

## Database Design

### Events Buffer

Optimized for write-heavy workload with efficient sync queries.

Indexes:
- `idx_synced`: (synced, timestamp) for enviador polling
- `idx_device_timestamp`: (device_id, timestamp) for device queries
- `idx_event_id`: (event_id) for deduplication

### Line Config

Single row per device with JSONB flexibility.

Trigger:
- Auto-increment config_version on UPDATE
- Auto-update updated_at timestamp

### Health Logs

Time-series data for service monitoring.

Index:
- `idx_service_timestamp`: (service, timestamp) for time-range queries

## Performance Considerations

### Vision Detector

- Target FPS: 10-15
- Frame processing latency: <100ms
- ROI extraction overhead: minimal
- Signal computation: parallel where possible

### Resiliencia

- Insert latency: <10ms
- Deduplication cache: 10,000 events in-memory
- Concurrent writes: supported via PostgreSQL

### Enviador

- Batch size: 50 events
- Poll interval: 5 seconds
- Network timeout: 30 seconds
- Memory footprint: minimal (no caching)

## Failure Modes & Recovery

### Camera Connection Loss

- OpenCVAdapter detects failure
- Automatic reconnection attempts (3x)
- Watchdog alerts on prolonged failure

### Database Unavailable

- Resiliencia cannot accept events
- Detector receives 500 error
- Events lost (acceptable for edge use case)

### Cloud Unreachable

- Enviador retry with exponential backoff
- Events accumulate in events_buffer
- System continues detecting normally
- Sync resumes when connection restored

### Config Service Down

- Detector uses cached configuration
- Continues operating with last known config
- Polls for config service recovery

## Security Model

### Authentication Layers

| Layer | Mechanism | Scope |
|---|---|---|
| User sessions | JWT (HMAC-SHA256) + refresh tokens | Frontend, API |
| Edge devices | EDGE_API_KEY (global) or per-device key via gateway.device_registry | SSE, ingest |
| Service-to-service | X-Internal-Key header | /internal/* endpoints |
| Third-party integrations | Per-empresa API key (hashed, scoped) | /api/v1/integration/* |

### Password Storage

- bcrypt with cost factor 12 (`gen_salt('bf', 12)`)

### Network Isolation

- Services communicate via Docker internal network.
- Only gateway port (8888) is exposed externally.
- PostgreSQL exposed on 5434 for administrative access only.

### Input Validation

- JSON schema validation on all endpoints.
- SQL injection prevention via parameterized queries (pgx).
- Range checking on numeric parameters.

## Monitoring & Observability

### Health Endpoints

All services expose `/health` with:
- Service name
- Status (ok/degraded/error)
- Uptime
- Service-specific metrics

### Logging

- Structured logging (JSON where applicable)
- Log levels: DEBUG, INFO, WARN, ERROR
- Rotation policy: size-based

### Metrics

Key metrics exposed:
- Events detected per hour
- Events pending sync
- Average detection confidence
- Service uptime
- Error rates

## Future Enhancements

1. **DeepStream Integration**: GPU-accelerated preprocessing
2. **Multi-camera Support**: Parallel processing of N cameras
3. **YOLO Integration**: Object detection for enhanced classification
4. **ML Model Updates**: Over-the-air model deployment
5. **Edge ML Training**: Federated learning capability
6. **Time-series Analytics**: On-device trend analysis
7. **Alert System**: Real-time notifications via webhook

## References

- Hexagonal Architecture: Alistair Cockburn
- Event Sourcing: Martin Fowler
- CQRS Pattern: Greg Young
- Computer Vision: OpenCV Documentation
