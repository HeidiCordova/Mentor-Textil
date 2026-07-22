<script setup>
import { ref, computed, watch } from 'vue'
import { turnoDiaService } from '@/api/services/turnoDia.service'

const props = defineProps({
  plantaId: { type: [Number, String], default: null },
  lineaId:  { type: [Number, String], default: null },
  linea:    { type: Object, default: null }
})

const DIAS    = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado', 'Domingo']
const COLORES = ['#6366f1', '#22c55e', '#f59e0b', '#ef4444', '#06b6d4', '#a855f7', '#ec4899', '#84cc16']

const intervalConfig = computed(() => {
  const mode = props.linea?.mode || 'botellas'
  if (mode === 'textil') {
    return {
      intervalS: 1800,
      intervalMin: 30,
      label: 'textil (intervalos de 30 minutos)',
      description: 'Las líneas textil generan snapshots cada 30 minutos (:00 y :30)',
      validMinutes: [0, 30]
    }
  }
  return {
    intervalS: 300,
    intervalMin: 5,
    label: 'botellas (intervalos de 5 minutos)',
    description: 'Las líneas botellas generan snapshots cada 5 minutos',
    validMinutes: [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]
  }
})

function roundToInterval(horaStr) {
  if (!horaStr || !props.linea) return horaStr
  
  const [hh, mm] = horaStr.split(':').map(Number)
  const validMins = intervalConfig.value.validMinutes
  const closest = validMins.reduce((prev, curr) => 
    Math.abs(curr - mm) < Math.abs(prev - mm) ? curr : prev
  )
  
  return `${String(hh).padStart(2,'0')}:${String(closest).padStart(2,'0')}`
}

function isAlignedToInterval(horaStr) {
  if (!horaStr || !props.linea) return true
  
  const [, mm] = horaStr.split(':').map(Number)
  return intervalConfig.value.validMinutes.includes(mm)
}

function proximoLunes() {
  const d = new Date()
  const dia = d.getDay()
  const diff = dia === 0 ? 1 : (8 - dia)
  d.setDate(d.getDate() + diff)
  return d.toISOString().slice(0, 10)
}
function hoyISO() { return new Date().toISOString().slice(0, 10) }

const modo               = ref('fijo')
const loading            = ref(false)
const saving             = ref(false)
const autoModal          = ref(false)
const autoCount          = ref(3)
const turnosFijo         = ref([])
const diasTurnos         = ref(Object.fromEntries(DIAS.map((_, i) => [i, []])))
const seleccionado       = ref(null)
const renovacionSemanal  = ref(true)
const confirmarModoModal = ref(false)
const modoPendiente      = ref('')
const ultimoGuardado     = ref(null)
const vigenteDesdé       = ref(proximoLunes())
const versiones          = ref([])
const errRetroactivo     = ref('')
const historialPanel     = ref(false)
const historialModalOpen = ref(false)
const historialDetalle   = ref(null)
const loadingHistorial   = ref(false)
const cargaSeq           = ref(0)

const vigenciaEstado = computed(() => {
  if (!vigenteDesdé.value) return 'ok'
  const sel = vigenteDesdé.value
  const hoy = hoyISO()
  if (sel < hoy) return 'pasado'
  if (sel === hoy) return 'hoy'
  return 'futuro'
})

watch(() => [props.plantaId, props.lineaId], cargar, { immediate: true })

async function cargar() {
  const seq = ++cargaSeq.value
  if (!props.plantaId) {
    turnosFijo.value = []
    diasTurnos.value = Object.fromEntries(DIAS.map((_, i) => [i, []]))
    return
  }
  loading.value = true
  try {
    const params = { planta_id: props.plantaId }
    const lineaSolicitada = props.lineaId || null
    if (lineaSolicitada) params.linea_id = lineaSolicitada
    // Pedimos una fecha futura para obtener siempre la configuración más reciente guardada.
    params.fecha = '2999-12-31'
    const resp = await turnoDiaService.getAll(params)
    if (seq !== cargaSeq.value) return
    // Si ya hay línea seleccionada, ignorar respuesta vieja que vino sin linea_id.
    if (!lineaSolicitada && props.lineaId) return
    renovacionSemanal.value = resp?.renovacion_semanal ?? true
    versiones.value = resp?.versiones ?? []
    normalizarCarga(resp?.data ?? [], resp?.vigente_desde ?? '')
  } finally {
    if (seq === cargaSeq.value) loading.value = false
  }
}

function normalizarCarga(data, vigenteDesdeDB) {
  const dias = Object.fromEntries(DIAS.map((_, i) => [i, []]))
  data.forEach((t, gi) => {
    const d = t.dia_semana ?? 0
    dias[d].push({ id: t.id, nombre: t.nombre, hora_inicio: t.hora_inicio?.slice(0,5) ?? '00:00', hora_fin: t.hora_fin?.slice(0,5) ?? '08:00', color: t.color ?? COLORES[gi % COLORES.length], _nuevo: false })
  })
  const esIdentico = DIAS.every((_, i) => {
    if (i === 0) return true
    const sig = a => JSON.stringify(a.map(t => `${t.nombre}|${t.hora_inicio}|${t.hora_fin}`))
    return sig(dias[0]) === sig(dias[i])
  })
  if (data.length === 0 || esIdentico) {
    modo.value = 'fijo'
    turnosFijo.value = dias[0].length ? [...dias[0]] : []
  } else {
    modo.value = 'por-dia'
    diasTurnos.value = dias
  }

  if (data.length > 0) {
    if (modo.value === 'fijo') {
      ultimoGuardado.value = {
        modo: 'fijo',
        modoLabel: 'Fijo (igual todos los días)',
        renovacion: renovacionSemanal.value,
        fromDB: true,
        vigenteDesdé: vigenteDesdeDB,
        turnos: turnosFijo.value.map(t => ({ nombre: t.nombre, hora_inicio: t.hora_inicio, hora_fin: t.hora_fin, color: t.color }))
      }
    } else {
      ultimoGuardado.value = {
        modo: 'por-dia',
        modoLabel: 'Por día (horario independiente por día)',
        renovacion: renovacionSemanal.value,
        fromDB: true,
        vigenteDesdé: vigenteDesdeDB,
        dias: DIAS.map((dia, idx) => ({ dia, lista: (dias[idx] ?? []).map(t => ({ nombre: t.nombre, hora_inicio: t.hora_inicio, hora_fin: t.hora_fin, color: t.color })) }))
      }
    }
  } else {
    ultimoGuardado.value = null
  }
}

function toMin(h) { if (!h) return 0; const [hh,mm] = h.split(':').map(Number); return hh*60+(mm||0) }
function toHora(min) { const h=Math.floor(min/60)%24, m=min%60; return `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}` }
function durMin(t) { const d=(toMin(t.hora_fin)-toMin(t.hora_inicio)+1440)%1440; return d===0?1440:d }

function bloquesDe(lista) {
  return lista.map((t, i) => {
    const start = toMin(t.hora_inicio)
    const dur   = durMin(t)
    const over  = start + dur > 1440
    return {
      ...t,
      _idx:  i,
      left:  (start / 1440) * 100,
      width: over ? ((1440 - start) / 1440) * 100 : (dur / 1440) * 100,
      _wrap: over ? (((start + dur) % 1440) / 1440) * 100 : 0
    }
  })
}
const bloquesFijo = computed(() => bloquesDe(turnosFijo.value))
function bloquesDia(idx) { return bloquesDe(diasTurnos.value[idx] ?? []) }

