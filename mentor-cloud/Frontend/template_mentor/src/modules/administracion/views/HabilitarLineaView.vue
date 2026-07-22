<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'
import Loading from '@/shared/components/ui/Loading.vue'
import Alert from '@/shared/components/ui/Alert.vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'

// ── Estado ────────────────────────────────────────────────────────
const companias  = ref([])
const plantas    = ref([])
const lineas     = ref([])

const companiaId = ref(null)
const plantaId   = ref(null)

const cargando      = ref(false)
const provisionando = ref(null)  // lineaId o 'planta' que se está provisionando
const successMsg    = ref('')
const errorMsg      = ref('')

// ── Computed ──────────────────────────────────────────────────────
const plantasFiltradas = computed(() =>
  companiaId.value
    ? plantas.value.filter(p => p.empresa_id === companiaId.value)
    : plantas.value
)

const plantaActual = computed(() =>
  plantas.value.find(p => p.id === plantaId.value) || null
)

const plantaProvisionada = computed(() =>
  !!plantaActual.value?.db_name
)

const lineasFiltradas = computed(() =>
  plantaId.value
    ? lineas.value.filter(l => l.planta_id === plantaId.value)
    : []
)

// db_schemas viene del backend como array de IDs de línea que ya tienen schema
const schemasCreados = computed(() =>
  plantaActual.value?.db_schemas || []
)

function lineaProvisionada(lineaId) {
  return schemasCreados.value.includes(lineaId)
}

// ── Carga inicial ─────────────────────────────────────────────────
onMounted(async () => {
  cargando.value = true
  try {
    const [empR, plnR, linR] = await Promise.all([
      companyService.getAll(),
      plantService.getAll(),
      lineService.getAll()
    ])
    companias.value = empR.data?.data || empR.data || []
    plantas.value   = plnR.data?.data || plnR.data || []
    lineas.value    = linR.data?.data || linR.data || []

    if (companias.value.length > 0) companiaId.value = companias.value[0].id
  } catch (e) {
    errorMsg.value = 'Error cargando datos: ' + (e.message || e)
  } finally {
    cargando.value = false
  }
})

async function seleccionarPlanta(id) {
  plantaId.value = id
  successMsg.value = ''
  errorMsg.value   = ''
}

// ── Provisionar Planta ────────────────────────────────────────────
async function provisionarPlanta() {
  if (!plantaId.value) return
  provisionando.value = 'planta'
  errorMsg.value = ''
  try {
    await plantService.provision(plantaId.value)
    successMsg.value = `Planta provisionada. Base de datos mentor_planta_${plantaId.value} creada.`
    await recargarPlantas()
  } catch (e) {
    errorMsg.value = e.response?.data?.error || e.message || 'Error al provisionar planta'
  } finally {
    provisionando.value = null
  }
}

// ── Provisionar Línea ─────────────────────────────────────────────
async function provisionarLinea(lineaId, lineaNombre) {
  provisionando.value = lineaId
  errorMsg.value = ''
  try {
    await lineService.provisionLinea(plantaId.value, lineaId)
    successMsg.value = `Schema linea_${lineaId} creado para "${lineaNombre}".`
    await recargarPlantas()
  } catch (e) {
    errorMsg.value = e.response?.data?.error || e.message || 'Error al provisionar línea'
  } finally {
    provisionando.value = null
  }
}

async function recargarPlantas() {
  const plnR = await plantService.getAll()
  plantas.value = plnR.data?.data || plnR.data || []
}
</script>

