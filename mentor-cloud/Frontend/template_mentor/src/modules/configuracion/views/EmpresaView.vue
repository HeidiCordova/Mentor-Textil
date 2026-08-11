<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Modal from '@/shared/components/ui/Modal.vue'
import FormField from '@/shared/components/forms/FormField.vue'
import SelectFilter from '@/shared/components/forms/SelectFilter.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { deviceService } from '@/api/services/device.service'
import { locacionService } from '@/api/services/locacion.service'
import { useAuthStore } from '@/stores/auth'
import { useModal } from '@/shared/composables/useModal'
import { useApi } from '@/shared/composables/useApi'

const authStore = useAuthStore()
const activeTab = ref('empresa')
const { isOpen, open: openModal, close: closeModal } = useModal()
const { loading, error, execute } = useApi()

const companies = ref([])
const plants = ref([])
const lines = ref([])
const locations = ref([])
const devices = ref([])

const selectedCompany = ref(null)
const selectedPlant = ref(null)
const selectedLine = ref(null)
const selectedLocation = ref(null)

const selectedRows = ref([])
const isEditing = ref(false)
const showDeleteConfirm = ref(false)
const itemToDelete = ref(null)

const canEdit = computed(() => selectedRows.value.length === 1)
const canDelete = computed(() => selectedRows.value.length > 0)

const sameID = (a, b) => String(a) === String(b)

const selectedPlantData = computed(() => {
  const selected = selectedPlant.value
  if (selected === null || selected === undefined || selected === '') return null

  return plants.value.find(p => String(p.id) === String(selected)) || null
})
const isEnergyPlanta = computed(() => (selectedPlantData.value?.tipo || '').toLowerCase() === 'energia')

const isSuperadmin = computed(() => authStore.hasRole('superadmin'))
const provisioning = ref(null)
const showProvisionConfirm = ref(false)
const plantToProvision = ref(null)
const provisionError = ref('')
const canProvision = computed(() => {
  if (!isSuperadmin.value) return false
  if (activeTab.value !== 'planta') return false
  if (selectedRows.value.length !== 1) return false
  return !selectedRows.value[0].db_name
})

const handleProvision = (plant) => {
  if (!plant) plant = selectedRows.value[0]
  if (!plant) return
  plantToProvision.value = plant
  provisionError.value = ''
  showProvisionConfirm.value = true
}

const confirmProvision = async () => {
  const plant = plantToProvision.value
  if (!plant) return
  provisioning.value = plant.id
  provisionError.value = ''
  try {
    await plantService.provision(plant.id)
    showProvisionConfirm.value = false
    plantToProvision.value = null
    await loadData()
  } catch (err) {
    provisionError.value = err.response?.data?.error || 'Error al provisionar la base de datos'
  } finally {
    provisioning.value = null
  }
}

const toggleRowSelection = (item) => {
  selectedRows.value = [item]
}

const isRowSelected = (item) => {
  return selectedRows.value.some(row => row.id === item.id)
}

watch(activeTab, () => {
  selectedRows.value = []
})

const empresaForm = ref({
  nombreComercial: '',
  descripcion: '',
  denominacion: '',
  documento: '',
  correoElectronico: '',
  tipoDocumento: 'RUC',
  telefono: '',
  telefono1: '',
  sitioWeb: '',
  representante: '',
  telefonoRepresentante: '',
  logo: '',
  direccion: '',
  estado: true
})

const plantaForm = ref({
  empresaId: null,
  nombre: '',
  descripcion: '',
  tipo: 'produccion',
  zonaHoraria: '',
  creado: '',
  modificado: ''
})

const lineaForm = ref({
  empresaId: null,
  plantaId: null,
  nombre: '',
  descripcion: '',
  tipo: 'Producción',
  codigoAlertas: '',
  subtipo: 'Cámara',
  mode: 'textil',
  creado: '',
  estado: true
})

const locacionForm = ref({
  empresaId: null,
  plantaId: null,
  lineaId: null,
  nombre: '',
  descripcion: '',
  creado: '',
  modificado: ''
})

const dispositivoForm = ref({
  dispositivoId: null,
  empresaId: null,
  plantaId: null,
  lineaId: null,
  locacionId: null,
  nombre: '',
  device_id: ''
})

const companyOptions = computed(() =>
  companies.value.map(c => ({ value: c.id, label: c.nombre }))
)

const plantOptions = computed(() =>
  plants.value
    .filter(p => !selectedCompany.value || sameID(p.empresa_id, selectedCompany.value))
    .map(p => ({ value: p.id, label: p.nombre }))
)

const lineOptions = computed(() =>
  lines.value
    .filter(l => !selectedPlant.value || sameID(l.planta_id, selectedPlant.value))
    .map(l => ({ value: l.id, label: l.nombre }))
)

const locationOptions = computed(() =>
  locations.value
    .filter(loc => !selectedLine.value || sameID(loc.linea_id, selectedLine.value))
    .map(loc => ({ value: loc.id, label: loc.nombre }))
)

const filteredData = computed(() => {
  switch(activeTab.value) {
    case 'empresa':
      return companies.value
    case 'planta':
      return plants.value.filter(p => !selectedCompany.value || sameID(p.empresa_id, selectedCompany.value))
    case 'linea':
      return lines.value.filter(l =>
        (!selectedPlant.value || sameID(l.planta_id, selectedPlant.value))
      )
    case 'locacion':
      return locations.value.filter(loc =>
        (!selectedLine.value || sameID(loc.linea_id, selectedLine.value))
      )
    case 'dispositivo':
      return devices.value.filter(d =>
        (!selectedCompany.value || sameID(d.empresa_id, selectedCompany.value))
      )
    default:
      return []
  }
})

