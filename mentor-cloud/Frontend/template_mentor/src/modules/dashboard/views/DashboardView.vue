<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useApi } from '@/shared/composables/useApi'
import { useAuthStore } from '@/stores/auth'
import { oeeService } from '@/api/services/oee.service'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import Card from '@/shared/components/ui/Card.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, PieChart, GaugeChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
} from 'echarts/components'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChart,
  BarChart,
  PieChart,
  GaugeChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
])

const { loading } = useApi()
const authStore = useAuthStore()
const stats = ref(null)
const oeeResumen = ref(null)

// ── Filtros ────────────────────────────────────────────────────────────────
const isAdmin = computed(() => authStore.user?.role === 'ADMIN')
const empresas   = ref([])
const plantas    = ref([])
const lineas     = ref([])
const empresaSel = ref(null)
const plantaSel  = ref(null)
const lineaSel   = ref(null)

// ── Datos para gráficos (poblados desde APIs) ──────────────────────────────
const productionData = ref([])
const oeeData = ref([
  { name: 'Disponibilidad', value: 0 },
  { name: 'Rendimiento',    value: 0 },
  { name: 'Calidad',        value: 0 }
])
const lineasData = ref([])  // OEE por línea (reemplaza plantasData)

// Configuraciones de gráficos
const productionChartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(0, 0, 0, 0.8)',
    borderColor: '#667eea',
    textStyle: { color: '#fff' }
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    top: '10%',
    containLabel: true
  },
  xAxis: {
    type: 'category',
    data: productionData.value.map(d => d.time),
    axisLine: { lineStyle: { color: '#e5e7eb' } },
    axisLabel: { color: '#6b7280' }
  },
  yAxis: {
    type: 'value',
    axisLine: { lineStyle: { color: '#e5e7eb' } },
    axisLabel: { color: '#6b7280' },
    splitLine: { lineStyle: { color: '#f3f4f6' } }
  },
  series: [{
    data: productionData.value.map(d => d.value),
    type: 'line',
    smooth: true,
    lineStyle: { color: '#667eea', width: 3 },
    areaStyle: {
      color: {
        type: 'linear',
        x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(102, 126, 234, 0.3)' },
          { offset: 1, color: 'rgba(102, 126, 234, 0.05)' }
        ]
      }
    },
    itemStyle: { color: '#667eea' }
  }]
}))

const oeeChartOption = computed(() => ({
  tooltip: {
    trigger: 'item',
    backgroundColor: 'rgba(0, 0, 0, 0.8)',
    borderColor: '#667eea',
    textStyle: { color: '#fff' }
  },
  legend: {
    bottom: '5%',
    left: 'center'
  },
  series: [{
    type: 'pie',
    radius: ['40%', '70%'],
    avoidLabelOverlap: false,
    itemStyle: {
      borderRadius: 10,
      borderColor: '#fff',
      borderWidth: 2
    },
    label: {
      show: true,
      formatter: '{b}: {c}%'
    },
    emphasis: {
      label: {
        show: true,
        fontSize: 16,
        fontWeight: 'bold'
      }
    },
    data: oeeData.value.map((item, index) => ({
      ...item,
      itemStyle: {
        color: ['#10b981', '#667eea', '#f59e0b'][index]
      }
    }))
  }]
}))

const plantasChartOption = computed(() => ({ // legacy alias
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(0,0,0,0.8)',
    borderColor: '#667eea',
    textStyle: { color: '#fff' },
    axisPointer: { type: 'shadow' }
  },
  grid: { left: '3%', right: '4%', bottom: '3%', top: '3%', containLabel: true },
  xAxis: {
    type: 'category',
    data: lineasData.value.map(d => d.name),
    axisLine: { lineStyle: { color: '#e5e7eb' } },
    axisLabel: { color: '#6b7280' }
  },
  yAxis: {
    type: 'value',
    min: 0, max: 100,
    axisLine: { lineStyle: { color: '#e5e7eb' } },
    axisLabel: { color: '#6b7280', formatter: '{value}%' },
    splitLine: { lineStyle: { color: '#f3f4f6' } }
  },
  series: [{
    data: lineasData.value.map((d, i) => ({
      value: d.value,
      itemStyle: {
        color: {
          type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: ['#667eea','#10b981','#f59e0b','#ef4444','#06b6d4','#8b5cf6'][i % 6] },
            { offset: 1, color: ['#764ba2','#059669','#d97706','#dc2626','#0891b2','#7c3aed'][i % 6] }
          ]
        }
      }
    })),
    type: 'bar',
    barWidth: '50%',
    label: { show: true, position: 'top', color: '#374151', formatter: '{c}%' }
  }]
}))


