<template>
  <div class="grafica-pareto">
    <!-- Encabezado -->
    <div class="titulo-principal">
      GRÁFICA DE PARETO
    </div>

    <!-- Mensaje de selección -->
    <div class="mensaje-seleccion">
      Seleccione las variables:
    </div>

    <!-- Formulario de Filtros - 3 Columnas -->
    <div class="formulario-grid-nuevo">
      <!-- FECHAS -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">FECHAS</h3>
        <div class="campo-grupo-nuevo">
          <label>Inicio</label>
          <div class="fecha-slider-box">
            <input type="date" v-model="inicioFecha" class="campo-entrada-nuevo campo-fecha" />
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
        <div class="campo-grupo-nuevo">
          <label>Fin</label>
          <div class="fecha-slider-box">
            <input type="date" v-model="finFecha" class="campo-entrada-nuevo campo-fecha" />
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
      </div>

      <!-- DISPOSITIVO -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">DISPOSITIVO</h3>

        <h4 class="subseccion-label">Compañía</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="companiaSeleccionada" class="campo-entrada-nuevo" @change="onEmpresaChange">
            <option :value="null" disabled>Seleccionar</option>
            <option v-for="comp in companias" :key="comp.id" :value="comp.id">{{ comp.nombre }}</option>
          </select>
        </div>

        <h4 class="subseccion-label">Planta</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="plantaSeleccionada" class="campo-entrada-nuevo" @change="onPlantaChange">
            <option :value="null" disabled>Seleccionar</option>
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">{{ planta.nombre }}</option>
          </select>
        </div>

        <h4 class="subseccion-label">Línea</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="lineaSeleccionada" class="campo-entrada-nuevo">
            <option :value="null">Todas</option>
            <option v-for="linea in lineasFiltradas" :key="linea.id" :value="linea.id">{{ linea.nombre }}</option>
          </select>
        </div>
      </div>

      <!-- OPCIONES -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">OPCIONES</h3>

        <h4 class="subseccion-label">Agrupar por</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="agruparPor" class="campo-entrada-nuevo">
            <option value="categoria_nombre">Categoría General</option>
            <option value="subcategoria_nombre">Categoría de Parada</option>
            <option value="stop_type">Tipo de Parada</option>
          </select>
        </div>

        <h4 class="subseccion-label">Métrica</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="metrica" class="campo-entrada-nuevo">
            <option value="duracion">Tiempo de Parada (seg)</option>
            <option value="frecuencia">Frecuencia (# paradas)</option>
          </select>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>
        <div v-if="cargando" class="cargando-msg">Cargando...</div>
      </div>
    </div>

    <!-- Botón Buscar -->
    <div class="seccion-buscar">
      <button @click="buscar" class="btn-buscar-principal">
        BUSCAR
      </button>
    </div>

    <!-- Sin resultados -->
    <div v-if="!cargando && consultaRealizada && paradas.length === 0" class="estado-vacio-pareto">
      No se encontraron paradas para los filtros seleccionados.
    </div>

    <!-- Gráfica de Pareto -->
    <div v-if="datosPareto.length > 0" class="contenedor-pareto">
      <div class="pareto-header">
        <h3>Análisis de Pareto - {{ getTituloAnalisis() }}</h3>
        <button @click="exportarExcel" class="btn-exportar">
          Exportar Excel
        </button>
      </div>

      <!-- Principio 80/20 -->
      <div class="principio-8020">
        <div class="card-principio">
          <div class="principio-titulo">Principio 80/20</div>
          <div class="principio-descripcion">
            El <strong>{{ porcentaje8020 }}%</strong> de las causas representan el <strong>80%</strong> del impacto total
          </div>
          <div class="principio-causas">
            <strong>Causas vitales ({{ causasVitales.length }}):</strong>
            {{ causasVitales.map(c => c.categoria).join(', ') }}
          </div>
        </div>
      </div>

      <!-- Gráfico Pareto -->
      <div class="grafico-container">
        <svg viewBox="0 0 1200 500" preserveAspectRatio="none">
          <!-- Grid -->
          <g v-for="i in 5" :key="'grid-' + i">
            <line
              :x1="100"
              :y1="50 + (i - 1) * 80"
              :x2="1150"
              :y2="50 + (i - 1) * 80"
              stroke="#e5e7eb"
              stroke-width="1"
            />
            <text
              :x="80"
              :y="50 + (i - 1) * 80 + 5"
              text-anchor="end"
              fill="#6b7280"
              font-size="12"
            >
              {{ (100 - (i - 1) * 25) }}
            </text>
          </g>

          <!-- Barras -->
          <g v-for="(dato, index) in datosPareto" :key="'barra-' + index">
            <rect
              :x="100 + (index * (1050 / datosPareto.length)) + 10"
              :y="370 - (dato.valor / valorMaximo * 320)"
              :width="(1050 / datosPareto.length) - 30"
              :height="dato.valor / valorMaximo * 320"
              fill="#3b82f6"
              class="barra-pareto"
              @mouseenter="mostrarTooltip($event, dato)"
              @mouseleave="ocultarTooltip"
            />
            <text
              :x="100 + (index * (1050 / datosPareto.length)) + (1050 / datosPareto.length) / 2"
              y="460"
              text-anchor="end"
              fill="#374151"
              font-size="11"
              transform="rotate(-45 100 460)"
            >
              {{ dato.categoria }}
            </text>
          </g>

          <!-- Línea acumulativa -->
          <path
            :d="generarPathAcumulado()"
            stroke="#ef4444"
            stroke-width="3"
            fill="none"
            stroke-linecap="round"
          />

          <!-- Puntos acumulados -->
          <g v-for="(dato, index) in datosPareto" :key="'punto-' + index">
            <circle
              :cx="100 + (index * (1050 / datosPareto.length)) + (1050 / datosPareto.length) / 2"
              :cy="370 - (dato.porcentajeAcumulado * 3.2)"
              r="5"
              fill="#ef4444"
              class="punto-acumulado"
            />
            <text
              :x="100 + (index * (1050 / datosPareto.length)) + (1050 / datosPareto.length) / 2"
              :y="370 - (dato.porcentajeAcumulado * 3.2) - 10"
              text-anchor="middle"
              fill="#ef4444"
              font-size="11"
              font-weight="bold"
            >
              {{ dato.porcentajeAcumulado.toFixed(0) }}%
            </text>
          </g>

          <!-- Línea 80% -->
          <line x1="100" y1="114" x2="1150" y2="114" stroke="#22c55e" stroke-width="2" stroke-dasharray="5,5"/>
          <text x="1160" y="118" fill="#22c55e" font-size="12" font-weight="bold">80%</text>

          <!-- Eje X -->
          <line x1="100" y1="370" x2="1150" y2="370" stroke="#9ca3af" stroke-width="2"/>

          <!-- Eje Y derecho (porcentaje) -->
          <g v-for="i in 5" :key="'grid-right-' + i">
            <text
              x="1170"
              :y="50 + (i - 1) * 80 + 5"
              text-anchor="start"
              fill="#ef4444"
              font-size="12"
              font-weight="bold"
            >
              {{ (100 - (i - 1) * 25) }}%
            </text>
          </g>
        </svg>

        <!-- Leyenda -->
        <div class="grafico-leyenda">
          <span class="leyenda-item">
            <span class="leyenda-cuadro" style="background: #3b82f6"></span> Frecuencia
          </span>
          <span class="leyenda-item">
            <span class="leyenda-linea" style="background: #ef4444"></span> % Acumulado
          </span>
          <span class="leyenda-item">
            <span class="leyenda-linea-punteada"></span> Regla 80/20
          </span>
        </div>
      </div>

      <!-- Tabla de Datos -->
      <div class="tabla-container">
        <table class="tabla-datos">
          <thead>
            <tr>
              <th>Categoría</th>
              <th>Frecuencia</th>
              <th>Porcentaje</th>
              <th>% Acumulado</th>
              <th>Clasificación</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="dato in datosPareto" :key="dato.id">
              <td>{{ dato.categoria }}</td>
              <td>{{ dato.valor.toLocaleString() }}</td>
              <td>{{ dato.porcentaje.toFixed(1) }}%</td>
              <td>
                <div class="celda-acumulado">
                  <span>{{ dato.porcentajeAcumulado.toFixed(1) }}%</span>
                  <div class="barra-acumulado">
                    <div 
                      class="barra-progreso-acumulado" 
                      :style="{ width: dato.porcentajeAcumulado + '%' }"
                    ></div>
                  </div>
                </div>
              </td>
              <td>
                <span :class="['badge-clasificacion', dato.clasificacion.toLowerCase()]">
                  {{ dato.clasificacion }}
                </span>
              </td>
            </tr>
          </tbody>
          <tfoot>
            <tr>
              <td><strong>TOTAL</strong></td>
              <td><strong>{{ totalFrecuencia.toLocaleString() }}</strong></td>
              <td><strong>100%</strong></td>
              <td><strong>-</strong></td>
              <td><strong>-</strong></td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>

    <!-- Tooltip -->
    <Teleport to="body">
      <div
        v-if="tooltipVisible"
        class="tooltip-pareto"
        :style="{ left: tooltipX + 'px', top: tooltipY + 'px' }"
      >
        <div class="tooltip-titulo">{{ tooltipData.categoria }}</div>
        <div class="tooltip-linea">
          <strong>Frecuencia:</strong> {{ tooltipData.valor?.toLocaleString() }}
        </div>
        <div class="tooltip-linea">
          <strong>Porcentaje:</strong> {{ tooltipData.porcentaje?.toFixed(1) }}%
        </div>
        <div class="tooltip-linea">
          <strong>% Acumulado:</strong> {{ tooltipData.porcentajeAcumulado?.toFixed(1) }}%
        </div>
        <div class="tooltip-linea">
          <strong>Clasificación:</strong> {{ tooltipData.clasificacion }}
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { turnoService } from '@/api/services/turno.service'
import { stopsService } from '@/api/services/stops.service'

// ── Maestros ────────────────────────────────────────────────────────────────
const companias = ref([])
const plantas   = ref([])
const lineas    = ref([])
const turnos    = ref([])   // { id, nombre, hora_inicio, hora_fin, planta_id }

// ── Selecciones de filtro ───────────────────────────────────────────────────
const companiaSeleccionada = ref(null)
const plantaSeleccionada   = ref(null)
const lineaSeleccionada    = ref(null)
const agruparPor           = ref('categoria_nombre')
const metrica              = ref('duracion')

// ── Estado ──────────────────────────────────────────────────────────────────
const cargando          = ref(false)
const error             = ref(null)
const consultaRealizada = ref(false)
const paradas           = ref([])   // raw stops from API
const datosPareto       = ref([])

const lineaMode = computed(() => lineas.value.find(l => l.id === lineaSeleccionada.value)?.mode ?? 'botellas')
const intervalConfig = computed(() => lineaMode.value === 'textil'
  ? { msStep: 1800000, sliderMax: 1, minuteMultiplier: 30, divisor: 30 }
  : { msStep: 300000,  sliderMax: 11, minuteMultiplier: 5,  divisor: 5  })

// ── Fechas ──────────────────────────────────────────────────────────────────
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

const now    = new Date()
const hace7d = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)

