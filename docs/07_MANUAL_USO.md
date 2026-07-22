# 7. MANUAL DE USO — Instructivo para Usuarios

## 7.1 Perfiles de Usuario

| Perfil | Acceso | Funciones principales |
|--------|--------|----------------------|
| **Operador de línea** | Tablet App | Registrar paradas, ver producción actual, cambiar producto |
| **Supervisor de planta** | UI Local + Cloud Dashboard | Monitorear OEE, revisar paradas, gestionar líneas |
| **Administrador** | Cloud Dashboard completo | Configurar plantas, usuarios, productos, dispositivos |
| **Técnico de mantenimiento** | UI Local | Calibrar cámaras, ajustar parámetros, diagnosticar |

---

## 7.2 Acceso al Sistema

### 7.2.1 Dashboard Cloud (Supervisores y Administradores)

```
URL: http://[IP-servidor]:8888
Ejemplo: http://152.53.253.59:8888
```

1. Abrir navegador web (Chrome, Firefox o Edge recomendados)
2. Ingresar URL del servidor
3. Introducir credenciales (usuario y contraseña)
4. El sistema redirige al dashboard principal

### 7.2.2 UI Local (Supervisores y Técnicos)

```
URL: http://[IP-jetson]:8080
Ejemplo: http://192.168.100.31:8080
```

Accesible desde cualquier dispositivo conectado a la red de planta. No requiere internet.

### 7.2.3 Tablet App (Operadores)

```
URL: http://[IP-jetson]:8090
Ejemplo: http://192.168.100.31:8090
```

Accesible desde tablets o smartphones conectados a la red de planta.

---

## 7.3 Dashboard Cloud — Guía de Uso

### 7.3.1 Pantalla Principal — OEE en Tiempo Real

Al ingresar al sistema, se muestra el dashboard principal con:

| Elemento | Descripción |
|---------|-------------|
| **Indicador OEE** | Porcentaje de eficiencia general (Disponibilidad × Rendimiento × Calidad) |
| **Disponibilidad** | Porcentaje de tiempo que la línea estuvo operando vs tiempo planificado |
| **Rendimiento** | Velocidad real vs velocidad nominal del producto actual |
| **Producción del turno** | Unidades producidas en el turno actual |
| **Estado de líneas** | Indicador visual de cada línea (verde=operando, rojo=parada) |

### 7.3.2 Gestión de Paradas

**Ver paradas:**
1. Ir a menú lateral → **Paradas**
2. Seleccionar rango de fechas
3. Seleccionar planta y línea
4. Se muestra tabla con todas las paradas registradas

**Información de cada parada:**
- Hora de inicio y fin
- Duración
- Categoría (programada / no programada)
- Subcategoría (ejemplo: cambio de producto, falla mecánica, etc.)

### 7.3.3 Reportes y Análisis

**Análisis Pareto:**
1. Menú → **Dashboard** → Sección Pareto
2. Seleccionar periodo y línea
3. Se muestra gráfico de barras ordenado por frecuencia/duración de paradas
4. Identificar las principales causas de parada

**Exportar a Excel:**
1. En cualquier vista de datos, buscar botón "Exportar"
2. Se descarga archivo Excel (.xlsx) con los datos filtrados

### 7.3.4 Configuración (solo Administradores)

**Gestionar productos:**
1. Menú → **Configuración** → **Productos**
2. Agregar nuevo producto: código, nombre, velocidad nominal
3. Asignar producto a línea(s)

**Gestionar turnos:**
1. Menú → **Configuración** → **Turnos**
2. Definir nombre, hora inicio, hora fin
3. Los turnos se sincronizan automáticamente a los dispositivos edge

**Gestionar usuarios:**
1. Menú → **Configuración** → **Usuarios**
2. Crear usuario con: nombre, email, rol (operador/supervisor/admin)
3. El usuario recibe credenciales para acceso al dashboard

**Gestionar dispositivos:**
1. Menú → **Configuración** → **Dispositivos**
2. Ver estado de conexión de cada Jetson
3. Registrar nuevo dispositivo con su API Key

---

## 7.4 UI Local — Guía de Uso

### 7.4.1 Dashboard Local

Muestra en tiempo real:
- Estado de cada servicio (vision, resiliencia, enviador, gateway)
- Eventos pendientes en buffer
- Última sincronización con la nube
- Producción acumulada del turno

