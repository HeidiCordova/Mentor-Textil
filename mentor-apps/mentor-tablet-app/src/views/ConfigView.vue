<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'
import { useConfigStore } from '@/stores/config'
import { useConnectionStore } from '@/stores/connection'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { getBaseURL, setCloudLineaId, api, setApiMode, getApiMode } from '@/services/api'

const connection = useConnectionStore()
const configStore = useConfigStore()
const pl = usePlantasLineasStore()

// ── Modo de línea ──────────────────────────────────────────────────────────
const MODE_PRESETS = {
  textil:   { micro_stop_max_s: 120,  stop_max_s: 1800, snapshot_interval_s: 1800, vel_unit: 'uh', vel_nominal_us: 0.00833 },
  botellas: { micro_stop_max_s: 210,  stop_max_s: 300,  snapshot_interval_s: 300,  vel_unit: 'us', vel_nominal_us: 0.5 },
} as const

type ModoKey = keyof typeof MODE_PRESETS

const modoActivo = computed<ModoKey | null>(() => {
  const c = configStore.config as Record<string, unknown>
  const mode = c.mode as string | undefined
  if (mode === 'textil' || mode === 'botellas') return mode
  return null
})

const savingMode = ref(false)
const confirmModo = ref<ModoKey | null>(null)

function pedirConfirmacion(modo: ModoKey): void {
  if (modoActivo.value === modo) return
  confirmModo.value = modo
}

function cancelarConfirmacion(): void {
  confirmModo.value = null
}

async function confirmarCambioModo(): Promise<void> {
  const modo = confirmModo.value
  if (!modo) return
  confirmModo.value = null
  savingMode.value = true
  const preset = MODE_PRESETS[modo]
  const currentOee = (configStore.config.oee as Record<string, unknown>) ?? {}
  const patch = {
    mode: modo,
    oee: { ...currentOee, ...preset },
  }
  await configStore.updateConfig(patch)
  // Si el cloud también está accesible (HYBRID), sincronizar el modo allá
  if (connection.mode === 'HYBRID' && pl.selectedLineaId) {
    try {
      const prevMode = getApiMode()
      setApiMode('CLOUD')
      setCloudLineaId(pl.selectedLineaId)
      await api.updateConfig({ mode: modo })
      setApiMode(prevMode)
    } catch {
      // sincronización cloud no crítica, continuar
    }
  }
  savingMode.value = false
}

// ── Edición ────────────────────────────────────────────────────────────────
const editMode = ref(false)
const saving = ref(false)

interface RoiBuf    { x: number; y: number; width: number; height: number }
interface ThreshBuf { edge: number; color: number; flow: number; dy: number; beige: number; high: number; low: number }
interface FsmBuf    { n_frames: number; cooldown: number; exit_frames: number; max_wait_exit_frames: number }

const roiBuf    = ref<RoiBuf>({ x: 120, y: 60, width: 320, height: 200 })
const threshBuf = ref<ThreshBuf>({ edge: 0.4, color: 0.6, flow: 0.5, dy: 5.0, beige: 0.35, high: 0.7, low: 0.3 })
const fsmBuf    = ref<FsmBuf>({ n_frames: 3, cooldown: 8, exit_frames: 5, max_wait_exit_frames: 50 })

function loadBuffers(): void {
  const c = configStore.config as Record<string, unknown>
  const roi = c.roi as Record<string, number> | undefined
  if (roi) {
    roiBuf.value = { x: roi.x ?? 120, y: roi.y ?? 60, width: roi.width ?? 320, height: roi.height ?? 200 }
  }
  const th = c.thresholds as Record<string, number> | undefined
  if (th) {
    threshBuf.value = {
      edge: th.edge ?? 0.4, color: th.color ?? 0.6, flow: th.flow ?? 0.5,
      dy: th.dy ?? 5.0, beige: th.beige ?? 0.35, high: th.high ?? 0.7, low: th.low ?? 0.3,
    }
  }
  const fsm = c.fsm as Record<string, number> | undefined
  if (fsm) {
    fsmBuf.value = {
      n_frames: fsm.n_frames ?? 3, cooldown: fsm.cooldown ?? 8,
      exit_frames: fsm.exit_frames ?? 5, max_wait_exit_frames: fsm.max_wait_exit_frames ?? 50,
    }
  }
}

