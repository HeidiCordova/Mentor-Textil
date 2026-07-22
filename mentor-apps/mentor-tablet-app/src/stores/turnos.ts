import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/services/api'
import type { Turno } from '@/types'

// ─── helpers ──────────────────────────────────────────────
function timeToMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(':').map(Number)
  return h * 60 + (m ?? 0)
}

function isOvernight(turno: Turno): boolean {
  return timeToMinutes(turno.hora_fin) <= timeToMinutes(turno.hora_inicio)
}

function isNowInTurno(turno: Turno): boolean {
  const now = new Date()
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const start = timeToMinutes(turno.hora_inicio)
  const end = timeToMinutes(turno.hora_fin)
  if (!isOvernight(turno)) {
    return nowMin >= start && nowMin < end
  } else {
    // e.g. 22:00 → 06:00  spans midnight
    return nowMin >= start || nowMin < end
  }
}

function fmt12(hhmm: string): string {
  const [h, m] = hhmm.split(':').map(Number)
  const ampm = h < 12 ? 'AM' : 'PM'
  const h12 = h % 12 === 0 ? 12 : h % 12
  return `${h12}:${String(m ?? 0).padStart(2, '0')} ${ampm}`
}

// ─── fallback (hardcoded, usado si no hay turnos en DB) ───
function fallbackActiveTurno(): { nombre: string; hora_inicio: string; hora_fin: string } {
  const h = new Date().getHours()
  if (h >= 6 && h < 14) return { nombre: 'A', hora_inicio: '06:00', hora_fin: '14:00' }
  if (h >= 14 && h < 22) return { nombre: 'B', hora_inicio: '14:00', hora_fin: '22:00' }
  return { nombre: 'C', hora_inicio: '22:00', hora_fin: '06:00' }
}

// ─── store ────────────────────────────────────────────────
export const useTurnosStore = defineStore('turnos', () => {
  const turnos = ref<Turno[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchTurnos(params?: { planta_id?: number }): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const res = await api.listTurnos(params)
      turnos.value = (res.data ?? []).filter((t) => t.activo)
    } catch (e) {
      console.warn('[turnos] fetch failed, using fallback', e)
      turnos.value = []
    } finally {
      loading.value = false
    }
  }

  /** Turno activo según hora actual. null si no hay turnos en DB. */
  const activeTurno = computed<Turno | null>(() => {
    if (turnos.value.length === 0) return null
    return turnos.value.find((t) => isNowInTurno(t)) ?? turnos.value[0]
  })

  /**
   * ISO string del inicio del turno activo.
   * Se usa como parámetro `since` para fetchStops.
   */
  function shiftSince(): string {
    const turno = activeTurno.value
    const fallback = fallbackActiveTurno()
    const horario = turno ?? fallback

    const now = new Date()
    const d = new Date(now)
    const startMin = timeToMinutes(horario.hora_inicio)
    const endMin = timeToMinutes(horario.hora_fin)
    const h = Math.floor(startMin / 60)
    const m = startMin % 60

    // Turno nocturno que arrancó ayer
    if (endMin <= startMin) {
      const nowMin = now.getHours() * 60 + now.getMinutes()
      if (nowMin < endMin) {
        d.setDate(d.getDate() - 1)
      }
    }

    d.setHours(h, m, 0, 0)
    return d.toISOString()
  }

  /** Etiqueta para mostrar en la línea de tiempo (label inferior) */
  const shiftLabel = computed<string>(() => {
    const turno = activeTurno.value
    const fallback = fallbackActiveTurno()
    const horario = turno ?? fallback
    const nombre = turno ? turno.nombre : horario.nombre

    const now = new Date()
    const fmt = `${now.getMonth() + 1}/${now.getDate()}/${String(now.getFullYear()).slice(2)}`

    return `Turno: ${nombre}  Desde: ${fmt}, ${fmt12(horario.hora_inicio)}  Hasta: ${fmt12(horario.hora_fin)}`
  })

  return {
    turnos,
    loading,
    error,
    fetchTurnos,
    activeTurno,
    shiftSince,
    shiftLabel
  }
})
