<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import SvgIcon from '@/components/SvgIcon.vue'
import { useConnectionStore } from '@/stores/connection'
import type { DeviceEntry } from '@/types'
import { discoverDefaults, loadSavedDevices, saveDevice, removeDevice, probeDevice } from '@/services/discovery'

interface DeviceProbe extends DeviceEntry {
  reachable: boolean
}

const router = useRouter()
const connection = useConnectionStore()

const devices = ref<DeviceProbe[]>([])
const scanning = ref(false)
const connectingUrl = ref('')
const manualUrl = ref('')
const manualProbing = ref(false)
const cloudUrl = ref(localStorage.getItem('cloud_url') || `http://${window.location.hostname}:8888`)
const cloudProbing = ref(false)
const cloudError = ref('')

async function scan(): Promise<void> {
  scanning.value = true
  const discovered = await discoverDefaults()
  // Los discovered tienen prioridad — si el mismo host aparece con otro puerto, reemplazar
  const savedRaw = loadSavedDevices()
  const discoveredHosts = new Set(discovered.map((d) => new URL(d.url).hostname))
  // Filtrar guardados que sean del mismo host que algún discovered (evita duplicados de puerto)
  const saved = savedRaw.filter((d) => {
    try { return !discoveredHosts.has(new URL(d.url).hostname) } catch { return false }
  })
  const all = [...saved.map((d) => ({ ...d, reachable: false })), ...discovered.map((d) => ({ ...d, reachable: true }))]
  devices.value = all
  // Sondear los guardados que no fueron redescubiertos
  for (const d of devices.value) {
    if (!d.reachable) d.reachable = await probeDevice(d.url)
  }
  scanning.value = false
}

async function addManual(): Promise<void> {
  const url = manualUrl.value.trim()
  if (!url) return
  manualProbing.value = true
  const reachable = await probeDevice(url)
  const entry: DeviceProbe = {
    id: crypto.randomUUID(),
    name: url,
    url,
    lastSeen: reachable ? Date.now() : 0,
    reachable
  }
  devices.value.push(entry)
  if (reachable) saveDevice(entry)
  manualUrl.value = ''
  manualProbing.value = false
}

async function connectCloud(): Promise<void> {
  const url = cloudUrl.value.trim().replace(/\/$/, '')
  if (!url) return
  cloudProbing.value = true
  cloudError.value = ''
  connection.connectToCloud(url)
  const ok = await connection.probeCloud()
  if (ok) {
    router.push('/login')
  } else {
    cloudError.value = 'No se pudo conectar al servidor'
  }
  cloudProbing.value = false
}

function connectTo(device: DeviceProbe): void {
  if (connectingUrl.value) return   // evitar doble click
  connectingUrl.value = device.url
  saveDevice(device)
  connection.connectToEdge(device.url)
  router.push('/login').then((failure) => {
    if (failure) {
      window.location.href = '/login'
    }
  }).catch(() => {
    window.location.href = '/login'
  })
}

function remove(device: DeviceProbe): void {
  removeDevice(device.url)
  devices.value = devices.value.filter((d) => d.url !== device.url)
}

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const edgeParam = params.get('edge')
  if (edgeParam) {
    const reachable = await probeDevice(edgeParam)
    if (reachable) {
      const entry: DeviceEntry = { id: crypto.randomUUID(), name: 'Jetson Edge', url: edgeParam, lastSeen: Date.now() }
      saveDevice(entry)
      connection.connectToEdge(edgeParam)
      router.push('/login')
      return
    }
  }
  scan()
})
</script>

