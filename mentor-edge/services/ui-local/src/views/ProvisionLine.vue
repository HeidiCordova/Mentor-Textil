<template>
  <div class="max-w-xl mx-auto space-y-6">

    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Habilitar Línea</h1>
        <p class="text-xs text-gray-500 mt-0.5">Se actualiza automáticamente cada 15 segundos</p>
      </div>
      <button @click="loadAll" :disabled="loadingAll"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-40 transition-colors">
        <svg class="w-3.5 h-3.5" :class="loadingAll ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Actualizar
      </button>
    </div>

    <!-- Info: flujo normal no requiere esta página -->
    <div class="flex items-start gap-3 px-4 py-3 rounded-lg bg-blue-900/20 border border-blue-500/20 text-xs text-blue-300">
      <svg class="w-4 h-4 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      <span>Al crear un dispositivo desde <strong>Dispositivos → Nuevo Dispositivo</strong>, el schema se crea automáticamente. Esta página es para líneas sincronizadas desde la nube que aún no tienen schema local.</span>
    </div>

    <!-- Líneas pendientes de provisionar -->
    <div class="bg-slate-800 rounded-xl border border-slate-700 p-5">
      <div class="flex items-center gap-2 mb-4">
        <span class="w-2 h-2 rounded-full bg-amber-400 shrink-0"></span>
        <h2 class="text-sm font-semibold text-white">Líneas pendientes</h2>
        <span v-if="pending.length > 0"
          class="ml-auto text-[10px] font-bold px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
          {{ pending.length }}
        </span>
      </div>

      <div v-if="loadingAll" class="text-gray-500 text-xs py-3 text-center">Detectando líneas...</div>
      <div v-else-if="pending.length === 0"
        class="flex items-center gap-2 text-xs text-gray-500 py-3">
        <svg class="w-4 h-4 text-green-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
        </svg>
        Todas las líneas registradas ya están provisionadas.
      </div>
      <ul v-else class="space-y-2">
        <li v-for="id in pending" :key="id"
          class="flex items-center justify-between gap-3 px-4 py-3 rounded-lg bg-amber-900/10 border border-amber-500/20">
          <div class="flex items-center gap-2.5 min-w-0">
            <span class="w-2 h-2 rounded-full bg-amber-400 shrink-0 animate-pulse"></span>
            <div>
              <p class="text-sm font-mono font-semibold text-white">Línea {{ id }}</p>
              <p class="text-[10px] text-gray-500 mt-0.5">Config registrada · schema pendiente</p>
            </div>
          </div>
          <button @click="provisionOne(id)" :disabled="provisioningId === id"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold
                   bg-emerald-700 hover:bg-emerald-600 text-white disabled:opacity-50 transition-colors shrink-0">
            <svg v-if="provisioningId === id" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            {{ provisioningId === id ? 'Creando...' : 'Provisionar' }}
          </button>
        </li>
      </ul>
    </div>

    <!-- Schemas activos -->
    <div class="bg-slate-800 rounded-xl border border-slate-700 p-5">
      <div class="flex items-center gap-2 mb-3">
        <span class="w-2 h-2 rounded-full bg-green-500 shrink-0"></span>
        <h2 class="text-sm font-semibold text-white">Schemas provisionados</h2>
        <span class="ml-auto text-[10px] text-gray-500">{{ schemas.length }} activo{{ schemas.length !== 1 ? 's' : '' }}</span>
      </div>
      <div v-if="loadingAll" class="text-gray-500 text-xs">Cargando...</div>
      <div v-else-if="schemas.length === 0" class="text-gray-500 text-xs">No hay schemas de línea creados.</div>
      <ul v-else class="space-y-1.5">
        <li v-for="s in schemas" :key="s"
          class="flex items-center gap-2 text-xs text-gray-300 py-2 px-3 bg-slate-700/40 rounded-lg">
          <span class="w-1.5 h-1.5 rounded-full bg-green-400 shrink-0"></span>
          <span class="font-mono">{{ s }}</span>
        </li>
      </ul>
    </div>

    <!-- Provisionar manualmente (nuevo ID no registrado) -->
    <div class="bg-slate-800 rounded-xl border border-slate-700 p-5">
      <div class="flex items-center gap-2 mb-3">
        <span class="w-2 h-2 rounded-full bg-blue-400 shrink-0"></span>
        <h2 class="text-sm font-semibold text-white">Aprovisionar ID nuevo</h2>
      </div>
      <p class="text-xs text-gray-500 mb-3">Para líneas que aún no tienen config registrada.</p>

      <form @submit.prevent="provisionManual" class="flex gap-2">
        <input
          v-model="manualId"
          type="text"
          placeholder="Ej: 5"
          class="flex-1 bg-slate-900 border border-slate-600 rounded-lg px-3 py-2 text-white text-sm
                 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <button type="submit" :disabled="manualLoading || !manualId.trim()"
          class="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium
                 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 disabled:cursor-not-allowed
                 text-white transition-colors shrink-0">
          <svg v-if="manualLoading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          {{ manualLoading ? 'Creando...' : 'Crear Schema' }}
        </button>
      </form>

      <div v-if="manualResult" class="mt-3 p-3 rounded-lg text-xs" :class="manualResultClass">
        <p class="font-medium">{{ manualResult.title }}</p>
        <p class="mt-0.5 opacity-80">{{ manualResult.detail }}</p>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { configService } from '../services/api.js'

