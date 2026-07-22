<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { deviceService } from '@/api/services/device.service'
import { plantService } from '@/api/services/plant.service'
import { useApi } from '@/shared/composables/useApi'
import { useModal } from '@/shared/composables/useModal'

const devices = ref([])
const plantas = ref([])
const loading = ref(false)
const provisionedKey = ref(null)
const provisionedDeviceID = ref('')
const provisionedTipo = ref('oee')
const keyAction = ref('')
const searchQuery = ref('')
const filterEstado = ref('')

const form = ref({
  device_id: '',
  nombre: '',
  tipo: 'oee',
  planta_id: null,
  linea_id: null
})
const editingId = ref(null)
const formError = ref('')

const createModal = useModal()
const keyModal = useModal()
const confirmModal = useModal()
const confirmAction = ref(null)
const confirmTarget = ref(null)

const filteredDevices = computed(() => {
  let result = devices.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(d =>
      d.device_id.toLowerCase().includes(q) ||
      d.nombre.toLowerCase().includes(q) ||
      String(d.linea_id || '').includes(q)
    )
  }
  if (filterEstado.value) {
    result = result.filter(d => d.estado === filterEstado.value)
  }
  return result
})

const stats = computed(() => {
  const total = devices.value.length
  const online = devices.value.filter(d => d.estado === 'online').length
  const offline = devices.value.filter(d => d.estado === 'offline' && d.activo).length
  const revoked = devices.value.filter(d => !d.activo).length
  return { total, online, offline, revoked }
})

