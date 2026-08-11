# Mentor Edge

Plataforma industrial de monitoreo OEE distribuida en dos capas: Edge (Jetson) y Cloud (Docker).

## Arquitectura General

| Capa | Servicios | Funcion |
|---|---|---|
| Edge | vision-event-detector, resiliencia, enviador, edge-config-service, ui-local | Deteccion en tiempo real, OEE local, buffer offline |
| Cloud | cloud-gateway, cloud-identity, cloud-ingest, cloud-config, cloud-analytics, cloud-integration, cloud-frontend | Gestion multi-tenant, dashboards, sincronizacion, integraciones |

---

## Cloud Deployment

### Requisitos

- Linux (Ubuntu 22.04+ recomendado)
- Docker 24+ y Docker Compose v2
- 4 GB RAM minimo (8 GB recomendado)
- 20 GB disco libre
- Puerto 8888 abierto (gateway)

### Instalacion

```bash
cd mentor-cloud/infrastructure/docker
cp .env.example .env
```

Editar `.env` con valores de produccion:

| Variable | Descripcion | Ejemplo |
|---|---|---|
| POSTGRES_PASSWORD | Password de PostgreSQL | (generado con openssl rand) |
| JWT_SECRET | Secret HMAC-SHA256 para JWT (min 32 chars) | (generado con openssl rand -base64 48) |
| EDGE_API_KEY | API key compartida con dispositivos edge | (generado con openssl rand -base64 32) |
| INTERNAL_API_KEY | Key para comunicacion entre servicios | (generado con openssl rand -base64 32) |
| CORS_ORIGINS | Dominios permitidos para CORS | https://monitor.mentoredge.io |
| DATABASE_URL | Connection string PostgreSQL | postgres://mentor:{PASSWORD}@postgres-cloud:5432/mentor_cloud?sslmode=disable |

### Deploy

```bash
cd mentor-cloud
chmod +x deploy.sh
./deploy.sh
```

El script:
1. Verifica que `.env` exista.
2. Construye las imagenes (`--no-cache`).
3. Levanta los servicios.
4. Valida health de los 5 servicios backend.

### Verificacion

```bash
# Health individual
curl http://localhost:8888/health

# Health agregado (todos los servicios)
curl http://localhost:8888/api/health

# Logs
docker compose -f infrastructure/docker/docker-compose.cloud.yml logs --tail=50
```

### Servicios y Puertos

| Servicio | Puerto interno | Puerto expuesto | Descripcion |
|---|---|---|---|
| postgres-cloud | 5432 | 5434 | Base de datos compartida |
| cloud-identity | 8081 | - | Autenticacion y usuarios |
| cloud-ingest | 8082 | - | Ingesta de datos IoT |
| cloud-config | 8083 | - | Configuracion industrial |
| cloud-analytics | 8084 | - | Dashboards y reportes |
| cloud-integration | 8085 | - | API para terceros |
| cloud-frontend | 80 | - | SPA Vue 3 |
| cloud-gateway | 8888 | 8888 | Punto de entrada unico |

### Base de Datos Cloud

Schemas: `identity`, `config`, `ingest`, `analytics`, `gateway`, `integration`.

Backup:
```bash
docker exec mentor-cloud-postgres pg_dump -U mentor mentor_cloud > backup_cloud.sql
```

Restore:
```bash
docker exec -i mentor-cloud-postgres psql -U mentor mentor_cloud < backup_cloud.sql
```

### Conectar Dispositivo Edge

1. Registrar el dispositivo en cloud-config o via gateway provisioning.
2. Configurar en el Jetson las variables:
   - `CLOUD_GATEWAY_URL=https://<dominio>:8888`
   - `EDGE_API_KEY=<misma key que en .env cloud>`
3. El dispositivo se conectara via SSE y recibira la config filtrada.

---

## Edge Deployment (Jetson)

## Configuración

### ROI (Region of Interest)

Desde la UI Local, sección Configuración:
- Definir coordenadas [x, y, width, height]
- Ajustar visualmente sobre preview de cámara

### Umbrales de Detección

- **edge**: Sensibilidad a bordes (0.0 - 1.0)
- **color**: Sensibilidad a cambio de color (0.0 - 1.0)
- **flow**: Sensibilidad a flujo óptico (0.0 - 1.0)

