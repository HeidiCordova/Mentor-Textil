# Mentor Edge — Documentación Detallada del Cloud (OEE/Producción)

## Visión General

El cloud es el centro de operaciones para gestión multi-tenant, ingesta de datos OEE, dashboards analíticos y configuración centralizada. Consta de **6 microservicios Go**, un **frontend Vue 3** y una **base de datos PostgreSQL 16** con arquitectura multi-tenant.

---

## Arquitectura

```
┌────────────────────────────────────────────────────────────────────┐
│                         CLOUD (VPS Linux)                          │
│                                                                    │
│        Browser / Tablet Cloud                Edge Jetson           │
│              │ (JWT)                           │ (API Key)         │
│              ▼                                 ▼                   │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │              cloud-gateway :8888 (Go, net/http)           │     │
│  │  Router · JWT/APIKey Middleware · SSE Hub · Rate Limiter  │     │
│  │  Camera Hub · Audit Logger · Scope Enforcer               │     │
│  └───┬──────┬──────┬──────┬──────┬──────┬───────────────────┘     │
│      │      │      │      │      │      │                          │
│      ▼      ▼      ▼      ▼      ▼      ▼                         │
│  ┌──────┐┌──────┐┌──────┐┌──────┐┌──────┐┌──────────┐            │
│  │iden- ││ingest││config││analy-││integ-││ frontend │            │
│  │tity  ││:8082 ││:8083 ││tics  ││ration││ :80      │            │
│  │:8081 ││      ││      ││:8084 ││:8085 ││ Vue 3    │            │
│  └──────┘└──────┘└──────┘└──────┘└──────┘└──────────┘            │
│                                                                    │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │                 PostgreSQL 16 :5432                         │    │
│  │  Master DB (mentor_planta_0):                              │    │
│  │    identity · config · gateway · integration               │    │
│  │  Tenant DBs (mentor_planta_1, mentor_planta_2, ...):       │    │
│  │    linea_{id}.oee_snapshots · paradas · production_runs    │    │
│  └───────────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────────┘
```

---

## Servicios

### 1. cloud-gateway (Go, net/http — :8888)

#### Propósito
Router central que maneja todas las formas de autenticación, proxying a servicios internos, SSE hub para browsers y edge devices, rate limiting y audit logging.

#### Middlewares

| Middleware | Aplicación | Función |
|-----------|------------|---------|
| JWT | `/api/*` (excepto auth) | Extrae user_id, empresa_id, role del token |
| APIKey | `/api/v1/edge/*` | Valida X-API-Key del dispositivo edge |
| InternalKey | `/internal/*` | Valida X-Internal-Key entre servicios |
| ScopeEnforcer | Todas las rutas autenticadas | Inyecta X-Planta-ID, X-Linea-ID basado en request |
| RateLimiter | Todas | Token-bucket: edge 10/s, user 20/s |
| AuditLogger | Todas | Registra user/device, IP, latencia, status |

#### Rutas Edge (API Key)

```
POST   /api/v1/edge/*                → Proxy a cloud-ingest :8082
GET    /api/v1/edge/commands          → Poll comandos pendientes
POST   /api/v1/edge/commands/:id      → ACK de comando ejecutado
GET    /api/v1/edge/stream            → SSE para actualizaciones en tiempo real
GET    /api/v1/edge/plants-lines      → Config de referencia (plantas/líneas)
GET    /api/v1/edge/velocidad-nominal → Velocidades nominales actuales
GET    /api/v1/edge/linea-config      → Configuración por línea
```

#### Rutas Cloud Tablet (JWT + SSE)

```
POST   /api/v1/cloud/commands/:deviceId  → Enviar comando a edge específico
GET    /api/v1/cloud/stream?token=JWT    → SSE en tiempo real (token en URL para EventSource)
```

#### Rutas Auth (Sin JWT)

```
POST   /api/auth/login    → user/password → JWT access_token + refresh_token
POST   /api/auth/logout   → Revocar refresh token
POST   /api/auth/refresh  → Renovar access_token con refresh_token
```

#### Rutas Config (JWT → proxy a cloud-config :8083)

