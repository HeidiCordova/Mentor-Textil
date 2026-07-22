<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { plantService }       from '@/api/services/plant.service'
import { lineService }        from '@/api/services/line.service'
import { oeeLabService }      from '@/api/services/oee-lab.service'

// ─── Constantes ──────────────────────────────────────────────────────────────

const CIRCUM = 2 * Math.PI * 40

const UNIDADES_VEL = [
  { key: 'us',  label: 'u/seg',  factor: 1    },
  { key: 'um',  label: 'u/min',  factor: 60   },
  { key: 'uh',  label: 'u/hora', factor: 3600 },
]

const DEFAULTS = {
  tTurnoS:                28800,
  tRefrigerioS:             900,
  tCapacitacionS:           600,
  tMantenimientoS:          300,
  tParadasProgramadasS:    1800,
  tMicroparadasS:          1200,
  velocidadNominal:         4.0,
  unidadVel:               'uh',
  unidadesProducidas:      1200,
  unidadesDefectuosas:       24,
}

// ─── Estado del formulario ────────────────────────────────────────────────────

const form = reactive({ ...DEFAULTS })

const scenarioNombre = ref('')
const scenarioNotas  = ref('')
const lineaID        = ref(null)

// ─── Filtros ──────────────────────────────────────────────────────────────────

const plantas = ref([])
const lineas  = ref([])
const plantaID = ref(null)

async function cargarFiltros() {
  try {
    const [pla] = await Promise.all([plantService.getAll()])
    plantas.value = pla.data ?? []
  } catch { /* ignorado */ }
}

watch(plantaID, async (val) => {
  lineas.value = []
  lineaID.value = null
  if (!val) return
  try {
    const r = await lineService.getAll({ planta_id: val })
    lineas.value = r.data ?? []
  } catch { /* ignorado */ }
})

// ─── Cadena de derivados OEE (ISO 22400) ─────────────────────────────────────

// Parada Obligatoria = Refrigerio + Capacitación + Mantenimiento
const tParadaObligatoria = computed(() =>
  form.tRefrigerioS + form.tCapacitacionS + form.tMantenimientoS
)

// Tiempo Disponible = Disponible Total − Parada Obligatoria
const tDisponible = computed(() =>
  Math.max(0, form.tTurnoS - tParadaObligatoria.value)
)

// Tiempo Operativo = Tiempo Disponible − Parada No Obligatoria
const tOperativo = computed(() =>
  Math.max(0, tDisponible.value - form.tParadasProgramadasS)
)

// Disponibilidad = Tiempo Operativo / Tiempo Disponible
const disponibilidad = computed(() => {
  if (tDisponible.value <= 0) return 0
  return Math.min(1, tOperativo.value / tDisponible.value)
})

// Tiempo Neto = Tiempo Operativo − Microparadas
const tNeto = computed(() =>
  Math.max(0, tOperativo.value - form.tMicroparadasS)
)

const velUs = computed(() => {
  const u = UNIDADES_VEL.find(u => u.key === form.unidadVel)
  return form.velocidadNominal / (u?.factor ?? 1)
})

// Tiempo Nominal = Producción / Velocidad Nominal
const tNominalProd = computed(() => {
  if (velUs.value <= 0) return 0
  return form.unidadesProducidas / velUs.value
})

// Tiempo Perdido por Baja Velocidad = max(0, Tiempo Neto − Tiempo Nominal)
const tBajaVelocidad = computed(() =>
  Math.max(0, tNeto.value - tNominalProd.value)
)

// Rendimiento = (Tiempo Operativo − Microparadas − T.Baja Vel.) / Tiempo Operativo
const rendimiento = computed(() => {
  if (tOperativo.value <= 0) return 0
  const num = tOperativo.value - form.tMicroparadasS - tBajaVelocidad.value
  return Math.min(1, Math.max(0, num / tOperativo.value))
})

// Calidad = (Producción − Merma) / Producción
const calidad = computed(() => {
  if (form.unidadesProducidas <= 0) return 0
  return Math.min(1, Math.max(0,
    (form.unidadesProducidas - form.unidadesDefectuosas) / form.unidadesProducidas
  ))
})

const oee = computed(() =>
  disponibilidad.value * rendimiento.value * calidad.value
)

const unidadesTeorico = computed(() =>
  Math.round(tOperativo.value * velUs.value)
)

// ─── Barra de distribución de tiempo ─────────────────────────────────────────

const barSegments = computed(() => {
  const t       = form.tTurnoS || 1
  const oblig   = Math.max(0, tParadaObligatoria.value)
  const nOblig  = Math.max(0, form.tParadasProgramadasS)
  const micro   = Math.max(0, form.tMicroparadasS)
  const bajaVel = Math.max(0, tBajaVelocidad.value)
  const nominal = Math.max(0, Math.min(tNominalProd.value, tNeto.value))
  return [
    { label: 'Producción nominal',    color: '#10b981', val: nominal  },
    { label: 'Baja velocidad',        color: '#84cc16', val: bajaVel  },
    { label: 'Microparadas',          color: '#fbbf24', val: micro    },
    { label: 'Parada no obligatoria', color: '#f97316', val: nOblig   },
    { label: 'Parada obligatoria',    color: '#475569', val: oblig    },
  ].map(s => ({ ...s, pct: (s.val / t * 100).toFixed(1) }))
})

// ─── Análisis automático ──────────────────────────────────────────────────────

const analisis = computed(() => {
  const lines = []
  const d = disponibilidad.value
  const r = rendimiento.value
  const c = calidad.value
  const o = oee.value
  const min = Math.min(d, r, c)

  if (o === 0) return []

  if (min === d && d < 0.95)
    lines.push('La disponibilidad es el factor más limitante. Reducir paradas no planificadas tendrá el mayor impacto.')
  else if (min === r && r < 0.95)
    lines.push('El rendimiento es el factor más limitante. Las microparadas o velocidad sub-nominal reducen la producción efectiva.')
  else if (min === c && c < 0.95)
    lines.push('La calidad es el factor más limitante. Cada unidad defectuosa multiplicada por la velocidad nominal representa tiempo productivo perdido.')

  if (form.tMicroparadasS > 0 && tOperativo.value > 0) {
    const pct = (form.tMicroparadasS / tOperativo.value * 100).toFixed(1)
    lines.push(`Las microparadas representan el ${pct}% del tiempo operativo (${fmtSeg(form.tMicroparadasS)}).`)
  }

  if (tBajaVelocidad.value > 60 && tOperativo.value > 0) {
    const pct = (tBajaVelocidad.value / tOperativo.value * 100).toFixed(1)
    lines.push(`Tiempo perdido por baja velocidad: ${fmtSeg(tBajaVelocidad.value)} (${pct}% del operativo).`)
  }

  if (o >= 0.85)
    lines.push('OEE World Class (≥ 85%): el proceso supera el estándar industrial de referencia.')
  else if (o >= 0.65)
    lines.push('OEE por encima del promedio industrial (65–85%). Existe margen de mejora identificable.')
  else
    lines.push('OEE por debajo del promedio industrial (< 65%). Se requieren acciones de mejora prioritarias.')

  return lines
})

// ─── Guardar / cargar historial ───────────────────────────────────────────────

const historial      = ref([])
const histLoading    = ref(false)
const saving         = ref(false)
const saveMsg        = ref('')
const saveMsgType    = ref('')

async function cargarHistorial() {
  histLoading.value = true
  try {
    const res = await oeeLabService.list()
    historial.value = res.data ?? []
  } catch { historial.value = [] }
  finally { histLoading.value = false }
}

