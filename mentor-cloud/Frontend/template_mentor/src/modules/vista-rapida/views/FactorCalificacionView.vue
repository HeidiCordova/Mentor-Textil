<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import { useApi } from '@/shared/composables/useApi'

const { loading, error, execute } = useApi()

const empresas = ref([
  { id: 1, nombre: 'Art Atlas' }
])

const plantas = ref([
  { id: 1, empresaId: 1, nombre: 'Planta Textil Lima' }
])

const reportes = ref([
  { id: 1, nombre: 'Factor de Calificacion de Suministro' },
  { id: 2, nombre: 'Factor de Calificacion Detallado' }
])

const empresaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const reporteSeleccionado = ref(1)
const fechaInicio = ref('2025/11/01 00:00')
const fechaFin = ref('2025/11/19 05:45')

const plantasFiltradas = computed(() => {
  if (!empresaSeleccionada.value) return []
  return plantas.value.filter(p => p.empresaId === parseInt(empresaSeleccionada.value))
})

const factorCalificacion = ref(0.5)
const metaCalificacion = ref(0.5)
const puntoHover = ref(null)
const posicionTooltip = ref({ x: 0, y: 0 })

const datosGrafico = ref([
  { tiempo: 0, valor: 0.52, fecha: '01/11/2025 00:00' },
  { tiempo: 10, valor: 0.51, fecha: '03/11/2025 06:00' },
  { tiempo: 20, valor: 0.53, fecha: '05/11/2025 12:00' },
  { tiempo: 30, valor: 0.52, fecha: '07/11/2025 18:00' },
  { tiempo: 40, valor: 0.54, fecha: '09/11/2025 20:00' },
  { tiempo: 50, valor: 0.52, fecha: '11/11/2025 22:00' },
  { tiempo: 60, valor: 0.51, fecha: '13/11/2025 08:00' },
  { tiempo: 70, valor: 0.53, fecha: '15/11/2025 14:00' },
  { tiempo: 80, valor: 0.52, fecha: '17/11/2025 16:00' },
  { tiempo: 90, valor: 0.50, fecha: '18/11/2025 22:00' },
  { tiempo: 100, valor: 0.51, fecha: '19/11/2025 05:45' }
])

const pathDatos = ref('')

function generarPathSVG() {
  if (datosGrafico.value.length === 0) return ''
  
  const width = 100
  const height = 100
  
  let path = `M 0,${100 - (datosGrafico.value[0].valor * 100)}`
  
  for (let i = 1; i < datosGrafico.value.length; i++) {
    const x = (datosGrafico.value[i].tiempo / 100) * width
    const y = 100 - (datosGrafico.value[i].valor * 100)
    path += ` L ${x},${y}`
  }
  
  return path
}

function generarPathArea() {
  if (datosGrafico.value.length === 0) return ''
  
  const width = 100
  const height = 100
  
  let path = `M 0,100 L 0,${100 - (datosGrafico.value[0].valor * 100)}`
  
  for (let i = 1; i < datosGrafico.value.length; i++) {
    const x = (datosGrafico.value[i].tiempo / 100) * width
    const y = 100 - (datosGrafico.value[i].valor * 100)
    path += ` L ${x},${y}`
  }
  
  path += ` L 100,100 Z`
  
  return path
}

function mostrarTooltip(punto, event) {
  puntoHover.value = punto
  const rect = event.target.getBoundingClientRect()
  posicionTooltip.value = {
    x: rect.left + window.scrollX,
    y: rect.top + window.scrollY - 10
  }
}

function ocultarTooltip() {
  puntoHover.value = null
}

function buscar() {
  console.log('Buscar datos')
  pathDatos.value = generarPathSVG()
}

onMounted(() => {
  pathDatos.value = generarPathSVG()
})
</script>

