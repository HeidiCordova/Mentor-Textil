# Lógica de Clasificación, Tiempos y Detector de Presencia

> Documento técnico — Mentor Edge  
> Última actualización: 21 de marzo de 2026  
> Versión del sistema: commit `b3e6e7f`

---

## 1. Arquitectura General del Detector

El detector de eventos corre en el **Jetson Orin** dentro del servicio `vision-event-detector` (Python). Procesa el stream de cámara frame a frame y clasifica el estado de la máquina usando dos ROIs independientes:

```
Cámara RTSP
    │
    ▼
┌───────────────────────────────────┐
│       vision-event-detector       │
│                                   │
│  ROI de CONTEO (beige)            │
│  ┌────────────────────────────┐   │
│  │  BeigeSignal               │   │
│  │  fusion_score → EventFSM   │──►│ CORTE (evento producción)
│  └────────────────────────────┘   │
│                                   │
│  ROI de PRESENCIA (movimiento)    │
│  ┌────────────────────────────┐   │
│  │  PresenceDetector          │──►│ Produciendo / Detenida
│  │  slow-window diff          │   │ (micro-parada / parada)
│  └────────────────────────────┘   │
└───────────────────────────────────┘
         │
         ▼
   edge-config-service (Go)
         │
         ▼
   PostgreSQL → cloud-ingest → nube
```

---

## 2. ROI de Conteo — Detección por Color Beige

### 2.1 Propósito

Detectar el paso de cada **prenda** frente a la cámara contando cuántas veces el color
característico de la tela (beige/naranja) cubre una fracción mínima del ROI de conteo.

### 2.2 Cómo el detector "ve" la prenda

Por cada frame el servicio extrae un `fusion_score` de 0 a 1 que representa qué porcentaje
de los píxeles del ROI coincide con el rango HSV calibrado de la tela. Este score se pasa
directamente a la FSM de eventos.

### 2.3 Parámetros de sensibilidad

| Parámetro | Clave config | Rango | Default | Descripción |
|---|---|---|---|---|
| Umbral alto (histeresis) | `beige` / `high` | 0.30 – 0.95 | 0.70 | Score mínimo para "prenda presente" |
| Umbral bajo (histeresis) | `low` | 0.05 – 0.50 | 0.30 | Score máximo para "prenda ausente" |

### 2.4 Rango HSV de la tela

Configurable desde **Modo Lab → pestaña Beige**:

| Canal | Mínimo | Máximo | Nota |
|---|---|---|---|
| Tono (H) | 12° | 50° | Naranja-amarillo |
| Saturación (S) | 8 | 160 | Excluye grises |
| Brillo (V) | 100 | 255 | Excluye sombras |

> **Calibrar por tela**: ajustar H_min/H_max hasta que el overlay naranja cubra solo la
> tela sin incluir fondo. Una saturación mínima alta ayuda a excluir blancos y beiges claros.

---

## 3. Máquina de Estados (EventFSM)

### 3.1 Diagrama de estados

```
           score > high
IDLE ─────(N frames)──────► DETECTING
  ▲                              │
  │ score < low                  │ confirmado → +1 unidad
  │                              ▼
  │                          WAIT_EXIT
  │                              │
  │        exit_low_streak       │  (histéresis simétrica +
  │        >= exit_frames        │   timeout condicional)
  │              ▼               │
  └──────── COOLDOWN ◄───────────┘
```

### 3.2 Descripción de estados

| Estado | Descripción |
|---|---|
| `IDLE` | Sin prenda. Esperando señal alta. |
| `DETECTING` | Se detectó beige. Confirmando por `n_frames` consecutivos. |
| `WAIT_EXIT` | Evento registrado (+1 prenda). Esperando que la prenda salga del ROI. |
| `COOLDOWN` | Anti-rebote post-salida. Bloqueado por `cooldown_frames`. |

### 3.3 Parámetros de la FSM

