# 1. GENERALIDADES — Componentes Tecnológicos

## 1.1 Descripción General del Producto

**MENTOR EDGE** es una plataforma industrial de monitoreo de producción basada en visión artificial, diseñada para medir la Eficiencia General de los Equipos (OEE) en tiempo real en líneas de producción textil e industrial.

El sistema opera con una arquitectura distribuida de dos capas:

| Capa | Ubicación | Función Principal |
|------|-----------|-------------------|
| **Edge** | Dispositivo Jetson en planta | Detección en tiempo real mediante cámara, cálculo OEE local, operación offline |
| **Cloud** | Servidor en la nube | Gestión centralizada, dashboards analíticos, configuración remota, multi-tenancy |

### Problema que resuelve

En la industria textil y manufacturera, el conteo de unidades producidas y la detección de paradas se realiza manualmente o con sensores mecánicos costosos. MENTOR EDGE reemplaza estos métodos con **una sola cámara IP** que detecta el paso de productos mediante algoritmos de visión artificial, eliminando contacto físico con la línea de producción.

---

## 1.2 Componentes de Hardware

### 1.2.1 Dispositivo Edge — NVIDIA Jetson Orin

| Especificación | Detalle |
|----------------|---------|
| **Modelo** | NVIDIA Jetson Orin Nano / Orin NX |
| **GPU** | NVIDIA Ampere con aceleradores NVDEC, OFA, VIC |
| **CPU** | ARM Cortex-A78AE (6-8 cores) |
| **RAM** | 8 GB / 16 GB LPDDR5 |
| **Almacenamiento** | NVMe SSD 256 GB |
| **Conectividad** | Ethernet Gigabit, WiFi, USB 3.0 |
| **Consumo** | 7W - 25W (configurable) |
| **SO** | JetPack 6.0 (Ubuntu 22.04 + CUDA + TensorRT) |
| **Temperatura** | Operación: -25°C a 80°C |

**Aceleradores de hardware dedicados:**
- **NVDEC** — Decodificación H.264/H.265 por hardware (libera CPU del procesamiento de video)
- **OFA (Optical Flow Accelerator)** — Cálculo de flujo óptico en chip dedicado (~6.5ms latencia)
- **VIC (Video Image Composer)** — Conversiones de espacio de color por hardware

### 1.2.2 Cámara IP Industrial

| Especificación | Detalle |
|----------------|---------|
| **Protocolo** | RTSP sobre Ethernet |
| **Resolución** | 1080p / 720p configurable |
| **FPS** | 15-30 fps |
| **Tipo** | Cámara IP industrial con lente varifocal |
| **Montaje** | Soporte articulado sobre estructura de línea |
| **Alimentación** | PoE (Power over Ethernet) o 12V DC |
| **Protección** | IP66 (resistente a polvo y agua) |

### 1.2.3 Infraestructura de Red

| Componente | Función |
|------------|---------|
| Switch PoE | Alimentación y conectividad de cámaras |
| Router/Firewall | Conexión a internet para sincronización cloud |
| Cableado Cat6 | Conexión entre cámara, Jetson y red local |

### 1.2.4 Servidor Cloud

| Especificación | Detalle |
|----------------|---------|
| **Tipo** | VPS Linux (Ubuntu 22.04+) |
| **CPU** | 4 vCPU mínimo |
| **RAM** | 8 GB recomendado |
| **Disco** | 50 GB SSD |
| **Puerto** | 8888 (gateway API) |
| **Base de datos** | PostgreSQL 16 |
| **Containerización** | Docker 24+ con Docker Compose v2 |

---

## 1.3 Componentes de Software

### 1.3.1 Capa Edge (Jetson)

| Servicio | Lenguaje | Puerto | Función |
|----------|----------|--------|---------|
| **vision-event-detector** | Python 3.10 | 8001 | Motor de visión artificial con procesamiento GPU |
| **resiliencia** | Go 1.24 | 8002 | Buffer local de eventos con deduplicación |
| **enviador** | Go 1.24 | 8003 | Sincronización edge→cloud con retry exponencial |
| **edge-config-service** | Go 1.24 | 8004 | Gestión de configuración dinámica |
| **edge-gateway** | Go 1.24 | 8005 | Gateway local, SSE, sincronización cloud→edge |
| **ui-local** | Vue 3 / Nginx | 8080 | Dashboard web local de monitoreo |
| **tablet-app** | Vue 3 / Capacitor | 8090 | App para tablets de operadores |
| **PostgreSQL** | PostgreSQL 14 | 5432 | Base de datos local |

### 1.3.2 Capa Cloud (Servidor)

| Servicio | Lenguaje | Puerto | Función |
|----------|----------|--------|---------|
| **cloud-gateway** | Go 1.24 | 8888 | API gateway, SSE hub, rate limiting, auditoría |
| **cloud-identity** | Go 1.24 | 8081 | Autenticación JWT, usuarios, roles, empresas |
| **cloud-ingest** | Go 1.24 | 8082 | Ingesta de datos OEE desde dispositivos edge |
| **cloud-config** | Go 1.24 | 8083 | Configuración centralizada: plantas, líneas, productos |
| **cloud-analytics** | Go 1.24 | 8084 | Dashboards, reportes Pareto, análisis OEE |
| **cloud-integration** | Go 1.24 | 8085 | API para sistemas de terceros (ERP, MES) |
| **cloud-frontend** | Vue 3 / Nginx | 80 | SPA de gestión y dashboards |
| **PostgreSQL** | PostgreSQL 16 | 5432 | Base de datos multi-tenant |
| **Prometheus** | — | 9090 | Métricas de infraestructura |
| **Grafana** | — | 3000 | Visualización de métricas operativas |

