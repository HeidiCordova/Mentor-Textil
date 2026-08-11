<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { deviceService } from '@/api/services/device.service'
import { locacionService } from '@/api/services/locacion.service'
import { oeeService } from '@/api/services/oee.service'
import { variableService } from '@/api/services/variable.service'

// ── Paleta de colores para variables dinámicas ──────────────────────────────
const PALETA = [
  '#3b82f6','#ef4444','#22c55e','#f59e0b','#8b5cf6',
  '#ec4899','#14b8a6','#f97316','#06b6d4','#84cc16',
  '#6366f1','#a855f7','#0ea5e9','#d97706','#10b981'
]

// ── Variable de producción (conteo) — va siempre primera ────────────────────
const PROD_VAR = { id: 'produccion', nombre: 'Producción (Conteo)' }

// ── Maestros (cargados desde la API) ────────────────────────────────────────
const companias   = ref([])
const todasPlantas  = ref([])
const todasLineas   = ref([])
const dispositivos  = ref([])
const ubicaciones   = ref([])
const variables     = ref([])

// ── Selecciones ──────────────────────────────────────────────────────────────
const companiaSeleccionada    = ref(null)
const plantaSeleccionada      = ref(null)
const lineaSeleccionada       = ref(null)
const ubicacionSeleccionada   = ref(null)
const dispositivoSeleccionado = ref(null)
const agrupamientoSeleccionado = ref('5min')

// ── Agrupamientos (estáticos) ────────────────────────────────────────────────
const agrupamientos = ref([
  { id: 1, nombre: '5 minutos',  valor: '5min' },
  { id: 2, nombre: '15 minutos', valor: '15min' },
  { id: 3, nombre: '30 minutos', valor: '30min' },
  { id: 4, nombre: '1 hora',     valor: '1h'   }
])

// ── Listas filtradas (cascada) ────────────────────────────────────────────────
const plantasFiltradas = computed(() => {
  if (!companiaSeleccionada.value) return todasPlantas.value
  return todasPlantas.value.filter(
    p => p.empresa_id === parseInt(companiaSeleccionada.value)
  )
})

const lineasFiltradas = computed(() => {
  if (!plantaSeleccionada.value) return todasLineas.value
  return todasLineas.value.filter(
    l => l.planta_id === parseInt(plantaSeleccionada.value)
  )
})

// El flujo textil trabaja con snapshots cada 30 minutos.
const modoLinea = computed(() => 'textil')
const pasoMin = computed(() => 30)
const maxMinIdx = computed(() => 1)

// ── Fechas (sliders separados) ─────────────────────────────────────────────
const pad2 = n => String(n).padStart(2, '0')

// Configuración de intervalo adaptativa según modo de línea
const intervalConfig = computed(() => ({
  msStep: 1800000,
  sliderMax: 1,
  minuteMultiplier: 30,
  divisor: 30,
  minIntervalS: 1800
}))

function roundDownInterval(d) {
  const ms = d.getTime()
  return new Date(ms - (ms % intervalConfig.value.msStep))
}

function splitDate(d) {
  const r = roundDownInterval(d)
  return {
    fecha: `${r.getFullYear()}-${pad2(r.getMonth() + 1)}-${pad2(r.getDate())}`,
    horaNum: r.getHours(),
    minIdx: Math.floor(r.getMinutes() / intervalConfig.value.divisor)
  }
}

const _now    = new Date()
const _hace7d = new Date(_now.getTime() - 7 * 24 * 60 * 60 * 1000)

const iniParts = splitDate(_hace7d)
const finParts = splitDate(_now)

const inicioFecha   = ref(iniParts.fecha)
const inicioHoraNum = ref(iniParts.horaNum)
const inicioMinIdx  = ref(iniParts.minIdx)

const finFecha   = ref(finParts.fecha)
const finHoraNum = ref(finParts.horaNum)
const finMinIdx  = ref(finParts.minIdx)

const inicioHora   = computed(() => pad2(inicioHoraNum.value))
const inicioMinuto = computed(() => pad2(inicioMinIdx.value * intervalConfig.value.minuteMultiplier))
const finHora      = computed(() => pad2(finHoraNum.value))
const finMinuto    = computed(() => pad2(finMinIdx.value * intervalConfig.value.minuteMultiplier))

const fechaInicio = computed(() => `${inicioFecha.value}T${inicioHora.value}:${inicioMinuto.value}`)
const fechaFin    = computed(() => `${finFecha.value}T${finHora.value}:${finMinuto.value}`)

