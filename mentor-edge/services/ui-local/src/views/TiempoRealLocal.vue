<template>
  <div class="dashboard-tv-fullscreen">

    <!-- Modal Selección de Línea -->
    <Teleport to="body">
      <div v-if="!lineaSeleccionada" class="modal-seleccion-fullscreen">
        <div class="modal-contenido">
          <div v-if="cargandoLineas" class="modal-loading">
            <svg class="spin" fill="none" viewBox="0 0 24 24" width="40" height="40">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="#3b82f6" stroke-width="4"/>
              <path fill="#3b82f6" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            <p>Cargando líneas...</p>
          </div>
          <template v-else>
            <h2>Seleccionar Línea</h2>
            <p v-if="nombreDispositivo" class="subtitle-modal">{{ nombreDispositivo }}</p>
            <div v-if="lineas.length" class="lineas-grid">
              <button
                v-for="l in lineas"
                :key="l.id"
                @click="seleccionarLinea(l.id)"
                class="btn-linea">
                {{ l.nombre }}
              </button>
            </div>
            <p v-else class="no-lineas">No hay líneas configuradas en este dispositivo.</p>
          </template>
        </div>
      </div>
    </Teleport>

    <!-- Dashboard Principal -->
    <div v-if="lineaSeleccionada" class="dashboard-content">

      <!-- Header -->
      <header class="header-tv">
        <div class="header-left">
          <h1>{{ nombreDispositivo }}</h1>
          <p>Monitoreo local</p>
          <select class="select-linea" :value="lineaSeleccionada" @change="cambiarLinea(Number($event.target.value))">
            <option v-for="l in lineas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
          </select>
        </div>
        <div class="header-center">
          <div class="fecha">{{ fechaActual }}</div>
          <div class="hora">{{ horaActual }}</div>
        </div>
        <div class="header-right">
          <div class="turno">Turno: <span>{{ turnoActual }}</span></div>
          <div class="live-indicator">
            <span class="dot"></span>
            EN VIVO
          </div>
        </div>
      </header>

      <!-- Grid Principal -->
      <main v-if="datosLinea" class="grid-dashboard">

        <!-- Fila 1 -->
        <section class="fila-principal">

          <!-- OEE Circular -->
          <div class="oee-hero">
            <svg viewBox="0 0 300 300" class="oee-svg">
              <defs>
                <linearGradient id="oeeGradLocal" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" :stop-color="getColorOEE(datosLinea.oee)" stop-opacity="0.8"/>
                  <stop offset="100%" :stop-color="getColorOEE(datosLinea.oee)"/>
                </linearGradient>
              </defs>
              <circle cx="150" cy="150" r="130" fill="none" stroke="#1e293b" stroke-width="12" opacity="0.15"/>
              <circle
                cx="150" cy="150" r="130"
                fill="none"
                stroke="url(#oeeGradLocal)"
                stroke-width="12"
                :stroke-dasharray="`${(datosLinea.oee / 100) * 816.81} 816.81`"
                transform="rotate(-90 150 150)"
                stroke-linecap="round"
                class="oee-circle"/>
            </svg>
            <div class="oee-content">
              <div class="oee-valor">{{ datosLinea.oee.toFixed(1) }}<span>%</span></div>
              <div class="oee-label">OEE</div>
              <div :class="['oee-badge', getClaseOEE(datosLinea.oee)]">{{ getEstadoOEE(datosLinea.oee) }}</div>
            </div>
          </div>

          <!-- Componentes OEE -->
          <div class="componentes-grid">
            <div class="componente disponibilidad">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
              </svg>
              <div class="comp-info">
                <div class="comp-label">DISPONIBILIDAD</div>
                <div class="comp-valor">{{ datosLinea.disponibilidad.toFixed(1) }}%</div>
                <div class="comp-barra">
                  <div class="barra-fill" :style="{ width: datosLinea.disponibilidad + '%', background: '#3b82f6' }"></div>
                </div>
              </div>
            </div>
            <div class="componente rendimiento">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <path d="M13 2L3 14h8l-1 8 10-12h-8l1-8z"/>
              </svg>
              <div class="comp-info">
                <div class="comp-label">RENDIMIENTO</div>
                <div class="comp-valor">{{ datosLinea.rendimiento.toFixed(1) }}%</div>
                <div class="comp-barra">
                  <div class="barra-fill" :style="{ width: datosLinea.rendimiento + '%', background: '#8b5cf6' }"></div>
                </div>
              </div>
            </div>
            <div class="componente calidad">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <div class="comp-info">
                <div class="comp-label">CALIDAD</div>
                <div class="comp-valor">{{ datosLinea.calidad.toFixed(1) }}%</div>
                <div class="comp-barra">
                  <div class="barra-fill" :style="{ width: datosLinea.calidad + '%', background: '#10b981' }"></div>
                </div>
              </div>
            </div>
            <div class="componente velocidad">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
              </svg>
              <div class="comp-info">
                <div class="comp-label">VELOCIDAD ACTUAL</div>
                <div class="comp-valor">{{ datosLinea.velocidadActual }}</div>
                <div class="comp-meta">Teórica: {{ datosLinea.velocidadTeorica }}</div>
              </div>
            </div>
            <div class="componente defectos">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
              <div class="comp-info">
                <div class="comp-label">DEFECTOS</div>
                <div class="comp-valor">{{ datosLinea.defectosTurno }}</div>
                <div class="comp-meta">Tasa: {{ datosLinea.produccionTurno > 0 ? ((datosLinea.defectosTurno / datosLinea.produccionTurno) * 100).toFixed(2) : '0.00' }}%</div>
              </div>
            </div>
            <div class="componente paradas">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <circle cx="12" cy="12" r="10"/><rect x="9" y="9" width="6" height="6" fill="currentColor"/>
              </svg>
              <div class="comp-info">
                <div class="comp-label">PARADAS</div>
                <div class="comp-valor">{{ datosLinea.paradasTurno }}</div>
                <div class="comp-meta">Tiempo: {{ datosLinea.tiempoParadas }}</div>
              </div>
            </div>
          </div>

          <!-- Estado y Producto -->
          <div class="info-panel">
            <div class="estado-card">
              <div :class="['estado-badge', estadoEfectivo.toLowerCase().replace(/\s/g,'-')]">
                <span class="pulse-dot"></span>
                {{ estadoEfectivo.toUpperCase() }}
              </div>
              <div class="tiempo-estado">{{ datosLinea.tiempoEstado }}</div>
              <div v-if="datosLinea.motivoParada && estadoEfectivo !== 'Produciendo'" class="motivo">
                {{ datosLinea.motivoParada }}
              </div>
            </div>
            <div class="producto-card">
              <div class="producto-label">PRODUCTO ACTUAL</div>
              <div class="producto-nombre">{{ datosLinea.productoActual }}</div>
              <div class="producto-sku">{{ datosLinea.skuActual }}</div>
            </div>
          </div>
        </section>

        <!-- Fila Gráfico + Cámara -->
        <section class="fila-grafico" @mouseleave="hoverIdx = -1">

          <!-- Mitad izquierda: gráfico OEE -->
          <div class="grafico-half">
            <div class="grafico-canvas" ref="canvasRef" @mousemove="onGraficoHover" @mouseleave="hoverIdx = -1">
              <svg :viewBox="`0 0 ${SVG_W} ${SVG_H}`" preserveAspectRatio="none" class="grafico-svg">
                <g class="grid">
                  <line v-for="i in 5" :key="'h'+i"
                    :x1="PAD_L" :y1="PAD_T + (i-1) * STEP_Y" :x2="SVG_W - PAD_R" :y2="PAD_T + (i-1) * STEP_Y"
                    stroke="#334155" stroke-width="1" opacity="0.25"/>
                </g>
                <g class="labels-y">
                  <text v-for="(val, i) in [100, 75, 50, 25, 0]" :key="'y'+i"
                    :x="PAD_L - 12" :y="PAD_T + i * STEP_Y + 5"
                    text-anchor="end" fill="#94a3b8" font-size="13" font-weight="600">{{ val }}%</text>
                </g>
                <g class="labels-x">
                  <template v-for="(punto, idx) in datosTendencia" :key="'x'+idx">
                    <text v-if="xLabelVisible(idx)"
                      :x="ptX(idx)" :y="SVG_H - 8"
                      text-anchor="middle" fill="#94a3b8" font-size="12" font-weight="500">{{ punto.tiempo }}</text>
                  </template>
                </g>
                <path :d="areaPath('oee')"            fill="url(#fillOee)"/>
                <path :d="areaPath('disponibilidad')" fill="url(#fillDisp)"/>
                <path :d="areaPath('rendimiento')"    fill="url(#fillRend)"/>
                <path :d="areaPath('calidad')"        fill="url(#fillCal)"/>
                <defs>
                  <linearGradient id="fillOee"  x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#3b82f6" stop-opacity="0.25"/>
                    <stop offset="100%" stop-color="#3b82f6" stop-opacity="0.02"/>
                  </linearGradient>
                  <linearGradient id="fillDisp" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#f97316" stop-opacity="0.18"/>
                    <stop offset="100%" stop-color="#f97316" stop-opacity="0.01"/>
                  </linearGradient>
                  <linearGradient id="fillRend" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#22c55e" stop-opacity="0.18"/>
                    <stop offset="100%" stop-color="#22c55e" stop-opacity="0.01"/>
                  </linearGradient>
                  <linearGradient id="fillCal"  x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#ef4444" stop-opacity="0.12"/>
                    <stop offset="100%" stop-color="#ef4444" stop-opacity="0.01"/>
                  </linearGradient>
                </defs>
                <path :d="generarPathMetrica('calidad')"       stroke="#ef4444" stroke-width="2.5" fill="none" stroke-linejoin="round" stroke-linecap="round"/>
                <path :d="generarPathMetrica('disponibilidad')" stroke="#f97316" stroke-width="2.5" fill="none" stroke-linejoin="round" stroke-linecap="round"/>
                <path :d="generarPathMetrica('rendimiento')"    stroke="#22c55e" stroke-width="2.5" fill="none" stroke-linejoin="round" stroke-linecap="round"/>
                <path :d="generarPathMetrica('oee')"            stroke="#3b82f6" stroke-width="3"   fill="none" stroke-linejoin="round" stroke-linecap="round"/>
                <template v-for="(p, idx) in datosTendencia" :key="'dots'+idx">
                  <circle :cx="ptX(idx)" :cy="ptY(p.oee)"            r="3" fill="#3b82f6" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                  <circle :cx="ptX(idx)" :cy="ptY(p.disponibilidad)" r="3" fill="#f97316" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                  <circle :cx="ptX(idx)" :cy="ptY(p.rendimiento)"    r="3" fill="#22c55e" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                  <circle :cx="ptX(idx)" :cy="ptY(p.calidad)"        r="3" fill="#ef4444" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                </template>
                <line v-if="hoverIdx >= 0"
                  :x1="ptX(hoverIdx)" :y1="PAD_T" :x2="ptX(hoverIdx)" :y2="PAD_T + CHART_H"
                  stroke="#cbd5e1" stroke-width="1" stroke-dasharray="4,3" opacity="0.5"/>
                <template v-if="hoverIdx >= 0 && datosTendencia[hoverIdx]">
                  <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].oee)"            r="6" fill="#3b82f6" stroke="#0f172a" stroke-width="2"/>
                  <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].disponibilidad)" r="6" fill="#f97316" stroke="#0f172a" stroke-width="2"/>
                  <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].rendimiento)"    r="6" fill="#22c55e" stroke="#0f172a" stroke-width="2"/>
                  <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].calidad)"        r="6" fill="#ef4444" stroke="#0f172a" stroke-width="2"/>
                </template>
              </svg>
              <div v-if="hoverIdx >= 0 && datosTendencia[hoverIdx]" class="chart-tooltip"
                :style="{ left: tooltipLeft + 'px' }">
                <div class="ct-time">{{ datosTendencia[hoverIdx].tiempo }}</div>
                <div class="ct-row"><span class="ct-dot" style="background:#ef4444"></span> Calidad <span class="ct-val">{{ datosTendencia[hoverIdx].calidad.toFixed(1) }} %</span></div>
                <div class="ct-row"><span class="ct-dot" style="background:#f97316"></span> Disponibilidad <span class="ct-val">{{ datosTendencia[hoverIdx].disponibilidad.toFixed(1) }} %</span></div>
                <div class="ct-row"><span class="ct-dot" style="background:#22c55e"></span> Rendimiento <span class="ct-val">{{ datosTendencia[hoverIdx].rendimiento.toFixed(1) }} %</span></div>
                <div class="ct-row"><span class="ct-dot" style="background:#3b82f6"></span> OEE <span class="ct-val">{{ datosTendencia[hoverIdx].oee.toFixed(1) }} %</span></div>
              </div>
            </div>
            <div class="grafico-leyenda">
              <div class="leyenda-item"><span class="linea-color" style="background:#3b82f6"></span><span class="ley-dot" style="background:#3b82f6"></span> OEE: {{ datosLinea.oee.toFixed(1) }}%</div>
              <div class="leyenda-item"><span class="linea-color" style="background:#f97316"></span><span class="ley-dot" style="background:#f97316"></span> Disponibilidad: {{ datosLinea.disponibilidad.toFixed(1) }}%</div>
              <div class="leyenda-item"><span class="linea-color" style="background:#22c55e"></span><span class="ley-dot" style="background:#22c55e"></span> Rendimiento: {{ datosLinea.rendimiento.toFixed(1) }}%</div>
              <div class="leyenda-item"><span class="linea-color" style="background:#ef4444"></span><span class="ley-dot" style="background:#ef4444"></span> Calidad: {{ datosLinea.calidad.toFixed(1) }}%</div>
            </div>
          </div>

          <!-- Mitad derecha: cámara -->
          <div class="camera-half">
            <div v-if="!noCameraConfigured" class="camera-container" ref="cameraContainer">
              <canvas ref="streamCanvas" class="camera-feed"></canvas>
              <canvas ref="roiCanvas" class="camera-roi-canvas"></canvas>
              <div v-if="cameraStreaming" class="camera-live-badge">
                <span class="camera-live-dot"></span>
                EN VIVO
              </div>
              <div v-if="!cameraStreaming && !cameraOffline" class="camera-connecting">
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="32" height="32" opacity="0.5">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                    d="M15 10l4.553-2.069A1 1 0 0121 8.82v6.36a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z"/>
                </svg>
                <p>Conectando...</p>
              </div>
              <div v-if="cameraOffline" class="camera-offline">
                <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4">
                  <path d="M23 7l-7 5 7 5V7z" stroke-opacity="0.5"/>
                  <rect x="1" y="5" width="15" height="14" rx="2" ry="2" stroke-opacity="0.5"/>
                  <line x1="1" y1="1" x2="23" y2="23" stroke="#ef4444" stroke-width="2"/>
                </svg>
                <p>Sin señal de cámara</p>
                <span>Reintentando conexión...</span>
              </div>
            </div>
            <div v-else class="camera-placeholder">
              <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4">
                <path d="M23 7l-7 5 7 5V7z" stroke-opacity="0.5"/>
                <rect x="1" y="5" width="15" height="14" rx="2" ry="2" stroke-opacity="0.5"/>
                <line x1="1" y1="1" x2="23" y2="23" stroke="#64748b" stroke-width="2"/>
              </svg>
              <p>Sin cámara configurada</p>
              <span>Configúrela en la sección Dispositivos</span>
            </div>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { configService } from '../services/api'

