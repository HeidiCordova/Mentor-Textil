# ENERGY 04 — Servicio Cloud: energy-ingest

## 1. Descripcion

`energy-ingest` es el microservicio Go responsable de recibir, validar y persistir los snapshots eléctricos enviados por los dispositivos edge. Implementa arquitectura hexagonal (puertos y adaptadores) y soporta multitenancy con doble escritura: schema global `energy.*` y schema especifico de planta.

---

## 2. Arquitectura Hexagonal

```mermaid
graph TD
    subgraph Adaptadores_Entrada["Adaptadores de Entrada"]
        HTTP_EDGE["HTTP Edge\nPOST /api/v1/energy/snapshots\nAPIKey Auth"]
        HTTP_DASH["HTTP Dashboard\nGET/POST/PUT/DELETE /api/energy/*\nJWT Auth"]
    end

    subgraph Aplicacion["Capa de Aplicacion"]
        SVC["EnergyService\nProcessBatch()\nGetSnapshots()\nGetMeters()\nCRUD Meters"]
    end

    subgraph Puertos["Puertos (Interfaces)"]
        PORT_REPO["EnergyRepository\n(interface)"]
        PORT_SCOPE["ScopeResolver\n(interface)"]
    end

    subgraph Adaptadores_Salida["Adaptadores de Salida"]
        REPO_GLOBAL["EnergyRepo\nenergy.* (global)"]
        REPO_PLANTA["PlantaEnergyRepo\nplanta_N.* (Option B)"]
        SCOPE_REPO["ScopeResolver\ngateway.device_registry"]
    end

    subgraph DB["PostgreSQL Cloud"]
        DB_GLOBAL["Schema energy.*"]
        DB_PLANTA["Schema planta_N.*"]
        DB_GW["Schema gateway.*"]
    end

    HTTP_EDGE --> SVC
    HTTP_DASH --> SVC
    SVC --> PORT_REPO
    SVC --> PORT_SCOPE
    PORT_REPO --> REPO_GLOBAL
    PORT_SCOPE --> SCOPE_REPO
    SVC --> REPO_PLANTA
    REPO_GLOBAL --> DB_GLOBAL
    REPO_PLANTA --> DB_PLANTA
    SCOPE_REPO --> DB_GW
```

---

## 3. Endpoints Registrados

```mermaid
graph LR
    subgraph EdgeGroup["/api/v1/energy — APIKey Auth"]
        E1["POST /snapshots\nIngesta de lote desde edge"]
    end

    subgraph DashGroup["/api/energy — JWT Auth"]
        D1["GET /snapshots\nConsulta paginada con filtros"]
        D2["GET /meters\nListar medidores de la empresa"]
        D3["GET /meters/:id\nObtener medidor por ID"]
        D4["POST /meters\nCrear medidor"]
        D5["PUT /meters/:id\nActualizar medidor"]
        D6["DELETE /meters/:id\nEliminar medidor"]
    end
```

| Ruta | Metodo | Auth | Descripcion |
|---|---|---|---|
| `/api/v1/energy/snapshots` | POST | API Key | Recibir lote de snapshots desde edge |
| `/api/energy/snapshots` | GET | JWT | Consultar snapshots con filtros y paginacion |
| `/api/energy/meters` | GET | JWT | Listar medidores de la empresa del token |
| `/api/energy/meters/:id` | GET | JWT | Obtener medidor por ID |
| `/api/energy/meters` | POST | JWT | Crear medidor |
| `/api/energy/meters/:id` | PUT | JWT | Actualizar nombre, ubicacion o estado |
| `/api/energy/meters/:id` | DELETE | JWT | Eliminar medidor |

---

## 4. Flujo de Procesamiento: ProcessBatch

```mermaid
flowchart TD
    START(["POST /api/v1/energy/snapshots\ndeviceID desde X-Device-ID"])
    SCOPE["ScopeResolver.ResolveByDevice()\nSELECT empresa_id, planta_id, linea_id\nFROM gateway.device_registry"]
    LAST_SEEN["UpdateLastSeen()\n(goroutine async)\nUPDATE last_seen_at = NOW()"]
    CAN_PLANTA{scope != nil\nY planta_id != nil\nY linea_id != nil?}
    
    LOOP_START["Para cada EnergyRecord en el lote"]
    EXTRACT["ExtractFields(head, data)\nNormalizar aliases Modbus → campos float64"]
    BUILD_SNAP["Construir EnergySnapshot\ncon todos los campos electricos"]
    
    WRITE_PLANTA["PlantaEnergyRepo.UpsertSnapshot()\nEscritura en planta_N.*\n(Option B — multitenancy)"]
    WRITE_GLOBAL["EnergyRepo.UpsertSnapshot()\nEscritura en energy.*\n(backward compat dashboard)"]
    UPSERT_METER["UpsertMeter()\nen global y/o planta"]
    
    LOG_SYNC["LogSync()\nRegistrar en device_sync_log"]
    METRICS["Actualizar Prometheus\nsnapshots_total + batch_duration"]
    RESP(["200 OK\n{'received': N}"])

    START --> SCOPE
    SCOPE --> LAST_SEEN
    SCOPE --> CAN_PLANTA
    CAN_PLANTA -->|Si| LOOP_START
    CAN_PLANTA -->|No, solo global| LOOP_START
    LOOP_START --> EXTRACT
    EXTRACT --> BUILD_SNAP
    BUILD_SNAP --> WRITE_PLANTA
    WRITE_PLANTA --> WRITE_GLOBAL
    WRITE_GLOBAL --> UPSERT_METER
    UPSERT_METER --> LOOP_START
    LOOP_START -->|Fin del lote| LOG_SYNC
    LOG_SYNC --> METRICS
    METRICS --> RESP
```

