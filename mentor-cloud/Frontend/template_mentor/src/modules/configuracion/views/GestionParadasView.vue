<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import * as XLSX from 'xlsx'

// ============================================
// 📦 DATA MOCK - Categorías y Etiquetas
// ============================================
// TODO: Reemplazar con llamada a API: GET /api/categorias
const categoriasMock = ref([
  { idCategoria: 1, nombreCategoria: 'CATEGORÍA_GENERAL', idEtiqueta: 1, etiqueta: 'parada programada' },
  { idCategoria: 1, nombreCategoria: 'CATEGORÍA_GENERAL', idEtiqueta: 2, etiqueta: 'parada no programada' },
  { idCategoria: 2, nombreCategoria: 'CATEGORÍA_DE_PÉRDIDA', idEtiqueta: 3, etiqueta: 'CAMBIOS' },
  { idCategoria: 2, nombreCategoria: 'CATEGORÍA_DE_PÉRDIDA', idEtiqueta: 4, etiqueta: 'SANEAMIENTOS' },
  { idCategoria: 2, nombreCategoria: 'CATEGORÍA_DE_PÉRDIDA', idEtiqueta: 5, etiqueta: 'CHARLAS Y REUNIONES' },
  { idCategoria: 2, nombreCategoria: 'CATEGORÍA_DE_PÉRDIDA', idEtiqueta: 6, etiqueta: 'PARADA PROGRAMADA' },
  { idCategoria: 3, nombreCategoria: 'CATEGORÍA_DE_PARADA', idEtiqueta: 7, etiqueta: 'PARADAS OBLIGATORIA' },
  { idCategoria: 3, nombreCategoria: 'CATEGORÍA_DE_PARADA', idEtiqueta: 8, etiqueta: 'PARADA DE MAQUINAS' },
])

// ============================================
// 📦 DATA MOCK - Paradas con estructura de 4 niveles
// ============================================
// TODO: Reemplazar con llamada a API: GET /api/paradas?lineaId={id}
const paradasMock = ref([
  {
    id: 1,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 7,
    nivel1: 'PARADAS OBLIGATORIA',
    nivel1Id: 101,
    nivel2: 'ALMUERZO/CENA',
    nivel2Id: 201,
    nivel3: '',
    nivel3Id: null,
    nivel4: '',
    nivel4Id: null
  },
  {
    id: 2,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 7,
    nivel1: 'PARADAS OBLIGATORIA',
    nivel1Id: 101,
    nivel2: 'CAPACITACION PERSONAL OBLIGATORIA',
    nivel2Id: 202,
    nivel3: '',
    nivel3Id: null,
    nivel4: '',
    nivel4Id: null
  },
  {
    id: 3,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 7,
    nivel1: 'PARADAS OBLIGATORIA',
    nivel1Id: 101,
    nivel2: 'CHARLA DE 5 MIN',
    nivel2Id: 203,
    nivel3: '',
    nivel3Id: null,
    nivel4: '',
    nivel4Id: null
  },
  {
    id: 4,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 7,
    nivel1: 'PARADAS OBLIGATORIA',
    nivel1Id: 101,
    nivel2: 'INSPECCION DE CALIDAD',
    nivel2Id: 204,
    nivel3: '',
    nivel3Id: null,
    nivel4: '',
    nivel4Id: null
  },
  {
    id: 5,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 8,
    nivel1: 'PARADA DE MAQUINAS',
    nivel1Id: 102,
    nivel2: 'CORTADORA',
    nivel2Id: 205,
    nivel3: 'CAMBIO DE CUCHILLA',
    nivel3Id: 301,
    nivel4: 'CUCHILLA A',
    nivel4Id: 401
  },
  {
    id: 6,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 8,
    nivel1: 'PARADA DE MAQUINAS',
    nivel1Id: 102,
    nivel2: 'CORTADORA',
    nivel2Id: 205,
    nivel3: 'CAMBIO DE CUCHILLA',
    nivel3Id: 301,
    nivel4: 'CUCHILLA B',
    nivel4Id: 402
  },
  {
    id: 7,
    nombreVariable: 'PLC-L1-Tipo de Parada Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 3,
    etiquetaId: 8,
    nivel1: 'PARADA DE MAQUINAS',
    nivel1Id: 102,
    nivel2: 'MAQUINA DE COSTURA',
    nivel2Id: 206,
    nivel3: 'AJUSTE DE ETIQUETAS',
    nivel3Id: 302,
    nivel4: '',
    nivel4Id: null
  },
  {
    id: 8,
    nombreVariable: 'PLC-L1-Tipo de Parada No Programada',
    linea: 'TEXTIL LINEA 1',
    categoriaId: 1,
    etiquetaId: 2,
    nivel1: 'FALLAS MECANICAS',
    nivel1Id: 103,
    nivel2: 'MOTOR',
    nivel2Id: 207,
    nivel3: 'SOBRECALENTAMIENTO',
    nivel3Id: 303,
    nivel4: '',
    nivel4Id: null
  }
])

