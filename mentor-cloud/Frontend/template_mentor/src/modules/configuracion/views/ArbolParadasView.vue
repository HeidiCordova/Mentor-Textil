<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { arbolParadasService } from '@/api/services/arbolParadas.service'
import { useApi } from '@/shared/composables/useApi'
import * as XLSX from 'xlsx'
import ExcelJS from 'exceljs'

const { loading, error, execute } = useApi()
const fileInput = ref(null)
const uploadLoading = ref(false)
const successMsg = ref('')
const errorMsg = ref('')
const importSummary = ref(null)

const empresas = ref([])
const plantas = ref([])
const lineas = ref([])
const expandedCategories = ref([])

const empresaFiltro = ref('')
const plantaFiltro = ref('')
const lineaFiltro = ref('')

const programadas = ref([])
const noProgramadas = ref([])

const plantasFiltradas = computed(() => {
  if (!empresaFiltro.value) return []
  return plantas.value.filter(p => p.empresa_id === parseInt(empresaFiltro.value))
})

const lineasFiltradas = computed(() => {
  if (!plantaFiltro.value) return []
  return lineas.value.filter(l => l.planta_id === parseInt(plantaFiltro.value))
})

const loadCascade = async () => {
  await execute(async () => {
    const [empRes, plantRes, lineRes] = await Promise.all([
      companyService.getAll(),
      plantService.getAll(),
      lineService.getAll()
    ])
    empresas.value = Array.isArray(empRes) ? empRes : (empRes?.data || [])
    plantas.value = Array.isArray(plantRes) ? plantRes : (plantRes?.data || [])
    lineas.value = Array.isArray(lineRes) ? lineRes : (lineRes?.data || [])
  })
}

const loadArbol = async () => {
  if (!lineaFiltro.value) {
    programadas.value = []
    noProgramadas.value = []
    return
  }
  try {
    const res = await arbolParadasService.get(lineaFiltro.value)
    programadas.value = res.programadas || []
    noProgramadas.value = res.no_programadas || []
  } catch {
    programadas.value = []
    noProgramadas.value = []
  }
}

watch(empresaFiltro, () => {
  const opts = plantasFiltradas.value
  plantaFiltro.value = opts.length ? opts[0].id : ''
})
watch(plantaFiltro, () => {
  const opts = lineasFiltradas.value
  lineaFiltro.value = opts.length ? opts[0].id : ''
})
watch(lineaFiltro, loadArbol)

// Flat virtual tree: soporta hasta 7 niveles dinámicamente.
// Los campos por nivel en orden descendente:
const LEVEL_FIELDS = ['categoria', 'subcategoria', 'subcategoria_2', 'subcategoria_3', 'subcategoria_4', 'descripcion_parada', 'maquina']

const buildFlatNodes = (tipoKey, tipoLabel, rows) => {
  const nodes = []
  nodes.push({ key: tipoKey, parentKey: null, depth: 0, label: tipoLabel, isLeaf: false, childCount: rows.length })

  // Calcula cuántos niveles reales tiene un row (profundidad del último campo no vacío)
  const rowDepth = (row) => {
    let d = 0
    for (let i = 0; i < LEVEL_FIELDS.length; i++) {
      if (row[LEVEL_FIELDS[i]]) d = i + 1
    }
    return d
  }

  // Agrupa rows por el valor del campo en `levelIdx`, bajo una clave padre dada
  const groupByLevel = (parentKey, parentDepth, levelIdx, levelRows) => {
    const groups = new Map()
    for (const row of levelRows) {
      const val = row[LEVEL_FIELDS[levelIdx]] || ''
      if (!groups.has(val)) groups.set(val, [])
      groups.get(val).push(row)
    }

    for (const [label, groupRows] of groups) {
      const nodeKey = `${parentKey}::${label}`
      const depth = parentDepth + 1

      // ¿Algún row tiene datos más allá de este nivel?
      const hasChildren = groupRows.some(r => rowDepth(r) > levelIdx + 1)

      if (hasChildren && label) {
        nodes.push({ key: nodeKey, parentKey, depth, label, isLeaf: false, childCount: groupRows.length })
        // Recursión: siguiente nivel
        if (levelIdx + 1 < LEVEL_FIELDS.length) {
          groupByLevel(nodeKey, depth, levelIdx + 1, groupRows)
        }
      } else {
        // Son hojas en este nivel
        for (const row of groupRows) {
          nodes.push({ key: `leaf::${row.id}`, parentKey, depth, label, isLeaf: true, row })
        }
      }
    }
  }

  groupByLevel(tipoKey, 0, 0, rows)
  return nodes
}

