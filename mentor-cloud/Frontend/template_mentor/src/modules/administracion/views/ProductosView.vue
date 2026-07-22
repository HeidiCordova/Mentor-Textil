<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import * as XLSX from 'xlsx'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { productoCaractService } from '@/api/services/productoCaract.service'

import { useAuthStore } from '@/stores/auth'

// convierte índice 0-based a letra de columna Excel (0='A', 1='B', 2='C'...)
function colLetter(idx) {
  let s = '', n = idx + 1
  while (n > 0) { s = String.fromCharCode(65 + ((n - 1) % 26)) + s; n = Math.floor((n - 1) / 26) }
  return s
}

const auth = useAuthStore()

// ─── Filtros ──────────────────────────────────────────────────────────────────
const empresas  = ref([])
const plantas   = ref([])
const lineas    = ref([])

const empresaID = ref(null)
const plantaID  = ref(null)
const lineaID   = ref(null)
const busqueda  = ref('')

const plantasFiltradas = computed(() =>
  empresaID.value ? plantas.value.filter(p => p.empresa_id === Number(empresaID.value)) : []
)
const lineasFiltradas = computed(() =>
  plantaID.value ? lineas.value.filter(l => l.planta_id === Number(plantaID.value)) : []
)

// ─── Columnas = variables de producto configuradas por línea ────────────────
const columnas    = ref([])
const catalogoMap = ref({})  // variable_id → string[]

async function cargarCatalogo() {
  if (!lineaID.value) { catalogoMap.value = {}; return }
  try {
    const res = await productoCaractService.getCatalogoAll({ linea_id: lineaID.value })
    catalogoMap.value = res.data ?? {}
  } catch { catalogoMap.value = {} }
}

function getCatalogoOpciones(variableId) {
  return catalogoMap.value[variableId] ?? catalogoMap.value[String(variableId)] ?? []
}

// ─── Productos ────────────────────────────────────────────────────────────────
const productos    = ref([])
const loadingProds = ref(false)
const paginaProds  = ref(1)
const PPP          = 10

const prodFiltrados = computed(() => {
  const b = busqueda.value.toLowerCase()
  return productos.value.filter(p =>
    !b || p.codigo.toLowerCase().includes(b) || p.nombre.toLowerCase().includes(b)
  )
})
const prodPaginados = computed(() => {
  const s = (paginaProds.value - 1) * PPP
  return prodFiltrados.value.slice(s, s + PPP)
})
const totalProds = computed(() => Math.ceil(prodFiltrados.value.length / PPP))

let _reqLineaID = null
async function cargarProductos() {
  if (!lineaID.value) { columnas.value = []; productos.value = []; return }
  const currentLinea = lineaID.value
  _reqLineaID = currentLinea
  loadingProds.value = true
  try {
    const params = { linea_id: currentLinea }
    if (empresaID.value) params.empresa_id = empresaID.value
    const [res] = await Promise.all([
      productoCaractService.getCaracteristicas(params),
      cargarCatalogo()
    ])
    // descartar si el usuario ya cambió de línea
    if (_reqLineaID !== currentLinea) return
    columnas.value = res.columnas ?? []
    productos.value = res.productos ?? []
  } finally {
    if (_reqLineaID === currentLinea) loadingProds.value = false
  }
}

// ─── Configurador de columnas (linea_producto_vars) ─────────────────────────
const showColConfig     = ref(false)
const colConfigLoading  = ref(false)
const colConfigGuardando = ref(false)
const colConfigError    = ref('')
const varDisponibles    = ref([])  // { id, nombre } de tipo OTRO no configuradas
const colsConfig        = ref([])  // { variable_id, nombre_col, orden, _nombre }  ← editables

async function abrirColConfig() {
  if (!lineaID.value) return
  showColConfig.value = true
  colConfigLoading.value = true
  colConfigError.value = ''
  try {
    const [varsRes, lpvRes] = await Promise.all([
      productoCaractService.getVariablesLinea({ linea_id: lineaID.value }),
      productoCaractService.getLineaVars({ linea_id: lineaID.value })
    ])
    const todasVars = varsRes.data ?? []
    const configured = lpvRes.data ?? []
    const configuredIds = new Set(configured.map(c => c.variable_id))

    colsConfig.value = configured
      .sort((a, b) => a.orden - b.orden)
      .map(c => ({ ...c, _nombre: todasVars.find(v => v.id === c.variable_id)?.nombre ?? c.variable_nombre ?? '' }))

    varDisponibles.value = todasVars
      .filter(v => !configuredIds.has(v.id))
      .sort((a, b) => a.nombre.localeCompare(b.nombre))
  } catch (e) {
    colConfigError.value = e?.response?.data?.error || e.message || 'Error al cargar'
  } finally {
    colConfigLoading.value = false
  }
}