// ============================================
// 🎛️ ESTADO DE LA VISTA
// ============================================
const loading = ref(false)
const saving = ref(false)
const uploadLoading = ref(false)
const uploadSuccess = ref(false)
const uploadError = ref('')
const saveSuccess = ref(false)

// Filtros
const lineaFiltro = ref('TEXTIL LINEA 1')
const categoriaFiltro = ref('')
const etiquetaFiltro = ref('')
const busqueda = ref('')

// Líneas disponibles (mock)
const lineasDisponibles = ref([
  { id: 1, nombre: 'TEXTIL LINEA 1' },
  { id: 2, nombre: 'Línea 2' },
  { id: 3, nombre: 'Línea 3' }
])

// Panel de detalle
const paradaSeleccionada = ref(null)
const mostrarPanelDetalle = ref(false)

// Input file reference
const fileInput = ref(null)

// Modo de edición
const editandoFila = ref(null)

// ============================================
// 📊 COMPUTED - Datos filtrados
// ============================================
const paradasFiltradas = computed(() => {
  let resultado = paradasMock.value

  // Filtrar por línea
  if (lineaFiltro.value) {
    resultado = resultado.filter(p => p.linea === lineaFiltro.value)
  }

  // Filtrar por categoría
  if (categoriaFiltro.value) {
    resultado = resultado.filter(p => {
      const cat = categoriasMock.value.find(c => c.idCategoria === p.categoriaId)
      return cat && cat.nombreCategoria === categoriaFiltro.value
    })
  }

  // Filtrar por etiqueta
  if (etiquetaFiltro.value) {
    resultado = resultado.filter(p => {
      const etiq = categoriasMock.value.find(c => c.idEtiqueta === p.etiquetaId)
      return etiq && etiq.etiqueta === etiquetaFiltro.value
    })
  }

  // Buscar por texto
  if (busqueda.value.trim()) {
    const texto = busqueda.value.toLowerCase()
    resultado = resultado.filter(p => 
      p.nombreVariable.toLowerCase().includes(texto) ||
      p.nivel1.toLowerCase().includes(texto) ||
      p.nivel2.toLowerCase().includes(texto) ||
      p.nivel3.toLowerCase().includes(texto) ||
      p.nivel4.toLowerCase().includes(texto)
    )
  }

  return resultado
})

const categoriasUnicas = computed(() => {
  const nombres = new Set(categoriasMock.value.map(c => c.nombreCategoria))
  return Array.from(nombres)
})

const etiquetasUnicas = computed(() => {
  let etiquetas = categoriasMock.value
  
  if (categoriaFiltro.value) {
    etiquetas = etiquetas.filter(c => c.nombreCategoria === categoriaFiltro.value)
  }
  
  return etiquetas.map(c => c.etiqueta)
})

// ============================================
// 🔧 FUNCIONES - Acciones
// ============================================
const seleccionarParada = (parada) => {
  paradaSeleccionada.value = { ...parada }
  mostrarPanelDetalle.value = true
}

