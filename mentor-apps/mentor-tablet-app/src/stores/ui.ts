import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUIStore = defineStore('ui', () => {
  const goToNowTrigger = ref(0)
  const registerStopTrigger = ref(0)
  const multiSelectMode = ref(false)
  const multiSelectedIds = ref<string[]>([])
  const calendarGoToTrigger = ref(0)
  const calendarTargetMs = ref(0)

  function triggerGoToNow() { goToNowTrigger.value++ }
  function triggerRegisterStop() { registerStopTrigger.value++ }
  function triggerGoToDate(ms: number) {
    calendarTargetMs.value = ms
    calendarGoToTrigger.value++
  }

  function toggleMultiSelect(): void {
    multiSelectMode.value = !multiSelectMode.value
    if (!multiSelectMode.value) multiSelectedIds.value = []
  }

  function toggleMultiSelectStop(stopId: string): void {
    const idx = multiSelectedIds.value.indexOf(stopId)
    if (idx >= 0) multiSelectedIds.value.splice(idx, 1)
    else multiSelectedIds.value.push(stopId)
  }

  function clearMultiSelected(): void {
    multiSelectedIds.value = []
  }

  return {
    goToNowTrigger,
    registerStopTrigger,
    calendarGoToTrigger,
    calendarTargetMs,
    multiSelectMode,
    multiSelectedIds,
    triggerGoToNow,
    triggerRegisterStop,
    triggerGoToDate,
    toggleMultiSelect,
    toggleMultiSelectStop,
    clearMultiSelected
  }
})
