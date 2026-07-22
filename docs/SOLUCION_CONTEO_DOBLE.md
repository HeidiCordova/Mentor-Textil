# Solución a Problemas de Conteo Doble

## Fecha: 2026-05-11

## Problemas Identificados

### Problema 2: Conteo doble en transición entre paneles/vistas

**Ubicación**: `mentor-edge/services/yolo-counter/app/domain/line_counter.py`

**Causa raíz**: 
- Cuando se llamaba `update_line()` para cambiar la configuración de la línea de conteo, se reseteaba completamente el tracker (`_frame_shape = (0, 0)` y `_zone = None`)
- Esto causaba que `_ensure_zone()` creara un nuevo tracker ByteTrack desde cero
- Los objetos que ya habían cruzado la línea perdían sus IDs de tracking
- Al reaparecer con nuevos IDs, eran contados nuevamente

**Síntoma**: "que no cuente doble, y cuando haiga esa transición entre panel y panel... cuente la unidad"

### Problema 3: frame_skip rompe el tracking

**Ubicación**: `mentor-edge/services/yolo-counter/app/application/counter_service.py`

**Causa raíz**:
- El código original saltaba frames completamente: cuando `frame_count % frame_skip != 0`, se hacía `continue` sin procesar el frame
- ByteTrack necesita continuidad de frames para mantener IDs consistentes
- Con frame_skip alto, el tracker perdía objetos rápidamente y los reasignaba con nuevos IDs
- Esto generaba conteos dobles cuando una botella cruzaba la línea pero el tracker la perdía y la redetectaba

## Soluciones Implementadas

### Solución 1: Preservar el tracker durante cambios de configuración

**Archivo**: `line_counter.py`

**Cambios**:

1. **Nuevo atributo para tracking de IDs cruzados**:
```python
self._crossed_ids: Set[int] = set()
self._max_crossed_ids = 1000
```

2. **Modificación de `_ensure_zone()`**:
```python
# Solo crear un nuevo tracker si no existe
# Esto preserva los IDs de tracking durante cambios de configuración
if self._tracker is None:
    self._tracker = sv.ByteTrack(
        track_activation_threshold=0.25,
        lost_track_buffer=30,  # Aumentado de 15 a 30
        minimum_matching_threshold=0.8,
        frame_rate=15,
    )
```

3. **Modificación de `update_line()`**:
```python
if changed:
    # Solo resetear la zona, NO el tracker
    # Esto preserva los IDs de tracking durante cambios de configuración
    self._frame_shape = (0, 0)
    self._zone = None
    # NO resetear self._tracker ni self._crossed_ids
```

4. **Mejora del método `update()`**:
- Mantiene un registro de IDs que ya cruzaron la línea
- Solo cuenta objetos con IDs nuevos (no vistos antes)
- Limita el tamaño del set de IDs para evitar crecimiento infinito de memoria
- Incluye fallback si no hay IDs disponibles

### Solución 2: Mantener continuidad del tracker con frame_skip

**Archivo**: `counter_service.py`

**Cambio fundamental**:
```python
self._frame_count += 1
should_detect = (self._frame_count % self._frame_skip == 0)

roi_frame = self._roi.extract(frame) if self._roi else frame
h, w = roi_frame.shape[:2]

if should_detect:
    # Ejecutar detección YOLO solo cada N frames
    detections = self._yolo.detect_to_sv(roi_frame)
else:
    # En frames intermedios, usar detecciones vacías pero mantener el tracker activo
    # Esto permite que ByteTrack mantenga la continuidad de los IDs
    import supervision as sv
    detections = sv.Detections.empty()
    time.sleep(_SKIP_SLEEP)

# SIEMPRE actualizar el counter (incluso con detecciones vacías)
# Esto mantiene el tracker activo y evita pérdida de IDs
delta = self._counter.update(detections, h, w)
```

**Beneficios**:
- ByteTrack recibe TODOS los frames (aunque algunos con detecciones vacías)
- Mantiene la continuidad de los IDs de tracking
- Reduce el costo computacional ejecutando YOLO solo cada N frames
- Evita conteos dobles por pérdida de tracking

### Solución 3: Aumento del buffer de tracks perdidos

**Archivo**: `line_counter.py`

**Cambio**:
```python
lost_track_buffer=30,  # Aumentado de 15 a 30
```

**Beneficio**: Con frame_skip activo, los objetos pueden "desaparecer" por más tiempo. Un buffer mayor permite que el tracker mantenga los IDs por más frames sin detección.

## Parámetros Ajustables

### `lost_track_buffer` (actualmente 30)
- **Qué hace**: Número de frames que ByteTrack mantiene un track después de perder el objeto
- **Cuándo aumentar**: Si hay muchos frame_skip o las botellas se mueven muy rápido
- **Cuándo disminuir**: Si hay conteos fantasma de objetos que ya salieron del frame

### `_max_crossed_ids` (actualmente 1000)
- **Qué hace**: Límite de IDs históricos almacenados para prevenir conteo doble
- **Cuándo aumentar**: En líneas de producción muy largas con muchos objetos únicos
- **Cuándo disminuir**: Si hay problemas de memoria (muy improbable)

### `frame_skip` (configurable por instalación)
- **Qué hace**: Ejecuta YOLO solo cada N frames
- **Recomendación**: Mantener entre 1-3 para mejor tracking
- **Nota**: Con las soluciones implementadas, valores más altos son más seguros que antes

## Verificación

Para verificar que las soluciones funcionan:

1. **Test de transición de panel**:
   - Cambiar la configuración de la línea de conteo mientras hay objetos en tránsito
   - Verificar que no se cuentan doble
   - Revisar logs: debe aparecer "line updated: ... (tracker preserved)"

2. **Test de frame_skip**:
   - Configurar frame_skip alto (ej: 3-5)
   - Verificar que el conteo sigue siendo preciso
   - Revisar logs: no deben aparecer muchos "no tracker IDs available"

3. **Test de continuidad**:
   - Observar los logs de debug: "object X crossed line"
   - Verificar que cada objeto tiene un ID único y no se repite

## Logs de Debug

Para activar logs detallados, ajustar el nivel de logging:
```python
logger.setLevel(logging.DEBUG)
```

Logs relevantes:
- `"object %d crossed line (total crossed: %d)"` - Cuando un objeto cruza
- `"cleaned old crossed IDs, remaining: %d"` - Cuando se limpia el historial
- `"line updated: ... (tracker preserved)"` - Cuando se actualiza la configuración
- `"no tracker IDs available, using raw delta"` - Fallback (no debería ser frecuente)

## Impacto en Performance

- **Memoria**: +8 bytes por ID único trackeado (máximo ~8KB con 1000 IDs)
- **CPU**: Mínimo (solo verificación de set membership)
- **Precisión**: Significativamente mejorada en transiciones y con frame_skip

## Próximos Pasos (Opcional)

Si se siguen observando conteos dobles:

1. Implementar un sistema de "zonas de exclusión" donde objetos ya contados no pueden ser recontados hasta salir completamente del frame
2. Agregar validación de velocidad: objetos que se mueven demasiado rápido entre frames probablemente son el mismo objeto con nuevo ID
3. Implementar persistencia de IDs cruzados en disco para sobrevivir reinicios del servicio
