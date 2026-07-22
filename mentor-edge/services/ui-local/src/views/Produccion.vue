<template>
  <div class="flex flex-col h-full bg-slate-950 overflow-hidden">

    <!-- Header -->
    <div class="flex items-center gap-3 px-4 py-2.5 bg-slate-900 border-b border-slate-700/60 shrink-0">
      <!-- Icono + título -->
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-emerald-600/20 border border-emerald-500/30 flex items-center justify-center">
          <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M15 10l4.553-2.069A1 1 0 0121 8.82v6.36a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z"/>
          </svg>
        </div>
        <div>
          <span class="text-sm font-semibold text-white">Modo Producción</span>
          <span class="ml-2 text-xs font-mono text-gray-500">{{ deviceId }}</span>
        </div>
      </div>

      <!-- Badges de estado -->
      <div class="flex items-center gap-2 ml-2">
        <span class="flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full border"
          :class="detectorOk ? 'border-emerald-700 bg-emerald-950/50 text-emerald-300' : 'border-red-700 bg-red-950/50 text-red-300'">
          <span class="w-1.5 h-1.5 rounded-full" :class="detectorOk ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'"></span>
          Detector
        </span>
        <span class="flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full border"
          :class="cameraOk ? 'border-sky-700 bg-sky-950/50 text-sky-300' : 'border-red-700 bg-red-950/50 text-red-300'">
          <span class="w-1.5 h-1.5 rounded-full" :class="cameraOk ? 'bg-sky-400' : 'bg-red-400'"></span>
          Cámara
        </span>
        <span class="flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full border"
          :class="cloudConnected ? 'border-blue-700 bg-blue-950/50 text-blue-300' : 'border-slate-600 bg-slate-800 text-slate-400'">
          <span class="w-1.5 h-1.5 rounded-full" :class="cloudConnected ? 'bg-blue-400 animate-pulse' : 'bg-slate-500'"></span>
          Cloud
        </span>
      </div>

      <div class="flex-1"></div>

      <!-- Contadores de microparadas y paradas -->
      <div class="flex items-center gap-2 border-l border-slate-700/60 pl-3">

        <!-- Microparadas -->
        <div class="flex flex-col items-center px-3 py-1 rounded-lg border min-w-[88px]"
          :class="stopTrackerState === 'idle_wait' ? 'border-amber-500/70 bg-amber-950/50' : 'border-slate-700/60 bg-slate-800/40'">
          <div class="flex items-center gap-1.5 w-full justify-between">
            <span class="text-[9px] font-semibold uppercase tracking-widest"
              :class="stopTrackerState === 'idle_wait' ? 'text-amber-400' : 'text-gray-500'">Microparadas</span>
            <span class="text-xs font-bold font-mono"
              :class="stopTrackerState === 'idle_wait' ? 'text-amber-300' : 'text-amber-200/50'">{{ microparadasCount }}</span>
          </div>
          <div v-if="stopTrackerState === 'idle_wait'" class="flex items-center gap-1 mt-0.5">
            <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse shrink-0"></span>
            <span class="text-lg font-bold font-mono text-amber-300 tabular-nums leading-none">{{ fmtMMSS(idleDurationS) }}</span>
          </div>
          <div v-else class="text-[10px] font-mono text-gray-600 mt-0.5">--:--</div>
        </div>

        <!-- Paradas -->
        <div class="flex flex-col items-center px-3 py-1 rounded-lg border min-w-[88px]"
          :class="stopTrackerState === 'stop_open' ? 'border-red-500/70 bg-red-950/50' : 'border-slate-700/60 bg-slate-800/40'">
          <div class="flex items-center gap-1.5 w-full justify-between">
            <span class="text-[9px] font-semibold uppercase tracking-widest"
              :class="stopTrackerState === 'stop_open' ? 'text-red-400' : 'text-gray-500'">Paradas</span>
            <span class="text-xs font-bold font-mono"
              :class="stopTrackerState === 'stop_open' ? 'text-red-300' : 'text-red-200/50'">{{ paradasCount }}</span>
          </div>
          <div v-if="stopTrackerState === 'stop_open'" class="flex items-center gap-1 mt-0.5">
            <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse shrink-0"></span>
            <span class="text-lg font-bold font-mono text-red-300 tabular-nums leading-none">{{ fmtMMSS(idleDurationS) }}</span>
          </div>
          <div v-else class="text-[10px] font-mono text-gray-600 mt-0.5">--:--</div>
        </div>

      </div>

      <!-- Turno actual -->
      <span v-if="turnoNombre" class="text-xs font-mono text-amber-400 bg-amber-950/40 border border-amber-700/40 rounded px-2 py-0.5">
        Turno: {{ turnoNombre }}
      </span>

      <!-- Ir a Config -->
      <button @click="$router.push('/config/' + deviceId)"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs text-gray-400 hover:text-white hover:bg-slate-700 transition-colors">
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
        </svg>
        Configurar
      </button>
    </div>

    <!-- Body principal -->
    <div class="flex flex-1 overflow-hidden">

      <!-- ── Panel izquierdo: métricas ──────────────────────────────────── -->
      <div class="w-72 bg-slate-900 border-r border-slate-700/60 flex flex-col overflow-hidden shrink-0">

        <!-- Contadores principales -->
        <div class="grid grid-cols-2 gap-px border-b border-slate-700/60">
          <!-- Detecciones hoy -->
          <div class="bg-slate-900 p-4 flex flex-col items-center text-center">
            <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-1">Hoy</p>
            <p class="text-4xl font-bold font-mono"
              :class="flashToday ? 'text-emerald-300' : 'text-white'"
              style="transition: color .3s">{{ todayCount }}</p>
            <p class="text-[10px] text-gray-600 mt-0.5">detecciones</p>
          </div>
          <!-- Detecciones sesión (desde que se abrió la página) -->
          <div class="bg-slate-900/60 p-4 flex flex-col items-center text-center">
            <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-1">Sesión</p>
            <p class="text-4xl font-bold font-mono"
              :class="flashSession ? 'text-emerald-300' : 'text-slate-300'"
              style="transition: color .3s">{{ sessionCount }}</p>
            <p class="text-[10px] text-gray-600 mt-0.5">nuevas</p>
          </div>
        </div>

        <!-- Estado del detector (mini-FSM JS, igual que Lab) -->
        <div class="p-4 border-b border-slate-700/60">
          <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-2">Detector</p>
          <div class="flex items-center gap-3 text-xs font-mono flex-wrap">
            <span class="px-2 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wider"
              :class="{
                'bg-blue-900/60 text-blue-300':    jsFsmState === 'DETECTING',
                'bg-slate-800 text-gray-400':       jsFsmState === 'IDLE',
                'bg-orange-900/60 text-orange-300': jsFsmState === 'WAIT_EXIT',
                'bg-violet-900/60 text-violet-300': jsFsmState === 'COOLDOWN',
              }">
              FSM: {{ jsFsmState }}
            </span>
            <span class="text-gray-400">
              <span :class="beigeRatio >= (thresholds?.beige ?? 0.35) ? 'text-emerald-300' : 'text-white'">
                {{ Math.round(beigeRatio * 100) }}%
              </span>
              <span class="text-gray-600"> / min {{ Math.round((thresholds?.beige ?? 0.35) * 100) }}%</span>
            </span>
          </div>
        </div>

        <!-- Último evento detectado -->
        <div class="p-4 border-b border-slate-700/60">
          <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-2">Último evento</p>
          <template v-if="lastEvent">
            <div class="flex items-start gap-2">
              <!-- dot pulsante cuando es reciente (<30s) -->
              <span class="w-2 h-2 rounded-full mt-1 shrink-0"
                :class="lastEventAge < 30 ? 'bg-emerald-400 animate-pulse' : 'bg-slate-500'"></span>
              <div>
                <p class="text-xs font-mono text-white">{{ lastEvent.event_type }}</p>
                <p class="text-[11px] text-gray-400 mt-0.5">{{ formatRelative(lastEvent.timestamp) }}</p>
                <p class="text-[10px] font-mono text-gray-600 mt-0.5">{{ formatTime(lastEvent.timestamp) }}</p>
              </div>
            </div>
          </template>
          <p v-else class="text-xs text-gray-600 italic">Sin eventos registrados hoy</p>
        </div>

        <!-- ROI configurado -->
        <div class="p-4 border-b border-slate-700/60">
          <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-2">ROI activo</p>
          <template v-if="roi && roi.length >= 4">
            <div class="grid grid-cols-2 gap-x-3 gap-y-1 text-xs font-mono">
              <div><span class="text-gray-500">X: </span><span class="text-white">{{ roi[0] }}</span></div>
              <div><span class="text-gray-500">Y: </span><span class="text-white">{{ roi[1] }}</span></div>
              <div><span class="text-gray-500">W: </span><span class="text-white">{{ roi[2] }}</span></div>
              <div><span class="text-gray-500">H: </span><span class="text-white">{{ roi[3] }}</span></div>
            </div>
            <p v-if="roi[4] > 0" class="text-[10px] text-gray-500 mt-1 font-mono">Margen inferior: {{ roi[4] }}px</p>
          </template>
          <p v-else class="text-xs text-gray-600 italic">Sin ROI configurado</p>
        </div>

        <!-- Umbrales de histéresis -->
        <div class="p-4 border-b border-slate-700/60">
          <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-2">Histéresis</p>
          <div v-if="thresholds" class="flex items-center gap-3 text-xs font-mono">
            <span class="text-emerald-300">Alto {{ (thresholds.high || 0).toFixed(2) }}</span>
            <span class="text-red-300">Bajo {{ (thresholds.low || 0).toFixed(2) }}</span>
          </div>
          <p v-else class="text-xs text-gray-600 italic">—</p>
        </div>

        <!-- Feed de eventos recientes -->
        <div class="flex-1 min-h-0 p-4 flex flex-col">
          <p class="text-[10px] font-semibold text-gray-500 uppercase tracking-widest mb-2">Eventos recientes</p>
          <div class="flex-1 min-h-0 overflow-y-auto space-y-1 scrollbar-thin scrollbar-thumb-slate-600">
            <div v-if="recentEvents.length === 0" class="text-xs text-gray-600 italic">Sin eventos hoy</div>
            <div v-for="ev in recentEvents" :key="ev.event_id || ev.id"
              class="flex items-center gap-2 py-1 border-b border-slate-800/60 group">
              <span class="w-1.5 h-1.5 rounded-full shrink-0"
                :class="ev.event_type?.includes('stop') ? 'bg-orange-400' : 'bg-blue-400'"></span>
              <div class="min-w-0 flex-1">
                <p class="text-[11px] font-mono text-white truncate">{{ ev.event_type }}</p>
                <p class="text-[10px] text-gray-500">{{ formatTime(ev.timestamp) }}</p>
              </div>
              <span v-if="ev.payload?.score != null"
                class="text-[10px] font-mono text-gray-500 shrink-0">
                {{ Number(ev.payload.score).toFixed(2) }}
              </span>
            </div>
          </div>
        </div>

      </div>

      <!-- ── Panel derecho: cámara con overlay ROI ──────────────────────── -->
      <div class="flex-1 bg-black flex flex-col overflow-hidden">

        <!-- Barra superior de cámara -->
        <div class="flex items-center gap-3 px-4 py-2 bg-slate-900/80 border-b border-slate-700/60 shrink-0">
          <span class="flex items-center gap-1.5 text-xs font-mono"
            :class="cameraStreaming ? 'text-red-400' : 'text-gray-500'">
            <span class="w-2 h-2 rounded-full"
              :class="cameraStreaming ? 'bg-red-500 animate-pulse' : 'bg-slate-600'"></span>
            {{ cameraStreaming ? 'EN VIVO' : 'INICIANDO' }}
          </span>
          <span v-if="frameNaturalSize" class="text-xs font-mono text-gray-500">
            {{ frameNaturalSize }}
          </span>
          <div class="flex-1"></div>
          <!-- Estado del stop tracker (fuente: Python detector) -->
          <span class="flex items-center gap-1.5 text-xs font-medium px-2 py-0.5 rounded"
            :class="{
              'bg-emerald-900/50 text-emerald-300': stopTrackerState === 'producing',
              'bg-amber-900/50  text-amber-300':    stopTrackerState === 'idle_wait',
              'bg-red-900/50    text-red-300':       stopTrackerState === 'stop_open',
            }">
            <span class="w-1.5 h-1.5 rounded-full"
              :class="{
                'bg-emerald-400 animate-pulse': stopTrackerState === 'producing',
                'bg-amber-400 animate-pulse':   stopTrackerState === 'idle_wait',
                'bg-red-400 animate-pulse':     stopTrackerState === 'stop_open',
              }"></span>
            <span v-if="stopTrackerState === 'producing'">Produciendo</span>
            <span v-else-if="stopTrackerState === 'idle_wait'">Microparada · {{ idleDurationS.toFixed(0) }}s</span>
            <span v-else>Parada · {{ idleDurationS.toFixed(0) }}s</span>
          </span>
          <!-- Flash de detección activa -->
          <transition name="det-flash">
            <span v-if="showDetectionFlash"
              class="text-xs font-semibold text-emerald-300 bg-emerald-900/50 border border-emerald-600/50 px-2 py-0.5 rounded">
              ⚡ Detección
            </span>
          </transition>
        </div>

        <!-- Camera + canvas overlay -->
        <div ref="cameraContainer" class="flex-1 relative overflow-hidden flex items-center justify-center bg-black">

          <!-- Loading / sin cámara placeholder -->
          <div v-if="!cameraStreaming"
            class="absolute inset-0 flex flex-col items-center justify-center text-gray-600 gap-2 z-10 pointer-events-none">
            <template v-if="noCameraConfigured">
              <svg class="w-10 h-10 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                  d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/>
              </svg>
              <p class="text-sm text-slate-500">Sin cámara configurada</p>
              <p class="text-xs text-slate-600">Configura la URL en Configuración &rarr; Cámara</p>
            </template>
            <template v-else>
              <svg class="w-8 h-8 animate-spin text-slate-600" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              <p class="text-xs">Conectando con cámara...</p>
            </template>
          </div>

          <!-- MJPEG display canvas (fetch-based) -->
          <canvas
            v-if="configReady && cameraUrlConfig"
            ref="streamCanvas"
            class="block max-w-full max-h-full w-auto h-auto object-contain"
            :class="cameraStreaming ? 'opacity-100' : 'opacity-0'"
            style="transition: opacity .3s; align-self: center;"
          ></canvas>

          <!-- Canvas overlay para el ROI (readonly) -->
          <canvas
            ref="roiCanvas"
            class="absolute pointer-events-none"
            :style="canvasStyle"
          ></canvas>

          <!-- Borde pulsante cuando hay detección activa -->
          <div v-if="showDetectionFlash"
            class="absolute inset-0 border-4 border-emerald-400 rounded pointer-events-none"
            style="box-shadow: 0 0 24px 4px rgba(52,211,153,0.25)"></div>

        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { configService, gatewayService, detectorService } from '../services/api.js'