```
GET/POST   /api/plantas                     → CRUD de plantas
GET/PUT    /api/plantas/:id
GET        /api/lineas?planta_id=X          → Líneas por planta
POST       /api/lineas
GET/POST   /api/dispositivos                → Dispositivos edge
GET/POST   /api/variables                   → Variables OEE (CONTEO, MERMA)
GET/POST   /api/cat-paradas                 → Categorías de parada
GET        /api/arbol-paradas               → Árbol jerárquico de categorías
GET/POST   /api/canvas-oee                  → Matriz OEE para cálculos
GET/POST   /api/turno-dias                  → Definiciones de turnos
GET/POST   /api/velocidad-nominal           → Velocidades nominales
GET/POST   /api/productos                   → Catálogo de productos
GET/POST   /api/producto-caracteristicas    → Atributos de producto
GET/POST   /api/motivos-velocidad           → Razones de pérdida de velocidad
```

#### Rutas Analytics (JWT → proxy a cloud-analytics :8084)

```
GET        /api/oee/snapshots               → Series temporales de OEE
GET        /api/oee/summary                 → OEE promedio agregado
GET        /api/oee/latest                  → Último snapshot por línea
GET        /api/stops                       → Listar paradas con filtros
POST       /api/stops                       → Crear parada (dispatch a edge)
PUT        /api/stops/:id                   → Actualizar parada
POST       /api/stops/:id/justify           → Justificar parada
DELETE     /api/stops/:id                   → Eliminar parada
GET        /api/production-runs             → Listar corridas de producción
POST       /api/production-runs             → Crear/upsert corrida
DELETE     /api/production-runs/:id         → Eliminar corrida
GET        /api/dashboard/stats             → KPIs en tiempo real
GET        /api/dashboard/reportes          → Reportes por período
GET        /api/dashboard/graficos          → Datos para gráficos
GET        /api/analisis/general            → Análisis OEE general
GET        /api/analisis/produccion         → Análisis de producción
GET        /api/analisis/pareto             → Diagrama de Pareto (paradas)
GET        /api/analisis/combined           → Métricas combinadas
GET        /api/alarmas                     → Alertas configuradas
GET        /api/compromisos                 → Compromisos de mejora
```

#### Rutas Integration (JWT + X-API-Key externo)

```
POST       /api/integration                           → Crear API key externa
GET        /api/api-keys                              → Listar keys
DELETE     /api/integration/:id                       → Revocar key
GET        /api/v1/integration/oee/latest             → OEE actual (X-API-Key)
GET        /api/v1/integration/oee/snapshots          → Histórico OEE (X-API-Key)
GET        /api/v1/integration/paradas                → Paradas (X-API-Key)
```

#### Rutas Camera

```
POST   /api/v1/camera/push     → Edge pushea MJPEG (API Key)
GET    /api/v1/camera/stream   → Browser pull MJPEG (JWT o ?token=)
GET    /api/v1/camera/roi      → Región de interés actual
```

#### Rutas Internas (X-Internal-Key)

```
POST   /internal/broadcast   → Hub de eventos multi-conexión
POST   /internal/dispatch    → Envío de comandos a edge
POST   /internal/notify      → Notificación a browsers conectados
```

#### SSE Hub

El gateway mantiene conexiones SSE abiertas con:
- **Edge devices**: Reciben comandos y config sync en tiempo real
- **Browsers**: Reciben stop.changed, variable.updated y otros eventos

La tablet cloud se conecta con token en URL (`?token=JWT`) porque EventSource no soporta headers custom.

#### Camera Hub

Relay de MJPEG entre edge y browsers:
1. Edge hace `POST /api/v1/camera/push` con frames JPEG continuo
2. Browsers hacen `GET /api/v1/camera/stream` para recibir MJPEG
3. El gateway funciona como relay sin almacenar frames

#### Rate Limiting

Token-bucket por cliente con cleanup periódico:

| Tipo de Cliente | Capacidad | Refill/s |
|----------------|-----------|----------|
| Edge device | 20 | 10 |
| User (JWT) | 50 | 20 |

Buckets inactivos se evictan tras 30 minutos.

---

### 2. cloud-identity (Go, Gin — :8081)

#### Propósito
Gestión de usuarios, empresas, roles y tokens JWT. Punto de autenticación central.

#### Endpoints

