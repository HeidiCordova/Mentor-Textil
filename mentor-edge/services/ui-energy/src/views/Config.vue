<template>
  <div class="space-y-6 max-w-3xl">

    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Configuracion</h1>
        <p class="mt-0.5 text-sm text-gray-400">Dispositivo y medidores Modbus</p>
      </div>
      <span v-if="savedAt" class="flex items-center gap-1.5 text-xs text-green-400">
        <span class="w-1.5 h-1.5 rounded-full bg-green-400"></span>
        Guardado {{ savedAt }}
      </span>
    </div>

    <!-- Dispositivo -->
    <div class="card space-y-4">
      <h2 class="text-sm font-semibold text-gray-200 flex items-center gap-2">
        <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
        </svg>
        Dispositivo
      </h2>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="label">Device ID</label>
          <input v-model="form.device_id" class="input-field" placeholder="rpi-energy-01" />
        </div>
        <div>
          <label class="label">URL Cloud</label>
          <input v-model="form.cloud_url" class="input-field" placeholder="https://api.ejemplo.com" />
        </div>
        <div class="sm:col-span-2">
          <label class="label">Token por defecto
            <span class="ml-1 text-gray-600 font-normal normal-case text-[10px]">(usado cuando el medidor no tiene token propio)</span>
          </label>
          <input v-model="form.energy_api_key" type="password" class="input-field" placeholder="••••••••" autocomplete="off" />
        </div>
        <div>
          <label class="label">Intervalo de envio (s)</label>
          <input v-model="form.send_interval_s" type="number" min="5" class="input-field" placeholder="30" />
        </div>
        <div>
          <label class="label">Tamano de batch</label>
          <input v-model="form.batch_size" type="number" min="1" class="input-field" placeholder="50" />
        </div>
        <div>
          <label class="label">Recarga de config (s)</label>
          <input v-model="form.config_reload_s" type="number" min="5" class="input-field" placeholder="15" />
        </div>
      </div>
      <div class="flex items-center gap-4 pt-2">
        <button @click="saveDeviceConfig" :disabled="saving" class="btn-primary flex items-center gap-2">
          <svg v-if="!saving" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
          </svg>
          <svg v-else class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          {{ saving ? 'Guardando...' : 'Guardar dispositivo' }}
        </button>
        <p v-if="saveError" class="text-xs text-red-400">{{ saveError }}</p>
      </div>
    </div>

    <!-- Medidores -->
    <div class="card space-y-5">
      <div class="flex items-center justify-between">
        <h2 class="text-sm font-semibold text-gray-200 flex items-center gap-2">
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M13 10V3L4 14h7v7l9-11h-7z"/>
          </svg>
          Medidores
          <span class="text-xs text-gray-500 font-normal">({{ meters.length }} configurados)</span>
        </h2>
        <div class="flex items-center gap-2">
          <button @click="startScan" :disabled="scanning"
            class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
                   bg-slate-700 text-gray-200 hover:bg-slate-600 transition-colors disabled:opacity-50">
            <svg v-if="scanning" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M21 21l-4.35-4.35M11 19a8 8 0 100-16 8 8 0 000 16z"/>
            </svg>
            {{ scanning ? 'Detectando...' : 'Detectar' }}
          </button>
          <button @click="openAddPanel"
            class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-500 transition-colors">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            Agregar medidor
          </button>
        </div>
      </div>

      <!-- Resultados del scan -->
      <div v-if="scanResult" class="rounded-xl border border-slate-700/60 bg-slate-800/40 p-4 space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-xs font-semibold text-gray-300">
            Detectados {{ scanResult.found.length }} medidor(es) en el bus (Unit IDs 1–{{ scanResult.scanned }})
          </p>
          <button @click="scanResult = null" class="text-gray-600 hover:text-gray-400">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
        <div v-if="scanResult.found.length === 0" class="text-xs text-gray-500">
          Ningun medidor respondio. Verificar cableado RS-485 y Unit IDs configurados en los MC60.
        </div>
        <div v-else class="flex flex-wrap gap-2">
          <div v-for="uid in scanResult.found" :key="uid"
            class="flex items-center gap-2 px-3 py-2 rounded-lg bg-slate-700/60 border border-slate-600/50">
            <span class="text-xs font-mono text-green-400">Unit ID {{ uid }}</span>
            <button v-if="!meterExistsByUnitId(uid)" @click="addScannedMeter(uid)"
              class="text-[10px] px-2 py-0.5 rounded bg-blue-600 text-white hover:bg-blue-500 transition-colors">
              Agregar
            </button>
            <span v-else class="text-[10px] text-gray-500">ya existe</span>
          </div>
        </div>
        <p v-if="scanError" class="text-xs text-red-400">{{ scanError }}</p>
      </div>


      <div v-if="metersLoading" class="py-10 flex justify-center">
        <svg class="w-5 h-5 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
        </svg>
      </div>

      <div v-else-if="meters.length === 0"
        class="py-12 flex flex-col items-center gap-3 text-center">
        <svg class="w-10 h-10 text-slate-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M13 10V3L4 14h7v7l9-11h-7z"/>
        </svg>
        <p class="text-sm text-gray-500">Sin medidores configurados.</p>
        <p class="text-xs text-gray-600">Agrega uno con el boton superior.</p>
      </div>

      <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
        <button
          v-for="m in meters" :key="m.id"
          @click="openEditPanel(m)"
          class="group relative flex flex-col items-center gap-3 px-3 pt-5 pb-4 rounded-xl
                 bg-slate-800/60 border border-slate-700/60 cursor-pointer
                 hover:border-blue-500/70 hover:bg-slate-700/60 transition-all duration-150
                 focus:outline-none focus:ring-2 focus:ring-blue-500/50">
          <div class="relative">
            <MeterSvg width="96" height="148" />
            <span class="absolute -top-2.5 -right-2.5 min-w-[22px] h-[22px] px-1 rounded-full
                         bg-blue-600 border-2 border-slate-800 flex items-center justify-center
                         text-[10px] font-bold text-white shadow">
              {{ m.unit_id }}
            </span>
            <span v-if="m.cloud_token"
              class="absolute -top-2.5 -left-2.5 w-[18px] h-[18px] rounded-full
                     bg-slate-800 border border-slate-600 flex items-center justify-center"
              title="Token de nube configurado">
              <svg class="w-2.5 h-2.5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"/>
              </svg>
            </span>
            <span class="absolute -bottom-1 left-1/2 -translate-x-1/2 w-2 h-2 rounded-full
                         bg-green-400 shadow shadow-green-400/60 ring-2 ring-slate-800"></span>
          </div>
          <div class="text-center w-full px-1">
            <p class="text-[13px] font-semibold text-gray-200 truncate leading-tight">{{ m.meter_id }}</p>
            <p class="text-[11px] text-gray-500 mt-0.5">Modbus {{ m.unit_id }}</p>
          </div>
          <div class="pointer-events-none absolute inset-0 rounded-xl flex items-center justify-center
                      opacity-0 group-hover:opacity-100 transition-opacity bg-blue-900/10 border border-blue-500/30">
            <span class="px-2 py-1 rounded-md bg-blue-600/90 text-[11px] font-medium text-white">
              Configurar
            </span>
          </div>
        </button>
      </div>
    </div>

    <!-- Drawer -->
    <Teleport to="body">
      <div v-if="panel" class="fixed inset-0 z-50 flex items-stretch">
        <div class="flex-1 bg-black/60 backdrop-blur-sm" @click="closePanel" />
        <aside class="w-80 bg-slate-950 border-l border-slate-700/80 flex flex-col shadow-2xl overflow-hidden">

          <!-- Header -->
          <div class="flex items-center justify-between px-5 py-4 border-b border-slate-800 shrink-0">
            <div class="flex items-center gap-2.5">
              <div class="w-2 h-2 rounded-full bg-green-400 shadow shadow-green-400/60"></div>
              <h3 class="text-sm font-semibold text-white">
                {{ panel.mode === 'add' ? 'Nuevo medidor' : 'Configurar medidor' }}
              </h3>
            </div>
            <button @click="closePanel"
              class="p-1.5 rounded-lg text-gray-500 hover:text-white hover:bg-slate-800 transition-colors">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- Body -->
          <div class="flex-1 flex flex-col gap-6 p-5 overflow-y-auto">

            <!-- Ilustracion grande -->
            <div class="flex justify-center py-2">
              <div class="relative">
                <MeterSvg width="130" height="200" />
                <span class="absolute -top-3 -right-3 min-w-[30px] h-[30px] px-1.5 rounded-full
                             bg-blue-600 border-2 border-slate-950 flex items-center justify-center
                             text-[13px] font-bold text-white shadow-lg">
                  {{ panel.unitId || '?' }}
                </span>
                <span class="absolute -bottom-1 left-1/2 -translate-x-1/2 w-3 h-3 rounded-full
                             bg-green-400 shadow shadow-green-400/70 ring-2 ring-slate-950"></span>
              </div>
            </div>

            <!-- Identificador (solo edit) -->
            <div v-if="panel.mode === 'edit'"
              class="flex items-center gap-2 px-3 py-2 rounded-lg bg-slate-800/60 border border-slate-700/40">
              <span class="text-xs text-gray-500">ID interno:</span>
              <span class="text-xs font-mono text-gray-400">{{ panel.id }}</span>
              <span class="ml-auto text-[10px] px-1.5 py-0.5 rounded bg-green-900/40 text-green-400 border border-green-800/40">Activo</span>
            </div>

            <!-- Campos -->
            <div class="space-y-4">
              <div>
                <label class="label">Nombre / Meter ID</label>
                <input v-model="panel.meterId" class="input-field" placeholder="medidor-sala-a"
                  :disabled="panel.saving || panel.deleting"
                  @keydown.enter="saveMeter" />
              </div>
              <div>
                <label class="label">Modbus Unit ID</label>
                <div class="flex items-center gap-2">
                  <input v-model.number="panel.unitId" type="number" min="1" max="247"
                    class="input-field w-28" placeholder="1"
                    :disabled="panel.saving || panel.deleting"
                    @keydown.enter="saveMeter" />
                  <span class="text-xs text-gray-500">Rango: 1 – 247</span>
                </div>
              </div>
              <div>
                <label class="label">Ubicacion</label>
                <input v-model="panel.ubicacion" class="input-field" placeholder="Tablero principal sala A"
                  :disabled="panel.saving || panel.deleting"
                  @keydown.enter="saveMeter" />
              </div>

              <!-- Configuracion nube por medidor -->
              <div class="border-t border-slate-700/60 pt-4 space-y-4">
                <p class="text-[11px] font-semibold text-gray-400 uppercase tracking-wide flex items-center gap-1.5">
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"/>
                  </svg>
                  Nube
                </p>
                <div>
                  <label class="label">
                    Nombre en la nube
                    <span class="ml-1 text-gray-600 font-normal normal-case text-[10px]">(opcional)</span>
                  </label>
                  <input v-model="panel.cloudName" class="input-field"
                    placeholder="Igual al Meter ID si no se define"
                    :disabled="panel.saving || panel.deleting"
                    @keydown.enter="saveMeter" />
                  <p class="mt-1 text-[10px] text-gray-500">Identificador enviado al cloud para este medidor.</p>
                </div>
                <div>
                  <label class="label">
                    Token
                    <span class="ml-1 text-red-500 text-[10px]">*</span>
                  </label>
                  <input v-model="panel.cloudToken" type="password"
                    :class="['input-field', panel.error && !panel.cloudToken?.trim() ? 'border-red-500/60 ring-1 ring-red-500/40' : '']"
                    placeholder="Token de acceso a la nube"
                    :disabled="panel.saving || panel.deleting"
                    autocomplete="off"
                    @keydown.enter="saveMeter" />
                  <p class="mt-1 text-[10px] text-gray-500">Credencial que autentica este medidor en la nube.</p>
                </div>
              </div>
            </div>

            <p v-if="panel.error" class="text-xs text-red-400 px-0.5">{{ panel.error }}</p>

            <!-- Acciones -->
            <div class="mt-auto space-y-2 pt-2">
              <button @click="saveMeter" :disabled="panel.saving || panel.deleting"
                class="btn-primary w-full flex items-center justify-center gap-2 disabled:opacity-40">
                <svg v-if="panel.saving" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                </svg>
                {{ panel.saving
                  ? (panel.mode === 'add' ? 'Agregando...' : 'Guardando...')
                  : (panel.mode === 'add' ? 'Agregar medidor' : 'Guardar cambios') }}
              </button>

              <template v-if="panel.mode === 'edit'">
                <button v-if="!panel.confirmDelete"
                  @click="panel.confirmDelete = true"
                  :disabled="panel.saving || panel.deleting"
                  class="w-full px-4 py-2 rounded-lg text-xs font-medium text-red-400
                         border border-red-900/40 hover:bg-red-900/20 transition-colors disabled:opacity-40">
                  Eliminar medidor
                </button>
                <div v-else class="flex gap-2">
                  <button @click="panel.confirmDelete = false"
                    class="flex-1 px-3 py-2 rounded-lg text-xs font-medium text-gray-400
                           border border-slate-700 hover:bg-slate-800 transition-colors">
                    Cancelar
                  </button>
                  <button @click="doDelete" :disabled="panel.deleting"
                    class="flex-1 px-3 py-2 rounded-lg text-xs font-medium text-white
                           bg-red-700 hover:bg-red-600 transition-colors disabled:opacity-40 flex items-center justify-center gap-1.5">
                    <svg v-if="panel.deleting" class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                    </svg>
                    {{ panel.deleting ? '...' : 'Confirmar eliminacion' }}
                  </button>
                </div>
              </template>
            </div>

          </div>
        </aside>
      </div>
    </Teleport>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import MeterSvg from '../components/MeterSvg.vue'
