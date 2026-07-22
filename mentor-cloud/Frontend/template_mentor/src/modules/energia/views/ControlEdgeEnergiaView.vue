<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { energyService } from '@/api/services/energy.service'

const authStore = useAuthStore()

const loading = ref(false)
const result = ref('')
const wsStatus = ref('desconectado')
const wsEvents = ref([])

const configForm = ref({
  device_id: '',
  cloud_url: 'https://mentormonitor-ai.com',
  energy_api_key: '',
  send_interval_s: '30',
  batch_size: '50',
  config_reload_s: '60',
  command_poll_s: '3',
  write_api_url: 'http://localhost:8088/write-api'
})

const meterForm = ref({
  device_id: '',
  meter_id: '',
  unit_id: 1,
  ubicacion: ''
})

const meterDeleteForm = ref({
  device_id: '',
  meter_id: ''
})

const mc60Form = ref({
  device_id: '',
  unit_id: 1,
  command: 'set-time',
  paramsText: '{"meter_id":"medidor-01"}'
})

let ws = null
let reconnectTimer = null

function currentToken() {
  return authStore.token || localStorage.getItem('token') || ''
}

function pushEvent(evt) {
  const stamp = new Date().toLocaleTimeString('es-PE')
  wsEvents.value.unshift(`[${stamp}] ${evt}`)
  if (wsEvents.value.length > 40) wsEvents.value = wsEvents.value.slice(0, 40)
}

function connectWs() {
  const token = currentToken()
  if (!token) return
  try {
    wsStatus.value = 'conectando'
    const url = energyService.buildEdgeControlWsUrl(token)
    ws = new WebSocket(url)
    ws.onopen = () => {
      wsStatus.value = 'conectado'
      pushEvent('WebSocket conectado')
    }
    ws.onmessage = (e) => {
      pushEvent(`evento: ${e.data}`)
    }
    ws.onerror = () => {
      wsStatus.value = 'error'
    }
    ws.onclose = async () => {
      wsStatus.value = 'desconectado'
      pushEvent('WebSocket cerrado')
      try {
        await authStore.doRefreshToken()
      } catch {
        // ignora: el siguiente intento usa token actual de localStorage
      }
      reconnectTimer = setTimeout(connectWs, 2500)
    }
  } catch {
    wsStatus.value = 'error'
  }
}