---

## 5. Resolucion de Scope (Multitenancy)

```mermaid
sequenceDiagram
    participant INGEST as EnergyService
    participant SCOPE as ScopeResolver
    participant DB as gateway.device_registry

    INGEST->>SCOPE: ResolveByDevice(ctx, "rpi-energy-01")
    SCOPE->>DB: SELECT empresa_id, planta_id, linea_id\nWHERE device_id='rpi-energy-01' AND active=true
    DB-->>SCOPE: empresa_id=5, planta_id=12, linea_id=3
    SCOPE-->>INGEST: DeviceScope{EmpresaID:5, PlantaID:12, LineaID:3}
    
    Note over INGEST: Escritura dual:\n1. energy.snapshots (global)\n2. planta_12.energy_readings (linea 3)

    INGEST->>SCOPE: UpdateLastSeen("rpi-energy-01") [goroutine]
    SCOPE->>DB: UPDATE last_seen_at = NOW(), active = true
```

---

## 6. Doble Escritura (Option B)

```mermaid
graph LR
    BATCH["Lote recibido\ndevice_id = rpi-energy-01\nscope resuelto"]

    subgraph Global["Schema energy.* (siempre)"]
        G1["energy.snapshots\nBackward compat\nDashboard global"]
        G2["energy.meters\nRegistro de medidores"]
        G3["energy.device_sync_log\nAuditoria de sincronizacion"]
    end

    subgraph Planta["Schema planta_N.* (cuando aplica)"]
        P1["planta_12.energy_readings\nDatos por linea de produccion"]
        P2["planta_12.meters\nMedidores de la planta"]
        P3["planta_12.device_sync_log\nLog de sync especifico"]
    end

    BATCH --> G1
    BATCH --> G2
    BATCH --> G3
    BATCH -->|"canPlanta = true\n(scope tiene PlantaID + LineaID)"| P1
    BATCH --> P2
    BATCH --> P3
```

Si `canPlanta` es `false` (device no registrado o sin scope completo), solo se escribe en el schema global y se registra un error en el log sin interrumpir la ingesta.

---

## 7. Seguridad en Consultas del Dashboard

```mermaid
flowchart LR
    REQ["GET /api/energy/snapshots\n?device_id=X&planta_id=12"]
    JWT_MW["JWT Middleware\nExtrae claims del token"]
    HANDLER["QuerySnapshots Handler"]
    FORCE["empresaID = claims[empresa_id]\n(NUNCA desde query string)"]
    FILTER["SnapshotFilter{\n  DeviceID: query.device_id,\n  PlantaID: query.planta_id,\n  EmpresaID: <forzado desde JWT>\n}"]
    REPO["EnergyRepository.GetSnapshots()\nWHERE empresa_id = $N AND ..."]

    REQ --> JWT_MW
    JWT_MW --> HANDLER
    HANDLER --> FORCE
    FORCE --> FILTER
    FILTER --> REPO
```

El `empresa_id` se extrae exclusivamente del JWT para garantizar que cada usuario solo accede a los datos de su empresa. Un usuario no puede consultar datos de otra empresa manipulando el query string.

---

## 8. Metricas Prometheus

| Metrica | Tipo | Labels | Descripcion |
|---|---|---|---|
| `energy_snapshots_total` | Counter | `device_id` | Total de snapshots procesados por dispositivo |
| `energy_batch_duration_seconds` | Histogram | `device_id` | Duracion de procesamiento de cada lote |
| `energy_errors_total` | Counter | `reason` | Errores clasificados por etapa (`read_body`, `parse`, `process`) |

---

## 9. Estructura de Directorios del Servicio

```
energy-ingest/
├── cmd/
│   └── main.go                    # Bootstrap, DI manual
├── internal/
│   ├── adapters/
│   │   ├── handler/
│   │   │   └── energy_handler.go  # HTTP handlers (Gin)
│   │   └── repository/
│   │       ├── db.go              # Pool de conexiones
│   │       ├── energy_repo.go     # Impl. EnergyRepository global
│   │       ├── planta_energy_repo.go  # Impl. repo planta (Option B)
│   │       └── scope_resolver.go  # Resolucion de scope por device
│   ├── application/
│   │   └── energy_service.go      # Logica de negocio
│   ├── domain/
│   │   ├── models.go              # Entidades de dominio
│   │   └── extract.go             # Normalizacion aliases Modbus
│   ├── metrics/
│   │   └── metrics.go             # Definicion de metricas Prometheus
│   └── ports/
│       ├── repositories.go        # Interfaces EnergyRepository, ScopeResolver
│       └── middleware.go          # APIKeyAuth middleware
└── Dockerfile
```
