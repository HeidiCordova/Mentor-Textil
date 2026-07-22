<template>
  <div class="dashboard-tv-fullscreen">
    <!-- Modal de Selección de Compañía -->
    <Teleport to="body">
      <div v-if="!companiaSeleccionada" class="modal-seleccion-fullscreen">
        <div class="modal-contenido">
          <h2>Seleccionar Compañía</h2>
          <div class="companias-grid">
            <button
              v-for="comp in companias"
              :key="comp.id"
              @click="seleccionarCompania(comp.id)"
              class="btn-compania"
            >
              {{ comp.nombre }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Modal de Selección de Planta -->
    <Teleport to="body">
      <div v-if="companiaSeleccionada && !plantaSeleccionada" class="modal-seleccion-fullscreen">
        <div class="modal-contenido">
          <div class="modal-back">
            <button @click="volverCompania" class="btn-back">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 12H5M12 19l-7-7 7-7"/>
              </svg>
              Volver
            </button>
          </div>
          <h2>Seleccionar Planta</h2>
          <p class="subtitle-modal">{{ getNombreCompania() }}</p>
          <div class="plantas-grid">
            <button
              v-for="planta in plantasFiltradas"
              :key="planta.id"
              @click="seleccionarPlanta(planta.id)"
              class="btn-planta"
            >
              {{ planta.nombre }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Modal de Selección de Línea -->
    <Teleport to="body">
      <div v-if="companiaSeleccionada && plantaSeleccionada && !lineaSeleccionada" class="modal-seleccion-fullscreen">
        <div class="modal-contenido">
          <div class="modal-back">
            <button @click="volverPlanta" class="btn-back">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 12H5M12 19l-7-7 7-7"/>
              </svg>
              Volver
            </button>
          </div>
          <h2>Seleccionar Línea</h2>
          <p class="subtitle-modal">{{ getNombrePlanta() }}</p>
          <div class="lineas-grid">
            <button
              v-for="linea in lineas"
              :key="linea.id"
              @click="seleccionarLinea(linea.id)"
              class="btn-linea"
            >
              {{ linea.nombre }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Dashboard Principal -->
    <div v-if="companiaSeleccionada && plantaSeleccionada && lineaSeleccionada" class="dashboard-content">
      
      <!-- Header con info y hora -->
      <header class="header-tv">
        <div class="header-left">
          <h1>{{ getNombreCompania() }}</h1>
          <p>{{ getNombrePlanta() }}</p>
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
        
        <!-- Fila 1: OEE Principal + Componentes + Estado -->
        <section class="fila-principal">
          
          <!-- OEE Circular Grande -->
          <div class="oee-hero">
            <svg viewBox="0 0 300 300" class="oee-svg">
              <defs>
                <linearGradient id="oeeGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" :stop-color="getColorOEE(datosLinea.oee)" stop-opacity="0.8"/>
                  <stop offset="100%" :stop-color="getColorOEE(datosLinea.oee)"/>
                </linearGradient>
              </defs>
              <circle cx="150" cy="150" r="130" fill="none" stroke="#1e293b" stroke-width="12" opacity="0.15"/>
              <circle 
                cx="150" 
                cy="150" 
                r="130" 
                fill="none" 
                stroke="url(#oeeGrad)" 
                stroke-width="12"
                :stroke-dasharray="`${(datosLinea.oee / 100) * 816.81} 816.81`"
                transform="rotate(-90 150 150)"
                stroke-linecap="round"
                class="oee-circle"
              />
            </svg>
            <div class="oee-content">
              <div class="oee-valor">{{ datosLinea.oee.toFixed(1) }}<span>%</span></div>
              <div class="oee-label">OEE</div>
              <div :class="['oee-badge', getClaseOEE(datosLinea.oee)]">
                {{ getEstadoOEE(datosLinea.oee) }}
              </div>
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

            <!-- Velocidad Actual -->
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

            <!-- Defectos -->
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

            <!-- Paradas -->
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
              <div :class="['estado-badge', estadoEfectivo.toLowerCase()]">
                <span class="pulse-dot"></span>
                {{ estadoEfectivo.toUpperCase() }}
              </div>
              <div class="tiempo-estado">{{ datosLinea.tiempoEstado }}</div>
              <div v-if="datosLinea.motivoParada && estadoEfectivo !== 'Produciendo'" class="motivo">
                ⚠️ {{ datosLinea.motivoParada }}
              </div>
            </div>
            <div class="producto-card">
              <div class="producto-label">PRODUCTO ACTUAL</div>
              <div class="producto-nombre">{{ datosLinea.productoActual }}</div>
              <div class="producto-sku">{{ datosLinea.skuActual }}</div>
            </div>
          </div>
        </section>

        <!-- Gráfico de Tendencia + Cámara en tiempo real -->
        <section class="fila-grafico" @mouseleave="hoverIdx = -1">

          <!-- Mitad izquierda: gráfico OEE -->
          <div class="grafico-half">
          <div class="grafico-canvas" ref="canvasRef" @mousemove="onGraficoHover" @mouseleave="hoverIdx = -1">
            <svg :viewBox="`0 0 ${SVG_W} ${SVG_H}`" preserveAspectRatio="none" class="grafico-svg">
              <!-- Grid horizontal -->
              <g class="grid">
                <line v-for="i in 5" :key="'h'+i"
                  :x1="PAD_L" :y1="PAD_T + (i-1) * STEP_Y" :x2="SVG_W - PAD_R" :y2="PAD_T + (i-1) * STEP_Y"
                  stroke="#334155" stroke-width="1" opacity="0.25"/>
              </g>

              <!-- Labels Y -->
              <g class="labels-y">
                <text v-for="(val, i) in [100, 75, 50, 25, 0]" :key="'y'+i"
                  :x="PAD_L - 12" :y="PAD_T + i * STEP_Y + 5"
                  text-anchor="end" fill="#94a3b8" font-size="13" font-weight="600">{{ val }}%</text>
              </g>

              <!-- Labels X -->
              <g class="labels-x">
                <template v-for="(punto, idx) in datosTendencia" :key="'x'+idx">
                  <text v-if="xLabelVisible(idx)"
                    :x="ptX(idx)" :y="SVG_H - 8"
                    text-anchor="middle" fill="#94a3b8" font-size="12" font-weight="500">{{ punto.tiempo }}</text>
                </template>
              </g>

              <!-- Áreas de relleno -->
              <path :d="areaPath('oee')"           fill="url(#fillOee)"  />
              <path :d="areaPath('disponibilidad')" fill="url(#fillDisp)" />
              <path :d="areaPath('rendimiento')"    fill="url(#fillRend)" />
              <path :d="areaPath('calidad')"        fill="url(#fillCal)"  />

              <!-- Gradientes para áreas -->
              <defs>
                <linearGradient id="fillOee" x1="0" y1="0" x2="0" y2="1">
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
                <linearGradient id="fillCal" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="#ef4444" stop-opacity="0.12"/>
                  <stop offset="100%" stop-color="#ef4444" stop-opacity="0.01"/>
                </linearGradient>
              </defs>

              <!-- Líneas principales -->
              <path :d="generarPathMetrica('calidad')"        stroke="#ef4444" stroke-width="2.5" fill="none" stroke-linejoin="round" stroke-linecap="round"/>
              <path :d="generarPathMetrica('disponibilidad')"  stroke="#f97316" stroke-width="2.5" fill="none" stroke-linejoin="round" stroke-linecap="round"/>
              <path :d="generarPathMetrica('rendimiento')"     stroke="#22c55e" stroke-width="2.5" fill="none" stroke-linejoin="round" stroke-linecap="round"/>
              <path :d="generarPathMetrica('oee')"             stroke="#3b82f6" stroke-width="3"   fill="none" stroke-linejoin="round" stroke-linecap="round"/>

              <!-- Puntos en cada dato (dots visibles) -->
              <template v-for="(p, idx) in datosTendencia" :key="'dots'+idx">
                <circle :cx="ptX(idx)" :cy="ptY(p.oee)"            r="3" fill="#3b82f6" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                <circle :cx="ptX(idx)" :cy="ptY(p.disponibilidad)" r="3" fill="#f97316" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                <circle :cx="ptX(idx)" :cy="ptY(p.rendimiento)"    r="3" fill="#22c55e" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
                <circle :cx="ptX(idx)" :cy="ptY(p.calidad)"        r="3" fill="#ef4444" :opacity="hoverIdx === idx ? 1 : 0.7" class="dot-metric"/>
              </template>

              <!-- Línea vertical de hover -->
              <line v-if="hoverIdx >= 0"
                :x1="ptX(hoverIdx)" :y1="PAD_T" :x2="ptX(hoverIdx)" :y2="PAD_T + CHART_H"
                stroke="#cbd5e1" stroke-width="1" stroke-dasharray="4,3" opacity="0.5"/>

              <!-- Puntos grandes en hover -->
              <template v-if="hoverIdx >= 0 && datosTendencia[hoverIdx]">
                <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].oee)"            r="6" fill="#3b82f6" stroke="#0f172a" stroke-width="2"/>
                <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].disponibilidad)" r="6" fill="#f97316" stroke="#0f172a" stroke-width="2"/>
                <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].rendimiento)"    r="6" fill="#22c55e" stroke="#0f172a" stroke-width="2"/>
                <circle :cx="ptX(hoverIdx)" :cy="ptY(datosTendencia[hoverIdx].calidad)"        r="6" fill="#ef4444" stroke="#0f172a" stroke-width="2"/>
              </template>
            </svg>

            <!-- Tooltip flotante tipo Grafana -->
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
            <div class="leyenda-item"><span class="linea-color" style="background: #3b82f6"></span><span class="ley-dot" style="background:#3b82f6"></span> OEE: {{ datosLinea.oee.toFixed(1) }}%</div>
            <div class="leyenda-item"><span class="linea-color" style="background: #f97316"></span><span class="ley-dot" style="background:#f97316"></span> Disponibilidad: {{ datosLinea.disponibilidad.toFixed(1) }}%</div>
            <div class="leyenda-item"><span class="linea-color" style="background: #22c55e"></span><span class="ley-dot" style="background:#22c55e"></span> Rendimiento: {{ datosLinea.rendimiento.toFixed(1) }}%</div>
            <div class="leyenda-item"><span class="linea-color" style="background: #ef4444"></span><span class="ley-dot" style="background:#ef4444"></span> Calidad: {{ datosLinea.calidad.toFixed(1) }}%</div>
          </div>
          </div><!-- /grafico-half -->

          <!-- Mitad derecha: cámara en tiempo real -->
          <div class="camera-half">
            <div v-if="cameraStreamURL && !cameraError" class="camera-container" ref="cameraContainer">
              <img
                :src="cameraStreamURL"
                class="camera-feed"
                alt="Cámara en vivo"
                ref="cameraImg"
                @load="onCameraLoad"
                @error="onCameraError"
              />
              <canvas ref="roiCanvas" class="camera-roi-canvas" />
              <div class="camera-live-badge">
                <span class="camera-live-dot"></span>
                EN VIVO
              </div>
            </div>
            <div v-else-if="cameraStreamURL && cameraError" class="camera-placeholder camera-placeholder--offline">
              <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4">
                <path d="M23 7l-7 5 7 5V7z" stroke-opacity="0.5"/>
                <rect x="1" y="5" width="15" height="14" rx="2" ry="2" stroke-opacity="0.5"/>
                <line x1="1" y1="1" x2="23" y2="23" stroke="#ef4444" stroke-width="2"/>
              </svg>
              <p>Sin señal de cámara</p>
              <span>El dispositivo no está enviando video</span>
            </div>
            <div v-else class="camera-placeholder">
              <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4">
                <path d="M23 7l-7 5 7 5V7z" stroke-opacity="0.5"/>
                <rect x="1" y="5" width="15" height="14" rx="2" ry="2" stroke-opacity="0.5"/>
                <line x1="1" y1="1" x2="23" y2="23" stroke="#64748b" stroke-width="2"/>
              </svg>
              <p>Sin cámara asignada</p>
              <span>El dispositivo no tiene cámara configurada</span>
            </div>
          </div><!-- /camera-half -->

        </section>
      </main>
    </div>

    <!-- Tooltip -->
    <Teleport to="body">
      <div v-if="tooltipVisible" class="tooltip" :style="{ left: tooltipX + 'px', top: tooltipY + 'px' }">
        <div class="tooltip-time">{{ tooltipData.tiempo }}</div>
        <div class="tooltip-val">{{ tooltipData.valor }}</div>
        <div class="tooltip-label">unidades</div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { oeeService } from '@/api/services/oee.service'
