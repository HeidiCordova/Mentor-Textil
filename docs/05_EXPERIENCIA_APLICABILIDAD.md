# 5. EXPERIENCIA DE FUNCIONAMIENTO Y APLICABILIDAD

## 5.1 Implementación Actual — Art Atlas S.A.

### Datos generales

| Aspecto | Detalle |
|---------|---------|
| **Cliente** | Art Atlas S.A. |
| **Sector** | Industria textil |
| **Ubicación** | [COMPLETAR: ciudad, país] |
| **Líneas monitoreadas** | 4 líneas de producción (Maquina01, linea3, linea4, linea1) |
| **Planta ID** | 14 (en el sistema Mentor) |
| **Dispositivo Edge** | NVIDIA Jetson Orin, IP 192.168.100.31 |
| **Servidor Cloud** | VPS 152.53.253.59, puerto 8888 |

### Arquitectura desplegada

```
Art Atlas — Planta 14
├── Jetson Orin (192.168.100.31)
│   ├── Línea 1 (local) → Línea 14 (cloud) — linea1
│   ├── Línea 2 (local) → Línea 13 (cloud) — linea4
│   ├── Línea 3 (local) → Línea 12 (cloud) — linea3
│   └── Línea 4 (local) → Línea 11 (cloud) — Maquina01
│
├── Cámaras IP (1 por línea)
│   └── RTSP sobre red local
│
└── Cloud (152.53.253.59)
    ├── mentor_cloud (BD servicios centrales)
    └── mentor_planta_14 (BD operativa)
        ├── linea_11 (Maquina01)
        ├── linea_12 (linea3)
        ├── linea_13 (linea4)
        └── linea_14 (linea1)
```

### Productos configurados en producción

| ID | Código | Producto | Estado |
|----|--------|----------|--------|
| 1 | 1232 | Kir Gaseosa | Activo |
| 2 | 12312 | Cocacola | Activo |
| 3 | 32432432 | Inkakola | Activo |
| 4 | 111111 | Pepsi | Activo |
| 5 | 222 | Chilcano | Activo |

---

## 5.2 Fotografías del Sistema en Operación

> **NOTA:** Insertar fotografías reales de la instalación en Art Atlas y del dispositivo MENTOR.

### Fotos sugeridas a incluir

| # | Descripción de la foto | Ubicación |
|---|----------------------|-----------|
| F1 | Dispositivo Jetson Orin instalado en gabinete | Art Atlas — sala de equipos |
| F2 | Cámara IP montada sobre línea de producción | Art Atlas — línea 1 |
| F3 | Vista de la cámara (lo que "ve" el sistema) | Art Atlas — línea 1 |
| F4 | Dashboard cloud con gráficos OEE | Captura de pantalla del navegador |
| F5 | UI Local mostrando estado de servicios | Captura de pantalla http://192.168.100.31:8080 |
| F6 | Tablet del operador con la app Mentor | Art Atlas — planta |
| F7 | Pantalla de configuración de ROI | Captura de pantalla de calibración |
| F8 | Vista general de la planta con sistema instalado | Art Atlas — panorámica |
| F9 | Detalle del cableado (cámara → switch → Jetson) | Art Atlas — gabinete |
| F10 | Grafana con métricas de infraestructura | Captura de pantalla http://servidor:3000 |

### Formato sugerido para fotos

```
[INSERTAR FOTO F1]
Figura 1: Dispositivo NVIDIA Jetson Orin instalado en gabinete industrial
en la planta de Art Atlas. El dispositivo procesa video de 4 cámaras
simultáneamente con un consumo de ~25W.

[INSERTAR FOTO F2]
Figura 2: Cámara IP industrial montada sobre la estructura de la línea 1.
El soporte articulado permite ajustar el ángulo para una detección óptima.

[INSERTAR FOTO F3]
Figura 3: Vista desde la cámara mostrando la región de interés (ROI)
configurada. El rectángulo verde indica la zona de detección activa.
```

---

## 5.3 Experiencia de Funcionamiento en Art Atlas

### 5.3.1 Rendimiento del sistema

| Indicador | Valor medido |
|-----------|-------------|
| Tiempo de operación continua | 24/7 desde instalación |
| Precisión de detección | [COMPLETAR: % de precisión medido] |
| Latencia de detección | < 100ms por frame |
| Tiempo de sincronización edge→cloud | < 5 segundos |
| Recuperación tras corte eléctrico | < 30 segundos (automática) |
| Consumo de CPU por cámara | ~31% (con aceleración GPU) |
| Cámaras simultáneas | 4 (con margen para +1 adicional) |

### 5.3.2 Disponibilidad del sistema

```
Disponibilidad = Tiempo operativo / Tiempo total × 100

Tiempo total:     [COMPLETAR] horas
Tiempo operativo: [COMPLETAR] horas
Disponibilidad:   [COMPLETAR] %
```

