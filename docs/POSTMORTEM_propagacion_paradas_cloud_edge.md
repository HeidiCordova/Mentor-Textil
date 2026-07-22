# Postmortem: Paradas de Cloud no se propagaban al Edge

**Fecha:** 17 de marzo de 2026  
**Severidad:** Alta — operadores en planta no podían ver en la tablet del Jetson las paradas asignadas desde la tablet cloud  
**Estado:** Resuelto ✅

---

## Resumen ejecutivo

Las paradas creadas o asignadas desde la tablet cloud (192.168.100.X:5174) aparecían correctamente en el carril "Asignado" del cloud, pero **nunca llegaban a la tablet del Jetson Orin** (192.168.100.31:8090). La causa raíz fue que `ResolveDeviceID` consultaba la base de datos equivocada, dejando el campo `device_id` vacío en todas las paradas creadas desde la nube. Sin `device_id`, el sistema no sabía a qué dispositivo edge despachar los comandos `CREAR_PARADA` y `JUSTIFICAR_PARADA`.

---

## Arquitectura involucrada

```
Tablet Cloud ──POST /api/stops──► cloud-analytics (port 8084)
                                       │
                                       ├─► INSERT → mentor_planta_14.linea_14.paradas
                                       │           (scope JWT: por planta/linea)
                                       │
                                       └─► Dispatch CREAR_PARADA ──► edge-gateway (port 8005)
                                                                           │
                                                                           └─► INSERT → mentor_edge.linea_1.stops
                                                                                       (DB del Jetson)

SYNC_PARADAS (periódico):
  cloud-gateway ──► lee mentor_cloud.analytics.paradas  ← tabla DISTINTA
                        (≠ mentor_planta_14.linea_14.paradas)
```

### Bases de datos relevantes

| Pool | DB | Tabla | Uso |
|------|----|-------|-----|
| Master | `mentor_cloud` | `config.dispositivos` | Registro de dispositivos edge (device_id, linea_id) |
| Master | `mentor_cloud` | `analytics.paradas` | Lecturas internas, SYNC_PARADAS |
| Per-planta | `mentor_planta_14` | `linea_14.paradas` | Escrituras API con scope JWT |
| Edge | `mentor_edge` | `linea_1.stops` | DB local del Jetson Orin |

---

## Línea de tiempo del incidente

| Hora | Evento |
|------|--------|
| Sesión anterior | Se detecta que paradas creadas en cloud aparecen como "Asignado" en tablet cloud pero no en tablet edge |
| Sesión anterior | Se confirma que la tablet edge servía JS antiguo (sin fix de `await justifyStop`) |
| Sesión anterior | Se actualiza el dist de la tablet edge y se despliega código con `await justifyStop` |
| Esta sesión | Se descubre que edge DB solo tiene 2 paradas del turno actual, cloud tiene 11 |
| Esta sesión | Se identifica que 9 de 11 paradas tienen `device_id=''` en cloud |
| Esta sesión | Se encuentra la causa raíz: `ResolveDeviceID` consultaba DB incorrecta |
| Esta sesión | Se aplican 3 fixes en `cloud-analytics` y se despliega |
| Esta sesión | Se verifica E2E: parada tipo MECANICA creada en cloud → aparece en edge DB |
| Esta sesión | Se migran datos históricos: 16 paradas actualizadas con `device_id='1'` |
| Esta sesión | Se sincronizan manualmente 9 paradas del turno actual al edge |

---

## Causa raíz

### Bug principal: `ResolveDeviceID` en `stop_repo.go`

**Código roto:**
```go
func (r *stopRepo) ResolveDeviceID(ctx context.Context, lineaID int) (string, error) {
    // ❌ Resolve(ctx) retorna el pool "mentor_planta_14" (por-planta, con scope JWT)
    pool, schema, err := r.pr.Resolve(ctx)
    if err != nil {
        return "", err
    }
    // ❌ Tbl("linea_14", "config.dispositivos") genera "linea_14.dispositivos"
    //    Esta tabla NO EXISTE en mentor_planta_14
    tbl := multitenancy.Tbl(schema, "config.dispositivos")
    var deviceID string
    err = pool.QueryRow(ctx,
        `SELECT device_id FROM `+tbl+` WHERE linea_id = $1 AND activo = true LIMIT 1`,
        lineaID,
    ).Scan(&deviceID)
    return deviceID, err  // Siempre retornaba error o vacío
}
```

