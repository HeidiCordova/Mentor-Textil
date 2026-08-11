<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import TimelineCanvas from '@/components/TimelineCanvas.vue'
import StopAssignmentModal from '@/components/StopAssignmentModal.vue'
import ProductAssignmentModal from '@/components/ProductAssignmentModal.vue'
import MermaRegistrationModal from '@/components/MermaRegistrationModal.vue'
import SvgIcon from '@/components/SvgIcon.vue'
import { useStopsStore } from '@/stores/stops'
import { useMachineStore } from '@/stores/machine'
import { useCatalogStore } from '@/stores/catalog'
import type { HitResult } from '@/composables/useTimeline'
import type { Stop, StopType } from '@/types'
import { useUIStore } from '@/stores/ui'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { useConnectionStore } from '@/stores/connection'
import { api, getApiMode, setCloudLineaId, getCloudJWT } from '@/services/api'
import { sse } from '@/services/sse'
import { useProductionRunsStore } from '@/stores/productionRuns'
import { useTurnosStore } from '@/stores/turnos'
import { useConfigStore } from '@/stores/config'

const stopsStore = useStopsStore()
const machineStore = useMachineStore()
const catalogStore = useCatalogStore()
const pl = usePlantasLineasStore()
const connection = useConnectionStore()
const productionRunsStore = useProductionRunsStore()
const turnosStore = useTurnosStore()
const configStore = useConfigStore()

function scopeParams() {
  return {
    linea_id: pl.selectedLineaId ?? undefined,
    planta_id: pl.selectedPlantaId ?? undefined,
    empresa_id: (connection.operator?.empresa_id ?? undefined) as number | undefined
  }
}

function runsSince(): string {
  return new Date(Date.now() - 24 * 3600_000).toISOString()
}

const timelineRef = ref<InstanceType<typeof TimelineCanvas> | null>(null)
const showStopAssignment = ref(false)
const stopModalMode = ref<'create' | 'justify'>('create')
const stopTimeRange = ref({ start: 0, end: 0 })
const stopSlotRange = ref<{ start: number; end: number } | null>(null)
const showProductAssignment = ref(false)
const showMultiAssign = ref(false)
const showMermaModal = ref(false)
const selectedStop = ref<Stop | null>(null)

// ── Contador HOY (misma lógica que ProduccionView) ──────────────────────────
const todayCount = computed(() => {
  const midnight = new Date()
  midnight.setHours(0, 0, 0, 0)
  const ms = midnight.getTime()
  return machineStore.recentEvents.filter(e =>
    new Date(e.timestamp).getTime() >= ms &&
    (e.event_type === 'CORTE' || e.event_type === 'cut_detected')
  ).length
})

// ── Estado en vivo — polling Python directo (independiente de ProduccionView) ─
const liveState    = ref<'producing' | 'idle_wait' | 'stop_open' | 'offline'>('offline')
const liveIdleSecs = ref(0)
let _liveTimer: ReturnType<typeof setInterval> | null = null
let _tickTimer: ReturnType<typeof setInterval> | null = null
let _idleStartMs = 0

async function _pollLiveState() {
  try {
    const d = await fetch('/vision/status', { signal: AbortSignal.timeout(2000) }).then(r => r.json())
    const st: string = d.stop_tracker_state ?? 'offline'
    liveState.value = st as typeof liveState.value
    if (st === 'producing' || st === 'offline') {
      _idleStartMs = 0
      liveIdleSecs.value = 0
    } else {
      const pyS: number = d.idle_duration_s ?? 0
      _idleStartMs = Date.now() - pyS * 1000
      liveIdleSecs.value = Math.floor(pyS)
    }
  } catch {
    liveState.value = 'offline'
    _idleStartMs = 0
  }
}

function _tickLive() {
  if (_idleStartMs > 0)
    liveIdleSecs.value = Math.floor((Date.now() - _idleStartMs) / 1000)
}