const iniParts = splitDate(hace7d)
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

// ── Computed filtrados ──────────────────────────────────────────────────────
const plantasFiltradas = computed(() =>
  plantas.value.filter(p => p.empresa_id === companiaSeleccionada.value)
)
const lineasFiltradas = computed(() =>
  lineas.value.filter(l => l.planta_id === plantaSeleccionada.value)
)

// ── Helpers ─────────────────────────────────────────────────────────────────
function resolveLinea(linea_id) {
  return lineas.value.find(l => l.id === linea_id)?.nombre || '-'
}

function resolveTurno(isoDate) {
  if (!isoDate) return '-'
  const d = new Date(isoDate)
  const hh = d.getHours() * 3600 + d.getMinutes() * 60 + d.getSeconds()
  // Usar todos los turnos activos, sin filtrar por planta
  const lista = turnos.value.filter(t => t.activo !== false)
  for (const t of lista) {
    const [sh, sm] = t.hora_inicio.split(':').map(Number)
    const [eh, em] = t.hora_fin.split(':').map(Number)
    const start = sh * 3600 + sm * 60
    let end   = eh * 3600 + em * 60
    if (end <= start) end += 86400   // overnight
    const hAdj = end <= start && hh < start ? hh + 86400 : hh
    if (hAdj >= start && hAdj < end) return t.nombre
  }
  return 'Sin Turno'
}

