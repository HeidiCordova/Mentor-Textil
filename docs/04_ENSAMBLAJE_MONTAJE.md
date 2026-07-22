# 4. MÉTODOS DE FABRICACIÓN, ENSAMBLAJE Y MONTAJE

## 4.1 Descripción General del Sistema Físico

El sistema MENTOR EDGE no requiere fabricación de componentes propios. Se basa en **ensamblaje e integración** de componentes comerciales (COTS — Commercial Off-The-Shelf) con software propio.

### Componentes del kit por línea de producción

| # | Componente | Cantidad | Origen |
|---|-----------|----------|--------|
| 1 | NVIDIA Jetson Orin Nano/NX DevKit | 1 | Componente comercial (NVIDIA) |
| 2 | Cámara IP Industrial (PoE) | 1 por línea | Componente comercial |
| 3 | SSD NVMe 256 GB | 1 | Componente comercial |
| 4 | Soporte articulado para cámara | 1 por línea | Componente comercial o fabricado |
| 5 | Cable Ethernet Cat6 | Según distancia | Componente comercial |
| 6 | Switch PoE (8 puertos) | 1 por planta | Componente comercial |
| 7 | Gabinete industrial (IP54) | 1 | Componente comercial o fabricado |
| 8 | Fuente de alimentación 12V/5A | 1 | Componente comercial |
| 9 | MicroSD con imagen JetPack | 1 | Preparada en taller |

---

## 4.2 Proceso de Ensamblaje del Dispositivo Edge

### Paso 1: Preparación del Jetson

```
1.1  Desempacar NVIDIA Jetson Orin DevKit
1.2  Insertar SSD NVMe en la ranura M.2 del carrier board
1.3  Flashear JetPack 6.0 en el SSD:
     - Conectar Jetson a PC de desarrollo por USB-C
     - Ejecutar NVIDIA SDK Manager
     - Seleccionar JetPack 6.0 + componentes CUDA
     - Flash al SSD NVMe
1.4  Primer arranque:
     - Configurar usuario: orin / [contraseña]
     - Configurar red Ethernet (IP estática o DHCP)
     - Verificar GPU: nvidia-smi
```

### Paso 2: Instalación del Software Edge

```
2.1  Instalar Docker y Docker Compose:
     sudo apt-get update
     sudo apt-get install docker.io docker-compose-plugin
     
2.2  Copiar docker-compose.jetson-orin.yml al Jetson
2.3  Copiar imágenes Docker pre-construidas o construir en sitio
2.4  Levantar servicios:
     docker compose -f docker-compose.jetson-orin.yml up -d
     
2.5  Verificar servicios:
     docker ps  (8 contenedores activos)
     curl localhost:8005/health  (edge-gateway OK)
```

### Paso 3: Ensamblaje en Gabinete (opcional para entornos industriales)

```
3.1  Montar Jetson dentro de gabinete industrial IP54
3.2  Conectar fuente de alimentación 12V al carrier board
3.3  Pasar cable Ethernet por prensaestopa del gabinete
3.4  Instalar ventilación pasiva o forzada según temperatura ambiente
3.5  Etiquetar gabinete con:
     - ID del dispositivo
     - IP asignada
     - Líneas que monitorea
```

---

## 4.3 Montaje de la Cámara

### Ubicación recomendada

```
                     ┌─── Cámara IP (montada en estructura)
                     │    Ángulo: 45°-90° respecto a la línea
                     │    Distancia: 0.5m - 2m del producto
                     ▼
    ════════════════════════════════════
    ▶▶▶  Línea de producción  ▶▶▶▶▶▶▶    (dirección del producto)
    ════════════════════════════════════
```

### Criterios de instalación

| Criterio | Especificación |
|----------|---------------|
| **Ángulo** | 45° a 90° respecto al plano de la línea |
| **Distancia** | 0.5m a 2m del punto de detección |
| **Iluminación** | Evitar contraluz directo. Preferir iluminación lateral o difusa |
| **Vibración** | Montar en estructura rígida, no en partes móviles de la máquina |
| **Protección** | IP66 para ambientes con polvo o humedad |
| **Alimentación** | PoE (Power over Ethernet) — un solo cable para datos y energía |

### Procedimiento de montaje

```
M1. Identificar punto óptimo de visión sobre la línea
M2. Instalar soporte articulado en estructura superior o lateral
M3. Fijar cámara al soporte con tornillos de seguridad
M4. Conectar cable Cat6 desde cámara hasta switch PoE
M5. Verificar ángulo y enfoque en vista previa (UI Local)
M6. Ajustar apertura focal para cubrir ancho de línea completo
M7. Fijar posición final del soporte articulado
```

---

## 4.4 Conexionado de Red

### Diagrama de conexión

```
┌──────────┐     Cat6      ┌───────────┐     Cat6      ┌──────────┐
│ Cámara 1 │────────────── │           │ ──────────────│          │
│ (PoE)    │               │  Switch   │               │  Jetson  │
└──────────┘               │  PoE      │               │  Orin    │
┌──────────┐     Cat6      │  8 ports  │               │          │
│ Cámara 2 │────────────── │           │               │          │
│ (PoE)    │               │           │               └──────────┘
└──────────┘               │           │     Cat6      ┌──────────┐
                           │           │ ──────────────│  Router  │
                           └───────────┘               │ Internet │
                                                       └──────────┘
```