const allNodes = computed(() => [
  ...buildFlatNodes('prog',   'Tipo de Parada Programada',    programadas.value),
  ...buildFlatNodes('noprog', 'Tipo de Parada No Programada', noProgramadas.value)
])

const nodeByKey = computed(() => {
  const m = new Map()
  for (const n of allNodes.value) m.set(n.key, n)
  return m
})

// Solo muestra nodos cuya cadena de ancestros esté completamente expandida
const visibleNodes = computed(() => {
  const exp = new Set(expandedCategories.value)
  return allNodes.value.filter(node => {
    if (!node.parentKey) return true
    let cur = node
    while (cur.parentKey) {
      if (!exp.has(cur.parentKey)) return false
      cur = nodeByKey.value.get(cur.parentKey)
      if (!cur) return false
    }
    return true
  })
})

const BRANCH_BG = ['#eef2ff', '#f0f9ff', '#e0f2fe', '#f0fdf4']

const toggleCategory = (id) => {
  const idx = expandedCategories.value.indexOf(id)
  if (idx > -1) expandedCategories.value.splice(idx, 1)
  else expandedCategories.value.push(id)
}
const isExpanded = (id) => expandedCategories.value.includes(id)

const triggerDownload = (buffer, filename) => {
  const blob = new Blob([buffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

const COLS = [
  { key: 'CATEGORIA',           header: 'CATEGORIA',          width: 28 },
  { key: 'SUBCATEGORIA',        header: 'SUBCATEGORIA',       width: 32 },
  { key: 'SUBCATEGORIA_2',      header: 'SUBCATEGORIA_2',     width: 32 },
  { key: 'SUBCATEGORIA_3',      header: 'SUBCATEGORIA_3',     width: 32 },
  { key: 'SUBCATEGORIA_4',      header: 'SUBCATEGORIA_4',     width: 32 },
  { key: 'DESCRIPCION_PARADA',  header: 'DESCRIPCION_PARADA', width: 40 },
  { key: 'MAQUINA',             header: 'MAQUINA',            width: 22 }
]

const buildStyledSheet = (wb, sheetName, rows, headerArgb, altArgb) => {
  const ws = wb.addWorksheet(sheetName, {
    views: [{ state: 'frozen', ySplit: 1 }],
    pageSetup: { orientation: 'landscape' }
  })

  ws.columns = COLS.map(c => ({ key: c.key, width: c.width }))

  const headerRow = ws.addRow(COLS.map(c => c.header))
  headerRow.height = 24
  for (let col = 1; col <= COLS.length; col++) {
    const cell = headerRow.getCell(col)
    cell.fill      = { type: 'pattern', pattern: 'solid', fgColor: { argb: headerArgb } }
    cell.font      = { color: { argb: 'FFFFFFFF' }, bold: true, size: 11, name: 'Calibri' }
    cell.alignment = { horizontal: 'center', vertical: 'middle', wrapText: false }
    cell.border    = { bottom: { style: 'medium', color: { argb: 'FFCCCCCC' } } }
  }

  rows.forEach((r, i) => {
    const dr = ws.addRow(COLS.map(c => r[c.key]))
    dr.height = 18
    const bg = i % 2 === 0 ? 'FFFFFFFF' : altArgb
    for (let col = 1; col <= COLS.length; col++) {
      const cell = dr.getCell(col)
      cell.fill      = { type: 'pattern', pattern: 'solid', fgColor: { argb: bg } }
      cell.font      = { size: 10, name: 'Calibri' }
      cell.alignment = { vertical: 'middle' }
      cell.border    = { bottom: { style: 'thin', color: { argb: 'FFE5E7EB' } } }
    }
  })
}

const mapRowExport = (r) => ({
  CATEGORIA:          r.categoria || '',
  SUBCATEGORIA:       r.subcategoria || '',
  SUBCATEGORIA_2:     r.subcategoria_2 || '',
  SUBCATEGORIA_3:     r.subcategoria_3 || '',
  SUBCATEGORIA_4:     r.subcategoria_4 || '',
  DESCRIPCION_PARADA: r.descripcion_parada || '',
  MAQUINA:            r.maquina || ''
})

const emptyRow = () => ({
  CATEGORIA: '', SUBCATEGORIA: '', SUBCATEGORIA_2: '', SUBCATEGORIA_3: '', SUBCATEGORIA_4: '',
  DESCRIPCION_PARADA: '', MAQUINA: ''
})

const downloadTemplate = async () => {
  errorMsg.value = ''
  let progData = []
  let noProgData = []

  if (lineaFiltro.value) {
    try {
      const res = await arbolParadasService.exportar(lineaFiltro.value)
      progData  = (res.programadas    || []).map(mapRowExport)
      noProgData = (res.no_programadas || []).map(mapRowExport)
    } catch (err) {
      errorMsg.value = 'Error al obtener datos para plantilla: ' + (err.message || '')
    }
  }

  if (!progData.length)   progData   = [emptyRow()]
  if (!noProgData.length) noProgData = [emptyRow()]

  const wb = new ExcelJS.Workbook()
  wb.creator = 'Mentor Monitor'
  buildStyledSheet(wb, 'Paradas Programadas',    progData,   'FF15803D', 'FFF0FDF4')
  buildStyledSheet(wb, 'Paradas No Programadas', noProgData, 'FFB45309', 'FFFEFCE8')

  const buffer = await wb.xlsx.writeBuffer()
  triggerDownload(buffer, `arbol-paradas-linea-${lineaFiltro.value || 'plantilla'}.xlsx`)
}



const triggerFileInput = () => fileInput.value.click()

const parseSheet = (ws) => {
  if (!ws) return []
  return XLSX.utils.sheet_to_json(ws).map(r => ({
    categoria: r['CATEGORIA'] || '',
    subcategoria: r['SUBCATEGORIA'] || '',
    subcategoria_2: r['SUBCATEGORIA_2'] || '',
    subcategoria_3: r['SUBCATEGORIA_3'] || '',
    subcategoria_4: r['SUBCATEGORIA_4'] || '',
    descripcion_parada: r['DESCRIPCION_PARADA'] || '',
    maquina: r['MAQUINA'] || ''
  })).filter(r => r.categoria)
}

const handleFileUpload = async (event) => {
  const file = event.target.files[0]
  if (!file) return
  if (!lineaFiltro.value) {
    errorMsg.value = 'Seleccione una linea primero'
    return
  }

  uploadLoading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await file.arrayBuffer()
    const wb = XLSX.read(data)

    const prog = parseSheet(wb.Sheets['Paradas Programadas'])
    const noProg = parseSheet(wb.Sheets['Paradas No Programadas'])

    if (!prog.length && !noProg.length) {
      throw new Error('El archivo debe contener al menos una hoja con datos: "Paradas Programadas" o "Paradas No Programadas"')
    }

    const res = await arbolParadasService.importar({
      linea_id: parseInt(lineaFiltro.value),
      programadas: prog,
      no_programadas: noProg
    })

    await loadArbol()

    const d = res.diff || {}
    importSummary.value = {
      total: (d.agregadas || 0) + (d.modificadas || 0),
      diff: {
        agregadas:   d.agregadas   || 0,
        modificadas: d.modificadas || 0,
        eliminadas:  d.eliminadas  || 0
      },
      warning: res.variable_warning || null
    }
  } catch (err) {
    errorMsg.value = err.response?.data?.error || err.message || 'Error al procesar archivo'
  } finally {
    uploadLoading.value = false
    event.target.value = ''
  }
}

const closeImportModal = () => { importSummary.value = null }

const PREVIEW_LIMIT = 8
const diffPreview = (list) => list.slice(0, PREVIEW_LIMIT)
const diffExtra  = (list) => Math.max(0, list.length - PREVIEW_LIMIT)

onMounted(async () => {
  await loadCascade()
  if (empresas.value.length) {
    empresaFiltro.value = empresas.value[0].id
  }
})
</script>

<template>
  <div class="arbol-paradas-view">
    <div class="filters-section">
      <div class="filter-row">
        <div class="filter-item">
          <label class="filter-label">Compania</label>
          <select v-model="empresaFiltro" class="field-select">
            <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Planta</label>
          <select v-model="plantaFiltro" class="field-select">
            <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Linea</label>
          <select v-model="lineaFiltro" class="field-select">
            <option v-for="l in lineasFiltradas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
          </select>
        </div>
      </div>
    </div>

    <!-- Modal resultado importacion -->
    <Teleport to="body">
      <div v-if="importSummary" class="import-modal-overlay" @click.self="closeImportModal">
        <div class="import-modal">
          <div class="import-modal-header">
            <span class="import-modal-title">Resultado de Importacion</span>
            <button class="import-modal-close" @click="closeImportModal">&#x2715;</button>
          </div>

          <div class="import-modal-summary">
            <div class="summary-badge summary-badge--total">
              <span class="badge-num">{{ importSummary.total }}</span>
              <span class="badge-label">Total importados</span>
            </div>
            <div class="summary-badge summary-badge--green">
              <span class="badge-num">{{ importSummary.diff.agregadas }}</span>
              <span class="badge-label">Agregadas</span>
            </div>
            <div class="summary-badge summary-badge--amber">
              <span class="badge-num">{{ importSummary.diff.modificadas }}</span>
              <span class="badge-label">Modificadas</span>
            </div>
            <div class="summary-badge summary-badge--red">
              <span class="badge-num">{{ importSummary.diff.eliminadas }}</span>
              <span class="badge-label">Eliminadas</span>
            </div>
          </div>

          <div v-if="importSummary.warning" class="import-modal-warning">
            &#9888; {{ importSummary.warning }}
          </div>

          <div class="import-modal-body">
            <!-- Agregadas -->
            <div v-if="importSummary.diff.agregadas" class="diff-section">
              <div class="diff-section-header diff-section-header--green">Paradas Agregadas: {{ importSummary.diff.agregadas }} nodos</div>
            </div>

            <!-- Modificadas -->
            <div v-if="importSummary.diff.modificadas" class="diff-section">
              <div class="diff-section-header diff-section-header--amber">Paradas Modificadas: {{ importSummary.diff.modificadas }} nodos</div>
            </div>

            <!-- Eliminadas -->
            <div v-if="importSummary.diff.eliminadas" class="diff-section">
              <div class="diff-section-header diff-section-header--red">Paradas Eliminadas: {{ importSummary.diff.eliminadas }} nodos</div>
            </div>

            <div v-if="!importSummary.diff.agregadas && !importSummary.diff.modificadas && !importSummary.diff.eliminadas"
                 class="diff-no-changes">
              Sin cambios respecto al estado anterior.
            </div>
          </div>

          <div class="import-modal-footer">
            <button class="action-btn action-btn--green" @click="closeImportModal">Aceptar</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Card class="content-card">
      <div class="action-bar">
        <div class="action-bar-left">
          <button class="action-btn action-btn--green" @click="downloadTemplate">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            DESCARGAR PLANTILLA
          </button>
        </div>
        <div class="action-bar-right">
          <input ref="fileInput" type="file" accept=".xlsx,.xls" @change="handleFileUpload" style="display:none" />
          <button class="action-btn action-btn--amber" @click="triggerFileInput" :disabled="uploadLoading || !lineaFiltro">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="16 16 12 12 8 16"/>
              <line x1="12" y1="12" x2="12" y2="21"/>
              <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
            </svg>
            {{ uploadLoading ? 'IMPORTANDO...' : 'IMPORTAR EXCEL' }}
          </button>

        </div>
      </div>

      <Alert v-if="successMsg" type="success" :message="successMsg" class="mx-4 mt-4" />
      <Alert v-if="errorMsg" type="error" :message="errorMsg" class="mx-4 mt-4" />

      <Loading v-if="loading" />
      <Alert v-else-if="error" type="error" :message="error" />

      <div v-else-if="!lineaFiltro" class="empty-state">
        Seleccione empresa, planta y linea para ver el arbol de paradas.
      </div>

      <div v-else class="tree-container">
        <table class="tree-table">
          <thead>
            <tr>
              <th class="th-parada">Parada</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="node in visibleNodes" :key="node.key">
              <!-- Fila hoja: solo muestra el nombre del nivel mas profundo + datos -->
              <tr v-if="node.isLeaf" class="tree-row tree-leaf">
                <td :style="{ paddingLeft: `calc(${node.depth} * 1.5rem + 0.75rem)` }">
                  <span class="leaf-selectable">{{ node.label }}</span>
                </td>
              </tr>
              <!-- Fila rama (tipo, categoria, subcategoria) -->
              <tr v-else class="tree-row tree-branch"
                  :style="{ backgroundColor: BRANCH_BG[node.depth] || '#f9fafb' }">
                <td :style="{ paddingLeft: `calc(${node.depth} * 1.5rem + 0.5rem)` }">
                  <button
                    :class="['collapse-btn', node.depth === 0 ? 'collapse-btn--root' : 'collapse-btn--child']"
                    @click="toggleCategory(node.key)"
                  >
                    {{ isExpanded(node.key) ? '\u2212' : '+' }}
                  </button>
                  <span :class="node.depth === 0 ? 'category-title' : 'category-name'">{{ node.label }}</span>
                  <span :class="node.depth === 0 ? 'count-badge' : 'count-badge-sm'">{{ node.childCount }}</span>
                </td>
              </tr>
            </template>

            <tr v-if="!programadas.length && !noProgramadas.length">
              <td class="empty-row">
                No hay datos. Importe un archivo Excel para configurar el arbol de paradas.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.arbol-paradas-view {
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
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
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
  transition: border-color 0.2s;
  cursor: pointer;
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

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  background-color: #1e3a8a;
  flex-wrap: wrap;
}

.action-bar-left,
.action-bar-right {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  font-weight: 600;
  letter-spacing: 0.025em;
  cursor: pointer;
  transition: background-color 0.2s, opacity 0.2s;
  white-space: nowrap;
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn--green {
  background-color: #16a34a;
  color: white;
}

.action-btn--green:hover:not(:disabled) {
  background-color: #15803d;
}

.action-btn--amber {
  background-color: #d97706;
  color: white;
}

.action-btn--amber:hover:not(:disabled) {
  background-color: #b45309;
}

.action-btn--white {
  background-color: white;
  color: #1e3a8a;
}

.action-btn--white:hover:not(:disabled) {
  background-color: #e5e7eb;
}

.tree-container {
  overflow-x: auto;
}

.tree-table {
  width: 100%;
  font-size: 0.875rem;
  border-collapse: collapse;
}

.tree-table thead {
  background-color: #f3f4f6;
}

.tree-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 2px solid #e5e7eb;
}

.tree-table td {
  padding: 0.75rem 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
}

.tree-row td {
  vertical-align: middle;
}

.tree-branch td {
  font-weight: 600;
  padding: 0.75rem 1rem;
}

.tree-leaf {
  background-color: white;
}

.tree-leaf:hover {
  background-color: #eff6ff;
}

.th-parada {
  width: 30%;
}

.leaf-selectable {
  display: inline-block;
  font-weight: 600;
  color: #1e3a8a;
  border-bottom: 2px solid #3b82f6;
  padding-bottom: 1px;
  cursor: default;
  letter-spacing: 0.01em;
}

.collapse-btn {
  border: none;
  color: white;
  border-radius: 0.375rem;
  cursor: pointer;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.2s;
  flex-shrink: 0;
}

.collapse-btn--root {
  width: 32px;
  height: 32px;
  background-color: #1e3a8a;
  font-size: 1.25rem;
  margin-right: 0.75rem;
}

.collapse-btn--root:hover {
  background-color: #1e40af;
}

.collapse-btn--child {
  width: 24px;
  height: 24px;
  background-color: #3b82f6;
  font-size: 1rem;
  margin-right: 0.5rem;
}

.collapse-btn--child:hover {
  background-color: #2563eb;
}

.category-title {
  font-size: 1rem;
  font-weight: 700;
  color: #1e3a8a;
}

.category-name {
  font-size: 0.9375rem;
  font-weight: 600;
  color: #1e40af;
}

.count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 0.5rem;
  background-color: #dbeafe;
  color: #1e40af;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  margin-left: 0.5rem;
}

.count-badge-sm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 0.375rem;
  background-color: #e0e7ff;
  color: #3730a3;
  border-radius: 9999px;
  font-size: 0.6875rem;
  font-weight: 600;
  margin-left: 0.375rem;
}