// ── Estado general ────────────────────────────────────────────────────────────
const lineas             = ref([])
const cargandoLineas     = ref(true)
const lineaSeleccionada  = ref(null)
const nombreDispositivo  = ref('Mentor Edge')
const datosLinea         = ref(null)
const datosTendencia     = ref([])
const horaActual         = ref('')
const fechaActual        = ref('')
const turnoActual        = ref('')

// ── Cámara ────────────────────────────────────────────────────────────────────
const streamCanvas     = ref(null)
const roiCanvas        = ref(null)
const cameraContainer  = ref(null)
const cameraStreaming    = ref(false)
const noCameraConfigured = ref(false)
const cameraRetryCount   = ref(0)
const cameraOffline      = computed(() => !cameraStreaming.value && cameraRetryCount.value >= 2)

// ── Análisis de presencia ─────────────────────────────────────────────────────
const presenceMotion   = ref(false)
let   _offscreenCanvas = null
let   _prevPresencePx  = null

// ── Config de línea activa ────────────────────────────────────────────────────
const roi          = ref(null)
const roiPresencia = ref(null)
const cameraUrl    = ref('')

// ── Internals ─────────────────────────────────────────────────────────────────
let _streamAbort     = null
let _streamRetry     = null
let _refreshTimeout  = null
let _relojInterval   = null
let _statusInterval  = null
let _resizeObs       = null
const REFRESH_MS     = 60_000