import { stopsService } from '@/api/services/stops.service'
import { turnoService } from '@/api/services/turno.service'
import { velocidadNominalService } from '@/api/services/velocidadNominal.service'
import { deviceService } from '@/api/services/device.service'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// ── Estado ────────────────────────────────────────────────────────────────────
const companias          = ref([])
const plantas            = ref([])
const lineas             = ref([])

const companiaSeleccionada = ref(null)
const plantaSeleccionada   = ref(null)
const lineaSeleccionada    = ref(null)

const datosLinea     = ref(null)
const datosTendencia = ref([])
const horaActual     = ref('')
const fechaActual    = ref('')
const turnoActual    = ref('')
const intervalo      = ref(null)
const relojInterval  = ref(null)

// ── Cámara en tiempo real ─────────────────────────────────────────────────────
const deviceIdForCamera = ref('')
const roiCanvas         = ref(null)
const cameraImg         = ref(null)
const cameraContainer   = ref(null)
const cameraROI         = ref(null) // { roi, roi_presencia }
const cameraError       = ref(false)
let   _cameraResizeObs  = null

const presenceMotion    = ref(false)
let   _offscreenCanvas  = null
let   _prevPresencePx   = null
let   _analyzeRafId     = null
let   _analyzeSkip      = 0

const cameraStreamURL = computed(() => {
  if (!deviceIdForCamera.value || !authStore.token) return ''
  return `/api/v1/camera/stream?device_id=${encodeURIComponent(deviceIdForCamera.value)}&token=${encodeURIComponent(authStore.token)}`
})