const cerrarPanelDetalle = () => {
  mostrarPanelDetalle.value = false
  paradaSeleccionada.value = null
}

const guardarCambios = async () => {
  saving.value = true
  saveSuccess.value = false
  
  try {
    // TODO: Llamada a API para guardar cambios
    // await axios.put('/api/paradas/bulk', paradasMock.value)
    
    // Simular delay
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    saveSuccess.value = true
    setTimeout(() => {
      saveSuccess.value = false
    }, 3000)
  } catch (err) {
    console.error('Error al guardar:', err)
  } finally {
    saving.value = false
  }
}

const aplicarCambiosDetalle = () => {
  if (!paradaSeleccionada.value) return
  
  // Actualizar en el array principal
  const index = paradasMock.value.findIndex(p => p.id === paradaSeleccionada.value.id)
  if (index !== -1) {
    paradasMock.value[index] = { ...paradaSeleccionada.value }
  }
  
  cerrarPanelDetalle()
}

const editarFila = (paradaId) => {
  editandoFila.value = paradaId
}

const cancelarEdicionFila = () => {
  editandoFila.value = null
}

const guardarEdicionFila = (paradaId) => {
  editandoFila.value = null
  // Los cambios ya están en el objeto reactivo
}

// ============================================
// 📤 IMPORTAR/EXPORTAR EXCEL
// ============================================
const triggerFileInput = () => {
  fileInput.value.click()
}

const handleFileUpload = async (event) => {
  const file = event.target.files[0]
  if (!file) return

  uploadLoading.value = true
  uploadError.value = ''
  uploadSuccess.value = false

  try {
    const data = await file.arrayBuffer()
    const workbook = XLSX.read(data)
    
    // Buscar la primera hoja que contenga datos de paradas.
    const sheetName = workbook.SheetNames[0]
    const worksheet = workbook.Sheets[sheetName]
    
    if (!worksheet) {
      throw new Error('No se encontró ninguna hoja válida en el archivo')
    }

    const rows = XLSX.utils.sheet_to_json(worksheet)
    
    // Procesar filas del Excel
    const nuevasParadas = rows.map((row, index) => ({
      id: paradasMock.value.length + index + 1,
      nombreVariable: row['nombre dela variable'] || row['Nombre Variable'] || '',
      linea: lineaFiltro.value,
      categoriaId: parseInt(row['categoriaId']) || 1,
      etiquetaId: parseInt(row['etiquetaId']) || 1,
      nivel1: row['nivel 1'] || row['Nivel 1'] || '',
      nivel1Id: parseInt(row['id']) || null,
      nivel2: row['nivel 2'] || row['Nivel 2'] || '',
      nivel2Id: parseInt(row['id.1']) || null,
      nivel3: row['nivel 3'] || row['Nivel 3'] || '',
      nivel3Id: parseInt(row['id.2']) || null,
      nivel4: row['nivel 4'] || row['Nivel 4'] || '',
      nivel4Id: parseInt(row['id.3']) || null
    }))

    // Agregar las nuevas paradas
    paradasMock.value = [...paradasMock.value, ...nuevasParadas]
    
    uploadSuccess.value = true
    setTimeout(() => {
      uploadSuccess.value = false
    }, 3000)

  } catch (err) {
    uploadError.value = err.message || 'Error al procesar el archivo Excel'
  } finally {
    uploadLoading.value = false
    event.target.value = ''
  }
}

