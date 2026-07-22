<template>
  <div class="monitor-oee">
    <div class="titulo-principal">MONITOR OEE</div>

    <!-- Filtros -->
    <div class="filtros-bar">
      <div class="filtro-grupo">
        <label>Empresa</label>
        <select v-model="empresaId" class="campo" @change="onEmpresaChange">
          <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
        </select>
      </div>
      <div class="filtro-grupo">
        <label>Planta</label>
        <select v-model="plantaId" class="campo" @change="onPlantaChange">
          <option :value="null">Todas</option>
          <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
        </select>
      </div>
      <div class="filtro-grupo">
        <label>Linea</label>
        <select v-model="lineaId" class="campo">
          <option :value="null">Todas</option>
          <option v-for="l in lineasFiltradas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
        </select>
      </div>
      <div class="filtro-grupo">
        <label>Desde</label>
        <input type="datetime-local" v-model="fechaDesde" class="campo" />
      </div>
      <div class="filtro-grupo">
        <label>Hasta</label>
        <input type="datetime-local" v-model="fechaHasta" class="campo" />
      </div>
      <button class="btn-buscar" @click="cargarDatos" :disabled="cargando">
        {{ cargando ? 'Cargando...' : 'BUSCAR' }}
      </button>
    </div>

    <!-- KPI principales -->
    <div v-if="resumen" class="kpi-grid">
      <div class="kpi-card kpi-oee" @click="abrirDetalle('oee')">
        <div class="kpi-ring">
          <svg viewBox="0 0 120 120">
            <circle cx="60" cy="60" r="52" fill="none" stroke="#1e293b" stroke-width="10" opacity="0.15"/>
            <circle cx="60" cy="60" r="52" fill="none"
              :stroke="colorKPI(resumen.oee)" stroke-width="10"
              :stroke-dasharray="`${(resumen.oee / 100) * 326.7} 326.7`"
              transform="rotate(-90 60 60)" stroke-linecap="round"/>
          </svg>
          <span class="kpi-valor" :style="{ color: colorKPI(resumen.oee) }">
            {{ resumen.oee.toFixed(1) }}%
          </span>
        </div>
        <span class="kpi-label">OEE</span>
        <span class="kpi-estado" :class="claseKPI(resumen.oee)">{{ estadoKPI(resumen.oee) }}</span>
      </div>

      <div class="kpi-card" @click="abrirDetalle('disponibilidad')">
        <div class="kpi-barra-container">
          <div class="kpi-barra" :style="{ width: resumen.disponibilidad + '%', background: '#3b82f6' }"></div>
        </div>
        <span class="kpi-valor-inline" style="color:#3b82f6">{{ resumen.disponibilidad.toFixed(1) }}%</span>
        <span class="kpi-label">Disponibilidad</span>
        <span class="kpi-sub">T.Operativo / T.Disponible</span>
      </div>

      <div class="kpi-card" @click="abrirDetalle('rendimiento')">
        <div class="kpi-barra-container">
          <div class="kpi-barra" :style="{ width: resumen.rendimiento + '%', background: '#8b5cf6' }"></div>
        </div>
        <span class="kpi-valor-inline" style="color:#8b5cf6">{{ resumen.rendimiento.toFixed(1) }}%</span>
        <span class="kpi-label">Rendimiento</span>
        <span class="kpi-sub">Prod.Nominal / T.Operativo</span>
      </div>

      <div class="kpi-card" @click="abrirDetalle('calidad')">
        <div class="kpi-barra-container">
          <div class="kpi-barra" :style="{ width: resumen.calidad + '%', background: '#10b981' }"></div>
        </div>
        <span class="kpi-valor-inline" style="color:#10b981">{{ resumen.calidad.toFixed(1) }}%</span>
        <span class="kpi-label">Calidad</span>
        <span class="kpi-sub">(Conteo - Merma) / Conteo</span>
      </div>

      <div class="kpi-card kpi-produccion">
        <span class="kpi-valor-grande">{{ resumen.produccion_total.toLocaleString() }}</span>
        <span class="kpi-label">Produccion Total</span>
        <span class="kpi-sub">{{ resumen.snapshots }} snapshots</span>
      </div>
    </div>

    <!-- Grafica temporal -->
    <div v-if="snapshots.length > 0" class="grafico-section">
      <div class="grafico-header">
        <h3>Tendencia OEE</h3>
        <div class="grafico-toggles">
          <label v-for="s in series" :key="s.key" class="toggle-serie" :style="{ color: s.color }">
            <input type="checkbox" v-model="s.visible" /> {{ s.label }}
          </label>
        </div>
      </div>
      <div class="grafico-canvas">
        <svg :viewBox="`0 0 ${anchoGrafico} 320`" preserveAspectRatio="none">
          <g v-for="i in 5" :key="'g'+i">
            <line :x1="60" :y1="30+(i-1)*65" :x2="anchoGrafico-20" :y2="30+(i-1)*65" stroke="#e5e7eb" stroke-width="1"/>
            <text :x="55" :y="35+(i-1)*65" text-anchor="end" fill="#9ca3af" font-size="11">{{ 100-(i-1)*25 }}%</text>
          </g>
          <line x1="60" y1="290" :x2="anchoGrafico-20" y2="290" stroke="#9ca3af" stroke-width="1"/>
          <template v-for="s in seriesVisibles" :key="s.key">
            <path :d="buildPath(s.key)" :stroke="s.color" stroke-width="2" fill="none" stroke-linejoin="round"/>
          </template>
          <g v-for="(snap, idx) in snapshotsOrdenados" :key="'l'+idx">
            <text v-if="idx % labelStep === 0"
              :x="xPos(idx)" y="308" text-anchor="middle" fill="#9ca3af" font-size="10">
              {{ formatHora(snap.hora) }}
            </text>
          </g>
        </svg>
      </div>
    </div>

    <!-- Tabla detalle -->
    <div v-if="snapshots.length > 0" class="tabla-section">
      <table class="tabla-oee">
        <thead>
          <tr>
            <th>Hora</th>
            <th>OEE</th>
            <th>Disponibilidad</th>
            <th>Rendimiento</th>
            <th>Calidad</th>
            <th>Produccion</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in snapshotsOrdenados" :key="s.id">
            <td>{{ new Date(s.hora).toLocaleString() }}</td>
            <td><span class="badge" :class="claseKPI(s.oee)">{{ s.oee.toFixed(1) }}%</span></td>
            <td>{{ s.disponibilidad.toFixed(1) }}%</td>
            <td>{{ s.rendimiento.toFixed(1) }}%</td>
            <td>{{ s.calidad.toFixed(1) }}%</td>
            <td>{{ s.produccion }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal drill-down -->
    <Teleport to="body">
      <div v-if="detalleVisible" class="modal-overlay" @click.self="detalleVisible = false">
        <div class="modal-detalle">
          <div class="modal-header">
            <h3>{{ detalleMetrica.toUpperCase() }}</h3>
            <button @click="detalleVisible = false" class="btn-cerrar">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 6L6 18M6 6l12 12"/>
              </svg>
            </button>
          </div>
          <div class="modal-body">
            <div class="detalle-formula">
              <template v-if="detalleMetrica === 'oee'">
                <p><strong>OEE</strong> = Disponibilidad x Rendimiento x Calidad</p>
                <p>{{ resumen.disponibilidad.toFixed(1) }}% x {{ resumen.rendimiento.toFixed(1) }}% x {{ resumen.calidad.toFixed(1) }}% = <strong>{{ resumen.oee.toFixed(1) }}%</strong></p>
              </template>
              <template v-if="detalleMetrica === 'disponibilidad'">
                <p><strong>Disponibilidad</strong> = T.Operativo / T.Disponible</p>
                <p>T.Disponible = T.Turno - Parada Obligatoria</p>
                <p>T.Operativo = T.Disponible - Parada No Obligatoria</p>
              </template>
              <template v-if="detalleMetrica === 'rendimiento'">
                <p><strong>Rendimiento</strong> = T.Nominal.Prod / T.Operativo</p>
                <p>T.Nominal.Prod = Conteo / Vel.Nominal</p>
                <p>Perdida velocidad = T.Neto.Prod - T.Nominal.Prod</p>
              </template>
              <template v-if="detalleMetrica === 'calidad'">
                <p><strong>Calidad</strong> = (Conteo - Merma) / Conteo</p>
              </template>
            </div>
            <div class="detalle-grafico">
              <svg :viewBox="`0 0 ${anchoGrafico} 200`" preserveAspectRatio="none">
                <g v-for="i in 5" :key="'dg'+i">
                  <line :x1="60" :y1="20+(i-1)*40" :x2="anchoGrafico-20" :y2="20+(i-1)*40" stroke="#e5e7eb" stroke-width="1"/>
                  <text :x="55" :y="25+(i-1)*40" text-anchor="end" fill="#9ca3af" font-size="10">{{ 100-(i-1)*25 }}%</text>
                </g>
                <path :d="buildPathModal(detalleMetrica)" :stroke="colorMetrica(detalleMetrica)" stroke-width="2.5" fill="none"/>
              </svg>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Sin datos -->
    <div v-if="!cargando && !resumen && buscado" class="sin-datos">
      No se encontraron snapshots OEE para los filtros seleccionados.
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { oeeService } from '@/api/services/oee.service'
import client from '@/api/client'

const empresas = ref([])
const plantas = ref([])
const lineas = ref([])
const empresaId = ref(null)
const plantaId = ref(null)
const lineaId = ref(null)

const now = new Date()
const hoy = now.toISOString().slice(0, 10)
const fechaDesde = ref(hoy + 'T00:00')
const fechaHasta = ref(hoy + 'T23:59')

const cargando = ref(false)
const buscado = ref(false)
const resumen = ref(null)
const snapshots = ref([])
const detalleVisible = ref(false)
const detalleMetrica = ref('oee')

const series = reactive([
  { key: 'oee', label: 'OEE', color: '#f59e0b', visible: true },
  { key: 'disponibilidad', label: 'Disponibilidad', color: '#3b82f6', visible: true },
  { key: 'rendimiento', label: 'Rendimiento', color: '#8b5cf6', visible: true },
  { key: 'calidad', label: 'Calidad', color: '#10b981', visible: true }
])

const seriesVisibles = computed(() => series.filter(s => s.visible))
const anchoGrafico = computed(() => Math.max(800, snapshotsOrdenados.value.length * 12 + 100))
const labelStep = computed(() => Math.max(1, Math.floor(snapshotsOrdenados.value.length / 12)))

const snapshotsOrdenados = computed(() =>
  [...snapshots.value].sort((a, b) => new Date(a.hora) - new Date(b.hora))
)

const plantasFiltradas = computed(() =>
  plantas.value.filter(p => p.empresa_id === empresaId.value)
)

const lineasFiltradas = computed(() => {
  if (plantaId.value) return lineas.value.filter(l => l.planta_id === plantaId.value)
  const pids = plantasFiltradas.value.map(p => p.id)
  return lineas.value.filter(l => pids.includes(l.planta_id))
})

function onEmpresaChange() {
  plantaId.value = null
  lineaId.value = null
}
function onPlantaChange() {
  lineaId.value = null
}

async function cargarMaestros() {
  try {
    const [empRes, plaRes, linRes] = await Promise.all([
      client.get('/empresas'),
      client.get('/plantas'),
      client.get('/lineas')
    ])
    empresas.value = empRes.data || empRes || []
    plantas.value = plaRes.data || plaRes || []
    lineas.value = linRes.data || linRes || []
    if (empresas.value.length > 0 && !empresaId.value) {
      empresaId.value = empresas.value[0].id
    }
  } catch (e) {
    console.error('Error cargando maestros:', e)
  }
}

async function cargarDatos() {
  cargando.value = true
  buscado.value = true
  try {
    const params = {}
    if (empresaId.value) params.empresa_id = empresaId.value
    if (plantaId.value) params.planta_id = plantaId.value
    if (lineaId.value) params.linea_id = lineaId.value
    if (fechaDesde.value) params.from = new Date(fechaDesde.value).toISOString()
    if (fechaHasta.value) params.to = new Date(fechaHasta.value).toISOString()
    params.limit = 1000

    const [snapRes, sumRes] = await Promise.all([
      oeeService.getSnapshots(params),
      oeeService.getSummary(params)
    ])

    snapshots.value = snapRes.data || []
    resumen.value = sumRes
  } catch (e) {
    console.error('Error cargando OEE:', e)
    snapshots.value = []
    resumen.value = null
  } finally {
    cargando.value = false
  }
}

function xPos(idx) {
  const total = snapshotsOrdenados.value.length
  if (total <= 1) return 60
  return 60 + (idx / (total - 1)) * (anchoGrafico.value - 80)
}

function yPos(val) {
  return 290 - (val / 100) * 260
}

function buildPath(key) {
  const pts = snapshotsOrdenados.value
  if (pts.length === 0) return ''
  return 'M ' + pts.map((s, i) => `${xPos(i)},${yPos(s[key])}`).join(' L ')
}

function buildPathModal(key) {
  const pts = snapshotsOrdenados.value
  if (pts.length === 0) return ''
  return 'M ' + pts.map((s, i) => {
    const x = 60 + (i / Math.max(pts.length - 1, 1)) * (anchoGrafico.value - 80)
    const y = 180 - (s[key] / 100) * 160
    return `${x},${y}`
  }).join(' L ')
}

function formatHora(h) {
  const d = new Date(h)
  return d.getHours().toString().padStart(2, '0') + ':' + d.getMinutes().toString().padStart(2, '0')
}

function colorKPI(v) {
  if (v >= 85) return '#22c55e'
  if (v >= 60) return '#f59e0b'
  return '#ef4444'
}

function claseKPI(v) {
  if (v >= 85) return 'excelente'
  if (v >= 60) return 'aceptable'
  return 'critico'
}

function estadoKPI(v) {
  if (v >= 85) return 'EXCELENTE'
  if (v >= 60) return 'ACEPTABLE'
  return 'CRITICO'
}

function colorMetrica(key) {
  const map = { oee: '#f59e0b', disponibilidad: '#3b82f6', rendimiento: '#8b5cf6', calidad: '#10b981' }
  return map[key] || '#6b7280'
}

function abrirDetalle(metrica) {
  detalleMetrica.value = metrica
  detalleVisible.value = true
}

onMounted(() => {
  cargarMaestros()
})
</script>

<style scoped>
.monitor-oee {
  padding: 1.5rem;
  max-width: 1500px;
  margin: 0 auto;
}

.titulo-principal {
  background: #001f54;
  color: #fff;
  padding: 1rem 1.5rem;
  font-size: 1.25rem;
  font-weight: 700;
  border-radius: 6px;
  margin-bottom: 1.5rem;
  letter-spacing: 0.5px;
}

.filtros-bar {
  display: flex;
  gap: 1rem;
  align-items: flex-end;
  flex-wrap: wrap;
  margin-bottom: 1.5rem;
  background: #fff;
  padding: 1rem;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}

.filtro-grupo {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.filtro-grupo label {
  font-size: 0.75rem;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
}

.campo {
  padding: 0.5rem 0.75rem;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.875rem;
  min-width: 140px;
  background: #fff;
}

.btn-buscar {
  padding: 0.5rem 1.5rem;
  background: #001f54;
  color: #fff;
  border: none;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
  font-size: 0.875rem;
  white-space: nowrap;
}

.btn-buscar:hover { background: #002d7a; }
.btn-buscar:disabled { opacity: 0.6; cursor: not-allowed; }

/* KPI Grid */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.kpi-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  transition: box-shadow 0.2s, transform 0.15s;
}

.kpi-card:hover {
  box-shadow: 0 4px 16px rgba(0,0,0,0.1);
  transform: translateY(-2px);
}

.kpi-oee {
  border-left: 4px solid #f59e0b;
}

.kpi-ring { position: relative; width: 100px; height: 100px; }
.kpi-ring svg { width: 100%; height: 100%; }
.kpi-ring .kpi-valor {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  font-weight: 700;
}

.kpi-label { font-weight: 600; color: #334155; font-size: 0.875rem; }
.kpi-sub { font-size: 0.7rem; color: #94a3b8; text-align: center; }

.kpi-valor-inline { font-size: 1.75rem; font-weight: 700; }
.kpi-valor-grande { font-size: 2rem; font-weight: 700; color: #334155; }

.kpi-barra-container {
  width: 100%;
  height: 8px;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}

.kpi-barra {
  height: 100%;
  border-radius: 4px;
  transition: width 0.6s ease;
}

.kpi-estado {
  font-size: 0.7rem;
  font-weight: 700;
  padding: 0.15rem 0.5rem;
  border-radius: 3px;
}

.excelente { background: #dcfce7; color: #166534; }
.aceptable { background: #fef3c7; color: #92400e; }
.critico { background: #fee2e2; color: #991b1b; }

.kpi-produccion { border-left: 4px solid #334155; cursor: default; }

/* Grafico */
.grafico-section {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
  margin-bottom: 1.5rem;
}

.grafico-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.grafico-header h3 { font-size: 1rem; font-weight: 600; color: #334155; margin: 0; }

.grafico-toggles { display: flex; gap: 1rem; }
.toggle-serie { font-size: 0.8rem; font-weight: 600; display: flex; align-items: center; gap: 0.3rem; cursor: pointer; }
.toggle-serie input { accent-color: currentColor; }

.grafico-canvas { overflow-x: auto; }
.grafico-canvas svg { width: 100%; height: 320px; display: block; }

/* Tabla */
.tabla-section {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
}

.tabla-oee {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.tabla-oee th {
  background: #001f54;
  color: #fff;
  padding: 0.75rem 1rem;
  text-align: left;
  font-weight: 600;
}

.tabla-oee td {
  padding: 0.6rem 1rem;
  border-bottom: 1px solid #f1f5f9;
}

.tabla-oee tbody tr:hover { background: #f8fafc; }

.badge {
  padding: 0.15rem 0.5rem;
  border-radius: 3px;
  font-weight: 600;
  font-size: 0.8rem;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-detalle {
  background: #fff;
  border-radius: 8px;
  width: 90%;
  max-width: 800px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid #e2e8f0;
}

.modal-header h3 { margin: 0; font-size: 1.1rem; color: #001f54; }

.btn-cerrar {
  background: none;
  border: none;
  cursor: pointer;
  color: #64748b;
}

.modal-body { padding: 1.25rem; }

.detalle-formula {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 1rem;
  margin-bottom: 1rem;
}

.detalle-formula p { margin: 0.3rem 0; font-size: 0.9rem; color: #334155; }

.detalle-grafico svg { width: 100%; height: 200px; }

.sin-datos {
  text-align: center;
  padding: 3rem;
  color: #94a3b8;
  font-size: 0.95rem;
}

@media (max-width: 1024px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 640px) {
  .kpi-grid { grid-template-columns: 1fr; }
  .filtros-bar { flex-direction: column; }
}
</style>
