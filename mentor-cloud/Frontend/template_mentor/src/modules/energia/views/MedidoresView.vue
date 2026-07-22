<script setup>
import { ref, computed, onMounted } from 'vue'
import { energyService } from '@/api/services/energy.service'
import { useModal } from '@/shared/composables/useModal'

const meters = ref([])
const loading = ref(false)
const searchQuery = ref('')
const filterActivo = ref('')

const form = ref({ device_id: '', meter_id: '', nombre: '', ubicacion: '', planta_id: '' })
const editingId = ref(null)
const formError = ref('')

const createModal = useModal()
const confirmModal = useModal()
const confirmTarget = ref(null)

const filteredMeters = computed(() => {
  let result = meters.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(m =>
      (m.meter_id || '').toLowerCase().includes(q) ||
      (m.device_id || '').toLowerCase().includes(q) ||
      (m.nombre || '').toLowerCase().includes(q) ||
      (m.ubicacion || '').toLowerCase().includes(q)
    )
  }
  if (filterActivo.value !== '') {
    const activo = filterActivo.value === 'true'
    result = result.filter(m => m.activo === activo)
  }
  return result
})

const stats = computed(() => {
  const total = meters.value.length
  const activos = meters.value.filter(m => m.activo).length
  const inactivos = total - activos
  const plantas = new Set(meters.value.map(m => m.planta_id).filter(Boolean)).size
  return { total, activos, inactivos, plantas }
})

