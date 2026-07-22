<template>
  <div class="space-y-5">

    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Lecturas</h1>
        <p class="mt-0.5 text-sm text-gray-400">Ultimo snapshot almacenado por medidor</p>
      </div>
      <button @click="reload()" :disabled="!selectedMeter || loading"
        class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
               bg-slate-700 text-gray-200 hover:bg-slate-600 transition-colors disabled:opacity-40">
        <svg class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''"
          fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0
               0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Actualizar
        <span class="w-2 h-2 rounded-full shrink-0 ml-0.5"
          :class="loading ? 'bg-amber-400 animate-pulse' : snapshot ? 'bg-green-500' : 'bg-slate-600'">
        </span>
      </button>
    </div>

    <!-- Meter selector buttons -->
    <div v-if="meters.length > 0" class="flex flex-wrap gap-2">
      <button
        v-for="m in meters"
        :key="m.meter_id"
        @click="selectMeter(m.meter_id)"
        :class="[
          'flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium border transition-all duration-200',
          selectedMeter === m.meter_id
            ? 'bg-blue-600/20 border-blue-500/60 text-blue-300'
            : 'bg-slate-800/50 border-slate-700/40 text-gray-400 hover:border-slate-600 hover:text-gray-200'
        ]">
        <span class="w-2 h-2 rounded-full shrink-0"
          :class="selectedMeter === m.meter_id ? 'bg-blue-400' : 'bg-slate-600'">
        </span>
        {{ m.meter_id }}
      </button>
    </div>

    <!-- Snapshot metadata -->
    <div v-if="hora" class="text-xs text-gray-500 flex items-center gap-3">
      <span>Snapshot: <span class="font-mono text-gray-400">{{ fmtDate(hora) }}</span></span>
      <span class="text-slate-700">·</span>
      <span>Intervalo: <span class="font-mono text-gray-400">{{ intervalS }}s</span></span>
      <template v-if="historyRows.length > 0">
        <span class="text-slate-700">·</span>
        <span class="text-gray-600">{{ historyRows.length }} lecturas
          (~{{ Math.round(historyRows.length * intervalS / 60) }} min)</span>
      </template>
    </div>

    <!-- Empty state -->
    <div v-if="!selectedMeter"
      class="flex flex-col items-center gap-3 py-16 text-center">
      <svg class="w-10 h-10 text-slate-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1
             0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
      </svg>
      <p class="text-sm text-gray-500">Selecciona un medidor para ver sus lecturas.</p>
    </div>

    <!-- Error -->
    <div v-else-if="error"
      class="rounded-xl border border-red-900/40 bg-red-950/20 px-4 py-3 text-sm text-red-400">
      {{ error }}
    </div>

    <!-- Loading (initial) -->
    <div v-else-if="loading && !snapshot" class="flex justify-center py-16">
      <svg class="w-6 h-6 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
      </svg>
    </div>

    <!-- Groups grid -->
    <template v-else-if="snapshot">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div v-for="g in GROUPS" :key="g.title" class="card overflow-hidden">

          <!-- Group header -->
          <h2 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3">
            {{ g.title }}
          </h2>

          <!-- Values table with inline sparklines -->
          <table class="w-full text-sm">
            <tbody class="divide-y divide-slate-700/40">
              <tr v-for="k in g.keys" :key="k">
                <td class="py-1.5 font-mono text-gray-400 text-xs w-20 shrink-0">{{ k }}</td>
                <td class="py-1 w-24 shrink-0">
                  <svg v-if="KEY_HIST[k] && sparkHasData(KEY_HIST[k])"
                    viewBox="0 0 200 28" preserveAspectRatio="none"
                    style="width:72px;height:16px;display:block">
                    <path v-if="sparkArea(KEY_HIST[k])"
                      :d="sparkArea(KEY_HIST[k])"
                      :fill="g.chartColor + '18'" stroke="none"/>
                    <path v-if="sparkLine(KEY_HIST[k])"
                      :d="sparkLine(KEY_HIST[k])"
                      fill="none" :stroke="g.chartColor"
                      stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                  <div v-else-if="historyLoading && KEY_HIST[k]"
                    class="w-16 h-0.5 rounded bg-slate-700/50 animate-pulse my-2">
                  </div>
                </td>
                <td class="py-1.5 text-right font-mono text-white tabular-nums">
                  {{ fmt(val(k), g.dec, g.energy) }}
                </td>
                <td class="py-1.5 pl-2 text-xs text-gray-500 w-14">{{ g.unit }}</td>
              </tr>
            </tbody>
          </table>

        </div>
      </div>
    </template>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { getMeters, getLatestSnapshot, getMeterHistory } from '../services/api.js'