.pl-12 {
  padding-left: 3.5rem !important;
}

.empty-state {
  padding: 3rem;
  text-align: center;
  color: #6b7280;
  font-size: 0.875rem;
}

.empty-row {
  text-align: center;
  color: #9ca3af;
  padding: 2rem 1rem !important;
  font-style: italic;
}

/* ── Import result modal ─────────────────────────────────────── */
.import-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.55);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.import-modal {
  background: white;
  border-radius: 0.5rem;
  width: 100%;
  max-width: min(640px, 92vw);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
  overflow: hidden;
}

.import-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  background-color: #1e3a8a;
  flex-shrink: 0;
}

.import-modal-title {
  color: white;
  font-weight: 700;
  font-size: 1rem;
  letter-spacing: 0.025em;
}

.import-modal-close {
  background: transparent;
  border: none;
  color: white;
  font-size: 1.125rem;
  cursor: pointer;
  padding: 0.25rem;
  line-height: 1;
  opacity: 0.8;
  transition: opacity 0.15s;
}
.import-modal-close:hover { opacity: 1; }

.import-modal-summary {
  display: flex;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  background: #f8fafc;
  border-bottom: 1px solid #e5e7eb;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.summary-badge {
  flex: 1;
  min-width: 100px;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0.75rem 0.5rem;
  border-radius: 0.375rem;
  border: 1px solid transparent;
}