watch(selectedCompany, () => {
  selectedPlant.value = null
  selectedLine.value = null
  selectedLocation.value = null
})

watch(selectedPlant, () => {
  selectedLine.value = null
  selectedLocation.value = null
})

watch(selectedLine, () => {
  selectedLocation.value = null
})

// Cascada para form de Línea: plantas filtradas por empresa seleccionada en el form
const formLineasPlants = computed(() =>
  lineaForm.value.empresaId
    ? plants.value.filter(p => p.empresa_id === parseInt(lineaForm.value.empresaId))
    : plants.value
)

const subtiposPorTipo = computed(() => ({
  'Producción': [
    { value: 'Cámara', label: 'Cámara' },
    { value: 'Sensor', label: 'Sensor' }
  ],
  'Energía': [
    { value: 'Medidor MC60', label: 'Medidor MC60' },
    { value: 'Rogowski 337', label: 'Rogowski 337' },
    { value: 'Rogowski 537', label: 'Rogowski 537' }
  ]
}))

// Cascada para form de Locación
const formLocPlants = computed(() =>
  locacionForm.value.empresaId
    ? plants.value.filter(p => p.empresa_id === parseInt(locacionForm.value.empresaId))
    : plants.value
)
const formLocLines = computed(() =>
  locacionForm.value.plantaId
    ? lines.value.filter(l => l.planta_id === parseInt(locacionForm.value.plantaId))
    : lines.value
)

// Cascada para form de Dispositivo
const formDispPlants = computed(() =>
  dispositivoForm.value.empresaId
    ? plants.value.filter(p => p.empresa_id === parseInt(dispositivoForm.value.empresaId))
    : plants.value
)
const formDispLines = computed(() =>
  dispositivoForm.value.plantaId
    ? lines.value.filter(l => l.planta_id === parseInt(dispositivoForm.value.plantaId))
    : lines.value
)
const formDispLocations = computed(() =>
  dispositivoForm.value.lineaId
    ? locations.value.filter(loc => loc.linea_id === parseInt(dispositivoForm.value.lineaId))
    : locations.value
)

const provisionedDevices = computed(() =>
  devices.value.filter(d => d.activo && d.device_id)
)

const selectedDeviceInfo = computed(() =>
  devices.value.find(d => d.id === dispositivoForm.value.dispositivoId) || null
)

// Watchers de reset en cascada para forms
watch(() => lineaForm.value.empresaId, () => { lineaForm.value.plantaId = null })
watch(() => lineaForm.value.tipo, (nuevoTipo) => {
  const primero = subtiposPorTipo.value[nuevoTipo]?.[0]?.value ?? ''
  lineaForm.value.subtipo = primero
})
watch(() => locacionForm.value.empresaId, () => { locacionForm.value.plantaId = null; locacionForm.value.lineaId = null })
watch(() => locacionForm.value.plantaId, () => { locacionForm.value.lineaId = null })
watch(() => dispositivoForm.value.empresaId, () => { dispositivoForm.value.plantaId = null; dispositivoForm.value.lineaId = null; dispositivoForm.value.locacionId = null })
watch(() => dispositivoForm.value.plantaId, () => { dispositivoForm.value.lineaId = null; dispositivoForm.value.locacionId = null })
watch(() => dispositivoForm.value.lineaId, () => { dispositivoForm.value.locacionId = null })
watch(() => dispositivoForm.value.dispositivoId, (id) => {
  const d = devices.value.find(x => x.id === id)
  if (d) {
    dispositivoForm.value.nombre = d.nombre
    dispositivoForm.value.device_id = d.device_id
    if (d.empresa_id) dispositivoForm.value.empresaId = d.empresa_id
    if (d.planta_id) dispositivoForm.value.plantaId = d.planta_id
    if (d.linea_id) dispositivoForm.value.lineaId = d.linea_id
    if (d.locacion_id) dispositivoForm.value.locacionId = d.locacion_id
  }
})

const loadData = async () => {
  await execute(async () => {
    const [companiesRes, plantasRes, lineasRes, locRes, devicesRes] = await Promise.all([
      companyService.getAll(),
      plantService.getAll({}),
      lineService.getAll({}),
      locacionService.getAll({}),
      deviceService.getAll({})
    ])
    companies.value = companiesRes.data || []
    plants.value = plantasRes.data || []
    lines.value = lineasRes.data || []
    locations.value = locRes.data || []
    devices.value = devicesRes.data || []
  })
}

const openCreateModal = () => {
  isEditing.value = false
  resetForm()
  openModal()
}

const openEditModal = (item) => {
  if (!item && selectedRows.value.length === 1) {
    item = selectedRows.value[0]
  }
  if (!item) return

  isEditing.value = true
  if (activeTab.value === 'empresa') {
    empresaForm.value = {
      ...item,
      nombreComercial: item.nombre || item.nombreComercial || '',
      documento: item.ruc || item.documento || '',
      correoElectronico: item.email || item.correoElectronico || '',
      representante: item.responsable || item.representante || '',
      direccion: item.direccion || ''
    }
  } else if (activeTab.value === 'planta') {
    plantaForm.value = { ...item, empresaId: item.empresa_id, tipo: item.tipo || 'produccion' }
  } else if (activeTab.value === 'linea') {
    lineaForm.value = { ...item, plantaId: item.planta_id, estado: item.activo }
  } else if (activeTab.value === 'locacion') {
    locacionForm.value = {
      ...item,
      lineaId: item.linea_id,
      empresaId: item.empresa_id
    }
  } else if (activeTab.value === 'dispositivo') {
    dispositivoForm.value = {
      ...item,
      dispositivoId: item.id,
      empresaId: item.empresa_id,
      plantaId: item.planta_id,
      lineaId: item.linea_id,
      locacionId: item.locacion_id,
      device_id: item.device_id
    }
  }
  openModal()
}