const exportarExcel = () => {
  const workbook = XLSX.utils.book_new()

  // Preparar datos para exportar
  const dataExport = paradasFiltradas.value.map(p => ({
    'nombre dela variable': p.nombreVariable,
    'id': p.nivel1Id || '',
    'nivel 1': p.nivel1,
    'id.1': p.nivel2Id || '',
    'nivel 2': p.nivel2,
    'id.2': p.nivel3Id || '',
    'nivel 3': p.nivel3,
    'id.3': p.nivel4Id || '',
    'nivel 4': p.nivel4,
    'categoriaId': p.categoriaId,
    'etiquetaId': p.etiquetaId,
    'linea': p.linea
  }))

  const worksheet = XLSX.utils.json_to_sheet(dataExport)
  XLSX.utils.book_append_sheet(workbook, worksheet, lineaFiltro.value || 'Paradas')

  // Exportar hoja de categorías
  const categoriasExport = categoriasMock.value.map(c => ({
    'Id Categoria': c.idCategoria,
    'nombre de categoria': c.nombreCategoria,
    'Id etiqueta': c.idEtiqueta,
    'Etiqueta': c.etiqueta
  }))

  const wsCategorias = XLSX.utils.json_to_sheet(categoriasExport)
  XLSX.utils.book_append_sheet(workbook, wsCategorias, 'categorias')

  XLSX.writeFile(workbook, `gestion-paradas-${lineaFiltro.value || 'todas'}.xlsx`)
}

const descargarPlantilla = () => {
  const workbook = XLSX.utils.book_new()

  const plantillaData = [
    {
      'nombre dela variable': 'PLC-L1-Tipo de Parada Programada',
      'id': 101,
      'nivel 1': 'PARADAS OBLIGATORIA',
      'id.1': 201,
      'nivel 2': 'ALMUERZO/CENA',
      'id.2': '',
      'nivel 3': '',
      'id.3': '',
      'nivel 4': '',
      'categoriaId': 3,
      'etiquetaId': 7,
      'linea': 'TEXTIL LINEA 1'
    },
    {
      'nombre dela variable': 'PLC-L1-Tipo de Parada Programada',
      'id': 102,
      'nivel 1': 'PARADA DE MAQUINAS',
      'id.1': 205,
      'nivel 2': 'CORTADORA',
      'id.2': 301,
      'nivel 3': 'CAMBIO DE CUCHILLA',
      'id.3': 401,
      'nivel 4': 'CUCHILLA A',
      'categoriaId': 3,
      'etiquetaId': 8,
      'linea': 'TEXTIL LINEA 1'
    }
  ]

  const ws = XLSX.utils.json_to_sheet(plantillaData)
  XLSX.utils.book_append_sheet(workbook, ws, 'TEXTIL LINEA 1')

  // Agregar hoja de categorías
  const categoriasData = [
    { 'Id Categoria': 1, 'nombre de categoria': 'CATEGORÍA_GENERAL', 'Id etiqueta': 1, 'Etiqueta': 'parada programada' },
    { 'Id Categoria': 1, 'nombre de categoria': 'CATEGORÍA_GENERAL', 'Id etiqueta': 2, 'Etiqueta': 'parada no programada' },
    { 'Id Categoria': 2, 'nombre de categoria': 'CATEGORÍA_DE_PÉRDIDA', 'Id etiqueta': 3, 'Etiqueta': 'CAMBIOS' },
    { 'Id Categoria': 3, 'nombre de categoria': 'CATEGORÍA_DE_PARADA', 'Id etiqueta': 7, 'Etiqueta': 'PARADAS OBLIGATORIA' }
  ]

  const wsCat = XLSX.utils.json_to_sheet(categoriasData)
  XLSX.utils.book_append_sheet(workbook, wsCat, 'categorias')

  XLSX.writeFile(workbook, 'plantilla-gestion-paradas.xlsx')
}

const limpiarFiltros = () => {
  categoriaFiltro.value = ''
  etiquetaFiltro.value = ''
  busqueda.value = ''
}

// ============================================
// 🚀 LIFECYCLE
// ============================================
onMounted(() => {
  // TODO: Cargar datos desde API
  // loadCategorias()
  // loadParadas()
})
</script>

