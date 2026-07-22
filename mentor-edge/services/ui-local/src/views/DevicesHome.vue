<template>
  <div class="space-y-6">

    <!-- Header -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">Líneas de Producción</h1>
        <p class="text-sm text-gray-400 mt-1">Selecciona una línea para configurarla</p>
      </div>
      <div class="flex items-center gap-2">
        <button @click="load" :disabled="loading"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-40 transition-colors">
          <svg class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          Actualizar
        </button>
        <button @click="createNextLine" :disabled="creatingLine"
          class="flex items-center gap-1.5 px-4 py-1.5 rounded text-xs font-medium bg-emerald-700 hover:bg-emerald-600 text-white transition-colors disabled:opacity-40">
          <svg v-if="creatingLine" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          Nueva Línea
        </button>
      </div>
    </div>

    <!-- Ajustes globales del Jetson -->
    <div class="bg-slate-800/60 border border-slate-700/60 rounded-xl p-4">
      <p class="text-[11px] font-semibold text-gray-500 uppercase tracking-wider mb-3">Ajustes globales de este Jetson</p>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">

        <!-- ID del dispositivo (Jetson board identifier) -->
        <div>
          <label class="block text-[11px] text-gray-400 mb-1">
            ID de este dispositivo (Jetson)
            <span class="text-blue-500/70 ml-1">— identifica la placa física</span>
          </label>
          <div class="flex items-center gap-2">
            <input v-model="editDeviceId" type="text"
              :disabled="savingDeviceId"
              class="flex-1 px-3 py-1.5 rounded-lg text-sm font-mono bg-slate-900 border border-slate-600 text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none disabled:opacity-50"
              placeholder="Ej: jetson-orin-01"
              @keyup.enter="saveDeviceId">
            <svg v-if="savingDeviceId" class="w-3.5 h-3.5 animate-spin text-blue-400 shrink-0" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            <button v-else @click="saveDeviceId"
              class="shrink-0 px-3 py-1.5 rounded-lg text-xs font-medium bg-blue-700 hover:bg-blue-600 text-white transition-colors">
              Guardar
            </button>
            <Transition enter-from-class="opacity-0" enter-active-class="transition-opacity duration-300" leave-active-class="transition-opacity duration-300" leave-to-class="opacity-0">
              <span v-if="deviceIdSaved" class="text-[11px] text-emerald-400 shrink-0">✓</span>
            </Transition>
          </div>
          <p class="mt-1 text-[10px] text-gray-600">
            <template v-if="activeDeviceId">Dispositivo actual: <strong class="text-emerald-400">{{ activeDeviceId }}</strong>. Este ID se comparte entre todas las líneas.</template>
            <template v-else>Escribe un ID para identificar esta placa Jetson.</template>
          </p>
          <p v-if="deviceIdError" class="mt-1 text-[10px] text-red-400">{{ deviceIdError }}</p>
        </div>

        <!-- URL del servidor cloud -->
        <div>
          <label class="block text-[11px] text-gray-400 mb-1">
            URL del servidor cloud
            <span class="text-blue-500/70 ml-1">— guardado en base de datos, se replica</span>
          </label>
          <div class="flex items-center gap-2">
            <input v-model="defaultCloudUrl" type="text"
              :disabled="savingCloudUrl"
              class="flex-1 px-3 py-1.5 rounded-lg text-sm font-mono bg-slate-900 border border-slate-600 text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none disabled:opacity-50"
              placeholder="http://192.168.100.29:8888"
              @keyup.enter="saveCloudUrl">
            <svg v-if="savingCloudUrl" class="w-3.5 h-3.5 animate-spin text-blue-400 shrink-0" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            <button v-else @click="saveCloudUrl"
              class="shrink-0 px-3 py-1.5 rounded-lg text-xs font-medium bg-blue-700 hover:bg-blue-600 text-white transition-colors">
              Guardar
            </button>
            <Transition enter-from-class="opacity-0" enter-active-class="transition-opacity duration-300" leave-active-class="transition-opacity duration-300" leave-to-class="opacity-0">
              <span v-if="cloudUrlSaved" class="text-[11px] text-emerald-400 shrink-0">✓</span>
            </Transition>
          </div>
          <p class="mt-1 text-[10px] text-gray-600">Se pre-rellena al crear o configurar nuevas líneas.</p>
        </div>

      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-20 text-gray-500 text-sm">
      <svg class="w-6 h-6 animate-spin mx-auto mb-3 text-blue-400" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
      </svg>
      Cargando líneas...
    </div>

    <!-- Sin líneas -->
    <div v-else-if="lines.length === 0"
      class="text-center py-20 bg-slate-800 rounded-xl border border-dashed border-slate-600">
      <svg class="w-12 h-12 text-gray-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
      </svg>
      <p class="text-gray-400 text-sm mb-4">No hay líneas registradas aún</p>
      <button @click="createNextLine" :disabled="creatingLine"
        class="px-5 py-2 rounded-lg text-sm font-medium bg-emerald-700 hover:bg-emerald-600 text-white transition-colors disabled:opacity-40">
        + Crear primera línea
      </button>
    </div>

    <!-- Grid de líneas -->
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="l in lines" :key="l.linea_id"
        @click="$router.push('/config/' + l.linea_id)"
        class="bg-slate-800 rounded-xl border border-slate-700 p-5 cursor-pointer hover:border-blue-500/60 hover:bg-slate-750 transition-all group">

        <!-- Cabecera -->
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="w-9 h-9 rounded-lg bg-blue-600/20 border border-blue-500/30 flex items-center justify-center shrink-0">
              <svg class="w-4.5 h-4.5 w-[18px] h-[18px] text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
              </svg>
            </div>
            <div class="min-w-0">
              <p class="font-mono text-sm font-bold text-white truncate">Línea {{ l.linea_id }}</p>
              <p class="text-[11px] text-gray-500 mt-0.5">{{ l.line_name || 'Sin nombre OEE' }}</p>
            </div>
          </div>
          <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-slate-700 text-gray-400 shrink-0 ml-2 mt-0.5">
            v{{ l.config_version ?? '—' }}
          </span>
        </div>

        <!-- Metadatos -->
        <div class="space-y-2 mb-4">
          <div class="flex items-center gap-2 text-xs">
            <span class="text-gray-500 w-20 shrink-0">Modo</span>
            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium"
              :class="l.mode === 'textil' ? 'bg-indigo-900/40 text-indigo-300' : 'bg-amber-900/40 text-amber-300'">
              {{ l.mode ? (l.mode.charAt(0).toUpperCase() + l.mode.slice(1)) : '—' }}
            </span>
          </div>
          <div v-if="l.camera_url" class="flex items-center gap-2 text-xs">
            <span class="text-gray-500 w-20 shrink-0">Cámara</span>
            <span class="text-gray-400 font-mono text-[10px] truncate">{{ l.camera_url }}</span>
          </div>
        </div>

        <!-- Botones de acción -->
        <div class="pt-3 border-t border-slate-700/60 flex gap-2">
          <button @click.stop="$router.push('/config/' + l.linea_id)"
            class="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg text-xs font-semibold
                   bg-blue-600/15 hover:bg-blue-600 text-blue-300 hover:text-white
                   border border-blue-500/25 hover:border-blue-600 transition-all">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37a1.724 1.724 0 002.572-1.065z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            </svg>
            Configurar
          </button>
          <button @click.stop="confirmDelete(l)"
            class="flex items-center justify-center w-9 py-2 rounded-lg text-xs font-semibold
                   bg-red-600/10 hover:bg-red-600 text-red-400 hover:text-white
                   border border-red-500/20 hover:border-red-600 transition-all" title="Eliminar línea">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

  </div>

  <!-- Modal confirmación eliminar -->
  <Teleport to="body">
    <div v-if="deleteTarget" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div class="bg-slate-800 border border-slate-600 rounded-xl shadow-2xl p-6 w-full max-w-sm mx-4">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full bg-red-900/40 border border-red-500/30 flex items-center justify-center shrink-0">
            <svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
            </svg>
          </div>
          <div>
            <h3 class="text-white font-semibold text-sm">Eliminar línea</h3>
            <p class="text-gray-400 text-xs mt-0.5">Esta acción no se puede deshacer</p>
          </div>
        </div>
        <p class="text-gray-300 text-sm mb-5">
          ¿Confirmas eliminar la <span class="font-mono text-white font-bold">Línea {{ deleteTarget.linea_id }}</span>?
          Se borrarán su configuración y ROI permanentemente.
        </p>
        <div class="flex gap-2 justify-end">
          <button @click="deleteTarget = null" :disabled="deleting"
            class="px-4 py-2 rounded-lg text-xs font-medium bg-slate-700 hover:bg-slate-600 text-gray-300 transition-colors disabled:opacity-40">
            Cancelar
          </button>
          <button @click="doDelete" :disabled="deleting"
            class="flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-medium bg-red-700 hover:bg-red-600 text-white transition-colors disabled:opacity-40">
            <svg v-if="deleting" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ deleting ? 'Eliminando…' : 'Sí, eliminar' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>