**Código corregido:**
```go
func (r *stopRepo) ResolveDeviceID(ctx context.Context, lineaID int) (string, error) {
    // ✅ Master() retorna el pool "mentor_cloud" donde SÍ existe config.dispositivos
    pool := r.pr.Master()
    var deviceID string
    err := pool.QueryRow(ctx,
        `SELECT device_id FROM config.dispositivos WHERE linea_id = $1 AND activo = true LIMIT 1`,
        lineaID,
    ).Scan(&deviceID)
    return deviceID, err
}
```

**¿Por qué nadie lo notó antes?** La función retornaba error silenciosamente. El código continuaba con `device_id=""` sin fallar. Solo al llegar al guard `if stop.DeviceID != ""` se descartaba el dispatch, sin log de error visible.

---

## Fixes aplicados

### Fix 1 — `stop_repo.go`: ResolveDeviceID usa Master pool

**Archivo:** `mentor-cloud/services/cloud-analytics/internal/adapters/repository/stop_repo.go`

Cambiar de `r.pr.Resolve(ctx)` → `r.pr.Master()` para consultar `mentor_cloud.config.dispositivos`.

---

### Fix 2 — `stop_handler.go`: Resolver device_id ANTES del INSERT

**Archivo:** `mentor-cloud/services/cloud-analytics/internal/adapters/handler/stop_handler.go`

**Problema:** El Create handler resolvía `device_id` *después* de insertar en DB. El registro quedaba con `device_id=''`, y cuando llegaba al dispatch check `if req.DeviceID != ""` ya era tarde.

**Antes:**
```go
stop, err := h.repo.CreateStop(ctx, &req)   // ← INSERT con device_id=''
if err != nil { ... }
if req.DeviceID == "" && req.LineaID != nil {
    req.DeviceID, _ = h.repo.ResolveDeviceID(ctx, *req.LineaID)
}
if h.dispatcher != nil && stop.DeviceID != "" { // ← stop.DeviceID sigue siendo ''
    // nunca entra aquí
}
```

**Después:**
```go
// 1. Resolver device_id ANTES del INSERT
if req.DeviceID == "" && req.LineaID != nil {
    if resolved, err := h.repo.ResolveDeviceID(c.Request.Context(), *req.LineaID); err == nil {
        req.DeviceID = resolved   // ← ahora = "1"
    }
}
// 2. INSERT con device_id ya populado
stop, err := h.repo.CreateStop(ctx, &req)
// 3. Dispatch con device_id correcto
if h.dispatcher != nil && req.DeviceID != "" {
    // ✅ entra aquí, dispatcha CREAR_PARADA al Jetson
}
```

---

### Fix 3 — `stop_handler.go`: Fallback ResolveDeviceID en Update handler

**Archivo:** `mentor-cloud/services/cloud-analytics/internal/adapters/handler/stop_handler.go`

Para paradas históricas con `device_id=''`, el Update handler ahora intenta resolver el device_id antes de desechar el dispatch:

```go
if h.dispatcher != nil {
    devID := stop.DeviceID
    if devID == "" && stop.LineaID != nil {
        if resolved, err := h.repo.ResolveDeviceID(ctx, *stop.LineaID); err == nil {
            devID = resolved
        }
    }
    if devID != "" {
        // dispatch MODIFICAR_PARADA al edge
    }
}
```

---

### Fix 4 — `DashboardView.vue`: Race condition en justifyStop

**Archivo:** `mentor-apps/mentor-tablet-app/src/views/DashboardView.vue`

**Problema:** `handleCreateStop` llamaba `stopsStore.justifyStop()` sin `await`. La parada llegaba al edge sin `justified=true` porque el JUSTIFICAR_PARADA se lanzaba de forma fire-and-forget.

**Antes:**
```javascript
const created = await stopsStore.createStop(req)
stopsStore.justifyStop(created.stop_id, justifyData)  // ← sin await
```

**Después:**
```javascript
const created = await stopsStore.createStop(req)
await stopsStore.justifyStop(created.stop_id, justifyData)  // ← con await
```