function getDuracion(p) {
  // 1) duration_min del API → convertir a segundos
  const v = p.duration_min ?? p.duracion_min
  if (v != null && Number(v) > 0) return Math.round(Number(v) * 60 * 100) / 100
  // 2) calcular de started_at + ended_at → segundos
  if (p.started_at && (p.ended_at || p.fin)) {
    const ini = new Date(p.started_at || p.inicio)
    const fin = new Date(p.ended_at || p.fin)
    const segs = (fin - ini) / 1000
    if (segs > 0) return Math.round(segs * 100) / 100
  }
  return 0
}

function getCategoria(p) {
  return p.subcategoria_nombre || p.categoria_nombre || '-'
}

function getDescripcion(p) {
  return p.subcategoria_2_nombre || p.descripcion || p.reason || p.categoria_nombre || '-'
}

function formatFecha(iso) {
  if (!iso) return '-'
  const d = new Date(iso)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// ── Onchange handlers ───────────────────────────────────────────────────────
function onEmpresaChange() {
  plantaSeleccionada.value = null
  lineaSeleccionada.value  = null
}
function onPlantaChange() {
  lineaSeleccionada.value = null
}

// ── Cargar maestros ─────────────────────────────────────────────────────────
async function cargarMaestros() {
  try {
    const [empR, plnR, linR, turR] = await Promise.all([
      companyService.getAll(),
      plantService.getAll(),
      lineService.getAll(),
      turnoService.getAll()
    ])
    companias.value = empR.data?.data || empR.data || []
    plantas.value   = plnR.data?.data || plnR.data || []
    lineas.value    = linR.data?.data || linR.data || []
    turnos.value    = turR.data?.data || turR.data || []

    if (companias.value.length > 0 && !companiaSeleccionada.value) {
      companiaSeleccionada.value = companias.value[0].id
    }
  } catch (e) {
    error.value = 'Error cargando maestros: ' + (e.message || e)
  }
}

// ── Buscar ──────────────────────────────────────────────────────────────────
async function buscar() {
  if (!companiaSeleccionada.value) { error.value = 'Seleccione una compañía'; return }
  error.value         = null
  cargando.value      = true
  consultaRealizada.value = true
  datosPareto.value   = []
  paradas.value       = []

  try {
    const params = {
      empresa_id: companiaSeleccionada.value,
      limit: 1000
    }
    if (plantaSeleccionada.value)  params.planta_id = plantaSeleccionada.value
    if (lineaSeleccionada.value)   params.linea_id  = lineaSeleccionada.value
    if (fechaInicio.value)    params.since     = new Date(fechaInicio.value).toISOString()
    if (fechaFin.value)       params.until     = new Date(fechaFin.value).toISOString()

    const res = await stopsService.list(params)
    paradas.value = res.data?.data || res.data || res || []
    calcularPareto()
  } catch (e) {
    error.value = 'Error al obtener paradas: ' + (e.message || e)
  } finally {
    cargando.value = false
  }
}

// ── Calcular Pareto ─────────────────────────────────────────────────────────
function calcularPareto() {
  const mapa = new Map()
  for (const p of paradas.value) {
    let clave
    if (agruparPor.value === 'subcategoria_nombre') {
      clave = getCategoria(p)
    } else {
      clave = p[agruparPor.value] || p.categoria_nombre || 'Sin categoría'
    }
    const valor  = metrica.value === 'duracion'
      ? getDuracion(p)
      : 1
    mapa.set(clave, (mapa.get(clave) || 0) + valor)
  }

  const ordenado = [...mapa.entries()]
    .map(([cat, val]) => ({ categoria: cat, valor: val }))
    .sort((a, b) => b.valor - a.valor)

  const total = ordenado.reduce((s, d) => s + d.valor, 0)
  let acum = 0
  datosPareto.value = ordenado.map((d, i) => {
    const pct = total > 0 ? (d.valor / total) * 100 : 0
    acum += pct
    return {
      id: i + 1,
      categoria: d.categoria,
      valor: Math.round(d.valor * 100) / 100,
      porcentaje: pct,
      porcentajeAcumulado: acum,
      clasificacion: acum <= 80 ? 'A' : acum <= 95 ? 'B' : 'C'
    }
  })
}

// ── Computed para la gráfica ───────────────────────────────────────────────────────────
const valorMaximo = computed(() =>
  datosPareto.value.length ? Math.max(...datosPareto.value.map(d => d.valor)) : 1
)
const totalFrecuencia = computed(() =>
  datosPareto.value.reduce((s, d) => s + d.valor, 0)
)
const causasVitales = computed(() =>
  datosPareto.value.filter(d => d.porcentajeAcumulado <= 80)
)
const porcentaje8020 = computed(() =>
  datosPareto.value.length
    ? ((causasVitales.value.length / datosPareto.value.length) * 100).toFixed(0)
    : 0
)

// Tooltip
const tooltipVisible = ref(false)
const tooltipX       = ref(0)
const tooltipY       = ref(0)
const tooltipData    = ref({})

function generarPathAcumulado() {
  if (datosPareto.value.length === 0) return ''
  const n = datosPareto.value.length
  const puntos = datosPareto.value.map((dato, index) => {
    const x = 100 + (index * (1050 / n)) + (1050 / n) / 2
    const y = 370 - (dato.porcentajeAcumulado * 3.2)
    return `${x},${y}`
  })
  return `M ${puntos.join(' L ')}`
}

function getTituloAnalisis() {
  const map = {
    'categoria_nombre':    'Categoría General',
    'subcategoria_nombre': 'Categoría de Parada',
    'stop_type':           'Tipo de Parada'
  }
  return map[agruparPor.value] || agruparPor.value
}

function mostrarTooltip(event, dato) {
  tooltipData.value = dato
  tooltipX.value = event.clientX + 10
  tooltipY.value = event.clientY + 10
  tooltipVisible.value = true
}
function ocultarTooltip() { tooltipVisible.value = false }

// ── Exportar Excel (7 columnas — detalle de paradas) ────────────────────────
function exportarExcel() {
  if (paradas.value.length === 0) { alert('No hay datos para exportar'); return }

  const fechaRep = new Date().toISOString().split('T')[0]
  const esc = s => String(s ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')

  let xml = `<?xml version="1.0"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet">
 <Styles>
  <Style ss:ID="h">
   <Font ss:Bold="1" ss:Color="#FFFFFF"/>
   <Interior ss:Color="#001f54" ss:Pattern="Solid"/>
   <Alignment ss:Horizontal="Center" ss:Vertical="Center"/>
  </Style>
 </Styles>
 <Worksheet ss:Name="Paradas">
  <Table>
   <Row>
    <Cell ss:StyleID="h"><Data ss:Type="String">Fecha</Data></Cell>
    <Cell ss:StyleID="h"><Data ss:Type="String">Línea</Data></Cell>
    <Cell ss:StyleID="h"><Data ss:Type="String">Turno</Data></Cell>
    <Cell ss:StyleID="h"><Data ss:Type="String">Categoría General</Data></Cell>
    <Cell ss:StyleID="h"><Data ss:Type="String">Categoría de Paradas</Data></Cell>
    <Cell ss:StyleID="h"><Data ss:Type="String">Descripción de Parada</Data></Cell>
    <Cell ss:StyleID="h"><Data ss:Type="String">Tiempo de Parada (seg)</Data></Cell>
   </Row>`

  for (const p of paradas.value) {
    const fecha  = formatFecha(p.started_at || p.inicio)
    const linea  = esc(resolveLinea(p.linea_id))
    const turno  = esc(resolveTurno(p.started_at || p.inicio))
    const catG   = esc(p.categoria_nombre || '-')
    const catP   = esc(getCategoria(p))
    const desc   = esc(getDescripcion(p))
    const tiempo = getDuracion(p)
    xml += `
   <Row>
    <Cell><Data ss:Type="String">${fecha}</Data></Cell>
    <Cell><Data ss:Type="String">${linea}</Data></Cell>
    <Cell><Data ss:Type="String">${turno}</Data></Cell>
    <Cell><Data ss:Type="String">${catG}</Data></Cell>
    <Cell><Data ss:Type="String">${catP}</Data></Cell>
    <Cell><Data ss:Type="String">${desc}</Data></Cell>
    <Cell><Data ss:Type="Number">${tiempo}</Data></Cell>
   </Row>`
  }

  xml += `
  </Table>
 </Worksheet>
</Workbook>`

  const blob = new Blob([xml], { type: 'application/vnd.ms-excel' })
  const url  = URL.createObjectURL(blob)
  const a    = document.createElement('a')
  a.href = url
  a.download = `Pareto_Paradas_${fechaRep}.xls`
  document.body.appendChild(a); a.click()
  document.body.removeChild(a); URL.revokeObjectURL(url)
}

onMounted(cargarMaestros)
</script>

<style scoped>
.grafica-pareto {
  padding: clamp(1rem, 2vw, 2rem);
  max-width: 1400px;
  margin: 0 auto;
}

.titulo-principal {
  background: #001f54;
  color: white;
  padding: 1.5rem;
  text-align: center;
  font-size: 1.5rem;
  font-weight: bold;
  margin-bottom: 2rem;
  border-radius: 8px;
  letter-spacing: 1px;
}

/* Nuevos estilos */
.mensaje-seleccion {
  background: white;
  border-left: 4px solid #001f54;
  padding: 1rem 1.5rem;
  margin-bottom: 1.5rem;
  font-weight: 600;
  color: #001f54;
  border-radius: 4px;
}

.formulario-grid-nuevo {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.25rem;
  margin-bottom: 2rem;
}

.seccion-box-nuevo {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
}

.subtitulo-seccion-nuevo {
  background: #001f54;
  color: white;
  padding: 0.625rem;
  text-align: center;
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0 0 1rem 0;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.subseccion-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #475569;
  margin: 0.75rem 0 0.375rem 0;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.campo-grupo-nuevo {
  margin-bottom: 1rem;
}

.campo-grupo-nuevo label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #374151;
  font-size: 0.875rem;
}

.campo-entrada-nuevo {
  width: 100%;
  padding: 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.875rem;
  transition: border-color 0.2s;
}

.campo-entrada-nuevo:focus {
  outline: none;
  border-color: #001f54;
  box-shadow: 0 0 0 3px rgba(0, 31, 84, 0.1);
}

.error-msg {
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #fca5a5;
  border-radius: 6px;
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  margin-top: 0.5rem;
}
.cargando-msg {
  color: #475569;
  font-size: 0.85rem;
  padding: 0.5rem;
  text-align: center;
  font-style: italic;
}
.estado-vacio-pareto {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  padding: 2rem;
  text-align: center;
  color: #64748b;
  margin-bottom: 1.5rem;
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

.seccion-buscar {
  display: flex;
  justify-content: center;
  margin-bottom: 2rem;
}

.btn-buscar-principal {
  background: #001f54;
  color: white;
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

.btn-buscar-principal:hover {
  background: #001238;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 31, 84, 0.3);
}

/* Estilos antiguos */
.subtitulo-seccion {
  background: #001f54;
  color: white;
  padding: 0.75rem;
  text-align: center;
  font-size: 0.9rem;
  font-weight: bold;
  margin: 0 0 1rem 0;
  border-radius: 4px;
}

.formulario-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.seccion-box {
  background: #f5f5f5;
  padding: 1.5rem;
  border-radius: 8px;
}

.campo-grupo {
  margin-bottom: 1rem;
}

.campo-grupo label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #374151;
}

.campo-entrada {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 1rem;
}

.btn-buscar {
  width: 100%;
  padding: 0.75rem;
  background: #4a4a4a;
  color: white;
  border: none;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 1rem;
}

.btn-buscar:hover {
  background: #333;
}

.contenedor-pareto {
  margin-top: 2rem;
}

.pareto-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding: 1rem;
  background: #001f54;
  color: white;
  border-radius: 8px;
}

.pareto-header h3 {
  font-size: 1.25rem;
  margin: 0;
}

.btn-exportar {
  padding: 0.5rem 1.5rem;
  background: white;
  color: #001f54;
  border: none;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
}

.btn-exportar:hover {
  background: #f3f4f6;
}

.principio-8020 {
  margin-bottom: 2rem;
}

.card-principio {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.principio-titulo {
  font-size: 1.5rem;
  font-weight: bold;
  margin-bottom: 0.75rem;
}

.principio-descripcion {
  font-size: 1.1rem;
  margin-bottom: 1rem;
}

.principio-causas {
  font-size: 0.95rem;
  opacity: 0.9;
}

.grafico-container {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
}

.grafico-container svg {
  width: 100%;
  height: auto;
}

.barra-pareto {
  cursor: pointer;
  transition: opacity 0.2s;
}

.barra-pareto:hover {
  opacity: 0.8;
}

.punto-acumulado {
  cursor: pointer;
}

.grafico-leyenda {
  display: flex;
  justify-content: center;
  gap: 2rem;
  margin-top: 1.5rem;
}

.leyenda-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9rem;
  color: #6b7280;
}

.leyenda-cuadro {
  width: 20px;
  height: 20px;
  border-radius: 4px;
}

.leyenda-linea {
  width: 40px;
  height: 3px;
  display: block;
}

.leyenda-linea-punteada {
  width: 40px;
  height: 2px;
  background: repeating-linear-gradient(
    to right,
    #22c55e 0,
    #22c55e 5px,
    transparent 5px,
    transparent 10px
  );
}

.tabla-container {
  background: white;
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
  color: white;
}

.tabla-datos th {
  padding: 1rem;
  text-align: left;
  font-weight: 600;
}

.tabla-datos td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e5e7eb;
  color: #374151;
}

.tabla-datos tfoot {
  background: #f9fafb;
  font-weight: bold;
}

.tabla-datos tbody tr:hover {
  background: #f9fafb;
}

.celda-acumulado {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.barra-acumulado {
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.barra-progreso-acumulado {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #ef4444);
  transition: width 0.3s ease;
}

.badge-clasificacion {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 600;
}

.badge-clasificacion.a {
  background: #fee2e2;
  color: #991b1b;
}

.badge-clasificacion.b {
  background: #fef3c7;
  color: #92400e;
}

.badge-clasificacion.c {
  background: #dcfce7;
  color: #166534;
}

.tooltip-pareto {
  position: fixed;
  background: rgba(0, 0, 0, 0.9);
  color: white;
  padding: 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  pointer-events: none;
  z-index: 1000;
  backdrop-filter: blur(10px);
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  max-width: 300px;
}

.tooltip-titulo {
  font-weight: bold;
  margin-bottom: 0.5rem;
  font-size: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  padding-bottom: 0.5rem;
}

.tooltip-linea {
  margin: 0.25rem 0;
}

@media (max-width: 1024px) {
  .formulario-grid,
  .formulario-grid-nuevo {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 640px) {
  .formulario-grid,
  .formulario-grid-nuevo {
    grid-template-columns: 1fr;
  }
}
</style>
