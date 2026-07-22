<template>
  <div class="space-y-5">

    <!-- Encabezado -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Inicio</h1>
        <p class="mt-0.5 text-sm text-gray-400">Monitoreo en tiempo real del sistema edge</p>
      </div>
      <div class="flex items-center gap-2">
        <span v-if="status.device_id"
          class="hidden sm:inline px-2.5 py-1 bg-slate-700 rounded text-xs font-mono text-gray-300">
          {{ status.device_id }}
        </span>
        <span v-if="status.uptime"
          class="hidden sm:inline px-2.5 py-1 bg-slate-700/60 rounded text-xs text-gray-400">
          ↑ {{ formatUptime(status.uptime) }}
        </span>
        <button @click="loadAll" :disabled="loading"
          class="p-1.5 rounded hover:bg-slate-700 text-gray-400 hover:text-white transition-colors"
          title="Actualizar datos">
          <svg class="w-4 h-4" :class="loading ? 'animate-spin' : ''" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- KPI cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">

      <div class="kpi-card">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400 font-medium">Buffer pendiente</span>
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
          </svg>
        </div>
        <p class="text-2xl font-bold font-mono" :class="bufferClass">
          {{ status.buffer_pending ?? '—' }}
        </p>
        <p class="text-[11px] text-gray-500 mt-0.5">
          {{ status.buffer_pending === 0 ? 'Sin pendientes' : 'eventos en cola' }}
        </p>
      </div>

      <div class="kpi-card">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400 font-medium">Conexión nube</span>
          <span class="w-2.5 h-2.5 rounded-full"
            :class="status.cloud_connected ? 'bg-green-400 animate-pulse' : 'bg-red-400'"></span>
        </div>
        <p class="text-base font-bold leading-tight"
          :class="status.cloud_connected ? 'text-green-400' : 'text-red-400'">
          {{ status.cloud_connected ? 'Conectado' : 'Sin conexión' }}
        </p>
        <p class="text-[11px] text-gray-500 mt-0.5">SSE al cloud gateway</p>
      </div>

      <div class="kpi-card">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400 font-medium">Servicios activos</span>
          <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/>
          </svg>
        </div>
        <p class="text-2xl font-bold font-mono text-white">
          {{ servicesOK }}<span class="text-sm font-normal text-gray-500">/{{ serviceList.length }}</span>
        </p>
        <p class="text-[11px] text-gray-500 mt-0.5">servicios respondiendo</p>
      </div>

      <div class="kpi-card">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400 font-medium">Config. versión</span>
          <svg class="w-4 h-4 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
          </svg>
        </div>
        <p class="text-2xl font-bold font-mono text-white">{{ status.config_version ?? '—' }}</p>
        <p class="text-[11px] text-gray-500 mt-0.5">↑ {{ formatUptime(status.uptime) }} uptime</p>
      </div>

    </div>

    <!-- Hardware del Jetson -->
    <div class="card" v-if="sysMetrics || sysLoading">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
          </svg>
          <h2 class="text-sm font-semibold text-white">Hardware del Jetson</h2>
        </div>
        <span v-if="sysMetrics" class="text-[10px] text-gray-600 font-mono">{{ sysTs }}</span>
      </div>

      <div v-if="sysMetrics" class="grid grid-cols-2 lg:grid-cols-4 gap-3">

        <!-- CPU -->
        <div class="bg-slate-900/60 rounded-lg p-3">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-[11px] text-gray-400 uppercase tracking-wider">CPU</span>
            <span class="text-sm font-mono font-bold" :class="hwColor(sysMetrics.cpu_percent)">
              {{ sysMetrics.cpu_percent.toFixed(1) }}%
            </span>
          </div>
          <div class="w-full bg-slate-700 rounded-full h-1.5">
            <div class="h-1.5 rounded-full transition-all duration-500"
              :class="hwBarColor(sysMetrics.cpu_percent)"
              :style="`width: ${Math.min(sysMetrics.cpu_percent, 100).toFixed(1)}%`"></div>
          </div>
        </div>

        <!-- RAM -->
        <div class="bg-slate-900/60 rounded-lg p-3">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-[11px] text-gray-400 uppercase tracking-wider">RAM</span>
            <span class="text-sm font-mono font-bold" :class="hwColor(sysMetrics.ram_percent)">
              {{ sysMetrics.ram_percent.toFixed(1) }}%
            </span>
          </div>
          <div class="w-full bg-slate-700 rounded-full h-1.5">
            <div class="h-1.5 rounded-full transition-all duration-500"
              :class="hwBarColor(sysMetrics.ram_percent)"
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
            <span class="text-sm font-mono font-bold" :class="hwColor(sysMetrics.disk_percent)">
              {{ sysMetrics.disk_percent.toFixed(1) }}%
            </span>
          </div>
          <div class="w-full bg-slate-700 rounded-full h-1.5">
            <div class="h-1.5 rounded-full transition-all duration-500"
              :class="hwBarColor(sysMetrics.disk_percent)"
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
            <span class="text-sm font-mono font-bold text-violet-300">
              {{ sysMetrics.gpu_percent > 0 ? sysMetrics.gpu_percent.toFixed(1) + '%' : '—' }}
            </span>
          </div>
          <div class="w-full bg-slate-700 rounded-full h-1.5">
            <div class="h-1.5 rounded-full transition-all duration-500 bg-violet-500"
              :style="`width: ${Math.min(sysMetrics.gpu_percent, 100).toFixed(1)}%`"></div>
          </div>
          <p class="text-[10px] text-gray-500 mt-1">NVDEC</p>
        </div>

      </div>

      <!-- Temperaturas -->
      <div v-if="sysMetrics?.temps?.length" class="flex flex-wrap gap-1.5 mt-3">
        <div v-for="t in sysMetrics.temps" :key="t.name"
          class="flex items-center gap-1 bg-slate-900/60 rounded px-2 py-1 text-[11px]">
          <span class="text-gray-500">{{ t.name }}</span>
          <span class="font-mono font-semibold" :class="tempCol(t.celsius)">{{ t.celsius.toFixed(0) }}°C</span>
        </div>
      </div>

      <div v-else-if="!sysMetrics" class="text-xs text-gray-600 animate-pulse">Cargando métricas...</div>
    </div>

    <!-- Sincronización cloud → edge -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-blue-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"/>
          </svg>
          <h2 class="text-sm font-semibold text-white">
            Sincronización de configuración
            <span class="text-gray-500 font-normal text-xs ml-1">cloud → edge (filtrado por línea)</span>
          </h2>
        </div>
        <span v-if="syncLastUpdated" class="hidden sm:inline text-[10px] text-gray-600 font-mono">
          {{ syncLastUpdated }}
        </span>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-2">
        <div v-for="res in syncResources" :key="res.type"
          class="flex items-center gap-3 rounded-lg px-3.5 py-2.5 border transition-colors"
          :class="syncCardClass(res.type)">

          <div class="w-7 h-7 rounded-md flex items-center justify-center shrink-0 text-[13px]"
            :class="syncIconBg(res.type)" v-html="res.icon">
          </div>

          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-gray-200 truncate leading-tight">{{ res.label }}</p>
            <p class="text-[10px] mt-0.5 font-medium" :class="syncStatusTextClass(res.type)">
              {{ syncStatusLabel(res.type) }}
            </p>
          </div>

          <div class="text-right shrink-0">
            <p class="text-base font-bold font-mono"
              :class="syncCount(res.type) !== null ? 'text-white' : 'text-gray-600'">
              {{ syncCount(res.type) !== null ? syncCount(res.type) : '—' }}
            </p>
            <p class="text-[10px] text-gray-500 leading-tight">{{ syncTimeAgo(res.type) }}</p>
          </div>
        </div>
      </div>

      <p v-if="!commands.length && !loading"
        class="mt-4 text-xs text-center text-gray-600">
        Sin historial de comandos — el dispositivo no se ha conectado al cloud aún.
      </p>
    </div>

    <!-- Estado servicios + Conexión nube -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">

      <div class="card">
        <h2 class="text-sm font-semibold text-white mb-3">Estado de servicios</h2>
        <div class="space-y-1.5">
          <div v-for="svc in serviceList" :key="svc.name"
            class="flex items-center justify-between py-1.5 px-3 rounded-lg bg-slate-900/50">
            <div class="flex items-center gap-2.5">
              <span class="w-2 h-2 rounded-full shrink-0" :class="{
                'bg-emerald-400': svc.status === 'ok',
                'bg-yellow-400': svc.status === 'degraded',
                'bg-red-400': svc.status === 'error' || svc.status === 'unknown',
                'bg-gray-600': !svc.status,
              }"></span>
              <span class="text-sm text-gray-200">{{ svc.name }}</span>
            </div>
            <span class="text-xs font-medium font-mono" :class="{
              'text-emerald-400': svc.status === 'ok',
              'text-yellow-400': svc.status === 'degraded',
              'text-red-400': svc.status === 'error' || svc.status === 'unknown',
              'text-gray-600': !svc.status,
            }">{{ svc.status || '—' }}</span>
          </div>
        </div>
      </div>

      <div class="card">
        <h2 class="text-sm font-semibold text-white mb-3">Conexión a nube</h2>

        <div class="flex items-center gap-3 mb-4">
          <span class="w-3 h-3 rounded-full shrink-0 mt-0.5"
            :class="status.cloud_connected ? 'bg-green-400 animate-pulse' : 'bg-red-400'"></span>
          <div>
            <p class="text-sm font-semibold"
              :class="status.cloud_connected ? 'text-green-400' : 'text-red-400'">
              {{ status.cloud_connected ? 'Sincronizando con la nube' : 'Sin conexión a la nube' }}
            </p>
            <p class="text-xs text-gray-500 mt-0.5">
              {{ status.cloud_connected
                ? `${status.buffer_pending ?? 0} eventos pendientes de envío`
                : 'El Enviador no alcanza el cloud gateway' }}
            </p>
          </div>
        </div>

        <div class="space-y-2 text-xs border-t border-slate-700 pt-3">
          <div class="flex justify-between items-center">
            <span class="text-gray-400">Enviador</span>
            <span class="font-medium" :class="enviadorStatusClass">
              {{ health.deps?.enviador || '—' }}
            </span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-gray-400">Eventos en buffer</span>
            <span class="font-mono text-white">{{ status.buffer_pending ?? '—' }}</span>
          </div>
          <div v-if="enviadorHealth?.cloud_last_ok" class="flex justify-between items-center">
            <span class="text-gray-400">Último envío OK</span>
            <span class="font-mono text-white">{{ formatTime(enviadorHealth.cloud_last_ok) }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-gray-400">Canal SSE (config sync)</span>
            <span class="font-mono text-sm" :class="status.cloud_connected ? 'text-blue-300' : 'text-red-400'">
              {{ status.cloud_connected ? '↑ activo' : '✗ desconectado' }}
            </span>
          </div>
        </div>

        <transition name="fade">
          <div v-if="status.recent_errors?.length" class="mt-3 border-t border-slate-700 pt-3">
            <p class="text-[10px] text-red-400 font-medium mb-1.5">Errores recientes</p>
            <div v-for="e in status.recent_errors.slice(0, 3)" :key="e"
              class="text-[10px] font-mono text-red-300 bg-red-950/30 rounded px-2 py-1 mb-1 truncate">
              {{ e }}
            </div>
          </div>
        </transition>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { gatewayService } from '../services/api'

const loading         = ref(false)
const status          = ref({})
const health          = ref({})
const commands        = ref([])
const syncCounts      = ref({ categorias: null, productos: null, turnos: null })
const enviadorHealth  = ref(null)
const syncLastUpdated = ref('')
const sysMetrics      = ref(null)
const sysLoading      = ref(true)
const sysTs           = ref('')

// ── Recursos de sincronización cloud → edge
const svgPath = (d) =>
  `<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="${d}"/></svg>`

const syncResources = [
  { type: 'SYNC_CATALOG',                  label: 'Catálogo de paradas',
    icon: svgPath('M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M-.01 4h.01') },
  { type: 'SYNC_PRODUCTOS',                label: 'Productos',
    icon: svgPath('M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4') },
  { type: 'SYNC_TURNOS',                   label: 'Turnos',
    icon: svgPath('M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z') },
  { type: 'SYNC_VARIABLES',               label: 'Variables',
    icon: svgPath('M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z') },
  { type: 'SYNC_LINEA_PRODUCTO_VARS',      label: 'Vars. Línea / Producto',
    icon: svgPath('M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4') },
  { type: 'SYNC_PRODUCTO_CARACTERISTICAS', label: 'Características de producto',
    icon: svgPath('M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z') },
]

// ── Comando más reciente por tipo
const latestByType = computed(() => {
  const map = {}
  for (const cmd of commands.value) {
    const prev = map[cmd.command_type]
    if (!prev || new Date(cmd.issued_at) > new Date(prev.issued_at)) {
      map[cmd.command_type] = cmd
    }
  }
  return map
})

const syncCount = (type) => {
  const counts = {
    SYNC_CATALOG:   syncCounts.value.categorias,
    SYNC_PRODUCTOS: syncCounts.value.productos,
    SYNC_TURNOS:    syncCounts.value.turnos,
  }
  return counts[type] ?? null
}

const syncStatusLabel = (type) => {
  const cmd = latestByType.value[type]
  if (!cmd)                    return 'Nunca sincronizado'
  if (cmd.status === 'APPLIED')  return 'Aplicado ✓'
  if (cmd.status === 'RECEIVED') return 'Procesando...'
  if (cmd.status === 'FAILED')   return 'Error en sync'
  return cmd.status
}

const syncCardClass = (type) => {
  const cmd = latestByType.value[type]
  if (!cmd)                    return 'bg-slate-800/40 border-slate-700/50'
  if (cmd.status === 'APPLIED') return 'bg-emerald-950/20 border-emerald-800/40'
  if (cmd.status === 'FAILED')  return 'bg-red-950/20 border-red-800/40'
  return 'bg-blue-950/20 border-blue-800/40'
}

const syncIconBg = (type) => {
  const cmd = latestByType.value[type]
  if (!cmd)                    return 'bg-slate-700/50 text-gray-500'
  if (cmd.status === 'APPLIED') return 'bg-emerald-900/50 text-emerald-400'
  if (cmd.status === 'FAILED')  return 'bg-red-900/50 text-red-400'
  return 'bg-blue-900/50 text-blue-400'
}

const syncStatusTextClass = (type) => {
  const cmd = latestByType.value[type]
  if (!cmd)                    return 'text-gray-600'
  if (cmd.status === 'APPLIED') return 'text-emerald-500'
  if (cmd.status === 'FAILED')  return 'text-red-400'
  return 'text-blue-400'
}

const syncTimeAgo = (type) => {
  const cmd = latestByType.value[type]
  const ts  = cmd?.applied_at || cmd?.issued_at
  if (!ts) return ''
  return timeAgo(new Date(ts))
}

// ── Servicios
const serviceList = computed(() => {
  const deps = health.value?.deps || {}
  return [
    { name: 'Edge Gateway',    status: health.value?.status },
    { name: 'Vision Detector', status: deps.detector },
    { name: 'Resiliencia',     status: deps.resiliencia },
    { name: 'Config Service',  status: deps.config },
    { name: 'Enviador',        status: deps.enviador },
  ]
})

const servicesOK = computed(() =>
  serviceList.value.filter(s => s.status === 'ok').length
)

const bufferClass = computed(() => {
  const p = status.value.buffer_pending
  if (p > 100) return 'text-red-400'
  if (p > 0)   return 'text-yellow-400'
  return 'text-white'
})

const hwColor = (pct) => {
  if (pct >= 85) return 'text-red-400'
  if (pct >= 60) return 'text-yellow-400'
  return 'text-emerald-400'
}
const hwBarColor = (pct) => {
  if (pct >= 85) return 'bg-red-500'
  if (pct >= 60) return 'bg-yellow-400'
  return 'bg-emerald-500'
}
const tempCol = (c) => c >= 80 ? 'text-red-400' : c >= 60 ? 'text-yellow-400' : 'text-sky-400'

const enviadorStatusClass = computed(() => {
  const s = health.value?.deps?.enviador
  if (s === 'ok') return 'text-emerald-400'
  if (s === 'degraded') return 'text-yellow-400'
  return 'text-red-400'
})

// ── Helpers
const formatUptime = (s) => {
  if (!s) return '--'
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

const formatTime = (ts) => {
  if (!ts) return '--'
  return new Date(ts).toLocaleTimeString('es-MX', {
    hour: '2-digit', minute: '2-digit', second: '2-digit'
  })
}

const timeAgo = (date) => {
  const secs = Math.floor((Date.now() - date) / 1000)
  if (secs < 60)   return `hace ${secs}s`
  if (secs < 3600) return `hace ${Math.floor(secs / 60)}m`
  return `hace ${Math.floor(secs / 3600)}h`
}

// ── Carga de datos
const loadAll = async () => {
  loading.value = true
  try {
    const [st, h, cmds, counts, env] = await Promise.allSettled([
      gatewayService.getStatus(),
      gatewayService.getAggregatedHealth(),
      gatewayService.getRecentCommands(30),
      gatewayService.getSyncCounts(),
      axios.get('/api/enviador/health', { timeout: 4000 }),
    ])
    if (st.status     === 'fulfilled') status.value    = st.value
    if (h.status      === 'fulfilled') health.value    = h.value
    if (cmds.status   === 'fulfilled') commands.value  = cmds.value
    if (counts.status === 'fulfilled') syncCounts.value = counts.value
    if (env.status    === 'fulfilled') enviadorHealth.value = env.value.data
    syncLastUpdated.value = new Date().toLocaleTimeString('es-MX', {
      hour: '2-digit', minute: '2-digit', second: '2-digit'
    })
  } finally {
    loading.value = false
  }
}

const loadSys = async () => {
  try {
    const data = await gatewayService.getSystemMetrics()
    sysMetrics.value = data
    sysTs.value = new Date().toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch { /* no crítico */ } finally {
    sysLoading.value = false
  }
}

let interval = null
let sysInterval = null
onMounted(() => {
  loadAll()
  loadSys()
  interval    = setInterval(loadAll, 8000)
  sysInterval = setInterval(loadSys, 5000)
})
onUnmounted(() => {
  if (interval)    clearInterval(interval)
  if (sysInterval) clearInterval(sysInterval)
})
</script>

<style scoped>
.card     { @apply bg-slate-800 rounded-lg shadow p-5; }
.kpi-card { @apply bg-slate-800 rounded-lg shadow px-4 py-3; }
.fade-enter-active, .fade-leave-active { transition: opacity .25s ease; }
.fade-enter-from, .fade-leave-to       { opacity: 0; }
</style>
