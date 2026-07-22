import { ref, computed } from 'vue'

export function useDateRange(options = {}) {
  const startDate = ref(options.startDate || null)
  const endDate = ref(options.endDate || null)

  const isValid = computed(() => {
    if (!startDate.value || !endDate.value) return false
    return new Date(startDate.value) <= new Date(endDate.value)
  })

  const rangeInDays = computed(() => {
    if (!isValid.value) return 0
    const diff = new Date(endDate.value) - new Date(startDate.value)
    return Math.ceil(diff / (1000 * 60 * 60 * 24))
  })

  function setRange(start, end) {
    startDate.value = start
    endDate.value = end
  }

  function setToday() {
    const today = new Date()
    startDate.value = formatDate(today)
    endDate.value = formatDate(today)
  }

  function setLastDays(days) {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - days)
    
    startDate.value = formatDate(start)
    endDate.value = formatDate(end)
  }

  function setThisWeek() {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - start.getDay())
    
    startDate.value = formatDate(start)
    endDate.value = formatDate(end)
  }

  function setThisMonth() {
    const end = new Date()
    const start = new Date(end.getFullYear(), end.getMonth(), 1)
    
    startDate.value = formatDate(start)
    endDate.value = formatDate(end)
  }

  function formatDate(date) {
    return date.toISOString().split('T')[0]
  }

  function reset() {
    startDate.value = options.startDate || null
    endDate.value = options.endDate || null
  }

  return {
    startDate,
    endDate,
    isValid,
    rangeInDays,
    setRange,
    setToday,
    setLastDays,
    setThisWeek,
    setThisMonth,
    reset
  }
}