```
POST   /auth/login       → { username, password } → { access_token, refresh_token, user }
POST   /auth/logout      → Revocar refresh token
POST   /auth/refresh     → { refresh_token } → { access_token }

GET    /usuarios          → Lista de usuarios (filtrado por empresa si no es ADMIN)
POST   /usuarios          → Crear usuario
GET    /usuarios/:id      → Detalle de usuario
PUT    /usuarios/:id      → Actualizar (rol, empresa_id, estado)
DELETE /usuarios/:id      → Desactivar usuario

GET    /empresas          → Lista de empresas (solo ADMIN)
POST   /empresas          → Crear empresa
PUT    /empresas/:id      → Actualizar empresa

GET    /roles             → Lista de roles disponibles
```

#### JWT Token

```json
{
  "user_id": 123,
  "username": "jperez",
  "empresa_id": 5,
  "user_role": "OPERATOR",
  "exp": 1726234567,
  "iat": 1726145167
}
```

#### Roles

| Rol | Permisos |
|-----|----------|
| ADMIN | Acceso total, todas las empresas, gestión de usuarios |
| SUPERADMIN | Acceso completo dentro de su empresa |
| ADMIN_PLANTA | Gestión de plantas bajo su empresa |
| OPERATOR | Operación de líneas asignadas |
| VIEWER | Solo lectura |
| SUPERVISOR | Supervisión + reportes |

#### Tablas

```sql
identity.usuarios (
    id          SERIAL PRIMARY KEY,
    username    VARCHAR UNIQUE,
    email       VARCHAR,
    nombre      VARCHAR,
    password_hash VARCHAR,
    rol_id      INT REFERENCES identity.roles,
    empresa_id  INT REFERENCES identity.empresas,
    activo      BOOLEAN DEFAULT TRUE
);

identity.empresas (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR,
    ruc         VARCHAR,
    direccion   TEXT,
    telefono    VARCHAR,
    email       VARCHAR,
    responsable VARCHAR,
    estado      BOOLEAN DEFAULT TRUE
);

identity.roles (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR,    -- ADMIN, OPERATOR, VIEWER, SUPERVISOR
    descripcion TEXT
);

identity.refresh_tokens (
    id          UUID PRIMARY KEY,
    usuario_id  INT REFERENCES identity.usuarios,
    token_hash  VARCHAR,
    expires_at  TIMESTAMPTZ,
    revocado    BOOLEAN DEFAULT FALSE
);

identity.api_keys (
    id          UUID PRIMARY KEY,
    nombre      VARCHAR,
    key_hash    VARCHAR,
    empresa_id  INT,
    scopes      JSONB,      -- ["oee:read", "paradas:read"]
    activo      BOOLEAN DEFAULT TRUE
);
```

---

### 3. cloud-ingest (Go, Gin — :8082)

#### Propósito
Punto único de ingesta para datos OEE, paradas y production runs enviados desde los dispositivos edge.

#### Endpoints Edge (API Key)

```
POST   /api/v1/edge/oee                  → Batch de OEE records
POST   /api/v1/edge/stops-sync           → Sincronizar paradas
POST   /api/v1/edge/production-runs-sync → Sincronizar production runs
POST   /api/v1/edge/heartbeat            → Latido + resolución de scope
GET    /api/v1/edge/pending-commands     → Poll de comandos cloud→edge
POST   /api/v1/edge/pending-commands/ack → ACK de ejecución
```

#### Endpoint JWT

```
GET    /datos-recibidos                  → Lista de raw events (admin)
```

#### Procesamiento de OEE

**Entrada**: JSON con formato columnar compacto

```json
{
  "code": "ISM_AQP_PERSISTENCIA_L7",
  "time": 1711270000000,
  "device_id": "jetson-orin-01",
  "interval_s": 300,
  "v": 5,
  "head": ["T_DISPONIBLE", "T_MICROPARADA", "T_PARADA_NO_ASIGNADA",
           "T_PARADA_PROGRAMADA", "T_REFRIGERIO", "T_CAPACITACION_OBLIGATORIA",
           "T_MANTENIMIENTO_PLANIFICADO", "CONTEO_1", "MERMA"],
  "data": ["300000", "15000", "0", "0", "0", "0", "0", "8500", "45"]
}
```

**Procesamiento**:
1. Valida estructura y tipos
2. Resuelve scope multi-tenant: device_id → planta_id + linea_id
3. Obtiene velocidad_nominal de la linea para calcular pérdida de velocidad
4. Calcula métricas OEE (ver fórmulas abajo)
5. Inserta en `linea_{id}.oee_snapshots`
6. Si hay cambio de variables, notifica al gateway para broadcast