const resetForm = () => {
  empresaForm.value = {
    nombreComercial: '',
    descripcion: '',
    denominacion: '',
    documento: '',
    correoElectronico: '',
    tipoDocumento: 'RUC',
    telefono: '',
    telefono1: '',
    sitioWeb: '',
    representante: '',
    telefonoRepresentante: '',
    logo: '',
    direccion: '',
    estado: true
  }
  plantaForm.value = { empresaId: selectedCompany.value, nombre: '', descripcion: '', tipo: 'produccion', zonaHoraria: 'America/Lima', creado: '', modificado: '' }
  lineaForm.value = { empresaId: selectedCompany.value, plantaId: selectedPlant.value, nombre: '', descripcion: '', tipo: 'Producción', codigoAlertas: '', subtipo: 'Cámara', mode: 'textil', creado: '', estado: true }
  locacionForm.value = { empresaId: selectedCompany.value, plantaId: selectedPlant.value, lineaId: selectedLine.value, nombre: '', descripcion: '', creado: '', modificado: '' }
  dispositivoForm.value = { dispositivoId: null, empresaId: selectedCompany.value, plantaId: selectedPlant.value, lineaId: selectedLine.value, locacionId: selectedLocation.value, nombre: '', device_id: '' }
}

const handleSubmit = async () => {
  await execute(async () => {
    if (activeTab.value === 'empresa') {
      const payload = {
        nombre: empresaForm.value.nombreComercial,
        ruc: empresaForm.value.documento,
        direccion: empresaForm.value.direccion,
        telefono: empresaForm.value.telefono,
        email: empresaForm.value.correoElectronico,
        responsable: empresaForm.value.representante,
        estado: !!empresaForm.value.estado
      }
      if (isEditing.value) {
        await companyService.update(empresaForm.value.id, payload)
      } else {
        await companyService.create(payload)
      }
    } else if (activeTab.value === 'planta') {
      const payload = {
        nombre: plantaForm.value.nombre,
        empresa_id: plantaForm.value.empresaId ? parseInt(plantaForm.value.empresaId) : null,
        tipo: plantaForm.value.tipo || 'produccion'
      }
      if (isEditing.value) {
        await plantService.update(plantaForm.value.id, payload)
      } else {
        await plantService.create(payload)
      }
    } else if (activeTab.value === 'linea') {
      const payload = {
        nombre: lineaForm.value.nombre,
        planta_id: lineaForm.value.plantaId ? parseInt(lineaForm.value.plantaId) : null,
        tipo: lineaForm.value.tipo || 'Producción',
        subtipo: lineaForm.value.subtipo || 'Cámara',
        mode: lineaForm.value.mode || 'textil',
        activo: lineaForm.value.estado !== false
      }
      if (isEditing.value) {
        await lineService.update(lineaForm.value.id, payload)
      } else {
        await lineService.create(payload)
      }
    } else if (activeTab.value === 'locacion') {
      const payload = {
        nombre: locacionForm.value.nombre,
        descripcion: locacionForm.value.descripcion || '',
        linea_id: locacionForm.value.lineaId ? parseInt(locacionForm.value.lineaId) : null,
        empresa_id: locacionForm.value.empresaId ? parseInt(locacionForm.value.empresaId) : null
      }
      if (isEditing.value) {
        await locacionService.update(locacionForm.value.id, payload)
      } else {
        await locacionService.create(payload)
      }
    } else if (activeTab.value === 'dispositivo') {
      const targetId = isEditing.value ? dispositivoForm.value.id : dispositivoForm.value.dispositivoId
      const payload = {
        nombre: dispositivoForm.value.nombre,
        empresa_id: dispositivoForm.value.empresaId ? parseInt(dispositivoForm.value.empresaId) : null,
        planta_id: dispositivoForm.value.plantaId ? parseInt(dispositivoForm.value.plantaId) : null,
        linea_id: dispositivoForm.value.lineaId ? parseInt(dispositivoForm.value.lineaId) : null,
        locacion_id: dispositivoForm.value.locacionId ? parseInt(dispositivoForm.value.locacionId) : null
      }
      await deviceService.update(targetId, payload)
    }
    await loadData()
    closeModal()
  })
}

const handleLogoUpload = (event) => {
  const file = event.target.files[0]
  if (file) {
    const reader = new FileReader()
    reader.onload = (e) => {
      empresaForm.value.logo = e.target.result
    }
    reader.readAsDataURL(file)
  }
}

const confirmDelete = (item) => {
  if (!item && selectedRows.value.length > 0) {
    showDeleteConfirm.value = true
    return
  }
  if (!item) return
  selectedRows.value = [item]
  showDeleteConfirm.value = true
}

const handleDelete = async () => {
  await execute(async () => {
    for (const item of selectedRows.value) {
      if (activeTab.value === 'empresa') {
        await companyService.delete(item.id)
      } else if (activeTab.value === 'planta') {
        await plantService.delete(item.id)
      } else if (activeTab.value === 'linea') {
        await lineService.delete(item.id)
      } else if (activeTab.value === 'locacion') {
        await locacionService.delete(item.id)
      } else if (activeTab.value === 'dispositivo') {
        await deviceService.revoke(item.id)
      }
    }
    await loadData()
    selectedRows.value = []
    showDeleteConfirm.value = false
  })
}

