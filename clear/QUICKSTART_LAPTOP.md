# 🚀 Guía de Inicio Rápido - Desarrollo en Laptop

## ✅ Archivos de configuración creados

Los siguientes archivos `.env` ya están configurados con los **valores reales de producción**:

- ✅ `mentor-cloud/infrastructure/docker/.env` → Cloud services (mismos secrets que el servidor)
- ✅ `mentor-cloud/Frontend/template_mentor/.env` → Frontend apuntando a localhost:8888

## 📋 Pre-requisitos

- Docker Desktop instalado y corriendo
- Node.js 18+ y npm
- Puerto 8888 disponible para cloud-gateway
- Puerto 5434 disponible para PostgreSQL
- Puerto 3000 o 5173 disponible para el frontend

## 🐳 Levantar el Cloud en tu laptop

### Paso 1: Iniciar servicios del Cloud

```powershell
cd c:\Users\User007\Desktop\Mentoredge\mentor-cloud\infrastructure\docker
docker compose -f docker-compose.cloud.yml up -d
```

Esto levanta:
- PostgreSQL en puerto **5434**
- cloud-gateway en puerto **8888**
- cloud-identity, cloud-ingest, cloud-config, cloud-analytics, cloud-integration
- energy-ingest en puerto **8086**

### Paso 2: Verificar que los servicios están corriendo

```powershell
docker compose -f docker-compose.cloud.yml ps
```

Todos los servicios deben estar "Up" y "healthy".

### Paso 3: Verificar el gateway

Abre en tu navegador:
```
http://localhost:8888/health
```

Deberías ver: `{"status": "ok"}`

## 🎨 Levantar el Frontend

En una nueva terminal:

```powershell
cd c:\Users\User007\Desktop\Mentoredge\mentor-cloud\Frontend\template_mentor

# Instalar dependencias (solo la primera vez)
npm install

# Iniciar servidor de desarrollo
npm run dev
```

El frontend estará en: **http://localhost:5173**

## 🔌 Conectar el Jetson al Cloud local

En el Jetson, edita el archivo `.env` para que apunte a tu laptop:

```bash
ssh usuario@<IP_JETSON>
nano ~/mentor-edge/infrastructure/docker/.env
```

Cambia esta línea:
```env
CLOUD_URL=http://<IP_DE_TU_LAPTOP>:8888
```

Por ejemplo, si tu laptop tiene IP `192.168.1.100`:
```env
CLOUD_URL=http://192.168.1.100:8888
```

Reinicia el enviador en el Jetson:
```bash
docker restart enviador
```

## 🧪 Verificar la conexión

### En el Cloud (laptop)

Ver logs del gateway:
```powershell
docker logs -f mentor-cloud-gateway
```

Deberías ver logs de conexión del Jetson.

### En la base de datos

```powershell
docker exec -it mentor-cloud-postgres psql -U mentor mentor_cloud

# Ver dispositivos conectados
SELECT * FROM gateway.device_registry;

# Ver últimos snapshots OEE recibidos
SELECT * FROM ingest.oee_snapshots ORDER BY received_at DESC LIMIT 10;
```

## 📱 Tablet App (opcional)

Si quieres probar la tablet app:

```powershell
cd c:\Users\User007\Desktop\Mentoredge\mentor-apps\mentor-tablet-app
npm install
npm run dev
```

La app estará en: **http://localhost:5173**

En la app, ve a **Settings** y configura:
- **Edge URL**: `http://<IP_JETSON>:8005`
- **Cloud URL**: `http://localhost:8888`

## 🛑 Detener todo

### Cloud
```powershell
cd c:\Users\User007\Desktop\Mentoredge\mentor-cloud\infrastructure\docker
docker compose -f docker-compose.cloud.yml down
```

### Frontend/Tablet (Ctrl+C en las terminales)

## 🔧 Troubleshooting

### El gateway no inicia
- Verifica que el puerto 8888 esté libre: `netstat -ano | findstr :8888`
- Revisa logs: `docker logs mentor-cloud-gateway`

### El frontend no conecta con el backend
- Verifica que el gateway esté corriendo en http://localhost:8888
- Abre las DevTools del navegador y revisa la consola (F12)

### El Jetson no envía datos
- Verifica la IP de tu laptop: `ipconfig` (buscar IPv4 de tu red local)
- Asegúrate de que el firewall de Windows no bloquee el puerto 8888
- Verifica en el Jetson: `docker logs enviador`

## 📚 Siguientes pasos

1. **Login en el Frontend**: Necesitas crear un usuario admin desde la BD o usar credenciales existentes
2. **Registrar dispositivos**: En el panel cloud, registra empresas, plantas y líneas
3. **Ver datos en tiempo real**: Los snapshots OEE del Jetson aparecerán en el dashboard

## 🔑 Credenciales importantes

- **PostgreSQL Cloud**: 
  - Usuario: `mentor`
  - Password: `SONEwDuYB8EbqQ7dVwccaWzEp66P7o`
  - Puerto: `5434`
  
- **EDGE_API_KEY** (para que el Jetson se autentique): 
  ```
  5mfpkkTsSMcOum2c8fPRYvFbA1fSm2tdtUK2up30
  ```

- **Cloud Gateway**: http://localhost:8888