const liveLabel = computed(() => {
  if (liveState.value === 'producing') return 'PRODUCIENDO'
  if (liveState.value === 'idle_wait') return 'MICROPARADA'
  if (liveState.value === 'stop_open') return 'DETENIDO'
  return 'OFFLINE'
})

function fmtSecs(s: number): string {
  const m = Math.floor(s / 60)
  const ss = s % 60
  if (m >= 60) {
    const h = Math.floor(m / 60)
    return `${h}:${String(m % 60).padStart(2, '0')}:${String(ss).padStart(2, '0')}`
  }
  return `${m}:${String(ss).padStart(2, '0')}`
}

const productTimeRange = ref({ start: 0, end: 0 })
const uiStore = useUIStore()

// React to header triggers
watch(() => uiStore.goToNowTrigger, async () => {
  // Primero recargar datos de las últimas 24h, luego hacer scroll
  const p = scopeParams()
  try {
    await Promise.allSettled([
      stopsStore.fetchStops({ ...p, since: runsSince(), limit: 500 }),
      productionRunsStore.fetchRuns({ ...p, since: runsSince(), limit: 500 })
    ])
  } catch { /* offline */ }
  timelineRef.value?.scrollToNow?.()
})

watch(() => uiStore.calendarGoToTrigger, async () => {
  const targetMs = uiStore.calendarTargetMs
  if (!targetMs) return
  const since = new Date(targetMs).toISOString()
  const until = new Date(targetMs + 24 * 3600_000).toISOString()
  const p = scopeParams()
  try {
    await Promise.allSettled([
      stopsStore.fetchStops({ ...p, since, until, limit: 500 }),
      productionRunsStore.fetchRuns({ ...p, since, until, limit: 500 })
    ])
  } catch { /* offline */ }
  timelineRef.value?.scrollToDate?.(targetMs)
})

watch(() => uiStore.registerStopTrigger, () => {
  stopModalMode.value = 'create'
  stopTimeRange.value = { start: Date.now() - 30 * 60_000, end: Date.now() }
  selectedStop.value = null
  showStopAssignment.value = true
})

function handleHitResult(result: HitResult): void {
  if ((result.lane === 'unassigned' || result.lane === 'assigned') && result.block.stopId) {
    const stop = stopsStore.stops.find((s) => s.stop_id === result.block.stopId)
    if (stop) {
      // Calcular rango de slot clickeado y verificar si es parcial (la parada excede el slot)
      stopSlotRange.value = null
      if (result.slotStart !== undefined && stop.ended_at) {
        const slotMs = ((configStore.config?.oee as Record<string, unknown> | undefined)?.snapshot_interval_s as number ?? 1800) * 1000
        const slotStart = result.slotStart
        const slotEnd = slotStart + slotMs
        const stopStartMs = new Date(stop.started_at).getTime()
        const stopEndMs = new Date(stop.ended_at).getTime()
        const hasBefore = slotStart > stopStartMs + 1000
        const hasAfter = slotEnd < stopEndMs - 1000
        if (hasBefore || hasAfter) {
          stopSlotRange.value = {
            start: Math.max(slotStart, stopStartMs),
            end: Math.min(slotEnd, stopEndMs)
          }
        }
      }
      selectedStop.value = stop
      stopModalMode.value = 'justify'
      showStopAssignment.value = true
    }
  } else if ((result.lane === 'unassigned' || result.lane === 'assigned') && !result.block.stopId) {
    stopModalMode.value = 'create'
    stopTimeRange.value = { start: result.block.start, end: result.block.end }
    selectedStop.value = null
    showStopAssignment.value = true
  } else if (result.lane === 'production') {
    // Usar el slot exacto clickeado como inicio (no el inicio del run completo),
    // así la asignación aplica "desde aquí hacia adelante"
    const start = result.slotStart ?? result.block.start
    productTimeRange.value = { start, end: result.block.end }
    showProductAssignment.value = true
  }
}