// ── SVG chart ─────────────────────────────────────────────────────────────────
const SVG_W   = 1600
const SVG_H   = 420
const PAD_L   = 70
const PAD_R   = 30
const PAD_T   = 30
const PAD_B   = 45
const CHART_H = SVG_H - PAD_T - PAD_B
const STEP_Y  = CHART_H / 4
const canvasRef   = ref(null)
const hoverIdx    = ref(-1)
const tooltipLeft = ref(0)

// ── Computed ──────────────────────────────────────────────────────────────────
const estadoEfectivo = computed(() => {
  if (!datosLinea.value) return '—'
  const d = datosLinea.value
  if (d.estado === 'Parada' || d.estado === 'Microparada') return d.estado
  if (roiPresencia.value) return presenceMotion.value ? 'Produciendo' : 'Detenida'
  return d.estado
})

// ── Helpers OEE ───────────────────────────────────────────────────────────────
function getColorOEE(v) { return v >= 85 ? '#22c55e' : v >= 70 ? '#f59e0b' : '#ef4444' }
function getClaseOEE(v) { return v >= 85 ? 'excelente' : v >= 70 ? 'aceptable' : 'critico' }
function getEstadoOEE(v) { return v >= 85 ? 'EXCELENTE' : v >= 70 ? 'ACEPTABLE' : 'CRÍTICO' }
function formatDuracionMin(min) {
  if (!min || isNaN(min)) return '—'
  const h = Math.floor(min / 60), m = Math.round(min % 60)
  return h > 0 ? `${h}h ${m}min` : `${m}min`
}
function turnoNombreActual(turnos) {
  if (!turnos?.length) return '—'
  const now = new Date(), hh = now.getHours() * 60 + now.getMinutes()
  const toMin = t => { if (!t) return 0; const [h, m] = t.split(':').map(Number); return h * 60 + m }
  for (const t of turnos) {
    const ini = toMin(t.hora_inicio || t.inicio), fin = toMin(t.hora_fin || t.fin)
    if (ini < fin ? (hh >= ini && hh < fin) : (hh >= ini || hh < fin))
      return t.nombre || t.name || `Turno ${t.id}`
  }
  return turnos[0]?.nombre || '—'
}

