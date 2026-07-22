import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { api } from '@/services/api'
import { useConnectionStore } from '@/stores/connection'

export interface EmpresaEntry {
  id: number
  nombre: string
}

export interface PlantaEntry {
  id: number
  nombre: string
  empresa_id: number
  empresa_nombre: string
}

export interface LineaEntry {
  id: number
  nombre: string
  planta_id: number
}

const STORAGE_KEY_EMPRESA = 'selected_empresa_id'
const STORAGE_KEY_PLANTA  = 'selected_planta_id'
const STORAGE_KEY_LINEA   = 'selected_linea_id'

export const usePlantasLineasStore = defineStore('plantasLineas', () => {
  const plantas = ref<PlantaEntry[]>([])
  const lineas  = ref<LineaEntry[]>([])
  const loaded  = ref(false)
  const loading = ref(false)

  const selectedEmpresaId = ref<number | null>(
    localStorage.getItem(STORAGE_KEY_EMPRESA) ? Number(localStorage.getItem(STORAGE_KEY_EMPRESA)) : null
  )
  const selectedPlantaId = ref<number | null>(
    localStorage.getItem(STORAGE_KEY_PLANTA) ? Number(localStorage.getItem(STORAGE_KEY_PLANTA)) : null
  )
  const selectedLineaId = ref<number | null>(
    localStorage.getItem(STORAGE_KEY_LINEA) ? Number(localStorage.getItem(STORAGE_KEY_LINEA)) : null
  )

  // ---- derivados ----
  const connection = useConnectionStore()

  /** rol normalizado a minúsculas */
  const rol = computed(() => (connection.operator?.rol ?? '').toLowerCase())

  const isAdmin       = computed(() => rol.value === 'admin' || rol.value === 'superadmin')
  const isAdminPlanta = computed(() => rol.value === 'admin_planta')
  const operadorEmpresaId = computed(() => connection.operator?.empresa_id ?? null)

  /** Empresas únicas derivadas de las plantas cargadas */
  const empresas = computed<EmpresaEntry[]>(() => {
    const map = new Map<number, string>()
    for (const p of plantas.value) {
      const nombre = p.empresa_nombre || p.nombre || `Empresa ${p.empresa_id}`
      if (!map.has(p.empresa_id)) map.set(p.empresa_id, nombre)
    }
    return Array.from(map.entries()).map(([id, nombre]) => ({ id, nombre }))
  })

  /** ADMIN ve todas las empresas; usuarios con empresa_id ven solo la suya; sin empresa_id ven todas */
  const empresasVisibles = computed<EmpresaEntry[]>(() => {
    if (isAdmin.value) return empresas.value
    const eid = operadorEmpresaId.value
    if (!eid) return empresas.value  // sin filtro de empresa: mostrar todas (ej. superadmin sin empresa_id)
    return empresas.value.filter(e => e.id === eid)
  })

  const empresaActual = computed<EmpresaEntry | null>(() =>
    empresasVisibles.value.find(e => e.id === selectedEmpresaId.value) ?? null
  )

  /** Plantas visibles: para ADMIN filtra por empresa seleccionada; para otros por empresa propia */
  const plantasVisibles = computed<PlantaEntry[]>(() => {
    // Admin o usuario sin empresa_id: filtra por empresa seleccionada (o ningún filtro si no hay selección)
    if (isAdmin.value || !operadorEmpresaId.value) {
      const eid = selectedEmpresaId.value
      if (!eid) return plantas.value  // sin empresa seleccionada: mostrar todas
      return plantas.value.filter(p => p.empresa_id === eid)
    }
    // Rol normal: solo plantas de su empresa
    return plantas.value.filter(p => p.empresa_id === operadorEmpresaId.value)
  })

  /** Líneas visibles según planta seleccionada (dentro de las plantas de la empresa) */
  const lineasVisibles = computed<LineaEntry[]>(() => {
    const validPlantaIds = new Set(plantasVisibles.value.map(p => p.id))
    const plantaId = selectedPlantaId.value
    if (plantaId && validPlantaIds.has(plantaId)) {
      return lineas.value.filter(l => l.planta_id === plantaId)
    }
    // Sin planta seleccionada: mostrar todas las líneas de las plantas visibles
    return lineas.value.filter(l => validPlantaIds.has(l.planta_id))
  })

  const plantaActual = computed<PlantaEntry | null>(() =>
    plantas.value.find(p => p.id === selectedPlantaId.value) ?? null
  )

  const lineaActual = computed<LineaEntry | null>(() =>
    lineas.value.find(l => l.id === selectedLineaId.value) ?? null
  )

  // ---- acciones ----
  async function load(): Promise<void> {
    // Solo saltar si ya tenemos datos reales — si plantas está vacío puede ser
    // una carga fallida previa (race condition en el arranque) y hay que reintentar
    if (loaded.value && plantas.value.length > 0) return
    loading.value = true
    try {
      const data = await api.plantsLines()
      plantas.value = data.plantas ?? []
      lineas.value  = data.lineas ?? []
      loaded.value  = true

      // Autoseleccionar empresa si ADMIN y no tiene ninguna válida
      if (isAdmin.value) {
        const validEmpresaIds = new Set(empresasVisibles.value.map(e => e.id))
        if (!selectedEmpresaId.value || !validEmpresaIds.has(selectedEmpresaId.value)) {
          if (empresasVisibles.value.length > 0) {
            selectedEmpresaId.value = empresasVisibles.value[0].id
            localStorage.setItem(STORAGE_KEY_EMPRESA, String(selectedEmpresaId.value))
          }
        }
      }

      // Validar selección contra las plantas visibles del usuario actual
      // (puede haber un valor stale de otro usuario/empresa en localStorage)
      const validPlantaIds = new Set(plantasVisibles.value.map(p => p.id))
      if (selectedPlantaId.value && !validPlantaIds.has(selectedPlantaId.value)) {
        selectedPlantaId.value = null
        localStorage.removeItem(STORAGE_KEY_PLANTA)
      }
      const validLineaIds = new Set(lineasVisibles.value.map(l => l.id))
      if (selectedLineaId.value && !validLineaIds.has(selectedLineaId.value)) {
        selectedLineaId.value = null
        localStorage.removeItem(STORAGE_KEY_LINEA)
      }

      // Si no hay selección válida, autoseleccionar la primera opción
      if (!selectedPlantaId.value && plantasVisibles.value.length > 0) {
        selectPlanta(plantasVisibles.value[0].id)
      }
      if (!selectedLineaId.value && lineasVisibles.value.length > 0) {
        selectLinea(lineasVisibles.value[0].id)
      }
    } catch {
      // cloud no disponible: continuar con datos vacíos
    } finally {
      loading.value = false
    }
  }

  function selectEmpresa(id: number): void {
    selectedEmpresaId.value = id
    localStorage.setItem(STORAGE_KEY_EMPRESA, String(id))
    // Resetear planta y línea al cambiar empresa
    selectedPlantaId.value = null
    localStorage.removeItem(STORAGE_KEY_PLANTA)
    selectedLineaId.value = null
    localStorage.removeItem(STORAGE_KEY_LINEA)
    // Autoseleccionar primera planta de la empresa elegida
    const primeraPlanta = plantas.value.find(p => p.empresa_id === id)
    if (primeraPlanta) selectPlanta(primeraPlanta.id)
  }

  function selectPlanta(id: number): void {
    selectedPlantaId.value = id
    localStorage.setItem(STORAGE_KEY_PLANTA, String(id))
    // Al cambiar planta, resetear línea a la primera de esa planta
    const primera = lineas.value.find(l => l.planta_id === id)
    if (primera) selectLinea(primera.id)
    else {
      selectedLineaId.value = null
      localStorage.removeItem(STORAGE_KEY_LINEA)
    }
  }

  function selectLinea(id: number): void {
    selectedLineaId.value = id
    localStorage.setItem(STORAGE_KEY_LINEA, String(id))
  }

  function reset(): void {
    plantas.value = []
    lineas.value  = []
    loaded.value  = false
    selectedEmpresaId.value = null
    selectedPlantaId.value  = null
    selectedLineaId.value   = null
    localStorage.removeItem(STORAGE_KEY_EMPRESA)
    localStorage.removeItem(STORAGE_KEY_PLANTA)
    localStorage.removeItem(STORAGE_KEY_LINEA)
  }

  // Cuando el operador inicia sesión (empresa_id pasa de null a un valor),
  // disparar auto-selección si los datos ya están cargados y no hay selección.
  watch(operadorEmpresaId, (newId) => {
    if (!newId || !loaded.value) return
    // Para admin, autoseleccionar empresa si no tiene ninguna
    if (isAdmin.value && !selectedEmpresaId.value && empresasVisibles.value.length > 0) {
      selectedEmpresaId.value = empresasVisibles.value[0].id
      localStorage.setItem(STORAGE_KEY_EMPRESA, String(selectedEmpresaId.value))
    }
    if (!selectedPlantaId.value && plantasVisibles.value.length > 0) {
      selectPlanta(plantasVisibles.value[0].id)
    }
    if (!selectedLineaId.value && lineasVisibles.value.length > 0) {
      selectLinea(lineasVisibles.value[0].id)
    }
  })

  return {
    plantas, lineas, loaded, loading,
    selectedEmpresaId, selectedPlantaId, selectedLineaId,
    isAdmin, isAdminPlanta,
    empresas, empresasVisibles, empresaActual,
    plantasVisibles, lineasVisibles,
    plantaActual, lineaActual,
    load, selectEmpresa, selectPlanta, selectLinea, reset
  }
})
