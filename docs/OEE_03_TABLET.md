# Mentor Edge — Documentación Detallada de la Tablet (OEE/Producción)

## Visión General

La aplicación tablet es la interfaz principal del operador para monitoreo de OEE en tiempo real, justificación de paradas y gestión de producción. Opera en dos modos:

- **EDGE**: Conexión directa al Jetson en la misma red (latencia ~1-10ms)
- **CLOUD**: Conexión al servidor central para acceso remoto (latencia ~100-500ms)

---

## Stack Tecnológico

| Componente | Tecnología | Versión |
|------------|------------|---------|
| Framework | Vue 3 (Composition API + `<script setup>`) | 3.4.21 |
| State Management | Pinia | 2.1.7 |
| Routing | Vue Router 4 | 4.3 |
| Styling | Tailwind CSS + dark mode | 3.4 |
| Mobile | Capacitor 6 (iOS/Android) | 6.x |
| Real-time | Server-Sent Events (SSE) | — |
| Build | Vite | — |
| UI | Todo custom (sin librería externa) | — |

---

## Estructura de la Aplicación

```
mentor-tablet-app/
├── src/
│   ├── App.vue                    → Root component + connection manager
│   ├── main.ts                    → Entry point con Pinia + Router
│   ├── router/                    → 9 rutas con guards de autenticación
│   ├── views/
│   │   ├── LoginView.vue          → Autenticación edge/cloud
│   │   ├── DeviceSelectView.vue   → Descubrimiento de dispositivos edge
│   │   ├── DashboardView.vue      → Timeline de producción (vista principal)
│   │   ├── ProduccionView.vue     → Detector de visión en vivo
│   │   ├── StopsView.vue          → Gestión de paradas
│   │   ├── ConfigView.vue         → Configuración de detector
│   │   ├── StatusView.vue         → Diagnóstico del sistema
│   │   └── HistorialView.vue      → Historial de paradas
│   ├── components/
│   │   ├── TimelineCanvas.vue     → Rendering canvas de timeline
│   │   ├── AppHeader.vue          → Barra superior con contadores
│   │   ├── AppNav.vue             → Sidebar de navegación
│   │   ├── StopAssignmentModal.vue→ Selector de categoría jerárquica
│   │   ├── ProductAssignmentModal.vue → Selector de producto
│   │   ├── MermaRegistrationModal.vue → Registro de defectos
│   │   ├── CalendarModal.vue      → Navegación por fecha
│   │   ├── VelocidadNominalModal.vue → Edición de velocidades
│   │   └── StopForm.vue           → Formulario de creación de parada
│   ├── stores/                    → 11 stores Pinia
│   ├── services/
│   │   ├── api.ts                 → Cliente HTTP dual (edge/cloud)
│   │   └── sse.ts                 → Cliente SSE con auto-reconnect
│   ├── composables/
│   │   ├── useTimeline.ts         → Lógica de timeline (800+ líneas)
│   │   └── useThrottle.ts         → Rate limiting
│   └── types/
│       └── index.ts               → Interfaces TypeScript
```

---

## Rutas

| Ruta | Vista | Propósito | Auth |
|------|-------|-----------|------|
| `/` | — | Redirect a `/login` | No |
| `/login` | LoginView | Autenticación edge/cloud | No |
| `/device` | DeviceSelectView | Selección manual de dispositivo edge | No |
| `/dashboard` | DashboardView | **Vista principal**: Timeline de producción + paradas | Sí |
| `/produccion` | ProduccionView | Detector de visión en vivo + FSM local | Sí |
| `/stops` | StopsView | Lista y justificación de paradas | Sí |
| `/config` | ConfigView | Configuración de ROI, thresholds, calibración | Sí |
| `/status` | StatusView | Diagnóstico de servicios + buffer | Sí |
| `/historial` | HistorialView | Historial de paradas con búsqueda | Sí |

---

## Stores (Estado Global — Pinia)

### 1. Connection Store

