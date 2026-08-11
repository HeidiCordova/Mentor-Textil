# Arquitectura Detallada de Mentor Edge

## Tabla de Contenidos

1. [Visión General](#visión-general)
2. [Arquitectura Edge](#arquitectura-edge)
3. [Microservicios Edge](#microservicios-edge)
4. [Arquitectura Cloud](#arquitectura-cloud)
5. [Flujos de Datos](#flujos-de-datos)
6. [Patrones de Concurrencia](#patrones-de-concurrencia)
7. [Base de Datos](#base-de-datos)
8. [Comunicación Edge-Cloud](#comunicación-edge-cloud)
9. [Despliegue](#despliegue)

---

## Visión General

**Mentor Edge** es una plataforma industrial distribuida para monitoreo de producción en tiempo real mediante visión artificial. Opera en dos capas:

- **Edge Layer** (Jetson Orin): Procesamiento local offline-first con buffer confiable
- **Cloud Layer** (VPS Linux): Gestión multi-tenant, analytics y dashboards centralizados

### Principios de Diseño

1. **Offline-First**: Edge opera independientemente; sincronización diferida al cloud
2. **Hexagonal Architecture**: Lógica de dominio desacoplada de infraestructura
3. **Event-Driven**: Flujo asíncrono de eventos desde cámara hasta cloud
4. **Multi-Tenant**: Esquemas cloud por empresa_id; edge recibe config filtrada
5. **Zero-Trust**: EDGE_API_KEY para dispositivos, JWT para usuarios, INTERNAL_API_KEY entre servicios

---

## Arquitectura Edge

### Topología de Red

```
┌─────────────────────────────────────────────────────────────┐
│                    Jetson Orin Nano/AGX                     │
├─────────────────────────────────────────────────────────────┤
│  Camera RTSP (192.168.x.x:554)                              │
│     ↓                                                        │
│  vision-event-detector :8001 (network_mode: host)           │
│     ↓                                                        │
│  edge-gateway :8005 ← tablet/browser                        │
│     ↓                                                        │
│  resiliencia :8002 → PostgreSQL :5432 → enviador :8003      │
│     ↓                                                        │
│  edge-config-service :8004                                  │
│     ↓                                                        │
│  ui-local :8080 (interfaz técnico local)                    │
└─────────────────────────────────────────────────────────────┘
         ↓ HTTPS
    Cloud Gateway :8888
```

### Stack de Contenedores

```yaml
services:
  postgres:        alpine:15
  resiliencia:     golang:1.22-alpine
  enviador:        golang:1.22-alpine
  edge-config:     golang:1.22-alpine
  edge-gateway:    golang:1.22-alpine
  vision-detector: ubuntu:22.04 + OpenCV
  ui-local:        nginx:alpine
```

---

## Microservicios Edge

### 1. vision-event-detector

**Lenguaje**: Python 3.10  
**Puerto**: 8001  
**Network**: host  
**GPU**: NVIDIA CUDA required  

#### Responsabilidades

- Captura de video RTSP en tiempo real
- Procesamiento de múltiples señales de visión
- Detección de eventos industriales (corte textil, producción)
- Gestión de ROIs configurables
- Calibración automática de thresholds
- Tracking de paradas con FSM

#### Arquitectura Hexagonal

```
domain/
  ├── roi_manager.py          # Gestión de ROIs
  ├── signal_processor.py     # 4 señales: Edge, HSV, Flow, Beige
  ├── fusion_engine.py        # Combinación de señales con pesos
  ├── event_detector.py       # FSM (IDLE → DETECTING → CONFIRMING → COOLDOWN)
  └── stop_tracker.py         # Tracking de paradas industriales

ports/
  ├── frame_input.py          # ABC para captura de frames
  ├── config_port.py          # ABC para config dinámica
  └── event_output.py         # ABC para envío de eventos

adapters/
  ├── opencv_adapter.py       # Implementación RTSP con OpenCV
  ├── config_client.py        # HTTP a edge-config-service
  ├── http_event_adapter.py   # POST a resiliencia
  └── gateway_client.py       # Cliente para edge-gateway

application/
  └── detector_service.py     # Loop principal + watchdog
```

#### Señales de Visión

##### 1. Edge Signal
```python
# Detección de bordes con Canny
edges = cv2.Canny(gray, threshold1=50, threshold2=150)
edge_density = np.count_nonzero(edges) / (h * w)
```

##### 2. Histogram Signal
```python
# Correlación de histogramas HSV
hsv = cv2.cvtColor(frame, cv2.COLOR_BGR2HSV)
hist = cv2.calcHist([hsv], [0, 1], None, [50, 60], [0, 180, 0, 256])
cv2.normalize(hist, hist, 0, 1, cv2.NORM_MINMAX)
correlation = cv2.compareHist(hist, baseline, cv2.HISTCMP_CORREL)
```

##### 3. Flow Signal
```python
# Flujo óptico denso con Farneback
flow = cv2.calcOpticalFlowFarneback(
    prev_gray, curr_gray,
    None, 0.5, 3, 15, 3, 5, 1.2, 0
)
vertical_flow = np.mean(np.abs(flow[..., 1]))
```

##### 4. Beige Signal
```python
# Cobertura de color beige/crema dentro del ROI textil
beige_ratio = image_processor.beige_ratio(roi_frame)
```

#### Fusion Engine

```python
def compute_score(self) -> float:
    w = self.weights
    s = self.signals
    return (
        w.edge * s.edge +
        w.color * s.color +
        w.flow * s.flow +
        w.beige * s.beige
    )
```

#### FSM de Eventos

```
┌──────┐  score > high_th  ┌───────────┐
│ IDLE ├──────────────────►│ DETECTING │
└──────┘                    └─────┬─────┘
   ▲                              │ n_frames confirmados
   │                              ▼
   │                        ┌────────────┐
   │  cooldown_s elapsed    │ CONFIRMING │
   │◄───────────────────────┤ (emit evt) │
   │                        └────────────┘
   │                              │
   │                              ▼
   │                        ┌──────────┐
   └────────────────────────┤ COOLDOWN │
                            └──────────┘
```

#### Stop Tracker

```python
class StopTracker:
    states = ["PRODUCING", "IDLE_WAIT", "STOP_OPEN"]
    
    def update(self, is_producing: bool):
        if not is_producing:
            self.idle_ticks += 1
            if self.idle_ticks >= idle_debounce:
                elapsed = self.idle_duration_s
                if elapsed < micro_stop_max_s:
                    # Microparada retroactiva (cerrada)
                    action = "CREAR_MICROPARADA"
                else:
                    # Parada larga (abierta)
                    action = "ABRIR_PARADA"
        else:
            if self.state == "STOP_OPEN":
                action = "CERRAR_PARADA"
            self.reset()
```

#### Configuración

```json
{
  "line_id": 1,
  "mode": "textil",
  "frame_backend": "opencv",
  "camera_url": "rtsp://user:pass@192.168.x.x:554/stream",
  "roi": {
    "x": 100, "y": 50, "width": 800, "height": 600
  },
  "fusion": {
    "weight_edge": 0.25,
    "weight_color": 0.30,
    "weight_flow": 0.35,
    "weight_beige": 0.10
  },
  "fsm": {
    "high_threshold": 0.6,
    "low_threshold": 0.4,
    "n_frames_confirm": 3,
    "cooldown_s": 2.0
  },
  "stop_tracking": {
    "micro_stop_max_s": 120.0,
    "idle_debounce_ticks": 3
  }
}
```

---

### 2. resiliencia

**Lenguaje**: Go 1.22  
**Puerto**: 8002  
**Base de datos**: PostgreSQL  

#### Responsabilidades

- Buffer confiable de eventos con garantía de entrega
- Deduplicación por UUID
- Preservación de orden temporal
- Gestión de retry_count
- Limpieza de eventos sincronizados

#### Schema de Base de Datos

```sql
CREATE TABLE IF NOT EXISTS public.events_buffer (
    event_id         UUID PRIMARY KEY,
    device_id        VARCHAR(100) NOT NULL,
    event_type       VARCHAR(50) NOT NULL,
    timestamp        TIMESTAMPTZ NOT NULL,
    payload          JSONB NOT NULL,
    synced           BOOLEAN DEFAULT false,
    retry_count      INTEGER DEFAULT 0,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    last_attempt_at  TIMESTAMPTZ
);

CREATE INDEX idx_events_synced ON events_buffer(synced, timestamp);
CREATE INDEX idx_events_device ON events_buffer(device_id, timestamp DESC);
```

#### API

```
POST /events
Body: {
  "event_id": "uuid-v4",
  "device_id": "jetson-01",
  "event_type": "CORTE",
  "timestamp": "2026-04-28T10:30:00Z",
  "payload": {"roi_id": 1, "confidence": 0.85}
}

GET /events/pending?limit=100
→ eventos con synced=false ordenados por timestamp

PATCH /events/:event_id/ack
→ marca synced=true

GET /stats
→ {"total": 1500, "pending": 23, "synced": 1477}
```

#### Goroutines

```go
// 1. Limpieza cada 1h: elimina eventos synced con >7 días
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            repo.DeleteOldSynced(7 * 24 * time.Hour)
        case <-ctx.Done():
            return
        }
    }
}()
```

---

### 3. enviador

**Lenguaje**: Go 1.22  
**Puerto**: 8003  
**Cloud target**: cloud-gateway:8888  

#### Responsabilidades

- Sincronización batch de eventos a cloud
- Retry exponencial con backoff
- Gestión de comandos pendientes desde cloud
- Sincronización de stops justificados
- Sincronización de snapshots OEE

#### Goroutines (4 concurrentes)

```go
// 1. Mode sync: 30s interval
go func() {
    ticker := time.NewTicker(30 * time.Second)
    for {
        select {
        case <-ticker.C:
            mode := fetchOperationalMode()
            applyModeChanges(mode)
        case <-ctx.Done():
            return
        }
    }
}()

// 2. Command applier: 3s interval
go func() {
    ticker := time.NewTicker(3 * time.Second)
    for {
        select {
        case <-ticker.C:
            cmds := fetchPendingCommands()
            for _, cmd := range cmds {
                executeCommand(cmd)
            }
        case <-ctx.Done():
            return
        }
    }
}()

// 3. Stop sync: 3s interval (no espera OEE)
go func() {
    ticker := time.NewTicker(3 * time.Second)
    for {
        select {
        case <-ticker.C:
            stops := fetchPendingStops()
            sendStopsToCloud(stops)
        case <-ctx.Done():
            return
        }
    }
}()

// 4. OEE sync: 5min interval (configurable)
go func() {
    interval := time.Duration(oeeIntervalS) * time.Second
    ticker := time.NewTicker(interval)
    for {
        select {
        case <-ticker.C:
            events := resilienciaClient.GetPending(1000)
            if len(events) > 0 {
                sendBatchToCloud(events)
            }
        case <-ctx.Done():
            return
        }
    }
}()
```

#### Retry Policy

```go
type RetryPolicy struct {
    InitialDelay  time.Duration  // 2s
    MaxDelay      time.Duration  // 5min
    BackoffFactor float64        // 2.0
    MaxRetries    int            // 5
}

func (p *RetryPolicy) NextDelay(attempt int) time.Duration {
    delay := float64(p.InitialDelay) * math.Pow(p.BackoffFactor, float64(attempt))
    if delay > float64(p.MaxDelay) {
        delay = float64(p.MaxDelay)
    }
    return time.Duration(delay)
}
```

#### Batch Sync

```go
func (s *SenderService) SendBatch(events []Event) error {
    body := BatchRequest{
        DeviceID: s.deviceID,
        Events:   events,
    }
    
    req, _ := http.NewRequest("POST", s.cloudURL+"/api/v1/edge/events", toJSON(body))
    req.Header.Set("X-API-Key", s.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := s.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 200 {
        // ACK eventos en resiliencia
        for _, evt := range events {
            s.resilienciaClient.Ack(evt.EventID)
        }
    }
    
    return nil
}
```

---

### 4. edge-config-service

**Lenguaje**: Go 1.22  
**Puerto**: 8004  
**Base de datos**: PostgreSQL schema `config`  

#### Responsabilidades

- Configuración dinámica con versionado
- Validación de parámetros
- Hot-reload sin reinicio de servicios
- Rollback a versiones anteriores

#### Schema

```sql
CREATE SCHEMA IF NOT EXISTS config;

CREATE TABLE config.line_config (
    line_id           INTEGER PRIMARY KEY,
    mode              VARCHAR(20) NOT NULL DEFAULT 'textil',
    camera_url        TEXT NOT NULL,
    frame_backend     VARCHAR(20) DEFAULT 'opencv',
    roi               JSONB NOT NULL,
    fusion            JSONB NOT NULL,
    fsm               JSONB NOT NULL,
    stop_tracking     JSONB,
    version           INTEGER DEFAULT 1,
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE config.config_history (
    id                SERIAL PRIMARY KEY,
    line_id           INTEGER NOT NULL,
    config_snapshot   JSONB NOT NULL,
    version           INTEGER NOT NULL,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
```

#### API

```
GET  /config?line_id=1        → config actual con version
PUT  /config?line_id=1        → actualiza config, auto-incrementa version
GET  /config/history?line_id=1&version=3  → rollback a versión específica
POST /config/validate         → valida JSON sin guardar
```

#### Validación

```go
func ValidateLineConfig(cfg *LineConfig) error {
    if cfg.ROI.Width <= 0 || cfg.ROI.Height <= 0 {
        return errors.New("ROI dimensions must be > 0")
    }
    
    if cfg.Fusion.WeightEdge < 0 || cfg.Fusion.WeightEdge > 1 {
        return errors.New("fusion weights must be in [0, 1]")
    }
    
    if cfg.FSM.NFramesConfirm < 1 || cfg.FSM.NFramesConfirm > 30 {
        return errors.New("n_frames_confirm must be in [1, 30]")
    }
    
    if cfg.Mode != "textil" {
        return errors.New("mode must be 'textil'")
    }
    
    return nil
}
```

---

### 5. edge-gateway

**Lenguaje**: Go 1.22  
**Puerto**: 8005  
**Role**: Single entry point para tablet/browser  

#### Responsabilidades

- Routing HTTP a microservicios internos
- Autenticación por token (`AUTH_TOKEN`)
- SSE (Server-Sent Events) para UI reactiva
- Stream de cámara MJPEG via ffmpeg
- Gestión de comandos operacionales
- Métricas del sistema Jetson
- Health checks agregados
- Audit log de acciones

#### Arquitectura

```
cmd/main.go                    # Entry point + DI
internal/
  domain/
    ├── command.go             # Comandos operacionales
    ├── metrics.go             # Métricas del sistema
    └── health.go              # Estado de servicios
  ports/
    ├── sse_broker.go          # Broadcast SSE
    ├── command_executor.go    # Ejecución de comandos
    └── metrics_provider.go    # Proveedor de métricas
  adapters/
    ├── sse_broker.go          # Implementación con channels
    ├── postgres_repo.go       # Repo de comandos/audit
    ├── system_metrics.go      # Lectura /proc, thermal_zone
    ├── camera_push.go         # ffmpeg MJPEG streamer
    └── cloud_sse_client.go    # Cliente SSE del cloud
  application/
    ├── gateway_service.go     # Lógica de negocio
    └── command_service.go     # Orquestación de comandos
```

#### SSE Broker

```go
type SSEBrokerImpl struct {
    mu       sync.RWMutex
    clients  map[string]chan []byte
}

func (b *SSEBrokerImpl) Subscribe(clientID string) <-chan []byte {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    ch := make(chan []byte, 64)
    b.clients[clientID] = ch
    return ch
}

func (b *SSEBrokerImpl) Broadcast(data []byte) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    for _, ch := range b.clients {
        select {
        case ch <- data:
        default:
            // Cliente lento, skip para no bloquear
        }
    }
}

func (b *SSEBrokerImpl) Unsubscribe(clientID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if ch, exists := b.clients[clientID]; exists {
        close(ch)
        delete(b.clients, clientID)
    }
}
```

#### Camera Stream (MJPEG)

```go
func (c *CameraPush) Start(ctx context.Context) error {
    cmd := exec.CommandContext(ctx,
        "ffmpeg",
        "-rtsp_transport", "tcp",
        "-i", c.cameraURL,
        "-vf", "fps=10,scale=640:-1",
        "-q:v", "5",
        "-f", "mjpeg",
        "pipe:1",
    )
    
    pr, pw := io.Pipe()
    cmd.Stdout = pw
    
    // Goroutine: lectura de frames desde ffmpeg
    go func() {
        defer pw.Close()
        if err := cmd.Run(); err != nil {
            log.Printf("[camera] ffmpeg error: %v", err)
        }
    }()
    
    // Goroutine: broadcast MJPEG boundaries
    go func() {
        buf := make([]byte, 0, 128*1024)
        scanner := bufio.NewScanner(pr)
        scanner.Buffer(buf, 1024*1024)
        scanner.Split(mjpegBoundarySplit)
        
        for scanner.Scan() {
            frame := scanner.Bytes()
            c.broker.Broadcast(frame)
        }
    }()
    
    return nil
}
```

#### Métricas del Sistema

```go
type SystemMetrics struct {
    CPUPercent     float64        `json:"cpu_percent"`
    MemoryUsedMB   int            `json:"memory_used_mb"`
    MemoryTotalMB  int            `json:"memory_total_mb"`
    DiskUsedGB     float64        `json:"disk_used_gb"`
    DiskTotalGB    float64        `json:"disk_total_gb"`
    Temperatures   []TempReading  `json:"temperatures"`
    Uptime         string         `json:"uptime"`
}

func (sm *SystemMetricsAdapter) GetMetrics() (*SystemMetrics, error) {
    // Paralelizar lecturas con timeout
    type result struct {
        cpu  float64
        mem  MemInfo
        disk DiskInfo
        temp []TempReading
    }
    
    ch := make(chan result, 1)
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    go func() {
        r := result{
            cpu:  readCPU(),
            mem:  readMemInfo(),
            disk: readDiskInfo("/"),
            temp: readThermalZones(),
        }
        select {
        case ch <- r:
        case <-ctx.Done():
        }
    }()
    
    select {
    case r := <-ch:
        return buildMetrics(r), nil
    case <-ctx.Done():
        return nil, errors.New("timeout reading metrics")
    }
}
```

#### Health Check Agregado

```go
func (gw *GatewayService) CheckHealth() map[string]HealthStatus {
    services := []string{
        "vision-detector",
        "resiliencia",
        "enviador",
        "edge-config",
    }
    
    ch := make(chan depResult, len(services))
    
    for _, svc := range services {
        go func(name string) {
            url := gw.serviceURLs[name] + "/health"
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()
            
            req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
            resp, err := gw.httpClient.Do(req)
            
            status := HealthStatus{Service: name}
            if err != nil {
                status.Status = "unhealthy"
                status.Error = err.Error()
            } else {
                resp.Body.Close()
                status.Status = "healthy"
            }
            
            ch <- depResult{name, status}
        }(svc)
    }
    
    results := make(map[string]HealthStatus)
    for i := 0; i < len(services); i++ {
        r := <-ch
        results[r.name] = r.status
    }
    
    return results
}
```

#### Comandos Operacionales

```sql
CREATE TABLE IF NOT EXISTS public.commands (
    id             SERIAL PRIMARY KEY,
    command_type   VARCHAR(50) NOT NULL,  -- START_TURNO, STOP_TURNO, CHANGE_PRODUCTO
    payload        JSONB NOT NULL,
    status         VARCHAR(20) DEFAULT 'pending',  -- pending | executed | failed
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    executed_at    TIMESTAMPTZ
);
```

```go
type Command struct {
    ID          int       `json:"id"`
    CommandType string    `json:"command_type"`
    Payload     any       `json:"payload"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
    ExecutedAt  *time.Time `json:"executed_at,omitempty"`
}

// Comandos disponibles
const (
    CMD_START_TURNO       = "START_TURNO"
    CMD_STOP_TURNO        = "STOP_TURNO"
    CMD_CHANGE_PRODUCTO   = "CHANGE_PRODUCTO"
    CMD_ADJUST_VELOCIDAD  = "ADJUST_VELOCIDAD"
    CMD_RELOAD_CONFIG     = "RELOAD_CONFIG"
)
```

---

### 6. energy-sender

**Lenguaje**: Go 1.22  
**Puerto**: 8086  
**Deploy**: Raspberry Pi con Modbus RTU  

#### Responsabilidades

- Lectura de config operacional desde PostgreSQL
- Envío de datos de consumo energético al cloud
- Hot-reload de config sin reiniciar contenedor

#### Schema

```sql
CREATE SCHEMA IF NOT EXISTS energy;

CREATE TABLE energy.config (
    key   VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL
);

-- Valores de configuración
INSERT INTO energy.config (key, value) VALUES
('device_id', 'rpienergy01'),
('cloud_url', 'https://api.mentoredge.io'),
('energy_api_key', 'CHANGE_ME'),
('poll_interval_s', '300'),
('send_batch_size', '100');

CREATE TABLE energy.readings (
    id             SERIAL PRIMARY KEY,
    meter_id       INTEGER NOT NULL,
    timestamp      TIMESTAMPTZ NOT NULL,
    voltage_v      FLOAT,
    current_a      FLOAT,
    power_kw       FLOAT,
    energy_kwh     FLOAT,
    synced         BOOLEAN DEFAULT false
);
```

#### API

```
GET  /health              → {"status":"ok"}
GET  /config              → config actual desde DB
POST /config/reload       → relee config desde DB
GET  /readings/pending    → readings con synced=false
```

---

### 7. ui-local

**Framework**: Vue 3 Composition API  
**Puerto**: 8080  
**Build**: Vite + Tailwind CSS  

#### Vistas

```
src/
  views/
    ├── Dashboard.vue         # Estado en tiempo real (SSE)
    ├── Config.vue            # Editor de parámetros + ROI
    ├── Health.vue            # Health checks de servicios
    ├── Metrics.vue           # CPU, RAM, temperatura
    └── Lab.vue               # Visualización del detector
  components/
    ├── ROIEditor.vue         # Canvas interactivo
    ├── HealthCard.vue        # Card de estado por servicio
    ├── MetricChart.vue       # Gráficos time-series
    └── EventLog.vue          # Log de eventos recientes
```

#### SSE Integration

```javascript
// composables/useSSE.js
export function useSSE() {
  const events = ref([])
  let eventSource = null
  
  onMounted(() => {
    eventSource = new EventSource('/edge/stream')
    
    eventSource.addEventListener('event', (e) => {
      const data = JSON.parse(e.data)
      events.value.unshift(data)
      if (events.value.length > 100) {
        events.value.pop()
      }
    })
    
    eventSource.onerror = () => {
      console.error('SSE connection lost, reconnecting...')
      setTimeout(() => {
        eventSource.close()
        eventSource = new EventSource('/edge/stream')
      }, 5000)
    }
  })
  
  onUnmounted(() => {
    eventSource?.close()
  })
  
  return { events }
}
```

---

## Arquitectura Cloud

### Mapa de Servicios

| Servicio | Puerto | Framework | Responsabilidad |
|---|---|---|---|
| cloud-gateway | 8888 | net/http | API gateway, SSE hub, rate limiting, audit |
| cloud-identity | 8081 | Gin | Auth JWT, users, roles, empresas |
| cloud-ingest | 8082 | Gin | Ingesta OEE, stops, production runs |
| cloud-config | 8083 | Gin | Plantas, líneas, dispositivos, variables |
| cloud-analytics | 8084 | Gin | Dashboards, Pareto, reportes OEE |
| cloud-integration | 8085 | Gin | APIs externas, third-party |
| energy-ingest | 8087 | Gin | Ingesta de datos de consumo energético |
| cloud-frontend | 80 | Vue 3 + Nginx | SPA multi-tenant |
| postgres-cloud | 5432 | PostgreSQL 16 | Base de datos compartida |

### Flujo de Requests

```
Browser / Jetson
      ↓
cloud-gateway :8888
  ├─→ /api/auth/*           → cloud-identity :8081
  ├─→ /api/v1/edge/*        → cloud-ingest :8082
  ├─→ /api/plantas/*        → cloud-config :8083
  ├─→ /api/dashboard/*      → cloud-analytics :8084
  ├─→ /api/integration/*    → cloud-integration :8085
  ├─→ /api/energy/*         → energy-ingest :8087
  └─→ /* (catch-all)        → cloud-frontend :80
```

### Autenticación

| Path Pattern | Auth Type | Header |
|---|---|---|
| `/api/auth/*` | None | - |
| `/api/v1/edge/*` | API Key | X-API-Key |
| `/api/*` | JWT | Authorization: Bearer |
| `/internal/*` | Internal Key | X-Internal-Key |

### SSE Real-Time Push

```go
// Gateway mantiene conexiones SSE abiertas con cada Jetson
// Envia comandos en tiempo real:

type SSEMessage struct {
    Type    string `json:"type"`
    Payload any    `json:"payload"`
}

// Tipos de mensajes
const (
    MSG_SYNC_CATALOG          = "SYNC_CATALOG"
    MSG_SYNC_PRODUCTOS        = "SYNC_PRODUCTOS"
    MSG_SYNC_TURNOS           = "SYNC_TURNOS"
    MSG_COMMAND               = "COMMAND"
    MSG_HEARTBEAT             = "HEARTBEAT"
)

// Jetson recibe y aplica cambios sin polling
```

### Rate Limiting

```go
type TokenBucket struct {
    capacity    int
    tokens      int
    refillRate  int  // tokens/second
    lastRefill  time.Time
    mu          sync.Mutex
}

func (b *TokenBucket) Allow() bool {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    elapsed := time.Since(b.lastRefill)
    refilled := int(elapsed.Seconds()) * b.refillRate
    b.tokens = min(b.capacity, b.tokens + refilled)
    b.lastRefill = time.Now()
    
    if b.tokens > 0 {
        b.tokens--
        return true
    }
    return false
}
```

---

## Flujos de Datos

### Flujo de Evento Completo

```
1. Camera RTSP
   ↓ 30 fps
2. vision-event-detector (Python)
   - Captura frame
   - Procesa 4 señales
   - FSM detecta evento
   - Genera UUID
   ↓ HTTP POST
3. resiliencia (Go)
   - Deduplica por UUID
   - Persiste en PostgreSQL
   - Responde 200 OK
   ↓ Polling 5s
4. enviador (Go)
   - Lee batch pendientes
   - Agrupa por device_id
   - Retry exponencial
   ↓ HTTPS POST
5. cloud-gateway :8888
   - Valida X-API-Key
   - Rate limit check
   - Audit log
   ↓ Proxy
6. cloud-ingest :8082
   - Valida payload
   - Escribe a DB
   - Broadcast SSE a dashboards
   ↓ PostgreSQL
7. mentor_cloud.ingest.raw_events
   - Indexado por timestamp
   - Particionado por fecha
```

### Flujo de Configuración

```
1. Usuario en frontend (Vue)
   ↓ HTTP PUT
2. cloud-gateway :8888
   - Valida JWT
   - Extrae empresa_id
   ↓
3. cloud-config :8083
   - Actualiza configuración
   - Incrementa version
   - Escribe en DB
   ↓ Trigger
4. cloud-gateway
   - Detecta cambio
   - Genera comando SYNC_*
   ↓ SSE push
5. Jetson enviador
   - Recibe comando via SSE
   - Escribe en commands table
   ↓
6. Jetson enviador (goroutine)
   - Lee pending commands
   - Ejecuta localmente
   - Notifica servicios
   ↓
7. vision-event-detector
   - Polling config cada 10s
   - Hot-reload sin reinicio
```

---

## Patrones de Concurrencia

### Channels en Go

```go
// 1. Unbuffered (sincrónico)
ch := make(chan int)
go func() { ch <- 42 }()
val := <-ch

// 2. Buffered (asincrónico)
ch := make(chan []byte, 64)
go func() {
    for msg := range ch {
        process(msg)
    }
}()

// 3. Select multi-channel
select {
case msg := <-ch1:
    handle(msg)
case <-timeout:
    log.Println("timeout")
case <-ctx.Done():
    return
}
```

### WaitGroup para Shutdown

```go
var wg sync.WaitGroup

// Lanzar N goroutines
for i := 0; i < N; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        worker(id)
    }(i)
}

// Esperar a que terminen todas
wg.Wait()
log.Println("all workers done")
```

### Context para Cancelación

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-time.After(5 * time.Second):
            doWork()
        case <-ctx.Done():
            log.Println("worker cancelled")
            return
        }
    }
}()

// Cancelar desde otra goroutine
cancel()
```

---

## Base de Datos

### Edge PostgreSQL

```sql
-- Database: mentor_edge

-- Schema: public
CREATE TABLE public.events_buffer (
    event_id         UUID PRIMARY KEY,
    device_id        VARCHAR(100),
    event_type       VARCHAR(50),
    timestamp        TIMESTAMPTZ,
    payload          JSONB,
    synced           BOOLEAN DEFAULT false,
    retry_count      INTEGER DEFAULT 0
);

-- Schema: linea_1
CREATE SCHEMA linea_1;

CREATE TABLE linea_1.stops (
    id                SERIAL PRIMARY KEY,
    line_id           INTEGER NOT NULL,
    start_time        TIMESTAMPTZ NOT NULL,
    end_time          TIMESTAMPTZ,
    duration_s        INTEGER,
    is_microparada    BOOLEAN DEFAULT false,
    categoria_id      INTEGER,
    justificacion     TEXT,
    synced            BOOLEAN DEFAULT false
);

CREATE TABLE linea_1.oee_snapshots (
    id                SERIAL PRIMARY KEY,
    line_id           INTEGER NOT NULL,
    timestamp         TIMESTAMPTZ NOT NULL,
    availability      FLOAT,
    performance       FLOAT,
    quality           FLOAT,
    oee               FLOAT,
    produced_units    INTEGER,
    synced            BOOLEAN DEFAULT false
);

-- Schema: config
CREATE SCHEMA config;

CREATE TABLE config.line_config (
    line_id           INTEGER PRIMARY KEY,
    mode              VARCHAR(20),
    camera_url        TEXT,
    roi               JSONB,
    fusion            JSONB,
    fsm               JSONB,
    version           INTEGER DEFAULT 1
);
```

### Cloud PostgreSQL

```sql
-- Database: mentor_cloud

-- Schema: identity
CREATE SCHEMA identity;

CREATE TABLE identity.empresas (
    id               SERIAL PRIMARY KEY,
    nombre           VARCHAR(200) NOT NULL,
    ruc              VARCHAR(20),
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE identity.usuarios (
    id               SERIAL PRIMARY KEY,
    empresa_id       INTEGER REFERENCES identity.empresas(id),
    email            VARCHAR(200) UNIQUE NOT NULL,
    password_hash    TEXT NOT NULL,
    role             VARCHAR(50),
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- Schema: config
CREATE SCHEMA config;

CREATE TABLE config.plantas (
    id               SERIAL PRIMARY KEY,
    empresa_id       INTEGER REFERENCES identity.empresas(id),
    nombre           VARCHAR(200),
    ubicacion        VARCHAR(200)
);

CREATE TABLE config.lineas (
    id               SERIAL PRIMARY KEY,
    planta_id        INTEGER REFERENCES config.plantas(id),
    empresa_id       INTEGER,
    nombre           VARCHAR(200),
    codigo           VARCHAR(50) UNIQUE
);

CREATE TABLE config.dispositivos (
    id               SERIAL PRIMARY KEY,
    device_id        VARCHAR(100) UNIQUE,
    linea_id         INTEGER REFERENCES config.lineas(id),
    tipo             VARCHAR(50),
    api_key          VARCHAR(200)
);

-- Schema: ingest
CREATE SCHEMA ingest;

CREATE TABLE ingest.raw_events (
    id               BIGSERIAL PRIMARY KEY,
    device_id        VARCHAR(100),
    event_type       VARCHAR(50),
    timestamp        TIMESTAMPTZ,
    payload          JSONB,
    received_at      TIMESTAMPTZ DEFAULT NOW()
) PARTITION BY RANGE (timestamp);

CREATE TABLE ingest.oee_snapshots (
    id               BIGSERIAL PRIMARY KEY,
    device_id        VARCHAR(100),
    linea_id         INTEGER,
    timestamp        TIMESTAMPTZ,
    availability     FLOAT,
    performance      FLOAT,
    quality          FLOAT,
    oee              FLOAT,
    produced_units   INTEGER
) PARTITION BY RANGE (timestamp);

-- Schema: analytics
CREATE SCHEMA analytics;

CREATE TABLE analytics.paradas (
    id               SERIAL PRIMARY KEY,
    linea_id         INTEGER,
    start_time       TIMESTAMPTZ,
    end_time         TIMESTAMPTZ,
    duration_s       INTEGER,
    categoria_id     INTEGER,
    justificacion    TEXT
);

-- Schema: gateway
CREATE SCHEMA gateway;

CREATE TABLE gateway.audit_log (
    id               BIGSERIAL PRIMARY KEY,
    user_id          INTEGER,
    device_id        VARCHAR(100),
    action           VARCHAR(100),
    resource         VARCHAR(200),
    timestamp        TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE gateway.commands (
    id               SERIAL PRIMARY KEY,
    device_id        VARCHAR(100),
    command_type     VARCHAR(50),
    payload          JSONB,
    status           VARCHAR(20),
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Comunicación Edge-Cloud

### Protocolo SSE (Server-Sent Events)

```
Edge Device → GET /api/v1/edge/stream
              Headers: X-API-Key: <device_api_key>

Cloud Gateway →
  HTTP/1.1 200 OK
  Content-Type: text/event-stream
  Cache-Control: no-cache
  Connection: keep-alive

  event: connected
  data: {"message":"stream established"}

  event: command
  data: {"type":"SYNC_CATALOG","payload":{...}}

  event: heartbeat
  data: {"timestamp":"2026-04-28T10:30:00Z"}
```

### Comandos Sincronizados

```json
{
  "type": "SYNC_CATALOG",
  "payload": {
    "version": 5,
    "categorias": [
      {
        "id": 1,
        "nombre": "Falta de Materia Prima",
        "tipo": "programada"
      }
    ]
  }
}

{
  "type": "START_TURNO",
  "payload": {
    "turno_id": 1,
    "operador": "Juan Perez",
    "timestamp": "2026-04-28T06:00:00Z"
  }
}
```

---

## Despliegue

### Edge (Jetson)

```bash
# 1. Clonar repo
git clone https://github.com/alonsosss/Mentoredge.git
cd Mentoredge/mentor-edge

# 2. Configurar .env
cd infrastructure/docker
cp .env.example .env
nano .env  # ajustar DEVICE_ID, CAMERA_URL, CLOUD_URL, etc.

# 3. Levantar stack
docker compose -f docker-compose.jetson-orin.yml up -d

# 4. Verificar
docker ps
docker logs vision-event-detector -f
curl http://localhost:8005/health
```

### Cloud (VPS)

```bash
# 1. Clonar repo
git clone https://github.com/alonsosss/Mentoredge.git
cd Mentoredge/mentor-cloud

# 2. Configurar .env
cp .env.example .env
nano .env  # ajustar DB_PASSWORD, JWT_SECRET, etc.

# 3. Migrar base de datos
cd infrastructure/database
psql -U postgres -f 01_schemas.sql
psql -U postgres -f 02_identity.sql
psql -U postgres -f 03_config.sql
# ... resto de migraciones

# 4. Levantar servicios
cd ../docker
docker compose up -d

# 5. Verificar
docker ps
curl http://localhost:8888/health
```

### Raspberry Pi Energy

```bash
# 1. Clonar repo
git clone https://github.com/alonsosss/Mentoredge.git
cd Mentoredge/mentor-edge

# 2. Configurar .env
cd infrastructure/docker
cp .env.raspberry-energy .env
nano .env  # ajustar DEVICE_ID, METER_ID_1, etc.

# 3. Configurar udev para puerto serie estable
sudo cp 99-mc60-serial.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules

# 4. Levantar stack
docker compose -f docker-compose.raspberry-energy.yml up -d

# 5. Verificar Node-RED
curl http://localhost:1881
# Acceder a http://<raspberry-ip>:1881 para configurar flow
```

---

## Monitoreo y Logs

### Logs por Servicio

```bash
# Edge
docker logs vision-event-detector --tail 100 -f
docker logs resiliencia --tail 100 -f
docker logs enviador --tail 100 -f
docker logs edge-gateway --tail 100 -f

# Cloud
docker logs cloud-gateway --tail 100 -f
docker logs cloud-ingest --tail 100 -f
```

### Métricas

```bash
# Edge
curl http://localhost:8005/metrics

# Cloud
curl http://localhost:8888/metrics
```

### Health Checks

```bash
# Edge
curl http://localhost:8005/health

# Cloud
curl http://localhost:8888/health
```

---

## Conclusión

**Mentor Edge** es una plataforma industrial robusta que combina:
- Procesamiento local en tiempo real con vision artificial
- Buffer confiable offline-first
- Sincronización cloud con retry garantizado
- Interfaces modernas para operarios y gerentes
- Arquitectura escalable y mantenible

La concurrencia en Go permite alta eficiencia con goroutines livianas, mientras que la arquitectura hexagonal en Python facilita testing y evolución del código de visión.
