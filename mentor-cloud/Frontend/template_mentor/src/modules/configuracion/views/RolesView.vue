<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Modal from '@/shared/components/ui/Modal.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import companyService from '@/api/services/company.service'
import { userService } from '@/api/services/user.service'
import { plantService } from '@/api/services/plant.service'
import { roleService } from '@/api/services/role.service'
import { useAuthStore } from '@/stores/auth'
import { useModal } from '@/shared/composables/useModal'
import { useApi } from '@/shared/composables/useApi'

const authStore = useAuthStore()
const { isOpen: isRolOpen, open: openRolModal, close: closeRolModal } = useModal()
const { isOpen: isAsignOpen, open: openAsignModal, close: closeAsignModal } = useModal()
const { loading, error, execute } = useApi()

const roles = ref([])
const usuarios = ref([])
const empresas = ref([])
const plantas = ref([])

const selectedRolRows = ref([])
const selectedUserRows = ref([])
const isEditingRol = ref(false)
const showDeleteRolConfirm = ref(false)

const canEditRol = computed(() => selectedRolRows.value.length === 1)
const canDeleteRol = computed(() => selectedRolRows.value.length > 0)
const canEditUser = computed(() => selectedUserRows.value.length === 1)

const MENU_OPCIONES = [
  {
    grupo: 'Principal',
    items: [{ key: 'dashboard', label: 'Inicio / Dashboard' }]
  },
  {
    grupo: 'Configuración',
    items: [
      { key: 'configuracion/empresa', label: 'Empresa' },
      { key: 'configuracion/usuario', label: 'Usuario' },
      { key: 'configuracion/roles', label: 'Roles' },
      { key: 'configuracion/archivos', label: 'Archivos' },
      { key: 'configuracion/mantenimiento', label: 'Mantenimiento' },
      { key: 'configuracion/variables', label: 'Variables' },
      { key: 'configuracion/arbol-paradas', label: 'Árbol de Paradas' },

      { key: 'configuracion/dispositivos', label: 'Dispositivos' }
    ]
  },
  {
    grupo: 'Administración',
    items: [
      { key: 'administracion/tipo-documento', label: 'Tipo de Documento' },
      { key: 'administracion/calendarizacion', label: 'Calendarización' },
      { key: 'administracion/turnos', label: 'Turnos' },
      { key: 'administracion/productos', label: 'Productos' }
    ]
  },
  {
    grupo: 'Análisis',
    items: [
      { key: 'analisis/general', label: 'General' },
      { key: 'analisis/energia', label: 'Energía' },
      { key: 'analisis/produccion', label: 'Producción' }
    ]
  },
  {
    grupo: 'Vista Rápida',
    items: [
      { key: 'vista-rapida/factor-calificacion', label: 'Factor de Calificación' },
      { key: 'vista-rapida/oee-turno', label: 'OEE por Turno' }
    ]
  },
  {
    grupo: 'Análisis Avanzado',
    items: [
      { key: 'analisis-avanzado/generador-consultas', label: 'Generador de Consultas' }
    ]
  },
  {
    grupo: 'Análisis de Energía',
    items: [
      { key: 'analisis-energia/consumo-electrico-tarifario', label: 'Consumo Eléctrico Tarifario' },
      { key: 'analisis-energia/factor-calificacion', label: 'Factor de Calificación' }
    ]
  },
  {
    grupo: 'Análisis de Producción',
    items: [
      { key: 'analisis-produccion/linea-tiempo', label: 'Línea de Tiempo' },
      { key: 'analisis-produccion/historia-linea', label: 'Historia de Línea' },
      { key: 'analisis-produccion/grafica-oee', label: 'Gráfica de OEE' },
      { key: 'analisis-produccion/grafica-pareto', label: 'Gráfica de Pareto' },
      { key: 'analisis-produccion/tiempo-real', label: 'Tiempo Real' }
    ]
  },
  {
    grupo: 'Otros',
    items: [
      { key: 'reportes', label: 'Reportes' },
      { key: 'alarmas', label: 'Alarmas' },
      { key: 'datos-recibidos', label: 'Datos Recibidos' },
      { key: 'compromisos', label: 'Compromisos' }
    ]
  }
]

