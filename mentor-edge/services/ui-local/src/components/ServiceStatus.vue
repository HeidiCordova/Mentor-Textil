<template>
  <div class="flex items-center justify-between p-3 bg-slate-700 rounded">
    <span class="text-sm font-medium text-gray-300">{{ name }}</span>
    <span 
      class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
      :class="statusClass"
    >
      <span class="w-2 h-2 mr-1.5 rounded-full" :class="dotClass"></span>
      {{ statusText }}
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: String,
  status: String
})

const statusClass = computed(() => {
  if (props.status === 'ok') return 'bg-green-900 text-green-300'
  if (props.status === 'degraded') return 'bg-yellow-900 text-yellow-300'
  return 'bg-red-900 text-red-300'
})

const dotClass = computed(() => {
  if (props.status === 'ok') return 'bg-green-400'
  if (props.status === 'degraded') return 'bg-yellow-400'
  return 'bg-red-400'
})

const statusText = computed(() => {
  if (props.status === 'ok') return 'Activo'
  if (props.status === 'degraded') return 'Degradado'
  return 'Inactivo'
})
</script>