// ── Props ────────────────────────────────────────────────────────────────────
const props = defineProps({
  deviceIdProp: { type: String, default: '' },
  isNew:        { type: Boolean, default: false },
})

// ── Route ───────────────────────────────────────────────────────────────────
const route    = useRoute()
const deviceId = ref(props.deviceIdProp || route.params.deviceId || '')

// ── Config / ROI — cargados exclusivamente desde edge-config-service ─────────
const roi          = ref(null)   // [x, y, w, h, margin?]
const roiPresencia = ref(null)   // [x, y, w, h] o null
const thresholds   = ref(null)

// ── Estado del detector Python (paradas y estado global) ────────────────────
const isDetecting       = ref(false)   // d.detecting (Python)
const presenceMotion    = ref(false)   // calculado localmente por _analyzePresence
const fusionScore       = ref(0.0)     // d.fusion_score
const beigeRatio        = ref(0.0)     // calculado localmente por _analyzeBeige
const motionScore       = ref(0.0)     // calculado localmente por _analyzePresence
const stopTrackerState  = ref('producing')

// ── Mini-FSM JS (copia exacta del Lab — espeja event_fsm.py) ─────────────────
const jsFsmState  = ref('IDLE')    // 'IDLE'|'DETECTING'|'WAIT_EXIT'|'COOLDOWN'
const fsmConfig   = ref(null)      // cfg.fsm — mismos params que usa Python
let   _confirmN   = 0
let   _exitLowN   = 0
let   _exitHighN  = 0
let   _waitExitN  = 0
let   _cooldownN  = 0

