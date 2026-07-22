import { ref } from 'vue'
import { useUiStore } from '@/stores/ui'

export function useApi() {
  const loading = ref(false)
  const error = ref(null)
  const uiStore = useUiStore()

  async function execute(apiCall, options = {}) {
    loading.value = true
    error.value = null
    
    if (options.showGlobalLoading) {
      uiStore.setLoading(true)
    }

    try {
      const response = await apiCall()
      return { data: response, error: null }
    } catch (err) {
      error.value = err.response?.data?.error || err.response?.data?.message || err.message || 'Error desconocido'
      return { data: null, error: error.value }
    } finally {
      loading.value = false
      if (options.showGlobalLoading) {
        uiStore.setLoading(false)
      }
    }
  }

  return {
    loading,
    error,
    execute
  }
}