---

### Fix 5 — `stop_handler.go`: Guard en Justify handler para device_id vacío

**Archivo:** `mentor-cloud/services/cloud-analytics/internal/adapters/handler/stop_handler.go`

Antes del fix de ResolveDeviceID existente, el Justify handler llamaba al dispatcher con `device_id=''`, lo que causaba que el edge-gateway devolviera HTTP 400. Se agregó el guard `if stop.DeviceID != ""`.

---

## Migración de datos históricos

Como existían paradas con `device_id=''` previas al fix, se ejecutaron estas correcciones manuales:

```sql
-- En mentor_planta_14: actualizar paradas sin device_id
UPDATE linea_14.paradas SET device_id = '1' WHERE device_id = '';
-- → 16 rows updated

-- En mentor_edge (Jetson): insertar las 9 paradas del turno actual
-- que nunca llegaron por el bug
INSERT INTO linea_1.stops (...) VALUES (...) × 9
ON CONFLICT (stop_id) DO UPDATE SET ...;
-- → INSERT 0 9
```

---

## Problema arquitectural identificado (deuda técnica)

Existe una **inconsistencia estructural** entre dos tablas de paradas:

| Tabla | Quién escribe | Quién lee |
|-------|--------------|-----------|
| `mentor_planta_14.linea_14.paradas` | API `/api/stops` (JWT scope) | Tablet app, cloud-analytics |
| `mentor_cloud.analytics.paradas` | Internal services | SYNC_PARADAS, reportes |

**Impacto:** SYNC_PARADAS (mecanismo de recuperación cloud→edge) lee de `analytics.paradas`, pero el API escribe en `linea_14.paradas`. Son tablas físicamente distintas.

**Mitigación actual:** El handler de Create ahora despacha `CREAR_PARADA` inmediatamente al crear (entrega directa), por lo que SYNC_PARADAS solo es el mecanismo de fallback.

**Solución pendiente (largo plazo):** Unificar ambas tablas o hacer que SYNC_PARADAS filtre también desde las tablas por-planta agrupadas por `device_id`.

---

## Lecciones aprendidas

1. **Errores silenciosos ocultan bugs críticos.** `ResolveDeviceID` fallaba sin log. Se debe loggear siempre los errores de resolución de identidad de dispositivo.

2. **El multitenancy tiene doble pool.** En este sistema, `Resolve(ctx)` → pool por-planta, `Master()` → pool global. Siempre verificar cuál es el correcto para cada query.

3. **`device_id` debe resolverse antes de persistir**, no después. Si el registro llega a DB sin device_id, pierde el ciclo de despacho.

4. **`stop_type: "manual"` no es válido en el edge.** Solo son válidos los tipos en el enum del dominio `domain/stop.go`. Usar `MECANICA`, `ELECTRICA`, `OTRA`, etc.

5. **Dos tablas para el mismo dato es una bomba de tiempo.** `linea_14.paradas` y `analytics.paradas` deben unificarse.

---

## Verificación post-fix

```bash
# Confirmar ResolveDeviceID funciona (log en cloud-analytics):
docker logs mentor-cloud-analytics | grep "ResolveDeviceID"
# Esperado: ResolveDeviceID linea=14 -> device=1

# Confirmar parada llega al edge:
docker exec docker-postgres-1 psql -U mentor -d mentor_edge \
  -c "SELECT stop_id, stop_type, justified FROM linea_1.stops ORDER BY created_at DESC LIMIT 5;"

# Confirmar comando fue ejecutado en edge:
docker exec docker-postgres-1 psql -U mentor -d mentor_edge \
  -c "SELECT command_type, status FROM gateway.commands ORDER BY created_at DESC LIMIT 5;"
```

---

## Archivos modificados (commit `a301bbe`)

| Archivo | Cambio |
|---------|--------|
| `cloud-analytics/internal/adapters/repository/stop_repo.go` | `ResolveDeviceID`: Master pool en lugar de Resolve |
| `cloud-analytics/internal/adapters/handler/stop_handler.go` | Create: resolver antes de INSERT; Update: fallback ResolveDeviceID; Justify: guard device_id |
| `mentor-tablet-app/src/views/DashboardView.vue` | `await justifyStop` en handleCreateStop |