const idleDurationS     = ref(0)
let   _prevStopState    = 'producing'
let   _maxIdleDurationS = 0
let   _idleStartTime    = null   // ms timestamp — para medir duró parada en JS
let   _motionFrames     = 0      // frames consecutivos CON movimiento
let   _noMotionFrames   = 0      // frames consecutivos SIN movimiento
const MOTION_CONFIRM    = 8      // frames para confirmar movimiento (~0.25s a 30fps)
const NOMOTION_CONFIRM  = 20     // frames para confirmar parada (~0.6s a 30fps)

// ── Estado de servicios ───────────────────────────────────────────────────────
const detectorOk     = ref(false)
const cameraOk       = ref(false)
const cloudConnected = ref(false)
const turnoNombre    = ref('')

// ── Contadores ────────────────────────────────────────────────────────────────
const todayCount      = ref(0)
const sessionCount    = ref(0)
const flashToday      = ref(false)
const flashSession    = ref(false)
const microparadasCount = ref(0)
const paradasCount      = ref(0)

// ── Eventos ───────────────────────────────────────────────────────────────────
const recentEvents    = ref([])
const lastEvent       = ref(null)
const lastEventAge    = ref(9999)
const sessionOpenedAt = new Date()

// ── Flash de detección ────────────────────────────────────────────────────────
const showDetectionFlash = ref(false)
let   flashTimer         = null