| Parámetro | Clave config | Rango | Default | Descripción |
|---|---|---|---|---|
| Frames de confirmación | `n_frames` | 1 – 30 | 3 | Frames consecutivos por encima del umbral alto para registrar evento |
| Cooldown | `cooldown_frames` | 0 – 60 | 8 | Frames bloqueados tras confirmar salida |
| Frames de salida | `exit_frames` | 1 – 100 | 5 | Frames consecutivos con beige < low para confirmar que la prenda salió |
| Timeout máx. espera | `max_wait_exit_frames` | — | 750 | Frames máximos en WAIT_EXIT **solo dispara si beige < low en ese momento** |

> **Nota sobre fps**: a 12.5 fps, 750 frames ≈ 60 segundos. Suficiente para prendas
> de ciclo largo que permanecen en el ROI varios segundos antes de salir.

---

## 4. Anti-rebote para Prendas de Paso Lento (Histéresis Simétrica)

### 4.1 El problema

Las prendas textiles con costura negra y cola beige producen la siguiente secuencia de scores al atravesar el ROI:

```
Tiempo →  [beige alto] → [costura negra: LOW] → [cola beige: HIGH] → [fondo: LOW]
```

Sin histéresis, la costura (1-3 frames en LOW) terminaría el WAIT_EXIT y la cola beige
volvería a contar como una segunda prenda → **doble conteo**.

### 4.2 La solución

El `exit_low_streak` (contador de salida) solo se **resetea** si el beige sube sostenidamente
durante `exit_frames` frames **consecutivos**. Una costura de 2-3 frames no es suficiente para cancelar el progreso de salida si fue menor que `exit_frames`.

```python
# event_fsm.py — lógica clave de WAIT_EXIT
if score < low_threshold:
    exit_low_streak  += 1
    exit_high_streak  = 0        # reinicia el contador de "rebote alto"
else:
    exit_high_streak += 1
    # Solo resetea el progreso de salida si el beige vuelve SOSTENIDAMENTE
    if exit_high_streak >= exit_frames:
        exit_low_streak  = 0
        exit_high_streak = 0

exit_confirmed = exit_low_streak >= exit_frames

# Timeout solo dispara si la prenda YA no está en el ROI
timed_out = (wait_exit_total >= max_wait_exit_frames) and (score < low_threshold)

if exit_confirmed or timed_out:
    → pasar a COOLDOWN
```

### 4.3 Guía de ajuste para prendas problemáticas

| Síntoma | Causa probable | Ajuste |
|---|---|---|
| Cuenta 2 por cada prenda | Costura o cola beige termina WAIT_EXIT | Subir `exit_frames` (5 → 10 → 20) |
| No cuenta (prenda lenta en ROI > 60s) | Timeout no dispara (beige alto) | Normal: FSM espera. Revisar `low_threshold` |
| Cuenta 0 pese a beige visible | Umbral alto demasiado exigente | Bajar `high_threshold` (0.70 → 0.50) |
| Cuenta en cada frame | Umbral bajo demasiado alto | Bajar `low_threshold` (0.30 → 0.15) |

---

## 5. ROI de Presencia — Detección de Movimiento (Slow-Window Diff)

### 5.1 Propósito

Determinar si la **máquina está produciendo o detenida** independientemente del color de
la tela. Funciona con cualquier tejido, incluso blanco o negros puros.

### 5.2 Por qué slow-window y no frame-to-frame

En industria textil un ciclo completo de prenda puede tomar 8-40 minutos.  
A 12.5 fps, la tela se mueve ~0.016 px/frame — invisible en un diff frame a frame.  
Comparar el frame actual con el de **N segundos atrás** hace el movimiento detectable.

### 5.3 Algoritmo

```
frame[t]  ──────────────────────────────────┐
                                             ▼  diff absoluto
frame[t - presencia_window] ───────────► |A - B| > presencia_pixel_threshold
                                             │
                                             ▼
                               motion_pixels / total_pixels
                                             │
                               >= presencia_scale ?
                               ┌────YES────► produciendo (hold=0)
                               └────NO─────► exit_hold_counter++
                                             │
                               exit_hold >= presencia_hold ?
                               └────YES────► DETENIDA
```

### 5.4 Parámetros de calibración

