# 2. DISEÑO DEL SOFTWARE — Diseño y Desarrollo del Software

## 2.1 Filosofía de Diseño

El sistema MENTOR EDGE fue diseñado bajo los siguientes principios arquitectónicos:

| Principio | Descripción |
|-----------|-------------|
| **Offline-First** | El dispositivo Edge opera de forma completamente autónoma; la sincronización con la nube es diferida y no bloquea la operación |
| **Arquitectura Hexagonal** | La lógica de dominio está desacoplada de la infraestructura en cada microservicio |
| **Event-Driven** | Flujo asíncrono de eventos desde la cámara hasta la nube |
| **Multi-Tenant** | La nube soporta múltiples empresas, plantas y líneas con aislamiento de datos |
| **Zero-Trust** | Autenticación separada entre dispositivos (API Key), usuarios (JWT) y servicios internos (Internal Key) |

---

## 2.2 Arquitectura General

```
                          CLOUD (Docker Compose en VPS)
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
 │   │         (API proxy, SSE hub, auditoría, CORS)       │  │
 │   └──────┬──────────────────┬───────────────────┬───────┘  │
 │          │                  │                   │          │
 │   ┌──────┴───────┐   ┌─────┴────────┐  ┌──────┴───────┐  │
 │   │  cloud-      │   │  cloud-      │  │  cloud-      │  │
 │   │  analytics   │   │  integration │  │  frontend    │  │
 │   │  :8084       │   │  :8085       │  │  :80 (nginx) │  │
 │   └──────────────┘   └──────────────┘  └──────────────┘  │
 │                                                             │
 │   ┌─────────────────────────────────────────────────────┐  │
 │   │              PostgreSQL 16  :5432                    │  │
 │   │   (identity | config | ingest | analytics |         │  │
 │   │    gateway | integration)                           │  │
 │   └─────────────────────────────────────────────────────┘  │
 └──────────────────────────────┬──────────────────────────────┘
                                │ HTTPS :8888
           ┌────────────────────┼────────────────────┐
           ▼                    ▼                    ▼
    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
    │ Jetson #1   │     │ Jetson #2   │     │ Jetson #N   │
    └─────────────┘     └─────────────┘     └─────────────┘
```

---

## 2.3 Microservicios Edge (Dispositivo Jetson)

### 2.3.1 Vision Event Detector — Motor de Visión Artificial

**Lenguaje:** Python 3.10 con aceleración GPU NVIDIA

**Arquitectura Hexagonal:**

```
┌─────────────────────────────────────────────────┐
│                   DOMINIO                        │
│  ┌───────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ ROI       │ │ Fusión   │ │ FSM (Máquina  │  │
│  │ Manager   │ │ Señales  │ │ de Estados)   │  │
│  └───────────┘ └──────────┘ └───────────────┘  │
│  ┌───────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ Calibra-  │ │ Watchdog │ │ Señales:      │  │
│  │ ción      │ │          │ │ Edge,Hist,    │  │
│  │           │ │          │ │ Flow,Beige    │  │
│  └───────────┘ └──────────┘ └───────────────┘  │
├─────────────────────────────────────────────────┤
│                   PUERTOS                        │
│  FrameInput    │  ConfigPort    │  EventOutput   │
├─────────────────────────────────────────────────┤
│                 ADAPTADORES                      │
│  OpenCVAdapter │  ConfigClient  │  HTTPAdapter   │
│  GStreamer     │  (HTTP→8004)   │  (HTTP→8002)   │
└─────────────────────────────────────────────────┘
```

**Pipeline de procesamiento por frame:**

```
RTSP Camera (H.264)
    │
    ▼ NVDEC (decodificación por hardware)
    │
    ▼ Extracción de ROI (región de interés)
    │
    ├──→ Edge Signal (Canny, densidad de bordes)
    ├──→ Histogram Signal (correlación HSV)
    ├──→ Flow Signal (flujo óptico vía OFA chip)
    └──→ Beige Signal (detección de color HSV)
    │
    ▼ Motor de Fusión (pesos configurables)
    │
    ▼ FSM: IDLE → DETECTING → CONFIRMING → COOLDOWN
    │
    ▼ Evento confirmado → POST a Resiliencia
```