// ── Cámara ───────────────────────────────────────────────────────────────────
const cameraUrlConfig    = ref('')
const configReady        = ref(false)
const cameraStreaming    = ref(false)
const noCameraConfigured = ref(false)
const frameNaturalSize   = ref('')
const streamCanvas       = ref(null)
const roiCanvas          = ref(null)
const cameraContainer    = ref(null)
const canvasStyle        = ref({ left: '0px', top: '0px', width: '0px', height: '0px' })
let   _offscreenCanvas   = null

// ── Intervalos ────────────────────────────────────────────────────────────────
let pollInterval       = null
let statusInterval     = null
let ageInterval        = null
let idleTickerInterval = null
let sseSource          = null
let _statusInterval    = null
let resizeObserver     = null

// Tracking del estado del detector Python
let microStopMaxS       = 120
let _prevDetectorState  = 'producing'
let _detectorMaxIdle    = 0
let _detectorLastPollAt = 0
let _detectorLastIdleS  = 0

// ── Buffers de análisis local (espeja lógica del Lab) ─────────────────────────
let _presenceBuffer = []
let _presenceN      = 0

// ── Formatters ────────────────────────────────────────────────────────────────
function fmtMMSS(secs) {
  const s = Math.floor(secs)
  const m = Math.floor(s / 60)
  return `${String(m).padStart(2,'0')}:${String(s % 60).padStart(2,'0')}`
}

function formatTime(ts) {
  if (!ts) return '—'
  return new Date(ts).toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatRelative(ts) {
  if (!ts) return '—'
  const diff = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (diff < 60)   return `hace ${diff}s`
  if (diff < 3600) return `hace ${Math.floor(diff / 60)}min`
  return `hace ${Math.floor(diff / 3600)}h`
}

function midnightLocal() {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d
}

// ── Flash helper ──────────────────────────────────────────────────────────────
function triggerFlash() {
  showDetectionFlash.value = true
  flashSession.value       = true
  flashToday.value         = true
  clearTimeout(flashTimer)
  flashTimer = setTimeout(() => {
    showDetectionFlash.value = false
    flashSession.value       = false
    flashToday.value         = false
  }, 1500)
}

// ── Cargar config — ÚNICA fuente: edge-config-service ────────────────────────
async function loadConfig() {
  try {
    if (!deviceId.value || deviceId.value === '__new__') {
      const devices = await configService.getDevices()
      if (devices && devices.length > 0) deviceId.value = devices[0].id
    }
    if (!deviceId.value || deviceId.value === '__new__' || props.isNew) {
      configReady.value      = true
      noCameraConfigured.value = true
      return
    }
    const cfg = await configService.getConfig(deviceId.value)
    roi.value             = cfg.roi          ?? null
    thresholds.value      = cfg.thresholds   ?? null
    fsmConfig.value       = cfg.fsm          ?? null
    cameraUrlConfig.value = cfg.camera?.url  ?? cfg.camera_url ?? ''

    // Umbral microparada desde config OEE
    const ms = cfg.oee?.micro_stop_max_s
    if (typeof ms === 'number' && ms > 0) microStopMaxS = ms

    // ROI de presencia
    const rp = cfg.roi_presencia
    if (rp && typeof rp === 'object') {
      roiPresencia.value = Array.isArray(rp)
        ? (rp.length >= 4 ? rp : null)
        : (rp.width > 0 && rp.height > 0 ? [rp.x, rp.y, rp.width, rp.height] : null)
    } else {
      roiPresencia.value = null
    }
    _roiDrawn = false
    nextTick(() => drawRoi())
  } catch { /* non-fatal */ }
  configReady.value = true
  if (!cameraUrlConfig.value) noCameraConfigured.value = true
}

// ── Cargar eventos del día ────────────────────────────────────────────────────
async function loadEvents() {
  try {
    const since  = midnightLocal().toISOString()
    const events = await fetch(
      `/api/gateway/edge/events/recent?limit=200&since=${encodeURIComponent(since)}`
    ).then(r => r.json())
    if (!Array.isArray(events)) return

    events.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))

    const prevCount = todayCount.value
    todayCount.value   = events.length
    recentEvents.value = events.slice(0, 30)
    lastEvent.value    = events[0] ?? null
    // sessionCount ahora lo maneja la mini-FSM JS en tiempo real
    if (events.length > prevCount) {
      flashToday.value = true
      clearTimeout(flashTimer)
      flashTimer = setTimeout(() => { flashToday.value = false }, 1500)
    }
  } catch { /* non-fatal */ }
}

// ── Estado de servicios ───────────────────────────────────────────────────────
async function loadStatus() {
  const [statusRes, detectorRes, turnoRes] = await Promise.allSettled([
    gatewayService.getStatus(),
    detectorService.getHealth(),
    fetch('/api/gateway/edge/current-turno').then(r => r.json()),
  ])
  if (statusRes.status === 'fulfilled')  cloudConnected.value = statusRes.value.cloud_connected ?? false
  if (detectorRes.status === 'fulfilled') {
    detectorOk.value = detectorRes.value.status === 'ok'
    cameraOk.value   = detectorRes.value.camera  ?? false
  }
  if (turnoRes.status === 'fulfilled') turnoNombre.value = turnoRes.value?.nombre ?? ''
}