### 7.4.2 Configuración de Cámara y ROI

**Definir Región de Interés (ROI):**

1. Ir a **Configuración**
2. Seleccionar la línea a configurar
3. En la vista de cámara, dibujar un rectángulo sobre la zona donde pasan los productos
4. El rectángulo debe cubrir el ancho completo de la zona de detección
5. Guardar configuración

```
Vista de la cámara:
┌─────────────────────────────────┐
│                                 │
│     ┌──────────────────┐        │
│     │  ROI (zona de    │        │
│     │  detección)      │        │
│     └──────────────────┘        │
│  ═══════════════════════════    │  ← Línea de producción
│                                 │
└─────────────────────────────────┘
```

### 7.4.3 Calibración

**Procedimiento de calibración automática:**

1. Ir a **Configuración** → seleccionar línea
2. Asegurarse de que **no hay producto** pasando por la línea
3. Presionar **"Iniciar Calibración"**
4. Esperar ~3 segundos (el sistema captura 30 frames de referencia)
5. Mensaje de confirmación: "Calibración completada"
6. Verificar pasando un producto: debe detectarse correctamente

> **IMPORTANTE:** Calibrar siempre con la línea vacía (sin producto). La calibración aprende cómo se ve el "fondo" para luego detectar cambios cuando pasa un producto.

### 7.4.4 Ajuste de Parámetros

| Parámetro | Rango | Qué hace | Cuándo ajustar |
|-----------|-------|----------|----------------|
| **Umbral Edge** | 0.0 - 1.0 | Sensibilidad a bordes del producto | Si no detecta productos con bordes difusos |
| **Umbral Color** | 0.0 - 1.0 | Sensibilidad a cambio de color | Si la iluminación varía mucho |
| **Umbral Flow** | 0.0 - 1.0 | Sensibilidad al movimiento | Si la línea es muy lenta o rápida |
| **n_frames** | 1 - 30 | Frames de confirmación antes de emitir evento | Aumentar si hay falsos positivos |
| **cooldown** | 0 - 60 | Frames de anti-rebote después de detectar | Aumentar si cuenta doble |

**Guía rápida de ajuste:**
- Muchos **falsos positivos** (cuenta de más) → Aumentar `n_frames` y `cooldown`
- **No detecta** productos → Reducir umbrales (más sensible)
- Cuenta **doble** el mismo producto → Aumentar `cooldown`
- Detecta **muy tarde** → Reducir `n_frames`

### 7.4.5 Diagnóstico de Estado

La sección **Estado** muestra:

| Indicador | Verde | Amarillo | Rojo |
|-----------|-------|----------|------|
| Vision Detector | Procesando video | Reconectando cámara | Sin conexión a cámara |
| Resiliencia | Buffer operativo | Más de 1000 eventos pendientes | BD local caída |
| Enviador | Sincronizando | Cloud inalcanzable (reintentando) | Error persistente |
| Edge Gateway | Conectado a cloud | Reconectando SSE | Sin conexión |
| Config Service | Configuración activa | Versión desactualizada | Servicio caído |

---

## 7.5 Tablet App — Guía de Uso para Operadores

### 7.5.1 Pantalla principal

La tablet muestra una interfaz simplificada con:
- **Producción del turno:** Contador de unidades producidas
- **OEE actual:** Indicador visual de eficiencia
- **Estado de línea:** Operando / En parada
- **Producto actual:** Nombre y código del producto en producción

### 7.5.2 Registrar una parada

Cuando la línea se detiene:

1. La app detecta automáticamente la parada
2. Aparece prompt para **categorizar la parada**
3. Seleccionar tipo: **Programada** o **No programada**
4. Seleccionar subcategoría:
   - Programada: Cambio de producto, Mantenimiento, Descanso, Limpieza
   - No programada: Falla mecánica, Falla eléctrica, Falta de material, etc.
5. Confirmar
6. La parada queda registrada con hora de inicio, categoría y operador

### 7.5.3 Cambio de producto

1. Seleccionar icono de **cambio de producto**
2. Elegir el nuevo producto de la lista (sincronizada desde la nube)
3. Confirmar
4. Los contadores se reinician para el nuevo producto
5. La velocidad nominal se actualiza automáticamente

