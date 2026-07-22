import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useFiltersStore = defineStore('filters', () => {
  const dateRange = ref({
    start: null,
    end: null
  })
  
  const selectedCompany = ref(null)
  const selectedPlant = ref(null)
  const selectedLine = ref(null)
  const selectedDevice = ref(null)
  const selectedVariable = ref(null)
  const aggregation = ref('5 minutos')

  function setDateRange(start, end) {
    dateRange.value = { start, end }
  }

  function setCompany(company) {
    selectedCompany.value = company
    selectedPlant.value = null
    selectedLine.value = null
    selectedDevice.value = null
  }

  function setPlant(plant) {
    selectedPlant.value = plant
    selectedLine.value = null
    selectedDevice.value = null
  }

  function setLine(line) {
    selectedLine.value = line
    selectedDevice.value = null
    aggregation.value = line?.mode === 'textil' ? '30 minutos' : '5 minutos'
  }

  function setDevice(device) {
    selectedDevice.value = device
  }

  function setVariable(variable) {
    selectedVariable.value = variable
  }

  function setAggregation(value) {
    aggregation.value = value
  }

  function reset() {
    dateRange.value = { start: null, end: null }
    selectedCompany.value = null
    selectedPlant.value = null
    selectedLine.value = null
    selectedDevice.value = null
    selectedVariable.value = null
    aggregation.value = '5 minutos'
  }

  return {
    dateRange,
    selectedCompany,
    selectedPlant,
    selectedLine,
    selectedDevice,
    selectedVariable,
    aggregation,
    setDateRange,
    setCompany,
    setPlant,
    setLine,
    setDevice,
    setVariable,
    setAggregation,
    reset
  }
})
