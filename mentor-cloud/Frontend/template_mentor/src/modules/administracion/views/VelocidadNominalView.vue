<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { velocidadNominalService } from '@/api/services/velocidadNominal.service'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

// ─── Tab activo ───────────────────────────────────────────────────────────────
const activeTab = ref('config')

// ─── Filtros ──────────────────────────────────────────────────────────────────
const empresas = ref([])
const plantas  = ref([])
const lineas   = ref([])

const empresaID = ref(null)
const plantaID  = ref(null)
const lineaID   = ref(null)
const busqueda  = ref('')

const plantasFiltradas = computed(() =>
  empresaID.value ? plantas.value.filter(p => p.empresa_id === Number(empresaID.value)) : []
)
const lineasFiltradas = computed(() =>
  plantaID.value ? lineas.value.filter(l => l.planta_id === Number(plantaID.value)) : []
)

// ─── Config: datos ────────────────────────────────────────────────────────────
const filas       = ref([])
const editadas    = ref({})
const motivo      = ref('')
const loading     = ref(false)
const guardando   = ref(false)
const guardadoOk  = ref(false)
const errorMsg    = ref('')

const filasFiltradas = computed(() => {
  const b = busqueda.value.trim().toLowerCase()
  if (!b) return filas.value
  return filas.value.filter(f =>
    f.sku.toLowerCase().includes(b) || f.descripcion.toLowerCase().includes(b)
  )
})

const hayCambios = computed(() => {
  return filas.value.some(f => {
    const d = editadas.value[f.producto_id]
    if (!d) return false
    return Number(d.velocidad_us) !== f.velocidad_us ||
           Number(d.factor_conv)  !== f.factor_conv
  })
})

async function cargarFiltros() {
  const [emp, pla] = await Promise.all([
    companyService.getAll(),
    plantService.getAll()
  ])
  empresas.value = emp.data ?? []
  plantas.value  = pla.data ?? []
  if (!empresaID.value && auth.user?.empresa_id)
    empresaID.value = auth.user.empresa_id
}

async function cargarDatos() {
  if (!lineaID.value) { filas.value = []; editadas.value = {}; return }
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await velocidadNominalService.getByLinea({ linea_id: lineaID.value })
    filas.value    = res.data ?? []
    editadas.value = {}
    filas.value.forEach(f => {
      editadas.value[f.producto_id] = {
        velocidad_us: f.velocidad_us,
        factor_conv:  f.factor_conv
      }
    })
  } catch (e) {
    errorMsg.value = e?.response?.data?.error || e.message || 'Error al cargar'
  } finally {
    loading.value = false
  }
}

async function guardarTodo() {
  if (!lineaID.value) return
  guardando.value  = true
  guardadoOk.value = false
  errorMsg.value   = ''
  try {
    const motivoTrim = motivo.value.trim()
    const payload = filas.value.map(f => ({
      linea_id:     f.linea_id,
      producto_id:  f.producto_id,
      velocidad_us: Number(editadas.value[f.producto_id]?.velocidad_us ?? 0),
      factor_conv:  Number(editadas.value[f.producto_id]?.factor_conv  ?? 1),
      motivo:       motivoTrim || undefined
    }))
    await velocidadNominalService.save(payload)
    filas.value = filas.value.map(f => ({
      ...f,
      velocidad_us: editadas.value[f.producto_id]?.velocidad_us ?? f.velocidad_us,
      factor_conv:  editadas.value[f.producto_id]?.factor_conv  ?? f.factor_conv
    }))
    motivo.value     = ''
    guardadoOk.value = true
    setTimeout(() => { guardadoOk.value = false }, 2500)
  } catch (e) {
    errorMsg.value = e?.response?.data?.error || e.message || 'Error al guardar'
  } finally {
    guardando.value = false
  }
}

// ─── Historial ────────────────────────────────────────────────────────────────
const logRows    = ref([])
const logLoading = ref(false)
const logError   = ref('')
const logLimit   = ref(100)

