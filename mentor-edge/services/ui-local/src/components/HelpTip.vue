<template>
  <span class="help-tip" :data-tip="text">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none"
         stroke="currentColor" stroke-width="2"
         stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="12" r="10"/>
      <path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3m.08 4h.01"/>
    </svg>
  </span>
</template>

<script setup>
defineProps({ text: { type: String, required: true } })
</script>

<style scoped>
.help-tip {
  position: relative;
  display: inline-flex;
  align-items: center;
  margin-left: 4px;
  color: #475569;
  cursor: help;
  vertical-align: middle;
  transition: color 0.15s ease;
}
.help-tip:hover { color: #94a3b8; }
.help-tip::after {
  content: attr(data-tip);
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 400;
  line-height: 1.5;
  color: #e2e8f0;
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
  white-space: normal;
  width: 260px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.15s ease;
  z-index: 50;
}
.help-tip:hover::after { opacity: 1; }
</style>
