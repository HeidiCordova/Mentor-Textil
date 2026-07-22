<script setup>
import { ref, computed, onMounted, markRaw, nextTick } from 'vue'
import { VueFlow, useVueFlow, Panel } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'

import VariableNode from '../components/oee/VariableNode.vue'
import FormulaNode from '../components/oee/FormulaNode.vue'
import KpiNode     from '../components/oee/KpiNode.vue'
import OeeNode     from '../components/oee/OeeNode.vue'

import { canvasOeeService } from '@/api/services/canvas.service'
import { plantService }     from '@/api/services/plant.service'
import { lineService }      from '@/api/services/line.service'
import { deviceService }    from '@/api/services/device.service'

const nodeTypes = markRaw({
  variableNode: markRaw(VariableNode),
  formulaNode:  markRaw(FormulaNode),
  kpiNode:      markRaw(KpiNode),
  oeeNode:      markRaw(OeeNode)
})

const nodes = ref([])
const edges = ref([])

const { fitView, onConnect, addEdges, toObject, setNodes, setEdges } = useVueFlow()

onConnect(conn => addEdges([{ ...conn, animated: true }]))

const scope        = ref('system')
const plantas      = ref([])
const lineas       = ref([])
const dispositivos = ref([])

const plantaId      = ref(null)
const lineaId       = ref(null)
const dispositivoId = ref(null)

const loading      = ref(false)
const saving       = ref(false)
const dirty        = ref(false)
const msgStatus    = ref('')
const canvasNombre = ref('Fórmula OEE')
const canvasId     = ref(null)
async function loadCatalogs() {
  try {
    const pr = await plantService.getAll()
    plantas.value = pr.data ?? pr
  } catch { /* ignored */ }
}

async function onPlantaChange() {
  lineaId.value       = null
  dispositivoId.value = null
  dispositivos.value  = []
  lineas.value        = []
  if (!plantaId.value) return
  try {
    const r = await lineService.getAll({ planta_id: plantaId.value })
    lineas.value = r.data ?? r
  } catch { /* ignored */ }
}

async function onLineaChange(lineaIdVal) {
  dispositivoId.value = null
  dispositivos.value = []
  if (!lineaIdVal) return
  try {
    const r = await deviceService.getAll({ linea_id: lineaIdVal })
    dispositivos.value = r.data ?? r
  } catch { /* ignored */ }
}

async function loadCanvas() {
  loading.value   = true
  dirty.value     = false
  msgStatus.value = ''
  try {
    const params = {}
    if (scope.value === 'planta' && plantaId.value)
      params.planta_id = plantaId.value
    if (scope.value === 'dispositivo' && dispositivoId.value)
      params.dispositivo_id = dispositivoId.value

    const data = await canvasOeeService.get(params)
    applyCanvasData(data)
  } catch (e) {
    console.error('canvas-oee load:', e)
  } finally {
    loading.value = false
    nextTick(() => fitView({ padding: 0.1, includeHiddenNodes: false }))
  }
}

function applyCanvasData(data) {
  const grafo         = data?.grafo ?? data
  canvasId.value      = data?.id ?? null
  canvasNombre.value  = data?.nombre ?? 'Fórmula OEE'
  setNodes(grafo.nodes ?? [])
  setEdges(grafo.edges ?? [])
}


async function saveCanvas() {
  saving.value    = true
  msgStatus.value = ''
  try {
    const flowObj = toObject()
    const body = {
      scope: scope.value,
      nombre: canvasNombre.value,
      grafo: { nodes: flowObj.nodes, edges: flowObj.edges }
    }
    if (scope.value === 'planta' && plantaId.value)
      body.planta_id = parseInt(plantaId.value)
    if (scope.value === 'dispositivo' && dispositivoId.value)
      body.dispositivo_id = parseInt(dispositivoId.value)

    await canvasOeeService.save(body)
    dirty.value = false
    msgStatus.value = 'saved'
    setTimeout(() => { msgStatus.value = '' }, 3000)
  } catch (e) {
    msgStatus.value = 'error'
    console.error('canvas-oee save:', e)
  } finally {
    saving.value = false
  }
}