async function guardarSesion() {
  if (!scenarioNombre.value.trim()) {
    saveMsg.value = 'Ingresá un nombre para la sesión.'
    saveMsgType.value = 'err'
    return
  }
  saving.value = true
  saveMsg.value = ''
  try {
    const inputs = {
      t_turno_s:               form.tTurnoS,
      t_refrigerio_s:          form.tRefrigerioS,
      t_capacitacion_s:        form.tCapacitacionS,
      t_mantenimiento_s:       form.tMantenimientoS,
      t_paradas_programadas_s: form.tParadasProgramadasS,
      t_microparadas_s:        form.tMicroparadasS,
      velocidad_nominal:       form.velocidadNominal,
      unidad_vel:              form.unidadVel,
      unidades_producidas:     form.unidadesProducidas,
      unidades_defectuosas:    form.unidadesDefectuosas,
    }
    const results = {
      t_disponible:     tDisponible.value,
      t_operativo:      tOperativo.value,
      t_neto:           tNeto.value,
      t_baja_velocidad: tBajaVelocidad.value,
      disponibilidad:   +disponibilidad.value.toFixed(4),
      rendimiento:      +rendimiento.value.toFixed(4),
      calidad:          +calidad.value.toFixed(4),
      oee:              +oee.value.toFixed(4),
      unidades_teorico: unidadesTeorico.value,
    }
    await oeeLabService.save({
      nombre:   scenarioNombre.value.trim(),
      linea_id: lineaID.value ? Number(lineaID.value) : undefined,
      notas:    scenarioNotas.value.trim(),
      inputs,
      results,
    })
    saveMsg.value = 'Sesión guardada'
    saveMsgType.value = 'ok'
    setTimeout(() => { saveMsg.value = '' }, 3000)
    await cargarHistorial()
  } catch (e) {
    saveMsg.value = e?.response?.data?.error || 'Error al guardar'
    saveMsgType.value = 'err'
  } finally {
    saving.value = false
  }
}

function cargarSesion(s) {
  const i = s.inputs ?? {}
  form.tTurnoS              = i.t_turno_s               ?? DEFAULTS.tTurnoS
  form.tRefrigerioS         = i.t_refrigerio_s          ?? DEFAULTS.tRefrigerioS
  form.tCapacitacionS       = i.t_capacitacion_s        ?? DEFAULTS.tCapacitacionS
  form.tMantenimientoS      = i.t_mantenimiento_s       ?? DEFAULTS.tMantenimientoS
  form.tParadasProgramadasS = i.t_paradas_programadas_s ?? DEFAULTS.tParadasProgramadasS
  form.tMicroparadasS       = i.t_microparadas_s        ?? DEFAULTS.tMicroparadasS
  form.velocidadNominal     = i.velocidad_nominal       ?? DEFAULTS.velocidadNominal
  form.unidadVel            = i.unidad_vel              ?? DEFAULTS.unidadVel
  form.unidadesProducidas   = i.unidades_producidas     ?? DEFAULTS.unidadesProducidas
  form.unidadesDefectuosas  = i.unidades_defectuosas    ?? DEFAULTS.unidadesDefectuosas
  scenarioNombre.value      = s.nombre                  ?? ''
  scenarioNotas.value       = s.notas                   ?? ''
  lineaID.value             = s.linea_id                ?? null
}

async function eliminarSesion(id) {
  if (!confirm('¿Eliminar esta sesión del historial?')) return
  try {
    await oeeLabService.remove(id)
    await cargarHistorial()
  } catch { /* ignorado */ }
}

function nuevaSesion() {
  Object.assign(form, DEFAULTS)
  scenarioNombre.value = ''
  scenarioNotas.value  = ''
  lineaID.value        = null
}

// ─── Computed bidireccional (slider ↔ input numérico en minutos) ─────────────

const tTurnoMin = computed({
  get: () => Math.round(form.tTurnoS / 60),
  set: (v) => { form.tTurnoS = Math.max(0, Math.min(86400, (parseInt(v) || 0) * 60)) },
})
const tRefrigerioMin = computed({
  get: () => Math.round(form.tRefrigerioS / 60),
  set: (v) => { form.tRefrigerioS = Math.max(0, (parseInt(v) || 0) * 60) },
})
const tCapacitacionMin = computed({
  get: () => Math.round(form.tCapacitacionS / 60),
  set: (v) => { form.tCapacitacionS = Math.max(0, (parseInt(v) || 0) * 60) },
})
const tMantenimientoMin = computed({
  get: () => Math.round(form.tMantenimientoS / 60),
  set: (v) => { form.tMantenimientoS = Math.max(0, (parseInt(v) || 0) * 60) },
})
const tProgramadasMin = computed({
  get: () => Math.round(form.tParadasProgramadasS / 60),
  set: (v) => { form.tParadasProgramadasS = Math.max(0, (parseInt(v) || 0) * 60) },
})
const tMicroMin = computed({
  get: () => Math.round(form.tMicroparadasS / 60),
  set: (v) => { form.tMicroparadasS = Math.max(0, (parseInt(v) || 0) * 60) },
})

// ─── Utilidades ───────────────────────────────────────────────────────────────

function fmtSeg(s) {
  const n = Math.round(s)
  if (n < 60) return `${n}s`
  const h = Math.floor(n / 3600)
  const m = Math.floor((n % 3600) / 60)
  const r = n % 60
  if (h > 0) return (m === 0 && r === 0) ? `${h}h` : r === 0 ? `${h}h ${m}m` : `${h}h ${m}m ${r}s`
  return r === 0 ? `${m}min` : `${m}m ${r}s`
}

function fp(v, dec = 1) {
  return (v * 100).toFixed(dec) + '%'
}

function gcolor(v) {
  if (v >= 0.85) return '#10b981'
  if (v >= 0.65) return '#f59e0b'
  return '#ef4444'
}

function gaugeOffset(v) {
  return CIRCUM * (1 - Math.max(0, Math.min(1, v)))
}

