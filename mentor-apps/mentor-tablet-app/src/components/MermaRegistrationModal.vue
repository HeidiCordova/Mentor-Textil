<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'
import { api } from '@/services/api'

const emit = defineEmits<{
  saved:   [valor: number]
  cancel:  []
}>()

// ── hora automática ──────────────────────────────────────────────────
const now = ref(new Date())
let _ticker: ReturnType<typeof setInterval>
onMounted(() => {
  _ticker = setInterval(() => { now.value = new Date() }, 1000)
})
onUnmounted(() => {
  clearInterval(_ticker)
})

function fmt(d: Date) {
  return d.toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
}

// ── input ────────────────────────────────────────────────────────────
const rawValue  = ref<string>('')
const inputRef  = ref<HTMLInputElement | null>(null)
const saving    = ref(false)
const errorMsg  = ref('')

const numValue = computed(() => {
  const n = parseInt(rawValue.value, 10)
  return isNaN(n) ? 0 : n
})

const canSave = computed(() => numValue.value > 0)

function onKeydown(e: KeyboardEvent) {
  // Sólo números y teclas de control
  if (
    !/^\d$/.test(e.key) &&
    !['Backspace', 'Delete', 'ArrowLeft', 'ArrowRight', 'Tab', 'Enter'].includes(e.key)
  ) {
    e.preventDefault()
  }
  if (e.key === 'Enter' && canSave.value) confirm()
  if (e.key === 'Escape') emit('cancel')
}

// ── acciones ─────────────────────────────────────────────────────────
async function confirm() {
  if (!canSave.value || saving.value) return
  errorMsg.value = ''
  saving.value = true
  try {
    await api.setVariableValor('MERMA', String(numValue.value))
    emit('saved', numValue.value)
  } catch (err: unknown) {
    errorMsg.value = err instanceof Error ? err.message : 'Error al guardar'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  setTimeout(() => inputRef.value?.focus(), 50)
})
</script>

<template>
  <!-- Overlay -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
    @click.self="emit('cancel')"
  >
    <!-- Card -->
    <div class="w-[92vw] max-w-sm rounded-2xl bg-[#1a2234] border border-orange-500/30 shadow-2xl shadow-orange-900/30 overflow-hidden">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 bg-orange-600/20 border-b border-orange-500/20">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-full bg-orange-500/20 flex items-center justify-center">
            <SvgIcon name="plus" :size="16" class="text-orange-400" />
          </div>
          <div>
            <h2 class="text-white font-bold text-base leading-tight">Registrar Merma</h2>
            <p class="text-orange-300/80 text-xs">Unidades con defecto / descarte</p>
          </div>
        </div>
        <button class="w-7 h-7 rounded-full flex items-center justify-center text-slate-400 hover:text-white hover:bg-slate-700/60 transition-colors" @click="emit('cancel')">
          ✕
        </button>
      </div>

      <!-- Body -->
      <div class="px-6 py-5 space-y-5">

        <!-- Hora automática -->
        <div class="flex items-center gap-3 px-4 py-3 rounded-xl bg-slate-800/60 border border-slate-700/40">
          <div class="w-8 h-8 rounded-full bg-cyan-500/15 flex items-center justify-center shrink-0">
            <svg class="w-4 h-4 text-cyan-400" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 15"/>
            </svg>
          </div>
          <div>
            <p class="text-slate-400 text-xs mb-0.5">Hora de registro</p>
            <p class="text-white font-mono font-semibold text-sm tracking-wide">{{ fmt(now) }}</p>
          </div>
          <span class="ml-auto text-xs bg-cyan-500/15 text-cyan-400 px-2 py-0.5 rounded-full font-medium">Auto</span>
        </div>

        <!-- Input cantidad -->
        <div>
          <label class="block text-slate-300 text-sm font-medium mb-2">Cantidad de piezas con merma</label>
          <div class="relative">
            <input
              ref="inputRef"
              v-model="rawValue"
              type="number"
              min="1"
              inputmode="numeric"
              pattern="[0-9]*"
              placeholder="0"
              class="w-full bg-slate-800/70 border border-slate-600/50 text-white text-3xl font-bold text-center rounded-xl px-4 py-4 focus:outline-none focus:border-orange-500/70 focus:ring-2 focus:ring-orange-500/20 placeholder-slate-600 transition-all"
              :class="{ 'border-red-500/60': errorMsg }"
              @keydown="onKeydown"
            />
            <span class="absolute right-4 top-1/2 -translate-y-1/2 text-slate-500 text-sm font-medium pointer-events-none">uds</span>
          </div>
          <p v-if="errorMsg" class="mt-1.5 text-red-400 text-xs">{{ errorMsg }}</p>
          <p class="mt-1.5 text-slate-500 text-xs">Ingresa el número de unidades descartadas o defectuosas en este turno.</p>
        </div>

      </div>

      <!-- Footer -->
      <div class="flex gap-3 px-6 pb-5">
        <button
          class="flex-1 py-3 rounded-xl bg-slate-700/50 text-slate-300 font-semibold text-sm hover:bg-slate-700 transition-colors"
          @click="emit('cancel')"
        >
          Cancelar
        </button>
        <button
          class="flex-[2] py-3 rounded-xl font-bold text-sm transition-all flex items-center justify-center gap-2"
          :class="canSave && !saving
            ? 'bg-orange-600 text-white hover:bg-orange-500 shadow-lg shadow-orange-900/30'
            : 'bg-slate-700/60 text-slate-500 cursor-not-allowed'"
          :disabled="!canSave || saving"
          @click="confirm"
        >
          <span v-if="saving" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          <span>{{ saving ? 'Guardando…' : 'Registrar' }}</span>
        </button>
      </div>

    </div>
  </div>
</template>
