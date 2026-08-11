<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import { useApi } from '@/shared/composables/useApi'

const { loading, error, execute } = useApi()

const empresas = ref([
  { id: 1, nombre: 'Art Atlas' }
])

const plantas = ref([
  { id: 1, nombre: 'Histórico Planta Textil', empresaId: 1 }
])

// TODO: Reemplazar con llamada a API: GET /api/estructuras-tarifarias
const estructurasTarifarias = ref([
  {
    id: 1,
    nombre: 'MT3',
    periodo: '2',
    pais: 'PERÚ',
    zonaHora: 'América/Lima',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 2,
    nombre: 'MT4',
    periodo: '2',
    pais: 'MAJES',
    zonaHora: 'América/Lima',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 3,
    nombre: 'Vista estacional A4',
    periodo: '2',
    pais: 'Brasil',
    zonaHora: 'América/Lima',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 4,
    nombre: 'Prueba',
    periodo: '4',
    pais: 'Costa Rica',
    zonaHora: '',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 5,
    nombre: 'Cliente Libre',
    periodo: '3',
    pais: 'Perú',
    zonaHora: 'América/Lima',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 6,
    nombre: 'ENEL',
    periodo: '3',
    pais: 'PERÚ',
    zonaHora: 'América/Lima',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 7,
    nombre: 'COELBA',
    periodo: '2',
    pais: 'BRASIL',
    zonaHora: 'América/Bahia',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  },
  {
    id: 8,
    nombre: 'ENEL',
    periodo: '2',
    pais: 'REPÚBLICA DOMINICANA',
    zonaHora: 'América/Santo_Domingo',
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: ''
  }
])

// TODO: Reemplazar con llamada a API: GET /api/periodos?estructuraId={id}
const periodos = ref([
  {
    id: 1,
    descripcion: 'Periodo fuera de Punta',
    costeKwh: '1',
    moneda: 'Suelas',
    creado: '18 de octubre de 2016, 14:38:15'
  },
  {
    id: 2,
    descripcion: 'Hora Punta',
    costeKwh: '1',
    moneda: 'Suelas',
    creado: '18 de octubre de 2016, 14:38:15'
  }
])

const tabActiva = ref('estructura') // 'estructura' o 'calendarizacion'

const empresaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)

const estructuraSeleccionada = ref(null)
const nombreEstructura = ref('')
const paisEstructura = ref('')
const zonaHorariaEstructura = ref('')

const mostrarModalAgregar = ref(false)
const mostrarModalEditar = ref(false)
const mostrarModalEliminar = ref(false)
const itemSeleccionado = ref(null)

const saveSuccess = ref(false)
const deleteSuccess = ref(false)
const saveError = ref('')

// ============================================
// 📊 COMPUTED
// ============================================
const plantasFiltradas = computed(() => {
  if (!empresaSeleccionada.value) return plantas.value
  return plantas.value.filter(p => p.empresaId === parseInt(empresaSeleccionada.value))
})

const cambiarTab = (tab) => {
  tabActiva.value = tab
}

const seleccionarEstructura = (estructura) => {
  estructuraSeleccionada.value = estructura
  nombreEstructura.value = estructura.nombre
  paisEstructura.value = estructura.pais
  zonaHorariaEstructura.value = estructura.zonaHora
}

const asignarPeriodos = () => {
  if (!estructuraSeleccionada.value) {
    alert('Por favor seleccione una estructura tarifaria primero')
    return
  }
  // TODO: Implementar lógica de asignación de periodos
  console.log('Asignar periodos para:', estructuraSeleccionada.value)
}

const abrirModalAgregarEstructura = () => {
  itemSeleccionado.value = {
    nombre: '',
    periodo: '',
    pais: '',
    zonaHora: ''
  }
  mostrarModalAgregar.value = true
}

const abrirModalEditarEstructura = (estructura) => {
  itemSeleccionado.value = { ...estructura }
  mostrarModalEditar.value = true
}

const abrirModalEliminarEstructura = (estructura) => {
  itemSeleccionado.value = estructura
  mostrarModalEliminar.value = true
}

const agregarEstructura = () => {
  const nuevoId = Math.max(...estructurasTarifarias.value.map(e => e.id), 0) + 1
  estructurasTarifarias.value.push({
    id: nuevoId,
    ...itemSeleccionado.value,
    descripcion: '',
    costeKwh: '',
    moneda: '',
    creado: new Date().toLocaleString()
  })
  
  saveSuccess.value = true
  setTimeout(() => saveSuccess.value = false, 3000)
  cerrarModales()
}