const gaugeOption = computed(() => ({
  series: [{
    type: 'gauge',
    startAngle: 180,
    endAngle: 0,
    min: 0,
    max: 100,
    splitNumber: 10,
    axisLine: {
      lineStyle: {
        width: 20,
        color: [
          [0.3, '#ef4444'],
          [0.7, '#f59e0b'],
          [1, '#10b981']
        ]
      }
    },
    pointer: {
      itemStyle: {
        color: '#667eea'
      }
    },
    axisTick: {
      distance: -20,
      length: 5,
      lineStyle: {
        color: '#fff',
        width: 2
      }
    },
    splitLine: {
      distance: -20,
      length: 20,
      lineStyle: {
        color: '#fff',
        width: 4
      }
    },
    axisLabel: {
      color: '#464646',
      distance: 15,
      fontSize: 12
    },
    detail: {
      valueAnimation: true,
      formatter: '{value}%',
      color: '#667eea',
      fontSize: 24,
      offsetCenter: [0, '70%']
    },
    data: [{ value: parseFloat((oeeResumen.value?.oee ?? stats.value?.eficienciaPromedio ?? 0).toFixed(1)) }]
  }]
}))

// ── Carga de catálogos ────────────────────────────────────────────────────
async function cargarEmpresas() {
  try {
    const res = await companyService.getAll()
    empresas.value = res?.data?.data || res?.data || res || []
  } catch { empresas.value = [] }
}

async function cargarPlantas(empresaId) {
  try {
    const res = await plantService.getAll({ empresa_id: empresaId })
    plantas.value = res?.data?.data || res?.data || res || []
  } catch { plantas.value = [] }
}

async function cargarLineas(plantaId) {
  try {
    const res = await lineService.getAll({ planta_id: plantaId })
    lineas.value = res?.data?.data || res?.data || res || []
  } catch { lineas.value = [] }
}

// ── Carga de datos OEE ────────────────────────────────────────────────────
async function cargarOEEData() {
  if (!lineaSel.value && !plantaSel.value) {
    oeeResumen.value = null
    productionData.value = []
    oeeData.value = [
      { name: 'Disponibilidad', value: 0 },
      { name: 'Rendimiento',    value: 0 },
      { name: 'Calidad',        value: 0 }
    ]
    lineasData.value = []
    updateStats()
    return
  }

  try {
    const params = lineaSel.value
      ? { linea_id: lineaSel.value }
      : { planta_id: plantaSel.value }

    const [summaryRes, snapsRes] = await Promise.allSettled([
      oeeService.getSummary(params),
      lineaSel.value ? oeeService.getSnapshots({ linea_id: lineaSel.value, limite: 24 }) : Promise.resolve(null)
    ])

    if (summaryRes.status === 'fulfilled' && summaryRes.value) {
      const s = summaryRes.value
      oeeResumen.value = s
      oeeData.value = [
        { name: 'Disponibilidad', value: parseFloat((s.disponibilidad ?? 0).toFixed(1)) },
        { name: 'Rendimiento',    value: parseFloat((s.rendimiento    ?? 0).toFixed(1)) },
        { name: 'Calidad',        value: parseFloat((s.calidad        ?? 0).toFixed(1)) }
      ]
    }

    if (snapsRes.status === 'fulfilled' && snapsRes.value) {
      const raw = snapsRes.value
      const arr = Array.isArray(raw?.data?.data) ? raw.data.data
               : Array.isArray(raw?.data)        ? raw.data
               : []
      productionData.value = arr.slice().reverse().map(s => ({
        time:  new Date(s.hora || s.timestamp || s.fecha)
                 .toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' }),
        value: parseInt(s.produccion ?? 0)
      }))
    }

    // OEE por línea (para el chart de barras)
    if (!lineaSel.value && lineas.value.length) {
      const latests = await Promise.allSettled(
        lineas.value.map(l => oeeService.getLatest(l.id))
      )
      lineasData.value = lineas.value.map((l, i) => ({
        name:  l.nombre,
        value: latests[i].status === 'fulfilled'
          ? parseFloat((latests[i].value?.oee ?? 0).toFixed(1))
          : 0
      }))
    } else {
      lineasData.value = []
    }
  } catch (e) {
    console.warn('[Dashboard] error OEE:', e)
  }

  updateStats()
}