// Estado efectivo: si hay ROI de presencia configurada, lo determina el análisis de frames
// en tiempo real (presenceMotion). Las paradas siguen teniendo prioridad.
const estadoEfectivo = computed(() => {
  if (!datosLinea.value) return '—'
  const d = datosLinea.value
  if (d.estado === 'Parada' || d.estado === 'Microparada') return d.estado
  if (cameraROI.value?.roi_presencia) {
    return presenceMotion.value ? 'Produciendo' : 'Detenida'
  }
  return d.estado
})

async function fetchDeviceForLinea(lineaId) {
  cameraError.value = false
  try {
    const res = await deviceService.getAll({ linea_id: lineaId })
    const devices = Array.isArray(res?.data) ? res.data
                  : Array.isArray(res) ? res : []
    deviceIdForCamera.value = devices[0]?.device_id || ''
    if (deviceIdForCamera.value) {
      fetchCameraROI(deviceIdForCamera.value, lineaId)
    }
  } catch {
    deviceIdForCamera.value = ''
  }
}

async function fetchCameraROI(deviceId, lineaId) {
  try {
    const token = authStore.token
    const lineaParam = lineaId ? `&linea_id=${encodeURIComponent(lineaId)}` : ''
    const res = await fetch(`/api/v1/camera/roi?device_id=${encodeURIComponent(deviceId)}${lineaParam}&token=${encodeURIComponent(token)}`)
    if (res.ok) {
      const data = await res.json()
      // Si el edge reportó explícitamente que esta línea no tiene cámara,
      // ocultar el stream aunque el dispositivo esté activo.
      if (data?.has_camera === false) {
        deviceIdForCamera.value = ''
        return
      }
      cameraROI.value = data
      drawCameraROI()
    }
  } catch { /* non-fatal */ }
}

function onCameraLoad() {
  cameraError.value = false
  drawCameraROI()
  _startFrameAnalysis()
}

function onCameraError() {
  cameraError.value = true
}

function parseROIArray(v) {
  if (!v) return null
  if (Array.isArray(v) && v.length >= 4) return v.map(Number)
  if (typeof v === 'object' && v.width > 0) return [v.x, v.y, v.width, v.height]
  return null
}

// ── Análisis de frames en tiempo real (mismo algoritmo que UI local) ──────────
// Lee el img MJPEG → offscreen canvas → diff inter-frame en ROI presencia.
function analyzeFrameImg(img) {
  const iw = img.naturalWidth, ih = img.naturalHeight
  if (iw < 4 || ih < 4) return
  if (!_offscreenCanvas) _offscreenCanvas = document.createElement('canvas')
  if (_offscreenCanvas.width !== iw || _offscreenCanvas.height !== ih) {
    _offscreenCanvas.width = iw; _offscreenCanvas.height = ih
  }
  const ctx = _offscreenCanvas.getContext('2d', { willReadFrequently: true })
  ctx.drawImage(img, 0, 0)

  const roiPres = parseROIArray(cameraROI.value?.roi_presencia)
  if (!roiPres) return

  const srcW = cameraROI.value?.source_w || 1280
  const srcH = cameraROI.value?.source_h || 720
  const scaleX = iw / srcW, scaleY = ih / srcH
  const [px, py, pw, ph] = roiPres
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
    presenceMotion.value = (sum / n) > 8.0
  }
  _prevPresencePx = gray
}

function _startFrameAnalysis() {
  if (_analyzeRafId) return
  function loop() {
    _analyzeSkip++
    if (_analyzeSkip >= 3) {
      _analyzeSkip = 0
      const img = cameraImg.value
      if (img && img.naturalWidth > 0) {
        try { analyzeFrameImg(img) } catch { /* ignore */ }
      }
    }
    _analyzeRafId = requestAnimationFrame(loop)
  }
  _analyzeRafId = requestAnimationFrame(loop)
}

function _stopFrameAnalysis() {
  if (_analyzeRafId) { cancelAnimationFrame(_analyzeRafId); _analyzeRafId = null }
  _prevPresencePx = null
  presenceMotion.value = false
}

function drawCameraROI() {
  const img       = cameraImg.value
  const canvas    = roiCanvas.value
  const container = cameraContainer.value
  if (!img || !canvas || !container) return

  const roi     = parseROIArray(cameraROI.value?.roi)
  const roiPres = parseROIArray(cameraROI.value?.roi_presencia)
  if (!roi && !roiPres) {
    canvas.width = 0; canvas.height = 0
    return
  }

  const rect    = img.getBoundingClientRect()
  const conRect = container.getBoundingClientRect()
  const left    = rect.left - conRect.left
  const top     = rect.top  - conRect.top
  const w       = rect.width
  const h       = rect.height

  canvas.style.left   = left + 'px'
  canvas.style.top    = top  + 'px'
  canvas.width  = w
  canvas.height = h

  const ctx   = canvas.getContext('2d')
  // Las coords ROI vienen en espacio de la cámara original (source_w × source_h).
  // El stream ffmpeg está escalado a 640 px, así que NO usar img.naturalWidth.
  const srcW  = cameraROI.value?.source_w || 1280
  const srcH  = cameraROI.value?.source_h || 720
  const sx    = w / srcW
  const sy    = h / srcH

  ctx.clearRect(0, 0, w, h)

  // ── ROI principal (dashed emerald) ────────────────────────────────────────
  if (roi) {
    const [rx, ry, rw, rh] = roi
    const x  = rx * sx, y  = ry * sy, bw = rw * sx, bh = rh * sy
    ctx.strokeStyle = 'rgba(52,211,153,0.9)'
    ctx.lineWidth   = 2
    ctx.setLineDash([6, 3])
    ctx.strokeRect(x, y, bw, bh)
    ctx.setLineDash([])
    // Corner accents
    const cL = Math.min(12, bw * 0.15, bh * 0.3)
    ctx.lineWidth = 3; ctx.strokeStyle = 'rgba(52,211,153,1)'
    ctx.beginPath(); ctx.moveTo(x, y + cL); ctx.lineTo(x, y); ctx.lineTo(x + cL, y); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(x + bw - cL, y); ctx.lineTo(x + bw, y); ctx.lineTo(x + bw, y + cL); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(x, y + bh - cL); ctx.lineTo(x, y + bh); ctx.lineTo(x + cL, y + bh); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(x + bw - cL, y + bh); ctx.lineTo(x + bw, y + bh); ctx.lineTo(x + bw, y + bh - cL); ctx.stroke()
    // Label
    ctx.fillStyle = 'rgba(0,0,0,0.5)'; ctx.fillRect(x, y - 18 < 0 ? y + 2 : y - 18, 30, 16)
    ctx.fillStyle = 'rgba(52,211,153,1)'; ctx.font = '11px monospace'
    ctx.fillText('ROI', x + 4, y - 18 < 0 ? y + 13 : y - 5)
  }

  // ── ROI Presencia — color y label dinámicos según estado de la línea ───────
  if (roiPres) {
    const [px, py, pw, ph] = roiPres
    const x  = px * sx, y  = py * sy, bw = pw * sx, bh = ph * sy
    const producing = cameraROI.value?.roi_presencia ? presenceMotion.value : datosLinea.value?.estado === 'Produciendo'
    const pColor    = producing ? 'rgba(52,211,153,0.9)'  : 'rgba(248,113,113,0.85)'
    const pFill     = producing ? 'rgba(52,211,153,0.08)' : 'rgba(248,113,113,0.05)'
    const pLabel    = producing ? 'En movimiento'         : 'Detenida'
    const pLabelW   = producing ? 90                      : 58
    ctx.fillStyle = pFill
    ctx.fillRect(x, y, bw, bh)
    ctx.strokeStyle = pColor
    ctx.lineWidth   = producing ? 2 : 1.5; ctx.setLineDash([5, 4])
    ctx.strokeRect(x, y, bw, bh)
    ctx.setLineDash([])
    // Label
    const labelY = y - 18 < 0 ? y + 2 : y - 18
    ctx.fillStyle = producing ? 'rgba(6,78,59,0.85)' : 'rgba(69,10,10,0.85)'
    ctx.fillRect(x, labelY, pLabelW, 16)
    ctx.fillStyle = pColor; ctx.font = 'bold 11px monospace'
    ctx.fillText(pLabel, x + 4, labelY < y ? labelY + 11 : labelY + 13)
  }
}