// Al cambiar modo de línea, ajustar índices de minutos si exceden el nuevo máximo
watch(intervalConfig, () => {
  inicioMinIdx.value = Math.min(inicioMinIdx.value, intervalConfig.value.sliderMax)
  finMinIdx.value    = Math.min(finMinIdx.value,    intervalConfig.value.sliderMax)
  agrupamientoSeleccionado.value = '30min'
})

// ── Estado del gráfico ───────────────────────────────────────────────────────
const variablesAgregadas = ref([])
const datosGrafico       = ref({})
const tooltipVisible     = ref(false)
const tooltipData        = ref({ x: 0, y: 0, fecha: '', valor: 0, variable: '' })
const mostrarTabla       = ref(false)
const cargando           = ref(false)
const mensajeError       = ref(null)

// Mapa dinámico varId → color (se popula al agregar variables)
const coloresVariables = ref({})

// ── Helpers ──────────────────────────────────────────────────────────────────
function _asignarColor(varId) {
  if (!coloresVariables.value[varId]) {
    const idx = Object.keys(coloresVariables.value).length % PALETA.length
    coloresVariables.value[varId] = PALETA[idx]
  }
}

function _isoDesde(str) {
  // Acepta 'YYYY-MM-DDTHH:MM' o 'YYYY/MM/DD HH:MM'
  return str.replace(/\//g, '-').replace(' ', 'T') + (str.includes(':') && str.length === 16 ? ':00' : '')
}

// ── Carga de maestros ────────────────────────────────────────────────────────
async function cargarMaestros() {
  try {
    const [empRes, plnRes, linRes] = await Promise.all([
      companyService.getAll(),
      plantService.getAll(),
      lineService.getAll()
    ])
    companias.value    = empRes.data?.data || empRes.data || []
    todasPlantas.value = plnRes.data?.data || plnRes.data || []
    todasLineas.value  = linRes.data?.data || linRes.data || []

    if (companias.value.length > 0)
      companiaSeleccionada.value = companias.value[0].id
  } catch (e) {
    mensajeError.value = 'Error al cargar maestros'
  }
}

async function cargarDispositivos() {
  if (!lineaSeleccionada.value) { dispositivos.value = []; return }
  try {
    const res = await deviceService.getAll()
    const todos = res.data?.data || res.data || []
    dispositivos.value = todos.filter(d => d.linea_id === parseInt(lineaSeleccionada.value))
    dispositivoSeleccionado.value = dispositivos.value.length > 0 ? dispositivos.value[0].id : null
  } catch { dispositivos.value = [] }
}

async function cargarUbicaciones() {
  if (!lineaSeleccionada.value) { ubicaciones.value = []; return }
  try {
    const res = await locacionService.getAll({ linea_id: lineaSeleccionada.value })
    ubicaciones.value = res.data?.data || res.data || []
    ubicacionSeleccionada.value = ubicaciones.value.length > 0 ? ubicaciones.value[0].id : null
  } catch { ubicaciones.value = [] }
}

async function cargarVariables() {
  variables.value = []
  variablesAgregadas.value = []
  datosGrafico.value = {}
  coloresVariables.value = {}

  if (!lineaSeleccionada.value) return

  // Producción (conteo) siempre primero
  const resultado = [PROD_VAR]

  // Variables del catálogo real (config.variables) filtradas por dispositivo, sin duplicados
  try {
    const params = {}
    if (dispositivoSeleccionado.value) params.dispositivo_id = dispositivoSeleccionado.value
    const res = await variableService.getAll(params)
    const vars = res.data?.data || res.data || []
    const clavesSeen = new Set()
    vars
      .filter(v => v.activo !== false)
      .forEach(v => {
        if (!clavesSeen.has(v.clave)) {
          clavesSeen.add(v.clave)
          resultado.push({
            id:      v.clave,
            nombre:  v.nombre,
            tipo:    v.tipo   || 'OTRO',
            formula: v.valor  || ''
          })
        }
      })
  } catch { /* si falla, solo se muestra produccion */ }

  // Ocultar las variables acumulativas crudas cuando ya existe su versión DELTA.
  // Ej.: "Conteo Unitario Principal" desaparece porque "...por intervalo" la cubre mejor.
  const clavesConDelta = new Set(
    resultado
      .filter(v => v.tipo === 'DERIVADA' && v.formula.startsWith('DELTA:'))
      .map(v => v.formula.replace('DELTA:', ''))
  )
  variables.value = resultado.filter(v => !clavesConDelta.has(v.id))
}

// ── Watchers cascada ─────────────────────────────────────────────────────────
watch(companiaSeleccionada, () => {
  plantaSeleccionada.value = plantasFiltradas.value.length > 0 ? plantasFiltradas.value[0].id : null
})

watch(plantaSeleccionada, () => {
  lineaSeleccionada.value = lineasFiltradas.value.length > 0 ? lineasFiltradas.value[0].id : null
})

watch(lineaSeleccionada, async () => {
  await Promise.all([cargarDispositivos(), cargarUbicaciones(), cargarVariables()])
})

// ── Gestión de variables del gráfico ─────────────────────────────────────────
function toggleVariable(variable) {
  const index = variablesAgregadas.value.findIndex(v => v.id === variable.id)
  if (index > -1) {
    variablesAgregadas.value.splice(index, 1)
  } else {
    _asignarColor(variable.id)
    variablesAgregadas.value.push({ ...variable })
  }
}

function eliminarVariable(id) {
  const index = variablesAgregadas.value.findIndex(v => v.id === id)
  if (index > -1) {
    variablesAgregadas.value.splice(index, 1)
    if (variablesAgregadas.value.length === 0) {
      datosGrafico.value = {}
    }
  }
}


// ── Tooltip ────────────────────────────────────────────────────────────────
function mostrarTooltip(event, punto, variable) {
  const svg = event.currentTarget.closest('svg')
  const rect = svg.getBoundingClientRect()
  
  tooltipData.value = {
    x: event.clientX - rect.left,
    y: event.clientY - rect.top,
    fecha: formatearFechaTooltip(punto.fecha),
    valor: punto.valor,
    variable: variable.nombre
  }
  
  tooltipVisible.value = true
}

function ocultarTooltip() {
  tooltipVisible.value = false
}

function formatearFechaTooltip(fecha) {
  const date = new Date(fecha)
  const dia = String(date.getDate()).padStart(2, '0')
  const mes = String(date.getMonth() + 1).padStart(2, '0')
  const anio = date.getFullYear()
  const hora = String(date.getHours()).padStart(2, '0')
  const minutos = String(date.getMinutes()).padStart(2, '0')
  return `${dia}/${mes}/${anio} ${hora}:${minutos}`
}

function formatearFecha(fecha) {
  const date = new Date(fecha)
  const mes = String(date.getMonth() + 1).padStart(2, '0')
  const dia = String(date.getDate()).padStart(2, '0')
  const hora = String(date.getHours()).padStart(2, '0')
  const minutos = String(date.getMinutes()).padStart(2, '0')
  return `${mes}/${dia} ${hora}:${minutos}`
}

function generarPathLinea() {
  if (datosGrafico.value.length === 0) return ''
  
  let path = ''
  datosGrafico.value.forEach((punto, index) => {
    const x = (index / (datosGrafico.value.length - 1)) * 1200
    const y = 300 - ((punto.valor / 70000) * 300)
    
    if (index === 0) {
      path += `M ${x} ${y}`
    } else {
      path += ` L ${x} ${y}`
    }
  })
  
  return path
}

function generarPathLineaVariable(variableId) {
  const datos = datosGrafico.value[variableId]
  if (!datos || datos.length === 0) return ''
  if (datos.length === 1) return `M 600 150` // punto único al centro

  const allValues = Object.values(datosGrafico.value).flat().map(d => d.valor).filter(v => isFinite(v))
  const maxValor  = allValues.length > 0 ? Math.max(...allValues) : 1
  const escala   = maxValor > 0 ? maxValor : 1   // evitar división por cero

  let path = ''
  datos.forEach((punto, index) => {
    const x = (index / (datos.length - 1)) * 1200
    const y = 300 - ((punto.valor / escala) * 280)
    path += index === 0 ? `M ${x} ${y}` : ` L ${x} ${y}`
  })
  return path
}

function generarPathArea() {
  return ''
}

async function buscar() {
  if (!lineaSeleccionada.value && !dispositivoSeleccionado.value) {
    mensajeError.value = 'Seleccione al menos una línea'
    return
  }
  if (variablesAgregadas.value.length === 0) {
    mensajeError.value = 'Seleccione al menos una variable'
    return
  }
  if (!fechaInicio.value || !fechaFin.value) {
    mensajeError.value = 'Seleccione un rango de fechas'
    return
  }
  if (new Date(fechaInicio.value) >= new Date(fechaFin.value)) {
    mensajeError.value = 'La fecha de inicio debe ser anterior a la fecha de fin'
    return
  }
  mensajeError.value = null
  cargando.value = true
  datosGrafico.value = {}

  try {
    const params = {
      from:  _isoDesde(fechaInicio.value),
      to:    _isoDesde(fechaFin.value),
      limit: 2000,
      min_interval_s: intervalConfig.value.minIntervalS
    }
    if (companiaSeleccionada.value) params.empresa_id = companiaSeleccionada.value
    if (plantaSeleccionada.value)   params.planta_id  = plantaSeleccionada.value
    if (lineaSeleccionada.value)    params.linea_id   = lineaSeleccionada.value
    // No enviar device_id al endpoint OEE — el filtro por linea_id es suficiente
    // y el gateway rechaza IDs numéricos (espera UUID) causando 400

    const res = await oeeService.getSnapshots(params)
    const snapshots = res.data?.data || res.data || []

    // Inicializar arrays para cada variable seleccionada
    const datos = {}
    variablesAgregadas.value.forEach(v => { datos[v.id] = [] })

    // Ordenar snapshots cronológicamente
    snapshots.sort((a, b) => new Date(a.hora) - new Date(b.hora))

    // Índice rápido: clave → posición en head[] (tomamos del primer snap con head)
    const headIdx = {}
    const snapConHead = snapshots.find(s => Array.isArray(s.head) && s.head.length > 0)
    if (snapConHead) {
      snapConHead.head.forEach((h, i) => { headIdx[h] = i })
    }

    // Extraer valor crudo (acumulado o por-intervalo) de un snapshot dado una variable
    function valorSnap(snap, varId) {
      if (varId === 'produccion') return snap.produccion
      const i = headIdx[varId]
      return i !== undefined && Array.isArray(snap.data) ? snap.data[i] : null
    }

    snapshots.forEach((snap, si) => {
      const ts = snap.hora || snap.fecha
      variablesAgregadas.value.forEach(v => {
        let val = null

        if (v.tipo === 'DERIVADA' && v.formula.startsWith('DELTA:')) {
          // Variable derivada: delta entre snapshot actual y anterior
          const srcClave = v.formula.replace('DELTA:', '')
          const curr = valorSnap(snap, srcClave)
          const prev = si > 0 ? valorSnap(snapshots[si - 1], srcClave) : null
          if (curr !== null && curr !== undefined && prev !== null && prev !== undefined) {
            const currNum = parseFloat(curr)
            const prevNum = parseFloat(prev)
            if (!isNaN(currNum) && !isNaN(prevNum)) {
              const delta = currNum - prevNum
              val = delta >= 0 ? delta : currNum // reset del contador
            }
          }
        } else {
          val = valorSnap(snap, v.id)
        }

        if (val !== null && val !== undefined && val !== '') {
          const numVal = parseFloat(val)
          if (!isNaN(numVal) && isFinite(numVal)) {
            datos[v.id].push({
              fecha:          ts,
              valor:          numVal,
              timestamp:      new Date(ts).getTime(),
              variableId:     v.id,
              variableNombre: v.nombre
            })
          }
        }
      })
    })

    datosGrafico.value = datos
  } catch (e) {
    mensajeError.value = 'Error al obtener datos: ' + (e?.message || e)
  } finally {
    cargando.value = false
  }
}

function copiarDatos() {
  const datosTabla = generarDatosTabla()
  let texto = 'Fecha'
  
  variablesAgregadas.value.forEach(variable => {
    texto += `\t${variable.nombre}`
  })
  texto += '\n'
  
  datosTabla.forEach(fila => {
    texto += fila.fecha
    variablesAgregadas.value.forEach(variable => {
      const val = fila[variable.id]
      const formatted = (typeof val === 'number' && isFinite(val)) ? val : ''
      texto += `\t${formatted}`
    })
    texto += '\n'
  })
  
  navigator.clipboard.writeText(texto)
  alert('Datos copiados al portapapeles')
}

function exportarExcel() {
  const datosTabla = generarDatosTabla()
  
  let html = '<html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel" xmlns="http://www.w3.org/TR/REC-html40">'
  html += '<head><meta charset="utf-8"><style>table { border-collapse: collapse; } th, td { border: 1px solid #ddd; padding: 8px; } th { background-color: #f2f2f2; font-weight: bold; }</style></head>'
  html += '<body><table>'
  
  html += '<thead><tr><th>Fecha</th>'
  variablesAgregadas.value.forEach(variable => {
    html += `<th>${variable.nombre}</th>`
  })
  html += '</tr></thead><tbody>'
  
  datosTabla.forEach(fila => {
    html += '<tr>'
    html += `<td>${fila.fecha}</td>`
    variablesAgregadas.value.forEach(variable => {
      const val = fila[variable.id]
      const formatted = (typeof val === 'number' && isFinite(val)) ? val : ''
      html += `<td>${formatted}</td>`
    })
    html += '</tr>'
  })
  
  html += '</tbody></table></body></html>'
  
  const blob = new Blob([html], { type: 'application/vnd.ms-excel' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `variables_${new Date().getTime()}.xls`
  link.click()
}

function generarDatosTabla() {
  if (Object.keys(datosGrafico.value).length === 0) return []
  
  const primeraVariable = Object.keys(datosGrafico.value)[0]
  const datos = datosGrafico.value[primeraVariable]
  
  return datos.map((punto, index) => {
    const fila = {
      fecha: formatearFechaTabla(punto.fecha)
    }
    
    variablesAgregadas.value.forEach(variable => {
      const datosVar = datosGrafico.value[variable.id]
      if (datosVar && datosVar[index]) {
        fila[variable.id] = datosVar[index].valor
      }
    })
    
    return fila
  })
}

function formatearFechaTabla(fecha) {
  const date = new Date(fecha)
  const dia = String(date.getDate()).padStart(2, '0')
  const mes = String(date.getMonth() + 1).padStart(2, '0')
  const anio = date.getFullYear()
  const hora = String(date.getHours()).padStart(2, '0')
  const minutos = String(date.getMinutes()).padStart(2, '0')
  return `${dia}/${mes}/${anio} ${hora}:${minutos}`
}

onMounted(() => {
  cargarMaestros()
})
</script>

<template>
  <div class="generador-consultas-view">
    <div class="page-header">
      <h1 class="page-title">GENERADOR DE CONSULTAS - R</h1>
    </div>

    <Card class="content-card">
      <div class="form-section">
        <h3 class="section-title">Seleccione las variables:</h3>

        <div class="form-layout">
          <div class="form-box">
            <h4 class="box-title">FECHAS</h4>
            <div class="form-group">
              <label class="form-label">Inicio</label>
              <div class="fecha-slider-box">
                <input type="date" v-model="inicioFecha" class="form-control campo-fecha" />
                <div class="tiempo-display">{{ inicioHora }}:{{ inicioMinuto }}</div>
                <div class="slider-fila">
                  <span class="slider-label">Hora</span>
                  <input type="range" min="0" max="23" step="1" v-model.number="inicioHoraNum" class="slider-tiempo" />
                  <span class="slider-val">{{ inicioHora }}</span>
                </div>
                <div class="slider-fila">
                  <span class="slider-label">Min</span>
                  <input type="range" min="0" :max="intervalConfig.sliderMax" step="1" v-model.number="inicioMinIdx" class="slider-tiempo" />
                  <span class="slider-val">{{ inicioMinuto }}</span>
                </div>
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">Fin</label>
              <div class="fecha-slider-box">
                <input type="date" v-model="finFecha" class="form-control campo-fecha" />
                <div class="tiempo-display">{{ finHora }}:{{ finMinuto }}</div>
                <div class="slider-fila">
                  <span class="slider-label">Hora</span>
                  <input type="range" min="0" max="23" step="1" v-model.number="finHoraNum" class="slider-tiempo" />
                  <span class="slider-val">{{ finHora }}</span>
                </div>
                <div class="slider-fila">
                  <span class="slider-label">Min</span>
                  <input type="range" min="0" :max="intervalConfig.sliderMax" step="1" v-model.number="finMinIdx" class="slider-tiempo" />
                  <span class="slider-val">{{ finMinuto }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="form-box">
            <h4 class="box-title">AGRUPAMIENTO</h4>
            <div class="form-group">
              <select v-model="agrupamientoSeleccionado" class="form-control">
                <option v-for="agrupamiento in agrupamientos" :key="agrupamiento.id" :value="agrupamiento.valor">
                  {{ agrupamiento.nombre }}
                </option>
              </select>
            </div>
          </div>

          <div class="form-box">
            <h4 class="box-title">Compañía</h4>
            <div class="form-group">
              <select v-model="companiaSeleccionada" class="form-control">
                <option v-for="compania in companias" :key="compania.id" :value="compania.id">
                  {{ compania.nombre }}
                </option>
              </select>
            </div>

            <h4 class="box-title">Planta</h4>
            <div class="form-group">
              <select v-model="plantaSeleccionada" class="form-control">
                <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
                  {{ planta.nombre }}
                </option>
              </select>
            </div>

            <h4 class="box-title">Línea</h4>
            <div class="form-group">
              <select v-model="lineaSeleccionada" class="form-control">
                <option v-for="linea in lineasFiltradas" :key="linea.id" :value="linea.id">
                  {{ linea.nombre }}
                </option>
              </select>
            </div>

            <h4 class="box-title">Locación</h4>
            <div class="form-group">
              <select v-model="ubicacionSeleccionada" class="form-control">
                <option v-for="ubicacion in ubicaciones" :key="ubicacion.id" :value="ubicacion.id">
                  {{ ubicacion.nombre }}
                </option>
              </select>
            </div>
          </div>

          <div class="form-box">
            <h4 class="box-title">Dispositivo</h4>
            <div class="form-group">
              <select v-model="dispositivoSeleccionado" class="form-control">
                <option v-for="dispositivo in dispositivos" :key="dispositivo.id" :value="dispositivo.id">
                  {{ dispositivo.nombre }}
                </option>
              </select>
            </div>

            <h4 class="box-title">Variable</h4>
            <div class="variable-list">
              <label v-for="variable in variables" :key="variable.id" class="variable-checkbox">
                <input 
                  type="checkbox" 
                  :value="variable.id"
                  :checked="variablesAgregadas.some(v => v.id === variable.id)"
                  @change="toggleVariable(variable)"
                />
                <span>{{ variable.nombre }}</span>
              </label>
            </div>
          </div>
        </div>

        <div v-if="variablesAgregadas.length > 0" class="variables-agregadas">
          <div class="variable-tag" v-for="variable in variablesAgregadas" :key="variable.id">
            <span class="variable-icon">✕</span>
            <span class="variable-name">{{ variable.nombre }}</span>
            <button class="remove-variable" @click="eliminarVariable(variable.id)">×</button>
          </div>
        </div>

        <div class="buscar-section">
          <Button variant="primary" size="md" @click="buscar">
            BUSCAR
          </Button>
        </div>

        <div v-if="Object.keys(datosGrafico).length > 0" class="grafico-container">
          <div class="grafico-header">
            <h4 class="grafico-titulo">Variables Seleccionadas</h4>
          </div>
          
          <div class="grafico-principal">
            <svg viewBox="0 0 1200 300" class="grafico-svg" preserveAspectRatio="none">
              <defs>
                <linearGradient v-for="variable in variablesAgregadas" :key="'gradient-' + variable.id" 
                                :id="'gradientArea-' + variable.id" x1="0%" y1="0%" x2="0%" y2="100%">
                  <stop offset="0%" :style="`stop-color:${coloresVariables[variable.id]};stop-opacity:0.2`" />
                  <stop offset="100%" :style="`stop-color:${coloresVariables[variable.id]};stop-opacity:0.05`" />
                </linearGradient>
              </defs>
              
              <g>
                <line v-for="i in 6" :key="'h-line-' + i" 
                      :x1="0" :y1="i * 50" :x2="1200" :y2="i * 50" 
                      stroke="#e5e7eb" stroke-width="1" />
              </g>
              
              <g v-for="variable in variablesAgregadas" :key="'linea-' + variable.id">
                <path v-if="datosGrafico[variable.id] && datosGrafico[variable.id].length > 0"
                      :d="generarPathLineaVariable(variable.id)" 
                      fill="none" 
                      :stroke="coloresVariables[variable.id]" 
                      stroke-width="2.5" />
              </g>
              
              <g v-for="variable in variablesAgregadas" :key="'puntos-' + variable.id">
                <circle v-for="(punto, index) in (datosGrafico[variable.id] || [])" 
                        v-show="index % 2 === 0"
                        :key="'punto-' + variable.id + '-' + index"
                        :cx="(index / ((datosGrafico[variable.id] || []).length - 1)) * 1200" 
                        :cy="300 - ((punto.valor / Math.max(1, ...Object.values(datosGrafico).flat().map(d => d.valor))) * 280)"
                        r="5" 
                        :fill="coloresVariables[variable.id]"
                        class="punto-interactivo"
                        @mouseenter="mostrarTooltip($event, punto, variable)"
                        @mouseleave="ocultarTooltip" />
              </g>
            </svg>
            
            <div v-if="tooltipVisible" class="tooltip-grafico" 
                 :style="{ left: tooltipData.x + 'px', top: tooltipData.y + 'px' }">
              <div class="tooltip-content">
                <div class="tooltip-variable">{{ tooltipData.variable }}</div>
                <div class="tooltip-fecha">{{ tooltipData.fecha }}</div>
                <div class="tooltip-valor">Valor: <strong>{{ (typeof tooltipData.valor === 'number' && isFinite(tooltipData.valor)) ? tooltipData.valor.toLocaleString() : '-' }}</strong></div>
              </div>
            </div>
          </div>

          <div class="grafico-leyenda">
            <div class="leyenda-items">
              <div v-for="variable in variablesAgregadas" :key="'leyenda-' + variable.id" class="leyenda-item">
                <span class="leyenda-color" :style="{ background: coloresVariables[variable.id] }"></span>
                <span class="leyenda-texto">{{ variable.nombre }}</span>
              </div>
            </div>
            <div class="leyenda-acciones">
              <label class="mostrar-data">
                <input type="checkbox" v-model="mostrarTabla" />
                <span>MOSTRAR DATA</span>
              </label>
              <div class="botones-accion">
                <Button variant="secondary" size="sm" @click="copiarDatos">
                  COPIAR DATA
                </Button>
                <Button variant="success" size="sm" @click="exportarExcel">
                  EXPORTAR EXCEL
                </Button>
              </div>
            </div>
          </div>

          <div v-if="mostrarTabla" class="tabla-datos">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Fecha</th>
                  <th v-for="variable in variablesAgregadas" :key="'th-' + variable.id">
                    {{ variable.nombre }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(fila, index) in generarDatosTabla()" :key="'fila-' + index">
                  <td>{{ fila.fecha }}</td>
                  <td v-for="variable in variablesAgregadas" :key="'td-' + variable.id + '-' + index">
                    {{ (typeof fila[variable.id] === 'number' && isFinite(fila[variable.id])) ? fila[variable.id].toLocaleString() : '-' }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.generador-consultas-view {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding-bottom: 2rem;
}

.page-header {
  background: linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%);
  padding: 1rem 2rem;
  border-radius: 0;
  box-shadow: 0 4px 16px rgba(30, 58, 138, 0.25);
  position: relative;
  overflow: hidden;
}

.page-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(90deg, transparent 0%, rgba(255, 255, 255, 0.1) 50%, transparent 100%);
  pointer-events: none;
}

.page-title {
  font-size: 1.25rem;
  font-weight: 800;
  color: white;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  position: relative;
}

.content-card {
  border-radius: 0;
  border: none;
  border-top: 3px solid #3b82f6;
  padding: 1.5rem 2rem;
  background: #ffffff;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.section-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: #1e3a8a;
  margin: 0;
  padding: 0.65rem 1rem;
  border-left: 4px solid #3b82f6;
  background: linear-gradient(to right, #eff6ff 0%, #ffffff 100%);
  box-shadow: 0 1px 4px rgba(59, 130, 246, 0.08);
}

.form-layout {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1.25rem;
}

.form-box {
  background: #fafafa;
  border: 1px solid #d1d5db;
  border-radius: 0.625rem;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.06);
  transition: all 0.25s ease;
}

.form-box:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12), 0 2px 4px rgba(0, 0, 0, 0.08);
  border-color: #9ca3af;
  transform: translateY(-2px);
}

.box-title {
  font-size: 0.8rem;
  font-weight: 700;
  color: #1f2937;
  margin: 0;
  text-transform: capitalize;
  letter-spacing: 0.02em;
  padding-bottom: 0.625rem;
  border-bottom: 1px solid #e5e7eb;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.form-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: #374151;
  text-transform: capitalize;
  letter-spacing: 0.01em;
  margin-bottom: 0.125rem;
}

.form-control {
  width: 100%;
  padding: 0.625rem 0.875rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  background-color: #ffffff;
  color: #1f2937;
  transition: all 0.2s ease;
  font-weight: 400;
}

.form-control:hover {
  border-color: #9ca3af;
}

.form-control:focus {
  outline: none;
  border-color: #6b7280;
  box-shadow: 0 0 0 3px rgba(107, 114, 128, 0.1);
}

.form-group-inline {
  display: flex;
  gap: 0.6rem;
  align-items: center;
}

.form-group-inline .form-control {
  flex: 1;
}

.buscar-section {
  display: flex;
  justify-content: center;
  padding: 1.5rem;
  background: transparent;
  margin-top: 1rem;
}

.buscar-section:hover {
  transform: none;
}

@media (max-width: 768px) {
  .form-layout {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .content-card {
    padding: 1rem 1.25rem;
  }

  .page-header {
    padding: 0.875rem 1.25rem;
  }

  .page-title {
    font-size: 1.1rem;
  }

  .form-box {
    padding: 0.875rem;
  }
}

.variables-agregadas {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  padding: 1rem;
  background: #f9fafb;
  border-radius: 0.5rem;
  border: 1px solid #e5e7eb;
  margin-bottom: 1rem;
}

.variable-tag {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #6b7280;
  color: white;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  font-weight: 500;
}

.variable-icon {
  font-size: 0.75rem;
  opacity: 0.8;
}

.variable-name {
  flex: 1;
}

.remove-variable {
  background: transparent;
  border: none;
  color: white;
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
  padding: 0 0.25rem;
  opacity: 0.8;
  transition: opacity 0.2s;
}

.remove-variable:hover {
  opacity: 1;
}

.grafico-container {
  margin-top: 2rem;
  padding: 1.5rem;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.grafico-header {
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 2px solid #e5e7eb;
}

.grafico-titulo {
  font-size: 0.9rem;
  font-weight: 700;
  color: #1f2937;
  margin: 0;
}

.grafico-principal {
  width: 100%;
  height: 300px;
  margin-bottom: 1rem;
  position: relative;
}

.grafico-svg {
  width: 100%;
  height: 100%;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
  background: #fafafa;
}

.punto-interactivo {
  cursor: pointer;
  transition: r 0.2s;
}

.punto-interactivo:hover {
  r: 7;
}

.tooltip-grafico {
  position: absolute;
  pointer-events: none;
  transform: translate(-50%, -100%);
  margin-top: -10px;
  z-index: 1000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translate(-50%, -90%);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -100%);
  }
}

.tooltip-content {
  background: rgba(30, 41, 59, 0.95);
  color: white;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  min-width: 200px;
  backdrop-filter: blur(10px);
}

.tooltip-variable {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.5rem;
  color: #60a5fa;
}

.tooltip-fecha {
  font-size: 0.8125rem;
  color: #cbd5e1;
  margin-bottom: 0.5rem;
}

.tooltip-valor {
  font-size: 0.875rem;
  color: white;
}

.tooltip-valor strong {
  font-weight: 700;
  color: #fbbf24;
}

.grafico-leyenda {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 1rem;
  border-top: 1px solid #e5e7eb;
  gap: 1rem;
}

.leyenda-items {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  flex: 1;
}

.leyenda-acciones {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.botones-accion {
  display: flex;
  gap: 0.5rem;
}

.leyenda-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.leyenda-color {
  width: 20px;
  height: 3px;
  border-radius: 2px;
}

.leyenda-texto {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #374151;
}

.mostrar-data {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: #4b5563;
  cursor: pointer;
  white-space: nowrap;
}

.mostrar-data input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.tabla-datos {
  margin-top: 1.5rem;
  overflow-x: auto;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}

.data-table thead {
  background-color: #f9fafb;
  border-bottom: 2px solid #e5e7eb;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-weight: 700;
  color: #374151;
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}

.data-table tbody tr {
  border-bottom: 1px solid #e5e7eb;
  transition: background-color 0.2s;
}

.data-table tbody tr:hover {
  background-color: #f9fafb;
}

.data-table td {
  padding: 0.75rem 1rem;
  color: #4b5563;
}

.variable-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 250px;
  overflow-y: auto;
  padding: 0.5rem;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 0.375rem;
}

.variable-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  cursor: pointer;
  border-radius: 0.25rem;
  transition: background-color 0.2s;
  font-size: 0.8125rem;
  color: #374151;
}

.variable-checkbox:hover {
  background-color: #f3f4f6;
}

.variable-checkbox input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
  flex-shrink: 0;
}

.variable-checkbox span {
  flex: 1;
}

.fecha-slider-box {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.campo-fecha {
  width: 100%;
}
.tiempo-display {
  font-size: 1.5rem;
  font-weight: 700;
  color: #001f54;
  letter-spacing: 0.12em;
  text-align: center;
  padding: 0.1rem 0;
}
.slider-fila {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.slider-label {
  font-size: 0.72rem;
  color: #6b7280;
  width: 2.2rem;
  text-align: right;
  flex-shrink: 0;
}
.slider-val {
  font-size: 0.8rem;
  font-weight: 600;
  color: #001f54;
  width: 1.8rem;
  text-align: center;
  flex-shrink: 0;
}
.slider-tiempo {
  flex: 1;
  accent-color: #001f54;
  cursor: pointer;
}
</style>