function agregarColumna(v) {
  colsConfig.value.push({
    variable_id: v.id,
    nombre_col: v.nombre,
    orden: colsConfig.value.length + 1,
    _nombre: v.nombre
  })
  varDisponibles.value = varDisponibles.value.filter(x => x.id !== v.id)
}

function quitarColumna(idx) {
  const col = colsConfig.value[idx]
  varDisponibles.value.push({ id: col.variable_id, nombre: col._nombre })
  varDisponibles.value.sort((a, b) => a.nombre.localeCompare(b.nombre))
  colsConfig.value.splice(idx, 1)
  colsConfig.value.forEach((c, i) => { c.orden = i + 1 })
}

function moverColumna(idx, dir) {
  const arr = colsConfig.value
  const swap = idx + dir
  if (swap < 0 || swap >= arr.length) return
  ;[arr[idx], arr[swap]] = [arr[swap], arr[idx]]
  arr.forEach((c, i) => { c.orden = i + 1 })
}

async function guardarColConfig() {
  colConfigGuardando.value = true
  colConfigError.value = ''
  try {
    await productoCaractService.saveLineaVars({
      linea_id: Number(lineaID.value),
      columnas: colsConfig.value.map((c, i) => ({
        variable_id: c.variable_id,
        nombre_col: c.nombre_col || c._nombre,
        orden: i + 1
      }))
    })
    showColConfig.value = false
    await cargarProductos()
  } catch (e) {
    colConfigError.value = e?.response?.data?.error || e.message || 'Error al guardar'
  } finally {
    colConfigGuardando.value = false
  }
}

// ─── Formulario Agregar / Editar ──────────────────────────────────────────────
const showForm     = ref(false)
const editandoId   = ref(null)   // null = nuevo, número = editar
const formSKU      = ref('')
const formDesc     = ref('')
const formValores  = ref({})     // variable_id → valor string
const guardandoForm = ref(false)
const formError    = ref('')

function abrirAgregar() {
  editandoId.value  = null
  formSKU.value     = ''
  formDesc.value    = ''
  formValores.value = {}
  columnas.value.forEach(c => { formValores.value[c.variable_id] = '' })
  formError.value   = ''
  showForm.value    = true
}

function abrirEditar(prod) {
  editandoId.value  = prod.id
  formSKU.value     = prod.codigo
  formDesc.value    = prod.nombre
  formValores.value = {}
  columnas.value.forEach(c => {
    formValores.value[c.variable_id] = prod.valores?.[c.variable_id]?.valor ?? ''
  })
  formError.value = ''
  showForm.value  = true
}

async function guardarProductoForm() {
  if (!formSKU.value.trim()) { formError.value = 'SKU es requerido'; return }
  guardandoForm.value = true; formError.value = ''
  try {
    let pid = editandoId.value
    if (!pid) {
      // verificar si ya existe en la lista cargada
      const existe = productos.value.find(
        p => p.codigo.trim().toUpperCase() === formSKU.value.trim().toUpperCase()
      )
      if (existe) {
        pid = existe.id
      } else {
        const res = await productoCaractService.crearProducto({
          codigo: formSKU.value.trim(),
          nombre: formDesc.value.trim() || formSKU.value.trim(),
          empresa_id: empresaID.value ? Number(empresaID.value) : null,
          linea_id: Number(lineaID.value)
        })
        pid = res.id
      }
    }
    const valores = columnas.value.map(c => ({
      variable_id: c.variable_id,
      valor: String(formValores.value[c.variable_id] ?? '').trim()
    }))
    await productoCaractService.saveCaracteristicas({
      linea_id: Number(lineaID.value),
      productos: [{ producto_id: pid, valores }]
    })
    showForm.value = false
    await cargarProductos()
  } catch (err) {
    formError.value = err?.response?.data?.error || err.message || 'Error al guardar'
  } finally {
    guardandoForm.value = false
  }
}