#### Cálculo OEE en Cloud

```
// Paradas obligatorias (no afectan disponibilidad)
Parada_Obligatoria = T_PARADA_PROGRAMADA + T_REFRIGERIO +
                     T_CAPACITACION_OBLIGATORIA + T_MANTENIMIENTO_PLANIFICADO

// Tiempo disponible neto
T_Disponible = max(T_DISPONIBLE - Parada_Obligatoria, 0)

// Paradas no obligatorias
T_Parada_No_Obligatoria = T_PARADA_NO_PROGRAMADA + T_PARADA_NO_ASIGNADA
T_Operativo = max(T_Disponible - T_Parada_No_Obligatoria, 0)

// DISPONIBILIDAD
Disponibilidad = (T_Operativo / T_Disponible) × 100

// Velocidad nominal (units per second)
T_Nominal_Produccion = CONTEO_1 / velocidad_us
T_Neto = max(T_Operativo - T_MICROPARADA, 0)
T_Perdida_Velocidad = max(T_Neto - T_Nominal_Produccion, 0)

// RENDIMIENTO
Rendimiento = ((T_Operativo - T_MICROPARADA - T_Perdida_Velocidad) / T_Operativo) × 100

// CALIDAD
Calidad = ((CONTEO_1 - MERMA) / CONTEO_1) × 100

// OEE
OEE = (Disponibilidad/100) × (Rendimiento/100) × (Calidad/100) × 100
```

#### Heartbeat

Los dispositivos edge envían heartbeat cada 60s:
- Cloud responde con scope canónico (device_id, empresa_id, planta_id)
- Edge actualiza su configuración si el scope cambió
- Permite detección de dispositivos offline

#### Tablas

```sql
-- Eventos crudos (auditoría)
ingest.raw_events (
    id          BIGSERIAL PRIMARY KEY,
    device_id   VARCHAR,
    event_type  VARCHAR,
    payload     JSONB,
    received_at TIMESTAMPTZ DEFAULT NOW()
);

-- OEE snapshots calculados (por línea tenant)
linea_{id}.oee_snapshots (
    id              BIGSERIAL PRIMARY KEY,
    device_id       VARCHAR,
    fecha           DATE,
    hora            TIMESTAMPTZ,
    disponibilidad  REAL,
    rendimiento     REAL,
    calidad         REAL,
    oee             REAL,
    produccion      INT,
    interval_s      INT
);
```

---

### 4. cloud-config (Go, Gin — :8083)

#### Propósito
Gestiona toda la configuración maestra: plantas, líneas, dispositivos, productos, categorías de parada, turnos y velocidades nominales.

#### Endpoints Principales (JWT)

```
GET/POST   /api/plantas                    → CRUD de plantas
GET/PUT    /api/plantas/:id
GET        /api/lineas?planta_id=X         → Líneas por planta
POST       /api/lineas                     → Crear línea (ejecuta template SQL)
GET/POST   /api/dispositivos               → CRUD de dispositivos edge
GET/POST   /api/variables                  → Variables dinámicas (CONTEO_1, MERMA)
GET/POST   /api/cat-paradas                → Categorías jerárquicas de parada
GET        /api/arbol-paradas              → Árbol completo de categorías
GET/POST   /api/productos                  → Catálogo de productos
GET/POST   /api/canvas-oee                 → Matriz OEE para cálculos
GET/POST   /api/turno-dias                 → Definiciones de turnos de trabajo
GET/POST   /api/velocidad-nominal          → Velocidades nominales (u/s)
GET/POST   /api/motivos-velocidad          → Razones de pérdida de velocidad
GET/POST   /api/producto-caracteristicas   → Atributos de producto
```

#### Endpoint Edge (Sin auth)

```
GET    /api/v1/edge/velocidad-nominal  → Edge lee velocidad actual sin JWT
```

#### Provisioning de Línea

Cuando se crea una nueva línea:
1. Se ejecuta el template SQL que crea el schema `linea_{id}`
2. Se crean todas las tablas (oee_snapshots, paradas, production_runs, etc.)
3. Se insertan categorías de parada obligatorias por defecto
4. Se notifica al gateway para broadcast de sync

#### Scope Enforcement