onMounted(() => {
  const eid = authStore.user?.empresa_id
  if (eid) selectedCompany.value = eid
  loadData()
})
</script>

<template>
  <div class="empresa-view">
    <div class="tabs-container">
      <button 
        @click="activeTab = 'empresa'" 
        :class="['tab', { active: activeTab === 'empresa' }]"
      >
        EMPRESA
      </button>
      <button 
        @click="activeTab = 'planta'" 
        :class="['tab', { active: activeTab === 'planta' }]"
      >
        PLANTA
      </button>
      <button 
        @click="activeTab = 'linea'" 
        :class="['tab', { active: activeTab === 'linea' }]"
      >
        {{ isEnergyPlanta ? 'RASPBERRY' : 'LÍNEA' }}
      </button>
      <button 
        @click="activeTab = 'locacion'" 
        :class="['tab', { active: activeTab === 'locacion' }]"
      >
        {{ isEnergyPlanta ? 'MEDIDOR' : 'LOCACIÓN' }}
      </button>
      <button 
        @click="activeTab = 'dispositivo'" 
        :class="['tab', { active: activeTab === 'dispositivo' }]"
      >
        DISPOSITIVO
      </button>
    </div>

    <Card class="content-card">
      <div class="action-bar">
        <Button @click="openCreateModal" variant="primary" size="sm">
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
          </svg>
          AGREGAR
        </Button>
        <Button 
          v-if="activeTab === 'empresa' || activeTab === 'planta' || activeTab === 'linea' || activeTab === 'locacion' || activeTab === 'dispositivo'" 
          @click="openEditModal()"
          :disabled="!canEdit"
          variant="secondary" 
          size="sm"
        >
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
          </svg>
          EDITAR
        </Button>
        <Button 
          v-if="activeTab === 'empresa' || activeTab === 'planta' || activeTab === 'linea' || activeTab === 'locacion' || activeTab === 'dispositivo'" 
          @click="confirmDelete()"
          :disabled="!canDelete"
          variant="danger" 
          size="sm"
        >
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
          </svg>
          ELIMINAR
        </Button>
        <Button
          v-if="activeTab === 'planta' && isSuperadmin"
          @click="handleProvision()"
          :disabled="!canProvision || provisioning !== null"
          variant="secondary"
          size="sm"
        >
          <svg v-if="provisioning === null" class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          <svg v-else class="w-4 h-4 mr-1 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" opacity="0.25"/>
            <path d="M12 2a10 10 0 019.95 9" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
          {{ provisioning !== null ? 'PROVISIONANDO...' : canProvision ? 'ASIGNAR TABLAS' : selectedRows.length === 1 && selectedRows[0]?.db_name ? 'BD YA ASIGNADA' : 'ASIGNAR TABLAS' }}
        </Button>
      </div>

      <div v-if="activeTab !== 'empresa'" class="filters-bar">
        <div class="filter-group">
          <label class="filter-label">Empresa</label>
          <SelectFilter 
            v-model="selectedCompany" 
            :options="companyOptions"
            placeholder="Seleccione una Empresa"
          />
        </div>
        <div v-if="activeTab !== 'planta'" class="filter-group">
          <label class="filter-label">Planta</label>
          <SelectFilter 
            v-model="selectedPlant" 
            :options="plantOptions"
            placeholder="Seleccione una Planta"
          />
        </div>
        <div v-if="activeTab === 'locacion' || activeTab === 'dispositivo'" class="filter-group">
          <label class="filter-label">{{ isEnergyPlanta ? 'Raspberry' : 'Línea' }}</label>
          <SelectFilter 
            v-model="selectedLine" 
            :options="lineOptions"
            placeholder="Seleccione una Línea"
          />
        </div>
        <div v-if="activeTab === 'dispositivo'" class="filter-group">
          <label class="filter-label">Locación</label>
          <SelectFilter 
            v-model="selectedLocation" 
            :options="locationOptions"
            placeholder="Toda la Línea"
          />
        </div>
      </div>

      <Loading v-if="loading" />
      <Alert v-else-if="error" type="error" :message="error" />

      <div v-else class="table-container">
        <table v-if="activeTab === 'empresa'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Denominación</th>
              <th>Dirección</th>
              <th>Documento</th>
              <th>Correo electrónico</th>
              <th>Teléf 1</th>
              <th>Teléf 2</th>
              <th>Representante</th>
              <th>Estado</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(item, index) in filteredData" 
              :key="item.id"
              @click="toggleRowSelection(item)"
              :class="{ 'row-selected': isRowSelected(item) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(item)"
                  @click.stop="toggleRowSelection(item)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ item.nombre }}</td>
              <td>{{ item.denominacion || item.ruc }}</td>
              <td>{{ item.direccion }}</td>
              <td>{{ item.documento || item.ruc }}</td>
              <td>{{ item.correo || item.email }}</td>
              <td>{{ item.telefono1 || item.telefono }}</td>
              <td>{{ item.telefono2 || '-' }}</td>
              <td>{{ item.representante || item.responsable }}</td>
              <td>
                <span :class="['badge', item.estado ? 'badge-active' : 'badge-inactive']">
                  {{ item.estado ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>

        <table v-else-if="activeTab === 'planta'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Empresa</th>
              <th>Líneas</th>
              <th>Base de Datos</th>
              <th>Estado</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(item, index) in filteredData" 
              :key="item.id"
              @click="toggleRowSelection(item)"
              :class="{ 'row-selected': isRowSelected(item) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(item)"
                  @click.stop="toggleRowSelection(item)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ item.nombre }}</td>
              <td>{{ item.compania || '-' }}</td>
              <td>{{ item.lineas || 0 }}</td>
              <td>
                <span v-if="item.db_name" class="badge badge-active" :title="item.db_name">
                  ✓ {{ item.db_name }}
                </span>
                <span v-else-if="provisioning === item.id" class="badge" style="background:#e0e7ff;color:#3730a3">
                  ⏳ provisionando...
                </span>
                <span v-else class="badge badge-inactive">sin BD</span>
              </td>
              <td>
                <span :class="['badge', item.activo ? 'badge-active' : 'badge-inactive']">
                  {{ item.activo ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>

        <table v-else-if="activeTab === 'linea'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Planta ID</th>
              <th>Estado</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(item, index) in filteredData" 
              :key="item.id"
              @click="toggleRowSelection(item)"
              :class="{ 'row-selected': isRowSelected(item) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(item)"
                  @click.stop="toggleRowSelection(item)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ item.nombre }}</td>
              <td>{{ plants.find(p => p.id === item.planta_id)?.nombre || item.planta_id || '-' }}</td>
              <td>
                <span :class="['badge', item.activo ? 'badge-active' : 'badge-inactive']">
                  {{ item.activo ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>

        <table v-else-if="activeTab === 'locacion'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Línea</th>
              <th>Descripción</th>
              <th>Estado</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(item, index) in filteredData" 
              :key="item.id"
              @click="toggleRowSelection(item)"
              :class="{ 'row-selected': isRowSelected(item) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(item)"
                  @click.stop="toggleRowSelection(item)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ item.nombre }}</td>
              <td>{{ item.linea_nombre || '-' }}</td>
              <td>{{ item.descripcion || '-' }}</td>
              <td>
                <span :class="['badge', item.activo ? 'badge-active' : 'badge-inactive']">
                  {{ item.activo ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>

        <table v-else-if="activeTab === 'dispositivo'" class="data-table">
          <thead>
            <tr>
              <th class="checkbox-cell"></th>
              <th>N°</th>
              <th>Nombre</th>
              <th>Device ID</th>
              <th>Línea</th>
              <th>Estado</th>
              <th>Activo</th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="(item, index) in filteredData" 
              :key="item.id"
              @click="toggleRowSelection(item)"
              :class="{ 'row-selected': isRowSelected(item) }"
            >
              <td class="checkbox-cell">
                <input 
                  type="checkbox" 
                  class="table-checkbox"
                  :checked="isRowSelected(item)"
                  @click.stop="toggleRowSelection(item)"
                />
              </td>
              <td>{{ index + 1 }}</td>
              <td class="table-name">{{ item.nombre }}</td>
              <td><code>{{ item.device_id }}</code></td>
              <td>{{ lines.find(l => l.id === item.linea_id)?.nombre || '-' }}</td>
              <td>
                <span :class="['badge', item.estado === 'online' ? 'badge-active' : 'badge-inactive']">
                  {{ item.estado || 'offline' }}
                </span>
              </td>
              <td>
                <span :class="['badge', item.activo ? 'badge-active' : 'badge-inactive']">
                  {{ item.activo ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
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

    <Modal v-model="isOpen" @close="closeModal" :title="isEditing ? 'Editar' : 'Agregar'" size="xl">
      <form @submit.prevent="handleSubmit" class="form-container">
        <div v-if="activeTab === 'empresa'" class="form-content">
          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
              </svg>
              Información de la Empresa
            </h4>
            <div class="form-grid-improved">
              <FormField v-model="empresaForm.nombreComercial" label="Nombre Comercial" required />
              <FormField v-model="empresaForm.descripcion" label="Descripción" />
              <FormField v-model="empresaForm.denominacion" label="Denominación" />
              <FormField v-model="empresaForm.documento" label="Documento" required />
              <FormField v-model="empresaForm.correoElectronico" label="Correo electrónico de la empresa" type="email" required class="col-span-2" />
              <div class="form-field-wrapper">
                <label class="field-label">Tipo de documento</label>
                <select v-model="empresaForm.tipoDocumento" class="field-select">
                  <option value="RUC">RUC</option>
                  <option value="DNI">DNI</option>
                  <option value="CE">CE</option>
                  <option value="Pasaporte">Pasaporte</option>
                </select>
              </div>
            </div>
          </div>

          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z"/>
              </svg>
              Contacto
            </h4>
            <div class="form-grid-improved">
              <FormField v-model="empresaForm.telefono" label="Teléfono" required />
              <FormField v-model="empresaForm.telefono1" label="Teléfono 1" />
              <FormField v-model="empresaForm.sitioWeb" label="Sitio web" class="col-span-2" />
              <FormField v-model="empresaForm.representante" label="Representante" />
              <FormField v-model="empresaForm.telefonoRepresentante" label="Teléfono - Representante" />
            </div>
          </div>

          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
              </svg>
              Logo
            </h4>
            <div class="logo-upload-container">
              <div class="logo-preview">
                <img v-if="empresaForm.logo" :src="empresaForm.logo" alt="Logo" class="logo-image" />
                <div v-else class="logo-placeholder">
                  <svg class="logo-placeholder-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                  </svg>
                  <p class="logo-placeholder-text">Formato imagen 200px x 200px</p>
                </div>
              </div>
              <div class="logo-upload-actions">
                <input type="file" id="logoUpload" accept="image/*" class="hidden-input" @change="handleLogoUpload" />
                <label for="logoUpload" class="upload-button">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                  </svg>
                  Seleccionar archivo
                </label>
                <button v-if="empresaForm.logo" type="button" class="clear-button" @click="empresaForm.logo = ''">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                  </svg>
                  Ningún archivo
                </button>
              </div>
            </div>
          </div>

          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"/>
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"/>
              </svg>
              Dirección empresa
            </h4>
            <div class="form-grid-improved">
              <FormField v-model="empresaForm.direccion" label="Dirección" class="col-span-2" required />
            </div>
          </div>

          <div class="form-section-inline">
            <div class="status-toggle">
              <label class="toggle-label">
                <input v-model="empresaForm.estado" type="checkbox" class="toggle-input" />
                <span class="toggle-slider"></span>
                <span class="toggle-text">Activo</span>
              </label>
            </div>
          </div>
        </div>

        <div v-else-if="activeTab === 'planta'" class="form-content">
          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
              </svg>
              Información de la Planta
            </h4>
            <div class="form-grid-improved">
              <div class="form-field-wrapper col-span-2">
                <label class="field-label">Empresas</label>
                <select v-model="plantaForm.empresaId" class="field-select" required>
                  <option value="">Seleccione una empresa</option>
                  <option v-for="company in companies" :key="company.id" :value="company.id">
                    {{ company.nombre }}
                  </option>
                </select>
              </div>
              <FormField v-model="plantaForm.nombre" label="Nombre" required placeholder="INFORMACION" class="col-span-2" />
              <div class="form-field-wrapper col-span-2">
                <label class="field-label">Tipo</label>
                <select v-model="plantaForm.tipo" class="field-select" required>
                  <option value="produccion">Producción</option>
                  <option value="energia">Energía</option>
                </select>
              </div>
              <FormField v-model="plantaForm.descripcion" label="Descripción" placeholder="Información" class="col-span-2" />
              <div class="form-field-wrapper col-span-2">
                <label class="field-label">Zona Horaria</label>
                <select v-model="plantaForm.zonaHoraria" class="field-select" required>
                  <option value="America/Lima">America/Lima</option>
                  <option value="America/Bogota">America/Bogota</option>
                  <option value="America/Mexico_City">America/Mexico_City</option>
                  <option value="America/Argentina/Buenos_Aires">America/Argentina/Buenos_Aires</option>
                  <option value="America/Santiago">America/Santiago</option>
                  <option value="America/Caracas">America/Caracas</option>
                  <option value="America/Sao_Paulo">America/Sao_Paulo</option>
                  <option value="America/Bahia">America/Bahia</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="activeTab === 'linea'" class="form-content">
          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z"/>
              </svg>
              Información de la Línea
            </h4>
            <div class="form-grid-improved">
              <div class="form-field-wrapper">
                <label class="field-label">Empresas</label>
                <select v-model="lineaForm.empresaId" class="field-select" required>
                  <option value="">Seleccione una empresa</option>
                  <option v-for="company in companies" :key="company.id" :value="company.id">
                    {{ company.nombre }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper">
                <label class="field-label">Plantas</label>
                <select v-model="lineaForm.plantaId" class="field-select" required>
                  <option value="">Seleccione una planta</option>
                  <option v-for="plant in formLineasPlants" :key="plant.id" :value="plant.id">
                    {{ plant.nombre }}
                  </option>
                </select>
              </div>
              <FormField v-model="lineaForm.nombre" label="Nombre" required placeholder="INFORMACION" class="col-span-2" />
              <FormField v-model="lineaForm.descripcion" label="Descripción" placeholder="Descripción" class="col-span-2" />
              <div class="form-field-wrapper">
                <label class="field-label">Tipo</label>
                <select v-model="lineaForm.tipo" class="field-select" required>
                  <option value="Producción">Producción</option>
                  <option value="Energía">Energía</option>
                </select>
              </div>
              <FormField v-model="lineaForm.codigoAlertas" label="Código de alertas" placeholder="Código de alertas">
                <template #label>
                  <div class="flex items-center gap-2">
                    <span>Código de alertas</span>
                    <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                  </div>
                </template>
              </FormField>
              <div class="form-field-wrapper col-span-2">
                <label class="field-label">Subtipo</label>
                <select v-model="lineaForm.subtipo" class="field-select" required>
                  <option v-for="op in subtiposPorTipo[lineaForm.tipo]" :key="op.value" :value="op.value">
                    {{ op.label }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper col-span-2">
                <label class="field-label">Modo de detección</label>
                <select v-model="lineaForm.mode" class="field-select" required>
                  <option value="textil">Textil (microparada &lt; 2 min · snapshot cada 30 min)</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="activeTab === 'locacion'" class="form-content">
          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"/>
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"/>
              </svg>
              Información de la Locación
            </h4>
            <div class="form-grid-improved">
              <div class="form-field-wrapper">
                <label class="field-label">Empresas</label>
                <select v-model="locacionForm.empresaId" class="field-select" required>
                  <option value="">Seleccione una empresa</option>
                  <option v-for="company in companies" :key="company.id" :value="company.id">
                    {{ company.nombre }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper">
                <label class="field-label">Plantas</label>
                <select v-model="locacionForm.plantaId" class="field-select" required>
                  <option value="">Seleccione una planta</option>
                  <option v-for="plant in formLocPlants" :key="plant.id" :value="plant.id">
                    {{ plant.nombre }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper col-span-2">
                <label class="field-label">Líneas</label>
                <select v-model="locacionForm.lineaId" class="field-select" required>
                  <option value="">Seleccione una línea</option>
                  <option v-for="line in formLocLines" :key="line.id" :value="line.id">
                    {{ line.nombre }}
                  </option>
                </select>
              </div>
              <FormField v-model="locacionForm.nombre" label="Nombre" required placeholder="INFORMACION" class="col-span-2" />
              <FormField v-model="locacionForm.descripcion" label="Descripción" placeholder="Descripción" class="col-span-2" />
            </div>
          </div>
        </div>

        <div v-else-if="activeTab === 'dispositivo'" class="form-content">
          <div class="form-section">
            <h4 class="section-title">
              <svg class="section-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/>
              </svg>
              Información del Dispositivo
            </h4>
            <div class="form-grid-improved">
              <div v-if="!isEditing" class="form-field-wrapper col-span-2">
                <label class="field-label">Dispositivo provisionado <span class="text-red-500">*</span></label>
                <select v-model="dispositivoForm.dispositivoId" class="field-select" required>
                  <option :value="null">Seleccione un dispositivo</option>
                  <option v-for="d in provisionedDevices" :key="d.id" :value="d.id">
                    {{ d.device_id }} — {{ d.nombre }}{{ d.linea_id ? ' (ya asignado)' : '' }}
                  </option>
                </select>
                <p v-if="provisionedDevices.length === 0" class="field-hint">No hay dispositivos provisionados. Crea uno en Configuración &gt; Dispositivos.</p>
              </div>
              <div v-if="isEditing" class="form-field-wrapper col-span-2">
                <label class="field-label">Dispositivo</label>
                <input :value="dispositivoForm.device_id + ' — ' + dispositivoForm.nombre" class="field-input" disabled />
              </div>
              <div class="form-field-wrapper">
                <label class="field-label">Empresa</label>
                <select v-model="dispositivoForm.empresaId" class="field-select" required>
                  <option value="">Seleccione una empresa</option>
                  <option v-for="company in companies" :key="company.id" :value="company.id">
                    {{ company.nombre }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper">
                <label class="field-label">Planta</label>
                <select v-model="dispositivoForm.plantaId" class="field-select" required>
                  <option value="">Seleccione una planta</option>
                  <option v-for="plant in formDispPlants" :key="plant.id" :value="plant.id">
                    {{ plant.nombre }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper">
                <label class="field-label">Línea <span class="text-red-500">*</span></label>
                <select v-model="dispositivoForm.lineaId" class="field-select" required>
                  <option value="">Seleccione una línea</option>
                  <option v-for="line in formDispLines" :key="line.id" :value="line.id">
                    {{ line.nombre }}
                  </option>
                </select>
              </div>
              <div class="form-field-wrapper">
                <label class="field-label">Locación</label>
                <select v-model="dispositivoForm.locacionId" class="field-select">
                  <option :value="null">Sin locación</option>
                  <option v-for="loc in formDispLocations" :key="loc.id" :value="loc.id">
                    {{ loc.nombre }}
                  </option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <div class="form-footer">
          <Button type="button" @click="closeModal" variant="secondary" size="md">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            Cancelar
          </Button>
          <Button type="submit" variant="primary" size="md" :loading="loading">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
            {{ isEditing ? 'Actualizar' : 'Guardar' }}
          </Button>
        </div>
      </form>
    </Modal>

    <Modal v-model="showDeleteConfirm" @close="showDeleteConfirm = false" title="Confirmar Eliminación" size="sm">
      <div class="delete-confirm">
        <svg class="delete-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
        </svg>
        <p class="delete-text">
          ¿Está seguro de eliminar 
          <strong>{{ selectedRows.length }} {{ selectedRows.length === 1 ? 'registro' : 'registros' }}</strong>?
        </p>
        <p class="delete-warning">Esta acción no se puede deshacer.</p>
      </div>
      <div class="form-actions">
        <Button @click="showDeleteConfirm = false" variant="secondary">Cancelar</Button>
        <Button @click="handleDelete" variant="danger" :loading="loading">Eliminar</Button>
      </div>
    </Modal>

    <!-- Modal confirmación provisioning -->
    <Modal v-model="showProvisionConfirm" @close="showProvisionConfirm = false" title="Asignar Base de Datos" size="sm" :closeOnBackdrop="provisioning === null">
      <div class="delete-confirm">
        <svg class="w-16 h-16 mx-auto mb-4 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <ellipse cx="12" cy="5" rx="9" ry="3" stroke-width="2"/>
          <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" stroke-width="2"/>
          <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" stroke-width="2"/>
          <path d="M12 8v6m-3-3h6" stroke-width="2"/>
        </svg>
        <p class="delete-text">
          Crear BD dedicada para
          <strong>{{ plantToProvision?.nombre }}</strong>
        </p>
        <p class="delete-warning">
          Se creará la base de datos <code class="bg-gray-100 px-1 rounded text-xs font-mono">mentor_planta_{{ plantToProvision?.id }}</code> con un usuario y schemas independientes por cada línea registrada.
        </p>
        <div v-if="provisionError" class="mt-3 rounded-md bg-red-50 border border-red-200 p-3 text-sm text-red-700 text-left">
          {{ provisionError }}
        </div>
      </div>
      <div class="form-actions">
        <Button @click="showProvisionConfirm = false" variant="secondary" :disabled="provisioning !== null">Cancelar</Button>
        <Button @click="confirmProvision" variant="primary" :loading="provisioning !== null">
          <svg v-if="!provisioning" class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
          </svg>
          {{ provisioning !== null ? 'Provisionando...' : 'Confirmar' }}
        </Button>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.empresa-view {
  @apply space-y-0;
}

.tabs-container {
  @apply flex border-b border-gray-200 bg-white;
}

.tab {
  @apply px-6 py-3 text-sm font-medium text-gray-600 border-b-2 border-transparent hover:text-gray-900 hover:border-gray-300 transition-colors;
}

.tab.active {
  @apply text-blue-700 border-blue-700;
}

.content-card {
  @apply rounded-t-none border-t-0;
}

.action-bar {
  @apply flex gap-2 p-4 bg-blue-900 rounded-t-lg;
}

.filters-bar {
  @apply grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 p-4 bg-white border-b border-gray-200;
}

.filter-group {
  @apply space-y-1;
}

.filter-label {
  @apply block text-sm font-medium text-gray-700;
}

.table-container {
  @apply overflow-x-auto;
}

.data-table {
  @apply w-full text-sm;
}

.data-table thead {
  @apply bg-gray-100;
}

.data-table th {
  @apply px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider border-b border-gray-200;
}

.data-table tbody tr {
  @apply transition-colors duration-150 cursor-pointer;
}

.data-table tbody tr:hover {
  @apply bg-blue-50;
}

.data-table tbody tr:nth-child(even) {
  @apply bg-gray-50;
}

.data-table tbody tr:nth-child(even):hover {
  @apply bg-blue-50;
}

.data-table tbody tr.row-selected {
  @apply bg-blue-100 border-l-4 border-blue-600;
}

.data-table td {
  @apply px-4 py-3 text-gray-900 border-b border-gray-200;
}

.data-table td.table-name {
  font-weight: 600;
  color: #111827;
}

.checkbox-cell {
  @apply w-12 text-center;
}

.table-checkbox {
  @apply w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-2 focus:ring-blue-500 cursor-pointer;
}

.badge {
  @apply px-3 py-1 rounded-full text-xs font-semibold inline-block;
}

.badge-active {
  @apply bg-green-100 text-green-800;
}

.badge-inactive {
  @apply bg-gray-100 text-gray-600;
}

.pagination {
  @apply flex items-center justify-center gap-4 p-4 bg-gray-50 border-t border-gray-200;
}

.pagination-btn {
  @apply px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded hover:bg-gray-50;
}

.pagination-info {
  @apply text-sm font-medium text-gray-700;
}


.form-container {
  padding: 0;
}

.form-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-section {
  background: linear-gradient(to bottom, #f8fafc, #ffffff);
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.form-section-inline {
  padding: 1rem 0;
  border-top: 1px solid #e5e7eb;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 1rem;
  font-weight: 600;
  color: #1e40af;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 2px solid #dbeafe;
}

.section-icon {
  width: 1.5rem;
  height: 1.5rem;
  color: #3b82f6;
}

.form-grid-improved {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.25rem;
}

.form-grid-improved .col-span-2 {
  grid-column: span 2;
}

.status-toggle {
  display: flex;
  justify-content: flex-start;
  padding: 1rem;
  background: #f0f9ff;
  border-radius: 8px;
  border: 1px solid #bfdbfe;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  user-select: none;
}

.toggle-input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 24px;
  background-color: #cbd5e1;
  border-radius: 24px;
  transition: all 0.3s ease;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.1);
}

.toggle-slider::before {
  content: '';
  position: absolute;
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  border-radius: 50%;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.toggle-input:checked + .toggle-slider {
  background-color: #3b82f6;
}

.toggle-input:checked + .toggle-slider::before {
  transform: translateX(24px);
}

.toggle-text {
  font-size: 0.95rem;
  font-weight: 500;
  color: #1e293b;
}

.form-footer {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  padding-top: 1.5rem;
  margin-top: 1rem;
  border-top: 2px solid #e5e7eb;
}

.form-footer button {
  min-width: 120px;
}

.form-field-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  margin-bottom: 0.25rem;
}

.field-select {
  width: 100%;
  padding: 0.625rem 0.75rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: white;
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
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

.logo-upload-container {
  display: flex;
  gap: 1.5rem;
  align-items: flex-start;
}

.logo-preview {
  flex-shrink: 0;
  width: 200px;
  height: 200px;
  border: 2px dashed #cbd5e1;
  border-radius: 8px;
  overflow: hidden;
  background: #f8fafc;
}

.logo-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.logo-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: #94a3b8;
}

.logo-placeholder-icon {
  width: 3rem;
  height: 3rem;
}

.logo-placeholder-text {
  font-size: 0.75rem;
  text-align: center;
  padding: 0 1rem;
}

.logo-upload-actions {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding-top: 1rem;
}

.hidden-input {
  display: none;
}

.upload-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: white;
  background-color: #3b82f6;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
}

.upload-button:hover {
  background-color: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 4px 6px rgba(59, 130, 246, 0.2);
}

.clear-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #dc2626;
  background-color: #fee2e2;
  border: 1px solid #fecaca;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-button:hover {
  background-color: #fecaca;
  border-color: #fca5a5;
}

.form-grid {
  @apply grid grid-cols-2 gap-4;
}

.checkbox-field {
  @apply col-span-2;
}

.checkbox-label {
  @apply flex items-center gap-3 cursor-pointer;
}

.checkbox-input {
  @apply w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-2 focus:ring-blue-500;
}

.checkbox-text {
  @apply text-sm font-medium text-gray-700;
}

.form-actions {
  @apply flex gap-3 justify-end pt-4 border-t border-gray-200;
}

.delete-confirm {
  @apply text-center py-4;
}

.delete-icon {
  @apply w-16 h-16 mx-auto text-red-500 mb-4;
}

.delete-text {
  @apply text-gray-900 mb-2;
}

.delete-warning {
  @apply text-sm text-gray-500;
}
</style>