const tooltipVisible = ref(false)
const tooltipX       = ref(0)
const tooltipY       = ref(0)
const tooltipData    = ref({})

// plantasFiltradas ya están pre-filtradas al cargar
const plantasFiltradas = computed(() => plantas.value)

// ── Lifecycle ─────────────────────────────────────────────────────────────────
onMounted(async () => {
  actualizarFechaHora()
  relojInterval.value = setInterval(actualizarFechaHora, 1000)

  await cargarCompanias()

  const compId  = route.query.c     ? parseInt(route.query.c)     : null
  const plantId = route.query.plant ? parseInt(route.query.plant) : null
  const lineId  = route.query.line  ? parseInt(route.query.line)  : null

  if (compId && plantId && lineId) {
    companiaSeleccionada.value = compId
    await cargarPlantas(compId)
    plantaSeleccionada.value = plantId
    await cargarLineas(plantId)
    lineaSeleccionada.value  = lineId
    await iniciarMonitoreo()
    fetchDeviceForLinea(lineId)
  }

  // ResizeObserver: redibuja el canvas ROI cuando el contenedor de cámara cambia de tamaño
  _cameraResizeObs = new ResizeObserver(() => drawCameraROI())
  if (cameraContainer.value) _cameraResizeObs.observe(cameraContainer.value)

  // SSE: escuchar eventos en tiempo real del cloud-gateway
  _conectarSSEBrowser()
})

// El contenedor de cámara se monta después (condicionado a cameraStreamURL).
// Con watch lo observamos en cuanto aparece en el DOM.
watch(cameraContainer, (el) => {
  if (_cameraResizeObs) _cameraResizeObs.disconnect()
  if (el) _cameraResizeObs.observe(el)
})

// Redibujar el canvas ROI cuando cambia el estado (polling 60s) o la detección en vivo.
watch(() => datosLinea.value?.estado, () => drawCameraROI())
watch(presenceMotion, () => drawCameraROI())

onUnmounted(() => {
  if (intervalo.value)     clearTimeout(intervalo.value)
  if (relojInterval.value) clearInterval(relojInterval.value)
  if (_cameraResizeObs)    _cameraResizeObs.disconnect()
  _stopFrameAnalysis()
  _desconectarSSEBrowser()
})

// ── SSE browser — tiempo real cloud ──────────────────────────────────────────
let _sseSource = null
let _sseReconnectTimer = null
let _sseReconnectMs = 1000
const SSE_RECONNECT_MAX_MS = 30_000

function _conectarSSEBrowser() {
  const token = authStore.token
  if (!token) return
  const empresaId = authStore.user?.empresa_id || companiaSeleccionada.value
  let url = `/api/v1/cloud/stream?token=${encodeURIComponent(token)}`
  if (empresaId) url += `&empresa_id=${empresaId}`
  _sseSource = new EventSource(url)

  _sseSource.onopen = () => {
    _sseReconnectMs = 1000
  }

  _sseSource.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      if (msg.type === 'stop.changed' && msg.linea_id === lineaSeleccionada.value) {
        cargarDatosSafe()
      }
    } catch { /* ignorar mensajes malformados */ }
  }

  _sseSource.onerror = () => {
    _sseSource?.close()
    _sseSource = null
    _sseReconnectTimer = setTimeout(() => {
      _sseReconnectTimer = null
      _conectarSSEBrowser()
    }, _sseReconnectMs)
    _sseReconnectMs = Math.min(_sseReconnectMs * 2, SSE_RECONNECT_MAX_MS)
  }
}

function _desconectarSSEBrowser() {
  if (_sseReconnectTimer) { clearTimeout(_sseReconnectTimer); _sseReconnectTimer = null }
  if (_sseSource) { _sseSource.close(); _sseSource = null }
}

// ── Carga de catálogos ────────────────────────────────────────────────────────
async function cargarCompanias() {
  try {
    const res = await companyService.getAll()
    companias.value = res.data?.data || res.data || []
  } catch { companias.value = [] }
}

async function cargarPlantas(empresaId) {
  try {
    const res = await plantService.getAll({ empresa_id: empresaId })
    plantas.value = res.data?.data || res.data || []
  } catch { plantas.value = [] }
}

async function cargarLineas(plantaId) {
  try {
    const res = await lineService.getAll({ planta_id: plantaId })
    lineas.value = res.data?.data || res.data || []
  } catch { lineas.value = [] }
}

// ── Selección modales ─────────────────────────────────────────────────────────
async function seleccionarCompania(id) {
  companiaSeleccionada.value = id
  await cargarPlantas(id)
}

async function seleccionarPlanta(id) {
  plantaSeleccionada.value = id
  lineaSeleccionada.value  = null
  datosLinea.value     = null
  datosTendencia.value = []
  await cargarLineas(id)
}

async function seleccionarLinea(id) {
  lineaSeleccionada.value = id
  router.replace({ query: { c: companiaSeleccionada.value, plant: plantaSeleccionada.value, line: id } })
  await iniciarMonitoreo()
  fetchDeviceForLinea(id)
}

async function cambiarLinea(id) {
  lineaSeleccionada.value = id
  datosLinea.value     = null
  datosTendencia.value = []
  deviceIdForCamera.value = ''
  _stopFrameAnalysis()
  router.replace({ query: { c: companiaSeleccionada.value, plant: plantaSeleccionada.value, line: id } })
  await iniciarMonitoreo()
  fetchDeviceForLinea(id)
}

function volverPlanta() {
  plantaSeleccionada.value = null
  lineaSeleccionada.value  = null
  lineas.value = []
  datosLinea.value     = null
  datosTendencia.value = []
  if (intervalo.value) clearTimeout(intervalo.value)
}

function volverCompania() {
  companiaSeleccionada.value = null
  plantaSeleccionada.value   = null
  lineaSeleccionada.value    = null
  plantas.value = []
  lineas.value  = []
  datosLinea.value     = null
  datosTendencia.value = []
  if (intervalo.value) clearTimeout(intervalo.value)
}

// ── Helpers de nombre ─────────────────────────────────────────────────────────
function getNombreCompania() {
  return companias.value.find(c => c.id === companiaSeleccionada.value)?.nombre || ''
}
function getNombrePlanta() {
  return plantas.value.find(p => p.id === plantaSeleccionada.value)?.nombre || ''
}

