<script setup>
import { ref, computed, onMounted } from 'vue'
import { integrationService } from '@/api/services/integration.service'
import companyService from '@/api/services/company.service'
import { useAuthStore } from '@/stores/auth'

// ─── doc tabs ──────────────────────────────────────────────────────────────
const DOC_TABS = [
  { id: 'general',    label: 'General' },
  { id: 'auth',       label: 'Autenticacion' },
  { id: 'endpoints',  label: 'Endpoints' },
  { id: 'responses',  label: 'Respuestas' },
  { id: 'examples',   label: 'Ejemplos' },
  { id: 'errors',     label: 'Errores' },
]
const docTab = ref('general')

const authStore = useAuthStore()
const isAdmin = computed(() => ['ADMIN', 'superadmin', 'ADMIN_PLANTA'].includes(authStore.user?.role))

const keys = ref([])
const empresas = ref([])
const loading = ref(false)

const empresaId = ref('')
const filtroNombre = ref('')
const filtroEstado = ref('todas')

const filteredKeys = computed(() => {
  let list = keys.value
  if (filtroEstado.value === 'activas')   list = list.filter(k => k.activo)
  if (filtroEstado.value === 'revocadas') list = list.filter(k => !k.activo)
  const q = filtroNombre.value.trim().toLowerCase()
  if (q) list = list.filter(k => k.nombre?.toLowerCase().includes(q) || k.key_prefix?.toLowerCase().includes(q))
  return list
})

const showCreate = ref(false)
const createNombre = ref('')
const createEmpresaId = ref('')
const creating = ref(false)

const showKeyResult = ref(false)
const newKeyPlaintext = ref('')
const copied = ref(false)

async function cargarEmpresas() {
  const res = await companyService.getAll()
  empresas.value = res?.data ?? res ?? []
}

async function cargarKeys() {
  loading.value = true
  try {
    const eid = isAdmin.value ? empresaId.value : authStore.user?.empresa_id
    // no-admin sin empresa asignada → nada que mostrar
    if (!isAdmin.value && !eid) { keys.value = []; return }
    const res = await integrationService.listKeys(eid)
    keys.value = Array.isArray(res) ? res : (res?.data ?? [])
  } catch { keys.value = [] }
  finally { loading.value = false }
}

function resolveEmpresaId() {
  if (isAdmin.value) {
    const raw = createEmpresaId.value || empresaId.value
    const n = Number(raw)
    return isNaN(n) || n === 0 ? null : n
  }
  return Number(authStore.user?.empresa_id ?? 0) || null
}

async function crearKey() {
  if (!createNombre.value.trim()) return
  const empresa_id = resolveEmpresaId()
  if (!empresa_id) {
    alert('Selecciona una empresa antes de crear la clave.')
    return
  }
  creating.value = true
  try {
    const res = await integrationService.createKey({ nombre: createNombre.value.trim(), empresa_id })
    const data = res?.data ?? res
    newKeyPlaintext.value = data.key || ''
    showCreate.value = false
    showKeyResult.value = true
    createNombre.value = ''
    empresaId.value = String(data.empresa_id ?? empresa_id)
    await cargarKeys()
  } catch (e) {
    alert(e?.response?.data?.error || e.message || 'Error creando clave')
  } finally { creating.value = false }
}

async function revocarKey(id) {
  if (!confirm('Revocar esta clave API? Los sistemas que la usen perderan acceso.')) return
  try {
    const eid = isAdmin.value ? empresaId.value : authStore.user?.empresa_id
    await integrationService.revokeKey(id, eid)
    await cargarKeys()
  } catch (e) {
    alert(e?.response?.data?.error || e.message || 'Error revocando')
  }
}