function updateStats() {
  const oee = parseFloat((oeeResumen.value?.oee ?? 0).toFixed(1))
  stats.value = {
    totalPlantas:       plantas.value.length  || 0,
    totalLineas:        lineas.value.length   || 0,
    totalDispositivos:  lineas.value.filter(l => l.activo !== false).length || 0,
    alarmasActivas:     0,
    eficienciaPromedio: oee,
    produccionHoy:      parseInt(oeeResumen.value?.produccion_total ?? 0)
  }
}

// ── Watchers ──────────────────────────────────────────────────────────────
watch(empresaSel, async (id) => {
  plantaSel.value = null
  lineaSel.value  = null
  plantas.value   = []
  lineas.value    = []
  oeeResumen.value = null
  if (id) await cargarPlantas(id)
  updateStats()
})

watch(plantaSel, async (id) => {
  lineaSel.value = null
  lineas.value   = []
  oeeResumen.value = null
  if (id) {
    await cargarLineas(id)
    await cargarOEEData()
  } else {
    updateStats()
  }
})

watch(lineaSel, async () => {
  await cargarOEEData()
})

onMounted(async () => {
  // Inicializar stats vacío para que el template renderice
  stats.value = { totalPlantas: 0, totalLineas: 0, totalDispositivos: 0, alarmasActivas: 0, eficienciaPromedio: 0, produccionHoy: 0 }

  if (isAdmin.value) {
    await cargarEmpresas()
    // Auto-seleccionar empresa del usuario admin
    if (authStore.user?.empresa_id) {
      empresaSel.value = authStore.user.empresa_id
    }
  } else {
    // No-admin: usar empresa del usuario
    const empresaId = authStore.user?.empresa_id
    if (empresaId) {
      empresaSel.value = empresaId
      await cargarPlantas(empresaId)
    }
    updateStats()
  }
})
</script>