async function fetchMeters() {
  loading.value = true
  try {
    const res = await energyService.getMeters()
    meters.value = res.data || []
  } catch {
    meters.value = []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = { device_id: '', meter_id: '', nombre: '', ubicacion: '', planta_id: '' }
  editingId.value = null
  formError.value = ''
}

function openCreate() {
  resetForm()
  createModal.open()
}

function openEdit(meter) {
  form.value = {
    device_id: meter.device_id,
    meter_id: meter.meter_id,
    nombre: meter.nombre || '',
    ubicacion: meter.ubicacion || '',
    planta_id: meter.planta_id || ''
  }
  editingId.value = meter.id
  formError.value = ''
  createModal.open()
}

async function submitForm() {
  formError.value = ''
  if (!form.value.device_id || !form.value.meter_id) {
    formError.value = 'device_id y meter_id son obligatorios'
    return
  }

  loading.value = true
  try {
    if (editingId.value) {
      await energyService.updateMeter(editingId.value, {
        nombre: form.value.nombre || null,
        ubicacion: form.value.ubicacion || null
      })
    } else {
      const payload = {
        device_id: form.value.device_id,
        meter_id: form.value.meter_id,
        nombre: form.value.nombre,
        ubicacion: form.value.ubicacion
      }
      if (form.value.planta_id) payload.planta_id = parseInt(form.value.planta_id)
      await energyService.createMeter(payload)
    }
    createModal.close()
    await fetchMeters()
  } catch (e) {
    formError.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

function requestDelete(meter) {
  confirmTarget.value = meter
  confirmModal.open()
}

async function executeDelete() {
  loading.value = true
  try {
    await energyService.deleteMeter(confirmTarget.value.id)
    confirmModal.close()
    await fetchMeters()
  } catch (e) {
    formError.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

async function toggleActivo(meter) {
  loading.value = true
  try {
    await energyService.updateMeter(meter.id, { activo: !meter.activo })
    await fetchMeters()
  } catch (e) {
    formError.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('es-PE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchMeters)
</script>

<template>
  <div class="medidores-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">Medidores de Energia</h1>
        <p class="view-subtitle">Registro y gestion de medidores MEATROL conectados via Modbus RTU</p>
      </div>
      <button class="btn btn-primary" @click="openCreate">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M12 5v14m-7-7h14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
        Registrar medidor
      </button>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-label">Total</div>
      </div>
      <div class="stat-card stat-activo">
        <div class="stat-value">{{ stats.activos }}</div>
        <div class="stat-label">Activos</div>
      </div>
      <div class="stat-card stat-inactivo">
        <div class="stat-value">{{ stats.inactivos }}</div>
        <div class="stat-label">Inactivos</div>
      </div>
      <div class="stat-card stat-plantas">
        <div class="stat-value">{{ stats.plantas }}</div>
        <div class="stat-label">Plantas</div>
      </div>
    </div>

    <div class="table-toolbar">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Buscar por meter_id, device, nombre o ubicacion..."
        class="search-input"
      />
      <select v-model="filterActivo" class="filter-select">
        <option value="">Todos</option>
        <option value="true">Activos</option>
        <option value="false">Inactivos</option>
      </select>
    </div>

    <div class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>Estado</th>
            <th>Meter ID</th>
            <th>Device ID</th>
            <th>Nombre</th>
            <th>Ubicacion</th>
            <th>Planta</th>
            <th>Creado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !filteredMeters.length">
            <td colspan="8" class="table-empty">Cargando...</td>
          </tr>
          <tr v-else-if="!filteredMeters.length">
            <td colspan="8" class="table-empty">No hay medidores registrados</td>
          </tr>
          <tr v-for="m in filteredMeters" :key="m.id" :class="{ 'row-inactive': !m.activo }">
            <td>
              <span :class="['status-badge', m.activo ? 'status-activo' : 'status-inactivo']">
                {{ m.activo ? 'Activo' : 'Inactivo' }}
              </span>
            </td>
            <td class="font-mono text-sm text-secondary-400">{{ m.meter_id }}</td>
            <td class="font-mono text-sm">{{ m.device_id }}</td>
            <td>{{ m.nombre || '-' }}</td>
            <td>{{ m.ubicacion || '-' }}</td>
            <td>{{ m.planta_id || '-' }}</td>
            <td>{{ formatDate(m.created_at) }}</td>
            <td>
              <div class="action-buttons">
                <button class="btn-icon" title="Editar" @click="openEdit(m)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" stroke="currentColor" stroke-width="2"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" stroke="currentColor" stroke-width="2"/></svg>
                </button>
                <button
                  :class="['btn-icon', m.activo ? 'btn-icon-warning' : 'btn-icon-success']"
                  :title="m.activo ? 'Desactivar' : 'Activar'"
                  @click="toggleActivo(m)"
                >
                  <svg v-if="m.activo" width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M18.36 6.64A9 9 0 115.64 18.36 9 9 0 0118.36 6.64zM6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
                  <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M9 11l3 3L22 4M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </button>
                <button class="btn-icon btn-icon-danger" title="Eliminar" @click="requestDelete(m)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M10 11v6M14 11v6" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal crear/editar -->
    <Teleport to="body">
      <div v-if="createModal.isOpen.value" class="modal-overlay" @click.self="createModal.close()">
        <div class="modal-content">
          <div class="modal-header">
            <h2>{{ editingId ? 'Editar medidor' : 'Registrar medidor' }}</h2>
            <button class="btn-close" @click="createModal.close()">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
            </button>
          </div>
          <form @submit.prevent="submitForm" class="modal-body">
            <div v-if="formError" class="form-error">{{ formError }}</div>
            <div class="form-row">
              <div class="form-group">
                <label>Device ID</label>
                <input v-model="form.device_id" type="text" placeholder="jetson-planta-01" :disabled="!!editingId" />
                <span class="form-hint">Dispositivo edge que lee el medidor</span>
              </div>
              <div class="form-group">
                <label>Meter ID</label>
                <input v-model="form.meter_id" type="text" placeholder="mc60-linea-01" :disabled="!!editingId" />
                <span class="form-hint">Identificador Modbus del medidor</span>
              </div>
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>Nombre</label>
                <input v-model="form.nombre" type="text" placeholder="MEATROL MC60 - Linea 1" />
              </div>
              <div class="form-group">
                <label>Ubicacion</label>
                <input v-model="form.ubicacion" type="text" placeholder="Tablero electrico Nave A" />
              </div>
            </div>
            <div class="form-group" v-if="!editingId">
              <label>Planta ID</label>
              <input v-model="form.planta_id" type="number" placeholder="Opcional" />
              <span class="form-hint">ID de planta asociada (opcional)</span>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" @click="createModal.close()">Cancelar</button>
              <button type="submit" class="btn btn-primary" :disabled="loading">
                {{ editingId ? 'Guardar' : 'Registrar' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>

    <!-- Modal confirmacion eliminar -->
    <Teleport to="body">
      <div v-if="confirmModal.isOpen.value" class="modal-overlay" @click.self="confirmModal.close()">
        <div class="modal-content modal-confirm">
          <div class="modal-header">
            <h2>Eliminar medidor</h2>
          </div>
          <div class="modal-body">
            <p class="confirm-device">{{ confirmTarget?.meter_id }} - {{ confirmTarget?.nombre || confirmTarget?.device_id }}</p>
            <p class="confirm-message">Se eliminara permanentemente este medidor y su historial de snapshots asociado. Esta accion no se puede deshacer.</p>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="confirmModal.close()">Cancelar</button>
            <button class="btn btn-danger" @click="executeDelete" :disabled="loading">Eliminar</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.medidores-view { @apply p-6; }
.view-header { @apply flex items-center justify-between mb-6; }
.view-title { @apply text-2xl font-bold text-white; }
.view-subtitle { @apply text-sm text-gray-400 mt-1; }

.stats-grid { @apply grid grid-cols-4 gap-4 mb-6; }
.stat-card { @apply bg-primary-800 rounded-lg p-4 border border-primary-700; }
.stat-value { @apply text-2xl font-bold text-white; }
.stat-label { @apply text-xs text-gray-400 mt-1 uppercase tracking-wide; }
.stat-activo .stat-value { @apply text-emerald-400; }
.stat-inactivo .stat-value { @apply text-gray-400; }
.stat-plantas .stat-value { @apply text-blue-400; }

.table-toolbar { @apply flex gap-3 mb-4; }
.search-input {
  @apply flex-1 bg-primary-800 border border-primary-700 rounded-lg px-4 py-2;
  @apply text-white placeholder-gray-500 text-sm focus:outline-none focus:border-secondary-500;
}
.filter-select {
  @apply bg-primary-800 border border-primary-700 rounded-lg px-3 py-2;
  @apply text-white text-sm focus:outline-none focus:border-secondary-500;
}

.table-container { @apply bg-primary-800 rounded-lg border border-primary-700 overflow-hidden; }
.data-table { @apply w-full; }
.data-table th {
  @apply text-left px-4 py-3 text-xs uppercase tracking-wider text-gray-400;
  @apply bg-primary-900 border-b border-primary-700;
}
.data-table td { @apply px-4 py-3 text-sm text-gray-300 border-b border-primary-700; }
.data-table tr:last-child td { @apply border-b-0; }
.row-inactive { @apply opacity-50; }
.table-empty { @apply text-center py-8 text-gray-500; }

.status-badge { @apply inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium; }
.status-activo { @apply bg-emerald-900/50 text-emerald-400; }
.status-inactivo { @apply bg-gray-700/50 text-gray-400; }

.action-buttons { @apply flex items-center gap-1; }
.btn-icon { @apply p-1.5 rounded text-gray-400 hover:text-white hover:bg-primary-700 transition-colors; }
.btn-icon-warning { @apply hover:text-amber-400; }
.btn-icon-danger { @apply hover:text-red-400; }
.btn-icon-success { @apply hover:text-emerald-400; }

.btn { @apply inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors; }
.btn-primary { @apply bg-secondary-600 text-white hover:bg-secondary-700; }
.btn-secondary { @apply bg-primary-700 text-gray-300 hover:bg-primary-600; }
.btn-danger { @apply bg-red-600 text-white hover:bg-red-700; }
.btn:disabled { @apply opacity-50 cursor-not-allowed; }

.modal-overlay { @apply fixed inset-0 bg-black/60 flex items-center justify-center z-50; }
.modal-content { @apply bg-primary-800 rounded-xl border border-primary-700 w-full max-w-lg shadow-2xl; }
.modal-header { @apply flex items-center justify-between px-6 py-4 border-b border-primary-700; }
.modal-header h2 { @apply text-lg font-semibold text-white; }
.btn-close { @apply text-gray-400 hover:text-white transition-colors; }
.modal-body { @apply px-6 py-4; }
.modal-footer { @apply flex justify-end gap-3 px-6 py-4 border-t border-primary-700; }

.form-group { @apply mb-4; }
.form-group label { @apply block text-sm font-medium text-gray-300 mb-1; }
.form-group input, .form-group select {
  @apply w-full bg-primary-900 border border-primary-600 rounded-lg px-3 py-2;
  @apply text-white text-sm focus:outline-none focus:border-secondary-500;
}
.form-group input:disabled { @apply opacity-50 cursor-not-allowed; }
.form-hint { @apply text-xs text-gray-500 mt-1 block; }
.form-row { @apply grid grid-cols-2 gap-4; }
.form-error { @apply bg-red-900/30 border border-red-700 text-red-400 rounded-lg px-4 py-2 mb-4 text-sm; }

.modal-confirm .modal-body { @apply space-y-3; }
.confirm-device { @apply text-white font-mono text-sm bg-primary-900 rounded px-3 py-2; }
.confirm-message { @apply text-gray-400 text-sm; }
</style>