function fmtFecha(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  if (isNaN(d)) return ts
  return d.toLocaleString('es-AR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

onMounted(() => {
  cargarFiltros()
  cargarHistorial()
})

// ─── Tab activo ─────────────────────────────────────────────────────────────────────────
const activeTab = ref('iso22400')

// ─── Laboratorio por Eventos (modelo estadístico) ─────────────────────────────
const evForm = reactive({
  tDisponibleMin:      480,
  tOperativoMin:       420,
  multSigma:           2,
  multParada:          3,
  unidadesDefectuosas: 0,
})

const ciclos = ref([16, 16, 17, 15, 16, 18, 16, 15].map(v => ({ v })))

function evAddCiclo() {
  ciclos.value.push({ v: +evMediana.value.toFixed(1) || 16 })
}
function evRemoveCiclo(i) {
  if (ciclos.value.length > 1) ciclos.value.splice(i, 1)
}
function evResetCiclos() {
  ciclos.value = [16, 16, 17, 15, 16, 18, 16, 15].map(v => ({ v }))
  Object.assign(evForm, { tDisponibleMin: 480, tOperativoMin: 420, multSigma: 2, multParada: 3, unidadesDefectuosas: 0 })
}

function evMedianFn(arr) {
  if (!arr.length) return 0
  const s = [...arr].sort((a, b) => a - b)
  const m = Math.floor(s.length / 2)
  return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2
}
function evStdFn(arr) {
  if (arr.length < 2) return 0
  const avg = arr.reduce((a, b) => a + b, 0) / arr.length
  return Math.sqrt(arr.reduce((sum, v) => sum + (v - avg) ** 2, 0) / (arr.length - 1))
}

const evCicloVals = computed(() =>
  ciclos.value.map(c => +c.v || 0).filter(v => v > 0)
)
const evMediana = computed(() => evMedianFn(evCicloVals.value))
const evSigma   = computed(() => evStdFn(evCicloVals.value))
const evTLimite = computed(() => evMediana.value + evForm.multSigma * evSigma.value)
const evTParada = computed(() => evForm.multParada * evMediana.value)

const evCiclosClasif = computed(() =>
  ciclos.value.map(c => {
    const v = +c.v || 0
    if (v <= 0) return { v, estado: 'invalido' }
    return {
      v,
      estado: v <= evTLimite.value ? 'normal'
             : v <  evTParada.value ? 'lenta'
             : 'parada',
    }
  })
)

const evNNormal = computed(() => evCiclosClasif.value.filter(c => c.estado === 'normal').length)
const evNLenta  = computed(() => evCiclosClasif.value.filter(c => c.estado === 'lenta').length)
const evNParada = computed(() => evCiclosClasif.value.filter(c => c.estado === 'parada').length)

const evProdReal     = computed(() => evCicloVals.value.length)
const evProdEsperada = computed(() => {
  if (evMediana.value <= 0) return 0
  return evForm.tOperativoMin / evMediana.value
})
const evProdBuena = computed(() => Math.max(0, evProdReal.value - evForm.unidadesDefectuosas))

const evDisponibilidad = computed(() => {
  if (evForm.tDisponibleMin <= 0) return 0
  return Math.min(1, evForm.tOperativoMin / evForm.tDisponibleMin)
})
const evRendimiento = computed(() => {
  if (evProdEsperada.value <= 0) return 0
  return Math.min(1, evProdReal.value / evProdEsperada.value)
})
const evCalidad = computed(() => {
  if (evProdReal.value <= 0) return 0
  return Math.min(1, evProdBuena.value / evProdReal.value)
})
const evOee = computed(() => evDisponibilidad.value * evRendimiento.value * evCalidad.value)

function evEstadoLabel(e) {
  if (e === 'normal') return 'Normal'
  if (e === 'lenta')  return 'Lenta'
  if (e === 'parada') return 'Parada'
  return '—'
}

function fmtMin(m) {
  const n = +m
  if (isNaN(n)) return '—'
  if (n < 1 && n > 0) return `${Math.round(n * 60)}s`
  const h = Math.floor(n / 60)
  const r = Math.round((n % 60) * 10) / 10
  if (h > 0) return r === 0 ? `${h}h` : `${h}h ${Math.round(r)}m`
  return n % 1 === 0 ? `${n} min` : `${n.toFixed(1)} min`
}

const evBarSegments = computed(() => {
  const tDisp = evForm.tDisponibleMin || 1
  const tNoOp = Math.max(0, evForm.tDisponibleMin - evForm.tOperativoMin)
  const valid  = evCiclosClasif.value.filter(c => c.estado !== 'invalido')
  const tNormProd = evNNormal.value * evMediana.value
  const tPerdVel  = valid.filter(c => c.estado === 'lenta')
    .reduce((s, c) => s + (c.v - evMediana.value), 0)
  const tParadas = valid.filter(c => c.estado === 'parada')
    .reduce((s, c) => s + c.v, 0)
  return [
    { label: 'Producción nominal', color: '#10b981', val: tNormProd },
    { label: 'Pérdida velocidad', color: '#84cc16', val: tPerdVel  },
    { label: 'Paradas/Micro',     color: '#fbbf24', val: tParadas  },
    { label: 'No operativo',      color: '#475569', val: tNoOp     },
  ].map(s => ({ ...s, pct: (s.val / tDisp * 100).toFixed(1) }))
})
</script>

<template>
  <div class="lab-wrap">

    <!-- ── Cabecera ── -->
    <div class="lab-header">
      <div class="lab-header-left">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"/>
        </svg>
        <span>LABORATORIO OEE</span>
      </div>
      <div class="lab-header-right">
        <span class="lab-std-badge">ISO 22400</span>
      </div>
    </div>

    <!-- ── Barra de escenario ── -->
    <div class="lab-scenario-bar">
      <input
        v-model="scenarioNombre"
        class="lab-inp lab-inp-nombre"
        placeholder="Nombre del escenario..."
        maxlength="200"
      />

      <div class="lab-scenario-selects">
        <select v-model="plantaID" class="lab-sel">
          <option value="">Planta...</option>
          <option v-for="p in plantas" :key="p.id" :value="p.id">{{ p.nombre }}</option>
        </select>
        <select v-model="lineaID" class="lab-sel" :disabled="!plantaID">
          <option value="">Línea...</option>
          <option v-for="l in lineas" :key="l.id" :value="l.id">{{ l.nombre }}</option>
        </select>
      </div>

      <input
        v-model="scenarioNotas"
        class="lab-inp lab-inp-notas"
        placeholder="Notas opcionales..."
        maxlength="500"
      />

      <div class="lab-scenario-actions">
        <span v-if="saveMsg" :class="['lab-msg', saveMsgType === 'ok' ? 'lab-msg-ok' : 'lab-msg-err']">
          {{ saveMsg }}
        </span>
        <button class="lab-btn lab-btn-ghost" @click="nuevaSesion">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/>
          </svg>
          Nueva
        </button>
        <button class="lab-btn lab-btn-primary" :disabled="saving" @click="guardarSesion">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
          </svg>
          {{ saving ? 'Guardando...' : 'Guardar sesión' }}
        </button>
      </div>
    </div>

    <!-- ── Pestañas ── -->
    <div class="lab-tabs">
      <button :class="['lab-tab', { active: activeTab === 'iso22400' }]" @click="activeTab = 'iso22400'">ISO 22400</button>
      <button :class="['lab-tab', { active: activeTab === 'eventos' }]" @click="activeTab = 'eventos'">Por Eventos</button>
    </div>

    <!-- ── Espacio de trabajo: inputs + resultados ── -->
    <div class="lab-workspace" v-show="activeTab === 'iso22400'">

      <!-- ─ Panel izquierdo: entradas ─ -->
      <div class="lab-panel lab-panel-inputs">

        <!-- Bloque 1: Tiempo Disponible -->
        <div class="lab-block">
          <div class="lab-block-title">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path stroke-linecap="round" d="M12 6v6l4 2"/>
            </svg>
            Tiempo Disponible
          </div>
          <div class="lab-formula">= Disponible Total &minus; Parada Obligatoria</div>

          <div class="lab-slider-row">
            <span class="lab-slider-label">Disponible Total</span>
            <div class="lab-slider-track">
              <input type="range" v-model.number="form.tTurnoS"
                min="0" max="86400" step="900" class="lab-slider"/>
              <input type="number" v-model.number="tTurnoMin"
                min="0" max="1440" step="1" class="lab-slider-num-inp"/>
              <span class="lab-slider-unit">min</span>
            </div>
          </div>

          <div class="lab-sub-header">Parada Obligatoria = Refrigerio + Capacitación + Mantenimiento</div>

          <div class="lab-slider-row">
            <span class="lab-slider-label">
              <span class="lab-dot" style="background:#64748b"></span>
              Refrigerio
            </span>
            <div class="lab-slider-track">
              <input type="range" v-model.number="form.tRefrigerioS"
                min="0" :max="form.tTurnoS" step="60" class="lab-slider lab-slider-oblig"/>
              <input type="number" v-model.number="tRefrigerioMin"
                min="0" step="1" class="lab-slider-num-inp"/>
              <span class="lab-slider-unit">min</span>
            </div>
          </div>

          <div class="lab-slider-row">
            <span class="lab-slider-label">
              <span class="lab-dot" style="background:#64748b"></span>
              Capacitación
            </span>
            <div class="lab-slider-track">
              <input type="range" v-model.number="form.tCapacitacionS"
                min="0" :max="form.tTurnoS" step="60" class="lab-slider lab-slider-oblig"/>
              <input type="number" v-model.number="tCapacitacionMin"
                min="0" step="1" class="lab-slider-num-inp"/>
              <span class="lab-slider-unit">min</span>
            </div>
          </div>

          <div class="lab-slider-row">
            <span class="lab-slider-label">
              <span class="lab-dot" style="background:#64748b"></span>
              Mantenimiento
            </span>
            <div class="lab-slider-track">
              <input type="range" v-model.number="form.tMantenimientoS"
                min="0" :max="form.tTurnoS" step="60" class="lab-slider lab-slider-oblig"/>
              <input type="number" v-model.number="tMantenimientoMin"
                min="0" step="1" class="lab-slider-num-inp"/>
              <span class="lab-slider-unit">min</span>
            </div>
          </div>

          <div class="lab-derived">
            <span class="lab-derived-label">Parada Obligatoria</span>
            <span class="lab-derived-val">{{ fmtSeg(tParadaObligatoria) }}</span>
          </div>
          <div class="lab-derived lab-derived-result">
            <span class="lab-derived-label">Tiempo Disponible</span>
            <span class="lab-derived-val">{{ fmtSeg(tDisponible) }}</span>
          </div>
        </div>

        <!-- Bloque 2: Disponibilidad -->
        <div class="lab-block">
          <div class="lab-block-title">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2"/>
              <path stroke-linecap="round" d="M9 9h6M9 15h6"/>
            </svg>
            Disponibilidad
          </div>
          <div class="lab-formula">= Tiempo Operativo / Tiempo Disponible</div>
          <div class="lab-sub-header">Tiempo Operativo = Tiempo Disponible &minus; Parada No Obligatoria</div>

          <div class="lab-slider-row">
            <span class="lab-slider-label">
              <span class="lab-dot" style="background:#f97316"></span>
              Paradas Programadas
            </span>
            <div class="lab-slider-track">
              <input type="range" v-model.number="form.tParadasProgramadasS"
                min="0" :max="tDisponible" step="60" class="lab-slider lab-slider-parada"/>
              <input type="number" v-model.number="tProgramadasMin"
                min="0" step="1" class="lab-slider-num-inp"/>
              <span class="lab-slider-unit">min</span>
            </div>
          </div>

          <div class="lab-derived">
            <span class="lab-derived-label">Tiempo Operativo</span>
            <span class="lab-derived-val">{{ fmtSeg(tOperativo) }}</span>
          </div>
          <div class="lab-derived lab-derived-result">
            <span class="lab-derived-label">Disponibilidad</span>
            <span class="lab-derived-val" :style="{ color: gcolor(disponibilidad) }">{{ fp(disponibilidad) }}</span>
          </div>
        </div>

        <!-- Bloque 3: Rendimiento -->
        <div class="lab-block">
          <div class="lab-block-title">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/>
            </svg>
            Rendimiento
          </div>
          <div class="lab-formula">= (T.Operativo &minus; Microparadas &minus; T.Baja Vel.) / T.Operativo</div>

          <div class="lab-slider-row">
            <span class="lab-slider-label">
              <span class="lab-dot" style="background:#fbbf24"></span>
              Microparadas
            </span>
            <div class="lab-slider-track">
              <input type="range" v-model.number="form.tMicroparadasS"
                min="0" :max="tOperativo" step="60" class="lab-slider lab-slider-micro"/>
              <input type="number" v-model.number="tMicroMin"
                min="0" step="1" class="lab-slider-num-inp"/>
              <span class="lab-slider-unit">min</span>
            </div>
          </div>

          <div class="lab-field-row">
            <span class="lab-field-label">Velocidad nominal</span>
            <div class="lab-vel-group">
              <input type="number" min="0" step="0.01"
                v-model.number="form.velocidadNominal" class="lab-inp-num"/>
              <div class="lab-unit-btns">
                <button v-for="u in UNIDADES_VEL" :key="u.key"
                  :class="['lab-unit-btn', { active: form.unidadVel === u.key }]"
                  @click="form.unidadVel = u.key">{{ u.label }}</button>
              </div>
            </div>
          </div>

          <div class="lab-field-row">
            <span class="lab-field-label">Unidades producidas</span>
            <input type="number" min="0" step="1"
              v-model.number="form.unidadesProducidas" class="lab-inp-num"/>
          </div>

          <div class="lab-chain">
            <div class="lab-chain-row">
              <span class="lab-chain-label">T.Neto = T.Operativo &minus; Microparadas</span>
              <span class="lab-chain-val">{{ fmtSeg(tNeto) }}</span>
            </div>
            <div class="lab-chain-row">
              <span class="lab-chain-label">T.Nominal = Producción / Vel.Nominal</span>
              <span class="lab-chain-val">{{ fmtSeg(tNominalProd) }}</span>
            </div>
            <div class="lab-chain-row">
              <span class="lab-chain-label">T.Baja Vel. = max(0, T.Neto &minus; T.Nominal)</span>
              <span class="lab-chain-val">{{ fmtSeg(tBajaVelocidad) }}</span>
            </div>
          </div>
          <div class="lab-derived lab-derived-result">
            <span class="lab-derived-label">Rendimiento</span>
            <span class="lab-derived-val" :style="{ color: gcolor(rendimiento) }">{{ fp(rendimiento) }}</span>
          </div>
        </div>

        <!-- Bloque 4: Calidad -->
        <div class="lab-block">
          <div class="lab-block-title">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round"
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            Calidad
          </div>
          <div class="lab-formula">= (Producción &minus; Merma) / Producción</div>

          <div class="lab-field-row">
            <span class="lab-field-label">Merma (unidades defectuosas)</span>
            <input type="number" min="0" step="1"
              v-model.number="form.unidadesDefectuosas" class="lab-inp-num"/>
          </div>

          <div class="lab-derived">
            <span class="lab-derived-label">Producción &minus; Merma</span>
            <span class="lab-derived-val">{{ (form.unidadesProducidas - form.unidadesDefectuosas).toLocaleString('es-AR') }} u</span>
          </div>
          <div class="lab-derived lab-derived-result">
            <span class="lab-derived-label">Calidad</span>
            <span class="lab-derived-val" :style="{ color: gcolor(calidad) }">{{ fp(calidad) }}</span>
          </div>
        </div>

      </div>

      <!-- ─ Panel derecho: resultados ─ -->
      <div class="lab-panel lab-panel-results">

        <!-- OEE principal -->
        <div class="lab-oee-main" :style="{ borderColor: gcolor(oee), color: gcolor(oee) }">
          <div class="lab-oee-val">{{ fp(oee) }}</div>
          <div class="lab-oee-label">OEE</div>
        </div>

        <!-- 3 indicadores circulares -->
        <div class="lab-gauges">
          <div v-for="(item, idx) in [
            { val: disponibilidad, label: 'Disponibilidad' },
            { val: rendimiento,    label: 'Rendimiento'    },
            { val: calidad,        label: 'Calidad'        },
          ]" :key="idx" class="lab-gauge-wrap">
            <svg viewBox="0 0 100 100" class="lab-gauge-svg">
              <circle cx="50" cy="50" r="40" fill="none"
                stroke-width="8" class="lab-gauge-track"/>
              <circle cx="50" cy="50" r="40" fill="none"
                stroke-width="8"
                stroke-linecap="round"
                :stroke="gcolor(item.val)"
                :stroke-dasharray="`${CIRCUM} ${CIRCUM}`"
                :stroke-dashoffset="gaugeOffset(item.val)"
                transform="rotate(-90 50 50)"/>
              <text x="50" y="46" text-anchor="middle" font-size="18"
                font-weight="700" font-family="-apple-system,sans-serif"
                :fill="gcolor(item.val)">
                {{ Math.round(item.val * 100) }}%
              </text>
              <text x="50" y="64" text-anchor="middle" font-size="8.5"
                font-family="-apple-system,sans-serif" fill="#94a3b8">
                {{ item.label }}
              </text>
            </svg>
          </div>
        </div>

        <!-- Barra de distribución de tiempo -->
        <div class="lab-bar-section">
          <div class="lab-section-label">Distribución del turno</div>
          <div class="lab-bar">
            <div
              v-for="seg in barSegments" :key="seg.label"
              class="lab-bar-seg"
              :style="{ width: seg.pct + '%', background: seg.color }"
              :title="`${seg.label}: ${fmtSeg(seg.val)} (${seg.pct}%)`"
            />
          </div>
          <table class="lab-legend">
            <tr v-for="seg in barSegments" :key="seg.label">
              <td><span class="lab-dot" :style="{ background: seg.color }"/></td>
              <td class="lab-legend-label">{{ seg.label }}</td>
              <td class="lab-legend-time">{{ fmtSeg(seg.val) }}</td>
              <td class="lab-legend-pct">{{ seg.pct }}%</td>
            </tr>
          </table>
        </div>

        <!-- Análisis automático -->
        <div v-if="analisis.length > 0" class="lab-analysis">
          <div class="lab-section-label">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path stroke-linecap="round" d="M12 16v-4M12 8h.01"/>
            </svg>
            Análisis
          </div>
          <ul class="lab-analysis-list">
            <li v-for="(line, i) in analisis" :key="i">{{ line }}</li>
          </ul>
        </div>

      </div>
    </div>

    <!-- ── Historial de sesiones ── -->
    <div class="lab-history" v-show="activeTab === 'iso22400'">
      <div class="lab-history-header">
        <span class="lab-section-label">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          Historial de sesiones
        </span>
        <button class="lab-btn lab-btn-ghost" @click="cargarHistorial" :disabled="histLoading">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
          </svg>
          Actualizar
        </button>
      </div>

      <div v-if="histLoading" class="lab-empty">Cargando...</div>
      <div v-else-if="historial.length === 0" class="lab-empty">
        No hay sesiones guardadas. Configurá los parámetros y guardá un escenario.
      </div>

      <table v-else class="lab-table">
        <thead>
          <tr>
            <th>Nombre</th>
            <th>Línea</th>
            <th>OEE</th>
            <th>Disp.</th>
            <th>Rend.</th>
            <th>Cal.</th>
            <th>Unid. producidas</th>
            <th>Fecha</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in historial" :key="s.id">
            <td class="lab-td-nombre">{{ s.nombre }}</td>
            <td class="lab-td-num">{{ s.linea_id ?? '—' }}</td>
            <td>
              <span class="lab-oee-badge" :style="{ background: gcolor(s.results?.oee ?? 0) + '22', color: gcolor(s.results?.oee ?? 0) }">
                {{ fp(s.results?.oee ?? 0) }}
              </span>
            </td>
            <td class="lab-td-pct">{{ fp(s.results?.disponibilidad ?? 0) }}</td>
            <td class="lab-td-pct">{{ fp(s.results?.rendimiento ?? 0) }}</td>
            <td class="lab-td-pct">{{ fp(s.results?.calidad ?? 0) }}</td>
            <td class="lab-td-num">{{ (s.inputs?.unidades_producidas ?? 0).toLocaleString('es-AR') }}</td>
            <td class="lab-td-fecha">{{ fmtFecha(s.created_at) }}</td>
            <td class="lab-td-actions">
              <button class="lab-btn-icon" title="Cargar en el laboratorio" @click="cargarSesion(s)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round"
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
                </svg>
              </button>
              <button class="lab-btn-icon lab-btn-del" title="Eliminar sesión" @click="eliminarSesion(s.id)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- ── TAB: Por Eventos ── -->
    <div v-show="activeTab === 'eventos'">
      <div class="lab-workspace">

        <!-- Panel izquierdo -->
        <div class="lab-panel lab-panel-inputs">

          <!-- Bloque 1: Turno -->
          <div class="lab-block">
            <div class="lab-block-title">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><path stroke-linecap="round" d="M12 6v6l4 2"/>
              </svg>
              Parámetros del Turno
            </div>
            <div class="lab-formula">Disponibilidad (A) = T.Operativo / T.Disponible</div>
            <div class="ev-field">
              <span class="ev-field-label">Tiempo Disponible</span>
              <div class="ev-inp-row">
                <input type="number" v-model.number="evForm.tDisponibleMin" min="1" step="1" class="lab-inp-num"/>
                <span class="lab-slider-unit">min</span>
              </div>
            </div>
            <div class="ev-field">
              <span class="ev-field-label">Tiempo Operativo</span>
              <div class="ev-inp-row">
                <input type="number" v-model.number="evForm.tOperativoMin" min="0" step="1" class="lab-inp-num"/>
                <span class="lab-slider-unit">min</span>
              </div>
            </div>
            <div class="lab-derived lab-derived-result">
              <span class="lab-derived-label">Disponibilidad (A)</span>
              <span class="lab-derived-val" :style="{ color: gcolor(evDisponibilidad) }">{{ fp(evDisponibilidad) }}</span>
            </div>
          </div>

          <!-- Bloque 2: Umbrales estadísticos -->
          <div class="lab-block">
            <div class="lab-block-title">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
              </svg>
              Umbrales Estadísticos
            </div>
            <div class="lab-formula">T_límite = T̃ + n&middot;σ &nbsp;&nbsp; T_parada = m × T̃</div>
            <div class="ev-field">
              <span class="ev-field-label">Multiplicador σ (n)</span>
              <input type="number" v-model.number="evForm.multSigma" min="0.5" max="5" step="0.5" class="lab-inp-num"/>
            </div>
            <div class="ev-field">
              <span class="ev-field-label">Multiplicador parada (m)</span>
              <input type="number" v-model.number="evForm.multParada" min="1" max="10" step="0.5" class="lab-inp-num"/>
            </div>
            <div class="ev-stats-grid">
              <div class="ev-stat"><span class="ev-stat-lbl">Mediana (T̃)</span><span class="ev-stat-v">{{ evMediana.toFixed(2) }} min</span></div>
              <div class="ev-stat"><span class="ev-stat-lbl">σ</span><span class="ev-stat-v">{{ evSigma.toFixed(2) }} min</span></div>
              <div class="ev-stat ev-stat-warn"><span class="ev-stat-lbl">T_límite</span><span class="ev-stat-v">{{ evTLimite.toFixed(2) }} min</span></div>
              <div class="ev-stat ev-stat-danger"><span class="ev-stat-lbl">T_parada</span><span class="ev-stat-v">{{ evTParada.toFixed(2) }} min</span></div>
            </div>
            <div class="ev-clasif-row">
              <span class="ev-badge ev-badge-normal">{{ evNNormal }} Normal</span>
              <span class="ev-badge ev-badge-lenta">{{ evNLenta }} Lenta</span>
              <span class="ev-badge ev-badge-parada">{{ evNParada }} Parada</span>
            </div>
          </div>

          <!-- Bloque 3: Tabla Excel de ciclos -->
          <div class="lab-block">
            <div class="lab-block-title">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2"/>
                <path stroke-linecap="round" d="M3 9h18M3 15h18M9 3v18"/>
              </svg>
              Tiempos de Ciclo
            </div>
            <div class="lab-formula">T_ciclo,i = t_i − t_(i−1)</div>
            <div class="ev-xls-wrap">
              <div class="ev-xls-head">
                <span class="ev-xh-n">#</span>
                <span class="ev-xh-ciclo">T.ciclo (min)</span>
                <span class="ev-xh-estado">Estado</span>
                <span></span>
              </div>
              <div v-for="(c, i) in ciclos" :key="i" class="ev-xls-row">
                <span class="ev-xc-n">{{ i + 1 }}</span>
                <div class="ev-xc-ciclo">
                  <input type="number" v-model.number="c.v" min="0.1" step="0.1" class="ev-cell-inp"/>
                </div>
                <div class="ev-xc-estado">
                  <span :class="['ev-badge', `ev-badge-${evCiclosClasif[i]?.estado ?? 'invalido'}`]">
                    {{ evEstadoLabel(evCiclosClasif[i]?.estado) }}
                  </span>
                </div>
                <button class="ev-del-row" @click="evRemoveCiclo(i)" :disabled="ciclos.length <= 1">×</button>
              </div>
            </div>
            <div class="ev-xls-actions">
              <button class="lab-btn lab-btn-ghost" @click="evAddCiclo">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" d="M12 4v16m8-8H4"/>
                </svg>
                Agregar fila
              </button>
              <button class="lab-btn lab-btn-ghost" @click="evResetCiclos">Restablecer</button>
            </div>
          </div>

          <!-- Bloque 4: Rendimiento y Calidad -->
          <div class="lab-block">
            <div class="lab-block-title">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/>
              </svg>
              Rendimiento y Calidad
            </div>
            <div class="lab-formula">P = Prod.real / Prod.esperada &nbsp;&nbsp; Q = Buena / Total</div>
            <div class="lab-chain">
              <div class="lab-chain-row">
                <span class="lab-chain-label">Prod. esperada = T.Operativo / T̃ciclo</span>
                <span class="lab-chain-val">{{ evProdEsperada.toFixed(1) }} u</span>
              </div>
              <div class="lab-chain-row">
                <span class="lab-chain-label">Prod. real = eventos registrados</span>
                <span class="lab-chain-val">{{ evProdReal }} u</span>
              </div>
            </div>
            <div class="ev-field">
              <span class="ev-field-label">Unidades defectuosas</span>
              <input type="number" v-model.number="evForm.unidadesDefectuosas" min="0" step="1" class="lab-inp-num"/>
            </div>
            <div class="lab-derived">
              <span class="lab-derived-label">Producción buena</span>
              <span class="lab-derived-val">{{ evProdBuena }} u</span>
            </div>
            <div class="lab-derived lab-derived-result">
              <span class="lab-derived-label">Rendimiento (P)</span>
              <span class="lab-derived-val" :style="{ color: gcolor(evRendimiento) }">{{ fp(evRendimiento) }}</span>
            </div>
            <div class="lab-derived lab-derived-result">
              <span class="lab-derived-label">Calidad (Q)</span>
              <span class="lab-derived-val" :style="{ color: gcolor(evCalidad) }">{{ fp(evCalidad) }}</span>
            </div>
          </div>

        </div>

        <!-- Panel derecho: resultados -->
        <div class="lab-panel lab-panel-results">

          <div class="lab-oee-main" :style="{ borderColor: gcolor(evOee), color: gcolor(evOee) }">
            <div class="lab-oee-val">{{ fp(evOee) }}</div>
            <div class="lab-oee-label">OEE = A × P × Q</div>
          </div>

          <div class="lab-gauges">
            <div v-for="(item, idx) in [
              { val: evDisponibilidad, label: 'Disponibilidad (A)' },
              { val: evRendimiento,    label: 'Rendimiento (P)'    },
              { val: evCalidad,        label: 'Calidad (Q)'        },
            ]" :key="idx" class="lab-gauge-wrap">
              <svg viewBox="0 0 100 100" class="lab-gauge-svg">
                <circle cx="50" cy="50" r="40" fill="none" stroke-width="8" class="lab-gauge-track"/>
                <circle cx="50" cy="50" r="40" fill="none" stroke-width="8" stroke-linecap="round"
                  :stroke="gcolor(item.val)"
                  :stroke-dasharray="`${CIRCUM} ${CIRCUM}`"
                  :stroke-dashoffset="gaugeOffset(item.val)"
                  transform="rotate(-90 50 50)"/>
                <text x="50" y="46" text-anchor="middle" font-size="18" font-weight="700"
                  font-family="-apple-system,sans-serif" :fill="gcolor(item.val)">
                  {{ Math.round(item.val * 100) }}%
                </text>
                <text x="50" y="64" text-anchor="middle" font-size="7.5"
                  font-family="-apple-system,sans-serif" fill="#94a3b8">
                  {{ item.label }}
                </text>
              </svg>
            </div>
          </div>

          <!-- Estadísticas del modelo -->
          <div class="lab-bar-section">
            <div class="lab-section-label">Estadísticas del modelo</div>
            <table class="ev-formula-table">
              <tr>
                <td class="ev-ft-lbl">Mediana (T̃ciclo)</td>
                <td class="ev-ft-formula">median(T_ciclo)</td>
                <td class="ev-ft-val">{{ evMediana.toFixed(2) }} min</td>
              </tr>
              <tr>
                <td class="ev-ft-lbl">Desv. estándar (σ)</td>
                <td class="ev-ft-formula">std(T_ciclo)</td>
                <td class="ev-ft-val">{{ evSigma.toFixed(2) }} min</td>
              </tr>
              <tr>
                <td class="ev-ft-lbl">T límite variación</td>
                <td class="ev-ft-formula">T̃ + {{ evForm.multSigma }}·σ</td>
                <td class="ev-ft-val ev-ft-warn">{{ evTLimite.toFixed(2) }} min</td>
              </tr>
              <tr>
                <td class="ev-ft-lbl">T umbral parada</td>
                <td class="ev-ft-formula">{{ evForm.multParada }} × T̃</td>
                <td class="ev-ft-val ev-ft-danger">{{ evTParada.toFixed(2) }} min</td>
              </tr>
              <tr>
                <td class="ev-ft-lbl">Prod. esperada</td>
                <td class="ev-ft-formula">T.Op / T̃ciclo</td>
                <td class="ev-ft-val">{{ evProdEsperada.toFixed(1) }} u</td>
              </tr>
              <tr>
                <td class="ev-ft-lbl">Prod. real</td>
                <td class="ev-ft-formula">count(eventos)</td>
                <td class="ev-ft-val">{{ evProdReal }} u</td>
              </tr>
            </table>
          </div>

          <!-- Barra de distribución -->
          <div class="lab-bar-section">
            <div class="lab-section-label">Distribución estimada del turno</div>
            <div class="lab-bar">
              <div v-for="seg in evBarSegments" :key="seg.label"
                class="lab-bar-seg"
                :style="{ width: seg.pct + '%', background: seg.color }"
                :title="`${seg.label}: ${fmtMin(seg.val)} (${seg.pct}%)`"
              />
            </div>
            <table class="lab-legend">
              <tr v-for="seg in evBarSegments" :key="seg.label">
                <td><span class="lab-dot" :style="{ background: seg.color }"/></td>
                <td class="lab-legend-label">{{ seg.label }}</td>
                <td class="lab-legend-time">{{ fmtMin(seg.val) }}</td>
                <td class="lab-legend-pct">{{ seg.pct }}%</td>
              </tr>
            </table>
          </div>

        </div>

      </div>
    </div>

  </div>