// ─── Descargar (plantilla vacía si no hay datos, con datos si los hay) ────────
async function descargar() {
  if (!lineaID.value) return
  await cargarProductos()  // refresca columnas + productos
  const lineaNombre = lineasFiltradas.value.find(l => l.id === Number(lineaID.value))?.nombre ?? `linea-${lineaID.value}`
  const cols = columnas.value
  const headers = ['SKU', 'Descripción', ...cols.map(c => c.nombre_col)]

  let filas
  if (prodFiltrados.value.length > 0) {
    // con datos
    filas = prodFiltrados.value.map(p => {
      const row = { 'SKU': p.codigo, 'Descripción': p.nombre }
      cols.forEach(c => { row[c.nombre_col] = p.valores?.[c.variable_id]?.valor ?? '' })
      return row
    })
  } else {
    // plantilla vacía con fila ejemplo
    const ej = { 'SKU': '', 'Descripción': '' }
    cols.forEach(c => { ej[c.nombre_col] = '' })
    filas = [ej]
  }

  const wb = XLSX.utils.book_new()
  const ws = XLSX.utils.json_to_sheet(filas, { header: headers })

  // — Dropdowns: catálogo de valores por línea (ya cargado en estado)
  const catMap = catalogoMap.value

  cols.forEach((col, i) => {
    const valores = catMap[col.variable_id] ?? catMap[String(col.variable_id)]
    if (valores && valores.length > 0) {
      const letter = colLetter(2 + i)  // A=SKU, B=Desc, C=primera col producto
      const sqref  = `${letter}2:${letter}1000`
      if (!ws['!datavalidation']) ws['!datavalidation'] = {}
      ws['!datavalidation'][sqref] = {
        type: 'list',
        sqref,
        formula1: '"' + valores.join(',') + '"'
      }
    }
  })

  XLSX.utils.book_append_sheet(wb, ws, lineaNombre)

  // Hoja auxiliar de catálogo (siempre útil como referencia visual)
  const hasCatalogo = cols.some(c => (catMap[c.variable_id]?.length ?? 0) > 0)
  if (hasCatalogo) {
    const catRows = [['Columna', 'Valores Permitidos']]
    cols.forEach(col => {
      const valores = catMap[col.variable_id]
      if (valores && valores.length > 0) catRows.push([col.nombre_col, valores.join(', ')])
    })
    XLSX.utils.book_append_sheet(wb, XLSX.utils.aoa_to_sheet(catRows), 'Catálogo')
  }

  XLSX.writeFile(wb, `productos-${lineaNombre}.xlsx`)
}

// ─── Importar ─────────────────────────────────────────────────────────────────
const importando      = ref(false)
const importResultado = ref('')
const fileInputRef    = ref(null)

function abrirImportar() {
  if (!lineaID.value) return
  fileInputRef.value?.click()
}

async function onArchivoImport(e) {
  const file = e.target.files?.[0]
  if (!file || !lineaID.value) return
  e.target.value = ''
  importando.value = true; importResultado.value = ''
  try {
    const buf  = await file.arrayBuffer()
    const wb   = XLSX.read(buf, { type: 'array' })
    const ws   = wb.Sheets[wb.SheetNames[0]]
    const rows = XLSX.utils.sheet_to_json(ws, { defval: '' })
    if (!rows.length) { importResultado.value = 'err:Archivo vacío'; return }

    // cabecera → variable_id  (por clave de variable)
    const colMap = {}
    columnas.value.forEach(c => { colMap[c.nombre_col] = c.variable_id })

    // obtener catálogo para validación (usa el ya cargado en estado)
    const catMap = catalogoMap.value

    const existentes = {}
    productos.value.forEach(p => { existentes[p.codigo.trim().toUpperCase()] = p.id })

    const batch = []; let creados = 0; const errores = []; let avisos = 0

    for (const row of rows) {
      const sku  = String(row['SKU'] ?? row['sku'] ?? '').trim()
      const desc = String(row['Descripción'] ?? row['Descripcion'] ?? '').trim()
      if (!sku) continue

      let pid = existentes[sku.toUpperCase()] ?? null
      if (!pid) {
        try {
          const res = await productoCaractService.crearProducto({
            codigo: sku, nombre: desc || sku,
            linea_id: Number(lineaID.value)
          })
          pid = res.id; existentes[sku.toUpperCase()] = pid; creados++
        } catch { errores.push(sku); continue }
      }

      const valores = []
      for (const [col, vid] of Object.entries(colMap)) {
        const val = String(row[col] ?? '').trim()
        const catValores = catMap[vid] ?? catMap[String(vid)]
        // aviso si valor no está en catálogo (pero se importa igual)
        if (catValores && catValores.length > 0 && val &&
            !catValores.some(cv => cv.toLowerCase() === val.toLowerCase())) {
          avisos++
        }
        valores.push({ variable_id: vid, valor: val })
      }
      batch.push({ producto_id: pid, valores })
    }

    if (batch.length > 0) {
      await productoCaractService.saveCaracteristicas({
        linea_id: Number(lineaID.value), productos: batch
      })
    }

    importResultado.value = `ok:${batch.length} importados${creados ? ` (${creados} nuevos)` : ''}${avisos ? ` ⚠${avisos} val. fuera de catálogo` : ''}${errores.length ? ` — ${errores.length} saltados` : ''}`
    setTimeout(() => { importResultado.value = '' }, 5000)
    await cargarProductos()
  } catch (err) {
    importResultado.value = 'err:' + (err?.response?.data?.error || err.message || 'Error')
  } finally {
    importando.value = false
  }
}

