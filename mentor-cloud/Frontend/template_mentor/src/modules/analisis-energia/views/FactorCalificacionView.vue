<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'

const companias = ref([
  { id: 1, nombre: 'Art Atlas' }
])

const plantas = ref([
  { id: 1, companiaId: 1, nombre: 'Planta Textil Lima' }
])

const lineas = ref([
  { id: 1, nombre: 'LINEA 05' },
  { id: 2, nombre: 'Línea 1' }
])

const ubicaciones = ref([
  { id: 1, nombre: 'Toda la Línea' },
  { id: 2, nombre: 'Linea 05 Locacion' }
])

const dispositivos = ref([
  { id: 1, nombre: 'Medidor de Energía - Circutor' },
  { id: 2, nombre: 'Linea 05 Produccion' }
])

const periodos = ref([
  { id: 1, nombre: 'Periodo fuera de Punta', valor: 'fuera_punta' },
  { id: 2, nombre: 'Periodo Punta', valor: 'punta' },
  { id: 3, nombre: 'Todo el Periodo', valor: 'todo' }
])

const variables = ref([
  { id: 1, nombre: '∫ EReal (kWh) (Energia Real (kWh))', valor: 'energia_real' },
  { id: 2, nombre: '∫ EReactiva (kVArh) (Energia Reactiva (kVArh))', valor: 'energia_reactiva' },
  { id: 3, nombre: 'Factor de Potencia', valor: 'factor_potencia' }
])

const companiaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const lineaSeleccionada = ref(1)
const ubicacionSeleccionada = ref(1)
const dispositivoSeleccionado = ref(1)
const periodoSeleccionado = ref('fuera_punta')
const variableSeleccionada = ref('energia_real')

const fechaInicio = ref('2025/11/01 00:00')
const fechaFin = ref('2025/11/19 23:59')

const fechaInicioInput = computed({
  get: () => {
    const [fecha, hora] = fechaInicio.value.split(' ')
    const [year, month, day] = fecha.split('/')
    return `${year}-${month}-${day}T${hora}`
  },
  set: (value) => {
    if (value) {
      const date = new Date(value)
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      fechaInicio.value = `${year}/${month}/${day} ${hours}:${minutes}`
    }
  }
})

const fechaFinInput = computed({
  get: () => {
    const [fecha, hora] = fechaFin.value.split(' ')
    const [year, month, day] = fecha.split('/')
    return `${year}-${month}-${day}T${hora}`
  },
  set: (value) => {
    if (value) {
      const date = new Date(value)
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      fechaFin.value = `${year}/${month}/${day} ${hours}:${minutes}`
    }
  }
})

const plantasFiltradas = computed(() => {
  if (!companiaSeleccionada.value) return []
  return plantas.value.filter(p => p.companiaId === parseInt(companiaSeleccionada.value))
})

const datosFactorCalificacion = ref([])
const mostrarResultados = ref(false)
const datosGrafico = ref({})

function buscar() {
  generarDatosFactorCalificacion()
  mostrarResultados.value = true
}