</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { configService } from '../services/api.js'

const router = useRouter()

const lines = ref([])
const loading = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)

// Device ID global del Jetson (identifica la placa física)
const activeDeviceId = ref('')
const editDeviceId = ref('')
const savingDeviceId = ref(false)
const deviceIdSaved = ref(false)
const deviceIdError = ref('')

// URL del servidor cloud: guardado en DB (_system), se replica
const defaultCloudUrl = ref('')
const cloudUrlSaved = ref(false)
const savingCloudUrl = ref(false)

const creatingLine = ref(false)

const saveCloudUrl = async () => {
  savingCloudUrl.value = true
  try {
    await configService.setSystemDefaults({ cloud: { url: defaultCloudUrl.value.trim() } })
    cloudUrlSaved.value = true
    setTimeout(() => { cloudUrlSaved.value = false }, 2000)
  } catch (e) {
    console.error('Error al guardar URL cloud:', e)
  } finally {
    savingCloudUrl.value = false
  }
}

const saveDeviceId = async () => {
  const newId = editDeviceId.value.trim()
  if (!newId) return
  deviceIdError.value = ''
  savingDeviceId.value = true
  try {
    await configService.setDeviceId(newId)
    activeDeviceId.value = newId
    deviceIdSaved.value = true
    setTimeout(() => { deviceIdSaved.value = false }, 3000)
  } catch (e) {
    deviceIdError.value = e.response?.data || e.message || 'Error al guardar device_id'
  } finally {
    savingDeviceId.value = false
  }
}