<template>
  <div class="dashboard-view">
    <div class="dashboard-header">
      <h1 class="dashboard-title">Dashboard</h1>
      <p class="dashboard-subtitle">Resumen del sistema de monitoreo</p>
    </div>

    <!-- Filtros de scope -->
    <div class="filtros-bar">
      <!-- Empresa: sólo ADMIN ve el selector; los demás ven el nombre fijo -->
      <div v-if="isAdmin" class="filtro-item">
        <label>Empresa</label>
        <select v-model="empresaSel" class="filtro-select">
          <option :value="null">— Seleccionar —</option>
          <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
        </select>
      </div>
      <div v-else-if="authStore.user?.company" class="filtro-item">
        <label>Empresa</label>
        <span class="filtro-fixed">{{ authStore.user.company }}</span>
      </div>

      <div v-if="empresaSel" class="filtro-item">
        <label>Planta</label>
        <select v-model="plantaSel" class="filtro-select">
          <option :value="null">— Todas las plantas —</option>
          <option v-for="p in plantas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
        </select>
      </div>

      <div v-if="plantaSel" class="filtro-item">
        <label>Línea</label>
        <select v-model="lineaSel" class="filtro-select">
          <option :value="null">— Todas las líneas —</option>
          <option v-for="l in lineas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
        </select>
      </div>
    </div>

    <Loading v-if="loading" />

    <div v-else-if="stats" class="dashboard-content">
      <!-- KPI Cards -->
      <div class="kpi-grid">
        <div class="kpi-card primary">
          <div class="kpi-icon">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z"/>
            </svg>
          </div>
          <div class="kpi-content">
            <div class="kpi-value">{{ stats.totalPlantas }}</div>
            <div class="kpi-label">Total Plantas</div>
            <div class="kpi-sublabel">Plantas activas</div>
          </div>
          <div class="kpi-trend up">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 14l5-5 5 5z"/>
            </svg>
            <span>12%</span>
          </div>
        </div>

        <div class="kpi-card success">
          <div class="kpi-icon">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3 3h18v2H3V3zm0 4h18v2H3V7zm0 4h18v2H3v-2zm0 4h18v2H3v-2zm0 4h18v2H3v-2z"/>
            </svg>
          </div>
          <div class="kpi-content">
            <div class="kpi-value">{{ stats.totalLineas }}</div>
            <div class="kpi-label">Total Líneas</div>
            <div class="kpi-sublabel">Líneas configuradas</div>
          </div>
          <div class="kpi-trend up">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 14l5-5 5 5z"/>
            </svg>
            <span>8%</span>
          </div>
        </div>

        <div class="kpi-card warning">
          <div class="kpi-icon">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 6h-2.18c.11-.31.18-.65.18-1a2.996 2.996 0 00-5.5-1.65l-.5.67-.5-.68C10.96 2.54 10.05 2 9 2 7.34 2 6 3.34 6 5c0 .35.07.69.18 1H4c-1.11 0-1.99.89-1.99 2L2 19c0 1.11.89 2 2 2h16c1.11 0 2-.89 2-2V8c0-1.11-.89-2-2-2zm-5-2c.55 0 1 .45 1 1s-.45 1-1 1-1-.45-1-1 .45-1 1-1zM9 4c.55 0 1 .45 1 1s-.45 1-1 1-1-.45-1-1 .45-1 1-1zm11 15H4v-2h16v2zm0-5H4V8h5.08L7 10.83 8.62 12 12 7.4l3.38 4.6L17 10.83 14.92 8H20v6z"/>
            </svg>
          </div>
          <div class="kpi-content">
            <div class="kpi-value">{{ stats.totalDispositivos }}</div>
            <div class="kpi-label">Dispositivos</div>
            <div class="kpi-sublabel">Dispositivos monitoreados</div>
          </div>
          <div class="kpi-trend up">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7 14l5-5 5 5z"/>
            </svg>
            <span>5%</span>
          </div>
        </div>

        <div class="kpi-card danger">
          <div class="kpi-icon">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
            </svg>
          </div>
          <div class="kpi-content">
            <div class="kpi-value">{{ stats.alarmasActivas }}</div>
            <div class="kpi-label">Alarmas Activas</div>
            <div class="kpi-sublabel">Requieren atención</div>
          </div>
          <div class="kpi-badge">¡Atención!</div>
        </div>
      </div>

      <!-- Efficiency Gauge -->
      <div class="chart-section">
        <Card class="chart-card">
          <template #header>
            <div class="card-header-content">
              <h3 class="card-title">Eficiencia General (OEE)</h3>
              <span class="card-badge">Tiempo Real</span>
            </div>
          </template>
          <div class="gauge-container">
            <VChart :option="gaugeOption" class="chart" />
            <div class="gauge-stats">
              <div class="gauge-stat">
                <span class="gauge-stat-label">Meta</span>
                <span class="gauge-stat-value">85%</span>
              </div>
              <div class="gauge-stat">
                <span class="gauge-stat-label">Actual</span>
                <span class="gauge-stat-value success">{{ stats.eficienciaPromedio }}%</span>
              </div>
              <div class="gauge-stat">
                <span class="gauge-stat-label">Variación</span>
                <span :class="['gauge-stat-value', stats.eficienciaPromedio >= 85 ? 'success' : 'danger']">
                  {{ stats.eficienciaPromedio >= 85 ? '+' : '' }}{{ (stats.eficienciaPromedio - 85).toFixed(1) }}%
                </span>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <!-- Charts Grid -->
      <div class="charts-grid">
        <!-- Production Chart -->
        <Card class="chart-card">
          <template #header>
            <div class="card-header-content">
              <h3 class="card-title">Producción del Día</h3>
              <div class="card-actions">
                <button class="btn-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11z"/>
                  </svg>
                </button>
                <button class="btn-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 8c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm0 2c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm0 6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z"/>
                  </svg>
                </button>
              </div>
            </div>
          </template>
          <VChart :option="productionChartOption" class="chart" />
          <div class="chart-footer">
            <div class="chart-stat">
              <span class="chart-stat-label">Total Hoy</span>
              <span class="chart-stat-value">{{ stats.produccionHoy }} unidades</span>
            </div>
            <div class="chart-stat">
              <span class="chart-stat-label">Promedio/hora</span>
              <span class="chart-stat-value">{{ Math.round(stats.produccionHoy / 24) }}</span>
            </div>
          </div>
        </Card>

        <!-- OEE Breakdown Chart -->
        <Card class="chart-card">
          <template #header>
            <div class="card-header-content">
              <h3 class="card-title">Desglose OEE</h3>
              <span class="card-badge success">Óptimo</span>
            </div>
          </template>
          <VChart :option="oeeChartOption" class="chart" />
          <div class="chart-footer">
            <div class="oee-legend">
              <div class="oee-item">
                <span class="oee-dot" style="background: #10b981;"></span>
                <span>Disponibilidad</span>
              </div>
              <div class="oee-item">
                <span class="oee-dot" style="background: #667eea;"></span>
                <span>Rendimiento</span>
              </div>
              <div class="oee-item">
                <span class="oee-dot" style="background: #f59e0b;"></span>
                <span>Calidad</span>
              </div>
            </div>
          </div>
        </Card>

        <!-- Plants Production Chart -->
        <Card class="chart-card full-width">
          <template #header>
            <div class="card-header-content">
              <h3 class="card-title">OEE por Línea</h3>
              <div class="card-filters">
                <button class="filter-btn active">Hoy</button>
                <button class="filter-btn">Semana</button>
                <button class="filter-btn">Mes</button>
              </div>
            </div>
          </template>
          <VChart :option="plantasChartOption" class="chart" />
        </Card>
      </div>

      <!-- Quick Stats -->
      <div class="quick-stats">
        <div class="stat-item">
          <div class="stat-icon blue">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
            </svg>
          </div>
          <div class="stat-details">
            <span class="stat-value">24/7</span>
            <span class="stat-label">Monitoreo Continuo</span>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon green">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
            </svg>
          </div>
          <div class="stat-details">
            <span class="stat-value">98.5%</span>
            <span class="stat-label">Uptime del Sistema</span>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon orange">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M13 7h-2v4H7v2h4v4h2v-4h4v-2h-4V7zm-1-5C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z"/>
            </svg>
          </div>
          <div class="stat-details">
            <span class="stat-value">156</span>
            <span class="stat-label">Sensores Activos</span>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon purple">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zM9 17H7v-7h2v7zm4 0h-2V7h2v10zm4 0h-2v-4h2v4z"/>
            </svg>
          </div>
          <div class="stat-details">
            <span class="stat-value">1.2M</span>
            <span class="stat-label">Datos Procesados</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-view {
  padding: 1.5rem;
  background: linear-gradient(135deg, #f5f7fa 0%, #e8ecf1 100%);
  min-height: 100vh;
}

.dashboard-header {
  margin-bottom: 2rem;
  text-align: center;
}

.dashboard-title {
  font-size: 2.5rem;
  font-weight: 800;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 0.5rem;
}

.dashboard-subtitle {
  color: #64748b;
  font-size: 1.1rem;
}

/* Barra de filtros */
.filtros-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: flex-end;
  background: white;
  border-radius: 12px;
  padding: 1rem 1.5rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 1px 4px rgba(0,0,0,0.07);
}
.filtro-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 180px;
}
.filtro-item label {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: #64748b;
}
.filtro-select {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 0.45rem 0.75rem;
  font-size: 0.9rem;
  color: #1e293b;
  background: #f8fafc;
  cursor: pointer;
  outline: none;
}
.filtro-select:focus { border-color: #667eea; background: white; }
.filtro-fixed {
  padding: 0.45rem 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: #1e293b;
}

/* KPI Cards Grid */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.kpi-card {
  background: white;
  border-radius: 16px;
  padding: 1.5rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.kpi-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(90deg, var(--card-color-start), var(--card-color-end));
}

.kpi-card.primary {
  --card-color-start: #667eea;
  --card-color-end: #764ba2;
}

.kpi-card.success {
  --card-color-start: #10b981;
  --card-color-end: #059669;
}

.kpi-card.warning {
  --card-color-start: #f59e0b;
  --card-color-end: #d97706;
}

.kpi-card.danger {
  --card-color-start: #ef4444;
  --card-color-end: #dc2626;
}

.kpi-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.1);
}