function generarDatosFactorCalificacion() {
  const datos = []
  const fechaInicial = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T') + ':00')
  const fechaFinal = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T') + ':00')
  
  const dias = Math.ceil((fechaFinal - fechaInicial) / (1000 * 60 * 60 * 24))
  
  for (let i = 0; i < dias; i++) {
    const fecha = new Date(fechaInicial.getTime() + (i * 24 * 60 * 60 * 1000))
    
    const potenciaActiva = 200 + Math.random() * 100
    const potenciaReactiva = 50 + Math.random() * 40
    const potenciaAparente = Math.sqrt(Math.pow(potenciaActiva, 2) + Math.pow(potenciaReactiva, 2))
    const factorPotencia = potenciaActiva / potenciaAparente
    
    // Factor de calificación (penalización si FP < 0.96)
    let factorCalificacion = 1.0
    if (factorPotencia < 0.96) {
      factorCalificacion = 0.96 / factorPotencia
    }
    
    const energiaActiva = potenciaActiva * 24 // kWh por día
    const energiaReactiva = potenciaReactiva * 24 // kVArh por día
    const penalizacion = energiaActiva * (factorCalificacion - 1)
    
    datos.push({
      fecha: fecha.toISOString().split('T')[0],
      energiaActiva: Math.round(energiaActiva * 100) / 100,
      energiaReactiva: Math.round(energiaReactiva * 100) / 100,
      factorPotencia: Math.round(factorPotencia * 1000) / 1000,
      factorCalificacion: Math.round(factorCalificacion * 1000) / 1000,
      penalizacion: Math.round(penalizacion * 100) / 100
    })
  }
  
  datosFactorCalificacion.value = datos
  
  // Generar datos para gráfico
  datosGrafico.value = {
    factorPotencia: datos.map(d => ({ fecha: d.fecha, valor: d.factorPotencia })),
    factorCalificacion: datos.map(d => ({ fecha: d.fecha, valor: d.factorCalificacion }))
  }
}

function generarPathMetrica(metrica) {
  const datos = datosGrafico.value[metrica]
  if (!datos || datos.length === 0) return ''
  
  const maxValor = Math.max(...datos.map(d => d.valor))
  const minValor = Math.min(...datos.map(d => d.valor))
  const rango = maxValor - minValor || 1
  
  let path = ''
  datos.forEach((punto, index) => {
    const x = (index / (datos.length - 1)) * 1200
    const y = 280 - (((punto.valor - minValor) / rango) * 260)
    
    if (index === 0) {
      path += `M ${x} ${y}`
    } else {
      path += ` L ${x} ${y}`
    }
  })
  
  return path
}

function formatearFecha(fecha) {
  const date = new Date(fecha)
  const dia = String(date.getDate()).padStart(2, '0')
  const mes = String(date.getMonth() + 1).padStart(2, '0')
  return `${dia}/${mes}`
}

const tooltipVisible = ref(false)
const tooltipData = ref({})
const tooltipStyle = ref({})

function mostrarTooltip(event, punto, metrica) {
  tooltipData.value = { ...punto, metrica }
  tooltipVisible.value = true
  
  const rect = event.target.getBoundingClientRect()
  tooltipStyle.value = {
    position: 'fixed',
    left: `${rect.left + rect.width / 2}px`,
    top: `${rect.top}px`,
    transform: 'translate(-50%, -100%)',
    pointerEvents: 'none',
    zIndex: 9999
  }
}

function ocultarTooltip() {
  tooltipVisible.value = false
}

const promedios = computed(() => {
  if (datosFactorCalificacion.value.length === 0) return {}
  
  const count = datosFactorCalificacion.value.length
  
  return {
    factorPotencia: (datosFactorCalificacion.value.reduce((sum, d) => sum + d.factorPotencia, 0) / count).toFixed(3),
    factorCalificacion: (datosFactorCalificacion.value.reduce((sum, d) => sum + d.factorCalificacion, 0) / count).toFixed(3),
    penalizacionTotal: datosFactorCalificacion.value.reduce((sum, d) => sum + d.penalizacion, 0).toFixed(2)
  }
})

const estadoCalificacion = computed(() => {
  const fp = parseFloat(promedios.value.factorPotencia)
  if (fp >= 0.96) return { estado: 'ÓPTIMO', color: '#22c55e', icon: '✓' }
  if (fp >= 0.92) return { estado: 'ACEPTABLE', color: '#f59e0b', icon: '⚠' }
  return { estado: 'CRÍTICO', color: '#ef4444', icon: '✕' }
})