.summary-badge--total { background: #eff6ff; border-color: #bfdbfe; }
.summary-badge--green { background: #f0fdf4; border-color: #bbf7d0; }
.summary-badge--amber { background: #fffbeb; border-color: #fde68a; }
.summary-badge--red   { background: #fef2f2; border-color: #fecaca; }

.badge-num {
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1;
}
.summary-badge--total .badge-num { color: #1d4ed8; }
.summary-badge--green .badge-num { color: #15803d; }
.summary-badge--amber .badge-num { color: #b45309; }
.summary-badge--red   .badge-num { color: #dc2626; }

.badge-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: #6b7280;
  margin-top: 0.25rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.import-modal-warning {
  background: #fffbeb;
  border-top: 1px solid #fde68a;
  padding: 0.625rem 1.25rem;
  font-size: 0.8125rem;
  color: #92400e;
  flex-shrink: 0;
}

.import-modal-body {
  overflow-y: auto;
  padding: 1rem 1.25rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.diff-section {
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  overflow: hidden;
}

.diff-section-header {
  padding: 0.5rem 0.875rem;
  font-size: 0.8125rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.diff-section-header--green { background: #dcfce7; color: #15803d; }
.diff-section-header--amber { background: #fef9c3; color: #92400e; }
.diff-section-header--red   { background: #fee2e2; color: #b91c1c; }

.diff-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.diff-table th {
  padding: 0.4rem 0.875rem;
  background: #f9fafb;
  text-align: left;
  font-size: 0.7rem;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  border-bottom: 1px solid #e5e7eb;
}
.diff-table td {
  padding: 0.4rem 0.875rem;
  color: #111827;
  border-bottom: 1px solid #f3f4f6;
}
.diff-table tr:last-child td { border-bottom: none; }

.diff-changes {
  color: #6b7280;
  font-style: italic;
  font-size: 0.75rem;
}

.diff-more-row td {
  text-align: center;
  color: #6b7280;
  font-style: italic;
  background: #f9fafb;
}

.diff-no-changes {
  text-align: center;
  color: #6b7280;
  padding: 1.5rem;
  font-style: italic;
}

.import-modal-footer {
  padding: 0.875rem 1.25rem;
  border-top: 1px solid #e5e7eb;
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
}
</style>