// ── SSE para refreshar eventos en tiempo real ─────────────────────────────────
function startSSE() {
  if (sseSource) return
  sseSource = new EventSource('/api/gateway/edge/stream')
  sseSource.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data)
      if (data.type === 'stop_created' || data.type === 'stop_closed') loadEvents()
    } catch { /* ignore */ }
  }
}

// ── Poll detector Python — fuente de verdad para stop tracker ────────────────
async function pollDetectorStatus() {
  try {
    const r = await fetch('/api/detector/status', { signal: AbortSignal.timeout(1000) })
    if (!r.ok) return
    const d = await r.json()
    isDetecting.value = !!d.detecting
    fusionScore.value = d.fusion_score ?? 0.0

    const newState    = d.stop_tracker_state ?? null
    const newIdleSecs = typeof d.idle_duration_s === 'number' ? d.idle_duration_s : null

    if (newState) {
      _detectorLastPollAt = Date.now()  // Python respondió — bloquear JS fallback
      // Transición producing → idle_wait: contar microparada al instante
      if (newState === 'idle_wait' && _prevDetectorState === 'producing') {
        microparadasCount.value++
      }
      // Transición idle_wait → stop_open: mover de microparada a parada
      if (newState === 'stop_open' && _prevDetectorState === 'idle_wait') {
        microparadasCount.value--
        paradasCount.value++
      }
      // Reset del seguimiento JS al volver a produciendo
      if (newState === 'producing' && _prevDetectorState !== 'producing') {
        _idleStartTime = null
      }
      _prevDetectorState     = newState
      stopTrackerState.value = newState
      if (newIdleSecs !== null) {
        idleDurationS.value  = newIdleSecs
        _detectorLastIdleS   = newIdleSecs
      }
    }
  } catch { /* timeout o red — mantener estado anterior */ }
}

function startStatusPolling() {
  pollDetectorStatus()
  _statusInterval = setInterval(pollDetectorStatus, 500)
}

function stopStatusPolling() {
  clearInterval(_statusInterval)
  _statusInterval = null
}

// ── Helpers de análisis visual (copiados del Lab) ───────────────────────────
function rgbToHsv(r, g, b) {
  r /= 255; g /= 255; b /= 255
  const mx = Math.max(r, g, b), mn = Math.min(r, g, b), d = mx - mn
  let h = 0
  if (d !== 0) {
    if      (mx === r) h = ((g - b) / d + (g < b ? 6 : 0)) / 6
    else if (mx === g) h = ((b - r) / d + 2) / 6
    else               h = ((r - g) / d + 4) / 6
  }
  return [h * 360, d === 0 ? 0 : d / mx * 255, mx * 255]
}

// Mini-FSM JS — copia exacta de labAnalyzeFrame() FSM block (espeja event_fsm.py)
function _runFsm(beigePct) {
  const nFrames    = fsmConfig.value?.n_frames             ?? 3
  const cooldown   = fsmConfig.value?.cooldown             ?? 8
  const exitFrames = fsmConfig.value?.exit_frames          ?? 5
  const maxWait    = fsmConfig.value?.max_wait_exit_frames ?? 750
  const highThr    = thresholds.value?.high  ?? thresholds.value?.beige ?? 0.35
  const lowThr     = thresholds.value?.low   ?? (highThr * 0.5)
  const aboveHigh  = beigePct >= highThr
  const belowLow   = beigePct <  lowThr

  if (jsFsmState.value === 'IDLE') {
    if (aboveHigh) { _confirmN = 1; jsFsmState.value = 'DETECTING' }
  } else if (jsFsmState.value === 'DETECTING') {
    if (aboveHigh) {
      _confirmN++
      if (_confirmN >= nFrames) {
        sessionCount.value++
        showDetectionFlash.value = true
        flashSession.value = true
        clearTimeout(flashTimer)
        flashTimer = setTimeout(() => {
          showDetectionFlash.value = false
          flashSession.value = false
        }, 1500)
        jsFsmState.value = 'WAIT_EXIT'
        _confirmN = 0; _exitLowN = 0; _waitExitN = 0
      }
    } else if (belowLow) {
      _confirmN = 0; jsFsmState.value = 'IDLE'
    }
  } else if (jsFsmState.value === 'WAIT_EXIT') {
    _waitExitN++
    if (belowLow) {
      _exitLowN++; _exitHighN = 0
    } else {
      _exitHighN++
      if (_exitHighN >= exitFrames) { _exitLowN = 0; _exitHighN = 0 }
    }
    const exitConfirmed = _exitLowN >= exitFrames
    const timedOut = _waitExitN >= maxWait && belowLow
    if (exitConfirmed || timedOut) {
      jsFsmState.value = 'COOLDOWN'
      _cooldownN = cooldown; _exitLowN = 0; _exitHighN = 0; _waitExitN = 0
    }
  } else if (jsFsmState.value === 'COOLDOWN') {
    _cooldownN--
    if (_cooldownN <= 0) jsFsmState.value = 'IDLE'
  }
}

// Análisis de beige en el ROI de conteo — misma lógica que labAnalyzeFrame()
function _analyzeBeige(sc) {
  if (!sc || !roi.value || roi.value.length < 4) return
  const [rx, ry, rw, rh, rMargin = 0] = roi.value
  const activeH = Math.max(1, rh - rMargin)
  const crx = Math.max(0, Math.min(rx, sc.width  - 1))
  const cry = Math.max(0, Math.min(ry, sc.height - 1))
  const crw = Math.max(1, Math.min(rw, sc.width  - crx))
  const crh = Math.max(1, Math.min(activeH, sc.height - cry))
  try {
    const data = sc.getContext('2d', { willReadFrequently: true }).getImageData(crx, cry, crw, crh).data
    const br   = thresholds.value?.beige_range ?? {}
    const hMin = br.h_min ?? 15, hMax = br.h_max ?? 35
    const sMin = br.s_min ?? 30, sMax = br.s_max ?? 130
    const vMin = br.v_min ?? 120
    let beigePx = 0
    const n = crw * crh
    for (let i = 0; i < data.length; i += 4) {
      const [h, s, v] = rgbToHsv(data[i], data[i + 1], data[i + 2])
      if (h >= hMin && h <= hMax && s >= sMin && s <= sMax && v >= vMin) beigePx++
    }
    const ratio = n > 0 ? beigePx / n : 0.0
    beigeRatio.value = ratio
    _runFsm(ratio)
  } catch { /* ignorar */ }
}