async function handleAssignStop(stopId: string, categoriaId: number, category: string, tipoParada: string): Promise<void> {
  // Si hay un rango de slot seleccionado, hacer split en lugar de justificar la parada completa
  if (stopSlotRange.value) {
    const slotRange = stopSlotRange.value
    const originalStop = stopsStore.stops.find((s) => s.stop_id === stopId)
    stopSlotRange.value = null
    showStopAssignment.value = false
    selectedStop.value = null
    if (originalStop) await doSplitAndJustify(originalStop, categoriaId, category, tipoParada, slotRange)
    return
  }

  const idx = stopsStore.stops.findIndex((s) => s.stop_id === stopId)
  const stopType = (tipoParada || 'OTRA') as Stop['stop_type']
  const originalStop = idx >= 0 ? { ...stopsStore.stops[idx] } : null
  if (idx >= 0) {
    stopsStore.stops.splice(idx, 1, {
      ...stopsStore.stops[idx],
      justified: true,
      stop_type: stopType,
      reason: category,
      category,
      categoria_id: categoriaId,
      justified_by: 'operator',
      justified_at: new Date().toISOString()
    })
  }
  timelineRef.value?.syncData()
  showStopAssignment.value = false
  selectedStop.value = null
  const ok = await stopsStore.justifyStop(stopId, {
    stop_type: stopType,
    reason: category,
    category,
    categoria_id: categoriaId,
    justified_by: 'operator'
  })
  if (!ok && originalStop) {
    const revertIdx = stopsStore.stops.findIndex((s) => s.stop_id === stopId)
    if (revertIdx >= 0) stopsStore.stops.splice(revertIdx, 1, originalStop)
    timelineRef.value?.syncData()
  }
}

async function doSplitAndJustify(
  originalStop: Stop,
  categoriaId: number,
  category: string,
  tipoParada: string,
  slotRange: { start: number; end: number }
): Promise<void> {
  const stopType = (tipoParada || 'OTRA') as Stop['stop_type']
  const stopStartMs = new Date(originalStop.started_at).getTime()
  const stopEndMs = new Date(originalStop.ended_at!).getTime()
  const { start: slotStart, end: slotEnd } = slotRange
  const hasBefore = slotStart > stopStartMs + 1000
  const hasAfter = slotEnd < stopEndMs - 1000
  const scope = {
    linea_id: pl.selectedLineaId ?? undefined,
    planta_id: pl.selectedPlantaId ?? undefined,
    empresa_id: pl.plantaActual?.empresa_id ?? connection.operator?.empresa_id ?? undefined
  }

  // Optimistic: quitar la parada original y agregar fragmentos temporales
  const origIdx = stopsStore.stops.findIndex((s) => s.stop_id === originalStop.stop_id)
  if (origIdx >= 0) stopsStore.stops.splice(origIdx, 1)

  const now = new Date().toISOString()
  const tmpBefore = `tmp-before-${Date.now()}`
  const tmpSlot = `tmp-slot-${Date.now()}`
  const tmpAfter = `tmp-after-${Date.now()}`

  if (hasBefore) {
    stopsStore.stops.push({ stop_id: tmpBefore, stop_type: 'OTRA', started_at: originalStop.started_at, ended_at: new Date(slotStart).toISOString(), justified: false, source: 'operator' } as any)
  }
  stopsStore.stops.push({ stop_id: tmpSlot, stop_type: stopType, started_at: new Date(slotStart).toISOString(), ended_at: new Date(slotEnd).toISOString(), justified: true, reason: category, category, categoria_id: categoriaId, justified_by: 'operator', justified_at: now, source: 'operator' } as any)
  if (hasAfter) {
    stopsStore.stops.push({ stop_id: tmpAfter, stop_type: 'OTRA', started_at: new Date(slotEnd).toISOString(), ended_at: originalStop.ended_at, justified: false, source: 'operator' } as any)
  }
  timelineRef.value?.syncData()

  try {
    // 1. Eliminar parada original del backend
    await api.deleteStop(originalStop.stop_id, scope.linea_id, scope.planta_id)

    // 2. Crear fragmento anterior (unjustified)
    if (hasBefore) {
      const created = await stopsStore.createStop({ stop_type: 'OTRA', started_at: originalStop.started_at, ended_at: new Date(slotStart).toISOString(), source: 'operator', ...scope })
      if (created) {
        const i = stopsStore.stops.findIndex((s) => s.stop_id === tmpBefore)
        if (i >= 0) stopsStore.stops.splice(i, 1, created)
      }
    }

    // 3. Crear y justificar el fragmento del slot clickeado
    const createdSlot = await stopsStore.createStop({ stop_type: stopType, started_at: new Date(slotStart).toISOString(), ended_at: new Date(slotEnd).toISOString(), category, categoria_id: categoriaId, reason: category, source: 'operator', ...scope })
    if (createdSlot) {
      await stopsStore.justifyStop(createdSlot.stop_id, { stop_type: stopType, reason: category, category, categoria_id: categoriaId, justified_by: 'operator' })
      const i = stopsStore.stops.findIndex((s) => s.stop_id === tmpSlot)
      if (i >= 0) stopsStore.stops.splice(i, 1, createdSlot)
    }

    // 4. Crear fragmento posterior (unjustified)
    if (hasAfter) {
      const created = await stopsStore.createStop({ stop_type: 'OTRA', started_at: new Date(slotEnd).toISOString(), ended_at: originalStop.ended_at!, source: 'operator', ...scope })
      if (created) {
        const i = stopsStore.stops.findIndex((s) => s.stop_id === tmpAfter)
        if (i >= 0) stopsStore.stops.splice(i, 1, created)
      }
    }

    timelineRef.value?.syncData()
  } catch (e) {
    console.error('[split] falló, revirtiendo', e)
    stopsStore.stops = stopsStore.stops.filter((s) => s.stop_id !== tmpBefore && s.stop_id !== tmpSlot && s.stop_id !== tmpAfter)
    stopsStore.stops.push(originalStop)
    timelineRef.value?.syncData()
  }
}

