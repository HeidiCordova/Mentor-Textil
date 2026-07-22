<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useApi } from '@/shared/composables/useApi'
import { useModal } from '@/shared/composables/useModal'
import { useTable } from '@/shared/composables/useTable'
import { plantService } from '@/api/services/plant.service'
import companyService from '@/api/services/company.service'
import { useAuthStore } from '@/stores/auth'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Modal from '@/shared/components/ui/Modal.vue'
import FormField from '@/shared/components/forms/FormField.vue'
import Loading from '@/shared/components/ui/Loading.vue'

const authStore = useAuthStore()
const { loading, execute } = useApi()
const modal = useModal()
const table = useTable()

const empresas = ref([])
const editingPlant = ref(null)
const provisioning = ref(null)
const form = ref({
  nombre: '',
  empresa_id: '',
  lineas: 0
})

const empresaFiltro = computed(() => authStore.user?.empresa_id || null)

onMounted(async () => {
  const companiesRes = await companyService.getAll()
  empresas.value = companiesRes.data || []
  await loadPlants()
})

async function loadPlants() {
  const params = {}
  if (empresaFiltro.value) params.empresa_id = empresaFiltro.value
  const { data } = await execute(() => plantService.getAll(params))
  if (data) {
    table.setItems(data.data, data.total)
  }
}

function openCreateModal() {
  editingPlant.value = null
  form.value = { nombre: '', empresa_id: empresaFiltro.value || '', lineas: 0 }
  modal.open()
}

function openEditModal(plant) {
  editingPlant.value = plant
  form.value = { nombre: plant.nombre, empresa_id: plant.empresa_id || '', lineas: plant.lineas || 0 }
  modal.open()
}

async function handleSave() {
  const payload = {
    nombre: form.value.nombre,
    empresa_id: form.value.empresa_id ? parseInt(form.value.empresa_id) : null,
    lineas: form.value.lineas || 0
  }
  if (editingPlant.value) {
    await execute(() => plantService.update(editingPlant.value.id, payload))
  } else {
    await execute(() => plantService.create(payload))
  }
  modal.close()
  await loadPlants()
}

async function handleDelete(id) {
  if (confirm('Esta seguro de eliminar esta planta?')) {
    await execute(() => plantService.delete(id))
    await loadPlants()
  }
}

async function handleProvision(plant) {
  if (!confirm(`Provisionar BD dedicada para "${plant.nombre}"? Este proceso puede tardar unos segundos.`)) return
  provisioning.value = plant.id
  try {
    await plantService.provision(plant.id)
    await loadPlants()
  } catch (err) {
    alert(err.response?.data?.error || 'Error al provisionar')
  } finally {
    provisioning.value = null
  }
}

const expandedRow = ref(null)

function toggleDetail(id) {
  expandedRow.value = expandedRow.value === id ? null : id
}

function formatDate(iso) {
  if (!iso) return '-'
  return new Date(iso).toLocaleDateString('es-PE', { year: 'numeric', month: 'short', day: 'numeric' })
}

const isSuperadmin = computed(() => authStore.hasRole('superadmin'))
</script>