const TODOS_LOS_PERMISOS = MENU_OPCIONES.flatMap(g => g.items.map(i => i.key))

const rolForm = ref({ nombre: '', descripcion: '', permisos: [...TODOS_LOS_PERMISOS] })

const editForm = ref({ empresaId: '', usuarioId: '', rolId: '', plantaId: '' })
const empresaFiltro = ref('')

const plantasFiltradas = computed(() => {
  if (!empresaFiltro.value) return plantas.value
  return plantas.value.filter(p => p.empresa_id === parseInt(empresaFiltro.value))
})

const grupoSeleccionado = (grupo) => grupo.items.every(i => rolForm.value.permisos.includes(i.key))
const grupoIndeterminate = (grupo) => !grupoSeleccionado(grupo) && grupo.items.some(i => rolForm.value.permisos.includes(i.key))
const todosSeleccionados = computed(() => TODOS_LOS_PERMISOS.every(k => rolForm.value.permisos.includes(k)))
const algunoSeleccionado = computed(() => !todosSeleccionados.value && TODOS_LOS_PERMISOS.some(k => rolForm.value.permisos.includes(k)))

function toggleTodos() {
  if (todosSeleccionados.value) {
    rolForm.value.permisos = []
  } else {
    rolForm.value.permisos = [...TODOS_LOS_PERMISOS]
  }
}

function toggleGrupo(grupo) {
  const keys = grupo.items.map(i => i.key)
  if (grupoSeleccionado(grupo)) {
    rolForm.value.permisos = rolForm.value.permisos.filter(k => !keys.includes(k))
  } else {
    const nuevos = keys.filter(k => !rolForm.value.permisos.includes(k))
    rolForm.value.permisos = [...rolForm.value.permisos, ...nuevos]
  }
}

function togglePermiso(key) {
  const idx = rolForm.value.permisos.indexOf(key)
  if (idx >= 0) {
    rolForm.value.permisos.splice(idx, 1)
  } else {
    rolForm.value.permisos.push(key)
  }
}

const toggleRolRow = (item) => { selectedRolRows.value = [item] }
const isRolSelected = (item) => selectedRolRows.value.some(r => r.id === item.id)
const toggleUserRow = (item) => { selectedUserRows.value = [item] }
const isUserSelected = (item) => selectedUserRows.value.some(r => r.id === item.id)

const loadData = async () => {
  await execute(async () => {
    const [companiesRes, rolesRes] = await Promise.all([
      companyService.getAll(),
      roleService.getAll()
    ])
    empresas.value = companiesRes.data || []
    const rawRoles = Array.isArray(rolesRes) ? rolesRes : (rolesRes.data || [])
    roles.value = rawRoles.map(r => ({ ...r, permisos: r.permisos || [] }))

    const empresaId = empresaFiltro.value || authStore.user?.empresa_id || null
    const plantaParams = empresaId ? { empresa_id: empresaId } : {}
    const [plantasRes, usersRes] = await Promise.all([
      plantService.getAll(plantaParams),
      userService.getAll(empresaId ? { empresa_id: empresaId } : {})
    ])
    plantas.value = plantasRes.data || []
    usuarios.value = (usersRes.data || []).map(u => ({
      ...u,
      empresaId: u.empresa_id,
      rolId: u.rol_id
    }))
  })
}

const openCreateRolModal = () => {
  isEditingRol.value = false
  rolForm.value = { nombre: '', descripcion: '', permisos: [...TODOS_LOS_PERMISOS] }
  openRolModal()
}

const openEditRolModal = () => {
  if (selectedRolRows.value.length !== 1) return
  const r = selectedRolRows.value[0]
  isEditingRol.value = true
  rolForm.value = {
    id: r.id,
    nombre: r.nombre,
    descripcion: r.descripcion || '',
    permisos: [...(r.permisos || TODOS_LOS_PERMISOS)]
  }
  openRolModal()
}

const handleSubmitRol = async () => {
  await execute(async () => {
    const payload = {
      nombre: rolForm.value.nombre,
      descripcion: rolForm.value.descripcion,
      permisos: rolForm.value.permisos
    }
    if (isEditingRol.value) {
      await roleService.update(rolForm.value.id, payload)
    } else {
      await roleService.create(payload)
    }
    await loadData()
    closeRolModal()
    selectedRolRows.value = []
  })
}

