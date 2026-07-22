<template>
  <div class="space-y-6 max-w-3xl">

    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white">Programar medidor</h1>
      <p class="mt-0.5 text-sm text-gray-400">Escritura directa de parametros via Modbus al dispositivo fisico</p>
    </div>

    <!-- Aviso -->
    <div class="flex items-start gap-3 rounded-xl border border-amber-900/40 bg-amber-950/20 px-4 py-3">
      <svg class="w-4 h-4 mt-0.5 shrink-0 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
      </svg>
      <p class="text-sm text-amber-300">
        Estas operaciones escriben directamente al hardware. Verifique los valores antes de aplicar.
        Un parametro incorrecto puede afectar las mediciones o deshabilitar el dispositivo.
      </p>
    </div>

    <!-- Selector de medidor -->
    <div class="card space-y-4">
      <h2 class="section-title">
        <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M13 10V3L4 14h7v7l9-11h-7z"/>
        </svg>
        Seleccionar medidor
      </h2>

      <div class="flex flex-wrap gap-2">
        <button v-for="m in meters" :key="m.id"
          @click="selectMeter(m)"
          :class="['px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors',
            selectedMeter?.id === m.id
              ? 'bg-blue-600 border-blue-500 text-white'
              : 'bg-slate-800 border-slate-700 text-gray-400 hover:border-slate-500 hover:text-gray-200']">
          {{ m.meter_id }}
          <span class="font-mono text-[10px] opacity-60 ml-1">UID:{{ m.unit_id }}</span>
        </button>
        <p v-if="!meters.length" class="text-xs text-gray-500 italic">No hay medidores configurados</p>
      </div>

      <div v-if="selectedMeter" class="flex items-center gap-3 pt-1">
        <button @click="loadDeviceConfig" :disabled="loadingCfg"
          class="btn-primary flex items-center gap-2 text-sm disabled:opacity-40">
          <svg v-if="loadingCfg" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          Leer config actual
        </button>
        <p v-if="loadError" class="text-xs text-red-400">{{ loadError }}</p>
      </div>

      <!-- Resumen config actual -->
      <div v-if="deviceCfg" class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1">
        <div v-for="item in cfgSummary" :key="item.label"
          class="px-3 py-2 rounded-lg bg-slate-900/60 border border-slate-700/40">
          <p class="text-[10px] text-gray-500 uppercase tracking-wide">{{ item.label }}</p>
          <p class="text-xs font-medium mt-0.5" :class="item.color || 'text-gray-200'">{{ item.value }}</p>
        </div>
      </div>
    </div>

    <!-- Secciones de escritura -->
    <template v-if="selectedMeter">

      <!-- TC -->
      <div class="card space-y-4">
        <h2 class="section-title">
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z"/>
          </svg>
          Transformador de corriente (TC)
        </h2>

        <div v-if="deviceCfg?.ct_mode === 'Rogowski'"
          class="flex items-start gap-2 rounded-lg border border-amber-900/40 bg-amber-950/20 px-3 py-2.5">
          <svg class="w-4 h-4 mt-0.5 shrink-0 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          </svg>
          <p class="text-xs text-amber-300 leading-relaxed">
            Sensor detectado: <strong>Rogowski (Rcoil)</strong>. Para este tipo de sensor la Relacion TC en
            los parametros de red debe mantenerse en <strong>1.0000</strong>. Modificarla causara errores
            en las mediciones de corriente. Los parametros del sensor se configuran mediante ME231 Assistant
            en la seccion IABC Calibrate.
          </p>
        </div>

        <div class="flex items-end gap-4 flex-wrap">
          <div>
            <label class="label">Primario (A)</label>
            <input v-model.number="ct.primary" type="number" min="1" class="input-field w-32" placeholder="200"/>
          </div>
          <div>
            <label class="label">Secundario (A)</label>
            <input v-model.number="ct.secondary" type="number" min="1" class="input-field w-32" placeholder="5"/>
          </div>
          <div class="pb-1 min-w-[100px]">
            <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">Relacion calculada</p>
            <p class="text-sm font-mono" :class="ctRatioValid ? 'text-blue-300' : 'text-gray-600'">
              {{ ctRatioValid ? (ct.primary / ct.secondary).toFixed(4) : '—' }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-3 pt-1">
          <button @click="requestCT" :disabled="busy.ct || !ctRatioValid" class="write-btn disabled:opacity-40">
            <svg v-if="busy.ct" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ busy.ct ? 'Aplicando...' : 'Aplicar TC' }}
          </button>
          <span v-if="results.ct" :class="['inline-flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-md',
            results.ct.ok ? 'text-green-300 bg-green-900/30' : 'text-red-300 bg-red-900/30']">
            <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path v-if="results.ct.ok" stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
              <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            {{ results.ct.msg }}
          </span>
        </div>
      </div>

      <!-- Parametros del sistema -->
      <div class="card space-y-4">
        <h2 class="section-title">
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          Parametros del sistema
        </h2>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="label">Modo de cableado</label>
            <select v-model.number="sys.wiring" class="input-field">
              <option v-if="sys.wiring === 0" :value="0" disabled>3P4W_4CT — Trifasico 4 hilos, 4 TC (solo lectura)</option>
              <option :value="1">3P4W_3CT — Trifasico 4 hilos, 3 TC</option>
              <option :value="2">3P3W_3CT — Trifasico 3 hilos, 3 TC</option>
              <option :value="3">3P3W_2CT — Trifasico 3 hilos, 2 TC</option>
              <option :value="4">1P3W — Monofasico 3 hilos</option>
              <option :value="5">1P2W — Monofasico 2 hilos</option>
            </select>
            <p v-if="sys.wiring === 0" class="mt-1 text-[11px] text-amber-400">El modo 3P4W_4CT no puede escribirse via instruccion 1001. Seleccione otro modo antes de aplicar.</p>
          </div>
          <div>
            <label class="label">Frecuencia de red (Hz)</label>
            <select v-model.number="sys.freq" class="input-field">
              <option :value="50">50 Hz</option>
              <option :value="60">60 Hz</option>
            </select>
          </div>
          <div>
            <label class="label">Tension nominal (V)</label>
            <input v-model.number="sys.nominal" type="number" min="1" class="input-field" placeholder="220"/>
          </div>
          <div>
            <label class="label">Relacion TP — VT ratio</label>
            <input v-model.number="sys.vt" type="number" min="0.0001" step="0.0001" class="input-field" placeholder="1.0"/>
          </div>
        </div>

        <div class="flex items-center gap-3 pt-1">
          <button @click="requestSys" :disabled="busy.sys" class="write-btn disabled:opacity-40">
            <svg v-if="busy.sys" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ busy.sys ? 'Aplicando...' : 'Aplicar sistema' }}
          </button>
          <span v-if="results.sys" :class="['inline-flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-md',
            results.sys.ok ? 'text-green-300 bg-green-900/30' : 'text-red-300 bg-red-900/30']">
            <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path v-if="results.sys.ok" stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
              <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            {{ results.sys.msg }}
          </span>
        </div>
      </div>

      <!-- Direccion de corriente -->
      <div class="card space-y-4">
        <h2 class="section-title">
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"/>
          </svg>
          Direccion de corriente
        </h2>
        <p class="text-xs text-gray-500 -mt-2">
          Invertir cuando el TC esta instalado al reves. Normal = Positivo.
        </p>

        <div class="flex flex-wrap gap-5">
          <div v-for="phase in ['l1','l2','l3']" :key="phase">
            <p class="label mb-2">Fase {{ phase.toUpperCase() }}</p>
            <div class="flex rounded-lg overflow-hidden border border-slate-700 text-xs font-medium">
              <button
                @click="dir[phase] = 'pos'"
                :class="['px-4 py-2 transition-colors',
                  dir[phase] === 'pos' ? 'bg-green-700 text-white' : 'bg-slate-800 text-gray-400 hover:bg-slate-700']">
                Positivo
              </button>
              <button
                @click="dir[phase] = 'rev'"
                :class="['px-4 py-2 transition-colors border-l border-slate-700',
                  dir[phase] === 'rev' ? 'bg-amber-700 text-white' : 'bg-slate-800 text-gray-400 hover:bg-slate-700']">
                Reverso
              </button>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-3 pt-1">
          <button @click="requestDir" :disabled="busy.dir" class="write-btn disabled:opacity-40">
            <svg v-if="busy.dir" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ busy.dir ? 'Aplicando...' : 'Aplicar direccion' }}
          </button>
          <span v-if="results.dir" :class="['inline-flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-md',
            results.dir.ok ? 'text-green-300 bg-green-900/30' : 'text-red-300 bg-red-900/30']">
            <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path v-if="results.dir.ok" stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
              <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            {{ results.dir.msg }}
          </span>
        </div>
      </div>

      <!-- Mantenimiento -->
      <div class="card space-y-5">
        <h2 class="section-title">
          <svg class="w-4 h-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          Mantenimiento
        </h2>

        <div class="space-y-2">
          <p class="text-xs font-medium text-gray-300">Sincronizar reloj del dispositivo</p>
          <p class="text-xs text-gray-500">Escribe la hora actual UTC al medidor. Util tras un corte de alimentacion.</p>
          <div class="flex items-center gap-3">
            <button @click="requestTime" :disabled="busy.time" class="write-btn disabled:opacity-40">
              <svg v-if="busy.time" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              {{ busy.time ? 'Sincronizando...' : 'Sincronizar reloj' }}
            </button>
            <span v-if="results.time" :class="['inline-flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-md',
              results.time.ok ? 'text-green-300 bg-green-900/30' : 'text-red-300 bg-red-900/30']">
              <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path v-if="results.time.ok" stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
                <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
              </svg>
              {{ results.time.msg }}
            </span>
          </div>
        </div>

        <div class="border-t border-slate-700/60 pt-4 space-y-2">
          <p class="text-xs font-medium text-gray-300">Resetear contadores</p>
          <p class="text-xs text-gray-500">Esta accion es irreversible. Los valores de energia acumulada se perderan.</p>
          <div class="flex items-center gap-3 flex-wrap">
            <select v-model="resetType" class="input-field w-44">
              <option value="energy">Energia</option>
              <option value="demand">Demanda</option>
              <option value="maxmin">Max / Min</option>
              <option value="tariff">Tarifas</option>
              <option value="all">Todo</option>
            </select>
            <button @click="requestReset" :disabled="busy.reset"
              class="px-4 py-2 rounded-lg text-xs font-medium text-red-400 border border-red-900/40 hover:bg-red-900/20 transition-colors disabled:opacity-40">
              <svg v-if="busy.reset" class="inline w-3.5 h-3.5 animate-spin mr-1 -mt-0.5" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              {{ busy.reset ? 'Reseteando...' : 'Resetear' }}
            </button>
            <span v-if="results.reset" :class="['inline-flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-md',
              results.reset.ok ? 'text-green-300 bg-green-900/30' : 'text-red-300 bg-red-900/30']">
              <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path v-if="results.reset.ok" stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
                <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12"/>
              </svg>
              {{ results.reset.msg }}
            </span>
          </div>
        </div>
      </div>

    </template>

    <!-- Modal de confirmacion de escritura -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="pendingWrite" class="fixed inset-0 z-50 flex items-center justify-center">
          <div class="absolute inset-0 bg-black/60" @click="pendingWrite = null"></div>
          <div class="relative z-10 w-full max-w-sm mx-4 bg-slate-900 border border-slate-700 rounded-2xl p-6 shadow-2xl">
            <div class="flex items-start gap-3 mb-5">
              <svg class="w-5 h-5 mt-0.5 shrink-0 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
              </svg>
              <div class="space-y-1">
                <p class="text-sm font-semibold text-white">{{ pendingWrite.label }}</p>
                <p class="text-xs text-gray-400 leading-relaxed">{{ pendingWrite.description }}</p>
                <div v-if="pendingWrite.detail"
                  class="mt-2 text-xs text-gray-300 bg-slate-800/80 border border-slate-700/40 rounded-lg px-3 py-2 font-mono break-all">
                  {{ pendingWrite.detail }}
                </div>
              </div>
            </div>
            <div class="flex justify-end gap-2">
              <button @click="pendingWrite = null"
                class="px-4 py-2 rounded-lg text-xs text-gray-400 border border-slate-700 hover:bg-slate-800 transition-colors">
                Cancelar
              </button>
              <button @click="confirmWrite"
                class="px-4 py-2 rounded-lg text-xs font-semibold text-white bg-amber-600 hover:bg-amber-500 transition-colors">
                Confirmar
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Toast global -->
    <Teleport to="body">
      <Transition name="toast">
        <div v-if="toast"
          :class="['fixed bottom-6 right-6 z-50 px-4 py-3 rounded-xl shadow-2xl text-sm font-medium flex items-center gap-2',
            toast.ok
              ? 'bg-green-900/90 border border-green-700/50 text-green-200'
              : 'bg-red-900/90 border border-red-700/50 text-red-200']">
          <svg v-if="toast.ok" class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
          </svg>
          <svg v-else class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
          {{ toast.msg }}
        </div>
      </Transition>
    </Teleport>

  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getMeters, getMeterDeviceConfig, meterSetCT, meterSetSys, meterSetDir, meterSetTime, meterReset } from '../services/api.js'