function actualizarFechaHora() {
  const now = new Date()
  horaActual.value  = now.toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  fechaActual.value = now.toLocaleDateString('es-ES', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })
}

function formatDuracionMin(min) {
  if (min === null || min === undefined || isNaN(min)) return '—'
  const h = Math.floor(min / 60)
  const m = Math.round(min % 60)
  return h > 0 ? `${h}h ${m}min` : `${m}min`
}

function turnoNombreActual(turnos) {
  if (!turnos?.length) return '—'
  const now = new Date()
  const hh  = now.getHours() * 60 + now.getMinutes()
  const toMin = t => { if (!t) return 0; const [h, m] = t.split(':').map(Number); return h * 60 + m }
  for (const t of turnos) {
    const ini = toMin(t.hora_inicio || t.inicio)
    const fin = toMin(t.hora_fin    || t.fin)
    if (ini < fin ? (hh >= ini && hh < fin) : (hh >= ini || hh < fin))
      return t.nombre || t.name || `Turno ${t.id}`
  }
  return turnos[0]?.nombre || '—'
}

// ── Monitoreo principal ───────────────────────────────────────────────────────
const REFRESH_MS = 30_000  // cada 30 s

async function iniciarMonitoreo() {
  if (intervalo.value) { clearTimeout(intervalo.value); intervalo.value = null }
  await cargarDatosSafe()
  programarSiguiente()
}

function programarSiguiente() {
  if (intervalo.value) clearTimeout(intervalo.value)
  intervalo.value = setTimeout(async () => {
    await cargarDatosSafe()
    programarSiguiente()           // solo programa el siguiente DESPUÉS de terminar
  }, REFRESH_MS)
}

async function cargarDatosSafe() {
  try { await cargarDatos() } catch (e) { console.warn('[TiempoReal] error refresh:', e) }
}

async function cargarDatos() {
  if (!lineaSeleccionada.value) return
  const lineaId = lineaSeleccionada.value

  const plantaId = plantaSeleccionada.value

  const lineaObj   = lineas.value.find(l => l.id === lineaId)
  const lineaMode  = lineaObj?.mode ?? 'botellas'
  const minIS      = lineaMode === 'textil' ? 1800 : 300
  // 8 h de datos: 16 snapshots textil (30 min) ó 96 snapshots botellas (5 min)
  const snapLimit  = lineaMode === 'textil' ? 16 : 96

  const [latestRes, snapsRes, stopsRes, turnosRes, velRes] = await Promise.allSettled([
    oeeService.getLatest(lineaId, plantaId),
    oeeService.getSnapshots({ linea_id: lineaId, planta_id: plantaId, limit: snapLimit, min_interval_s: minIS }),
    stopsService.list({ linea_id: lineaId, planta_id: plantaId }),
    turnoService.getAll({ linea_id: lineaId }),
    velocidadNominalService.getByLinea({ linea_id: lineaId })
  ])

  // ── OEE latest ───────────────────────────────────────────────────────────
  // IMPORTANTE: el interceptor axios (client.js) ya retorna response.data
  // directamente, por lo que latestRes.value YA ES el OEESnapshot (el JSON body)
  // NO hacer raw?.data porque eso accedería al campo "data: []int64" del snapshot
  const latest = latestRes.status === 'fulfilled' ? latestRes.value : null

  const oee            = parseFloat(latest?.oee            ?? 0)
  const disponibilidad = parseFloat(latest?.disponibilidad ?? 0)
  const rendimiento    = parseFloat(latest?.rendimiento    ?? 0)
  const calidad        = parseFloat(latest?.calidad        ?? 0)
  const produccionTurno = parseInt(latest?.produccion      ?? 0, 10)

  // ── Snapshots ─────────────────────────────────────────────────────────────
  const snapsRaw = snapsRes.status === 'fulfilled' ? snapsRes.value : null
  const snapArr  = Array.isArray(snapsRaw?.data?.data) ? snapsRaw.data.data
                 : Array.isArray(snapsRaw?.data)       ? snapsRaw.data
                 : []

  // Velocidad actual = delta de producción últimos 2 snapshots
  let velocidadActual = '—'
  if (snapArr.length >= 2) {
    const d = parseInt(snapArr.at(-1)?.produccion ?? 0) - parseInt(snapArr.at(-2)?.produccion ?? 0)
    if (d > 0) velocidadActual = `${d} u/5min`
  }

  // Merma/defectos del último snapshot
  let defectosTurno = 0
  const ultimo = snapArr.at(-1)
  if (Array.isArray(ultimo?.head) && Array.isArray(ultimo?.data)) {
    const idxM = ultimo.head.findIndex(h => /merma/i.test(h))
    if (idxM >= 0) defectosTurno = parseInt(ultimo.data[idxM] ?? 0)
  }

  // ── Paradas ───────────────────────────────────────────────────────────────
  const stopsRaw  = stopsRes.status === 'fulfilled' ? stopsRes.value : null
  const paradas   = Array.isArray(stopsRaw?.data?.data) ? stopsRaw.data.data
                  : Array.isArray(stopsRaw?.data)        ? stopsRaw.data
                  : []

  // Parada activa: sin fin definido
  const paradaActiva = paradas.find(s =>
    s.activo || s.activa || !s.fin && !s.ended_at
  )

  const totalMinParadas = paradas.reduce((acc, s) => {
    const finTs = s.fin || s.ended_at
    if (!finTs) return acc
    return acc + (new Date(finTs) - new Date(s.inicio || s.started_at || s.created_at)) / 60000
  }, 0)

  let estado      = 'Produciendo'
  let motivoParada = null
  let tiempoEstado = '—'

  if (paradaActiva) {
    const tipo = (paradaActiva.tipo || paradaActiva.tipo_parada || paradaActiva.stop_type || '').toUpperCase()
    estado       = tipo.includes('MICRO') ? 'Microparada' : 'Parada'
    motivoParada = paradaActiva.categoria_nombre || paradaActiva.categoria || paradaActiva.motivo || paradaActiva.nombre || paradaActiva.descripcion || null
    const ini    = new Date(paradaActiva.inicio || paradaActiva.started_at || paradaActiva.created_at)
    tiempoEstado = formatDuracionMin((Date.now() - ini.getTime()) / 60000)
  } else if (oee === 0 && produccionTurno === 0) {
    estado = 'Detenida'
    tiempoEstado = '—'
  }

  // ── Turno ─────────────────────────────────────────────────────────────────
  const turnosRaw  = turnosRes.status === 'fulfilled' ? turnosRes.value : null
  const turnosArr  = Array.isArray(turnosRaw?.data?.data) ? turnosRaw.data.data
                   : Array.isArray(turnosRaw?.data)        ? turnosRaw.data
                   : []
  turnoActual.value = turnoNombreActual(turnosArr)

  // ── Velocidad nominal ─────────────────────────────────────────────────────
  const velRaw  = velRes.status === 'fulfilled' ? velRes.value : null
  const velArr  = Array.isArray(velRaw?.data?.data) ? velRaw.data.data
                : Array.isArray(velRaw?.data)        ? velRaw.data
                : []
  const velNominal     = velArr[0]?.velocidad_nominal ?? null
  const velocidadTeorica = velNominal ? `${velNominal} u/min` : '—'
  const metaTurno        = velNominal ? Math.round(velNominal * 480) : 0

  // ── Armar datosLinea ──────────────────────────────────────────────────────
  datosLinea.value = {
    estado,
    tiempoEstado,
    motivoParada,
    productoActual: latest?.run_nombre || '—',
    skuActual:      latest?.run_sku || '—',
    oee,
    disponibilidad,
    rendimiento,
    calidad,
    produccionTurno,
    metaTurno,
    velocidadActual,
    velocidadTeorica,
    defectosTurno,
    paradasTurno:      paradas.length,
    tiempoParadas:     formatDuracionMin(totalMinParadas),
    tiempoParadaTurno: Math.round(totalMinParadas),
    metaMinuto:        velNominal ? velNominal / 60 : 0
  }

  // ── Tendencia ─────────────────────────────────────────────────────────────
  // El API devuelve snapshots descendentes (más reciente primero) → invertir
  // para que el eje X vaya de más antiguo (izquierda) a más reciente (derecha)
  if (snapArr.length) {
    datosTendencia.value = snapArr.slice().reverse().map(s => ({
      tiempo:          new Date(s.hora || s.fecha || s.timestamp)
                         .toLocaleTimeString('es-ES', { hour: '2-digit', minute: '2-digit' }),
      oee:             parseFloat(s.oee            ?? 0),
      disponibilidad:  parseFloat(s.disponibilidad ?? 0),
      rendimiento:     parseFloat(s.rendimiento    ?? 0),
      calidad:         parseFloat(s.calidad        ?? 0)
    }))
  }
}