Cada request JWT incluye `empresa_id`. El servicio filtra automáticamente:
- Plantas: `WHERE empresa_id = $empresa`
- Líneas: `WHERE planta_id IN (SELECT id FROM plantas WHERE empresa_id = $empresa)`
- Excepción: rol ADMIN ve todo sin filtro

#### Encriptación de Credenciales

Las credenciales de BD por planta (para multi-tenant) se almacenan encriptadas con AES:
```
admin.planta_databases (
    planta_id   INT PRIMARY KEY,
    db_host     VARCHAR,
    db_name     VARCHAR,
    db_user_enc VARCHAR,    -- Encriptado AES
    db_pass_enc VARCHAR     -- Encriptado AES
)
```

#### Tablas Config

```sql
config.plantas (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR,
    empresa_id  INT
);

config.lineas (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR,
    planta_id   INT REFERENCES config.plantas
);

config.dispositivos (
    id          SERIAL PRIMARY KEY,
    device_id   VARCHAR UNIQUE,
    linea_id    INT,
    planta_id   INT,
    empresa_id  INT,
    estado      VARCHAR,
    ultimo_ping TIMESTAMPTZ
);

config.variables (
    id              SERIAL PRIMARY KEY,
    nombre          VARCHAR,
    clave           VARCHAR,    -- CONTEO_1, MERMA, etc.
    valor           TEXT,
    tipo            VARCHAR,
    dispositivo_id  INT
);

config.productos (
    id          SERIAL PRIMARY KEY,
    codigo      VARCHAR,
    nombre      VARCHAR,
    empresa_id  INT
);

config.categoria_paradas (
    id          SERIAL PRIMARY KEY,
    nombre      VARCHAR,
    padre_id    INT,        -- Jerárquico (NULL = raíz)
    empresa_id  INT,
    linea_id    INT         -- NULL = global
);
```

---

### 5. cloud-analytics (Go, Gin — :8084)

#### Propósito
Motor de cálculos OEE, consultas analíticas, gestión de paradas y reporting. Es el servicio que la tablet y el frontend cloud consultan para datos operacionales.

#### Endpoints OEE (JWT)

```
GET   /api/oee/snapshots?linea_id=X&desde=...&hasta=...&min_interval_s=300
      → Array de OEESnapshot con Disponibilidad/Rendimiento/Calidad/OEE
      → JOIN con production_runs para contexto de producto

GET   /api/oee/summary?linea_id=X&desde=...&hasta=...
      → Promedio agregado de métricas OEE

GET   /api/oee/latest?linea_id=X
      → Último snapshot con métricas actuales
```

#### Endpoints de Paradas (JWT)

```
GET   /api/stops?linea_id=X&justified=false&open=true&since=...&until=...
      → CloudStop[] con categoría, duración, justificación

GET   /api/stops/summary?linea_id=X&hours=24
      → { total_duration_min, count_by_category, open_count }

POST  /api/stops
      Body: { linea_id, device_id, inicio, categoria_id, reason }
      → Crea parada en analytics.paradas
      → Inserta pending_command para propagación a edge
      → Notifica a browsers vía SSE

PUT   /api/stops/:id
      → Actualiza propiedades de parada

POST  /api/stops/:id/justify
      Body: { reason, categoria_id }
      → Marca justified=true
      → Notifica a browsers vía SSE

DELETE /api/stops/:id
      → Elimina parada
```

#### Endpoints Production Runs (JWT)

```
GET   /api/production-runs?device_id=X&linea_id=Y
      → ProductionRun[] con producto, SKU, timestamps

POST  /api/production-runs
      Body: { run_id, device_id, linea_id, producto_id, nombre, sku, started_at, ended_at }
      → Upsert corrida
      → Si device_id empieza con "cloud-", crea pending_command para propagación a edge

DELETE /api/production-runs/:id
```

#### Endpoints Dashboard (JWT)

```
GET   /api/dashboard/stats?empresa_id=X
      → { oee_avg, disponibilidad, rendimiento, calidad, produccion_total, stops_count }

GET   /api/dashboard/reportes?from=...&to=...
      → Reportes con KPIs históricos

GET   /api/dashboard/graficos
      → Series temporales para gráficos
```

#### Endpoints Análisis (JWT)

```
GET   /api/analisis/general?linea_id=X&desde=...&hasta=...
      → Análisis OEE completo por período

GET   /api/analisis/produccion
      → Análisis de producción (conteos, rendimiento)

GET   /api/analisis/pareto
      → Diagrama de Pareto (razones de parada ordenadas por impacto)

GET   /api/analisis/combined
      → Métricas combinadas de múltiples dimensiones
```