const meters        = ref([])
const selectedMeter = ref(null)
const deviceCfg     = ref(null)
const loadingCfg    = ref(false)
const loadError     = ref('')
const pendingWrite  = ref(null)
const resetType     = ref('energy')
const toast         = ref(null)

const ct  = ref({ primary: 200, secondary: 5 })
const sys = ref({ wiring: 1, freq: 50, nominal: 220, vt: 1.0 })
const dir = ref({ l1: 'pos', l2: 'pos', l3: 'pos' })
const busy    = ref({ ct: false, sys: false, dir: false, time: false, reset: false })
const results = ref({ ct: null, sys: null, dir: null, time: null, reset: null })

const ctRatioValid = computed(() => ct.value.primary > 0 && ct.value.secondary > 0)

const cfgSummary = computed(() => {
  if (!deviceCfg.value) return []
  const c = deviceCfg.value
  return [
    { label: 'Cableado',         value: c.wiring_name },
    { label: 'Frecuencia',       value: `${c.freq} Hz` },
    { label: 'Tension nominal',  value: `${c.nominal_v} V` },
    { label: 'Relacion TC',      value: c.ct_ratio.toFixed(4) },
    { label: 'Tipo sensor',      value: c.ct_mode, color: c.ct_mode === 'Rogowski' ? 'text-amber-400' : 'text-gray-200' },
    { label: 'L1', value: c.dir_l1 === 'rev' ? 'Reverso' : 'Positivo', color: c.dir_l1 === 'rev' ? 'text-amber-400' : 'text-green-400' },
    { label: 'L2', value: c.dir_l2 === 'rev' ? 'Reverso' : 'Positivo', color: c.dir_l2 === 'rev' ? 'text-amber-400' : 'text-green-400' },
    { label: 'L3', value: c.dir_l3 === 'rev' ? 'Reverso' : 'Positivo', color: c.dir_l3 === 'rev' ? 'text-amber-400' : 'text-green-400' },
    { label: 'Dir. Modbus',      value: c.slave_addr },
  ]
})

