<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Modal from '@/shared/components/ui/Modal.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { variableService } from '@/api/services/variable.service'
import { deviceService } from '@/api/services/device.service'
import { locacionService } from '@/api/services/locacion.service'
import { arbolParadasService } from '@/api/services/arbolParadas.service'
import { productoCaractService } from '@/api/services/productoCaract.service'
import { useAuthStore } from '@/stores/auth'
import { useModal } from '@/shared/composables/useModal'
import { useApi } from '@/shared/composables/useApi'

const authStore = useAuthStore()
const { isOpen: isCreateOpen, open: openCreateModal, close: closeCreateModal } = useModal()
const { isOpen: isEditOpen, open: openEditModal, close: closeEditModal } = useModal()
const { loading, error, execute } = useApi()

const variables = ref([])
const empresas = ref([])
const plantas = ref([])
const lineas = ref([])
const locaciones = ref([])
const dispositivos = ref([])
const selectedRows = ref([])
const activeTab = ref('continuas')

const paradaArbol = ref({ programadas: [], no_programadas: [] })
const loadingArbol = ref(false)
const searchArbol  = ref('')
const arbolPage    = ref(1)
const ARBOL_PAGE   = 150    // registros visibles por página

const isParadaType = (tipo) => tipo === 'PARADA_PROGRAMADA' || tipo === 'PARADA_NO_PROGRAMADA'

// ─── Catálogo de valores por línea+variable ───────────────────────────────────
const catalogoEditando    = ref([])
const catalogoNuevoValor  = ref('')
const guardandoCatalogo   = ref(false)
const catalogoGuardadoOk  = ref(false)
// ID de la variable en el schema de planta (distinto al ID cloud)
const plantaVarId = ref(null)

// Resuelve el ID planta de una variable cloud buscando por nombre en /variables-linea
async function resolvePlantaVarId(cloudNombre) {
  plantaVarId.value = null
  if (!lineaFiltro.value || !cloudNombre) return
  try {
    const res = await productoCaractService.getVariablesLinea({ linea_id: lineaFiltro.value })
    const match = (res.data || []).find(v => v.nombre === cloudNombre)
    if (match) plantaVarId.value = match.id
  } catch { /* ignorar */ }
}

async function loadCatalogo(variableId) {
  catalogoEditando.value = []
  const vid = plantaVarId.value ?? variableId
  if (!lineaFiltro.value || !vid) return
  try {
    const res = await productoCaractService.getCatalogo({
      linea_id: lineaFiltro.value,
      variable_id: vid
    })
    catalogoEditando.value = res.data ?? []
  } catch { /* sin catálogo previo = ok */ }
}

function agregarCatalogoValor() {
  const v = catalogoNuevoValor.value.trim().toUpperCase()
  if (!v) return
  if (!catalogoEditando.value.includes(v)) catalogoEditando.value.push(v)
  catalogoNuevoValor.value = ''
}

async function guardarCatalogo() {
  const vid = plantaVarId.value ?? editForm.value.id
  if (!lineaFiltro.value || !vid) return
  guardandoCatalogo.value = true
  catalogoGuardadoOk.value = false
  try {
    await productoCaractService.saveCatalogo({
      linea_id: Number(lineaFiltro.value),
      variable_id: Number(vid),
      valores: catalogoEditando.value
    })
    catalogoGuardadoOk.value = true
    setTimeout(() => { catalogoGuardadoOk.value = false }, 2500)
  } finally {
    guardandoCatalogo.value = false
  }
}

const canEdit = computed(() => selectedRows.value.length === 1)

const createForm = ref({
  empresaId: '',
  plantaId: '',
  lineaId: '',
  locacionId: '',
  dispositivoId: '',
  nombre: '',
  nombreCorto: '',
  archivo: '',
  columnaArchivo: '',
  unidad: '',
  tipo: 'ENERGIA',
  variableRelacionada: '',
  color: '#000000',
  visibleConsulta: 'Activado',
  estado: 'Activado',
  leyenda: []
})

const editForm = ref({
  id: '',
  empresaId: '',
  plantaId: '',
  lineaId: '',
  locacionId: '',
  dispositivoId: '',
  nombre: '',
  nombreCorto: '',
  archivo: '',
  columnaArchivo: '',
  unidad: '',
  tipo: 'ENERGIA',
  variableRelacionada: '',
  color: '#000000',
  visibleConsulta: 'Activado',
  estado: 'Activado',
  leyenda: []
})

const toggleRowSelection = (item) => {
  selectedRows.value = [item]
}

const isRowSelected = (item) => {
  return selectedRows.value.some(row => row.id === item.id)
}

const loadData = async () => {
  await execute(async () => {
    const companiesRes = await companyService.getAll()
    empresas.value = companiesRes.data || []

    const empresaId = empresaFiltro.value || authStore.user?.empresa_id || null

    if (empresaId) {
      const plantasRes = await plantService.getAll({ empresa_id: empresaId })
      plantas.value = plantasRes.data || []
    }

    if (plantaFiltro.value) {
      const lineasRes = await lineService.getAll({ planta_id: plantaFiltro.value })
      lineas.value = lineasRes.data || []
    }

    if (empresaId) {
      const dispRes = await deviceService.getAll({ empresa_id: empresaId })
      dispositivos.value = dispRes.data || []
    }

    if (lineaFiltro.value) {
      const locRes = await locacionService.getAll({ linea_id: lineaFiltro.value })
      locaciones.value = locRes.data || []
    } else {
      locaciones.value = []
    }

    const varParams = {}
    if (dispositivoFiltro.value) {
      varParams.dispositivo_id = dispositivoFiltro.value
    }
    const variablesRes = await variableService.getAll(varParams)
    const allVars = variablesRes.data || []
    variables.value = allVars.filter(v => {
      if (activeTab.value === 'continuas') return v.tipo !== 'CONSTANTE' && v.tipo !== 'PROPORCION'
      if (activeTab.value === 'constantes') return v.tipo === 'CONSTANTE'
      if (activeTab.value === 'proporciones') return v.tipo === 'PROPORCION'
      return true
    })
  })
}