const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL || '/api/gateway'

const schemas = ref([])
const configDevices = ref([])
const loadingAll = ref(false)
const provisioningId = ref(null)

// IDs con config pero sin schema → pendientes de provisionar
const pending = computed(() => {
  const schemaIds = new Set(schemas.value.map(s => s.replace('linea_', '')))
  return configDevices.value.filter(id => !schemaIds.has(String(id)))
})

const manualId = ref('')
const manualLoading = ref(false)
const manualResult = ref(null)
const manualResultClass = ref('')

async function loadAll() {
  loadingAll.value = true
  try {
    const [schemasResp, devicesResp] = await Promise.allSettled([
      axios.get(`${GATEWAY_URL}/edge/schemas`),
      configService.getDevices(),
    ])
    if (schemasResp.status === 'fulfilled')
      schemas.value = schemasResp.value.data?.schemas || []
    if (devicesResp.status === 'fulfilled')
      configDevices.value = (devicesResp.value?.devices || []).filter(id => id !== '_system')
  } finally {
    loadingAll.value = false
  }
}

async function provisionOne(id) {
  provisioningId.value = id
  try {
    await axios.post(`${GATEWAY_URL}/edge/provision`, { linea_id: id })
  } catch (err) {
    if (err.response?.status !== 409) console.error(err)
  } finally {
    provisioningId.value = null
    await loadAll()
  }
}

async function provisionManual() {
  const id = manualId.value.trim()
  if (!id) return
  manualLoading.value = true
  manualResult.value = null
  try {
    const resp = await axios.post(`${GATEWAY_URL}/edge/provision`, { linea_id: id })
    manualResult.value = {
      title: `Schema ${resp.data.schema} creado`,
      detail: 'Todas las tablas fueron creadas. La línea está lista para operar.'
    }
    manualResultClass.value = 'bg-green-900/40 border border-green-700 text-green-300'
    manualId.value = ''
    await loadAll()
  } catch (err) {
    if (err.response?.status === 409) {
      manualResult.value = { title: 'El schema ya existe', detail: 'No se requiere acción adicional.' }
      manualResultClass.value = 'bg-yellow-900/40 border border-yellow-700 text-yellow-300'
    } else {
      manualResult.value = { title: 'Error al crear schema', detail: err.response?.data?.error || err.message }
      manualResultClass.value = 'bg-red-900/40 border border-red-700 text-red-300'
    }
  } finally {
    manualLoading.value = false
  }
}

let refreshInterval = null
onMounted(() => { loadAll(); refreshInterval = setInterval(loadAll, 15000) })
onUnmounted(() => { if (refreshInterval) clearInterval(refreshInterval) })
</script>
