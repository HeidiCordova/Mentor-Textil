<template>
  <div class="space-y-6">

    <div>
      <h1 class="text-2xl font-bold text-white">Dashboard</h1>
      <p class="mt-0.5 text-sm text-gray-400">Estado del sistema de medicion de energia</p>
    </div>

    <!-- Stats cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">

      <div class="card">
        <p class="label">Total snapshots</p>
        <p class="text-2xl font-bold text-white">{{ stats.total ?? '—' }}</p>
      </div>

      <div class="card">
        <p class="label">Sincronizados</p>
        <p class="text-2xl font-bold text-green-400">{{ stats.synced ?? '—' }}</p>
      </div>

      <div class="card">
        <p class="label">Pendientes</p>
        <p class="text-2xl font-bold" :class="(stats.pending ?? 0) > 0 ? 'text-amber-400' : 'text-gray-400'">
          {{ stats.pending ?? '—' }}
        </p>
      </div>

      <div class="card">
        <p class="label">Ultima sincronizacion</p>
        <p class="text-xs font-mono text-gray-300 leading-snug mt-1">
          {{ stats.last_sync ? fmtDate(stats.last_sync) : 'Sin datos' }}
        </p>
      </div>

    </div>

    <!-- Medidores activos -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-semibold text-gray-200">Medidores configurados</h2>
        <span class="text-xs text-gray-500">{{ meters.length }} activo{{ meters.length !== 1 ? 's' : '' }}</span>
      </div>
      <div v-if="meters.length === 0" class="text-sm text-gray-500 py-2">
        Sin medidores configurados. Ve a Configuracion para agregarlos.
      </div>
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <div v-for="m in meters" :key="m.id"
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-slate-700/50 border border-slate-600">
          <div class="w-8 h-8 rounded-full bg-blue-600/20 border border-blue-500/30 flex items-center justify-center shrink-0">
            <span class="text-xs font-bold text-blue-400">{{ m.id }}</span>
          </div>
          <div class="min-w-0">
            <p class="text-sm font-medium text-gray-200 truncate">{{ m.meter_id }}</p>
            <p class="text-xs text-gray-500">Unit ID: {{ m.unit_id }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Ultimos snapshots -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-semibold text-gray-200">Ultimos snapshots</h2>
        <div class="flex gap-1">
          <button v-for="f in filters" :key="f.value"
            @click="activeFilter = f.value; loadSnapshots()"
            class="px-2.5 py-1 text-xs rounded font-medium transition-colors"
            :class="activeFilter === f.value ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white hover:bg-slate-700'">
            {{ f.label }}
          </button>
        </div>
      </div>

      <div v-if="loadingSnap" class="py-6 flex justify-center">
        <svg class="w-5 h-5 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
        </svg>
      </div>

      <div v-else-if="snapshots.length === 0" class="text-sm text-gray-500 py-4 text-center">
        Sin datos
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-xs">
          <thead>
            <tr class="text-gray-500 border-b border-slate-700">
              <th class="text-left pb-2 pr-4 font-medium">Medidor</th>
              <th class="text-left pb-2 pr-4 font-medium">Hora</th>
              <th class="text-left pb-2 pr-4 font-medium">Intervalo</th>
              <th class="text-left pb-2 font-medium">Estado</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-700/50">
            <tr v-for="s in snapshots" :key="s.id" class="text-gray-300">
              <td class="py-2 pr-4 font-mono">{{ s.meter_id }}</td>
              <td class="py-2 pr-4 font-mono">{{ fmtDate(s.hora) }}</td>
              <td class="py-2 pr-4">{{ s.interval_s }}s</td>
              <td class="py-2">
                <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium"
                  :class="s.synced ? 'bg-green-900/40 text-green-400' : 'bg-amber-900/40 text-amber-400'">
                  <span class="w-1.5 h-1.5 rounded-full"
                    :class="s.synced ? 'bg-green-400' : 'bg-amber-400'"></span>
                  {{ s.synced ? 'Sync' : 'Pendiente' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getStats, getSnapshots, getMeters } from '../services/api.js'

const stats = ref({})
const snapshots = ref([])
const meters = ref([])
const loadingSnap = ref(true)
const activeFilter = ref('')

const filters = [
  { label: 'Todos', value: '' },
  { label: 'Pendientes', value: 'pending' },
  { label: 'Sincronizados', value: 'synced' },
]

function fmtDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleString('es', { dateStyle: 'short', timeStyle: 'short' })
}

async function loadSnapshots() {
  loadingSnap.value = true
  try {
    snapshots.value = await getSnapshots(activeFilter.value)
  } finally {
    loadingSnap.value = false
  }
}

onMounted(async () => {
  const [s, m] = await Promise.allSettled([getStats(), getMeters()])

  if (s.status === 'fulfilled') stats.value = s.value
  if (m.status === 'fulfilled') meters.value = m.value

  await loadSnapshots()
})
</script>