import { getConfig, saveConfig, getMeters, createMeter, updateMeter, deleteMeter, scanMeters } from '../services/api.js'

const form = ref({
  device_id: '',
  cloud_url: '',
  energy_api_key: '',
  send_interval_s: '',
  batch_size: '',
  config_reload_s: '',
})

const meters        = ref([])
const metersLoading = ref(true)
const saving        = ref(false)
const saveError     = ref('')
const savedAt       = ref('')
const panel         = ref(null)
const scanning      = ref(false)
const scanResult    = ref(null)
const scanError     = ref('')

async function saveDeviceConfig() {
  saving.value   = true
  saveError.value = ''
  try {
    await saveConfig({ ...form.value })
    savedAt.value = new Date().toLocaleTimeString('es', { timeStyle: 'short' })
  } catch (e) {
    saveError.value = e.response?.data?.error ?? 'Error al guardar'
  } finally {
    saving.value = false
  }
}

function openAddPanel() {
  panel.value = { mode: 'add', id: null, meterId: '', unitId: 1, ubicacion: '',
                  cloudName: '', cloudToken: '',
                  saving: false, deleting: false, confirmDelete: false, error: '' }
}

function openEditPanel(m) {
  panel.value = { mode: 'edit', id: m.id, meterId: m.meter_id, unitId: m.unit_id, ubicacion: m.ubicacion || '',
                  cloudName: m.cloud_name || '', cloudToken: m.cloud_token || '',
                  saving: false, deleting: false, confirmDelete: false, error: '' }
}