async function handleCreateStop(stopType: StopType, categoriaId: number, category: string, startMs: number, endMs: number): Promise<void> {
  const tempId = `tmp-${Date.now()}`
  stopsStore.stops.unshift({
    stop_id: tempId,
    stop_type: stopType,
    started_at: new Date(startMs).toISOString(),
    ended_at: new Date(endMs).toISOString(),
    justified: true,
    reason: category,
    category,
    categoria_id: categoriaId,
    justified_by: 'operator',
    justified_at: new Date().toISOString(),
    source: 'operator'
  } as any)
  timelineRef.value?.syncData()
  showStopAssignment.value = false
  selectedStop.value = null
  stopsStore.createStop({
    stop_type: stopType,
    started_at: new Date(startMs).toISOString(),
    ended_at: new Date(endMs).toISOString(),
    category,
    categoria_id: categoriaId,
    reason: category,
    source: 'operator',
    linea_id: pl.selectedLineaId ?? undefined,
    planta_id: pl.selectedPlantaId ?? undefined,
    empresa_id: pl.plantaActual?.empresa_id ?? connection.operator?.empresa_id ?? undefined
  }).then(async (created) => {
    const idx = stopsStore.stops.findIndex((s) => s.stop_id === tempId)
    if (created) {
      const realStop = { ...created, justified: true, reason: category, category, categoria_id: categoriaId }
      if (idx >= 0) stopsStore.stops.splice(idx, 1, realStop)
      timelineRef.value?.syncData()
      // Persist justified=true server-side (createStop does NOT set justified)
      await stopsStore.justifyStop(created.stop_id, {
        stop_type: stopType,
        reason: category,
        category,
        categoria_id: categoriaId,
        justified_by: 'operator'
      })
    } else if (idx >= 0) {
      stopsStore.stops.splice(idx, 1)
    }
  }).catch(() => {})
}

