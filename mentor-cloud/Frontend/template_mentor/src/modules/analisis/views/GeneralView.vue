<script setup>
import { ref, onMounted } from 'vue'
import { useApi } from '@/shared/composables/useApi'
import { useFiltersStore } from '@/stores/filters'
import { analysisService } from '@/api/services/analysis.service'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import DateRangePicker from '@/shared/components/forms/DateRangePicker.vue'
import SelectFilter from '@/shared/components/forms/SelectFilter.vue'
import Loading from '@/shared/components/ui/Loading.vue'

const { loading, execute } = useApi()
const filtersStore = useFiltersStore()

const data = ref(null)

const companies = ref([
  { value: 1, label: 'Industrias San Miguel' }
])

const plants = ref([
  { value: 1, label: 'ISM Arequipa' },
  { value: 2, label: 'Histórico Embotelladora San Miguel' }
])

onMounted(async () => {
  await loadData()
})

async function loadData() {
  const { data: response } = await execute(() => analysisService.getGeneral({
    dateRange: filtersStore.dateRange,
    plant: filtersStore.selectedPlant
  }))
  
  if (response) {
    data.value = response
  }
}

function handleSearch() {
  loadData()
}
</script>

<template>
  <div class="analisis-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">Análisis General</h1>
        <p class="view-subtitle">Visualice datos de análisis general</p>
      </div>
      <Button variant="secondary">Exportar Paradas de Planta</Button>
    </div>

    <Card title="Filtros">
      <div class="filters-grid">
        <DateRangePicker
          v-model="filtersStore.dateRange"
          label="Fechas"
        />
        
        <SelectFilter
          v-model="filtersStore.selectedCompany"
          :options="companies"
          label="Compañía"
          placeholder="Seleccione compañía"
        />
        
        <SelectFilter
          v-model="filtersStore.selectedPlant"
          :options="plants"
          label="Planta"
          placeholder="Seleccione planta"
        />
        
        <div class="filter-actions">
          <Button @click="handleSearch">Buscar</Button>
        </div>
      </div>
    </Card>

    <Card title="Gráfico de Análisis">
      <Loading v-if="loading" />
      
      <div v-else-if="data" class="chart-container">
        <p class="text-center text-gray-500">Gráfico ECharts se mostrará aquí</p>
        <p class="text-center text-sm text-gray-400 mt-2">Datos cargados: {{ data.data?.length }} registros</p>
      </div>
      
      <div v-else class="empty-state">
        <p>No hay datos para mostrar</p>
        <p class="text-sm text-gray-500 mt-2">Configure los filtros y haga clic en Buscar</p>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.analisis-view {
  @apply space-y-6;
}

.view-header {
  @apply flex items-start justify-between;
}

.view-title {
  @apply text-2xl font-bold text-gray-900;
}

.view-subtitle {
  @apply text-gray-600 mt-1;
}

.filters-grid {
  @apply grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4;
}

.filter-actions {
  @apply flex items-end;
}

.chart-container {
  @apply py-12;
}

.empty-state {
  @apply text-center py-12;
}
</style>
