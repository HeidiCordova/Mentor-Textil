<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { useAuthStore } from '@/stores/auth'
import TurnoTimelineEditor from '@/modules/administracion/components/turnos/TurnoTimelineEditor.vue'

const authStore = useAuthStore()

const empresas = ref([])
const plantas  = ref([])
const lineas   = ref([])

const empresaId = ref('')
const plantaId  = ref('')
const lineaId   = ref('')

const plantasFiltradas = computed(() =>
  empresaId.value ? plantas.value.filter(p => p.empresa_id === parseInt(empresaId.value)) : []
)

const lineaSeleccionada = computed(() =>
  lineaId.value ? lineas.value.find(l => l.id === parseInt(lineaId.value)) : null
)

async function cargarEmpresas() {
  const res = await companyService.getAll()
  empresas.value = res?.data ?? []
}

async function cargarPlantas() {
  if (!empresaId.value) { plantas.value = []; return }
  const res = await plantService.getAll({ empresa_id: empresaId.value })
  plantas.value = res?.data ?? []
}

async function cargarLineas() {
  if (!plantaId.value) { lineas.value = []; return }
  const res = await lineService.getAll({ planta_id: plantaId.value })
  lineas.value = res?.data ?? []
}

watch(empresaId, () => { plantaId.value = ''; lineaId.value = ''; plantas.value = []; lineas.value = []; cargarPlantas() })
watch(plantaId,  () => { lineaId.value = ''; lineas.value = []; cargarLineas() })

onMounted(async () => {
  await cargarEmpresas()
  const eid = authStore.user?.empresa_id
  if (eid) {
    empresaId.value = String(eid)
    await cargarPlantas()
  }
})
</script>

<template>
  <div class="tv-wrap">
    <div class="tv-header">
      <h1 class="tv-title">Configuración de Turnos</h1>
    </div>

    <div class="tv-filters">
      <div class="tv-filter-group">
        <label class="tv-label">Empresa</label>
        <select v-model="empresaId" class="tv-select">
          <option value="">Seleccionar empresa</option>
          <option v-for="e in empresas" :key="e.id" :value="String(e.id)">{{ e.nombre }}</option>
        </select>
      </div>

      <div class="tv-filter-group">
        <label class="tv-label">Planta</label>
        <select v-model="plantaId" class="tv-select" :disabled="!empresaId">
          <option value="">Seleccionar planta</option>
          <option v-for="p in plantasFiltradas" :key="p.id" :value="String(p.id)">{{ p.nombre }}</option>
        </select>
      </div>

      <div class="tv-filter-group">
        <label class="tv-label">Línea (opcional)</label>
        <select v-model="lineaId" class="tv-select" :disabled="!plantaId">
          <option value="">Todas las líneas</option>
          <option v-for="l in lineas" :key="l.id" :value="String(l.id)">{{ l.nombre }}</option>
        </select>
      </div>
    </div>

    <div class="tv-body">
      <TurnoTimelineEditor 
        :planta-id="plantaId || null" 
        :linea-id="lineaId || null"
        :linea="lineaSeleccionada" 
      />
    </div>
  </div>
</template>

<style scoped>
.tv-wrap{display:flex;flex-direction:column;gap:20px;padding:24px}
.tv-header{display:flex;align-items:center;justify-content:space-between}
.tv-title{font-size:18px;font-weight:700;color:#1e293b;margin:0}
.tv-filters{display:flex;gap:16px;flex-wrap:wrap;background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;padding:16px}
.tv-filter-group{display:flex;flex-direction:column;gap:4px;min-width:200px}
.tv-label{font-size:12px;font-weight:600;color:#475569;text-transform:uppercase;letter-spacing:.04em}
.tv-select{padding:7px 12px;border:1px solid #cbd5e1;border-radius:7px;font-size:13px;color:#1e293b;background:#fff;cursor:pointer}
.tv-select:disabled{opacity:.5;cursor:not-allowed}
.tv-select:focus{outline:2px solid #6366f1;border-color:transparent}
.tv-body{background:#fff;border:1px solid #e2e8f0;border-radius:10px;padding:20px}
</style>
