<script setup>
import { Handle, Position } from '@vue-flow/core'

defineProps({
  data: { type: Object, required: true }
})

const fuenteColor = {
  turno: '#8b5cf6',
  variable: '#0ea5e9',
  config: '#f59e0b'
}
</script>

<template>
  <div class="variable-node">
    <div class="vn-header">
      <span class="vn-clave">{{ data.clave }}</span>
      <span class="vn-badge" :style="{ background: fuenteColor[data.fuente] || '#64748b' }">
        {{ data.fuente }}
      </span>
    </div>
    <div class="vn-desc">{{ data.descripcion }}</div>

    <Handle type="source" :position="Position.Right" id="out" style="background:#64748b" />
  </div>
</template>

<style scoped>
.variable-node {
  background: #1e293b;
  border: 1.5px solid #475569;
  border-radius: 8px;
  padding: 8px 12px;
  min-width: 200px;
  max-width: 240px;
  font-family: monospace;
  cursor: grab;
}
.vn-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}
.vn-clave {
  font-size: 11px;
  font-weight: 700;
  color: #94a3b8;
  letter-spacing: 0.04em;
}
.vn-badge {
  font-size: 9px;
  color: #fff;
  padding: 1px 5px;
  border-radius: 999px;
  text-transform: uppercase;
  font-weight: 600;
}
.vn-desc {
  font-size: 10px;
  color: #64748b;
  line-height: 1.3;
  white-space: normal;
  word-break: break-word;
}
</style>