// Stop tracker JS — fallback cuando Python no responde (>3s sin poll exitoso)
function _updateStopTracker(rawMotion) {
  if (rawMotion) {
    _motionFrames++
    _noMotionFrames = 0
  } else {
    _noMotionFrames++
    _motionFrames = 0
  }

  const confirmedMotion   = _motionFrames   >= MOTION_CONFIRM
  const confirmedNoMotion = _noMotionFrames >= NOMOTION_CONFIRM

  // Python respondió hace <10s → es la fuente de verdad, JS no toca el estado
  const pythonFresh   = _detectorLastPollAt > 0 && (Date.now() - _detectorLastPollAt) < 10_000
  // Python alguna vez respondió con estado de parada → JS NUNCA puede setear 'producing'
  const pythonStopped = _detectorLastPollAt > 0 && _prevDetectorState !== 'producing'

  if (confirmedMotion) {
    _idleStartTime = null
    if (pythonFresh)   return  // Python reciente: no tocar
    if (pythonStopped) return  // Python dijo parada: solo Python puede reanudar
    _maxIdleDurationS      = 0
    stopTrackerState.value = 'producing'
    idleDurationS.value    = 0
    _idleStartTime         = null
  } else if (confirmedNoMotion) {
    if (!_idleStartTime) _idleStartTime = Date.now()
    if (pythonFresh) return
    const elapsed          = (Date.now() - _idleStartTime) / 1000
    idleDurationS.value    = elapsed
    _maxIdleDurationS      = Math.max(_maxIdleDurationS, elapsed)
    const prevJsState      = stopTrackerState.value
    const nextJsState      = elapsed < microStopMaxS ? 'idle_wait' : 'stop_open'
    if (prevJsState === 'producing' && nextJsState === 'idle_wait') {
      microparadasCount.value++
    }
    if (prevJsState === 'idle_wait' && nextJsState === 'stop_open') {
      microparadasCount.value--
      paradasCount.value++
    }
    stopTrackerState.value = nextJsState
  }
}

// Análisis de presencia (slow-window diff) — espeja _labUpdatePresence()
function _analyzePresence(sc) {
  if (!sc || !roiPresencia.value || roiPresencia.value.length < 4) return
  const [px, py, pw, ph] = roiPresencia.value
  if (pw < 8 || ph < 8) { presenceMotion.value = false; return }
  const x0 = Math.max(0, px), y0 = Math.max(0, py)
  const x1 = Math.min(sc.width,  px + pw)
  const y1 = Math.min(sc.height, py + ph)
  const cw = x1 - x0, ch = y1 - y0
  if (cw < 8 || ch < 8) { presenceMotion.value = false; return }
  try {
    const data = sc.getContext('2d', { willReadFrequently: true }).getImageData(x0, y0, cw, ch).data
    const n    = cw * ch
    const gray = new Uint8Array(n)
    for (let i = 0; i < n; i++) gray[i] = (data[i*4] + data[i*4+1] + data[i*4+2]) / 3
    if (_presenceN !== n) { _presenceBuffer = []; _presenceN = n }
    const PIXEL_THR   = thresholds.value?.presencia_pixel_threshold ?? 8
    const MOTION_AREA = thresholds.value?.presencia_scale           ?? 0.005
    const WIN         = Math.max(5, Math.round((thresholds.value?.presencia_window ?? 375) * 0.08))
    let diffSum = 0, motionPx = 0
    if (_presenceBuffer.length >= WIN) {
      const ref = _presenceBuffer[0]
      for (let i = 0; i < n; i++) {
        const d = Math.abs(gray[i] - ref[i])
        diffSum += d
        if (d > PIXEL_THR) motionPx++
      }
      motionScore.value    = diffSum / n   // Δpx promedio (igual que labMotionScore)
      const rawMotion      = (motionPx / n) >= MOTION_AREA
      presenceMotion.value = rawMotion
      _updateStopTracker(rawMotion)
    } else {
      // Buffer aún calentando: no cambiar estado hasta tener datos
      _updateStopTracker(false)
    }
    _presenceBuffer.push(gray)
    if (_presenceBuffer.length > WIN) _presenceBuffer.shift()
  } catch { /* ignorar */ }
}

// ── Dibujar ROI overlay — mismo estilo que el Lab ────────────────────────────
let _roiDrawn = false