<template>
  <div class="factor-calificacion-view">
    <div class="page-header">
      <h1 class="page-title">FACTOR DE CALIFICACIÓN</h1>
    </div>

    <Card class="content-card">
      <div class="filtros-section">
        <div class="filter-item">
          <label class="filter-label">Inicio</label>
          <input type="text" v-model="fechaInicio" class="filter-input" />
        </div>

        <div class="filter-item">
          <label class="filter-label">Fin</label>
          <input type="text" v-model="fechaFin" class="filter-input" />
        </div>

        <div class="filter-item">
          <label class="filter-label">Planta</label>
          <select v-model="plantaSeleccionada" class="filter-select">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              Histórico Planta Textil
            </option>
          </select>
        </div>

        <div class="filter-item">
          <label class="filter-label">Reporte</label>
          <select v-model="reporteSeleccionado" class="filter-select">
            <option v-for="reporte in reportes" :key="reporte.id" :value="reporte.id">
              {{ reporte.nombre }}
            </option>
          </select>
        </div>
      </div>

      <div class="buscar-section">
        <Button variant="primary" size="md" @click="buscar">
          BUSCAR
        </Button>
      </div>

      <div class="chart-section">
        <h2 class="chart-title">FACTOR DE CALIFICACIÓN SUMINISTRO 221125</h2>
        <p class="chart-subtitle">
          Factor de Calificacion Actual:<span class="highlight-blue">FACTOR DE CALIFICACIÓN</span>{{ factorCalificacion }}
        </p>

        <div class="chart-container">
          <div class="chart-y-axis">
            <div class="y-label">FACTOR DE CALIFICACIÓN Energía Suministro</div>
            <div class="y-values">
              <span>1</span>
              <span>0.9</span>
              <span>0.8</span>
              <span>0.7</span>
              <span>0.6</span>
              <span>0.5</span>
              <span>0.4</span>
              <span>0.3</span>
              <span>0.2</span>
              <span>0.1</span>
              <span>0</span>
            </div>
          </div>

          <div class="chart-area">
            <svg class="chart-svg" viewBox="0 0 100 100" preserveAspectRatio="none">
              <defs>
                <linearGradient id="areaGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                  <stop offset="0%" style="stop-color:#3b82f6;stop-opacity:0.4" />
                  <stop offset="100%" style="stop-color:#3b82f6;stop-opacity:0.05" />
                </linearGradient>
                <filter id="shadow">
                  <feDropShadow dx="0" dy="0.5" stdDeviation="0.5" flood-color="#3b82f6" flood-opacity="0.5"/>
                </filter>
              </defs>
              
              <g class="grid-lines">
                <line v-for="i in 11" :key="i" x1="0" :y1="(i-1)*10" x2="100" :y2="(i-1)*10" stroke="#e5e7eb" stroke-width="0.2" />
              </g>
              
              <line x1="0" :y1="50" x2="100" :y2="50" stroke="#ef4444" stroke-width="0.5" stroke-dasharray="2,2" />
              
              <path v-if="pathDatos" :d="generarPathArea()" fill="url(#areaGradient)" />
              
              <path v-if="pathDatos" :d="pathDatos" fill="none" stroke="#3b82f6" stroke-width="0.8" filter="url(#shadow)" />
              
              <g v-for="(punto, index) in datosGrafico" :key="index">
                <circle 
                  :cx="punto.tiempo" 
                  :cy="100 - (punto.valor * 100)"
                  r="1.5"
                  fill="#3b82f6"
                  stroke="white"
                  stroke-width="0.5"
                  class="data-point"
                  @mouseenter="mostrarTooltip(punto, $event)"
                  @mouseleave="ocultarTooltip"
                  style="cursor: pointer;"
                />
                <circle 
                  :cx="punto.tiempo" 
                  :cy="100 - (punto.valor * 100)"
                  r="3"
                  fill="transparent"
                  @mouseenter="mostrarTooltip(punto, $event)"
                  @mouseleave="ocultarTooltip"
                  style="cursor: pointer;"
                />
              </g>
            </svg>
            <div class="line-label-meta">Meta {{ metaCalificacion }}</div>
            
            <div v-if="puntoHover" class="tooltip" :style="{ left: puntoHover.tiempo + '%', bottom: (puntoHover.valor * 100) + '%' }">
              <div class="tooltip-content">
                <div class="tooltip-fecha">{{ puntoHover.fecha }}</div>
                <div class="tooltip-valor">
                  <span class="tooltip-label">Factor:</span>
                  <span class="tooltip-number">{{ puntoHover.valor.toFixed(2) }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="chart-y-axis-right">
            <div class="y-values-right">
              <span>1</span>
              <span>0.9</span>
              <span>0.8</span>
              <span>0.7</span>
              <span>0.6</span>
              <span>0.5</span>
              <span>0.4</span>
              <span>0.3</span>
              <span>0.2</span>
              <span>0.1</span>
              <span>0</span>
            </div>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.factor-calificacion-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding-bottom: 2rem;
}