function closePanel() {
  panel.value = null
}

async function saveMeter() {
  const p = panel.value
  if (!p) return
  const mid = (p.meterId || '').trim()
  const uid = Number(p.unitId)
  const tok = (p.cloudToken || '').trim()
  if (!mid) { p.error = 'Nombre requerido'; return }
  if (uid < 1 || uid > 247) { p.error = 'Unit ID fuera de rango (1–247)'; return }
  if (!tok) { p.error = 'Token requerido'; return }
  p.saving = true
  p.error  = ''
  try {
    if (p.mode === 'add') {
      await createMeter({ meter_id: mid, unit_id: uid, ubicacion: p.ubicacion || undefined,
                          cloud_name: p.cloudName || undefined, cloud_token: tok })
    } else {
      await updateMeter(p.id, { meter_id: mid, unit_id: uid, ubicacion: p.ubicacion || undefined,
                                cloud_name: p.cloudName || undefined, cloud_token: tok })
    }
    await loadMeters()
    closePanel()
  } catch (e) {
    p.error = e.response?.data?.error ?? 'Error al guardar'
  } finally {
    p.saving = false
  }
}

async function doDelete() {
  const p = panel.value
  if (!p || p.mode !== 'edit') return
  p.deleting = true
  p.error    = ''
  try {
    await deleteMeter(p.id)
    await loadMeters()
    closePanel()
  } catch (e) {
    p.error = e.response?.data?.error ?? 'Error al eliminar'
  } finally {
    p.deleting = false
  }
}

