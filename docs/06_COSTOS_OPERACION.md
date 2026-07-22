# 6. CÁLCULO DE COSTOS DE OPERACIÓN

## 6.1 Costos de Hardware (Inversión Inicial)

### 6.1.1 Kit por planta (1 Jetson + hasta 5 líneas)

| Componente | Cantidad | Precio Unitario (USD) | Total (USD) |
|-----------|----------|----------------------|-------------|
| NVIDIA Jetson Orin Nano 8GB DevKit | 1 | ~499 | 499 |
| SSD NVMe 256GB | 1 | ~35 | 35 |
| Fuente de alimentación 12V/5A | 1 | ~15 | 15 |
| Gabinete industrial IP54 | 1 | ~80 | 80 |
| Switch PoE 8 puertos | 1 | ~90 | 90 |
| **Subtotal base (por planta)** | | | **719** |

### 6.1.2 Kit por línea adicional

| Componente | Cantidad | Precio Unitario (USD) | Total (USD) |
|-----------|----------|----------------------|-------------|
| Cámara IP Industrial PoE (1080p) | 1 | ~80-150 | 80-150 |
| Cable Ethernet Cat6 (30m) | 1 | ~15 | 15 |
| Soporte articulado para cámara | 1 | ~25 | 25 |
| **Subtotal por línea** | | | **120-190** |

### 6.1.3 Costo total según configuración

| Configuración | Hardware Base | Cámaras | **Total Hardware** |
|---------------|-------------|---------|-------------------|
| 1 Jetson + 1 línea | $719 | $120-190 | **$839 - $909** |
| 1 Jetson + 2 líneas | $719 | $240-380 | **$959 - $1,099** |
| 1 Jetson + 4 líneas | $719 | $480-760 | **$1,199 - $1,479** |
| 1 Jetson + 5 líneas | $719 | $600-950 | **$1,319 - $1,669** |

> **Nota:** Un solo Jetson Orin soporta hasta 5 cámaras simultáneamente con el procesamiento optimizado actual (31% CPU por cámara).

---

## 6.2 Costos de Infraestructura Cloud

### 6.2.1 Servidor Cloud (VPS)