### 7.5.4 Modo offline

Si se pierde la conexión con el Jetson:
- La app muestra indicador de "Sin conexión"
- Los datos se almacenan localmente en la tablet
- Al reconectarse, los datos se sincronizan automáticamente

---

## 7.6 Procedimientos Comunes

### 7.6.1 Inicio de turno

1. Verificar que el dashboard cloud muestra datos actualizados
2. En tablet, verificar producto correcto seleccionado
3. Verificar que los contadores están en cero (o revisar producción acumulada)

### 7.6.2 Fin de turno

1. Revisar producción total del turno en dashboard
2. Verificar que todas las paradas están categorizadas
3. Exportar reporte del turno si es necesario

### 7.6.3 Cambio de producto en línea

1. Registrar parada por "Cambio de producto" en tablet
2. Realizar el cambio físico en la línea
3. Seleccionar nuevo producto en la tablet
4. Si es necesario, recalibrar la detección desde UI Local
5. Reiniciar producción

### 7.6.4 Reconexión de cámara

Si la cámara pierde conexión:

1. Verificar cable Ethernet (cámara → switch)
2. Verificar alimentación PoE (LED del switch encendido)
3. Si persiste: reiniciar la cámara desconectando y reconectando el cable
4. El sistema reconecta automáticamente en ~10 segundos
5. Si no reconecta: ir a UI Local → Estado → verificar URL RTSP

---

## 7.7 Resolución de Problemas

| Problema | Causa probable | Solución |
|---------|---------------|---------|
| Dashboard no carga | Sin internet o servidor cloud caído | Verificar internet. Usar UI Local mientras tanto |
| No se detectan productos | ROI mal configurada o sin calibración | Recalibrar. Verificar ROI cubre zona de paso |
| Cuenta de más (falsos positivos) | Umbrales muy sensibles | Aumentar n_frames y cooldown |
| No cuenta (falsos negativos) | Umbrales muy altos | Reducir umbrales. Recalibrar |
| "Sin conexión a cloud" | Internet caído en planta | El sistema sigue operando offline. Se sincroniza al reconectar |
| Tablet no conecta | No está en red de planta | Verificar WiFi del tablet conectado a red correcta |
| Imagen de cámara congelada | Cámara desconectada o RTSP caído | Revisar cableado. Reiniciar cámara |
| Datos no aparecen en cloud | Enviador en retry | Verificar estado en UI Local → Estado. Datos llegarán al reconectar |

---

## 7.8 Contacto de Soporte

| Tipo de soporte | Canal | Tiempo de respuesta |
|----------------|-------|-------------------|
| Soporte técnico nivel 1 | [COMPLETAR: email/teléfono] | [COMPLETAR] |
| Soporte técnico nivel 2 | [COMPLETAR: email/teléfono] | [COMPLETAR] |
| Emergencias (sistema caído) | [COMPLETAR: teléfono directo] | [COMPLETAR] |

---

## 7.9 Glosario

| Término | Definición |
|---------|-----------|
| **OEE** | Overall Equipment Effectiveness — Eficiencia General de los Equipos. Fórmula: Disponibilidad × Rendimiento × Calidad |
| **ROI** | Region of Interest — Zona de la imagen de la cámara donde se realiza la detección |
| **FSM** | Finite State Machine — Máquina de estados que confirma la detección de un evento |
| **Edge** | Dispositivo local (Jetson) que procesa en planta sin depender de internet |
| **Cloud** | Servidor en la nube que centraliza datos y dashboards |
| **SSE** | Server-Sent Events — Protocolo para enviar actualizaciones en tiempo real del cloud al edge |
| **Parada programada** | Parada planificada: cambio de producto, mantenimiento, descanso |
| **Parada no programada** | Parada inesperada: falla mecánica, eléctrica, falta de material |
| **Calibración** | Proceso de aprendizaje del fondo/referencia para la detección |
| **Velocidad nominal** | Velocidad teórica máxima de producción por producto (unidades/tiempo) |
| **Turno** | Periodo de trabajo con hora de inicio y fin definidas |
| **Sync** | Sincronización de datos entre el dispositivo edge y la nube |
| **Buffer** | Almacenamiento temporal local de eventos cuando no hay conexión a internet |
