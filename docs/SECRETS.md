# Configuración de Secretos y Variables de Entorno

> **IMPORTANTE:** Los archivos `.env` con valores reales **nunca se suben a git**.  
> Este documento explica exactamente qué archivos crear, dónde ubicarlos y qué valores colocar.

---

## Resumen de archivos necesarios

| Archivo | Entorno | Quién lo usa |
|---|---|---|
| `mentor-cloud/infrastructure/docker/.env` | Servidor Cloud | Docker Compose del cloud |
| `mentor-edge/infrastructure/docker/.env` | Jetson Orin (Edge) | Docker Compose del edge |
| `mentor-edge/infrastructure/docker/.env.raspberry-energy` | Raspberry Pi | Docker Compose de energy stack |

---

## 1. Cloud — `mentor-cloud/infrastructure/docker/.env`

### Crear el archivo

```bash
cp mentor-cloud/infrastructure/docker/.env.example mentor-cloud/infrastructure/docker/.env
```

### Valores actuales de producción (servidor 152.53.253.59)

```env
DATABASE_URL=postgres://mentor:SONEwDuYB8EbqQ7dVwccaWzEp66P7o@postgres-cloud:5432/mentor_cloud?sslmode=disable
JWT_SECRET=Dx6RCSw5W+Z4RbqJImEinz9yOJDodDnlxdyXbHrz0pn5R9eTP9D3qd8ZdiyRYgw3
CORS_ORIGINS=https://mentormonitor-ai.com,https://www.mentormonitor-ai.com,https://aplicacion.mentormonitor-ai.com,https://app.mentormonitor-ai.com,https://tablet.mentormonitor-ai.com

POSTGRES_DB=mentor_cloud
POSTGRES_USER=mentor
POSTGRES_PASSWORD=SONEwDuYB8EbqQ7dVwccaWzEp66P7o

EDGE_API_KEY=1a6972ea4616461a44296edd5a1e5b1d438457594c5e5458df82c36c06033279
INTERNAL_API_KEY=yoTRfyAW2NfQryJwFqjoxArAnfneBB7bZPDDidsR

IDENTITY_PORT=8081
INGEST_PORT=8082
CONFIG_PORT=8083
ANALYTICS_PORT=8084

ENCRYPTION_KEY=2f79f135c1a2093d4ea7d4abcf0886f54b79a17a1356145f7bb50d95c86d5500
LINEA_TEMPLATE_SQL=/app/12_linea_template.sql
```

> `postgres-cloud` es el nombre del servicio PostgreSQL dentro de la red Docker Compose; no es una IP externa.

### Generar valores seguros desde terminal

```bash
# Para JWT_SECRET, POSTGRES_PASSWORD, etc.:
openssl rand -base64 32
```

---

## 2. Edge / Jetson Orin — `mentor-edge/infrastructure/docker/.env`

### Crear el archivo

```bash
cp mentor-edge/infrastructure/docker/.env.example mentor-edge/infrastructure/docker/.env
```

### Variables a completar

```env
# Base de datos local en Jetson
POSTGRES_DB=mentor_edge
POSTGRES_USER=mentor
POSTGRES_PASSWORD=<PASSWORD_LOCAL>       # solo acceso local, puede ser simple

# Identificación del dispositivo
DEVICE_ID=jetson-orin-XXXX               # ej: jetson-orin-planta01-linea1
LINE_CODE=MENTOR_PLANTAXX_LX             # ej: MENTOR_PLANTA01_L1

# Cámara IP
CAMERA_URL=rtsp://usuario:pass@192.168.X.X:554/stream

# Procesamiento de video
FRAME_BACKEND=opencv                     # opciones: opencv | gstreamer

# Intervalo OEE en segundos
OEE_INTERVAL=300

# IDs de base de datos cloud (obtener del panel cloud tras registrar empresa/línea)
EMPRESA_ID=0
LINEA_ID=0

# Conexión al Cloud
CLOUD_URL=https://api.tu-dominio.com     # URL del cloud-gateway  
CLOUD_API_KEY=<EDGE_API_KEY>             # DEBE coincidir con EDGE_API_KEY del cloud

# Zona horaria
TZ=America/Lima
```

