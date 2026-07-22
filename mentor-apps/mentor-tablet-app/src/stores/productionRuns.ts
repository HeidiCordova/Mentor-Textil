import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ProductionRun, UpsertProductionRunRequest } from '@/types'
import { api } from '@/services/api'
import { sse } from '@/services/sse'

export const useProductionRunsStore = defineStore('productionRuns', () => {
  const runs = ref<ProductionRun[]>([])
  const loading = ref(false)

  let _lastParams: { since?: string; until?: string; linea_id?: number; planta_id?: number; limit?: number } = {}

  async function fetchRuns(params?: {
    since?: string
    until?: string
    linea_id?: number
    planta_id?: number
    limit?: number
  }): Promise<void> {
    if (params !== undefined) {
      _lastParams = params
    }
    loading.value = true
    try {
      runs.value = await api.listProductionRuns(_lastParams)
    } catch {
      // retener estado previo
    } finally {
      loading.value = false
    }
  }

  async function upsert(data: UpsertProductionRunRequest, params?: { linea_id?: number; planta_id?: number }): Promise<ProductionRun[] | null> {
    try {
      const updated = await api.upsertProductionRun(data, params)
      runs.value = updated
      return updated
    } catch {
      return null
    }
  }

  async function remove(runId: string): Promise<boolean> {
    try {
      const updated = await api.deleteProductionRun(runId)
      runs.value = updated
      return true
    } catch {
      return false
    }
  }

  function bindSSE(): void {
    sse.on('production_runs_updated', () => {
      fetchRuns().catch(() => {})
    })
  }

  return { runs, loading, fetchRuns, upsert, remove, bindSSE }
})
