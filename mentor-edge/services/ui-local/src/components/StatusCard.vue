<template>
  <div class="bg-slate-800 rounded-lg shadow p-6">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-gray-400">{{ title }}</p>
        <p class="mt-2 text-3xl font-semibold text-white">{{ value }}</p>
      </div>
      <div class="flex-shrink-0">
        <div 
          class="w-12 h-12 rounded-md flex items-center justify-center"
          :class="iconBgClass"
        >
          <svg class="w-6 h-6" :class="iconClass" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path v-if="icon === 'queue'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
            <path v-else-if="icon === 'server'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"/>
            <path v-else-if="icon === 'cloud'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"/>
          </svg>
        </div>
      </div>
    </div>
    <div class="mt-4">
      <span 
        class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
        :class="statusBadgeClass"
      >
        {{ statusLabel }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  title: String,
  value: [String, Number],
  status: String,
  icon: String
})

const iconBgClass = computed(() => {
  if (props.status === 'ok') return 'bg-green-900'
  if (props.status === 'warning') return 'bg-yellow-900'
  return 'bg-red-900'
})

const iconClass = computed(() => {
  if (props.status === 'ok') return 'text-green-400'
  if (props.status === 'warning') return 'text-yellow-400'
  return 'text-red-400'
})

const statusBadgeClass = computed(() => {
  if (props.status === 'ok') return 'bg-green-900 text-green-300'
  if (props.status === 'warning') return 'bg-yellow-900 text-yellow-300'
  return 'bg-red-900 text-red-300'
})

const statusLabel = computed(() => {
  if (props.status === 'ok') return 'Operativo'
  if (props.status === 'warning') return 'Advertencia'
  return 'Error'
})
</script>