// ─── Init ─────────────────────────────────────────────────────────────────────
async function cargarFiltros() {
  const [empRes, plantRes] = await Promise.all([companyService.getAll(), plantService.getAll()])
  empresas.value = empRes.data ?? []
  plantas.value  = plantRes.data ?? []
  if (!empresaID.value && auth.user?.empresa_id) empresaID.value = auth.user.empresa_id
}

watch(empresaID, async () => {
  plantaID.value = null; lineaID.value = null
})
watch(plantaID, async () => {
  if (!plantaID.value) { lineas.value = []; lineaID.value = null; return }
  const res = await lineService.getAll({ planta_id: plantaID.value })
  lineas.value = res.data ?? []; lineaID.value = null
})
watch(lineaID, async () => {
  paginaProds.value = 1
  busqueda.value = ''
  await cargarProductos()
})

onMounted(async () => {
  await cargarFiltros()
})
</script>

<template>
  <div class="pv-wrap">

    <div class="pv-header">
      <h1 class="pv-title">PRODUCTOS</h1>
    </div>

    <!-- ── Filtros ── -->
    <div class="pv-filters">
      <div class="pv-filter-group">
        <label>Empresa</label>
        <select v-model="empresaID" class="pv-select">
          <option value="">Todas</option>
          <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
        </select>
      </div>
      <div class="pv-filter-group">
        <label>Planta</label>
        <select v-model="plantaID" class="pv-select" :disabled="!empresaID">
          <option value="">Seleccionar...</option>
          <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
        </select>
      </div>
      <div class="pv-filter-group">
        <label>Línea</label>
        <select v-model="lineaID" class="pv-select" :disabled="!plantaID">
          <option value="">Seleccionar...</option>
          <option v-for="l in lineasFiltradas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
        </select>
      </div>
      <div class="pv-filter-group pv-search">
        <label>Búsqueda</label>
        <div class="pv-search-wrap">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path stroke-linecap="round" d="M21 21l-4.35-4.35"/></svg>
          <input v-model="busqueda" class="pv-input" placeholder="Buscar SKU o descripción..." />
        </div>
      </div>
    </div>

    <!-- ══ Productos ══ -->
    <div class="pv-content">
      <div class="pv-action-bar">
        <button class="pv-btn success" :disabled="!lineaID" @click="abrirAgregar">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" d="M12 4v16m8-8H4"/></svg>
          Agregar
        </button>
        <button class="pv-btn info" :disabled="!lineaID" @click="abrirColConfig" title="Configurar qué variables aparecen como columnas en el formulario de producto">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path stroke-linecap="round" d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>
          Columnas
        </button>
        <span v-if="!lineaID" class="pv-hint">Selecciona empresa, planta y línea</span>
        <div style="margin-left:auto; display:flex; gap:.5rem; align-items:center;">
          <button class="pv-btn accent" :disabled="!lineaID" @click="descargar">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4 4l-4-4m4 4l4-4"/></svg>
            Descargar
          </button>
          <button class="pv-btn secondary" :disabled="!lineaID || importando" @click="abrirImportar">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/></svg>
            {{ importando ? 'Importando...' : 'Importar' }}
          </button>
          <span v-if="importResultado.startsWith('ok')" class="pv-ok-badge">{{ importResultado.slice(3) }}</span>
          <span v-if="importResultado.startsWith('err')" class="pv-err-badge">{{ importResultado.slice(4) }}</span>
          <input ref="fileInputRef" type="file" accept=".xlsx,.xls,.csv" style="display:none" @change="onArchivoImport" />
        </div>
      </div>

      <div v-if="!lineaID" class="pv-empty-state">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#94a3b8" stroke-width="1.5"><path stroke-linecap="round" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 10V7"/></svg>
        <p>Selecciona empresa, planta y línea para ver los productos</p>
      </div>

      <template v-else>
        <div v-if="loadingProds" class="pv-loading"><div class="pv-spinner"></div> Cargando...</div>
        <div v-else class="pv-table-scroll">
          <table class="pv-table pv-table-prod">
            <thead>
              <tr>
                <th style="width:44px">N°</th>
                <th style="min-width:90px">SKU</th>
                <th style="min-width:200px">Descripción</th>
                <th v-for="col in columnas" :key="col.variable_id" style="min-width:130px">{{ col.nombre_col }}</th>
                <th style="width:36px"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="prodPaginados.length===0"><td :colspan="3+columnas.length" class="pv-empty">Sin productos para esta línea</td></tr>
              <tr v-for="(p, i) in prodPaginados" :key="p.id" class="pv-row" @dblclick="abrirEditar(p)">
                <td class="pv-num">{{ (paginaProds-1)*PPP + i + 1 }}</td>
                <td class="pv-sku">{{ p.codigo }}</td>
                <td>{{ p.nombre }}</td>
                <td v-for="col in columnas" :key="col.variable_id" class="pv-val">
                  {{ p.valores?.[col.variable_id]?.valor ?? '—' }}
                </td>
                <td style="width:36px;text-align:center">
                  <button class="pv-edit-btn" @click.stop="abrirEditar(p)" title="Editar">✎</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="pv-footer">
          <span class="pv-count">{{ prodFiltrados.length }} registros</span>
          <div class="pv-pagination">
            <button class="pv-page-btn" :disabled="paginaProds===1" @click="paginaProds--">Anterior</button>
            <span class="pv-page-info">{{ paginaProds }} / {{ totalProds || 1 }}</span>
            <button class="pv-page-btn" :disabled="paginaProds>=totalProds" @click="paginaProds++">Siguiente</button>
          </div>
        </div>
      </template>
    </div>

    <!-- ══ Modal configurar columnas ══ -->
    <teleport to="body">
      <div v-if="showColConfig" class="pv-modal-overlay" @click.self="showColConfig = false">
        <div class="pv-modal pv-modal--col">
          <div class="pv-modal-header">
            <span>CONFIGURAR COLUMNAS DE PRODUCTO</span>
            <button class="pv-modal-close" @click="showColConfig = false">✕</button>
          </div>
          <div class="pv-modal-body">
            <div v-if="colConfigLoading" style="text-align:center;padding:2rem;color:#6b7280">Cargando variables...</div>
            <template v-else>
              <p style="font-size:.75rem;color:#6b7280;margin:0 0 .75rem">Define qué variables aparecen como columnas al crear o editar un producto en esta línea.</p>

              <div class="pv-col-layout">
                <!-- Columna izquierda: disponibles -->
                <div class="pv-col-panel">
                  <div class="pv-col-panel-title">VARIABLES DISPONIBLES</div>
                  <div class="pv-col-panel-body">
                    <div v-if="varDisponibles.length === 0" class="pv-col-empty">Sin variables adicionales</div>
                    <div v-for="v in varDisponibles" :key="v.id" class="pv-col-item">
                      <span>{{ v.nombre }}</span>
                      <button class="pv-col-add-btn" @click="agregarColumna(v)">+ Agregar</button>
                    </div>
                  </div>
                </div>

                <!-- Columna derecha: configuradas -->
                <div class="pv-col-panel">
                  <div class="pv-col-panel-title">COLUMNAS ACTIVAS <span style="color:#94a3b8;font-weight:400">(orden y nombre personalizable)</span></div>
                  <div class="pv-col-panel-body">
                    <div v-if="colsConfig.length === 0" class="pv-col-empty">Ninguna columna configurada</div>
                    <div v-for="(col, idx) in colsConfig" :key="col.variable_id" class="pv-col-item pv-col-item--active">
                      <div class="pv-col-order">{{ idx + 1 }}</div>
                      <input v-model="col.nombre_col" class="pv-col-name-input" placeholder="Nombre de columna" />
                      <span class="pv-col-var-ref">({{ col._nombre || col.variable_id }})</span>
                      <div class="pv-col-actions">
                        <button class="pv-col-move" :disabled="idx === 0" @click="moverColumna(idx, -1)" title="Subir">▲</button>
                        <button class="pv-col-move" :disabled="idx === colsConfig.length - 1" @click="moverColumna(idx, 1)" title="Bajar">▼</button>
                        <button class="pv-col-remove" @click="quitarColumna(idx)" title="Quitar">✕</button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="colConfigError" class="pv-form-error" style="margin-top:.5rem">{{ colConfigError }}</div>
            </template>
          </div>
          <div class="pv-modal-footer">
            <button class="pv-btn success" :disabled="colConfigGuardando || colConfigLoading" @click="guardarColConfig">
              {{ colConfigGuardando ? 'Guardando...' : 'Guardar' }}
            </button>
            <button class="pv-btn secondary" @click="showColConfig = false">Cancelar</button>
          </div>
        </div>
      </div>
    </teleport>

    <!-- ══ Modal formulario ══ -->
    <teleport to="body">
      <div v-if="showForm" class="pv-modal-overlay" @click.self="showForm = false">
        <div class="pv-modal">
          <div class="pv-modal-header">
            <span>{{ editandoId ? 'EDITAR PRODUCTO' : 'AGREGAR PRODUCTO' }}</span>
            <button class="pv-modal-close" @click="showForm = false">✕</button>
          </div>

          <div class="pv-modal-body">
            <!-- Empresa / Planta / Línea (solo lectura) -->
            <div class="pv-form-row3">
              <div class="pv-form-group">
                <label>Empresa</label>
                <input :value="empresas.find(e => e.id === Number(empresaID))?.nombre ?? '—'" class="pv-field pv-field--ro" readonly />
              </div>
              <div class="pv-form-group">
                <label>Planta</label>
                <input :value="plantas.find(p => p.id === Number(plantaID))?.nombre ?? '—'" class="pv-field pv-field--ro" readonly />
              </div>
              <div class="pv-form-group">
                <label>Línea</label>
                <input :value="lineasFiltradas.find(l => l.id === Number(lineaID))?.nombre ?? '—'" class="pv-field pv-field--ro" readonly />
              </div>
            </div>

            <div class="pv-form-divider"></div>

            <!-- SKU + Descripción -->
            <div class="pv-form-row2">
              <div class="pv-form-group">
                <label>SKU <span class="pv-req">*</span></label>
                <input v-model="formSKU" class="pv-field" :readonly="!!editandoId"
                  :class="{ 'pv-field--ro': !!editandoId }" placeholder="Código único" />
              </div>
              <div class="pv-form-group">
                <label>Descripción <span class="pv-req">*</span></label>
                <input v-model="formDesc" class="pv-field" placeholder="Nombre del producto" />
              </div>
            </div>

            <!-- Características dinámicas -->
            <div v-if="columnas.length > 0" class="pv-form-caract">
              <div class="pv-form-caract-title">CARACTERÍSTICAS</div>
              <div class="pv-form-row2">
                <div v-for="col in columnas" :key="col.variable_id" class="pv-form-group">
                  <label>{{ col.nombre_col }}</label>
                  <!-- Dropdown si tiene catálogo -->
                  <select v-if="getCatalogoOpciones(col.variable_id).length > 0"
                    v-model="formValores[col.variable_id]" class="pv-field">
                    <option value="">— Seleccionar —</option>
                    <option v-for="op in getCatalogoOpciones(col.variable_id)" :key="op" :value="op">{{ op }}</option>
                  </select>
                  <!-- Texto libre si no hay catálogo -->
                  <input v-else v-model="formValores[col.variable_id]" class="pv-field"
                    :placeholder="col.nombre_col" />
                </div>
              </div>
            </div>

            <div v-if="formError" class="pv-form-error">{{ formError }}</div>
          </div>

          <div class="pv-modal-footer">
            <button class="pv-btn success" :disabled="guardandoForm" @click="guardarProductoForm">
              {{ guardandoForm ? 'Guardando...' : 'Guardar' }}
            </button>
            <button class="pv-btn secondary" @click="showForm = false">Cancelar</button>
          </div>
        </div>
      </div>
    </teleport>

  </div>