function drawRoi() {
  const sc        = streamCanvas.value
  const canvas    = roiCanvas.value
  const container = cameraContainer.value
  if (!sc || !canvas || !container || !roi.value || roi.value.length < 4) return

  const rect     = sc.getBoundingClientRect()
  const contRect = container.getBoundingClientRect()
  const left   = rect.left - contRect.left
  const top    = rect.top  - contRect.top
  const width  = rect.width
  const height = rect.height

  canvas.width  = width
  canvas.height = height
  canvasStyle.value = { left: left + 'px', top: top + 'px', width: width + 'px', height: height + 'px' }

  const ctx = canvas.getContext('2d')
  ctx.clearRect(0, 0, width, height)

  const natW   = sc.width  || 1280
  const natH   = sc.height || 720
  const scaleX = width  / natW
  const scaleY = height / natH

  const [rx, ry, rw, rh, rMargin = 0] = roi.value
  const sx = rx * scaleX
  const sy = ry * scaleY
  const sw = rw * scaleX
  const sh = rh * scaleY

  // ── ROI de conteo ─────────────────────────────────────────────────────────
  const beige      = beigeRatio.value          // 0-1 del Python
  const beigeThresh = thresholds.value?.beige ?? 0.35
  const detecting  = isDetecting.value
  const aboveThresh = beige >= beigeThresh

  // Borde: verde si supera umbral, rojo si no (igual que Lab)
  ctx.strokeStyle = aboveThresh ? '#22c55e' : '#ef4444'
  ctx.lineWidth   = aboveThresh ? 2.5 : 2
  ctx.strokeRect(sx, sy, sw, sh)

  // Barra de beige% en la parte superior del ROI
  const barH    = 6
  const beigeW  = Math.round(beige * sw)
  const threshX = sx + Math.round(beigeThresh * sw)

  ctx.fillStyle = 'rgba(0,0,0,0.65)'
  ctx.fillRect(sx, sy - barH - 2, sw, barH + 2)
  ctx.fillStyle = aboveThresh ? 'rgba(34,197,94,0.9)' : 'rgba(251,146,36,0.85)'
  ctx.fillRect(sx, sy - barH - 1, beigeW, barH)

  // Línea vertical de umbral
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth   = 1.5
  ctx.setLineDash([3, 2])
  ctx.beginPath()
  ctx.moveTo(threshX, sy - barH - 3)
  ctx.lineTo(threshX, sy + 1)
  ctx.stroke()
  ctx.setLineDash([])

  // Score text: "XX% / min YY%"
  const beigeLabel = `${Math.round(beige * 100)}% / min ${Math.round(beigeThresh * 100)}%`
  ctx.fillStyle = aboveThresh ? '#86efac' : '#fca5a5'
  ctx.font      = 'bold 11px monospace'
  ctx.fillText(beigeLabel, sx + 4, sy + 14)

  // Badge "Detectando" en esquina superior derecha si está activo
  if (detecting) {
    ctx.font = 'bold 10px monospace'
    const cw = ctx.measureText('Detectando').width + 10
    ctx.fillStyle = 'rgba(30,58,138,0.85)'
    ctx.fillRect(sx + sw - cw - 2, sy + 2, cw, 16)
    ctx.fillStyle = '#93c5fd'
    ctx.fillText('Detectando', sx + sw - cw + 3, sy + 13)
  }

  // Corner accents
  const cLen = Math.min(12, sw * 0.15, sh * 0.3)
  const cColor = aboveThresh ? '#22c55e' : '#ef4444'
  ctx.lineWidth   = 3
  ctx.strokeStyle = cColor
  ctx.beginPath(); ctx.moveTo(sx, sy + cLen); ctx.lineTo(sx, sy); ctx.lineTo(sx + cLen, sy); ctx.stroke()
  ctx.beginPath(); ctx.moveTo(sx + sw - cLen, sy); ctx.lineTo(sx + sw, sy); ctx.lineTo(sx + sw, sy + cLen); ctx.stroke()
  ctx.beginPath(); ctx.moveTo(sx, sy + sh - cLen); ctx.lineTo(sx, sy + sh); ctx.lineTo(sx + cLen, sy + sh); ctx.stroke()
  ctx.beginPath(); ctx.moveTo(sx + sw - cLen, sy + sh); ctx.lineTo(sx + sw, sy + sh); ctx.lineTo(sx + sw, sy + sh - cLen); ctx.stroke()

  // Línea de margen
  if (rMargin > 0) {
    const mY = (ry + rh - rMargin) * scaleY
    ctx.strokeStyle = 'rgba(251, 191, 36, 0.7)'
    ctx.lineWidth   = 1
    ctx.setLineDash([4, 4])
    ctx.beginPath(); ctx.moveTo(sx, mY); ctx.lineTo(sx + sw, mY); ctx.stroke()
    ctx.setLineDash([])
  }

  // ── ROI de presencia ──────────────────────────────────────────────────────
  if (roiPresencia.value && roiPresencia.value.length >= 4) {
    const [px, py, pw, ph] = roiPresencia.value
    const psx = px * scaleX
    const psy = py * scaleY
    const psw = pw * scaleX
    const psh = ph * scaleY

    const motion  = presenceMotion.value
    const motSc   = motionScore.value   // 0-1 (fracción de píxeles que cambiaron)
    const pColor  = motion ? '#34d399' : '#f87171'

    ctx.fillStyle = motion ? 'rgba(52,211,153,0.08)' : 'rgba(248,113,113,0.08)'
    ctx.fillRect(psx, psy, psw, psh)
    ctx.strokeStyle = pColor
    ctx.lineWidth   = motion ? 2 : 1.5
    ctx.setLineDash([6, 3])
    ctx.strokeRect(psx, psy, psw, psh)
    ctx.setLineDash([])

    // Badge estado
    const pLabel = motion ? 'En movimiento' : 'Detenida'
    ctx.font = 'bold 10px monospace'
    const textW = ctx.measureText(pLabel).width + 10
    ctx.fillStyle = motion ? 'rgba(6,78,59,0.85)' : 'rgba(69,10,10,0.85)'
    ctx.fillRect(psx + 2, psy + 2, textW, 16)
    ctx.fillStyle = pColor
    ctx.fillText(pLabel, psx + 6, psy + 13)

    // Barra de movimiento en la parte inferior del ROI de presencia (Δpx igual que Lab)
    const barW = Math.min(psw, Math.round(Math.min(motSc / 20, 1) * psw))
    ctx.fillStyle = 'rgba(0,0,0,0.6)'
    ctx.fillRect(psx, psy + psh - 8, psw, 8)
    ctx.fillStyle = motion ? 'rgba(52,211,153,0.9)' : 'rgba(248,113,113,0.7)'
    ctx.fillRect(psx, psy + psh - 7, barW, 6)
    ctx.fillStyle = motion ? '#86efac' : '#fca5a5'
    ctx.font = 'bold 9px monospace'
    ctx.fillText(`mv ${motSc.toFixed(1)}`, psx + 4, psy + psh - 1)
  }
}

