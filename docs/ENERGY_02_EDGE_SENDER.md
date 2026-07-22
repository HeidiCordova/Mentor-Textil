# ENERGY 02 — Servicio Edge: energy-sender

## 1. Descripcion

`energy-sender` es un microservicio Go desplegado en la Raspberry Pi. Su responsabilidad es leer snapshots eléctricos almacenados localmente por Node-RED, enviarlos al cloud en lotes y marcarlos como sincronizados. También expone una UI web local para configuracion y monitoreo en tiempo real.

---

## 2. Arquitectura Interna

```mermaid
graph TD
    subgraph ContainerRPi["Contenedor rpi-energy-sender"]
        MAIN["main.go\nLoop principal"]
        WEB["web.go\nHTTP handlers"]
        CFG["runtimeConfig\nSync.RWMutex"]
        
        MAIN -->|"lee configuracion"| CFG
        MAIN -->|"goroutine"| RELOAD["Config Reload\ncada config_reload_s"]
        MAIN -->|"goroutine"| HTTP_SRV["HTTP Server\n:8086"]
        MAIN -->|"ticker"| SEND_LOOP["Send Loop\ncada send_interval_s"]
        
        HTTP_SRV --> WEB
        WEB -->|"GET /api/config"| CFG
        WEB -->|"PUT /api/config"| CFG
        
        RELOAD -->|"UPDATE CFG"| CFG
    end

    subgraph DB["PostgreSQL Local\nmentor_energy"]
        ECFG["energy.config\n(clave-valor)"]
        ESNAP["energy.snapshots\n(synced flag)"]
    end

    subgraph Cloud["Cloud"]
        ENDPOINT["POST /api/v1/energy/snapshots"]
    end

    RELOAD -->|"SELECT key,value"| ECFG
    SEND_LOOP -->|"SELECT WHERE NOT synced\nLIMIT batch_size"| ESNAP
    SEND_LOOP -->|"HTTPS + X-Device-ID\n+ X-API-Key"| ENDPOINT
    SEND_LOOP -->|"UPDATE synced=true"| ESNAP
    WEB -->|"SELECT"| ECFG
    WEB -->|"SELECT"| ESNAP
```

---

## 3. Adquisicion de Datos: Node-RED + Modbus RTU

```mermaid
graph LR
    MC60["Medidor MC60\nDireccion Modbus: 1\nRS-485 / RTU"]
    USB["Adaptador USB-Serial\n/dev/mc60-mbus\n(udev: 99-mc60-serial.rules)"]
    NR["Node-RED Flow\nmc60_energy.json"]
    DB["PostgreSQL\nenergy.snapshots\nsynced=false"]

    MC60 -->|"Modbus RTU\n9600 bps"| USB
    USB -->|"Device mount\ndocker --device"| NR
    NR -->|"Parseo de registros\nINSERT batch"| DB

    note1["Intervalo configurable\nvia METER_POLL_INTERVAL_MS\ndefault: 300000ms (5 min)"]
    NR -.-> note1
```

Node-RED accede al dispositivo serie via el grupo `dialout` (GID 20), montado como dispositivo Docker con nombre estable definido por una regla udev.

---

## 4. Logica del Loop de Envio

```mermaid
flowchart TD
    START([Inicio del ticker])
    CHECK_KEY{energy_api_key\nconfigurado?}
    WAIT["Esperar 10s\ny reintentar"]
    INTERVAL_CHANGE{send_interval_s\ncambio?}
    NEW_TICK["Crear nuevo ticker\ncon nuevo intervalo"]
    TICK_WAIT["Esperar siguiente tick"]
    SEND_BATCH["sendBatch()\nSELECT snapshots WHERE NOT synced\nLIMIT batch_size"]
    MORE{Retorna true?\n(quedan pendientes)}
    CATCHUP["Enviar siguiente lote\nsin esperar tick\n(backlog catchup)"]
    DONE([Esperar siguiente tick])

    START --> CHECK_KEY
    CHECK_KEY -->|No| WAIT --> CHECK_KEY
    CHECK_KEY -->|Si| INTERVAL_CHANGE
    INTERVAL_CHANGE -->|Si| NEW_TICK --> TICK_WAIT
    INTERVAL_CHANGE -->|No| TICK_WAIT
    TICK_WAIT --> SEND_BATCH
    SEND_BATCH --> MORE
    MORE -->|Si| CATCHUP --> SEND_BATCH
    MORE -->|No| DONE --> TICK_WAIT
```