async function fetchDevices() {
  loading.value = true
  try {
    const res = await deviceService.getAll()
    devices.value = res.data || []
  } catch (e) {
    devices.value = []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = { device_id: '', nombre: '', tipo: 'oee', planta_id: null, linea_id: null }
  editingId.value = null
  formError.value = ''
}

function openCreate() {
  resetForm()
  createModal.open()
}

function openEdit(device) {
  form.value = {
    device_id: device.device_id,
    nombre: device.nombre,
    tipo: device.tipo || 'oee',
    planta_id: device.planta_id || null,
    linea_id: device.linea_id || null
  }
  editingId.value = device.id
  formError.value = ''
  createModal.open()
}

async function submitForm() {
  formError.value = ''
  if (!form.value.device_id || !form.value.nombre) {
    formError.value = 'device_id y nombre son obligatorios'
    return
  }
  if (form.value.tipo === 'energia' && !form.value.planta_id) {
    formError.value = 'Selecciona una planta para el medidor de energía'
    return
  }

  const payload = {
    device_id: form.value.device_id,
    nombre: form.value.nombre,
    planta_id: form.value.planta_id || undefined,
    linea_id: form.value.linea_id || undefined
  }

  loading.value = true
  try {
    if (editingId.value) {
      await deviceService.update(editingId.value, payload)
      createModal.close()
    } else {
      const res = await deviceService.create(payload)
      provisionedKey.value = res.api_key
      provisionedDeviceID.value = form.value.device_id
      provisionedTipo.value = form.value.tipo
      keyAction.value = 'Dispositivo aprovisionado'
      createModal.close()
      keyModal.open()
    }
    await fetchDevices()
  } catch (e) {
    formError.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

function requestRevoke(device) {
  confirmTarget.value = device
  confirmAction.value = 'revoke'
  confirmModal.open()
}

function requestRegenerate(device) {
  confirmTarget.value = device
  confirmAction.value = 'regenerate'
  confirmModal.open()
}

function requestActivate(device) {
  confirmTarget.value = device
  confirmAction.value = 'activate'
  confirmModal.open()
}

function requestDelete(device) {
  confirmTarget.value = device
  confirmAction.value = 'delete'
  confirmModal.open()
}

async function executeConfirm() {
  loading.value = true
  try {
    const id = confirmTarget.value.id
    if (confirmAction.value === 'revoke') {
      await deviceService.revoke(id)
    } else if (confirmAction.value === 'regenerate') {
      const res = await deviceService.regenerateKey(id)
      provisionedKey.value = res.api_key
      provisionedDeviceID.value = confirmTarget.value.device_id
      provisionedTipo.value = confirmTarget.value.tipo || 'oee'
      keyAction.value = 'Clave regenerada'
      confirmModal.close()
      keyModal.open()
    } else if (confirmAction.value === 'activate') {
      const res = await deviceService.activate(id)
      provisionedKey.value = res.api_key
      provisionedDeviceID.value = confirmTarget.value.device_id
      provisionedTipo.value = confirmTarget.value.tipo || 'oee'
      keyAction.value = 'Dispositivo reactivado'
      confirmModal.close()
      keyModal.open()
    } else if (confirmAction.value === 'delete') {
      await deviceService.deleteDevice(id)
    }
    confirmModal.close()
    await fetchDevices()
  } catch (e) {
    formError.value = e.response?.data?.error || e.message
  } finally {
    loading.value = false
  }
}

const confirmMessages = {
  revoke: 'Se revocara el acceso de este dispositivo. No podra conectarse al cloud hasta ser reactivado.',
  regenerate: 'Se generara una nueva API key. La clave anterior dejara de funcionar inmediatamente.',
  activate: 'Se reactivara el dispositivo con una nueva API key.',
  delete: 'Se eliminara permanentemente el dispositivo y su token API. Esta accion no se puede deshacer.'
}

const confirmTitles = {
  revoke: 'Revocar dispositivo',
  regenerate: 'Regenerar API Key',
  activate: 'Reactivar dispositivo',
  delete: 'Eliminar dispositivo'
}

let keyCopied = ref(false)
let snippetCopied = ref(false)

function copyKey() {
  if (provisionedKey.value) {
    navigator.clipboard.writeText(provisionedKey.value)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 2000)
  }
}

function envSnippet() {
  if (provisionedTipo.value === 'energia') {
    return `# Token a pegar en la UI del RPi (Config > Medidores > [medidor] > Token):\n${provisionedKey.value}\n\n# Device ID registrado en cloud:\n${provisionedDeviceID.value}`
  }
  return `DEVICE_ID=${provisionedDeviceID.value}\nCLOUD_API_KEY=${provisionedKey.value}`
}

function envSnippetLabel() {
  if (provisionedTipo.value === 'energia') {
    return 'Token para configurar en la UI del RPi'
  }
  return 'Líneas a actualizar en el .env del Jetson'
}

function copySnippet() {
  navigator.clipboard.writeText(envSnippet())
  snippetCopied.value = true
  setTimeout(() => { snippetCopied.value = false }, 2000)
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('es-PE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

let refreshInterval = null

async function fetchPlantas() {
  try {
    const res = await plantService.getAll()
    plantas.value = Array.isArray(res) ? res : (res?.data || [])
  } catch {
    plantas.value = []
  }
}

onMounted(() => {
  fetchDevices()
  fetchPlantas()
  refreshInterval = setInterval(fetchDevices, 15000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div class="dispositivos-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">Dispositivos</h1>
        <p class="view-subtitle">Aprovisionamiento y gestion de dispositivos edge</p>
      </div>
      <button class="btn btn-primary" @click="openCreate">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M12 5v14m-7-7h14" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
        Aprovisionar
      </button>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-label">Total</div>
      </div>
      <div class="stat-card stat-online">
        <div class="stat-value">{{ stats.online }}</div>
        <div class="stat-label">Online</div>
      </div>
      <div class="stat-card stat-offline">
        <div class="stat-value">{{ stats.offline }}</div>
        <div class="stat-label">Offline</div>
      </div>
      <div class="stat-card stat-revoked">
        <div class="stat-value">{{ stats.revoked }}</div>
        <div class="stat-label">Revocados</div>
      </div>
    </div>

    <div class="table-toolbar">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Buscar por device_id o nombre..."
        class="search-input"
      />
      <select v-model="filterEstado" class="filter-select">
        <option value="">Todos</option>
        <option value="online">Online</option>
        <option value="offline">Offline</option>
        <option value="revoked">Revocados</option>
      </select>
    </div>

    <div class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>Estado</th>
            <th>Device ID</th>
            <th>Linea ID</th>
            <th>Nombre</th>
            <th>Planta</th>
            <th>Ultimo contacto</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && !filteredDevices.length">
            <td colspan="7" class="table-empty">Cargando...</td>
          </tr>
          <tr v-else-if="!filteredDevices.length">
            <td colspan="7" class="table-empty">No hay dispositivos registrados</td>
          </tr>
          <tr v-for="d in filteredDevices" :key="d.id" :class="{ 'row-revoked': !d.activo }">
            <td>
              <span :class="['status-badge', `status-${d.estado}`]">{{ d.estado }}</span>
            </td>
            <td class="font-mono text-sm">{{ d.device_id }}</td>
            <td class="font-mono text-sm text-blue-400">{{ d.linea_id || '-' }}</td>
            <td>{{ d.nombre }}</td>
            <td>{{ d.linea_nombre || '-' }}</td>
            <td>{{ formatDate(d.last_seen_at) }}</td>
            <td>
              <div class="action-buttons">
                <button v-if="d.activo" class="btn-icon" title="Editar" @click="openEdit(d)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" stroke="currentColor" stroke-width="2"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" stroke="currentColor" stroke-width="2"/></svg>
                </button>
                <button v-if="d.activo" class="btn-icon btn-icon-warning" title="Regenerar key" @click="requestRegenerate(d)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 11-7.78 7.78 5.5 5.5 0 017.78-7.78zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </button>
                <button v-if="d.activo" class="btn-icon btn-icon-danger" title="Revocar" @click="requestRevoke(d)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M18.36 6.64A9 9 0 115.64 18.36 9 9 0 0118.36 6.64zM6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
                </button>
                <button v-if="!d.activo" class="btn-icon btn-icon-success" title="Reactivar" @click="requestActivate(d)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M9 11l3 3L22 4M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </button>
                <button class="btn-icon btn-icon-danger" title="Eliminar permanentemente" @click="requestDelete(d)">
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
            <h2>{{ editingId ? 'Editar dispositivo' : 'Aprovisionar dispositivo' }}</h2>
            <button class="btn-close" @click="createModal.close()">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M18 6L6 18M6 6l12 12" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
            </button>
          </div>
          <form @submit.prevent="submitForm" class="modal-body">
            <div v-if="formError" class="form-error">{{ formError }}</div>
            <div class="form-group">
              <label>Tipo de dispositivo</label>
              <select v-model="form.tipo" :disabled="!!editingId" class="form-select">
                <option value="oee">OEE — Jetson Orin / Edge gateway</option>
                <option value="energia">Medidor de energía — Raspberry Pi</option>
              </select>
            </div>
            <div class="form-group">
              <label>Device ID</label>
              <input
                v-model="form.device_id"
                type="text"
                :placeholder="form.tipo === 'energia' ? 'medidor-01-planta-gloria' : 'jetson-planta-linea'"
                :disabled="!!editingId"
              />
              <span class="form-hint">Identificador único en el cloud (debe coincidir con el device_id registrado)</span>
            </div>
            <div class="form-group">
              <label>Nombre</label>
              <input
                v-model="form.nombre"
                type="text"
                :placeholder="form.tipo === 'energia' ? 'Medidor Frigorifico Gloria — Tablero Principal' : 'Jetson Orin — Línea Telar 1'"
              />
            </div>
            <div v-if="form.tipo === 'energia'" class="form-group">
              <label>Planta <span class="required">*</span></label>
              <select v-model="form.planta_id" class="form-select">
                <option :value="null" disabled>Seleccionar planta...</option>
                <option v-for="p in plantas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
              </select>
              <span class="form-hint">Necesario para asociar datos al tenant correcto</span>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" @click="createModal.close()">Cancelar</button>
              <button type="submit" class="btn btn-primary" :disabled="loading">
                {{ editingId ? 'Guardar' : 'Aprovisionar' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>

    <!-- Modal API Key -->
    <Teleport to="body">
      <div v-if="keyModal.isOpen.value" class="modal-overlay">
        <div class="modal-content modal-key">
          <div class="modal-header">
            <h2>{{ keyAction }}</h2>
          </div>
          <div class="modal-body">
            <div class="key-warning">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" stroke="currentColor" stroke-width="2"/><line x1="12" y1="9" x2="12" y2="13" stroke="currentColor" stroke-width="2"/><line x1="12" y1="17" x2="12.01" y2="17" stroke="currentColor" stroke-width="2"/></svg>
              <span>Esta clave se muestra una unica vez. Copiala antes de cerrar.</span>
            </div>
            <div class="key-display">
              <code>{{ provisionedKey }}</code>
              <button class="btn-copy" @click="copyKey" :title="keyCopied ? 'Copiado' : 'Copiar'">
                <svg v-if="!keyCopied" width="18" height="18" viewBox="0 0 24 24" fill="none"><rect x="9" y="9" width="13" height="13" rx="2" stroke="currentColor" stroke-width="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke="currentColor" stroke-width="2"/></svg>
                <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none"><path d="M20 6L9 17l-5-5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
              </button>
            </div>
            <div class="key-env-block">
              <div class="key-env-header">
                <span>{{ envSnippetLabel() }}</span>
                <button class="btn-copy" @click="copySnippet" :title="snippetCopied ? 'Copiado' : 'Copiar snippet'">
                  <svg v-if="!snippetCopied" width="16" height="16" viewBox="0 0 24 24" fill="none"><rect x="9" y="9" width="13" height="13" rx="2" stroke="currentColor" stroke-width="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke="currentColor" stroke-width="2"/></svg>
                  <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M20 6L9 17l-5-5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </button>
              </div>
              <pre class="key-env-snippet">{{ envSnippet() }}</pre>
            </div>
            <div class="key-cmd-block">
              <template v-if="provisionedTipo === 'energia'">
                <span class="key-cmd-label">Pasos en la UI del RPi:</span>
                <pre class="key-env-snippet">1. Abrir http://&lt;IP-RPi&gt;:8090 → Config → Medidores
2. Editar el medidor correspondiente
3. Campo "Token": pegar el token copiado arriba
4. Campo "Nombre en la nube": usar el Device ID registrado
5. Guardar — el medidor comenzara a enviar con su token propio</pre>
              </template>
              <template v-else>
                <span class="key-cmd-label">Luego ejecutar en el Jetson:</span>
                <pre class="key-env-snippet">cd ~/Mentoredge/mentor-edge/infrastructure/docker
docker compose -f docker-compose.jetson-orin.yml up -d edge-gateway enviador resiliencia</pre>
              </template>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-primary" @click="keyModal.close()">Entendido</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Modal confirmacion -->
    <Teleport to="body">
      <div v-if="confirmModal.isOpen.value" class="modal-overlay" @click.self="confirmModal.close()">
        <div class="modal-content modal-confirm">
          <div class="modal-header">
            <h2>{{ confirmTitles[confirmAction] }}</h2>
          </div>
          <div class="modal-body">
            <p class="confirm-device">{{ confirmTarget?.device_id }} - {{ confirmTarget?.nombre }}</p>
            <p class="confirm-message">{{ confirmMessages[confirmAction] }}</p>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="confirmModal.close()">Cancelar</button>
            <button
              :class="['btn', (confirmAction === 'revoke' || confirmAction === 'delete') ? 'btn-danger' : 'btn-primary']"
              @click="executeConfirm"
              :disabled="loading"
            >
              Confirmar
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.dispositivos-view {
  @apply p-6;
}

.view-header {
  @apply flex items-center justify-between mb-6;
}

.view-title {
  @apply text-2xl font-bold text-white;
}

.view-subtitle {
  @apply text-sm text-gray-400 mt-1;
}

.stats-grid {
  @apply grid grid-cols-4 gap-4 mb-6;
}

.stat-card {
  @apply bg-primary-800 rounded-lg p-4 border border-primary-700;
}

.stat-value {
  @apply text-2xl font-bold text-white;
}

.stat-label {
  @apply text-xs text-gray-400 mt-1 uppercase tracking-wide;
}

.stat-online .stat-value { @apply text-emerald-400; }
.stat-offline .stat-value { @apply text-gray-400; }
.stat-revoked .stat-value { @apply text-red-400; }

.table-toolbar {
  @apply flex gap-3 mb-4;
}

.search-input {
  @apply flex-1 bg-primary-800 border border-primary-700 rounded-lg px-4 py-2;
  @apply text-white placeholder-gray-500 text-sm;
  @apply focus:outline-none focus:border-secondary-500;
}

.filter-select {
  @apply bg-primary-800 border border-primary-700 rounded-lg px-3 py-2;
  @apply text-white text-sm;
  @apply focus:outline-none focus:border-secondary-500;
}

.table-container {
  @apply bg-primary-800 rounded-lg border border-primary-700 overflow-hidden;
}

.data-table {
  @apply w-full;
}

.data-table th {
  @apply text-left px-4 py-3 text-xs uppercase tracking-wider text-gray-400;
  @apply bg-primary-900 border-b border-primary-700;
}

.data-table td {
  @apply px-4 py-3 text-sm text-gray-300 border-b border-primary-700;
}

.data-table tr:last-child td {
  @apply border-b-0;
}

.row-revoked {
  @apply opacity-50;
}

.table-empty {
  @apply text-center py-8 text-gray-500;
}

.status-badge {
  @apply inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium;
}

.status-online {
  @apply bg-emerald-900/50 text-emerald-400;
}

.status-offline {
  @apply bg-gray-700/50 text-gray-400;
}

.status-revoked {
  @apply bg-red-900/50 text-red-400;
}

.action-buttons {
  @apply flex items-center gap-1;
}

.btn-icon {
  @apply p-1.5 rounded text-gray-400 hover:text-white hover:bg-primary-700 transition-colors;
}

.btn-icon-warning { @apply hover:text-amber-400; }
.btn-icon-danger { @apply hover:text-red-400; }
.btn-icon-success { @apply hover:text-emerald-400; }

.btn {
  @apply inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors;
}

.btn-primary {
  @apply bg-secondary-600 text-white hover:bg-secondary-700;
}

.btn-secondary {
  @apply bg-primary-700 text-gray-300 hover:bg-primary-600;
}

.btn-danger {
  @apply bg-red-600 text-white hover:bg-red-700;
}

.btn:disabled {
  @apply opacity-50 cursor-not-allowed;
}

.modal-overlay {
  @apply fixed inset-0 bg-black/60 flex items-center justify-center z-50;
}

.modal-content {
  @apply bg-primary-800 rounded-xl border border-primary-700 w-full max-w-lg shadow-2xl;
}

.modal-header {
  @apply flex items-center justify-between px-6 py-4 border-b border-primary-700;
}

.modal-header h2 {
  @apply text-lg font-semibold text-white;
}

.btn-close {
  @apply text-gray-400 hover:text-white transition-colors;
}

.modal-body {
  @apply px-6 py-4;
}

.modal-footer {
  @apply flex justify-end gap-3 px-6 py-4 border-t border-primary-700;
}

.form-group {
  @apply mb-4;
}

.form-group label {
  @apply block text-sm font-medium text-gray-300 mb-1;
}

.form-group input,
.form-group select {
  @apply w-full bg-primary-900 border border-primary-600 rounded-lg px-3 py-2;
  @apply text-white text-sm;
  @apply focus:outline-none focus:border-secondary-500;
}

.form-group input:disabled {
  @apply opacity-50 cursor-not-allowed;
}

.form-hint {
  @apply text-xs text-gray-500 mt-1 block;
}

.form-row {
  @apply grid grid-cols-2 gap-4;
}

.form-error {
  @apply bg-red-900/30 border border-red-700 text-red-400 rounded-lg px-4 py-2 mb-4 text-sm;
}

.modal-key .modal-body {
  @apply space-y-4;
}

.key-warning {
  @apply flex items-center gap-3 bg-amber-900/30 border border-amber-700 rounded-lg px-4 py-3;
  @apply text-amber-400 text-sm;
}

.key-display {
  @apply flex items-center gap-2 bg-primary-900 border border-primary-600 rounded-lg p-3;
}

.key-display code {
  @apply flex-1 text-secondary-400 text-sm font-mono break-all;
}

.btn-copy {
  @apply flex-shrink-0 p-2 text-gray-400 hover:text-white transition-colors rounded hover:bg-primary-700;
}

.key-env-block {
  @apply bg-gray-900 border border-gray-700 rounded-lg p-3 space-y-2;
}

.key-env-header {
  @apply flex items-center justify-between text-xs text-gray-400;
}

.key-env-header code {
  @apply text-secondary-400;
}

.key-env-snippet {
  @apply font-mono text-xs text-green-400 whitespace-pre bg-transparent m-0 leading-relaxed;
}

.key-cmd-block {
  @apply bg-gray-900 border border-blue-900 rounded-lg p-3 space-y-2;
}

.key-cmd-label {
  @apply text-xs text-blue-400 block mb-1;
}

.confirm-device {
  @apply font-mono text-sm text-secondary-400 bg-primary-900 rounded px-3 py-2 mb-3;
}

.confirm-message {
  @apply text-sm text-gray-300;
}

.form-select {
  @apply w-full bg-primary-900 border border-primary-600 rounded-lg px-3 py-2;
  @apply text-white text-sm;
  @apply focus:outline-none focus:border-secondary-500;
  @apply disabled:opacity-50 disabled:cursor-not-allowed;
}

.required {
  @apply text-red-400 ml-0.5;
}
</style>