// ── Carga de líneas ───────────────────────────────────────────────────────────
async function cargarLineas() {
  cargandoLineas.value = true
  try {
    const res = await configService.getLines()
    const ids = Array.isArray(res) ? res : (res?.lines || [])
    const configs = await Promise.allSettled(
      ids.map(id => configService.getConfig(id).then(c => ({
        id,
        nombre: c.oee?.line_name || c.line_name || c.nombre || ('Línea ' + id),
        config: c
      })))
    )
    lineas.value = configs.filter(r => r.status === 'fulfilled').map(r => r.value)
    if (lineas.value.length === 1) seleccionarLinea(lineas.value[0].id)
  } catch { /* non-fatal */ } finally {
    cargandoLineas.value = false
  }
}

async function seleccionarLinea(id) {
  _pararStream()
  _pararStatusPolling()
  presenceMotion.value = false
  _prevPresencePx = null
  cameraStreaming.value = false
  lineaSeleccionada.value = id
  await _cargarConfigLinea(id)
  await _iniciarMonitoreo()
  _iniciarStream()
  _iniciarStatusPolling()
}

async function cambiarLinea(id) {
  if (id === lineaSeleccionada.value) return
  _pararStream()
  _pararStatusPolling()
  if (_refreshTimeout) { clearTimeout(_refreshTimeout); _refreshTimeout = null }
  datosLinea.value = null
  datosTendencia.value = []
  presenceMotion.value = false
  _prevPresencePx = null
  cameraStreaming.value = false
  lineaSeleccionada.value = id
  await _cargarConfigLinea(id)
  await _iniciarMonitoreo()
  _iniciarStream()
  _iniciarStatusPolling()
}

// ── Config de línea ───────────────────────────────────────────────────────────
async function _cargarConfigLinea(id) {
  try {
    const cfg = await configService.getConfig(id)
    cameraUrl.value = cfg.camera?.url || cfg.camera_url || ''
    roi.value       = cfg.roi || null
    noCameraConfigured.value = !cameraUrl.value
    const rp = cfg.roi_presencia
    if (rp && typeof rp === 'object') {
      roiPresencia.value = Array.isArray(rp)
        ? (rp.length >= 4 ? rp : null)
        : (rp.width > 0 ? [rp.x, rp.y, rp.width, rp.height] : null)
    } else {
      roiPresencia.value = null
    }
  } catch { /* non-fatal */ }
}

// ── Monitoreo (polling OEE) ───────────────────────────────────────────────────
async function _iniciarMonitoreo() {
  if (_refreshTimeout) { clearTimeout(_refreshTimeout); _refreshTimeout = null }
  await _cargarDatosSafe()
  _programarSiguiente()
}

function _programarSiguiente() {
  if (_refreshTimeout) clearTimeout(_refreshTimeout)
  _refreshTimeout = setTimeout(async () => {
    await _cargarDatosSafe()
    _programarSiguiente()
  }, REFRESH_MS)
}

async function _cargarDatosSafe() {
  try { await _cargarDatos() } catch { /* non-fatal */ }
}

async function _cargarDatos() {
  const lid = lineaSeleccionada.value
  if (!lid) return

  const [stopsRes, turnoRes, velRes, runsRes] = await Promise.allSettled([
    fetch(`/api/gateway/edge/stops?linea_id=${lid}`).then(r => r.json()),
    fetch(`/api/gateway/edge/current-turno?linea_id=${lid}`).then(r => r.json()),
    fetch(`/api/gateway/edge/catalogs/velocidad-nominal?linea_id=${lid}`).then(r => r.json()),
    fetch(`/api/gateway/edge/production-runs?linea_id=${lid}&limit=1`).then(r => r.json()),
  ])

  const paradas = stopsRes.status === 'fulfilled'
    ? (Array.isArray(stopsRes.value?.data) ? stopsRes.value.data
      : Array.isArray(stopsRes.value) ? stopsRes.value : [])
    : []

  const turnoData = turnoRes.status === 'fulfilled' ? turnoRes.value : null
  if (Array.isArray(turnoData)) turnoActual.value = turnoNombreActual(turnoData)
  else turnoActual.value = turnoData?.nombre || turnoData?.name || '—'

  const velArr = velRes.status === 'fulfilled'
    ? (Array.isArray(velRes.value?.data) ? velRes.value.data
      : Array.isArray(velRes.value) ? velRes.value : [])
    : []
  const velNominal = velArr[0]?.velocidad_nominal ?? null
  const velocidadTeorica = velNominal ? `${velNominal} u/min` : '—'

  const runsArr = runsRes.status === 'fulfilled'
    ? (Array.isArray(runsRes.value?.data) ? runsRes.value.data
      : Array.isArray(runsRes.value) ? runsRes.value : [])
    : []
  const run = runsArr[0] || null

  const paradaActiva = paradas.find(s => s.activo || s.activa || (!s.fin && !s.ended_at))
  const totalMinParadas = paradas.reduce((acc, s) => {
    const finTs = s.fin || s.ended_at
    if (!finTs) return acc
    return acc + (new Date(finTs) - new Date(s.inicio || s.started_at || s.created_at)) / 60000
  }, 0)

  let estado = 'Produciendo', motivoParada = null, tiempoEstado = '—'
  if (paradaActiva) {
    const tipo = (paradaActiva.tipo || paradaActiva.tipo_parada || paradaActiva.stop_type || '').toUpperCase()
    estado       = tipo.includes('MICRO') ? 'Microparada' : 'Parada'
    motivoParada = paradaActiva.categoria_nombre || paradaActiva.categoria || paradaActiva.motivo || null
    const ini    = new Date(paradaActiva.inicio || paradaActiva.started_at || paradaActiva.created_at)
    tiempoEstado = formatDuracionMin((Date.now() - ini.getTime()) / 60000)
  }

  const produccionTurno = run?.total_produced ?? run?.produccion ?? 0
  const metaTurno = velNominal ? Math.round(velNominal * 480) : 0
  const dispMin   = Math.max(0, 480 - totalMinParadas)
  const disponibilidad = Math.min((dispMin / 480) * 100, 100)
  const rendimiento    = metaTurno > 0 ? Math.min((produccionTurno / metaTurno) * 100, 100) : 0
  const calidad        = produccionTurno > 0 ? 100 : 100
  const oee            = parseFloat(((disponibilidad * rendimiento * calidad) / 10000).toFixed(1))

  datosLinea.value = {
    estado,
    tiempoEstado,
    motivoParada,
    oee,
    disponibilidad: parseFloat(disponibilidad.toFixed(1)),
    rendimiento:    parseFloat(rendimiento.toFixed(1)),
    calidad:        parseFloat(calidad.toFixed(1)),
    produccionTurno,
    velocidadActual: '—',
    velocidadTeorica,
    defectosTurno:   0,
    paradasTurno:    paradas.length,
    tiempoParadas:   formatDuracionMin(totalMinParadas),
    productoActual:  run?.producto_nombre || run?.run_nombre || '—',
    skuActual:       run?.sku || run?.run_sku || '—',
  }

  // Agregar snapshot a tendencia (máx 96 puntos)
  const snap = {
    tiempo:         new Date().toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' }),
    oee,
    disponibilidad: parseFloat(disponibilidad.toFixed(1)),
    rendimiento:    parseFloat(rendimiento.toFixed(1)),
    calidad:        parseFloat(calidad.toFixed(1)),
  }
  datosTendencia.value = [...datosTendencia.value.slice(-95), snap]

  // Enriquecer con velocidad desde status
  try {
    const st = await fetch(`/api/gateway/edge/status?linea_id=${lid}`).then(r => r.json())
    if (datosLinea.value && st.flow_rate != null)
      datosLinea.value.velocidadActual = st.flow_rate.toFixed(1) + ' u/min'
  } catch { /* non-fatal */ }
}