const hayOverlapFijo = computed(() => {
  const s = [...bloquesFijo.value].sort((a,b) => a.left - b.left)
  return s.some((b,i) => i > 0 && s[i-1].left + s[i-1].width > b.left + 0.1)
})

function esSelFijo(b)      { return seleccionado.value?._modo==='fijo'    && seleccionado.value._idx===b._idx }
function esSelDia(idx, b)  { return seleccionado.value?._modo==='por-dia' && seleccionado.value._dia===idx && seleccionado.value._idx===b._idx }

function seleccionarFijo(b)     { seleccionado.value = { ...b, _modo:'fijo' } }
function seleccionarDia(idx, b) { seleccionado.value = { ...b, _modo:'por-dia', _dia: idx } }

function nuevaTurnoFijo() {
  const t = { nombre:`Turno ${turnosFijo.value.length+1}`, hora_inicio:'06:00', hora_fin:'14:00', color: COLORES[turnosFijo.value.length%COLORES.length], _nuevo:true }
  turnosFijo.value.push(t)
  seleccionado.value = { ...t, _idx: turnosFijo.value.length-1, _modo:'fijo' }
}

function nuevaTurnoDia(idx) {
  const lista = diasTurnos.value[idx]
  const t = { nombre:`Turno ${lista.length+1}`, hora_inicio:'06:00', hora_fin:'14:00', color: COLORES[lista.length%COLORES.length], _nuevo:true }
  lista.push(t)
  seleccionado.value = { ...t, _idx: lista.length-1, _modo:'por-dia', _dia: idx }
}

function aplicarEdicion() {
  if (!seleccionado.value) return
  const s = seleccionado.value
  const lista = s._modo==='fijo' ? turnosFijo.value : diasTurnos.value[s._dia]
  if (lista) lista[s._idx] = { ...lista[s._idx], nombre: s.nombre, hora_inicio: s.hora_inicio, hora_fin: s.hora_fin, color: s.color }
  seleccionado.value = null
}

function eliminar() {
  if (!seleccionado.value) return
  const s = seleccionado.value
  const lista = s._modo==='fijo' ? turnosFijo.value : diasTurnos.value[s._dia]
  if (lista) lista.splice(s._idx, 1)
  seleccionado.value = null
}

function autoGenerarFijo() {
  const n = Math.max(1,Math.min(8,autoCount.value)), mpt = Math.floor(1440/n)
  turnosFijo.value = Array.from({length:n},(_,i) => ({ nombre:`Turno ${i+1}`, hora_inicio:toHora(i*mpt), hora_fin:toHora(((i+1)*mpt)%1440), color:COLORES[i%COLORES.length], _nuevo:true }))
  autoModal.value = false; seleccionado.value = null
}

function autoGenerarDia(idx) {
  const n = Math.max(1,Math.min(8,autoCount.value)), mpt = Math.floor(1440/n)
  diasTurnos.value[idx] = Array.from({length:n},(_,i) => ({ nombre:`Turno ${i+1}`, hora_inicio:toHora(i*mpt), hora_fin:toHora(((i+1)*mpt)%1440), color:COLORES[i%COLORES.length], _nuevo:true }))
}

function copiarDiaAnterior(idx) {
  if (idx===0) return
  diasTurnos.value[idx] = diasTurnos.value[idx-1].map(t => ({ ...t, _nuevo:true, id:null }))
}

function previewChip(i) { const n=Math.max(1,Math.min(8,autoCount.value)),mpt=Math.floor(1440/n); return `${toHora(i*mpt)} – ${toHora(((i+1)*mpt)%1440)}` }

function solicitarCambioModo(nuevoModo, event) {
  if (nuevoModo === modo.value) return
  event?.currentTarget?.blur()
  const hayDatos = modo.value === 'fijo'
    ? turnosFijo.value.length > 0
    : DIAS.some((_, i) => (diasTurnos.value[i] ?? []).length > 0)
  if (hayDatos) {
    modoPendiente.value = nuevoModo
    confirmarModoModal.value = true
  } else {
    modo.value = nuevoModo
    seleccionado.value = null
  }
}

function cancelarCambioModo() {
  confirmarModoModal.value = false
  modoPendiente.value = ''
}

function confirmarCambioModo() {
  if (modoPendiente.value === 'fijo') turnosFijo.value = []
  else diasTurnos.value = Object.fromEntries(DIAS.map((_, i) => [i, []]))
  modo.value = modoPendiente.value
  seleccionado.value = null
  modoPendiente.value = ''
  confirmarModoModal.value = false
}

async function guardar() {
  if (!props.plantaId) return
  if (vigenciaEstado.value === 'pasado') return
  errRetroactivo.value = ''
  
  let turnos = []
  if (modo.value === 'fijo') {
    if (turnosFijo.value.length === 0) {
      errRetroactivo.value = 'Debe configurar al menos un turno antes de guardar'
      return
    }
    DIAS.forEach((_,dia) => turnosFijo.value.forEach(t => { 
      turnos.push({ dia_semana:dia, nombre:t.nombre, hora_inicio:t.hora_inicio, hora_fin:t.hora_fin, color:t.color }) 
    }))
  } else {
    DIAS.forEach((_,dia) => (diasTurnos.value[dia]??[]).forEach(t => { 
      turnos.push({ dia_semana:dia, nombre:t.nombre, hora_inicio:t.hora_inicio, hora_fin:t.hora_fin, color:t.color }) 
    }))
    if (turnos.length === 0) {
      errRetroactivo.value = 'Debe configurar al menos un turno antes de guardar'
      return
    }
  }
  
  saving.value = true
  try {
    const lineaId = props.lineaId ? Number(props.lineaId) : null
    
    const payload = {
      planta_id: Number(props.plantaId),
      linea_id: lineaId,
      vigente_desde: vigenteDesdé.value,
      renovacion_semanal: renovacionSemanal.value,
      turnos
    }
    
    console.log('Guardando turnos:', payload)
    
    const snapModo  = modo.value
    const snapRenov = renovacionSemanal.value
    const snapFijo  = turnosFijo.value.map(t => ({ nombre: t.nombre, hora_inicio: t.hora_inicio, hora_fin: t.hora_fin, color: t.color }))
    const snapDias  = DIAS.map((dia, idx) => ({ dia, lista: (diasTurnos.value[idx] ?? []).map(t => ({ nombre: t.nombre, hora_inicio: t.hora_inicio, hora_fin: t.hora_fin, color: t.color })) }))
    const snapVig   = vigenteDesdé.value
    
    try {
      await turnoDiaService.save(payload)
    } catch (e) {
      const data = e?.response?.data
      console.error('Error guardando turnos:', e?.response?.status, data)
      
      if (data?.error === 'cambio_retroactivo') {
        errRetroactivo.value = data.mensaje || 'Fecha retroactiva bloqueada.'
        return
      }
      
      errRetroactivo.value = data?.mensaje || data?.error || 'Error al guardar: ' + (e?.message || 'Error desconocido')
      return
    }
    await cargar()
    const ahora = new Date()
    const horaStr = `${String(ahora.getHours()).padStart(2,'0')}:${String(ahora.getMinutes()).padStart(2,'0')}`
    if (snapModo === 'fijo') {
      ultimoGuardado.value = { modo: 'fijo', modoLabel: 'Fijo (igual todos los días)', renovacion: snapRenov, fromDB: false, hora: horaStr, vigenteDesdé: snapVig, turnos: snapFijo }
    } else {
      ultimoGuardado.value = { modo: 'por-dia', modoLabel: 'Por día (horario independiente por día)', renovacion: snapRenov, fromDB: false, hora: horaStr, vigenteDesdé: snapVig, dias: snapDias }
    }
    vigenteDesdé.value = proximoLunes()
  } finally {
    saving.value = false
  }
}