### Configuración de red

| Dispositivo | IP Sugerida | Puerto |
|------------|-------------|--------|
| Jetson Orin | 192.168.100.31 | - |
| Cámara Línea 1 | 192.168.100.101 | RTSP :554 |
| Cámara Línea 2 | 192.168.100.102 | RTSP :554 |
| Switch PoE | 192.168.100.1 | - |
| Router/Gateway | 192.168.100.254 | - |

---

## 4.5 Configuración del Servidor Cloud

### Requisitos del servidor

| Requisito | Especificación |
|-----------|---------------|
| OS | Ubuntu 22.04 LTS o superior |
| CPU | 4 vCPU mínimo |
| RAM | 8 GB recomendado |
| Disco | 50 GB SSD |
| Red | Puerto 8888 abierto (gateway) |
| Docker | 24+ con Docker Compose v2 |

### Procedimiento de instalación

```
C1. Provisionar VPS con Ubuntu 22.04
C2. Instalar Docker y Docker Compose
C3. Clonar repositorio mentor-cloud
C4. Copiar y configurar archivo .env:
    - POSTGRES_PASSWORD (generado con openssl rand)
    - JWT_SECRET (min 32 caracteres)
    - EDGE_API_KEY (compartida con dispositivos edge)
    - INTERNAL_API_KEY (inter-servicios)
    - CORS_ORIGINS (dominio del frontend)
C5. Ejecutar script de deploy:
    ./deploy.sh
C6. Verificar salud de servicios:
    curl http://localhost:8888/api/health
```

---

## 4.6 Registro de Dispositivo Edge en la Nube

Una vez instalados tanto el Edge como el Cloud:

```
R1. Acceder al dashboard cloud (http://servidor:8888)
R2. Ir a Configuración → Dispositivos → Nuevo Dispositivo
R3. Asignar:
    - Nombre del dispositivo
    - Planta
    - Línea(s) asociada(s)
R4. Obtener el EDGE_API_KEY y configurar en el Jetson:
    - Editar variables de entorno en docker-compose
    - CLOUD_GATEWAY_URL=https://<servidor>:8888
    - EDGE_API_KEY=<key generada>
R5. Reiniciar servicios en el Jetson
R6. Verificar conexión SSE en logs:
    docker logs docker-edge-gateway-1 | grep "SSE connected"
```

---

## 4.7 Calibración Inicial

Después del montaje físico y registro:

```
CAL1. Acceder a UI Local (http://192.168.100.31:8080)
CAL2. Ir a sección Configuración
CAL3. Definir ROI (Región de Interés):
      - Dibujar rectángulo sobre la zona de paso de producto
CAL4. Iniciar Calibración Automática:
      - El sistema captura 30 frames de referencia
      - Genera histograma base del fondo
CAL5. Ajustar umbrales si es necesario:
      - Edge: sensibilidad a bordes
      - Color: sensibilidad a cambio de color
      - Flow: sensibilidad a movimiento
CAL6. Verificar detección con producto real pasando por la línea
CAL7. Ajustar cooldown y n_frames según velocidad de la línea
```

---

## 4.8 Lista de Verificación Post-Instalación

| # | Verificación | Estado |
|---|-------------|--------|
| 1 | Jetson enciende correctamente | ☐ |
| 2 | Docker reporta 8 contenedores activos | ☐ |
| 3 | Cámara(s) accesible(s) por RTSP | ☐ |
| 4 | UI Local accesible en :8080 | ☐ |
| 5 | Vista previa de cámara muestra imagen correcta | ☐ |
| 6 | ROI configurada sobre zona de detección | ☐ |
| 7 | Calibración ejecutada exitosamente | ☐ |
| 8 | Detección de productos funciona (contar 10 productos) | ☐ |
| 9 | Conexión SSE con cloud establecida | ☐ |
| 10 | Datos aparecen en dashboard cloud | ☐ |
| 11 | Tablet app conecta al Jetson | ☐ |
| 12 | Gabinete cerrado y etiquetado | ☐ |

---

## 4.9 Herramientas Necesarias para Instalación

| Herramienta | Uso |
|-------------|-----|
| Laptop con SSH | Configuración inicial del Jetson |
| Destornillador Phillips | Montaje de SSD y soporte de cámara |
| Crimpadora RJ45 | Preparación de cables Ethernet |
| Tester de red | Verificación de conectividad |
| Escalera o plataforma | Montaje de cámara en altura |
| Nivel | Alineación de cámara |
| Bridas plásticas | Gestión de cables |
| Etiquetadora | Identificación de equipos |

---

## 4.10 Tiempo Estimado de Instalación

| Actividad | Tiempo |
|-----------|--------|
| Preparación del Jetson (flash + software) | 2 horas |
| Montaje de cámara(s) y cableado | 1-2 horas |
| Conexionado de red | 30 minutos |
| Configuración de software y registro en cloud | 1 hora |
| Calibración y ajuste fino | 1-2 horas |
| Pruebas y verificación | 1 hora |
| **Total por línea** | **6-8 horas** |

Para instalaciones de múltiples líneas en la misma planta, el tiempo se reduce significativamente ya que el Jetson y la red se comparten.