// ── Stream MJPEG (fetch loop) ─────────────────────────────────────────────────
function _iniciarStream() {
  if (_streamAbort || !cameraUrl.value) return
  _streamAbort = new AbortController()
  _runStream(_streamAbort.signal)
}

function _pararStream() {
  _streamAbort?.abort(); _streamAbort = null
  clearTimeout(_streamRetry); _streamRetry = null
  cameraStreaming.value = false
  cameraRetryCount.value = 0
}

async function _runStream(signal) {
  try {
    const resp = await fetch('/api/gateway/edge/camera/stream', { signal })
    if (!resp.ok) {
      if (resp.status === 400) { noCameraConfigured.value = true; return }
      throw new Error(`HTTP ${resp.status}`)
    }
    const reader = resp.body.getReader()
    let buf = new Uint8Array(512 * 1024), len = 0, rendering = false

    const find = (b1, b2, from = 0) => {
      for (let i = from; i < len - 1; i++) if (buf[i] === b1 && buf[i + 1] === b2) return i
      return -1
    }
    const consume = s => {
      if (s > 0 && s < len) { buf.copyWithin(0, s, len); len -= s }
      else if (s >= len) len = 0
    }

    let pending = null
    const scheduleRender = () => {
      if (rendering || !pending || signal.aborted) return
      rendering = true
      const blob = pending; pending = null
      requestAnimationFrame(async () => {
        try {
          const bm = await createImageBitmap(blob)
          if (!signal.aborted) {
            const sc = streamCanvas.value
            if (sc) {
              if (sc.width !== bm.width || sc.height !== bm.height) {
                sc.width = bm.width; sc.height = bm.height
                if (!cameraStreaming.value) {
                  cameraStreaming.value = true
                  cameraRetryCount.value = 0
                  nextTick(_dibujarROI)
                }
              }
              sc.getContext('2d').drawImage(bm, 0, 0)
            }
            _analizarFrame(bm)
          }
          bm.close()
        } catch { /* ignorar frames corruptos */ }
        rendering = false
        if (pending) scheduleRender()
      })
    }

    while (!signal.aborted) {
      const { done, value } = await reader.read()
      if (done) break
      if (len + value.length > buf.length) {
        const nb = new Uint8Array(Math.max(buf.length * 2, len + value.length))
        nb.set(buf.subarray(0, len)); buf = nb
      }
      buf.set(value, len); len += value.length
      let last = null
      while (len > 4) {
        const si = find(0xFF, 0xD8)
        if (si < 0) { len = len > 1 ? 1 : len; break }
        if (si > 0) consume(si)
        const ei = find(0xFF, 0xD9, 2)
        if (ei < 0) break
        last = buf.slice(0, ei + 2); consume(ei + 2)
      }
      if (last) { pending = new Blob([last], { type: 'image/jpeg' }); scheduleRender() }
      if (len > 1024 * 1024) { buf[0] = buf[len - 1]; len = 1 }
    }
  } catch (e) {
    if (!signal.aborted) {
      cameraStreaming.value = false
      cameraRetryCount.value++
      _streamRetry = setTimeout(_iniciarStream, 4000)
    }
  }
}

// ── Análisis de presencia ─────────────────────────────────────────────────────
function _analizarFrame(bitmap) {
  if (!roiPresencia.value) return
  const iw = bitmap.width, ih = bitmap.height
  if (iw < 4 || ih < 4) return
  if (!_offscreenCanvas) _offscreenCanvas = document.createElement('canvas')
  if (_offscreenCanvas.width !== iw || _offscreenCanvas.height !== ih) {
    _offscreenCanvas.width = iw; _offscreenCanvas.height = ih
  }
  const ctx = _offscreenCanvas.getContext('2d', { willReadFrequently: true })
  ctx.drawImage(bitmap, 0, 0)

  const scaleX = iw / 1280, scaleY = ih / 720
  const [px, py, pw, ph] = roiPresencia.value
  const cpx = Math.round(Math.max(0, px * scaleX))
  const cpy = Math.round(Math.max(0, py * scaleY))
  const cpw = Math.round(Math.min(pw * scaleX, iw - cpx))
  const cph = Math.round(Math.min(ph * scaleY, ih - cpy))
  if (cpw < 4 || cph < 4) return

  const { data } = ctx.getImageData(cpx, cpy, cpw, cph)
  const n = cpw * cph
  const gray = new Uint8Array(n)
  for (let i = 0; i < n; i++) gray[i] = (data[i * 4] + data[i * 4 + 1] + data[i * 4 + 2]) / 3
  if (_prevPresencePx && _prevPresencePx.length === n) {
    let sum = 0
    for (let i = 0; i < n; i++) sum += Math.abs(gray[i] - _prevPresencePx[i])
    const prev = presenceMotion.value
    presenceMotion.value = (sum / n) > 8.0
    if (prev !== presenceMotion.value) _dibujarROI()
  }
  _prevPresencePx = gray
}