// ── History key mapping (register key → history API field) ─────────────────

const KEY_HIST = {
  Ia: 'iavg', Ib: 'iavg', Ic: 'iavg', In: 'iavg', Iavg: 'iavg',
  Va: 'vavg', Vb: 'vavg', Vc: 'vavg', Vavg: 'vavg',
  Vab: 'vlavg', Vbc: 'vlavg', Vca: 'vlavg', VLavg: 'vlavg',
  Pa: 'p', Pb: 'p', Pc: 'p', P: 'p',
  Qa: 'q', Qb: 'q', Qc: 'q', Q: 'q',
  Sa: 's_va', Sb: 's_va', Sc: 's_va', S: 's_va',
  PFa: 'pf', PFb: 'pf', PFc: 'pf', PF: 'pf',
  DPFa: 'dpf', DPFb: 'dpf', DPFc: 'dpf', DPF: 'dpf',
  FreqA: 'freq', FreqB: 'freq', FreqC: 'freq', Freq: 'freq',
  EPAImp: 'ep_imp', EPBImp: 'ep_imp', EPCImp: 'ep_imp', EPImp: 'ep_imp',
  EPAExp: 'ep_exp', EPBExp: 'ep_exp', EPCExp: 'ep_exp', EPExp: 'ep_exp',
  EQAImp: 'eq_imp', EQBImp: 'eq_imp', EQCImp: 'eq_imp', EQImp: 'eq_imp',
  EQAExp: 'eq_exp', EQBExp: 'eq_exp', EQCExp: 'eq_exp', EQExp: 'eq_exp',
  ESA: 'es', ESB: 'es', ESC: 'es', ES: 'es',
}

// ── Groups definition ─────────────────────────────────────────────────────────

const GROUPS = [
  { title: 'Corrientes',               unit: 'A',     dec: 3, energy: false, chartKey: 'iavg',   chartColor: '#22d3ee', keys: ['Ia','Ib','Ic','In','Iavg'] },
  { title: 'Tension fase-neutro',      unit: 'V',     dec: 1, energy: false, chartKey: 'vavg',   chartColor: '#fbbf24', keys: ['Va','Vb','Vc','Vavg','V0'] },
  { title: 'Tension fase-fase',        unit: 'V',     dec: 1, energy: false, chartKey: 'vlavg',  chartColor: '#fb923c', keys: ['Vab','Vbc','Vca','VLavg'] },
  { title: 'Potencia activa',          unit: 'kW',    dec: 3, energy: false, chartKey: 'p',      chartColor: '#60a5fa', keys: ['Pa','Pb','Pc','P'] },
  { title: 'Potencia reactiva',        unit: 'kVAR',  dec: 3, energy: false, chartKey: 'q',      chartColor: '#a78bfa', keys: ['Qa','Qb','Qc','Q'] },
  { title: 'Potencia aparente',        unit: 'kVA',   dec: 3, energy: false, chartKey: 's_va',   chartColor: '#818cf8', keys: ['Sa','Sb','Sc','S'] },
  { title: 'Factor de potencia',       unit: '',      dec: 4, energy: false, chartKey: 'pf',     chartColor: '#34d399', keys: ['PFa','PFb','PFc','PF'] },
  { title: 'DPF (fundamental)',        unit: '',      dec: 4, energy: false, chartKey: 'dpf',    chartColor: '#6ee7b7', keys: ['DPFa','DPFb','DPFc','DPF'] },
  { title: 'Frecuencia',               unit: 'Hz',    dec: 2, energy: false, chartKey: 'freq',   chartColor: '#a3e635', keys: ['FreqA','FreqB','FreqC','Freq'] },
  { title: 'Energia activa importada', unit: 'kWh',   dec: 3, energy: true,  chartKey: 'ep_imp', chartColor: '#fb923c', keys: ['EPAImp','EPBImp','EPCImp','EPImp'] },
  { title: 'Energia activa exportada', unit: 'kWh',   dec: 3, energy: true,  chartKey: 'ep_exp', chartColor: '#f87171', keys: ['EPAExp','EPBExp','EPCExp','EPExp'] },
  { title: 'Energia reactiva import.', unit: 'kVARh', dec: 3, energy: true,  chartKey: 'eq_imp', chartColor: '#c084fc', keys: ['EQAImp','EQBImp','EQCImp','EQImp'] },
  { title: 'Energia reactiva export.', unit: 'kVARh', dec: 3, energy: true,  chartKey: 'eq_exp', chartColor: '#e879f9', keys: ['EQAExp','EQBExp','EQCExp','EQExp'] },
  { title: 'Energia aparente',         unit: 'kVAh',  dec: 3, energy: true,  chartKey: 'es',     chartColor: '#f472b6', keys: ['ESA','ESB','ESC','ES'] },
  { title: 'THD corriente',            unit: '%',     dec: 2, energy: false, chartKey: null,     chartColor: null,      keys: ['THDia','THDib','THDic'] },
  { title: 'THD tension',              unit: '%',     dec: 2, energy: false, chartKey: null,     chartColor: null,      keys: ['THDua','THDub','THDuc'] },
]