const load = async () => {
  loading.value = true
  try {
    // Cargar defaults del sistema (cloud URL desde DB)
    const sys = await configService.getSystemDefaults()
    if (sys?.cloud?.url) {
      defaultCloudUrl.value = sys.cloud.url
    } else {
      defaultCloudUrl.value = 'http://152.53.253.59:8888'
    }
    // Cargar device_id global
    try {
      const devData = await configService.getDeviceId()
      activeDeviceId.value = devData?.device_id || ''
      if (!editDeviceId.value) editDeviceId.value = activeDeviceId.value
    } catch { /* no device_id yet */ }

    // Cargar lista de líneas
    const data = await configService.getLines()
    const lineIds = data?.lines ?? []

    const metas = await Promise.allSettled(
      lineIds.map(id =>
        configService.getConfig(id).then(cfg => ({
          linea_id: id,
          config_version: cfg.config_version ?? 0,
          line_name: cfg.oee?.line_name || '',
          mode: cfg.mode || '',
          camera_url: cfg.camera?.url || '',
        }))
      )
    )
    lines.value = metas
      .filter(r => r.status === 'fulfilled')
      .map(r => r.value)
  } catch {
    lines.value = []
  } finally {
    loading.value = false
  }
}

const createNextLine = async () => {
  const usedIds = lines.value.map(l => l.linea_id).filter(n => !isNaN(n))
  const nextId = usedIds.length > 0 ? Math.max(...usedIds) + 1 : 1
  // Navegar a la config sin pre-guardar en BD.
  // La línea solo se crea cuando el usuario presiona "Guardar" en la config.
  router.push('/config/' + nextId + '?new=1')
}

const confirmDelete = (line) => {
  deleteTarget.value = line
}

const doDelete = async () => {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await configService.deleteLine(deleteTarget.value.linea_id)
    lines.value = lines.value.filter(l => l.linea_id !== deleteTarget.value.linea_id)
    deleteTarget.value = null
  } catch (err) {
    console.error('Error al eliminar línea:', err)
  } finally {
    deleting.value = false
  }
}

onMounted(load)
</script>
