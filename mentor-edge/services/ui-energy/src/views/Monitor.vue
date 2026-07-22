<template>
  <div class="space-y-5 max-w-5xl">

    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Monitor</h1>
        <p class="mt-0.5 text-sm text-gray-400">Estado en tiempo real del bus RS-485</p>
      </div>
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-1.5">
          <span v-if="loading" class="w-2 h-2 rounded-full bg-blue-400 animate-pulse"></span>
          <span v-else class="w-2 h-2 rounded-full bg-slate-600"></span>
          <span class="text-xs text-gray-500">
            {{ loading ? 'Actualizando...' : `Actualiza en ${countdown}s` }}
          </span>
        </div>
        <button @click="refresh"
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
                 bg-slate-700 text-gray-200 hover:bg-slate-600 transition-colors">
          <svg class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''"
            fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0
                 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          Actualizar
        </button>
      </div>
    </div>

    <!-- Error -->
    <p v-if="error" class="text-xs text-red-400 px-1">{{ error }}</p>

    <!-- Loading state (initial) -->
    <div v-if="loading && meters.length === 0" class="flex justify-center py-16">
      <svg class="w-6 h-6 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
      </svg>
    </div>

    <!-- Empty state -->
    <div v-else-if="!loading && meters.length === 0"
      class="flex flex-col items-center gap-3 py-16 text-center">
      <svg class="w-10 h-10 text-slate-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M13 10V3L4 14h7v7l9-11h-7z"/>
      </svg>
      <p class="text-sm text-gray-500">Sin medidores activos.</p>
      <p class="text-xs text-gray-600">Agrega medidores en la seccion Configuracion.</p>
    </div>

    <template v-else>

      <!-- Meter selector buttons -->
      <div class="flex flex-wrap gap-2">
        <button
          v-for="m in meters"
          :key="m.meter_id"
          @click="selected = m.meter_id"
          :class="[
            'flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium border transition-all duration-200',
            selected === m.meter_id
              ? (m.online
                  ? 'bg-blue-600/20 border-blue-500/60 text-blue-300'
                  : 'bg-slate-700/60 border-slate-500/60 text-slate-300')
              : 'bg-slate-800/50 border-slate-700/40 text-gray-400 hover:border-slate-600 hover:text-gray-200'
          ]">
          <!-- Online dot -->
          <span :class="[
            'w-2 h-2 rounded-full shrink-0',
            m.online ? 'bg-green-400' : 'bg-slate-600'
          ]" :style="m.online && selected === m.meter_id
            ? 'animation: pulse-dot 2s ease-in-out infinite' : ''">
          </span>
          <span>{{ m.meter_id }}</span>
          <!-- Unit id badge -->
          <span class="ml-0.5 min-w-[18px] h-[18px] px-1 rounded-full bg-slate-700
                       flex items-center justify-center text-[9px] font-bold text-gray-400">
            {{ m.unit_id }}
          </span>
        </button>
      </div>

      <!-- Detail panel for selected meter -->
      <div v-if="selectedMeter"
        :class="[
          'rounded-2xl border transition-all duration-300',
          selectedMeter.online
            ? 'bg-slate-800/70 border-slate-700/60'
            : 'bg-slate-900/50 border-slate-800/60'
        ]">

        <!-- Panel header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-slate-700/50">
          <div class="flex items-center gap-3">
            <span :class="[
              'w-3 h-3 rounded-full shrink-0',
              selectedMeter.online ? 'bg-green-400 shadow shadow-green-400/50' : 'bg-slate-600'
            ]" :style="selectedMeter.online ? 'animation: pulse-dot 2s ease-in-out infinite' : ''">
            </span>
            <div>
              <p class="text-base font-semibold text-white">{{ selectedMeter.meter_id }}</p>
              <p :class="['text-xs font-medium', selectedMeter.online ? 'text-green-400' : 'text-slate-500']">
                {{ selectedMeter.online ? 'En linea' : 'Sin datos' }}
                <span class="text-gray-600 font-normal ml-2">
                  · UID {{ selectedMeter.unit_id }}
                  <template v-if="selectedMeter.last_hora">
                    · {{ fmtAge(selectedMeter.age_s) }}
                  </template>
                </span>
              </p>
            </div>
          </div>
          <MeterSvg width="52" height="80" />
        </div>

        <!-- Main metrics grid -->
        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-px bg-slate-700/30">
          <!-- Tension -->
          <div class="bg-slate-800/60 px-5 py-5">
            <p class="text-[10px] uppercase tracking-widest text-gray-600 mb-1">Tension</p>
            <p class="text-2xl font-mono font-bold text-white leading-none">
              {{ fmt(selectedMeter.vavg, 1) ?? '—' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">V</p>
          </div>
          <!-- Corriente -->
          <div class="bg-slate-800/60 px-5 py-5">
            <p class="text-[10px] uppercase tracking-widest text-gray-600 mb-1">Corriente</p>
            <p class="text-2xl font-mono font-bold text-white leading-none">
              {{ fmt(selectedMeter.iavg, 2) ?? '—' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">A</p>
          </div>
          <!-- Potencia activa -->
          <div class="bg-slate-800/60 px-5 py-5">
            <p class="text-[10px] uppercase tracking-widest text-gray-600 mb-1">Potencia</p>
            <p :class="[
              'text-2xl font-mono font-bold leading-none',
              (selectedMeter.p ?? 0) > 0 ? 'text-blue-300' : 'text-gray-400'
            ]">
              {{ fmtWVal(selectedMeter.p) ?? '—' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">{{ fmtWUnit(selectedMeter.p) }}</p>
          </div>
          <!-- Energia -->
          <div class="bg-slate-800/60 px-5 py-5">
            <p class="text-[10px] uppercase tracking-widest text-gray-600 mb-1">Energia</p>
            <p class="text-2xl font-mono font-bold text-amber-300 leading-none">
              {{ fmtEnergiaVal(selectedMeter.ep_imp) ?? '—' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">{{ fmtEnergiaUnit(selectedMeter.ep_imp) }}</p>
          </div>
          <!-- Frecuencia -->
          <div class="bg-slate-800/60 px-5 py-5">
            <p class="text-[10px] uppercase tracking-widest text-gray-600 mb-1">Frecuencia</p>
            <p class="text-2xl font-mono font-bold text-white leading-none">
              {{ fmt(selectedMeter.freq, 1) ?? '—' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">Hz</p>
          </div>
          <!-- Factor de potencia -->
          <div class="bg-slate-800/60 px-5 py-5">
            <p class="text-[10px] uppercase tracking-widest text-gray-600 mb-1">Factor P.</p>
            <p class="text-2xl font-mono font-bold text-white leading-none">
              {{ fmt(selectedMeter.pf, 2) ?? '—' }}
            </p>
            <p class="text-xs text-gray-500 mt-1">cos φ</p>
          </div>
        </div>

        <!-- Charts: series temporales -->
        <div class="px-6 py-4 border-t border-slate-700/50">
          <div class="flex items-center justify-between mb-3">
            <span class="text-[10px] uppercase tracking-widest text-gray-600">Historico</span>
            <span class="text-[10px] text-gray-600">
              <span v-if="historyLoading" class="animate-pulse">Cargando...</span>
              <template v-else-if="historyRows.length > 0">
                {{ historyRows.length }} lecturas
                · ~{{ Math.round(historyRows.length * pollIntervalS / 60) }} min
              </template>
              <template v-else>Sin historial</template>
            </span>
          </div>
          <div class="grid grid-cols-3 gap-3">
            <div class="bg-slate-900/60 rounded-xl p-3">
              <p class="text-[10px] uppercase tracking-wider font-medium text-amber-400/80 mb-2">
                Tension <span class="text-gray-600 normal-case font-normal">(V)</span>
              </p>
              <div class="relative" style="height:90px">
                <canvas ref="canvasV"></canvas>
              </div>
            </div>
            <div class="bg-slate-900/60 rounded-xl p-3">
              <p class="text-[10px] uppercase tracking-wider font-medium text-cyan-400/80 mb-2">
                Corriente <span class="text-gray-600 normal-case font-normal">(A)</span>
              </p>
              <div class="relative" style="height:90px">
                <canvas ref="canvasI"></canvas>
              </div>
            </div>
            <div class="bg-slate-900/60 rounded-xl p-3">
              <p class="text-[10px] uppercase tracking-wider font-medium text-blue-400/80 mb-2">
                Potencia <span class="text-gray-600 normal-case font-normal">(W)</span>
              </p>
              <div class="relative" style="height:90px">
                <canvas ref="canvasP"></canvas>
              </div>
            </div>
          </div>
        </div>

        <!-- Progress bar footer -->
        <div class="px-6 py-3 flex items-center gap-4">
          <span class="text-[10px] text-gray-600 shrink-0">Prox. lectura</span>
          <div class="flex-1 h-1 rounded-full bg-slate-700 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-1000"
              :class="selectedMeter.online ? 'bg-blue-500/60' : 'bg-slate-600/40'"
              :style="`width: ${progressPct(selectedMeter.age_s)}%`">
            </div>
          </div>
          <span class="text-[10px] font-mono text-gray-600 shrink-0">
            ~{{ Math.max(0, pollIntervalS - (selectedMeter.age_s ?? pollIntervalS)) }}s
          </span>
        </div>

      </div>

      <!-- All meters compact summary row -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div v-for="m in meters" :key="'sum-' + m.meter_id"
          @click="selected = m.meter_id"
          :class="[
            'rounded-xl border px-4 py-3 cursor-pointer transition-all duration-200',
            selected === m.meter_id
              ? 'border-blue-500/40 bg-slate-800/80'
              : 'border-slate-700/40 bg-slate-800/30 hover:bg-slate-800/60'
          ]">
          <div class="flex items-center justify-between mb-2">
            <span class="text-xs font-semibold text-gray-300 truncate">{{ m.meter_id }}</span>
            <span :class="['text-[10px] font-medium', m.online ? 'text-green-400' : 'text-slate-500']">
              {{ m.online ? 'Online' : 'Offline' }}
            </span>
          </div>
          <p :class="[
            'text-lg font-mono font-bold leading-none',
            (m.p ?? 0) > 0 ? 'text-blue-300' : 'text-gray-500'
          ]">
            {{ fmtW(m.p) ?? '—' }}
          </p>
          <p class="text-[10px] text-gray-600 mt-1">
            {{ fmt(m.vavg, 1) ?? '—' }} V &nbsp;·&nbsp; {{ fmt(m.iavg, 2) ?? '—' }} A
          </p>
        </div>
      </div>

    </template>

  </div>
</template>

<script setup>
import {
  ref, computed, watch, nextTick, onMounted, onUnmounted
} from 'vue'
import {
  Chart,
  LineElement, PointElement, LineController,
  CategoryScale, LinearScale,
  Tooltip, Filler
} from 'chart.js'
import MeterSvg from '../components/MeterSvg.vue'
import { getLiveStatus, getMeterHistory } from '../services/api.js'

Chart.register(LineElement, PointElement, LineController, CategoryScale, LinearScale, Tooltip, Filler)

const REFRESH_INTERVAL = 10

const meters        = ref([])
const pollIntervalS = ref(300)
const loading       = ref(false)
const error         = ref('')
const countdown     = ref(REFRESH_INTERVAL)
const selected      = ref(null)

// ── Chart state ───────────────────────────────────────────────────────────────

const canvasV = ref(null)
const canvasI = ref(null)
const canvasP = ref(null)
const historyRows    = ref([])
const historyLoading = ref(false)

let chartV = null, chartI = null, chartP = null

let pollTimer      = null
let countdownTimer = null

// ── Computed ──────────────────────────────────────────────────────────────────

const selectedMeter = computed(() =>
  meters.value.find(m => m.meter_id === selected.value) ?? null
)

// ── Formatters ────────────────────────────────────────────────────────────────

function fmt(val, decimals = 1) {
  if (val === null || val === undefined) return null
  return Number(val).toFixed(decimals)
}

function fmtW(val) {
  if (val === null || val === undefined) return null
  const w = Number(val)
  if (Math.abs(w) >= 1000) return `${(w / 1000).toFixed(2)} kW`
  return `${w.toFixed(0)} W`
}

function fmtWVal(val) {
  if (val === null || val === undefined) return null
  const w = Number(val)
  if (Math.abs(w) >= 1000) return (w / 1000).toFixed(2)
  return w.toFixed(0)
}
function fmtWUnit(val) {
  if (val === null || val === undefined) return 'W'
  return Math.abs(Number(val)) >= 1000 ? 'kW' : 'W'
}

function fmtEnergiaVal(val) {
  if (val === null || val === undefined) return null
  const wh = Number(val)
  if (Math.abs(wh) >= 1_000_000_000) return (wh / 1_000_000_000).toFixed(3)
  if (Math.abs(wh) >= 1_000_000)     return (wh / 1_000_000).toFixed(3)
  if (Math.abs(wh) >= 1_000)         return (wh / 1_000).toFixed(2)
  return wh.toFixed(0)
}
function fmtEnergiaUnit(val) {
  if (val === null || val === undefined) return 'Wh'
  const wh = Math.abs(Number(val))
  if (wh >= 1_000_000_000) return 'GWh'
  if (wh >= 1_000_000)     return 'MWh'
  if (wh >= 1_000)         return 'kWh'
  return 'Wh'
}

function fmtAge(ageS) {
  if (ageS === null || ageS === undefined) return '—'
  const s = Math.abs(ageS)
  if (s < 60)   return `hace ${s}s`
  if (s < 3600) return `hace ${Math.floor(s / 60)}m ${s % 60}s`
  return `hace ${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`
}

function progressPct(ageS) {
  if (ageS === null || ageS === undefined) return 0
  const pct = (ageS / pollIntervalS.value) * 100
  return Math.min(100, Math.max(0, pct))
}

// ── Charts ────────────────────────────────────────────────────────────────────

const CHART_OPTIONS = {
  responsive: true,
  maintainAspectRatio: false,
  animation: { duration: 300 },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: 'rgba(15, 23, 42, 0.95)',
      borderColor: 'rgba(71, 85, 105, 0.5)',
      borderWidth: 1,
      titleColor: '#94a3b8',
      bodyColor: '#e2e8f0',
      padding: 8,
      displayColors: false,
    }
  },
  scales: {
    x: {
      ticks: { color: '#475569', font: { size: 9 }, maxTicksLimit: 6, maxRotation: 0 },
      grid:  { color: 'rgba(51, 65, 85, 0.35)', drawBorder: false },
      border: { display: false },
    },
    y: {
      ticks: { color: '#475569', font: { size: 9 }, maxTicksLimit: 4 },
      grid:  { color: 'rgba(51, 65, 85, 0.35)', drawBorder: false },
      border: { display: false },
    }
  }
}

function destroyCharts() {
  if (chartV) { chartV.destroy(); chartV = null }
  if (chartI) { chartI.destroy(); chartI = null }
  if (chartP) { chartP.destroy(); chartP = null }
}

function makeChart(canvas, data, borderColor, fillColor) {
  if (!canvas) return null
  return new Chart(canvas.getContext('2d'), {
    type: 'line',
    data: {
      labels: data.labels,
      datasets: [{
        data:            data.values,
        borderColor,
        backgroundColor: fillColor,
        borderWidth:     1.5,
        pointRadius:     data.values.length > 24 ? 0 : 2,
        pointHoverRadius: 4,
        tension:         0.35,
        fill:            true,
        spanGaps:        true,
      }]
    },
    options: CHART_OPTIONS,
  })
}

function buildCharts(rows) {
  const labels = rows.map(r => {
    const d = new Date(r.hora)
    const hh = d.getHours().toString().padStart(2, '0')
    const mm = d.getMinutes().toString().padStart(2, '0')
    return `${hh}:${mm}`
  })

  chartV = makeChart(
    canvasV.value,
    { labels, values: rows.map(r => r.vavg) },
    '#fbbf24', 'rgba(251,191,36,0.10)'
  )
  chartI = makeChart(
    canvasI.value,
    { labels, values: rows.map(r => r.iavg) },
    '#22d3ee', 'rgba(34,211,238,0.10)'
  )
  chartP = makeChart(
    canvasP.value,
    { labels, values: rows.map(r => r.p) },
    '#60a5fa', 'rgba(96,165,250,0.10)'
  )
}

async function loadHistory(meterId) {
  historyLoading.value = true
  historyRows.value    = []
  destroyCharts()
  try {
    const res = await getMeterHistory(meterId, 48)
    historyRows.value = res.rows ?? []
    await nextTick()
    buildCharts(historyRows.value)
  } catch (_) {
    // silently ignore — charts won't show
  } finally {
    historyLoading.value = false
  }
}

watch(selected, async (newVal) => {
  destroyCharts()
  if (newVal) {
    await nextTick()
    loadHistory(newVal)
  }
})

// ── Data fetch ────────────────────────────────────────────────────────────────

async function refresh() {
  loading.value   = true
  error.value     = ''
  countdown.value = REFRESH_INTERVAL
  try {
    const res = await getLiveStatus()
    meters.value        = res.meters ?? []
    pollIntervalS.value = res.poll_interval_s ?? 300
    if (!selected.value && meters.value.length > 0) {
      selected.value = meters.value[0].meter_id
    }
  } catch (e) {
    error.value = e.response?.data?.error ?? 'Error al obtener estado del bus'
  } finally {
    loading.value = false
  }
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

onMounted(() => {
  refresh()
  pollTimer      = setInterval(refresh, REFRESH_INTERVAL * 1000)
  countdownTimer = setInterval(() => {
    countdown.value = countdown.value > 0 ? countdown.value - 1 : REFRESH_INTERVAL
  }, 1000)
})

onUnmounted(() => {
  clearInterval(pollTimer)
  clearInterval(countdownTimer)
  destroyCharts()
})
</script>

<style scoped>
@keyframes pulse-dot {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(74, 222, 128, 0.6); }
  50%       { opacity: 0.8; box-shadow: 0 0 0 5px rgba(74, 222, 128, 0); }
}
</style>
