# ENERGY 05 — Frontend Cloud: UI de Energia

## 1. Descripcion

La UI cloud del subsistema de energía esta implementada en Vue 3 y se divide en dos modulos: `energia` (monitoreo en tiempo real y gestion de medidores) y `analisis-energia` (analisis tarifario y factor de calificacion). Todos los datos provienen del servicio `energy-ingest` via la capa `energy.service.js`.

---

## 2. Mapa de Modulos y Vistas

```mermaid
graph TD
    subgraph Modulo_Energia["Modulo: energia"]
        MON["MonitorEnergiaView\n/energia/monitor\nSnapshots en tiempo real\nKPIs agregados por medidor"]
        MED["MedidoresView\n/energia/medidores\nCRUD de medidores\nAsignacion empresa/planta"]
    end

    subgraph Modulo_Analisis["Modulo: analisis-energia"]
        TAR["ConsumoElectricoTarifarioView\n/analisis-energia/consumo-tarifario\nAnalisis de consumo por estructura tarifaria\nMT3 / MT4 / BT3"]
        FAC["FactorCalificacionView\n/analisis-energia/factor-calificacion\nCalificacion del factor de potencia\nPor periodo y ubicacion"]
    end

    subgraph Modulo_Analisis2["Modulo: analisis (OEE)"]
        ENER["EnergiaView\n/analisis/energia\nVista integrada OEE + Energia"]
    end

    subgraph Service["Capa de Servicio"]
        SVC["energy.service.js\nWraps fetch → /api/energy/*"]
    end

    MON --> SVC
    MED --> SVC
    TAR --> SVC
    FAC --> SVC
    ENER --> SVC
```

---

## 3. MonitorEnergiaView — Logica de Componente

```mermaid
flowchart TD
    MOUNT([onMounted])
    FETCH_M["fetchMeters()\nGET /api/energy/meters"]
    FETCH_S["fetchSnapshots()\nGET /api/energy/snapshots\n?meter_id&from&to&limit&offset"]

    LATEST["computed: latestByMeter\nUltimo snapshot por cada meter_id"]
    SUMMARY["computed: summaryCards\nPromedio de KPIs entre medidores\n- Potencia Activa (W)\n- Potencia Reactiva (VAR)\n- Factor de Potencia\n- Energia Activa (kWh)"]
    PAGES["computed: totalPages\nCeil(total / pageSize=25)"]

    TABLE["Tabla paginada\ncolumnas: hora, meter_id,\nV avg, I avg, P activa,\nP reactiva, FP, Freq, E activa"]
    DETAIL["Panel de detalle\nSnapshot seleccionado\nTodos los campos electricos"]
    FILTERS["Filtros\nmeter_id (select)\nfrom / to (datetime)"]

    MOUNT --> FETCH_M
    MOUNT --> FETCH_S
    FETCH_S --> LATEST
    LATEST --> SUMMARY
    FETCH_S --> PAGES
    SUMMARY --> TABLE
    PAGES --> TABLE
    FILTERS -->|"applyFilters()\npage=1"| FETCH_S
    TABLE -->|"click fila"| DETAIL
```

### KPIs del panel de resumen

| Campo | Etiqueta | Unidad | Color |
|---|---|---|---|
| `potencia_activa` | Potencia Activa | W | Amarillo |
| `potencia_reactiva` | Potencia Reactiva | VAR | Azul |
| `factor_potencia` | Factor de Potencia | — | Verde esmeralda |
| `energia_activa` | Energia Activa | kWh | Morado |

### Columnas de la tabla principal

| Campo | Etiqueta | Formato |
|---|---|---|
| `hora` | Hora | datetime |
| `meter_id` | Medidor | string |
| `voltaje_avg` | V avg | V |
| `corriente_avg` | I avg | A |
| `potencia_activa` | P activa | W |
| `potencia_reactiva` | P reactiva | VAR |
| `factor_potencia` | FP | decimal |
| `frecuencia_hz` | Freq | Hz |
| `energia_activa` | E activa | kWh |

---

## 4. MedidoresView — CRUD de Medidores