function handleAssignProduct(
  productoId: number | null,
  sku: string | null,
  description: string | null,
  startMs: number,
  endMs: number
): void {
  productionRunsStore.upsert({
    producto_id: productoId ?? undefined,
    sku: sku ?? undefined,
    nombre: description ?? undefined,
    started_at: new Date(startMs).toISOString(),
    ended_at: endMs > 0 ? new Date(endMs).toISOString() : undefined,
    linea_id: pl.selectedLineaId ?? undefined
  }, {
    linea_id: pl.selectedLineaId ?? undefined,
    planta_id: pl.selectedPlantaId ?? undefined
  }).catch(() => {})
  showProductAssignment.value = false
}

async function handleMultiAssign(categoriaId: number, category: string, tipoParada: string): Promise<void> {
  const ids = [...uiStore.multiSelectedIds]  // snapshot before clear
  showMultiAssign.value = false
  uiStore.clearMultiSelected()
  uiStore.toggleMultiSelect()
  const stops = stopsStore.stops
  const stopType = (tipoParada || 'OTRA') as Stop['stop_type']
  for (const stopId of ids) {
    const idx = stops.findIndex((s) => s.stop_id === stopId)
    if (idx >= 0) {
      stops.splice(idx, 1, {
        ...stops[idx],
        justified: true,
        stop_type: stopType,
        reason: category,
        category,
        categoria_id: categoriaId,
        justified_by: connection.operator?.nombre ?? connection.operator?.username ?? 'operator',
        justified_at: new Date().toISOString()
      })
    }
  }
  timelineRef.value?.syncData()
  await Promise.all(ids.map(stopId =>
    stopsStore.justifyStop(stopId, {
      stop_type: stopType,
      reason: category,
      category,
      categoria_id: categoriaId,
      justified_by: connection.operator?.nombre ?? connection.operator?.username ?? 'operator'
    })
  ))
}

watch([() => pl.selectedLineaId, () => pl.selectedPlantaId], async () => {
  timelineRef.value?.resetAutoFit?.()
  if (pl.selectedLineaId) setCloudLineaId(pl.selectedLineaId)
  // Reconectar SSE con empresa_id correcto (necesario para superadmin que no tiene empresa_id en JWT)
  const empresaId = pl.plantaActual?.empresa_id ?? connection.operator?.empresa_id ?? undefined
  if (empresaId && connection.cloudURL) {
    const jwt = getCloudJWT()
    if (jwt) sse.connectCloud(connection.cloudURL, jwt, empresaId)
  }
  await turnosStore.fetchTurnos({ planta_id: pl.selectedPlantaId ?? undefined })
  const p = scopeParams()
  try {
    await Promise.allSettled([
      configStore.fetchConfig(),
      stopsStore.fetchStops({ ...p, since: runsSince(), limit: 500 }),
      stopsStore.fetchSummary(24, p),
      productionRunsStore.fetchRuns({ ...p, since: runsSince(), limit: 500 }),
      catalogStore.fetchStopCategories(pl.selectedLineaId ?? undefined),
      catalogStore.fetchProducts(pl.selectedLineaId ?? undefined),
      machineStore.fetchStatus(pl.selectedLineaId ?? undefined),
      machineStore.fetchBuffer(pl.selectedLineaId ?? undefined),
      machineStore.fetchRecentEvents(1000, runsSince(), pl.selectedLineaId ?? undefined)
    ])
  } catch { /* offline */ }
  timelineRef.value?.scrollToNow()
})