---

## 1.4 Tecnologías Principales

### Procesamiento de Visión Artificial

| Tecnología | Versión | Uso |
|------------|---------|-----|
| OpenCV | 4.8.0 (NVIDIA-compiled con CUDA + GStreamer) | Procesamiento de imágenes y video |
| NVIDIA VPI | 3.0 | Vision Programming Interface para Jetson |
| GStreamer | Integrado en JetPack | Pipeline de video con decodificación hardware |
| NumPy | 1.24+ | Cómputo matricial para señales |

### Backend

| Tecnología | Versión | Uso |
|------------|---------|-----|
| Go | 1.23 / 1.24 | Microservicios backend (gateway, sync, config) |
| Gin | 1.10.0 | Framework HTTP para Go |
| pgx/v5 | 5.7.2 | Driver PostgreSQL nativo para Go |
| JWT | v5 | Autenticación basada en tokens |

### Frontend

| Tecnología | Versión | Uso |
|------------|---------|-----|
| Vue 3 | 3.4+ | Framework SPA (Composition API) |
| Tailwind CSS | 3.3+ | Diseño de interfaces |
| Apache ECharts | 5.4.3 | Gráficos OEE, Pareto, tendencias |
| Vue Flow | 1.48+ | Diagramas de flujo de procesos |
| Capacitor | 6.0 | App nativa para tablets (Android/iOS) |
| Vite | 5.0+ | Bundler y servidor de desarrollo |

### Infraestructura

| Tecnología | Versión | Uso |
|------------|---------|-----|
| Docker | 24+ | Contenedorización de todos los servicios |
| Docker Compose | v2 | Orquestación de microservicios |
| Nginx | 1.25 | Servidor web para frontends |
| PostgreSQL | 14 (edge) / 16 (cloud) | Bases de datos relacionales |

---

## 1.5 Algoritmos de Detección

El sistema utiliza **fusión multi-modal de señales** para detectar eventos de producción:

| Señal | Método | Costo Computacional | Descripción |
|-------|--------|---------------------|-------------|
| **Edge** | Detección de bordes Canny | Medio | Mide densidad de bordes para detectar presencia de producto |
| **Histogram** | Correlación de histograma HSV | Bajo | Detecta cambios de color respecto a referencia calibrada |
| **Flow** | Flujo óptico (OFA accelerator) | Muy bajo (hardware) | Detecta movimiento vertical del producto |
| **Beige** | Rango HSV inRange | Bajo | Detecta colores específicos del producto textil |

### Máquina de Estados Finitos (FSM)

```
IDLE → (señal > umbral_alto) → DETECTING
DETECTING → (N frames confirmados) → CONFIRMING → EVENTO EMITIDO
CONFIRMING → COOLDOWN → (anti-rebote) → IDLE
```

- **n_frames**: 1-30 frames de confirmación antes de confirmar evento
- **cooldown**: 0-60 frames de enfriamiento anti-rebote
- **Calibración automática**: El sistema aprende el color de referencia en 30 frames

---

## 1.6 Comunicación entre Capas

```
┌──────────── PLANTA (Edge) ────────────┐    ┌──────────── NUBE (Cloud) ─────────────┐
│                                        │    │                                        │
│  Cámara IP ──RTSP──→ Vision Detector   │    │   cloud-gateway :8888                  │
│                          │              │    │      │                                 │
│                     HTTP POST           │    │   cloud-config (configuración)         │
│                          │              │    │   cloud-ingest (ingesta OEE)           │
│                          ▼              │    │   cloud-analytics (dashboards)         │
│                     Resiliencia         │    │   cloud-identity (auth)                │
│                          │              │    │   cloud-frontend (SPA)                 │
│                     PostgreSQL          │    │      │                                 │
│                          │              │    │   PostgreSQL 16 (multi-tenant)         │
│                          ▼              │    │                                        │
│                      Enviador ──────────┼───→│   /api/v1/edge/oee (REST)              │
│                                        │    │                                        │
│  Edge Gateway ←─────────SSE────────────┼────│   /api/v1/edge/stream (SSE push)       │
│       │                                │    │                                        │
│  Config Service                        │    │                                        │
│  UI Local / Tablet                     │    │                                        │
└────────────────────────────────────────┘    └────────────────────────────────────────┘
```

### Protocolos

| Dirección | Protocolo | Endpoint | Datos |
|-----------|-----------|----------|-------|
| Edge → Cloud | REST (HTTPS) | `/api/v1/edge/oee` | Snapshots OEE, paradas, production runs |
| Cloud → Edge | SSE (Server-Sent Events) | `/api/v1/edge/stream` | Configuración, productos, turnos, comandos |
| Cámara → Jetson | RTSP | `rtsp://ip:554/stream` | Video H.264 en tiempo real |
| Detector → Buffer | HTTP | `POST /events` | Eventos de producción con confianza |

### Seguridad

| Mecanismo | Aplicación |
|-----------|------------|
| EDGE_API_KEY | Autenticación de dispositivos edge |
| JWT (Bearer) | Autenticación de usuarios en dashboard |
| INTERNAL_API_KEY | Comunicación entre microservicios cloud |
| Rate Limiting | Token-bucket: 20 req/s edge, 50 req/s usuarios |