// ── State ──────────────────────────────────────────────────────────────────────

const meters         = ref([])
const selectedMeter  = ref('')
const snapshot       = ref(null)
const hora           = ref('')
const intervalS      = ref(0)
const loading        = ref(false)
const error          = ref('')
const historyRows    = ref([])
const historyLoading = ref(false)

let timer = null

// ── Snapshot helpers ──────────────────────────────────────────────────────────

function val(key) {
  if (!snapshot.value) return null
  const idx = snapshot.value.head.indexOf(key)
  return idx === -1 ? null : snapshot.value.data[idx]
}

function fmt(raw, dec, isEnergy) {
  if (raw === null || raw === undefined || raw === '') return '—'
  const n = parseFloat(raw)
  if (isNaN(n)) return '—'
  return (isEnergy ? n / 1000 : n).toFixed(dec)
}

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('es', { dateStyle: 'short', timeStyle: 'medium' })
}

// ── Sparkline helpers (pure SVG, no library) ──────────────────────────────────

const SVG_W = 200, SVG_H = 28

function _nums(key) {
  return historyRows.value.map(r => {
    const v = r[key]
    if (v === null || v === undefined) return null
    const n = Number(v)
    return isNaN(n) ? null : n
  })
}

function sparkHasData(key) {
  return _nums(key).filter(v => v !== null).length >= 2
}

function _coords(key) {
  const nums = _nums(key)
  const valid = nums.filter(v => v !== null)
  if (valid.length < 2) return null
  const min = Math.min(...valid)
  const max = Math.max(...valid)
  const range = max - min || 1
  return nums.map((v, i) => {
    if (v === null) return null
    return {
      x: +((i / (nums.length - 1)) * SVG_W).toFixed(1),
      y: +(SVG_H - ((v - min) / range) * (SVG_H - 4) - 2).toFixed(1),
    }
  })
}

function sparkLine(key) {
  const coords = _coords(key)
  if (!coords) return null
  let d = '', gap = true
  for (const pt of coords) {
    if (!pt) { gap = true; continue }
    d += gap ? `M ${pt.x} ${pt.y} ` : `L ${pt.x} ${pt.y} `
    gap = false
  }
  return d.trim() || null
}

function sparkArea(key) {
  const coords = _coords(key)
  if (!coords) return null
  const pts = coords.filter(Boolean)
  if (pts.length < 2) return null
  return (
    `M ${pts[0].x} ${SVG_H} ` +
    pts.map(p => `L ${p.x} ${p.y}`).join(' ') +
    ` L ${pts[pts.length - 1].x} ${SVG_H} Z`
  )
}

// ── Data loading ──────────────────────────────────────────────────────────────

async function load() {
  if (!selectedMeter.value) return
  loading.value = true
  error.value   = ''
  try {
    const d = await getLatestSnapshot(selectedMeter.value)
    snapshot.value  = d
    hora.value      = d.hora
    intervalS.value = d.interval_s
  } catch (e) {
    error.value    = e?.response?.data?.error ?? 'No se pudo cargar el snapshot'
    snapshot.value = null
  } finally {
    loading.value = false
  }
}

async function loadHistory(meterId) {
  historyLoading.value = true
  historyRows.value    = []
  try {
    const res = await getMeterHistory(meterId, 48)
    historyRows.value = res.rows ?? []
  } catch (_) {}
  finally {
    historyLoading.value = false
  }
}

async function selectMeter(meterId) {
  selectedMeter.value = meterId
  snapshot.value      = null
  historyRows.value   = []
  await load()
  loadHistory(meterId)
}

function reload() {
  load()
  if (selectedMeter.value) loadHistory(selectedMeter.value)
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

onMounted(async () => {
  try {
    meters.value = await getMeters()
    if (meters.value.length === 1) {
      await selectMeter(meters.value[0].meter_id)
    }
  } catch (_) {}
  timer = setInterval(() => { if (selectedMeter.value) load() }, 30000)
})

onUnmounted(() => { clearInterval(timer) })
</script>
