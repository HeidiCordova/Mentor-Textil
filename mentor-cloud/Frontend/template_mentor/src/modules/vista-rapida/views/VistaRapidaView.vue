<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import { useApi } from '@/shared/composables/useApi'

const { loading, error, execute } = useApi()

const tabActiva = ref('FACTOR DE CALIFICACIÓN')

const empresas = ref([
  { id: 1, nombre: 'Industrias San Miguel' }
])

const plantas = ref([
  { id: 1, empresaId: 1, nombre: 'ISM Arequipa' }
])

const empresaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)

const plantasFiltradas = computed(() => {
  if (!empresaSeleccionada.value) return []
  return plantas.value.filter(p => p.empresaId === parseInt(empresaSeleccionada.value))
})

onMounted(() => {
  
})
</script>

<template>
  <div class="vista-rapida-view">
    <div class="page-header">
      <h1 class="page-title">VISTA RÁPIDA</h1>
    </div>

    <Card class="content-card">
      <div class="tabs-header">
        <button 
          :class="['tab-button', { 'tab-active': tabActiva === 'FACTOR DE CALIFICACIÓN' }]"
          @click="tabActiva = 'FACTOR DE CALIFICACIÓN'"
        >
          FACTOR DE CALIFICACIÓN
        </button>
        <button 
          :class="['tab-button', { 'tab-active': tabActiva === 'OEE POR TURNO' }]"
          @click="tabActiva = 'OEE POR TURNO'"
        >
          OEE POR TURNO
        </button>
      </div>

      <div v-if="tabActiva === 'FACTOR DE CALIFICACIÓN'" class="tab-content">
        <div class="filtros-row">
          <div class="filter-item">
            <label class="filter-label">Empresa</label>
            <select v-model="empresaSeleccionada" class="filter-select">
              <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
                {{ empresa.nombre }}
              </option>
            </select>
          </div>

          <div class="filter-item">
            <label class="filter-label">Planta</label>
            <select v-model="plantaSeleccionada" class="filter-select" :disabled="!empresaSeleccionada">
              <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
                {{ planta.nombre }}
              </option>
            </select>
          </div>
        </div>

        <div class="empty-state">
          <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
          </svg>
          <p class="empty-text">Factor de Calificación</p>
        </div>
      </div>

      <div v-else class="tab-content">
        <div class="filtros-row">
          <div class="filter-item">
            <label class="filter-label">Empresa</label>
            <select v-model="empresaSeleccionada" class="filter-select">
              <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
                {{ empresa.nombre }}
              </option>
            </select>
          </div>

          <div class="filter-item">
            <label class="filter-label">Planta</label>
            <select v-model="plantaSeleccionada" class="filter-select" :disabled="!empresaSeleccionada">
              <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
                {{ planta.nombre }}
              </option>
            </select>
          </div>
        </div>

        <div class="empty-state">
          <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <p class="empty-text">OEE por Turno</p>
        </div>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.vista-rapida-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding-bottom: 2rem;
}

.page-header {
  background-color: #1e3a8a;
  padding: 1rem 2rem;
  border-radius: 0;
}

.page-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: white;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.content-card {
  border-radius: 0;
  border: 1px solid #e5e7eb;
}

.tabs-header {
  display: flex;
  background-color: #f3f4f6;
  border-bottom: 2px solid #e5e7eb;
}

.tab-button {
  flex: 1;
  padding: 0.75rem 1rem;
  background-color: transparent;
  color: #374151;
  border: none;
  cursor: pointer;
  font-weight: 600;
  font-size: 0.75rem;
  transition: all 0.2s;
  border-bottom: 3px solid transparent;
}

.tab-button:hover {
  background-color: #e5e7eb;
}

.tab-active {
  background-color: white;
  color: #1e3a8a;
  border-bottom: 3px solid #1e3a8a;
}

.tab-content {
  padding: 1.5rem;
}

.filtros-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.filter-select {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background-color: white;
  color: #374151;
  cursor: pointer;
}

.filter-select:disabled {
  background-color: #f3f4f6;
  cursor: not-allowed;
  opacity: 0.6;
}

.filter-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.3);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
}

.empty-icon {
  width: 4rem;
  height: 4rem;
  color: #d1d5db;
  margin-bottom: 1rem;
}

.empty-text {
  font-size: 1rem;
  color: #6b7280;
  margin: 0;
}

@media (max-width: 768px) {
  .filtros-row {
    grid-template-columns: 1fr;
  }

  .tabs-header {
    flex-direction: column;
  }

  .tab-button {
    border-bottom: none;
    border-left: 3px solid transparent;
  }

  .tab-active {
    border-bottom: none;
    border-left: 3px solid #1e3a8a;
  }
}
</style>