Gestiona la conexión y autenticación entre la tablet y el backend (edge o cloud).

**Estado**:
- `mode`: EDGE | CLOUD | HYBRID | OFFLINE
- `edgeURL`, `cloudURL`: URLs base para API calls
- `authenticated`: Boolean de sesión activa
- `operator`: Datos del operador {nombre, rol, empresa_id}
- `health`: Estado del edge gateway
- `sseConnected`: Stream de eventos activo

**Métodos principales**:
- `connectToEdge(url)` → Establece conexión + bind SSE
- `connectToCloud(url)` → Conexión cloud + JWT
- `login(username, password)` → Login edge local
- `cloudLogin(username, password)` → Login cloud con JWT + refresh_token
- `probe()` → Health check periódico cada 10s

### 2. Machine Store

Estado del dispositivo edge y eventos recientes.

**Estado**:
- `status`: {device_id, cloud_connected, uptime}
- `bufferSummary`: {total, pending, synced, dead}
- `recentEvents`: Últimos 200 eventos cut_detected

**Métodos**:
- `fetchStatus(lineaId)` → Estado del dispositivo
- `fetchRecentEvents(limit, since, lineaId)` → Historial de detecciones
- `bindSSE()` → Subscripción a eventos en tiempo real

### 3. Stops Store

Gestión completa de paradas de producción — **crítico para OEE**.

**Estado**:
- `stops`: Array de todas las paradas
- `summary`: Estadísticas agregadas {by_type, total_downtime_ms}
- `openStops`: Filtro computed para paradas activas
- `unjustifiedStops`: Filtro computed para paradas sin categorizar
- `selectedStopId`: Parada seleccionada para edición

**Métodos**:
- `fetchStops(params)` → Cargar paradas con filtros fecha/línea
- `fetchSummary(hours)` → Estadísticas de paradas
- `createStop(data)` → Registrar parada manual
- `justifyStop(stopId, data)` → Categorizar + cerrar parada
- `closeStop(stopId)` → Marcar parada como terminada
- `deleteStop(stopId)` → Eliminar parada errónea
- `bindSSE()` → Escuchar stop.changed, stop_created, stop_closed en tiempo real

### 4. ProductionRuns Store

Gestión de corridas de producción con producto/SKU.

**Estado**:
- `runs`: Lista de corridas {run_id, sku, nombre, started_at, ended_at}

**Métodos**:
- `fetchRuns(params)` → Cargar corridas por rango de fechas
- `upsert(data)` → Crear o actualizar corrida
- `remove(runId)` → Eliminar corrida
- `bindSSE()` → Escuchar production_runs_updated

### 5. PlantasLineas Store

Jerarquía organizacional con control de acceso basado en roles.

**Estado**:
- `plantas`: Array de plantas {id, nombre}
- `lineas`: Array de líneas {id, nombre, planta_id}
- `selectedEmpresaId`, `selectedPlantaId`, `selectedLineaId`: Selecciones actuales (persistidas en localStorage)
- `rol`: Computed desde operator.rol (admin | superadmin | admin_planta | operator)

**Comportamiento**:
- Auto-selecciona primera planta al cambiar empresa
- Filtrado por rol: admins ven todo, operadores solo su empresa
- Resets en cascada al cambiar selección superior

### 6. Config Store

Configuración del detector de visión y parámetros OEE.

**Estado**:
- `config.mode`: "textil" (120s micro, 1800s snapshot) | "botellas" (210s micro, 300s snapshot)
- `config.roi`: {x, y, width, height}
- `config.thresholds`: {edge, color, flow, dy, beige, high, low}
- `config.fsm`: {n_frames, cooldown, exit_frames, max_wait_exit_frames}
- `config.oee`: {snapshot_interval_s, vel_unit}
- `version`: Contador de versión

**Métodos**:
- `fetchConfig()` → Cargar desde config-service
- `updateConfig(patch)` → Guardar cambios (parcial)
- `startCalibration()` → Iniciar captura de referencia
- `bindSSE()` → Escuchar config.updated

