import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/services/api'
import { sse } from '@/services/sse'

export const useConfigStore = defineStore('config', () => {
  const config = ref<Record<string, unknown>>({})
  const version = ref(0)
  const loading = ref(false)

  async function fetchConfig(): Promise<void> {
    loading.value = true
    try {
      const data = await api.getConfig()
      config.value = data
      if (typeof data.config_version === 'number') {
        version.value = data.config_version
      }
    } catch {
      // retain previous
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(patch: Record<string, unknown>): Promise<boolean> {
    try {
      const data = await api.updateConfig(patch)
      config.value = data
      if (typeof data.config_version === 'number') {
        version.value = data.config_version
      }
      return true
    } catch {
      return false
    }
  }

  async function startCalibration(): Promise<boolean> {
    try {
      await api.startCalibration()
      return true
    } catch {
      return false
    }
  }

  function bindSSE(): void {
    sse.on('config.updated', () => {
      fetchConfig()
    })
  }

  return {
    config,
    version,
    loading,
    fetchConfig,
    updateConfig,
    startCalibration,
    bindSSE
  }
})
