# ENERGY 06 — Modelo de Base de Datos

## 1. Descripcion

Documenta los schemas de base de datos del subsistema de energía en ambos extremos: la base local de la Raspberry Pi (`mentor_energy`) y el schema cloud (`energy.*` en PostgreSQL cloud). Incluye el vinculo con `gateway.device_registry` para la resolucion de scope multitenancy.

---

## 2. Diagrama Entidad-Relacion: Base Local Edge

```mermaid
erDiagram
    energy_config {
        TEXT key PK
        TEXT value
        TIMESTAMPTZ updated_at
    }

    energy_snapshots_local {
        BIGSERIAL id PK
        VARCHAR device_id
        VARCHAR meter_id
        TIMESTAMPTZ hora
        INT interval_s
        TEXT_ARRAY head
        TEXT_ARRAY data
        BOOLEAN synced
        TIMESTAMPTZ created_at
    }

    energy_config ||--o{ energy_snapshots_local : "parametriza envio"
```

---

## 3. Diagrama Entidad-Relacion: Schema Cloud `energy.*`

```mermaid
erDiagram
    energy_meters {
        SERIAL id PK
        TEXT device_id
        TEXT meter_id
        INTEGER empresa_id FK
        INTEGER planta_id FK
        TEXT nombre
        TEXT ubicacion
        BOOLEAN activo
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    energy_snapshots {
        BIGSERIAL id PK
        TEXT device_id
        TEXT meter_id
        TIMESTAMPTZ hora
        INTEGER interval_s
        TEXT_ARRAY head
        TEXT_ARRAY data
        INTEGER empresa_id FK
        INTEGER planta_id FK
        DOUBLE corriente_a
        DOUBLE corriente_b
        DOUBLE corriente_c
        DOUBLE corriente_avg
        DOUBLE voltaje_a
        DOUBLE voltaje_b
        DOUBLE voltaje_c
        DOUBLE voltaje_avg
        DOUBLE voltaje_ab
        DOUBLE voltaje_bc
        DOUBLE voltaje_ac
        DOUBLE potencia_activa
        DOUBLE potencia_reactiva
        DOUBLE potencia_aparente
        DOUBLE factor_potencia
        DOUBLE frecuencia_hz
        DOUBLE energia_activa
        DOUBLE energia_reactiva
        DOUBLE energia_aparente
        DOUBLE thd_ia
        DOUBLE thd_ib
        DOUBLE thd_ic
        DOUBLE thd_ua
        DOUBLE thd_ub
        DOUBLE thd_uc
        TIMESTAMPTZ created_at
    }

    energy_device_sync_log {
        BIGSERIAL id PK
        TEXT device_id
        INTEGER batch_size
        TEXT status
        TEXT error_msg
        TIMESTAMPTZ created_at
    }

    gateway_device_registry {
        TEXT device_id PK
        INTEGER empresa_id FK
        INTEGER planta_id FK
        INTEGER linea_id FK
        BOOLEAN active
        TIMESTAMPTZ last_seen_at
    }

    energy_meters ||--o{ energy_snapshots : "device_id + meter_id"
    gateway_device_registry ||--o{ energy_snapshots : "resuelve scope"
    gateway_device_registry ||--o{ energy_device_sync_log : "audita sync"
```

---

## 4. Descripcion Detallada: `energy.snapshots` (Cloud)

### Campos de Identificacion

| Columna | Tipo | Descripcion |
|---|---|---|
| `id` | BIGSERIAL PK | Clave primaria autoincremental |
| `device_id` | TEXT | ID del dispositivo edge (desde `X-Device-ID`) |
| `meter_id` | TEXT | ID del medidor Modbus dentro del dispositivo |
| `hora` | TIMESTAMPTZ | Timestamp de la medicion (del edge) |
| `interval_s` | INTEGER | Intervalo de muestreo en segundos |
| `head` | TEXT[] | Nombres originales de columnas Modbus |
| `data` | TEXT[] | Valores originales como strings |
| `empresa_id` | INTEGER | Resuelto desde `gateway.device_registry` |
| `planta_id` | INTEGER | Resuelto desde `gateway.device_registry` |
| `created_at` | TIMESTAMPTZ | Timestamp de insercion en cloud |

### Campos Electricos — Corrientes

| Columna | Tipo | Unidad | Descripcion |
|---|---|---|---|
| `corriente_a` | DOUBLE PRECISION | A | Corriente fase A |
| `corriente_b` | DOUBLE PRECISION | A | Corriente fase B |
| `corriente_c` | DOUBLE PRECISION | A | Corriente fase C |
| `corriente_avg` | DOUBLE PRECISION | A | Corriente promedio trifasica |

### Campos Electricos — Voltajes

| Columna | Tipo | Unidad | Descripcion |
|---|---|---|---|
| `voltaje_a` | DOUBLE PRECISION | V | Voltaje fase A (fase-neutro) |
| `voltaje_b` | DOUBLE PRECISION | V | Voltaje fase B (fase-neutro) |
| `voltaje_c` | DOUBLE PRECISION | V | Voltaje fase C (fase-neutro) |
| `voltaje_avg` | DOUBLE PRECISION | V | Voltaje promedio trifasico |
| `voltaje_ab` | DOUBLE PRECISION | V | Voltaje linea A-B (fase-fase) |
| `voltaje_bc` | DOUBLE PRECISION | V | Voltaje linea B-C (fase-fase) |
| `voltaje_ac` | DOUBLE PRECISION | V | Voltaje linea A-C (fase-fase) |