// ── ROI overlay canvas ────────────────────────────────────────────────────────
function _dibujarROI() {
  const sc     = streamCanvas.value
  const canvas = roiCanvas.value
  const cont   = cameraContainer.value
  if (!sc || !canvas || !cont) return
  if (!roi.value && !roiPresencia.value) { canvas.width = 0; canvas.height = 0; return }

  const rect  = sc.getBoundingClientRect()
  const cRect = cont.getBoundingClientRect()
  canvas.style.left = (rect.left - cRect.left) + 'px'
  canvas.style.top  = (rect.top  - cRect.top)  + 'px'
  const w = rect.width, h = rect.height
  canvas.width = w; canvas.height = h

  const ctx  = canvas.getContext('2d')
  const natW = sc.width  || 1280
  const natH = sc.height || 720
  const sx   = w / natW, sy = h / natH
  ctx.clearRect(0, 0, w, h)

  if (roi.value && roi.value.length >= 4) {
    const [rx, ry, rw, rh] = roi.value
    const x = rx * sx, y = ry * sy, bw = rw * sx, bh = rh * sy
    ctx.strokeStyle = 'rgba(52,211,153,0.9)'; ctx.lineWidth = 2
    ctx.setLineDash([6, 3]); ctx.strokeRect(x, y, bw, bh); ctx.setLineDash([])
    const cL = Math.min(12, bw * 0.15, bh * 0.3)
    ctx.lineWidth = 3; ctx.strokeStyle = 'rgba(52,211,153,1)'
    ctx.beginPath(); ctx.moveTo(x, y + cL); ctx.lineTo(x, y); ctx.lineTo(x + cL, y); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(x + bw - cL, y); ctx.lineTo(x + bw, y); ctx.lineTo(x + bw, y + cL); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(x, y + bh - cL); ctx.lineTo(x, y + bh); ctx.lineTo(x + cL, y + bh); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(x + bw - cL, y + bh); ctx.lineTo(x + bw, y + bh); ctx.lineTo(x + bw, y + bh - cL); ctx.stroke()
    ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillRect(x, y - 18 < 0 ? y + 2 : y - 18, 30, 16)
    ctx.fillStyle = 'rgba(52,211,153,1)'; ctx.font = '11px monospace'
    ctx.fillText('ROI', x + 4, y - 18 < 0 ? y + 13 : y - 5)
  }

  if (roiPresencia.value) {
    const [px, py, pw, ph] = roiPresencia.value
    const x = px * sx, y = py * sy, bw = pw * sx, bh = ph * sy
    const producing = presenceMotion.value
    const pColor    = producing ? 'rgba(52,211,153,0.9)'  : 'rgba(248,113,113,0.85)'
    const pFill     = producing ? 'rgba(52,211,153,0.08)' : 'rgba(248,113,113,0.05)'
    const pLabel    = producing ? 'En movimiento'         : 'Detenida'
    const pLabelW   = producing ? 90 : 58
    ctx.fillStyle = pFill; ctx.fillRect(x, y, bw, bh)
    ctx.strokeStyle = pColor; ctx.lineWidth = producing ? 2 : 1.5
    ctx.setLineDash([5, 4]); ctx.strokeRect(x, y, bw, bh); ctx.setLineDash([])
    const ly = y - 18 < 0 ? y + 2 : y - 18
    ctx.fillStyle = producing ? 'rgba(6,78,59,0.85)' : 'rgba(69,10,10,0.85)'
    ctx.fillRect(x, ly, pLabelW, 16)
    ctx.fillStyle = pColor; ctx.font = 'bold 11px monospace'
    ctx.fillText(pLabel, x + 4, ly < y ? ly + 11 : ly + 13)
  }
}

// ── Polling detector status ───────────────────────────────────────────────────
function _iniciarStatusPolling() {
  _pollDetectorStatus()
  _statusInterval = setInterval(_pollDetectorStatus, 500)
}
function _pararStatusPolling() {
  clearInterval(_statusInterval); _statusInterval = null
}
async function _pollDetectorStatus() {
  try {
    const lid = lineaSeleccionada.value
    const r   = await fetch(`/api/detector/status${lid ? '?linea_id=' + lid : ''}`,
                            { signal: AbortSignal.timeout(1000) })
    if (!r.ok) return
    const d = await r.json()
    const prev = presenceMotion.value
    presenceMotion.value = !!d.presence_motion
    if (prev !== presenceMotion.value) _dibujarROI()
  } catch { /* timeout — mantener estado anterior */ }
}

// ── SVG chart helpers ─────────────────────────────────────────────────────────
function ptX(idx) {
  const n = datosTendencia.value.length
  if (n <= 1) return PAD_L
  return PAD_L + idx * ((SVG_W - PAD_L - PAD_R) / (n - 1))
}
function ptY(val) { return PAD_T + CHART_H - (val / 100) * CHART_H }
function xLabelVisible(idx) {
  const n = datosTendencia.value.length
  if (n <= 10) return true
  return idx % Math.max(1, Math.floor(n / 8)) === 0
}
function generarPathMetrica(metrica) {
  if (!datosTendencia.value.length) return ''
  return 'M ' + datosTendencia.value.map((p, i) => `${ptX(i)},${ptY(p[metrica])}`).join(' L ')
}
function areaPath(metrica) {
  if (!datosTendencia.value.length) return ''
  const n = datosTendencia.value.length, base = PAD_T + CHART_H
  const line = datosTendencia.value.map((p, i) => `${ptX(i)},${ptY(p[metrica])}`).join(' L ')
  return `M ${ptX(0)},${base} L ${line} L ${ptX(n - 1)},${base} Z`
}
function onGraficoHover(e) {
  if (!canvasRef.value || !datosTendencia.value.length) return
  const rect = canvasRef.value.getBoundingClientRect()
  const svgX = ((e.clientX - rect.left) / rect.width) * SVG_W
  const n    = datosTendencia.value.length
  const gap  = (SVG_W - PAD_L - PAD_R) / Math.max(n - 1, 1)
  const best = Math.max(0, Math.min(n - 1, Math.round((svgX - PAD_L) / gap)))
  hoverIdx.value    = best
  tooltipLeft.value = Math.max(10, Math.min(rect.width - 200, (ptX(best) / SVG_W) * rect.width - 90))
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────
function actualizarFechaHora() {
  const now = new Date()
  horaActual.value = now.toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  fechaActual.value = now.toLocaleDateString('es-ES', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })
}

onMounted(async () => {
  actualizarFechaHora()
  _relojInterval = setInterval(actualizarFechaHora, 1000)
  await cargarLineas()
  _resizeObs = new ResizeObserver(() => _dibujarROI())
})

watch(cameraContainer, el => {
  if (_resizeObs) _resizeObs.disconnect()
  if (el) _resizeObs.observe(el)
})
watch(presenceMotion, () => _dibujarROI())

onUnmounted(() => {
  _pararStream()
  _pararStatusPolling()
  if (_refreshTimeout) clearTimeout(_refreshTimeout)
  clearInterval(_relojInterval)
  _resizeObs?.disconnect()
})
</script>