```mermaid
stateDiagram-v2
    [*] --> Listado: onMounted → GET /api/energy/meters

    Listado --> Creando: click "Nuevo medidor"
    Creando --> Listado: POST /api/energy/meters\n+ refetch

    Listado --> Editando: click "Editar"
    Editando --> Listado: PUT /api/energy/meters/:id\n+ refetch

    Listado --> Eliminando: click "Eliminar"
    Eliminando --> Listado: DELETE /api/energy/meters/:id\n+ refetch
```

Campos del formulario de medidor:

| Campo | Tipo | Descripcion |
|---|---|---|
| `device_id` | string | ID del dispositivo edge asociado |
| `meter_id` | string | ID del medidor Modbus |
| `nombre` | string | Nombre descriptivo |
| `ubicacion` | string | Ubicacion fisica en planta |
| `empresa_id` | int | Empresa (desde JWT en creacion) |
| `planta_id` | int | Planta de produccion |

---

## 5. ConsumoElectricoTarifarioView — Analisis Tarifario

```mermaid
flowchart LR
    subgraph Filtros["Panel de Filtros"]
        COMP["Compania"]
        PLANT["Planta"]
        LINE["Linea"]
        UBIC["Ubicacion"]
        DISP["Dispositivo"]
        AGRUP["Agrupamiento\n5min / 15min / 30min / 1h"]
        TAR["Estructura Tarifaria\nMT3 / MT4 / BT3"]
        VAR["Variable\nEnergia Real / Reactiva"]
        RANGO["Rango de fechas\nfecha_inicio / fecha_fin"]
    end

    subgraph Salida["Resultados"]
        GRAFICO["Grafico de barras\npor franja tarifaria\nHP / HFP / HP PLUS / BASE"]
        TABLA["Tabla de consumo\npor periodo"]
    end

    Filtros -->|"buscar()"| Salida
```

### Estructuras Tarifarias Soportadas

| Codigo | Nombre | Descripcion |
|---|---|---|
| `MT3` | Media Tension 3 | Tarifa en media tension, sin discriminacion horaria |
| `MT4` | Media Tension 4 | Tarifa en media tension con dos periodos |
| `BT3` | Baja Tension 3 | Tarifa en baja tension |

### Franjas Horarias (colores de graficos)

| Franja | Color |
|---|---|
| HP (Horas Punta) | Rojo `#ef4444` |
| HFP (Horas Fuera Punta) | Ambar `#f59e0b` |
| HP PLUS | Violeta `#8b5cf6` |
| BASE | Azul `#3b82f6` |

---

## 6. Capa de Servicio: energy.service.js

```mermaid
graph TD
    subgraph SVC["energy.service.js"]
        GS["getSnapshots(params)\nGET /api/energy/snapshots"]
        GM["getMeters()\nGET /api/energy/meters"]
        GMID["getMeterById(id)\nGET /api/energy/meters/:id"]
        CM["createMeter(data)\nPOST /api/energy/meters"]
        UM["updateMeter(id, data)\nPUT /api/energy/meters/:id"]
        DM["deleteMeter(id)\nDELETE /api/energy/meters/:id"]
    end

    subgraph Params_Snapshots["Parametros getSnapshots"]
        P1["limit: number (default 50)"]
        P2["offset: number"]
        P3["meter_id?: string"]
        P4["device_id?: string"]
        P5["from?: ISO datetime string"]
        P6["to?: ISO datetime string"]
        P7["planta_id?: number"]
    end

    GS --- Params_Snapshots
```

Todos los metodos incluyen el JWT Bearer Token via el interceptor Axios/Fetch global del frontend. El `empresa_id` nunca se envia desde el cliente; el backend lo extrae del token.

---

## 7. Flujo de Navegacion

```mermaid
graph LR
    NAV["Sidebar\nNavegacion principal"]

    NAV --> M1["Energia\n(menu padre)"]
    NAV --> M2["Analisis de Energia\n(menu padre)"]

    M1 --> V1["Monitor de Energia\n/energia/monitor"]
    M1 --> V2["Medidores\n/energia/medidores"]

    M2 --> V3["Consumo Electrico Tarifario\n/analisis-energia/consumo-tarifario"]
    M2 --> V4["Factor de Calificacion\n/analisis-energia/factor-calificacion"]
```
