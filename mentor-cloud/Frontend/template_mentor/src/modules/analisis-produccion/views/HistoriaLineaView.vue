<template>
  <div class="historia-linea">
    <!-- Encabezado -->
    <div class="titulo-principal">
      HISTORIA DE LÍNEA
    </div>

    <!-- Mensaje de selección -->
    <div class="mensaje-seleccion">
      <h3>Seleccione las variables:</h3>
    </div>

    <!-- Formulario de Filtros - 3 Columnas -->
    <div class="formulario-grid-nuevo">
      <!-- FECHAS -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">FECHAS</h3>
        <div class="campo-grupo-nuevo">
          <label>Inicio</label>
          <input 
            type="datetime-local" 
            v-model="fechaInicioInput"
            class="campo-entrada-nuevo"
          />
        </div>
        <div class="campo-grupo-nuevo">
          <label>Fin</label>
          <input 
            type="datetime-local" 
            v-model="fechaFinInput"
            class="campo-entrada-nuevo"
          />
        </div>
        <h3 class="subtitulo-seccion-nuevo">Compañía</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="companiaSeleccionada" class="campo-entrada-nuevo">
            <option v-for="comp in companias" :key="comp.id" :value="comp.id">
              {{ comp.nombre }}
            </option>
          </select>
        </div>
        <h3 class="subtitulo-seccion-nuevo">Planta</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="plantaSeleccionada" class="campo-entrada-nuevo">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
        <h3 class="subtitulo-seccion-nuevo">Línea</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="lineaSeleccionada" class="campo-entrada-nuevo">
            <option v-for="linea in lineasFiltradas" :key="linea.id" :value="linea.id">
              {{ linea.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- VARIABLES (3 categorías como en la imagen) -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">VARIABLES</h3>
        
        <h4 class="subseccion-label">Tiempo Disponible</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableTiempoDisponible" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesTiempoDisponible" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <h4 class="subseccion-label">Tiempo Parada No Asignada</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableParadaNoAsignada" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesParadaNoAsignada" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <h4 class="subseccion-label">Tiempo de Microparadas</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableMicroparadas" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesMicroparadas" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <h4 class="subseccion-label">Tiempos de Paradas Obligatorios</h4>
        <div class="variables-tags-container">
          <div v-for="(tag, index) in paradasObligatorias" :key="index" class="variable-tag-small">
            {{ tag.nombre }}
            <button @click="eliminarParadaObligatoria(index)" class="btn-tag-remove">×</button>
          </div>
        </div>
        <div class="campo-grupo-nuevo">
          <select v-model="variableParadaObligatoria" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesParadasObligatorias" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
          <button @click="agregarParadaObligatoria" class="btn-add-small">Agregar</button>
        </div>
      </div>

      <!-- VARIABLES (Columna 3) -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">VARIABLES</h3>
        
        <h4 class="subseccion-label">Tiempo Parada Programada</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableParadaProgramada" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesParadaProgramada" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <h4 class="subseccion-label">Tipo parada programada</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableTipoParadaProgramada" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesTipoParadaProgramada" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <h4 class="subseccion-label">Tiempo Parada No Programada</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableParadaNoProgramada" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesParadaNoProgramada" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <h4 class="subseccion-label">Tipo parada no programada</h4>
        <div class="campo-grupo-nuevo">
          <select v-model="variableTipoParadaNoProgramada" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variablesTipoParadaNoProgramada" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>
      </div>
    </div>

    <!-- Botón Buscar -->
    <div class="seccion-buscar">
      <button @click="buscar" class="btn-buscar-principal">
        BUSCAR
      </button>
    </div>

    <!-- Gráfico de Historia -->
    <div v-if="datosHistoria.length > 0" class="contenedor-grafico">
      <div class="grafico-header">
        <h3>Evolución Histórica de la Línea</h3>
        <button @click="exportarExcel" class="btn-exportar">
          Exportar Excel
        </button>
      </div>

      <!-- Métricas OEE -->
      <div class="cards-oee">
        <div class="card-oee oee">
          <div class="card-oee-label">OEE</div>
          <div class="card-oee-valor">{{ promedioOEE }}%</div>
          <div class="card-oee-barra">
            <div class="barra-progreso" :style="{ width: promedioOEE + '%', background: getColorOEE(promedioOEE) }"></div>
          </div>
        </div>

        <div class="card-oee disponibilidad">
          <div class="card-oee-label">Disponibilidad</div>
          <div class="card-oee-valor">{{ promedioDisponibilidad }}%</div>
          <div class="card-oee-barra">
            <div class="barra-progreso" :style="{ width: promedioDisponibilidad + '%', background: '#3b82f6' }"></div>
          </div>
        </div>

        <div class="card-oee rendimiento">
          <div class="card-oee-label">Rendimiento</div>
          <div class="card-oee-valor">{{ promedioRendimiento }}%</div>
          <div class="card-oee-barra">
            <div class="barra-progreso" :style="{ width: promedioRendimiento + '%', background: '#8b5cf6' }"></div>
          </div>
        </div>

        <div class="card-oee calidad">
          <div class="card-oee-label">Calidad</div>
          <div class="card-oee-valor">{{ promedioCalidad }}%</div>
          <div class="card-oee-barra">
            <div class="barra-progreso" :style="{ width: promedioCalidad + '%', background: '#10b981' }"></div>
          </div>
        </div>
      </div>

      <!-- Gráfico -->
      <div class="grafico-container">
        <svg viewBox="0 0 1200 400" preserveAspectRatio="none">
          <defs>
            <linearGradient id="gradienteOEE" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stop-color="#3b82f6" stop-opacity="0.3"/>
              <stop offset="100%" stop-color="#3b82f6" stop-opacity="0.05"/>
            </linearGradient>
          </defs>

          <!-- Grid -->
          <g v-for="i in 5" :key="'grid-' + i">
            <line
              :x1="50"
              :y1="50 + (i - 1) * 70"
              :x2="1150"
              :y2="50 + (i - 1) * 70"
              stroke="#e5e7eb"
              stroke-width="1"
            />
            <text
              :x="30"
              :y="50 + (i - 1) * 70 + 5"
              text-anchor="end"
              fill="#6b7280"
              font-size="12"
            >
              {{ 100 - (i - 1) * 25 }}%
            </text>
          </g>

          <!-- Área bajo la curva -->
          <path
            :d="generarAreaPath()"
            fill="url(#gradienteOEE)"
          />

          <!-- Línea OEE -->
          <path
            :d="generarPath('oee')"
            stroke="#3b82f6"
            stroke-width="3"
            fill="none"
            stroke-linecap="round"
          />

          <!-- Puntos -->
          <g v-for="(punto, index) in datosHistoria" :key="'punto-' + index">
            <circle
              :cx="50 + (index * (1100 / (datosHistoria.length - 1)))"
              :cy="330 - (punto.oee * 2.8)"
              r="6"
              fill="#3b82f6"
              class="punto-grafico"
              @mouseenter="mostrarTooltip($event, punto)"
              @mouseleave="ocultarTooltip"
            />
          </g>

          <!-- Eje X -->
          <line x1="50" y1="330" x2="1150" y2="330" stroke="#9ca3af" stroke-width="2"/>
          
          <!-- Etiquetas X -->
          <g v-for="(punto, index) in datosHistoria" :key="'label-' + index">
            <text
              v-if="index % Math.ceil(datosHistoria.length / 10) === 0"
              :x="50 + (index * (1100 / (datosHistoria.length - 1)))"
              y="350"
              text-anchor="middle"
              fill="#6b7280"
              font-size="11"
            >
              {{ punto.label }}
            </text>
          </g>
        </svg>
      </div>

      <!-- Tabla de Datos -->
      <div class="tabla-container">
        <table class="tabla-datos">
          <thead>
            <tr>
              <th>{{ getTituloAgrupacion() }}</th>
              <th>OEE (%)</th>
              <th>Disponibilidad (%)</th>
              <th>Rendimiento (%)</th>
              <th>Calidad (%)</th>
              <th>Producción Real</th>
              <th>Producción Objetivo</th>
              <th>Tiempo Parada (min)</th>
              <th>Defectos</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="dato in datosHistoria" :key="dato.id">
              <td>{{ dato.label }}</td>
              <td>
                <span :class="['badge-oee', getClaseOEE(dato.oee)]">
                  {{ dato.oee.toFixed(1) }}%
                </span>
              </td>
              <td>{{ dato.disponibilidad.toFixed(1) }}%</td>
              <td>{{ dato.rendimiento.toFixed(1) }}%</td>
              <td>{{ dato.calidad.toFixed(1) }}%</td>
              <td>{{ dato.produccionReal.toLocaleString() }}</td>
              <td>{{ dato.produccionObjetivo.toLocaleString() }}</td>
              <td>{{ dato.tiempoParada }}</td>
              <td>{{ dato.defectos }}</td>
            </tr>
          </tbody>
          <tfoot>
            <tr>
              <td><strong>PROMEDIO</strong></td>
              <td><strong>{{ promedioOEE }}%</strong></td>
              <td><strong>{{ promedioDisponibilidad }}%</strong></td>
              <td><strong>{{ promedioRendimiento }}%</strong></td>
              <td><strong>{{ promedioCalidad }}%</strong></td>
              <td><strong>{{ totalProduccionReal.toLocaleString() }}</strong></td>
              <td><strong>{{ totalProduccionObjetivo.toLocaleString() }}</strong></td>
              <td><strong>{{ totalTiempoParada }}</strong></td>
              <td><strong>{{ totalDefectos }}</strong></td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>

    <!-- Tooltip -->
    <Teleport to="body">
      <div
        v-if="tooltipVisible"
        class="tooltip-grafico"
        :style="{ left: tooltipX + 'px', top: tooltipY + 'px' }"
      >
        <div class="tooltip-titulo">{{ tooltipData.label }}</div>
        <div class="tooltip-linea">
          <strong>OEE:</strong> {{ tooltipData.oee?.toFixed(1) }}%
        </div>
        <div class="tooltip-linea">
          <strong>Disponibilidad:</strong> {{ tooltipData.disponibilidad?.toFixed(1) }}%
        </div>
        <div class="tooltip-linea">
          <strong>Rendimiento:</strong> {{ tooltipData.rendimiento?.toFixed(1) }}%
        </div>
        <div class="tooltip-linea">
          <strong>Calidad:</strong> {{ tooltipData.calidad?.toFixed(1) }}%
        </div>
        <div class="tooltip-linea">
          <strong>Producción:</strong> {{ tooltipData.produccionReal?.toLocaleString() }} / {{ tooltipData.produccionObjetivo?.toLocaleString() }}
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

// Datos maestros
const companias = ref([
  { id: 1, nombre: 'Art Atlas' },
  { id: 2, nombre: 'Industrial Tech' }
])

const plantas = ref([
  { id: 1, nombre: 'Planta Textil Lima', companiaId: 1 },
  { id: 2, nombre: 'Planta Sur', companiaId: 1 },
  { id: 3, nombre: 'Planta Central', companiaId: 2 }
])

const lineas = ref([
  { id: 1, nombre: 'LINEA 05', plantaId: 1 },
  { id: 2, nombre: 'Línea 2', plantaId: 1 },
  { id: 3, nombre: 'Línea 3', plantaId: 2 }
])

// Variables por categoría
const variablesTiempoDisponible = ref([
  { id: 1, nombre: 'T_Disponible (PLC-L5-Tiempo Disponible Mentor)' }
])

const variablesParadaNoAsignada = ref([
  { id: 1, nombre: 'LS-T-No Asignada (L5-Tiempo-Parada No Asignada)' }
])

const variablesMicroparadas = ref([
  { id: 1, nombre: 'LS-T-Microparadas (L5-Tiempo-Microparadas)' }
])

const variablesParadasObligatorias = ref([
  { id: 1, nombre: 'PLC-L5-Refrigio' },
  { id: 2, nombre: 'PLC-L5-Capacitación Obligatoria' },
  { id: 3, nombre: 'PLC-L5-Mantenimiento Planificado' }
])

const variablesParadaProgramada = ref([
  { id: 1, nombre: 'T_PP (PLC-L5-Tiempo de Parada Programada)' }
])

const variablesTipoParadaProgramada = ref([
  { id: 1, nombre: 'Tipo PP (PLC-L5-Tipo de Parada Programada)' }
])

const variablesParadaNoProgramada = ref([
  { id: 1, nombre: 'T_PNP (PLC-L5-Tiempo de Parada No Programada)' }
])

const variablesTipoParadaNoProgramada = ref([
  { id: 1, nombre: 'Tipo PNP (PLC-L5-Tipo de Parada No Programada)' }
])

// Selecciones
const companiaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const lineaSeleccionada = ref(1)

// Variables seleccionadas
const variableTiempoDisponible = ref(1)
const variableParadaNoAsignada = ref(1)
const variableMicroparadas = ref(1)
const variableParadaObligatoria = ref('')
const paradasObligatorias = ref([])
const variableParadaProgramada = ref(1)
const variableTipoParadaProgramada = ref(1)
const variableParadaNoProgramada = ref(1)
const variableTipoParadaNoProgramada = ref(1)

// Fechas
const fechaInicio = ref('2025/11/14 00:00')
const fechaFin = ref('2025/11/19 08:15')

const fechaInicioInput = computed({
  get: () => fechaInicio.value.replace(' ', 'T').replace(/\//g, '-'),
  set: (val) => fechaInicio.value = val.replace('T', ' ').replace(/-/g, '/')
})

const fechaFinInput = computed({
  get: () => fechaFin.value.replace(' ', 'T').replace(/\//g, '-'),
  set: (val) => fechaFin.value = val.replace('T', ' ').replace(/-/g, '/')
})

// Filtros
const plantasFiltradas = computed(() => {
  return plantas.value.filter(p => p.companiaId === companiaSeleccionada.value)
})

const lineasFiltradas = computed(() => {
  return lineas.value.filter(l => l.plantaId === plantaSeleccionada.value)
})

// Funciones para paradas obligatorias
function agregarParadaObligatoria() {
  if (!variableParadaObligatoria.value) {
    return
  }
  
  const variable = variablesParadasObligatorias.value.find(v => v.id === variableParadaObligatoria.value)
  if (variable && !paradasObligatorias.value.find(p => p.id === variable.id)) {
    paradasObligatorias.value.push(variable)
  }
  
  variableParadaObligatoria.value = ''
}

function eliminarParadaObligatoria(index) {
  paradasObligatorias.value.splice(index, 1)
}

// Datos
const datosHistoria = ref([])

// Tooltip
const tooltipVisible = ref(false)
const tooltipX = ref(0)
const tooltipY = ref(0)
const tooltipData = ref({})

// Promedios
const promedioOEE = computed(() => {
  const suma = datosHistoria.value.reduce((acc, d) => acc + d.oee, 0)
  return (suma / datosHistoria.value.length || 0).toFixed(1)
})

const promedioDisponibilidad = computed(() => {
  const suma = datosHistoria.value.reduce((acc, d) => acc + d.disponibilidad, 0)
  return (suma / datosHistoria.value.length || 0).toFixed(1)
})

const promedioRendimiento = computed(() => {
  const suma = datosHistoria.value.reduce((acc, d) => acc + d.rendimiento, 0)
  return (suma / datosHistoria.value.length || 0).toFixed(1)
})

const promedioCalidad = computed(() => {
  const suma = datosHistoria.value.reduce((acc, d) => acc + d.calidad, 0)
  return (suma / datosHistoria.value.length || 0).toFixed(1)
})

const totalProduccionReal = computed(() => {
  return datosHistoria.value.reduce((acc, d) => acc + d.produccionReal, 0)
})

const totalProduccionObjetivo = computed(() => {
  return datosHistoria.value.reduce((acc, d) => acc + d.produccionObjetivo, 0)
})

const totalTiempoParada = computed(() => {
  return datosHistoria.value.reduce((acc, d) => acc + d.tiempoParada, 0)
})

const totalDefectos = computed(() => {
  return datosHistoria.value.reduce((acc, d) => acc + d.defectos, 0)
})

function buscar() {
  generarDatosHistoria()
}

function generarDatosHistoria() {
  const datos = []
  const inicio = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T'))
  const fin = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T'))
  const diffDays = Math.ceil((fin - inicio) / (1000 * 60 * 60 * 24))
  const numDias = Math.min(diffDays, 30) // Máximo 30 días

  for (let i = 0; i < numDias; i++) {
    const fecha = new Date(inicio)
    fecha.setDate(fecha.getDate() + i)
    
    const disponibilidad = 75 + Math.random() * 20
    const rendimiento = 80 + Math.random() * 15
    const calidad = 92 + Math.random() * 7
    const oee = (disponibilidad * rendimiento * calidad) / 10000

    datos.push({
      id: i + 1,
      label: `${fecha.getDate()}/${fecha.getMonth() + 1}`,
      oee: oee,
      disponibilidad: disponibilidad,
      rendimiento: rendimiento,
      calidad: calidad,
      produccionReal: Math.floor(8000 + Math.random() * 4000),
      produccionObjetivo: 12000,
      tiempoParada: Math.floor(30 + Math.random() * 90),
      defectos: Math.floor(10 + Math.random() * 50)
    })
  }

  datosHistoria.value = datos
}

function generarPath(metrica) {
  if (datosHistoria.value.length === 0) return ''
  
  const puntos = datosHistoria.value.map((punto, index) => {
    const x = 50 + (index * (1100 / (datosHistoria.value.length - 1)))
    const y = 330 - (punto[metrica] * 2.8)
    return `${x},${y}`
  })
  
  return `M ${puntos.join(' L ')}`
}

function generarAreaPath() {
  if (datosHistoria.value.length === 0) return ''
  
  const puntos = datosHistoria.value.map((punto, index) => {
    const x = 50 + (index * (1100 / (datosHistoria.value.length - 1)))
    const y = 330 - (punto.oee * 2.8)
    return `${x},${y}`
  })
  
  const path = `M 50,330 L ${puntos.join(' L ')} L ${50 + (1100)},330 Z`
  return path
}

function getColorOEE(oee) {
  if (oee >= 85) return '#22c55e'
  if (oee >= 70) return '#f59e0b'
  return '#ef4444'
}

function getClaseOEE(oee) {
  if (oee >= 85) return 'excelente'
  if (oee >= 70) return 'aceptable'
  return 'critico'
}

function getTituloAgrupacion() {
  return 'Fecha'
}

function mostrarTooltip(event, punto) {
  tooltipData.value = punto
  tooltipX.value = event.clientX + 10
  tooltipY.value = event.clientY + 10
  tooltipVisible.value = true
}

function ocultarTooltip() {
  tooltipVisible.value = false
}

function exportarExcel() {
  const compania = companias.value.find(c => c.id === companiaSeleccionada.value)?.nombre || 'Empresa'
  const fechaReporte = new Date().toISOString().split('T')[0]

  let html = `
    <html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">
    <head>
      <meta charset="utf-8">
      <style>
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #001f54; color: white; font-weight: bold; }
      </style>
    </head>
    <body>
      <h2>Historia de Línea</h2>
      <table>
        <thead>
          <tr>
            <th>${getTituloAgrupacion()}</th>
            <th>OEE (%)</th>
            <th>Disponibilidad (%)</th>
            <th>Rendimiento (%)</th>
            <th>Calidad (%)</th>
            <th>Producción Real</th>
            <th>Producción Objetivo</th>
            <th>Tiempo Parada (min)</th>
            <th>Defectos</th>
          </tr>
        </thead>
        <tbody>
  `

  datosHistoria.value.forEach(dato => {
    html += `
      <tr>
        <td>${dato.label}</td>
        <td>${dato.oee.toFixed(1)}</td>
        <td>${dato.disponibilidad.toFixed(1)}</td>
        <td>${dato.rendimiento.toFixed(1)}</td>
        <td>${dato.calidad.toFixed(1)}</td>
        <td>${dato.produccionReal}</td>
        <td>${dato.produccionObjetivo}</td>
        <td>${dato.tiempoParada}</td>
        <td>${dato.defectos}</td>
      </tr>
    `
  })

  html += `
        </tbody>
      </table>
    </body>
    </html>
  `

  const blob = new Blob([html], { type: 'application/vnd.ms-excel' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${compania.replace(/\s+/g, '_')}_Historia_Linea_${fechaReporte}.xls`
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  generarDatosHistoria()
})
</script>

<style scoped>
.historia-linea {
  padding: 1.5rem;
  background: #f5f7fa;
  min-height: 100vh;
}

.titulo-principal {
  background: #001f54;
  color: white;
  padding: 1.5rem;
  text-align: center;
  font-size: 1.5rem;
  font-weight: bold;
  margin-bottom: 1.5rem;
  border-radius: 8px;
  letter-spacing: 1px;
}

/* Mensaje de selección */
.mensaje-seleccion {
  background: white;
  border-left: 4px solid #001f54;
  padding: 1rem 1.5rem;
  margin-bottom: 1.5rem;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.mensaje-seleccion h3 {
  margin: 0;
  color: #001f54;
  font-size: 1.125rem;
  font-weight: 600;
}

/* Formulario Grid - 3 Columnas */
.formulario-grid-nuevo {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

.seccion-box-nuevo {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
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
  letter-spacing: 0.5px;
}

.subseccion-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #475569;
  margin: 0.75rem 0 0.375rem 0;
}

.campo-grupo-nuevo {
  margin-bottom: 1rem;
  position: relative;
}

.campo-grupo-nuevo:last-child {
  margin-bottom: 0;
}

.campo-grupo-nuevo label {
  display: block;
  margin-bottom: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #475569;
}

.campo-entrada-nuevo {
  width: 100%;
  padding: 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.875rem;
  transition: all 0.2s;
  background: white;
}

.campo-entrada-nuevo:focus {
  outline: none;
  border-color: #001f54;
  box-shadow: 0 0 0 3px rgba(0, 31, 84, 0.1);
}

/* Tags de variables pequeños */
.variables-tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-height: 40px;
  padding: 0.5rem;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background: #f8fafc;
  margin-bottom: 0.5rem;
}

.variable-tag-small {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  background: #001f54;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.btn-tag-remove {
  background: transparent;
  color: white;
  border: none;
  cursor: pointer;
  font-size: 1rem;
  line-height: 1;
  padding: 0;
  margin-left: 0.25rem;
}

.btn-tag-remove:hover {
  color: #ef4444;
}

.btn-add-small {
  width: 100%;
  padding: 0.5rem;
  background: white;
  color: #001f54;
  border: 2px solid #001f54;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 0.5rem;
  transition: all 0.2s;
  font-size: 0.8125rem;
}

.btn-add-small:hover {
  background: #001f54;
  color: white;
}

/* Botón Buscar */
.seccion-buscar {
  text-align: center;
  margin-bottom: 2rem;
}

.btn-buscar-principal {
  padding: 0.875rem 3rem;
  background: #001f54;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 2px 4px rgba(0, 31, 84, 0.3);
  letter-spacing: 1px;
}

.btn-buscar-principal:hover {
  background: #003087;
  transform: translateY(-1px);
  box-shadow: 0 4px 6px rgba(0, 31, 84, 0.4);
}

/* Contenedor gráfico */
.contenedor-grafico {
  margin-top: 2rem;
}

.grafico-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding: 1rem;
  background: #001f54;
  color: white;
  border-radius: 8px;
}

.grafico-header h3 {
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

.cards-oee {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.card-oee {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.card-oee-label {
  font-size: 0.875rem;
  color: #6b7280;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.card-oee-valor {
  font-size: 2rem;
  font-weight: bold;
  color: #001f54;
  margin-bottom: 0.75rem;
}

.card-oee-barra {
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
}

.barra-progreso {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
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

.punto-grafico {
  cursor: pointer;
  transition: r 0.2s;
}

.punto-grafico:hover {
  r: 8;
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

.badge-oee {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 600;
}

.badge-oee.excelente {
  background: #dcfce7;
  color: #166534;
}

.badge-oee.aceptable {
  background: #fef3c7;
  color: #92400e;
}

.badge-oee.critico {
  background: #fee2e2;
  color: #991b1b;
}

.tooltip-grafico {
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
  .formulario-grid {
    grid-template-columns: 1fr;
  }

  .cards-oee {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .cards-oee {
    grid-template-columns: 1fr;
  }
}
</style>
