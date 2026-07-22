# Base de Datos Multi-Tenant y Tablet en Tiempo Real

## Tabla de Contenidos

1. [Arquitectura Multi-Tenant Cloud](#arquitectura-multi-tenant-cloud)
2. [Aislamiento de Datos](#aislamiento-de-datos)
3. [Provisioning de Plantas](#provisioning-de-plantas)
4. [Pool Manager y Resolución de Scope](#pool-manager-y-resolución-de-scope)
5. [Esquemas por Línea](#esquemas-por-línea)
6. [Tablet: Comunicación en Tiempo Real](#tablet-comunicación-en-tiempo-real)
7. [Dual Mode: Edge vs Cloud](#dual-mode-edge-vs-cloud)
8. [Flujos de Datos Tablet](#flujos-de-datos-tablet)

---

## Arquitectura Multi-Tenant Cloud

### Concepto

**Mentor Cloud** implementa **multi-tenancy a nivel de base de datos**: cada planta industrial tiene su propia base de datos PostgreSQL física, con esquemas internos por línea de producción.

```
mentor_cloud (DB maestra — config global)
  ├─ identity.*         (empresas, usuarios, roles)
  ├─ config.*           (plantas, lineas, dispositivos)
  ├─ admin.*            (planta_databases — catálogo de DBs)
  └─ gateway.*          (device_registry, audit_log)

mentor_planta_14 (DB por planta)
  ├─ linea_14.*         (datos operacionales línea 14)
  ├─ linea_15.*         (datos operacionales línea 15)
  └─ linea_16.*         (datos operacionales línea 16)

mentor_planta_20 (DB por planta)
  ├─ linea_27.*         (datos operacionales línea 27)
  └─ linea_28.*         (datos operacionales línea 28)
```

### Ventajas

1. **Aislamiento físico**: datos de cada planta en BD separada
2. **Escalabilidad horizontal**: cada planta puede migrar a su propia instancia RDS
3. **Backups independientes**: recuperación granular por planta
4. **Performance**: índices y queries aislados, sin interferencia
5. **Compliance**: cumplimiento de regulaciones de datos por región
6. **Multi-región**: plantas en diferentes países con DBs locales

---

## Aislamiento de Datos

### Niveles de Aislamiento

```
Empresa
  └── Planta (DB física separada)
      └── Línea (schema dentro de DB planta)
          └── Datos operacionales (tablas en schema línea)
```

### Tabla admin.planta_databases

**Ubicación**: `mentor_cloud.admin.planta_databases`

Esta tabla es la **fuente de verdad** para localizar la base de datos de cada planta.

```sql
CREATE TABLE admin.planta_databases (
    id               SERIAL      PRIMARY KEY,
    planta_id        INT         NOT NULL UNIQUE REFERENCES config.plantas(id),

    -- Conexión
    db_name          TEXT        NOT NULL UNIQUE,        -- ej: mentor_planta_14
    pg_user          TEXT        NOT NULL,               -- ej: mentor_planta14
    pg_password_enc  TEXT        NOT NULL,               -- AES-256-GCM encriptado
    host             TEXT        NOT NULL DEFAULT 'localhost',
    port             INT         NOT NULL DEFAULT 5432,

    -- Tipo de instancia
    instance_type    TEXT        NOT NULL DEFAULT 'shared'
                                 CHECK (instance_type IN ('shared', 'dedicated')),

    -- Campos RDS (solo para instance_type='dedicated')
    rds_instance_id  TEXT,
    rds_region       TEXT,
    rds_arn          TEXT,
    rds_class        TEXT,       -- ej: db.t3.micro

    -- Estado
    provisioned      BOOLEAN     NOT NULL DEFAULT false,
    provisioned_at   TIMESTAMPTZ,
    lineas_creadas   INT[]       NOT NULL DEFAULT '{}',  -- [14, 15, 16]

    creado_en        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Tipos de Instancia

#### 1. Shared (VPS)

Múltiples BDs de plantas en el mismo proceso PostgreSQL, mismo servidor.

```
VPS (192.168.100.10:5432)
  ├─ mentor_cloud
  ├─ mentor_planta_14
  ├─ mentor_planta_20
  └─ mentor_planta_35
```

**Ventajas**: costo reducido, fácil administración  
**Limitaciones**: recursos compartidos, single point of failure

#### 2. Dedicated (RDS)

Cada planta con su propia instancia PostgreSQL en AWS RDS.

```
RDS us-east-1
  ├─ mentor-planta14.xyz.rds.amazonaws.com:5432
  │  └─ mentor_planta_14
  └─ mentor-planta20.abc.rds.amazonaws.com:5432
     └─ mentor_planta_20

VPS (solo mentor_cloud)
  └─ mentor_cloud
```

**Ventajas**: aislamiento total, escalabilidad, backups automáticos RDS  
**Casos de uso**: plantas grandes (>10 líneas), clientes enterprise, compliance

---

## Provisioning de Plantas

### Flujo de Creación

```
1. Admin crea planta en frontend
   ↓
2. POST /api/admin/provision-planta
   ↓
3. cloud-config valida permisos
   ↓
4. Genera credenciales aleatorias
   ↓
5. Encripta password con AES-256-GCM
   ↓
6. Ejecuta CREATE DATABASE mentor_planta_{id}
   ↓
7. Inserta registro en admin.planta_databases
   ↓
8. Marca provisioned=true
```

### Código de Provisioning

```go
// provisioner.go
type DBProvisioner interface {
    CreateDatabase(ctx context.Context, dbName string) error
    CreateSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string) error
}

type PostgresProvisioner struct {
    masterPool *pgxpool.Pool
    templates  map[string]string  // schema_name → SQL template
}

func (p *PostgresProvisioner) CreateDatabase(ctx context.Context, dbName string) error {
    // CREATE DATABASE no puede ejecutarse en transaction
    _, err := p.masterPool.Exec(ctx, "CREATE DATABASE "+dbName)
    if err != nil {
        return fmt.Errorf("create database %s: %w", dbName, err)
    }
    return nil
}

func (p *PostgresProvisioner) CreateSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string) error {
    template := p.templates["linea"]
    sql := strings.ReplaceAll(template, "{schema}", schemaName)
    
    _, err := pool.Exec(ctx, sql)
    if err != nil {
        return fmt.Errorf("create schema %s: %w", schemaName, err)
    }
    return nil
}
```

### Encriptación de Credenciales

```go
// credentials.go
type AESCredentialProvider struct {
    key []byte  // 32 bytes (AES-256)
}

func NewAESCredentialProvider(keyHex string) (*AESCredentialProvider, error) {
    key, err := hex.DecodeString(keyHex)
    if err != nil || len(key) != 32 {
        return nil, errors.New("invalid encryption key: must be 32 bytes hex")
    }
    return &AESCredentialProvider{key: key}, nil
}

func (p *AESCredentialProvider) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(p.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (p *AESCredentialProvider) Decrypt(encoded string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(p.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}
```

**Variable de entorno requerida**:
```bash
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

Sin esta key, `PlantaPoolManager` será `nil` y todas las queries irán a `mentor_cloud` (modo legacy).

---

## Pool Manager y Resolución de Scope

### PlantaPoolManager

Gestiona un pool de conexiones `pgxpool.Pool` por cada base de datos de planta.

```go
type PlantaPoolManager struct {
    masterDB    *pgxpool.Pool                  // mentor_cloud
    credentials *AESCredentialProvider
    provisioner DBProvisioner
    dsnCfg      DSNConfig
    cache       cache.Store
    pools       sync.Map                       // planta_id → *pgxpool.Pool
}

func (m *PlantaPoolManager) Get(ctx context.Context, plantaID int) (*pgxpool.Pool, error) {
    // 1. Buscar en sync.Map (memoria)
    if v, ok := m.pools.Load(plantaID); ok {
        return v.(*pgxpool.Pool), nil
    }
    
    // 2. Consultar admin.planta_databases
    entry, err := m.resolveEntry(ctx, plantaID)
    if err != nil {
        return nil, err
    }
    
    // 3. Desencriptar password
    password, err := m.credentials.Decrypt(entry.PGPasswordEnc)
    if err != nil {
        return nil, err
    }
    
    // 4. Construir DSN
    dsn := fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
        entry.Host, entry.Port, entry.DBName, entry.PGUser, password, m.dsnCfg.SSLMode,
    )
    
    // 5. Crear pool con pgxpool
    pool, err := pgxpool.New(ctx, dsn)
    if err != nil {
        return nil, err
    }
    
    // 6. Almacenar en sync.Map (cache permanente)
    actual, loaded := m.pools.LoadOrStore(plantaID, pool)
    if loaded {
        pool.Close()
        return actual.(*pgxpool.Pool), nil
    }
    
    return pool, nil
}
```

### Resolución de Scope

El **scope** define el contexto de multi-tenancy: empresa, planta y línea.

```go
type Scope struct {
    EmpresaID int
    PlantaID  int
    LineaID   int
}

// WithScope inyecta scope en el contexto
func WithScope(ctx context.Context, scope Scope) context.Context {
    return context.WithValue(ctx, scopeKey, scope)
}

// ScopeFrom extrae scope del contexto
func ScopeFrom(ctx context.Context) (Scope, bool) {
    scope, ok := ctx.Value(scopeKey).(Scope)
    return scope, ok
}
```

### PoolResolver

Abstracción que resuelve qué pool usar según el scope del contexto.

```go
type PoolResolver struct {
    master *pgxpool.Pool           // mentor_cloud
    mgr    *PlantaPoolManager
}

func (r *PoolResolver) Master() *pgxpool.Pool {
    return r.master
}

func (r *PoolResolver) Resolve(ctx context.Context) (*pgxpool.Pool, string, error) {
    scope, ok := ScopeFrom(ctx)
    if !ok || r.mgr == nil {
        // Sin scope o sin PlantaPoolManager → mentor_cloud
        return r.master, "", nil
    }
    
    // Obtener pool de la planta
    pool, err := r.mgr.Get(ctx, scope.PlantaID)
    if err != nil {
        return nil, "", err
    }
    
    // Calcular schema de la línea
    schema := fmt.Sprintf("linea_%d", scope.LineaID)
    
    return pool, schema, nil
}
```

### Flujo de Query con Scope

```go
// Handler en cloud-analytics
func (h *StopHandler) List(c *gin.Context) {
    empresaID := c.GetInt("empresa_id")      // desde JWT
    plantaID := c.Query("planta_id")
    lineaID := c.Query("linea_id")
    
    // Inyectar scope en contexto
    ctx := multitenancy.WithScope(c.Request.Context(), multitenancy.Scope{
        EmpresaID: empresaID,
        PlantaID:  parseInt(plantaID),
        LineaID:   parseInt(lineaID),
    })
    
    // Repositorio usa PoolResolver
    stops, err := h.repo.List(ctx)
    // ...
}

// Repositorio
type StopRepo struct {
    pr *multitenancy.PoolResolver
}

func (r *StopRepo) List(ctx context.Context) ([]Stop, error) {
    // Resolver pool y schema según scope
    pool, schema, err := r.pr.Resolve(ctx)
    if err != nil {
        return nil, err
    }
    
    var query string
    if schema != "" {
        // Query en schema de línea (mentor_planta_X.linea_Y.stops)
        query = fmt.Sprintf("SELECT * FROM %s.stops ORDER BY start_time DESC LIMIT 100", schema)
    } else {
        // Fallback a tabla legacy (mentor_cloud.analytics.paradas)
        query = "SELECT * FROM analytics.paradas WHERE empresa_id = $1 ORDER BY start_time DESC LIMIT 100"
    }
    
    rows, err := pool.Query(ctx, query, ...)
    // ...
}
```

---

## Esquemas por Línea

Cada línea de producción tiene su propio **schema** dentro de la base de datos de su planta.

### Estructura

```sql
-- Ejemplo: Planta 14 con 3 líneas

mentor_planta_14
  ├─ linea_14.stops
  ├─ linea_14.oee_snapshots
  ├─ linea_14.production_runs
  ├─ linea_14.parada_programada
  ├─ linea_14.parada_no_programada
  ├─ linea_14.productos
  ├─ linea_14.turnos
  ├─ linea_14.variables
  ├─ linea_14.pending_commands
  ├─ linea_14.raw_events
  ├─ linea_14.device_sync_log
  │
  ├─ linea_15.stops
  ├─ linea_15.oee_snapshots
  └─ ...
```

### Template SQL

**Archivo**: `mentor-edge/infrastructure/database/linea_template.sql`

Se usa un template con placeholder `{schema}` que se reemplaza programáticamente:

```sql
CREATE SCHEMA IF NOT EXISTS {schema};

-- ═══════════════════════════════════════════════════════════════
-- PARADAS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.stops (
    id                  SERIAL PRIMARY KEY,
    device_id           VARCHAR(64),
    start_time          TIMESTAMPTZ NOT NULL,
    end_time            TIMESTAMPTZ,
    duration_s          INTEGER,
    is_microparada      BOOLEAN DEFAULT false,
    categoria_id        INTEGER,
    tipo_categoria      VARCHAR(20),  -- 'programada' | 'no_programada'
    justificacion       TEXT,
    justificado_por     VARCHAR(128),
    synced              BOOLEAN DEFAULT false,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_{schema}_stops_start
    ON {schema}.stops (start_time DESC);
CREATE INDEX IF NOT EXISTS idx_{schema}_stops_synced
    ON {schema}.stops (synced) WHERE synced = false;

-- ═══════════════════════════════════════════════════════════════
-- OEE SNAPSHOTS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.oee_snapshots (
    id                  SERIAL PRIMARY KEY,
    device_id           VARCHAR(64),
    hora                TIMESTAMPTZ NOT NULL,
    availability        FLOAT,
    performance         FLOAT,
    quality             FLOAT,
    oee                 FLOAT,
    produced_units      INTEGER,
    target_units        INTEGER,
    rejected_units      INTEGER,
    downtime_s          INTEGER,
    synced              BOOLEAN DEFAULT false,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_{schema}_oee_device_hora
    ON {schema}.oee_snapshots (device_id, hora DESC);

-- ═══════════════════════════════════════════════════════════════
-- PRODUCTION RUNS
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.production_runs (
    id                  SERIAL PRIMARY KEY,
    run_id              UUID UNIQUE,
    device_id           VARCHAR(64),
    producto_id         INTEGER,
    turno_id            INTEGER,
    operador_id         VARCHAR(128),
    start_time          TIMESTAMPTZ NOT NULL,
    end_time            TIMESTAMPTZ,
    units_produced      INTEGER DEFAULT 0,
    units_rejected      INTEGER DEFAULT 0,
    active              BOOLEAN DEFAULT true,
    synced              BOOLEAN DEFAULT false,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_{schema}_runs_device_active
    ON {schema}.production_runs (device_id, active, start_time DESC);

-- ═══════════════════════════════════════════════════════════════
-- COMANDOS PENDIENTES (para sync cloud → edge)
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.pending_commands (
    id                  SERIAL PRIMARY KEY,
    command_id          UUID UNIQUE,
    device_id           VARCHAR(64) NOT NULL,
    command_type        VARCHAR(50) NOT NULL,  -- 'justificar_parada' | 'upsert_production_run'
    payload             JSONB NOT NULL,
    status              VARCHAR(20) DEFAULT 'pending',  -- pending | acked | expired
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    acked_at            TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_{schema}_pending_cmd_status
    ON {schema}.pending_commands (status, created_at DESC)
    WHERE status = 'pending';

-- ═══════════════════════════════════════════════════════════════
-- CATÁLOGOS SINCRONIZADOS DEL CLOUD
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS {schema}.parada_programada (
    id                  INTEGER PRIMARY KEY,
    nombre              VARCHAR(200),
    codigo              VARCHAR(100),
    padre_id            INTEGER,
    linea_id            INTEGER,
    empresa_id          INTEGER,
    orden               INTEGER,
    creado_en           TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS {schema}.parada_no_programada (
    id                  INTEGER PRIMARY KEY,
    nombre              VARCHAR(200),
    codigo              VARCHAR(100),
    padre_id            INTEGER,
    linea_id            INTEGER,
    empresa_id          INTEGER,
    orden               INTEGER,
    creado_en           TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS {schema}.productos (
    id                  INTEGER PRIMARY KEY,
    nombre              VARCHAR(200),
    codigo              VARCHAR(100),
    descripcion         TEXT,
    empresa_id          INTEGER,
    linea_id            INTEGER,
    activo              BOOLEAN,
    creado_en           TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS {schema}.turnos (
    id                  INTEGER PRIMARY KEY,
    nombre              VARCHAR(100),
    hora_inicio         TIME,
    hora_fin            TIME,
    planta_id           INTEGER,
    activo              BOOLEAN,
    creado_en           TIMESTAMPTZ
);

-- ... más tablas de catálogo
```

### Creación de Schema

```bash
#!/bin/bash
# init_line.sh

LINEA_ID=$1
SCHEMA="linea_${LINEA_ID}"

sed "s/{schema}/${SCHEMA}/g" linea_template.sql | psql -h localhost -U mentor_planta14 -d mentor_planta_14
```

---

## Tablet: Comunicación en Tiempo Real

La **tablet** (aplicación Vue 3) se comunica con el edge/cloud en tiempo real usando **SSE (Server-Sent Events)** y polling estratégico.

### Arquitectura de Comunicación

```
Tablet Vue 3
    ↓ SSE connection
Edge Gateway :8005  (modo EDGE)
    ↓ broadcast events
vision-event-detector, resiliencia, enviador
    
    O

Tablet Vue 3
    ↓ SSE connection + JWT
Cloud Gateway :8888  (modo CLOUD)
    ↓ broadcast events
cloud-ingest, cloud-analytics
```

### SSE Client (TypeScript)

**Archivo**: `mentor-tablet-app/src/services/sse.ts`

```typescript
import type { SSEMessage } from '@/types'

type SSEHandler = (msg: SSEMessage) => void

const RECONNECT_BASE_MS = 1000
const RECONNECT_MAX_MS = 30000

export class SSEClient {
  private url = ''
  private source: EventSource | null = null
  private handlers = new Map<string, Set<SSEHandler>>()
  private globalHandlers = new Set<SSEHandler>()
  private errorHandlers = new Set<() => void>()
  private reconnectMs = RECONNECT_BASE_MS
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private active = false

  // Conectar a edge-gateway
  connect(baseURL: string): void {
    this.url = `${baseURL}/edge/stream`
    this.active = true
    this.reconnectMs = RECONNECT_BASE_MS
    this.open()
  }

  // Conectar a cloud-gateway con JWT
  connectCloud(cloudURL: string, jwt: string, empresaId?: number): void {
    if (!empresaId) return  // superadmin sin empresa no puede conectar
    
    const params = new URLSearchParams({
      token: jwt,
      empresa_id: String(empresaId)
    })
    this.url = `${cloudURL}/api/v1/cloud/stream?${params.toString()}`
    this.active = true
    this.reconnectMs = RECONNECT_BASE_MS
    this.open()
  }

  disconnect(): void {
    this.active = false
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.source) {
      this.source.close()
      this.source = null
    }
  }

  // Suscribir handler a tipo específico de mensaje
  on(eventType: string, handler: SSEHandler): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set())
    }
    this.handlers.get(eventType)!.add(handler)
    
    // Retornar función de cleanup
    return () => this.handlers.get(eventType)?.delete(handler)
  }

  // Suscribir handler global (recibe todos los mensajes)
  onAny(handler: SSEHandler): () => void {
    this.globalHandlers.add(handler)
    return () => this.globalHandlers.delete(handler)
  }

  onError(handler: () => void): () => void {
    this.errorHandlers.add(handler)
    return () => this.errorHandlers.delete(handler)
  }

  get connected(): boolean {
    return this.source?.readyState === EventSource.OPEN
  }

  private open(): void {
    if (!this.active) return
    if (this.source) {
      this.source.close()
    }

    this.source = new EventSource(this.url)

    this.source.onopen = () => {
      this.reconnectMs = RECONNECT_BASE_MS
    }

    this.source.onmessage = (ev: MessageEvent) => {
      try {
        const msg: SSEMessage = JSON.parse(ev.data)
        this.dispatch(msg)
      } catch {
        // mensaje malformado, ignorar
      }
    }

    this.source.addEventListener('connected', (ev: Event) => {
      const me = ev as MessageEvent
      try {
        const payload = JSON.parse(me.data)
        this.dispatch({ type: 'connected', payload, ts: Date.now() })
      } catch {}
    })

    this.source.onerror = () => {
      this.source?.close()
      this.source = null
      this.errorHandlers.forEach((h) => h())
      this.scheduleReconnect()
    }
  }

  private dispatch(msg: SSEMessage): void {
    // Handlers específicos del tipo
    const typed = this.handlers.get(msg.type)
    if (typed) {
      typed.forEach((h) => h(msg))
    }
    
    // Handlers globales
    this.globalHandlers.forEach((h) => h(msg))
  }

  private scheduleReconnect(): void {
    if (!this.active) return
    
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.open()
    }, this.reconnectMs)
    
    // Backoff exponencial con límite
    this.reconnectMs = Math.min(this.reconnectMs * 2, RECONNECT_MAX_MS)
  }
}

export const sse = new SSEClient()
```

### Tipos de Mensajes SSE

```typescript
// types/index.ts
export interface SSEMessage {
  type: string
  payload: any
  ts: number
}

// Tipos de mensajes del edge
const SSE_TYPES = {
  // Conexión
  CONNECTED: 'connected',
  HEARTBEAT: 'heartbeat',
  
  // Eventos de producción
  EVENT: 'event',              // evento de corte/producción
  STOP_CREATED: 'stop_created',
  STOP_CLOSED: 'stop_closed',
  
  // Comandos
  COMMAND_APPLIED: 'command_applied',
  COMMAND_FAILED: 'command_failed',
  
  // Config
  CONFIG_UPDATED: 'config_updated',
  
  // Estado
  STATUS_UPDATE: 'status_update',
  HEALTH_UPDATE: 'health_update'
}
```

### Uso en Componentes Vue

```typescript
// DashboardView.vue
import { onMounted, onUnmounted } from 'vue'
import { sse } from '@/services/sse'
import { useStopsStore } from '@/stores/stops'

const stopsStore = useStopsStore()

onMounted(() => {
  // Conectar SSE
  sse.connect('http://192.168.100.33:8005')
  
  // Suscribirse a eventos de paradas
  const unsubStopCreated = sse.on('stop_created', (msg) => {
    stopsStore.addStop(msg.payload)
  })
  
  const unsubStopClosed = sse.on('stop_closed', (msg) => {
    stopsStore.updateStop(msg.payload.id, msg.payload)
  })
  
  // Cleanup al desmontar
  onUnmounted(() => {
    unsubStopCreated()
    unsubStopClosed()
    sse.disconnect()
  })
})
```

### Polling Complementario

Aunque SSE es la columna vertebral, algunos datos requieren polling:

```typescript
// Estado del detector de visión (Python directo)
async function pollLiveState() {
  try {
    const data = await fetch('/vision/status', {
      signal: AbortSignal.timeout(2000)
    }).then(r => r.json())
    
    liveState.value = data.stop_tracker_state  // 'producing' | 'idle_wait' | 'stop_open'
    liveIdleSecs.value = data.idle_duration_s
  } catch {
    liveState.value = 'offline'
  }
}

// Polling cada 3s
const timer = setInterval(pollLiveState, 3000)

onUnmounted(() => clearInterval(timer))
```

**Por qué polling aquí**:
- El detector Python no emite SSE (solo HTTP endpoints)
- Estado muy volátil (cambia cada frame)
- Latencia aceptable para UI (3s)

---

## Dual Mode: Edge vs Cloud

La tablet soporta **dos modos de operación**:

### Modo EDGE (Local)

Conexión directa al Jetson en la misma red local.

```
Tablet ──HTTP/SSE──> Edge Gateway :8005 ──> Servicios Edge
```

**Características**:
- Sin autenticación (token simple)
- Baja latencia (<50ms)
- Funciona sin internet
- Acceso completo a todas las funciones

**Configuración**:
```typescript
setBaseURL('http://192.168.100.33:8005')
setApiMode('EDGE')
sse.connect('http://192.168.100.33:8005')
```

### Modo CLOUD (Remoto)

Conexión a través del cloud gateway con autenticación JWT.

```
Tablet ──HTTPS/JWT──> Cloud Gateway :8888 ──> Microservicios Cloud
```

**Características**:
- Autenticación JWT por usuario
- Multi-tenant: scope por empresa/planta/línea
- Acceso desde cualquier lugar
- Latencia mayor (~200-500ms)
- Funciones limitadas (no acceso directo a cámara)

**Configuración**:
```typescript
const jwt = await login(email, password)
setCloudJWT(jwt)
setBaseURL('https://api.mentoredge.io')
setApiMode('CLOUD')
sse.connectCloud('https://api.mentoredge.io', jwt, empresaId)
```

### API Client con Modo Dual

```typescript
// services/api.ts
let apiMode: 'EDGE' | 'CLOUD' = 'EDGE'
let cloudJWT = ''
let cloudLineaId = 0

async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const url = `${baseURL}${path}`
  
  const authHeaders: Record<string, string> = {}
  if (apiMode === 'CLOUD' && cloudJWT) {
    authHeaders['Authorization'] = `Bearer ${cloudJWT}`
  }
  
  const res = await fetch(url, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders
    },
    body: body ? JSON.stringify(body) : undefined
  })
  
  if (!res.ok) {
    if (res.status === 401 && apiMode === 'CLOUD') {
      // JWT expirado → logout y redirigir a login
      clearSession()
      router.push('/login')
    }
    throw new Error(`HTTP ${res.status}`)
  }
  
  return res.json()
}

// Ejemplo de endpoint que adapta según el modo
export const api = {
  async getStops(lineaId: number): Promise<Stop[]> {
    if (apiMode === 'EDGE') {
      return request('GET', `/edge/stops?linea_id=${lineaId}`)
    } else {
      // Cloud requiere planta_id también
      const plantaId = getSelectedPlantaId()
      return request('GET', `/api/stops?linea_id=${lineaId}&planta_id=${plantaId}`)
    }
  }
}
```

---

## Flujos de Datos Tablet

### 1. Inicio de Sesión (Cloud)

```
1. Usuario ingresa email/password
   ↓
2. POST /api/auth/login
   ↓
3. cloud-identity valida credenciales
   ↓
4. Genera JWT con claims: user_id, empresa_id, role
   ↓
5. Retorna JWT + refresh_token
   ↓
6. Tablet almacena en localStorage
   ↓
7. Conecta SSE con JWT
   ↓
8. Carga catálogos: plantas, líneas, productos
```

### 2. Visualización de Dashboard

```
Tablet carga DashboardView.vue
   ↓
1. Conecta SSE
   ↓
2. Suscribe handlers:
   - stop_created
   - stop_closed
   - event
   - oee_snapshot
   ↓
3. Fetch inicial:
   - GET /api/stops?linea_id=14 (últimas 100)
   - GET /api/oee/snapshots?linea_id=14&hours=24
   - GET /api/production-runs/active?linea_id=14
   ↓
4. Renderiza timeline Canvas
   ↓
5. Recibe eventos SSE en tiempo real:
   
   event: stop_created
   data: {"id":45,"start_time":"2026-04-28T10:30:00Z",...}
   
   ↓
6. Store Pinia actualiza estado reactivo
   ↓
7. Timeline se re-renderiza automáticamente
```

### 3. Justificación de Parada

```
1. Usuario hace clic en parada sin justificar
   ↓
2. Abre modal StopAssignmentModal.vue
   ↓
3. Carga árbol de categorías:
   GET /api/categorias?tipo=no_programada&linea_id=14
   ↓
4. Usuario selecciona categoría y escribe justificación
   ↓
5. POST /api/stops/45/justify
   Body: {
     categoria_id: 12,
     tipo_categoria: 'no_programada',
     justificacion: 'Cambio de bobina',
     justificado_por: 'operator@empresa.com'
   }
   ↓
6. cloud-analytics actualiza:
   - UPDATE linea_14.stops SET categoria_id=12, justificacion='...'
   ↓
7. Encola comando para edge:
   INSERT INTO linea_14.pending_commands (
     command_type='justificar_parada',
     payload='{"stop_id":45,"categoria_id":12,...}'
   )
   ↓
8. Emite SSE broadcast:
   event: stop_updated
   data: {"id":45,"categoria_id":12,...}
   ↓
9. Todas las tablets conectadas reciben update
   ↓
10. Store actualiza stop localmente
   ↓
11. UI refleja cambio instantáneamente
```

### 4. Sincronización Cloud → Edge

```
1. Enviador en Jetson polling cada 3s:
   GET /api/v1/edge/pending-commands?device_id=jetson-01
   ↓
2. cloud-gateway consulta:
   SELECT * FROM linea_14.pending_commands
   WHERE device_id='jetson-01' AND status='pending'
   ↓
3. Retorna lista de comandos
   ↓
4. Enviador aplica localmente:
   UPDATE linea_1.stops SET categoria_id=12, justificacion='...'
   WHERE id=45
   ↓
5. Enviador ACKnowledge:
   POST /api/v1/edge/commands/ack
   Body: {"command_ids": ["uuid1","uuid2"]}
   ↓
6. Cloud marca comandos:
   UPDATE linea_14.pending_commands SET status='acked', acked_at=NOW()
```

### 5. Cambio de Producto

```
1. Usuario selecciona producto en dropdown
   ↓
2. POST /edge/production-runs/change-product (EDGE)
   O
   POST /api/production-runs/change-product (CLOUD)
   
   Body: {
     producto_id: 7,
     operador_id: 'juan.perez',
     timestamp: '2026-04-28T11:00:00Z'
   }
   ↓
3. Edge Gateway (o Cloud):
   - Cierra run activo anterior:
     UPDATE linea_1.production_runs
     SET end_time=NOW(), active=false
     WHERE active=true
   
   - Crea nuevo run:
     INSERT INTO linea_1.production_runs (
       run_id, producto_id, turno_id, operador_id,
       start_time, active
     ) VALUES (uuid_generate_v4(), 7, 1, 'juan.perez', NOW(), true)
   ↓
4. Emite SSE:
   event: production_run_changed
   data: {"new_run_id":"...","producto_id":7}
   ↓
5. UI actualiza indicador de producto actual
```

---

## Stores Reactivos (Pinia)

La tablet usa **Pinia** (Vue store) para gestionar estado global reactivo.

### Ejemplo: Stops Store

```typescript
// stores/stops.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Stop } from '@/types'

export const useStopsStore = defineStore('stops', () => {
  const stops = ref<Stop[]>([])
  const loading = ref(false)
  
  // Computed: paradas sin justificar
  const unjustified = computed(() => 
    stops.value.filter(s => !s.categoria_id && !s.is_microparada)
  )
  
  // Computed: paradas de hoy
  const today = computed(() => {
    const midnight = new Date()
    midnight.setHours(0, 0, 0, 0)
    return stops.value.filter(s => new Date(s.start_time) >= midnight)
  })
  
  // Actions
  async function fetchStops(lineaId: number) {
    loading.value = true
    try {
      stops.value = await api.getStops(lineaId)
    } finally {
      loading.value = false
    }
  }
  
  function addStop(stop: Stop) {
    stops.value.unshift(stop)
  }
  
  function updateStop(id: number, updates: Partial<Stop>) {
    const idx = stops.value.findIndex(s => s.id === id)
    if (idx >= 0) {
      stops.value[idx] = { ...stops.value[idx], ...updates }
    }
  }
  
  return {
    stops,
    loading,
    unjustified,
    today,
    fetchStops,
    addStop,
    updateStop
  }
})
```

### Integración con SSE

```typescript
// En DashboardView.vue o App.vue
import { useStopsStore } from '@/stores/stops'
import { sse } from '@/services/sse'

const stopsStore = useStopsStore()

onMounted(() => {
  // Fetch inicial
  stopsStore.fetchStops(selectedLineaId.value)
  
  // Suscribir a updates en tiempo real
  sse.on('stop_created', (msg) => {
    stopsStore.addStop(msg.payload)
  })
  
  sse.on('stop_updated', (msg) => {
    stopsStore.updateStop(msg.payload.id, msg.payload)
  })
  
  sse.on('stop_closed', (msg) => {
    stopsStore.updateStop(msg.payload.id, {
      end_time: msg.payload.end_time,
      duration_s: msg.payload.duration_s
    })
  })
})
```

---

## Resumen

### Multi-Tenancy

1. **Aislamiento físico**: cada planta tiene su propia base de datos PostgreSQL
2. **Credenciales encriptadas**: AES-256-GCM en `admin.planta_databases`
3. **PlantaPoolManager**: gestiona pools de conexión por planta con cache
4. **Scope resolution**: `WithScope(ctx, Scope{})` inyecta contexto de tenant
5. **Schemas por línea**: `linea_{id}.*` dentro de cada DB de planta
6. **Escalabilidad**: soporte para instancias dedicadas (RDS) cuando crece

### Tablet en Tiempo Real

1. **SSE (Server-Sent Events)**: comunicación unidireccional servidor → cliente
2. **Reconexión automática**: backoff exponencial con límite de 30s
3. **Dual mode**: EDGE (local, sin auth) vs CLOUD (remoto, JWT)
4. **Polling complementario**: solo para datos muy volátiles (estado visión)
5. **Pinia stores**: estado reactivo centralizado
6. **Type-safe**: TypeScript end-to-end con interfaces compartidas

### Garantías

- **Consistencia**: comandos idempotentes con ACK explícito
- **Latencia baja**: <100ms en EDGE, <500ms en CLOUD
- **Offline-first**: edge opera independiente, sync diferida
- **Seguridad**: JWT con expiración, refresh tokens, HTTPS en cloud
- **Escalabilidad**: cada planta aislada, puede crecer sin afectar otras
