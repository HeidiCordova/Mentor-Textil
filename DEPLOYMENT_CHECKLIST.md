# ✅ Checklist de Deployment - Solución Conteo Doble

## Pre-Deployment

### 1. Backup
- [ ] Backup de `line_counter.py`
  ```bash
  cp mentor-edge/services/yolo-counter/app/domain/line_counter.py \
     mentor-edge/services/yolo-counter/app/domain/line_counter.py.backup
  ```
- [ ] Backup de `counter_service.py`
  ```bash
  cp mentor-edge/services/yolo-counter/app/application/counter_service.py \
     mentor-edge/services/yolo-counter/app/application/counter_service.py.backup
  ```
- [ ] Backup de configuración actual
  ```bash
  systemctl show yolo-counter > yolo-counter-config.backup
  ```

### 2. Verificación de Archivos
- [ ] Verificar que todos los archivos modificados están presentes
- [ ] Verificar sintaxis Python
  ```bash
  python -m py_compile mentor-edge/services/yolo-counter/app/domain/line_counter.py
  python -m py_compile mentor-edge/services/yolo-counter/app/application/counter_service.py
  ```

### 3. Revisión de Configuración
- [ ] Revisar valor actual de `frame_skip`
- [ ] Documentar configuración actual de línea de conteo
- [ ] Verificar versión de supervision instalada
  ```bash
  pip show supervision
  ```

### 4. Preparación de Monitoreo
- [ ] Preparar dashboard de monitoreo
- [ ] Configurar alertas temporales
- [ ] Preparar script de comparación de conteos

---

## Deployment

### 1. Ventana de Mantenimiento
- [ ] Notificar a usuarios sobre mantenimiento
- [ ] Programar ventana de bajo tráfico (si es posible)
- [ ] Tiempo estimado: 15-30 minutos

### 2. Detener Servicio
```bash
systemctl stop yolo-counter
```
- [ ] Servicio detenido correctamente
- [ ] Verificar que no hay procesos huérfanos
  ```bash
  ps aux | grep yolo
  ```

### 3. Actualizar Archivos
- [ ] Los archivos ya están actualizados en el repositorio
- [ ] Verificar permisos de archivos
  ```bash
  ls -la mentor-edge/services/yolo-counter/app/domain/line_counter.py
  ls -la mentor-edge/services/yolo-counter/app/application/counter_service.py
  ```

### 4. Verificación Pre-Start
- [ ] Ejecutar script de verificación
  ```bash
  cd mentor-edge/services/yolo-counter
  python verify_double_count_fix.py
  ```
- [ ] Todas las verificaciones deben pasar (✓)

### 5. Iniciar Servicio
```bash
systemctl start yolo-counter
```
- [ ] Servicio iniciado correctamente
- [ ] Verificar estado
  ```bash
  systemctl status yolo-counter
  ```

---

## Post-Deployment (Primeros 15 minutos)

### 1. Verificación Inmediata
- [ ] Servicio está corriendo
  ```bash
  systemctl is-active yolo-counter
  ```
- [ ] No hay errores en logs
  ```bash
  journalctl -u yolo-counter -n 50 --no-pager
  ```
- [ ] Proceso está consumiendo recursos normales
  ```bash
  top -p $(pgrep -f yolo-counter)
  ```

### 2. Verificación de Logs
Buscar estos mensajes en los logs:
- [ ] ✅ "counter service started"
- [ ] ✅ "line updated: ... (tracker preserved)" (si hay cambio de config)
- [ ] ✅ "object X crossed line" (cuando hay conteo)
- [ ] ❌ NO debe aparecer frecuentemente: "no tracker IDs available"

```bash
journalctl -u yolo-counter -f | grep -E "tracker preserved|crossed line|tracker IDs"
```

### 3. Verificación Funcional
- [ ] Sistema está detectando objetos
- [ ] Conteos se están registrando
- [ ] No hay errores de Python/traceback
- [ ] Memoria estable (no crece)

---

## Post-Deployment (Primera Hora)

### 1. Monitoreo Continuo
- [ ] Revisar logs cada 15 minutos
- [ ] Verificar conteos vs. período anterior
- [ ] Monitorear uso de CPU y memoria
- [ ] Verificar que no hay warnings inusuales