function meterExistsByUnitId(uid) {
  return meters.value.some(m => m.unit_id === uid)
}

async function startScan() {
  scanning.value  = true
  scanResult.value = null
  scanError.value  = ''
  try {
    scanResult.value = await scanMeters(1, 100)
  } catch (e) {
    scanError.value = e.response?.data?.error ?? 'Error al escanear el bus'
  } finally {
    scanning.value = false
  }
}

async function addScannedMeter(uid) {
  scanError.value = ''
  try {
    await createMeter({ meter_id: `medidor-${String(uid).padStart(2, '0')}`, unit_id: uid })
    await loadMeters()
  } catch (e) {
    scanError.value = e.response?.data?.error ?? `Error al agregar Unit ID ${uid}`
  }
}

async function loadMeters() {
  metersLoading.value = true
  try {
    meters.value = await getMeters()
  } catch {
    meters.value = []
  } finally {
    metersLoading.value = false
  }
}

onMounted(async () => {
  try {
    const items = await getConfig()
    const cfg = {}
    for (const { key, value } of items) cfg[key] = value
    form.value.device_id       = cfg.device_id       ?? ''
    form.value.cloud_url       = cfg.cloud_url        ?? ''
    form.value.energy_api_key  = cfg.energy_api_key   ?? ''
    form.value.send_interval_s = cfg.send_interval_s  ?? ''
    form.value.batch_size      = cfg.batch_size        ?? ''
    form.value.config_reload_s = cfg.config_reload_s  ?? ''
  } catch {
    saveError.value = 'No se pudo cargar la configuracion'
  }
  await loadMeters()
})
</script>
