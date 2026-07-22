export function required(value) {
  if (Array.isArray(value)) return value.length > 0
  return !!value || value === 0
}

export function email(value) {
  if (!value) return true
  const pattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return pattern.test(value)
}

export function minLength(value, min) {
  if (!value) return true
  return value.length >= min
}

export function maxLength(value, max) {
  if (!value) return true
  return value.length <= max
}

export function numeric(value) {
  if (!value && value !== 0) return true
  return !isNaN(Number(value))
}

export function positive(value) {
  if (!value && value !== 0) return true
  return Number(value) > 0
}

export function min(value, minValue) {
  if (!value && value !== 0) return true
  return Number(value) >= minValue
}

export function max(value, maxValue) {
  if (!value && value !== 0) return true
  return Number(value) <= maxValue
}

export function url(value) {
  if (!value) return true
  try {
    new URL(value)
    return true
  } catch {
    return false
  }
}

export function pattern(value, regex) {
  if (!value) return true
  return regex.test(value)
}
