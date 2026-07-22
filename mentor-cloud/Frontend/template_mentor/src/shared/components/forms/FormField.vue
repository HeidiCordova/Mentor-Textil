<script setup>
const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: ''
  },
  label: {
    type: String,
    default: ''
  },
  type: {
    type: String,
    default: 'text'
  },
  placeholder: {
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
  },
  required: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue'])

function handleInput(event) {
  emit('update:modelValue', event.target.value)
}
</script>

<template>
  <div class="form-field">
    <label v-if="label" class="form-label">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>
    <input
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :class="['form-input', { 'form-input-error': error }]"
      @input="handleInput"
    />
    <p v-if="error" class="form-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.form-field {
  @apply w-full;
}

.form-label {
  @apply block text-sm font-medium text-gray-700 dark:text-slate-300 mb-1;
}

.form-input {
  @apply w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg;
  @apply bg-white dark:bg-slate-700 text-gray-900 dark:text-slate-100;
  @apply placeholder:text-gray-400 dark:placeholder:text-slate-500;
  @apply focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent;
  @apply disabled:bg-gray-100 dark:disabled:bg-slate-800 disabled:cursor-not-allowed;
}

.form-input-error {
  @apply border-red-500 focus:ring-red-500;
}

.form-error {
  @apply text-sm text-red-600 mt-1;
}
</style>
