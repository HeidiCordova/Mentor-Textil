<script setup>
import { ref, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import { useApi } from '@/shared/composables/useApi'

const { loading, error, execute } = useApi()

// TODO: Reemplazar con llamada a API: GET /api/tipos-documentos
const tiposDocumentos = ref([
  {
    id: 1,
    nombreCorto: 'RUC',
    descripcion: 'REG. UNICO DE CONTRIBUYENTES'
  },
  {
    id: 2,
    nombreCorto: 'LE / DNI',
    descripcion: 'LIBRETA ELECTORAL O DNI'
  },
  {
    id: 3,
    nombreCorto: 'CARNET EXT.',
    descripcion: 'CARNET DE EXTRANJERIA'
  },
  {
    id: 4,
    nombreCorto: 'PASAPORTE',
    descripcion: 'PASAPORTE'
  },
  {
    id: 5,
    nombreCorto: 'P. NAC.',
    descripcion: 'PART. DE NACIMIENTO-IDENTIDAD'
  }
])

const mostrarModalAgregar = ref(false)
const mostrarModalEditar = ref(false)
const mostrarModalEliminar = ref(false)

const documentoSeleccionado = ref(null)
const nuevoDocumento = ref({
  nombreCorto: '',
  descripcion: ''
})

const saveSuccess = ref(false)
const deleteSuccess = ref(false)
const saveError = ref('')

const paginaActual = ref(1)
const registrosPorPagina = 10

const abrirModalAgregar = () => {
  nuevoDocumento.value = {
    nombreCorto: '',
    descripcion: ''
  }
  mostrarModalAgregar.value = true
}

const cerrarModalAgregar = () => {
  mostrarModalAgregar.value = false
  nuevoDocumento.value = {
    nombreCorto: '',
    descripcion: ''
  }
}

const abrirModalEditar = (documento) => {
  documentoSeleccionado.value = { ...documento }
  mostrarModalEditar.value = true
}

const cerrarModalEditar = () => {
  mostrarModalEditar.value = false
  documentoSeleccionado.value = null
}

const abrirModalEliminar = (documento) => {
  documentoSeleccionado.value = documento
  mostrarModalEliminar.value = true
}

const cerrarModalEliminar = () => {
  mostrarModalEliminar.value = false
  documentoSeleccionado.value = null
}

const agregarDocumento = async () => {
  try {
    // TODO: Llamada a API
    // await axios.post('/api/tipos-documentos', nuevoDocumento.value)

    // Agregar localmente (mock)
    const nuevoId = Math.max(...tiposDocumentos.value.map(d => d.id), 0) + 1
    tiposDocumentos.value.push({
      id: nuevoId,
      ...nuevoDocumento.value
    })

    saveSuccess.value = true
    setTimeout(() => {
      saveSuccess.value = false
    }, 3000)

    cerrarModalAgregar()
  } catch (err) {
    saveError.value = err.message || 'Error al agregar tipo de documento'
    setTimeout(() => {
      saveError.value = ''
    }, 3000)
  }
}

const editarDocumento = async () => {
  try {
    // TODO: Llamada a API
    // await axios.put(`/api/tipos-documentos/${documentoSeleccionado.value.id}`, documentoSeleccionado.value)

    // Actualizar localmente (mock)
    const index = tiposDocumentos.value.findIndex(d => d.id === documentoSeleccionado.value.id)
    if (index !== -1) {
      tiposDocumentos.value[index] = { ...documentoSeleccionado.value }
    }

    saveSuccess.value = true
    setTimeout(() => {
      saveSuccess.value = false
    }, 3000)

    cerrarModalEditar()
  } catch (err) {
    saveError.value = err.message || 'Error al editar tipo de documento'
    setTimeout(() => {
      saveError.value = ''
    }, 3000)
  }
}

const eliminarDocumento = async () => {
  try {
    // TODO: Llamada a API
    // await axios.delete(`/api/tipos-documentos/${documentoSeleccionado.value.id}`)

    // Eliminar localmente (mock)
    tiposDocumentos.value = tiposDocumentos.value.filter(d => d.id !== documentoSeleccionado.value.id)

    deleteSuccess.value = true
    setTimeout(() => {
      deleteSuccess.value = false
    }, 3000)

    cerrarModalEliminar()
  } catch (err) {
    saveError.value = err.message || 'Error al eliminar tipo de documento'
    setTimeout(() => {
      saveError.value = ''
    }, 3000)
  }
}

onMounted(() => {
  // TODO: Cargar datos desde API
})
</script>

<template>
  <div class="tipo-documento-view">
    <!-- Header -->
    <div class="page-header">
      <h1 class="page-title">TIPOS DE DOCUMENTOS</h1>
    </div>

    <!-- Alertas -->
    <Alert v-if="saveSuccess" type="success" message="✓ Cambios guardados correctamente" class="mb-4" />
    <Alert v-if="deleteSuccess" type="success" message="✓ Tipo de documento eliminado correctamente" class="mb-4" />
    <Alert v-if="saveError" type="error" :message="saveError" class="mb-4" />

    <!-- Contenido Principal -->
    <Card class="content-card">
      <div class="action-bar">
        <Button variant="primary" size="sm" @click="abrirModalAgregar">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          AGREGAR
        </Button>
      </div>

      <Loading v-if="loading" />
      <Alert v-else-if="error" type="error" :message="error" />

      <div v-else class="table-container">
        <table class="documentos-table">
          <thead>
            <tr>
              <th style="width: 60px;">N°</th>
              <th>Nombre corto ↕</th>
              <th>Descripción ↕</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(doc, index) in tiposDocumentos" :key="doc.id">
              <td class="text-center">{{ index + 1 }}</td>
              <td>{{ doc.nombreCorto }}</td>
              <td>{{ doc.descripcion }}</td>
            </tr>
          </tbody>
        </table>

        <!-- Paginación -->
        <div class="pagination">
          <button class="pagination-btn" disabled>
            ANTERIOR
          </button>
          <span class="pagination-info">1</span>
          <button class="pagination-btn" disabled>
            PRÓXIMO
          </button>
        </div>
      </div>
    </Card>

    <!-- Modal AGREGAR -->
    <Transition name="modal">
      <div v-if="mostrarModalAgregar" class="modal-overlay" @click="cerrarModalAgregar">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3 class="modal-title">Agregar Tipo de Documento</h3>
            <button @click="cerrarModalAgregar" class="modal-close">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="form-group">
              <label class="form-label">Nombre Corto *</label>
              <input 
                v-model="nuevoDocumento.nombreCorto" 
                type="text" 
                class="form-input"
                placeholder="Ej: RUC"
                maxlength="50"
              />
            </div>

            <div class="form-group">
              <label class="form-label">Descripción *</label>
              <input 
                v-model="nuevoDocumento.descripcion" 
                type="text" 
                class="form-input"
                placeholder="Ej: REG. UNICO DE CONTRIBUYENTES"
                maxlength="200"
              />
            </div>
          </div>

          <div class="modal-footer">
            <Button variant="secondary" size="sm" @click="cerrarModalAgregar">
              Cancelar
            </Button>
            <Button 
              variant="primary" 
              size="sm" 
              @click="agregarDocumento"
              :disabled="!nuevoDocumento.nombreCorto || !nuevoDocumento.descripcion"
            >
              Guardar
            </Button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Modal EDITAR -->
    <Transition name="modal">
      <div v-if="mostrarModalEditar" class="modal-overlay" @click="cerrarModalEditar">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3 class="modal-title">Editar Tipo de Documento</h3>
            <button @click="cerrarModalEditar" class="modal-close">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="form-group">
              <label class="form-label">Nombre Corto *</label>
              <input 
                v-model="documentoSeleccionado.nombreCorto" 
                type="text" 
                class="form-input"
                maxlength="50"
              />
            </div>

            <div class="form-group">
              <label class="form-label">Descripción *</label>
              <input 
                v-model="documentoSeleccionado.descripcion" 
                type="text" 
                class="form-input"
                maxlength="200"
              />
            </div>
          </div>

          <div class="modal-footer">
            <Button variant="secondary" size="sm" @click="cerrarModalEditar">
              Cancelar
            </Button>
            <Button 
              variant="primary" 
              size="sm" 
              @click="editarDocumento"
              :disabled="!documentoSeleccionado?.nombreCorto || !documentoSeleccionado?.descripcion"
            >
              Guardar Cambios
            </Button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Modal ELIMINAR -->
    <Transition name="modal">
      <div v-if="mostrarModalEliminar" class="modal-overlay" @click="cerrarModalEliminar">
        <div class="modal-content modal-sm" @click.stop>
          <div class="modal-header">
            <h3 class="modal-title">Confirmar Eliminación</h3>
            <button @click="cerrarModalEliminar" class="modal-close">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="confirm-message">
              <svg class="confirm-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
              </svg>
              <p class="confirm-text">
                ¿Está seguro que desea eliminar el tipo de documento 
                <strong>{{ documentoSeleccionado?.nombreCorto }}</strong>?
              </p>
              <p class="confirm-warning">Esta acción no se puede deshacer.</p>
            </div>
          </div>

          <div class="modal-footer">
            <Button variant="secondary" size="sm" @click="cerrarModalEliminar">
              Cancelar
            </Button>
            <Button variant="danger" size="sm" @click="eliminarDocumento">
              Eliminar
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.tipo-documento-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.page-header {
  background-color: #1e3a8a;
  padding: 1.5rem 2rem;
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

.action-bar {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  background-color: #1e3a8a;
}

.table-container {
  overflow-x: auto;
}

.documentos-table {
  width: 100%;
  font-size: 0.875rem;
  border-collapse: collapse;
}

.documentos-table thead {
  background-color: #f3f4f6;
}

.documentos-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
}

.documentos-table td {
  padding: 0.875rem 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
}

.documentos-table tbody tr:hover {
  background-color: #f9fafb;
}

.text-center {
  text-align: center;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid #e5e7eb;
}

.pagination-btn {
  padding: 0.5rem 1rem;
  background-color: #e5e7eb;
  color: #6b7280;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: not-allowed;
  text-transform: uppercase;
}

.pagination-btn:not(:disabled) {
  background-color: #3b82f6;
  color: white;
  cursor: pointer;
}

.pagination-btn:not(:disabled):hover {
  background-color: #2563eb;
}

.pagination-info {
  padding: 0.5rem 1rem;
  background-color: #e5e7eb;
  color: #374151;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 600;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  background-color: white;
  border-radius: 0.75rem;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  max-width: 500px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-content.modal-sm {
  max-width: 400px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e5e7eb;
}

.modal-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: #111827;
  margin: 0;
}

.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background-color: transparent;
  color: #6b7280;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: all 0.2s;
}

.modal-close:hover {
  background-color: #f3f4f6;
  color: #111827;
}

.modal-body {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1.25rem;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.5rem;
}

.form-input {
  width: 100%;
  padding: 0.625rem 0.875rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
}

.form-input:hover {
  border-color: #9ca3af;
}

.form-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1.5rem;
  border-top: 1px solid #e5e7eb;
  background-color: #f9fafb;
}

.confirm-message {
  text-align: center;
}

.confirm-icon {
  width: 3rem;
  height: 3rem;
  color: #f59e0b;
  margin: 0 auto 1rem;
}

.confirm-text {
  font-size: 1rem;
  color: #374151;
  margin: 0 0 0.5rem 0;
}

.confirm-warning {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.3s ease;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95);
}

@media (max-width: 768px) {
  .modal-content {
    max-width: 100%;
    margin: 0;
    border-radius: 0;
    max-height: 100vh;
  }
}
</style>