<template>
  <div class="gestion-paradas-view">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <div>
          <h1 class="page-title">Gestión de Paradas</h1>
          <p class="page-subtitle">Configuración de categorías y niveles para las líneas de producción</p>
        </div>
        <div class="header-actions">
          <input 
            ref="fileInput" 
            type="file" 
            accept=".xlsx,.xls" 
            @change="handleFileUpload" 
            style="display: none"
          />
          <Button variant="success" size="sm" @click="descargarPlantilla">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            Descargar Plantilla
          </Button>
          <Button variant="warning" size="sm" @click="triggerFileInput" :disabled="uploadLoading">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
            </svg>
            {{ uploadLoading ? 'Importando...' : 'Importar desde Excel' }}
          </Button>
          <Button variant="primary" size="sm" @click="guardarCambios" :disabled="saving">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
            </svg>
            {{ saving ? 'Guardando...' : 'Guardar Cambios' }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Alertas -->
    <Alert v-if="uploadSuccess" type="success" message="✓ Datos importados correctamente desde Excel" class="mb-4" />
    <Alert v-if="uploadError" type="error" :message="uploadError" class="mb-4" />
    <Alert v-if="saveSuccess" type="success" message="✓ Cambios guardados correctamente" class="mb-4" />

    <!-- Filtros -->
    <Card class="filters-card">
      <div class="filters-header">
        <h3 class="filters-title">
          <svg class="w-5 h-5 inline-block mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"/>
          </svg>
          Filtros
        </h3>
        <Button variant="secondary" size="xs" @click="limpiarFiltros">
          Limpiar filtros
        </Button>
      </div>
      
      <div class="filters-grid">
        <div class="filter-item">
          <label class="filter-label">Línea</label>
          <select v-model="lineaFiltro" class="filter-select">
            <option value="">Todas las líneas</option>
            <option v-for="linea in lineasDisponibles" :key="linea.id" :value="linea.nombre">
              {{ linea.nombre }}
            </option>
          </select>
        </div>

        <div class="filter-item">
          <label class="filter-label">Categoría</label>
          <select v-model="categoriaFiltro" class="filter-select">
            <option value="">Todas las categorías</option>
            <option v-for="cat in categoriasUnicas" :key="cat" :value="cat">
              {{ cat }}
            </option>
          </select>
        </div>

        <div class="filter-item">
          <label class="filter-label">Etiqueta</label>
          <select v-model="etiquetaFiltro" class="filter-select">
            <option value="">Todas las etiquetas</option>
            <option v-for="etiq in etiquetasUnicas" :key="etiq" :value="etiq">
              {{ etiq }}
            </option>
          </select>
        </div>

        <div class="filter-item">
          <label class="filter-label">Búsqueda</label>
          <div class="search-input-wrapper">
            <svg class="search-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
            <input 
              v-model="busqueda" 
              type="text" 
              placeholder="Buscar por nombre o niveles..." 
              class="filter-search"
            />
          </div>
        </div>
      </div>

      <div class="filters-summary">
        <span class="summary-badge">{{ paradasFiltradas.length }} registros encontrados</span>
      </div>
    </Card>

    <!-- Tabla Principal y Panel de Detalle -->
    <div class="content-layout">
      <!-- Tabla de Paradas -->
      <Card class="table-card">
        <div class="table-header">
          <h3 class="table-title">Paradas Configuradas</h3>
          <Button variant="info" size="sm" @click="exportarExcel">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            Exportar Excel
          </Button>
        </div>

        <Loading v-if="loading" />
        
        <div v-else class="table-wrapper">
          <table class="paradas-table">
            <thead>
              <tr>
                <th style="width: 40px;">#</th>
                <th style="min-width: 250px;">Nombre de la Variable</th>
                <th style="min-width: 180px;">Nivel 1</th>
                <th style="min-width: 180px;">Nivel 2</th>
                <th style="min-width: 180px;">Nivel 3</th>
                <th style="min-width: 180px;">Nivel 4</th>
                <th style="width: 120px;">Acciones</th>
              </tr>
            </thead>
            <tbody>
              <tr 
                v-for="(parada, index) in paradasFiltradas" 
                :key="parada.id"
                :class="{ 'row-selected': paradaSeleccionada?.id === parada.id }"
                @click="seleccionarParada(parada)"
              >
                <td class="text-center">{{ index + 1 }}</td>
                <td>
                  <div class="variable-name">{{ parada.nombreVariable }}</div>
                  <div class="variable-meta">{{ parada.linea }}</div>
                </td>
                <td>
                  <input 
                    v-if="editandoFila === parada.id"
                    v-model="parada.nivel1"
                    class="edit-input"
                    @click.stop
                  />
                  <span v-else class="nivel-badge nivel-1">{{ parada.nivel1 }}</span>
                </td>
                <td>
                  <input 
                    v-if="editandoFila === parada.id"
                    v-model="parada.nivel2"
                    class="edit-input"
                    @click.stop
                  />
                  <span v-else class="nivel-badge nivel-2">{{ parada.nivel2 }}</span>
                </td>
                <td>
                  <input 
                    v-if="editandoFila === parada.id"
                    v-model="parada.nivel3"
                    class="edit-input"
                    placeholder="(vacío)"
                    @click.stop
                  />
                  <span v-else class="nivel-badge nivel-3">{{ parada.nivel3 || '—' }}</span>
                </td>
                <td>
                  <input 
                    v-if="editandoFila === parada.id"
                    v-model="parada.nivel4"
                    class="edit-input"
                    placeholder="(vacío)"
                    @click.stop
                  />
                  <span v-else class="nivel-badge nivel-4">{{ parada.nivel4 || '—' }}</span>
                </td>
                <td @click.stop>
                  <div class="action-buttons">
                    <template v-if="editandoFila === parada.id">
                      <button @click="guardarEdicionFila(parada.id)" class="btn-action btn-save" title="Guardar">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                        </svg>
                      </button>
                      <button @click="cancelarEdicionFila" class="btn-action btn-cancel" title="Cancelar">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                        </svg>
                      </button>
                    </template>
                    <template v-else>
                      <button @click="editarFila(parada.id)" class="btn-action btn-edit" title="Editar">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                        </svg>
                      </button>
                    </template>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>

          <div v-if="paradasFiltradas.length === 0" class="empty-state">
            <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
            </svg>
            <p class="empty-text">No se encontraron paradas con los filtros aplicados</p>
          </div>
        </div>
      </Card>

      <!-- Panel de Detalle (lateral) -->
      <Transition name="slide">
        <Card v-if="mostrarPanelDetalle && paradaSeleccionada" class="detail-panel">
          <div class="detail-header">
            <h3 class="detail-title">Detalle de Parada</h3>
            <button @click="cerrarPanelDetalle" class="btn-close">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div class="detail-body">
            <div class="detail-section">
              <label class="detail-label">Nombre de la Variable</label>
              <input v-model="paradaSeleccionada.nombreVariable" class="detail-input" />
            </div>

            <div class="detail-section">
              <label class="detail-label">Línea</label>
              <select v-model="paradaSeleccionada.linea" class="detail-input">
                <option v-for="linea in lineasDisponibles" :key="linea.id" :value="linea.nombre">
                  {{ linea.nombre }}
                </option>
              </select>
            </div>

            <div class="detail-section">
              <label class="detail-label">Categoría</label>
              <select v-model="paradaSeleccionada.categoriaId" class="detail-input">
                <option v-for="cat in categoriasUnicas" :key="cat" :value="categoriasMock.find(c => c.nombreCategoria === cat)?.idCategoria">
                  {{ cat }}
                </option>
              </select>
            </div>

            <div class="detail-section">
              <label class="detail-label">Etiqueta</label>
              <select v-model="paradaSeleccionada.etiquetaId" class="detail-input">
                <option v-for="etiq in categoriasMock" :key="etiq.idEtiqueta" :value="etiq.idEtiqueta">
                  {{ etiq.etiqueta }}
                </option>
              </select>
            </div>

            <div class="divider"></div>

            <div class="detail-section">
              <label class="detail-label">
                <span class="nivel-dot nivel-1"></span>
                Nivel 1
              </label>
              <input v-model="paradaSeleccionada.nivel1" class="detail-input" />
              <input v-model.number="paradaSeleccionada.nivel1Id" type="number" class="detail-input-small" placeholder="ID" />
            </div>

            <div class="detail-section">
              <label class="detail-label">
                <span class="nivel-dot nivel-2"></span>
                Nivel 2
              </label>
              <input v-model="paradaSeleccionada.nivel2" class="detail-input" />
              <input v-model.number="paradaSeleccionada.nivel2Id" type="number" class="detail-input-small" placeholder="ID" />
            </div>

            <div class="detail-section">
              <label class="detail-label">
                <span class="nivel-dot nivel-3"></span>
                Nivel 3
              </label>
              <input v-model="paradaSeleccionada.nivel3" class="detail-input" placeholder="(opcional)" />
              <input v-model.number="paradaSeleccionada.nivel3Id" type="number" class="detail-input-small" placeholder="ID" />
            </div>

            <div class="detail-section">
              <label class="detail-label">
                <span class="nivel-dot nivel-4"></span>
                Nivel 4
              </label>
              <input v-model="paradaSeleccionada.nivel4" class="detail-input" placeholder="(opcional)" />
              <input v-model.number="paradaSeleccionada.nivel4Id" type="number" class="detail-input-small" placeholder="ID" />
            </div>
          </div>

          <div class="detail-footer">
            <Button variant="secondary" size="sm" @click="cerrarPanelDetalle">
              Cancelar
            </Button>
            <Button variant="primary" size="sm" @click="aplicarCambiosDetalle">
              <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
              </svg>
              Aplicar Cambios
            </Button>
          </div>
        </Card>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.gestion-paradas-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding-bottom: 2rem;
}