// ── SVG chart constants & state ───────────────────────────────────────────────
const SVG_W   = 1600
const SVG_H   = 420
const PAD_L   = 70
const PAD_R   = 30
const PAD_T   = 30
const PAD_B   = 45
const CHART_H = SVG_H - PAD_T - PAD_B   // 345
const STEP_Y  = CHART_H / 4             // 86.25

const canvasRef   = ref(null)
const hoverIdx    = ref(-1)
const tooltipLeft = ref(0)

function ptX(idx) {
  const n = datosTendencia.value.length
  if (n <= 1) return PAD_L
  return PAD_L + idx * ((SVG_W - PAD_L - PAD_R) / (n - 1))
}

function ptY(val) {
  return PAD_T + CHART_H - (val / 100) * CHART_H
}

function xLabelVisible(idx) {
  const n = datosTendencia.value.length
  if (n <= 10) return true
  const step = Math.max(1, Math.floor(n / 8))
  return idx % step === 0
}

function onGraficoHover(e) {
  if (!canvasRef.value || !datosTendencia.value.length) return
  const rect = canvasRef.value.getBoundingClientRect()
  const relX = e.clientX - rect.left
  const pct  = relX / rect.width
  const svgX = pct * SVG_W
  const n    = datosTendencia.value.length
  const gap  = (SVG_W - PAD_L - PAD_R) / Math.max(n - 1, 1)
  let best   = Math.round((svgX - PAD_L) / gap)
  best = Math.max(0, Math.min(n - 1, best))
  hoverIdx.value = best

  // tooltip position relative to canvas
  const tooltipPx = (ptX(best) / SVG_W) * rect.width
  // keep inside bounds
  tooltipLeft.value = Math.max(10, Math.min(rect.width - 200, tooltipPx - 90))
}

// ── SVG path helpers ──────────────────────────────────────────────────────────
function generarPathMetrica(metrica) {
  if (!datosTendencia.value.length) return ''
  return 'M ' + datosTendencia.value.map((p, i) =>
    `${ptX(i)},${ptY(p[metrica])}`
  ).join(' L ')
}

function areaPath(metrica) {
  if (!datosTendencia.value.length) return ''
  const n     = datosTendencia.value.length
  const base  = PAD_T + CHART_H
  const line  = datosTendencia.value.map((p, i) => `${ptX(i)},${ptY(p[metrica])}`).join(' L ')
  return `M ${ptX(0)},${base} L ${line} L ${ptX(n - 1)},${base} Z`
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

function getEstadoOEE(oee) {
  if (oee >= 85) return 'EXCELENTE'
  if (oee >= 70) return 'ACEPTABLE'
  return 'CRÍTICO'
}

function mostrarTooltip(event, punto) {
  tooltipVisible.value = true
  tooltipX.value = event.clientX
  tooltipY.value = event.clientY
  tooltipData.value = punto
}

function ocultarTooltip() {
  tooltipVisible.value = false
}
</script>

<style scoped>
/* Base - Fullscreen que cubre TODO */
.dashboard-tv-fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #0a0e27 0%, #0f172a 50%, #1e293b 100%);
  overflow: auto;
  z-index: 9998;
}

/* Modal - Cubre TODO incluyendo sidebar */
.modal-seleccion-fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(12px);
}

.modal-contenido {
  background: white;
  border-radius: 20px;
  padding: clamp(1.5rem, 3vw, 3rem);
  max-width: min(800px, 92vw);
  width: 90%;
  box-shadow: 0 30px 60px rgba(0, 0, 0, 0.6);
}

.modal-contenido h2 {
  text-align: center;
  font-size: 2rem;
  color: #001f54;
  font-weight: 800;
  margin-bottom: 2rem;
}

.companias-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1.5rem;
}

.btn-compania {
  background: linear-gradient(135deg, #001f54, #002a70);
  color: white;
  border: none;
  padding: 2rem;
  border-radius: 14px;
  font-size: 1.35rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 6px 12px rgba(0, 31, 84, 0.3);
}

.btn-compania:hover {
  transform: translateY(-6px) scale(1.02);
  box-shadow: 0 12px 24px rgba(0, 31, 84, 0.4);
}

/* Modal Back Button */
.modal-back {
  margin-bottom: 1.5rem;
}

.btn-back {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background: transparent;
  border: 2px solid #001f54;
  color: #001f54;
  padding: 0.75rem 1.5rem;
  border-radius: 10px;
  font-size: 1rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-back:hover {
  background: #001f54;
  color: white;
  transform: translateX(-4px);
}

.subtitle-modal {
  text-align: center;
  font-size: 1.25rem;
  color: #64748b;
  margin: -1rem 0 2rem 0;
  font-weight: 600;
}

/* Grid de Plantas */
.plantas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1.5rem;
}

.btn-planta {
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
  border: none;
  padding: 2rem;
  border-radius: 14px;
  font-size: 1.35rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 6px 12px rgba(16, 185, 129, 0.3);
}

.btn-planta:hover {
  transform: translateY(-6px) scale(1.02);
  box-shadow: 0 12px 24px rgba(16, 185, 129, 0.4);
}

/* Grid de Líneas */
.lineas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1.5rem;
}

.btn-linea {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white;
  border: none;
  padding: 2rem;
  border-radius: 14px;
  font-size: 1.35rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 6px 12px rgba(59, 130, 246, 0.3);
  text-transform: uppercase;
}

.btn-linea:hover {
  transform: translateY(-6px) scale(1.02);
  box-shadow: 0 12px 24px rgba(59, 130, 246, 0.4);
}