| Especificación | Costo Mensual (USD) |
|---------------|-------------------|
| VPS Linux 4 vCPU / 8 GB RAM / 80 GB SSD | ~$20 - $40 |
| Dominio (opcional) | ~$1/mes |
| Certificado SSL (Let's Encrypt) | Gratuito |
| **Total mensual cloud** | **$20 - $40** |

**Proveedores de referencia:**
- Contabo: ~$13/mes (VPS S SSD)
- Hetzner: ~$16/mes (CX31)
- DigitalOcean: ~$24/mes (4 vCPU)
- AWS Lightsail: ~$40/mes (4 vCPU)

### 6.2.2 Escalabilidad del servidor

| Plantas conectadas | RAM recomendada | Costo mensual estimado |
|-------------------|----------------|----------------------|
| 1-3 plantas | 4 GB | $15 - $25 |
| 4-10 plantas | 8 GB | $25 - $45 |
| 11-20 plantas | 16 GB | $45 - $80 |
| 20+ plantas | 32 GB | $80 - $150 |

---

## 6.3 Costos de Operación Mensual

### 6.3.1 Consumo eléctrico del Edge

| Componente | Consumo | Horas/mes | kWh/mes | Costo (USD)* |
|-----------|---------|-----------|---------|-------------|
| Jetson Orin (modo 25W) | 25W | 720 | 18 | ~$2.70 |
| Switch PoE | 15W | 720 | 10.8 | ~$1.62 |
| Cámaras PoE (×4) | 12W c/u | 720 | 34.6 | ~$5.18 |
| **Total eléctrico mensual** | | | **63.4** | **~$9.50** |

*Tarifa estimada: $0.15/kWh (varía por país)

### 6.3.2 Resumen de costos de operación mensual

| Concepto | Costo Mensual (USD) |
|---------|-------------------|
| Servidor cloud (VPS) | $20 - $40 |
| Electricidad (edge, 4 líneas) | ~$10 |
| Internet en planta (asumiendo existente) | $0 (ya existe) |
| Mantenimiento de software | $0 (actualizaciones incluidas) |
| **Total operación mensual** | **$30 - $50** |

---

## 6.4 Costos de Mantenimiento

### 6.4.1 Mantenimiento preventivo

| Actividad | Frecuencia | Costo estimado |
|-----------|-----------|---------------|
| Limpieza de lente de cámara | Semanal | $0 (operador de planta) |
| Verificación de estado del sistema (dashboard) | Diario | $0 (automático + revisar) |
| Backup de base de datos cloud | Automático (diario) | $0 (script incluido) |
| Actualización de software | Trimestral | Incluido en soporte |

### 6.4.2 Mantenimiento correctivo (estimado)

| Evento | Probabilidad anual | Costo estimado |
|--------|-------------------|---------------|
| Reemplazo de cámara IP | 5% | $80-150 |
| Reemplazo de cable Ethernet | 2% | $15 |
| Reemplazo de switch PoE | 2% | $90 |
| Reemplazo de fuente de alimentación | 3% | $15 |
| Reemplazo de Jetson Orin | <1% | $499 |

**Costo anual estimado de mantenimiento correctivo:** ~$20-30 (ponderado por probabilidad)

---

## 6.5 Costo Total de Propiedad (TCO) — 3 Años

### Escenario: 1 planta con 4 líneas

| Concepto | Año 1 | Año 2 | Año 3 | Total 3 años |
|---------|-------|-------|-------|-------------|
| **Hardware (inversión)** | $1,199-1,479 | $0 | $0 | $1,199-1,479 |
| **Instalación** | $500* | $0 | $0 | $500 |
| **Cloud (VPS)** | $360-480 | $360-480 | $360-480 | $1,080-1,440 |
| **Electricidad** | $114 | $114 | $114 | $342 |
| **Mantenimiento** | $25 | $25 | $25 | $75 |
| **Total** | **$2,198-2,598** | **$499-619** | **$499-619** | **$3,196-3,836** |

*Costo de instalación estimado (incluye mano de obra de 1 día)

### TCO mensualizado

| Periodo | Costo/mes por línea |
|---------|-------------------|
| Año 1 (incluye hardware) | ~$46 - $54 por línea |
| Año 2+ (solo operación) | ~$10 - $13 por línea |
| **Promedio 3 años** | **~$22 - $27 por línea/mes** |

---

## 6.6 Comparativa de Costos vs Alternativas

| Solución | Costo inicial (4 líneas) | Costo mensual | TCO 3 años |
|---------|------------------------|--------------|-----------|
| **MENTOR EDGE** | ~$1,500 | ~$50 | ~$3,500 |
| Sensores mecánicos + PLC | ~$4,000-8,000 | ~$100 (mantenimiento) | ~$7,600-11,600 |
| Sistema SCADA comercial | ~$15,000-30,000 | ~$200-500 (licencias) | ~$22,200-48,000 |
| Visión artificial premium (Cognex, Keyence) | ~$20,000-40,000 | ~$100-300 | ~$23,600-50,800 |

---

## 6.7 Retorno de Inversión (ROI)

### Ahorros potenciales por visibilidad OEE

| Fuente de ahorro | Ahorro mensual estimado |
|-----------------|----------------------|
| Identificación de paradas no reportadas | [COMPLETAR según producción] |
| Reducción de tiempo de parada por respuesta rápida | [COMPLETAR] |
| Eliminación de registro manual de producción | [COMPLETAR: horas-hombre ahorradas] |
| Optimización de cambios de turno/producto | [COMPLETAR] |

### Fórmula ROI

```
ROI = (Ahorro anual - Costo anual de operación) / Inversión inicial × 100

Ejemplo con ahorro conservador de $500/mes:
ROI = ($6,000 - $600) / $1,500 × 100 = 360% en el primer año
Payback: ~3 meses
```

> **NOTA:** Los valores de ahorro deben completarse con datos reales del cliente.
> Un estudio de McKinsey indica que la implementación de monitoreo OEE en tiempo real
> típicamente mejora la eficiencia de producción entre 5% y 15%.

---

## 6.8 Notas Importantes

1. **Sin costos de licencia recurrentes:** El software es propio y no requiere licencias de terceros
2. **Escalable:** Agregar una línea solo requiere una cámara adicional ($120-190)
3. **Sin obsolescencia de hardware:** El Jetson Orin tiene soporte NVIDIA hasta 2033+
4. **Precios de referencia:** Los precios indicados son aproximados y pueden variar según proveedor y país
5. **Moneda:** Todos los precios están expresados en dólares estadounidenses (USD)
