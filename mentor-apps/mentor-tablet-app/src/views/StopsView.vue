<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'
import StopForm from '@/components/StopForm.vue'
import { useStopsStore } from '@/stores/stops'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { useConnectionStore } from '@/stores/connection'
import type { Stop, CreateStopRequest, JustifyStopRequest } from '@/types'

const stopsStore = useStopsStore()
const pl = usePlantasLineasStore()
const connection = useConnectionStore()
const justifyingStop = ref<Stop | null>(null)
const filter = ref<'all' | 'open' | 'unjustified'>('all')

function scopeParams() {
  return {
    linea_id: pl.selectedLineaId ?? undefined,
    planta_id: pl.selectedPlantaId ?? undefined,
    empresa_id: (connection.operator?.empresa_id ?? undefined) as number | undefined
  }
}

const filteredStops = computed(() => {
  switch (filter.value) {
    case 'open':
      return stopsStore.openStops
    case 'unjustified':
      return stopsStore.unjustifiedStops
    default:
      return stopsStore.stops
  }
})

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit' })
}

function formatDuration(ms?: number): string {
  if (!ms) return '--'
  const mins = Math.round(ms / 60000)
  if (mins < 60) return `${mins} min`
  return `${Math.floor(mins / 60)}h ${mins % 60}m`
}

function stopColor(stop: Stop): string {
  if (stop.justified) return 'border-l-stop-justified'
  if (stop.ended_at) return 'border-l-stop-assigned'
  return 'border-l-stop-unassigned'
}

function startJustify(stop: Stop): void {
  justifyingStop.value = stop
}

async function handleJustify(data: CreateStopRequest | JustifyStopRequest): Promise<void> {
  if (!justifyingStop.value) return
  if ('reason' in data && 'category' in data) {
    const justify = data as JustifyStopRequest
    // Inyectar el nombre del operador para trazabilidad
    if (!justify.justified_by) {
      justify.justified_by = connection.operator?.nombre ?? connection.operator?.username ?? 'operator'
    }
    await stopsStore.justifyStop(justifyingStop.value.stop_id, justify)
  }
  justifyingStop.value = null
}

async function handleClose(stop: Stop): Promise<void> {
  await stopsStore.closeStop(stop.stop_id)
}

onMounted(() => {
  const p = scopeParams()
  stopsStore.fetchStops({ ...p, limit: 200 })
  stopsStore.fetchSummary(24, p)
})

watch(() => pl.selectedLineaId, () => {
  const p = scopeParams()
  stopsStore.fetchStops({ ...p, limit: 200 })
  stopsStore.fetchSummary(24, p)
})
</script>

