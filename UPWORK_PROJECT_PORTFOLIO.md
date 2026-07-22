# Mentor Edge - Portfolio Project Description

## Título del Proyecto
**Plataforma Industrial IoT Edge-Cloud para Monitoreo OEE con Visión Artificial**

---

## Tu Rol
**Arquitecto de Software Full-Stack e Ingeniero de Sistemas Edge/Cloud IoT**

---

## Descripción del Proyecto

Plataforma industrial distribuida para monitoreo de producción y energía en tiempo real mediante visión artificial, IoT y telemetría industrial. Sistema completo de medición OEE (Overall Equipment Effectiveness) con tres componentes principales:

**1. Edge Computing - Jetson Orin (Visión Artificial)**
- Detección de eventos de producción mediante cámara IP y YOLO v8
- Procesamiento en tiempo real con GPU NVIDIA + TensorRT
- Buffer offline-first con resiliencia de 6 meses en PostgreSQL local
- Microservicios en Go y Python con arquitectura hexagonal

**2. Tablet App - Interfaz de Operador**
- App multiplataforma con Vue 3 + Capacitor
- Control de línea en tiempo real, justificación de paradas
- Sincronización bidireccional via SSE (Server-Sent Events)
- Dashboard de producción y OEE instantáneo

**3. Energy Monitoring - Raspberry Pi (Telemetría)**
- Adquisición de datos eléctricos via Modbus RTU desde medidores MEATROL MC60
- Node-RED para lectura de 27 variables (voltajes trifásicos, corrientes, potencias, THD)
- Pipeline automático: Modbus → PostgreSQL local → Cloud con retry resiliente
- Análisis tarifario y monitoreo de consumo energético

**Cloud Backend Multi-Tenant**
- 6 microservicios Go con API Gateway centralizado
- Multi-tenancy con aislamiento por empresa y planta
- Dashboards analíticos, reportes Pareto, integraciones ERP/MES
- PostgreSQL 16 con schemas por dominio

---

## Stack Tecnológico

### Edge Computing (Jetson Orin)
- **Lenguajes**: Python 3.10, Go 1.24
- **Computer Vision**: OpenCV, YOLO v8, TensorRT, CUDA
- **Hardware Acceleration**: NVIDIA NVDEC, OFA (Optical Flow Accelerator)
- **Servicios**: vision-event-detector, resiliencia, enviador, edge-gateway, edge-config
- **Base de Datos**: PostgreSQL 14
- **Containerización**: Docker + Docker Compose

### Tablet Application
- **Framework**: Vue 3 + Vite
- **Mobile**: Capacitor (Android/iOS)
- **UI**: Tailwind CSS
- **Build**: TypeScript, Nginx
- **Features**: SSE real-time, control de turnos, justificación paradas

### Energy Monitoring (Raspberry Pi)
- **Plataforma**: Node-RED 3.1.9
- **Protocolo Industrial**: Modbus RTU (RS-485)
- **Hardware**: MEATROL MC60 energy meter
- **Lenguaje Backend**: Go 1.24 (energy-sender service)
- **Variables**: 27 parámetros eléctricos (V, I, P, Q, S, PF, Freq, THD, Energy)
- **Persistencia**: PostgreSQL 15 Alpine

### Cloud Services
- **Backend**: Go 1.24 (6 microservices)
- **API Gateway**: Go + net/http con proxy reverso
- **Servicios**: cloud-identity, cloud-ingest, cloud-config, cloud-analytics, cloud-integration, energy-ingest
- **Frontend**: Vue 3 + TypeScript + Vite
- **Base de Datos**: PostgreSQL 16
- **Autenticación**: JWT + API Keys multi-nivel
- **Real-time**: Server-Sent Events (SSE)
- **Deployment**: Docker Swarm / Docker Compose

---

## Habilidades Técnicas Aplicadas

### Lenguajes de Programación
- Go (Golang)
- Python
- JavaScript/TypeScript
- SQL

### Frameworks y Librerías
- Vue.js 3 (Composition API)
- Gin (Go web framework)
- OpenCV
- TensorRT
- Capacitor
- Tailwind CSS

### Tecnologías de Edge/IoT
- NVIDIA Jetson Platform
- Modbus RTU/RS-485
- Node-RED
- RTSP Camera Streaming
- Industrial Protocols

### Computer Vision y AI
- YOLO v8 Object Detection
- Optical Flow Processing
- Hardware-Accelerated Video Decoding
- TensorRT Optimization
- FSM-based Event Detection