async function cargarLog() {
  if (!lineaID.value) { logRows.value = []; return }
  logLoading.value = true
  logError.value   = ''
  try {
    const res = await velocidadNominalService.getLog({
      linea_id: lineaID.value,
      limit: logLimit.value
    })
    logRows.value = res.data ?? []
  } catch (e) {
    logError.value = e?.response?.data?.error || e.message || 'Error al cargar historial'
  } finally {
    logLoading.value = false
  }
}

function formatFecha(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  if (isNaN(d)) return ts
  return d.toLocaleString('es-AR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit'
  })
}

// ─── Watches ──────────────────────────────────────────────────────────────────
watch(empresaID, () => { plantaID.value = null; lineaID.value = null })
watch(plantaID, async () => {
  if (!plantaID.value) { lineas.value = []; lineaID.value = null; return }
  const res = await lineService.getAll({ planta_id: plantaID.value })
  lineas.value = res.data ?? []; lineaID.value = null
})
watch(lineaID, () => {
  busqueda.value = ''
  cargarDatos()
  if (activeTab.value === 'historial') cargarLog()
  if (activeTab.value === 'motivos') cargarMotivos()
})
watch(activeTab, (tab) => {
  if (tab === 'historial' && lineaID.value) cargarLog()
  if (tab === 'motivos'   && lineaID.value) cargarMotivos()
})

onMounted(cargarFiltros)

// ─── Motivos ──────────────────────────────────────────────────────────────────
const motivosList    = ref([])
const motivosLoading = ref(false)
const motivosError   = ref('')
const nuevoMotivo    = ref('')
const editandoID     = ref(null)
const editandoTexto  = ref('')
const guardandoMot   = ref(false)

async function cargarMotivos() {
  if (!lineaID.value) { motivosList.value = []; return }
  motivosLoading.value = true
  motivosError.value   = ''
  try {
    const res = await velocidadNominalService.getMotivos({ linea_id: lineaID.value })
    motivosList.value = res.data ?? []
  } catch (e) {
    motivosError.value = e?.response?.data?.error || e.message || 'Error al cargar motivos'
  } finally {
    motivosLoading.value = false
  }
}

async function agregarMotivo() {
  const texto = nuevoMotivo.value.trim()
  if (!texto || !lineaID.value) return
  guardandoMot.value = true
  try {
    await velocidadNominalService.createMotivo(lineaID.value, { texto })
    nuevoMotivo.value = ''
    await cargarMotivos()
  } catch (e) {
    motivosError.value = e?.response?.data?.error || e.message || 'Error al crear'
  } finally {
    guardandoMot.value = false
  }
}

function iniciarEdicion(m) {
  editandoID.value    = m.id
  editandoTexto.value = m.texto
}

async function guardarEdicion(m) {
  const texto = editandoTexto.value.trim()
  if (!texto) return
  guardandoMot.value = true
  try {
    await velocidadNominalService.updateMotivo(lineaID.value, m.id, { texto })
    editandoID.value = null
    await cargarMotivos()
  } catch (e) {
    motivosError.value = e?.response?.data?.error || e.message || 'Error al guardar'
  } finally {
    guardandoMot.value = false
  }
}

async function toggleActivo(m) {
  guardandoMot.value = true
  try {
    if (m.activo) {
      await velocidadNominalService.deleteMotivo(lineaID.value, m.id)
    } else {
      await velocidadNominalService.updateMotivo(lineaID.value, m.id, { activo: true })
    }
    await cargarMotivos()
  } catch (e) {
    motivosError.value = e?.response?.data?.error || e.message || 'Error al actualizar'
  } finally {
    guardandoMot.value = false
  }
}
</script>