#### Query OEE con Contexto de Producto

```sql
SELECT s.id, s.device_id, s.hora,
       s.disponibilidad, s.rendimiento, s.calidad, s.oee,
       pr.nombre, pr.sku
FROM linea_42.oee_snapshots s
LEFT JOIN linea_42.production_runs pr ON
    s.device_id = pr.device_id
    AND s.hora >= pr.started_at
    AND (pr.ended_at IS NULL OR s.hora <= pr.ended_at)
WHERE s.hora >= $1 AND s.hora <= $2
ORDER BY s.hora DESC
```

#### Tablas por Línea

```sql
-- Paradas (schema linea_{id})
analytics.paradas (
    id              BIGSERIAL PRIMARY KEY,
    stop_id         UUID UNIQUE,
    device_id       VARCHAR,
    linea_id        INT,
    categoria_id    INT,
    categoria_nombre VARCHAR,
    stop_type       VARCHAR,
    inicio          TIMESTAMPTZ,
    fin             TIMESTAMPTZ,
    duracion_min    REAL,
    justified       BOOLEAN DEFAULT FALSE,
    reason          TEXT,
    justified_by    VARCHAR,
    source          VARCHAR     -- detector, operator, cloud, system
);

-- Production runs (schema linea_{id})
analytics.production_runs (
    id              BIGSERIAL PRIMARY KEY,
    run_id          UUID UNIQUE,
    device_id       VARCHAR,
    linea_id        INT,
    producto_id     INT,
    nombre          VARCHAR,
    sku             VARCHAR,
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ
);
```

#### Flujo: Parada Creada desde Cloud

```
1. Tablet POST /api/stops { linea_id: 42, device_id: "jetson-14-42", categoria_id: 5 }
2. Gateway JWT auth → route a analytics
3. Analytics INSERT INTO linea_42.analytics.paradas (...)
4. Analytics INSERT INTO analytics.pending_commands (
     device_id, command_type='APPLY_STOP', payload, status='pending')
5. Analytics POST /internal/notify → gateway broadcast SSE
6. Browser recibe event stop.changed → actualiza UI
7. Edge poll GET /api/v1/edge/pending-commands → recibe APPLY_STOP
8. Edge ejecuta localmente → ACK back
```

---

### 6. cloud-integration (Go, Gin — :8085)

#### Propósito
API keys y endpoints para integraciones con sistemas externos (ERP, MES, BI).

#### Endpoints de Gestión (JWT)

```
POST   /api/integration   → Crear API key con scopes
GET    /api/api-keys       → Listar keys activas
DELETE /api/integration/:id → Revocar key
```

#### Endpoints de Consulta (X-API-Key)

```
GET   /api/v1/integration/oee/latest?linea_id=X
      Header: X-API-Key: sk_live_...
      Scope requerido: oee:read
      → Último OEE snapshot

GET   /api/v1/integration/oee/snapshots?linea_id=X&limite=100
      Scope requerido: snapshots:read
      → Array de snapshots históricos

GET   /api/v1/integration/paradas?linea_id=X&desde=...&hasta=...
      Scope requerido: paradas:read
      → Array de paradas
```

#### Scopes Disponibles

| Scope | Acceso |
|-------|--------|
| `oee:read` | `/oee/latest` |
| `snapshots:read` | `/oee/snapshots` históricos |
| `paradas:read` | Lista de paradas |

---

## Multi-Tenancy

### Arquitectura de Dos Niveles

**Nivel 1 — Master Database** (`mentor_planta_0`):
Contiene datos compartidos entre todas las empresas.

```
identity.usuarios
identity.empresas
identity.roles
identity.refresh_tokens
identity.api_keys
config.plantas
config.lineas
config.dispositivos
config.variables
config.productos
config.categoria_paradas
admin.planta_databases    ← Registro de BDs tenant con credenciales encriptadas
gateway.audit_log
gateway.commands
```

**Nivel 2 — Tenant Databases** (`mentor_planta_1`, `mentor_planta_2`, ...):
Base de datos dedicada por planta para aislamiento total de datos operacionales.

