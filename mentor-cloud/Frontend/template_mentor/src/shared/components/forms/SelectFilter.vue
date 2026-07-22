<script setup>
const props = defineProps({
  modelValue: {
    type: [String, Number, Boolean, null],
    default: null
  },
  options: {
    type: Array,
    required: true
  },
  label: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: 'Seleccione una opción'
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
  },
  valueKey: {
    type: String,
    default: 'value'
  },
  labelKey: {
    type: String,
    default: 'label'
  }
})

const emit = defineEmits(['update:modelValue'])

function handleChange(event) {
  const raw = event.target.value
  if (raw === '') {
    emit('update:modelValue', null)
    return
  }
  const num = Number(raw)
  emit('update:modelValue', Number.isNaN(num) ? raw : num)
}
</script>

<template>
  <div class="form-field">
    <label v-if="label" class="form-label">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>
    <select
      :value="modelValue"
      :disabled="disabled"
      :class="['form-select', { 'form-select-error': error }]"
      @change="handleChange"
    >
      <option value="">{{ placeholder }}</option>
      <option
        v-for="option in options"
        :key="option[valueKey]"
        :value="option[valueKey]"
      >
        {{ option[labelKey] }}
      </option>
    </select>
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

.form-select {
  @apply w-full px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg;
  @apply bg-white dark:bg-slate-700 text-gray-900 dark:text-slate-100;
  @apply focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent;
  @apply disabled:bg-gray-100 dark:disabled:bg-slate-800 disabled:cursor-not-allowed;
}

.form-select-error {
  @apply border-red-500 focus:ring-red-500;
}

.form-error {
  @apply text-sm text-red-600 mt-1;
}
</style>