let toastTimer = null
function showToast(ok, msg) {
  clearTimeout(toastTimer)
  toast.value = { ok, msg }
  toastTimer = setTimeout(() => { toast.value = null }, 4000)
}

function setResult(key, ok, msg) {
  results.value[key] = { ok, msg }
  setTimeout(() => { results.value[key] = null }, 6000)
}

function requestWrite(label, description, detail, action) {
  pendingWrite.value = { label, description, detail, action }
}

function confirmWrite() {
  if (!pendingWrite.value) return
  const action = pendingWrite.value.action
  pendingWrite.value = null
  action()
}

function requestCT() {
  requestWrite(
    'Aplicar Transformador de Corriente',
    'Escribe la relacion TC al hardware del medidor. Afecta directamente las mediciones de corriente.',
    `Primario: ${ct.value.primary} A  /  Secundario: ${ct.value.secondary} A  →  Relacion: ${(ct.value.primary / ct.value.secondary).toFixed(4)}`,
    applySetCT
  )
}

function requestSys() {
  const wNames = { 0: '3P4W_4CT', 1: '3P4W_3CT', 2: '3P3W_3CT', 3: '3P3W_2CT', 4: '1P3W', 5: '1P2W' }
  requestWrite(
    'Aplicar Parametros del Sistema',
    'Escribe cableado, frecuencia, tension nominal y ratio TP al hardware.',
    `Cableado: ${wNames[sys.value.wiring] ?? sys.value.wiring}  /  Frecuencia: ${sys.value.freq} Hz  /  Vnominal: ${sys.value.nominal} V  /  VT: ${sys.value.vt}`,
    applySetSys
  )
}

