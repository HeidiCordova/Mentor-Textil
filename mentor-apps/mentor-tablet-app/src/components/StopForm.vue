<script setup lang="ts">
import { ref, computed } from 'vue'
import SvgIcon from './SvgIcon.vue'
import type { Stop, CreateStopRequest, JustifyStopRequest, StopType, StopSource } from '@/types'

const props = defineProps<{
  mode: 'create' | 'justify'
  stop?: Stop | null
}>()

const emit = defineEmits<{
  submit: [data: CreateStopRequest | JustifyStopRequest]
  cancel: []
}>()

const stopTypes: { value: StopType; label: string }[] = [
  { value: 'MECANICA', label: 'Mecanica' },
  { value: 'ELECTRICA', label: 'Electrica' },
  { value: 'FALTA_MATERIAL', label: 'Falta de material' },
  { value: 'CALIDAD', label: 'Calidad' },
  { value: 'CAMBIO_FORMATO', label: 'Cambio de formato' },
  { value: 'PROGRAMADA', label: 'Programada' },
  { value: 'MICROPARADA', label: 'Microparada' },
  { value: 'PARADA_NO_ASIGNADA', label: 'No asignada' },
  { value: 'OTRA', label: 'Otra' }
]

const stopSources: { value: StopSource; label: string }[] = [
  { value: 'operator', label: 'Operador' },
  { value: 'system', label: 'Sistema' }
]

const selectedType = ref<StopType>(props.stop?.stop_type || 'OTRA')
const selectedSource = ref<StopSource>('operator')
const reason = ref(props.stop?.reason || '')
const category = ref(props.stop?.category || '')

const isValid = computed(() => {
  if (props.mode === 'create') return !!selectedType.value
  return !!reason.value.trim() && !!category.value.trim()
})

function handleSubmit(): void {
  if (!isValid.value) return

  if (props.mode === 'create') {
    const data: CreateStopRequest = {
      stop_type: selectedType.value,
      source: selectedSource.value,
      started_at: new Date().toISOString(),
      reason: reason.value || undefined,
      category: category.value || undefined
    }
    emit('submit', data)
  } else {
    const data: JustifyStopRequest = {
      stop_type: selectedType.value,
      reason: reason.value,
      category: category.value
    }
    emit('submit', data)
  }
}
</script>

<template>
  <div class="bg-edge-800 rounded-xl border border-edge-700/50 p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold text-edge-100">
        {{ mode === 'create' ? 'Registrar Parada' : 'Justificar Parada' }}
      </h3>
      <button
        class="p-1 rounded-md hover:bg-edge-700/50 text-edge-400 hover:text-edge-200 transition-colors"
        @click="$emit('cancel')"
      >
        <SvgIcon name="close" :size="18" />
      </button>
    </div>

    <div v-if="mode === 'justify' && stop" class="mb-4 p-3 rounded-lg bg-edge-900/50 border border-edge-700/30">
      <div class="grid grid-cols-2 gap-2 text-xs">
        <div>
          <span class="text-edge-500">Tipo actual:</span>
          <span class="ml-1 text-edge-300">{{ stop.stop_type }}</span>
        </div>
        <div>
          <span class="text-edge-500">Fuente:</span>
          <span class="ml-1 text-edge-300">{{ stop.source }}</span>
        </div>
        <div>
          <span class="text-edge-500">Inicio:</span>
          <span class="ml-1 text-edge-300">{{ new Date(stop.started_at).toLocaleTimeString() }}</span>
        </div>
        <div v-if="stop.duration_ms">
          <span class="text-edge-500">Duracion:</span>
          <span class="ml-1 text-edge-300">{{ Math.round(stop.duration_ms / 60000) }} min</span>
        </div>
      </div>
    </div>

    <div class="space-y-4">
      <div>
        <label class="block text-xs font-medium text-edge-400 mb-1.5">Tipo de parada</label>
        <div class="grid grid-cols-3 gap-1.5">
          <button
            v-for="t in stopTypes"
            :key="t.value"
            class="px-2 py-1.5 text-[11px] rounded-md border transition-colors"
            :class="[
              selectedType === t.value
                ? 'border-production-active bg-production-active/20 text-edge-100'
                : 'border-edge-700/50 text-edge-400 hover:border-edge-600'
            ]"
            @click="selectedType = t.value"
          >
            {{ t.label }}
          </button>
        </div>
      </div>

      <div v-if="mode === 'create'">
        <label class="block text-xs font-medium text-edge-400 mb-1.5">Fuente</label>
        <div class="flex gap-2">
          <button
            v-for="s in stopSources"
            :key="s.value"
            class="px-3 py-1.5 text-xs rounded-md border transition-colors"
            :class="[
              selectedSource === s.value
                ? 'border-production-active bg-production-active/20 text-edge-100'
                : 'border-edge-700/50 text-edge-400 hover:border-edge-600'
            ]"
            @click="selectedSource = s.value"
          >
            {{ s.label }}
          </button>
        </div>
      </div>

      <div>
        <label class="block text-xs font-medium text-edge-400 mb-1.5">Razon</label>
        <textarea
          v-model="reason"
          :rows="mode === 'justify' ? 3 : 2"
          class="w-full px-3 py-2 text-sm bg-edge-900 border border-edge-700/50 rounded-lg text-edge-200 placeholder-edge-600 focus:border-production-active focus:outline-none resize-none"
          placeholder="Describa la causa de la parada..."
        />
      </div>

      <div>
        <label class="block text-xs font-medium text-edge-400 mb-1.5">Categoria</label>
        <input
          v-model="category"
          type="text"
          class="w-full px-3 py-2 text-sm bg-edge-900 border border-edge-700/50 rounded-lg text-edge-200 placeholder-edge-600 focus:border-production-active focus:outline-none"
          placeholder="Categoria de la parada..."
        />
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <button
          class="px-4 py-2 text-xs font-medium rounded-lg border border-edge-700/50 text-edge-400 hover:text-edge-200 hover:border-edge-600 transition-colors"
          @click="$emit('cancel')"
        >
          Cancelar
        </button>
        <button
          class="px-4 py-2 text-xs font-medium rounded-lg transition-colors"
          :class="[
            isValid
              ? 'bg-production-active text-white hover:bg-blue-600'
              : 'bg-edge-700 text-edge-500 cursor-not-allowed'
          ]"
          :disabled="!isValid"
          @click="handleSubmit"
        >
          <span class="flex items-center gap-1">
            <SvgIcon :name="mode === 'create' ? 'plus' : 'check'" :size="14" />
            {{ mode === 'create' ? 'Registrar' : 'Justificar' }}
          </span>
        </button>
      </div>
    </div>
  </div>
</template>
