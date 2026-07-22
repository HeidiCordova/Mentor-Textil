<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'
import { api } from '@/services/api'
import type { Stop } from '@/types'

/* ── state ── */
const stops    = ref<Stop[]>([])
const loading  = ref(false)
const error    = ref('')
const filter   = ref<'ALL' | 'MICROPARADA' | 'PARADA'>('ALL')
let timer: ReturnType<typeof setInterval> | null = null

/* ── fetch ── */
async function load() {
  loading.value = true
  error.value   = ''
  try {
    stops.value = await api.listStops({ limit: 200 })
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Error al cargar paradas'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 10_000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })

/* ── computed ── */
const filtered = computed(() => {
  const list = stops.value
  if (filter.value === 'MICROPARADA') return list.filter(s => s.stop_type === 'MICROPARADA')
  if (filter.value === 'PARADA')      return list.filter(s => s.stop_type !== 'MICROPARADA')
  return list
})

const totalMicro  = computed(() => stops.value.filter(s => s.stop_type === 'MICROPARADA').length)
const totalParada = computed(() => stops.value.filter(s => s.stop_type !== 'MICROPARADA').length)

const totalDownMs = computed(() =>
  stops.value.reduce((sum, s) => {
    if (s.duration_ms) return sum + s.duration_ms
    if (s.ended_at) return sum + (new Date(s.ended_at).getTime() - new Date(s.started_at).getTime())
    return sum + (Date.now() - new Date(s.started_at).getTime())
  }, 0)
)

const openCount = computed(() => stops.value.filter(s => !s.ended_at).length)

/* ── helpers ── */
function fmtTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString('es', { day: '2-digit', month: 'short' })
}
function duration(s: Stop): string {
  let ms = s.duration_ms
  if (!ms && s.ended_at) ms = new Date(s.ended_at).getTime() - new Date(s.started_at).getTime()
  if (!ms) ms = Date.now() - new Date(s.started_at).getTime()
  if (ms < 60_000) return `${Math.round(ms / 1000)}s`
  if (ms < 3_600_000) return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`
  return `${Math.floor(ms / 3_600_000)}h ${Math.floor((ms % 3_600_000) / 60_000)}m`
}
function fmtMs(ms: number): string {
  if (ms < 60_000) return `${Math.round(ms / 1000)}s`
  if (ms < 3_600_000) return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`
  return `${Math.floor(ms / 3_600_000)}h ${Math.floor((ms % 3_600_000) / 60_000)}m`
}

function typeBadge(t: string): { label: string; cls: string } {
  if (t === 'MICROPARADA')        return { label: 'Micro', cls: 'bg-amber-500/20 text-amber-400 border-amber-500/40' }
  if (t === 'PARADA_NO_ASIGNADA') return { label: 'No Asig.', cls: 'bg-red-500/20 text-red-400 border-red-500/40' }
  if (t === 'PROGRAMADA')         return { label: 'Program.', cls: 'bg-blue-500/20 text-blue-400 border-blue-500/40' }
  if (t === 'MECANICA')           return { label: 'Mecánica', cls: 'bg-orange-500/20 text-orange-400 border-orange-500/40' }
  if (t === 'ELECTRICA')          return { label: 'Eléctrica', cls: 'bg-purple-500/20 text-purple-400 border-purple-500/40' }
  return { label: t.replace(/_/g, ' '), cls: 'bg-slate-500/20 text-slate-400 border-slate-500/40' }
}

function sourceBadge(s: string): string {
  const m: Record<string, string> = { detector: '🤖', operator: '👷', cloud: '☁️', system: '⚙️' }
  return m[s] || s
}
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <div class="px-4 pt-4 pb-3">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-semibold text-edge-100">Historial de Paradas</h2>
        <button
          @click="load"
          :disabled="loading"
          class="flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-medium rounded-lg
                 bg-edge-700/50 text-edge-300 hover:bg-edge-700 transition-colors"
        >
          <SvgIcon name="refresh" :size="14" :class="loading ? 'animate-spin' : ''" />
          Actualizar
        </button>
      </div>

      <!-- Summary cards -->
      <div class="grid grid-cols-4 gap-2.5 mt-3">
        <div class="p-2.5 rounded-lg bg-edge-800 border border-edge-700/50">
          <div class="text-[9px] text-edge-500 uppercase tracking-wider font-semibold">Total</div>
          <div class="text-lg font-bold text-edge-100 mt-0.5">{{ stops.length }}</div>
        </div>
        <div class="p-2.5 rounded-lg bg-amber-950/30 border border-amber-500/30">
          <div class="text-[9px] text-amber-400/80 uppercase tracking-wider font-semibold">Microparadas</div>
          <div class="text-lg font-bold text-amber-400 mt-0.5">{{ totalMicro }}</div>
        </div>
        <div class="p-2.5 rounded-lg bg-red-950/30 border border-red-500/30">
          <div class="text-[9px] text-red-400/80 uppercase tracking-wider font-semibold">Paradas</div>
          <div class="text-lg font-bold text-red-400 mt-0.5">{{ totalParada }}</div>
        </div>
        <div class="p-2.5 rounded-lg bg-edge-800 border border-edge-700/50">
          <div class="text-[9px] text-edge-500 uppercase tracking-wider font-semibold">Tiempo caído</div>
          <div class="text-lg font-bold text-edge-100 mt-0.5">{{ fmtMs(totalDownMs) }}</div>
        </div>
      </div>

      <!-- Filters -->
      <div class="flex items-center gap-2 mt-3">
        <button
          v-for="f in (['ALL', 'MICROPARADA', 'PARADA'] as const)" :key="f"
          @click="filter = f"
          class="px-3 py-1 text-[11px] font-medium rounded-full border transition-colors"
          :class="filter === f
            ? 'bg-production-active/20 text-production-active border-production-active/50'
            : 'bg-edge-800 text-edge-400 border-edge-700/50 hover:text-edge-200'"
        >
          {{ f === 'ALL' ? 'Todas' : f === 'MICROPARADA' ? 'Microparadas' : 'Paradas' }}
        </button>
        <span class="ml-auto text-[10px] text-edge-500">
          {{ filtered.length }} registros · {{ openCount }} abiertas
        </span>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="mx-4 mb-2 p-2.5 rounded-lg bg-red-950/40 border border-red-500/30 text-red-400 text-xs">
      {{ error }}
    </div>

    <!-- Table -->
    <div class="flex-1 overflow-y-auto px-4 pb-4">
      <!-- Headers -->
      <div class="sticky top-0 z-10 grid grid-cols-[72px_90px_1fr_80px_70px_60px_44px] gap-2 px-3 py-2
                  text-[9px] font-semibold uppercase tracking-wider text-edge-500
                  bg-edge-950/95 backdrop-blur-sm border-b border-edge-700/30">
        <span>Fecha</span>
        <span>Hora</span>
        <span>Tipo</span>
        <span>Duración</span>
        <span>Fuente</span>
        <span>Estado</span>
        <span></span>
      </div>

      <!-- Loading skeleton -->
      <div v-if="loading && !stops.length" class="space-y-2 mt-2">
        <div v-for="i in 8" :key="i" class="h-10 rounded-lg bg-edge-800/60 animate-pulse" />
      </div>

      <!-- Empty -->
      <div v-else-if="!filtered.length" class="flex flex-col items-center justify-center py-16 text-edge-500">
        <SvgIcon name="clock" :size="32" class="mb-3 opacity-40" />
        <span class="text-sm">Sin paradas registradas</span>
      </div>

      <!-- Rows -->
      <div v-else class="space-y-1 mt-1">
        <div
          v-for="s in filtered" :key="s.stop_id"
          class="grid grid-cols-[72px_90px_1fr_80px_70px_60px_44px] gap-2 items-center
                 px-3 py-2 rounded-lg bg-edge-800/60 border border-edge-700/30
                 hover:bg-edge-800 transition-colors"
          :class="!s.ended_at ? 'border-l-2 border-l-amber-400' : ''"
        >
          <!-- Fecha -->
          <span class="text-[11px] text-edge-400">{{ fmtDate(s.started_at) }}</span>

          <!-- Hora -->
          <span class="text-[11px] font-mono text-edge-200">{{ fmtTime(s.started_at) }}</span>

          <!-- Tipo -->
          <span
            class="inline-flex items-center w-fit px-2 py-0.5 text-[10px] font-semibold rounded-full border"
            :class="typeBadge(s.stop_type).cls"
          >
            {{ typeBadge(s.stop_type).label }}
          </span>

          <!-- Duración -->
          <span class="text-[11px] font-mono" :class="!s.ended_at ? 'text-amber-400' : 'text-edge-200'">
            {{ duration(s) }}
            <span v-if="!s.ended_at" class="text-[8px] ml-0.5">⏱</span>
          </span>

          <!-- Fuente -->
          <span class="text-[11px] text-edge-400">{{ sourceBadge(s.source) }} {{ s.source }}</span>

          <!-- Estado -->
          <span
            class="text-[10px] font-semibold"
            :class="s.justified ? 'text-green-400' : s.ended_at ? 'text-edge-500' : 'text-amber-400'"
          >
            {{ s.justified ? '✓ Just.' : s.ended_at ? 'Cerrada' : 'Abierta' }}
          </span>

          <!-- Synced -->
          <span class="text-center text-[10px]" :title="s.synced ? 'Sincronizado' : 'Pendiente'">
            {{ s.synced ? '☁️' : '📤' }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