function requestDir() {
  const label = v => v === 'rev' ? 'Reverso' : 'Positivo'
  requestWrite(
    'Aplicar Direccion de Corriente',
    'Escribe la configuracion de inversion de fases al hardware.',
    `L1: ${label(dir.value.l1)}  /  L2: ${label(dir.value.l2)}  /  L3: ${label(dir.value.l3)}`,
    applySetDir
  )
}

function requestTime() {
  requestWrite(
    'Sincronizar Reloj del Dispositivo',
    'Escribe la hora actual de Lima (UTC-5) al medidor. La operacion es segura pero modifica el reloj interno.',
    new Date().toLocaleString('es-PE', { timeZone: 'America/Lima', hour12: false }),
    applySetTime
  )
}

function requestReset() {
  const labels = { energy: 'Energia', demand: 'Demanda', maxmin: 'Max/Min', tariff: 'Tarifas', all: 'Todo' }
  requestWrite(
    'Resetear Contadores — Accion Irreversible',
    'Los valores acumulados se perderan permanentemente. Esta accion no se puede deshacer.',
    `Tipo de reset: ${labels[resetType.value] ?? resetType.value}`,
    applyReset
  )
}

function selectMeter(m) {
  selectedMeter.value = m
  deviceCfg.value     = null
  loadError.value     = ''
  pendingWrite.value  = null
  results.value       = { ct: null, sys: null, dir: null, time: null, reset: null }
}