/* Select de línea en header */
.select-linea {
  margin-top: 0.4rem;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.4);
  color: #93c5fd;
  padding: 0.35rem 0.75rem;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 700;
  cursor: pointer;
  outline: none;
  text-transform: uppercase;
}

.select-linea option {
  background: #0f172a;
  color: #f1f5f9;
}

/* Dashboard Content */
.dashboard-content {
  padding: 1.5rem;
}

/* Header */
.header-tv {
  background: rgba(15, 23, 42, 0.8);
  border-radius: 12px;
  padding: 1rem 1.5rem;
  margin-bottom: 1rem;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 1.5rem;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.header-left h1 {
  font-size: 1.25rem;
  font-weight: 800;
  color: #f1f5f9;
  text-transform: uppercase;
  margin: 0 0 0.3rem 0;
}

.header-left p {
  font-size: 0.95rem;
  color: #94a3b8;
  margin: 0;
}

.header-center {
  text-align: center;
}

.fecha {
  font-size: 0.85rem;
  color: #cbd5e1;
  text-transform: capitalize;
}

.hora {
  font-size: 1.75rem;
  font-weight: 800;
  color: #f1f5f9;
  font-family: 'Courier New', monospace;
  letter-spacing: 2px;
}

.header-right {
  text-align: right;
}

.turno {
  font-size: 1rem;
  color: #94a3b8;
  margin-bottom: 0.3rem;
}

.turno span {
  font-weight: 800;
  color: #3b82f6;
  font-size: 1.35rem;
}

.live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  color: #10b981;
  font-weight: 700;
}

.live-indicator .dot {
  width: 12px;
  height: 12px;
  background: #10b981;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
}

/* Selector Líneas - Oculto en vista TV (filtrado por URL) */
.selector-lineas {
  display: none;
}

.linea-btn {
  background: rgba(30, 41, 59, 0.6);
  border: 2px solid rgba(148, 163, 184, 0.2);
  border-radius: 12px;
  padding: 1.25rem;
  cursor: pointer;
  transition: all 0.3s;
}

.linea-btn:hover {
  background: rgba(30, 41, 59, 0.9);
  border-color: rgba(59, 130, 246, 0.5);
  transform: translateY(-4px);
}

.linea-btn.active {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.15), rgba(139, 92, 246, 0.15));
  border-color: #3b82f6;
  box-shadow: 0 0 30px rgba(59, 130, 246, 0.3);
}

.linea-nombre {
  font-size: 1.25rem;
  font-weight: 800;
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.linea-estado {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.8rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 700;
  text-transform: uppercase;
}

.linea-estado.produciendo {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.linea-estado.parada {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.linea-estado.microparada {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  animation: pulse 2s infinite;
}

/* Grid Dashboard */
.grid-dashboard {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* Fila Principal */
.fila-principal {
  display: grid;
  grid-template-columns: minmax(200px, 280px) 1fr minmax(200px, 280px);
  gap: 0.75rem;
}

/* OEE Hero */
.oee-hero {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.8), rgba(15, 23, 42, 0.9));
  border-radius: 12px;
  padding: 0.75rem;
  position: relative;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
}

.oee-svg {
  width: 100%;
  height: auto;
}

.oee-circle {
  transition: stroke-dasharray 1s ease;
  filter: drop-shadow(0 0 10px currentColor);
}

.oee-content {
  position: absolute;
  text-align: center;
}

.oee-valor {
  font-size: 2.75rem;
  font-weight: 900;
  color: #f1f5f9;
  line-height: 1;
}

.oee-valor span {
  font-size: 1.5rem;
  opacity: 0.8;
}

.oee-label {
  font-size: 0.85rem;
  font-weight: 700;
  color: #94a3b8;
  margin: 0.2rem 0 0.4rem;
  letter-spacing: 1.5px;
}

.oee-badge {
  padding: 0.4rem 1rem;
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 800;
  letter-spacing: 1px;
}

.oee-badge.excelente {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
}

.oee-badge.aceptable {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.oee-badge.critico {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

/* Componentes */
.componentes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.75rem;
}

.componente {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.6), rgba(15, 23, 42, 0.8));
  border-radius: 10px;
  padding: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.componente svg {
  color: currentColor;
}

.componente.disponibilidad {
  color: #3b82f6;
}

.componente.rendimiento {
  color: #8b5cf6;
}

.componente.calidad {
  color: #10b981;
}

.componente.velocidad {
  color: #3b82f6;
}

.componente.defectos {
  color: #f59e0b;
}

.componente.paradas {
  color: #ef4444;
}

.comp-info {
  text-align: center;
  width: 100%;
}

.comp-label {
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: #cbd5e1;
  margin-bottom: 0.25rem;
}

.comp-valor {
  font-size: 1.5rem;
  font-weight: 900;
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.comp-meta {
  font-size: 0.75rem;
  color: #64748b;
  font-weight: 600;
}

.comp-barra {
  width: 100%;
  height: 10px;
  background: rgba(15, 23, 42, 0.8);
  border-radius: 5px;
  overflow: hidden;
}

.barra-fill {
  height: 100%;
  border-radius: 5px;
  transition: width 1s ease;
  box-shadow: 0 0 12px currentColor;
}

/* Info Panel */
.info-panel {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.estado-card {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.7), rgba(15, 23, 42, 0.9));
  border-radius: 10px;
  padding: 0.75rem;
  text-align: center;
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.estado-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: 0.5px;
  margin-bottom: 0.75rem;
}

.estado-badge.produciendo {
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.2), rgba(34, 197, 94, 0.1));
  color: #22c55e;
  border: 2px solid rgba(34, 197, 94, 0.4);
}

.estado-badge.parada {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.2), rgba(239, 68, 68, 0.1));
  color: #ef4444;
  border: 2px solid rgba(239, 68, 68, 0.4);
}

.estado-badge.microparada {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.2), rgba(245, 158, 11, 0.1));
  color: #f59e0b;
  border: 2px solid rgba(245, 158, 11, 0.4);
}

.estado-badge.detenida {
  background: linear-gradient(135deg, rgba(156, 163, 175, 0.2), rgba(156, 163, 175, 0.1));
  color: #9ca3af;
  border: 2px solid rgba(156, 163, 175, 0.4);
}

.pulse-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: currentColor;
  animation: pulse 2s infinite;
}

.tiempo-estado {
  font-size: 1.1rem;
  color: #cbd5e1;
  font-weight: 600;
}

.motivo {
  margin-top: 1rem;
  padding: 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 8px;
  color: #fca5a5;
  font-weight: 600;
}

.producto-card {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.7), rgba(15, 23, 42, 0.9));
  border-radius: 16px;
  padding: 1.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.producto-label {
  font-size: 0.85rem;
  color: #64748b;
  font-weight: 700;
  letter-spacing: 1px;
  margin-bottom: 0.75rem;
}

.producto-nombre {
  font-size: 1.15rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.producto-sku {
  font-size: 0.95rem;
  color: #94a3b8;
  font-family: 'Courier New', monospace;
}

/* Fila Métricas */
.fila-metricas {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem;
}

.metrica {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.7), rgba(15, 23, 42, 0.9));
  border-radius: 10px;
  padding: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.metrica svg {
  flex-shrink: 0;
}

.metrica.produccion svg {
  color: #22c55e;
}