async function verHistorialDetalle(vigenteDesde) {
  if (!props.plantaId) return
  loadingHistorial.value = true
  historialModalOpen.value = true
  try {
    const params = { planta_id: props.plantaId, fecha: vigenteDesde }
    if (props.lineaId) params.linea_id = props.lineaId
    const resp = await turnoDiaService.getAll(params)
    const data = resp?.data ?? []
    
    const dias = Object.fromEntries(DIAS.map((_, i) => [i, []]))
    data.forEach(t => {
      const d = t.dia_semana ?? 0
      dias[d].push({
        nombre: t.nombre,
        hora_inicio: t.hora_inicio?.slice(0, 5) ?? '00:00',
        hora_fin: t.hora_fin?.slice(0, 5) ?? '08:00',
        color: t.color ?? '#6366f1',
        creado_en: t.creado_en
      })
    })
    
    const esIdentico = DIAS.every((_, i) => {
      if (i === 0) return true
      const sig = a => JSON.stringify(a.map(t => `${t.nombre}|${t.hora_inicio}|${t.hora_fin}`))
      return sig(dias[0]) === sig(dias[i])
    })
    
    const turnoActualNombre = calcularTurnoActual(dias[0].length ? dias[0] : (dias[1] ?? []))
    
    historialDetalle.value = {
      vigente_desde: vigenteDesde,
      renovacion_semanal: resp?.renovacion_semanal ?? true,
      modo: esIdentico ? 'fijo' : 'por-dia',
      turnos_fijo: dias[0],
      turnos_por_dia: dias,
      total_registros: data.length,
      turno_actual: turnoActualNombre,
      creado_en: data[0]?.creado_en || vigenteDesde,
      activo: vigenteDesde === versiones.value[0]?.vigente_desde
    }
  } catch (e) {
    console.error('Error al cargar historial:', e)
  } finally {
    loadingHistorial.value = false
  }
}

function calcularTurnoActual(turnos) {
  if (!turnos?.length) return '—'
  const now = new Date()
  const hh = now.getHours() * 60 + now.getMinutes()
  const toMin = t => { if (!t) return 0; const [h, m] = t.split(':').map(Number); return h * 60 + m }
  
  for (const t of turnos) {
    const ini = toMin(t.hora_inicio)
    const fin = toMin(t.hora_fin)
    if (ini < fin ? (hh >= ini && hh < fin) : (hh >= ini || hh < fin)) {
      return t.nombre
    }
  }
  return turnos[0]?.nombre || '—'
}

function durMinHist(t) {
  const ini = toMin(t.hora_inicio)
  const fin = toMin(t.hora_fin)
  const d = (fin - ini + 1440) % 1440
  return d === 0 ? 1440 : d
}

function cerrarHistorialModal() {
  historialModalOpen.value = false
  historialDetalle.value = null
}

</script>

