<template>
  <div class="space-y-5">

    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Registro de Dispositivo</h1>
        <p class="mt-0.5 text-sm text-gray-400">Mapeo dispositivo - linea y estado de conexion al cloud</p>
      </div>
      <button @click="refresh" :disabled="loading"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-40 transition-colors">
        <svg class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        {{ loading ? 'Cargando...' : 'Actualizar' }}
      </button>
    </div>

    <!-- Estado de conexion -->
    <div class="card">
      <div class="flex items-center gap-3 mb-4">
        <span class="w-3 h-3 rounded-full shrink-0"
          :class="status.cloud_connected ? 'bg-green-400 animate-pulse' : 'bg-red-400'"></span>
        <h2 class="text-sm font-semibold text-white">
          {{ status.cloud_connected ? 'Conexion al cloud establecida' : 'Sin conexion al cloud' }}
        </h2>
      </div>

      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="bg-slate-900/60 rounded-lg p-3">
          <p class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Device ID</p>
          <p class="text-lg font-mono font-bold text-white">{{ status.device_id || '-' }}</p>
        </div>
        <div class="bg-slate-900/60 rounded-lg p-3">
          <p class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Linea ID (cloud)</p>
          <p class="text-lg font-mono font-bold text-blue-400">{{ status.linea_id || '-' }}</p>
        </div>
        <div class="bg-slate-900/60 rounded-lg p-3">
          <p class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Uptime</p>
          <p class="text-lg font-mono font-bold text-white">{{ formatUptime(status.uptime) }}</p>
        </div>
        <div class="bg-slate-900/60 rounded-lg p-3">
          <p class="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Buffer pendiente</p>
          <p class="text-lg font-mono font-bold" :class="status.buffer_pending > 0 ? 'text-yellow-400' : 'text-green-400'">
            {{ status.buffer_pending ?? '-' }}
          </p>
        </div>
      </div>
    </div>

    <!-- Tabla mapeo device - linea -->
    <div class="card">
      <h2 class="text-sm font-semibold text-white mb-3">Mapeo Device - Linea</h2>
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead>
            <tr class="text-[10px] text-gray-500 uppercase tracking-wider border-b border-slate-700">
              <th class="pb-2 pr-4">Estado</th>
              <th class="pb-2 pr-4">Device ID</th>
              <th class="pb-2 pr-4">Linea ID</th>
              <th class="pb-2 pr-4">Schema local</th>
              <th class="pb-2 pr-4">Cloud</th>
              <th class="pb-2">Config version</th>
            </tr>
          </thead>
          <tbody>
            <tr class="border-b border-slate-700/50">
              <td class="py-3 pr-4">
                <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-medium"
                  :class="status.cloud_connected
                    ? 'bg-green-900/40 text-green-400'
                    : 'bg-red-900/40 text-red-400'">
                  <span class="w-1.5 h-1.5 rounded-full"
                    :class="status.cloud_connected ? 'bg-green-400' : 'bg-red-400'"></span>
                  {{ status.cloud_connected ? 'online' : 'offline' }}
                </span>
              </td>
              <td class="py-3 pr-4 font-mono font-medium text-white">{{ status.device_id || '-' }}</td>
              <td class="py-3 pr-4 font-mono font-medium text-blue-400">{{ status.linea_id || '-' }}</td>
              <td class="py-3 pr-4">
                <span v-if="activeSchema" class="font-mono text-green-400">{{ activeSchema }}</span>
                <span v-else class="text-gray-600">-</span>
              </td>
              <td class="py-3 pr-4">
                <span class="inline-flex items-center gap-1.5">
                  <span class="w-1.5 h-1.5 rounded-full"
                    :class="status.cloud_connected ? 'bg-green-400' : 'bg-red-400'"></span>
                  <span class="text-gray-300">{{ status.cloud_connected ? 'Sincronizando' : 'Desconectado' }}</span>
                </span>
              </td>
              <td class="py-3 font-mono text-gray-300">v{{ status.config_version ?? '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Schemas existentes -->
    <div class="card">
      <h2 class="text-sm font-semibold text-white mb-3">Schemas de linea disponibles</h2>
      <div v-if="schemas.length === 0" class="text-gray-500 text-sm">No hay schemas creados.</div>
      <div v-else class="grid grid-cols-2 md:grid-cols-4 gap-2">
        <div v-for="s in schemas" :key="s"
          class="flex items-center gap-2 text-sm py-2 px-3 rounded-lg"
          :class="s === activeSchema ? 'bg-blue-900/30 border border-blue-700/50 text-blue-400' : 'bg-slate-700/40 text-gray-300'">
          <span class="w-2 h-2 rounded-full shrink-0"
            :class="s === activeSchema ? 'bg-blue-400' : 'bg-gray-600'"></span>
          <span class="font-mono text-xs">{{ s }}</span>
          <span v-if="s === activeSchema" class="ml-auto text-[9px] uppercase tracking-wider text-blue-500">activo</span>
        </div>
      </div>
    </div>

    <!-- Errores recientes -->
    <div v-if="status.recent_errors && status.recent_errors.length > 0" class="card">
      <h2 class="text-sm font-semibold text-red-400 mb-3">Errores recientes</h2>
      <ul class="space-y-1">
        <li v-for="(err, i) in status.recent_errors" :key="i"
          class="text-xs text-red-300 bg-red-900/20 px-3 py-1.5 rounded font-mono">
          {{ err }}
        </li>
      </ul>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { gatewayService } from '../services/api.js'
import axios from 'axios'

const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL || '/api/gateway'

const status = ref({})
const schemas = ref([])
const loading = ref(false)
let interval = null

const activeSchema = computed(() => {
  if (!status.value.device_id) return null
  const expected = 'linea_' + status.value.device_id
  return schemas.value.includes(expected) ? expected : schemas.value[0] || null
})

function formatUptime(seconds) {
  if (!seconds && seconds !== 0) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

async function refresh() {
  loading.value = true
  try {
    const [st, sc] = await Promise.allSettled([
      gatewayService.getStatus(),
      axios.get(`${GATEWAY_URL}/edge/schemas`)
    ])
    if (st.status === 'fulfilled') status.value = st.value
    if (sc.status === 'fulfilled') schemas.value = sc.value.data?.schemas || []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  interval = setInterval(refresh, 10000)
})

onUnmounted(() => {
  if (interval) clearInterval(interval)
})
</script>

<style scoped>
.card {
  @apply bg-slate-800 rounded-xl border border-slate-700 p-5;
}
</style>