```
linea_42/
├── oee_snapshots         → Métricas OEE por intervalo
├── paradas               → Paradas detectadas y justificadas
├── production_runs       → Corridas de producción
├── alarmas               → Alertas configuradas
├── pending_commands      → Comandos cloud→edge pendientes

linea_43/
├── (mismo schema)
```

### PoolResolver — Resolución de Conexión

```
1. Request llega con contexto: planta_id=14, linea_id=42
2. multitenancy.ScopeFromRequest() extrae Scope{PlantaID: 14, LineaID: 42}
3. PoolResolver.Resolve(ctx):
   a. Consulta PlantaPoolManager
   b. Busca en admin.planta_databases WHERE planta_id=14
   c. Desencripta credenciales con AES
   d. Crea/reutiliza pgx.Pool (cached en sync.Map)
   e. Retorna (*Pool, "linea_42", nil)
4. Servicio ejecuta: SELECT * FROM linea_42.oee_snapshots WHERE ...
```

Cada planta tiene su propia base de datos y pool de conexiones. Los pools se crean lazy y se cachean.

---

## Frontend Cloud (Vue 3)

### Stack

- Vue 3 + Vite
- Pinia (estado)
- Vue Router 4
- Tailwind CSS + dark mode

### Módulos

```
/modules
├── auth/                     → Login
├── dashboard/                → KPIs en tiempo real, widgets OEE
├── configuracion/
│   ├── EmpresaView           → Gestión de empresas
│   ├── PlantasView           → Gestión de plantas
│   ├── LineasView            → Gestión de líneas
│   ├── DispositivosView      → Gestión de dispositivos edge
│   ├── VariablesView         → Variables OEE del sistema
│   └── ArbolParadasView      → Editor de categorías de parada
├── administracion/
│   ├── TurnosView            → Configuración de turnos
│   ├── ProductosView         → Catálogo de productos
│   ├── CanvasOEEView         → Editor de matriz OEE
│   ├── LaboratorioOEEView    → Simulador de cálculo OEE
│   └── VelocidadNominalView  → Velocidades nominales por producto
├── analisis/
│   ├── GeneralView           → Series temporales OEE
│   └── ProduccionView        → Análisis de producción
├── vista-rapida/             → Vistas rápidas (OEE por turno)
└── reportes/                 → Generación de reportes
```

### SSE en Frontend

```javascript
const eventSource = new EventSource(
  '/api/v1/cloud/stream?token=' + authStore.access_token
)
eventSource.addEventListener('stop.changed', (e) => {
  // Actualizar UI con nueva parada
})
eventSource.addEventListener('variable.updated', (e) => {
  // Refrescar variables en store
})
```

---

## Scoped Sync — Sincronización por Scope

Cuando un edge device se conecta al SSE del cloud, recibe sincronización filtrada por su scope (empresa/planta/línea):

| Recurso | Tipo de Comando | Fuente |
|---------|----------------|--------|
| categoria_paradas | SYNC_CATALOG | cloud-config |
| productos | SYNC_PRODUCTOS | cloud-config |
| turnos | SYNC_TURNOS | cloud-config |
| variables | SYNC_VARIABLES | cloud-config |
| usuarios | SYNC_USUARIOS | cloud-identity |
| plantas_lineas | SYNC_PLANTAS_LINEAS | cloud-config |
| velocidad_nominal | SYNC_VELOCIDAD_NOMINAL | cloud-config |
| paradas | SYNC_PARADAS | cloud-analytics |

Esto permite que el edge tenga una copia local actualizada de catálogos y configuración, funcionando offline-first.

---

## Autenticación — Matriz Completa

| Path | Tipo Auth | Validado Por | Claims |
|------|-----------|-------------|--------|
| `/api/auth/*` | Ninguno | cloud-identity | — |
| `/api/v1/edge/*` | X-API-Key | gateway middleware | device_id, empresa_id |
| `/api/*` (resto) | Bearer JWT | gateway middleware | user_id, empresa_id, role |
| `/internal/*` | X-Internal-Key | gateway middleware | service_name |
| `/api/v1/integration/*` | X-API-Key | cloud-integration | empresa_id, scopes[] |
| `/*` (catch-all) | Ninguno | Sirve frontend SPA | — |

### Patrón de Autorización en Handlers

