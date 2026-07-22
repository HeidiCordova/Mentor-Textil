import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { CategoryTreeNode, ProductEntry } from '@/types'
import { api } from '@/services/api'
import { sse } from '@/services/sse'

export const useCatalogStore = defineStore('catalog', () => {
  const stopCategories = ref<CategoryTreeNode[]>([])
  const products = ref<ProductEntry[]>([])
  const loaded = ref(false)

  let _lastLineaId: number | undefined

  async function fetchStopCategories(linea_id?: number): Promise<void> {
    if (linea_id !== undefined) _lastLineaId = linea_id
    try {
      stopCategories.value = await api.stopCategories({ linea_id: _lastLineaId })
    } catch {
      stopCategories.value = []
    }
  }

  async function fetchProducts(linea_id?: number): Promise<void> {
    if (linea_id !== undefined) _lastLineaId = linea_id
    try {
      products.value = await api.productCatalog({ linea_id: _lastLineaId })
    } catch {
      products.value = []
    }
  }

  async function loadAll(linea_id?: number): Promise<void> {
    if (linea_id !== undefined) _lastLineaId = linea_id
    await Promise.allSettled([fetchStopCategories(_lastLineaId), fetchProducts(_lastLineaId)])
    loaded.value = true
  }

  function bindSSE(): void {
    // Modo EDGE: el edge-gateway publica catalogs_synced cuando recibe un comando SYNC
    sse.on('catalogs_synced', () => {
      fetchStopCategories(_lastLineaId).catch(() => {})
      fetchProducts(_lastLineaId).catch(() => {})
    })
    // Modo CLOUD: el cloud-gateway publica catalog.changed cuando alguien modifica el árbol
    sse.on('catalog.changed', () => {
      fetchStopCategories(_lastLineaId).catch(() => {})
    })
  }

  return {
    stopCategories,
    products,
    loaded,
    fetchStopCategories,
    fetchProducts,
    loadAll,
    bindSSE
  }
})