const openCreate = () => {
  resetCreateForm()
  openCreateModal()
}

const openEdit = () => {
  if (selectedRows.value.length === 1) {
    const item = selectedRows.value[0]
    editForm.value = { ...item }
    if (isParadaType(item.tipo) && !editForm.value.unidad) {
      editForm.value.unidad = 'segundos'
    }
    openEditModal()
    // cargar catálogo si hay filtro de línea y no es variable de parada
    if (lineaFiltro.value && !isParadaType(item.tipo)) {
      resolvePlantaVarId(item.nombre).then(() => loadCatalogo(item.id))
    } else {
      plantaVarId.value = null
      catalogoEditando.value = []
    }
    if (isParadaType(item.tipo)) {
      const linea = lineaFiltro.value || (dispositivos.value.find(d => d.id === (item.dispositivo_id || item.dispositivoId))?.linea_id)
      if (linea) reloadArbol(linea)
    }
  }
}

const resetCreateForm = () => {
  createForm.value = {
    empresaId: empresaFiltro.value || '',
    plantaId: plantaFiltro.value || '',
    lineaId: lineaFiltro.value || '',
    locacionId: '',
    dispositivoId: dispositivoFiltro.value || '',
    nombre: '',
    nombreCorto: '',
    archivo: '',
    columnaArchivo: '',
    unidad: '',
    tipo: 'ENERGIA',
    variableRelacionada: '',
    color: '#000000',
    visibleConsulta: 'Activado',
    estado: 'Activado',
    leyenda: []
  }
}

const handleCreate = async () => {
  await execute(async () => {
    await variableService.create({
      nombre: createForm.value.nombre,
      clave: createForm.value.nombreCorto,
      valor: createForm.value.columnaArchivo || '',
      tipo: createForm.value.tipo,
      dispositivo_id: createForm.value.dispositivoId ? parseInt(createForm.value.dispositivoId) : null,
      planta_id: createForm.value.plantaId ? parseInt(createForm.value.plantaId) : null,
      empresa_id: createForm.value.empresaId ? parseInt(createForm.value.empresaId) : null
    })
    await loadData()
    closeCreateModal()
  })
}

const handleEdit = async () => {
  await execute(async () => {
    await variableService.update(editForm.value.id, {
      nombre: editForm.value.nombre,
      clave: editForm.value.nombreCorto || editForm.value.clave || editForm.value.nombre,
      valor: editForm.value.columnaArchivo || editForm.value.valor || '',
      tipo: editForm.value.tipo,
      dispositivo_id: editForm.value.dispositivoId ? parseInt(editForm.value.dispositivoId) : (editForm.value.dispositivo_id || null),
      planta_id: editForm.value.plantaId ? parseInt(editForm.value.plantaId) : (editForm.value.planta_id || null),
      empresa_id: editForm.value.empresaId ? parseInt(editForm.value.empresaId) : (editForm.value.empresa_id || null),
      activo: editForm.value.estado === 'Activado'
    })
    await loadData()
    closeEditModal()
    selectedRows.value = []
  })
}

const addLeyenda = () => {}

// Recarga el árbol desde la API — llamado al abrir el modal y desde el botón Actualizar
function reloadArbol(lineaId) {
  const linea = lineaId ||
    lineaFiltro.value ||
    dispositivos.value.find(d => d.id === (editForm.value.dispositivo_id || editForm.value.dispositivoId))?.linea_id
  if (!linea) return
  loadingArbol.value = true
  searchArbol.value  = ''
  arbolPage.value    = 1
  paradaArbol.value  = { programadas: [], no_programadas: [] }
  arbolParadasService.get(linea)
    .then(r => { paradaArbol.value = r || { programadas: [], no_programadas: [] } })
    .finally(() => { loadingArbol.value = false })
}

// Árbol aplanado con metadatos completos por nodo
const paradaArbolRows = computed(() => {
  if (!editForm.value.tipo) return []
  const rows = editForm.value.tipo === 'PARADA_PROGRAMADA'
    ? paradaArbol.value.programadas
    : paradaArbol.value.no_programadas
  if (!rows || !rows.length) return []

  const nodes = []
  let lastCat = null
  let lastSub = null

  for (const r of rows) {
    const cat  = r.categoria    || '—'
    const sub  = r.subcategoria || '—'
    const sub2 = r.subcategoria_2 || ''
    const desc = r.descripcion_parada || ''
    const leafLabel = sub2 || sub
    const fullPath  = sub2 ? `${cat} › ${sub} › ${sub2}` : `${cat} › ${sub}`

    if (cat !== lastCat) {
      nodes.push({ type: 'cat', label: cat })
      lastCat = cat
      lastSub = null
    }
    if (sub2) {
      if (sub !== lastSub) {
        nodes.push({ type: 'sub', label: sub })
        lastSub = sub
      }
    }
    nodes.push({ type: 'leaf', dbId: r.id, label: leafLabel, fullPath, desc })
  }
  return nodes
})