async function loadDeviceConfig() {
  loadingCfg.value = true
  loadError.value  = ''
  deviceCfg.value  = null
  try {
    const cfg = await getMeterDeviceConfig(selectedMeter.value.unit_id)
    deviceCfg.value = cfg
    sys.value.wiring  = cfg.wiring
    sys.value.freq    = cfg.freq
    sys.value.nominal = cfg.nominal_v
    sys.value.vt      = cfg.vt_ratio
    dir.value.l1 = cfg.dir_l1
    dir.value.l2 = cfg.dir_l2
    dir.value.l3 = cfg.dir_l3
    ct.value.secondary = 5
    ct.value.primary   = Math.round(cfg.ct_ratio * ct.value.secondary)
  } catch (e) {
    loadError.value = e.response?.data?.error ?? 'No se pudo leer el dispositivo'
  } finally {
    loadingCfg.value = false
  }
}

async function applySetCT() {
  busy.value.ct = true
  results.value.ct = null
  try {
    const r = await meterSetCT(selectedMeter.value.unit_id, selectedMeter.value.meter_id,
                               ct.value.primary, ct.value.secondary)
    setResult('ct', r.success, r.message)
    showToast(r.success, r.message)
  } catch (e) {
    const msg = e.response?.data?.error ?? 'Error al escribir TC'
    setResult('ct', false, msg)
    showToast(false, msg)
  } finally {
    busy.value.ct = false
  }
}

async function applySetSys() {
  busy.value.sys = true
  results.value.sys = null
  try {
    const r = await meterSetSys(selectedMeter.value.unit_id, selectedMeter.value.meter_id, {
      wiring:  sys.value.wiring,
      freq:    sys.value.freq,
      nominal: sys.value.nominal,
      vt:      sys.value.vt,
    })
    setResult('sys', r.success, r.message)
    showToast(r.success, r.message)
  } catch (e) {
    const msg = e.response?.data?.error ?? 'Error al escribir sistema'
    setResult('sys', false, msg)
    showToast(false, msg)
  } finally {
    busy.value.sys = false
  }
}

async function applySetDir() {
  busy.value.dir = true
  results.value.dir = null
  try {
    const r = await meterSetDir(selectedMeter.value.unit_id, selectedMeter.value.meter_id,
                                dir.value.l1, dir.value.l2, dir.value.l3)
    setResult('dir', r.success, r.message)
    showToast(r.success, r.message)
  } catch (e) {
    const msg = e.response?.data?.error ?? 'Error al escribir direccion'
    setResult('dir', false, msg)
    showToast(false, msg)
  } finally {
    busy.value.dir = false
  }
}

async function applySetTime() {
  busy.value.time = true
  results.value.time = null
  try {
    const r = await meterSetTime(selectedMeter.value.unit_id, selectedMeter.value.meter_id)
    setResult('time', r.success, r.message)
    showToast(r.success, r.message)
  } catch (e) {
    const msg = e.response?.data?.error ?? 'Error al sincronizar reloj'
    setResult('time', false, msg)
    showToast(false, msg)
  } finally {
    busy.value.time = false
  }
}

async function applyReset() {
  busy.value.reset = true
  results.value.reset = null
  try {
    const r = await meterReset(selectedMeter.value.unit_id, selectedMeter.value.meter_id, resetType.value)
    setResult('reset', r.success, r.message)
    showToast(r.success, r.message)
  } catch (e) {
    const msg = e.response?.data?.error ?? 'Error al resetear contadores'
    setResult('reset', false, msg)
    showToast(false, msg)
  } finally {
    busy.value.reset = false
  }
}

onMounted(async () => {
  try {
    meters.value = await getMeters()
  } catch {
    // meters list unavailable
  }
})
</script>

<style scoped>
.section-title {
  @apply text-sm font-semibold text-gray-200 flex items-center gap-2;
}
.write-btn {
  @apply flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-semibold text-white
         bg-blue-600 hover:bg-blue-500 border border-blue-500/50 transition-colors
         disabled:cursor-not-allowed;
}
.toast-enter-active,
.toast-leave-active {
  transition: all 0.2s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.15s ease;
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