### Arquitectura y Patrones
- Microservices Architecture
- Hexagonal Architecture (Ports & Adapters)
- Event-Driven Design
- Offline-First Pattern
- Multi-Tenancy Design
- API Gateway Pattern
- CQRS (Command Query Separation)

### DevOps e Infraestructura
- Docker & Docker Compose
- PostgreSQL (Multi-Schema Design)
- Nginx Reverse Proxy
- Linux System Administration
- Git Version Control

### Protocolos y Comunicación
- REST API
- Server-Sent Events (SSE)
- WebSocket
- Modbus RTU
- RTSP Streaming
- HTTPS/TLS

---

## Características Técnicas Destacadas

### Resiliencia y Confiabilidad
- Buffer local de 6 meses en edge para operación offline
- Retry exponencial con backoff en sincronización cloud
- Deduplicación de eventos por hash SHA-256
- Manejo de desconexiones sin pérdida de datos
- Catchup automático tras reconexión

### Rendimiento
- Procesamiento en tiempo real: 30 FPS con latencia <100ms
- Aceleración GPU para decodificación de video y optical flow
- Pipeline asíncrono con goroutines (Go concurrency)
- Optimización TensorRT para inferencia YOLO
- Batch processing para ingesta de datos

### Escalabilidad
- Arquitectura multi-tenant con aislamiento por empresa
- Schemas PostgreSQL dinámicos por planta
- Rate limiting y throttling en API Gateway
- Horizontal scaling capability
- Configuration hot-reload sin reinicio

### Seguridad
- Autenticación multi-nivel (JWT + API Keys)
- Zero-trust entre capas
- API Key por dispositivo edge
- Internal API Key para comunicación inter-servicios
- Audit logging completo en gateway

### Monitoreo y Observabilidad
- Dashboard local en cada edge device (:8080)
- UI de configuración para sistema de energía (:8086)
- Logs estructurados en todos los servicios
- Health checks y status endpoints
- Métricas de sincronización y latencia

---

## Componentes del Sistema

### 1. Subsistema OEE (Jetson Orin)

**Hardware:**
- NVIDIA Jetson Orin Nano/NX (8-16GB RAM)
- Cámara IP industrial RTSP (1080p)
- NVMe SSD 256GB
- Consumo: 7W-25W

**Servicios Edge:**
- `vision-event-detector` (Python :8001) - Motor de visión con 4 señales: Edge Detection, HSV Color, Optical Flow, YOLO
- `resiliencia` (Go :8002) - Buffer PostgreSQL con deduplicación
- `enviador` (Go :8003) - Sincronización edge→cloud con 6 goroutines
- `edge-gateway` (Go :8005) - API unificada + SSE broker
- `edge-config-service` (Go :8004) - Gestión de configuración dinámica
- `yolo-counter` (Python :8006) - Conteo de productos con YOLO
- `ui-local` (Vue 3 :8080) - Dashboard web local

**Capacidades:**
- Detección de corte en líneas textiles
- Tracking de paradas con máquina de estados (FSM)
- Cálculo OEE local en tiempo real
- Gestión de ROIs configurables
- Fusión multi-señal con pesos adaptativos

### 2. Tablet Application

**Tecnologías:**
- Vue 3 + Composition API
- Capacitor para Android/iOS
- Tailwind CSS para UI responsive
- TypeScript para type safety
- Vite para build optimizado

**Funcionalidades:**
- Control de inicio/fin de turno
- Visualización de OEE en tiempo real
- Justificación de paradas con categorías
- Selección de productos y velocidades nominales
- Modo offline con sincronización diferida
- Notificaciones push para alarmas

### 3. Energy Monitoring System

**Hardware:**
- Raspberry Pi 4/5
- Medidor MEATROL MC60 (Modbus RTU)
- Conversor USB-RS485

**Pipeline de Datos:**
1. **Adquisición**: Node-RED lee 27 variables via Modbus RTU cada 10s
2. **Procesamiento**: Conversión de registros Modbus a valores Float/Int64
3. **Buffer Local**: PostgreSQL almacena snapshots cada 5 minutos
4. **Sincronización**: Go service envía batch al cloud cada 30s
5. **Cloud Storage**: PostgreSQL cloud con schemas por planta
6. **Visualización**: Frontend Vue 3 con análisis tarifario

