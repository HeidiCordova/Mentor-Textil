# Mentor Edge Textil

Sistema Edge industrial para detección de eventos de corte textil mediante visión artificial en dispositivos Jetson.

## Arquitectura

Sistema distribuido offline-first con sincronización cloud:

```
DeepStream (host) → vision-event-detector → resiliencia → PostgreSQL local → enviador → Cloud
```

## Microservicios

### vision-event-detector (Python)
Motor de visión con arquitectura hexagonal. Procesa video RTSP y genera eventos industriales robustos.

### resiliencia (Go)
Buffer confiable que persiste eventos localmente con deduplicación y orden temporal.

### enviador (Go)
Sincronizador Edge-Cloud con retry exponencial y confirmación ACK.

### edge-config-service (Go)
Servicio de configuración dinámica con versionado y validación.

### ui-local (Vue)
Interfaz web para monitoreo, configuración de ROI y visualización de estado.

## Despliegue

```bash
cd infrastructure/docker
docker-compose up -d
```

## Puertos

- vision-event-detector: 8001
- resiliencia: 8002
- enviador: 8003
- edge-config-service: 8004
- ui-local: 8080
- postgres: 5432

## Requisitos

- NVIDIA Jetson con GPU
- DeepStream 6.0+
- Docker 20.10+
- PostgreSQL 14+

## Licencia

Proprietary - Patent Pending