**Optimización de CPU por aceleración hardware:**

| Etapa | CPU antes | CPU después | Ahorro |
|-------|-----------|-------------|--------|
| Decodificación video | 25% | ~0% (NVDEC) | −25% |
| Flujo óptico | 30% | ~0% (OFA) | −30% |
| Conversión color | 11% | ~0% (VIC) | −11% |
| **Total por cámara** | **106%** | **30.8%** | **−71%** |

Esto permite procesar **5+ cámaras simultáneamente** en un solo Jetson.

### 2.3.2 Resiliencia — Buffer de Eventos

**Lenguaje:** Go 1.24

**Función:** Garantizar la persistencia local de cada evento detectado, incluso sin conexión a internet.

**Características:**
- Deduplicación por UUID de evento (caché en memoria de 10,000 eventos)
- Ordenamiento temporal garantizado
- Almacenamiento en PostgreSQL local con schema por línea (`linea_1`, `linea_2`)
- Mantenimiento horario automático (limpieza y optimización)

**Esquema de datos:**
```sql
linea_N.events_buffer (
    event_id    UUID UNIQUE,
    device_id   VARCHAR,
    event_type  VARCHAR,
    timestamp   TIMESTAMPTZ,
    payload     JSONB,
    synced      BOOLEAN DEFAULT false,
    retry_count INT DEFAULT 0
)
```

### 2.3.3 Enviador — Sincronización Edge → Cloud

**Lenguaje:** Go 1.24

**Función:** Transmitir eventos buffereados hacia la nube de forma confiable.

**Política de Retry:**
| Parámetro | Valor |
|-----------|-------|
| Delay inicial | 2 segundos |
| Delay máximo | 5 minutos |
| Factor backoff | 2.0x |
| Reintentos máximos | 5 |

**Ciclo de sincronización:**
```
1. Consultar eventos no sincronizados (batch de 50)
2. Enviar a cloud-gateway: POST /api/v1/edge/oee
3. Si éxito: marcar synced_at = NOW()
4. Sincronizar paradas y production runs
5. Heartbeat cada 60s (detectar conectividad)
6. Repetir ciclo cada 5 segundos
```

### 2.3.4 Edge Config Service — Configuración Dinámica

**Lenguaje:** Go 1.24

**Función:** Fuente única de verdad para la configuración de cada línea en el dispositivo.

**Estructura de configuración:**
```
LineConfig
├── ROI (región de interés: x, y, width, height)
├── Thresholds (umbrales de señales: edge, color, flow)
├── BeigeHSVRange (parámetros de detección de color)
├── FSMConfig (n_frames, cooldown)
├── Camera (URL RTSP, backend, frame_skip)
├── OEEConfig (nombre línea, intervalo de snapshot)
├── CloudConfig (URL gateway, API key)
└── TabletConfig (configuración de tablet)
```

**Validaciones:**
- ROI dimensiones > 0
- Umbrales ∈ [0, 1]
- n_frames ∈ [1, 30]
- cooldown ∈ [0, 60]
- Modo = textil

**Versionamiento automático:** Cada actualización incrementa `config_version` via trigger PostgreSQL.

### 2.3.5 Edge Gateway — Gateway Local

**Lenguaje:** Go 1.24

**Función:** Punto de entrada único para la comunicación con la nube y orquestación de líneas.

**Características:**
- Conexión SSE persistente con cloud-gateway (Server-Sent Events)
- Recepción de comandos de sincronización en tiempo real
- Traducción de IDs: `cloud_linea_id` → `local_linea_id`
- Sincronización de catálogos: productos, turnos, categorías, variables
- Pool de conexiones DB (15 max, 5 idle)
- Orquestación multi-línea

