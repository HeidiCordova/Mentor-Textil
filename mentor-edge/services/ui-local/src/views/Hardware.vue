<template>
  <div class="space-y-5">

    <!-- Encabezado -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Hardware del Jetson</h1>
        <p class="mt-0.5 text-sm text-gray-400">Recursos del sistema en tiempo real</p>
      </div>
      <span class="text-[11px] text-gray-600 font-mono">{{ ts }}</span>
    </div>

    <!-- Barras principales -->
    <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">

      <!-- CPU -->
      <div class="card">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-emerald-900/40 border border-emerald-700/40 flex items-center justify-center">
              <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
              </svg>
            </div>
            <span class="text-sm font-semibold text-white">CPU</span>
          </div>
          <span v-if="m" class="text-2xl font-bold font-mono" :class="col(m.cpu_percent)">
            {{ m.cpu_percent.toFixed(1) }}%
          </span>
        </div>
        <div class="w-full bg-slate-700 rounded-full h-3">
          <div v-if="m" class="h-3 rounded-full transition-all duration-500"
            :class="bar(m.cpu_percent)"
            :style="`width: ${Math.min(m.cpu_percent, 100).toFixed(1)}%`"></div>
        </div>
        <p class="text-[11px] text-gray-500 mt-2">6 núcleos ARM Cortex-A78AE</p>
      </div>

      <!-- RAM -->
      <div class="card">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-sky-900/40 border border-sky-700/40 flex items-center justify-center">
              <svg class="w-4 h-4 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
              </svg>
            </div>
            <span class="text-sm font-semibold text-white">RAM</span>
          </div>
          <span v-if="m" class="text-2xl font-bold font-mono" :class="col(m.ram_percent)">
            {{ m.ram_percent.toFixed(1) }}%
          </span>
        </div>
        <div class="w-full bg-slate-700 rounded-full h-3">
          <div v-if="m" class="h-3 rounded-full transition-all duration-500"
            :class="bar(m.ram_percent)"
            :style="`width: ${Math.min(m.ram_percent, 100).toFixed(1)}%`"></div>
        </div>
        <p v-if="m" class="text-[11px] text-gray-500 mt-2 font-mono">
          {{ m.ram_used_mb.toLocaleString() }} / {{ m.ram_total_mb.toLocaleString() }} MB
        </p>
      </div>

      <!-- Disco -->
      <div class="card">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-amber-900/40 border border-amber-700/40 flex items-center justify-center">
              <svg class="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"/>
              </svg>
            </div>
            <span class="text-sm font-semibold text-white">Disco</span>
          </div>
          <span v-if="m" class="text-2xl font-bold font-mono" :class="col(m.disk_percent)">
            {{ m.disk_percent.toFixed(1) }}%
          </span>
        </div>
        <div class="w-full bg-slate-700 rounded-full h-3">
          <div v-if="m" class="h-3 rounded-full transition-all duration-500"
            :class="bar(m.disk_percent)"
            :style="`width: ${Math.min(m.disk_percent, 100).toFixed(1)}%`"></div>
        </div>
        <p v-if="m" class="text-[11px] text-gray-500 mt-2 font-mono">
          {{ m.disk_used_gb.toFixed(1) }} / {{ m.disk_total_gb.toFixed(1) }} GB
        </p>
      </div>

      <!-- GPU -->
      <div class="card">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-violet-900/40 border border-violet-700/40 flex items-center justify-center">
              <svg class="w-4 h-4 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13 10V3L4 14h7v7l9-11h-7z"/>
              </svg>
            </div>
            <span class="text-sm font-semibold text-white">GPU</span>
          </div>
          <span v-if="m" class="text-2xl font-bold font-mono text-violet-300">
            {{ m.gpu_percent > 0 ? m.gpu_percent.toFixed(1) + '%' : '—' }}
          </span>
        </div>
        <!-- Barra de carga -->
        <p class="text-[10px] text-gray-500 mb-1 uppercase tracking-wide">Carga</p>
        <div class="w-full bg-slate-700 rounded-full h-3 mb-3">
          <div v-if="m" class="h-3 rounded-full transition-all duration-500 bg-violet-500"
            :style="`width: ${Math.min(m.gpu_percent, 100).toFixed(1)}%`"></div>
        </div>
        <!-- Frecuencia -->
        <template v-if="m && m.gpu_freq_mhz > 0">
          <div class="flex items-center justify-between mb-1">
            <p class="text-[10px] text-gray-500 uppercase tracking-wide">Frecuencia</p>
            <p class="text-[11px] font-mono text-violet-300">
              {{ m.gpu_freq_mhz }} <span class="text-gray-500">/ {{ m.gpu_max_freq_mhz }} MHz</span>
            </p>
          </div>
          <div class="w-full bg-slate-700 rounded-full h-2 mb-3">
            <div class="h-2 rounded-full transition-all duration-500 bg-violet-400/60"
              :style="`width: ${m.gpu_max_freq_mhz > 0 ? ((m.gpu_freq_mhz / m.gpu_max_freq_mhz) * 100).toFixed(1) : 0}%`"></div>
          </div>
        </template>
        <p class="text-[11px] text-gray-500">Ampere · NVDEC · CUDA 12.6</p>
      </div>

    </div>

    <!-- Temperaturas -->
    <div class="card" v-if="m?.temps?.length">
      <h2 class="text-sm font-semibold text-white mb-4">Temperaturas</h2>
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-3">
        <div v-for="t in m.temps" :key="t.name"
          class="bg-slate-900/60 rounded-lg p-3 flex flex-col items-center gap-1">
          <span class="text-[11px] text-gray-500 uppercase tracking-wider text-center leading-tight">{{ t.name }}</span>
          <span class="text-xl font-bold font-mono" :class="tempCol(t.celsius)">{{ t.celsius.toFixed(0) }}°</span>
          <div class="w-full bg-slate-700 rounded-full h-1">
            <div class="h-1 rounded-full transition-all duration-500"
              :class="tempBar(t.celsius)"
              :style="`width: ${Math.min((t.celsius / 100) * 100, 100).toFixed(0)}%`"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- Sin datos -->
    <div v-if="!m && !loading" class="card text-center text-sm text-gray-500 py-8">
      No se pudo obtener métricas del hardware. Verificá que el edge-gateway esté corriendo.
    </div>
    <div v-if="loading && !m" class="card text-center text-xs text-gray-600 animate-pulse py-6">
      Obteniendo métricas...
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { gatewayService } from '../services/api'

const m       = ref(null)
const ts      = ref('')
const loading = ref(true)

const col = (pct) => pct >= 85 ? 'text-red-400' : pct >= 60 ? 'text-yellow-400' : 'text-emerald-400'
const bar = (pct) => pct >= 85 ? 'bg-red-500' : pct >= 60 ? 'bg-yellow-400' : 'bg-emerald-500'
const tempCol = (c) => c >= 80 ? 'text-red-400' : c >= 60 ? 'text-yellow-400' : 'text-sky-400'
const tempBar = (c) => c >= 80 ? 'bg-red-500' : c >= 60 ? 'bg-yellow-400' : 'bg-sky-400'

const load = async () => {
  try {
    const data = await gatewayService.getSystemMetrics()
    m.value = data
    ts.value = new Date().toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch { /* no crítico */ } finally {
    loading.value = false
  }
}

let interval = null
onMounted(() => { load(); interval = setInterval(load, 5000) })
onUnmounted(() => { if (interval) clearInterval(interval) })
</script>

<style scoped>
.card { @apply bg-slate-800 rounded-lg shadow p-5; }
</style>
