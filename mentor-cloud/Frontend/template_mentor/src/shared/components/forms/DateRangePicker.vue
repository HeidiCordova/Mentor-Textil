<script setup>
const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  },
  label: {
    type: String,
    default: ''
  },
  error: {
    type: String,
    default: ''
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

function handleStartChange(event) {
  emit('update:modelValue', {
    ...props.modelValue,
    start: event.target.value
  })
}

function handleEndChange(event) {
  emit('update:modelValue', {
    ...props.modelValue,
    end: event.target.value
  })
}
</script>

<template>
  <div class="form-field">
    <label v-if="label" class="form-label">{{ label }}</label>
    <div class="date-range-inputs">
      <input
        type="datetime-local"
        :value="modelValue.start"
        :disabled="disabled"
        :class="['form-input', { 'form-input-error': error }]"
        @input="handleStartChange"
      />
      <span class="date-range-separator">-</span>
      <input
        type="datetime-local"
        :value="modelValue.end"
        :disabled="disabled"
        :class="['form-input', { 'form-input-error': error }]"
        @input="handleEndChange"
      />
    </div>
    <p v-if="error" class="form-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.form-field {
  @apply w-full;
}

.form-label {
  @apply block text-sm font-medium text-gray-700 mb-1;
}

.date-range-inputs {
  @apply flex items-center gap-2;
}

.date-range-separator {
  @apply text-gray-500 flex-shrink-0;
}

.form-input {
  @apply flex-1 px-3 py-2 border border-gray-300 rounded-lg;
  @apply focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent;
  @apply disabled:bg-gray-100 disabled:cursor-not-allowed;
}

.form-input-error {
  @apply border-red-500 focus:ring-red-500;
}

.form-error {
  @apply text-sm text-red-600 mt-1;
}
</style>