const editarEstructura = () => {
  const index = estructurasTarifarias.value.findIndex(e => e.id === itemSeleccionado.value.id)
  if (index !== -1) {
    estructurasTarifarias.value[index] = { ...itemSeleccionado.value }
  }
  
  saveSuccess.value = true
  setTimeout(() => saveSuccess.value = false, 3000)
  cerrarModales()
}

const eliminarEstructura = () => {
  estructurasTarifarias.value = estructurasTarifarias.value.filter(e => e.id !== itemSeleccionado.value.id)
  
  deleteSuccess.value = true
  setTimeout(() => deleteSuccess.value = false, 3000)
  cerrarModales()
}

const cerrarModales = () => {
  mostrarModalAgregar.value = false
  mostrarModalEditar.value = false
  mostrarModalEliminar.value = false
  itemSeleccionado.value = null
}

onMounted(() => {
  // TODO: Cargar datos desde API
})
</script>

<template>
  <div class="calendarizacion-view">
    <!-- Filtros superiores (solo en tab Calendarización) -->
    <div v-if="tabActiva === 'calendarizacion'" class="filters-section">
      <div class="filter-row">
        <div class="filter-item">
          <label class="filter-label">Empresas</label>
          <select v-model="empresaSeleccionada" class="filter-select">
            <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
              {{ empresa.nombre }}
            </option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Plantas</label>
          <select v-model="plantaSeleccionada" class="filter-select">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
      </div>
    </div>

    <!-- Alertas -->
    <Alert v-if="saveSuccess" type="success" message="✓ Cambios guardados correctamente" class="mb-4" />
    <Alert v-if="deleteSuccess" type="success" message="✓ Registro eliminado correctamente" class="mb-4" />
    <Alert v-if="saveError" type="error" :message="saveError" class="mb-4" />

    <!-- Tabs -->
    <div class="tabs-container">
      <button 
        :class="['tab-button', { active: tabActiva === 'estructura' }]"
        @click="cambiarTab('estructura')"
      >
        ESTRUCTURA TARIFARIA
      </button>
      <button 
        :class="['tab-button', { active: tabActiva === 'calendarizacion' }]"
        @click="cambiarTab('calendarizacion')"
      >
        CALENDARIZACIÓN
      </button>
    </div>

    <!-- Contenido por Tab -->
    <div class="content-layout">
      <!-- TAB: ESTRUCTURA TARIFARIA -->
      <div v-if="tabActiva === 'estructura'" class="tab-content">
        <div class="two-columns">
          <!-- Columna Izquierda: Tabla -->
          <Card class="content-card left-column">
            <div class="card-header">
              <h3 class="card-title">ESTRUCTURA TARIFARIA</h3>
            </div>

            <div class="action-bar">
              <Button variant="primary" size="sm" @click="abrirModalAgregarEstructura">
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                </svg>
                AGREGAR
              </Button>
              <Button variant="secondary" size="sm" @click="abrirModalEditarEstructura(estructuraSeleccionada)" :disabled="!estructuraSeleccionada">
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                </svg>
                EDITAR
              </Button>
              <Button variant="danger" size="sm" @click="abrirModalEliminarEstructura(estructuraSeleccionada)" :disabled="!estructuraSeleccionada">
                <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                ELIMINAR
              </Button>
            </div>

            <Loading v-if="loading" />

            <div v-else class="table-wrapper">
              <table class="data-table">
                <thead>
                  <tr>
                    <th style="width: 50px;">N°</th>
                    <th>Nombre</th>
                    <th>Período(s)</th>
                    <th>País</th>
                    <th>Zona hora</th>
                    <th>Descripción</th>
                    <th>Coste del kWh</th>
                    <th>Moneda</th>
                    <th>Creado</th>
                  </tr>
                </thead>
                <tbody>
                  <tr 
                    v-for="(estructura, index) in estructurasTarifarias" 
                    :key="estructura.id"
                    :class="{ 'selected': estructuraSeleccionada?.id === estructura.id }"
                    @click="seleccionarEstructura(estructura)"
                  >
                    <td class="text-center">{{ index + 1 }}</td>
                    <td>{{ estructura.nombre }}</td>
                    <td>{{ estructura.periodo }}</td>
                    <td>{{ estructura.pais }}</td>
                    <td>{{ estructura.zonaHora }}</td>
                    <td>{{ estructura.descripcion }}</td>
                    <td>{{ estructura.costeKwh }}</td>
                    <td>{{ estructura.moneda }}</td>
                    <td>{{ estructura.creado }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </Card>

          <!-- Columna Derecha: Períodos -->
          <Card class="content-card right-column">
            <div class="card-header">
              <h3 class="card-title">PERIODOS</h3>
            </div>

            <div class="periodos-empty" v-if="!estructuraSeleccionada">
              <p>Seleccione una estructura tarifaria para ver sus períodos</p>
            </div>
            <div v-else class="periodos-content">
              <p class="info-text">No hay períodos configurados para esta estructura</p>
            </div>
          </Card>
        </div>
      </div>

      <!-- TAB: CALENDARIZACIÓN -->
      <div v-else class="tab-content">
        <div class="two-columns">
          <!-- Columna Izquierda: Formulario -->
          <Card class="content-card left-column">
            <div class="card-header">
              <h3 class="card-title">ESTRUCTURA TARIFARIA</h3>
            </div>

            <div class="form-container">
              <div class="form-group">
                <label class="form-label">Estructura Tarifaria</label>
                <select v-model="estructuraSeleccionada" class="form-input">
                  <option :value="null" disabled>Seleccione...</option>
                  <option v-for="estructura in estructurasTarifarias" :key="estructura.id" :value="estructura">
                    {{ estructura.nombre }}
                  </option>
                </select>
              </div>

              <Button variant="secondary" size="sm" @click="asignarPeriodos" class="mt-3">
                ASIGNAR PERIODOS
              </Button>

              <div v-if="estructuraSeleccionada" class="mt-4">
                <div class="form-group">
                  <label class="form-label">Nombre</label>
                  <input v-model="nombreEstructura" type="text" class="form-input" readonly />
                </div>

                <div class="form-group">
                  <label class="form-label">País</label>
                  <input v-model="paisEstructura" type="text" class="form-input" readonly />
                </div>

                <div class="form-group">
                  <label class="form-label">Zona Horaria</label>
                  <input v-model="zonaHorariaEstructura" type="text" class="form-input" readonly />
                </div>
              </div>
            </div>
          </Card>

          <!-- Columna Derecha: Periodos -->
          <Card class="content-card right-column">
            <div class="card-header">
              <h3 class="card-title">PERIODOS</h3>
            </div>

            <div v-if="estructuraSeleccionada" class="table-wrapper">
              <table class="data-table periodos-table">
                <thead>
                  <tr>
                    <th>Descripción</th>
                    <th>Coste del kWh</th>
                    <th>Moneda</th>
                    <th>Creado</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="periodo in periodos" :key="periodo.id">
                    <td>{{ periodo.descripcion }}</td>
                    <td>{{ periodo.costeKwh }}</td>
                    <td>{{ periodo.moneda }}</td>
                    <td>{{ periodo.creado }}</td>
                  </tr>
                </tbody>
              </table>

              <div class="periodos-footer">
                <Button variant="info" size="sm">
                  VOLVER
                </Button>
              </div>
            </div>
            <div v-else class="periodos-empty">
              <p>Seleccione una estructura tarifaria para ver sus períodos</p>
            </div>
          </Card>
        </div>
      </div>
    </div>

    <!-- Modal AGREGAR -->
    <Transition name="modal">
      <div v-if="mostrarModalAgregar" class="modal-overlay" @click="cerrarModales">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3 class="modal-title">Agregar Estructura Tarifaria</h3>
            <button @click="cerrarModales" class="modal-close">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="form-group">
              <label class="form-label">Nombre *</label>
              <input v-model="itemSeleccionado.nombre" type="text" class="form-input" />
            </div>

            <div class="form-group">
              <label class="form-label">Período(s) *</label>
              <input v-model="itemSeleccionado.periodo" type="text" class="form-input" />
            </div>

            <div class="form-group">
              <label class="form-label">País *</label>
              <input v-model="itemSeleccionado.pais" type="text" class="form-input" />
            </div>

            <div class="form-group">
              <label class="form-label">Zona Horaria</label>
              <input v-model="itemSeleccionado.zonaHora" type="text" class="form-input" placeholder="Ej: América/Lima" />
            </div>
          </div>

          <div class="modal-footer">
            <Button variant="secondary" size="sm" @click="cerrarModales">Cancelar</Button>
            <Button variant="primary" size="sm" @click="agregarEstructura" :disabled="!itemSeleccionado?.nombre || !itemSeleccionado?.periodo || !itemSeleccionado?.pais">
              Guardar
            </Button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Modal EDITAR -->
    <Transition name="modal">
      <div v-if="mostrarModalEditar" class="modal-overlay" @click="cerrarModales">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3 class="modal-title">Editar Estructura Tarifaria</h3>
            <button @click="cerrarModales" class="modal-close">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="form-group">
              <label class="form-label">Nombre *</label>
              <input v-model="itemSeleccionado.nombre" type="text" class="form-input" />
            </div>

            <div class="form-group">
              <label class="form-label">Período(s) *</label>
              <input v-model="itemSeleccionado.periodo" type="text" class="form-input" />
            </div>

            <div class="form-group">
              <label class="form-label">País *</label>
              <input v-model="itemSeleccionado.pais" type="text" class="form-input" />
            </div>

            <div class="form-group">
              <label class="form-label">Zona Horaria</label>
              <input v-model="itemSeleccionado.zonaHora" type="text" class="form-input" />
            </div>
          </div>

          <div class="modal-footer">
            <Button variant="secondary" size="sm" @click="cerrarModales">Cancelar</Button>
            <Button variant="primary" size="sm" @click="editarEstructura">Guardar Cambios</Button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Modal ELIMINAR -->
    <Transition name="modal">
      <div v-if="mostrarModalEliminar" class="modal-overlay" @click="cerrarModales">
        <div class="modal-content modal-sm" @click.stop>
          <div class="modal-header">
            <h3 class="modal-title">Confirmar Eliminación</h3>
            <button @click="cerrarModales" class="modal-close">
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
                ¿Está seguro que desea eliminar 
                <strong>{{ itemSeleccionado?.nombre }}</strong>?
              </p>
              <p class="confirm-warning">Esta acción no se puede deshacer.</p>
            </div>
          </div>

          <div class="modal-footer">
            <Button variant="secondary" size="sm" @click="cerrarModales">Cancelar</Button>
            <Button variant="danger" size="sm" @click="eliminarEstructura">Eliminar</Button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.calendarizacion-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.filters-section {
  background-color: white;
  border: 1px solid #e5e7eb;
  padding: 1.5rem;
}

.filter-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1rem;
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
  padding: 0.625rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  transition: all 0.2s;
}