async function resetDefault() {
  if (!confirm('¿Restablecer este canvas al default OEE estándar? Se perderán los cambios no guardados.')) return
  loading.value = true
  try {
    const body = { scope: scope.value }
    if (scope.value === 'planta' && plantaId.value)
      body.planta_id = parseInt(plantaId.value)
    if (scope.value === 'dispositivo' && dispositivoId.value)
      body.dispositivo_id = parseInt(dispositivoId.value)

    await canvasOeeService.resetDefault(body)
    await loadCanvas()
  } catch (e) {
    console.error('canvas-oee reset:', e)
  } finally {
    loading.value = false
  }
}


async function onScopeChange() {
  plantaId.value      = null
  lineaId.value       = null
  dispositivoId.value = null
  lineas.value        = []
  dispositivos.value  = []
  if (scope.value === 'system') await loadCanvas()
}

function onNodeDragStop() { dirty.value = true }
function onEdgesChange()  { dirty.value = true }
function onNodesChange()  { dirty.value = true }

const scopeDesc = computed(() => {
  if (scope.value === 'system')
    return 'Mostrando el default del sistema. Todos los dispositivos/plantas sin override usan esta fórmula.'
  if (scope.value === 'planta')
    return plantaId.value
      ? 'Override para la planta seleccionada. Sobreescribe el default del sistema para toda la planta.'
      : 'Selecciona una planta para ver o crear su override.'
  if (scope.value === 'dispositivo')
    return dispositivoId.value
      ? 'Override para esta línea. Tiene la máxima prioridad sobre planta y sistema.'
      : 'Selecciona una línea para ver o crear su override.'
  return ''
})

onMounted(async () => {
  await loadCatalogs()
  await loadCanvas()
})
</script>

<template>
  <div class="canvas-oee-page">

    <!-- ── toolbar ── -->
    <div class="cov-toolbar">
      <div class="cov-toolbar-left">
        <h1 class="cov-title">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="6" cy="12" r="3"/><circle cx="18" cy="6" r="3"/><circle cx="18" cy="18" r="3"/>
            <line x1="9" y1="11" x2="15" y2="7"/><line x1="9" y1="13" x2="15" y2="17"/>
          </svg>
          Canvas OEE
        </h1>

        <div class="cov-scope-group">
          <label class="cov-scope-label">Nivel:</label>
          <div class="cov-scope-tabs">
            <button
              v-for="s in [{v:'system',l:'Sistema'},{v:'planta',l:'Planta'},{v:'dispositivo',l:'Línea'}]"
              :key="s.v"
              :class="['cov-scope-tab', { active: scope === s.v }]"
              @click="scope = s.v; onScopeChange()"
            >{{ s.l }}</button>
          </div>
        </div>

        <template v-if="scope !== 'system'">
          <select class="cov-select" v-model="plantaId" @change="onPlantaChange(); if(scope==='planta') loadCanvas()">
            <option :value="null">— Planta —</option>
            <option v-for="p in plantas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
          </select>
        </template>

        <template v-if="scope === 'dispositivo'">
          <select class="cov-select" :disabled="!plantaId" v-model="lineaId" @change="onLineaChange(lineaId)">
            <option :value="null">— Línea —</option>
            <option v-for="l in lineas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
          </select>
          <select class="cov-select" :disabled="!dispositivos.length" v-model="dispositivoId" @change="loadCanvas()">
            <option :value="null">— Dispositivo —</option>
            <option v-for="d in dispositivos" :key="d.id" :value="d.id">{{ d.nombre }}</option>
          </select>
        </template>
      </div>

      <div class="cov-toolbar-right">
        <transition name="fade">
          <span v-if="msgStatus === 'saved'" class="cov-status ok">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
            Guardado
          </span>
          <span v-else-if="msgStatus === 'error'" class="cov-status err">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            Error al guardar
          </span>
          <span v-else-if="dirty" class="cov-status pending">
            <svg width="8" height="8" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="12"/></svg>
            Sin guardar
          </span>
        </transition>

        <button class="cov-btn secondary" @click="resetDefault" :disabled="loading">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/>
          </svg>
          Cargar Default
        </button>

        <button class="cov-btn primary" @click="saveCanvas" :disabled="saving || loading">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/>
          </svg>
          {{ saving ? 'Guardando' : 'Guardar' }}
        </button>
      </div>
    </div>

    <!-- ── scope indicator ── -->
    <div class="cov-scope-desc">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/>
      </svg>
      {{ scopeDesc }}
    </div>

    <div class="cov-canvas-wrap">
      <div v-if="loading" class="cov-loading-overlay">
        <div class="cov-spinner"></div>
        <span>Cargando</span>
      </div>
      <VueFlow
        v-model:nodes="nodes"
        v-model:edges="edges"
        :node-types="nodeTypes"
        fit-view-on-init
        :default-edge-options="{ animated: true, style: { stroke: '#6366f1', strokeWidth: 1.5 } }"
        class="cov-vueflow"
        @node-drag-stop="onNodeDragStop"
        @edges-change="onEdgesChange"
        @nodes-change="onNodesChange"
      >
        <Background pattern-color="#334155" :gap="20" :size="1.5" />
        <Controls />
        <MiniMap
          node-color="#6366f1"
          :style="{ background: '#0f172a', border: '1px solid #334155', borderRadius: '8px' }"
        />


        <Panel position="top-left" class="cov-legend">
          <div class="legend-title">Tipos de nodo</div>
          <div class="legend-item"><span class="leg-dot" style="background:#475569"></span> Variable / Entrada</div>
          <div class="legend-item"><span class="leg-dot" style="background:#6366f1"></span> Cálculo intermedio</div>
          <div class="legend-item"><span class="leg-dot" style="background:#3b82f6"></span> KPI Disponibilidad</div>
          <div class="legend-item"><span class="leg-dot" style="background:#22c55e"></span> KPI Rendimiento</div>
          <div class="legend-item"><span class="leg-dot" style="background:#f59e0b"></span> KPI Calidad</div>
          <div class="legend-item"><span class="leg-dot" style="background:#ef4444"></span> OEE Final</div>
        </Panel>
      </VueFlow>
    </div>

  </div>