<template>
  <div class="tte-wrap">

    <div class="tte-toolbar">
      <div class="tte-modo-toggle">
        <button :class="['tte-modo-btn',{active:modo==='fijo'}]" @click="solicitarCambioModo('fijo', $event)">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="4" rx="2"/></svg>
          Fijo
        </button>
        <button :class="['tte-modo-btn',{active:modo==='por-dia'}]" @click="solicitarCambioModo('por-dia', $event)">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="3" rx="1"/><rect x="3" y="10" width="18" height="3" rx="1"/><rect x="3" y="16" width="18" height="3" rx="1"/></svg>
          Por día
        </button>
      </div>

      <template v-if="modo==='fijo'">
        <button class="tte-btn secondary" @click="autoModal=true">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
          Auto-generar
        </button>
        <button class="tte-btn secondary" @click="nuevaTurnoFijo">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/></svg>
          Añadir turno
        </button>
        <span v-if="hayOverlapFijo" class="tte-warn">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>
          Turnos solapados
        </span>
      </template>

      <button class="tte-btn primary" style="margin-left:auto"
        :disabled="saving || !plantaId || vigenciaEstado==='pasado'"
        :title="vigenciaEstado==='pasado'?'No se puede guardar en una fecha pasada — protege el OEE':undefined"
        @click="guardar">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
        {{ saving ? 'Guardando...' : 'Guardar cambios' }}
      </button>
    </div>

    <div v-if="linea" class="tte-interval-banner">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"></circle>
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 16v-4m0-4h.01"></path>
      </svg>
      <div class="tte-interval-content">
        <strong>Configuración de {{ linea.nombre }}:</strong> Modo <span class="tte-interval-mode">{{ intervalConfig.label }}</span>
        <div class="tte-interval-desc">{{ intervalConfig.description }}</div>
        <div class="tte-interval-hint">
          <strong>Recomendación:</strong> Configure los turnos con horarios que coincidan con estos intervalos 
          (minutos en <strong>{{ intervalConfig.validMinutes.join(', ') }}</strong>) para alinearse con la generación de snapshots de OEE.
        </div>
      </div>
    </div>

    <div class="tte-vigencia-row">
      <div class="tte-vig-group">
        <label class="tte-vig-label">Vigente desde</label>
        <input v-model="vigenteDesdé" type="date" class="tte-input tte-date-input" />
        <span v-if="vigenciaEstado==='pasado'" class="tte-vig-alert danger">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/></svg>
          Fecha pasada — no se puede guardar (protege el OEE)
        </span>
        <span v-else-if="vigenciaEstado==='hoy'" class="tte-vig-alert warn">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>
          Aplica hoy — afectará el cálculo de OEE desde esta fecha
        </span>
        <span v-else class="tte-vig-alert ok">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
          Fecha futura — sin impacto en OEE histórico
        </span>
      </div>
      <button class="tte-btn secondary tte-btn-xs" style="margin-left:auto" @click="historialPanel=!historialPanel">
        <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
        {{ historialPanel ? 'Ocultar historial' : versiones.length > 0 ? `Historial de versiones (${versiones.length})` : 'Ver historial' }}
      </button>
    </div>

    <div v-if="errRetroactivo" class="tte-err-banner">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"/></svg>
      {{ errRetroactivo }}
      <button class="tte-close" @click="errRetroactivo=''"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg></button>
    </div>

    <div v-if="historialPanel" class="tte-historial">
      <div class="tte-historial-head">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
        Historial de configuraciones guardadas
      </div>
      <div v-if="versiones.length === 0" class="tte-historial-empty">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <p>No hay versiones guardadas todavía</p>
        <span>Guarda una configuración de turnos para comenzar a ver el historial</span>
      </div>
      <div v-else class="tte-historial-body">
        <div v-for="(v,i) in versiones" :key="v.vigente_desde" 
          class="tte-historial-row" 
          :class="{active: i===0}"
          @click="verHistorialDetalle(v.vigente_desde)"
          style="cursor: pointer;"
          title="Click para ver detalles">
          <span class="tte-hist-fecha">{{ v.vigente_desde }}</span>
          <span class="tte-hist-turnos">{{ v.total_turnos }} registros</span>
          <span v-if="i===0" class="tte-badge badge-on" style="font-size:10px">activa</span>
          <span v-else class="tte-badge" style="background:#f1f5f9;color:#64748b;font-size:10px">histórica</span>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-left: auto;">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7"/>
          </svg>
        </div>
      </div>
    </div>

    <div class="tte-options-row">
      <span class="tte-options-label">Modo activo:</span>
      <span class="tte-badge" :class="modo==='fijo'?'badge-fijo':'badge-dia'">{{ modo==='fijo' ? 'Fijo (igual todos los días)' : 'Por día (horario independiente por día)' }}</span>
      <span class="tte-options-sep"></span>
      <button class="tte-toggle" :class="{on:renovacionSemanal}" @click="renovacionSemanal=!renovacionSemanal" :title="renovacionSemanal?'Desactivar renovación automática semanal':'Activar renovación automática semanal'">
        <span class="tte-toggle-knob"></span>
      </button>
      <span class="tte-options-label">Renovación semanal automática</span>
      <span v-if="renovacionSemanal" class="tte-badge badge-on">activa</span>
      <span v-else class="tte-badge badge-off">inactiva</span>
    </div>

    <div v-if="ultimoGuardado" class="tte-saved-banner">
      <div class="tte-saved-head">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
        <span v-if="ultimoGuardado.fromDB">Configuración activa en BD</span>
        <span v-else>Guardado a las {{ ultimoGuardado.hora }}</span>
        <span v-if="ultimoGuardado.vigenteDesdé" class="tte-saved-vig">desde {{ ultimoGuardado.vigenteDesdé }}</span>
        <span class="tte-saved-badge badge-fijo" v-if="ultimoGuardado.modo==='fijo'">Fijo</span>
        <span class="tte-saved-badge badge-dia" v-else>Por día</span>
        <span class="tte-saved-badge" :class="ultimoGuardado.renovacion?'badge-on':'badge-off'">Renovación {{ ultimoGuardado.renovacion ? 'activa' : 'inactiva' }}</span>
        <button class="tte-close" style="margin-left:auto" @click="ultimoGuardado=null">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>
        </button>
      </div>
      <div v-if="ultimoGuardado.modo==='fijo'" class="tte-saved-body">
        <span v-for="(t,i) in ultimoGuardado.turnos" :key="i" class="tte-saved-chip" :style="{borderLeftColor:t.color}">
          <span class="tte-idot" :style="{background:t.color}"></span>
          <strong>{{ t.nombre }}</strong>
          <span>{{ t.hora_inicio }} – {{ t.hora_fin }}</span>
        </span>
      </div>
      <div v-else class="tte-saved-body tte-saved-dias">
        <div v-for="d in ultimoGuardado.dias" :key="d.dia" class="tte-saved-dia-row">
          <span class="tte-saved-dia-label">{{ d.dia }}</span>
          <span v-if="d.lista.length===0" class="tte-saved-nada">sin turnos</span>
          <span v-for="(t,i) in d.lista" :key="i" class="tte-saved-chip" :style="{borderLeftColor:t.color}">
            <span class="tte-idot" :style="{background:t.color}"></span>
            <strong>{{ t.nombre }}</strong>
            <span>{{ t.hora_inicio }} – {{ t.hora_fin }}</span>
          </span>
        </div>
      </div>
    </div>

    <div v-if="loading" class="tte-cargando">
      <div class="tte-spinner"></div><span>Cargando</span>
    </div>

    <template v-else-if="!plantaId">
      <div class="tte-empty">Selecciona una planta para configurar los turnos.</div>
    </template>

    <template v-else>
      <div v-if="modo==='fijo'" class="tte-timeline-outer">
        <div class="tte-hours-row">
          <span v-for="h in 25" :key="h" class="tte-hlabel" :style="{left:((h-1)/24*100)+'%'}">{{ String(h-1).padStart(2,'0') }}h</span>
        </div>
        <div class="tte-bar-wrap">
          <div class="tte-bar-bg"></div>
          <template v-for="b in bloquesFijo" :key="b._idx">
            <div class="tte-block" :class="{'is-selected':esSelFijo(b)}"
              :style="{left:b.left+'%',width:b.width+'%',background:b.color}" @click="seleccionarFijo(b)">
              <span class="tte-blabel">{{ b.nombre }}</span>
              <span class="tte-btime">{{ b.hora_inicio }}–{{ b._wrap>0 ? '00:00' : b.hora_fin }}</span>
            </div>
            <div v-if="b._wrap>0" class="tte-block tte-block-wrap" :class="{'is-selected':esSelFijo(b)}"
              :style="{left:'0%',width:b._wrap+'%',background:b.color}" @click="seleccionarFijo(b)">
              <span class="tte-blabel">{{ b.nombre }}</span>
              <span class="tte-btime">00:00–{{ b.hora_fin }}</span>
            </div>
          </template>
          <div v-for="h in 23" :key="'g'+h" class="tte-gridline" :style="{left:(h/24*100)+'%'}"></div>
        </div>
        <div v-if="turnosFijo.length===0" class="tte-bar-hint">Sin turnos — usa Añadir o Auto-generar</div>
      </div>

      <div v-else class="tte-dias-grid">
        <div v-for="(dia,idx) in DIAS" :key="idx" class="tte-dia-row">
          <div class="tte-dia-label">{{ dia }}</div>
          <div class="tte-dia-bar-wrap">
            <div class="tte-bar-bg"></div>
            <template v-for="b in bloquesDia(idx)" :key="b._idx">
              <div class="tte-block tte-block-sm" :class="{'is-selected':esSelDia(idx,b)}"
                :style="{left:b.left+'%',width:b.width+'%',background:b.color}"
                @click="seleccionarDia(idx,b)">
                <span class="tte-blabel">{{ b.nombre }}</span>
              </div>
              <div v-if="b._wrap>0" class="tte-block tte-block-sm tte-block-wrap" :class="{'is-selected':esSelDia(idx,b)}"
                :style="{left:'0%',width:b._wrap+'%',background:b.color}"
                @click="seleccionarDia(idx,b)">
                <span class="tte-blabel">{{ b.nombre }}</span>
              </div>
            </template>
            <div v-for="h in 23" :key="'g'+h" class="tte-gridline" :style="{left:(h/24*100)+'%'}"></div>
          </div>
          <div class="tte-dia-actions">
            <button class="tte-btn secondary tte-btn-xs" @click="nuevaTurnoDia(idx)" title="Añadir">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/></svg>
            </button>
            <button class="tte-btn secondary tte-btn-xs" @click="autoGenerarDia(idx)" title="Auto-generar">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
            </button>
            <button v-if="idx>0" class="tte-btn secondary tte-btn-xs" @click="copiarDiaAnterior(idx)" title="Copiar día anterior">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
            </button>
          </div>
        </div>
        <div class="tte-hours-legend">
          <div class="tte-dia-label"></div>
          <div style="flex:1;position:relative;height:16px;">
            <span v-for="h in 25" :key="h" class="tte-hlabel" :style="{left:((h-1)/24*100)+'%'}">{{ String(h-1).padStart(2,'0') }}h</span>
          </div>
        </div>
      </div>

      <div v-if="seleccionado" class="tte-editor">
        <div class="tte-editor-head">
          <span>Editando turno</span>
          <button class="tte-close" @click="seleccionado=null">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>
          </button>
        </div>
        <div class="tte-editor-body">
          <div class="tte-field"><label>Nombre</label><input v-model="seleccionado.nombre" class="tte-input" type="text"/></div>
          <div class="tte-field">
            <label>Hora inicio</label>
            <div style="flex:1; display:flex; flex-direction:column; gap:4px;">
              <input v-model="seleccionado.hora_inicio" class="tte-input" type="time"/>
              <span v-if="linea && !isAlignedToInterval(seleccionado.hora_inicio)" class="tte-field-warn">
                No alineado al intervalo. Sugerencia: {{ roundToInterval(seleccionado.hora_inicio) }}
              </span>
            </div>
          </div>
          <div class="tte-field">
            <label>Hora fin</label>
            <div style="flex:1; display:flex; flex-direction:column; gap:4px;">
              <input v-model="seleccionado.hora_fin" class="tte-input" type="time"/>
              <span v-if="linea && !isAlignedToInterval(seleccionado.hora_fin)" class="tte-field-warn">
                No alineado al intervalo. Sugerencia: {{ roundToInterval(seleccionado.hora_fin) }}
              </span>
            </div>
          </div>
          <div class="tte-field">
            <label>Color</label>
            <div class="tte-colors">
              <span v-for="col in COLORES" :key="col" class="tte-cdot" :class="{active:seleccionado.color===col}" :style="{background:col}" @click="seleccionado.color=col"></span>
            </div>
          </div>
          <div class="tte-editor-actions">
            <button class="tte-btn danger" @click="eliminar">Eliminar</button>
            <button class="tte-btn secondary" @click="seleccionado=null">Cancelar</button>
            <button class="tte-btn primary" @click="aplicarEdicion">Aplicar</button>
          </div>
        </div>
      </div>

      <div v-if="modo==='fijo' && turnosFijo.length" class="tte-list">
        <div v-for="(t,i) in turnosFijo" :key="i" class="tte-item"
          :class="{'is-selected':esSelFijo({_idx:i})}" @click="seleccionarFijo({...t,_idx:i})">
          <span class="tte-idot" :style="{background:t.color}"></span>
          <span class="tte-iname">{{ t.nombre }}</span>
          <span class="tte-irange">{{ t.hora_inicio }} – {{ t.hora_fin }}</span>
          <span class="tte-idur">{{ durMin(t) }} min</span>
          <span v-if="t._nuevo" class="tte-inew">nuevo</span>
        </div>
      </div>
    </template>

    <Teleport to="body">
      <div v-if="confirmarModoModal" class="tte-overlay" @click.self="cancelarCambioModo">
        <div class="tte-modal">
          <div class="tte-modal-title">Cambiar modo de turno</div>
          <p class="tte-modal-desc">
            Estás cambiando de <strong>{{ modo === 'fijo' ? 'Fijo' : 'Por día' }}</strong>
            a <strong>{{ modoPendiente === 'fijo' ? 'Fijo' : 'Por día' }}</strong>.<br>
            Los turnos configurados en el modo actual se borrarán. Esta acción no se puede deshacer a menos que guardes primero.
          </p>
          <div class="tte-editor-actions">
            <button class="tte-btn secondary" @click="cancelarCambioModo">Cancelar</button>
            <button class="tte-btn danger" @click="confirmarCambioModo">Sí, cambiar modo</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="autoModal" class="tte-overlay" @click.self="autoModal=false">
        <div class="tte-modal">
          <div class="tte-modal-title">Auto-generar turnos</div>
          <p class="tte-modal-desc">Divide las 24 horas en turnos iguales.</p>
          <div class="tte-field"><label>N.º de turnos</label><input v-model.number="autoCount" class="tte-input" type="number" min="1" max="8"/></div>
          <div class="tte-chips">
            <span v-for="i in Math.max(1,Math.min(8,autoCount))" :key="i" class="tte-chip" :style="{background:COLORES[(i-1)%COLORES.length]}">
              T{{ i }}: {{ previewChip(i-1) }}
            </span>
          </div>
          <div class="tte-editor-actions">
            <button class="tte-btn secondary" @click="autoModal=false">Cancelar</button>
            <button class="tte-btn primary" @click="autoGenerarFijo">Generar</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="historialModalOpen" class="tte-overlay" @click.self="cerrarHistorialModal">
        <div class="tte-modal tte-modal-large">
          <div class="tte-modal-header">
            <div class="tte-modal-title-group">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
              </svg>
              <h3 class="tte-modal-title">Historial de Configuración de Turnos</h3>
            </div>
            <button class="tte-close" @click="cerrarHistorialModal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <div v-if="loadingHistorial" class="tte-modal-loading">
            <div class="tte-spinner"></div>
            <span>Cargando detalles...</span>
          </div>

          <div v-else-if="historialDetalle" class="tte-modal-body">
            <div class="tte-trace-header">
              <div class="tte-trace-row">
                <div class="tte-trace-item">
                  <span class="tte-trace-label">Vigente desde:</span>
                  <span class="tte-trace-value">{{ historialDetalle.vigente_desde }}</span>
                  <span v-if="historialDetalle.activo" class="tte-badge badge-on" style="font-size: 11px; margin-left: 6px;">ACTIVA</span>
                  <span v-else class="tte-badge" style="background:#f1f5f9;color:#64748b;font-size:11px;margin-left:6px;">histórica</span>
                </div>
                <div class="tte-trace-item">
                  <span class="tte-trace-label">Creado:</span>
                  <span class="tte-trace-value">{{ new Date(historialDetalle.creado_en).toLocaleString('es-ES') }}</span>
                </div>
              </div>
              <div class="tte-trace-row">
                <div class="tte-trace-item">
                  <span class="tte-trace-label">Modificado por:</span>
                  <span class="tte-trace-value">Sistema (Admin)</span>
                  <span class="tte-trace-hint">pendiente: auditoría de usuario</span>
                </div>
                <div class="tte-trace-item">
                  <span class="tte-trace-label">Total registros:</span>
                  <span class="tte-trace-value">{{ historialDetalle.total_registros }}</span>
                </div>
              </div>
              <div class="tte-trace-row">
                <div class="tte-trace-item">
                  <span class="tte-trace-label">Modo:</span>
                  <span class="tte-badge" :class="historialDetalle.modo==='fijo'?'badge-fijo':'badge-dia'">
                    {{ historialDetalle.modo === 'fijo' ? 'Fijo (igual todos los días)' : 'Por día (independiente)' }}
                  </span>
                </div>
                <div class="tte-trace-item">
                  <span class="tte-trace-label">Renovación semanal:</span>
                  <span class="tte-badge" :class="historialDetalle.renovacion_semanal?'badge-on':'badge-off'">
                    {{ historialDetalle.renovacion_semanal ? 'Activa' : 'Inactiva' }}
                  </span>
                </div>
              </div>
              <div v-if="historialDetalle.activo" class="tte-trace-current">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <polyline points="12 6 12 12 16 14"/>
                </svg>
                <span class="tte-trace-label">Turno actual en ejecución:</span>
                <span class="tte-trace-current-value">{{ historialDetalle.turno_actual }}</span>
              </div>
            </div>

            <div class="tte-trace-section">
              <h4 class="tte-trace-section-title">Turnos Configurados</h4>
              
              <div v-if="historialDetalle.modo === 'fijo'" class="tte-trace-timeline">
                <div class="tte-hours-row">
                  <span v-for="h in 25" :key="h" class="tte-hlabel" :style="{left:((h-1)/24*100)+'%'}">{{ String(h-1).padStart(2,'0') }}h</span>
                </div>
                <div class="tte-bar-wrap tte-bar-wrap-modal">
                  <div class="tte-bar-bg"></div>
                  <template v-for="(t, i) in historialDetalle.turnos_fijo" :key="i">
                    <div class="tte-block tte-block-historial"
                      :class="{'tte-block-active': historialDetalle.turno_actual === t.nombre && historialDetalle.activo}"
                      :style="{left: (toMin(t.hora_inicio)/1440*100)+'%', width: (durMinHist(t)/1440*100)+'%', background: t.color}">
                      <span class="tte-blabel">{{ t.nombre }}</span>
                      <span class="tte-btime">{{ t.hora_inicio }}–{{ t.hora_fin }}</span>
                      <div v-if="historialDetalle.turno_actual === t.nombre && historialDetalle.activo" class="tte-block-badge">
                        <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10"/></svg>
                        En curso
                      </div>
                    </div>
                  </template>
                  <div v-for="h in 23" :key="'g'+h" class="tte-gridline" :style="{left:(h/24*100)+'%'}"></div>
                </div>
              </div>
              
              <div v-if="historialDetalle.modo === 'fijo'" class="tte-turnos-grid" style="margin-top: 20px;">
                <div v-for="(t, i) in historialDetalle.turnos_fijo" :key="i" class="tte-turno-card">
                  <div class="tte-turno-color" :style="{background: t.color}"></div>
                  <div class="tte-turno-info">
                    <div class="tte-turno-nombre">{{ t.nombre }}</div>
                    <div class="tte-turno-horario">{{ t.hora_inicio }} – {{ t.hora_fin }}</div>
                  </div>
                  <div v-if="historialDetalle.turno_actual === t.nombre && historialDetalle.activo" class="tte-turno-badge">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                      <circle cx="12" cy="12" r="10"/>
                    </svg>
                    En curso
                  </div>
                </div>
              </div>

              <div v-else class="tte-turnos-por-dia">
                <div v-for="(dia, idx) in DIAS" :key="idx" class="tte-dia-section">
                  <div class="tte-dia-header">{{ dia }}</div>
                  <div v-if="historialDetalle.turnos_por_dia[idx]?.length">
                    <div class="tte-trace-timeline-dia" style="margin-top: 12px;">
                      <div class="tte-bar-wrap tte-bar-wrap-modal-sm">
                        <div class="tte-bar-bg"></div>
                        <template v-for="(t, i) in historialDetalle.turnos_por_dia[idx]" :key="i">
                          <div class="tte-block tte-block-sm tte-block-historial"
                            :style="{left: (toMin(t.hora_inicio)/1440*100)+'%', width: (durMinHist(t)/1440*100)+'%', background: t.color}">
                            <span class="tte-blabel">{{ t.nombre }}</span>
                          </div>
                        </template>
                        <div v-for="h in 23" :key="'g'+h" class="tte-gridline" :style="{left:(h/24*100)+'%'}"></div>
                      </div>
                    </div>
                    <div class="tte-turnos-grid" style="margin-top: 12px;">
                      <div v-for="(t, i) in historialDetalle.turnos_por_dia[idx]" :key="i" class="tte-turno-card tte-turno-card-sm">
                        <div class="tte-turno-color" :style="{background: t.color}"></div>
                        <div class="tte-turno-info">
                          <div class="tte-turno-nombre">{{ t.nombre }}</div>
                          <div class="tte-turno-horario">{{ t.hora_inicio }} – {{ t.hora_fin }}</div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div v-else class="tte-empty-dia">Sin turnos configurados</div>
                </div>
              </div>
            </div>
          </div>

          <div class="tte-modal-footer">
            <button class="tte-btn secondary" @click="cerrarHistorialModal">Cerrar</button>
          </div>
        </div>
      </div>
    </Teleport>

  </div>
