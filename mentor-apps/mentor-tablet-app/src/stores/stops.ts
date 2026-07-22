import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Stop, StopSummary, CreateStopRequest, JustifyStopRequest } from '@/types'
import { api } from '@/services/api'
import { sse } from '@/services/sse'

export const useStopsStore = defineStore('stops', () => {
  const stops = ref<Stop[]>([])
  const summary = ref<StopSummary | null>(null)
  const loading = ref(false)
  const selectedStopId = ref<string | null>(null)

  let _lastParams: Record<string, unknown> = {}

  function activeLineaId(): number | undefined {
    return (_lastParams as { linea_id?: number }).linea_id
  }

  function activePlantaId(): number | undefined {
    return (_lastParams as { planta_id?: number }).planta_id
  }

  const openStops = computed(() => stops.value.filter((s) => !s.ended_at))
  const unjustifiedStops = computed(() =>
    stops.value.filter((s) => !s.justified && s.ended_at)
  )
  const selectedStop = computed(() =>
    stops.value.find((s) => s.stop_id === selectedStopId.value) || null
  )

  async function fetchStops(params?: {
    since?: string
    until?: string
    limit?: number
    linea_id?: number
    planta_id?: number
    empresa_id?: number
  }): Promise<void> {
    if (params !== undefined) {
      _lastParams = params
    }
    loading.value = true
    try {
      stops.value = await api.listStops(_lastParams as Parameters<typeof api.listStops>[0])
    } catch {
      //
    } finally {
      loading.value = false
    }
  }

  async function fetchSummary(hours = 24, params?: { linea_id?: number; empresa_id?: number; planta_id?: number }): Promise<void> {
    try {
      summary.value = await api.stopsSummary({ hours, ...params })
    } catch {}
  }

  async function createStop(data: CreateStopRequest): Promise<Stop | null> {
    try {
      const stop = await api.createStop(data)
      upsertStop(stop)
      return stop
    } catch {
      return null
    }
  }

  async function justifyStop(stopId: string, data: JustifyStopRequest): Promise<boolean> {
    try {
      const updated = await api.justifyStop(stopId, data, activeLineaId(), activePlantaId())
      const idx = stops.value.findIndex((s) => s.stop_id === stopId)
      if (idx >= 0) stops.value.splice(idx, 1, updated)
      return true
    } catch {
      return false
    }
  }

  async function closeStop(stopId: string): Promise<boolean> {
    try {
      const updated = await api.closeStop(stopId, activeLineaId(), activePlantaId())
      const idx = stops.value.findIndex((s) => s.stop_id === stopId)
      if (idx >= 0) stops.value.splice(idx, 1, updated)
      return true
    } catch {
      return false
    }
  }

  async function deleteStop(stopId: string): Promise<boolean> {
    try {
      await api.deleteStop(stopId, activeLineaId(), activePlantaId())
      stops.value = stops.value.filter((s) => s.stop_id !== stopId)
      return true
    } catch {
      return false
    }
  }

  function upsertStop(stop: Stop): void {
    const idx = stops.value.findIndex((s) => s.stop_id === stop.stop_id)
    if (idx >= 0) {
      stops.value.splice(idx, 1, stop)
    } else {
      stops.value.unshift(stop)
    }
  }

  function selectStop(stopId: string | null): void {
    selectedStopId.value = stopId
  }

  function bindSSE(): void {
    sse.on('stop.changed', (msg) => {
      const data = msg.payload as { stop_id?: string } | null | undefined
      if (data?.stop_id) {
        // Modo EDGE: actualizar la parada individual
        api.getStop(data.stop_id, activeLineaId(), activePlantaId()).then(upsertStop).catch(() => {})
      } else if (!data) {
        // Modo CLOUD: el evento solo trae linea_id, recargar todas las paradas
        fetchStops().catch(() => {})
      }
    })
    sse.on('stops_synced', () => {
      fetchStops().catch(() => {})
    })
    sse.on('stop_created', (msg) => {
      const data = msg.payload as { stop_id?: string }
      if (data?.stop_id) {
        api.getStop(data.stop_id, activeLineaId(), activePlantaId()).then(upsertStop).catch(() => {})
      }
    })
    sse.on('stop_closed', (msg) => {
      const data = msg.payload as { stop_id?: string }
      if (data?.stop_id) {
        api.getStop(data.stop_id, activeLineaId(), activePlantaId()).then(upsertStop).catch(() => {})
      }
    })
    sse.on('stop_deleted', (msg) => {
      const data = msg.payload as { stop_id?: string }
      if (data?.stop_id) {
        stops.value = stops.value.filter(s => s.stop_id !== data.stop_id)
      }
    })
  }

  return {
    stops,
    summary,
    loading,
    selectedStopId,
    openStops,
    unjustifiedStops,
    selectedStop,
    fetchStops,
    fetchSummary,
    createStop,
    justifyStop,
    closeStop,
    deleteStop,
    selectStop,
    upsertStop,
    bindSSE
  }
})