const handleDeleteRol = async () => {
  await execute(async () => {
    for (const r of selectedRolRows.value) {
      await roleService.delete(r.id)
    }
    await loadData()
    selectedRolRows.value = []
    showDeleteRolConfirm.value = false
  })
}

const openAsignEditModal = () => {
  if (selectedUserRows.value.length !== 1) return
  const item = selectedUserRows.value[0]
  editForm.value = {
    empresaId: item.empresaId || item.empresa_id || '',
    usuarioId: item.id,
    rolId: item.rolId || item.rol_id || '',
    plantaId: ''
  }
  openAsignModal()
}

const handleSubmitAsign = async () => {
  await execute(async () => {
    const userId = editForm.value.usuarioId
    const rolId = editForm.value.rolId ? parseInt(editForm.value.rolId) : null
    const empresaId = editForm.value.empresaId ? parseInt(editForm.value.empresaId) : null
    const user = usuarios.value.find(u => u.id === userId)
    if (user) {
      await userService.update(userId, {
        username: user.username,
        nombre: user.nombre,
        email: user.email,
        rol_id: rolId,
        empresa_id: empresaId,
        activo: user.activo
      })
    }
    await loadData()
    closeAsignModal()
    selectedUserRows.value = []
  })
}

watch(empresaFiltro, () => loadData())

onMounted(() => loadData())
</script>

