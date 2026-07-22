<template>
  <div class="rd-wrapper">
    <!-- Header oscuro -->
    <div class="rd-header">REGISTRO DE DATOS</div>

    <!-- Panel de filtros -->
    <div class="rd-filtros">
      <!-- Fila 1: selectores en cascada -->
      <div class="filtros-row">
        <div class="filtro-group">
          <label>Empresa</label>
          <select v-model="sel.empresaId" @change="onEmpresaChange">
            <option value="">— Empresa —</option>
            <option v-for="e in empresas" :key="e.id" :value="e.id">{{ e.nombre }}</option>
          </select>
        </div>
        <div class="filtro-group">
          <label>Planta</label>
          <select v-model="sel.plantaId" @change="onPlantaChange" :disabled="!sel.empresaId">
            <option value="">— Planta —</option>
            <option v-for="p in plantasFiltradas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
          </select>
        </div>
        <div class="filtro-group">
          <label>Línea</label>
          <select v-model="sel.lineaId" @change="onLineaChange" :disabled="!sel.plantaId">
            <option value="">— Línea —</option>
            <option v-for="l in lineasFiltradas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
          </select>
        </div>
        <div v-if="!esLineaEnergia" class="filtro-group">
          <label>Tipo de Variable</label>
          <select v-model="sel.tipoEvento">
            <option value="oee">Variable Continua (OEE)</option>
            <option value="corte">Corte</option>
            <option value="">Todos</option>
          </select>
        </div>
        <div v-if="esLineaEnergia" class="modo-pill modo-energia">
          ⚡ Medidor Energía
        </div>
        <div v-if="modoLinea" class="modo-pill" :class="`modo-${modoLinea}`">
          {{ modoLinea === 'botellas' ? '🍶' : '🧵' }}
          {{ modoLinea }} · c/{{ pasoMin }}min
        </div>
      </div>

      <!-- Fila 2: date-time pickers con sliders + acciones -->
      <div class="filtros-row filtros-row-dt">
        <!-- Picker INICIO -->
        <div class="dt-picker">
          <div class="dt-picker-title">INICIO</div>
          <input v-model="desdeDate" type="date" class="dt-date-input" />
          <div class="dt-clock">{{ pad(desdeHora) }}:{{ pad(desdeMins) }}</div>
          <div class="dt-slider-row">
            <span class="dt-lbl">Hora</span>
            <input type="range" v-model.number="desdeHora" min="0" max="23" step="1" class="dt-slider" />
            <span class="dt-num">{{ desdeHora }}</span>
          </div>
          <div class="dt-slider-row">
            <span class="dt-lbl">Min</span>
            <input type="range" v-model.number="desdeMins" min="0" :max="maxMins" :step="pasoMin" class="dt-slider" />
            <span class="dt-num">{{ desdeMins }}</span>
          </div>
        </div>

        <!-- Picker FIN -->
        <div class="dt-picker">
          <div class="dt-picker-title">FIN</div>
          <input v-model="hastaDate" type="date" class="dt-date-input" />
          <div class="dt-clock">{{ pad(hastaHora) }}:{{ pad(hastaMins) }}</div>
          <div class="dt-slider-row">
            <span class="dt-lbl">Hora</span>
            <input type="range" v-model.number="hastaHora" min="0" max="23" step="1" class="dt-slider" />
            <span class="dt-num">{{ hastaHora }}</span>
          </div>
          <div class="dt-slider-row">
            <span class="dt-lbl">Min</span>
            <input type="range" v-model.number="hastaMins" min="0" :max="maxMins" :step="pasoMin" class="dt-slider" />
            <span class="dt-num">{{ hastaMins }}</span>
          </div>
        </div>

        <!-- Acciones -->
        <div class="dt-acciones">
          <button class="btn-buscar" @click="buscar" :disabled="cargando">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/>
            </svg>
            BUSCAR
          </button>
          <div class="spacer"></div>
          <button class="btn-accion" disabled>✏ EDITAR CELDA</button>
          <button class="btn-accion btn-descarga" @click="descargarCSV" :disabled="filas.length === 0">
            ⬇ DESCARGAR DATA SELECCIONADA
          </button>
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="rd-error">{{ error }}</div>

    <!-- Tabla -->
    <div class="rd-tabla-wrapper">
      <table class="rd-tabla">
        <thead>
          <tr>
            <th class="th-num">ID</th>
            <th class="th-fecha" @click="toggleOrden" style="cursor:pointer">
              Fecha <span class="orden-icon">{{ ordenAsc ? '↑' : '↓' }}</span>
            </th>
            <th v-for="col in columnas" :key="col.key" :title="col.label">{{ col.key }}</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="cargando">
            <tr v-for="n in 8" :key="n" class="tr-skeleton">
              <td :colspan="2 + columnas.length"><div class="skeleton"></div></td>
            </tr>
          </template>
          <template v-else-if="filas.length === 0">
            <tr>
              <td :colspan="2 + columnas.length" class="td-vacio">
                Sin datos para los filtros seleccionados
              </td>
            </tr>
          </template>
          <template v-else>
            <tr v-for="(fila, i) in filasOrdenadas" :key="fila.id"
                :class="i % 2 === 1 ? 'tr-par' : ''">
              <td class="td-num">{{ fila.id }}</td>
              <td class="td-fecha">{{ fila.fecha }}</td>
              <td v-for="col in columnas" :key="col.key" class="td-dato">
                {{ fila.valores[col.key] ?? '' }}
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- Pie: total + paginación -->
    <div class="rd-pie">
      <span class="total-info">{{ total }} registros encontrados</span>
      <div class="paginacion">
        <button @click="irPagina(pagina - 1)" :disabled="pagina === 0 || cargando">‹ Anterior</button>
        <span>Pág. {{ pagina + 1 }} / {{ totalPaginas }}</span>
        <button @click="irPagina(pagina + 1)" :disabled="pagina + 1 >= totalPaginas || cargando">Siguiente ›</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import companyService from '@/api/services/company.service'