// ── Stream MJPEG (fetch loop con multipart parsing) ──────────────────────────
let _streamAbort = null
let _streamRetryTimer = null

function startStream() {
  if (_streamAbort) return
  _streamAbort = new AbortController()
  _runMjpegStream(_streamAbort.signal)
}

async function _runMjpegStream(signal) {
  try {
    const host = location.hostname
    const resp = await fetch(`http://${host}:8001/stream`, { signal })
    if (!resp.ok) {
      cameraStreaming.value = false
      _streamRetryTimer = setTimeout(() => startStream(), 2000)
      return
    }

    const reader = resp.body.getReader()
    let buffer = new Uint8Array()
    const boundary = '--frame'

    while (!signal.aborted) {
      const { done, value } = await reader.read()
      if (done) break

      // Concatenar nuevo chunk al buffer
      const newBuffer = new Uint8Array(buffer.length + value.length)
      newBuffer.set(buffer)
      newBuffer.set(value, buffer.length)
      buffer = newBuffer

      // Buscar boundary
      const boundaryIndex = _findBoundary(buffer, boundary)
      if (boundaryIndex === -1) continue

      // Buscar inicio de JPEG (FF D8)
      let jpegStart = -1
      for (let i = boundaryIndex; i < buffer.length - 1; i++) {
        if (buffer[i] === 0xFF && buffer[i + 1] === 0xD8) {
          jpegStart = i
          break
        }
      }
      if (jpegStart === -1) continue

      // Buscar fin de JPEG (FF D9)
      let jpegEnd = -1
      for (let i = jpegStart + 2; i < buffer.length - 1; i++) {
        if (buffer[i] === 0xFF && buffer[i + 1] === 0xD9) {
          jpegEnd = i + 2
          break
        }
      }
      if (jpegEnd === -1) continue

      // Extraer JPEG completo
      const jpegData = buffer.slice(jpegStart, jpegEnd)
      buffer = buffer.slice(jpegEnd)

      // Renderizar frame
      try {
        const blob = new Blob([jpegData], { type: 'image/jpeg' })
        const bm = await createImageBitmap(blob)
        const sc = streamCanvas.value
        if (sc && !signal.aborted) {
          if (sc.width !== bm.width || sc.height !== bm.height) {
            sc.width = bm.width
            sc.height = bm.height
            frameNaturalSize.value = `${bm.width} × ${bm.height}`
            if (!cameraStreaming.value) {
              cameraStreaming.value = true
              cameraOk.value = true
              nextTick(drawRoi)
            }
          }
          sc.getContext('2d', { willReadFrequently: true }).drawImage(bm, 0, 0)
          _analyzeBeige(sc)
          _analyzePresence(sc)
          drawRoi()
        }
        bm.close()
      } catch (e) {
        console.warn('Frame decode error:', e)
      }
    }
  } catch (err) {
    if (!signal.aborted) {
      console.warn('Stream error:', err)
      cameraStreaming.value = false
      _streamRetryTimer = setTimeout(() => {
        if (_streamAbort) startStream()
      }, 2000)
    }
  }
}

function _findBoundary(buffer, boundary) {
  const boundaryBytes = new TextEncoder().encode(boundary)
  for (let i = 0; i <= buffer.length - boundaryBytes.length; i++) {
    let match = true
    for (let j = 0; j < boundaryBytes.length; j++) {
      if (buffer[i + j] !== boundaryBytes[j]) {
        match = false
        break
      }
    }
    if (match) return i
  }
  return -1
}

function stopStream() {
  clearTimeout(_streamRetryTimer)
  _streamRetryTimer = null
  if (_streamAbort) {
    _streamAbort.abort()
    _streamAbort = null
  }
  cameraStreaming.value = false
}

// ── Actualizar edad del último evento ────────────────────────────────────────
function updateAge() {
  if (lastEvent.value) lastEventAge.value = Math.floor((Date.now() - new Date(lastEvent.value.timestamp)) / 1000)
}

// ── Ticker de parada: interpola idleDurationS cada 1s entre polls HTTP ────────
function tickIdle() {
  if (_detectorLastPollAt > 0 && stopTrackerState.value !== 'producing') {
    const interpolated = _detectorLastIdleS + (Date.now() - _detectorLastPollAt) / 1000
    idleDurationS.value = interpolated
    _detectorMaxIdle    = Math.max(_detectorMaxIdle, interpolated)
  }
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────
onMounted(async () => {
  await Promise.all([loadConfig(), loadStatus(), loadEvents()])
  pollInterval       = setInterval(loadEvents,  8_000)
  statusInterval     = setInterval(loadStatus, 15_000)
  ageInterval        = setInterval(updateAge,   1_000)
  idleTickerInterval = setInterval(tickIdle,    1_000)
  startSSE()
  startStream()
  startStatusPolling()
  resizeObserver = new ResizeObserver(() => { drawRoi() })
  if (cameraContainer.value) resizeObserver.observe(cameraContainer.value)
})

onBeforeUnmount(() => {
  clearInterval(pollInterval)
  clearInterval(statusInterval)
  clearInterval(ageInterval)
  clearInterval(idleTickerInterval)
  clearTimeout(flashTimer)
  stopStatusPolling()
  stopStream()
  if (sseSource) { sseSource.close(); sseSource = null }
  if (resizeObserver) resizeObserver.disconnect()
})
</script>

<style scoped>
.card { @apply bg-slate-800 rounded-lg shadow p-6; }
.det-flash-enter-active, .det-flash-leave-active { transition: opacity .25s ease, transform .25s ease; }
.det-flash-enter-from, .det-flash-leave-to { opacity: 0; transform: scale(0.9); }
</style>