function exportarExcel() {
  if (datosFactorCalificacion.value.length === 0) return
  
  // Obtener nombres de variables y configuración seleccionada
  const variableNombre = variables.value.find(v => v.valor === variableSeleccionada.value)?.nombre || 'Variable'
  const dispositivoNombre = dispositivos.value.find(d => d.id === dispositivoSeleccionado.value)?.nombre || 'Dispositivo'
  const periodoNombre = periodos.value.find(p => p.valor === periodoSeleccionado.value)?.nombre || 'Periodo'
  
  let html = `
    <html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">
    <head>
      <meta charset="UTF-8">
      <xml>
        <x:ExcelWorkbook>
          <x:ExcelWorksheets>
            <x:ExcelWorksheet>
              <x:Name>Factor de Calificación</x:Name>
              <x:WorksheetOptions><x:DisplayGridlines/></x:WorksheetOptions>
            </x:ExcelWorksheet>
          </x:ExcelWorksheets>
        </x:ExcelWorkbook>
      </xml>
    </head>
    <body>
      <h2>Factor de Calificación</h2>
      <p><strong>Variable:</strong> ${variableNombre}</p>
      <p><strong>Dispositivo:</strong> ${dispositivoNombre}</p>
      <p><strong>Periodo:</strong> ${periodoNombre}</p>
      <p><strong>Fecha:</strong> ${fechaInicio.value} - ${fechaFin.value}</p>
      <br/>
      <table border="1">
        <thead>
          <tr>
            <th>Fecha</th>
            <th>Energía Activa (kWh)</th>
            <th>Energía Reactiva (kVArh)</th>
            <th>Factor de Potencia</th>
            <th>Factor de Calificación</th>
            <th>Penalización (kWh)</th>
            <th>Estado</th>
          </tr>
        </thead>
        <tbody>
  `
  
  datosFactorCalificacion.value.forEach(dato => {
    const estado = dato.factorPotencia >= 0.96 ? 'ÓPTIMO' : dato.factorPotencia >= 0.92 ? 'ACEPTABLE' : 'CRÍTICO'
    html += `
      <tr>
        <td>${formatearFecha(dato.fecha)}</td>
        <td>${dato.energiaActiva.toFixed(2)}</td>
        <td>${dato.energiaReactiva.toFixed(2)}</td>
        <td>${dato.factorPotencia.toFixed(3)}</td>
        <td>${dato.factorCalificacion.toFixed(3)}</td>
        <td>${dato.penalizacion.toFixed(2)}</td>
        <td>${estado}</td>
      </tr>
    `
  })
  
  html += `
          <tr style="font-weight: bold; background-color: #f0f9ff;">
            <td colspan="3">PROMEDIOS</td>
            <td>${promedios.value.factorPotencia}</td>
            <td>${promedios.value.factorCalificacion}</td>
            <td>${promedios.value.penalizacionTotal}</td>
            <td>${estadoCalificacion.value.estado}</td>
          </tr>
        </tbody>
      </table>
    </body>
    </html>
  `
  
  const blob = new Blob([html], { type: 'application/vnd.ms-excel' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  
  // Nombre del archivo con información de la configuración
  const compania = companias.value.find(c => c.id === companiaSeleccionada.value)?.nombre || 'Factor'
  const fechaReporte = new Date().toISOString().split('T')[0]
  link.download = `${compania.replace(/\s+/g, '_')}_Factor_Calificacion_${fechaReporte}.xls`
  
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  
})
</script>

<template>
  <div class="factor-container">
    <h1 class="titulo-principal">FACTOR DE CALIFICACIÓN</h1>
    
    <div class="subtitulo-seccion">Seleccione la variable a monitorear:</div>

    <div class="formulario-grid">
      <!-- FECHAS -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">FECHAS</h3>
        <div class="campo-grupo">
          <label>Inicio</label>
          <input type="datetime-local" v-model="fechaInicioInput" />
        </div>
        <div class="campo-grupo">
          <label>Fin</label>
          <input type="datetime-local" v-model="fechaFinInput" />
        </div>
        <div class="campo-grupo">
          <label>Periodo</label>
          <select v-model="periodoSeleccionado">
            <option v-for="periodo in periodos" :key="periodo.id" :value="periodo.valor">
              {{ periodo.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- CONFIGURACIÓN -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">Compañía</h3>
        <div class="campo-grupo">
          <select v-model="companiaSeleccionada">
            <option v-for="compania in companias" :key="compania.id" :value="compania.id">
              {{ compania.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Planta</h3>
        <div class="campo-grupo">
          <select v-model="plantaSeleccionada">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Línea</h3>
        <div class="campo-grupo">
          <select v-model="lineaSeleccionada">
            <option v-for="linea in lineas" :key="linea.id" :value="linea.id">
              {{ linea.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Locación</h3>
        <div class="campo-grupo">
          <select v-model="ubicacionSeleccionada">
            <option v-for="ubicacion in ubicaciones" :key="ubicacion.id" :value="ubicacion.id">
              {{ ubicacion.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- DISPOSITIVO Y VARIABLE -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">Dispositivo</h3>
        <div class="campo-grupo">
          <select v-model="dispositivoSeleccionado">
            <option v-for="dispositivo in dispositivos" :key="dispositivo.id" :value="dispositivo.id">
              {{ dispositivo.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Variable</h3>
        <div class="campo-grupo">
          <select v-model="variableSeleccionada">
            <option v-for="variable in variables" :key="variable.id" :value="variable.valor">
              {{ variable.nombre }}
            </option>
          </select>
        </div>
      </div>
    </div>

    <div class="buscar-container">
      <Button @click="buscar" variant="primary" class="btn-buscar">
        BUSCAR
      </Button>
    </div>

    <!-- RESULTADOS -->
    <div v-if="mostrarResultados && datosFactorCalificacion.length > 0">
      <!-- TARJETAS DE INDICADORES -->
      <div class="indicadores-grid">
        <div class="indicador-card estado-card" :style="{ borderLeftColor: estadoCalificacion.color }">
          <div class="indicador-icon" :style="{ color: estadoCalificacion.color }">{{ estadoCalificacion.icon }}</div>
          <div class="indicador-content">
            <div class="indicador-label">ESTADO</div>
            <div class="indicador-valor" :style="{ color: estadoCalificacion.color }">{{ estadoCalificacion.estado }}</div>
          </div>
        </div>
        
        <div class="indicador-card">
          <div class="indicador-content">
            <div class="indicador-label">FACTOR DE POTENCIA PROMEDIO</div>
            <div class="indicador-valor">{{ promedios.factorPotencia }}</div>
            <div class="indicador-meta">Meta: ≥ 0.96</div>
          </div>
        </div>
        
        <div class="indicador-card">
          <div class="indicador-content">
            <div class="indicador-label">FACTOR DE CALIFICACIÓN PROMEDIO</div>
            <div class="indicador-valor">{{ promedios.factorCalificacion }}</div>
            <div class="indicador-meta">Ideal: 1.000</div>
          </div>
        </div>
        
        <div class="indicador-card penalizacion-card">
          <div class="indicador-content">
            <div class="indicador-label">PENALIZACIÓN TOTAL</div>
            <div class="indicador-valor penalizacion">{{ promedios.penalizacionTotal }} kWh</div>
          </div>
        </div>
      </div>

      <!-- GRÁFICO -->
      <div class="grafico-container">
        <div class="grafico-header">
          <h3>Evolución del Factor de Potencia y Calificación</h3>
          <div class="leyenda-tarifas">
            <div class="leyenda-item">
              <span class="leyenda-color" style="background-color: #3b82f6;"></span>
              <span>Factor de Potencia</span>
            </div>
            <div class="leyenda-item">
              <span class="leyenda-color" style="background-color: #ef4444;"></span>
              <span>Factor de Calificación</span>
            </div>
            <div class="leyenda-item referencia">
              <span class="leyenda-linea"></span>
              <span>Meta (0.96)</span>
            </div>
          </div>
        </div>
        
        <svg viewBox="0 0 1200 300" preserveAspectRatio="none" class="grafico-svg">
          <defs>
            <filter id="shadowFactor">
              <feGaussianBlur in="SourceAlpha" stdDeviation="2"/>
              <feOffset dx="0" dy="2" result="offsetblur"/>
              <feComponentTransfer>
                <feFuncA type="linear" slope="0.3"/>
              </feComponentTransfer>
              <feMerge>
                <feMergeNode/>
                <feMergeNode in="SourceGraphic"/>
              </feMerge>
            </filter>
          </defs>

          <!-- Línea de referencia 0.96 -->
          <line 
            x1="0" 
            y1="150" 
            x2="1200" 
            y2="150"
            stroke="#22c55e"
            stroke-width="1.5"
            stroke-dasharray="5,5"
            opacity="0.5"
          />

          <!-- Factor de Potencia -->
          <path 
            :d="generarPathMetrica('factorPotencia')" 
            fill="none" 
            stroke="#3b82f6"
            stroke-width="2.5"
            filter="url(#shadowFactor)"
          />
          
          <!-- Factor de Calificación -->
          <path 
            :d="generarPathMetrica('factorCalificacion')" 
            fill="none" 
            stroke="#ef4444"
            stroke-width="2.5"
            filter="url(#shadowFactor)"
          />
          
          <!-- Puntos Factor de Potencia -->
          <circle 
            v-for="(punto, index) in datosGrafico.factorPotencia" 
            :key="`fp-${index}`"
            v-show="index % Math.max(1, Math.floor(datosGrafico.factorPotencia.length / 20)) === 0"
            :cx="(index / (datosGrafico.factorPotencia.length - 1)) * 1200"
            :cy="280 - (((punto.valor - Math.min(...datosGrafico.factorPotencia.map(d => d.valor), ...datosGrafico.factorCalificacion.map(d => d.valor))) / (Math.max(...datosGrafico.factorPotencia.map(d => d.valor), ...datosGrafico.factorCalificacion.map(d => d.valor)) - Math.min(...datosGrafico.factorPotencia.map(d => d.valor), ...datosGrafico.factorCalificacion.map(d => d.valor)))) * 260)"
            r="4"
            fill="#3b82f6"
            class="punto-interactivo"
            @mouseenter="mostrarTooltip($event, punto, 'Factor de Potencia')"
            @mouseleave="ocultarTooltip"
          />
          
          <!-- Puntos Factor de Calificación -->
          <circle 
            v-for="(punto, index) in datosGrafico.factorCalificacion" 
            :key="`fc-${index}`"
            v-show="index % Math.max(1, Math.floor(datosGrafico.factorCalificacion.length / 20)) === 0"
            :cx="(index / (datosGrafico.factorCalificacion.length - 1)) * 1200"
            :cy="280 - (((punto.valor - Math.min(...datosGrafico.factorPotencia.map(d => d.valor), ...datosGrafico.factorCalificacion.map(d => d.valor))) / (Math.max(...datosGrafico.factorPotencia.map(d => d.valor), ...datosGrafico.factorCalificacion.map(d => d.valor)) - Math.min(...datosGrafico.factorPotencia.map(d => d.valor), ...datosGrafico.factorCalificacion.map(d => d.valor)))) * 260)"
            r="4"
            fill="#ef4444"
            class="punto-interactivo"
            @mouseenter="mostrarTooltip($event, punto, 'Factor de Calificación')"
            @mouseleave="ocultarTooltip"
          />
        </svg>

        <Teleport to="body">
          <div v-if="tooltipVisible" :style="tooltipStyle" class="tooltip-grafico">
            <div class="tooltip-contenido">
              <div class="tooltip-titulo">{{ tooltipData.metrica }}</div>
              <div class="tooltip-fecha">{{ formatearFecha(tooltipData.fecha) }}</div>
              <div class="tooltip-valor">{{ tooltipData.valor ? tooltipData.valor.toFixed(3) : '0.000' }}</div>
            </div>
          </div>
        </Teleport>
      </div>

      <!-- TABLA -->
      <div class="tabla-container">
        <div class="tabla-header">
          <h3>Detalle Diario del Factor de Calificación</h3>
          <Button @click="exportarExcel" variant="secondary" class="btn-exportar">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" style="margin-right: 0.5rem;">
              <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4m4-5l5 5m0 0l5-5m-5 5V3" stroke="currentColor" stroke-width="2"/>
            </svg>
            Exportar Excel
          </Button>
        </div>
        <div class="tabla-scroll">
          <table class="tabla-factor">
            <thead>
              <tr>
                <th>Fecha</th>
                <th>Energía Activa (kWh)</th>
                <th>Energía Reactiva (kVArh)</th>
                <th>Factor de Potencia</th>
                <th>Factor de Calificación</th>
                <th>Penalización (kWh)</th>
                <th>Estado</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(dato, index) in datosFactorCalificacion" :key="index">
                <td>{{ formatearFecha(dato.fecha) }}</td>
                <td>{{ dato.energiaActiva.toFixed(2) }}</td>
                <td>{{ dato.energiaReactiva.toFixed(2) }}</td>
                <td>{{ dato.factorPotencia.toFixed(3) }}</td>
                <td :class="{ 'penalizado': dato.factorCalificacion > 1 }">
                  {{ dato.factorCalificacion.toFixed(3) }}
                </td>
                <td class="penalizacion-valor">{{ dato.penalizacion.toFixed(2) }}</td>
                <td>
                  <span 
                    class="estado-badge"
                    :class="{
                      'estado-optimo': dato.factorPotencia >= 0.96,
                      'estado-aceptable': dato.factorPotencia >= 0.92 && dato.factorPotencia < 0.96,
                      'estado-critico': dato.factorPotencia < 0.92
                    }"
                  >
                    {{ dato.factorPotencia >= 0.96 ? 'ÓPTIMO' : dato.factorPotencia >= 0.92 ? 'ACEPTABLE' : 'CRÍTICO' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.factor-container {
  padding: 1.5rem;
}

.titulo-principal {
  font-size: 1.6rem;
  font-weight: 700;
  color: white;
  background: #001f54;
  padding: 1rem 1.5rem;
  margin: -1.5rem -1.5rem 1.5rem -1.5rem;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.subtitulo-seccion {
  font-size: 1.1rem;
  font-weight: 600;
  color: white;
  background: #001f54;
  padding: 0.8rem 1.5rem;
  margin: -0.5rem -1.5rem 1.5rem -1.5rem;
}

.formulario-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.seccion-box {
  background-color: #f5f5f5;
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 1rem;
}

.seccion-titulo {
  font-size: 0.9rem;
  font-weight: 700;
  color: #001f54;
  margin-bottom: 0.6rem;
  text-transform: none;
  letter-spacing: 0;
}

.mt-3 {
  margin-top: 1rem;
}

.campo-grupo {
  margin-bottom: 0.6rem;
}

.campo-grupo:last-child {
  margin-bottom: 0;
}

.campo-grupo label {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.3rem;
}

.campo-grupo input,
.campo-grupo select {
  width: 100%;
  padding: 0.5rem 0.7rem;
  font-size: 0.875rem;
  border: 1px solid #ccc;
  border-radius: 4px;
  background-color: white;
  transition: all 0.2s;
}

.campo-grupo input:focus,
.campo-grupo select:focus {
  outline: none;
  border-color: #001f54;
  box-shadow: 0 0 0 2px rgba(0, 31, 84, 0.1);
}

.buscar-container {
  display: flex;
  justify-content: center;
  margin: 2rem 0;
}

.btn-buscar {
  padding: 0.7rem 4rem;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: 1px;
  text-transform: uppercase;
  background: #4a4a4a;
  border: none;
  color: white;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-buscar:hover {
  background: #333;
}

.indicadores-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.indicador-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 1.2rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 1rem;
}

.estado-card {
  border-left: 4px solid;
}

.penalizacion-card {
  background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
  border-left: 4px solid #ef4444;
}

.indicador-icon {
  font-size: 2rem;
  font-weight: 700;
}

.indicador-content {
  flex: 1;
}

.indicador-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 0.5rem;
}

.indicador-valor {
  font-size: 1.6rem;
  font-weight: 700;
  color: #001f54;
}

.indicador-valor.penalizacion {
  color: #ef4444;
}

.indicador-meta {
  font-size: 0.75rem;
  color: #9ca3af;
  margin-top: 0.3rem;
}

.grafico-container {
  margin: 2rem 0;
  background-color: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.grafico-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.grafico-header h3 {
  font-size: 1.1rem;
  font-weight: 700;
  color: #001f54;
}

.leyenda-tarifas {
  display: flex;
  gap: 1.5rem;
}

.leyenda-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: #374151;
}

.leyenda-item.referencia {
  opacity: 0.7;
}

.leyenda-color {
  width: 16px;
  height: 16px;
  border-radius: 3px;
}

.leyenda-linea {
  width: 20px;
  height: 2px;
  background: #22c55e;
  display: inline-block;
}

.grafico-svg {
  width: 100%;
  height: 300px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: linear-gradient(to bottom, #f9fafb 0%, white 100%);
}

.punto-interactivo {
  cursor: pointer;
  transition: all 0.2s;
}

.punto-interactivo:hover {
  r: 6;
  filter: drop-shadow(0 0 4px currentColor);
}

.tooltip-grafico {
  animation: fadeIn 0.2s;
  margin-bottom: 10px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translate(-50%, -100%) scale(0.9);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -100%) scale(1);
  }
}

.tooltip-contenido {
  background: rgba(30, 41, 59, 0.95);
  backdrop-filter: blur(8px);
  color: white;
  padding: 0.6rem 0.9rem;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  white-space: nowrap;
}

.tooltip-titulo {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #94a3b8;
  margin-bottom: 0.2rem;
}

.tooltip-fecha {
  font-size: 0.75rem;
  color: #cbd5e1;
  margin-bottom: 0.3rem;
}

.tooltip-valor {
  font-size: 0.95rem;
  font-weight: 700;
  color: #60a5fa;
}

.tabla-container {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.tabla-header {
  background: #001f54;
  color: white;
  padding: 1rem 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.tabla-header h3 {
  font-size: 1rem;
  font-weight: 700;
  margin: 0;
}

.btn-exportar {
  display: flex;
  align-items: center;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  background: white;
  color: #001f54;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s;
}

.btn-exportar:hover {
  background: #f0f9ff;
  transform: translateY(-1px);
}

.tabla-scroll {
  overflow-x: auto;
  max-height: 500px;
  overflow-y: auto;
}

.tabla-factor {
  width: 100%;
  border-collapse: collapse;
}

.tabla-factor thead {
  background: #f9fafb;
  position: sticky;
  top: 0;
  z-index: 10;
}

.tabla-factor th {
  padding: 0.8rem 1rem;
  text-align: left;
  font-size: 0.85rem;
  font-weight: 700;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
}

.tabla-factor tbody tr {
  border-bottom: 1px solid #f3f4f6;
  transition: background-color 0.15s;
}

.tabla-factor tbody tr:hover {
  background-color: #f9fafb;
}

.tabla-factor td {
  padding: 0.7rem 1rem;
  font-size: 0.85rem;
  color: #374151;
}

.tabla-factor td.penalizado {
  color: #ef4444;
  font-weight: 600;
}

.tabla-factor td.penalizacion-valor {
  color: #dc2626;
  font-weight: 600;
}

.estado-badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.estado-optimo {
  background: #d1fae5;
  color: #065f46;
}

.estado-aceptable {
  background: #fef3c7;
  color: #92400e;
}

.estado-critico {
  background: #fee2e2;
  color: #991b1b;
}

@media (max-width: 768px) {
  .formulario-grid {
    grid-template-columns: 1fr;
  }

  .indicadores-grid {
    grid-template-columns: 1fr 1fr;
  }

  .titulo-principal {
    font-size: 1.2rem;
  }

  .subtitulo-seccion {
    font-size: 0.95rem;
  }
}
</style>