### 2.3.6 UI Local — Dashboard Web

**Tecnología:** Vue 3 + Tailwind CSS + Vite

**Vistas:**
- **Dashboard** — Estado en tiempo real de la producción
- **Configuración** — Ajuste de ROI, umbrales, parámetros FSM
- **Estado** — Diagnóstico de servicios y conectividad
- **Paradas** — Registro de paradas con categorización
- **Dispositivos** — Registro y provisioning de dispositivos

### 2.3.7 Tablet App — App para Operadores

**Tecnología:** Vue 3 + Pinia + Capacitor 6

**Función:** Interfaz simplificada para operadores en planta, con capacidades offline.

**Capacidades nativas (Capacitor):**
- Detección de estado de red (online/offline)
- Almacenamiento local de preferencias
- Compatible con Android e iOS

---

## 2.4 Microservicios Cloud (Servidor)

### 2.4.1 Cloud Gateway — Punto de Entrada Único

**Puerto:** 8888

**Funciones:**
- Proxy reverso hacia todos los microservicios
- Hub de Server-Sent Events (SSE) para dispositivos edge
- Rate limiting por token-bucket (20 req/s edge, 50 req/s usuario)
- Auditoría de todas las peticiones
- Middleware de autenticación (JWT, API Key, Internal Key)

**Matriz de autenticación:**

| Ruta | Tipo de Auth |
|------|-------------|
| `/api/auth/*` | Ninguna (login) |
| `/api/v1/edge/*` | X-API-Key (dispositivo) |
| `/api/*` | Bearer JWT (usuario) |
| `/internal/*` | X-Internal-Key (servicio a servicio) |
| `/*` | Ninguna (frontend SPA) |

### 2.4.2 Cloud Identity — Autenticación y Usuarios

**Puerto:** 8081

**Funciones:** Gestión de empresas, usuarios, roles, autenticación JWT, refresh tokens, API keys.

### 2.4.3 Cloud Config — Configuración Centralizada

**Puerto:** 8083

**Funciones:** CRUD de plantas, líneas, dispositivos, productos, turnos, variables, categorías de paradas, velocidades nominales.

**Arquitectura multi-tenant por base de datos:**
```
config.lineas (linea_id=14) 
    → config.plantas (planta_id=14) 
        → admin.planta_databases → "mentor_planta_14"
            → linea_14.productos (datos reales)
```

El sistema `resolvePlantaPool` enruta dinámicamente las consultas a la BD correcta según la planta de cada línea.

### 2.4.4 Cloud Ingest — Ingesta de Datos IoT

**Puerto:** 8082

**Funciones:** Recepción y almacenamiento de snapshots OEE, sincronización de paradas, production runs desde dispositivos edge.

**Métricas Prometheus:** Exporta contadores de eventos ingestados, latencias, errores.

### 2.4.5 Cloud Analytics — Dashboards y Reportes

**Puerto:** 8084

**Funciones:** Consultas OEE por turno/día/rango, análisis Pareto de paradas, gestión de alarmas, compromisos de producción.

### 2.4.6 Cloud Integration — API Externa

**Puerto:** 8085

**Funciones:** API con autenticación por API key para sistemas externos (ERP, MES). Exposición de datos OEE y paradas para integración con terceros.

### 2.4.7 Cloud Frontend — Dashboard de Gestión

**Tecnología:** Vue 3 + ECharts + Vue Flow + Excel Export

**Características:**
- Gráficos interactivos OEE con Apache ECharts
- Diagramas de flujo de procesos con Vue Flow
- Exportación a Excel (exceljs/xlsx)
- Gestión de configuración de plantas, líneas, productos
- Monitoreo en tiempo real de dispositivos edge

---

## 2.5 Base de Datos

### 2.5.1 Edge (PostgreSQL 14)

