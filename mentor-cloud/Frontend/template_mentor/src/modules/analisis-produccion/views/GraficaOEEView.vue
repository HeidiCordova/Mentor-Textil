<template>
  <div class="grafica-oee">
    <div class="titulo-principal">GRAFICA DE OEE</div>

    <div class="formulario-grid">
      <div class="seccion-box">
        <h3 class="subtitulo-seccion">FECHAS</h3>
        <div class="campo-grupo">
          <label>Inicio</label>
          <div class="fecha-slider-box">
            <input type="date" v-model="inicioFecha" class="campo-entrada campo-fecha" />
            <div class="tiempo-display">{{ inicioHora }}:{{ inicioMinuto }}</div>
            <div class="slider-fila">
              <span class="slider-label">Hora</span>
              <input type="range" min="0" max="23" step="1" v-model.number="inicioHoraNum" class="slider-tiempo" />
              <span class="slider-val">{{ inicioHora }}</span>
            </div>
            <div class="slider-fila">
              <span class="slider-label">Min</span>
              <input type="range" min="0" :max="intervalConfig.sliderMax" step="1" v-model.number="inicioMin5Idx" class="slider-tiempo" />
              <span class="slider-val">{{ inicioMinuto }}</span>
            </div>
          </div>
        </div>
        <div class="campo-grupo">
          <label>Fin</label>
          <div class="fecha-slider-box">
            <input type="date" v-model="finFecha" class="campo-entrada campo-fecha" />
            <div class="tiempo-display">{{ finHora }}:{{ finMinuto }}</div>
            <div class="slider-fila">
              <span class="slider-label">Hora</span>
              <input type="range" min="0" max="23" step="1" v-model.number="finHoraNum" class="slider-tiempo" />
              <span class="slider-val">{{ finHora }}</span>
            </div>
            <div class="slider-fila">
              <span class="slider-label">Min</span>
              <input type="range" min="0" :max="intervalConfig.sliderMax" step="1" v-model.number="finMin5Idx" class="slider-tiempo" />
              <span class="slider-val">{{ finMinuto }}</span>
            </div>
          </div>
        </div>

        <h3 class="subtitulo-seccion">TIPO DE GRAFICO</h3>
        <div class="campo-grupo campo-grupo-botones">
          <button
            @click="tipoGrafica = 'barras'"
            :class="['btn-tipo-grafico', { active: tipoGrafica === 'barras' }]"
          >Barras</button>
          <button
            @click="tipoGrafica = 'lineal'"
            :class="['btn-tipo-grafico', { active: tipoGrafica === 'lineal' }]"
          >Lineal</button>
        </div>

        <h3 class="subtitulo-seccion">TIPO DE CONSULTA</h3>
        <div class="grid-consulta">
          <button
            @click="tipoConsulta = 'consulta'"
            :class="['btn-consulta', { active: tipoConsulta === 'consulta' }]"
          >Por Consulta</button>
          <button
            @click="tipoConsulta = 'lote'"
            :class="['btn-consulta', { active: tipoConsulta === 'lote' }]"
          >Por Lote</button>
          <button
            @click="tipoConsulta = 'turno'"
            :class="['btn-consulta', { active: tipoConsulta === 'turno' }]"
          >Por Turno</button>
          <button class="btn-consulta disabled" disabled>
            Por Consulta Acumulativa
            <span class="badge-dev">En desarrollo</span>
          </button>
          <button class="btn-consulta disabled" disabled>
            Por Lote Acumulativo
            <span class="badge-dev">En desarrollo</span>
          </button>
          <button class="btn-consulta disabled" disabled>
            Por Turno Acumulativo
            <span class="badge-dev">En desarrollo</span>
          </button>
          <button class="btn-consulta disabled" disabled>
            Por Turno Variables Base
            <span class="badge-dev">En desarrollo</span>
          </button>
        </div>
      </div>

      <div class="seccion-box">
        <h3 class="subtitulo-seccion">DISPOSITIVO</h3>

        <h4 class="subseccion-label">Compania</h4>
        <div class="campo-grupo">
          <select v-model="empresaId" class="campo-entrada" @change="onEmpresaChange">
            <option :value="null" disabled>Seleccionar</option>
            <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
          </select>
        </div>

        <h4 class="subseccion-label">Planta</h4>
        <div class="campo-grupo">
          <select v-model="plantaId" class="campo-entrada" @change="onPlantaChange">
            <option :value="null" disabled>Seleccionar</option>
            <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
          </select>
        </div>

        <h4 class="subseccion-label">Linea</h4>
        <div class="campo-grupo">
          <select v-model="lineaId" class="campo-entrada">
            <option :value="null">Todas</option>
            <option v-for="l in lineasFiltradas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
          </select>
        </div>
      </div>

      <div class="seccion-box">
        <h3 class="subtitulo-seccion">{{ tituloVariables }}</h3>

        <div v-if="!consultaRealizada" class="vars-placeholder">
          Realiza una consulta para ver las variables disponibles.
        </div>

        <div v-else-if="columnasVariables.length === 0" class="vars-placeholder">
          No se encontraron variables en los datos.
        </div>

        <template v-else>
          <div v-for="col in columnasVariables" :key="col" class="campo-grupo">
            <div class="variable-caja">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#3b82f6" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
              </svg>
              {{ col }}
            </div>
          </div>
        </template>
      </div>
    </div>

    <div class="seccion-buscar">
      <button @click="buscar" class="btn-buscar-principal" :disabled="cargando">
        {{ cargando ? 'CARGANDO...' : 'BUSCAR' }}
      </button>
    </div>

    <div v-if="cargando" class="estado-carga">
      <svg class="spinner" width="40" height="40" viewBox="0 0 50 50">
        <circle cx="25" cy="25" r="20" fill="none" stroke="#001f54" stroke-width="4" stroke-dasharray="80 40" stroke-linecap="round"/>
      </svg>
    </div>

    <div v-if="error" class="estado-error">{{ error }}</div>

    <div v-if="!cargando && snapshots.length > 0" class="contenedor-visualizacion">
      <div class="indicador-oee-principal">
        <div class="indicador-content">
          <div class="indicador-label">OEE Global</div>
          <div class="indicador-valor" :style="{ color: getColorOEE(resumen.oee) }">
            {{ resumen.oee.toFixed(1) }}%
          </div>
          <div class="indicador-estado" :class="getClaseOEE(resumen.oee)">
            {{ getEstadoOEE(resumen.oee) }}
          </div>
        </div>
        <div class="indicador-visual">
          <svg viewBox="0 0 200 200" class="grafico-circular">
            <circle cx="100" cy="100" r="80" fill="none" stroke="#e5e7eb" stroke-width="20"/>
            <circle cx="100" cy="100" r="80" fill="none"
              :stroke="getColorOEE(resumen.oee)" stroke-width="20"
              :stroke-dasharray="`${(resumen.oee / 100) * 502.4} 502.4`"
              transform="rotate(-90 100 100)" stroke-linecap="round"/>
          </svg>
        </div>
      </div>

      <div class="cards-componentes">
        <div class="card-componente">
          <div class="componente-header">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#3b82f6" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
            </svg>
            <span>Disponibilidad</span>
          </div>
          <div class="componente-valor">{{ resumen.disponibilidad.toFixed(1) }}%</div>
          <div class="componente-barra">
            <div class="barra-progreso" :style="{ width: Math.min(resumen.disponibilidad, 100) + '%', background: '#3b82f6' }"></div>
          </div>
          <div class="componente-formula">T_Operativo / T_Disponible</div>
        </div>
        <div class="card-componente">
          <div class="componente-header">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#8b5cf6" stroke-width="2">
              <path d="M13 2L3 14h8l-1 8 10-12h-8l1-8z"/>
            </svg>
            <span>Rendimiento</span>
          </div>
          <div class="componente-valor">{{ resumen.rendimiento.toFixed(1) }}%</div>
          <div class="componente-barra">
            <div class="barra-progreso" :style="{ width: Math.min(resumen.rendimiento, 100) + '%', background: '#8b5cf6' }"></div>
          </div>
          <div class="componente-formula">T_Ciclo_Nominal / T_Operativo</div>
        </div>
        <div class="card-componente">
          <div class="componente-header">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2">
              <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <span>Calidad</span>
          </div>
          <div class="componente-valor">{{ resumen.calidad.toFixed(1) }}%</div>
          <div class="componente-barra">
            <div class="barra-progreso" :style="{ width: Math.min(resumen.calidad, 100) + '%', background: '#10b981' }"></div>
          </div>
          <div class="componente-formula">(Conteo - Merma) / Conteo</div>
        </div>
      </div>

      <div class="grafico-principal">
        <div class="grafico-header">
          <h3>Analisis Detallado OEE</h3>
          <div class="grafico-header-actions">
            <span class="snapshots-count">{{ resumen.snapshots }} registros</span>
            <button @click="exportarExcel" class="btn-exportar">Exportar Excel</button>
          </div>
        </div>

        <div v-if="tipoGrafica === 'barras'" class="grafico-container">
          <svg :viewBox="`0 0 ${svgWidth} 400`" preserveAspectRatio="xMidYMid meet">
            <g v-for="i in 5" :key="'grid-' + i">
              <line :x1="80" :y1="gridY(i)" :x2="svgWidth - 50" :y2="gridY(i)" stroke="#e5e7eb" stroke-width="1"/>
              <text :x="60" :y="gridY(i) + 5" text-anchor="end" fill="#6b7280" font-size="12">{{ gridLabel(i) }}%</text>
            </g>
            <g v-for="(d, i) in datosGrafico" :key="'barra-' + i">
              <rect :x="barX(i)" :y="barY(d.oee)" :width="barWidth" :height="barH(d.oee)"
                :fill="getColorOEE(d.oee)" class="barra-oee"
                @mouseenter="mostrarTooltip($event, d)" @mouseleave="ocultarTooltip"/>
              <text :x="barX(i) + barWidth / 2" y="355" text-anchor="middle" fill="#6b7280" font-size="10">
                {{ d.label }}
              </text>
            </g>
            <line :x1="80" y1="330" :x2="svgWidth - 50" y2="330" stroke="#9ca3af" stroke-width="2"/>
          </svg>
        </div>

        <div v-else class="grafico-container">
          <svg :viewBox="`0 0 ${svgWidth} 400`" preserveAspectRatio="xMidYMid meet">
            <g v-for="i in 5" :key="'grid-' + i">
              <line :x1="80" :y1="gridY(i)" :x2="svgWidth - 50" :y2="gridY(i)" stroke="#e5e7eb" stroke-width="1"/>
              <text :x="60" :y="gridY(i) + 5" text-anchor="end" fill="#6b7280" font-size="12">{{ gridLabel(i) }}%</text>
            </g>
            <path :d="generarPath('oee')" stroke="#001f54" stroke-width="3" fill="none" stroke-linecap="round"/>
            <path :d="generarPath('disponibilidad')" stroke="#3b82f6" stroke-width="2" fill="none" stroke-dasharray="6,3"/>
            <path :d="generarPath('rendimiento')" stroke="#8b5cf6" stroke-width="2" fill="none" stroke-dasharray="6,3"/>
            <path :d="generarPath('calidad')" stroke="#10b981" stroke-width="2" fill="none" stroke-dasharray="6,3"/>
            <g v-for="(d, i) in datosGrafico" :key="'pt-' + i">
              <circle :cx="lineX(i)" :cy="lineY(d.oee)" r="5" fill="#001f54" class="punto-grafico"
                @mouseenter="mostrarTooltip($event, d)" @mouseleave="ocultarTooltip"/>
            </g>
            <g v-for="(d, i) in datosGrafico" :key="'lbl-' + i">
              <text v-if="i % labelStep === 0" :x="lineX(i)" y="355" text-anchor="middle" fill="#6b7280" font-size="10">
                {{ d.label }}
              </text>
            </g>
            <line :x1="80" y1="330" :x2="svgWidth - 50" y2="330" stroke="#9ca3af" stroke-width="2"/>
          </svg>
          <div class="grafico-leyenda">
            <span class="leyenda-item"><span class="leyenda-linea" style="background:#001f54"></span> OEE</span>
            <span class="leyenda-item"><span class="leyenda-linea" style="background:#3b82f6"></span> Disponibilidad</span>
            <span class="leyenda-item"><span class="leyenda-linea" style="background:#8b5cf6"></span> Rendimiento</span>
            <span class="leyenda-item"><span class="leyenda-linea" style="background:#10b981"></span> Calidad</span>
          </div>
        </div>
      </div>

      <div class="tabla-container">
        <table class="tabla-datos">
          <thead>
            <tr>
              <th>Fecha / Hora</th>
              <th>OEE (%)</th>
              <th>Disponibilidad (%)</th>
              <th>Rendimiento (%)</th>
              <th>Calidad (%)</th>
              <th>Produccion</th>
              <th v-for="col in columnasVariables" :key="col">{{ col }}</th>
              <th>Estado</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in snapshotsPagina" :key="s.id">
              <td>{{ formatHora(s.hora) }}</td>
              <td><span :class="['badge-oee', getClaseOEE(s.oee)]">{{ s.oee.toFixed(1) }}%</span></td>
              <td>{{ s.disponibilidad.toFixed(1) }}%</td>
              <td>{{ s.rendimiento.toFixed(1) }}%</td>
              <td>{{ s.calidad.toFixed(1) }}%</td>
              <td>{{ s.produccion }}</td>
              <td v-for="col in columnasVariables" :key="col">{{ getVar(s, col) }}</td>
              <td><span :class="['badge-estado', getClaseOEE(s.oee)]">{{ getEstadoOEE(s.oee) }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="totalPaginas > 1" class="paginacion">
        <button class="pag-btn" :disabled="paginaActual === 1" @click="paginaActual = 1">&laquo;</button>
        <button class="pag-btn" :disabled="paginaActual === 1" @click="paginaActual--">&lsaquo;</button>
        <span class="pag-info">Pág. {{ paginaActual }} / {{ totalPaginas }} &nbsp;&mdash;&nbsp; {{ snapshots.length }} registros</span>
        <button class="pag-btn" :disabled="paginaActual === totalPaginas" @click="paginaActual++">&#x203A;</button>
        <button class="pag-btn" :disabled="paginaActual === totalPaginas" @click="paginaActual = totalPaginas">&raquo;</button>
      </div>
      <div v-else-if="consultaRealizada && snapshots.length > 0" class="pag-total">
        {{ snapshots.length }} registros
      </div>
    </div>

    <div v-if="!cargando && consultaRealizada && snapshots.length === 0" class="estado-vacio">
      No se encontraron registros OEE para los filtros seleccionados.
    </div>

    <Teleport to="body">
      <div v-if="tooltipVisible" class="tooltip-oee" :style="{ left: tooltipX + 'px', top: tooltipY + 'px' }">
        <div class="tooltip-titulo">{{ tooltipData.label }}</div>
        <div class="tooltip-linea"><strong>OEE:</strong> {{ tooltipData.oee?.toFixed(1) }}%</div>
        <div class="tooltip-linea"><strong>Disponibilidad:</strong> {{ tooltipData.disponibilidad?.toFixed(1) }}%</div>
        <div class="tooltip-linea"><strong>Rendimiento:</strong> {{ tooltipData.rendimiento?.toFixed(1) }}%</div>
        <div class="tooltip-linea"><strong>Calidad:</strong> {{ tooltipData.calidad?.toFixed(1) }}%</div>
        <div class="tooltip-linea"><strong>Produccion:</strong> {{ tooltipData.produccion ?? 0 }}</div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { oeeService } from '@/api/services/oee.service'

const empresas = ref([])
const plantas = ref([])
const lineas = ref([])

const empresaId = ref(null)
const plantaId = ref(null)
const lineaId = ref(null)
const tipoGrafica = ref('barras')
const tipoConsulta = ref('consulta')

const lineaMode = computed(() => lineas.value.find(l => l.id === lineaId.value)?.mode ?? 'textil')
const intervalConfig = computed(() => ({
  msStep: 1800000,
  sliderMax: 1,
  minuteMultiplier: 30,
  divisor: 30,
  minIntervalS: 1800
}))

const pad2 = n => String(n).padStart(2, '0')

function roundDownInterval(d) {
  const ms = d.getTime()
  return new Date(ms - (ms % intervalConfig.value.msStep))
}

function splitDate(d) {
  const r = roundDownInterval(d)
  return {
    fecha: `${r.getFullYear()}-${pad2(r.getMonth() + 1)}-${pad2(r.getDate())}`,
    horaNum: r.getHours(),
    min5Idx: Math.floor(r.getMinutes() / intervalConfig.value.divisor)
  }
}

const now = new Date()
const hace24h = new Date(now.getTime() - 24 * 60 * 60 * 1000)

const iniParts = splitDate(hace24h)
const finParts = splitDate(now)

const inicioFecha   = ref(iniParts.fecha)
const inicioHoraNum = ref(iniParts.horaNum)
const inicioMin5Idx = ref(iniParts.min5Idx)

const finFecha   = ref(finParts.fecha)
const finHoraNum = ref(finParts.horaNum)
const finMin5Idx = ref(finParts.min5Idx)

const inicioHora   = computed(() => pad2(inicioHoraNum.value))
const inicioMinuto = computed(() => pad2(inicioMin5Idx.value * intervalConfig.value.minuteMultiplier))
const finHora      = computed(() => pad2(finHoraNum.value))
const finMinuto    = computed(() => pad2(finMin5Idx.value * intervalConfig.value.minuteMultiplier))

watch(lineaMode, () => {
  inicioMin5Idx.value = Math.min(inicioMin5Idx.value, intervalConfig.value.sliderMax)
  finMin5Idx.value    = Math.min(finMin5Idx.value,    intervalConfig.value.sliderMax)
})

const fechaInicio = computed(() => `${inicioFecha.value}T${inicioHora.value}:${inicioMinuto.value}`)
const fechaFin    = computed(() => `${finFecha.value}T${finHora.value}:${finMinuto.value}`)

const snapshots = ref([])
const resumen = ref({ disponibilidad: 0, rendimiento: 0, calidad: 0, oee: 0, produccion_total: 0, snapshots: 0 })
const cargando = ref(false)
const error = ref(null)
const consultaRealizada = ref(false)

const tooltipVisible = ref(false)
const tooltipX = ref(0)
const tooltipY = ref(0)
const tooltipData = ref({})



const plantasFiltradas = computed(() => plantas.value.filter(p => p.empresa_id === empresaId.value))
const lineasFiltradas = computed(() => lineas.value.filter(l => l.planta_id === plantaId.value))

const POR_PAGINA = 50
const paginaActual = ref(1)
const totalPaginas = computed(() => Math.max(1, Math.ceil(snapshots.value.length / POR_PAGINA)))
const snapshotsPagina = computed(() => {
  const inicio = (paginaActual.value - 1) * POR_PAGINA
  return snapshots.value.slice(inicio, inicio + POR_PAGINA)
})

const columnasVariables = computed(() => {
  const seen = new Set()
  const cols = []
  snapshots.value.forEach(s => {
    (s.head || []).forEach(k => {
      if (!seen.has(k)) { seen.add(k); cols.push(k) }
    })
  })
  return cols
})

const TITULOS_CONSULTA = {
  consulta:            'VARIABLES POR CONSULTA',
  consultaAcumulativa: 'VARIABLES ACUMULATIVAS POR CONSULTA',
  lote:                'VARIABLES POR LOTE',
  loteAcumulativa:     'VARIABLES ACUMULATIVAS POR LOTE',
  turno:               'VARIABLES POR TURNO',
  turnoAcumulativo:    'VARIABLES ACUMULATIVAS POR TURNO',
  turnoVariablesBase:  'VARIABLES BASE POR TURNO',
}
const tituloVariables = computed(() => TITULOS_CONSULTA[tipoConsulta.value] || 'VARIABLES')

function onEmpresaChange() {
  plantaId.value = null
  lineaId.value = null
}

function onPlantaChange() {
  lineaId.value = null
}

watch(empresaId, () => {
  const match = plantasFiltradas.value
  if (match.length === 1) plantaId.value = match[0].id
})

const datosGrafico = computed(() => {
  return [...snapshots.value].reverse().map(s => ({
    id: s.id,
    label: formatHoraCorta(s.hora),
    oee: s.oee,
    disponibilidad: s.disponibilidad,
    rendimiento: s.rendimiento,
    calidad: s.calidad,
    produccion: s.produccion
  }))
})

const svgWidth = computed(() => Math.max(1200, datosGrafico.value.length * 30))
const chartW = computed(() => svgWidth.value - 130)
const barWidth = computed(() => Math.max(4, (chartW.value / datosGrafico.value.length) - 4))
const labelStep = computed(() => Math.max(1, Math.ceil(datosGrafico.value.length / 12)))

function gridY(i) { return 50 + (i - 1) * 70 }
function gridLabel(i) { return 100 - (i - 1) * 25 }
function barX(i) { return 80 + i * (chartW.value / datosGrafico.value.length) + 2 }
function barY(v) { return 330 - (v * 2.8) }
function barH(v) { return Math.max(0, v * 2.8) }
function lineX(i) {
  const len = datosGrafico.value.length
  return len <= 1 ? 80 : 80 + i * (chartW.value / (len - 1))
}
function lineY(v) { return 330 - (v * 2.8) }

function generarPath(metrica) {
  const d = datosGrafico.value
  if (d.length === 0) return ''
  return 'M ' + d.map((p, i) => `${lineX(i)},${lineY(p[metrica])}`).join(' L ')
}

function formatHora(iso) {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`
}

function formatHoraCorta(iso) {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

function fechaDisplay(ymd) {
  if (!ymd) return ''
  const [y, m, d] = ymd.split('-')
  return `${d}/${m}/${y}`
}

function getColorOEE(v) {
  if (v >= 85) return '#22c55e'
  if (v >= 60) return '#f59e0b'
  return '#ef4444'
}

function getClaseOEE(v) {
  if (v >= 85) return 'excelente'
  if (v >= 60) return 'aceptable'
  return 'critico'
}

function getEstadoOEE(v) {
  if (v >= 85) return 'EXCELENTE'
  if (v >= 60) return 'ACEPTABLE'
  return 'CRITICO'
}

function mostrarTooltip(event, dato) {
  tooltipData.value = dato
  tooltipX.value = event.clientX + 12
  tooltipY.value = event.clientY + 12
  tooltipVisible.value = true
}

function ocultarTooltip() { tooltipVisible.value = false }

async function cargarMaestros() {
  try {
    const [empRes, plnRes, linRes] = await Promise.all([
      companyService.getAll(),
      plantService.getAll(),
      lineService.getAll()
    ])
    empresas.value = empRes.data?.data || empRes.data || []
    plantas.value = plnRes.data?.data || plnRes.data || []
    lineas.value = linRes.data?.data || linRes.data || []

    if (empresas.value.length > 0 && !empresaId.value) {
      empresaId.value = empresas.value[0].id
    }
  } catch (e) {
    error.value = 'Error al cargar datos maestros'
  }
}

async function buscar() {
  if (!empresaId.value) { error.value = 'Seleccione una empresa'; return }
  cargando.value = true
  error.value = null
  consultaRealizada.value = true
  paginaActual.value = 1

  const params = { empresa_id: empresaId.value, limit: 500, min_interval_s: intervalConfig.value.minIntervalS }
  if (plantaId.value) params.planta_id = plantaId.value
  if (lineaId.value) params.linea_id = lineaId.value
  if (fechaInicio.value) params.from = new Date(fechaInicio.value).toISOString()
  if (fechaFin.value) params.to = new Date(fechaFin.value).toISOString()

  try {
    const [snapRes, sumRes] = await Promise.all([
      oeeService.getSnapshots(params),
      oeeService.getSummary(params)
    ])
    snapshots.value = snapRes.data?.data || snapRes.data || []
    const s = sumRes || {}
    resumen.value = {
      disponibilidad: s.disponibilidad || 0,
      rendimiento: s.rendimiento || 0,
      calidad: s.calidad || 0,
      oee: s.oee || 0,
      produccion_total: s.produccion_total || 0,
      snapshots: s.snapshots || 0
    }
  } catch (e) {
    error.value = 'Error al consultar datos OEE'
    snapshots.value = []
  } finally {
    cargando.value = false
  }
}

const EXCEL_NS = 'urn:schemas-microsoft-com:office:spreadsheet'

function xmlHead(sheetName) {
  return `<?xml version="1.0"?><?mso-application progid="Excel.Sheet"?>` +
    `<Workbook xmlns="${EXCEL_NS}" xmlns:ss="${EXCEL_NS}">` +
    `<Styles><Style ss:ID="h"><Font ss:Bold="1" ss:Color="#FFFFFF"/>` +
    `<Interior ss:Color="#001f54" ss:Pattern="Solid"/>` +
    `<Alignment ss:Horizontal="Center"/></Style></Styles>` +
    `<Worksheet ss:Name="${sheetName}"><Table>`
}

function xmlFoot() { return '</Table></Worksheet></Workbook>' }

function xmlRow(cells) {
  return '<Row>' + cells.map(c => {
    const style = c.style ? ` ss:StyleID="${c.style}"` : ''
    const type = c.type || 'String'
    return `<Cell${style}><Data ss:Type="${type}">${c.v}</Data></Cell>`
  }).join('') + '</Row>'
}

function descargar(xml, nombre) {
  const blob = new Blob([xml], { type: 'application/vnd.ms-excel' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = nombre
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function promedioGrupo(grupo) {
  const n = grupo.length
  return {
    oee: grupo.reduce((a, s) => a + s.oee, 0) / n,
    disponibilidad: grupo.reduce((a, s) => a + s.disponibilidad, 0) / n,
    rendimiento: grupo.reduce((a, s) => a + s.rendimiento, 0) / n,
    calidad: grupo.reduce((a, s) => a + s.calidad, 0) / n,
    produccion: grupo.reduce((a, s) => a + s.produccion, 0)
  }
}

function agruparPorClave(claveFn) {
  const mapa = new Map()
  snapshots.value.forEach(s => {
    const k = claveFn(s)
    if (!mapa.has(k)) mapa.set(k, [])
    mapa.get(k).push(s)
  })
  return mapa
}

function exportarExcel() {
  if (snapshots.value.length === 0) return
  const fecha = new Date().toISOString().split('T')[0]
  const tipo = tipoConsulta.value
  if (tipo === 'consulta') exportPorConsulta(fecha)
  else if (tipo === 'consultaAcumulativa') exportConsultaAcumulativa(fecha)
  else if (tipo === 'lote') exportPorLote(fecha)
  else if (tipo === 'loteAcumulativa') exportLoteAcumulativo(fecha)
  else if (tipo === 'turno') exportPorTurno(fecha)
  else if (tipo === 'turnoAcumulativo') exportTurnoAcumulativo(fecha)
  else if (tipo === 'turnoVariablesBase') exportTurnoVariablesBase(fecha)
}

function exportPorConsulta(fecha) {
  const lineaNombre = lineasFiltradas.value.find(l => l.id === lineaId.value)?.nombre || 'Todas'

  const hdrs = [
    'Fecha Inicio', 'Hora Inicio', 'Fecha Fin', 'Hora Fin', 'Linea', 'Turno',
    'OEE', 'Disponibilidad', 'Rendimiento', 'Calidad',
    'Tiempo Disponible Total',
    'Tiempo de Parada Obligatoria',
    'Tiempo de Parada no Obligatoria',
    'Tiempo de Microparadas',
    'Tiempo Nominal de Produccion',
    'Tiempo Perdido por Baja Velocidad',
    'Produccion',
    'Mermas'
  ]

  let xml = xmlHead('Por Consulta')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))

  snapshots.value.forEach(s => {
    const lineaRow = lineas.value.find(l => l.id === s.linea_id)?.nombre || lineaNombre
    xml += xmlRow([
      { v: fechaDisplay(s.fecha) },
      { v: formatHora(s.hora) },
      { v: fechaDisplay(s.fecha) },
      { v: formatHora(s.hora) },
      { v: lineaRow },
      { v: s.turno || 'Sin Turno' },
      { v: s.oee.toFixed(2),            type: 'Number' },
      { v: s.disponibilidad.toFixed(2), type: 'Number' },
      { v: s.rendimiento.toFixed(2),    type: 'Number' },
      { v: s.calidad.toFixed(2),        type: 'Number' },
      { v: getVar(s, 'T_DISPONIBLE'),          type: 'Number' },
      { v: getVar(s, 'T_PARADA_PROGRAMADA'),   type: 'Number' },
      { v: getVar(s, 'T_PARADA_NO_PROGRAMADA'), type: 'Number' },
      { v: getVar(s, 'T_MICROPARADA'),         type: 'Number' },
      { v: getVar(s, 'T_NOMINAL'),             type: 'Number' },
      { v: getVar(s, 'T_BAJA_VELOCIDAD'),      type: 'Number' },
      { v: s.produccion,                type: 'Number' },
      { v: getVar(s, 'MERMA'),                 type: 'Number' }
    ])
  })

  xml += xmlFoot()
  descargar(xml, `OEE_Consulta_${fecha}.xls`)
}

function exportConsultaAcumulativa(fecha) {
  const hdrs = ['Fecha/Hora', 'OEE (%)', 'Disponibilidad (%)', 'Rendimiento (%)', 'Calidad (%)', 'Produccion', 'Produccion Acumulada', 'Estado']
  let xml = xmlHead('Consulta Acumulativa')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))
  let acum = 0
  snapshots.value.forEach(s => {
    acum += s.produccion
    xml += xmlRow([
      { v: formatHora(s.hora) },
      { v: s.oee.toFixed(2), type: 'Number' },
      { v: s.disponibilidad.toFixed(2), type: 'Number' },
      { v: s.rendimiento.toFixed(2), type: 'Number' },
      { v: s.calidad.toFixed(2), type: 'Number' },
      { v: s.produccion, type: 'Number' },
      { v: acum, type: 'Number' },
      { v: getEstadoOEE(s.oee) }
    ])
  })
  xml += xmlFoot()
  descargar(xml, `OEE_ConsultaAcum_${fecha}.xls`)
}

function getVar(s, nombre) {
  const idx = s.head?.indexOf(nombre)
  if (idx === undefined || idx === -1) return 0
  return s.data?.[idx] ?? 0
}

function sumarVar(grupo, nombre) {
  return grupo.reduce((a, s) => a + getVar(s, nombre), 0)
}

const VARS_FIJAS = [
  { key: 'T_DISPONIBLE',          label: 'T. Disponible (s)' },
  { key: 'T_PARADA_PROGRAMADA',   label: 'T. Parada Obligatoria (s)' },
  { key: 'T_PARADA_NO_PROGRAMADA', label: 'T. Parada No Obligatoria (s)' },
  { key: 'T_MICROPARADA',         label: 'T. Microparadas (s)' },
  { key: 'T_NOMINAL',             label: 'T. Nominal Produccion (s)' },
  { key: 'T_BAJA_VELOCIDAD',      label: 'T. Perdido Baja Velocidad (s)' },
  { key: 'MERMA',                 label: 'Mermas' },
]

function exportPorLote(fecha) {
  const lineaNombre = lineasFiltradas.value.find(l => l.id === lineaId.value)?.nombre || 'Todas'

  const hdrs = [
    'Fecha Inicio', 'Hora Inicio', 'Fecha Fin', 'Hora Fin', 'Linea', 'Lote',
    'OEE', 'Disponibilidad', 'Rendimiento', 'Calidad',
    'Tiempo Disponible Total',
    'Tiempo de Parada Obligatoria',
    'Tiempo de Parada no Obligatoria',
    'Tiempo de Microparadas',
    'Tiempo Nominal de Produccion',
    'Tiempo Perdido por Baja Velocidad',
    'Produccion',
    'Mermas'
  ]

  let xml = xmlHead('Por Lote')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))

  agruparPorClave(s => s.run_nombre || s.turno || s.fecha).forEach((grupo, clave) => {
    const p = promedioGrupo(grupo)
    const primero = grupo[0]
    const ultimo = grupo[grupo.length - 1]
    const lineaRow = lineas.value.find(l => l.id === primero.linea_id)?.nombre || lineaNombre

    xml += xmlRow([
      { v: fechaDisplay(primero.fecha) },
      { v: formatHora(primero.hora) },
      { v: fechaDisplay(ultimo.fecha) },
      { v: formatHora(ultimo.hora) },
      { v: lineaRow },
      { v: clave },
      { v: p.oee.toFixed(2),          type: 'Number' },
      { v: p.disponibilidad.toFixed(2), type: 'Number' },
      { v: p.rendimiento.toFixed(2),  type: 'Number' },
      { v: p.calidad.toFixed(2),      type: 'Number' },
      { v: sumarVar(grupo, 'T_DISPONIBLE'),          type: 'Number' },
      { v: sumarVar(grupo, 'T_PARADA_PROGRAMADA'),   type: 'Number' },
      { v: sumarVar(grupo, 'T_PARADA_NO_PROGRAMADA'), type: 'Number' },
      { v: sumarVar(grupo, 'T_MICROPARADA'),         type: 'Number' },
      { v: sumarVar(grupo, 'T_NOMINAL'),             type: 'Number' },
      { v: sumarVar(grupo, 'T_BAJA_VELOCIDAD'),      type: 'Number' },
      { v: p.produccion,              type: 'Number' },
      { v: sumarVar(grupo, 'MERMA'),                 type: 'Number' }
    ])
  })

  xml += xmlFoot()
  descargar(xml, `OEE_Lote_${fecha}.xls`)
}

function exportLoteAcumulativo(fecha) {
  const hdrs = ['Fecha', 'OEE (%) Prom', 'Disponibilidad (%)', 'Rendimiento (%)', 'Calidad (%)', 'Produccion Dia', 'Produccion Acumulada', 'Estado']
  let xml = xmlHead('Lote Acumulativo')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))
  let acum = 0
  agruparPorClave(s => s.fecha).forEach((grupo, clave) => {
    const p = promedioGrupo(grupo)
    acum += p.produccion
    xml += xmlRow([
      { v: clave },
      { v: p.oee.toFixed(2), type: 'Number' },
      { v: p.disponibilidad.toFixed(2), type: 'Number' },
      { v: p.rendimiento.toFixed(2), type: 'Number' },
      { v: p.calidad.toFixed(2), type: 'Number' },
      { v: p.produccion, type: 'Number' },
      { v: acum, type: 'Number' },
      { v: getEstadoOEE(p.oee) }
    ])
  })
  xml += xmlFoot()
  descargar(xml, `OEE_LoteAcum_${fecha}.xls`)
}

function exportPorTurno(fecha) {
  const lineaNombre = lineas.value.find(l => l.id === lineaId.value)?.nombre || 'Todas'

  const hdrs = [
    'Fecha Inicio', 'Hora Inicio', 'Fecha Fin', 'Hora Fin', 'Linea', 'Turno',
    'OEE', 'Disponibilidad', 'Rendimiento', 'Calidad',
    'Tiempo Disponible Total',
    'Tiempo de Parada Obligatoria',
    'Tiempo de Parada No Obligatoria',
    'Tiempo de Microparadas',
    'Tiempo Nominal de Produccion',
    'Tiempo Perdido por Baja Velocidad',
    'Produccion',
    'Mermas'
  ]

  let xml = xmlHead('Por Turno')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))

  agruparPorClave(s => s.turno || 'Sin Turno').forEach((grupo, clave) => {
    const p = promedioGrupo(grupo)
    const primero = grupo[0]
    const ultimo = grupo[grupo.length - 1]
    const lineaRow = lineas.value.find(l => l.id === primero.linea_id)?.nombre || lineaNombre

    xml += xmlRow([
      { v: fechaDisplay(primero.fecha) },
      { v: formatHora(primero.hora) },
      { v: fechaDisplay(ultimo.fecha) },
      { v: formatHora(ultimo.hora) },
      { v: lineaRow },
      { v: clave },
      { v: p.oee.toFixed(2),            type: 'Number' },
      { v: p.disponibilidad.toFixed(2), type: 'Number' },
      { v: p.rendimiento.toFixed(2),    type: 'Number' },
      { v: p.calidad.toFixed(2),        type: 'Number' },
      { v: sumarVar(grupo, 'T_DISPONIBLE'),           type: 'Number' },
      { v: sumarVar(grupo, 'T_PARADA_PROGRAMADA'),    type: 'Number' },
      { v: sumarVar(grupo, 'T_PARADA_NO_PROGRAMADA'), type: 'Number' },
      { v: sumarVar(grupo, 'T_MICROPARADA'),          type: 'Number' },
      { v: sumarVar(grupo, 'T_NOMINAL'),              type: 'Number' },
      { v: sumarVar(grupo, 'T_BAJA_VELOCIDAD'),       type: 'Number' },
      { v: p.produccion,                type: 'Number' },
      { v: sumarVar(grupo, 'MERMA'),                  type: 'Number' }
    ])
  })

  xml += xmlFoot()
  descargar(xml, `OEE_Turno_${fecha}.xls`)
}

function exportTurnoAcumulativo(fecha) {
  const hdrs = ['Turno', 'OEE (%) Prom', 'Disponibilidad (%)', 'Rendimiento (%)', 'Calidad (%)', 'Produccion Turno', 'Produccion Acumulada', 'Estado']
  let xml = xmlHead('Turno Acumulativo')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))
  let acum = 0
  agruparPorClave(s => s.turno || 'Sin Turno').forEach((grupo, clave) => {
    const p = promedioGrupo(grupo)
    acum += p.produccion
    xml += xmlRow([
      { v: clave },
      { v: p.oee.toFixed(2), type: 'Number' },
      { v: p.disponibilidad.toFixed(2), type: 'Number' },
      { v: p.rendimiento.toFixed(2), type: 'Number' },
      { v: p.calidad.toFixed(2), type: 'Number' },
      { v: p.produccion, type: 'Number' },
      { v: acum, type: 'Number' },
      { v: getEstadoOEE(p.oee) }
    ])
  })
  xml += xmlFoot()
  descargar(xml, `OEE_TurnoAcum_${fecha}.xls`)
}

function exportTurnoVariablesBase(fecha) {
  const primer = snapshots.value.find(s => s.head?.length > 0)
  const head = primer?.head || []
  const hdrs = ['Turno', 'Fecha/Hora', 'OEE (%)', 'Disponibilidad (%)', 'Rendimiento (%)', 'Calidad (%)', 'Produccion', ...head]
  let xml = xmlHead('Turno Variables Base')
  xml += xmlRow(hdrs.map(v => ({ v, style: 'h' })))
  snapshots.value.forEach(s => {
    const vars = head.map((_, i) => ({ v: s.data?.[i] ?? 0, type: 'Number' }))
    xml += xmlRow([
      { v: s.turno || 'Sin Turno' },
      { v: formatHora(s.hora) },
      { v: s.oee.toFixed(2), type: 'Number' },
      { v: s.disponibilidad.toFixed(2), type: 'Number' },
      { v: s.rendimiento.toFixed(2), type: 'Number' },
      { v: s.calidad.toFixed(2), type: 'Number' },
      { v: s.produccion, type: 'Number' },
      ...vars
    ])
  })
  xml += xmlFoot()
  descargar(xml, `OEE_TurnoVarBase_${fecha}.xls`)
}

onMounted(cargarMaestros)
</script>

<style scoped>
.grafica-oee {
  padding: clamp(1rem, 2vw, 2rem);
  max-width: 1400px;
  margin: 0 auto;
}

.titulo-principal {
  background: #001f54;
  color: #fff;
  padding: 1.25rem;
  text-align: center;
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 2rem;
  border-radius: 8px;
  letter-spacing: 1px;
}

.formulario-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.25rem;
  margin-bottom: 2rem;
}

.seccion-box {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
}

.subtitulo-seccion {
  background: #001f54;
  color: #fff;
  padding: 0.625rem;
  text-align: center;
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0 0 1rem;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.subseccion-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #475569;
  margin: 0.75rem 0 0.375rem;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.campo-grupo {
  margin-bottom: 1rem;
}

.campo-grupo label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #374151;
  font-size: 0.875rem;
}

.campo-entrada {
  width: 100%;
  padding: 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.875rem;
  transition: border-color 0.2s;
}

.campo-entrada:focus {
  outline: none;
  border-color: #001f54;
  box-shadow: 0 0 0 3px rgba(0, 31, 84, 0.1);
}

.campo-grupo-botones {
  display: flex;
  gap: 0.5rem;
}

.btn-tipo-grafico {
  flex: 1;
  padding: 0.625rem;
  background: #fff;
  border: 1.5px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.875rem;
  font-weight: 500;
  color: #475569;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-tipo-grafico:hover {
  border-color: #001f54;
  color: #001f54;
}

.btn-tipo-grafico.active {
  background: #001f54;
  border-color: #001f54;
  color: #fff;
}

.grid-consulta {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.4rem;
}

.btn-consulta {
  position: relative;
  padding: 0.55rem 0.4rem;
  background: #fff;
  border: 1.5px solid #cbd5e1;
  border-radius: 6px;
  font-size: 0.78rem;
  font-weight: 500;
  color: #475569;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
  line-height: 1.3;
}

.btn-consulta:hover:not(.disabled) {
  border-color: #001f54;
  color: #001f54;
}

.btn-consulta.active {
  background: #001f54;
  border-color: #001f54;
  color: #fff;
}

.btn-consulta.disabled {
  background: #f1f5f9;
  border-color: #e2e8f0;
  color: #94a3b8;
  cursor: not-allowed;
  font-size: 0.73rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
}

.badge-dev {
  display: inline-block;
  background: #e2e8f0;
  color: #94a3b8;
  font-size: 0.62rem;
  font-weight: 600;
  padding: 0.1rem 0.35rem;
  border-radius: 99px;
  white-space: nowrap;
}

.variable-caja {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.8rem;
  color: #334155;
  background: #f8fafc;
  font-family: monospace;
  min-height: 36px;
}

.seccion-buscar {
  display: flex;
  justify-content: center;
  margin-bottom: 2rem;
}

.btn-buscar-principal {
  background: #001f54;
  color: #fff;
  border: none;
  padding: 0.875rem 3rem;
  border-radius: 6px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 2px 4px rgba(0, 31, 84, 0.2);
  letter-spacing: 0.5px;
}

.btn-buscar-principal:hover:not(:disabled) {
  background: #001238;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 31, 84, 0.3);
}

.btn-buscar-principal:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.estado-carga {
  display: flex;
  justify-content: center;
  padding: 3rem;
}

.spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.estado-error {
  text-align: center;
  color: #dc2626;
  padding: 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  margin-bottom: 1rem;
}

.estado-vacio {
  text-align: center;
  color: #6b7280;
  padding: 3rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.contenedor-visualizacion {
  margin-top: 2rem;
}

.indicador-oee-principal {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
}

.indicador-content { flex: 1; }

.indicador-label {
  font-size: 1.25rem;
  color: #6b7280;
  margin-bottom: 0.5rem;
}

.indicador-valor {
  font-size: 4rem;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 0.5rem;
}

.indicador-estado {
  font-size: 1.25rem;
  font-weight: 600;
  padding: 0.5rem 1.5rem;
  border-radius: 8px;
  display: inline-block;
}

.indicador-estado.excelente { background: #dcfce7; color: #166534; }
.indicador-estado.aceptable { background: #fef3c7; color: #92400e; }
.indicador-estado.critico { background: #fee2e2; color: #991b1b; }

.indicador-visual { width: 200px; height: 200px; }
.grafico-circular { width: 100%; height: 100%; }

.cards-componentes {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.card-componente {
  background: #fff;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.componente-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
  color: #374151;
  font-weight: 600;
}

.componente-valor {
  font-size: 2rem;
  font-weight: 700;
  color: #001f54;
  margin-bottom: 0.75rem;
}

.componente-barra {
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 0.75rem;
}

.barra-progreso {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.componente-formula {
  font-size: 0.75rem;
  color: #9ca3af;
  font-style: italic;
}

.grafico-principal {
  background: #fff;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
}

.grafico-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 2px solid #e5e7eb;
}

.grafico-header h3 {
  font-size: 1.25rem;
  color: #001f54;
  margin: 0;
}

.grafico-header-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.snapshots-count {
  font-size: 0.8rem;
  color: #6b7280;
}

.btn-exportar {
  padding: 0.5rem 1.5rem;
  background: #001f54;
  color: #fff;
  border: none;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
}

.btn-exportar:hover { background: #002a75; }

.grafico-container {
  margin: 2rem 0;
  overflow-x: auto;
}

.grafico-container svg {
  width: 100%;
  height: auto;
  min-height: 300px;
}

.barra-oee {
  cursor: pointer;
  transition: opacity 0.2s;
}

.barra-oee:hover { opacity: 0.8; }

.punto-grafico {
  cursor: pointer;
  transition: r 0.2s;
}

.punto-grafico:hover { r: 8; }

.grafico-leyenda {
  display: flex;
  justify-content: center;
  gap: 2rem;
  margin-top: 1rem;
}

.leyenda-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: #6b7280;
}

.leyenda-linea {
  width: 32px;
  height: 3px;
  display: block;
  border-radius: 2px;
}

.tabla-container {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow-x: auto;
}

.tabla-datos {
  width: 100%;
  border-collapse: collapse;
}

.tabla-datos thead {
  background: #001f54;
  color: #fff;
}

.tabla-datos th {
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.875rem;
}

.tabla-datos td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e5e7eb;
  color: #374151;
  font-size: 0.875rem;
}

.tabla-datos tbody tr:hover { background: #f9fafb; }

.badge-oee, .badge-estado {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.8125rem;
  font-weight: 600;
}

.badge-oee.excelente, .badge-estado.excelente { background: #dcfce7; color: #166534; }
.badge-oee.aceptable, .badge-estado.aceptable { background: #fef3c7; color: #92400e; }
.badge-oee.critico, .badge-estado.critico { background: #fee2e2; color: #991b1b; }

.tooltip-oee {
  position: fixed;
  background: rgba(0, 15, 40, 0.95);
  color: #fff;
  padding: 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  pointer-events: none;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  max-width: 280px;
}

.tooltip-titulo {
  font-weight: 700;
  margin-bottom: 0.5rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  padding-bottom: 0.5rem;
}

.tooltip-linea { margin: 0.25rem 0; }

.paginacion {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}

.pag-btn {
  background: #001f54;
  color: #fff;
  border: none;
  border-radius: 4px;
  padding: 0.4rem 0.75rem;
  font-size: 1rem;
  cursor: pointer;
  transition: background 0.2s;
}

.pag-btn:hover:not(:disabled) { background: #003080; }

.pag-btn:disabled {
  background: #c8d0dd;
  cursor: default;
}

.pag-info {
  font-size: 0.875rem;
  color: #4b5563;
  padding: 0 0.5rem;
}

.pag-total {
  text-align: center;
  font-size: 0.8rem;
  color: #6b7280;
  margin-top: 0.5rem;
}

.vars-placeholder {
  font-size: 0.8rem;
  color: #9ca3af;
  text-align: center;
  padding: 1rem 0.5rem;
  font-style: italic;
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

@media (max-width: 1024px) {
  .formulario-grid { grid-template-columns: 1fr 1fr; }
  .cards-componentes { grid-template-columns: 1fr 1fr; }
}

@media (max-width: 640px) {
  .formulario-grid { grid-template-columns: 1fr; }
  .cards-componentes { grid-template-columns: 1fr; }
  .grid-consulta { grid-template-columns: 1fr; }
}
</style>
