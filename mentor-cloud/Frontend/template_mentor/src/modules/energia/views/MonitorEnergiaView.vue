<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { energyService } from '@/api/services/energy.service'

const snapshots = ref([])
const meters = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = 25

const filterMeter = ref('')
const filterFrom = ref('')
const filterTo = ref('')

const selectedSnapshot = ref(null)
const detailOpen = ref(false)

const summaryFields = [
  { key: 'potencia_activa', label: 'Potencia Activa', unit: 'W', color: 'text-yellow-400' },
  { key: 'potencia_reactiva', label: 'Potencia Reactiva', unit: 'VAR', color: 'text-blue-400' },
  { key: 'factor_potencia', label: 'Factor de Potencia', unit: '', color: 'text-emerald-400' },
  { key: 'consumo_activa', label: 'Consumo Activo', unit: 'kWh', color: 'text-purple-400', isSum: true }
]

const tableColumns = [
  { key: 'hora', label: 'Hora', format: 'date' },
  { key: 'meter_id', label: 'Medidor' },
  { key: 'voltaje_avg', label: 'V avg', unit: 'V' },
  { key: 'corriente_avg', label: 'I avg', unit: 'A' },
  { key: 'potencia_activa', label: 'P activa', unit: 'W' },
  { key: 'potencia_reactiva', label: 'P reactiva', unit: 'VAR' },
  { key: 'factor_potencia', label: 'FP' },
  { key: 'frecuencia_hz', label: 'Freq', unit: 'Hz' },
  { key: 'consumo_activa', label: 'Consumo Act.', unit: 'kWh' },
  { key: 'consumo_reactiva', label: 'Consumo React.', unit: 'kVARh' },
  { key: 'consumo_aparente', label: 'Consumo Apar.', unit: 'kVAh' },
]

const latestByMeter = computed(() => {
  const map = {}
  for (const s of snapshots.value) {
    if (!map[s.meter_id] || new Date(s.hora) > new Date(map[s.meter_id].hora)) {
      map[s.meter_id] = s
    }
  }
  return map
})