```
mentor_edge
├── config (schema compartido)
│   └── line_config, device_configs
├── linea_1 (schema por línea)
│   ├── productos, turnos, variables
│   ├── cat_programada, cat_no_programada
│   ├── velocidad_nominal, linea_producto_vars
│   ├── producto_caracteristicas
│   ├── raw_events, oee_snapshots
│   ├── production_runs, alarmas
│   └── turno_dia, canvas_oee
└── linea_2 ... linea_N (si hay múltiples líneas)
```

### 2.5.2 Cloud (PostgreSQL 16)

**Base de datos de servicios:**
```
mentor_cloud
├── identity (empresas, usuarios, roles, tokens)
├── config   (plantas, líneas, dispositivos, variables)
├── ingest   (raw_events, oee_snapshots, sync_log)
├── analytics (paradas, alarmas, compromisos, production_runs)
├── gateway  (audit_log, commands, device_registry)
├── integration (api_keys)
└── admin    (planta_databases — registro de BDs por planta)
```

**Bases de datos por planta:**
```
mentor_planta_14 (ejemplo: planta Art Atlas)
├── linea_11 (schema)
├── linea_12 (schema)
├── linea_13 (schema)
└── linea_14 (schema)
    ├── productos, turnos, variables
    ├── cat_programada, cat_no_programada
    ├── velocidad_nominal, linea_producto_vars
    ├── raw_events, oee_snapshots
    ├── production_runs, alarmas
    └── ...
```

---

## 2.6 Flujo de Datos Completo

```
1. DETECCIÓN (Tiempo real, cada frame ~66ms)
   Cámara RTSP → NVDEC → ROI → Señales → Fusión → FSM → Evento

2. PERSISTENCIA LOCAL (< 10ms)
   Evento → Resiliencia (dedup + buffer) → PostgreSQL local

3. SINCRONIZACIÓN A NUBE (cada 5s)
   Enviador → polling DB → batch 50 eventos → REST → Cloud Ingest

4. SINCRONIZACIÓN DESDE NUBE (tiempo real vía SSE)
   Cloud Config → Gateway SSE → Edge Gateway → PostgreSQL local
   (productos, turnos, categorías, variables, velocidades)

5. VISUALIZACIÓN
   PostgreSQL → Cloud Analytics → API → Cloud Frontend (ECharts)
   PostgreSQL local → Edge Gateway → UI Local / Tablet
```

---

## 2.7 Resiliencia y Tolerancia a Fallos

| Escenario | Comportamiento |
|-----------|----------------|
| **Sin internet** | Edge sigue detectando y buffereando. Sync se reanuda automáticamente |
| **Cámara desconectada** | Reconexión automática (3 intentos). Watchdog alerta si es prolongada |
| **BD local caída** | Resiliencia no puede aceptar eventos; detector recibe error 500 |
| **Cloud caído** | Enviador aplica retry exponencial. Eventos se acumulan localmente |
| **Config service caído** | Detector usa configuración cacheada en memoria |

---

## 2.8 Stack Tecnológico Consolidado

| Categoría | Tecnología | Versión |
|-----------|-----------|---------|
| Visión | OpenCV (CUDA + GStreamer) | 4.8.0 |
| Visión | NVIDIA VPI | 3.0 |
| Backend | Go | 1.23 / 1.24 |
| Backend | Gin (HTTP framework) | 1.10.0 |
| Backend | pgx/v5 (PostgreSQL) | 5.7.2 |
| Frontend | Vue 3 | 3.4+ |
| Frontend | Tailwind CSS | 3.3+ |
| Frontend | Apache ECharts | 5.4.3 |
| Frontend | Capacitor | 6.0 |
| Datos | PostgreSQL | 14 (edge) / 16 (cloud) |
| Infra | Docker / Docker Compose | 24+ / v2 |
| Infra | Nginx | 1.25 |
| Monitoreo | Prometheus | 2.53.0 |
| Monitoreo | Grafana | 10.4.3 |
| Auth | JWT (HMAC-SHA256) | v5 |