Accesibles desde **Modo Lab → Sensibilidad ROI Presencia**:

| Parámetro | Clave config | Rango slider | Default | Descripción |
|---|---|---|---|---|
| Ventana de comparación | `presencia_window` | 2 – 60 s | 375 frames (30 s) | Cuántos frames atrás comparar |
| Sensibilidad de píxel | `presencia_pixel_threshold` | 2 – 40 | 8 | Cambio mínimo de intensidad para contar un píxel como "en movimiento" |
| Área mínima con movimiento | `presencia_scale` | 0.1% – 10% | 0.5% | Fracción del ROI que debe mostrar movimiento |
| Tiempo para declarar parada | `presencia_hold` | 10 – 300 s | 750 frames (60 s) | Frames consecutivos sin movimiento para transicionar a "Detenida" |

#### Guía de calibración por tipo de máquina

| Tipo de máquina | `presencia_window` | `presencia_scale` | `presencia_hold` |
|---|---|---|---|
| Remalladora rápida | 2 – 5 s | 0.5% – 2% | 10 – 20 s |
| Recta industrial media | 5 – 15 s | 0.3% – 1% | 20 – 40 s |
| Máquina de ciclo largo (>15 min/prenda) | 20 – 40 s | 0.1% – 0.5% | 60 – 120 s |

> **Consejo**: Con la máquina produciendo, el badge `Δmv` en el Lab debe mostrar un valor > 0.  
> Ajustar `presencia_pixel_threshold` hasta que sea claro el contraste entre produciendo y detenida.

### 5.5 Clasificación de paradas

| Duración sin movimiento | Tipo | Color en Lab |
|---|---|---|
| 0 – 2 min | Micro-parada | Ámbar |
| > 2 min | Parada | Rojo |

> El umbral de 2 minutos es para visualización en el Lab. En producción se aplica la
> configuración del módulo de paradas del cloud (`micro_stop_max_s`).

---

## 6. Modo Lab — Panel de Calibración en Vivo

### 6.1 Cronómetro de estado (barra superior)

```
● Produciendo  1m 23s   Δ3.6px  |  7 uds [WAIT_EXIT]  ↺
```

| Indicador | Significado |
|---|---|
| `● Produciendo Xs` | Máquina en movimiento. Timer reinicia al detenerse. |
| `● Micro-parada Xs` | Detenida < 2 min. Timer reinicia al retomar producción. |
| `● Parada Xs` | Detenida ≥ 2 min. |
| `Δ X.Xpx` | Score de movimiento promedio del diff lento. |
| `N uds [ESTADO]` | Unidades detectadas en la sesión + estado FSM actual. |
| `↺` | Reinicia el contador de unidades a 0. |

### 6.2 Contador de unidades en Lab

El contador usa la misma lógica que el backend (4 estados: IDLE→DETECTING→WAIT_EXIT→COOLDOWN),
implementada en JavaScript frame a frame en el navegador. Los parámetros que usa son los
sliders del panel **Máquina de Estados (FSM)** en el Lab.

El badge del contador cambia de color según el estado FSM actual:

| Color del badge | Estado FSM |
|---|---|
| Azul | DETECTING (confirmando N frames) |
| Naranja | WAIT_EXIT (esperando que la prenda salga) |
| Violeta | COOLDOWN (anti-rebote) |

### 6.3 Flujo de calibración recomendado

1. Abrir **Modo Lab** → activar **En vivo**
2. Dibujar el **ROI de conteo** sobre la zona de paso de la prenda (amarillo)
3. Dibujar el **ROI de presencia** sobre la zona de movimiento de la tela (verde punteado)
4. Activar **Análisis → Beige**
5. Ajustar **Tono H** hasta que el overlay naranja cubra solo la tela
6. Ajustar **Umbral Beige** hasta que el borde del ROI esté verde solo cuando hay prenda
7. Pasar prendas manualmente y verificar que el contador sube de 1 en 1
8. Si cuenta doble: subir **exit_frames** (5→10→20) o subir **Cooldown**
9. Ir a **Sensibilidad ROI Presencia** y verificar que el badge cambia a `Micro-parada` al detener la máquina
10. Ajustar **Tiempo para declarar parada** según la tolerancia de la operación
11. Hacer click en **Guardar** para persistir la config en el Jetson