function startEdit(): void {
  loadBuffers()
  editMode.value = true
}

function cancelEdit(): void {
  editMode.value = false
}

async function saveConfig(): Promise<void> {
  saving.value = true
  const patch = {
    roi: [roiBuf.value.x, roiBuf.value.y, roiBuf.value.width, roiBuf.value.height],
    thresholds: { ...threshBuf.value },
    fsm: { ...fsmBuf.value },
  }
  const success = await configStore.updateConfig(patch)
  saving.value = false
  if (success) editMode.value = false
}

// ── Calibración ─────────────────────────────────────────────────────────────
const calibrating = ref(false)
const calibProgress = ref(0)
const calibDone = ref(false)
let calibPollInterval: ReturnType<typeof setInterval> | null = null

async function handleCalibration(): Promise<void> {
  calibrating.value = true
  calibDone.value = false
  calibProgress.value = 0
  await configStore.startCalibration()
  calibPollInterval = setInterval(pollCalibProgress, 800)
}

async function pollCalibProgress(): Promise<void> {
  try {
    const base = getBaseURL()
    const res = await fetch(`${base}/edge/calibration/status`)
    if (!res.ok) return
    const data = await res.json() as { active: boolean; progress: number }
    calibProgress.value = Math.round(data.progress * 100)
    if (!data.active) {
      calibrating.value = false
      calibDone.value = true
      if (calibPollInterval) clearInterval(calibPollInterval)
      calibPollInterval = null
    }
  } catch { /* ignora si no hay respuesta */ }
}

onUnmounted(() => {
  if (calibPollInterval) clearInterval(calibPollInterval)
})

onMounted(async () => {
  if (connection.isCloudOnly) return
  await configStore.fetchConfig()
  loadBuffers()
})
</script>

