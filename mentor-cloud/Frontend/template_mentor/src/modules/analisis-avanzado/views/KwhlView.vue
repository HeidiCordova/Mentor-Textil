<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import { useApi } from '@/shared/composables/useApi'

const { loading, error, execute } = useApi()

const companias = ref([
  { id: 1, nombre: 'Industrias San Miguel' }
])

const plantas = ref([
  { id: 1, companiaId: 1, nombre: 'ISM Arequipa' }
])

const lineas = ref([
  { id: 1, nombre: 'LINEA 05' }
])

const ubicaciones = ref([
  { id: 1, nombre: 'Linea 05 Locacion' }
])

const dispositivos = ref([
  { id: 1, nombre: 'Linea 05 Produccion' }
])

const clasificaciones = ref([
  { id: 1, nombre: 'Cálculo', valor: 'calculo' },
  { id: 2, nombre: 'Lectura', valor: 'lectura' }
])

const agrupamientos = ref([
  { id: 1, nombre: '5 minutos', valor: '5min' },
  { id: 2, nombre: '15 minutos', valor: '15min' },
  { id: 3, nombre: '30 minutos', valor: '30min' },
  { id: 4, nombre: '1 hora', valor: '1h' }
])

const variablesEnergia = ref([
  { id: 1, nombre: 'Marca (PLC-L5-Marca)' }
])

const variablesProduccion = ref([
  { id: 1, nombre: 'L5-C-Litros (L5-Conteo-Produccion Litros)' }
])

const variablesLimite = ref([
  { id: 1, nombre: 'L5-T-Parada No Obligatoria (L5-Tiempo-Parada No Obligatoria (calculo OEE))' }
])

const companiaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const lineaSeleccionada = ref(1)
const ubicacionSeleccionada = ref(1)
const dispositivoSeleccionado = ref(1)
const clasificacionSeleccionada = ref('calculo')
const agrupamientoSeleccionado = ref('5min')
const variableEnergiaSeleccionada = ref(1)
const variableProduccionSeleccionada = ref(1)
const variableLimiteSeleccionada = ref(1)

const fechaInicio = ref('2025/11/14 00:00')
const fechaFin = ref('2025/11/19 06:55')

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

const datosGrafico = ref([])
const mostrarResultados = ref(false)

function buscar() {
  generarDatosGrafico()
  mostrarResultados.value = true
}

function generarDatosGrafico() {
  const datos = []
  const fechaInicial = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T') + ':00')
  const fechaFinal = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T') + ':00')
  
  const diferenciaMs = fechaFinal - fechaInicial
  const cantidadPuntos = 50
  const intervaloMs = diferenciaMs / (cantidadPuntos - 1)
  
  for (let i = 0; i < cantidadPuntos; i++) {
    const fecha = new Date(fechaInicial.getTime() + (i * intervaloMs))
    const kwhLitro = 0.15 + Math.sin(i / 8) * 0.03 + Math.random() * 0.02
    
    datos.push({
      fecha: fecha.toISOString(),
      kwhLitro: Math.round(kwhLitro * 1000) / 1000
    })
  }
  
  datosGrafico.value = datos
}