</template>

<style scoped>
.lab-wrap {
  padding: 0 0 40px;
  min-height: 100%;
  background: var(--lab-bg, #f1f5f9);
  color: var(--lab-text, #0f172a);
}

/* Cabecera */
.lab-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 24px;
  background: var(--lab-surface, #ffffff);
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
}
.lab-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--lab-text, #0f172a);
}
.lab-header-left svg { color: #0047d1; }
.lab-std-badge {
  font-size: 11px;
  font-weight: 600;
  color: #64748b;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  padding: 2px 8px;
}

/* Barra de escenario */
.lab-scenario-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 24px;
  background: var(--lab-surface, #ffffff);
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
  flex-wrap: wrap;
}
.lab-inp {
  height: 32px;
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
  padding: 0 10px;
  font-size: 13px;
  background: var(--lab-input-bg, #f8fafc);
  color: var(--lab-text, #0f172a);
  outline: none;
  transition: border-color .15s;
}
.lab-inp:focus { border-color: #0047d1; }
.lab-inp-nombre { flex: 1; min-width: 180px; }
.lab-inp-notas  { flex: 1.5; min-width: 160px; }

.lab-scenario-selects { display: flex; gap: 6px; }
.lab-sel {
  height: 32px;
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
  padding: 0 8px;
  font-size: 13px;
  background: var(--lab-input-bg, #f8fafc);
  color: var(--lab-text, #0f172a);
  cursor: pointer;
}
.lab-sel:disabled { opacity: 0.45; cursor: not-allowed; }

.lab-scenario-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}
.lab-msg {
  font-size: 12px;
  font-weight: 500;
  padding: 4px 10px;
  border-radius: 4px;
}
.lab-msg-ok  { background: #d1fae5; color: #065f46; }
.lab-msg-err { background: #fee2e2; color: #991b1b; }

/* Botones */
.lab-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 32px;
  padding: 0 14px;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity .15s, background .15s;
}
.lab-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.lab-btn-primary {
  background: #0047d1;
  color: #fff;
}
.lab-btn-primary:hover:not(:disabled) { background: #002a6b; }
.lab-btn-ghost {
  background: var(--lab-surface, #fff);
  color: var(--lab-text, #0f172a);
  border: 1px solid var(--lab-border, #e2e8f0);
}
.lab-btn-ghost:hover:not(:disabled) { background: var(--lab-hover, #f1f5f9); }

/* Workspace 2 columnas */
.lab-workspace {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 16px;
  padding: 16px 24px 0;
  align-items: start;
}

/* Paneles */
.lab-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Bloques de entrada */
.lab-block {
  background: var(--lab-surface, #ffffff);
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.lab-block-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  color: #64748b;
  text-transform: uppercase;
}

/* Filas de slider */
.lab-slider-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.lab-slider-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--lab-text-muted, #475569);
}
.lab-slider-track {
  display: flex;
  align-items: center;
  gap: 10px;
}
.lab-slider {
  flex: 1;
  height: 4px;
  accent-color: #0047d1;
  cursor: pointer;
}
.lab-slider-micro  { accent-color: #fbbf24; }
.lab-slider-parada { accent-color: #f97316; }
.lab-slider-mayor  { accent-color: #ef4444; }
.lab-slider-num-inp {
  width: 54px;
  height: 26px;
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 5px;
  padding: 0 5px;
  font-size: 12px;
  font-weight: 600;
  text-align: right;
  background: var(--lab-input-bg, #f8fafc);
  color: var(--lab-text, #0f172a);
  outline: none;
  flex-shrink: 0;
}
.lab-slider-num-inp:focus { border-color: #0047d1; }
.lab-slider-unit {
  font-size: 11px;
  color: #94a3b8;
  flex-shrink: 0;
}

/* Etiqueta de fórmula en bloque */
.lab-formula {
  font-size: 11px;
  font-family: ui-monospace, 'Cascadia Code', monospace;
  color: #0047d1;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 5px;
  padding: 5px 9px;
  line-height: 1.4;
}
html.dark .lab-formula {
  background: #1e3a5f22;
  border-color: #1d4ed855;
  color: #60a5fa;
}

/* Sub-encabezado de grupo */
.lab-sub-header {
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: #94a3b8;
  text-transform: uppercase;
  padding: 2px 0 4px;
  border-bottom: 1px dashed var(--lab-border, #e2e8f0);
}

/* Derivado resaltado (resultado del bloque) */
.lab-derived-result {
  border-color: #bfdbfe;
  background: #eff6ff;
}
.lab-derived-result .lab-derived-label { color: #3b82f6; }
html.dark .lab-derived-result {
  border-color: #1d4ed855;
  background: #1e3a5f22;
}

/* Cadena de derivados intermedios */
.lab-chain {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 6px 8px;
  background: var(--lab-derived-bg, #f8fafc);
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
}
.lab-chain-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.lab-chain-label {
  font-size: 10.5px;
  color: #94a3b8;
  font-family: ui-monospace, 'Cascadia Code', monospace;
  flex: 1;
}
.lab-chain-val {
  font-size: 11.5px;
  font-weight: 700;
  color: var(--lab-text, #0f172a);
  white-space: nowrap;
}

/* Slider color para paradas obligatorias */
.lab-slider-oblig { accent-color: #64748b; }

/* Filas de número */
.lab-field-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.lab-field-label {
  font-size: 12.5px;
  color: var(--lab-text-muted, #475569);
  flex: 1;
}
.lab-inp-num {
  width: 100px;
  height: 30px;
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
  padding: 0 8px;
  font-size: 13px;
  text-align: right;
  background: var(--lab-input-bg, #f8fafc);
  color: var(--lab-text, #0f172a);
  outline: none;
}
.lab-inp-num:focus { border-color: #0047d1; }

/* Grupo velocidad + unidad */
.lab-vel-group {
  display: flex;
  align-items: center;
  gap: 6px;
}
.lab-unit-btns {
  display: flex;
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
  overflow: hidden;
}
.lab-unit-btn {
  padding: 4px 9px;
  font-size: 11px;
  font-weight: 600;
  background: var(--lab-input-bg, #f8fafc);
  color: var(--lab-text-muted, #64748b);
  border: none;
  cursor: pointer;
  transition: background .12s, color .12s;
}
.lab-unit-btn + .lab-unit-btn { border-left: 1px solid var(--lab-border, #e2e8f0); }
.lab-unit-btn.active {
  background: #0047d1;
  color: #fff;
}

/* Filas de derivados */
.lab-derived {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 7px 10px;
  background: var(--lab-derived-bg, #f8fafc);
  border-radius: 6px;
  border: 1px solid var(--lab-border, #e2e8f0);
}
.lab-derived-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.lab-derived-label {
  font-size: 11.5px;
  color: #94a3b8;
}
.lab-derived-val {
  font-size: 13px;
  font-weight: 700;
  color: var(--lab-text, #0f172a);
}

/* Dot */
.lab-dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* Panel de resultados */
.lab-panel-results {
  background: var(--lab-surface, #ffffff);
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 8px;
  padding: 24px;
  gap: 24px;
}

/* OEE principal */
.lab-oee-main {
  text-align: center;
  padding: 20px 32px 16px;
  border: 3px solid;
  border-radius: 12px;
  display: inline-block;
  align-self: center;
  margin: 0 auto;
  min-width: 160px;
  transition: border-color .3s, color .3s;
}
.lab-oee-val {
  font-size: 54px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.02em;
}
.lab-oee-label {
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.12em;
  margin-top: 6px;
  opacity: 0.65;
}

/* Indicadores circulares */
.lab-gauges {
  display: flex;
  justify-content: space-around;
  gap: 12px;
}
.lab-gauge-wrap {
  flex: 1;
  max-width: 110px;
}
.lab-gauge-svg {
  width: 100%;
  height: auto;
}
.lab-gauge-track {
  stroke: #e2e8f0;
}

/* Barra distribución */
.lab-bar-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.lab-section-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: #64748b;
}
.lab-bar {
  display: flex;
  height: 20px;
  border-radius: 4px;
  overflow: hidden;
  gap: 1px;
}
.lab-bar-seg {
  transition: width .4s ease;
}
.lab-legend {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.lab-legend tr { border-bottom: 1px solid var(--lab-border, #e2e8f0); }
.lab-legend tr:last-child { border-bottom: none; }
.lab-legend td { padding: 5px 4px; }
.lab-legend-label {
  color: var(--lab-text-muted, #475569);
  padding-left: 8px;
  width: 100%;
}
.lab-legend-time {
  white-space: nowrap;
  font-weight: 600;
  color: var(--lab-text, #0f172a);
  padding-right: 8px;
  text-align: right;
}
.lab-legend-pct {
  white-space: nowrap;
  color: #94a3b8;
  text-align: right;
  min-width: 44px;
}

/* Análisis */
.lab-analysis {
  background: var(--lab-derived-bg, #f8fafc);
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 8px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.lab-analysis-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.lab-analysis-list li {
  font-size: 12.5px;
  color: var(--lab-text-muted, #475569);
  line-height: 1.5;
  padding-left: 14px;
  position: relative;
}
.lab-analysis-list li::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #94a3b8;
}

/* Historial */
.lab-history {
  margin: 16px 24px 0;
  background: var(--lab-surface, #ffffff);
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 8px;
  overflow: hidden;
}
.lab-history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
}
.lab-empty {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: #94a3b8;
}
.lab-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.lab-table thead tr {
  background: var(--lab-derived-bg, #f8fafc);
}
.lab-table th {
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: #64748b;
  text-align: left;
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
}
.lab-table td {
  padding: 9px 12px;
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
  color: var(--lab-text, #0f172a);
}
.lab-table tbody tr:last-child td { border-bottom: none; }
.lab-table tbody tr:hover { background: var(--lab-hover, #f8fafc); }

.lab-td-nombre  { font-weight: 500; }
.lab-td-num     { font-variant-numeric: tabular-nums; }
.lab-td-pct     { font-variant-numeric: tabular-nums; color: #64748b; }
.lab-td-fecha   { font-size: 12px; color: #94a3b8; white-space: nowrap; }
.lab-td-actions {
  white-space: nowrap;
  display: flex;
  gap: 4px;
  align-items: center;
}

.lab-oee-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 700;
}

.lab-btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  color: var(--lab-text-muted, #64748b);
  transition: background .12s;
}
.lab-btn-icon:hover { background: var(--lab-hover, #f1f5f9); }
.lab-btn-del:hover  { background: #fee2e2; color: #ef4444; border-color: #fca5a5; }

/* ── Pestañas ────────────────────────────────────────────────────────────────── */
.lab-tabs {
  display: flex;
  padding: 0 24px;
  background: var(--lab-surface, #ffffff);
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
}
.lab-tab {
  padding: 10px 20px;
  font-size: 12.5px;
  font-weight: 600;
  border: none;
  background: none;
  cursor: pointer;
  color: var(--lab-text-muted, #64748b);
  border-bottom: 2px solid transparent;
  transition: color .15s, border-color .15s;
  margin-bottom: -1px;
}
.lab-tab:hover { color: var(--lab-text, #0f172a); }
.lab-tab.active { color: #0047d1; border-bottom-color: #0047d1; }

/* ── Tab por eventos: campos genéricos ───────────────────────────────────────── */
.ev-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.ev-field-label {
  font-size: 12.5px;
  color: var(--lab-text-muted, #475569);
  flex: 1;
}
.ev-inp-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Estadísticas 2×2 */
.ev-stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}
.ev-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 7px 9px;
  background: var(--lab-derived-bg, #f8fafc);
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
}
.ev-stat-warn   { border-color: #fde68a; background: #fffbeb; }
.ev-stat-danger { border-color: #fca5a5; background: #fff1f2; }
html.dark .ev-stat-warn   { border-color: #78350f44; background: #78350f22; }
html.dark .ev-stat-danger { border-color: #7f1d1d44; background: #7f1d1d22; }
.ev-stat-lbl { font-size: 10px; color: #94a3b8; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; }
.ev-stat-v   { font-size: 13px; font-weight: 700; color: var(--lab-text, #0f172a); }
.ev-stat-warn   .ev-stat-v  { color: #b45309; }
.ev-stat-danger .ev-stat-v  { color: #b91c1c; }

/* Resumen clasificación */
.ev-clasif-row { display: flex; gap: 6px; flex-wrap: wrap; }
.ev-badge {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}
.ev-badge-normal   { background: #d1fae5; color: #065f46; }
.ev-badge-lenta    { background: #fef9c3; color: #854d0e; }
.ev-badge-parada   { background: #fee2e2; color: #991b1b; }
.ev-badge-invalido { background: #f1f5f9; color: #94a3b8; }
html.dark .ev-badge-normal  { background: #064e3b33; color: #6ee7b7; }
html.dark .ev-badge-lenta   { background: #78350f33; color: #fcd34d; }
html.dark .ev-badge-parada  { background: #7f1d1d33; color: #fca5a5; }

/* Tabla Excel de ciclos */
.ev-xls-wrap {
  border: 1px solid var(--lab-border, #e2e8f0);
  border-radius: 6px;
  overflow: hidden;
  max-height: 300px;
  overflow-y: auto;
}
.ev-xls-head {
  display: grid;
  grid-template-columns: 28px 1fr 88px 26px;
  padding: 5px 8px;
  background: var(--lab-derived-bg, #f8fafc);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: #64748b;
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
  position: sticky;
  top: 0;
  z-index: 1;
}
.ev-xls-row {
  display: grid;
  grid-template-columns: 28px 1fr 88px 26px;
  align-items: center;
  padding: 2px 8px;
  border-bottom: 1px solid var(--lab-border, #e2e8f0);
}
.ev-xls-row:last-child { border-bottom: none; }
.ev-xls-row:hover { background: var(--lab-hover, #f8fafc); }
.ev-xc-n, .ev-xh-n { font-size: 11px; color: #94a3b8; }
.ev-xc-ciclo { padding: 2px 4px; }
.ev-cell-inp {
  width: 100%;
  height: 27px;
  border: 1px solid transparent;
  border-radius: 4px;
  padding: 0 6px;
  font-size: 12.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  text-align: right;
  background: transparent;
  color: var(--lab-text, #0f172a);
  outline: none;
  transition: border-color .12s, background .12s;
}
.ev-cell-inp:focus { border-color: #0047d1; background: var(--lab-input-bg, #f8fafc); }
.ev-xc-estado { display: flex; align-items: center; }
.ev-del-row {
  width: 22px;
  height: 22px;
  border: none;
  background: none;
  color: #94a3b8;
  cursor: pointer;
  font-size: 14px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background .12s, color .12s;
}
.ev-del-row:hover:not(:disabled) { background: #fee2e2; color: #ef4444; }
.ev-del-row:disabled { opacity: 0.3; cursor: not-allowed; }
.ev-xls-actions { display: flex; gap: 6px; padding-top: 4px; }

/* Tabla de fórmulas (panel derecho) */
.ev-formula-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.ev-formula-table tr { border-bottom: 1px solid var(--lab-border, #e2e8f0); }
.ev-formula-table tr:last-child { border-bottom: none; }
.ev-formula-table td { padding: 5px 4px; }
.ev-ft-lbl     { color: var(--lab-text-muted, #475569); width: 42%; }
.ev-ft-formula {
  font-family: ui-monospace, 'Cascadia Code', monospace;
  font-size: 10.5px;
  color: #0047d1;
  width: 33%;
}
html.dark .ev-ft-formula { color: #60a5fa; }
.ev-ft-val     { text-align: right; font-weight: 600; color: var(--lab-text, #0f172a); white-space: nowrap; }
.ev-ft-warn    { color: #b45309; }
.ev-ft-danger  { color: #b91c1c; }
</style>

<style>
html.dark .lab-wrap {
  --lab-bg:         #0f172a;
  --lab-surface:    #1e293b;
  --lab-border:     #334155;
  --lab-text:       #f8fafc;
  --lab-text-muted: #94a3b8;
  --lab-input-bg:   #0f172a;
  --lab-derived-bg: #0f172a;
  --lab-hover:      #1e3a5f;
}
html.dark .lab-gauge-track { stroke: #334155; }
</style>