// Filtrado por búsqueda: cuando hay query, aplanar a solo hojas con fullPath mostrado
const paradaArbolFiltered = computed(() => {
  const q = searchArbol.value.trim().toLowerCase()
  if (!q) return paradaArbolRows.value
  return paradaArbolRows.value.filter(n =>
    n.type === 'leaf' &&
    (n.label.toLowerCase().includes(q) ||
     n.fullPath.toLowerCase().includes(q) ||
     String(n.dbId).includes(q))
  )
})

// Paginación virtual: muestra ARBOL_PAGE * página actual
const paradaArbolVisible  = computed(() => paradaArbolFiltered.value.slice(0, arbolPage.value * ARBOL_PAGE))
const paradaArbolHasMore  = computed(() => paradaArbolFiltered.value.length > arbolPage.value * ARBOL_PAGE)
const paradaArbolTotal    = computed(() => paradaArbolFiltered.value.filter(n => n.type === 'leaf').length)

watch(activeTab, () => {
  selectedRows.value = []
  loadData()
})

const empresaFiltro = ref('')
const plantaFiltro = ref('')
const lineaFiltro = ref('')
const locacionFiltro = ref('')
const dispositivoFiltro = ref('')

const variablesFiltradas = computed(() => variables.value)

const plantasFiltradas = computed(() => {
  if (!empresaFiltro.value) return plantas.value
  return plantas.value.filter(p => p.empresa_id === parseInt(empresaFiltro.value))
})

const lineasFiltradas = computed(() => {
  if (!plantaFiltro.value) return lineas.value
  return lineas.value.filter(l => l.planta_id === parseInt(plantaFiltro.value))
})

const locacionesFiltradas = computed(() => locaciones.value)

const dispositivosFiltrados = computed(() => {
  let list = dispositivos.value
  if (lineaFiltro.value) list = list.filter(d => d.linea_id === parseInt(lineaFiltro.value))
  if (locacionFiltro.value) list = list.filter(d => !d.locacion_id || d.locacion_id === parseInt(locacionFiltro.value))
  return list
})

watch(empresaFiltro, () => { plantaFiltro.value = ''; lineaFiltro.value = ''; locacionFiltro.value = ''; dispositivoFiltro.value = ''; loadData() })
watch(plantaFiltro, () => { lineaFiltro.value = ''; locacionFiltro.value = ''; dispositivoFiltro.value = ''; loadData() })
watch(lineaFiltro, () => { locacionFiltro.value = ''; dispositivoFiltro.value = ''; loadData() })
watch(locacionFiltro, () => { dispositivoFiltro.value = ''; loadData() })
watch(dispositivoFiltro, () => { loadData() })

watch(() => createForm.value.empresaId, () => {
  createForm.value.plantaId = ''
  createForm.value.lineaId = ''
  createForm.value.dispositivoId = ''
})
watch(() => createForm.value.plantaId, () => {
  createForm.value.lineaId = ''
  createForm.value.dispositivoId = ''
})
watch(() => editForm.value.empresaId, () => {
  editForm.value.plantaId = ''
  editForm.value.lineaId = ''
  editForm.value.dispositivoId = ''
})
watch(() => editForm.value.plantaId, () => {
  editForm.value.lineaId = ''
  editForm.value.dispositivoId = ''
})

onMounted(() => {
  const eid = authStore.user?.empresa_id
  if (eid) empresaFiltro.value = eid
  loadData()
})
</script>