<template>
  <div class="flex flex-col h-full p-4">

    <!-- Header -->
    <div class="flex items-center justify-between mb-4 shrink-0">
      <div>
        <h2 class="text-base font-semibold text-edge-100">Configuracion de Linea</h2>
        <p class="text-xs text-edge-500 mt-0.5">Version: {{ configStore.version }}</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="!editMode"
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-edge-700/50 text-edge-200 hover:bg-edge-600/50 transition-colors"
          @click="startEdit"
        >
          <SvgIcon name="settings" :size="14" />
          Editar
        </button>
        <template v-else>
          <button
            class="px-3 py-1.5 text-xs font-medium rounded-lg border border-edge-700/50 text-edge-400 hover:text-edge-200 transition-colors"
            @click="cancelEdit"
          >
            Cancelar
          </button>
          <button
            class="flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-lg bg-production-active text-white hover:bg-blue-600 transition-colors disabled:opacity-50"
            :disabled="saving"
            @click="saveConfig"
          >
            <SvgIcon v-if="saving" name="refresh" :size="14" class="animate-spin" />
            <SvgIcon v-else name="check" :size="14" />
            Guardar
          </button>
        </template>
      </div>
    </div>

    <div v-if="configStore.loading" class="flex items-center justify-center py-12">
      <SvgIcon name="refresh" :size="24" class="animate-spin text-edge-500" />
    </div>

    <!-- Modo cloud: configuración del dispositivo no disponible -->
    <div v-else-if="connection.isCloudOnly" class="flex-1 flex flex-col items-center justify-center gap-4 text-center px-6">
      <div class="w-16 h-16 rounded-full bg-blue-500/10 border border-blue-500/20 flex items-center justify-center">
        <SvgIcon name="wifi" :size="28" class="text-blue-400" />
      </div>
      <div>
        <h3 class="text-sm font-semibold text-edge-200 mb-1">Configuración solo disponible en modo Edge</h3>
        <p class="text-xs text-edge-500 leading-relaxed max-w-xs">
          Los parámetros de visión (ROI, umbrales, cámara, calibración) y el modo de línea
          se configuran directamente en el dispositivo Jetson cuando hay conexión directa al edge.
        </p>
      </div>
      <div class="px-4 py-3 rounded-xl bg-edge-800 border border-edge-700/40 text-xs text-edge-400 w-full max-w-xs">
        <div class="flex justify-between mb-1">
          <span class="text-edge-500">Modo actual</span>
          <span class="text-blue-400 font-semibold">☁ CLOUD</span>
        </div>
        <div class="flex justify-between">
          <span class="text-edge-500">Operador</span>
          <span class="text-edge-300">{{ connection.operatorId }}</span>
        </div>
      </div>
    </div>

    <div v-else class="flex-1 min-h-0 overflow-y-auto space-y-4">

      <!-- ── Modo de línea ───────────────────────────────────────────────── -->
      <div class="rounded-xl bg-edge-800 border border-edge-700/50 p-4">
        <h3 class="text-sm font-semibold text-edge-100 mb-1">Modo de linea</h3>
        <p class="text-xs text-edge-400 mb-3">
          Ajusta los tiempos de microparada, parada y la resolucion del timeline segun el tipo de proceso.
        </p>
        <div class="flex gap-3">
          <button
            v-for="modo in (['textil', 'botellas'] as const)"
            :key="modo"
            class="flex-1 py-2.5 px-3 rounded-lg text-xs font-semibold border transition-colors disabled:opacity-50"
            :class="modoActivo === modo
              ? 'bg-production-active/20 border-production-active text-production-active'
              : 'bg-edge-700/30 border-edge-700/50 text-edge-300 hover:bg-edge-700/60'"
            :disabled="savingMode"
            @click="pedirConfirmacion(modo)"
          >
            <span class="block capitalize">{{ modo }}</span>
            <span class="block text-[10px] font-normal opacity-70 mt-0.5">
              {{ modo === 'textil' ? 'Micro 2min · Snapshot 30min' : 'Micro 3.5min · Snapshot 5min' }}
            </span>
          </button>
        </div>
        <div v-if="savingMode" class="flex items-center gap-1.5 mt-2 text-xs text-edge-400">
          <SvgIcon name="refresh" :size="12" class="animate-spin" />
          Aplicando modo...
        </div>
      </div>

      <!-- ── ROI ──────────────────────────────────────────────────────────── -->
      <div class="rounded-xl bg-edge-800 border border-edge-700/50 p-4">
        <h3 class="text-sm font-semibold text-edge-100 mb-3">Zona de deteccion (ROI)</h3>
        <p class="text-xs text-edge-400 mb-3">
          El rectangulo amarillo sobre la imagen muestra la zona activa. Ajusta los valores y vera el cambio en tiempo real.
        </p>
        <div class="grid grid-cols-2 gap-3">
          <div v-for="field in (['x','y','width','height'] as const)" :key="field">
            <label class="block text-xs text-edge-400 mb-1">{{ field === 'x' ? 'X (izquierda)' : field === 'y' ? 'Y (arriba)' : field === 'width' ? 'Ancho' : 'Alto' }}</label>
            <input
              type="number" min="0"
              :value="roiBuf[field]"
              :disabled="!editMode"
              class="w-full px-2 py-1.5 text-sm rounded border bg-edge-900 text-edge-200 focus:outline-none transition-colors"
              :class="editMode ? 'border-edge-600 focus:border-production-active' : 'border-edge-700/40 opacity-60'"
              @input="roiBuf[field] = Number(($event.target as HTMLInputElement).value)"
            />
          </div>
        </div>
      </div>

      <!-- ── Umbrales de detección ─────────────────────────────────────────── -->
      <div class="rounded-xl bg-edge-800 border border-edge-700/50 p-4">
        <h3 class="text-sm font-semibold text-edge-100 mb-3">Umbrales de deteccion</h3>
        <div class="space-y-3">
          <div v-for="item in [
            { key: 'high',  label: 'Umbral ALTO (score > X = detectando)',  min: 0.01, max: 1, step: 0.01 },
            { key: 'low',   label: 'Umbral BAJO (score < X = sin objeto)',   min: 0.01, max: 1, step: 0.01 },
            { key: 'flow',  label: 'Gate de flujo (flow < X → score = 0)',   min: 0.01, max: 1, step: 0.01 },
            { key: 'dy',    label: 'Velocidad vertical minima (px/frame)',    min: 0.5, max: 50, step: 0.5  },
            { key: 'edge',  label: 'Peso bordes (Canny)',                    min: 0.01, max: 1, step: 0.01 },
            { key: 'color', label: 'Peso cambio de color (histograma)',      min: 0.01, max: 1, step: 0.01 },
            { key: 'beige', label: 'Peso cobertura beige',                   min: 0.01, max: 1, step: 0.01 },
          ] as const" :key="item.key">
            <div class="flex items-center justify-between">
              <label class="text-xs text-edge-400">{{ item.label }}</label>
              <span class="text-xs font-mono text-edge-300 ml-2 w-10 text-right">{{ threshBuf[item.key] }}</span>
            </div>
            <input
              type="range"
              :min="item.min" :max="item.max" :step="item.step"
              :value="threshBuf[item.key]"
              :disabled="!editMode"
              class="w-full h-1.5 rounded-full appearance-none bg-edge-700 accent-yellow-400 disabled:opacity-50"
              @input="threshBuf[item.key] = Number(($event.target as HTMLInputElement).value)"
            />
          </div>
        </div>
      </div>

      <!-- ── FSM ──────────────────────────────────────────────────────────── -->
      <div class="rounded-xl bg-edge-800 border border-edge-700/50 p-4">
        <h3 class="text-sm font-semibold text-edge-100 mb-1">Maquina de estados (FSM)</h3>
        <p class="text-xs text-edge-400 mb-3">Controla cuantos frames consecutivos se necesitan para confirmar un corte y evitar doble conteo.</p>
        <div class="grid grid-cols-2 gap-3">
          <div v-for="item in [
            { key: 'n_frames',            label: 'Frames para confirmar' },
            { key: 'cooldown',            label: 'Frames de espera post-corte' },
            { key: 'exit_frames',         label: 'Frames bajos para salida' },
            { key: 'max_wait_exit_frames',label: 'Timeout salida (frames)' },
          ] as const" :key="item.key">
            <div>
              <label class="block text-xs text-edge-400 mb-1">{{ item.label }}</label>
              <input
                type="number" min="1"
                :value="fsmBuf[item.key]"
                :disabled="!editMode"
                class="w-full px-2 py-1.5 text-sm rounded border bg-edge-900 text-edge-200 focus:outline-none transition-colors"
                :class="editMode ? 'border-edge-600 focus:border-production-active' : 'border-edge-700/40 opacity-60'"
                @input="fsmBuf[item.key] = Number(($event.target as HTMLInputElement).value)"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- ── Calibración ──────────────────────────────────────────────────── -->
      <div class="rounded-xl bg-edge-800 border border-edge-700/50 p-4">
        <h3 class="text-sm font-semibold text-edge-100 mb-2">Calibracion de referencia de color</h3>

        <!-- Instrucción pre-calibración -->
        <div class="flex items-start gap-2 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/30 mb-3">
          <SvgIcon name="alert" :size="14" class="text-yellow-400 mt-0.5 shrink-0" />
          <p class="text-xs text-yellow-300 leading-relaxed">
            <strong>Antes de calibrar:</strong> asegurate de que <strong>NO haya tela en la zona ROI</strong>.
            El sistema capturara 30 frames del fondo libre para aprender el color de referencia.
            Cuando el material pase, el cambio de color se detectara correctamente.
          </p>
        </div>

        <!-- Barra de progreso (solo durante calibración) -->
        <div v-if="calibrating" class="mb-3">
          <div class="flex items-center justify-between mb-1">
            <span class="text-xs text-edge-400">Capturando frames...</span>
            <span class="text-xs font-mono text-edge-300">{{ calibProgress }}%</span>
          </div>
          <div class="w-full h-2 rounded-full bg-edge-700 overflow-hidden">
            <div
              class="h-full rounded-full bg-yellow-400 transition-all duration-300"
              :style="{ width: calibProgress + '%' }"
            />
          </div>
        </div>

        <!-- Confirmación éxito -->
        <div v-if="calibDone && !calibrating" class="flex items-center gap-2 p-2 rounded-lg bg-green-500/10 border border-green-500/30 mb-3">
          <SvgIcon name="check" :size="13" class="text-green-400" />
          <span class="text-xs text-green-300">Calibracion completada. La referencia de color fue actualizada.</span>
        </div>

        <button
          class="flex items-center gap-1.5 px-4 py-2 text-xs font-medium rounded-lg bg-yellow-500/10 text-yellow-400 border border-yellow-500/30 hover:bg-yellow-500/20 transition-colors disabled:opacity-50"
          :disabled="calibrating"
          @click="handleCalibration"
        >
          <SvgIcon v-if="calibrating" name="refresh" :size="14" class="animate-spin" />
          <SvgIcon v-else name="refresh" :size="14" />
          {{ calibrating ? `Calibrando... ${calibProgress}%` : 'Iniciar Calibracion' }}
        </button>
      </div>

    </div>
  </div>

  <!-- ── Modal confirmación cambio de modo ─────────────────────────────── -->
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="confirmModo"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
        @click.self="cancelarConfirmacion"
      >
        <div class="w-72 rounded-2xl bg-edge-800 border border-edge-700/50 shadow-2xl p-5">
          <h4 class="text-sm font-semibold text-edge-100 mb-1">
            Cambiar a modo <span class="capitalize text-production-active">{{ confirmModo }}</span>
          </h4>
          <p class="text-xs text-edge-400 mb-4 leading-relaxed">
            Esto sobreescribira los tiempos de microparada, parada y el intervalo del timeline
            con los valores del preset
            <span class="font-semibold text-edge-200 capitalize">{{ confirmModo }}</span>.
          </p>
          <!-- Resumen del preset -->
          <div class="rounded-lg bg-edge-900 border border-edge-700/40 p-3 mb-4 text-xs space-y-1 text-edge-300">
            <div class="flex justify-between">
              <span>Microparada hasta</span>
              <span class="font-mono text-edge-100">{{ confirmModo === 'textil' ? '2 min' : '3.5 min' }}</span>
            </div>
            <div class="flex justify-between">
              <span>Parada desde</span>
              <span class="font-mono text-edge-100">{{ confirmModo === 'textil' ? '2 min' : '3.5 min' }}</span>
            </div>
            <div class="flex justify-between">
              <span>Snapshot / Timeline</span>
              <span class="font-mono text-edge-100">{{ confirmModo === 'textil' ? '30 min' : '5 min' }}</span>
            </div>
            <div class="flex justify-between">
              <span>Unidad velocidad</span>
              <span class="font-mono text-edge-100">{{ confirmModo === 'textil' ? 'u/h' : 'u/s' }}</span>
            </div>
          </div>
          <div class="flex gap-2">
            <button
              class="flex-1 py-2 text-xs font-medium rounded-lg border border-edge-700/50 text-edge-400 hover:text-edge-200 transition-colors"
              @click="cancelarConfirmacion"
            >Cancelar</button>
            <button
              class="flex-1 py-2 text-xs font-semibold rounded-lg bg-production-active text-white hover:bg-blue-600 transition-colors"
              @click="confirmarCambioModo"
            >Aplicar</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