function generarPathLinea() {
  if (datosGrafico.value.length === 0) return ''
  
  const maxValor = Math.max(...datosGrafico.value.map(d => d.kwhLitro))
  const minValor = Math.min(...datosGrafico.value.map(d => d.kwhLitro))
  const rango = maxValor - minValor
  
  let path = ''
  datosGrafico.value.forEach((punto, index) => {
    const x = (index / (datosGrafico.value.length - 1)) * 1200
    const y = 280 - (((punto.kwhLitro - minValor) / rango) * 260)
    
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
  const hora = String(date.getHours()).padStart(2, '0')
  const minutos = String(date.getMinutes()).padStart(2, '0')
  return `${dia}/${mes} ${hora}:${minutos}`
}

const tooltipVisible = ref(false)
const tooltipData = ref({})
const tooltipStyle = ref({})

function mostrarTooltip(event, punto) {
  tooltipData.value = punto
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

onMounted(() => {
  
})
</script>

<template>
  <div class="kwhl-container">
    <h1 class="titulo-principal">KWHL</h1>

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
          <label>Clasificación</label>
          <select v-model="clasificacionSeleccionada">
            <option v-for="clasificacion in clasificaciones" :key="clasificacion.id" :value="clasificacion.valor">
              {{ clasificacion.nombre }}
            </option>
          </select>
        </div>
        <div class="campo-grupo">
          <label>Agrupamiento</label>
          <select v-model="agrupamientoSeleccionado">
            <option v-for="agrupamiento in agrupamientos" :key="agrupamiento.id" :value="agrupamiento.valor">
              {{ agrupamiento.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- CONFIGURAR -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">CONFIGURAR</h3>
        <div class="campo-grupo">
          <label>Compañía</label>
          <select v-model="companiaSeleccionada">
            <option v-for="compania in companias" :key="compania.id" :value="compania.id">
              {{ compania.nombre }}
            </option>
          </select>
        </div>
        <div class="campo-grupo">
          <label>Planta</label>
          <select v-model="plantaSeleccionada">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
        <div class="campo-grupo">
          <label>Línea</label>
          <select v-model="lineaSeleccionada">
            <option v-for="linea in lineas" :key="linea.id" :value="linea.id">
              {{ linea.nombre }}
            </option>
          </select>
        </div>
        <div class="campo-grupo">
          <label>Locación</label>
          <select v-model="ubicacionSeleccionada">
            <option v-for="ubicacion in ubicaciones" :key="ubicacion.id" :value="ubicacion.id">
              {{ ubicacion.nombre }}
            </option>
          </select>
        </div>
        <div class="campo-grupo">
          <label>Dispositivo</label>
          <select v-model="dispositivoSeleccionado">
            <option v-for="dispositivo in dispositivos" :key="dispositivo.id" :value="dispositivo.id">
              {{ dispositivo.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- VARIABLES -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">VARIABLES</h3>
        
        <div class="variable-grupo">
          <label>VARIABLE DE ENERGÍA</label>
          <select v-model="variableEnergiaSeleccionada">
            <option v-for="variable in variablesEnergia" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <div class="variable-grupo">
          <label>VARIABLE DE PRODUCCIÓN</label>
          <select v-model="variableProduccionSeleccionada">
            <option v-for="variable in variablesProduccion" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>

        <div class="variable-grupo">
          <label>LÍMITE</label>
          <select v-model="variableLimiteSeleccionada">
            <option v-for="variable in variablesLimite" :key="variable.id" :value="variable.id">
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

    <!-- GRÁFICO -->
    <div v-if="mostrarResultados && datosGrafico.length > 0" class="grafico-container">
      <div class="grafico-header">
        <h3>KWH/L a través del tiempo</h3>
      </div>
      
      <svg viewBox="0 0 1200 300" preserveAspectRatio="none" class="grafico-svg">
        <defs>
          <linearGradient id="gradientKwhl" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" style="stop-color:#3b82f6;stop-opacity:0.3" />
            <stop offset="100%" style="stop-color:#3b82f6;stop-opacity:0.05" />
          </linearGradient>
          <filter id="shadow">
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

        <path 
          :d="generarPathLinea()" 
          fill="none" 
          stroke="#3b82f6" 
          stroke-width="2.5"
          filter="url(#shadow)"
        />
        
        <circle 
          v-for="(punto, index) in datosGrafico" 
          :key="index"
          v-show="index % 2 === 0"
          :cx="(index / (datosGrafico.length - 1)) * 1200"
          :cy="280 - (((punto.kwhLitro - Math.min(...datosGrafico.map(d => d.kwhLitro))) / (Math.max(...datosGrafico.map(d => d.kwhLitro)) - Math.min(...datosGrafico.map(d => d.kwhLitro)))) * 260)"
          r="5"
          fill="#3b82f6"
          class="punto-interactivo"
          @mouseenter="mostrarTooltip($event, punto)"
          @mouseleave="ocultarTooltip"
        />
      </svg>

      <Teleport to="body">
        <div v-if="tooltipVisible" :style="tooltipStyle" class="tooltip-grafico">
          <div class="tooltip-contenido">
            <div class="tooltip-titulo">KWH/L</div>
            <div class="tooltip-fecha">{{ formatearFecha(tooltipData.fecha) }}</div>
            <div class="tooltip-valor">{{ tooltipData.kwhLitro ? tooltipData.kwhLitro.toFixed(3) : '0.000' }} kWh/L</div>
          </div>
        </div>
      </Teleport>
    </div>
  </div>
</template>

<style scoped>
.kwhl-container {
  padding: 1.5rem;
}

.titulo-principal {
  font-size: 1.6rem;
  font-weight: 700;
  color: #001f54;
  margin-bottom: 1.5rem;
}

.formulario-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.seccion-box {
  background-color: #fafafa;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 1rem;
}

.seccion-titulo {
  font-size: 0.95rem;
  font-weight: 700;
  color: #001f54;
  margin-bottom: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.campo-grupo {
  margin-bottom: 0.8rem;
}

.campo-grupo:last-child {
  margin-bottom: 0;
}

.campo-grupo label {
  display: block;
  font-size: 0.8rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.3rem;
}

.campo-grupo input,
.campo-grupo select {
  width: 100%;
  padding: 0.45rem 0.6rem;
  font-size: 0.85rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background-color: white;
  transition: all 0.2s;
}

.campo-grupo input:focus,
.campo-grupo select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.variable-grupo {
  margin-bottom: 0.8rem;
}

.variable-grupo:last-child {
  margin-bottom: 0;
}

.variable-grupo label {
  display: block;
  font-size: 0.75rem;
  font-weight: 700;
  color: #001f54;
  margin-bottom: 0.3rem;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.variable-grupo select {
  width: 100%;
  padding: 0.45rem 0.6rem;
  font-size: 0.8rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background-color: white;
  transition: all 0.2s;
}

.buscar-container {
  display: flex;
  justify-content: center;
  margin: 1.5rem 0;
}

.btn-buscar {
  padding: 0.6rem 3rem;
  font-size: 0.95rem;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.grafico-container {
  margin-top: 2rem;
  background-color: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.grafico-header {
  margin-bottom: 1rem;
}

.grafico-header h3 {
  font-size: 1.1rem;
  font-weight: 700;
  color: #001f54;
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
  r: 7;
  filter: drop-shadow(0 0 4px rgba(59, 130, 246, 0.6));
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
</style>
