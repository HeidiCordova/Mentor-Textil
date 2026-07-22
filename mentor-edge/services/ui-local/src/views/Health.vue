<template>
  <div class="space-y-5">

    <!-- Encabezado -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Estado del sistema</h1>
        <p class="mt-0.5 text-sm text-gray-400">Diagnóstico detallado de servicios y sincronización</p>
      </div>
      <button @click="loadAll" :disabled="loading"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-40 transition-colors">
        <svg class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        {{ loading ? 'Verificando...' : 'Actualizar' }}
      </button>
    </div>

    <!-- Resumen global -->
    <div class="card">
      <div class="flex flex-wrap items-center gap-6">
        <div>
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Dispositivo</p>
          <p class="text-sm font-mono text-white font-medium">{{ edgeStatus.device_id || '—' }}</p>
        </div>
        <div>
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Uptime</p>
          <p class="text-sm font-mono text-white">{{ formatUptime(edgeStatus.uptime) }}</p>
        </div>
        <div>
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Config. versión</p>
          <p class="text-sm font-mono text-white">{{ edgeStatus.config_version ?? '—' }}</p>
        </div>
        <div>
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Buffer pendiente</p>
          <p class="text-sm font-mono font-bold" :class="bufferClass">{{ edgeStatus.buffer_pending ?? '—' }}</p>
        </div>
        <div>
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Conexión cloud</p>
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full"
              :class="edgeStatus.cloud_connected ? 'bg-green-400 animate-pulse' : 'bg-red-400'"></span>
            <p class="text-sm font-medium"
              :class="edgeStatus.cloud_connected ? 'text-green-400' : 'text-red-400'">
              {{ edgeStatus.cloud_connected ? 'Conectado' : 'Sin conexión' }}
            </p>
          </div>
        </div>
        <div class="ml-auto text-right hidden sm:block">
          <p class="text-[11px] text-gray-600">Último refresco</p>
          <p class="text-[11px] font-mono text-gray-500">{{ lastRefresh || '—' }}</p>
        </div>
      </div>
    </div>

    <!-- ── Métricas de hardware del Jetson ── -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
          </svg>
          <h2 class="text-sm font-semibold text-white">Hardware del Jetson</h2>
        </div>
        <span v-if="sysMetrics" class="text-[10px] text-gray-600">
          {{ sysMetrics._ts ? new Date(sysMetrics._ts).toLocaleTimeString('es-MX', {hour:'2-digit',minute:'2-digit',second:'2-digit'}) : '' }}
        </span>
      </div>

      <div v-if="sysMetrics" class="space-y-4">
        <!-- Grid de barras: CPU / RAM / Disco / GPU -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">

          <!-- CPU -->
          <div class="bg-slate-900/60 rounded-lg p-3">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-[11px] text-gray-400 uppercase tracking-wider">CPU</span>
              <span class="text-sm font-mono font-bold" :class="metricColor(sysMetrics.cpu_percent)">
                {{ sysMetrics.cpu_percent.toFixed(1) }}%
              </span>
            </div>
            <div class="w-full bg-slate-700 rounded-full h-2">
              <div class="h-2 rounded-full transition-all duration-500"
                :class="metricBarColor(sysMetrics.cpu_percent)"
                :style="`width: ${Math.min(sysMetrics.cpu_percent, 100).toFixed(1)}%`"></div>
            </div>
          </div>

          <!-- RAM -->
          <div class="bg-slate-900/60 rounded-lg p-3">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-[11px] text-gray-400 uppercase tracking-wider">RAM</span>
              <span class="text-sm font-mono font-bold" :class="metricColor(sysMetrics.ram_percent)">
                {{ sysMetrics.ram_percent.toFixed(1) }}%
              </span>
            </div>
            <div class="w-full bg-slate-700 rounded-full h-2">
              <div class="h-2 rounded-full transition-all duration-500"
                :class="metricBarColor(sysMetrics.ram_percent)"
                :style="`width: ${Math.min(sysMetrics.ram_percent, 100).toFixed(1)}%`"></div>
            </div>
            <p class="text-[10px] text-gray-500 mt-1 font-mono">
              {{ sysMetrics.ram_used_mb.toLocaleString() }} / {{ sysMetrics.ram_total_mb.toLocaleString() }} MB
            </p>
          </div>

          <!-- Disco -->
          <div class="bg-slate-900/60 rounded-lg p-3">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-[11px] text-gray-400 uppercase tracking-wider">Disco</span>
              <span class="text-sm font-mono font-bold" :class="metricColor(sysMetrics.disk_percent)">
                {{ sysMetrics.disk_percent.toFixed(1) }}%
              </span>
            </div>
            <div class="w-full bg-slate-700 rounded-full h-2">
              <div class="h-2 rounded-full transition-all duration-500"
                :class="metricBarColor(sysMetrics.disk_percent)"
                :style="`width: ${Math.min(sysMetrics.disk_percent, 100).toFixed(1)}%`"></div>
            </div>
            <p class="text-[10px] text-gray-500 mt-1 font-mono">
              {{ sysMetrics.disk_used_gb.toFixed(1) }} / {{ sysMetrics.disk_total_gb.toFixed(1) }} GB
            </p>
          </div>

          <!-- GPU -->
          <div class="bg-slate-900/60 rounded-lg p-3">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-[11px] text-gray-400 uppercase tracking-wider">GPU</span>
              <span class="text-sm font-mono font-bold" :class="metricColor(sysMetrics.gpu_percent)">
                {{ sysMetrics.gpu_percent > 0 ? sysMetrics.gpu_percent.toFixed(1) + '%' : '—' }}
              </span>
            </div>
            <div class="w-full bg-slate-700 rounded-full h-2">
              <div class="h-2 rounded-full transition-all duration-500 bg-violet-500"
                :style="`width: ${Math.min(sysMetrics.gpu_percent, 100).toFixed(1)}%`"></div>
            </div>
            <p class="text-[10px] text-gray-500 mt-1">Decode H264 en NVDEC</p>
          </div>

        </div>

        <!-- Temperaturas -->
        <div v-if="sysMetrics.temps && sysMetrics.temps.length" class="mt-1">
          <p class="text-[11px] text-gray-500 uppercase tracking-wider mb-2">Temperaturas</p>
          <div class="flex flex-wrap gap-2">
            <div v-for="t in sysMetrics.temps" :key="t.name"
              class="flex items-center gap-1.5 bg-slate-900/60 rounded px-2.5 py-1.5 text-xs">
              <svg class="w-3 h-3 shrink-0" :class="tempColor(t.celsius)" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3"/>
              </svg>
              <span class="text-gray-400">{{ t.name }}</span>
              <span class="font-mono font-semibold" :class="tempColor(t.celsius)">{{ t.celsius.toFixed(0) }}°C</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Estado de carga -->
      <div v-else class="flex items-center gap-2 text-xs text-gray-500 py-2">
        <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Obteniendo métricas del hardware...
      </div>
    </div>

    <!-- Grid de servicios -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <div v-for="svc in services" :key="svc.name"
        class="rounded-lg border px-4 py-3 transition-colors"
        :class="svcBorderClass(svc.status)">
        <div class="flex items-start justify-between mb-2">
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-2.5 rounded-full mt-0.5 shrink-0" :class="dotClass(svc.status)"></span>
            <span class="text-sm font-medium text-gray-100">{{ svc.name }}</span>
          </div>
          <span class="text-[11px] font-mono font-medium px-1.5 py-0.5 rounded"
            :class="badgeClass(svc.status)">
            {{ svc.status || '—' }}
          </span>
        </div>
        <div class="space-y-1 text-xs text-gray-500">
          <div v-if="svc.port" class="flex justify-between">
            <span>Puerto</span><span class="font-mono text-gray-400">:{{ svc.port }}</span>
          </div>
          <div v-if="svc.detail" class="flex justify-between">
            <span>{{ svc.detailLabel || 'Info' }}</span>
            <span class="font-mono text-gray-400 truncate max-w-[140px]">{{ svc.detail }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Historial de comandos SYNC -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-semibold text-white">Historial de sincronización (cloud → edge)</h2>
        <span class="text-[10px] text-gray-600">Últimos {{ commands.length }} comandos</span>
      </div>

      <div v-if="commands.length" class="overflow-x-auto -mx-1">
        <table class="w-full text-xs">
          <thead>
            <tr class="border-b border-slate-700">
              <th class="text-left text-gray-500 font-medium pb-2 pr-4">Tipo</th>
              <th class="text-left text-gray-500 font-medium pb-2 pr-4">Estado</th>
              <th class="text-left text-gray-500 font-medium pb-2 pr-4 hidden sm:table-cell">Emitido</th>
              <th class="text-left text-gray-500 font-medium pb-2">Aplicado</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-700/50">
            <tr v-for="cmd in syncCommands" :key="cmd.command_id" class="hover:bg-slate-700/20">
              <td class="py-2 pr-4">
                <span class="font-mono" :class="cmdTypeClass(cmd.command_type)">
                  {{ cmd.command_type }}
                </span>
              </td>
              <td class="py-2 pr-4">
                <span class="px-1.5 py-0.5 rounded text-[10px] font-medium"
                  :class="cmdStatusBadge(cmd.status)">
                  {{ cmd.status }}
                </span>
              </td>
              <td class="py-2 pr-4 text-gray-500 hidden sm:table-cell">
                {{ formatTs(cmd.issued_at) }}
              </td>
              <td class="py-2 text-gray-400">
                {{ cmd.applied_at ? formatTs(cmd.applied_at) : '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-else class="text-xs text-center text-gray-600 py-4">
        Sin historial de comandos SYNC.
      </p>
    </div>

    <!-- Errores recientes -->
    <div v-if="edgeStatus.recent_errors?.length" class="card border border-red-800/40">
      <h2 class="text-sm font-semibold text-red-400 mb-3">Errores recientes</h2>
      <div v-for="e in edgeStatus.recent_errors" :key="e"
        class="text-xs font-mono text-red-300 bg-red-950/30 rounded px-3 py-1.5 mb-1.5">
        {{ e }}
      </div>
    </div>

    <!-- Configuracion actual -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-sm font-semibold text-white">Configuración actual</h2>
          <p class="text-[11px] text-gray-500 mt-0.5">Snapshot del dispositivo <span class="font-mono text-cyan-400">{{ configDeviceId || '...' }}</span></p>
        </div>
        <a href="/config"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-blue-700 hover:bg-blue-600 text-white transition-colors">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
          </svg>
          Editar configuración
        </a>
      </div>

      <div v-if="currentConfig" class="space-y-4">
        <!-- Camera -->
        <div>
          <p class="text-[11px] text-gray-500 uppercase tracking-wider mb-2">Cámara</p>
          <div class="bg-slate-900 rounded p-3 grid grid-cols-2 sm:grid-cols-3 gap-3 text-xs">
            <div><p class="text-gray-500">URL</p><p class="font-mono text-gray-300 truncate" :title="currentConfig.camera?.url">{{ currentConfig.camera?.url || '(no configurada)' }}</p></div>
            <div><p class="text-gray-500">FPS</p><p class="font-mono text-white">{{ currentConfig.camera?.fps ?? '—' }}</p></div>
            <div><p class="text-gray-500">Modo</p><p class="font-mono text-cyan-300">{{ currentConfig.mode || '—' }}</p></div>
          </div>
        </div>
        <!-- OEE -->
        <div>
          <p class="text-[11px] text-gray-500 uppercase tracking-wider mb-2">OEE</p>
          <div class="bg-slate-900 rounded p-3 grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
            <div><p class="text-gray-500">Línea</p><p class="font-mono text-white">{{ currentConfig.oee?.line_name || '—' }}</p></div>
            <div><p class="text-gray-500">Snapshot</p><p class="font-mono text-white">{{ currentConfig.oee?.snapshot_interval_s ?? '—' }}s</p></div>
            <div><p class="text-gray-500">Micro-paro máx.</p><p class="font-mono text-white">{{ currentConfig.oee?.micro_stop_max_s ?? '—' }}s</p></div>
            <div><p class="text-gray-500">Paro máx.</p><p class="font-mono text-white">{{ currentConfig.oee?.stop_max_s ?? '—' }}s</p></div>
          </div>
        </div>
        <!-- Cloud -->
        <div>
          <p class="text-[11px] text-gray-500 uppercase tracking-wider mb-2">Cloud</p>
          <div class="bg-slate-900 rounded p-3 grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
            <div><p class="text-gray-500">URL</p><p class="font-mono text-gray-300 truncate" :title="currentConfig.cloud?.cloud_url">{{ currentConfig.cloud?.cloud_url || '(no configurada)' }}</p></div>
            <div><p class="text-gray-500">API Key</p><p class="font-mono text-gray-400">{{ currentConfig.cloud?.cloud_api_key ? '••••' + currentConfig.cloud.cloud_api_key.slice(-4) : '(no configurada)' }}</p></div>
            <div><p class="text-gray-500">Intervalo envío</p><p class="font-mono text-white">{{ currentConfig.cloud?.sync_interval_s ? Math.floor(currentConfig.cloud.sync_interval_s/60) + ' min' : '—' }}</p></div>
            <div><p class="text-gray-500">Versión config.</p><p class="font-mono text-white">v{{ currentConfig.config_version ?? '—' }}</p></div>
          </div>
        </div>
        <!-- Tablet -->
        <div v-if="currentConfig.tablet">
          <p class="text-[11px] text-gray-500 uppercase tracking-wider mb-2">Tablet</p>
          <div class="bg-slate-900 rounded p-3 grid grid-cols-3 gap-3 text-xs">
            <div><p class="text-gray-500">Host / IP</p><p class="font-mono text-white">{{ currentConfig.tablet.host || '—' }}</p></div>
            <div><p class="text-gray-500">Puerto App</p><p class="font-mono text-white">{{ currentConfig.tablet.port ?? '—' }}</p></div>
            <div><p class="text-gray-500">Puerto Gateway</p><p class="font-mono text-white">{{ currentConfig.tablet.gatewayPort ?? '—' }}</p></div>
          </div>
        </div>
        <!-- Deteccion -->
        <div>
          <p class="text-[11px] text-gray-500 uppercase tracking-wider mb-2">Detección (umbrales)</p>
          <div class="bg-slate-900 rounded p-3 grid grid-cols-3 sm:grid-cols-6 gap-3 text-xs">
            <div v-for="(val, key) in currentConfig.thresholds" :key="key">
              <p class="text-gray-500">{{ key }}</p>
              <p class="font-mono text-yellow-300">{{ val }}</p>
            </div>
          </div>
        </div>
      </div>
      <p v-else class="text-xs text-center text-gray-600 py-4">Cargando configuración...</p>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { gatewayService, resilienciaService } from '../services/api'

const loading     = ref(false)
const edgeStatus  = ref({})
const health      = ref({})
const bufferData  = ref({})
const commands    = ref([])
const lastRefresh = ref('')
const currentConfig  = ref(null)
const configDeviceId = ref('')
const sysMetrics     = ref(null)

// ── Lista de servicios con detalle enriquecido
const services = computed(() => {
  const deps = health.value?.deps || {}
  const b = bufferData.value
  return [
    {
      name: 'Edge Gateway',
      status: health.value?.status,
      port: 8005,
      detailLabel: 'Uptime',
      detail: formatUptime(health.value?.uptime || edgeStatus.value?.uptime),
    },
    {
      name: 'Vision Detector',
      status: deps.detector,
      port: 8001,
    },
    {
      name: 'Resiliencia',
      status: deps.resiliencia,
      port: 8002,
      detailLabel: 'Buffer',
      detail: b.pending_count != null ? `${b.pending_count} pendientes / ${b.total_count ?? '?'} total` : null,
    },
    {
      name: 'Config Service',
      status: deps.config,
      port: 8004,
      detailLabel: 'Versión cfg',
      detail: edgeStatus.value?.config_version != null ? `v${edgeStatus.value.config_version}` : null,
    },
    {
      name: 'Enviador',
      status: deps.enviador,
      port: 8003,
      detailLabel: 'Cloud',
      detail: deps.enviador === 'ok' ? 'Enviando datos' : 'Sin envio',
    },
  ]
})

// Filtra solo comandos de tipo SYNC
const syncCommands = computed(() =>
  commands.value
    .filter(c => c.command_type?.startsWith('SYNC_') || c.command_type === 'SYNC_CATALOG')
    .slice(0, 15)
)

const bufferClass = computed(() => {
  const p = edgeStatus.value.buffer_pending
  if (p > 100) return 'text-red-400'
  if (p > 0)   return 'text-yellow-400'
  return 'text-white'
})

// ── Helpers de color para métricas de hardware
const metricColor = (pct) => {
  if (pct >= 85) return 'text-red-400'
  if (pct >= 60) return 'text-yellow-400'
  return 'text-emerald-400'
}

const metricBarColor = (pct) => {
  if (pct >= 85) return 'bg-red-500'
  if (pct >= 60) return 'bg-yellow-400'
  return 'bg-emerald-500'
}

const tempColor = (celsius) => {
  if (celsius >= 80) return 'text-red-400'
  if (celsius >= 60) return 'text-yellow-400'
  return 'text-sky-400'
}

// ── Clase helpers
const dotClass = (status) => ({
  'bg-emerald-400': status === 'ok',
  'bg-yellow-400':  status === 'degraded',
  'bg-red-400':     status === 'error' || status === 'unknown',
  'bg-gray-600':    !status,
})

const svcBorderClass = (status) => {
  if (status === 'ok')       return 'bg-slate-800 border-slate-700'
  if (status === 'degraded') return 'bg-yellow-950/10 border-yellow-700/40'
  if (status === 'error' || status === 'unknown') return 'bg-red-950/10 border-red-800/40'
  return 'bg-slate-800 border-slate-700/50'
}

const badgeClass = (status) => {
  if (status === 'ok')       return 'bg-emerald-900/50 text-emerald-400'
  if (status === 'degraded') return 'bg-yellow-900/50 text-yellow-400'
  return 'bg-red-900/40 text-red-400'
}

const cmdTypeClass = (type) => {
  if (type?.startsWith('SYNC_')) return 'text-blue-300'
  return 'text-gray-300'
}

const cmdStatusBadge = (status) => {
  if (status === 'APPLIED')  return 'bg-emerald-900/50 text-emerald-400'
  if (status === 'RECEIVED') return 'bg-blue-900/50 text-blue-400'
  return 'bg-red-900/40 text-red-400'
}

// ── Formatters
const formatUptime = (s) => {
  if (!s) return '—'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) return `${h}h ${m}m ${sec}s`
  if (m > 0) return `${m}m ${sec}s`
  return `${sec}s`
}

const formatTs = (ts) => {
  if (!ts) return '—'
  const d = new Date(ts)
  return d.toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// ── Carga de datos
const loadAll = async () => {
  loading.value = true
  try {
    const [st, h, cmds, buf] = await Promise.allSettled([
      gatewayService.getStatus(),
      gatewayService.getAggregatedHealth(),
      gatewayService.getRecentCommands(20),
      resilienciaService.getBufferHealth(),
    ])
    if (st.status  === 'fulfilled') edgeStatus.value = st.value
    if (h.status   === 'fulfilled') health.value     = h.value
    if (cmds.status === 'fulfilled') commands.value  = cmds.value
    if (buf.status === 'fulfilled') bufferData.value = buf.value

    // Cargar configuración actual
    try {
      const { devices } = (await axios.get('/api/config/config/devices')).data
      const devId = devices?.[0] || 'jetson-default'
      configDeviceId.value = devId
      const cfg = (await axios.get('/api/config/config', { params: { device_id: devId } })).data
      currentConfig.value = cfg
    } catch { /* no critico */ }

    lastRefresh.value = new Date().toLocaleTimeString('es-MX', {
      hour: '2-digit', minute: '2-digit', second: '2-digit'
    })
  } finally {
    loading.value = false
  }
}

const loadSystemMetrics = async () => {
  try {
    const data = await gatewayService.getSystemMetrics()
    data._ts = Date.now()
    sysMetrics.value = data
  } catch { /* no crítico */ }
}

let interval = null
let sysInterval = null
onMounted(() => {
  loadAll()
  loadSystemMetrics()
  interval    = setInterval(loadAll, 10000)
  sysInterval = setInterval(loadSystemMetrics, 5000)
})
onUnmounted(() => {
  if (interval)    clearInterval(interval)
  if (sysInterval) clearInterval(sysInterval)
})
</script>

<style scoped>
.card { @apply bg-slate-800 rounded-lg shadow p-5; }
</style>