.filter-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.tabs-container {
  display: flex;
  border-bottom: 2px solid #e5e7eb;
  background-color: white;
}

.tab-button {
  padding: 1rem 2rem;
  border: none;
  background-color: transparent;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  border-bottom: 3px solid transparent;
  transition: all 0.2s;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.tab-button:hover {
  color: #111827;
  background-color: #f9fafb;
}

.tab-button.active {
  color: #1e3a8a;
  border-bottom-color: #1e3a8a;
  background-color: white;
}

.content-layout {
  background-color: white;
}

.tab-content {
  padding: 0;
}

.two-columns {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 0;
  min-height: 600px;
}

.content-card {
  border-radius: 0;
  border: none;
  border-right: 1px solid #e5e7eb;
}

.content-card.right-column {
  border-right: none;
}

.card-header {
  padding: 1.25rem 1.5rem;
  background-color: #1e3a8a;
}

.card-title {
  font-size: 1rem;
  font-weight: 700;
  color: white;
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.action-bar {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  background-color: #1e3a8a;
  border-bottom: 1px solid #2563eb;
}

.table-wrapper {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  font-size: 0.875rem;
  border-collapse: collapse;
}

.data-table thead {
  background-color: #f3f4f6;
  position: sticky;
  top: 0;
  z-index: 10;
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
  white-space: nowrap;
}

.data-table td {
  padding: 0.875rem 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
}

.data-table tbody tr {
  cursor: pointer;
  transition: background-color 0.15s;
}

.data-table tbody tr:hover {
  background-color: #f9fafb;
}

.data-table tbody tr.selected {
  background-color: #eff6ff;
  border-left: 3px solid #3b82f6;
}

.text-center {
  text-align: center;
}

.periodos-table {
  font-size: 0.8125rem;
}

.periodos-table th,
.periodos-table td {
  padding: 0.75rem 0.875rem;
}

.form-container {
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

.form-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:read-only {
  background-color: #f9fafb;
  color: #6b7280;
}

.periodos-empty {
  padding: 3rem 1.5rem;
  text-align: center;
  color: #6b7280;
  font-size: 0.9375rem;
}

.periodos-content {
  padding: 1.5rem;
}

.info-text {
  color: #6b7280;
  font-size: 0.875rem;
  text-align: center;
  margin: 2rem 0;
}

.periodos-footer {
  padding: 1.25rem 1.5rem;
  border-top: 1px solid #e5e7eb;
  display: flex;
  justify-content: flex-end;
}

.mt-3 {
  margin-top: 0.75rem;
}

.mt-4 {
  margin-top: 1rem;
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

@media (max-width: 1200px) {
  .two-columns {
    grid-template-columns: 1fr;
  }

  .content-card {
    border-right: none;
    border-bottom: 1px solid #e5e7eb;
  }
}

@media (max-width: 768px) {
  .filter-row {
    grid-template-columns: 1fr;
  }

  .tab-button {
    padding: 0.875rem 1rem;
    font-size: 0.8125rem;
  }
}
</style>