<template>
  <div class="habilitar-linea-view">
    <div class="page-header">
      <h1 class="page-title">HABILITAR LÍNEA</h1>
    </div>

    <Alert v-if="successMsg" type="success" :message="successMsg" class="mb-4" />
    <Alert v-if="errorMsg"   type="error"   :message="errorMsg"   class="mb-4" />

    <Card class="content-card">
      <!-- Filtros -->
      <div class="filtros-section">
        <div class="filter-item">
          <label class="filter-label">Compañía</label>
          <select v-model="companiaId" class="filter-select" @change="plantaId = null">
            <option v-for="c in companias" :key="c.id" :value="c.id">{{ c.nombre }}</option>
          </select>
        </div>
        <div class="filter-item">
          <label class="filter-label">Planta</label>
          <select v-model="plantaId" class="filter-select" @change="seleccionarPlanta(plantaId)">
            <option :value="null">Seleccione...</option>
            <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
          </select>
        </div>
      </div>

      <Loading v-if="cargando" />

      <div v-else-if="plantaId" class="provision-container">

        <!-- Status de la Planta -->
        <div class="planta-status-card" :class="plantaProvisionada ? 'status-ok' : 'status-pending'">
          <div class="planta-info">
            <span class="planta-label">{{ plantaActual?.nombre }}</span>
            <span v-if="plantaProvisionada" class="badge badge-ok">
              ✓ Base de datos: {{ plantaActual.db_name }}
            </span>
            <span v-else class="badge badge-pending">⚠ Sin base de datos</span>
          </div>
          <Button
            v-if="!plantaProvisionada"
            label="Provisionar Planta"
            variant="primary"
            :loading="provisionando === 'planta'"
            @click="provisionarPlanta"
          />
        </div>

        <!-- Tabla de líneas -->
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Línea</th>
                <th>Tipo</th>
                <th>Schema BD</th>
                <th>Estado</th>
                <th>Acción</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="lineasFiltradas.length === 0">
                <td colspan="6" class="empty-row">No hay líneas registradas en esta planta.</td>
              </tr>
              <tr v-for="linea in lineasFiltradas" :key="linea.id">
                <td>{{ linea.id }}</td>
                <td>{{ linea.nombre }}</td>
                <td>
                  <span :class="linea.tipo === 'Energía' ? 'badge badge-energia' : 'badge badge-produccion'">
                    {{ linea.tipo || 'Producción' }}
                  </span>
                </td>
                <td class="schema-name">linea_{{ linea.id }}</td>
                <td>
                  <span v-if="lineaProvisionada(linea.id)" class="badge badge-ok">✓ Activa</span>
                  <span v-else-if="!plantaProvisionada" class="badge badge-disabled">Planta sin BD</span>
                  <span v-else class="badge badge-pending">Pendiente</span>
                </td>
                <td>
                  <button
                    v-if="plantaProvisionada && !lineaProvisionada(linea.id)"
                    class="btn-provision"
                    :disabled="provisionando === linea.id"
                    @click="provisionarLinea(linea.id, linea.nombre)"
                  >
                    {{ provisionando === linea.id ? 'Creando...' : 'Provisionar' }}
                  </button>
                  <span v-else-if="lineaProvisionada(linea.id)" class="text-ok">—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

      </div>

      <div v-else class="empty-state">
        Seleccione una planta para ver el estado de sus líneas.
      </div>
    </Card>
  </div>
</template>

<style scoped>
.habilitar-linea-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding-bottom: 2rem;
}

.page-header {
  background-color: #1e3a8a;
  padding: 1rem 2rem;
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
  border: 1px solid #e5e7eb;
}

.filtros-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1rem;
  padding: clamp(1rem, 2vw, 1.5rem);
  background-color: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
}

.filter-item { display: flex; flex-direction: column; gap: 0.5rem; }

.filter-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.filter-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  background: white;
  color: #374151;
}

.provision-container { padding: 1.5rem; display: flex; flex-direction: column; gap: 1.5rem; }

.planta-status-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-radius: 0.5rem;
  border: 1.5px solid;
}

.status-ok      { background: #f0fdf4; border-color: #22c55e; }
.status-pending { background: #fffbeb; border-color: #f59e0b; }

.planta-info { display: flex; align-items: center; gap: 1rem; }

.planta-label { font-weight: 700; font-size: 1rem; color: #1e293b; }

.badge {
  display: inline-block;
  padding: 0.2rem 0.65rem;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 600;
}

.badge-ok         { background: #dcfce7; color: #15803d; }
.badge-pending    { background: #fef9c3; color: #92400e; }
.badge-disabled   { background: #f3f4f6; color: #9ca3af; }
.badge-energia    { background: #fef3c7; color: #b45309; }
.badge-produccion { background: #dbeafe; color: #1d4ed8; }

.table-container {
  overflow-x: auto;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
}

.data-table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }

.data-table thead { background: #f9fafb; }

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-weight: 600;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
}

.data-table td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e5e7eb;
  color: #374151;
}

.data-table tbody tr:hover { background: #f9fafb; }

.schema-name { font-family: monospace; font-size: 0.8rem; color: #6366f1; }

.btn-provision {
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 0.375rem;
  padding: 0.35rem 0.9rem;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-provision:hover:not(:disabled) { background: #1d4ed8; }
.btn-provision:disabled { background: #93c5fd; cursor: not-allowed; }

.text-ok { color: #9ca3af; font-size: 0.85rem; }

.empty-row { text-align: center; color: #9ca3af; padding: 2rem; }

.empty-state {
  padding: 3rem;
  text-align: center;
  color: #9ca3af;
  font-size: 0.95rem;
}

@media (max-width: 768px) {
  .filtros-section { grid-template-columns: 1fr; }
}
</style>
