<script setup>
defineProps({
  title: {
    type: String,
    default: ''
  },
  padding: {
    type: Boolean,
    default: true
  },
  shadow: {
    type: Boolean,
    default: true
  }
})
</script>

<template>
  <div :class="['card', { 'card-padding': padding, 'card-shadow': shadow }]">
    <div v-if="title || $slots.header" class="card-header">
      <slot name="header">
        <h3 class="card-title">{{ title }}</h3>
      </slot>
    </div>
    <div class="card-body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="card-footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.card {
  @apply bg-white dark:bg-slate-800 rounded-lg border border-gray-200 dark:border-slate-700;
}

.card-shadow {
  @apply shadow-sm;
}

.card-padding .card-body {
  @apply p-6;
}

.card-header {
  @apply px-6 py-4 border-b border-gray-200 dark:border-slate-700;
}

.card-title {
  @apply text-lg font-semibold text-gray-900 dark:text-slate-100;
}

.card-footer {
  @apply px-6 py-4 border-t border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-slate-900;
}
</style>
