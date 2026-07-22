<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import { useApi } from '@/shared/composables/useApi'

const { loading, error, execute } = useApi()

const empresas = ref([
  { id: 1, nombre: 'Industrias San Miguel' }
])

const plantas = ref([
  { id: 1, empresaId: 1, nombre: 'ISM Arequipa' }
])

const reportes = ref([
  { id: 1, nombre: 'INFORME DE PRODUCCIÓN (NUEVA VERSION)' },
  { id: 2, nombre: 'OEE por Turno Detallado' }
])

const empresaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const reporteSeleccionado = ref(1)
const fechaInicio = ref('2025/11/14 00:00')
const fechaFin = ref('2025/11/19 05:50')

const plantasFiltradas = computed(() => {
  if (!empresaSeleccionada.value) return []
  return plantas.value.filter(p => p.empresaId === parseInt(empresaSeleccionada.value))
})

const puntoHover = ref(null)

const datosGrafico = ref([
  { tiempo: 0, valor: 0, fecha: '14/11/2025 00:00' },
  { tiempo: 10, valor: 0, fecha: '14/11/2025 12:00' },
  { tiempo: 20, valor: 0, fecha: '15/11/2025 00:00' },
  { tiempo: 30, valor: 0, fecha: '15/11/2025 12:00' },
  { tiempo: 40, valor: 0, fecha: '16/11/2025 00:00' },
  { tiempo: 50, valor: 0, fecha: '16/11/2025 12:00' },
  { tiempo: 60, valor: 0, fecha: '17/11/2025 00:00' },
  { tiempo: 70, valor: 0, fecha: '17/11/2025 12:00' },
  { tiempo: 80, valor: 0, fecha: '18/11/2025 00:00' },
  { tiempo: 90, valor: 0, fecha: '18/11/2025 12:00' },
  { tiempo: 100, valor: 0, fecha: '19/11/2025 05:50' }
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
  <div class="oee-turno-view">
    <div class="page-header">
      <h1 class="page-title">OEE POR TURNO</h1>
    </div>

    <!-- Advertencia: Vista en desarrollo -->
    <div class="dev-warning">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
      </svg>
      <div class="dev-warning-content">
        <strong>⚙️ Vista en Construcción</strong>
        <p>
          Esta vista "OEE por Turno" aún no está completamente implementada. 
          Los datos mostrados son estáticos de demostración.
        </p>
        <p style="margin-top: 6px;">
          <strong>Pendiente de desarrollo:</strong>
        </p>
        <ul style="margin: 4px 0 0 20px; font-size: 12px;">
          <li>Integración con API backend real</li>
          <li>Agregación de snapshots por turno con filtro <code>min_interval_s</code></li>
          <li>Filtrado por empresa/planta/línea funcional</li>
          <li>Respeto de configuración de modo (textil 30min / botellas 5min)</li>
        </ul>
      </div>
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
              Historico Embotelladora San Miguel
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
        <h2 class="chart-title">GRÁFICO DE OEE POR TURNO</h2>

        <div class="chart-container">
          <div class="chart-y-axis">
            <div class="y-values">
              <span>1%</span>
              <span>0.9%</span>
              <span>0.8%</span>
              <span>0.7%</span>
              <span>0.6%</span>
              <span>0.5%</span>
              <span>0.4%</span>
              <span>0.3%</span>
              <span>0.2%</span>
              <span>0.1%</span>
              <span>0%</span>
            </div>
          </div>

          <div class="chart-area">
            <svg class="chart-svg" viewBox="0 0 100 100" preserveAspectRatio="none">
              <defs>
                <linearGradient id="areaGradientOee" x1="0%" y1="0%" x2="0%" y2="100%">
                  <stop offset="0%" style="stop-color:#10b981;stop-opacity:0.4" />
                  <stop offset="100%" style="stop-color:#10b981;stop-opacity:0.05" />
                </linearGradient>
                <filter id="shadowOee">
                  <feDropShadow dx="0" dy="0.5" stdDeviation="0.5" flood-color="#10b981" flood-opacity="0.5"/>
                </filter>
              </defs>
              
              <g class="grid-lines">
                <line v-for="i in 11" :key="i" x1="0" :y1="(i-1)*10" x2="100" :y2="(i-1)*10" stroke="#e5e7eb" stroke-width="0.2" />
              </g>
              
              <path v-if="pathDatos" :d="generarPathArea()" fill="url(#areaGradientOee)" />
              
              <path v-if="pathDatos" :d="pathDatos" fill="none" stroke="#10b981" stroke-width="0.8" filter="url(#shadowOee)" />
              
              <g v-for="(punto, index) in datosGrafico" :key="index">
                <circle 
                  :cx="punto.tiempo" 
                  :cy="100 - (punto.valor * 100)"
                  r="1.5"
                  fill="#10b981"
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
            
            <div v-if="puntoHover" class="tooltip" :style="{ left: puntoHover.tiempo + '%', bottom: (puntoHover.valor * 100) + '%' }">
              <div class="tooltip-content">
                <div class="tooltip-fecha">{{ puntoHover.fecha }}</div>
                <div class="tooltip-valor">
                  <span class="tooltip-label">OEE:</span>
                  <span class="tooltip-number">{{ (puntoHover.valor * 100).toFixed(1) }}%</span>
                </div>
              </div>
            </div>

            <div class="x-axis-label">Tiempo</div>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.oee-turno-view {
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
  margin: 0 0 2rem 0;
}

.dev-warning {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  background: #fef3c7;
  border: 2px solid #fbbf24;
  border-radius: 8px;
  color: #92400e;
  margin: 0 1rem;
}

.dev-warning svg {
  flex-shrink: 0;
  margin-top: 2px;
  color: #d97706;
}

.dev-warning-content {
  flex: 1;
}

.dev-warning-content strong {
  font-size: 14px;
  display: block;
  margin-bottom: 6px;
}

.dev-warning-content p {
  font-size: 13px;
  margin: 0;
  line-height: 1.5;
}

.dev-warning-content code {
  background: #fde68a;
  padding: 1px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  font-weight: 600;
}

.dev-warning-content ul {
  list-style: disc;
  padding-left: 20px;
}

.dev-warning-content li {
  margin: 3px 0;
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

.y-values {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 100%;
  min-height: 350px;
  padding-right: 0.5rem;
}

.y-values span {
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
  filter: drop-shadow(0 0 3px rgba(16, 185, 129, 0.8));
}

.x-axis-label {
  position: absolute;
  bottom: -2rem;
  right: 0;
  font-size: 0.875rem;
  color: #374151;
  font-weight: 600;
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
  background: linear-gradient(135deg, #059669 0%, #10b981 100%);
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
  border-top: 6px solid #10b981;
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
  color: #6ee7b7;
}

@media (max-width: 768px) {
  .filtros-section {
    grid-template-columns: 1fr;
  }

  .chart-container {
    flex-direction: column;
  }
}
</style>