**Variables Monitoreadas:**
- Voltajes trifásicos (fase y línea): Va, Vb, Vc, Vab, Vbc, Vac
- Corrientes trifásicas: Ia, Ib, Ic, In
- Potencias: Activa (P), Reactiva (Q), Aparente (S)
- Factor de Potencia (PF)
- Frecuencia (Hz)
- Energías acumuladas: Activa, Reactiva, Aparente (Int64)
- THD (Total Harmonic Distortion): corrientes y voltajes

**Node-RED Flow Features:**
- Configuración dinámica desde base de datos
- Unit ID configurable via variables globales
- Snapshot cada 5 minutos con timestamp normalizado
- Flag `synced` para control de envío al cloud
- Manejo robusto de errores Modbus

### 4. Cloud Platform

**Microservicios:**
- `cloud-gateway` (:8888) - Router, SSE hub, rate limiting, audit log
- `cloud-identity` (:8081) - Autenticación JWT, usuarios, roles
- `cloud-ingest` (:8082) - Ingesta de snapshots OEE desde edge
- `cloud-config` (:8083) - Configuración centralizada (plantas, líneas, productos)
- `cloud-analytics` (:8084) - Dashboards, Pareto, análisis OEE
- `cloud-integration` (:8085) - API para terceros (ERP/MES)
- `energy-ingest` (:8087) - Ingesta de datos eléctricos
- `cloud-frontend` (:80) - SPA Vue 3

**Base de Datos PostgreSQL:**
```
mentor_cloud
├── identity       (usuarios, roles, empresas, api_keys)
├── config         (plantas, líneas, dispositivos, productos, turnos)
├── ingest         (raw_events, oee_snapshots, device_sync_log)
├── analytics      (paradas, alarmas, compromisos, production_runs)
├── gateway        (audit_log, commands, device_registry)
├── integration    (api_keys para terceros)
└── energy         (meters, snapshots, config, sync_log)

planta_1, planta_2, ... (schemas dinámicos por planta)
```

**Features Cloud:**
- Multi-tenancy con filtrado por empresa_id
- SSE push para comandos en tiempo real
- Scoped sync: cada dispositivo recibe solo su config
- Rate limiting por IP y API Key
- Audit trail completo de operaciones

---

## Flujos de Datos Principales

### Flujo OEE (Producción)
```
Cámara RTSP → vision-event-detector (GPU) → resiliencia (buffer) →
enviador → HTTPS → cloud-gateway → cloud-ingest →
PostgreSQL Cloud → cloud-analytics → Frontend Dashboard
                                   ↓
                           Tablet (SSE real-time)
```

### Flujo Energy (Telemetría)
```
Medidor MC60 → Modbus RTU → Node-RED → PostgreSQL Local →
energy-sender (Go) → HTTPS → cloud-gateway → energy-ingest →
PostgreSQL Cloud → Frontend (Análisis Tarifario)
```

### Flujo Configuración (Bidireccional)
```
Frontend → cloud-config → PostgreSQL → cloud-gateway (SSE) →
edge-gateway → edge-config-service → PostgreSQL Edge →
vision-event-detector (hot-reload)
```

---

## Logros Técnicos

### Performance
- Procesamiento de video 30 FPS con latencia <100ms
- Decodificación H.264 por hardware (NVDEC)
- Optical flow acelerado por chip OFA (~6.5ms)
- Inferencia YOLO optimizada con TensorRT
- Sincronización cloud con throughput >1000 eventos/min

### Reliability
- Sistema offline-first con 6 meses de buffer
- Zero data loss ante cortes de red prolongados
- Retry exponencial con jitter aleatorio
- Deduplicación SHA-256 para prevenir duplicados
- Watchdog y auto-recovery de servicios

### Scalability
- Soporte para múltiples empresas, plantas y líneas
- Schemas PostgreSQL dinámicos por planta
- Cada edge opera independientemente
- Cloud escalable horizontalmente
- Configuration hot-reload sin downtime

### Security
- API Keys únicas por dispositivo
- JWT con refresh tokens para usuarios
- Internal API Key para comunicación inter-servicios
- HTTPS end-to-end
- Aislamiento de datos por tenant

---

## Desafíos Técnicos Resueltos

### 1. Procesamiento de Video en Tiempo Real
**Problema**: Procesar 30 FPS de video en Jetson con múltiples algoritmos consume mucha CPU/GPU.

**Solución**:
- Hardware acceleration con NVDEC para decodificación H.264
- OFA chip dedicado para optical flow
- TensorRT para optimización de YOLO
- Pipeline asíncrono con multiprocessing

