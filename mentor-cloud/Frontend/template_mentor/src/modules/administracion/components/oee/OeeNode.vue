<script setup>
import { Handle, Position } from '@vue-flow/core'

defineProps({
  data: { type: Object, required: true }
})
</script>

<template>
  <div class="oee-node">
    <!-- 3 entradas: Disponibilidad, Rendimiento, Calidad -->
    <Handle type="target" :position="Position.Left" id="a" style="top:25%; background:#3b82f6" />
    <Handle type="target" :position="Position.Left" id="b" style="top:50%; background:#22c55e" />
    <Handle type="target" :position="Position.Left" id="c" style="top:75%; background:#f59e0b" />

    <div class="oee-title">OEE</div>
    <div class="oee-formula">{{ data.formula }}</div>
    <div class="oee-labels">
      <span class="oee-d">D</span>
      <span class="oee-x">×</span>
      <span class="oee-r">R</span>
      <span class="oee-x">×</span>
      <span class="oee-c">C</span>
    </div>
  </div>
</template>

<style scoped>
.oee-node {
  background: radial-gradient(ellipse at center, #3b0764 0%, #1a0333 100%);
  border: 3px solid #ef4444;
  border-radius: 12px;
  padding: 16px 20px;
  min-width: 180px;
  text-align: center;
  cursor: grab;
  position: relative;
  box-shadow: 0 0 24px rgba(239,68,68,0.4);
}
.oee-title {
  font-size: 28px;
  font-weight: 900;
  color: #ef4444;
  letter-spacing: 0.15em;
  text-shadow: 0 0 12px rgba(239,68,68,0.6);
  line-height: 1;
  margin-bottom: 6px;
}
.oee-formula {
  font-size: 9px;
  color: #fca5a5;
  font-family: monospace;
  margin-bottom: 8px;
}
.oee-labels {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 6px;
  font-size: 16px;
  font-weight: 800;
}
.oee-d { color: #3b82f6; }
.oee-r { color: #22c55e; }
.oee-c { color: #f59e0b; }
.oee-x { color: #6b7280; }
</style>