const summaryCards = computed(() => {
  const latest = Object.values(latestByMeter.value)
  return summaryFields.map(f => {
    if (f.isSum) {
      const vals = snapshots.value
        .map(s => s[f.key])
        .filter(v => v !== null && v !== undefined && !isNaN(v))
      const sum = vals.reduce((a, b) => a + b, 0)
      return { ...f, value: sum.toFixed(2) }
    }
    if (!latest.length) return { ...f, value: '-' }
    const vals = latest.map(s => s[f.key]).filter(v => v !== undefined && v !== null)
    if (!vals.length) return { ...f, value: '-' }
    const avg = vals.reduce((a, b) => a + b, 0) / vals.length
    return { ...f, value: f.key === 'factor_potencia' ? avg.toFixed(3) : avg.toFixed(2) }
  })
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

async function fetchMeters() {
  try {
    const res = await energyService.getMeters()
    meters.value = res.data || []
  } catch {
    meters.value = []
  }
}

async function fetchSnapshots() {
  loading.value = true
  try {
    const params = { limit: pageSize, offset: (page.value - 1) * pageSize }
    if (filterMeter.value) params.meter_id = filterMeter.value
    if (filterFrom.value) params.from = filterFrom.value
    if (filterTo.value) params.to = filterTo.value
    const res = await energyService.getSnapshots(params)
    snapshots.value = res.data || []
    total.value = res.total || 0
  } catch {
    snapshots.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  page.value = 1
  fetchSnapshots()
}

function clearFilters() {
  filterMeter.value = ''
  filterFrom.value = ''
  filterTo.value = ''
  page.value = 1
  fetchSnapshots()
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    fetchSnapshots()
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    fetchSnapshots()
  }
}

function openDetail(snapshot) {
  selectedSnapshot.value = snapshot
  detailOpen.value = true
}

function closeDetail() {
  detailOpen.value = false
  selectedSnapshot.value = null
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('es-PE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatVal(val, col) {
  if (col.format === 'date') return formatDate(val)
  if (val === undefined || val === null) return '-'
  if (typeof val === 'number') return val.toFixed(2)
  return val
}

const detailGroups = [
  {
    title: 'Corriente',
    fields: [
      { key: 'corriente_a', label: 'Fase A', unit: 'A' },
      { key: 'corriente_b', label: 'Fase B', unit: 'A' },
      { key: 'corriente_c', label: 'Fase C', unit: 'A' },
      { key: 'corriente_avg', label: 'Promedio', unit: 'A' }
    ]
  },
  {
    title: 'Voltaje',
    fields: [
      { key: 'voltaje_a', label: 'Fase A', unit: 'V' },
      { key: 'voltaje_b', label: 'Fase B', unit: 'V' },
      { key: 'voltaje_c', label: 'Fase C', unit: 'V' },
      { key: 'voltaje_avg', label: 'Promedio', unit: 'V' },
      { key: 'voltaje_ab', label: 'A-B', unit: 'V' },
      { key: 'voltaje_bc', label: 'B-C', unit: 'V' },
      { key: 'voltaje_ac', label: 'A-C', unit: 'V' }
    ]
  },
  {
    title: 'Potencia',
    fields: [
      { key: 'potencia_activa', label: 'Activa', unit: 'W' },
      { key: 'potencia_reactiva', label: 'Reactiva', unit: 'VAR' },
      { key: 'potencia_aparente', label: 'Aparente', unit: 'VA' },
      { key: 'factor_potencia', label: 'Factor', unit: '' },
      { key: 'frecuencia_hz', label: 'Frecuencia', unit: 'Hz' }
    ]
  },
  {
    title: 'Consumo por Intervalo',
    fields: [
      { key: 'consumo_activa', label: 'Activo', unit: 'kWh' },
      { key: 'consumo_reactiva', label: 'Reactivo', unit: 'kVARh' },
      { key: 'consumo_aparente', label: 'Aparente', unit: 'kVAh' }
    ]
  },
  {
    title: 'Energia acumulada (contador)',
    fields: [
      { key: 'energia_activa', label: 'Activa', unit: 'kWh' },
      { key: 'energia_reactiva', label: 'Reactiva', unit: 'kVARh' },
      { key: 'energia_aparente', label: 'Aparente', unit: 'kVAh' }
    ]
  },
  {
    title: 'THD',
    fields: [
      { key: 'thd_ia', label: 'THD IA', unit: '%' },
      { key: 'thd_ib', label: 'THD IB', unit: '%' },
      { key: 'thd_ic', label: 'THD IC', unit: '%' },
      { key: 'thd_ua', label: 'THD UA', unit: '%' },
      { key: 'thd_ub', label: 'THD UB', unit: '%' },
      { key: 'thd_uc', label: 'THD UC', unit: '%' }
    ]
  }
]

let refreshInterval = null

onMounted(() => {
  fetchMeters()
  fetchSnapshots()
  refreshInterval = setInterval(fetchSnapshots, 30000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div class="monitor-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">Monitor Energetico</h1>
        <p class="view-subtitle">Lecturas en tiempo real de medidores de energia</p>
      </div>
      <div class="header-meta">
        <span class="meta-total">{{ total }} registros</span>
        <span class="meta-refresh">Auto-refresh 30s</span>
      </div>
    </div>

    <!-- Summary cards -->
    <div class="summary-grid">
      <div v-for="card in summaryCards" :key="card.key" class="summary-card">
        <div :class="['summary-value', card.color]">{{ card.value }}</div>
        <div class="summary-label">{{ card.label }}</div>
        <div class="summary-unit" v-if="card.unit">{{ card.unit }}</div>
      </div>
    </div>

    <!-- Filters -->
    <div class="filter-bar">
      <select v-model="filterMeter" class="filter-select">
        <option value="">Todos los medidores</option>
        <option v-for="m in meters" :key="m.meter_id" :value="m.meter_id">
          {{ m.nombre || m.meter_id }}
        </option>
      </select>
      <input v-model="filterFrom" type="datetime-local" class="filter-input" />
      <input v-model="filterTo" type="datetime-local" class="filter-input" />
      <button class="btn btn-primary btn-sm" @click="applyFilters">Filtrar</button>
      <button class="btn btn-secondary btn-sm" @click="clearFilters">Limpiar</button>
    </div>

    <!-- Data table -->
    <div class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th v-for="col in tableColumns" :key="col.key">
              {{ col.label }}
              <span v-if="col.unit" class="th-unit">({{ col.unit }})</span>
            </th>
            <th>Detalle</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !snapshots.length">
            <td :colspan="tableColumns.length + 1" class="table-empty">Cargando...</td>
          </tr>
          <tr v-else-if="!snapshots.length">
            <td :colspan="tableColumns.length + 1" class="table-empty">No hay snapshots</td>
          </tr>
          <tr v-for="s in snapshots" :key="s.id" class="table-row-hover">
            <td v-for="col in tableColumns" :key="col.key" :class="{ 'font-mono text-sm': col.key !== 'meter_id' }">
              {{ formatVal(s[col.key], col) }}
            </td>
            <td>
              <button class="btn-icon" title="Ver detalle completo" @click="openDetail(s)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" stroke="currentColor" stroke-width="2"/><circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2"/></svg>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div class="pagination" v-if="totalPages > 1">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="prevPage">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M15 18l-6-6 6-6" stroke="currentColor" stroke-width="2"/></svg>
        Anterior
      </button>
      <span class="pagination-info">{{ page }} / {{ totalPages }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="page >= totalPages" @click="nextPage">
        Siguiente
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none"><path d="M9 18l6-6-6-6" stroke="currentColor" stroke-width="2"/></svg>
      </button>
    </div>

    <!-- Detail panel -->
    <Teleport to="body">
      <div v-if="detailOpen" class="modal-overlay" @click.self="closeDetail">
        <div class="modal-content modal-detail">
          <div class="modal-header">
            <h2>Snapshot {{ selectedSnapshot?.meter_id }} - {{ formatDate(selectedSnapshot?.hora) }}</h2>
            <button class="btn-close" @click="closeDetail">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
            </button>
          </div>
          <div class="modal-body detail-body" v-if="selectedSnapshot">
            <div class="detail-meta">
              <span>Device: <code>{{ selectedSnapshot.device_id }}</code></span>
              <span>Intervalo: {{ selectedSnapshot.interval_s }}s</span>
            </div>
            <div v-for="group in detailGroups" :key="group.title" class="detail-group">
              <h3 class="detail-group-title">{{ group.title }}</h3>
              <div class="detail-grid">
                <div v-for="f in group.fields" :key="f.key" class="detail-field">
                  <span class="detail-field-label">{{ f.label }}</span>
                  <span class="detail-field-value">
                    {{ selectedSnapshot[f.key] != null ? Number(selectedSnapshot[f.key]).toFixed(3) : '-' }}
                    <span class="detail-field-unit" v-if="f.unit">{{ f.unit }}</span>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.monitor-view { @apply p-6; }
.view-header { @apply flex items-center justify-between mb-6; }
.view-title { @apply text-2xl font-bold text-white; }
.view-subtitle { @apply text-sm text-gray-400 mt-1; }
.header-meta { @apply flex items-center gap-4; }
.meta-total { @apply text-sm text-gray-300 font-mono; }
.meta-refresh { @apply text-xs text-gray-500 bg-primary-800 px-2 py-1 rounded; }

.summary-grid { @apply grid grid-cols-4 gap-4 mb-6; }
.summary-card { @apply bg-primary-800 rounded-lg p-4 border border-primary-700 text-center; }
.summary-value { @apply text-2xl font-bold; }
.summary-label { @apply text-xs text-gray-400 mt-1 uppercase tracking-wide; }
.summary-unit { @apply text-xs text-gray-500 mt-0.5; }

.filter-bar { @apply flex items-center gap-3 mb-4 flex-wrap; }
.filter-select {
  @apply bg-primary-800 border border-primary-700 rounded-lg px-3 py-2;
  @apply text-white text-sm focus:outline-none focus:border-secondary-500;
}
.filter-input {
  @apply bg-primary-800 border border-primary-700 rounded-lg px-3 py-2;
  @apply text-white text-sm focus:outline-none focus:border-secondary-500;
}

.table-container { @apply bg-primary-800 rounded-lg border border-primary-700 overflow-x-auto; }
.data-table { @apply w-full; }
.data-table th {
  @apply text-left px-3 py-3 text-xs uppercase tracking-wider text-gray-400;
  @apply bg-primary-900 border-b border-primary-700 whitespace-nowrap;
}
.th-unit { @apply text-gray-500 normal-case; }
.data-table td { @apply px-3 py-2.5 text-sm text-gray-300 border-b border-primary-700; }
.data-table tr:last-child td { @apply border-b-0; }
.table-empty { @apply text-center py-8 text-gray-500; }
.table-row-hover:hover { @apply bg-primary-700/40; }

.btn { @apply inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors; }
.btn-sm { @apply px-3 py-1.5 text-xs; }
.btn-primary { @apply bg-secondary-600 text-white hover:bg-secondary-700; }
.btn-secondary { @apply bg-primary-700 text-gray-300 hover:bg-primary-600; }
.btn:disabled { @apply opacity-50 cursor-not-allowed; }
.btn-icon { @apply p-1.5 rounded text-gray-400 hover:text-white hover:bg-primary-700 transition-colors; }

.pagination { @apply flex items-center justify-center gap-4 mt-4; }
.pagination-info { @apply text-sm text-gray-400; }

.modal-overlay { @apply fixed inset-0 bg-black/60 flex items-center justify-center z-50; }
.modal-content { @apply bg-primary-800 rounded-xl border border-primary-700 w-full shadow-2xl; }
.modal-detail { @apply max-w-3xl max-h-[85vh] flex flex-col; }
.modal-header { @apply flex items-center justify-between px-6 py-4 border-b border-primary-700 flex-shrink-0; }
.modal-header h2 { @apply text-lg font-semibold text-white; }
.btn-close { @apply text-gray-400 hover:text-white transition-colors; }
.modal-body { @apply px-6 py-4; }
.detail-body { @apply overflow-y-auto space-y-5; }

.detail-meta {
  @apply flex gap-6 text-sm text-gray-400 bg-primary-900 rounded-lg px-4 py-2;
}
.detail-meta code { @apply text-secondary-400 font-mono; }

.detail-group { @apply space-y-2; }
.detail-group-title { @apply text-sm font-semibold text-gray-300 uppercase tracking-wide border-b border-primary-700 pb-1; }
.detail-grid { @apply grid grid-cols-3 gap-3; }
.detail-field { @apply flex flex-col bg-primary-900 rounded px-3 py-2; }
.detail-field-label { @apply text-xs text-gray-500; }
.detail-field-value { @apply text-sm text-white font-mono; }
.detail-field-unit { @apply text-xs text-gray-500 ml-1; }
</style>
