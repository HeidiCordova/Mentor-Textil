<script setup>
defineProps({
  variant: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'secondary', 'success', 'danger', 'warning', 'ghost'].includes(value)
  },
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg'].includes(value)
  },
  disabled: {
    type: Boolean,
    default: false
  },
  loading: {
    type: Boolean,
    default: false
  },
  block: {
    type: Boolean,
    default: false
  }
})

defineEmits(['click'])
</script>

<template>
  <button
    :class="[
      'btn',
      `btn-${variant}`,
      `btn-${size}`,
      { 'btn-block': block, 'btn-loading': loading }
    ]"
    :disabled="disabled || loading"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="btn-spinner"></span>
    <slot v-else />
  </button>
</template>

<style scoped>
.btn {
  @apply inline-flex items-center justify-center font-medium rounded-lg transition-colors duration-200;
  @apply focus:outline-none focus:ring-2 focus:ring-offset-2;
  @apply disabled:opacity-50 disabled:cursor-not-allowed;
}

.btn-sm {
  @apply px-3 py-1.5 text-sm;
}

.btn-md {
  @apply px-4 py-2 text-base;
}

.btn-lg {
  @apply px-6 py-3 text-lg;
}

.btn-primary {
  @apply bg-primary-600 text-white hover:bg-primary-700;
  @apply focus:ring-primary-500;
}

.btn-secondary {
  @apply bg-secondary-600 text-white hover:bg-secondary-700;
  @apply focus:ring-secondary-500;
}

.btn-success {
  @apply bg-green-600 text-white hover:bg-green-700;
  @apply focus:ring-green-500;
}

.btn-danger {
  @apply bg-red-600 text-white hover:bg-red-700;
  @apply focus:ring-red-500;
}

.btn-warning {
  @apply bg-yellow-600 text-white hover:bg-yellow-700;
  @apply focus:ring-yellow-500;
}

.btn-ghost {
  @apply bg-transparent text-gray-700 hover:bg-gray-100;
  @apply focus:ring-gray-500;
}

.btn-block {
  @apply w-full;
}

.btn-loading {
  @apply relative text-transparent;
}

.btn-spinner {
  @apply absolute inset-0 flex items-center justify-center;
}

.btn-spinner::after {
  content: '';
  @apply w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin;
}
</style>