<template>
  <div class="flex items-center justify-center min-h-full p-6 bg-edge-950">
    <div class="w-full max-w-lg">
      <div class="text-center mb-8">
        <div class="flex items-center justify-center w-16 h-16 mx-auto mb-4 rounded-2xl bg-edge-800 border border-edge-700/50">
          <SvgIcon name="monitor" :size="32" class="text-production-active" />
        </div>
        <h1 class="text-xl font-bold text-edge-100">Mentor Tablet</h1>
        <p class="text-sm text-edge-400 mt-1">Seleccione un dispositivo o servidor cloud</p>
      </div>

      <div class="mb-6 p-4 rounded-xl bg-edge-900/50 border border-edge-700/30">
        <h2 class="text-xs font-semibold text-edge-300 uppercase tracking-wider mb-3">Servidor Cloud</h2>
        <div class="flex gap-2">
          <input
            v-model="cloudUrl"
            type="text"
            placeholder="http://192.168.100.31:8888"
            class="flex-1 px-3 py-2 text-sm bg-edge-800 border border-edge-700/50 rounded-lg text-edge-200 placeholder-edge-600 focus:border-production-active focus:outline-none font-mono"
            @keydown.enter="connectCloud"
          />
          <button
            class="px-4 py-2 text-sm font-medium rounded-lg bg-production-active text-white hover:bg-blue-600 transition-colors disabled:opacity-50"
            :disabled="!cloudUrl.trim() || cloudProbing"
            @click="connectCloud"
          >
            <SvgIcon v-if="cloudProbing" name="refresh" :size="16" class="animate-spin" />
            <span v-else>Conectar</span>
          </button>
        </div>
        <p v-if="cloudError" class="text-xs text-red-400 mt-2">{{ cloudError }}</p>
      </div>

      <div class="mb-3">
        <h2 class="text-xs font-semibold text-edge-300 uppercase tracking-wider mb-3">Dispositivos Edge</h2>
      </div>

      <div class="space-y-3 mb-6">
        <div
          v-for="device in devices"
          :key="device.url"
          class="flex items-center justify-between p-3 rounded-lg border transition-colors"
          :class="[
            device.reachable
              ? 'bg-edge-800 border-edge-700/50 hover:border-production-active/50 cursor-pointer'
              : 'bg-edge-800 border-edge-700/50 hover:border-production-active/50 cursor-pointer opacity-70'
          ]"
          @click="connectTo(device)"
          :style="connectingUrl === device.url ? 'opacity:0.6;pointer-events:none' : ''"
        >
          <div class="flex items-center gap-3">
            <div class="flex items-center justify-center w-10 h-10 rounded-lg"
              :class="device.reachable ? 'bg-green-500/10' : 'bg-edge-800'">
              <SvgIcon
                :name="device.reachable ? 'wifi' : 'wifi-off'"
                :size="18"
                :class="device.reachable ? 'text-green-400' : 'text-edge-600'"
              />
            </div>
            <div>
              <div class="text-sm font-medium" :class="device.reachable ? 'text-edge-100' : 'text-edge-500'">
                {{ connectingUrl === device.url ? 'Conectando...' : (device.name || 'Dispositivo') }}
              </div>
              <div class="text-xs text-edge-500">{{ device.url }}</div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span
              class="text-[10px] px-2 py-0.5 rounded-full"
              :class="device.reachable ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'"
            >{{ device.reachable ? 'Online' : 'Offline' }}</span>
            <button
              class="p-1 rounded hover:bg-edge-700/50 text-edge-500 hover:text-red-400 transition-colors"
              @click.stop="remove(device)"
            ><SvgIcon name="close" :size="14" /></button>
          </div>
        </div>

        <div v-if="devices.length === 0 && !scanning" class="text-center py-6 text-edge-500 text-sm">
          No se encontraron dispositivos edge
        </div>

        <div v-if="scanning" class="flex items-center justify-center py-4 gap-2 text-edge-400 text-sm">
          <SvgIcon name="refresh" :size="16" class="animate-spin" />
          <span>Buscando dispositivos...</span>
        </div>
      </div>

      <div class="flex gap-2 mb-4">
        <input
          v-model="manualUrl"
          type="text"
          placeholder="http://192.168.1.100:8005"
          class="flex-1 px-3 py-2 text-sm bg-edge-900 border border-edge-700/50 rounded-lg text-edge-200 placeholder-edge-600 focus:border-production-active focus:outline-none"
          @keydown.enter="addManual"
        />
        <button
          class="px-4 py-2 text-sm font-medium rounded-lg bg-production-active text-white hover:bg-blue-600 transition-colors disabled:opacity-50"
          :disabled="!manualUrl.trim() || manualProbing"
          @click="addManual"
        >
          <SvgIcon v-if="manualProbing" name="refresh" :size="16" class="animate-spin" />
          <span v-else>Agregar</span>
        </button>
      </div>

      <button
        class="w-full py-2.5 text-sm font-medium rounded-lg border border-edge-700/50 text-edge-300 hover:text-edge-100 hover:border-edge-600 transition-colors disabled:opacity-50"
        :disabled="scanning"
        @click="scan"
      >
        <span class="flex items-center justify-center gap-2">
          <SvgIcon name="refresh" :size="16" :class="scanning ? 'animate-spin' : ''" />
          Buscar de nuevo
        </span>
      </button>
    </div>
  </div>
</template>