### Campos Electricos — Potencias y Energia

| Columna | Tipo | Unidad | Descripcion |
|---|---|---|---|
| `potencia_activa` | DOUBLE PRECISION | W | Potencia activa total |
| `potencia_reactiva` | DOUBLE PRECISION | VAR | Potencia reactiva total |
| `potencia_aparente` | DOUBLE PRECISION | VA | Potencia aparente total |
| `factor_potencia` | DOUBLE PRECISION | — | Factor de potencia (0-1) |
| `frecuencia_hz` | DOUBLE PRECISION | Hz | Frecuencia de red |
| `energia_activa` | DOUBLE PRECISION | kWh | Energia activa acumulada |
| `energia_reactiva` | DOUBLE PRECISION | kVARh | Energia reactiva acumulada |
| `energia_aparente` | DOUBLE PRECISION | kVAh | Energia aparente acumulada |

### Campos Electricos — Distorsion Armonica Total (THD)

| Columna | Tipo | Unidad | Descripcion |
|---|---|---|---|
| `thd_ia` | DOUBLE PRECISION | % | THD de corriente fase A |
| `thd_ib` | DOUBLE PRECISION | % | THD de corriente fase B |
| `thd_ic` | DOUBLE PRECISION | % | THD de corriente fase C |
| `thd_ua` | DOUBLE PRECISION | % | THD de voltaje fase A |
| `thd_ub` | DOUBLE PRECISION | % | THD de voltaje fase B |
| `thd_uc` | DOUBLE PRECISION | % | THD de voltaje fase C |

---

## 5. Indices y Estrategia de Consulta

```mermaid
graph TD
    subgraph Tabla["energy.snapshots"]
        UK["UNIQUE (device_id, meter_id, hora)\nGarantiza idempotencia — UPSERT seguro"]
        I1["idx_energy_snapshots_device_hora\n(device_id, hora DESC)\nConsultas por dispositivo con rango temporal"]
        I2["idx_energy_snapshots_meter_hora\n(meter_id, hora DESC)\nConsultas por medidor especifico"]
        I3["idx_energy_snapshots_empresa_hora\n(empresa_id, hora DESC)\nAislamiento multitenancy — filtro primario del dashboard"]
        I4["idx_energy_snapshots_planta_hora\n(planta_id, hora DESC)\nConsultas por planta de produccion"]
    end
```

### Patron de UPSERT

La constraint `UNIQUE (device_id, meter_id, hora)` permite que los re-envios del edge (tras cortes de red) no generen duplicados. El repositorio usa `INSERT ... ON CONFLICT (device_id, meter_id, hora) DO UPDATE SET ...` para actualizar el registro si ya existe.

---

## 6. Tabla `energy.config` (Edge Local)

```mermaid
graph LR
    subgraph energy_config["energy.config (PostgreSQL Local)"]
        K1["device_id\nID del dispositivo"]
        K2["meter_id_1\nID del medidor"]
        K3["meter_unit_id\nDireccion Modbus"]
        K4["cloud_url\nURL destino"]
        K5["energy_api_key\nClave de autenticacion"]
        K6["send_interval_s\nIntervalo de envio"]
        K7["batch_size\nTamano de lote"]
        K8["config_reload_s\nIntervalo de recarga"]
    end

    SENDER["energy-sender\nlee al iniciar\ny cada config_reload_s"]
    SENDER --> energy_config
```

---

## 7. Relacion con gateway.device_registry

```mermaid
graph LR
    DEV["gateway.device_registry\ndevice_id = 'rpi-energy-01'\nempresa_id = 5\nplanta_id = 12\nlinea_id = 3\nactive = true\nlast_seen_at = NOW()"]

    INGEST["energy-ingest\nScopeResolver.ResolveByDevice()"]

    subgraph Destinos["Escritura segun scope"]
        GLOBAL["energy.snapshots\nenergy_id = 5\nplanta_id = 12"]
        PLANTA["planta_12.energy_readings\nlinea_id = 3"]
    end

    INGEST -->|"SELECT"| DEV
    DEV -->|"scope resuelto"| INGEST
    INGEST --> GLOBAL
    INGEST --> PLANTA
```

Cuando un dispositivo no esta registrado en `gateway.device_registry` o su `active = false`, la resolucion de scope falla y los snapshots se escriben unicamente en `energy.*` global sin asignacion de empresa/planta. El error se registra en el log pero no interrumpe la ingesta.

---

## 8. Migraciones Relevantes

| Archivo | Descripcion |
|---|---|
| `26_energy_local.sql` | Schema local edge: `energy.config` + `energy.snapshots` |
| `17_energy_schema.sql` | Schema cloud inicial: `energy.meters`, `energy.snapshots`, `energy.device_sync_log` + indices |
| `18_integration_energy_scope.sql` | Integracion de columnas `empresa_id` / `planta_id` en snapshots |
| `29_energy_schema.sql` | Revision y ajustes de constraints del schema cloud |
| `energy_linea_template.sql` | Template para crear schema de energia por planta (Option B) |