.kpi-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--card-color-start), var(--card-color-end));
  color: white;
  flex-shrink: 0;
}

.kpi-icon svg {
  width: 32px;
  height: 32px;
}

.kpi-content {
  flex: 1;
}

.kpi-value {
  font-size: 2rem;
  font-weight: 800;
  color: #1e293b;
  line-height: 1;
}

.kpi-label {
  font-size: 0.95rem;
  font-weight: 600;
  color: #475569;
  margin-top: 0.25rem;
}

.kpi-sublabel {
  font-size: 0.75rem;
  color: #94a3b8;
  margin-top: 0.25rem;
}

.kpi-trend {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 600;
}

.kpi-trend.up {
  background: #dcfce7;
  color: #16a34a;
}

.kpi-trend svg {
  width: 16px;
  height: 16px;
}

.kpi-badge {
  padding: 0.375rem 0.875rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 700;
  background: linear-gradient(135deg, #ef4444, #dc2626);
  color: white;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* Chart Section */
.chart-section {
  margin-bottom: 2rem;
}

.chart-card {
  background: white;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
}

.card-header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e2e8f0;
}

.card-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: #1e293b;
}

.card-badge {
  padding: 0.375rem 0.875rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
}

.card-badge.success {
  background: linear-gradient(135deg, #10b981, #059669);
}

.card-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: none;
  background: #f1f5f9;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: #e2e8f0;
  color: #475569;
}