/* ============================================
   HEADER
   ============================================ */
.page-header {
  background: linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%);
  padding: 2rem;
  border-radius: 0.75rem;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 2rem;
  flex-wrap: wrap;
}

.page-title {
  font-size: 2rem;
  font-weight: 700;
  color: white;
  margin: 0;
}

.page-subtitle {
  font-size: 1rem;
  color: rgba(255, 255, 255, 0.9);
  margin: 0.5rem 0 0 0;
}

.header-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

/* ============================================
   FILTROS
   ============================================ */
.filters-card {
  border: 1px solid #e5e7eb;
}

.filters-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 2px solid #e5e7eb;
}

.filters-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1.25rem;
  margin-bottom: 1rem;
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.filter-select,
.filter-search {
  padding: 0.625rem 0.875rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
}

.filter-select:hover,
.filter-search:hover {
  border-color: #9ca3af;
}

.filter-select:focus,
.filter-search:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-input-wrapper {
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.875rem;
  top: 50%;
  transform: translateY(-50%);
  width: 1.25rem;
  height: 1.25rem;
  color: #9ca3af;
  pointer-events: none;
}

.filter-search {
  padding-left: 2.75rem;
}

.filters-summary {
  display: flex;
  gap: 0.75rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #e5e7eb;
}