```go
// Middleware extrae JWT → contexto
func (m *Middleware) JWT(next http.Handler) http.Handler {
    claims := jwt.Parse(token, m.secret)
    ctx = context.WithValue(ctx, "user_id", claims.UserID)
    ctx = context.WithValue(ctx, "empresa_id", claims.EmpresaID)
    ctx = context.WithValue(ctx, "user_role", claims.Role)
}

// Handler filtra por empresa_id
func (h *Handler) ListStops(c *gin.Context) {
    empresaID := c.Request.Context().Value("empresa_id").(int64)
    // Query: WHERE empresa_id = $empresaID AND ...
}

// ADMIN bypass
func resolveEmpresaID(ctx) *int {
    if role == "ADMIN" { return nil }  // Sin filtro
    return &empresaID
}
```

---

## Flujos de Datos Completos

### Edge → Cloud (Ingesta continua)

```
1. vision-event-detector genera OEE snapshot (cada 300s)
2. POST /events a resiliencia → guarda en events_buffer
3. enviador goroutine main_sync (cada sync_interval_s):
   a. SELECT events WHERE synced=false AND dead=false LIMIT 50
   b. POST /api/v1/edge/oee al cloud con retry exponencial
   c. Cloud-ingest valida, resuelve scope, calcula OEE
   d. INSERT INTO linea_{id}.oee_snapshots
   e. Cloud responde 200 OK
   f. resiliencia marca synced=true
4. Si falla: retry_count++ → si >8 → marca dead
```

### Cloud → Edge (Comando)

```
1. Admin en cloud crea parada: POST /api/stops
2. cloud-analytics:
   a. INSERT INTO linea_{id}.analytics.paradas
   b. INSERT INTO pending_commands (type=APPLY_STOP, status=pending)
   c. POST /internal/notify → gateway SSE broadcast
3. Browser recibe stop.changed → actualiza dashboard
4. Edge enviador goroutine commands (cada 3s):
   a. GET /api/v1/edge/pending-commands
   b. Recibe [{command_id, type: APPLY_STOP, payload}]
   c. Ejecuta: INSERT INTO linea_{id}.stops
   d. POST /api/v1/edge/pending-commands/ack {command_id, status: executed}
5. Cloud marca comando como acknowledged
```

### Tablet Cloud → Edge (Justificación remota)

```
1. Tablet POST /api/stops/{id}/justify {reason, categoria_id}
2. Gateway JWT auth → route a analytics
3. analytics UPDATE paradas SET justified=true, reason=$reason
4. analytics POST /internal/notify → SSE broadcast
5. analytics INSERT pending_command (type=JUSTIFY_STOP, payload)
6. Edge poll pending-commands → ejecuta justificación local
7. Edge ACK → cloud marca acknowledged
```

---

## Deploy

### Docker Compose Cloud

```yaml
services:
  postgres:
    image: postgres:16
    ports: ["5432:5432"]

  cloud-gateway:
    build: ./services/cloud-gateway
    ports: ["8888:8888"]
    environment:
      - IDENTITY_URL=http://cloud-identity:8081
      - INGEST_URL=http://cloud-ingest:8082
      - CONFIG_URL=http://cloud-config:8083
      - ANALYTICS_URL=http://cloud-analytics:8084
      - INTEGRATION_URL=http://cloud-integration:8085

  cloud-identity:
    build: ./services/cloud-identity
    ports: ["8081:8081"]

  cloud-ingest:
    build: ./services/cloud-ingest
    ports: ["8082:8082"]

  cloud-config:
    build: ./services/cloud-config
    ports: ["8083:8083"]

  cloud-analytics:
    build: ./services/cloud-analytics
    ports: ["8084:8084"]

  cloud-integration:
    build: ./services/cloud-integration
    ports: ["8085:8085"]

  frontend:
    build: ./Frontend/template_mentor
    ports: ["80:80"]

  nginx:
    image: nginx
    ports: ["443:443"]      # TLS termination
    depends_on: [cloud-gateway, frontend]
```

### Variables de Entorno Clave

| Variable | Servicio | Descripción |
|----------|----------|-------------|
| `DATABASE_URL` | Todos | Conexión a PostgreSQL master |
| `JWT_SECRET` | identity, gateway | Secreto para firmar/verificar JWT |
| `INTERNAL_API_KEY` | gateway, analytics | Key para comunicación interna |
| `AES_KEY` | config | Clave para encriptar credenciales tenant |
| `EDGE_API_KEY` | gateway | Key para autenticar dispositivos edge |