function copiarKey() {
  navigator.clipboard.writeText(newKeyPlaintext.value)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

function formatFecha(iso) {
  if (!iso) return '-'
  return new Date(iso).toLocaleDateString('es-PE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(async () => {
  await cargarEmpresas()
  const eid = authStore.user?.empresa_id
  if (eid) {
    empresaId.value = String(eid)
    createEmpresaId.value = String(eid)
  } else if (isAdmin.value && empresas.value.length > 0) {
    empresaId.value = String(empresas.value[0].id)
    createEmpresaId.value = String(empresas.value[0].id)
  }
  await cargarKeys()
})
</script>

<template>
  <div class="ig-wrap">
    <!-- Header con gradiente sutil -->
    <div class="ig-hero">
      <div class="ig-hero-left">
        <div class="ig-hero-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#6366f1" stroke-width="2" stroke-linecap="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
        </div>
        <div>
          <h1 class="ig-title">API de Integracion</h1>
          <p class="ig-subtitle">Genera claves para que sistemas externos (BI, ERP, MES) consulten datos OEE, snapshots y paradas via REST.</p>
        </div>
      </div>
      <button class="ig-btn-create" @click="createEmpresaId = empresaId; showCreate = true">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        Nueva API Key
      </button>
    </div>

    <!-- Barra de filtros -->
    <div class="ig-filter-bar">
      <!-- Empresa (solo ADMIN) -->
      <template v-if="isAdmin">
        <label class="ig-filter-label">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 21h18M5 21V9l5 4V9l5 4V7a2 2 0 012-2h4v16"/></svg>
          Empresa
        </label>
        <select v-model="empresaId" class="ig-filter-select" style="min-width:180px" @change="cargarKeys">
          <option value="">Todas las empresas</option>
          <option v-for="e in empresas" :key="e.id" :value="String(e.id)">{{ e.nombre }}</option>
        </select>
        <div class="ig-filter-sep"></div>
      </template>
      <!-- Búsqueda por nombre -->
      <div class="ig-filter-search">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#94a3b8" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input v-model="filtroNombre" class="ig-filter-input" placeholder="Buscar por nombre o prefijo..." />
      </div>
      <!-- Toggle estado -->
      <div class="ig-filter-sep"></div>
      <div class="ig-estado-tabs">
        <button class="ig-estado-tab" :class="{ active: filtroEstado === 'todas' }"    @click="filtroEstado = 'todas'">Todas <span class="ig-tab-count">{{ keys.length }}</span></button>
        <button class="ig-estado-tab" :class="{ active: filtroEstado === 'activas' }"  @click="filtroEstado = 'activas'">Activas <span class="ig-tab-count ig-tab-ok">{{ keys.filter(k=>k.activo).length }}</span></button>
        <button class="ig-estado-tab" :class="{ active: filtroEstado === 'revocadas' }" @click="filtroEstado = 'revocadas'">Revocadas <span class="ig-tab-count ig-tab-off">{{ keys.filter(k=>!k.activo).length }}</span></button>
      </div>
    </div>

    <!-- Tarjetas resumen -->
    <div class="ig-stats">
      <div class="ig-stat-card">
        <div class="ig-stat-icon ig-stat-icon-total">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
        </div>
        <div class="ig-stat-info">
          <span class="ig-stat-num">{{ keys.length }}</span>
          <span class="ig-stat-label">Total claves</span>
        </div>
      </div>
      <div class="ig-stat-card">
        <div class="ig-stat-icon ig-stat-icon-active">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        </div>
        <div class="ig-stat-info">
          <span class="ig-stat-num">{{ keys.filter(k => k.activo).length }}</span>
          <span class="ig-stat-label">Activas</span>
        </div>
      </div>
      <div class="ig-stat-card">
        <div class="ig-stat-icon ig-stat-icon-revoked">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        </div>
        <div class="ig-stat-info">
          <span class="ig-stat-num">{{ keys.filter(k => !k.activo).length }}</span>
          <span class="ig-stat-label">Revocadas</span>
        </div>
      </div>
    </div>

    <!-- Tabla de claves -->
    <div class="ig-card">
      <div class="ig-card-header">
        <h2 class="ig-card-title">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
          Claves registradas
        </h2>
      </div>
      <div class="ig-card-body" v-if="filteredKeys.length || loading">
        <table class="ig-table">
          <thead>
            <tr>
              <th>Nombre</th>
              <th>Prefijo</th>
              <th v-if="isAdmin">Empresa</th>
              <th>Scopes</th>
              <th>Estado</th>
              <th>Creada</th>
              <th>Ultimo uso</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="k in filteredKeys" :key="k.id" :class="{ 'ig-row-dim': !k.activo }">
              <td>
                <div class="ig-name-cell">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#6366f1" stroke-width="2" stroke-linecap="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 11-7.778 7.778 5.5 5.5 0 017.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
                  <span class="ig-name">{{ k.nombre }}</span>
                </div>
              </td>
              <td><code class="ig-prefix">{{ k.key_prefix }}...</code></td>
              <td v-if="isAdmin" class="ig-date">{{ empresas.find(e=>String(e.id)===String(k.empresa_id))?.nombre ?? k.empresa_id ?? '-' }}</td>
              <td>
                <div class="ig-scope-list">
                  <span class="ig-scope" v-for="s in (k.scopes || [])" :key="s">{{ s }}</span>
                </div>
              </td>
              <td>
                <span class="ig-badge" :class="k.activo ? 'ig-badge-ok' : 'ig-badge-off'">
                  <span class="ig-badge-dot"></span>
                  {{ k.activo ? 'Activa' : 'Revocada' }}
                </span>
              </td>
              <td class="ig-date">{{ formatFecha(k.creado_en) }}</td>
              <td class="ig-date">{{ formatFecha(k.ultimo_uso) }}</td>
              <td>
                <button v-if="k.activo" class="ig-btn-revoke" @click="revocarKey(k.id)" title="Revocar clave">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                </button>
                <span v-else class="ig-text-muted">--</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else-if="!loading && !filteredKeys.length && keys.length" class="ig-empty">
        <div class="ig-empty-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#cbd5e1" stroke-width="1.5" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        </div>
        <p class="ig-empty-title">Sin resultados</p>
        <p class="ig-empty-desc">Ajusta los filtros para encontrar la clave.</p>
        <button class="ig-btn ig-btn-ghost" style="margin-top:4px" @click="filtroNombre='';filtroEstado='todas'">Limpiar filtros</button>
      </div>
      <div v-else-if="!loading" class="ig-empty">
        <div class="ig-empty-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#cbd5e1" stroke-width="1.5" stroke-linecap="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
        </div>
        <p class="ig-empty-title">Sin claves API</p>
        <p class="ig-empty-desc">Crea tu primera clave para habilitar el acceso de sistemas externos.</p>
        <button class="ig-btn-create ig-btn-sm" @click="showCreate = true">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Crear clave
        </button>
      </div>
      <div v-if="loading" class="ig-loading">
        <div class="ig-spinner"></div>
        Cargando claves...
      </div>
    </div>

    <!-- Documentacion de la API -->
    <div class="ig-card">
      <div class="ig-card-header">
        <h2 class="ig-card-title">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z"/><path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z"/></svg>
          Referencia de la API
        </h2>
        <span class="ig-card-tag">v1</span>
      </div>

      <!-- Tab nav -->
      <div class="ig-doc-tabs">
        <button
          v-for="t in DOC_TABS" :key="t.id"
          class="ig-doc-tab" :class="{ 'ig-doc-tab-active': docTab === t.id }"
          @click="docTab = t.id"
        >{{ t.label }}</button>
      </div>

      <!-- ── GENERAL ─────────────────────────────────────────────────────── -->
      <div v-if="docTab === 'general'" class="ig-docs-body">
        <p class="ig-docs-intro">
          La API REST de Mentor Monitor permite a sistemas externos (BI, ERP, MES, scripts) consultar datos OEE,
          snapshots historicos y paradas en tiempo real. Cada empresa tiene sus propias claves API con scopes especificos.
        </p>

        <div class="ig-info-grid">
          <div class="ig-info-card">
            <div class="ig-info-icon ig-info-icon-blue">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
            </div>
            <div>
              <div class="ig-info-title">Base URL</div>
              <code class="ig-inline-code">https://api.mentormonitor-ai.com</code>
            </div>
          </div>
          <div class="ig-info-card">
            <div class="ig-info-icon ig-info-icon-green">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
            </div>
            <div>
              <div class="ig-info-title">Formato</div>
              <span class="ig-info-val">JSON, UTF-8, HTTPS obligatorio</span>
            </div>
          </div>
          <div class="ig-info-card">
            <div class="ig-info-icon ig-info-icon-purple">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
            </div>
            <div>
              <div class="ig-info-title">Autenticacion</div>
              <span class="ig-info-val">Header <code class="ig-inline-code">X-API-Key</code></span>
            </div>
          </div>
          <div class="ig-info-card">
            <div class="ig-info-icon ig-info-icon-amber">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            </div>
            <div>
              <div class="ig-info-title">Rate limit</div>
              <span class="ig-info-val">1 000 req / 10 min por clave</span>
            </div>
          </div>
        </div>

        <div class="ig-section-title">Scopes disponibles</div>
        <table class="ig-ref-table">
          <thead>
            <tr><th>Scope</th><th>Acceso permitido</th></tr>
          </thead>
          <tbody>
            <tr><td><span class="ig-scope">oee:read</span></td><td>Ultimo snapshot OEE por linea</td></tr>
            <tr><td><span class="ig-scope">snapshots:read</span></td><td>Historico de snapshots OEE</td></tr>
            <tr><td><span class="ig-scope">paradas:read</span></td><td>Historial de paradas con filtro temporal</td></tr>
            <tr><td><span class="ig-scope" style="background:#d1fae5;color:#065f46">energy:read</span></td><td>Snapshots de energia electrica (MC60) — corrientes, voltajes, potencia, energía acumulada, THD. Paginado para Power BI.</td></tr>
          </tbody>
        </table>
      </div>

      <!-- ── AUTENTICACION ──────────────────────────────────────────────── -->
      <div v-if="docTab === 'auth'" class="ig-docs-body">
        <p class="ig-docs-intro">
          Cada peticion debe incluir la clave API en el header <code class="ig-inline-code">X-API-Key</code>.
          Los datos se filtran automaticamente por la empresa asociada a la clave — nunca se mezclan datos entre empresas.
        </p>

        <div class="ig-steps">
          <div class="ig-step">
            <div class="ig-step-num">1</div>
            <div>
              <div class="ig-step-title">Genera una clave</div>
              <p class="ig-step-desc">Haz clic en "Nueva API Key", asigna un nombre descriptivo (ej: <em>PowerBI Lima</em>) y selecciona los scopes necesarios.</p>
            </div>
          </div>
          <div class="ig-step">
            <div class="ig-step-num">2</div>
            <div>
              <div class="ig-step-title">Guarda la clave de forma segura</div>
              <p class="ig-step-desc">La clave completa se muestra <strong>una unica vez</strong>. Almacenala en un gestor de secretos o en las variables de entorno de tu sistema.</p>
            </div>
          </div>
          <div class="ig-step">
            <div class="ig-step-num">3</div>
            <div>
              <div class="ig-step-title">Incluye el header en cada peticion</div>
              <div class="ig-example-box" style="margin-top:8px">
                <div class="ig-example-header">
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
                  HTTP Header
                </div>
                <pre class="ig-code-block">X-API-Key: mk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx</pre>
              </div>
            </div>
          </div>
        </div>

        <div class="ig-warn-box">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#d97706" stroke-width="2" stroke-linecap="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          <div>
            <strong>Seguridad:</strong> Nunca expongas la clave en codigo fuente publico, repositorios git o logs.
            Usa variables de entorno o un gestor de secretos (HashiCorp Vault, AWS Secrets Manager, etc.).
            Si se ve comprometida, revocala inmediatamente desde esta pantalla.
          </div>
        </div>
      </div>

      <!-- ── ENDPOINTS ──────────────────────────────────────────────────── -->
      <div v-if="docTab === 'endpoints'" class="ig-docs-body">
        <!-- OEE Latest -->
        <div class="ig-ep-block">
          <div class="ig-ep-row">
            <span class="ig-method">GET</span>
            <code class="ig-ep-path">/api/v1/integration/oee/latest</code>
            <span class="ig-ep-scope">oee:read</span>
          </div>
          <p class="ig-ep-desc">Retorna el <strong>ultimo snapshot OEE</strong> calculado para la linea especificada. Ideal para dashboards en tiempo real.</p>
          <div class="ig-param-table-wrap">
            <table class="ig-ref-table">
              <thead><tr><th>Parametro</th><th>Tipo</th><th>Requerido</th><th>Descripcion</th></tr></thead>
              <tbody>
                <tr><td><code>linea_id</code></td><td>integer</td><td><span class="ig-badge ig-badge-ok" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>si</span></td><td>ID de la linea de produccion</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- OEE Snapshots -->
        <div class="ig-ep-block">
          <div class="ig-ep-row">
            <span class="ig-method">GET</span>
            <code class="ig-ep-path">/api/v1/integration/oee/snapshots</code>
            <span class="ig-ep-scope">snapshots:read</span>
          </div>
          <p class="ig-ep-desc">Retorna un <strong>historico paginado</strong> de snapshots OEE. Maximo 1 000 registros por peticion.</p>
          <div class="ig-param-table-wrap">
            <table class="ig-ref-table">
              <thead><tr><th>Parametro</th><th>Tipo</th><th>Requerido</th><th>Descripcion</th></tr></thead>
              <tbody>
                <tr><td><code>linea_id</code></td><td>integer</td><td><span class="ig-badge ig-badge-ok" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>si</span></td><td>ID de la linea</td></tr>
                <tr><td><code>limite</code></td><td>integer</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Cantidad maxima (default: 100, max: 1000)</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Energy snapshots -->
        <div class="ig-ep-block">
          <div class="ig-ep-row">
            <span class="ig-method">GET</span>
            <code class="ig-ep-path">/api/v1/integration/energy/snapshots</code>
            <span class="ig-ep-scope">energy:read</span>
          </div>
          <p class="ig-ep-desc">Retorna snapshots historicos del medidor de energia MC60: voltaje, corriente, potencia activa/reactiva, factor de potencia, frecuencia y THD.</p>
          <div class="ig-param-table-wrap">
            <table class="ig-ref-table">
              <thead><tr><th>Parametro</th><th>Tipo</th><th>Requerido</th><th>Descripcion</th></tr></thead>
              <tbody>
                <tr><td><code>planta_id</code></td><td>integer</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Filtra por planta especifica</td></tr>
                <tr><td><code>from</code></td><td>ISO 8601</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Inicio del rango (ej: <code>2024-06-01T00:00:00Z</code>)</td></tr>
                <tr><td><code>to</code></td><td>ISO 8601</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Fin del rango</td></tr>
                <tr><td><code>limit</code></td><td>integer</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Maximo de registros (default: 100, max: 1000)</td></tr>
                <tr><td><code>offset</code></td><td>integer</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Paginacion (default: 0)</td></tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Paradas -->
        <div class="ig-ep-block">
          <div class="ig-ep-row">
            <span class="ig-method">GET</span>
            <code class="ig-ep-path">/api/v1/integration/paradas</code>
            <span class="ig-ep-scope">paradas:read</span>
          </div>
          <p class="ig-ep-desc">Retorna las <strong>paradas de linea</strong> en el rango temporal indicado, incluyendo tipo, duracion y categoria.</p>
          <div class="ig-param-table-wrap">
            <table class="ig-ref-table">
              <thead><tr><th>Parametro</th><th>Tipo</th><th>Requerido</th><th>Descripcion</th></tr></thead>
              <tbody>
                <tr><td><code>linea_id</code></td><td>integer</td><td><span class="ig-badge ig-badge-ok" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>si</span></td><td>ID de la linea</td></tr>
                <tr><td><code>desde</code></td><td>ISO 8601</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Inicio del rango (ej: <code>2024-01-15T00:00:00Z</code>)</td></tr>
                <tr><td><code>hasta</code></td><td>ISO 8601</td><td><span class="ig-badge ig-badge-off" style="font-size:10px;padding:1px 8px"><span class="ig-badge-dot"></span>no</span></td><td>Fin del rango</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- ── RESPUESTAS ─────────────────────────────────────────────────── -->
      <div v-if="docTab === 'responses'" class="ig-docs-body">
        <p class="ig-docs-intro">Todas las respuestas exitosas retornan HTTP 200 con Content-Type <code class="ig-inline-code">application/json</code>.</p>

        <div class="ig-section-title">OEE Snapshot</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            GET /api/v1/integration/oee/latest → 200
          </div>
<pre class="ig-code-block">{
  "linea_id": 1,
  "linea": "Linea A",
  "oee": 82.5,
  "disponibilidad": 91.2,
  "rendimiento": 94.0,
  "calidad": 96.2,
  "produccion_real": 1240,
  "produccion_ideal": 1320,
  "tiempo_planificado": 28800,
  "tiempo_operativo": 26265,
  "timestamp": "2024-06-10T14:30:00Z"
}</pre>
        </div>

        <div class="ig-section-title">Parada</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            GET /api/v1/integration/paradas → 200 (un elemento del array)
          </div>
<pre class="ig-code-block">{
  "id": 487,
  "linea_id": 1,
  "linea": "Linea A",
  "inicio": "2024-06-10T10:15:00Z",
  "fin": "2024-06-10T10:42:00Z",
  "duracion_min": 27,
  "tipo": "PLANIFICADA",
  "categoria": "Mantenimiento preventivo",
  "observaciones": "Cambio de aceite"
}</pre>
        </div>

        <div class="ig-section-title">Errores</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            Error response
          </div>
<pre class="ig-code-block">{
  "error": "clave API invalida o revocada"
}</pre>
        </div>
      </div>

      <!-- ── EJEMPLOS ───────────────────────────────────────────────────── -->
      <div v-if="docTab === 'examples'" class="ig-docs-body">

        <div class="ig-section-title">cURL</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            OEE en tiempo real
          </div>
<pre class="ig-code-block">curl -s \
  -H "X-API-Key: mk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  "https://api.mentormonitor-ai.com/api/v1/integration/oee/latest?linea_id=1"</pre>
        </div>
        <div class="ig-example-box" style="margin-top:12px">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            Snapshots historicos
          </div>
<pre class="ig-code-block">curl -s \
  -H "X-API-Key: mk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  "https://api.mentormonitor-ai.com/api/v1/integration/oee/snapshots?linea_id=1&limite=50"</pre>
        </div>
        <div class="ig-example-box" style="margin-top:12px">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            Paradas por rango de fechas
          </div>
<pre class="ig-code-block">curl -s \
  -H "X-API-Key: mk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  "https://api.mentormonitor-ai.com/api/v1/integration/paradas?linea_id=1&desde=2024-06-01T00:00:00Z&hasta=2024-06-10T23:59:59Z"</pre>
        </div>

        <div class="ig-section-title" style="margin-top:20px">Python 3</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            requests
          </div>
<pre class="ig-code-block">import requests

API_KEY = "mk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
BASE    = "https://api.mentormonitor-ai.com/api/v1/integration"

headers = {"X-API-Key": API_KEY}

# Ultimo OEE
r = requests.get(f"{BASE}/oee/latest", params={"linea_id": 1}, headers=headers)
r.raise_for_status()
oee = r.json()
print(f"OEE: {oee['oee']}%")

# Snapshots
snapshots = requests.get(
    f"{BASE}/oee/snapshots",
    params={"linea_id": 1, "limite": 200},
    headers=headers
).json()</pre>
        </div>

        <div class="ig-section-title" style="margin-top:20px">JavaScript / Node.js</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            fetch (ESM)
          </div>
<pre class="ig-code-block">const API_KEY = process.env.MENTOR_API_KEY
const BASE    = "https://api.mentormonitor-ai.com/api/v1/integration"

async function getOEE(lineaId) {
  const res = await fetch(`${BASE}/oee/latest?linea_id=${lineaId}`, {
    headers: { "X-API-Key": API_KEY }
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

const oee = await getOEE(1)
console.log("OEE:", oee.oee)</pre>
        </div>

        <div class="ig-section-title" style="margin-top:20px">Power BI (Power Query M)</div>
        <div class="ig-example-box">
          <div class="ig-example-header">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
            Conector Web.Contents
          </div>
<pre class="ig-code-block">let
    ApiKey  = "mk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    Url     = "https://api.mentormonitor-ai.com/api/v1/integration/oee/snapshots?linea_id=1&limite=1000",
    Raw     = Web.Contents(Url, [Headers=[#"X-API-Key"=ApiKey]]),
    Json    = Json.Document(Raw),
    Tabla   = Table.FromList(Json, Splitter.SplitterByNothing()),
    Expandir = Table.ExpandRecordColumn(Tabla, "Column1",
                 {"linea_id","oee","disponibilidad","rendimiento","calidad","timestamp"})
in
    Expandir</pre>
        </div>
      </div>

      <!-- ── ERRORES ────────────────────────────────────────────────────── -->
      <div v-if="docTab === 'errors'" class="ig-docs-body">
        <p class="ig-docs-intro">La API usa codigos HTTP estandar. Todos los errores retornan un JSON con el campo <code class="ig-inline-code">error</code>.</p>

        <table class="ig-ref-table">
          <thead>
            <tr><th>Codigo</th><th>Significado</th><th>Causa comun</th><th>Solucion</th></tr>
          </thead>
          <tbody>
            <tr>
              <td><span class="ig-http-badge ig-http-4xx">400</span></td>
              <td>Bad Request</td>
              <td>Parametro faltante o invalido (ej: <code>linea_id</code> no numerico)</td>
              <td>Revisa los parametros obligatorios del endpoint</td>
            </tr>
            <tr>
              <td><span class="ig-http-badge ig-http-4xx">401</span></td>
              <td>Unauthorized</td>
              <td>Header <code>X-API-Key</code> ausente o vacio</td>
              <td>Incluye el header en cada peticion</td>
            </tr>
            <tr>
              <td><span class="ig-http-badge ig-http-4xx">403</span></td>
              <td>Forbidden</td>
              <td>Clave revocada, expirada o sin el scope necesario</td>
              <td>Genera una nueva clave con los scopes correctos</td>
            </tr>
            <tr>
              <td><span class="ig-http-badge ig-http-4xx">404</span></td>
              <td>Not Found</td>
              <td>La <code>linea_id</code> no existe o no pertenece a tu empresa</td>
              <td>Verifica el ID de la linea en la plataforma</td>
            </tr>
            <tr>
              <td><span class="ig-http-badge ig-http-429">429</span></td>
              <td>Too Many Requests</td>
              <td>Limite de 1 000 req / 10 min superado</td>
              <td>Implementa backoff exponencial o reduce la frecuencia</td>
            </tr>
            <tr>
              <td><span class="ig-http-badge ig-http-5xx">500</span></td>
              <td>Internal Error</td>
              <td>Error inesperado del servidor</td>
              <td>Reintenta pasados 30 segundos; contacta soporte si persiste</td>
            </tr>
          </tbody>
        </table>

        <div class="ig-warn-box" style="margin-top:16px">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#d97706" stroke-width="2" stroke-linecap="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
          <div>
            Para errores <strong>429</strong> aplica backoff exponencial: espera 1s, luego 2s, 4s, 8s...
            No re-intentes en bucle rapido o tu IP podria ser bloqueada temporalmente.
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Crear -->
    <Transition name="fade">
      <div v-if="showCreate" class="ig-overlay" @click.self="showCreate = false">
        <div class="ig-modal" @click.stop>
          <div class="ig-modal-header">
            <div class="ig-modal-icon">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#6366f1" stroke-width="2" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            </div>
            <h2 class="ig-modal-title">Nueva API Key</h2>
            <button class="ig-modal-close" @click="showCreate = false">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
          <div class="ig-modal-body">
            <div class="ig-field">
              <label class="ig-field-label">Nombre descriptivo</label>
              <input v-model="createNombre" class="ig-field-input" placeholder="Ej: PowerBI Produccion Lima" maxlength="100" @keyup.enter="crearKey" />
              <span class="ig-field-hint">Identifica el sistema que usara esta clave</span>
            </div>
            <div v-if="isAdmin" class="ig-field">
              <label class="ig-field-label">Empresa <span style="color:#dc2626">*</span></label>
              <select v-model="createEmpresaId" class="ig-field-select" :style="isAdmin && !createEmpresaId ? 'border-color:#fca5a5' : ''">
                <option value="">— Seleccionar empresa —</option>
                <option v-for="e in empresas" :key="e.id" :value="String(e.id)">{{ e.nombre }}</option>
              </select>
              <span v-if="isAdmin && !createEmpresaId" class="ig-field-hint" style="color:#dc2626">Requerido: selecciona la empresa para esta clave</span>
            </div>
            <div class="ig-modal-notice">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#f59e0b" stroke-width="2" stroke-linecap="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              La clave se mostrara una unica vez. Guardala en un lugar seguro.
            </div>
          </div>
          <div class="ig-modal-footer">
            <button class="ig-btn ig-btn-ghost" @click="showCreate = false">Cancelar</button>
            <button class="ig-btn ig-btn-primary" :disabled="creating || !createNombre.trim() || (isAdmin && !createEmpresaId)" @click="crearKey">
              <svg v-if="!creating" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/></svg>
              {{ creating ? 'Generando...' : 'Generar clave' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Modal Resultado -->
    <Transition name="fade">
      <div v-if="showKeyResult" class="ig-overlay">
        <div class="ig-modal" @click.stop>
          <div class="ig-modal-header">
            <div class="ig-modal-icon ig-modal-icon-success">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#16a34a" stroke-width="2.5" stroke-linecap="round"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            </div>
            <h2 class="ig-modal-title">Clave generada</h2>
          </div>
          <div class="ig-modal-body">
            <div class="ig-key-alert">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#ea580c" stroke-width="2" stroke-linecap="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              Copia esta clave ahora. No podras verla de nuevo.
            </div>
            <div class="ig-key-box">
              <code class="ig-key-value">{{ newKeyPlaintext }}</code>
              <button class="ig-btn-copy" :class="{ 'ig-btn-copied': copied }" @click="copiarKey">
                <svg v-if="!copied" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="20 6 9 17 4 12"/></svg>
                {{ copied ? 'Copiado' : 'Copiar' }}
              </button>
            </div>
          </div>
          <div class="ig-modal-footer">
            <button class="ig-btn ig-btn-primary" @click="showKeyResult = false; newKeyPlaintext = ''">Entendido</button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.ig-wrap{display:flex;flex-direction:column;gap:20px;padding:24px;max-width:1200px}

/* Hero header */
.ig-hero{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:20px 24px;background:linear-gradient(135deg,#f8fafc 0%,#eef2ff 100%);border:1px solid #e2e8f0;border-radius:12px}
.ig-hero-left{display:flex;align-items:center;gap:16px}
.ig-hero-icon{width:48px;height:48px;background:#fff;border:1px solid #e0e7ff;border-radius:12px;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.ig-title{font-size:20px;font-weight:700;color:#0f172a;margin:0;letter-spacing:-.01em}
.ig-subtitle{font-size:13px;color:#64748b;margin:4px 0 0;line-height:1.5}
.ig-btn-create{display:inline-flex;align-items:center;gap:7px;padding:10px 20px;background:#6366f1;color:#fff;border:none;border-radius:8px;font-size:13px;font-weight:600;cursor:pointer;transition:all .15s;white-space:nowrap;box-shadow:0 1px 3px rgba(99,102,241,.3)}
.ig-btn-create:hover{background:#4f46e5;box-shadow:0 4px 12px rgba(99,102,241,.35);transform:translateY(-1px)}
.ig-btn-sm{padding:8px 16px;font-size:12px}

/* Filtro */
.ig-filter-bar{display:flex;align-items:center;gap:10px;padding:10px 16px;background:#fff;border:1px solid #e2e8f0;border-radius:10px;flex-wrap:wrap}
.ig-filter-label{display:flex;align-items:center;gap:6px;font-size:12px;font-weight:600;color:#475569;text-transform:uppercase;letter-spacing:.04em;white-space:nowrap}
.ig-filter-select{padding:7px 12px;border:1px solid #cbd5e1;border-radius:7px;font-size:13px;color:#1e293b;background:#fff}
.ig-filter-select:focus{outline:2px solid #6366f1;border-color:transparent}
.ig-filter-sep{width:1px;height:24px;background:#e2e8f0;flex-shrink:0}
.ig-filter-search{display:flex;align-items:center;gap:7px;flex:1;min-width:180px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:7px;padding:6px 12px}
.ig-filter-search:focus-within{border-color:#6366f1;background:#fff;box-shadow:0 0 0 3px rgba(99,102,241,.1)}
.ig-filter-input{border:none;background:transparent;font-size:13px;color:#1e293b;outline:none;width:100%}
.ig-filter-input::placeholder{color:#94a3b8}
.ig-estado-tabs{display:flex;gap:2px;background:#f1f5f9;border-radius:8px;padding:3px}
.ig-estado-tab{padding:5px 12px;border:none;background:transparent;border-radius:6px;font-size:12px;font-weight:600;color:#64748b;cursor:pointer;transition:all .15s;display:flex;align-items:center;gap:5px;white-space:nowrap}
.ig-estado-tab:hover{color:#1e293b}
.ig-estado-tab.active{background:#fff;color:#1e293b;box-shadow:0 1px 3px rgba(0,0,0,.1)}
.ig-tab-count{font-size:11px;font-weight:700;padding:1px 6px;border-radius:10px;background:#e2e8f0;color:#64748b}
.ig-tab-ok{background:#dcfce7;color:#15803d}
.ig-tab-off{background:#fee2e2;color:#dc2626}

/* Stats */
.ig-stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,200px),1fr));gap:14px}
.ig-stat-card{display:flex;align-items:center;gap:14px;padding:16px 20px;background:#fff;border:1px solid #e2e8f0;border-radius:10px;transition:box-shadow .15s}
.ig-stat-card:hover{box-shadow:0 2px 8px rgba(0,0,0,.05)}
.ig-stat-icon{width:40px;height:40px;border-radius:10px;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.ig-stat-icon-total{background:#eef2ff;color:#6366f1}
.ig-stat-icon-active{background:#dcfce7;color:#16a34a}
.ig-stat-icon-revoked{background:#fef2f2;color:#dc2626}
.ig-stat-info{display:flex;flex-direction:column}
.ig-stat-num{font-size:22px;font-weight:700;color:#0f172a;line-height:1}
.ig-stat-label{font-size:12px;color:#64748b;margin-top:2px}

/* Card generico */
.ig-card{background:#fff;border:1px solid #e2e8f0;border-radius:12px;overflow:hidden}
.ig-card-header{display:flex;align-items:center;justify-content:space-between;padding:14px 20px;border-bottom:1px solid #f1f5f9;background:#fafbfc}
.ig-card-title{display:flex;align-items:center;gap:8px;font-size:14px;font-weight:700;color:#1e293b;margin:0}
.ig-card-tag{font-size:10px;font-weight:700;color:#6366f1;background:#eef2ff;padding:2px 8px;border-radius:4px;letter-spacing:.06em}
.ig-card-body{overflow-x:auto}

/* Tabla */
.ig-table{width:100%;border-collapse:collapse;font-size:13px}
.ig-table th{font-weight:600;color:#64748b;text-transform:uppercase;font-size:11px;letter-spacing:.05em;padding:10px 16px;text-align:left;border-bottom:1px solid #e2e8f0;background:#fafbfc}
.ig-table td{padding:12px 16px;border-bottom:1px solid #f1f5f9;color:#334155;vertical-align:middle}
.ig-table tbody tr:hover{background:#f8fafc}
.ig-table tbody tr:last-child td{border-bottom:none}
.ig-row-dim{opacity:.45}
.ig-name-cell{display:flex;align-items:center;gap:8px}
.ig-name{font-weight:600;color:#1e293b}
.ig-prefix{background:#f1f5f9;padding:3px 10px;border-radius:5px;font-size:12px;color:#6366f1;font-weight:500;border:1px solid #e2e8f0}
.ig-scope-list{display:flex;flex-wrap:wrap;gap:4px}
.ig-scope{background:#ede9fe;color:#7c3aed;font-size:10px;font-weight:600;padding:2px 8px;border-radius:4px}
.ig-badge{display:inline-flex;align-items:center;gap:5px;font-size:12px;font-weight:600;padding:3px 10px;border-radius:20px}
.ig-badge-dot{width:6px;height:6px;border-radius:50%}
.ig-badge-ok{background:#dcfce7;color:#15803d}.ig-badge-ok .ig-badge-dot{background:#16a34a}
.ig-badge-off{background:#fef2f2;color:#dc2626}.ig-badge-off .ig-badge-dot{background:#dc2626}
.ig-date{font-size:12px;color:#64748b;white-space:nowrap}
.ig-btn-revoke{display:inline-flex;align-items:center;justify-content:center;width:32px;height:32px;border:1px solid #fecaca;background:#fff;border-radius:7px;color:#dc2626;cursor:pointer;transition:all .15s}
.ig-btn-revoke:hover{background:#fef2f2;border-color:#dc2626}
.ig-text-muted{color:#d1d5db;font-size:12px}

/* Empty */
.ig-empty{display:flex;flex-direction:column;align-items:center;gap:10px;padding:56px 20px}
.ig-empty-icon{width:64px;height:64px;background:#f8fafc;border:2px dashed #e2e8f0;border-radius:16px;display:flex;align-items:center;justify-content:center}
.ig-empty-title{font-size:15px;font-weight:700;color:#475569;margin:0}
.ig-empty-desc{font-size:13px;color:#94a3b8;margin:0}
.ig-loading{display:flex;align-items:center;justify-content:center;gap:10px;padding:40px;color:#64748b;font-size:13px}
.ig-spinner{width:18px;height:18px;border:2px solid #e2e8f0;border-top-color:#6366f1;border-radius:50%;animation:spin .6s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}

/* Docs - viejo (compat) */
.ig-docs-body{padding:20px;display:flex;flex-direction:column;gap:14px}
.ig-docs-intro{font-size:13px;color:#64748b;margin:0;line-height:1.6}
.ig-docs-intro code,.ig-inline-code{background:#f1f5f9;padding:2px 7px;border-radius:4px;font-size:12px;color:#6366f1;font-weight:600}
.ig-ep-card{padding:14px 16px;background:#fafbfc;border:1px solid #f1f5f9;border-radius:8px;display:flex;flex-direction:column;gap:6px;transition:border-color .15s}
.ig-ep-card:hover{border-color:#c7d2fe}
.ig-ep-block{padding:14px 0;border-bottom:1px solid #f1f5f9;display:flex;flex-direction:column;gap:8px}
.ig-ep-block:last-child{border-bottom:none}
.ig-ep-row{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.ig-method{background:#dcfce7;color:#15803d;font-weight:700;font-size:11px;padding:3px 10px;border-radius:4px;letter-spacing:.04em;flex-shrink:0}
.ig-ep-path{background:#fff;padding:5px 12px;border-radius:6px;font-size:12px;color:#334155;border:1px solid #e2e8f0}
.ig-ep-path em{color:#6366f1;font-style:normal;font-weight:600}
.ig-ep-desc{font-size:12px;color:#64748b;margin:0}
.ig-ep-scope{display:inline-block;width:fit-content;font-size:10px;font-weight:600;color:#7c3aed;background:#ede9fe;padding:2px 8px;border-radius:4px}
.ig-example-box{background:#1e293b;border-radius:8px;overflow:hidden}
.ig-example-header{display:flex;align-items:center;gap:6px;padding:10px 14px;color:#94a3b8;font-size:12px;font-weight:600;border-bottom:1px solid #334155}
.ig-code-block{margin:0;padding:14px;font-size:12px;color:#e2e8f0;line-height:1.6;white-space:pre-wrap;word-break:break-all;font-family:'Fira Code',monospace}

/* Doc tabs */
.ig-doc-tabs{display:flex;gap:0;border-bottom:1px solid #e2e8f0;background:#fafbfc;padding:0 16px}
.ig-doc-tab{padding:10px 16px;font-size:13px;font-weight:500;color:#64748b;background:none;border:none;border-bottom:2px solid transparent;cursor:pointer;transition:all .15s;margin-bottom:-1px}
.ig-doc-tab:hover{color:#1e293b}
.ig-doc-tab-active{color:#6366f1;font-weight:700;border-bottom-color:#6366f1}

/* Info grid */
.ig-info-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,220px),1fr));gap:12px}
.ig-info-card{display:flex;align-items:flex-start;gap:12px;padding:14px;background:#fafbfc;border:1px solid #f1f5f9;border-radius:8px}
.ig-info-icon{width:34px;height:34px;border-radius:8px;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.ig-info-icon-blue{background:#eff6ff;color:#2563eb}
.ig-info-icon-green{background:#f0fdf4;color:#16a34a}
.ig-info-icon-purple{background:#f5f3ff;color:#7c3aed}
.ig-info-icon-amber{background:#fffbeb;color:#d97706}
.ig-info-title{font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;letter-spacing:.04em;margin-bottom:4px}
.ig-info-val{font-size:12px;color:#334155}

/* Section title */
.ig-section-title{font-size:12px;font-weight:700;color:#475569;text-transform:uppercase;letter-spacing:.06em;padding-bottom:6px;border-bottom:1px solid #f1f5f9}

/* Ref table */
.ig-ref-table{width:100%;border-collapse:collapse;font-size:13px}
.ig-ref-table th{font-weight:600;color:#64748b;font-size:11px;text-transform:uppercase;letter-spacing:.04em;padding:8px 14px;text-align:left;background:#fafbfc;border-bottom:1px solid #e2e8f0}
.ig-ref-table td{padding:10px 14px;border-bottom:1px solid #f1f5f9;color:#334155;vertical-align:middle}
.ig-ref-table td code{background:#f1f5f9;padding:2px 6px;border-radius:4px;font-size:11px;color:#6366f1}
.ig-ref-table tbody tr:last-child td{border-bottom:none}
.ig-param-table-wrap{overflow-x:auto;border:1px solid #f1f5f9;border-radius:8px}

/* Steps */
.ig-steps{display:flex;flex-direction:column;gap:18px}
.ig-step{display:flex;gap:14px;align-items:flex-start}
.ig-step-num{width:28px;height:28px;min-width:28px;background:#6366f1;color:#fff;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:13px;font-weight:700}
.ig-step-title{font-size:13px;font-weight:700;color:#1e293b;margin-bottom:4px}
.ig-step-desc{font-size:12px;color:#64748b;margin:0;line-height:1.6}
.ig-step-desc em{color:#6366f1;font-style:normal;font-weight:600}
.ig-step-desc strong{color:#1e293b}

/* Warn box */
.ig-warn-box{display:flex;align-items:flex-start;gap:10px;font-size:12px;color:#92400e;background:#fffbeb;padding:12px 16px;border-radius:8px;border:1px solid #fef3c7;line-height:1.5}
.ig-warn-box strong{color:#854d0e}

/* HTTP badges */
.ig-http-badge{display:inline-block;font-size:11px;font-weight:700;padding:2px 8px;border-radius:4px;letter-spacing:.02em}
.ig-http-4xx{background:#fef3c7;color:#d97706}
.ig-http-429{background:#ffe4e6;color:#e11d48}
.ig-http-5xx{background:#fee2e2;color:#dc2626}

/* Modal */
.ig-overlay{position:fixed;inset:0;background:rgba(15,23,42,.5);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;z-index:100;padding:20px}
.ig-modal{background:#fff;border-radius:14px;width:100%;max-width:min(480px,92vw);box-shadow:0 25px 50px rgba(0,0,0,.2);overflow:hidden}
.ig-modal-header{display:flex;align-items:center;gap:12px;padding:18px 22px;border-bottom:1px solid #f1f5f9}
.ig-modal-icon{width:40px;height:40px;background:#eef2ff;border-radius:10px;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.ig-modal-icon-success{background:#dcfce7}
.ig-modal-title{flex:1;font-size:16px;font-weight:700;color:#0f172a;margin:0}
.ig-modal-close{width:32px;height:32px;background:none;border:1px solid #e2e8f0;border-radius:7px;cursor:pointer;color:#94a3b8;display:flex;align-items:center;justify-content:center;transition:all .15s}
.ig-modal-close:hover{background:#f1f5f9;color:#475569}
.ig-modal-body{padding:20px 22px;display:flex;flex-direction:column;gap:16px}
.ig-modal-footer{display:flex;justify-content:flex-end;gap:8px;padding:16px 22px;border-top:1px solid #f1f5f9;background:#fafbfc}
.ig-modal-notice{display:flex;align-items:flex-start;gap:8px;font-size:12px;color:#92400e;background:#fffbeb;padding:10px 14px;border-radius:8px;border:1px solid #fef3c7;line-height:1.5}
.ig-field{display:flex;flex-direction:column;gap:5px}
.ig-field-label{font-size:12px;font-weight:600;color:#475569;text-transform:uppercase;letter-spacing:.04em}
.ig-field-input,.ig-field-select{padding:9px 14px;border:1px solid #cbd5e1;border-radius:8px;font-size:13px;color:#1e293b;background:#fff;transition:all .15s}
.ig-field-input:focus,.ig-field-select:focus{outline:none;border-color:#6366f1;box-shadow:0 0 0 3px rgba(99,102,241,.12)}
.ig-field-hint{font-size:11px;color:#94a3b8}
.ig-btn{display:inline-flex;align-items:center;gap:6px;padding:9px 18px;border:none;border-radius:8px;font-size:13px;font-weight:600;cursor:pointer;transition:all .15s}
.ig-btn-primary{background:#6366f1;color:#fff;box-shadow:0 1px 3px rgba(99,102,241,.25)}.ig-btn-primary:hover{background:#4f46e5}
.ig-btn-primary:disabled{opacity:.5;cursor:not-allowed}
.ig-btn-ghost{background:transparent;color:#64748b;border:1px solid #e2e8f0}.ig-btn-ghost:hover{background:#f8fafc}

/* Key result */
.ig-key-alert{display:flex;align-items:flex-start;gap:8px;font-size:13px;font-weight:600;color:#c2410c;background:#fff7ed;padding:12px 16px;border-radius:8px;border:1px solid #fed7aa;line-height:1.4}
.ig-key-box{display:flex;align-items:center;gap:10px;background:#0f172a;padding:14px 16px;border-radius:8px}
.ig-key-value{flex:1;font-size:13px;color:#a5b4fc;word-break:break-all;font-family:'Fira Code',monospace;line-height:1.5}
.ig-btn-copy{display:inline-flex;align-items:center;gap:5px;padding:7px 14px;border:1px solid #334155;background:#1e293b;color:#e2e8f0;border-radius:6px;font-size:12px;font-weight:600;cursor:pointer;transition:all .15s;white-space:nowrap}
.ig-btn-copy:hover{background:#334155;border-color:#475569}
.ig-btn-copied{background:#065f46;border-color:#065f46;color:#d1fae5}

/* Transitions */
.fade-enter-active,.fade-leave-active{transition:opacity .2s ease}
.fade-enter-from,.fade-leave-to{opacity:0}

@media(max-width:768px){
  .ig-hero{flex-direction:column;align-items:stretch}
  .ig-stats{grid-template-columns:1fr}
  .ig-table{font-size:12px}
  .ig-table th,.ig-table td{padding:8px 10px}
}
</style>
