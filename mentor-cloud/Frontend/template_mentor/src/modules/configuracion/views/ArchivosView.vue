<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Modal from '@/shared/components/ui/Modal.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import { useModal } from '@/shared/composables/useModal'
import { useApi } from '@/shared/composables/useApi'

const { isOpen: isCreateOpen, open: openCreateModal, close: closeCreateModal } = useModal()
const { isOpen: isEditOpen, open: openEditModal, close: closeEditModal } = useModal()
const { loading, error, execute } = useApi()

const archivos = ref([])
const empresas = ref([])
const plantas = ref([])
const selectedRows = ref([])

const canEdit = computed(() => selectedRows.value.length === 1)
const canDelete = computed(() => selectedRows.value.length > 0)

const createForm = ref({
  plantaId: '',
  nombreTecnico: '',
  nombre: '',
  frecuenciaSegundos: ''
})

const editForm = ref({
  id: '',
  plantaId: '',
  nombreTecnico: '',
  nombre: '',
  frecuenciaSegundos: ''
})

const toggleRowSelection = (item) => {
  selectedRows.value = [item]
}

const isRowSelected = (item) => {
  return selectedRows.value.some(row => row.id === item.id)
}

const loadData = async () => {
  await execute(async () => {
    // Mock empresas
    empresas.value = [
      { id: 1, nombre: 'Industrias San Miguel' }
    ]

    // Mock plantas
    plantas.value = [
      { id: 1, nombre: 'Histórico Embotelladora San Miguel', empresaId: 1 }
    ]

    // Mock archivos
    archivos.value = [
      { 
        id: 1,
        nombreTecnico: 'energíaKR',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 2,
        nombreTecnico: 'tú KR',
        nombre: 'archivo oeeKR',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 3,
        nombreTecnico: 'M221',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 4,
        nombreTecnico: 'M251',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 5,
        nombreTecnico: 'MEL1',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 6,
        nombreTecnico: 'MEL2',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 7,
        nombreTecnico: 'MEL3',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      },
      { 
        id: 8,
        nombreTecnico: 'MEL5',
        nombre: '',
        frecuenciaSegundos: 300,
        plantaId: 1
      }
    ]
  })
}

const openCreate = () => {
  resetCreateForm()
  openCreateModal()
}

const openEdit = () => {
  if (selectedRows.value.length === 1) {
    const item = selectedRows.value[0]
    editForm.value = { ...item }
    openEditModal()
  }
}

const resetCreateForm = () => {
  createForm.value = {
    plantaId: '',
    nombreTecnico: '',
    nombre: '',
    frecuenciaSegundos: ''
  }
}

const handleCreate = async () => {
  await execute(async () => {
    // Create logic
    await loadData()
    closeCreateModal()
  })
}

const handleEdit = async () => {
  await execute(async () => {
    // Update logic
    const archivo = archivos.value.find(a => a.id === editForm.value.id)
    if (archivo) {
      Object.assign(archivo, editForm.value)
    }
    await loadData()
    closeEditModal()
    selectedRows.value = []
  })
}

// Filtros
const empresaFiltro = ref(1)
const plantaFiltro = ref(1)

const archivosFiltrados = computed(() => {
  let filtered = archivos.value

  if (plantaFiltro.value) {
    filtered = filtered.filter(a => a.plantaId === parseInt(plantaFiltro.value))
  }

  return filtered
})

