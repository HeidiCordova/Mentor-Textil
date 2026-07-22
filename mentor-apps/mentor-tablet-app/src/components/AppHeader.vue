                                                                                            <script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import SvgIcon from './SvgIcon.vue'
import { api } from '@/services/api'
import CalendarModal from './CalendarModal.vue'
import VelocidadNominalModal from './VelocidadNominalModal.vue'
import { useConnectionStore } from '@/stores/connection'
import { useConfigStore } from '@/stores/config'
import { useStopsStore } from '@/stores/stops'
import { useUIStore } from '@/stores/ui'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { useTurnosStore } from '@/stores/turnos'
import { useProductionRunsStore } from '@/stores/productionRuns'
import { useMachineStore } from '@/stores/machine'
import { useDetectorStore } from '@/stores/detector'

defineProps<{
  darkMode: boolean
}>()

defineEmits<{
  toggleDark: []
}>()

const router = useRouter()
const connection = useConnectionStore()
const configStore = useConfigStore()
const stopsStore = useStopsStore()
const uiStore = useUIStore()
const pl = usePlantasLineasStore()
const turnosStore = useTurnosStore()
const productionRunsStore = useProductionRunsStore()
const machineStore = useMachineStore()
const _detectorStore = useDetectorStore()

const showProduccionModal = ref(false)

const showVelocidadModal = ref(false)
const velocidadActual = ref<number | null>(null)

const lineTitle = computed(() => {
  // 1. Nombre de la línea seleccionada en el dropdown (viene de sync_lineas)
  if (pl.lineaActual?.nombre) return pl.lineaActual.nombre.toUpperCase()
  // 2. line_name del config del dispositivo (fallback)
  const cfg = configStore.config as Record<string, unknown>
  if (cfg?.oee && typeof cfg.oee === 'object') {
    const oee = cfg.oee as Record<string, unknown>
    if (oee.line_name && typeof oee.line_name === 'string' && oee.line_name.trim())
      return (oee.line_name as string).toUpperCase()
  }
  return 'LÍNEA DE PRODUCCIÓN'
})

// Run activo: el que no tiene ended_at o el más reciente
const activeRun = computed(() => {
  const runs = productionRunsStore.runs
  if (!runs.length) return null
  const open = runs.find((r) => !r.ended_at)
  if (open) return open
  return runs.reduce((a, b) =>
    new Date(a.started_at) > new Date(b.started_at) ? a : b
  )
})

async function cargarVelocidad() {
  const run = activeRun.value
  if (!run) {
    velocidadActual.value = null
    return
  }
  try {
    const entries = await api.velocidadNominal({ linea_id: pl.selectedLineaId ?? undefined })
    const entry = entries.find((e) =>
      (run.producto_id != null && e.producto_id === run.producto_id) ||
      (run.sku != null && e.sku === run.sku)
    )
    velocidadActual.value = entry?.velocidad_us ?? null
  } catch {
    // silencioso
  }
}

onMounted(() => cargarVelocidad())
watch(activeRun, () => cargarVelocidad())

// Unidad configurada en el OEE config de la línea
const velUnit = computed(() => {
  const oee = configStore.config.oee as Record<string, unknown> | undefined
  const u = oee?.vel_unit as string | undefined
  if (u === 'uh') return 'u/h'
  if (u === 'um') return 'u/min'
  return 'u/s'
})

// Valor convertido a la unidad configurada, formateado limpiamente
const velDisplay = computed(() => {
  const v = velocidadActual.value
  if (v === null) return '—'
  const oee = configStore.config.oee as Record<string, unknown> | undefined
  const u = oee?.vel_unit as string | undefined
  let converted: number
  if (u === 'uh') converted = v * 3600
  else if (u === 'um') converted = v * 60
  else converted = v
  // Formato limpio: max 4 decimales, sin ceros innecesarios
  const s = converted.toPrecision(4)
  return String(parseFloat(s))
})

function onVelocidadClose() {
  showVelocidadModal.value = false
  cargarVelocidad()
}

const headerTitle = computed(() => {
  const product = activeRun.value?.nombre || activeRun.value?.sku || null
  if (product) return `${lineTitle.value} — ${product.toUpperCase()}`
  return lineTitle.value
})