<template>
  <div class="roles-view">
    <Loading v-if="loading" />
    <Alert v-if="error" type="error" :message="error" />

    <!-- Sección Gestión de Roles -->
    <Card>
      <div class="card-header-section">
        <h2 class="section-title">ROLES</h2>
        <div class="header-actions">
          <Button @click="openCreateRolModal" variant="primary" size="sm">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            AGREGAR
          </Button>
          <Button @click="openEditRolModal" :disabled="!canEditRol" variant="secondary" size="sm">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
            </svg>
            EDITAR
          </Button>
          <Button @click="showDeleteRolConfirm = true" :disabled="!canDeleteRol" variant="danger" size="sm">
            <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
            ELIMINAR
          </Button>
        </div>
      </div>

      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N</th>
              <th>ROL</th>
              <th>DESCRIPCIÓN</th>
              <th>PERMISOS</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(rol, index) in roles"
              :key="rol.id"
              @click="toggleRolRow(rol)"
              :class="{ 'row-selected': isRolSelected(rol) }"
            >
              <td class="checkbox-cell">
                <input type="checkbox" class="table-checkbox" :checked="isRolSelected(rol)" @click.stop="toggleRolRow(rol)" />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ rol.nombre }}</td>
              <td>{{ rol.descripcion || '-' }}</td>
              <td>
                <span class="badge-permisos">{{ (rol.permisos || []).length }} / {{ TODOS_LOS_PERMISOS.length }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Card>

    <!-- Sección Asignación de Roles a Usuarios -->
    <Card>
      <div class="card-header-section">
        <h2 class="section-title">ASIGNACIÓN DE ROLES</h2>
        <div class="header-actions">
          <div class="filter-group">
            <label class="filter-label">Empresa</label>
            <select v-model="empresaFiltro" class="filter-select">
              <option value="">Todas</option>
              <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">
                {{ empresa.nombre }}
              </option>
            </select>
          </div>
          <Button @click="openAsignEditModal" :disabled="!canEditUser" variant="primary" size="sm">
            EDITAR
          </Button>
        </div>
      </div>

      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N</th>
              <th>NOMBRE</th>
              <th>USERNAME</th>
              <th>CORREO</th>
              <th>ROL</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(usuario, index) in usuarios"
              :key="usuario.id"
              @click="toggleUserRow(usuario)"
              :class="{ 'row-selected': isUserSelected(usuario) }"
            >
              <td class="checkbox-cell">
                <input type="checkbox" class="table-checkbox" :checked="isUserSelected(usuario)" @click.stop="toggleUserRow(usuario)" />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ usuario.nombre }}</td>
              <td>{{ usuario.username }}</td>
              <td>{{ usuario.email }}</td>
              <td>{{ usuario.rol || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Card>

    <!-- Modal Crear/Editar Rol con permisos -->
    <Modal v-model="isRolOpen" @close="closeRolModal" :title="isEditingRol ? 'Editar Rol' : 'Crear Rol'" size="lg">
      <form @submit.prevent="handleSubmitRol" class="form-container">
        <div class="form-content">
          <div class="form-row">
            <label class="form-label">Nombre <span class="text-red-500">*</span></label>
            <input v-model="rolForm.nombre" class="field-input" required placeholder="Ej: ADMIN_PLANTA" />
          </div>
          <div class="form-row">
            <label class="form-label">Descripción</label>
            <input v-model="rolForm.descripcion" class="field-input" placeholder="Descripción del rol" />
          </div>

          <div class="permisos-section">
            <div class="permisos-header">
              <span class="form-label">Opciones del menú que puede ver</span>
              <label class="check-row check-all">
                <input
                  type="checkbox"
                  :checked="todosSeleccionados"
                  :indeterminate="algunoSeleccionado"
                  @change="toggleTodos"
                />
                <span>Seleccionar todo</span>
              </label>
            </div>

            <div class="permisos-grid">
              <div v-for="grupo in MENU_OPCIONES" :key="grupo.grupo" class="permiso-grupo">
                <label class="check-row grupo-label">
                  <input
                    type="checkbox"
                    :checked="grupoSeleccionado(grupo)"
                    :indeterminate="grupoIndeterminate(grupo)"
                    @change="toggleGrupo(grupo)"
                  />
                  <span class="grupo-nombre">{{ grupo.grupo }}</span>
                </label>
                <div class="grupo-items">
                  <label v-for="item in grupo.items" :key="item.key" class="check-row item-label">
                    <input
                      type="checkbox"
                      :checked="rolForm.permisos.includes(item.key)"
                      @change="togglePermiso(item.key)"
                    />
                    <span>{{ item.label }}</span>
                  </label>
                </div>
              </div>
            </div>

            <div class="permisos-count">
              <span>{{ rolForm.permisos.length }} de {{ TODOS_LOS_PERMISOS.length }} opciones seleccionadas</span>
            </div>
          </div>
        </div>

        <div class="form-footer">
          <Button type="button" @click="closeRolModal" variant="secondary" size="md">Cancelar</Button>
          <Button type="submit" variant="primary" size="md" :loading="loading">
            {{ isEditingRol ? 'Actualizar' : 'Guardar' }}
          </Button>
        </div>
      </form>
    </Modal>

    <!-- Modal Asignación -->
    <Modal v-model="isAsignOpen" @close="closeAsignModal" title="Asignar Rol" size="md">
      <form @submit.prevent="handleSubmitAsign" class="form-container">
        <div class="form-content">
          <div class="form-row">
            <label class="form-label">Empresa</label>
            <select v-model="editForm.empresaId" class="field-select">
              <option value="">Seleccione empresa</option>
              <option v-for="empresa in empresas" :key="empresa.id" :value="empresa.id">{{ empresa.nombre }}</option>
            </select>
          </div>
          <div class="form-row">
            <label class="form-label">Usuario</label>
            <select v-model="editForm.usuarioId" class="field-select" disabled>
              <option v-for="usuario in usuarios" :key="usuario.id" :value="usuario.id">{{ usuario.nombre }}</option>
            </select>
          </div>
          <div class="form-row">
            <label class="form-label">Rol</label>
            <select v-model="editForm.rolId" class="field-select" required>
              <option value="">Seleccione rol</option>
              <option v-for="rol in roles" :key="rol.id" :value="rol.id">{{ rol.nombre }}</option>
            </select>
          </div>
          <div class="info-message">
            <svg class="info-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <span>El rol define qué opciones del menú puede ver el usuario</span>
          </div>
          <div class="form-row">
            <label class="form-label">Planta</label>
            <select v-model="editForm.plantaId" class="field-select">
              <option value="">Seleccione planta</option>
              <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">{{ planta.nombre }}</option>
            </select>
          </div>
        </div>
        <div class="form-footer">
          <Button type="button" @click="closeAsignModal" variant="secondary" size="md">Cancelar</Button>
          <Button type="submit" variant="primary" size="md" :loading="loading">Guardar</Button>
        </div>
      </form>
    </Modal>

    <!-- Confirmar eliminación de rol -->
    <Modal v-model="showDeleteRolConfirm" @close="showDeleteRolConfirm = false" title="Confirmar Eliminación" size="sm">
      <div class="delete-confirm">
        <svg class="delete-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
        </svg>
        <p class="delete-text">¿Eliminar <strong>{{ selectedRolRows.length }}</strong> {{ selectedRolRows.length === 1 ? 'rol' : 'roles' }}?</p>
        <p class="delete-subtext">Esta acción no se puede deshacer.</p>
      </div>
      <div class="delete-actions">
        <Button @click="showDeleteRolConfirm = false" variant="secondary">Cancelar</Button>
        <Button @click="handleDeleteRol" variant="danger" :loading="loading">Eliminar</Button>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.roles-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.card-header-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1rem 0.5rem;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.section-title {
  font-size: 1rem;
  font-weight: 700;
  color: #1e40af;
  letter-spacing: 0.05em;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  white-space: nowrap;
}

.filter-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  cursor: pointer;
  min-width: 180px;
}

.filter-select:focus {
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

.data-table thead { background-color: #f3f4f6; }

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

.data-table tbody tr:hover { background-color: #eff6ff; }
.data-table tbody tr:nth-child(even) { background-color: #f9fafb; }
.data-table tbody tr:nth-child(even):hover { background-color: #eff6ff; }
.data-table tbody tr.row-selected {
  background-color: #dbeafe;
  border-left: 4px solid #2563eb;
}

.data-table td {
  padding: 0.75rem 1rem;
  color: #111827;
  border-bottom: 1px solid #e5e7eb;
}

.data-table td.table-name { font-weight: 600; }

.checkbox-cell { width: 3rem; text-align: center; }

.table-checkbox {
  width: 1.25rem;
  height: 1.25rem;
  cursor: pointer;
}

.badge-permisos {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  background-color: #dbeafe;
  color: #1e40af;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.form-container { padding: 0; }

.form-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.5rem;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.field-input, .field-select {
  width: 100%;
  padding: 0.625rem 0.75rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  transition: all 0.2s;
}

.field-input:focus, .field-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.field-select:disabled {
  background-color: #f3f4f6;
  cursor: not-allowed;
  opacity: 0.6;
}

/* Permisos */
.permisos-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1rem;
  background: #f8fafc;
}

.permisos-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid #e5e7eb;
}

.permisos-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 1rem;
  max-height: 340px;
  overflow-y: auto;
  padding-right: 0.25rem;
}

.permiso-grupo {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.check-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.8125rem;
  color: #374151;
  user-select: none;
}

.check-row input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
  accent-color: #2563eb;
  flex-shrink: 0;
}

.check-all { font-size: 0.875rem; font-weight: 600; color: #1e40af; }
.grupo-label { font-weight: 600; color: #1e3a8a; }
.grupo-nombre { font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.04em; }

.grupo-items {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  padding-left: 1.25rem;
}

.item-label { color: #4b5563; }

.permisos-count {
  font-size: 0.75rem;
  color: #6b7280;
  text-align: right;
  padding-top: 0.25rem;
  border-top: 1px solid #e5e7eb;
}

.info-message {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background-color: #fef3c7;
  border: 1px solid #fbbf24;
  border-radius: 0.5rem;
  font-size: 0.8125rem;
  color: #92400e;
  font-weight: 500;
}

.info-icon {
  width: 1.25rem;
  height: 1.25rem;
  color: #f59e0b;
  flex-shrink: 0;
}

.form-footer {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  padding: 1rem 1.5rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.delete-confirm {
  text-align: center;
  padding: 1rem 1.5rem;
}

.delete-icon {
  width: 3.5rem;
  height: 3.5rem;
  margin: 0 auto 0.75rem;
  color: #dc2626;
}

.delete-text { color: #111827; margin-bottom: 0.25rem; }
.delete-subtext { font-size: 0.875rem; color: #6b7280; }

.delete-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  padding: 1rem 1.5rem;
  border-top: 1px solid #e5e7eb;
}
</style>