### 7. Catalog Store

Árbol de categorías de paradas y lista de productos.

**Estado**:
- `stopCategories`: Árbol jerárquico de razones de parada
  ```
  Programada
  ├── Mantenimiento Planificado
  ├── Cambio de Formato
  └── Limpieza
  No Programada
  ├── Mecánica
  ├── Eléctrica
  └── Falta Material
  Refrigerio
  Capacitación
  ```
- `products`: Lista de productos {codigo, nombre, sku}

**Métodos**:
- `fetchStopCategories(lineaId)` → Cargar árbol de categorías
- `fetchProducts(lineaId)` → Cargar catálogo de productos
- `bindSSE()` → Escuchar catalogs_synced | catalog.changed

### 8. Detector Store

**FSM local en JavaScript** que replica la lógica del detector Python para continuidad cuando ProduccionView no está activa.

**Estado**:
- `trackerState`: "producing" | "idle_wait" | "stop_open" | "offline"
- `stopStartTime`: Timestamp cuando inició la parada
- `idleSecs`: Segundos transcurridos desde inicio de parada
- `microparadasCount`, `paradasCount`: Contadores de eventos
- `isProduccionActive`: Si ProduccionView está montada

**Background Mode**:
Cuando el operador navega fuera de ProduccionView, el detector store:
1. Conecta a `/vision/ws/stream` (WebSocket)
2. Recibe frames JPEG binarios
3. Ejecuta análisis de pixel-diff en ROI (detección de presencia)
4. Mantiene la FSM activa con los mismos estados que el detector Python
5. Throttles a ~7 fps para ahorrar CPU
6. Persiste estado en localStorage para sobrevivir recargas

### 9. Turnos Store

Gestión de turnos de trabajo.

**Estado**:
- `turnos`: Array de turnos {hora_inicio, hora_fin, activo}
- `activeTurno`: Turno actual calculado por hora del día

**Métodos**:
- `fetchTurnos(params)` → Cargar turnos de la planta
- `shiftSince()` → ISO timestamp del inicio del turno actual (para filtros "desde")
- `shiftLabel`: Display del turno (ej: "Turno: A Desde: 06:00 Hasta: 14:00")

Soporta turnos nocturnos (ej: 22:00 → 06:00).

### 10. UI Store

Comunicación cross-component y estado de multi-selección.

**Triggers**:
- `goToNowTrigger`: Scroll del timeline al momento actual
- `calendarGoToTrigger`: Navegar a fecha específica
- `registerStopTrigger`: Abrir modal de registro de parada
- `multiSelectMode`: Activar selección múltiple de paradas
- `multiSelectedStops`: Set de stop_ids seleccionados

---

## Capa de API (`api.ts`)

### Dual Mode: EDGE vs CLOUD

```typescript
// La API detecta el modo y usa el endpoint correcto
function getStops(params) {
  if (mode === 'EDGE') {
    return GET('/edge/stops', params)      // → edge-gateway :8005
  } else {
    return GET('/api/stops', params)        // → cloud-gateway :8888
  }
}
```

### Endpoints por Categoría

| Categoría | Método | Edge | Cloud | Propósito |
|-----------|--------|------|-------|-----------|
| Auth | POST | `/edge/auth/login` | `/api/auth/login` | Login + token |
| Status | GET | `/edge/status` | `/api/oee/latest` | Estado del dispositivo |
| Config | GET/PUT | `/edge/config` | `/api/linea-config` | Configuración visión/OEE |
| Buffer | GET | `/edge/buffer/summary` | N/A | Estado de sincronización |
| Events | GET | `/edge/events/recent` | `/api/oee/snapshots` | Eventos de producción |
| Stops | GET/POST | `/edge/stops` | `/api/stops` | CRUD de paradas |
| Justify | POST | `/edge/stops/{id}/justify` | `/api/stops/{id}/justify` | Justificar parada |
| Catalogs | GET | `/edge/catalogs/{type}` | `/api/arbol-paradas` | Categorías + productos |
| Runs | GET/POST/DEL | `/edge/production-runs` | `/api/production-runs` | Corridas de producción |
| Shifts | GET | `/edge/catalogs/sync/turnos` | `/api/turnos` | Turnos de trabajo |
| Speed | GET/PUT | `/edge/catalogs/velocidad-nominal` | `/api/velocidad-nominal` | Velocidades nominales |