async function enqueueConfig() {
  loading.value = true
  result.value = ''
  try {
    const payload = {
      device_id: configForm.value.device_id,
      values: {
        device_id: configForm.value.device_id,
        cloud_url: configForm.value.cloud_url,
        energy_api_key: configForm.value.energy_api_key,
        send_interval_s: configForm.value.send_interval_s,
        batch_size: configForm.value.batch_size,
        config_reload_s: configForm.value.config_reload_s,
        command_poll_s: configForm.value.command_poll_s,
        write_api_url: configForm.value.write_api_url
      }
    }
    await energyService.enqueueEdgeConfig(payload)
    result.value = 'Configuracion encolada en cloud'
  } catch (e) {
    result.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

async function enqueueMeterUpsert() {
  loading.value = true
  result.value = ''
  try {
    await energyService.enqueueMeterUpsert({
      device_id: meterForm.value.device_id,
      meter_id: meterForm.value.meter_id,
      unit_id: Number(meterForm.value.unit_id),
      ubicacion: meterForm.value.ubicacion || null
    })
    result.value = 'Upsert de medidor encolado'
  } catch (e) {
    result.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

async function enqueueMeterDelete() {
  loading.value = true
  result.value = ''
  try {
    await energyService.enqueueMeterDelete({
      device_id: meterDeleteForm.value.device_id,
      meter_id: meterDeleteForm.value.meter_id
    })
    result.value = 'Eliminacion de medidor encolada'
  } catch (e) {
    result.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

async function enqueueMc60Command() {
  loading.value = true
  result.value = ''
  try {
    let params = {}
    if (mc60Form.value.paramsText?.trim()) {
      params = JSON.parse(mc60Form.value.paramsText)
    }
    await energyService.enqueueMC60Command({
      device_id: mc60Form.value.device_id,
      unit_id: Number(mc60Form.value.unit_id),
      command: mc60Form.value.command,
      params
    })
    result.value = 'Comando MC60 encolado'
  } catch (e) {
    result.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

onMounted(connectWs)

onUnmounted(() => {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (ws) ws.close()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Control Edge Energia (Cloud)</h1>
        <p class="text-sm text-gray-400">Configura Raspberry y medidores desde la nube, con entrega confiable al edge</p>
      </div>
      <div class="text-xs px-3 py-1 rounded-full" :class="wsStatus === 'conectado' ? 'bg-emerald-900 text-emerald-300' : 'bg-amber-900 text-amber-300'">
        WS: {{ wsStatus }}
      </div>
    </div>

    <div v-if="result" class="rounded-lg border border-primary-700 bg-primary-800 px-4 py-3 text-sm text-gray-200">
      {{ result }}
    </div>

    <section class="rounded-xl border border-primary-700 bg-primary-800 p-5 space-y-3">
      <h2 class="text-lg font-semibold text-white">1) Configuracion Edge</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <input v-model="configForm.device_id" class="input" placeholder="device_id (ej: rpienergy-gloria-01)" />
        <input v-model="configForm.cloud_url" class="input" placeholder="cloud_url" />
        <input v-model="configForm.energy_api_key" class="input" placeholder="energy_api_key" />
        <input v-model="configForm.send_interval_s" class="input" placeholder="send_interval_s" />
        <input v-model="configForm.batch_size" class="input" placeholder="batch_size" />
        <input v-model="configForm.config_reload_s" class="input" placeholder="config_reload_s" />
        <input v-model="configForm.command_poll_s" class="input" placeholder="command_poll_s" />
        <input v-model="configForm.write_api_url" class="input" placeholder="write_api_url" />
      </div>
      <button class="btn" :disabled="loading" @click="enqueueConfig">Encolar Configuracion</button>
    </section>

    <section class="rounded-xl border border-primary-700 bg-primary-800 p-5 space-y-3">
      <h2 class="text-lg font-semibold text-white">2) Medidores</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <input v-model="meterForm.device_id" class="input" placeholder="device_id" />
        <input v-model="meterForm.meter_id" class="input" placeholder="meter_id" />
        <input v-model.number="meterForm.unit_id" type="number" min="1" max="247" class="input" placeholder="unit_id" />
        <input v-model="meterForm.ubicacion" class="input" placeholder="ubicacion" />
      </div>
      <button class="btn" :disabled="loading" @click="enqueueMeterUpsert">Encolar Upsert Medidor</button>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 pt-3 border-t border-primary-700">
        <input v-model="meterDeleteForm.device_id" class="input" placeholder="device_id" />
        <input v-model="meterDeleteForm.meter_id" class="input" placeholder="meter_id a desactivar" />
      </div>
      <button class="btn btn-danger" :disabled="loading" @click="enqueueMeterDelete">Encolar Delete Medidor</button>
    </section>

    <section class="rounded-xl border border-primary-700 bg-primary-800 p-5 space-y-3">
      <h2 class="text-lg font-semibold text-white">3) Comandos MC60</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <input v-model="mc60Form.device_id" class="input" placeholder="device_id" />
        <input v-model.number="mc60Form.unit_id" type="number" min="1" max="247" class="input" placeholder="unit_id" />
        <select v-model="mc60Form.command" class="input">
          <option value="set-ct">set-ct</option>
          <option value="set-sys">set-sys</option>
          <option value="set-dir">set-dir</option>
          <option value="set-time">set-time</option>
          <option value="reset">reset</option>
          <option value="scan">scan</option>
        </select>
      </div>
      <textarea v-model="mc60Form.paramsText" class="input min-h-[110px]" placeholder='params JSON, ej: {"meter_id":"medidor-01","type":"energy"}' />
      <button class="btn" :disabled="loading" @click="enqueueMc60Command">Encolar Comando MC60</button>
    </section>

    <section class="rounded-xl border border-primary-700 bg-primary-800 p-5">
      <h2 class="text-lg font-semibold text-white mb-3">Eventos WebSocket</h2>
      <div class="bg-primary-900 rounded-lg p-3 h-56 overflow-auto font-mono text-xs text-gray-300 space-y-1">
        <div v-for="(e, i) in wsEvents" :key="i">{{ e }}</div>
        <div v-if="wsEvents.length === 0" class="text-gray-500">Sin eventos aun...</div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.input {
  @apply w-full rounded-lg border border-primary-700 bg-primary-900 px-3 py-2 text-sm text-white;
  @apply focus:outline-none focus:ring-2 focus:ring-secondary-500;
}
.btn {
  @apply inline-flex items-center rounded-lg bg-secondary-600 px-4 py-2 text-sm font-medium text-white hover:bg-secondary-700;
}
.btn-danger {
  @apply bg-rose-700 hover:bg-rose-800;
}
</style>