### 2. Test de Transición
- [ ] Cambiar configuración de línea desde UI
- [ ] Verificar log: "line updated: ... (tracker preserved)"
- [ ] Verificar que no hay conteo doble
- [ ] Restaurar configuración original

### 3. Comparación de Conteos
- [ ] Comparar conteos con mismo período día anterior
- [ ] Diferencia esperada: conteos ligeramente menores (eliminación de dobles)
- [ ] Si diferencia > 20%, investigar

---

## Post-Deployment (Primeras 24 Horas)

### 1. Monitoreo de Memoria
- [ ] Verificar uso de memoria cada 4 horas
- [ ] Memoria debe mantenerse estable
- [ ] Buscar en logs: "cleaned old crossed IDs" (indica limpieza correcta)

### 2. Análisis de Precisión
- [ ] Realizar conteo manual de muestra (5-10 minutos)
- [ ] Comparar con conteo del sistema
- [ ] Precisión esperada: > 97%

### 3. Verificación de Performance
- [ ] CPU: debe ser igual o menor que antes (gracias a frame_skip)
- [ ] Latencia: sin cambios significativos
- [ ] Throughput: sin cambios significativos

---

## Criterios de Éxito

### ✅ Deployment Exitoso Si:
1. Servicio corriendo sin errores
2. Logs muestran "tracker preserved" en cambios de config
3. No aparece frecuentemente "no tracker IDs available"
4. Memoria estable (no crece)
5. Conteos precisos (> 97% vs. conteo manual)
6. No hay conteos dobles en transiciones

### ⚠️ Investigar Si:
1. Conteos > 20% diferentes vs. período anterior
2. Memoria crece constantemente
3. Muchos warnings "no tracker IDs available"
4. Errores de Python en logs
5. CPU significativamente mayor

### 🚨 Rollback Si:
1. Servicio no inicia
2. Errores críticos en logs
3. Conteos claramente incorrectos (> 30% diferencia)
4. Sistema inestable después de 2 horas

---

## Rollback Procedure

### Si es necesario hacer rollback:

1. **Detener servicio**
   ```bash
   systemctl stop yolo-counter
   ```

2. **Restaurar archivos**
   ```bash
   cp mentor-edge/services/yolo-counter/app/domain/line_counter.py.backup \
      mentor-edge/services/yolo-counter/app/domain/line_counter.py
   
   cp mentor-edge/services/yolo-counter/app/application/counter_service.py.backup \
      mentor-edge/services/yolo-counter/app/application/counter_service.py
   ```

3. **Reiniciar servicio**
   ```bash
   systemctl start yolo-counter
   ```

4. **Verificar**
   ```bash
   systemctl status yolo-counter
   journalctl -u yolo-counter -n 50
   ```

5. **Reportar problema**
   - Capturar logs completos
   - Documentar síntomas
   - Incluir métricas de antes/después

---

## Documentación Post-Deployment

### Completar después del deployment:

- [ ] Fecha y hora de deployment: _______________
- [ ] Versión anterior: _______________
- [ ] Versión nueva: 2026-05-11
- [ ] Persona responsable: _______________
- [ ] Incidentes durante deployment: _______________
- [ ] Tiempo de downtime: _______________
- [ ] Resultado: ✅ Exitoso / ⚠️ Con issues / 🚨 Rollback

### Métricas Baseline (registrar para comparación):

**Antes del deployment:**
- Conteos por hora: _______________
- Uso de CPU promedio: _______________
- Uso de memoria: _______________
- Conteos dobles observados: _______________

**Después del deployment (24h):**
- Conteos por hora: _______________
- Uso de CPU promedio: _______________
- Uso de memoria: _______________
- Conteos dobles observados: _______________

---

## Contactos de Emergencia

- **Equipo de desarrollo**: _______________
- **DevOps on-call**: _______________
- **Escalación**: _______________

---

## Notas Adicionales

Espacio para notas durante el deployment:

```
[Agregar notas aquí]
```

---

**Última actualización**: 2026-05-11  
**Versión del checklist**: 1.0