</template>

<style scoped>
.canvas-oee-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  overflow: hidden;
}


.cov-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.cov-toolbar-left, .cov-toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.cov-title {
  font-size: 16px;
  font-weight: 700;
  color: #e2e8f0;
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0;
  white-space: nowrap;
}
.cov-scope-group {
  display: flex;
  align-items: center;
  gap: 6px;
}
.cov-scope-label {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
}
.cov-scope-tabs {
  display: flex;
  background: #0f172a;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #334155;
}
.cov-scope-tab {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  color: #64748b;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.15s;
}
.cov-scope-tab.active {
  background: #6366f1;
  color: #fff;
}
.cov-scope-tab:not(.active):hover {
  background: #1e293b;
  color: #94a3b8;
}
.cov-select {
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 6px;
  color: #e2e8f0;
  font-size: 12px;
  padding: 4px 8px;
  cursor: pointer;
}
.cov-select:focus { outline: none; border-color: #6366f1; }

.cov-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.cov-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.cov-btn.primary {
  background: #6366f1;
  color: #fff;
}
.cov-btn.primary:not(:disabled):hover { background: #4f46e5; }
.cov-btn.secondary {
  background: #1e293b;
  border: 1px solid #475569;
  color: #94a3b8;
}
.cov-btn.secondary:not(:disabled):hover { border-color: #6366f1; color: #e2e8f0; }

.cov-status {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 999px;
}
.cov-status.ok      { background: #052e16; color: #4ade80; }
.cov-status.err     { background: #450a0a; color: #f87171; }
.cov-status.pending { background: #1c1003; color: #fbbf24; }


.cov-scope-desc {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  font-size: 11px;
  color: #64748b;
  background: #0f172a;
  border-bottom: 1px solid #1e293b;
  flex-shrink: 0;
}

.cov-loading-overlay {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: #0f172a;
  color: #64748b;
  font-size: 13px;
}
.cov-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #334155;
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }


.cov-canvas-wrap {
  flex: 1;
  min-height: 0;
  position: relative;
}
.cov-vueflow {
  width: 100%;
  height: 100%;
  background: #0f172a;
}


.cov-legend {
  background: #1e293b !important;
  border: 1px solid #334155 !important;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 11px;
}
.legend-title {
  font-size: 10px;
  font-weight: 700;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 6px;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #94a3b8;
  margin-bottom: 3px;
}
.leg-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}


.fade-enter-active, .fade-leave-active { transition: opacity 0.3s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/controls/dist/style.css';
@import '@vue-flow/minimap/dist/style.css';
</style>
