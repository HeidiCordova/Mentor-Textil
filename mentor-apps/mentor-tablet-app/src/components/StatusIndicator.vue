<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  color?: 'green' | 'yellow' | 'red' | 'blue' | 'default'
  icon?: string
}>()

const colorMap: Record<string, string> = {
  green: 'bg-green-500/10 border-green-500/30 text-green-400',
  yellow: 'bg-yellow-500/10 border-yellow-500/30 text-yellow-400',
  red: 'bg-red-500/10 border-red-500/30 text-red-400',
  blue: 'bg-blue-500/10 border-blue-500/30 text-blue-400',
  default: 'bg-edge-800 border-edge-700/50 text-edge-300'
}
</script>

<template>
  <div
    class="flex flex-col items-center justify-center p-3 rounded-lg border"
    :class="colorMap[color || 'default']"
  >
    <span class="text-[10px] uppercase tracking-wider opacity-70 mb-1">{{ label }}</span>
    <span class="text-lg font-bold tabular-nums">{{ value }}</span>
  </div>
</template>