.summary-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.375rem 0.875rem;
  background-color: #dbeafe;
  color: #1e40af;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 600;
}

/* ============================================
   LAYOUT CONTENIDO
   ============================================ */
.content-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.5rem;
  position: relative;
}

.content-layout:has(.detail-panel) {
  grid-template-columns: 1fr 400px;
}

/* ============================================
   TABLA
   ============================================ */
.table-card {
  border: 1px solid #e5e7eb;
  min-width: 0;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  background-color: #f9fafb;
  border-bottom: 2px solid #e5e7eb;
}

.table-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.table-wrapper {
  overflow-x: auto;
}

.paradas-table {
  width: 100%;
  font-size: 0.875rem;
  border-collapse: collapse;
}

.paradas-table thead {
  background-color: #f3f4f6;
  position: sticky;
  top: 0;
  z-index: 10;
}

.paradas-table th {
  padding: 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 700;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 2px solid #e5e7eb;
  white-space: nowrap;
}

.paradas-table td {
  padding: 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
  vertical-align: middle;
}

.paradas-table tbody tr {
  cursor: pointer;
  transition: background-color 0.15s;
}

.paradas-table tbody tr:hover {
  background-color: #f9fafb;
}

.paradas-table tbody tr.row-selected {
  background-color: #eff6ff;
  border-left: 3px solid #3b82f6;
}

