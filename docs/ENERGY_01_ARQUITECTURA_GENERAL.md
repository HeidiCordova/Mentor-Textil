# ENERGY 01 — Arquitectura General del Subsistema de Energía

## 1. Descripcion

El subsistema de energía mide, almacena y transmite datos eléctricos desde medidores físicos instalados en planta hasta la plataforma cloud, donde son visualizados y analizados. Está diseñado para operar con resiliencia ante cortes de conectividad, garantizando que ninguna medición se pierda.

---

## 2. Componentes del Sistema

```mermaid
graph LR
    subgraph Hardware["Hardware en Planta"]
        MC60["Medidor MEATROL MC60\nModbus RTU / RS-485"]
    end

    subgraph RPi["Raspberry Pi — Edge"]
        NR["Node-RED\nFlujo mc60_energy.json"]
        PG_LOCAL["PostgreSQL Local\nenergy.snapshots\nenergy.config"]
        SENDER["energy-sender\nGo — Puerto 8086"]
        UI_LOCAL["UI Local\nDashboard Web\n:8086"]
    end

    subgraph Internet["Transporte"]
        HTTPS["HTTPS\nPOST /api/v1/energy/snapshots\nX-Device-ID + X-API-Key"]
    end

    subgraph Cloud["Cloud — mentormonitor-ai.com"]
        GW["cloud-gateway\nProxy / Auth"]
        INGEST["energy-ingest\nGo — Puerto 8087"]
        PG_CLOUD["PostgreSQL Cloud\nenergy.* + planta_N.*"]
        FRONTEND["Frontend Vue 3\nMonitor / Medidores\nAnalisis Tarifario"]
    end

    MC60 -->|"Modbus RTU\nRS-485 USB"| NR
    NR -->|"INSERT snapshots"| PG_LOCAL
    PG_LOCAL -->|"SELECT not synced\nbatch"| SENDER
    SENDER --> HTTPS
    SENDER <-->|"GET/PUT /api/config\nGET /api/stats"| UI_LOCAL
    HTTPS --> GW
    GW --> INGEST
    INGEST -->|"UpsertSnapshot\nUpsertMeter\nLogSync"| PG_CLOUD
    PG_CLOUD --> FRONTEND
```

---

## 3. Stack Tecnologico por Capa

| Capa | Tecnologia | Rol |
|---|---|---|
| Hardware | MEATROL MC60 | Medicion electrica trifasica via Modbus RTU |
| Adquisicion | Node-RED 3.1.9 | Lectura Modbus RTU, persistencia local |
| Buffer Edge | PostgreSQL 15 (Alpine) | Cola local de snapshots con flag `synced` |
| Envio Edge | Go (energy-sender) | Envio batch al cloud, catchup ante desconexion |
| UI Edge | HTML/JS embebido en Go | Configuracion y monitoreo local en RPi |
| Ingesta Cloud | Go + Gin (energy-ingest) | Recepcion, validacion, persistencia multitenancy |
| Base Cloud | PostgreSQL (cloud) | Schema `energy.*` global + schemas por planta |
| Frontend Cloud | Vue 3 + Vite | Visualizacion, analisis tarifario, gestion de medidores |

---

## 4. Flujo de Datos de Extremo a Extremo

```mermaid
sequenceDiagram
    participant MC60 as Medidor MC60
    participant NR as Node-RED
    participant DB_E as PG Local
    participant SEND as energy-sender
    participant INGEST as energy-ingest
    participant DB_C as PG Cloud
    participant UI as Frontend Cloud

    MC60->>NR: Registros Modbus RTU (intervalo configurable, default 5min)
    NR->>DB_E: INSERT energy.snapshots (synced=false)

    loop Cada send_interval_s (default 30s)
        SEND->>DB_E: SELECT snapshots WHERE NOT synced LIMIT batch_size
        DB_E-->>SEND: Lote de snapshots
        SEND->>INGEST: POST /api/v1/energy/snapshots\n[X-Device-ID, X-API-Key]
        INGEST->>DB_C: UpsertSnapshot + UpsertMeter + LogSync
        DB_C-->>INGEST: OK
        INGEST-->>SEND: {"received": N}
        SEND->>DB_E: UPDATE snapshots SET synced=true WHERE id IN (...)
        Note over SEND: Si quedan pendientes,\nrepite sin esperar el ticker\n(backlog catchup)
    end

    UI->>INGEST: GET /api/energy/snapshots [JWT]
    INGEST->>DB_C: SELECT con filtro empresa_id (desde JWT)
    DB_C-->>INGEST: Snapshots
    INGEST-->>UI: JSON response
```

---

## 5. Modelo de Despliegue

```mermaid
graph TD
    subgraph RPi["Raspberry Pi (ARM64)"]
        DC_RPi["docker compose\n-f docker-compose.raspberry-energy.yml"]
        DC_RPi --> C1["rpi-energy-postgres\nPostgreSQL 15"]
        DC_RPi --> C2["rpi-node-red\nNode-RED 3.1.9\nDialout group (RS-485)"]
        DC_RPi --> C3["rpi-energy-sender\nGo binary ARM64\n:8086"]
        C1 -.->|healthcheck pg_isready| C2
        C1 -.->|healthcheck pg_isready| C3
    end

    subgraph Cloud["Cloud Server"]
        DC_Cloud["docker compose (cloud)"]
        DC_Cloud --> I1["energy-ingest\nGo + Gin\n:8087"]
        DC_Cloud --> I2["cloud-gateway\n:443"]
        DC_Cloud --> I3["PostgreSQL Cloud\n:5432"]
        I2 --> I1
        I1 --> I3
    end

    C3 -->|"HTTPS :443"| I2
```

---

## 6. Variables de Configuracion Edge

La configuracion operacional del edge se almacena en `energy.config` (PostgreSQL local) y se recarga en caliente sin reiniciar contenedores.

| Clave | Descripcion | Valor por Defecto |
|---|---|---|
| `device_id` | Identificador unico del dispositivo edge | *(vacio — obligatorio)* |
| `meter_id_1` | ID del medidor Modbus | *(vacio — obligatorio)* |
| `meter_unit_id` | Direccion Modbus del medidor | `1` |
| `cloud_url` | URL base del cloud | `https://mentormonitor-ai.com` |
| `energy_api_key` | API key de autenticacion al cloud | *(vacio — obligatorio)* |
| `send_interval_s` | Intervalo de envio en segundos | `30` |
| `batch_size` | Tamano maximo del lote por envio | `50` |
| `config_reload_s` | Intervalo de recarga de config desde DB | `60` |

---

## 7. Seguridad

```mermaid
graph LR
    subgraph EdgeAuth["Autenticacion Edge → Cloud"]
        AK["X-API-Key\n(Header HTTP)"]
        DID["X-Device-ID\n(Header HTTP)"]
    end

    subgraph CloudAuth["Autenticacion Dashboard → Cloud"]
        JWT["JWT Bearer Token\n(empresa_id en claims)"]
    end

    subgraph Enforcement["Aplicacion"]
        APK["Middleware APIKeyAuth\nPOST /api/v1/energy/snapshots"]
        JWTM["Middleware JWTAuth\nGET /api/energy/*"]
        SCOPE["empresa_id forzado\ndesde JWT — no desde query string"]
    end

    AK --> APK
    DID --> APK
    JWT --> JWTM
    JWTM --> SCOPE
```

- El `empresa_id` nunca se acepta desde el query string en endpoints del dashboard; siempre se extrae del JWT para garantizar aislamiento de datos entre empresas.
- La API key se almacena en `energy.config` en el edge y se transmite via HTTPS.