### Autenticación Cloud

```typescript
// JWT almacenado en sessionStorage
// Refresh automático cuando access_token expira
async function cloudRequest(method, url, data) {
  const token = sessionStorage.getItem('access_token')
  const response = await fetch(url, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  if (response.status === 401) {
    await refreshToken()  // POST /api/auth/refresh
    return retry(method, url, data)
  }
  return response
}
```

---

## SSE — Comunicación en Tiempo Real

### Conexión

```typescript
// Edge mode
const sse = new EventSource(`${edgeURL}/edge/stream`)

// Cloud mode (EventSource no soporta headers → token en URL)
const sse = new EventSource(`${cloudURL}/api/v1/cloud/stream?token=${jwt}`)
```

### Eventos Soportados

| Evento | Descripción | Acción en UI |
|--------|-------------|-------------|
| `event.created` | Nueva detección (CORTE) | Añadir marcador al timeline |
| `stop.changed` | Estado de parada cambió | Refetch de stops |
| `stop_created` | Parada creada | Añadir al timeline |
| `stop_closed` | Parada cerrada | Actualizar duración en timeline |
| `stop_deleted` | Parada eliminada | Remover del timeline |
| `stops_synced` | Sync batch completado | Refetch de stops |
| `catalogs_synced` | Catálogo actualizado | Recargar categorías/productos |
| `production_runs_updated` | Corrida modificada | Refetch de runs |
| `config.updated` | Configuración cambiada | Recargar config |

### Auto-Reconnect

Backoff exponencial: 1s → 2s → 4s → ... → 30s máximo. Reconecta automáticamente en caso de desconexión.

### Polling de Respaldo

Como redundancia al SSE:
- **Cloud**: Polling cada 20s
- **Edge**: Polling cada 30s

---

## Vistas del Operador

### DashboardView — Vista Principal

La vista que el operador usa el 90% del tiempo. Muestra un timeline de 3 carriles con producción, paradas y eventos del turno actual.

#### Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  AppHeader: [Empresa ▾] [Planta ▾] [Línea ▾]  Hoy: 847  ⚡12 │
├─────────────────────────────────────────────┬───────────────────┤
│                                             │  Estado Actual    │
│  Lane 0: Paradas Justificadas              │                   │
│  ██████████  ████  ████████████            │  🟢 Produciendo   │
│                                             │  Vel: 28.3 u/h    │
│  Lane 1: Paradas Sin Justificar            │                   │
│  ░░░░░░░░  ░░░░░░░░                       │  Último CORTE:     │
│                                             │  10:42:15          │
│  Lane 2: Production Runs                   │                   │
│  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓  ▒▒▒▒▒▒▒▒▒▒▒▒▒          │  ┌──────────────┐ │
│  SKU-001          SKU-002                  │  │ [Registrar    │ │
│                                             │  │  Parada]      │ │
│  | Event markers  ▼▼  ▼  ▼▼▼  ▼▼ ▼       │  │ [Merma]       │ │
│  ───────────────── NOW ─────────────       │  └──────────────┘ │
│                                             │                   │
│  [◀ Pan] [Hoy] [Zoom +/-] [Multi-select]  │  Turno: A         │
│                                             │  06:00 - 14:00    │
└─────────────────────────────────────────────┴───────────────────┘
```

#### Timeline Canvas (3 Carriles)

- **Lane 0 (Asignadas)**: Paradas justificadas + microparadas (bloques rojo/amarillo)
- **Lane 1 (Sin Asignar)**: Paradas sin justificar esperando categorización
- **Lane 2 (Producción)**: Corridas de producción por producto (colores por SKU)
- **Marcadores de eventos**: Detecciones CORTE del detector de visión
- **Línea "Ahora"**: Indicador del momento actual

#### Interacciones

| Acción | Resultado |
|--------|-----------|
| Click en parada sin asignar | Abre StopAssignmentModal para categorizar |
| Click en parada asignada | Opción de split-and-justify (dividir slot parcial) |
| Ctrl+Click paradas | Multi-select para justificación batch |
| Drag horizontal | Pan del timeline |
| Ctrl+Wheel / Pinch | Zoom in/out |
| Botón "Hoy" | Scroll al momento actual |
| Botón "Registrar Parada" | Abre formulario de parada manual |
| Botón "Merma" | Registra evento de defecto/desperdicio |

#### Split de Paradas

Funcionalidad avanzada para justificar porciones de una parada larga:
1. Operador click en el medio de una parada multi-slot
2. Solo el slot clickeado se categoriza
3. La parada original se divide en 3: [antes, justificada, después]
4. Los fragmentos antes/después quedan sin justificar

#### Datos Cargados

1. Al montar: fetch de 24h de stops, production runs y eventos del turno actual
2. SSE actualiza en tiempo real (nuevas paradas, eventos, cambios)
3. Polling de respaldo (20-30s) para redundancia
4. Timeline re-renders automáticamente al cambiar datos

---

### ProduccionView — Detector en Vivo

Vista de monitoreo del detector de visión con análisis en tiempo real.

#### Layout

```
┌───────────────────────┬──────────────────────────────────────────┐
│  Panel Izquierdo (30%) │  Panel Derecho (70%) — Stream en Vivo   │
│                        │                                          │
│  Detecciones:          │  ┌──────────────────────────────────────┐│
│  HOY: 847              │  │                                      ││
│  SESIÓN: 152           │  │    ┌──────────────┐                  ││
│                        │  │    │  ROI (amarillo)│                 ││
│  Estado FSM:           │  │    │  Beige: 42%   │                 ││
│  🟢 DETECTING          │  │    └──────────────┘                  ││
│  Beige: 42%            │  │                                      ││
│                        │  │    ┌──────────────┐                  ││
│  Última detección:     │  │    │ Presencia ROI │  (verde)        ││
│  10:42:15              │  │    └──────────────┘                  ││
│                        │  │                                      ││
│  ROI Config:           │  └──────────────────────────────────────┘│
│  x: 120  y: 80         │                                          │
│  w: 400  h: 300        │  ┌─ Flash de detección (cut_detected) ──┐│
│                        │  └──────────────────────────────────────┘│
│  Histéresis:           │                                          │
│  high: 0.7  low: 0.3  │                                          │
│                        │                                          │
│  Logs de detección:    │                                          │
│  10:42:15 CORTE 0.82  │                                          │
│  10:41:03 IDLE  0.12  │                                          │
│  10:40:55 DETECT 0.75 │                                          │
│  ...                   │                                          │
└───────────────────────┴──────────────────────────────────────────┘
```

#### Procesamiento Local en JavaScript

La ProduccionView ejecuta una **mini-FSM local** que replica la lógica del detector Python:

**Análisis de Beige**:
- Lee región ROI del canvas
- Convierte RGB → HSV
- Verifica: H ∈ [15°-35°], S ∈ [30-130], V > 120
- Si beigePct > 0.35 → prenda detectable

**Detección de Movimiento**:
- Buffer lento (últimos ~30 frames)
- Pixel-diff entre frame más antiguo y actual
- Si diffPixels/totalPixels > motion_threshold → movimiento detectado

**Stop Tracker Local**:
```
producing → sin detección → idle_wait (microparada)
idle_wait → timeout → stop_open (parada)
stop_open → detección → producing
```

**Contadores**:
- Microparadas: incrementa cuando producing → idle_wait
- Paradas: incrementa cuando idle_wait → stop_open
- Se resetean al recargar la página

---

### StopsView — Gestión de Paradas

Lista completa de paradas con capacidad de justificación.

#### Layout

```
┌──────────────────────────────────────────────────────────────┐
│  [Todas] [Abiertas] [Sin Justificar]                         │
├──────────────────────────────────────────────────────────────┤
│  Total: 23  │  Tiempo: 4h 32m  │  Sin justificar: 5  │      │
├──────────────────────────────────────────────────────────────┤
│  🤖 MICROPARADA  10:30-10:32 (2m)     Sin justificar  [Just]│
│  👷 PROGRAMADA   09:00-09:30 (30m)    Mantenimiento    ✓     │
│  🤖 PARADA       08:15-08:45 (30m)    Cambio formato   ✓     │
│  ☁️  MECÁNICA     07:50-08:10 (20m)    Falla motor      ✓     │
│  ...                                                          │
└──────────────────────────────────────────────────────────────┘
```

**Iconos de fuente**:
- 🤖 detector — Parada detectada automáticamente
- 👷 operator — Registrada por operador
- ☁️ cloud — Creada desde cloud
- ⚙️ system — Generada por el sistema

**Acciones**:
- Click "Justificar" → Modal de selección de categoría
- Click "Cerrar" → Marca parada como terminada
- Filtros por estado para enfoque rápido
- Auto-refresh por SSE + polling cada 10s

---

### ConfigView — Configuración del Detector

#### Secciones Disponibles

**Modo de Operación**:
```
[Textil]    → microparada: 120s, snapshot: 1800s
[Botellas]  → microparada: 210s, snapshot: 300s
```

**ROI (Región de Interés)**:
- Inputs: x, y, width, height
- Preview en vivo sobre el stream de cámara

**Thresholds (Umbrales)**:
- high/low: Histéresis de score de detección
- flow: Gate de flujo óptico
- dy: Velocidad vertical mínima
- edge/color/beige: Pesos del fusion engine

**FSM**:
- n_frames: Frames consecutivos para confirmar (3)
- cooldown: Frames anti-rebote post-detección (8)
- exit_frames: Frames bajo umbral para salir (5)
- max_wait_exit_frames: Timeout WAIT_EXIT (750)

**Calibración**:
1. Banner: "Retire la prenda del ROI antes de calibrar"
2. Captura 30 frames de referencia
3. Barra de progreso
4. Notificación de éxito al completar

**Restricción Cloud**: En modo CLOUD, toda la configuración de visión está deshabilitada con aviso "Disponible solo en modo EDGE".

---

### HistorialView — Historial de Paradas

Tabla auditable con filtros y ordenamiento:

```
┌───────────────────────────────────────────────────────────────┐
│  Total: 156  │  Microparadas: 89  │  Paradas: 67  │  12h 40m │
├───────────────────────────────────────────────────────────────┤
│  [Todas] [Microparadas] [Paradas]                             │
├───────┬───────┬──────────┬──────────┬────────┬───────┬───────┤
│ Fecha │ Hora  │ Tipo     │ Duración │ Fuente │ Estado│ Sync  │
├───────┼───────┼──────────┼──────────┼────────┼───────┼───────┤
│ 03/24 │ 10:30 │ Micro    │ 2m       │ 🤖     │ ✓     │ ✓    │
│ 03/24 │ 09:00 │ Program. │ 30m      │ 👷     │ ✓     │ ✓    │
│ 03/24 │ 08:15 │ Parada   │ 30m      │ 🤖     │ ⏱     │ ✓    │
└───────┴───────┴──────────┴──────────┴────────┴───────┴───────┘
```

- ⏱ indica paradas aún abiertas
- Columna Sync muestra si fue sincronizado al cloud

---

### StatusView — Diagnóstico del Sistema

#### Modo Edge

```
┌──────────────────────────────────────────────┐
│  Conexión: EDGE  │  SSE: ✓  │  Op: Juan     │
├──────────────────────────────────────────────┤
│  Device: jetson-orin-01  │  Uptime: 72h      │
│  Cloud: ✓ Conectado                           │
├──────────────────────────────────────────────┤
│  Servicios:                                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Detector │ │Resiliencia│ │ Enviador │     │
│  │  :8001   │ │  :8002   │ │  :8003   │     │
│  │  🟢 OK   │ │  🟢 OK   │ │  🟢 OK   │     │
│  └──────────┘ └──────────┘ └──────────┘     │
│  ┌──────────┐ ┌──────────┐                   │
│  │  Config  │ │ Gateway  │                   │
│  │  :8004   │ │  :8005   │                   │
│  │  🟢 OK   │ │  🟢 OK   │                   │
│  └──────────┘ └──────────┘                   │
├──────────────────────────────────────────────┤
│  Buffer: Total: 15,234 │ Pending: 42        │
│          Synced: 15,180 │ Dead: 12           │
└──────────────────────────────────────────────┘
```

#### Modo Cloud

A las limitaciones mostradas: "☁ Modo Cloud – diagnóstico de hardware no disponible" con información de conexión y SSE.

---

### LoginView — Autenticación

#### Flujo de Login

1. Intenta restaurar sesión cloud desde localStorage (JWT)
2. Si protocolo es HTTPS → fuerza modo cloud (sin probe edge)
3. Si HTTP → probe al Jetson (mismo origen + :8005)
4. Si edge alcanzable → formulario de login edge
5. Si no → formulario de login cloud
6. Post-login → cargar plantas/líneas → bind SSE → navegar a dashboard

#### Toggle

```
[Edge Local]  ← Conexión directa al Jetson
[Cloud]       ← Conexión al servidor central
```

---

### DeviceSelectView — Descubrimiento

Pantalla para conexión manual a dispositivos edge:

- URL del servidor cloud + botón conectar
- Lista de dispositivos descubiertos (con badge Online/Offline)
- Input manual para añadir URL de dispositivo
- Prueba automática del mismo origen (nginx proxy local)
- Prueba de puerto 8005 (acceso directo)
- Persistencia en localStorage de dispositivos guardados

---

## Componentes Reutilizables

| Componente | Propósito |
|------------|-----------|
| **TimelineCanvas** | Rendering canvas de alta performance con touch gestures |
| **AppHeader** | Barra superior: empresa/planta/línea, contadores, multi-select |
| **AppNav** | Sidebar: navegación, dark mode, logout |
| **StopAssignmentModal** | Selector de árbol jerárquico de categorías para justificación |
| **ProductAssignmentModal** | Selector de producto (SKU, nombre) para asignar corrida |
| **MermaRegistrationModal** | Formulario para registrar defecto/merma |
| **CalendarModal** | Picker de fecha para navegar el timeline |
| **VelocidadNominalModal** | Editor de velocidades nominales por producto |
| **StopForm** | Formulario alternativo para creación de parada |
| **StatusIndicator** | Mini gauge: total | pending | synced | dead |
| **SvgIcon** | Librería de 50+ iconos vectoriales |

---

## Composables

### useTimeline (800+ líneas)

El composable más complejo de la aplicación. Gestiona toda la lógica del timeline:

**Rendering**:
- Divide el timespan visible en slots (5min si zoom cercano, 30min si lejano)
- 3 carriles: paradas asignadas, sin asignar, production runs
- Marcadores de eventos CORTE
- Bloques sintéticos generados cuando no hay datos (detección de gaps)

**Interacción**:
- Hit-testing para clicks en stops y production blocks
- Multi-select con Ctrl+Click
- Pan horizontal (drag / botones)
- Zoom (Ctrl+Wheel / pinch gesture)
- "Live follow": auto-scroll al "ahora" (desactiva si usuario hace pan manual)

**Colores**:
- Cada SKU de producto recibe un color único de una paleta
- Paradas: rojo (parada mayor), amarillo (microparada), gris (sin asignar)

---

## Tipos TypeScript

```typescript
interface Stop {
  stop_id: string
  stop_type: 'MICROPARADA' | 'PARADA_NO_ASIGNADA' | 'PROGRAMADA' |
             'MECANICA' | 'ELECTRICA' | 'CAMBIO_FORMATO' |
             'FALTA_MATERIAL' | 'CALIDAD' | 'OTRA'
  started_at: string          // ISO 8601
  ended_at: string | null     // null = parada abierta
  duration_ms: number | null
  justified: boolean
  reason: string | null
  categoria_id: number | null
  justified_by: string | null
  source: 'detector' | 'operator' | 'cloud' | 'system'
  synced: boolean
}