.page-header {
  background-color: #001f54;
  padding: 1rem 2rem;
  border-radius: 0;
}

.page-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: white;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.content-card {
  border-radius: 0;
  border: 1px solid #e5e7eb;
  padding: 0;
  background-color: #f5f5f5;
}

.filtros-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
  gap: 1rem;
  padding: clamp(1rem, 2vw, 1.5rem);
  background-color: white;
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

.filter-select {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background-color: white;
  color: #374151;
  cursor: pointer;
}

.filter-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background-color: white;
  color: #374151;
}

.filter-select:focus,
.filter-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.3);
}

.buscar-section {
  display: flex;
  justify-content: center;
  padding: 1rem;
  background-color: #4a4a4a;
}

.chart-section {
  padding: 2rem;
  background-color: white;
}

.chart-title {
  font-size: 1rem;
  font-weight: 700;
  color: #374151;
  text-align: center;
  margin: 0 0 0.5rem 0;
}

.chart-subtitle {
  font-size: 0.875rem;
  color: #374151;
  text-align: center;
  margin: 0 0 2rem 0;
}

.highlight-blue {
  color: #3b82f6;
  font-weight: 600;
  margin: 0 0.5rem;
}

.chart-container {
  display: flex;
  gap: 1rem;
  min-height: 400px;
  position: relative;
}

.chart-y-axis {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
}

.y-label {
  writing-mode: vertical-rl;
  transform: rotate(180deg);
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 1rem;
}

.y-values {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 100%;
  min-height: 350px;
  padding-right: 0.5rem;
}

.y-values span,
.y-values-right span {
  font-size: 0.75rem;
  color: #6b7280;
}

.chart-area {
  flex: 1;
  position: relative;
  border-left: 2px solid #374151;
  border-bottom: 2px solid #374151;
  min-height: 350px;
}

.chart-svg {
  width: 100%;
  height: 100%;
  position: absolute;
  top: 0;
  left: 0;
}

.grid-lines line {
  vector-effect: non-scaling-stroke;
}

.data-point {
  transition: all 0.2s ease;
}

.data-point:hover {
  r: 2.5;
  filter: drop-shadow(0 0 3px rgba(59, 130, 246, 0.8));
}

.line-label-meta {
  position: absolute;
  right: 1rem;
  top: 50%;
  transform: translateY(-50%);
  font-size: 0.75rem;
  color: #ef4444;
  font-weight: 600;
  background-color: white;
  padding: 0.25rem 0.5rem;
  border: 1px solid #ef4444;
  border-radius: 0.25rem;
  z-index: 10;
}

.tooltip {
  position: absolute;
  transform: translate(-50%, -100%);
  pointer-events: none;
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
  background: linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%);
  color: white;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(255, 255, 255, 0.1);
  min-width: 180px;
  position: relative;
}

.tooltip-content::after {
  content: '';
  position: absolute;
  bottom: -6px;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-top: 6px solid #3b82f6;
}

.tooltip-fecha {
  font-size: 0.75rem;
  font-weight: 500;
  margin-bottom: 0.5rem;
  opacity: 0.9;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  padding-bottom: 0.25rem;
}

.tooltip-valor {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.tooltip-label {
  font-size: 0.875rem;
  font-weight: 500;
  opacity: 0.9;
}

.tooltip-number {
  font-size: 1.25rem;
  font-weight: 700;
  color: #60a5fa;
}

.chart-y-axis-right {
  display: flex;
  flex-direction: column;
}

.y-values-right {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 100%;
  min-height: 350px;
  padding-left: 0.5rem;
}

@media (max-width: 768px) {
  .filtros-section {
    grid-template-columns: 1fr;
  }

  .chart-container {
    flex-direction: column;
  }

  .y-label {
    writing-mode: horizontal-tb;
    transform: none;
  }
}
</style>
