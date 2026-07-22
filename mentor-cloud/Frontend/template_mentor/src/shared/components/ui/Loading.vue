<script setup>
defineProps({
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg'].includes(value)
  },
  text: {
    type: String,
    default: 'Cargando...'
  }
})
</script>

<template>
  <div class="loading-container">
    <div :class="['spinner', `spinner-${size}`]"></div>
    <p v-if="text" class="loading-text">{{ text }}</p>
  </div>
</template>

<style scoped>
.loading-container {
  @apply flex flex-col items-center justify-center gap-3;
}

.spinner {
  @apply border-4 border-gray-200 border-t-primary-600 rounded-full animate-spin;
}

.spinner-sm {
  @apply w-8 h-8 border-2;
}

.spinner-md {
  @apply w-12 h-12;
}

.spinner-lg {
  @apply w-16 h-16;
}

.loading-text {
  @apply text-gray-600 text-sm;
}
</style>