interface ProductionRun {
  run_id: string
  sku: string | null
  nombre: string | null
  started_at: string
  ended_at: string | null
  device_id: string
  synced: boolean
}

interface EdgeEvent {
  event_id: string
  event_type: 'cut_detected'
  timestamp: string
  payload: {}
  synced: boolean
  dead: boolean
}

interface CategoryTreeNode {
  id: number
  nombre: string
  tipo_parada?: string
  children?: CategoryTreeNode[]
}
```

---

## Comparación Edge vs Cloud Mode

| Funcionalidad | EDGE | CLOUD |
|---------------|------|-------|
| Latencia | ~1-10ms | ~100-500ms |
| Configuración visión | Completa (ROI, calibración, thresholds) | Solo lectura |
| Detector en vivo | Stream MJPEG desde Jetson | No disponible |
| Buffer status | Visible en tiempo real | No visible |
| Fuente de datos | BD local del Jetson | BD central cloud |
| Multi-sitio | Un solo dispositivo | Agregación multi-sitio |
| Soporte offline | Limitado (caché local) | Ninguno (requiere conexión) |
| Autenticación | Token local | JWT con refresh |
| SSE endpoint | `/edge/stream` | `/api/v1/cloud/stream?token=` |

---

## Jornada del Operador

### Inicio de Turno

1. Abre tablet → App.vue restaura sesión de localStorage o muestra login
2. Selecciona empresa/planta/línea (o carga de localStorage)
3. Dashboard se carga con timeline del turno actual

### Durante el Turno

1. **Monitoreo continuo**: Timeline muestra producción y paradas en tiempo real
2. **Captura de producción**: Navega a ProduccionView para ver detector en vivo
3. **Justificación de paradas**: Click en parada sin justificar → selecciona categoría → guarda
4. **Registro de corrida**: Asigna producto/SKU cuando cambia la producción
5. **Registro de merma**: Botón "Merma" para reportar defectos
6. **Velocidad nominal**: Edita velocidades si cambia el producto

### Diagnóstico (si hay problemas)

1. StatusView → verifica estado de cada servicio (detector, config, gateway, etc.)
2. Buffer → comprueba pendientes vs sincronizados vs muertos
3. ConfigView → ajusta thresholds si hay falsos positivos/negativos
4. Calibración → recaptura referencia de color si cambió la iluminación

### Fin de Turno

1. HistorialView → auditoría de todas las paradas del turno
2. Dashboard → captura de pantalla del resumen diario
3. Logout

---

## Performance

| Técnica | Descripción |
|---------|-------------|
| Canvas rendering | Timeline basado en Canvas (no DOM) para millones de eventos |
| SSE real-time | Evita overhead de polling con Server-Sent Events |
| Background detector | Continúa rastreando paradas cuando ProduccionView no está activa |
| Throttled polling | 20s (cloud) / 30s (edge) como respaldo redundante |
| Slot-based LOD | Renderiza solo slots visibles (nivel de detalle dinámico) |
| localStorage | Persiste estado para sobrevivir recargas de página |
| Touch gestures | Pinch-zoom y drag-pan optimizados para tablets |
| Lazy imports | Rutas con dynamic import para carga inicial rápida |