</template>

<style scoped>
.tte-wrap{display:flex;flex-direction:column;gap:18px;padding:16px 0}
.tte-toolbar{display:flex;align-items:center;gap:8px;flex-wrap:wrap}
.tte-modo-toggle{display:flex;border:1px solid #cbd5e1;border-radius:6px;overflow:hidden}
.tte-modo-btn{display:inline-flex;align-items:center;gap:4px;padding:5px 12px;font-size:12px;font-weight:500;cursor:pointer;border:none;background:#f8fafc;color:#64748b;transition:background .12s,color .12s}
.tte-modo-btn.active{background:#1e40af;color:#fff}
.tte-modo-btn:focus{outline:none}
.tte-btn{display:inline-flex;align-items:center;gap:5px;padding:6px 14px;border-radius:6px;font-size:13px;font-weight:500;cursor:pointer;border:none;transition:opacity .15s,background .15s}
.tte-btn:disabled{opacity:.45;cursor:not-allowed}
.tte-btn.primary{background:#1e40af;color:#fff}
.tte-btn.primary:hover:not(:disabled){background:#1d4ed8}
.tte-btn.secondary{background:#e2e8f0;color:#1e293b}
.tte-btn.secondary:hover{background:#cbd5e1}
.tte-btn.danger{background:#fee2e2;color:#991b1b}
.tte-btn.danger:hover{background:#fecaca}
.tte-btn-xs{padding:4px 7px !important;font-size:11px !important}
.tte-warn{display:inline-flex;align-items:center;gap:5px;padding:4px 10px;background:#fef3c7;color:#92400e;border-radius:6px;font-size:12px;font-weight:500}
.tte-cargando{display:flex;align-items:center;gap:10px;color:#94a3b8;font-size:13px;padding:24px;justify-content:center}
.tte-spinner{width:24px;height:24px;border:2px solid #e2e8f0;border-top-color:#6366f1;border-radius:50%;animation:spin .7s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.tte-empty{text-align:center;color:#94a3b8;font-size:13px;padding:28px}
.tte-timeline-outer{position:relative;padding-top:22px}
.tte-hours-row{position:absolute;top:0;left:0;right:0;height:18px}
.tte-hlabel{position:absolute;font-size:10px;color:#94a3b8;transform:translateX(-50%);user-select:none}
.tte-bar-wrap{position:relative;height:64px;border-radius:8px;overflow:visible}
.tte-bar-bg{position:absolute;inset:0;background:#f1f5f9;border-radius:8px;border:1px solid #e2e8f0}
.tte-bar-hint{text-align:center;font-size:12px;color:#94a3b8;margin-top:8px}
.tte-block{position:absolute;top:4px;height:56px;border-radius:6px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;overflow:hidden;border:2px solid transparent;transition:filter .12s,border-color .12s;min-width:20px}
.tte-block:hover{filter:brightness(1.1)}
.tte-block.is-selected{border-color:#1e40af;filter:brightness(1.08)}
.tte-blabel{font-size:11px;font-weight:700;color:#fff;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:90%;text-shadow:0 1px 2px rgba(0,0,0,.3)}
.tte-btime{font-size:9px;color:rgba(255,255,255,.85);white-space:nowrap}
.tte-gridline{position:absolute;top:0;bottom:0;width:1px;background:rgba(148,163,184,.35);pointer-events:none}
.tte-dias-grid{display:flex;flex-direction:column;gap:6px}
.tte-dia-row{display:flex;align-items:center;gap:8px}
.tte-dia-label{width:80px;flex-shrink:0;font-size:12px;font-weight:600;color:#475569;text-align:right}
.tte-dia-bar-wrap{flex:1;height:44px;position:relative;border-radius:6px;overflow:visible}
.tte-block-sm{top:2px;height:40px}
.tte-block-wrap{border-left:3px dashed rgba(255,255,255,.75)!important;border-radius:0 6px 6px 0}
.tte-dia-actions{display:flex;gap:4px;flex-shrink:0}
.tte-hours-legend{display:flex;align-items:center;gap:8px;margin-top:2px}
.tte-editor{background:#f8fafc;border:1px solid #e2e8f0;border-radius:10px;overflow:hidden}
.tte-editor-head{display:flex;align-items:center;justify-content:space-between;padding:10px 16px;background:#f1f5f9;border-bottom:1px solid #e2e8f0;font-size:12px;font-weight:700;color:#475569;text-transform:uppercase;letter-spacing:.05em}
.tte-close{background:none;border:none;cursor:pointer;color:#94a3b8;display:flex;align-items:center}
.tte-close:hover{color:#475569}
.tte-editor-body{padding:14px 16px;display:flex;flex-direction:column;gap:10px}
.tte-field{display:flex;align-items:center;gap:12px}
.tte-field label{font-size:12px;color:#64748b;width:90px;flex-shrink:0}
.tte-input{padding:5px 10px;border:1px solid #cbd5e1;border-radius:6px;font-size:13px;background:#fff;color:#1e293b;max-width:200px;flex:1}
.tte-input:focus{outline:2px solid #6366f1;border-color:transparent}
.tte-colors{display:flex;gap:6px}
.tte-cdot{width:22px;height:22px;border-radius:50%;cursor:pointer;border:2px solid transparent;transition:border-color .1s,transform .1s}
.tte-cdot:hover{transform:scale(1.15)}
.tte-cdot.active{border-color:#1e293b}
.tte-editor-actions{display:flex;gap:8px;justify-content:flex-end;padding-top:4px}
.tte-list{display:flex;flex-direction:column;gap:6px}
.tte-item{display:flex;align-items:center;gap:10px;padding:9px 14px;border-radius:8px;cursor:pointer;background:#f8fafc;border:1px solid #e2e8f0;transition:background .1s}
.tte-item:hover{background:#f1f5f9}
.tte-item.is-selected{background:#eff6ff;border-color:#93c5fd}
.tte-idot{width:12px;height:12px;border-radius:50%;flex-shrink:0}
.tte-iname{font-size:13px;font-weight:600;color:#1e293b;flex:1}
.tte-irange{font-size:12px;color:#64748b}
.tte-idur{font-size:11px;color:#94a3b8;background:#f1f5f9;border-radius:4px;padding:2px 6px}
.tte-inew{font-size:10px;font-weight:700;color:#15803d;background:#dcfce7;border-radius:4px;padding:2px 6px;text-transform:uppercase}
.tte-overlay{position:fixed;inset:0;background:rgba(0,0,0,.4);z-index:200;display:flex;align-items:center;justify-content:center}
.tte-modal{background:#fff;border-radius:12px;padding:24px;width:420px;display:flex;flex-direction:column;gap:14px;box-shadow:0 20px 60px rgba(0,0,0,.18)}
.tte-modal-title{font-size:15px;font-weight:700;color:#1e293b}
.tte-modal-desc{font-size:12px;color:#64748b;margin:0}
.tte-chips{display:flex;flex-wrap:wrap;gap:6px}
.tte-chip{font-size:11px;color:#fff;font-weight:600;padding:3px 10px;border-radius:5px}
.tte-saved-banner{background:#f0fdf4;border:1px solid #86efac;border-radius:10px;overflow:hidden}
.tte-saved-head{display:flex;align-items:center;gap:8px;padding:9px 14px;font-size:12px;font-weight:700;color:#15803d;border-bottom:1px solid #bbf7d0;flex-wrap:wrap}
.tte-saved-badge{font-size:11px;font-weight:600;padding:2px 8px;border-radius:5px}
.tte-saved-body{display:flex;flex-wrap:wrap;gap:8px;padding:10px 14px}
.tte-saved-chip{display:inline-flex;align-items:center;gap:6px;padding:4px 10px 4px 8px;background:#fff;border-radius:6px;border-left:3px solid #ccc;font-size:12px;color:#1e293b}
.tte-saved-chip strong{font-weight:700}
.tte-saved-dias{flex-direction:column;gap:6px}
.tte-saved-dia-row{display:flex;align-items:center;gap:8px;flex-wrap:wrap}
.tte-saved-dia-label{width:78px;flex-shrink:0;font-size:11px;font-weight:700;color:#475569;text-align:right}
.tte-saved-nada{font-size:11px;color:#94a3b8;font-style:italic}
.tte-saved-vig{font-size:11px;font-weight:600;padding:2px 8px;background:#dbeafe;color:#1d4ed8;border-radius:5px}
.tte-vigencia-row{display:flex;align-items:center;gap:12px;flex-wrap:wrap;padding:8px 14px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px}
.tte-vig-group{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.tte-vig-label{font-size:12px;font-weight:600;color:#475569;white-space:nowrap}
.tte-date-input{max-width:160px !important;flex:unset !important}
.tte-vig-alert{display:inline-flex;align-items:center;gap:5px;padding:4px 10px;border-radius:6px;font-size:12px;font-weight:500}
.tte-vig-alert.danger{background:#fee2e2;color:#991b1b}
.tte-vig-alert.warn{background:#fef3c7;color:#92400e}
.tte-vig-alert.ok{background:#dcfce7;color:#15803d}
.tte-err-banner{display:flex;align-items:center;gap:8px;padding:10px 14px;background:#fee2e2;border:1px solid #fca5a5;border-radius:8px;font-size:13px;color:#991b1b;font-weight:500}
.tte-err-banner .tte-close{margin-left:auto}
.tte-historial{border:1px solid #e2e8f0;border-radius:8px;overflow:hidden}
.tte-historial-head{display:flex;align-items:center;gap:7px;padding:9px 14px;background:#f1f5f9;font-size:12px;font-weight:700;color:#475569;border-bottom:1px solid #e2e8f0}
.tte-historial-body{display:flex;flex-direction:column}
.tte-historial-row{display:flex;align-items:center;gap:10px;padding:8px 14px;border-bottom:1px solid #f1f5f9;font-size:12px}
.tte-historial-row:last-child{border-bottom:none}
.tte-historial-row.active{background:#f0fdf4}
.tte-hist-fecha{font-weight:700;color:#1e293b;min-width:110px}
.tte-hist-turnos{color:#64748b;flex:1}
.tte-options-row{display:flex;align-items:center;gap:10px;padding:8px 14px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:8px;flex-wrap:wrap}
.tte-options-label{font-size:12px;color:#64748b;font-weight:500}
.tte-options-sep{flex:1}
.tte-badge{font-size:11px;font-weight:600;padding:2px 8px;border-radius:5px}
.badge-fijo{background:#eff6ff;color:#1d4ed8}
.badge-dia{background:#f5f3ff;color:#7c3aed}
.badge-on{background:#dcfce7;color:#15803d}
.badge-off{background:#f1f5f9;color:#64748b}
.tte-toggle{width:36px;height:20px;border-radius:10px;border:none;cursor:pointer;position:relative;transition:background .2s;background:#cbd5e1;padding:0;flex-shrink:0}
.tte-toggle.on{background:#22c55e}
.tte-toggle-knob{position:absolute;top:2px;left:2px;width:16px;height:16px;border-radius:50%;background:#fff;transition:left .2s;box-shadow:0 1px 3px rgba(0,0,0,.2)}
.tte-toggle.on .tte-toggle-knob{left:18px}
.tte-interval-banner{display:flex;gap:10px;padding:12px 16px;background:#eff6ff;border:1px solid#bfdbfe;border-radius:8px;font-size:12px;color:#1e3a8a}
.tte-interval-banner svg{flex-shrink:0;margin-top:2px;color:#3b82f6}
.tte-interval-content{flex:1;display:flex;flex-direction:column;gap:4px}
.tte-interval-mode{font-weight:700;color:#1d4ed8;background:#dbeafe;padding:1px 6px;border-radius:4px}
.tte-interval-desc{color:#475569;font-size:11px}
.tte-interval-hint{color:#64748b;font-size:11px;margin-top:2px;padding:6px 8px;background:#f8fafc;border-radius:4px;border-left:3px solid #94a3b8}
.tte-field-warn{font-size:11px;color:#92400e;background:#fef3c7;padding:4px 8px;border-radius:4px;display:inline-block}
.tte-modal-large{width:900px;max-width:95vw;max-height:90vh;overflow-y:auto}
.tte-modal-header{display:flex;align-items:center;justify-content:space-between;padding:16px 20px;border-bottom:2px solid #e2e8f0;background:#f8fafc}
.tte-modal-title-group{display:flex;align-items:center;gap:10px}
.tte-modal-title-group svg{color:#6366f1}
.tte-modal-title{font-size:16px;font-weight:700;color:#1e293b;margin:0}
.tte-modal-loading{display:flex;flex-direction:column;align-items:center;justify-content:center;padding:60px 20px;gap:14px;color:#64748b}
.tte-modal-body{padding:20px;max-height:calc(90vh - 180px);overflow-y:auto}
.tte-modal-footer{display:flex;justify-content:flex-end;padding:14px 20px;border-top:1px solid #e2e8f0;background:#f8fafc}
.tte-trace-header{background:#f0fdf4;border:1px solid #86efac;border-radius:8px;padding:16px;margin-bottom:20px}
.tte-trace-row{display:flex;gap:24px;margin-bottom:10px}
.tte-trace-row:last-child{margin-bottom:0}
.tte-trace-item{display:flex;align-items:center;gap:6px;flex:1}
.tte-trace-label{font-size:12px;font-weight:600;color:#15803d}
.tte-trace-value{font-size:13px;font-weight:700;color:#1e293b}
.tte-trace-hint{font-size:11px;color:#64748b;font-style:italic}
.tte-trace-current{display:flex;align-items:center;gap:8px;margin-top:12px;padding:10px 12px;background:#dcfce7;border-radius:6px;border-left:4px solid #22c55e}
.tte-trace-current svg{color:#15803d;flex-shrink:0}
.tte-trace-current-value{font-size:14px;font-weight:700;color:#15803d}
.tte-trace-section{margin-top:20px}
.tte-trace-section-title{font-size:14px;font-weight:700;color:#1e293b;margin:0 0 14px 0;padding-bottom:8px;border-bottom:2px solid #e2e8f0}
.tte-turnos-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:12px}
.tte-turno-card{display:flex;align-items:center;gap:10px;padding:12px 14px;background:#fff;border:1px solid #e2e8f0;border-radius:8px;transition:box-shadow .15s,transform .15s}
.tte-turno-card:hover{box-shadow:0 4px 12px rgba(0,0,0,.08);transform:translateY(-2px)}
.tte-turno-card-sm{padding:10px 12px}
.tte-turno-color{width:8px;height:40px;border-radius:4px;flex-shrink:0}
.tte-turno-info{flex:1;min-width:0}
.tte-turno-nombre{font-size:13px;font-weight:700;color:#1e293b;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.tte-turno-horario{font-size:12px;color:#64748b;margin-top:2px}
.tte-turno-badge{display:flex;align-items:center;gap:4px;font-size:11px;font-weight:600;color:#15803d;background:#dcfce7;padding:4px 8px;border-radius:5px}
.tte-turno-badge svg{animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.5}}
.tte-turnos-por-dia{display:flex;flex-direction:column;gap:16px}
.tte-dia-section{border:1px solid #e2e8f0;border-radius:8px;overflow:hidden}
.tte-dia-header{padding:8px 14px;background:#f1f5f9;font-size:12px;font-weight:700;color:#475569;border-bottom:1px solid #e2e8f0}
.tte-empty-dia{padding:20px;text-align:center;color:#94a3b8;font-size:12px;font-style:italic}
.tte-historial-row:hover{background:#f8fafc}
.tte-historial-empty{display:flex;flex-direction:column;align-items:center;justify-content:center;padding:48px 24px;gap:12px;color:#94a3b8}
.tte-historial-empty svg{color:#cbd5e1}
.tte-historial-empty p{font-size:14px;font-weight:600;color:#64748b;margin:0}
.tte-historial-empty span{font-size:12px;color:#94a3b8;text-align:center;max-width:320px}
.tte-trace-timeline{margin-bottom:12px;position:relative;padding-top:22px}
.tte-trace-timeline-dia{position:relative}
.tte-bar-wrap-modal{height:68px;position:relative;border-radius:8px;overflow:visible}
.tte-bar-wrap-modal-sm{height:48px;position:relative;border-radius:6px;overflow:visible}
.tte-block-historial{cursor:default;border:2px solid rgba(255,255,255,.3);box-shadow:0 2px 8px rgba(0,0,0,.15)}
.tte-block-historial:hover{filter:brightness(1.05)}
.tte-block-active{border:3px solid #15803d !important;box-shadow:0 0 0 2px rgba(21,128,61,.2),0 4px 12px rgba(0,0,0,.2) !important}
.tte-block-badge{position:absolute;top:-8px;right:4px;display:flex;align-items:center;gap:3px;font-size:9px;font-weight:700;color:#15803d;background:#dcfce7;padding:2px 6px;border-radius:4px;border:1px solid #86efac;text-transform:uppercase;letter-spacing:0.3px}
.tte-block-badge svg{animation:pulse 2s infinite}
</style>