<template>
  <div class="variables-view">
    <div class="filters-section">
      <div class="filter-row">
        <div class="filter-item">
          <label class="filter-label">Compañía</label>
          <select v-model="empresaFiltro" class="field-select">
            <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
              {{ empresa.nombre }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Planta</label>
          <select v-model="plantaFiltro" class="field-select">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Línea</label>
          <select v-model="lineaFiltro" class="field-select">
            <option v-for="linea in lineasFiltradas" :key="linea.id" :value="linea.id">
              {{ linea.nombre }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Locación</label>
          <select v-model="locacionFiltro" class="field-select">
            <option v-for="locacion in locacionesFiltradas" :key="locacion.id" :value="locacion.id">
              {{ locacion.nombre }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Dispositivo</label>
          <select v-model="dispositivoFiltro" class="field-select">
            <option v-for="dispositivo in dispositivosFiltrados" :key="dispositivo.id" :value="dispositivo.id">
              {{ dispositivo.nombre }}
            </option>
          </select>
        </div>
      </div>
    </div>

    <Card class="content-card">
      <div class="tabs-header">
        <button 
          :class="['tab-btn', { active: activeTab === 'continuas' }]"
          @click="activeTab = 'continuas'"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"/>
          </svg>
          VARIABLES CONTINUAS
        </button>
        <button 
          :class="['tab-btn', 'tab-btn-wip']"
          disabled
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z"/>
          </svg>
          VARIABLES CONSTANTES
          <span class="tab-wip-badge">En desarrollo</span>
        </button>
        <button 
          :class="['tab-btn', 'tab-btn-wip']"
          disabled
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z"/>
          </svg>
          PROPORCIONES
          <span class="tab-wip-badge">En desarrollo</span>
        </button>
      </div>

      <div class="action-bar">
        <Button @click="openCreate" variant="primary" size="sm">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          AGREGAR
        </Button>
        <Button 
          @click="openEdit"
          :disabled="!canEdit"
          variant="secondary" 
          size="sm"
        >
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
          </svg>
          EDITAR
        </Button>
      </div>

      <Loading v-if="loading" />
      <Alert v-else-if="error" type="error" :message="error" />

      <div v-else class="table-container">
        <!-- Tabla Variables Continuas -->
        <table v-if="activeTab === 'continuas'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Nombre Corto</th>
              <th>Archivo/Nombre</th>
              <th>Estado</th>
              <th>Unidad</th>
              <th>↑Creado</th>
              <th>Modificado</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(variable, index) in variablesFiltradas" 
              :key="variable.id"
              @click="toggleRowSelection(variable)"
              :class="{ 'row-selected': isRowSelected(variable) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(variable)"
                  @click.stop="toggleRowSelection(variable)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ variable.nombre }}</td>
              <td>{{ variable.nombreCorto }}</td>
              <td>{{ variable.archivo }}</td>
              <td>{{ variable.estado }}</td>
              <td>{{ variable.unidad }}</td>
              <td>{{ variable.creado }}</td>
              <td>{{ variable.modificado }}</td>
            </tr>
          </tbody>
        </table>

        <!-- Tabla Variables Constantes -->
        <table v-else-if="activeTab === 'constantes'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Nombre Corto</th>
              <th>Estado</th>
              <th>Unidad</th>
              <th>Valor</th>
              <th>↑Creado</th>
              <th>Modificado</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(variable, index) in variablesFiltradas" 
              :key="variable.id"
              @click="toggleRowSelection(variable)"
              :class="{ 'row-selected': isRowSelected(variable) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(variable)"
                  @click.stop="toggleRowSelection(variable)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ variable.nombre }}</td>
              <td>{{ variable.nombreCorto }}</td>
              <td>{{ variable.estado }}</td>
              <td>{{ variable.unidad }}</td>
              <td>{{ variable.valor }}</td>
              <td>{{ variable.creado }}</td>
              <td>{{ variable.modificado }}</td>
            </tr>
          </tbody>
        </table>

        <!-- Tabla Proporciones -->
        <table v-else-if="activeTab === 'proporciones'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Nombre Corto</th>
              <th>Estado</th>
              <th>Unidad</th>
              <th>Tipo</th>
              <th>Var1</th>
              <th>Operación.</th>
              <th>Var2</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(variable, index) in variablesFiltradas" 
              :key="variable.id"
              @click="toggleRowSelection(variable)"
              :class="{ 'row-selected': isRowSelected(variable) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(variable)"
                  @click.stop="toggleRowSelection(variable)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ variable.nombre }}</td>
              <td>{{ variable.nombreCorto }}</td>
              <td>{{ variable.estado }}</td>
              <td>{{ variable.unidad }}</td>
              <td>{{ variable.tipo }}</td>
              <td>{{ variable.var1 }}</td>
              <td>{{ variable.operacion }}</td>
              <td>{{ variable.var2 }}</td>
            </tr>
          </tbody>
        </table>

        <div class="pagination">
          <button class="pagination-btn">ANTERIOR</button>
          <span class="pagination-info">1</span>
          <button class="pagination-btn">PRÓXIMO</button>
        </div>
      </div>
    </Card>

    <!-- Modal Crear -->
    <Modal v-model="isCreateOpen" @close="closeCreateModal" title="AGREGAR NUEVA VARIABLE CONTINUA" size="lg">
      <form @submit.prevent="handleCreate" class="form-container">
        <div class="form-content">
          <div class="form-grid">
            <div class="form-row">
              <label class="form-label">Compañía</label>
              <select v-model="createForm.empresaId" class="field-input" required>
                <option value="" disabled>Seleccione</option>
                <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
                  {{ empresa.nombre }}
                </option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Planta</label>
              <select v-model="createForm.plantaId" class="field-input" required>
                <option value="" disabled>Seleccione</option>
                <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
                  {{ planta.nombre }}
                </option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Línea</label>
              <select v-model="createForm.lineaId" class="field-input" required>
                <option value="" disabled>Seleccione</option>
                <option v-for="linea in lineasFiltradas" :key="linea.id" :value="linea.id">
                  {{ linea.nombre }}
                </option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Locación</label>
              <select v-model="createForm.locacionId" class="field-input" required>
                <option value="" disabled>Seleccione</option>
                <option v-for="locacion in locacionesFiltradas" :key="locacion.id" :value="locacion.id">
                  {{ locacion.nombre }}
                </option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Dispositivo</label>
              <select v-model="createForm.dispositivoId" class="field-input" required>
                <option value="" disabled>Seleccione</option>
                <option v-for="dispositivo in dispositivosFiltrados" :key="dispositivo.id" :value="dispositivo.id">
                  {{ dispositivo.nombre }}
                </option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Nombre</label>
              <input 
                v-model="createForm.nombre" 
                type="text" 
                class="field-input" 
                placeholder="Nombre de varible continua"
                required
              />
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Nombre Corto</label>
              <input 
                v-model="createForm.nombreCorto" 
                type="text" 
                class="field-input" 
                placeholder="Nombre corto de la varible continua"
                required
              />
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Archivo</label>
              <select v-model="createForm.archivo" class="field-input" required>
                <option value="" disabled>Seleccione</option>
                <option value="energiasKR">energiasKR</option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Columna en Archivo</label>
              <input 
                v-model="createForm.columnaArchivo" 
                type="text" 
                class="field-input" 
                placeholder="Nombre de Columna"
                required
              />
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Unidad</label>
              <input 
                v-model="createForm.unidad" 
                type="text" 
                class="field-input" 
                placeholder="Unidad de la varible continua"
                required
              />
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Tipo</label>
              <select v-model="createForm.tipo" class="field-input" required>
                <option value="ENERGIA">ENERGIA</option>
                <option value="PRODUCCION">PRODUCCION</option>
                <option value="OTRO">OTRO</option>
              </select>
            </div>

            <div class="form-row">
              <label class="form-label">Variable relacionada</label>
              <select v-model="createForm.variableRelacionada" class="field-input">
                <option value="">Seleccione...</option>
              </select>
              <span class="field-hint">Debe completar este campo</span>
            </div>

            <div class="form-row">
              <label class="form-label">Color</label>
              <input 
                v-model="createForm.color" 
                type="text" 
                class="field-input"
              />
            </div>
          </div>

          <div class="form-section">
            <label class="section-label">Visible en consulta</label>
            <div class="button-group">
              <button 
                type="button"
                :class="['option-btn', { active: createForm.visibleConsulta === 'Activado' }]"
                @click="createForm.visibleConsulta = 'Activado'"
              >
                Activado
              </button>
              <button 
                type="button"
                :class="['option-btn', { active: createForm.visibleConsulta === 'Deshabilitado' }]"
                @click="createForm.visibleConsulta = 'Deshabilitado'"
              >
                Deshabilitado
              </button>
            </div>
          </div>

          <div class="form-section">
            <label class="section-label">Estado</label>
            <div class="button-group">
              <button 
                type="button"
                :class="['option-btn', { active: createForm.estado === 'Activado' }]"
                @click="createForm.estado = 'Activado'"
              >
                Activado
              </button>
              <button 
                type="button"
                :class="['option-btn', { active: createForm.estado === 'No almacenado' }]"
                @click="createForm.estado = 'No almacenado'"
              >
                No almacenado
              </button>
              <button 
                type="button"
                :class="['option-btn', { active: createForm.estado === 'Deshabilitado' }]"
                @click="createForm.estado = 'Deshabilitado'"
              >
                Deshabilitado
              </button>
            </div>
          </div>

          <div class="form-section">
            <div class="section-header">
              <label class="section-label">LEYENDA</label>
              <button type="button" class="add-btn" @click="addLeyenda">+</button>
            </div>
          </div>
        </div>

        <div class="form-footer">
          <Button type="submit" variant="primary" size="md" :loading="loading">
            GUARDAR
          </Button>
          <Button type="button" @click="closeCreateModal" variant="secondary" size="md">
            CANCELAR
          </Button>
        </div>
      </form>
    </Modal>

    <!-- Modal Editar -->
    <Modal v-model="isEditOpen" @close="closeEditModal"
      :title="isParadaType(editForm.tipo) ? 'EDITAR VARIABLE DE PARADA OEE' : 'EDITAR VARIABLE CONTINUA'"
      size="lg">
      <form @submit.prevent="handleEdit" class="form-container">
        <div class="form-content">

          <!-- ── Contexto: dispositivo / linea (solo lectura) ── -->
          <div class="ctx-banner">
            <span class="ctx-chip ctx-chip--device">
              <svg class="ctx-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
              </svg>
              {{ dispositivos.find(d => d.id === (editForm.dispositivo_id ?? editForm.dispositivoId))?.nombre || 'Sin dispositivo' }}
            </span>
            <span class="ctx-chip ctx-chip--linea">
              <svg class="ctx-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7"/>
              </svg>
              {{ lineas.find(l => l.id === parseInt(lineaFiltro))?.nombre || 'Línea' }}
            </span>
            <span v-if="isParadaType(editForm.tipo)" class="ctx-chip ctx-chip--tipo">
              {{ editForm.tipo === 'PARADA_PROGRAMADA' ? 'Parada Programada' : 'Parada No Programada' }}
            </span>
          </div>

          <!-- ── Variables de PARADA (layout simplificado) ── -->
          <template v-if="isParadaType(editForm.tipo)">
            <div class="form-grid">
              <div class="form-row" style="grid-column: 1 / -1">
                <label class="form-label">Nombre de la variable</label>
                <input v-model="editForm.nombre" type="text" class="field-input" required />
                <span class="field-hint-info">Nombre visible en dashboards y reportes OEE</span>
              </div>

              <div class="form-row">
                <label class="form-label">Clave de sistema</label>
                <input :value="editForm.clave || editForm.nombreCorto" type="text" class="field-input field-readonly" readonly />
                <span class="field-hint-info">No modificable — identificador del protocolo edge</span>
              </div>

              <div class="form-row">
                <label class="form-label">Unidad de tiempo</label>
                <input v-model="editForm.unidad" type="text" class="field-input" placeholder="segundos" />
              </div>

              <div class="form-row">
                <label class="form-label">Color de referencia</label>
                <div class="color-row">
                  <input v-model="editForm.color" type="color" class="color-picker" />
                  <input v-model="editForm.color" type="text" class="field-input color-text" placeholder="#000000" />
                </div>
              </div>
            </div>

            <div class="form-section">
              <label class="section-label">Estado</label>
              <div class="button-group">
                <button type="button" :class="['option-btn', { active: editForm.estado === 'Activado' }]" @click="editForm.estado = 'Activado'">Activado</button>
                <button type="button" :class="['option-btn', { active: editForm.estado === 'Deshabilitado' }]" @click="editForm.estado = 'Deshabilitado'">Deshabilitado</button>
              </div>
            </div>

            <!-- ── Panel Árbol de Paradas (solo para T_PARADA_PROGRAMADA / T_PARADA_NO_PROGRAMADA) ── -->
            <div v-if="editForm.clave === 'T_PARADA_PROGRAMADA' || editForm.clave === 'T_PARADA_NO_PROGRAMADA'" class="arbol-panel">
              <div class="arbol-panel__header">
                <div class="arbol-panel__title">
                  <svg class="ctx-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7h18M3 12h12M3 17h8"/>
                  </svg>
                  ÁRBOL — {{ editForm.tipo === 'PARADA_PROGRAMADA' ? 'Programadas' : 'No Programadas' }}
                  <span v-if="!loadingArbol && paradaArbolTotal" class="arbol-count">{{ paradaArbolTotal }} paradas</span>
                </div>
                <button type="button" class="arbol-refresh-btn" :disabled="loadingArbol" @click="reloadArbol()">
                  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="14" height="14" :class="{ 'spin': loadingArbol }">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                  </svg>
                  {{ loadingArbol ? 'Cargando...' : 'Actualizar' }}
                </button>
              </div>

              <!-- Buscador -->
              <div class="arbol-search-row">
                <input
                  v-model="searchArbol"
                  type="text"
                  class="arbol-search-input"
                  placeholder="Buscar por nombre, ruta o ID..."
                  @input="arbolPage = 1"
                />
              </div>

              <div class="arbol-panel__body">
                <div v-if="loadingArbol" class="arbol-empty">Cargando...</div>
                <div v-else-if="!paradaArbolRows.length" class="arbol-empty">
                  Sin árbol de paradas configurado para esta línea.<br/>
                  <span style="font-size:0.75rem;color:#9ca3af">Importa el árbol de paradas en la sección Árbol de Paradas.</span>
                </div>
                <div v-else-if="searchArbol && !paradaArbolFiltered.length" class="arbol-empty">
                  Sin resultados para "{{ searchArbol }}"
                </div>
                <template v-else>
                  <template v-for="(node, i) in paradaArbolVisible" :key="i">
                    <!-- Cabecera de categoría (solo sin búsqueda) -->
                    <div v-if="node.type === 'cat'" class="ap-cat-header">
                      {{ node.label }}
                    </div>
                    <!-- Subcategoría (solo sin búsqueda) -->
                    <div v-else-if="node.type === 'sub'" class="ap-sub-header">
                      {{ node.label }}
                    </div>
                    <!-- Hoja con ID real de BD -->
                    <div v-else class="ap-leaf-card">
                      <span class="ap-leaf-id">#{{ node.dbId }}</span>
                      <div class="ap-leaf-info">
                        <span v-if="searchArbol" class="ap-leaf-path">{{ node.fullPath }}</span>
                        <span class="ap-leaf-name">{{ node.label }}</span>
                        <span v-if="node.desc" class="ap-leaf-desc">{{ node.desc }}</span>
                      </div>
                    </div>
                  </template>

                  <!-- Carga progresiva -->
                  <div v-if="paradaArbolHasMore" class="arbol-load-more">
                    <button type="button" class="arbol-load-btn" @click="arbolPage++">
                      Mostrar más ({{ paradaArbolFiltered.length - arbolPage * ARBOL_PAGE > 0 ? paradaArbolFiltered.length - arbolPage * ARBOL_PAGE : '' }} restantes)
                    </button>
                  </div>
                </template>
              </div>
            </div>
          </template>

          <!-- ── Variables estándar (ENERGIA, PRODUCCION, etc.) ── -->
          <template v-else>
            <div class="form-grid">
              <div class="form-row" style="grid-column: 1 / -1">
                <label class="form-label">Nombre</label>
                <input v-model="editForm.nombre" type="text" class="field-input" required />
                <span class="field-hint-info">Se usa también como identificador corto</span>
              </div>

              <div class="form-row">
                <label class="form-label">Unidad</label>
                <input v-model="editForm.unidad" type="text" class="field-input" />
              </div>
            </div>

            <div class="form-section">
              <label class="section-label">Estado</label>
              <div class="button-group">
                <button type="button" :class="['option-btn', { active: editForm.estado === 'Activado' }]" @click="editForm.estado = 'Activado'">Activado</button>
                <button type="button" :class="['option-btn', { active: editForm.estado === 'No almacenado' }]" @click="editForm.estado = 'No almacenado'">No almacenado</button>
                <button type="button" :class="['option-btn', { active: editForm.estado === 'Deshabilitado' }]" @click="editForm.estado = 'Deshabilitado'">Deshabilitado</button>
              </div>
            </div>

            <!-- ── Catálogo de valores (solo cuando hay línea filtrada) ── -->
            <div v-if="lineaFiltro" class="form-section catalogo-section">
              <div class="section-header">
                <label class="section-label">
                  VALORES PERMITIDOS — <span class="catalogo-var-name">{{ editForm.nombre }}</span>
                  <span class="catalogo-hint">Cada variable tiene su propio catálogo por línea · Dropdown en Excel al descargar plantilla</span>
                </label>
                <span v-if="catalogoGuardadoOk" class="catalogo-saved">✓ Guardado</span>
              </div>

              <div v-if="catalogoEditando.length === 0" class="catalogo-empty">
                Sin valores definidos — texto libre al importar
              </div>
              <div v-else class="catalogo-chips">
                <div v-for="(val, idx) in catalogoEditando" :key="idx" class="catalogo-chip">
                  <span>{{ val }}</span>
                  <button type="button" class="chip-remove" @click="catalogoEditando.splice(idx, 1)">×</button>
                </div>
              </div>

              <div class="catalogo-add-row">
                <input
                  v-model="catalogoNuevoValor"
                  type="text"
                  class="field-input"
                  placeholder="Nuevo valor (Enter para agregar)"
                  @keydown.enter.prevent="agregarCatalogoValor"
                />
                <button type="button" class="add-btn" @click="agregarCatalogoValor">+</button>
              </div>

              <button
                type="button"
                class="catalogo-save-btn"
                :disabled="guardandoCatalogo"
                @click="guardarCatalogo"
              >
                {{ guardandoCatalogo ? 'Guardando...' : 'Guardar valores de catálogo' }}
              </button>
            </div>
          </template>

        </div>

        <div class="form-footer">
          <Button type="submit" variant="primary" size="md" :loading="loading">GUARDAR</Button>
          <Button type="button" @click="closeEditModal" variant="secondary" size="md">CANCELAR</Button>
        </div>
      </form>
    </Modal>
  </div>
</template>

<style scoped>
.variables-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.filters-section {
  background-color: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  padding: 1.5rem;
}

.filter-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: 1rem;
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.field-select {
  padding: 0.625rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
  cursor: pointer;
}

.field-select:hover {
  border-color: #9ca3af;
}

.field-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.content-card {
  border-radius: 0;
  border-top: none;
}

.tabs-header {
  display: flex;
  border-bottom: 2px solid #e5e7eb;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  background: none;
  border: none;
  border-bottom: 3px solid transparent;
  font-size: 0.875rem;
  font-weight: 600;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: #1e40af;
  background-color: #f0f9ff;
}

.tab-btn.active {
  color: #1e40af;
  border-bottom-color: #1e40af;
}

.tab-btn-wip {
  opacity: 0.45;
  cursor: not-allowed !important;
  pointer-events: none;
}

.tab-wip-badge {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  background: rgba(234, 179, 8, 0.15);
  color: #92400e;
  border: 1px solid rgba(234, 179, 8, 0.35);
  letter-spacing: 0.02em;
  margin-left: 0.25rem;
}

.action-bar {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  background-color: #1e3a8a;
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  font-size: 0.875rem;
}

.data-table thead {
  background-color: #f3f4f6;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid #e5e7eb;
}

.data-table tbody tr {
  transition: all 0.15s;
  cursor: pointer;
}

.data-table tbody tr:hover {
  background-color: #eff6ff;
}

.data-table tbody tr:nth-child(even) {
  background-color: #f9fafb;
}

.data-table tbody tr:nth-child(even):hover {
  background-color: #eff6ff;
}

.data-table tbody tr.row-selected {
  background-color: #dbeafe;
  border-left: 4px solid #2563eb;
}

.data-table td {
  padding: 0.75rem 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
}

.data-table td.table-name {
  font-weight: 600;
  color: #111827;
}

.checkbox-cell {
  width: 3rem;
  text-align: center;
}

.table-checkbox {
  width: 1.25rem;
  height: 1.25rem;
  color: #2563eb;
  border-color: #d1d5db;
  border-radius: 0.25rem;
  cursor: pointer;
}

.table-checkbox:focus {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 1rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.pagination-btn {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  background-color: white;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.pagination-btn:hover {
  background-color: #f9fafb;
}

.pagination-info {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.form-container {
  padding: 0;
}

.form-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1.5rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1rem;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.field-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
}

.field-input:hover {
  border-color: #9ca3af;
}

.field-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.field-hint {
  font-size: 0.75rem;
  color: #d97706;
  background-color: #fef3c7;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
}

.form-section {
  padding: 1rem 0;
  border-top: 1px solid #e5e7eb;
}

.section-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.75rem;
}

.button-group {
  display: flex;
  gap: 0.5rem;
}

.option-btn {
  padding: 0.5rem 1.5rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background-color: #f3f4f6;
  color: #374151;
  cursor: pointer;
  transition: all 0.2s;
}

.option-btn:hover {
  background-color: #e5e7eb;
}

.option-btn.active {
  background-color: #dbeafe;
  border-color: #3b82f6;
  color: #1e40af;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.add-btn {
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: white;
  border: 2px solid #1e40af;
  border-radius: 0.375rem;
  font-size: 1.5rem;
  color: #1e40af;
  cursor: pointer;
  transition: all 0.2s;
}

.add-btn:hover {
  background-color: #1e40af;
  color: white;
}

.form-footer {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  padding: 1rem 1.5rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.form-footer button {
  min-width: 120px;
}

/* Context banner */
.ctx-banner {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 0.5rem;
}
.ctx-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 600;
}
.ctx-chip--device { background: #dbeafe; color: #1e40af; }
.ctx-chip--linea  { background: #e0f2fe; color: #0369a1; }
.ctx-chip--tipo   { background: #fef3c7; color: #92400e; }
.ctx-icon { width: 0.9rem; height: 0.9rem; }

/* Readonly field */
.field-readonly {
  background: #f9fafb;
  color: #6b7280;
  cursor: not-allowed;
  font-family: monospace;
  font-size: 0.8rem;
}

/* Color row */
.color-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}
.color-picker {
  width: 2.5rem;
  height: 2.5rem;
  padding: 0.125rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  cursor: pointer;
  flex-shrink: 0;
}
.color-text { flex: 1; }

/* Hint info (blue) */
.field-hint-info {
  font-size: 0.72rem;
  color: #3b82f6;
}

/* Árbol panel */
.arbol-panel {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  overflow: hidden;
}
.arbol-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  background: #1e3a8a;
  font-size: 0.78rem;
  font-weight: 700;
  color: #fff;
  letter-spacing: 0.04em;
}
.arbol-panel__title {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.arbol-count {
  font-size: 0.7rem;
  font-weight: 600;
  background: rgba(255,255,255,0.2);
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  letter-spacing: 0;
}
.arbol-refresh-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.7rem;
  font-weight: 600;
  color: #1e3a8a;
  background: #fff;
  border: none;
  border-radius: 0.375rem;
  padding: 0.2rem 0.6rem;
  cursor: pointer;
  transition: opacity 0.15s;
}
.arbol-refresh-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.arbol-refresh-btn:hover:not(:disabled) { background: #dbeafe; }
.arbol-search-row {
  padding: 0.4rem 0.5rem;
  background: #f1f5f9;
  border-bottom: 1px solid #e2e8f0;
}
.arbol-search-input {
  width: 100%;
  font-size: 0.78rem;
  padding: 0.3rem 0.6rem;
  border: 1px solid #cbd5e1;
  border-radius: 0.375rem;
  outline: none;
  background: #fff;
}
.arbol-search-input:focus { border-color: #3b82f6; }
.arbol-load-more {
  display: flex;
  justify-content: center;
  padding: 0.5rem 0;
}
.arbol-load-btn {
  font-size: 0.75rem;
  font-weight: 600;
  color: #1d4ed8;
  background: #dbeafe;
  border: 1px solid #93c5fd;
  border-radius: 0.375rem;
  padding: 0.3rem 1.2rem;
  cursor: pointer;
}
.arbol-load-btn:hover { background: #bfdbfe; }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.arbol-panel__body {
  padding: 0.5rem;
  background: #f8fafc;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.arbol-empty {
  padding: 1rem 1.25rem;
  font-size: 0.82rem;
  color: #6b7280;
  text-align: center;
}
/* Cabecera categoría */
.ap-cat-header {
  padding: 0.3rem 0.75rem;
  background: #1e3a8a;
  color: #fff;
  font-size: 0.75rem;
  font-weight: 700;
  border-radius: 0.25rem;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  margin-top: 0.25rem;
}
/* Subcategoría */
.ap-sub-header {
  padding: 0.25rem 0.75rem;
  background: #dbeafe;
  color: #1e40af;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 0.25rem;
  margin-left: 0.5rem;
}
/* Tarjeta hoja */
.ap-leaf-card {
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
  padding: 0.4rem 0.75rem;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  margin-left: 1rem;
  transition: background 0.15s;
}
.ap-leaf-card:hover {
  background: #eff6ff;
}
.ap-leaf-id {
  flex-shrink: 0;
  font-size: 0.68rem;
  font-weight: 700;
  color: #fff;
  background: #3b82f6;
  padding: 0.1rem 0.45rem;
  border-radius: 999px;
  text-align: center;
  white-space: nowrap;
}
.ap-leaf-path {
  font-size: 0.68rem;
  color: #6b7280;
  font-style: italic;
  word-break: break-word;
}
.ap-leaf-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}
.ap-leaf-name {
  font-size: 0.8rem;
  font-weight: 600;
  color: #111827;
  word-break: break-word;
}
.ap-leaf-desc {
  font-size: 0.72rem;
  color: #6b7280;
  word-break: break-word;
}

/* ── Catálogo de valores ── */
.catalogo-section {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 0.5rem;
  padding: 1rem;
  margin-top: 0.25rem;
}
.catalogo-hint {
  display: block;
  font-size: 0.7rem;
  font-weight: 400;
  color: #16a34a;
  margin-top: 0.15rem;
}
.catalogo-var-name {
  color: #15803d;
  font-style: italic;
}
.catalogo-saved {
  font-size: 0.75rem;
  font-weight: 700;
  color: #15803d;
  background: #dcfce7;
  padding: 0.15rem 0.6rem;
  border-radius: 999px;
}
.catalogo-empty {
  font-size: 0.78rem;
  color: #6b7280;
  font-style: italic;
  margin-bottom: 0.75rem;
}
.catalogo-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-bottom: 0.75rem;
}
.catalogo-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  background: #dcfce7;
  border: 1px solid #86efac;
  border-radius: 999px;
  padding: 0.2rem 0.6rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: #15803d;
}
.chip-remove {
  border: none;
  background: none;
  color: #16a34a;
  cursor: pointer;
  font-size: 1rem;
  line-height: 1;
  padding: 0;
  transition: color 0.15s;
}
.chip-remove:hover { color: #dc2626; }
.catalogo-add-row {
  display: flex;
  gap: 0.4rem;
  margin-bottom: 0.75rem;
}
.catalogo-save-btn {
  width: 100%;
  padding: 0.5rem;
  background: #16a34a;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.8rem;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s;
}
.catalogo-save-btn:hover:not(:disabled) { background: #15803d; }
.catalogo-save-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
