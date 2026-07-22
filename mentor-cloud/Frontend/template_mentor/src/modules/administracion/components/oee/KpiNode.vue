<script setup>
import { Handle, Position } from '@vue-flow/core'

defineProps({
  data: { type: Object, required: true }
})

// Mapa de colores → variantes visuales
const borderMap = {
  '#3b82f6': { border: '#3b82f6', bg: '#172554', text: '#93c5fd', badge: '#1d4ed8' },  // azul - Disponibilidad
  '#22c55e': { border: '#22c55e', bg: '#052e16', text: '#86efac', badge: '#15803d' },  // verde - Rendimiento
  '#f59e0b': { border: '#f59e0b', bg: '#1c1003', text: '#fcd34d', badge: '#b45309' }   // amarillo - Calidad
}
</script>

<template>
  <div
    class="kpi-node"
    :style="{
      borderColor: data.color,
      background: borderMap[data.color]?.bg ?? '#0f172a'
    }"
  >
    <!-- 2 entradas: a arriba, b abajo -->
    <Handle type="target" :position="Position.Left" id="a" style="top:30%" :style="{ background: data.color }" />
    <Handle type="target" :position="Position.Left" id="b" style="top:55%" :style="{ background: data.color }" />
    <!-- 3ra entrada opcional para RENDIMIENTO -->
    <Handle type="target" :position="Position.Left" id="c" style="top:75%" :style="{ background: data.color }" />

    <div class="kn-label" :style="{ color: borderMap[data.color]?.text ?? '#e2e8f0' }">
      {{ data.label }}
    </div>
    <div class="kn-formula" :style="{ color: borderMap[data.color]?.text ?? '#94a3b8' }">
      {{ data.formula }}
    </div>

    <!-- Salida hacia OEE -->
    <Handle type="source" :position="Position.Right" id="out" :style="{ background: data.color }" />
  </div>
</template>

<style scoped>
.kpi-node {
  border: 2.5px solid;
  border-radius: 10px;
  padding: 12px 16px;
  min-width: 220px;
  cursor: grab;
  position: relative;
}
.kn-label {
  font-size: 14px;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  margin-bottom: 4px;
}
.kn-formula {
  font-size: 10px;
  font-family: monospace;
  opacity: 0.9;
  white-space: normal;
  word-break: break-word;
}
</style>