<template>
  <div class="planta-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">Gestión de Plantas</h1>
        <p class="view-subtitle">Administre las plantas del sistema</p>
      </div>
      <Button @click="openCreateModal">Nueva Planta</Button>
    </div>

    <Card>
      <Loading v-if="loading" />
      
      <div v-else-if="table.items.length > 0" class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Nombre</th>
              <th>Compañía</th>
              <th>Líneas</th>
              <th>Instancia</th>
              <th>Estado</th>
              <th>Acciones</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="plant in table.items" :key="plant.id">
              <td>{{ plant.id }}</td>
              <td>{{ plant.nombre }}</td>
              <td>{{ empresas.find(e => e.id === plant.empresa_id)?.nombre || '-' }}</td>
              <td>{{ plant.lineas }}</td>
              <td>
                <button v-if="plant.db_name" class="instance-badge instance-provisioned" @click="toggleDetail(plant.id)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <ellipse cx="12" cy="5" rx="9" ry="3"/>
                    <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
                    <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
                  </svg>
                  {{ plant.instance_type || 'shared' }}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ 'rotate-180': expandedRow === plant.id }">
                    <polyline points="6 9 12 15 18 9"/>
                  </svg>
                </button>
                <span v-else class="instance-badge instance-shared">master</span>
              </td>
              <td>
                <span :class="['status-badge', plant.activo ? 'status-active' : 'status-inactive']">
                  {{ plant.activo ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
              <td>
                <div class="action-buttons">
                  <button class="btn-icon" @click="openEditModal(plant)" title="Editar">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                      <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" stroke="currentColor" stroke-width="2"/>
                      <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" stroke="currentColor" stroke-width="2"/>
                    </svg>
                  </button>
                  <button
                    v-if="isSuperadmin && !plant.db_name"
                    class="btn-icon btn-icon-provision"
                    :disabled="provisioning === plant.id"
                    @click="handleProvision(plant)"
                    title="Provisionar BD"
                  >
                    <svg v-if="provisioning !== plant.id" width="18" height="18" viewBox="0 0 24 24" fill="none">
                      <ellipse cx="12" cy="5" rx="9" ry="3" stroke="currentColor" stroke-width="2"/>
                      <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" stroke="currentColor" stroke-width="2"/>
                      <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" stroke="currentColor" stroke-width="2"/>
                      <path d="M12 8v6m-3-3h6" stroke="currentColor" stroke-width="2"/>
                    </svg>
                    <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" class="animate-spin">
                      <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" opacity="0.25"/>
                      <path d="M12 2a10 10 0 019.95 9" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                    </svg>
                  </button>
                  <button class="btn-icon btn-icon-danger" @click="handleDelete(plant.id)" title="Eliminar">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                      <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" stroke="currentColor" stroke-width="2"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="expandedRow === plant.id && plant.db_name" class="detail-row">
              <td colspan="7">
                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="detail-label">BD</span>
                    <span class="detail-value">{{ plant.db_name }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">Host</span>
                    <span class="detail-value">{{ plant.db_host || 'localhost' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">Schemas</span>
                    <span class="detail-value">{{ plant.db_schemas?.length || 0 }} lineas</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">Provisionada</span>
                    <span class="detail-value">{{ formatDate(plant.provisioned_at) }}</span>
                  </div>
                  <div v-if="plant.instance_type === 'dedicated'" class="detail-item">
                    <span class="detail-label">RDS</span>
                    <span class="detail-value">{{ plant.rds_class || '-' }} / {{ plant.rds_region || '-' }}</span>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <div v-else class="empty-state">
        <p>No hay plantas registradas</p>
        <Button @click="openCreateModal" class="mt-4">Crear Primera Planta</Button>
      </div>
    </Card>

    <Modal v-model="modal.isOpen" :title="editingPlant ? 'Editar Planta' : 'Nueva Planta'">
      <form @submit.prevent="handleSave" class="form-grid">
        <FormField v-model="form.nombre" label="Nombre" required />
        <div class="form-field">
          <label class="form-label">Empresa</label>
          <select v-model="form.empresa_id" class="form-select" required>
            <option value="">Seleccionar empresa</option>
            <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
          </select>
        </div>
        <FormField v-model.number="form.lineas" label="Líneas" type="number" required />
      </form>

      <template #footer>
        <div class="modal-actions">
          <Button variant="ghost" @click="modal.close">Cancelar</Button>
          <Button @click="handleSave">Guardar</Button>
        </div>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.planta-view {
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

.table-container {
  @apply overflow-x-auto;
}

.data-table {
  @apply w-full;
}

.data-table thead {
  @apply bg-gray-50 border-b border-gray-200;
}

.data-table th {
  @apply px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider;
}

.data-table td {
  @apply px-4 py-4 whitespace-nowrap text-sm text-gray-900;
}

.data-table tbody tr {
  @apply border-b border-gray-200 hover:bg-gray-50 transition-colors;
}

.status-badge {
  @apply inline-flex px-2 py-1 text-xs font-semibold rounded-full;
}

.status-active {
  @apply bg-green-100 text-green-800;
}

.status-inactive {
  @apply bg-gray-100 text-gray-800;
}

.action-buttons {
  @apply flex items-center gap-2;
}

.btn-icon {
  @apply p-2 rounded-lg hover:bg-gray-100 text-gray-600 transition-colors;
}

.btn-icon-danger {
  @apply hover:bg-red-50 hover:text-red-600;
}

.btn-icon-provision {
  @apply hover:bg-indigo-50 hover:text-indigo-600;
}

.instance-badge {
  @apply inline-flex items-center gap-1 px-2 py-1 text-xs font-semibold rounded-full;
}

.instance-provisioned {
  @apply bg-indigo-100 text-indigo-800 cursor-pointer;
}

.instance-provisioned svg:last-child {
  transition: transform 0.2s ease;
}

.rotate-180 {
  transform: rotate(180deg);
}

.detail-row td {
  @apply bg-gray-50 px-4 py-3 border-b border-gray-200;
}

.detail-grid {
  @apply flex flex-wrap gap-6;
}

.detail-item {
  @apply flex flex-col gap-0.5;
}

.detail-label {
  @apply text-xs font-medium text-gray-500 uppercase tracking-wide;
}

.detail-value {
  @apply text-sm text-gray-900 font-mono;
}

.instance-shared {
  @apply bg-gray-100 text-gray-600;
}

.animate-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.empty-state {
  @apply text-center py-12;
}

.form-grid {
  @apply grid grid-cols-1 gap-4;
}

.modal-actions {
  @apply flex items-center justify-end gap-3;
}
</style>