onMounted(async () => {
  catalogStore.loadAll(pl.selectedLineaId ?? undefined)
  stopsStore.bindSSE()
  machineStore.bindSSE()
  catalogStore.bindSSE()
  productionRunsStore.bindSSE()

  // Reconectar SSE con empresa_id (para superadmin que no tiene empresa_id en JWT)
  if (connection.cloudURL) {
    const jwt = getCloudJWT()
    const empresaId = pl.plantaActual?.empresa_id ?? connection.operator?.empresa_id ?? undefined
    if (jwt && empresaId) sse.connectCloud(connection.cloudURL, jwt, empresaId)
  }

  await turnosStore.fetchTurnos({ planta_id: pl.selectedPlantaId ?? undefined })
  const p = scopeParams()
  if (pl.selectedLineaId) setCloudLineaId(pl.selectedLineaId)
  try {
    await Promise.allSettled([
      configStore.fetchConfig(),
      stopsStore.fetchStops({ ...p, since: runsSince(), limit: 500 }),
      stopsStore.fetchSummary(24, p),
      machineStore.fetchStatus(pl.selectedLineaId ?? undefined),
      machineStore.fetchBuffer(pl.selectedLineaId ?? undefined),
      machineStore.fetchRecentEvents(1000, runsSince(), pl.selectedLineaId ?? undefined),
      productionRunsStore.fetchRuns({ ...p, since: runsSince(), limit: 500 })
    ])
  } catch { /* offline */ }
  timelineRef.value?.fitToMode()

  // Polling estado en vivo desde Python (independiente de ProduccionView)
  _pollLiveState()
  _liveTimer = setInterval(_pollLiveState, 2000)
  _tickTimer = setInterval(_tickLive, 500)

  // Polling automático para mantener datos frescos (en CLOUD es el único mecanismo;
  // en EDGE complementa el SSE para production runs que no tienen trigger de NOTIFY)
  const pollInterval = getApiMode() === 'CLOUD' ? 20_000 : 30_000
  _pollTimer = setInterval(async () => {
    const pp = scopeParams()
    try {
      await Promise.allSettled([
        stopsStore.fetchStops({ ...pp, since: runsSince(), limit: 500 }),
        stopsStore.fetchSummary(24, pp),
        productionRunsStore.fetchRuns({ ...pp, since: runsSince(), limit: 500 }),
        machineStore.fetchStatus(),
        machineStore.fetchBuffer(),
        machineStore.fetchRecentEvents(1000, runsSince(), pp.linea_id)
      ])
      timelineRef.value?.syncData()
    } catch { /* offline */ }
  }, pollInterval)
})

let _pollTimer: ReturnType<typeof setInterval> | null = null

