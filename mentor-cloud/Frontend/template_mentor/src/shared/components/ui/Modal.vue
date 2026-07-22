<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    required: true
  },
  title: {
    type: String,
    default: ''
  },
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg', 'xl'].includes(value)
  },
  closeOnBackdrop: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['update:modelValue', 'close'])

const sizeClasses = computed(() => {
  const sizes = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl'
  }
  return sizes[props.size]
})

function close() {
  emit('update:modelValue', false)
  emit('close')
}

function handleBackdropClick() {
  if (props.closeOnBackdrop) {
    close()
  }
}
</script>

<template>
  <Transition name="modal">
    <div v-if="modelValue" class="modal-overlay" @click="handleBackdropClick">
      <div :class="['modal-content', sizeClasses]" @click.stop>
        <div v-if="title || $slots.header" class="modal-header">
          <slot name="header">
            <h3 class="modal-title">{{ title }}</h3>
          </slot>
          <button class="modal-close" @click="close">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        
        <div class="modal-body">
          <slot />
        </div>
        
        <div v-if="$slots.footer" class="modal-footer">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.modal-overlay {
  @apply fixed inset-0 z-50 flex items-center justify-center;
  @apply bg-black bg-opacity-50 p-4;
}

.modal-content {
  @apply bg-white dark:bg-slate-800 rounded-lg shadow-xl w-full;
  @apply max-h-[90vh] flex flex-col;
}

.modal-header {
  @apply flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-slate-700;
}

.modal-title {
  @apply text-xl font-semibold text-gray-900 dark:text-slate-100;
}

.modal-close {
  @apply text-gray-400 dark:text-slate-500 hover:text-gray-600 dark:hover:text-slate-300 transition-colors;
}

.modal-body {
  @apply px-6 py-4 overflow-y-auto flex-1;
  min-height: 0;
}

.modal-footer {
  @apply px-6 py-4 border-t border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-slate-900;
}

.modal-enter-active,
.modal-leave-active {
  @apply transition-opacity duration-300;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  @apply transition-transform duration-300;
}

.modal-enter-from,
.modal-leave-to {
  @apply opacity-0;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  @apply scale-95;
}
</style>