.metrica.velocidad svg {
  color: #3b82f6;
}

.metrica.defectos svg {
  color: #f59e0b;
}

.metrica.paradas svg {
  color: #ef4444;
}

.metrica-info {
  flex: 1;
}

.metrica-label {
  font-size: 0.75rem;
  color: #94a3b8;
  font-weight: 700;
  text-transform: uppercase;
  margin-bottom: 0.3rem;
}

.metrica-valor {
  font-size: 1.65rem;
  font-weight: 900;
  color: #f1f5f9;
  margin-bottom: 0.3rem;
}

.metrica-meta {
  font-size: 0.85rem;
  color: #64748b;
  font-weight: 600;
}

.progreso-bar {
  width: 100%;
  height: 6px;
  background: rgba(15, 23, 42, 0.8);
  border-radius: 3px;
  overflow: hidden;
  margin-top: 0.75rem;
}

.progreso {
  height: 100%;
  background: linear-gradient(90deg, #10b981, #22c55e);
  border-radius: 3px;
  transition: width 1s ease;
  box-shadow: 0 0 10px rgba(34, 197, 94, 0.5);
}

.metrica-pct {
  font-size: 0.95rem;
  color: #10b981;
  font-weight: 700;
  margin-top: 0.5rem;
}

/* Fila Gráfico */
.fila-grafico {
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.7), rgba(15, 23, 42, 0.9));
  border-radius: 12px;
  padding: 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
  display: flex;
  gap: 0.75rem;
}

.grafico-half {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.camera-half {
  flex: 1 1 0;
  min-width: 0;
  border-radius: 10px;
  overflow: hidden;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 380px;
}

.camera-container {
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.camera-roi-canvas {
  position: absolute;
  pointer-events: none;
  top: 0;
  left: 0;
}

.camera-feed {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  border-radius: 10px;
}

.camera-live-badge {
  position: absolute;
  bottom: 0.75rem;
  right: 0.75rem;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(0, 0, 0, 0.65);
  border: 1px solid rgba(239, 68, 68, 0.5);
  color: #ef4444;
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 1px;
  padding: 0.3rem 0.75rem;
  border-radius: 6px;
  backdrop-filter: blur(4px);
}

.camera-live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
  animation: pulse 1.8s infinite;
}

.camera-placeholder {
  position: absolute; inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.9rem;
  color: #64748b;
  text-align: center;
  padding: 2rem;
  background: #0f1623;
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 8px;
}

.camera-placeholder p {
  font-size: 1rem;
  font-weight: 700;
  color: #64748b;
  margin: 0;
}

.camera-placeholder span {
  font-size: 0.78rem;
  color: #475569;
}

.camera-placeholder--offline p {
  color: #f87171;
}

.camera-placeholder--offline {
  color: #94a3b8;
}

.grafico-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.2);
}

.grafico-header h3 {
  font-size: 1.15rem;
  font-weight: 800;
  color: #f1f5f9;
  margin: 0 0 0.3rem 0;
}

.grafico-header p {
  font-size: 0.75rem;
  color: #64748b;
  margin: 0;
}

.live-badge {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1.5rem;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 10px;
  font-size: 0.95rem;
  font-weight: 800;
  color: #ef4444;
  letter-spacing: 1px;
}

.dot-anim {
  width: 10px;
  height: 10px;
  background: #ef4444;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
}

.grafico-canvas {
  background: rgba(15, 23, 42, 0.5);
  border-radius: 10px;
  padding: 0.75rem 0.5rem;
  margin-bottom: 0;
  height: 380px;
  position: relative;
  overflow: hidden;
}

.grafico-svg {
  width: 100%;
  height: 100%;
}

.dot-metric {
  transition: opacity 0.15s;
  pointer-events: none;
}

/* Chart tooltip estilo Grafana */
.chart-tooltip {
  position: absolute;
  top: 8px;
  background: rgba(30, 41, 59, 0.96);
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 8px;
  padding: 0.6rem 0.85rem;
  pointer-events: none;
  z-index: 20;
  min-width: 180px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.5);
  font-size: 0.82rem;
}

.ct-time {
  color: #f1f5f9;
  font-weight: 800;
  font-size: 0.85rem;
  margin-bottom: 0.4rem;
  border-bottom: 1px solid rgba(148,163,184,0.15);
  padding-bottom: 0.35rem;
}

.ct-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #cbd5e1;
  padding: 0.15rem 0;
  font-weight: 600;
}

.ct-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.ct-val {
  margin-left: auto;
  font-weight: 800;
  color: #f1f5f9;
}

.ley-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.linea-animada {
  animation: draw 2s ease-out;
}

@keyframes draw {
  from {
    stroke-dasharray: 5000;
    stroke-dashoffset: 5000;
  }
  to {
    stroke-dasharray: 5000;
    stroke-dashoffset: 0;
  }
}

.punto {
  cursor: pointer;
  transition: all 0.2s;
}

.punto:hover {
  r: 7;
  filter: drop-shadow(0 0 8px currentColor);
}

.grafico-leyenda {
  display: flex;
  justify-content: center;
  gap: 3rem;
}

.leyenda-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 0.95rem;
  font-weight: 600;
  color: #cbd5e1;
}

.linea-color {
  width: 40px;
  height: 4px;
  border-radius: 2px;
}

.linea-meta-dash {
  width: 40px;
  height: 3px;
  background: #f59e0b;
  position: relative;
}

.linea-meta-dash::before,
.linea-meta-dash::after {
  content: '';
  position: absolute;
  width: 8px;
  height: 3px;
  background: #f59e0b;
}

.linea-meta-dash::before {
  left: -12px;
}

.linea-meta-dash::after {
  right: -12px;
}

.area-color {
  width: 40px;
  height: 12px;
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.3), rgba(59, 130, 246, 0.05));
  border-radius: 2px;
}

/* Tooltip (legacy - unused) */
.tooltip { display: none; }

/* Responsive */
@media (max-width: 1600px) {
  .fila-principal {
    grid-template-columns: minmax(180px, 280px) 1fr minmax(180px, 240px);
  }
}

@media (max-width: 1200px) {
  .fila-principal {
    grid-template-columns: 1fr 1fr;
  }
  .oee-hero {
    max-width: 280px;
    justify-self: center;
  }
  .info-panel {
    max-width: 280px;
    justify-self: center;
  }
  .componentes-grid {
    grid-column: 1 / -1;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 160px), 1fr));
  }
}

@media (max-width: 768px) {
  .fila-principal {
    grid-template-columns: 1fr;
  }
  .oee-hero {
    max-width: 220px;
  }
  .oee-valor {
    font-size: 2rem;
  }
  .info-panel {
    max-width: none;
  }
  .componentes-grid {
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 140px), 1fr));
  }
  .componente svg {
    width: 28px;
    height: 28px;
  }
  .header-tv {
    grid-template-columns: 1fr;
    text-align: center;
  }
  .selector-lineas {
    grid-template-columns: 1fr;
  }
  .companias-grid,
  .plantas-grid,
  .lineas-grid {
    grid-template-columns: 1fr;
  }
  .dashboard-content {
    padding: 0.75rem;
  }
  .modal-contenido h2 {
    font-size: 1.5rem;
  }
}
</style>