</template>

<style scoped>
/* ── Layout ── */
.pv-wrap     { display:flex; flex-direction:column; gap:0; min-height:100%; background:#f8fafc; }
.pv-header   { background:#1e3a8a; padding:.75rem 1.5rem; }
.pv-title    { margin:0; font-size:1.1rem; font-weight:700; color:#fff; letter-spacing:.08em; text-transform:uppercase; }

/* ── Filtros ── */
.pv-filters  { display:flex; flex-wrap:wrap; gap:.75rem; padding:.75rem 1rem; background:#fff;
               border-bottom:1px solid #e5e7eb; align-items:flex-end; }
.pv-filter-group { display:flex; flex-direction:column; gap:.25rem; min-width:130px; }
.pv-filter-group label { font-size:.7rem; font-weight:600; color:#6b7280; text-transform:uppercase; letter-spacing:.04em; }
.pv-search   { flex:1; min-width:200px; }
.pv-search-wrap { display:flex; align-items:center; gap:.4rem; background:#f3f4f6; border:1px solid #d1d5db;
                  border-radius:.375rem; padding:.35rem .6rem; }
.pv-search-wrap svg { opacity:.5; flex-shrink:0; }

/* ── Content ── */
.pv-content  { flex:1; display:flex; flex-direction:column; gap:0; padding:.75rem 1rem; }

/* ── Action bar ── */
.pv-action-bar { display:flex; align-items:center; flex-wrap:wrap; gap:.5rem;
                 padding:.6rem .75rem; background:#1e3a8a; border-radius:.3rem .3rem 0 0; }

/* ── Buttons ── */
.pv-btn { display:inline-flex; align-items:center; gap:.35rem; padding:.35rem .75rem;
          border:none; border-radius:.25rem; font-weight:700; font-size:.7rem; cursor:pointer;
          letter-spacing:.04em; transition:all .15s; white-space:nowrap; }
.pv-btn:disabled { opacity:.45; cursor:not-allowed; }
.pv-btn.secondary { background:#e5e7eb; color:#374151; }
.pv-btn.secondary:hover:not(:disabled) { background:#d1d5db; }
.pv-btn.accent  { background:#f59e0b; color:#fff; }
.pv-btn.accent:hover:not(:disabled)  { background:#d97706; }
.pv-btn.info    { background:#0ea5e9; color:#fff; }
.pv-btn.info:hover:not(:disabled)    { background:#0284c7; }

/* ── Inputs ── */
.pv-select, .pv-input {
  padding:.38rem .6rem; border:1px solid #d1d5db; border-radius:.3rem; font-size:.8rem;
  background:#fff; color:#374151; width:100%; box-sizing:border-box;
}
.pv-select:disabled { background:#f3f4f6; opacity:.6; cursor:not-allowed; }
.pv-select:focus,.pv-input:focus { outline:none; border-color:#3b82f6; box-shadow:0 0 0 2px rgba(59,130,246,.25); }

/* ── Table ── */
.pv-table-scroll { overflow-x:auto; border:1px solid #e5e7eb; border-top:none;
                   border-radius:0 0 .3rem .3rem; background:#fff; max-height:52vh; }
.pv-table        { width:100%; border-collapse:collapse; font-size:.78rem; }
.pv-table thead tr { background:#f1f5f9; position:sticky; top:0; z-index:1; }
.pv-table th     { padding:.55rem .75rem; text-align:left; font-weight:700; color:#475569;
                   border-bottom:2px solid #e2e8f0; white-space:nowrap; font-size:.7rem; letter-spacing:.05em; }
.pv-table td     { padding:.5rem .75rem; border-bottom:1px solid #f1f5f9; color:#374151; }
.pv-row:hover    { background:#f8fafc; }
.pv-num          { color:#94a3b8; font-size:.7rem; text-align:center; }
.pv-sku          { font-weight:700; color:#1e3a8a; font-family:monospace; font-size:.8rem; }
.pv-val          { font-size:.78rem; }
.pv-empty        { padding:1.5rem; text-align:center; color:#94a3b8; font-size:.8rem; font-style:italic; }

/* ── Badges ── */
.pv-ok-badge  { background:#dcfce7; color:#15803d; border-radius:.25rem;
                padding:.2rem .6rem; font-size:.7rem; font-weight:700; animation:fadein .2s; }
.pv-err-badge { background:#fee2e2; color:#dc2626; border-radius:.25rem;
                padding:.2rem .6rem; font-size:.7rem; font-weight:700; animation:fadein .2s; }
@keyframes fadein { from{opacity:0;transform:translateY(-4px)} to{opacity:1;transform:none} }

/* ── Footer ── */
.pv-footer   { display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap;
               padding:.5rem .75rem; background:#f8fafc; border:1px solid #e5e7eb; border-top:none;
               border-radius:0 0 .3rem .3rem; gap:.5rem; }
.pv-count    { font-size:.72rem; color:#64748b; }
.pv-pagination { display:flex; align-items:center; gap:.5rem; margin:0 auto; }
.pv-page-btn { padding:.3rem .75rem; background:#e5e7eb; border:none; border-radius:.2rem;
               font-size:.7rem; font-weight:600; cursor:pointer; }
.pv-page-btn:disabled { opacity:.45; cursor:not-allowed; }
.pv-page-btn:hover:not(:disabled) { background:#d1d5db; }
.pv-page-info { font-size:.72rem; font-weight:700; color:#475569; min-width:60px; text-align:center; }

/* ── States ── */
.pv-loading  { display:flex; align-items:center; gap:.6rem; padding:2rem; color:#64748b; font-size:.85rem; }
.pv-spinner  { width:18px; height:18px; border:2px solid #e5e7eb; border-top-color:#3b82f6;
               border-radius:50%; animation:spin .6s linear infinite; }
@keyframes spin { to { transform:rotate(360deg) } }
.pv-empty-state { display:flex; flex-direction:column; align-items:center; justify-content:center;
                  gap:.75rem; padding:4rem 2rem; color:#94a3b8; font-size:.85rem; }
.pv-hint     { font-size:.72rem; color:#93c5fd; }
.pv-btn.success { background:#16a34a; color:#fff; }
.pv-btn.success:hover:not(:disabled) { background:#15803d; }
.pv-edit-btn { background:none; border:none; cursor:pointer; font-size:.9rem; color:#94a3b8;
               padding:.15rem .3rem; border-radius:.2rem; transition:all .15s; }
.pv-edit-btn:hover { background:#dbeafe; color:#1e40af; }

/* ── Modal ── */
.pv-modal-overlay {
  position:fixed; inset:0; background:rgba(0,0,0,.45); z-index:1000;
  display:flex; align-items:center; justify-content:center; padding:1rem;
}
.pv-modal {
  background:#fff; border-radius:.5rem; width:100%; max-width:min(640px,92vw);
  max-height:90vh; display:flex; flex-direction:column;
  box-shadow:0 20px 60px rgba(0,0,0,.25);
}
.pv-modal-header {
  display:flex; align-items:center; justify-content:space-between;
  padding:.75rem 1.25rem; background:#1e3a8a; border-radius:.5rem .5rem 0 0;
  font-size:.85rem; font-weight:700; color:#fff; letter-spacing:.06em;
}
.pv-modal-close {
  background:none; border:none; color:#fff; font-size:1.1rem; cursor:pointer;
  opacity:.7; transition:opacity .15s;
}
.pv-modal-close:hover { opacity:1; }
.pv-modal-body {
  padding:1.25rem; overflow-y:auto; display:flex; flex-direction:column; gap:1rem;
}
.pv-modal-footer {
  display:flex; gap:.75rem; justify-content:flex-end;
  padding:.75rem 1.25rem; border-top:1px solid #e5e7eb; background:#f8fafc;
  border-radius:0 0 .5rem .5rem;
}
.pv-form-group { display:flex; flex-direction:column; gap:.3rem; }
.pv-form-group label { font-size:.7rem; font-weight:600; color:#6b7280;
                       text-transform:uppercase; letter-spacing:.04em; }
.pv-field {
  padding:.45rem .65rem; border:1px solid #d1d5db; border-radius:.3rem;
  font-size:.82rem; color:#111827; background:#fff; width:100%; box-sizing:border-box;
  transition:border-color .15s;
}
.pv-field:focus { outline:none; border-color:#3b82f6; box-shadow:0 0 0 2px rgba(59,130,246,.2); }
.pv-field--ro { background:#f3f4f6; color:#6b7280; cursor:default; }
.pv-form-row2 { display:grid; grid-template-columns:1fr 1fr; gap:.75rem; }
.pv-form-row3 { display:grid; grid-template-columns:1fr 1fr 1fr; gap:.75rem; }
.pv-form-divider { height:1px; background:#e5e7eb; margin:-.25rem 0; }
.pv-form-caract { background:#f0fdf4; border:1px solid #bbf7d0; border-radius:.4rem; padding:.75rem; }
.pv-form-caract-title {
  font-size:.68rem; font-weight:700; color:#16a34a; letter-spacing:.06em;
  margin-bottom:.6rem;
}
.pv-req { color:#dc2626; }
.pv-form-error {
  background:#fee2e2; color:#dc2626; border-radius:.3rem;
  padding:.4rem .75rem; font-size:.78rem; font-weight:600;
}

/* ── Modal columnas ── */
.pv-modal--col { max-width:min(800px,92vw); }
.pv-col-layout {
  display:grid; grid-template-columns:1fr 1fr; gap:1rem;
}
.pv-col-panel {
  border:1px solid #e5e7eb; border-radius:.4rem; overflow:hidden;
}
.pv-col-panel-title {
  background:#f1f5f9; padding:.45rem .75rem; font-size:.68rem; font-weight:700;
  color:#475569; letter-spacing:.05em; border-bottom:1px solid #e5e7eb;
}
.pv-col-panel-body {
  max-height:300px; overflow-y:auto; padding:.4rem;
  display:flex; flex-direction:column; gap:.3rem;
}
.pv-col-empty { font-size:.75rem; color:#94a3b8; text-align:center; padding:.75rem; font-style:italic; }
.pv-col-item {
  display:flex; align-items:center; gap:.5rem; padding:.4rem .6rem;
  border-radius:.3rem; background:#f8fafc; border:1px solid #e5e7eb;
  font-size:.78rem;
}
.pv-col-item--active { background:#eff6ff; border-color:#bfdbfe; }
.pv-col-add-btn {
  margin-left:auto; flex-shrink:0; padding:.2rem .55rem; background:#16a34a; color:#fff;
  border:none; border-radius:.2rem; font-size:.68rem; font-weight:700; cursor:pointer;
}
.pv-col-add-btn:hover { background:#15803d; }
.pv-col-order {
  flex-shrink:0; width:20px; text-align:center; font-weight:700; color:#94a3b8; font-size:.72rem;
}
.pv-col-name-input {
  flex:1; min-width:0; padding:.25rem .45rem; border:1px solid #d1d5db; border-radius:.2rem;
  font-size:.78rem; background:#fff;
}
.pv-col-name-input:focus { outline:none; border-color:#3b82f6; }
.pv-col-var-ref { color:#94a3b8; font-size:.7rem; white-space:nowrap; }
.pv-col-actions { display:flex; gap:.2rem; flex-shrink:0; }
.pv-col-move {
  background:#e5e7eb; border:none; border-radius:.2rem; padding:.15rem .35rem;
  font-size:.65rem; cursor:pointer; line-height:1;
}
.pv-col-move:disabled { opacity:.35; cursor:not-allowed; }
.pv-col-move:hover:not(:disabled) { background:#d1d5db; }
.pv-col-remove {
  background:#fee2e2; color:#dc2626; border:none; border-radius:.2rem;
  padding:.15rem .35rem; font-size:.7rem; cursor:pointer; line-height:1;
}
.pv-col-remove:hover { background:#fecaca; }

@media (max-width:640px) {
  .pv-filters { flex-direction:column; }
  .pv-filter-group { min-width:100%; }
  .pv-col-layout { grid-template-columns:1fr; }
}
</style>