import { plantService } from '@/api/services/plant.service'
import { lineService } from '@/api/services/line.service'
import { datosRecibidosService } from '@/api/services/datosRecibidos.service'

const LIMITE = 100

// ─── Helpers ──────────────────────────────────────────────────────────────────
function pad(n) { return String(n).padStart(2, '0') }

function hoy() {
  const d = new Date()
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function haceUnaSemana() {
  const d = new Date(Date.now() - 7 * 86400000)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// ─── Estado de filtros cascada ────────────────────────────────────────────────
const sel = ref({
  empresaId: '',
  plantaId: '',
  lineaId: '',
  tipoEvento: 'oee'
})

// ─── Estado de fecha/hora por sliders ────────────────────────────────────────
const desdeDate = ref(haceUnaSemana())
const desdeHora = ref(0)
const desdeMins = ref(0)

const hastaDate = ref(hoy())
const hastaHora = ref(new Date().getHours())
const hastaMins = ref(0)

// Ensambla ISO a partir de los sliders
const desdeISO = computed(() => {
  if (!desdeDate.value) return ''
  return `${desdeDate.value}T${pad(desdeHora.value)}:${pad(desdeMins.value)}:00`
})
const hastaISO = computed(() => {
  if (!hastaDate.value) return ''
  return `${hastaDate.value}T${pad(hastaHora.value)}:${pad(hastaMins.value)}:59`
})

// ─── Catálogos ────────────────────────────────────────────────────────────────
const empresas = ref([])
const plantas  = ref([])
const lineas   = ref([])

const plantasFiltradas = computed(() =>
  plantas.value.filter(p => !sel.value.empresaId || p.empresa_id == sel.value.empresaId)
)
const lineasFiltradas = computed(() =>
  lineas.value.filter(l => !sel.value.plantaId || l.planta_id == sel.value.plantaId)
)

// ─── Modo de la línea seleccionada ───────────────────────────────────────────
const lineaSeleccionada = computed(() =>
  lineas.value.find(l => l.id == sel.value.lineaId) ?? null
)

// Acepta campo 'mode', 'modo' o 'detection_mode' según lo que devuelva la API
const modoLinea = computed(() => {
  const l = lineaSeleccionada.value
  if (!l) return ''
  return l.mode || l.modo || l.detection_mode || ''
})

// Detección de línea de tipo Energía
const esLineaEnergia = computed(() => lineaSeleccionada.value?.tipo === 'Energía')

// Paso de minutos: textil=30min, botellas=5min
const pasoMin = computed(() => modoLinea.value === 'botellas' ? 5 : 30)
const maxMins = computed(() => modoLinea.value === 'botellas' ? 55 : 30)

// Cuando cambia el modo, ajustar los minutos al múltiplo válido más cercano y respetar el nuevo máximo
watch(pasoMin, (paso) => {
  const nuevoMax = modoLinea.value === 'botellas' ? 55 : 30
  desdeMins.value = Math.min(Math.round(desdeMins.value / paso) * paso, nuevoMax)
  hastaMins.value = Math.min(Math.round(hastaMins.value / paso) * paso, nuevoMax)
})

async function onEmpresaChange() {
  sel.value.plantaId = ''
  sel.value.lineaId  = ''
  try {
    const res = await plantService.getAll({ empresa_id: sel.value.empresaId })
    plantas.value = Array.isArray(res) ? res : (res.data ?? [])
  } catch {}
}

async function onPlantaChange() {
  sel.value.lineaId = ''
  try {
    const res = await lineService.getAll({ planta_id: sel.value.plantaId })
    lineas.value = Array.isArray(res) ? res : (res.data ?? [])
  } catch {}
}

function onLineaChange() {
  // pasoMin se recalcula solo; el watch snap ajusta los minutos
}

// ─── Tabla ────────────────────────────────────────────────────────────────────
const eventos   = ref([])
const total     = ref(0)
const pagina    = ref(0)
const cargando  = ref(false)
const error     = ref('')
const ordenAsc  = ref(true)

const offset      = computed(() => pagina.value * LIMITE)
const totalPaginas = computed(() => Math.max(1, Math.ceil(total.value / LIMITE)))

// Columnas fijas OEE en orden canónico. key = nombre en payload, label = encabezado visible.
const COLUMNAS_FIJAS = [
  { key: 'CONTEO_1',                    label: 'Conteo Unitario Principal' },
  { key: 'CONTEO_2',                    label: 'Conteo Unitario Principal Redundancia' },
  { key: 'T_DISPONIBLE',               label: 'Tiempo Disponible' },
  { key: 'T_MICROPARADA',              label: 'Tiempo de Microparada' },
  { key: 'T_PARADA_NO_ASIGNADA',       label: 'Tiempo de Parada No Asignada' },
  { key: 'MARCA',                       label: 'Marca' },
  { key: 'SABOR',                       label: 'Sabor' },
  { key: 'TAMANIO',                     label: 'Tamaño' },
  { key: 'MATERIAL',                    label: 'Material' },
  { key: 'DESTINO',                     label: 'Destino' },
  { key: 'T_REFRIGERIO',               label: 'Refrigerio' },
  { key: 'T_CAPACITACION_OBLIGATORIA', label: 'Capacitación Obligatoria' },
  { key: 'T_MANTENIMIENTO_PLANIFICADO',label: 'Mantenimiento Planificado' },
  { key: 'T_PARADA_PROGRAMADA',        label: 'Tiempo de Parada Programada' },
  { key: 'T_PARADA_NO_PROGRAMADA',     label: 'Tiempo de Parada No Programada' },
  { key: 'TIPO_PARADA_PROGRAMADA',     label: 'Tipo de Parada Programada' },
  { key: 'TIPO_PARADA_NO_PROGRAMADA',  label: 'Tipo de Parada No Programada' },
  { key: 'MERMA',                      label: 'Merma' },
]

// Columnas de energía para medidores MC60
const COLUMNAS_ENERGIA = [
  { key: 'CONSUMO_ACTIVA',     label: 'Consumo Activo (kWh)' },
  { key: 'CONSUMO_REACTIVA',   label: 'Consumo Reactivo (kVARh)' },
  { key: 'CONSUMO_APARENTE',   label: 'Consumo Aparente (kVAh)' },
  { key: 'ENERGIA_ACTIVA',     label: 'Contador Activo (Wh)' },
  { key: 'ENERGIA_REACTIVA',   label: 'Contador Reactivo (VArh)' },
  { key: 'ENERGIA_APARENTE',   label: 'Contador Aparente (VAh)' },
  { key: 'VOLTAJE_A',          label: 'Voltaje A (V)' },
  { key: 'VOLTAJE_B',          label: 'Voltaje B (V)' },
  { key: 'VOLTAJE_C',          label: 'Voltaje C (V)' },
  { key: 'VOLTAJE_AVG',        label: 'Volt. Prom (V)' },
  { key: 'CORRIENTE_A',        label: 'Corr. A (A)' },
  { key: 'CORRIENTE_B',        label: 'Corr. B (A)' },
  { key: 'CORRIENTE_C',        label: 'Corr. C (A)' },
  { key: 'CORRIENTE_AVG',      label: 'Corr. Prom (A)' },
  { key: 'POTENCIA_ACTIVA',    label: 'Pot. Activa (W)' },
  { key: 'POTENCIA_REACTIVA',  label: 'Pot. React. (VAr)' },
  { key: 'POTENCIA_APARENTE',  label: 'Pot. Apar. (VA)' },
  { key: 'FACTOR_POTENCIA',    label: 'FP' },
  { key: 'FRECUENCIA',         label: 'Freq. (Hz)' },
  { key: 'THD_IA',             label: 'THD IA (%)' },
  { key: 'THD_IB',             label: 'THD IB (%)' },
  { key: 'THD_IC',             label: 'THD IC (%)' },
  { key: 'THD_UA',             label: 'THD UA (%)' },
  { key: 'THD_UB',             label: 'THD UB (%)' },
  { key: 'THD_UC',             label: 'THD UC (%)' },
]

// Muestra columnas según tipo de línea
const columnas = computed(() => esLineaEnergia.value ? COLUMNAS_ENERGIA : COLUMNAS_FIJAS)

const filas = computed(() => {
  if (esLineaEnergia.value) {
    // Snapshots de energía — campos pre-extraídos del medidor MC60
    return eventos.value.map(ev => ({
      id:    ev.id,
      fecha: formatFecha(ev.hora),
      _ts:   new Date(ev.hora).getTime(),
      valores: {
        'CONSUMO_ACTIVA':    ev.consumo_activa    != null ? Number(ev.consumo_activa).toFixed(4)    : '',
        'CONSUMO_REACTIVA':  ev.consumo_reactiva  != null ? Number(ev.consumo_reactiva).toFixed(4)  : '',
        'CONSUMO_APARENTE':  ev.consumo_aparente  != null ? Number(ev.consumo_aparente).toFixed(4)  : '',
        'ENERGIA_ACTIVA':    ev.energia_activa    != null ? Number(ev.energia_activa).toFixed(0)    : '',
        'ENERGIA_REACTIVA':  ev.energia_reactiva  != null ? Number(ev.energia_reactiva).toFixed(0)  : '',
        'ENERGIA_APARENTE':  ev.energia_aparente  != null ? Number(ev.energia_aparente).toFixed(0)  : '',
        'VOLTAJE_A':         ev.voltaje_a         != null ? Number(ev.voltaje_a).toFixed(1)          : '',
        'VOLTAJE_B':         ev.voltaje_b         != null ? Number(ev.voltaje_b).toFixed(1)          : '',
        'VOLTAJE_C':         ev.voltaje_c         != null ? Number(ev.voltaje_c).toFixed(1)          : '',
        'VOLTAJE_AVG':       ev.voltaje_avg       != null ? Number(ev.voltaje_avg).toFixed(1)        : '',
        'CORRIENTE_A':       ev.corriente_a       != null ? Number(ev.corriente_a).toFixed(2)        : '',
        'CORRIENTE_B':       ev.corriente_b       != null ? Number(ev.corriente_b).toFixed(2)        : '',
        'CORRIENTE_C':       ev.corriente_c       != null ? Number(ev.corriente_c).toFixed(2)        : '',
        'CORRIENTE_AVG':     ev.corriente_avg     != null ? Number(ev.corriente_avg).toFixed(2)      : '',
        'POTENCIA_ACTIVA':   ev.potencia_activa   != null ? Number(ev.potencia_activa).toFixed(0)   : '',
        'POTENCIA_REACTIVA': ev.potencia_reactiva != null ? Number(ev.potencia_reactiva).toFixed(0) : '',
        'POTENCIA_APARENTE': ev.potencia_aparente != null ? Number(ev.potencia_aparente).toFixed(0) : '',
        'FACTOR_POTENCIA':   ev.factor_potencia   != null ? Number(ev.factor_potencia).toFixed(3)   : '',
        'FRECUENCIA':        ev.frecuencia_hz     != null ? Number(ev.frecuencia_hz).toFixed(2)      : '',
        'THD_IA':            ev.thd_ia            != null ? Number(ev.thd_ia).toFixed(2)            : '',
        'THD_IB':            ev.thd_ib            != null ? Number(ev.thd_ib).toFixed(2)            : '',
        'THD_IC':            ev.thd_ic            != null ? Number(ev.thd_ic).toFixed(2)            : '',
        'THD_UA':            ev.thd_ua            != null ? Number(ev.thd_ua).toFixed(2)            : '',
        'THD_UB':            ev.thd_ub            != null ? Number(ev.thd_ub).toFixed(2)            : '',
        'THD_UC':            ev.thd_uc            != null ? Number(ev.thd_uc).toFixed(2)            : '',
      }
    }))
  }
  // Eventos OEE — payload con head/data
  return eventos.value.map(ev => {
    const valores = {}
    if (ev.payload?.head && ev.payload?.data) {
      ev.payload.head.forEach((h, i) => { valores[h] = ev.payload.data[i] ?? '' })
    }
    return {
      id:    ev.id,
      fecha: formatFecha(ev.timestamp_edge),
      _ts:   new Date(ev.timestamp_edge).getTime(),
      valores
    }
  })
})

const filasOrdenadas = computed(() =>
  [...filas.value].sort((a, b) => ordenAsc.value ? a._ts - b._ts : b._ts - a._ts)
)

// ─── Carga ────────────────────────────────────────────────────────────────────
async function buscar() {
  pagina.value = 0
  await cargar()
}

async function cargar() {
  cargando.value = true
  error.value = ''
  try {
    const params = { limit: LIMITE, offset: offset.value }
    if (sel.value.plantaId)  params.planta_id  = sel.value.plantaId
    if (sel.value.empresaId) params.empresa_id = sel.value.empresaId
    if (desdeISO.value)      params.from = new Date(desdeISO.value).toISOString()
    if (hastaISO.value)      params.to   = new Date(hastaISO.value).toISOString()

    let res
    if (esLineaEnergia.value) {
      // Línea de tipo Energía → endpoint de energy-ingest
      res = await datosRecibidosService.listarEnergia(params)
      const data = res.data ?? res ?? []
      eventos.value = Array.isArray(data) ? data : []
    } else {
      // Línea OEE → endpoint de datos-recibidos (ingest.raw_events)
      if (sel.value.tipoEvento) params.event_type = sel.value.tipoEvento
      if (sel.value.lineaId)    params.linea_id   = sel.value.lineaId
      res = await datosRecibidosService.listar(params)
      const data = res.data ?? res ?? []
      eventos.value = (Array.isArray(data) ? data : []).map(ev => ({
        ...ev,
        payload: typeof ev.payload === 'string' ? JSON.parse(ev.payload) : ev.payload
      }))
    }
    total.value = res.total ?? eventos.value.length
  } catch (e) {
    error.value = e?.message ?? 'Error al cargar datos'
  } finally {
    cargando.value = false
  }
}

function irPagina(n) {
  if (n < 0 || n >= totalPaginas.value) return
  pagina.value = n
  cargar()
}

function toggleOrden() { ordenAsc.value = !ordenAsc.value }

// ─── Exportar CSV ─────────────────────────────────────────────────────────────
function descargarCSV() {
  const headers = ['N°', 'Fecha', ...columnas.value.map(c => c.label)]
  const rows = filasOrdenadas.value.map((f, i) => [
    offset.value + i + 1,
    f.fecha,
    ...columnas.value.map(c => f.valores[c.key] ?? '')
  ])
  const csv = [headers, ...rows].map(r => r.join(',')).join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `datos_recibidos_${Date.now()}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

// ─── Formato de fecha en tabla: dd/mm/yyyy HH:MM ─────────────────────────────
function formatFecha(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  const dd = pad(d.getDate())
  const mm = pad(d.getMonth() + 1)
  const yy = d.getFullYear()
  const HH = pad(d.getHours())
  const MM = pad(d.getMinutes())
  return `${dd}/${mm}/${yy} ${HH}:${MM}`
}

// ─── Init ─────────────────────────────────────────────────────────────────────
onMounted(async () => {
  try {
    const res = await companyService.getAll()
    empresas.value = Array.isArray(res) ? res : (res.data ?? [])
  } catch {}
  try {
    const rp = await plantService.getAll()
    plantas.value = Array.isArray(rp) ? rp : (rp.data ?? [])
  } catch {}
  try {
    const rl = await lineService.getAll()
    lineas.value = Array.isArray(rl) ? rl : (rl.data ?? [])
  } catch {}
})
</script>

<style scoped>
.rd-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #f0f2f5;
  font-family: inherit;
}

/* Header */
.rd-header {
  background: #0d1b3e;
  color: #fff;
  font-size: .95rem;
  font-weight: 700;
  letter-spacing: .05em;
  padding: .75rem 1.5rem;
}

/* Panel filtros */
.rd-filtros {
  background: #fff;
  border-bottom: 1px solid #dde1e8;
  padding: .9rem 1.5rem .85rem;
  display: flex;
  flex-direction: column;
  gap: .75rem;
}

.filtros-row {
  display: flex;
  flex-wrap: wrap;
  gap: .75rem;
  align-items: flex-end;
}

.filtro-group {
  display: flex;
  flex-direction: column;
  gap: .2rem;
}
.filtro-group label {
  font-size: .72rem;
  font-weight: 600;
  color: #555;
  text-transform: uppercase;
  letter-spacing: .03em;
}
.filtro-group select {
  border: 1px solid #c8cdd6;
  border-radius: 4px;
  padding: .35rem .6rem;
  font-size: .875rem;
  color: #1e293b;
  background: #fff;
  min-width: 170px;
  height: 34px;
}
.filtro-group select:focus { outline: 2px solid #3b82f6; outline-offset: -1px; }
.filtro-group select:disabled { background: #f4f5f8; color: #999; }

/* Pill de modo */
.modo-pill {
  align-self: center;
  padding: .3rem .75rem;
  border-radius: 99px;
  font-size: .75rem;
  font-weight: 700;
  border: 1px solid;
}
.modo-textil   { background: #eff6ff; color: #1d4ed8; border-color: #bfdbfe; }
.modo-botellas { background: #f0fdf4; color: #166534; border-color: #bbf7d0; }
.modo-energia  { background: #fefce8; color: #854d0e; border-color: #fde68a; }

/* Fila con date-time pickers */
.filtros-row-dt {
  align-items: flex-start;
  gap: 1rem;
}

/* Picker individual */
.dt-picker {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: .65rem .9rem;
  min-width: 200px;
}
.dt-picker-title {
  font-size: .7rem;
  font-weight: 700;
  letter-spacing: .06em;
  color: #64748b;
  margin-bottom: .4rem;
  text-transform: uppercase;
}
.dt-date-input {
  width: 100%;
  border: 1px solid #c8cdd6;
  border-radius: 5px;
  padding: .3rem .5rem;
  font-size: .85rem;
  color: #1e293b;
  background: #fff;
  margin-bottom: .4rem;
}
.dt-date-input:focus { outline: 2px solid #3b82f6; outline-offset: -1px; }

.dt-clock {
  text-align: center;
  font-size: 2rem;
  font-weight: 800;
  color: #0d1b3e;
  letter-spacing: .05em;
  line-height: 1;
  margin: .3rem 0 .5rem;
}

.dt-slider-row {
  display: flex;
  align-items: center;
  gap: .5rem;
  margin-bottom: .3rem;
}
.dt-lbl {
  font-size: .75rem;
  color: #64748b;
  width: 28px;
  flex-shrink: 0;
}
.dt-slider {
  flex: 1;
  height: 4px;
  border-radius: 99px;
  -webkit-appearance: none;
  appearance: none;
  background: #cbd5e1;
  outline: none;
  cursor: pointer;
}
.dt-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #0d1b3e;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0,0,0,.25);
}
.dt-slider::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #0d1b3e;
  cursor: pointer;
  border: none;
}
.dt-num {
  font-size: .8rem;
  font-weight: 700;
  color: #0d1b3e;
  width: 22px;
  text-align: right;
  flex-shrink: 0;
}

/* Acciones (buscar + descarga) */
.dt-acciones {
  display: flex;
  flex-direction: column;
  gap: .5rem;
  justify-content: flex-start;
  padding-top: 1.8rem;
}
.spacer { flex: 1; min-height: .5rem; }

.btn-buscar {
  display: inline-flex;
  align-items: center;
  gap: .4rem;
  background: #0d1b3e;
  color: #fff;
  border: none;
  border-radius: 5px;
  padding: .45rem 1.1rem;
  font-size: .85rem;
  font-weight: 700;
  letter-spacing: .04em;
  cursor: pointer;
  white-space: nowrap;
}
.btn-buscar:hover:not(:disabled) { background: #1a2f5e; }
.btn-buscar:disabled { opacity: .5; cursor: not-allowed; }

.btn-accion {
  background: #e8eaf0;
  color: #555;
  border: 1px solid #c8cdd6;
  border-radius: 4px;
  padding: .35rem .85rem;
  font-size: .8rem;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}
.btn-accion:disabled { opacity: .45; cursor: not-allowed; }
.btn-descarga { background: #e9f0fe; color: #1e40af; border-color: #bfcffa; }
.btn-descarga:not(:disabled):hover { background: #d4e4fd; }

/* Error */
.rd-error {
  margin: .5rem 1.5rem;
  background: #fee2e2;
  color: #991b1b;
  padding: .6rem 1rem;
  border-radius: 6px;
  font-size: .875rem;
}

/* Tabla */
.rd-tabla-wrapper {
  flex: 1;
  overflow: auto;
  background: #fff;
  margin-top: .5rem;
}
.rd-tabla {
  min-width: 100%;
  border-collapse: collapse;
  font-size: .8rem;
  white-space: nowrap;
}
.rd-tabla thead th {
  background: #fff;
  color: #2d3748;
  font-weight: 700;
  font-size: .75rem;
  padding: .5rem .75rem;
  border-bottom: 2px solid #c8cdd6;
  border-right: 1px solid #e4e7ee;
  text-align: center;
  position: sticky;
  top: 0;
  z-index: 2;
  white-space: nowrap;
  line-height: 1.2;
  vertical-align: bottom;
}
.th-num  { min-width: 38px; }
.th-fecha { min-width: 130px; text-align: left !important; }
.orden-icon { color: #3b82f6; font-size: .8rem; }

.rd-tabla tbody tr { border-bottom: 1px solid #e4e7ee; }
.rd-tabla tbody tr.tr-par { background: #f5f7fa; }
.rd-tabla tbody tr:hover  { background: #eaf1fb; }
.rd-tabla tbody td {
  padding: .38rem .75rem;
  color: #1e293b;
  text-align: center;
  border-right: 1px solid #e4e7ee;
}
.td-num   { font-weight: 600; color: #555; }
.td-fecha { text-align: left; font-family: monospace; font-size: .8rem; color: #374151; }
.td-dato  { font-family: monospace; }
.td-vacio { text-align: center; color: #9ca3af; padding: 3rem; }

/* Skeleton */
.tr-skeleton td { padding: .45rem .75rem; }
.skeleton {
  height: 1.1rem;
  background: linear-gradient(90deg,#f0f2f5 25%,#e2e6ec 50%,#f0f2f5 75%);
  background-size: 200% 100%;
  animation: shimmer 1.2s infinite;
  border-radius: 3px;
}
@keyframes shimmer { 0%{background-position:200% 0} 100%{background-position:-200% 0} }

/* Pie */
.rd-pie {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: .6rem 1.5rem;
  background: #fff;
  border-top: 1px solid #dde1e8;
  font-size: .82rem;
  color: #555;
}
.total-info { font-weight: 600; }
.paginacion { display: flex; align-items: center; gap: .75rem; }
.paginacion button {
  background: #e8eaf0;
  border: 1px solid #c8cdd6;
  border-radius: 4px;
  padding: .3rem .75rem;
  font-size: .8rem;
  cursor: pointer;
}
.paginacion button:disabled { opacity: .4; cursor: not-allowed; }
.paginacion button:not(:disabled):hover { background: #d4d8e4; }
</style>