### 2. Resiliencia Offline
**Problema**: Plantas industriales con conectividad intermitente.

**Solución**:
- PostgreSQL local como buffer con 6 meses de retención
- Deduplicación por hash SHA-256
- Retry exponencial con backoff configurable
- Catchup automático tras reconexión

### 3. Sincronización Bidireccional
**Problema**: Configuración debe propagarse desde cloud a edge en tiempo real.

**Solución**:
- Server-Sent Events (SSE) para push desde cloud
- Comandos SYNC_* con replay de comandos perdidos
- Hot-reload de configuración sin reinicio de servicios
- Versioning de configuración con timestamps

### 4. Multi-Tenancy en Edge y Cloud
**Problema**: Aislar datos de múltiples empresas compartiendo infraestructura.

**Solución**:
- Scoped sync: edge solo recibe config filtrada por empresa/planta
- Schemas PostgreSQL dinámicos por planta en cloud
- API Key única por dispositivo con metadata de empresa
- Row-level filtering en todas las queries

### 5. Lectura Modbus RTU Confiable
**Problema**: Medidores eléctricos con comunicación serial frágil.

**Solución**:
- Node-RED con retry automático y timeout configurable
- Buffer de comandos Modbus para prevenir colisiones
- Validación de buffers y conversión robusta Float/Int64
- Persistencia local con flag synced para garantizar envío

### 6. Detección de Eventos de Producción
**Problema**: Líneas de producción variables con cambios de iluminación y velocidad.

**Solución**:
- Fusión de 4 señales de visión con pesos adaptativos
- ROIs configurables por tipo de línea
- Calibración automática de thresholds
- FSM (Finite State Machine) para tracking de estados

---

## Métricas del Proyecto

- **Líneas de Código**: ~50,000+ líneas
- **Microservicios**: 8 servicios Go + 2 servicios Python
- **Servicios Edge**: 7 contenedores Docker por dispositivo Jetson
- **Servicios Cloud**: 8 contenedores Docker
- **Base de Datos**: 7 schemas PostgreSQL + schemas dinámicos por planta
- **APIs REST**: 120+ endpoints
- **Node-RED Flows**: 3 flows principales con 50+ nodos
- **Capacidad de Buffer**: 6 meses / ~13 millones de eventos por dispositivo
- **Variables Eléctricas**: 27 parámetros por snapshot cada 5 minutos
- **Latencia E2E**: <2 segundos desde detección hasta cloud
- **Uptime**: >99.5% en producción

---

## Deployment y Operación

### Edge Deployment (Jetson/Raspberry)
```bash
# Jetson Orin - OEE System
docker compose -f docker-compose.jetson.yml up -d

# Raspberry Pi - Energy Monitoring
docker compose -f docker-compose.rpi-energy.yml up -d
```

### Cloud Deployment
```bash
# Cloud VPS - Multi-Tenant Platform
cd mentor-cloud/infrastructure/docker
docker compose up -d
```

### Configuration Management
- Variables de entorno para secretos
- Configuración dinámica desde base de datos
- Hot-reload sin reinicio de servicios
- Schemas JSON para validación de contratos

---

## Documentación Técnica

El proyecto incluye documentación exhaustiva:
- Arquitectura detallada por componente
- Diagramas de flujo de datos
- Esquemas de base de datos con DDL completo
- API documentation (endpoints, payloads, responses)
- Deployment guides por plataforma
- Troubleshooting guides
- Performance tuning recommendations
- Security best practices

---

## URLs y Enlaces

**Cloud Platform**: https://mentormonitor-ai.com  
**Repositorio**: (Confidencial - disponible bajo NDA)  
**Demo Video**: (Disponible bajo solicitud)

---

## Período de Desarrollo

**Fecha**: 2024 - 2026 (en producción activa)  
**Estado**: Deployado en múltiples plantas industriales

---

## Testimonial / Impacto

Sistema actualmente operando en plantas textiles y frigoríficos, monitoreando producción y consumo energético 24/7. Ha permitido:
- Incremento de visibilidad de OEE del 0% a 100% en líneas sin instrumentación previa
- Reducción de costos de implementación en 70% vs sistemas SCADA tradicionales
- Detección automática de micro-paradas antes invisibles
- Análisis de consumo energético con granularidad de 5 minutos
- ROI alcanzado en <6 meses por planta
