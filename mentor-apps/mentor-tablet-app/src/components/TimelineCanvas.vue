<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useTimeline, type HitResult } from '@/composables/useTimeline'
import { useStopsStore } from '@/stores/stops'
import { useMachineStore } from '@/stores/machine'
import { useProductionRunsStore } from '@/stores/productionRuns'
import { useTurnosStore } from '@/stores/turnos'
import { useUIStore } from '@/stores/ui'

const emit = defineEmits<{
  hitResult: [result: HitResult]
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const containerRef = ref<HTMLDivElement | null>(null)

const timeline = useTimeline(canvasRef)
const stopsStore = useStopsStore()
const machineStore = useMachineStore()
const productionRunsStore = useProductionRunsStore()
const turnosStore = useTurnosStore()
const uiStore = useUIStore()

let resizeObserver: ResizeObserver | null = null
let touchState = { startX: 0, startMs: 0, pinchDist: 0 }

function syncData(): void {
  const shiftMs = turnosStore.shiftSince() ? new Date(turnosStore.shiftSince()).getTime() : 0
  timeline.updateData(stopsStore.stops, productionRunsStore.runs, machineStore.recentEvents, shiftMs)
}

function panLeft16m(): void  { timeline.panBy(-16 * 60_000) }
function panLeft1h(): void   { timeline.panBy(-3600_000) }
function panRight16m(): void { timeline.panBy(16 * 60_000) }
function panRight1h(): void  { timeline.panBy(3600_000) }

function handleCanvasClick(e: MouseEvent): void {
  const hit = timeline.hitTest(e.clientX, e.clientY)
  if (uiStore.multiSelectMode) {
    if (hit?.lane === 'unassigned') {
      // Si el hit ya tiene stopId (parada real) úsalo directo.
      // Si no (bloque sintético), buscar en el store qué parada no asignada cubre ese momento.
      const stopId = hit.block.stopId ?? (() => {
        const midMs = hit.block.start + (hit.block.end - hit.block.start) / 2
        const covering = stopsStore.stops.find(s =>
          !s.justified &&
          new Date(s.started_at).getTime() <= midMs &&
          (!s.ended_at || new Date(s.ended_at).getTime() > midMs)
        )
        return covering?.stop_id ?? null
      })()
      if (stopId) {
        uiStore.toggleMultiSelectStop(stopId)
        timeline.scheduleRender()
      }
    }
    return
  }
  if (hit) {
    timeline.selectedBlockId.value = hit.block.id
    // Para producción: guardar el slot exacto clickeado (no el run completo)
    timeline.selectedSlotMs.value = hit.slotStart ?? null
    emit('hitResult', hit)
  } else {
    timeline.selectedBlockId.value = null
    timeline.selectedSlotMs.value = null
  }
  timeline.scheduleRender()
}

function handleTouchStart(e: TouchEvent): void {
  if (e.touches.length === 1) {
    touchState.startX = e.touches[0].clientX
    touchState.startMs = timeline.viewStart.value
  } else if (e.touches.length === 2) {
    const dx = e.touches[1].clientX - e.touches[0].clientX
    const dy = e.touches[1].clientY - e.touches[0].clientY
    touchState.pinchDist = Math.sqrt(dx * dx + dy * dy)
  }
}

function handleTouchMove(e: TouchEvent): void {
  e.preventDefault()
  if (e.touches.length === 1) {
    const deltaX = touchState.startX - e.touches[0].clientX
    const canvas = canvasRef.value
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    const pxPerMs = rect.width / (timeline.viewEnd.value - timeline.viewStart.value)
    const deltaMs = deltaX / pxPerMs
    timeline.setWindow(
      touchState.startMs + deltaMs,
      touchState.startMs + deltaMs + (timeline.viewEnd.value - timeline.viewStart.value)
    )
  } else if (e.touches.length === 2) {
    const dx = e.touches[1].clientX - e.touches[0].clientX
    const dy = e.touches[1].clientY - e.touches[0].clientY
    const dist = Math.sqrt(dx * dx + dy * dy)
    if (touchState.pinchDist > 0) {
      const scale = touchState.pinchDist / dist
      const span = timeline.viewEnd.value - timeline.viewStart.value
      const center = (timeline.viewStart.value + timeline.viewEnd.value) / 2
      timeline.zoomTo(center, span * scale)
    }
    touchState.pinchDist = dist
  }
}

function handleWheel(e: WheelEvent): void {
  e.preventDefault()
  const span = timeline.viewEnd.value - timeline.viewStart.value
  if (e.ctrlKey || e.metaKey) {
    const factor = e.deltaY > 0 ? 1.15 : 0.87
    const mouseMs = timeline.xToMs(e.offsetX * (window.devicePixelRatio || 1))
    timeline.zoomTo(mouseMs, span * factor)
  } else {
    timeline.panBy(e.deltaX !== 0 ? (e.deltaX / 500) * span : (e.deltaY / 500) * span)
  }
}

const shiftLabel = computed(() => turnosStore.shiftLabel)

onMounted(() => {
  if (containerRef.value) {
    resizeObserver = new ResizeObserver(() => timeline.scheduleRender())
    resizeObserver.observe(containerRef.value)
  }
  timeline.start()
  syncData()
})

onUnmounted(() => {
  timeline.stop()
  resizeObserver?.disconnect()
})

watch([() => stopsStore.stops, () => productionRunsStore.runs, () => machineStore.recentEvents], syncData, { deep: true })
watch(() => turnosStore.activeTurno, syncData)
watch(() => uiStore.multiSelectedIds, () => timeline.scheduleRender(), { deep: true })

defineExpose({ assignProduct: timeline.assignProduct, scrollToNow: timeline.scrollToNow, scrollToDate: timeline.scrollToDate, resetAutoFit: timeline.resetAutoFit, syncData, fitToMode: timeline.fitToMode })
</script>

<template>
  <div ref="containerRef" class="flex flex-col w-full h-full select-none">
    <div class="flex items-stretch flex-1 min-h-0">

      <!-- Botones izquierda -->
      <div class="flex flex-col w-12 shrink-0">
        <button
          class="flex flex-col items-center justify-center flex-1 bg-sky-500 hover:bg-sky-400 active:bg-sky-600 text-white transition-colors border-b border-sky-600/60"
          title="Retroceder 16 minutos"
          @click="panLeft16m"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" />
          </svg>
          <span class="text-[9px] font-bold tracking-wide mt-0.5">16m</span>
        </button>
        <button
          class="flex flex-col items-center justify-center flex-1 bg-blue-700 hover:bg-blue-600 active:bg-blue-800 text-white transition-colors"
          title="Retroceder 1 hora"
          @click="panLeft1h"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M11 19l-7-7 7-7M18 19l-7-7 7-7" />
          </svg>
          <span class="text-[9px] font-bold tracking-wide mt-0.5">1h</span>
        </button>
      </div>

      <canvas
        ref="canvasRef"
        class="flex-1 min-w-0 h-full cursor-crosshair"
        @click="handleCanvasClick"
        @touchstart.passive="handleTouchStart"
        @touchmove="handleTouchMove"
        @wheel="handleWheel"
      />

      <!-- Botones derecha -->
      <div class="flex flex-col w-12 shrink-0">
        <button
          class="flex flex-col items-center justify-center flex-1 bg-sky-500 hover:bg-sky-400 active:bg-sky-600 text-white transition-colors border-b border-sky-600/60"
          title="Avanzar 16 minutos"
          @click="panRight16m"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" />
          </svg>
          <span class="text-[9px] font-bold tracking-wide mt-0.5">16m</span>
        </button>
        <button
          class="flex flex-col items-center justify-center flex-1 bg-blue-700 hover:bg-blue-600 active:bg-blue-800 text-white transition-colors"
          title="Avanzar 1 hora"
          @click="panRight1h"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 5l7 7-7 7M13 5l7 7-7 7" />
          </svg>
          <span class="text-[9px] font-bold tracking-wide mt-0.5">1h</span>
        </button>
      </div>
    </div>

    <div class="flex items-center justify-between px-4 py-1.5 bg-edge-900/80 border-t border-edge-700/30">
      <span class="text-[11px] text-edge-500">{{ shiftLabel }}</span>
      <div class="flex items-center gap-4 text-[10px] text-edge-400">
        <span class="flex items-center gap-1">
          <span class="w-3 h-2 rounded-sm bg-[#dc2626]" />
          Parada
        </span>
        <span class="flex items-center gap-1">
          <span class="w-3 h-2 rounded-sm bg-[#eab308]" />
          Microparada
        </span>
        <span class="flex items-center gap-1">
          <span class="w-3 h-2 rounded-sm bg-[#f87171]" />
          Sin asignar
        </span>
        <span class="flex items-center gap-1">
          <span class="w-3 h-2 rounded-sm bg-gradient-to-r from-[#ea580c] via-[#0891b2] to-[#7c3aed]" />
          Producto
        </span>
      </div>
      <span class="text-[11px] text-edge-500">
        {{ stopsStore.stops.length }} paradas
      </span>
    </div>
  </div>
</template>