<style scoped>
.dashboard-tv-fullscreen {
  position: fixed;
  top: 0; left: 0;
  width: 100vw; height: 100vh;
  background: linear-gradient(135deg, #0a0e27 0%, #0f172a 50%, #1e293b 100%);
  overflow: auto;
  z-index: 9998;
}

.modal-seleccion-fullscreen {
  position: fixed;
  top: 0; left: 0;
  width: 100vw; height: 100vh;
  background: rgba(0, 0, 0, 0.92);
  display: flex; align-items: center; justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(12px);
}

.modal-contenido {
  background: white;
  border-radius: 20px;
  padding: 3rem;
  max-width: 800px; width: 90%;
  box-shadow: 0 30px 60px rgba(0, 0, 0, 0.6);
}

.modal-contenido h2 {
  text-align: center; font-size: 2rem; color: #001f54;
  font-weight: 800; margin-bottom: 2rem;
}

.subtitle-modal {
  text-align: center; font-size: 1.25rem; color: #64748b;
  margin: -1rem 0 2rem 0; font-weight: 600;
}

.no-lineas {
  text-align: center; color: #64748b; font-size: 1rem;
}

.modal-loading {
  display: flex; flex-direction: column; align-items: center; gap: 1rem; padding: 2rem 0;
}
.modal-loading p { color: #64748b; font-size: 1rem; font-weight: 600; }
.spin { animation: spin .8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.lineas-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1.5rem;
}

.btn-linea {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white; border: none;
  padding: 2rem; border-radius: 14px;
  font-size: 1.35rem; font-weight: 700;
  cursor: pointer; transition: all 0.3s;
  box-shadow: 0 6px 12px rgba(59, 130, 246, 0.3);
  text-transform: uppercase;
}

.btn-linea:hover {
  transform: translateY(-6px) scale(1.02);
  box-shadow: 0 12px 24px rgba(59, 130, 246, 0.4);
}

.select-linea {
  margin-top: 0.4rem;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.4);
  color: #93c5fd; padding: 0.35rem 0.75rem;
  border-radius: 8px; font-size: 0.95rem; font-weight: 700;
  cursor: pointer; outline: none; text-transform: uppercase;
}
.select-linea option { background: #0f172a; color: #f1f5f9; }

.dashboard-content { padding: 1.5rem; }

.header-tv {
  background: rgba(15, 23, 42, 0.8);
  border-radius: 12px; padding: 1rem 1.5rem; margin-bottom: 1rem;
  display: grid; grid-template-columns: 1fr auto 1fr;
  align-items: center; gap: 1.5rem;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.header-left h1 {
  font-size: 1.25rem; font-weight: 800; color: #f1f5f9;
  text-transform: uppercase; margin: 0 0 0.3rem 0;
}
.header-left p { font-size: 0.95rem; color: #94a3b8; margin: 0; }
.header-center { text-align: center; }

.fecha { font-size: 0.85rem; color: #cbd5e1; text-transform: capitalize; }
.hora  { font-size: 1.75rem; font-weight: 800; color: #f1f5f9; font-family: 'Courier New', monospace; letter-spacing: 2px; }

.header-right { text-align: right; }
.turno { font-size: 1rem; color: #94a3b8; margin-bottom: 0.3rem; }
.turno span { font-weight: 800; color: #3b82f6; font-size: 1.35rem; }

.live-indicator {
  display: inline-flex; align-items: center; gap: 0.75rem;
  color: #10b981; font-weight: 700;
}
.live-indicator .dot {
  width: 12px; height: 12px; background: #10b981;
  border-radius: 50%; animation: pulse 2s infinite;
}
@keyframes pulse { 0%,100% { opacity:1; transform:scale(1); } 50% { opacity:.5; transform:scale(1.2); } }

.grid-dashboard { display: flex; flex-direction: column; gap: 0.75rem; }

.fila-principal { display: grid; grid-template-columns: 280px 1fr 280px; gap: 0.75rem; }

.oee-hero {
  background: linear-gradient(135deg, rgba(30,41,59,0.8), rgba(15,23,42,0.9));
  border-radius: 12px; padding: 0.75rem; position: relative;
  box-shadow: 0 12px 40px rgba(0,0,0,0.4); border: 1px solid rgba(255,255,255,0.05);
  display: flex; align-items: center; justify-content: center;
}
.oee-svg { width: 100%; height: auto; }
.oee-circle { transition: stroke-dasharray 1s ease; filter: drop-shadow(0 0 10px currentColor); }
.oee-content { position: absolute; text-align: center; }
.oee-valor { font-size: 2.75rem; font-weight: 900; color: #f1f5f9; line-height: 1; }
.oee-valor span { font-size: 1.5rem; opacity: 0.8; }
.oee-label { font-size: 0.85rem; font-weight: 700; color: #94a3b8; margin: 0.2rem 0 0.4rem; letter-spacing: 1.5px; }
.oee-badge { padding: 0.4rem 1rem; border-radius: 6px; font-size: 0.8rem; font-weight: 800; letter-spacing: 1px; }
.oee-badge.excelente { background: rgba(34,197,94,.2);  color: #22c55e; }
.oee-badge.aceptable { background: rgba(245,158,11,.2); color: #f59e0b; }
.oee-badge.critico   { background: rgba(239,68,68,.2);  color: #ef4444; }

.componentes-grid { display: grid; grid-template-columns: repeat(3,1fr); gap: 0.75rem; }

.componente {
  background: linear-gradient(135deg, rgba(30,41,59,.6), rgba(15,23,42,.8));
  border-radius: 10px; padding: 0.75rem; border: 1px solid rgba(255,255,255,.05);
  box-shadow: 0 8px 24px rgba(0,0,0,.3);
  display: flex; flex-direction: column; align-items: center; gap: 0.5rem;
}
.componente.disponibilidad { color: #3b82f6; }
.componente.rendimiento    { color: #8b5cf6; }
.componente.calidad        { color: #10b981; }
.componente.velocidad      { color: #3b82f6; }
.componente.defectos       { color: #f59e0b; }
.componente.paradas        { color: #ef4444; }
.comp-info { text-align: center; width: 100%; }
.comp-label { font-size: 0.7rem; font-weight: 700; letter-spacing: 0.5px; color: #cbd5e1; margin-bottom: 0.25rem; }
.comp-valor { font-size: 1.5rem; font-weight: 900; color: #f1f5f9; margin-bottom: 0.5rem; }
.comp-meta  { font-size: 0.75rem; color: #64748b; font-weight: 600; }
.comp-barra { width: 100%; height: 10px; background: rgba(15,23,42,.8); border-radius: 5px; overflow: hidden; }
.barra-fill { height: 100%; border-radius: 5px; transition: width 1s ease; box-shadow: 0 0 12px currentColor; }

.info-panel { display: flex; flex-direction: column; gap: 0.75rem; }

.estado-card {
  background: linear-gradient(135deg, rgba(30,41,59,.7), rgba(15,23,42,.9));
  border-radius: 10px; padding: 0.75rem; text-align: center;
  border: 1px solid rgba(255,255,255,.05); box-shadow: 0 8px 24px rgba(0,0,0,.3);
}

.estado-badge {
  display: inline-flex; align-items: center; gap: 0.5rem;
  padding: 0.5rem 1rem; border-radius: 8px; font-size: 1rem;
  font-weight: 800; letter-spacing: 0.5px; margin-bottom: 0.75rem;
}
.estado-badge.produciendo { background: linear-gradient(135deg,rgba(34,197,94,.2),rgba(34,197,94,.1)); color: #22c55e; border: 2px solid rgba(34,197,94,.4); }
.estado-badge.detenida    { background: linear-gradient(135deg,rgba(156,163,175,.2),rgba(156,163,175,.1)); color: #9ca3af; border: 2px solid rgba(156,163,175,.4); }
.estado-badge.parada      { background: linear-gradient(135deg,rgba(239,68,68,.2),rgba(239,68,68,.1)); color: #ef4444; border: 2px solid rgba(239,68,68,.4); }
.estado-badge.microparada { background: linear-gradient(135deg,rgba(245,158,11,.2),rgba(245,158,11,.1)); color: #f59e0b; border: 2px solid rgba(245,158,11,.4); }

.pulse-dot { width: 10px; height: 10px; border-radius: 50%; background: currentColor; animation: pulse 2s infinite; }
.tiempo-estado { font-size: 1.1rem; color: #cbd5e1; font-weight: 600; }
.motivo { margin-top: 1rem; padding: 0.75rem; background: rgba(239,68,68,.1); border-radius: 8px; color: #fca5a5; font-weight: 600; }

.producto-card {
  background: linear-gradient(135deg, rgba(30,41,59,.7), rgba(15,23,42,.9));
  border-radius: 16px; padding: 1.75rem; border: 1px solid rgba(255,255,255,.05);
  box-shadow: 0 8px 24px rgba(0,0,0,.3);
}
.producto-label  { font-size: 0.85rem; color: #64748b; font-weight: 700; letter-spacing: 1px; margin-bottom: 0.75rem; }
.producto-nombre { font-size: 1.15rem; font-weight: 700; color: #f1f5f9; margin-bottom: 0.5rem; }
.producto-sku    { font-size: 0.95rem; color: #94a3b8; font-family: 'Courier New', monospace; }

.fila-grafico {
  background: linear-gradient(135deg, rgba(30,41,59,.7), rgba(15,23,42,.9));
  border-radius: 12px; padding: 0.75rem; border: 1px solid rgba(255,255,255,.05);
  box-shadow: 0 12px 40px rgba(0,0,0,.4);
  display: flex; gap: 0.75rem;
}

.grafico-half { flex: 1 1 0; min-width: 0; display: flex; flex-direction: column; }

.grafico-canvas {
  background: rgba(15,23,42,.5); border-radius: 10px;
  padding: 0.75rem 0.5rem; height: 380px;
  position: relative; overflow: hidden;
}
.grafico-svg { width: 100%; height: 100%; }
.dot-metric { transition: opacity .15s; pointer-events: none; }

.chart-tooltip {
  position: absolute; top: 8px;
  background: rgba(30,41,59,.96); border: 1px solid rgba(148,163,184,.25);
  border-radius: 8px; padding: 0.6rem 0.85rem;
  pointer-events: none; z-index: 20; min-width: 180px;
  box-shadow: 0 8px 24px rgba(0,0,0,.5); font-size: 0.82rem;
}
.ct-time { color: #f1f5f9; font-weight: 800; font-size: 0.85rem; margin-bottom: 0.4rem; border-bottom: 1px solid rgba(148,163,184,.15); padding-bottom: 0.35rem; }
.ct-row  { display: flex; align-items: center; gap: 0.5rem; color: #cbd5e1; padding: 0.15rem 0; font-weight: 600; }
.ct-dot  { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.ct-val  { margin-left: auto; font-weight: 800; color: #f1f5f9; }

.grafico-leyenda { display: flex; justify-content: center; gap: 3rem; margin-top: 0.5rem; }
.leyenda-item { display: flex; align-items: center; gap: 0.75rem; font-size: 0.95rem; font-weight: 600; color: #cbd5e1; }
.linea-color  { width: 40px; height: 4px; border-radius: 2px; }
.ley-dot      { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }

.camera-half {
  flex: 1 1 0; min-width: 0; border-radius: 10px; overflow: hidden;
  background: rgba(15,23,42,.6); border: 1px solid rgba(255,255,255,.05);
  display: flex; align-items: center; justify-content: center; min-height: 380px;
}

.camera-container {
  position: relative; width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
}

.camera-feed {
  width: 100%; height: 100%; object-fit: cover; display: block; border-radius: 10px;
}

.camera-roi-canvas {
  position: absolute; pointer-events: none;
}

.camera-connecting {
  position: absolute; inset: 0;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.5rem;
}
.camera-connecting p { font-size: 0.9rem; color: #475569; margin: 0; }

.camera-live-badge {
  position: absolute; bottom: 0.75rem; right: 0.75rem;
  display: inline-flex; align-items: center; gap: 0.5rem;
  background: rgba(0,0,0,.65); border: 1px solid rgba(239,68,68,.5);
  color: #ef4444; font-size: 0.75rem; font-weight: 800; letter-spacing: 1px;
  padding: 0.3rem 0.75rem; border-radius: 6px; backdrop-filter: blur(4px);
}
.camera-live-dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: #ef4444; animation: pulse 1.8s infinite;
}

.camera-placeholder, .camera-offline {
  position: absolute; inset: 0;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 0.9rem; text-align: center; padding: 2rem;
  background: #0f1623;
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 8px;
}
.camera-placeholder { color: #64748b; }
.camera-offline     { color: #94a3b8; }
.camera-placeholder p, .camera-offline p {
  font-size: 1rem; font-weight: 700; margin: 0;
}
.camera-placeholder p { color: #64748b; }
.camera-offline p     { color: #f87171; }
.camera-placeholder span, .camera-offline span {
  font-size: 0.78rem;
}
.camera-placeholder span { color: #475569; }
.camera-offline span     { color: #64748b; }

@media (max-width: 1200px) {
  .fila-principal { grid-template-columns: 1fr; }
}
</style>