const plantasFiltradas = computed(() => {
  if (!empresaFiltro.value) {
    return plantas.value
  }
  return plantas.value.filter(p => p.empresaId === parseInt(empresaFiltro.value))
})

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="archivos-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">ARCHIVOS</h1>
      </div>
    </div>

    <Card class="content-card">
      <div class="action-bar">
        <Button @click="openCreate" variant="primary" size="sm">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          AGREGAR
        </Button>
        <Button 
          @click="openEdit"
          :disabled="!canEdit"
          variant="secondary" 
          size="sm"
        >
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
          </svg>
          EDITAR
        </Button>
      </div>

      <div class="filters-bar">
        <div class="filter-item">
          <label class="filter-label">Empresa</label>
          <select v-model="empresaFiltro" class="field-select-filter">
            <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
              {{ empresa.nombre }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Planta</label>
          <select v-model="plantaFiltro" class="field-select-filter">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
      </div>

      <Loading v-if="loading" />
      <Alert v-else-if="error" type="error" :message="error" />

      <div v-else class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre Técnico</th>
              <th>Nombre</th>
              <th>Frecuencia(Seg.)</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(archivo, index) in archivosFiltrados" 
              :key="archivo.id"
              @click="toggleRowSelection(archivo)"
              :class="{ 'row-selected': isRowSelected(archivo) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(archivo)"
                  @click.stop="toggleRowSelection(archivo)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ archivo.nombreTecnico }}</td>
              <td>{{ archivo.nombre }}</td>
              <td>{{ archivo.frecuenciaSegundos }}</td>
            </tr>
          </tbody>
        </table>

        <div class="pagination">
          <button class="pagination-btn">ANTERIOR</button>
          <span class="pagination-info">1</span>
          <button class="pagination-btn">PRÓXIMO</button>
        </div>
      </div>
    </Card>

    <!-- Modal Crear -->
    <Modal v-model="isCreateOpen" @close="closeCreateModal" title="CREAR ARCHIVO" size="md">
      <form @submit.prevent="handleCreate" class="form-container">
        <div class="form-content">
          <div class="form-row">
            <label class="form-label">Planta</label>
            <select v-model="createForm.plantaId" class="field-select" required>
              <option value="" disabled>Seleccione planta</option>
              <option v-for="planta in plantas" :key="planta.id" :value="planta.id">
                {{ planta.nombre }}
              </option>
            </select>
          </div>

          <div class="form-row">
            <label class="form-label">Nombre técnico</label>
            <input 
              v-model="createForm.nombreTecnico" 
              type="text" 
              class="field-input" 
              placeholder="Ingrese nombre"
              required
            />
          </div>

          <div class="form-row">
            <label class="form-label">Nombre</label>
            <input 
              v-model="createForm.nombre" 
              type="text" 
              class="field-input" 
              placeholder="Ingrese nombre"
            />
          </div>

          <div class="form-row">
            <label class="form-label">Frecuencia en Segundos</label>
            <input 
              v-model="createForm.frecuenciaSegundos" 
              type="number" 
              class="field-input" 
              placeholder="Ingrese Frecuencia"
              required
            />
          </div>
        </div>

        <div class="form-footer">
          <Button type="submit" variant="primary" size="md" :loading="loading">
            GUARDAR
          </Button>
          <Button type="button" @click="closeCreateModal" variant="secondary" size="md">
            CANCELAR
          </Button>
        </div>
      </form>
    </Modal>

    <!-- Modal Editar -->
    <Modal v-model="isEditOpen" @close="closeEditModal" title="EDITAR ARCHIVO" size="md">
      <form @submit.prevent="handleEdit" class="form-container">
        <div class="form-content">
          <div class="form-row">
            <label class="form-label">Planta</label>
            <select v-model="editForm.plantaId" class="field-select" required>
              <option value="" disabled>Seleccione planta</option>
              <option v-for="planta in plantas" :key="planta.id" :value="planta.id">
                {{ planta.nombre }}
              </option>
            </select>
          </div>

          <div class="form-row">
            <label class="form-label">Nombre técnico</label>
            <input 
              v-model="editForm.nombreTecnico" 
              type="text" 
              class="field-input" 
              required
            />
          </div>

          <div class="form-row">
            <label class="form-label">Nombre</label>
            <input 
              v-model="editForm.nombre" 
              type="text" 
              class="field-input" 
              placeholder="Ingrese nombre"
            />
          </div>

          <div class="form-row">
            <label class="form-label">Frecuencia en Segundos</label>
            <input 
              v-model="editForm.frecuenciaSegundos" 
              type="number" 
              class="field-input" 
              required
            />
          </div>
        </div>

        <div class="form-footer">
          <Button type="submit" variant="primary" size="md" :loading="loading">
            GUARDAR
          </Button>
          <Button type="button" @click="closeEditModal" variant="secondary" size="md">
            CANCELAR
          </Button>
        </div>
      </form>
    </Modal>
  </div>
</template>

<style scoped>
.archivos-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.view-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.view-title {
  font-size: 1.875rem;
  font-weight: 700;
  color: #111827;
}

.content-card {
  border-radius: 0;
  border-top: none;
}

.action-bar {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  background-color: #1e3a8a;
  border-radius: 0.5rem 0.5rem 0 0;
}

.filters-bar {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  background-color: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.field-select-filter {
  padding: 0.625rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
  cursor: pointer;
  min-width: 250px;
}

.field-select-filter:hover {
  border-color: #9ca3af;
}

.field-select-filter:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  font-size: 0.875rem;
}

.data-table thead {
  background-color: #f3f4f6;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid #e5e7eb;
}

.data-table tbody tr {
  transition: all 0.15s;
  cursor: pointer;
}

.data-table tbody tr:hover {
  background-color: #eff6ff;
}

.data-table tbody tr:nth-child(even) {
  background-color: #f9fafb;
}

.data-table tbody tr:nth-child(even):hover {
  background-color: #eff6ff;
}

.data-table tbody tr.row-selected {
  background-color: #dbeafe;
  border-left: 4px solid #2563eb;
}

.data-table td {
  padding: 0.75rem 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
}

.data-table td.table-name {
  font-weight: 600;
  color: #111827;
}

.checkbox-cell {
  width: 3rem;
  text-align: center;
}

.table-checkbox {
  width: 1.25rem;
  height: 1.25rem;
  color: #2563eb;
  border-color: #d1d5db;
  border-radius: 0.25rem;
  cursor: pointer;
}

.table-checkbox:focus {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 1rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.pagination-btn {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  background-color: white;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.pagination-btn:hover {
  background-color: #f9fafb;
}

.pagination-info {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.form-container {
  padding: 0;
}

.form-content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.5rem;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.field-select {
  width: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
  cursor: pointer;
}

.field-select:hover {
  border-color: #9ca3af;
}

.field-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.field-input {
  width: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
}

.field-input:hover {
  border-color: #9ca3af;
}

.field-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.field-input::placeholder {
  color: #9ca3af;
}

.form-footer {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  padding: 1rem 1.5rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.form-footer button {
  min-width: 120px;
}
</style>