### Desplegar en el Jetson

```bash
# Copiar el .env al Jetson
scp mentor-edge/infrastructure/docker/.env usuario@<IP_JETSON>:~/mentor-edge/infrastructure/docker/.env

# Levantar servicios
ssh usuario@<IP_JETSON>
cd ~/mentor-edge/infrastructure/docker
docker compose up -d
```

---

## 3. Raspberry Pi (Energy Stack) — `mentor-edge/infrastructure/docker/.env.raspberry-energy`

### Crear el archivo

```bash
cp mentor-edge/infrastructure/docker/.env.raspberry-energy.example \
   mentor-edge/infrastructure/docker/.env.raspberry-energy
```

> Si no existe el `.example`, crear el archivo directamente con este contenido:

```env
# Puerto serie RS-485 (adaptador USB → Modbus RTU)
# Valores comunes: /dev/ttyUSB0  /dev/ttyUSB1  /dev/ttyAMA0
MODBUS_SERIAL_DEV=/dev/ttyUSB0

# Intervalo de lectura del medidor en milisegundos
METER_POLL_INTERVAL_MS=60000

# Zona horaria
TZ=America/Lima
```

### Copiar a la Raspberry Pi

```bash
scp mentor-edge/infrastructure/docker/.env.raspberry-energy \
    py@192.168.0.121:~/mentor-edge/infrastructure/docker/.env.raspberry-energy
```

### Configuración operacional del medidor (energy-sender)

El medidor MC60 se configura desde la **Web UI** de `energy-sender`, **no** desde el `.env`:

```
http://192.168.0.121:8086
```

O directamente en la base de datos:

```bash
docker exec -it rpi-energy-postgres \
  psql -U mentor mentor_energy -c "SELECT * FROM energy.config;"
```

| Clave | Descripción | Ejemplo |
|---|---|---|
| `energy_api_key` | API key para enviar datos al cloud | `5mfpkk...` |
| `device_id` | Identificador de este medidor | `rpi-planta-01` |
| `cloud_url` | URL del endpoint cloud | `https://api.tu-dominio.com` |
| `cfg_meter_unit_id` | Dirección Modbus del medidor MC60 | `1` |

---

## Flujo completo tras clonar el repositorio

```bash
# 1. Clonar
git clone https://github.com/alonsosss/Mentoredge.git
cd Mentoredge

# 2. Crear los .env (nunca están en git)
cp mentor-cloud/infrastructure/docker/.env.example \
   mentor-cloud/infrastructure/docker/.env
cp mentor-edge/infrastructure/docker/.env.example \
   mentor-edge/infrastructure/docker/.env

# 3. Editar cada .env con los valores reales
nano mentor-cloud/infrastructure/docker/.env
nano mentor-edge/infrastructure/docker/.env

# Para la RPi, crear manualmente (ver sección 3)

# 4. Desplegar
#   Cloud: ver docs/DEPLOYMENT.md
#   Edge:  rsync + docker compose en el Jetson
#   RPi:   scp + docker compose en la Raspberry
```

---

## Verificación rápida

```bash
# Cloud — verificar que el .env existe y tiene las variables clave
grep -E "^(JWT_SECRET|POSTGRES_PASSWORD|EDGE_API_KEY)=" \
  mentor-cloud/infrastructure/docker/.env

# Edge — verificar device ID y cloud URL
grep -E "^(DEVICE_ID|CLOUD_URL|CLOUD_API_KEY)=" \
  mentor-edge/infrastructure/docker/.env

# RPi — verificar puerto serie
grep "MODBUS_SERIAL_DEV" \
  mentor-edge/infrastructure/docker/.env.raspberry-energy
```

---

## Seguridad

- Los archivos `.env` están en `.gitignore` — nunca se tracked en git
- No compartir estos archivos por Slack/email; usar un gestor de secretos (Vault, Bitwarden, etc.)
- Rotar las claves si alguien deja el equipo
- Para producción: `POSTGRES_PASSWORD` y `JWT_SECRET` deben tener mínimo 32 caracteres aleatorios