---

## 7. Clasificación de Tiempos de Inactividad

### 7.1 Cómo se acumulan los tiempos

Cada vez que el OEE recibe `is_producing = False`, acumula `idle_s` (segundos en idle
continuo). Cuando **vuelve a detectar producción**, clasifica ese idle acumulado:

```
idle_s < micro_stop_max_s  →  MICROPARADA
idle_s ≥ micro_stop_max_s  →  PARADA NO ASIGNADA
```

El valor configurado es `micro_stop_max_s = 210 segundos` para líneas convencionales.

### 7.2 Tipos de tiempo registrados

| Clave OEE | Origen | Descripción |
|---|---|---|
| `T_MICROPARADA` | Automático (detector) | Idle breve — el operador puede resolver solo |
| `T_PARADA_NO_ASIGNADA` | Automático (detector) | Idle largo — requiere registro de causa |
| `T_PARADA_PROGRAMADA` | Manual (tablet) | Parada planificada registrada por operador |
| `T_PARADA_NO_PROGRAMADA` | Manual (tablet) | Falla o parada inesperada registrada |
| `T_REFRIGERIO` | Manual (tablet) | Descanso |
| `T_CAPACITACION_OBLIGATORIA` | Manual (tablet) | Capacitación |
| `T_MANTENIMIENTO_PLANIFICADO` | Manual (tablet) | Mantenimiento |
| `T_DISPONIBLE` | Calculado | Derivado desde turnos del gateway |

### 7.3 El problema del umbral fijo en textil

El umbral fijo heredado de 210 s no representa el ciclo productivo textil: una prenda
tarda 8-15 minutos. Con 210 s, el sistema clasifica como parada
**mientras la máquina está fabricando normalmente**, generando OEE falso.

**Solución propuesta** (pendiente de implementar): umbral dinámico basado en velocidad nominal:

```
micro_stop_max_s = MAX( piso_minimo_s,  (1 / velocidad_us) × factor_tolerancia )
```

| Parámetro | Valor sugerido |
|---|---|
| `factor_tolerancia` | 2.0 (2 ciclos nominales) |
| `piso_minimo_s` | 30 s (para líneas muy rápidas) |

---

## 8. Archivos Relevantes

| Archivo | Descripción |
|---|---|
| `services/vision-event-detector/app/domain/fsm/event_fsm.py` | FSM de eventos con histéresis simétrica y timeout condicional |
| `services/vision-event-detector/app/domain/presence/presence_detector.py` | Detector de presencia slow-window diff |
| `services/vision-event-detector/app/application/detector_service.py` | Orquestador principal |
| `services/vision-event-detector/app/domain/roi/roi_manager.py` | Extracción de ROI con clamping |
| `services/ui-local/src/views/Config.vue` | Frontend Lab: calibración, cronómetros, contador de unidades, mini-FSM JS |
| `services/edge-config-service/internal/domain/config_model.go` | Modelo de config Go (incluye campos `presencia_*`) |

---

## 9. Notas de Producción y Deploy

- Al cambiar `Config.vue` en el Jetson se debe reconstruir la imagen:
  ```bash
  docker compose -f infrastructure/docker/docker-compose.jetson-orin.yml build --no-cache ui-local
  docker compose -f infrastructure/docker/docker-compose.jetson-orin.yml up -d ui-local
  ```
- Al cambiar `event_fsm.py` solo hace falta reiniciar el servicio Python:
  ```bash
  docker compose -f infrastructure/docker/docker-compose.jetson-orin.yml restart vision-event-detector
  ```
- El parámetro `presencia_window` reinicia el buffer interno del detector al actualizarse.
- Los contadores del Lab **no se guardan en BD** — son solo para verificar calibración.
- Al hacer **Guardar** en el Lab, la configuración se persiste en PostgreSQL local y el
  detector la recarga en caliente sin necesidad de reinicio.