<template>
  <div class="vn-wrap">

    <div class="vn-header">
      <h1 class="vn-title">OEE – VELOCIDAD NOMINAL</h1>
    </div>

    <!-- Filtros -->
    <div class="vn-filters">
      <div class="vn-filter-group">
        <label>Empresa</label>
        <select v-model="empresaID" class="vn-select">
          <option value="">Seleccionar...</option>
          <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
        </select>
      </div>
      <div class="vn-filter-group">
        <label>Planta</label>
        <select v-model="plantaID" class="vn-select" :disabled="!empresaID">
          <option value="">Seleccionar...</option>
          <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
        </select>
      </div>
      <div class="vn-filter-group">
        <label>Línea</label>
        <select v-model="lineaID" class="vn-select" :disabled="!plantaID">
          <option value="">Seleccionar...</option>
          <option v-for="l in lineasFiltradas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
        </select>
      </div>
    </div>

    <!-- Tabs -->
    <div class="vn-tabs">
      <button
        class="vn-tab"
        :class="{ 'vn-tab--active': activeTab === 'config' }"
        @click="activeTab = 'config'"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
          <circle cx="12" cy="12" r="3"/>
        </svg>
        Configuración
      </button>
      <button
        class="vn-tab"
        :class="{ 'vn-tab--active': activeTab === 'historial' }"
        @click="activeTab = 'historial'"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        Historial
      </button>
      <button
        class="vn-tab"
        :class="{ 'vn-tab--active': activeTab === 'motivos' }"
        @click="activeTab = 'motivos'"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
        </svg>
        Motivos
      </button>
    </div>

    <!-- ─── TAB: CONFIGURACIÓN ─────────────────────────────────────────────── -->
    <div v-show="activeTab === 'config'" class="vn-card">

      <div class="vn-bar" v-if="lineaID">
        <div class="vn-search-wrap">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/><path stroke-linecap="round" d="M21 21l-4.35-4.35"/>
          </svg>
          <input v-model="busqueda" class="vn-search" placeholder="Buscar SKU o descripción..." />
        </div>
        <div class="vn-bar-right">
          <select
            v-model="motivo"
            class="vn-motivo-input"
          >
            <option value="">Motivo del cambio (opcional)</option>
            <option v-for="m in motivosList.filter(x => x.activo)" :key="m.id" :value="m.texto">{{ m.texto }}</option>
          </select>
          <span v-if="guardadoOk" class="vn-ok-badge">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
            Guardado
          </span>
          <span v-if="errorMsg" class="vn-err-badge">{{ errorMsg }}</span>
          <button class="vn-btn-save" :disabled="guardando || !hayCambios" @click="guardarTodo">
            <svg v-if="!guardando" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
            </svg>
            {{ guardando ? 'Guardando...' : 'Guardar cambios' }}
          </button>
        </div>
      </div>

      <div v-if="!lineaID" class="vn-empty">Selecciona empresa, planta y línea para ver los productos.</div>
      <div v-else-if="loading" class="vn-empty">Cargando...</div>
      <div v-else-if="filasFiltradas.length === 0" class="vn-empty">
        No hay productos asignados a esta línea.<br>
        <small>Agrégalos desde <strong>Administración → Productos</strong>.</small>
      </div>

      <table v-else class="vn-table">
        <thead>
          <tr>
            <th style="width:48px">N°</th>
            <th>Código SKU</th>
            <th>Descripción</th>
            <th>Velocidad Nominal (Unidades/Segundo)</th>
            <th>Factor de conversión</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(fila, idx) in filasFiltradas"
            :key="fila.producto_id"
            :class="{ 'vn-row-changed': editadas[fila.producto_id]?.velocidad_us != fila.velocidad_us || editadas[fila.producto_id]?.factor_conv != fila.factor_conv }"
          >
            <td class="vn-td-num">{{ idx + 1 }}</td>
            <td class="vn-td-sku">{{ fila.sku }}</td>
            <td>{{ fila.descripcion }}</td>
            <td>
              <input
                type="number" min="0" step="0.00001" class="vn-input-num"
                v-model.number="editadas[fila.producto_id].velocidad_us"
                @keyup.enter="guardarTodo"
              />
            </td>
            <td>
              <input
                type="number" min="1" step="1" class="vn-input-num vn-input-narrow"
                v-model.number="editadas[fila.producto_id].factor_conv"
                @keyup.enter="guardarTodo"
              />
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="filasFiltradas.length > 0" class="vn-footer">
        {{ filasFiltradas.length }} producto{{ filasFiltradas.length !== 1 ? 's' : '' }}
        <span
          v-if="filasFiltradas.filter(f => (editadas[f.producto_id]?.velocidad_us ?? 0) === 0).length > 0"
          class="vn-warn-badge"
        >
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          </svg>
          {{ filasFiltradas.filter(f => (editadas[f.producto_id]?.velocidad_us ?? 0) === 0).length }} sin velocidad configurada
        </span>
      </div>
    </div>

    <!-- ─── TAB: HISTORIAL ────────────────────────────────────────────────── -->
    <div v-show="activeTab === 'historial'" class="vn-card">

      <div class="vn-bar" v-if="lineaID">
        <span class="vn-bar-title">Registro de cambios de velocidad nominal</span>
        <div class="vn-bar-right">
          <select v-model.number="logLimit" class="vn-limit-select" @change="cargarLog">
            <option :value="50">Últimos 50</option>
            <option :value="100">Últimos 100</option>
            <option :value="250">Últimos 250</option>
            <option :value="500">Últimos 500</option>
          </select>
          <button class="vn-btn-reload" @click="cargarLog" :disabled="logLoading">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            Actualizar
          </button>
        </div>
      </div>

      <div v-if="!lineaID" class="vn-empty">Selecciona empresa, planta y línea para ver el historial.</div>
      <div v-else-if="logLoading" class="vn-empty">Cargando historial...</div>
      <div v-else-if="logError" class="vn-empty vn-empty--error">{{ logError }}</div>
      <div v-else-if="logRows.length === 0" class="vn-empty">No hay registros de cambios para esta línea.</div>

      <table v-else class="vn-table vn-log-table">
        <thead>
          <tr>
            <th>Fecha / Hora</th>
            <th>Producto (SKU)</th>
            <th>Velocidad anterior</th>
            <th>Velocidad nueva</th>
            <th>Factor anterior</th>
            <th>Factor nuevo</th>
            <th>Origen</th>
            <th>Motivo</th>
            <th>Usuario</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in logRows" :key="row.id">
            <td class="vn-td-fecha">{{ formatFecha(row.cambiado_en) }}</td>
            <td class="vn-td-sku">{{ row.sku || '—' }}</td>
            <td class="vn-td-num">
              <span v-if="row.velocidad_us_anterior != null">{{ row.velocidad_us_anterior }}</span>
              <span v-else class="vn-muted">nuevo</span>
            </td>
            <td class="vn-td-num vn-td-nueva">{{ row.velocidad_us_nueva }}</td>
            <td class="vn-td-num">
              <span v-if="row.factor_conv_anterior != null">{{ row.factor_conv_anterior }}</span>
              <span v-else class="vn-muted">nuevo</span>
            </td>
            <td class="vn-td-num vn-td-nueva">{{ row.factor_conv_nueva }}</td>
            <td>
              <span :class="['vn-origen-badge', `vn-origen-${row.origen}`]">
                {{ row.origen }}
              </span>
            </td>
            <td class="vn-td-motivo">{{ row.motivo || '—' }}</td>
            <td class="vn-td-usuario">{{ row.usuario || '—' }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="logRows.length > 0" class="vn-footer">
        {{ logRows.length }} registro{{ logRows.length !== 1 ? 's' : '' }}
      </div>
    </div>

    <!-- ─── TAB: MOTIVOS ──────────────────────────────────────────────────── -->
    <div v-show="activeTab === 'motivos'" class="vn-card">

      <div class="vn-bar" v-if="lineaID">
        <span class="vn-bar-title">Motivos predefinidos para cambios de velocidad</span>
        <div class="vn-bar-right">
          <button class="vn-btn-reload" @click="cargarMotivos" :disabled="motivosLoading">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            Actualizar
          </button>
        </div>
      </div>

      <div v-if="!lineaID" class="vn-empty">Selecciona empresa, planta y línea para gestionar motivos.</div>
      <div v-else-if="motivosLoading" class="vn-empty">Cargando motivos...</div>
      <div v-else>
        <div v-if="motivosError" class="vn-err-badge" style="margin:12px 16px">{{ motivosError }}</div>

        <!-- Lista de motivos -->
        <table class="vn-table">
          <thead>
            <tr>
              <th style="width:40px">N°</th>
              <th>Texto del motivo</th>
              <th style="width:80px;text-align:center">Estado</th>
              <th style="width:120px;text-align:center">Acciones</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(m, idx) in motivosList" :key="m.id" :style="!m.activo ? 'opacity:.5' : ''">
              <td class="vn-td-num">{{ idx + 1 }}</td>
              <td>
                <template v-if="editandoID === m.id">
                  <input
                    v-model="editandoTexto"
                    class="vn-input-motivo-edit"
                    @keyup.enter="guardarEdicion(m)"
                    @keyup.escape="editandoID = null"
                    autofocus
                  />
                </template>
                <template v-else>{{ m.texto }}</template>
              </td>
              <td style="text-align:center">
                <span v-if="m.activo" class="vn-ok-badge" style="font-size:11px">Activo</span>
                <span v-else class="vn-err-badge" style="font-size:11px">Inactivo</span>
              </td>
              <td style="text-align:center">
                <div style="display:flex;gap:6px;justify-content:center;flex-wrap:wrap">
                  <template v-if="editandoID === m.id">
                    <button class="vn-btn-sm vn-btn-sm--ok" :disabled="guardandoMot" @click="guardarEdicion(m)">Guardar</button>
                    <button class="vn-btn-sm" @click="editandoID = null">Cancelar</button>
                  </template>
                  <template v-else>
                    <button class="vn-btn-sm" @click="iniciarEdicion(m)">Editar</button>
                    <button
                      class="vn-btn-sm"
                      :class="m.activo ? 'vn-btn-sm--del' : 'vn-btn-sm--ok'"
                      @click="toggleActivo(m)"
                    >{{ m.activo ? 'Desactivar' : 'Activar' }}</button>
                  </template>
                </div>
              </td>
            </tr>
            <tr v-if="motivosList.length === 0">
              <td colspan="4" class="vn-empty" style="padding:20px">No hay motivos configurados aún.</td>
            </tr>
          </tbody>
        </table>

        <!-- Agregar nuevo motivo -->
        <div class="vn-motivos-add">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/>
          </svg>
          <input
            v-model="nuevoMotivo"
            class="vn-input-motivo-new"
            placeholder="Escribe un nuevo motivo y presiona Agregar..."
            maxlength="255"
            @keyup.enter="agregarMotivo"
          />
          <button class="vn-btn-save" :disabled="!nuevoMotivo.trim() || guardandoMot" @click="agregarMotivo">
            {{ guardandoMot ? 'Agregando...' : 'Agregar' }}
          </button>
        </div>

        <div class="vn-footer">
          {{ motivosList.filter(m => m.activo).length }} activos · {{ motivosList.length }} total
          <span class="vn-muted" style="margin-left:8px">Los cambios se sincronizan automáticamente con el equipo Jetson.</span>
        </div>
      </div>
    </div>

  </div>
</template>

<style scoped>
.vn-wrap   { padding: 0; font-family: 'Inter', sans-serif; font-size: 13px; }
.vn-header { background: #1a2e6c; padding: 10px 20px; }
.vn-title  { color: #fff; font-size: 13px; font-weight: 700; letter-spacing: .05em; margin: 0; }

.vn-filters {
  display: flex; gap: 16px; flex-wrap: wrap;
  padding: 16px 20px; background: #fff; border-bottom: 1px solid #e5e7eb;
}
.vn-filter-group { display: flex; flex-direction: column; gap: 4px; min-width: 160px; }
.vn-filter-group label { font-size: 11px; font-weight: 600; color: #374151; text-transform: uppercase; }
.vn-select {
  border: 1px solid #d1d5db; border-radius: 6px; padding: 6px 10px;
  font-size: 13px; background: #fff; color: #111827; cursor: pointer;
}
.vn-select:disabled { background: #f9fafb; color: #9ca3af; cursor: default; }

/* Tabs */
.vn-tabs {
  display: flex; gap: 0; padding: 0 16px; background: #fff;
  border-bottom: 2px solid #e5e7eb; margin-top: 8px;
}
.vn-tab {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 18px; border: none; background: transparent;
  font-size: 12px; font-weight: 600; color: #6b7280; cursor: pointer;
  border-bottom: 2px solid transparent; margin-bottom: -2px;
  transition: color .15s, border-color .15s;
}
.vn-tab:hover { color: #374151; }
.vn-tab--active { color: #1a2e6c; border-bottom-color: #1a2e6c; }

.vn-card  { background: #fff; margin: 12px 16px 16px; border-radius: 8px; border: 1px solid #e5e7eb; overflow: hidden; }

.vn-bar   { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 16px; background: #1a2e6c; }
.vn-bar-right { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.vn-bar-title { color: #fff; font-size: 12px; font-weight: 600; letter-spacing: .03em; }

.vn-search-wrap { display: flex; align-items: center; gap: 6px; background: rgba(255,255,255,.12); border-radius: 6px; padding: 5px 10px; }
.vn-search-wrap svg { color: #fff; opacity: .7; flex-shrink: 0; }
.vn-search { background: transparent; border: none; outline: none; color: #fff; font-size: 13px; width: 200px; }
.vn-search::placeholder { color: rgba(255,255,255,.55); }

.vn-motivo-input {
  background: rgba(255,255,255,.12); border: 1px solid rgba(255,255,255,.25);
  border-radius: 6px; padding: 5px 10px; color: #fff; font-size: 12px;
  width: 220px; outline: none; cursor: pointer;
}
.vn-motivo-input option { background: #1a2e6c; color: #fff; }
.vn-motivo-input:focus { border-color: rgba(255,255,255,.6); }

.vn-limit-select {
  background: rgba(255,255,255,.12); border: 1px solid rgba(255,255,255,.25);
  border-radius: 6px; padding: 5px 10px; color: #fff; font-size: 12px; cursor: pointer;
}
.vn-limit-select option { background: #1a2e6c; }

.vn-btn-save {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 16px; border-radius: 6px; border: none; cursor: pointer;
  font-size: 12px; font-weight: 600; background: #16a34a; color: #fff; transition: opacity .15s;
}
.vn-btn-save:disabled { opacity: .45; cursor: default; }
.vn-btn-save:not(:disabled):hover { background: #15803d; }

.vn-btn-reload {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 14px; border-radius: 6px; border: 1px solid rgba(255,255,255,.3);
  background: transparent; color: #fff; font-size: 12px; font-weight: 600; cursor: pointer;
  transition: background .15s;
}
.vn-btn-reload:hover { background: rgba(255,255,255,.12); }
.vn-btn-reload:disabled { opacity: .45; cursor: default; }

.vn-ok-badge  { display: flex; align-items: center; gap: 4px; background: #dcfce7; color: #166534; padding: 3px 10px; border-radius: 12px; font-size: 11px; font-weight: 600; }
.vn-err-badge { background: #fee2e2; color: #991b1b; padding: 3px 10px; border-radius: 12px; font-size: 11px; }
.vn-warn-badge {
  display: flex; align-items: center; gap: 4px;
  background: #fef9c3; color: #854d0e; padding: 2px 8px; border-radius: 10px;
  font-size: 11px; font-weight: 600;
}

.vn-empty { padding: 40px; text-align: center; color: #6b7280; line-height: 1.8; }
.vn-empty--error { color: #dc2626; }

.vn-table { width: 100%; border-collapse: collapse; }
.vn-table thead th {
  background: #f8fafc; padding: 10px 14px; text-align: left;
  font-size: 11px; font-weight: 700; color: #374151;
  border-bottom: 2px solid #e5e7eb; white-space: nowrap;
}
.vn-table tbody tr { border-bottom: 1px solid #f1f5f9; transition: background .1s; }
.vn-table tbody tr:hover { background: #f8fafc; }
.vn-row-changed { background: #fffbeb !important; }

.vn-td-num    { color: #374151; font-size: 12px; padding: 8px 14px; text-align: right; white-space: nowrap; }
.vn-td-nueva  { font-weight: 600; color: #1a2e6c; }
.vn-td-sku    { font-weight: 600; padding: 8px 14px; }
.vn-td-fecha  { padding: 8px 14px; color: #6b7280; font-size: 12px; white-space: nowrap; }
.vn-td-motivo { padding: 8px 14px; color: #374151; max-width: 200px; }
.vn-td-usuario{ padding: 8px 14px; color: #374151; font-size: 12px; }
.vn-table td  { padding: 8px 14px; color: #374151; }
.vn-muted     { color: #9ca3af; font-style: italic; }

.vn-input-num {
  width: 120px; padding: 5px 8px; border: 1px solid #d1d5db;
  border-radius: 5px; font-size: 13px; text-align: right; transition: border-color .15s;
}
.vn-input-num:focus { outline: none; border-color: #3b82f6; box-shadow: 0 0 0 2px rgba(59,130,246,.15); }
.vn-input-narrow { width: 72px; }

.vn-origen-badge {
  display: inline-block; padding: 2px 8px; border-radius: 10px;
  font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: .03em;
}
.vn-origen-cloud { background: #dbeafe; color: #1d4ed8; }
.vn-origen-edge  { background: #d1fae5; color: #065f46; }

.vn-footer {
  padding: 8px 16px; background: #f8fafc; border-top: 1px solid #e5e7eb;
  color: #6b7280; font-size: 12px; display: flex; align-items: center; gap: 10px;
}

.vn-motivos-add {
  display: flex; align-items: center; gap: 10px;
  padding: 12px 16px; background: #f0f9ff; border-top: 1px solid #e0f2fe;
}
.vn-input-motivo-new {
  flex: 1; padding: 6px 10px; border: 1px solid #bae6fd; border-radius: 6px;
  font-size: 13px; outline: none; background: #fff;
}
.vn-input-motivo-new:focus { border-color: #3b82f6; box-shadow: 0 0 0 2px rgba(59,130,246,.15); }
.vn-input-motivo-edit {
  width: 100%; padding: 4px 8px; border: 1px solid #3b82f6; border-radius: 5px;
  font-size: 13px; outline: none;
}
.vn-btn-sm {
  padding: 3px 10px; font-size: 11px; font-weight: 600; border-radius: 5px;
  border: 1px solid #d1d5db; background: #fff; cursor: pointer; white-space: nowrap;
  transition: background .1s;
}
.vn-btn-sm:hover { background: #f3f4f6; }
.vn-btn-sm:disabled { opacity: .45; cursor: default; }
.vn-btn-sm--ok  { background: #dcfce7; border-color: #86efac; color: #166534; }
.vn-btn-sm--ok:hover { background: #bbf7d0; }
.vn-btn-sm--del { background: #fee2e2; border-color: #fca5a5; color: #991b1b; }
.vn-btn-sm--del:hover { background: #fecaca; }
</style>