onUnmounted(() => {
  if (_pollTimer !== null) {
    clearInterval(_pollTimer)
    _pollTimer = null
  }
  if (_liveTimer !== null) {
    clearInterval(_liveTimer)
    _liveTimer = null
  }
  if (_tickTimer !== null) {
    clearInterval(_tickTimer)
    _tickTimer = null
  }
})
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="relative flex-1 min-h-0">
      <TimelineCanvas ref="timelineRef" @hit-result="handleHitResult" />

      <!-- Barra flotante de selección múltiple -->
      <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0 -translate-y-2" leave-active-class="transition-all duration-150" leave-to-class="opacity-0 -translate-y-2">
        <div
          v-if="uiStore.multiSelectMode"
          class="absolute top-3 left-1/2 -translate-x-1/2 z-20 flex items-center gap-3 px-5 py-2.5 bg-cyan-500/95 backdrop-blur-sm rounded-full shadow-xl border border-cyan-400/50"
        >
          <span class="text-white font-semibold text-sm">
            {{ uiStore.multiSelectedIds.length > 0
              ? `${uiStore.multiSelectedIds.length} parada${uiStore.multiSelectedIds.length !== 1 ? 's' : ''} seleccionada${uiStore.multiSelectedIds.length !== 1 ? 's' : ''}`
              : 'Toca las paradas para seleccionar'
            }}
          </span>
          <button
            v-if="uiStore.multiSelectedIds.length > 0"
            class="px-4 py-1 bg-white text-cyan-700 font-bold text-sm rounded-full hover:bg-cyan-50 transition-colors"
            @click="showMultiAssign = true"
          >
            Asignar
          </button>
        </div>
      </Transition>

      <!-- Contador HOY detecciones (validadas por cámara) -->
      <div class="absolute top-3 left-3 z-10 flex items-center gap-2 px-3 py-1.5 bg-slate-900/90 backdrop-blur-sm rounded-full border border-slate-700/50 shadow-lg">
        <span class="text-[10px] font-semibold text-gray-400 uppercase">Hoy</span>
        <span class="text-lg font-bold font-mono text-white tabular-nums">{{ todayCount }}</span>
        <span class="text-[10px] text-gray-500">detecciones</span>
      </div>

      <!-- Estado en vivo desde Python (independiente de ProduccionView) -->
      <div class="absolute top-14 left-3 z-10 flex items-center gap-2 px-3 py-1.5 backdrop-blur-sm rounded-full border shadow-lg"
        :class="liveState === 'producing'
          ? 'bg-emerald-900/90 border-emerald-500/40'
          : liveState === 'idle_wait'
            ? 'bg-amber-900/90 border-amber-500/40'
            : liveState === 'stop_open'
              ? 'bg-red-900/90 border-red-500/40'
              : 'bg-slate-900/90 border-slate-600/40'">
        <span class="w-2.5 h-2.5 rounded-full shrink-0"
          :class="liveState === 'producing'
            ? 'bg-emerald-400 animate-pulse'
            : liveState === 'idle_wait'
              ? 'bg-amber-400 animate-pulse'
              : liveState === 'stop_open'
                ? 'bg-red-400 animate-pulse'
                : 'bg-gray-500'" />
        <span class="text-sm font-bold"
          :class="liveState === 'producing'
            ? 'text-emerald-300'
            : liveState === 'idle_wait'
              ? 'text-amber-300'
              : liveState === 'stop_open'
                ? 'text-red-300'
                : 'text-gray-400'">{{ liveLabel }}</span>
        <span v-if="liveState !== 'producing' && liveState !== 'offline'"
          class="text-sm font-mono tabular-nums"
          :class="liveState === 'idle_wait' ? 'text-amber-200' : 'text-red-200'">{{ fmtSecs(liveIdleSecs) }}</span>
      </div>

      <button
        class="absolute bottom-4 right-16 flex items-center gap-1.5 px-4 py-2.5 rounded-full bg-orange-600 text-white font-bold text-sm hover:bg-orange-500 transition-all shadow-lg shadow-orange-900/30 z-10"
        @click="showMermaModal = true"
      >
        <SvgIcon name="plus" :size="16" />
        + Merma
      </button>
    </div>

    <Teleport to="body">
      <StopAssignmentModal
        v-if="showStopAssignment"
        :mode="stopModalMode"
        :stop="selectedStop ?? undefined"
        :time-start="stopTimeRange.start"
        :time-end="stopTimeRange.end"
        :slot-time-start="stopSlotRange?.start"
        :slot-time-end="stopSlotRange?.end"
        @assign="handleAssignStop"
        @create="handleCreateStop"
        @cancel="showStopAssignment = false; selectedStop = null; stopSlotRange = null"
      />
    </Teleport>

    <Teleport to="body">
      <StopAssignmentModal
        v-if="showMultiAssign"
        mode="multi"
        :multi-count="uiStore.multiSelectedIds.length"
        @multi-assign="handleMultiAssign"
        @cancel="showMultiAssign = false"
      />
    </Teleport>

    <Teleport to="body">
      <ProductAssignmentModal
        v-if="showProductAssignment"
        :time-start="productTimeRange.start"
        :time-end="productTimeRange.end"
        @assign="handleAssignProduct"
        @cancel="showProductAssignment = false"
      />
    </Teleport>

    <Teleport to="body">
      <MermaRegistrationModal
        v-if="showMermaModal"
        @saved="showMermaModal = false"
        @cancel="showMermaModal = false"
      />
    </Teleport>


  </div>
</template>