El mecanismo de **backlog catchup** garantiza que tras un corte de internet, todos los snapshots pendientes se envian de forma consecutiva sin esperar el proximo tick, recuperando el estado sincronizado rapidamente.

---

## 5. Recarga de Configuracion en Caliente

```mermaid
sequenceDiagram
    participant OPS as Operador
    participant UI as UI Local :8086
    participant HANDLER as PUT /api/config
    participant DB as energy.config
    participant CFG as runtimeConfig

    OPS->>UI: Editar campo (ej. energy_api_key)
    UI->>HANDLER: PUT /api/config {"energy_api_key": "xxx"}
    HANDLER->>DB: INSERT ON CONFLICT UPDATE
    HANDLER->>CFG: cfg.load() inmediato
    CFG-->>HANDLER: OK
    HANDLER-->>UI: {"updated": 1, "status": "ok"}

    Note over CFG: Goroutine en background\ntambien recarga cada config_reload_s
```

---

## 6. Endpoints HTTP del energy-sender

| Metodo | Ruta | Descripcion | Auth |
|---|---|---|---|
| `GET` | `/health` | Estado del servicio y config activa | Ninguna |
| `GET` | `/` | UI web local (dashboard) | Ninguna |
| `GET` | `/api/config` | Listar todas las claves de configuracion | Ninguna |
| `PUT` | `/api/config` | Actualizar una o varias claves y recargar | Ninguna |
| `GET` | `/api/stats` | Total, sincronizados y pendientes | Ninguna |
| `GET` | `/api/snapshots?filter=[all\|synced\|pending]` | Ultimos 50 snapshots con filtro | Ninguna |
| `POST` | `/config/reload` | Forzar recarga de config desde DB | Ninguna |

> La UI local no requiere autenticacion ya que solo es accesible dentro de la red local de planta.

---

## 7. UI Local — Vistas

```mermaid
graph LR
    subgraph NAV["Navegacion Sidebar"]
        D["Dashboard"]
        C["Configuracion"]
        S["Snapshots"]
    end

    D --> KPIs["KPIs en tiempo real\nTotal / Sincronizados\nPendientes / Health"]
    D --> TABLE_SNAP["Tabla ultimos snapshots\n(ID, device, meter, hora, interval, synced)"]
    C --> FORM["Formulario clave-valor\nActualizacion inmediata"]
    S --> TABLE_FULL["Tabla con filtro\nall / synced / pending"]
```

---

## 8. Schema de Base de Datos Local

### `energy.config`

| Columna | Tipo | Descripcion |
|---|---|---|
| `key` | TEXT PK | Nombre del parametro |
| `value` | TEXT | Valor del parametro |
| `updated_at` | TIMESTAMPTZ | Ultima actualizacion |

### `energy.snapshots` (local)

| Columna | Tipo | Descripcion |
|---|---|---|
| `id` | BIGSERIAL PK | Identificador local |
| `device_id` | VARCHAR(100) | ID del dispositivo edge |
| `meter_id` | VARCHAR(100) | ID del medidor Modbus |
| `hora` | TIMESTAMPTZ | Timestamp de la medicion |
| `interval_s` | INT | Intervalo de muestreo en segundos |
| `head` | TEXT[] | Nombres de columnas Modbus |
| `data` | TEXT[] | Valores correspondientes (strings) |
| `synced` | BOOLEAN | `false` = pendiente de envio |
| `created_at` | TIMESTAMPTZ | Timestamp de insercion |

Indice parcial `idx_energy_snap_synced ON energy.snapshots (synced) WHERE NOT synced` optimiza la consulta del loop de envio.

---

## 9. Variables de Entorno del Contenedor

| Variable | Descripcion | Valor por Defecto |
|---|---|---|
| `POSTGRES_URL` | Cadena de conexion a PostgreSQL local | `postgresql://mentor:mentor@postgres:5432/mentor_energy` |
| `PORT` | Puerto HTTP del servicio | `8086` |
| `TZ` | Zona horaria | `America/Lima` |

El resto de la configuracion operacional (device_id, api_key, intervalos) se gestiona exclusivamente desde `energy.config` en la base de datos.