<template>
  <div class="flex flex-col h-full p-4">
    <div class="flex items-center justify-between mb-4 shrink-0">
      <h2 class="text-base font-semibold text-edge-100">Gestion de Paradas</h2>
      <div class="flex items-center gap-1">
        <button
          v-for="f in (['all', 'open', 'unjustified'] as const)"
          :key="f"
          class="px-3 py-1.5 text-xs rounded-md transition-colors"
          :class="[
            filter === f
              ? 'bg-edge-700 text-edge-100'
              : 'text-edge-400 hover:text-edge-200 hover:bg-edge-800'
          ]"
          @click="filter = f"
        >
          {{ f === 'all' ? 'Todas' : f === 'open' ? 'Abiertas' : 'Sin justificar' }}
        </button>
      </div>
    </div>

    <div v-if="stopsStore.summary" class="grid grid-cols-4 gap-3 mb-4 shrink-0">
      <div class="p-3 rounded-lg bg-edge-800 border border-edge-700/50">
        <div class="text-[10px] text-edge-500 uppercase tracking-wider">Total</div>
        <div class="text-lg font-bold text-edge-100 tabular-nums">{{ stopsStore.summary.total_stops }}</div>
      </div>
      <div class="p-3 rounded-lg bg-edge-800 border border-edge-700/50">
        <div class="text-[10px] text-edge-500 uppercase tracking-wider">Tiempo total</div>
        <div class="text-lg font-bold text-edge-100 tabular-nums">{{ formatDuration(stopsStore.summary.total_downtime_ms) }}</div>
      </div>
      <div class="p-3 rounded-lg bg-edge-800 border border-edge-700/50">
        <div class="text-[10px] text-edge-500 uppercase tracking-wider">Sin justificar</div>
        <div class="text-lg font-bold text-edge-100 tabular-nums">{{ stopsStore.summary.unjustified_stops }}</div>
      </div>
      <div class="p-3 rounded-lg bg-edge-800 border border-edge-700/50">
        <div class="text-[10px] text-edge-500 uppercase tracking-wider">Por tipo</div>
        <div class="text-xs text-edge-300 mt-1">
          <div v-for="(count, type) in stopsStore.summary.by_type" :key="type" class="flex justify-between">
            <span class="text-edge-500">{{ type }}</span>
            <span>{{ count }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto space-y-2">
      <div
        v-for="stop in filteredStops"
        :key="stop.stop_id"
        class="flex items-center justify-between p-3 rounded-lg bg-edge-800 border-l-4 border border-edge-700/50 transition-colors hover:bg-edge-750"
        :class="stopColor(stop)"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div class="shrink-0 flex items-center justify-center w-8 h-8 rounded-lg"
            :class="stop.justified ? 'bg-green-500/10' : stop.ended_at ? 'bg-red-500/10' : 'bg-yellow-500/10'">
            <SvgIcon
              :name="stop.justified ? 'check' : stop.ended_at ? 'alert' : 'clock'"
              :size="16"
              :class="stop.justified ? 'text-green-400' : stop.ended_at ? 'text-red-400' : 'text-yellow-400'"
            />
          </div>
          <div class="min-w-0">
            <div class="text-sm text-edge-100 font-medium">{{ stop.stop_type.replace(/_/g, ' ') }}</div>
            <div class="flex items-center gap-2 text-xs text-edge-500">
              <span>{{ formatTime(stop.started_at) }}</span>
              <span v-if="stop.ended_at">- {{ formatTime(stop.ended_at) }}</span>
              <span v-if="stop.duration_ms" class="text-edge-400">{{ formatDuration(stop.duration_ms) }}</span>
              <span class="text-edge-600">{{ stop.source }}</span>
            </div>
            <div v-if="stop.reason" class="text-xs text-edge-500 mt-0.5 truncate">{{ stop.reason }}</div>            <div v-if="stop.justified_by" class="flex items-center gap-1 text-[11px] mt-0.5">
              <span class="text-edge-600">Asignado por</span>
              <span class="text-blue-400 font-medium">{{ stop.justified_by }}</span>
              <span v-if="stop.justified_at" class="text-edge-700">· {{ formatTime(stop.justified_at) }}</span>
            </div>          </div>
        </div>

        <div class="flex items-center gap-1.5 shrink-0 ml-3">
          <button
            v-if="!stop.justified && stop.ended_at"
            class="px-2.5 py-1 text-[11px] font-medium rounded-md bg-production-active/10 text-production-active hover:bg-production-active/20 transition-colors"
            @click="startJustify(stop)"
          >
            Justificar
          </button>
          <button
            v-if="!stop.ended_at"
            class="px-2.5 py-1 text-[11px] font-medium rounded-md bg-stop-assigned/10 text-stop-assigned hover:bg-stop-assigned/20 transition-colors"
            @click="handleClose(stop)"
          >
            Cerrar
          </button>
        </div>
      </div>

      <div v-if="filteredStops.length === 0" class="flex flex-col items-center justify-center py-12 text-edge-500">
        <SvgIcon name="check" :size="32" class="mb-2 text-edge-600" />
        <span class="text-sm">No hay paradas en esta categoria</span>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="justifyingStop"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
        @click.self="justifyingStop = null"
      >
        <div class="w-full max-w-md mx-4">
          <StopForm
            mode="justify"
            :stop="justifyingStop"
            @submit="handleJustify"
            @cancel="justifyingStop = null"
          />
        </div>
      </div>
    </Teleport>
  </div>
</template>