.btn-icon svg {
  width: 18px;
  height: 18px;
}

.gauge-container {
  padding: 2rem;
}

.chart {
  width: 100%;
  height: 300px;
}

.gauge-stats {
  display: flex;
  justify-content: space-around;
  margin-top: 2rem;
  padding-top: 2rem;
  border-top: 1px solid #e2e8f0;
}

.gauge-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.gauge-stat-label {
  font-size: 0.875rem;
  color: #64748b;
  font-weight: 500;
}

.gauge-stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1e293b;
}

.gauge-stat-value.success {
  color: #10b981;
}

/* Charts Grid */
.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 400px), 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.chart-card.full-width {
  grid-column: 1 / -1;
}

.chart-footer {
  padding: 1rem 1.5rem;
  background: #f8fafc;
  border-top: 1px solid #e2e8f0;
}

.chart-stat {
  display: inline-flex;
  flex-direction: column;
  margin-right: 2rem;
}

.chart-stat-label {
  font-size: 0.75rem;
  color: #64748b;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.chart-stat-value {
  font-size: 1.25rem;
  font-weight: 700;
  color: #1e293b;
  margin-top: 0.25rem;
}

.oee-legend {
  display: flex;
  justify-content: center;
  gap: 2rem;
}

.oee-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: #475569;
  font-weight: 500;
}

.oee-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.card-filters {
  display: flex;
  gap: 0.5rem;
}

.filter-btn {
  padding: 0.5rem 1rem;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  background: white;
  color: #64748b;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-btn:hover {
  background: #f1f5f9;
  color: #475569;
}

.filter-btn.active {
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  border-color: transparent;
}

/* Quick Stats */
.quick-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
  gap: 1.5rem;
}

.stat-item {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.stat-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.08);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.stat-icon.blue {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
}

.stat-icon.green {
  background: linear-gradient(135deg, #10b981, #059669);
}

.stat-icon.orange {
  background: linear-gradient(135deg, #f59e0b, #d97706);
}

.stat-icon.purple {
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
}

.stat-icon svg {
  width: 24px;
  height: 24px;
}

.stat-details {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 800;
  color: #1e293b;
  line-height: 1;
}

.stat-label {
  font-size: 0.875rem;
  color: #64748b;
  margin-top: 0.25rem;
}

/* Responsive */
@media (max-width: 1024px) {
  .dashboard-title {
    font-size: 2rem;
  }
}

@media (max-width: 768px) {
  .dashboard-view {
    padding: 1rem;
  }

  .dashboard-title {
    font-size: 1.5rem;
  }

  .kpi-grid {
    grid-template-columns: 1fr;
  }

  .charts-grid {
    grid-template-columns: 1fr;
  }

  .quick-stats {
    grid-template-columns: 1fr;
  }
}
</style>