const dateLabel = computed(() => {
  const d = new Date()
  const days = ['Domingo', 'Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado']
  const months = [
    'Enero', 'Febrero', 'Marzo', 'Abril', 'Mayo', 'Junio',
    'Julio', 'Agosto', 'Septiembre', 'Octubre', 'Noviembre', 'Diciembre'
  ]
  return `${days[d.getDay()]} ${d.getDate()} de ${months[d.getMonth()]} del ${d.getFullYear()}`
})

function handleLogout(): void {
  connection.logout()
  connection.disconnect()
  router.push('/login')
}

const showCalendar = ref(false)

// ---- Estado del detector de visión ----
const detectorStatus   = computed(() => _detectorStore.trackerState)
const detectorIdleSecs = computed(() => _detectorStore.idleSecs)  // Use computed value that auto-updates

let _hdrTickTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  _detectorStore.startBackground()
  _hdrTickTimer = setInterval(() => { _detectorStore.tick() }, 1000)
})
onUnmounted(() => {
  _detectorStore.stopBackground()
  if (_hdrTickTimer !== null) { clearInterval(_hdrTickTimer); _hdrTickTimer = null }
})

function fmtMMSS(secs: number): string {
  const totalSecs = Math.floor(secs)  // Round down to integer
  const m = Math.floor(totalSecs / 60)
  const s = totalSecs % 60
  if (m >= 60) {
    const h = Math.floor(m / 60)
    const mm = m % 60
    return `${h}:${String(mm).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
  return `${m}:${String(s).padStart(2, '0')}`
}

// ── Contador de unidades del turno ──────────────────────────────────
const todayCount = computed(() => {
  const midnight = new Date()
  midnight.setHours(0, 0, 0, 0)
  const ms = midnight.getTime()
  return machineStore.recentEvents.filter(e =>
    new Date(e.timestamp).getTime() >= ms &&
    (e.event_type === 'CORTE' || e.event_type === 'cut_detected')
  ).length
})

const shiftUnits = computed(() => {
  const since = new Date(turnosStore.shiftSince()).getTime()
  return machineStore.recentEvents.filter(
    (e) => (e.event_type === 'cut_detected' || e.event_type === 'CORTE') && new Date(e.timestamp).getTime() >= since
  ).length
})

/** Desglose por hora: [{label: '14:00', count: 12}, ...] */
const shiftHourlyBreakdown = computed(() => {
  const since = new Date(turnosStore.shiftSince()).getTime()
  const events = machineStore.recentEvents.filter(
    (e) => (e.event_type === 'cut_detected' || e.event_type === 'CORTE') && new Date(e.timestamp).getTime() >= since
  )
  const buckets = new Map<string, number>()
  for (const e of events) {
    const d = new Date(e.timestamp)
    const label = `${String(d.getHours()).padStart(2, '0')}:00`
    buckets.set(label, (buckets.get(label) ?? 0) + 1)
  }
  return Array.from(buckets.entries())
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => a.label.localeCompare(b.label))
})
// ─────────────────────────────────────────────────────────────────────

function handleCalendarSelect(ms: number): void {
  showCalendar.value = false
  uiStore.triggerGoToDate(ms)
}
</script>

<template>
  <header
    class="flex flex-col shrink-0 border-b transition-colors"
    :class="darkMode ? 'bg-edge-950 border-edge-800/40' : 'bg-white border-slate-200'"
  >
    <!-- Row 1: Centred title + user/controls top-right -->
    <div class="relative flex items-center justify-center px-4 pt-3 pb-2">
      <div class="flex flex-col items-center max-w-[calc(100%-10rem)] sm:max-w-none">
        <h1
          class="text-base font-bold tracking-wide text-center uppercase leading-tight transition-colors"
          :class="darkMode ? 'text-white' : 'text-slate-900'"
        >
          {{ headerTitle }}
        </h1>
      </div>

      <!-- Top-right cluster -->
      <div class="absolute right-4 top-1/2 -translate-y-1/2 flex items-center gap-2">
        <!-- Unjustified badge -->
        <div
          class="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold"
          :class="
            stopsStore.unjustifiedStops.length > 0
              ? 'bg-red-500/15 text-red-400 border border-red-500/30'
              : 'bg-green-500/15 text-green-400 border border-green-500/30'
          "
        >
          <span
            class="w-1.5 h-1.5 rounded-full"
            :class="stopsStore.unjustifiedStops.length > 0 ? 'bg-red-400' : 'bg-green-400'"
          />
          {{ stopsStore.unjustifiedStops.length }} sin asignar
        </div>

        <!-- User dropdown -->
        <button
          class="flex items-center gap-1.5 px-2.5 py-1 rounded border transition-colors"
          :class="darkMode ? 'border-edge-600 bg-edge-900 hover:bg-edge-800' : 'border-slate-300 bg-slate-100 hover:bg-slate-200'"
        >
          <SvgIcon name="user" :size="12" :class="darkMode ? 'text-edge-300' : 'text-slate-500'" />
          <span class="text-xs font-medium" :class="darkMode ? 'text-edge-200' : 'text-slate-700'">{{ connection.operator?.nombre || 'Analista' }}</span>
          <SvgIcon name="chevron-down" :size="10" :class="darkMode ? 'text-edge-400' : 'text-slate-400'" />
        </button>

        <!-- Theme toggle -->
        <button
          class="flex items-center justify-center w-6 h-6 rounded transition-colors"
          :class="darkMode ? 'hover:bg-edge-800 text-edge-500 hover:text-edge-200' : 'hover:bg-slate-200 text-slate-500 hover:text-slate-700'"
          @click="$emit('toggleDark')"
        >
          <SvgIcon :name="darkMode ? 'sun' : 'moon'" :size="13" />
        </button>

        <!-- Logout -->
        <button
          class="flex items-center justify-center w-6 h-6 rounded hover:bg-red-500/15 hover:text-red-400 transition-colors"
          :class="darkMode ? 'text-edge-500' : 'text-slate-400'"
          title="Cerrar sesión"
          @click="handleLogout"
        >
          <SvgIcon name="logout" :size="13" />
        </button>
      </div>
    </div>

    <!-- Row 2: Action buttons — responsive -->
    <div class="flex flex-wrap sm:flex-nowrap sm:relative items-center justify-center sm:justify-between px-4 sm:px-8 pt-2.5 pb-3 gap-x-2 gap-y-2">
      <!-- Calendario + Velocidad nominal -->
      <div class="flex items-center gap-2 order-2 sm:order-1">
        <button
          class="flex items-center gap-2 px-5 py-2 rounded-full border-2 font-semibold text-sm transition-colors"
          :class="darkMode ? 'border-blue-500/70 text-blue-400 bg-transparent hover:bg-blue-500/10' : 'border-blue-600 text-blue-600 bg-transparent hover:bg-blue-50'"
          @click="showCalendar = true"
        >
          <SvgIcon name="calendar" :size="16" />
          Calendario
        </button>
        <button
          class="flex items-center gap-2 px-3 py-1.5 rounded-xl border transition-all"
          :class="darkMode
            ? 'border-indigo-500/40 bg-indigo-950/60 hover:bg-indigo-900/60 text-indigo-300'
            : 'border-indigo-400/60 bg-indigo-50 hover:bg-indigo-100 text-indigo-700'"
          title="Editar velocidad nominal por producto"
          @click="showVelocidadModal = true"
        >
          <SvgIcon name="bolt" :size="14" class="shrink-0 text-yellow-400" />
          <div class="flex flex-col items-start leading-none gap-0.5">
            <span class="text-[10px] font-medium opacity-70 uppercase tracking-wide">Vel. nominal</span>
            <span
              class="text-sm font-bold tabular-nums"
              :class="velocidadActual !== null
                ? (darkMode ? 'text-white' : 'text-indigo-800')
                : (darkMode ? 'text-edge-500' : 'text-slate-400')"
            >{{ velDisplay }} <span class="text-xs font-semibold opacity-80">{{ velUnit }}</span></span>
          </div>
        </button>
      </div>

      <!-- Tiempo Actual + date below — centrado en mobile, absoluto en sm+ -->
      <div class="w-full sm:w-auto order-1 sm:absolute sm:left-1/2 sm:-translate-x-1/2 flex flex-col items-center pointer-events-auto sm:pointer-events-none">
        <!-- Botón + indicador en la misma fila -->
        <div class="flex items-center gap-2">
          <button
            class="flex items-center gap-2 px-6 py-2 rounded-full font-semibold text-sm transition-colors shadow-lg pointer-events-auto"
            :class="darkMode
              ? 'bg-edge-800 text-white hover:bg-edge-700 border border-edge-600/50'
              : 'bg-slate-700 text-white hover:bg-slate-600 border border-slate-500/50'"
            @click="uiStore.triggerGoToNow()"
          >
            <SvgIcon name="clock" :size="16" />
            Tiempo Actual
          </button>

          <!-- Contadores HOY / Turno + Microparadas / Paradas -->
          <div class="flex items-center gap-3 px-3 py-1.5 rounded-full border select-none"
            :class="darkMode ? 'bg-edge-800/80 border-edge-600/40' : 'bg-slate-100 border-slate-300'">
            <div class="flex items-center gap-1.5">
              <span class="text-[10px] font-semibold uppercase"
                :class="darkMode ? 'text-gray-400' : 'text-slate-500'">Hoy</span>
              <span class="text-sm font-bold font-mono tabular-nums"
                :class="darkMode ? 'text-white' : 'text-slate-900'">{{ todayCount }}</span>
            </div>
            <span class="w-px h-4" :class="darkMode ? 'bg-edge-600' : 'bg-slate-300'" />
            <div class="flex items-center gap-1.5">
              <span class="text-[10px] font-semibold uppercase"
                :class="darkMode ? 'text-gray-400' : 'text-slate-500'">Turno</span>
              <span class="text-sm font-bold font-mono tabular-nums"
                :class="darkMode ? 'text-emerald-300' : 'text-emerald-600'">{{ shiftUnits }}</span>
            </div>
          </div>

          <!-- Microparadas -->
          <div class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-[10px] font-bold border select-none"
            :class="detectorStatus === 'idle_wait'
              ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
              : (darkMode ? 'bg-edge-800/60 text-gray-500 border-edge-600/30' : 'bg-slate-100 text-slate-500 border-slate-300')">
            <span class="w-1.5 h-1.5 rounded-full shrink-0"
              :class="detectorStatus === 'idle_wait' ? 'bg-amber-400 animate-pulse' : (darkMode ? 'bg-gray-600' : 'bg-slate-400')" />
            <span>Micro {{ _detectorStore.microparadasCount }}</span>
            <template v-if="detectorStatus === 'idle_wait'">
              <span class="font-mono tabular-nums">{{ fmtMMSS(detectorIdleSecs) }}</span>
            </template>
          </div>

          <!-- Paradas -->
          <div class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-[10px] font-bold border select-none"
            :class="detectorStatus === 'stop_open'
              ? 'bg-red-500/15 text-red-400 border-red-500/30'
              : (darkMode ? 'bg-edge-800/60 text-gray-500 border-edge-600/30' : 'bg-slate-100 text-slate-500 border-slate-300')">
            <span class="w-1.5 h-1.5 rounded-full shrink-0"
              :class="detectorStatus === 'stop_open' ? 'bg-red-400 animate-pulse' : (darkMode ? 'bg-gray-600' : 'bg-slate-400')" />
            <span>Paradas {{ _detectorStore.paradasCount }}</span>
            <template v-if="detectorStatus === 'stop_open'">
              <span class="font-mono tabular-nums">{{ fmtMMSS(detectorIdleSecs) }}</span>
            </template>
          </div>
        </div>

        <span class="text-xs mt-1.5" :class="darkMode ? 'text-edge-400' : 'text-slate-500'">{{ dateLabel }}</span>
        <span class="text-[11px] font-semibold text-blue-500 mt-0.5">
          {{ turnosStore.shiftLabel }}
        </span>
      </div>

      <!-- Contador turno + Selección múltiple -->
      <div class="flex items-center gap-2 order-3">
        <!-- Contador de unidades del turno -->
        <button
          class="flex items-center gap-2 px-4 py-2 rounded-full font-semibold text-sm transition-all shadow-lg"
          :class="darkMode
            ? 'bg-emerald-700/60 text-emerald-200 border border-emerald-600/40 hover:bg-emerald-700/80'
            : 'bg-emerald-100 text-emerald-800 border border-emerald-300 hover:bg-emerald-200'"
          title="Unidades producidas en el turno actual"
          @click="showProduccionModal = true"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span class="tabular-nums">{{ shiftUnits.toLocaleString() }} u</span>
        </button>

        <!-- Selección múltiple -->
        <button
          class="flex items-center gap-2 px-5 py-2 rounded-full font-semibold text-sm transition-all shadow-lg"
          :class="uiStore.multiSelectMode
            ? 'bg-cyan-400 text-slate-900 ring-2 ring-cyan-300 hover:bg-cyan-300'
            : 'bg-blue-600 text-white hover:bg-blue-500'"
          @click="uiStore.toggleMultiSelect()"
        >
          <SvgIcon name="layers" :size="16" />
          {{ uiStore.multiSelectMode ? 'Cancelar selección' : 'Selección múltiple' }}
        </button>
      </div>
    </div>
  </header>

  <CalendarModal
    v-if="showCalendar"
    @select="handleCalendarSelect"
    @cancel="showCalendar = false"
  />

  <Teleport to="body">
    <VelocidadNominalModal
      v-if="showVelocidadModal"
      @close="onVelocidadClose"
    />
  </Teleport>

  <!-- Modal: Producción del turno -->
  <Teleport to="body">
    <div
      v-if="showProduccionModal"
      class="fixed inset-0 z-[9999] flex items-center justify-center p-4"
      @click.self="showProduccionModal = false"
    >
      <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="showProduccionModal = false" />
      <div class="relative w-full max-w-sm rounded-2xl shadow-2xl overflow-hidden"
        :class="darkMode ? 'bg-edge-900 border border-edge-700' : 'bg-white border border-slate-200'">
        <!-- Cabecera -->
        <div class="flex items-center justify-between px-5 py-4 border-b"
          :class="darkMode ? 'border-edge-700' : 'border-slate-200'">
          <div>
            <h2 class="text-base font-bold" :class="darkMode ? 'text-white' : 'text-slate-900'">Produccion del Turno</h2>
            <p class="text-xs mt-0.5" :class="darkMode ? 'text-edge-400' : 'text-slate-500'">{{ turnosStore.shiftLabel }}</p>
          </div>
          <button class="w-7 h-7 flex items-center justify-center rounded-full transition-colors"
            :class="darkMode ? 'hover:bg-edge-700 text-edge-400' : 'hover:bg-slate-100 text-slate-500'"
            @click="showProduccionModal = false">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Total grande -->
        <div class="px-5 pt-5 pb-4 flex items-center gap-4">
          <div class="flex-1">
            <p class="text-xs uppercase tracking-widest font-semibold" :class="darkMode ? 'text-edge-400' : 'text-slate-500'">Total unidades</p>
            <p class="text-4xl font-black tabular-nums mt-1" :class="darkMode ? 'text-emerald-300' : 'text-emerald-600'">
              {{ shiftUnits.toLocaleString() }}
            </p>
          </div>
          <div class="w-14 h-14 rounded-xl flex items-center justify-center"
            :class="darkMode ? 'bg-emerald-800/40' : 'bg-emerald-100'">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-7 h-7" :class="darkMode ? 'text-emerald-300' : 'text-emerald-600'" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
          </div>
        </div>

        <!-- Desglose por hora -->
        <div class="px-5 pb-5">
          <p class="text-xs uppercase tracking-widest font-semibold mb-3" :class="darkMode ? 'text-edge-400' : 'text-slate-500'">Por hora</p>
          <div v-if="shiftHourlyBreakdown.length === 0" class="text-sm italic" :class="darkMode ? 'text-edge-500' : 'text-slate-400'">Sin datos en este turno.</div>
          <div v-else class="space-y-2">
            <div v-for="row in shiftHourlyBreakdown" :key="row.label" class="flex items-center gap-3">
              <span class="text-xs font-mono w-12 shrink-0" :class="darkMode ? 'text-edge-300' : 'text-slate-600'">{{ row.label }}</span>
              <div class="flex-1 h-5 rounded overflow-hidden" :class="darkMode ? 'bg-edge-800' : 'bg-slate-100'">
                <div
                  class="h-full rounded transition-all"
                  :class="darkMode ? 'bg-emerald-500' : 'bg-emerald-400'"
                  :style="{ width: shiftHourlyBreakdown.length ? (row.count / Math.max(...shiftHourlyBreakdown.map(r => r.count)) * 100) + '%' : '0%' }"
                />
              </div>
              <span class="text-xs font-bold tabular-nums w-10 text-right shrink-0"
                :class="darkMode ? 'text-white' : 'text-slate-800'">{{ row.count }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