### FSM (Máquina de Estados)

- **n_frames**: Frames de confirmación antes de evento (1-30)
- **cooldown**: Frames de enfriamiento anti-rebote (0-60)

## Calibración

Proceso automático para aprender color de referencia:

1. Acceder a UI Local → Configuración
2. Presionar "Iniciar Calibración"
3. Sistema captura 30 frames de referencia
4. Histograma base actualizado automáticamente

## Monitoreo

### Dashboard

- Eventos pendientes en buffer
- Estado de servicios
- Última sincronización

### Health Endpoints

```bash
curl http://localhost:8004/health
curl http://localhost:8002/health/buffer
```

## Gestión de Logs

Ver logs de servicio específico:
```bash
docker-compose logs -f vision-event-detector
docker-compose logs -f resiliencia
docker-compose logs -f enviador
```

Ver logs de todos los servicios:
```bash
docker-compose logs -f
```

## Base de Datos

### Tablas Principales

- **events_buffer**: Buffer de eventos con estado de sincronización
- **line_config**: Configuración por dispositivo
- **health_logs**: Histórico de salud de servicios
- **calibration_history**: Histórico de calibraciones

### Backup

```bash
docker exec mentor-postgres pg_dump -U postgres mentor_edge > backup.sql
```

### Restore

```bash
docker exec -i mentor-postgres psql -U postgres mentor_edge < backup.sql
```

## Comandos Útiles

Reiniciar servicios:
```bash
docker-compose restart
```

Detener servicios:
```bash
docker-compose down
```

Reconstruir y reiniciar:
```bash
docker-compose up -d --build
```

Ver estado de contenedores:
```bash
docker-compose ps
```

## Troubleshooting

### Detector no conecta a cámara

Verificar URL RTSP:
```bash
ffplay rtsp://user:pass@camera-ip:554/stream
```

### Eventos no sincronizan

Verificar conectividad cloud:
```bash
docker-compose logs enviador
```

Revisar eventos pendientes:
```bash
docker exec mentor-postgres psql -U postgres -d mentor_edge -c "SELECT COUNT(*) FROM events_buffer WHERE synced=false;"
```

### Configuración no actualiza

Verificar logs de config-service:
```bash
docker-compose logs edge-config-service
```

Verificar versión de configuración:
```bash
curl http://localhost:8004/config/version?device_id=jetson-default
```

## Arquitectura Hexagonal

### Domain Layer

Lógica de negocio pura:
- ROI Manager
- Signal Extractors (Edge, Color, Flow, Beige)
- Fusion Engine
- Event FSM
- Calibration
- Watchdog

### Ports

Interfaces abstractas:
- FrameInput
- ConfigPort
- EventOutput

### Adapters

Implementaciones concretas:
- OpenCVAdapter
- ConfigClient
- HTTPEventAdapter

## API Reference

### Config Service

**GET /config**
```json
{
  "roi": [120, 60, 320, 200],
  "thresholds": {"edge": 0.4, "color": 0.6, "flow": 0.5},
  "fsm": {"n_frames": 3, "cooldown": 8},
  "mode": "textil",
  "config_version": 12
}
```

**PUT /config**
```json
{
  "roi": [100, 50, 300, 180],
  "thresholds": {"edge": 0.5}
}
```

### Resiliencia

**POST /events**
```json
{
  "event_id": "uuid",
  "device_id": "jetson-01",
  "event_type": "CORTE",
  "timestamp": "2026-02-21T20:00:00Z",
  "payload": {
    "confidence": 0.92,
    "roi_id": "A1"
  }
}
```

**GET /health/buffer**
```json
{
  "pending_events": 240,
  "last_insert": "2026-02-21T19:58:00Z"
}
```

## Performance

- Throughput: 10-15 FPS en Jetson Nano
- Latencia detector→buffer: <50ms
- Capacidad offline: >10 horas sin sincronización
- Eventos/segundo: ~1-2 en producción textil

## Seguridad

- PostgreSQL con autenticación
- No exposición de servicios internos
- Validación de schemas JSON
- Sanitización de parámetros
- Rate limiting en endpoints

## Licencia

Proprietary - Patent Pending

(c) 2026 Mentor Edge. Todos los derechos reservados.

## Soporte

Para soporte técnico contactar a: support@mentoredge.com