**Causas de indisponibilidad registradas:**
| Causa | Frecuencia | Duración | Acción correctiva |
|-------|-----------|----------|-------------------|
| Actualización de software | Planificada | 5-10 min | Deploy durante turno muerto |
| Corte de internet | Ocasional | Variable | Sin impacto (offline-first) |
| Corte eléctrico | Raro | Variable | Arranque automático con Docker |
| [Otros] | [COMPLETAR] | [COMPLETAR] | [COMPLETAR] |

### 5.3.3 Beneficios reportados por Art Atlas

| Beneficio | Descripción |
|-----------|-------------|
| **Visibilidad OEE** | Primer acceso a datos de OEE en tiempo real por línea |
| **Detección de paradas** | Identificación automática de paradas con categorización |
| **Reducción de registro manual** | Eliminación del conteo manual de producción |
| **Datos para mejora continua** | Análisis Pareto de causas de parada |
| **Acceso remoto** | Dashboard cloud accesible desde cualquier ubicación |
| **Integración** | API disponible para conexión con ERP existente |

---

## 5.4 Aplicabilidad a Otros Sectores

El sistema MENTOR EDGE está diseñado con **modos de operación configurables** que permiten aplicarlo en diferentes industrias:

### 5.4.1 Sectores validados

| Sector | Aplicación | Estado |
|--------|-----------|--------|
| **Textil** | Detección de paso de tela/prenda en línea | ✅ En producción (Art Atlas) |
| **Bebidas** | Conteo de botellas/envases en línea | ✅ Piloto validado (ISM) |

### 5.4.2 Sectores aplicables (pendientes de validación)

| Sector | Aplicación potencial | Adaptación requerida |
|--------|---------------------|---------------------|
| **Alimentos** | Conteo de paquetes, detección de paradas en línea de envasado | Calibración de señales para el producto específico |
| **Farmacéutico** | Conteo de unidades (blisters, cajas) | Ajuste de ROI y umbrales |
| **Plásticos** | Detección de piezas inyectadas | Calibración para color y forma del producto |
| **Metalmecánica** | Conteo de piezas en conveyor | Evaluación de reflectividad metálica |
| **Papel/Cartón** | Detección de láminas o cajas | Calibración similar a textil |
| **Automotriz** | Conteo de piezas en línea de ensamble | Integración con líneas existentes |

### 5.4.3 Requisitos para nueva aplicación

Para aplicar el sistema en un nuevo sector se requiere:

1. **Evaluación visual** — Verificar que el producto sea distinguible del fondo por cámara
2. **Prueba de señales** — Validar cuáles señales (edge, color, flow) son efectivas
3. **Calibración** — Ajustar umbrales y parámetros FSM para la velocidad de la línea
4. **Validación** — Comparar conteo del sistema vs conteo manual durante 1-2 turnos

**Tiempo estimado de adaptación:** 1-2 días por tipo de producto/línea.

---

## 5.5 Comparativa con Alternativas del Mercado

| Característica | MENTOR EDGE | Sensores mecánicos | Sistemas SCADA con PLC | Sistemas de visión premium |
|---------------|-------------|-------------------|----------------------|---------------------------|
| **Costo por línea** | Bajo | Medio | Alto | Muy alto |
| **Instalación** | 1 día | 1-2 días | 1-2 semanas | 1 semana |
| **Contacto con producto** | No (visión) | Sí (mecánico) | Depende | No (visión) |
| **Multi-línea** | Sí (5+ cámaras) | 1 sensor por línea | Depende del PLC | Depende |
| **Dashboard web** | Incluido | No incluido | Requiere SCADA HMI | Varía |
| **Operación offline** | Sí | N/A | Parcial | Depende |
| **API para ERP** | Incluida | No | Requiere desarrollo | Varía |
| **Mantenimiento** | Mínimo | Frecuente (desgaste) | Medio | Medio |
| **Adaptabilidad** | Alta (software) | Baja (hardware) | Media | Alta |

---

## 5.6 Testimonios y Evidencia

> **NOTA:** Completar con testimonios reales del personal de Art Atlas e ISM.

### Testimonio sugerido 1 — Jefe de planta
```
"[COMPLETAR: Testimonio del jefe de planta sobre el impacto
del sistema en la productividad y visibilidad de la producción]"

— [Nombre], Jefe de Planta, Art Atlas S.A.
```

### Testimonio sugerido 2 — Gerente de operaciones
```
"[COMPLETAR: Testimonio sobre el valor de los datos OEE
para la toma de decisiones]"

— [Nombre], Gerente de Operaciones, Art Atlas S.A.
```

### Testimonio sugerido 3 — Referencia ISM
```
"[COMPLETAR: Testimonio del piloto en ISM sobre la
validación técnica del sistema]"

— [Nombre], [Cargo], Industrias San Miguel
```