.variable-name {
  font-weight: 600;
  color: #111827;
  margin-bottom: 0.25rem;
}

.variable-meta {
  font-size: 0.75rem;
  color: #6b7280;
}

.nivel-badge {
  display: inline-block;
  padding: 0.375rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
}

.nivel-badge.nivel-1 {
  background-color: #dbeafe;
  color: #1e40af;
}

.nivel-badge.nivel-2 {
  background-color: #d1fae5;
  color: #065f46;
}

.nivel-badge.nivel-3 {
  background-color: #fef3c7;
  color: #92400e;
}

.nivel-badge.nivel-4 {
  background-color: #fce7f3;
  color: #9f1239;
}

.edit-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #3b82f6;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background-color: white;
}

.edit-input:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
  justify-content: center;
}

.btn-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-edit {
  background-color: #dbeafe;
  color: #1e40af;
}

.btn-edit:hover {
  background-color: #bfdbfe;
}

.btn-save {
  background-color: #d1fae5;
  color: #065f46;
}

.btn-save:hover {
  background-color: #a7f3d0;
}

.btn-cancel {
  background-color: #fee2e2;
  color: #991b1b;
}

.btn-cancel:hover {
  background-color: #fecaca;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
}

.empty-icon {
  width: 4rem;
  height: 4rem;
  color: #d1d5db;
  margin-bottom: 1rem;
}

.empty-text {
  font-size: 1rem;
  color: #6b7280;
  margin: 0;
}

/* ============================================
   PANEL DE DETALLE
   ============================================ */
.detail-panel {
  border: 1px solid #e5e7eb;
  height: fit-content;
  max-height: calc(100vh - 200px);
  position: sticky;
  top: 1rem;
  overflow-y: auto;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  background: linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%);
  border-radius: 0.5rem 0.5rem 0 0;
}

.detail-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: white;
  margin: 0;
}

.btn-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background-color: rgba(255, 255, 255, 0.2);
  color: white;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-close:hover {
  background-color: rgba(255, 255, 255, 0.3);
}

.detail-body {
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.detail-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.detail-input {
  padding: 0.625rem 0.875rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
}

.detail-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.detail-input-small {
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  color: #6b7280;
  background-color: #f9fafb;
  transition: all 0.2s;
  margin-top: 0.25rem;
}

.detail-input-small:focus {
  outline: none;
  border-color: #3b82f6;
  background-color: white;
}

.divider {
  height: 1px;
  background-color: #e5e7eb;
  margin: 0.5rem 0;
}

.nivel-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.nivel-dot.nivel-1 {
  background-color: #3b82f6;
}

.nivel-dot.nivel-2 {
  background-color: #10b981;
}

.nivel-dot.nivel-3 {
  background-color: #f59e0b;
}

.nivel-dot.nivel-4 {
  background-color: #ec4899;
}

.detail-footer {
  display: flex;
  gap: 0.75rem;
  padding: 1.25rem 1.5rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
  border-radius: 0 0 0.5rem 0.5rem;
}

/* ============================================
   TRANSICIONES
   ============================================ */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

/* ============================================
   RESPONSIVE
   ============================================ */
@media (max-width: 1400px) {
  .content-layout:has(.detail-panel) {
    grid-template-columns: 1fr 350px;
  }
}

@media (max-width: 1200px) {
  .content-layout:has(.detail-panel) {
    grid-template-columns: 1fr;
  }

  .detail-panel {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: 400px;
    max-height: 100vh;
    z-index: 1000;
    box-shadow: -4px 0 6px -1px rgba(0, 0, 0, 0.1);
  }
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
  }

  .filters-grid {
    grid-template-columns: 1fr;
  }

  .detail-panel {
    width: 100%;
  }
}
</style>
