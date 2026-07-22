<script setup>
import { Handle, Position } from '@vue-flow/core'

defineProps({
  data: { type: Object, required: true }
})
</script>

<template>
  <div class="formula-node">
    <!-- Entradas: a y b posicionadas arriba/abajo a la izquierda -->
    <Handle type="target" :position="Position.Left" id="a" style="top:35%; background:#6366f1" />
    <Handle type="target" :position="Position.Left" id="b" style="top:65%; background:#6366f1" />

    <div class="fn-label">{{ data.label }}</div>
    <div class="fn-formula">{{ data.formula }}</div>

    <div v-if="data.inputLabels?.length" class="fn-inputs">
      <span class="fn-input-tag">{{ data.inputLabels[0] }}</span>
      <span class="fn-op">{{ data.operator }}</span>
      <span class="fn-input-tag">{{ data.inputLabels[1] }}</span>
    </div>

    <!-- Salida -->
    <Handle type="source" :position="Position.Right" id="out" style="background:#6366f1" />
  </div>
</template>

<style scoped>
.formula-node {
  background: #1e1b4b;
  border: 1.5px solid #6366f1;
  border-radius: 8px;
  padding: 10px 14px;
  min-width: 200px;
  cursor: grab;
  position: relative;
}
.fn-label {
  font-size: 12px;
  font-weight: 700;
  color: #a5b4fc;
  letter-spacing: 0.03em;
  margin-bottom: 4px;
}
.fn-formula {
  font-size: 15px;
  color: #e0e7ff;
  font-family: monospace;
  margin-bottom: 6px;
}
.fn-inputs {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.fn-input-tag {
  font-size: 9px;
  background: #312e81;
  color: #a5b4fc;
  padding: 1px 5px;
  border-radius: 4px;
  font-family: monospace;
}
.fn-op {
  font-size: 12px;
  color: #818cf8;
  font-weight: 700;
}
</style>
