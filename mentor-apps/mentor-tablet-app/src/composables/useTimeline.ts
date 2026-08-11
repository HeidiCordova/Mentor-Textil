import { ref, shallowRef, onUnmounted, type Ref } from 'vue'
import type { Stop, EdgeEvent, TimelineBlock, TimelineMarker, ProductionRun } from '@/types'
import { useUIStore } from '@/stores/ui'
import { useConfigStore } from '@/stores/config'
import { useDetectorStore } from '@/stores/detector'

export interface HitResult {
  block: TimelineBlock
  lane: 'assigned' | 'unassigned' | 'production'
  slotStart?: number
}

const DARK_COLORS = {
  background: '#0f172a',
  laneBackground: '#1e293b',
  laneBgUnassigned: '#1a2035',
  laneDivider: '#334155',
  gridLine: 'rgba(148,163,184,0.08)',
  gridText: '#475569',
  laneLabel: '#94a3b8',
  parada: '#dc2626',
  micro: '#eab308',
  paradaUnassigned: '#f87171',
  microUnassigned: '#fbbf24',
  slotEmpty: '#475569',
  slotEmptyProd: '#6d28d9',
  selectionRing: '#ffffff',
  nowLine: '#22d3ee',
  blockText: '#f8fafc',
  blockTextShadow: 'rgba(0,0,0,0.7)'
}

const LIGHT_COLORS = {
  background: '#f8fafc',
  laneBackground: '#f1f5f9',
  laneBgUnassigned: '#e8edf5',
  laneDivider: '#cbd5e1',
  gridLine: 'rgba(71,85,105,0.10)',
  gridText: '#94a3b8',
  laneLabel: '#475569',
  parada: '#dc2626',
  micro: '#ca8a04',
  paradaUnassigned: '#fca5a5',
  microUnassigned: '#fde047',
  slotEmpty: '#94a3b8',
  slotEmptyProd: '#7c3aed',
  selectionRing: '#0f172a',
  nowLine: '#0891b2',
  blockText: '#0f172a',
  blockTextShadow: 'rgba(255,255,255,0.6)'
}

const COLORS = { ...DARK_COLORS }

const PRODUCT_PALETTE = [
  '#ea580c', '#0891b2', '#7c3aed', '#059669', '#d946ef',
  '#2563eb', '#ca8a04', '#dc2626', '#0d9488', '#e11d48',
  '#4f46e5', '#16a34a', '#9333ea', '#c026d3', '#0284c7'
]

const productColorMap = new Map<string, string>()

const CFG = {
  leftMargin: 80,
  rightMargin: 70,
  timeAxisHeight: 36,
  laneProportion: [0.38, 0.32, 0.30] as readonly [number, number, number],
  gridIntervalMs: 15 * 60_000,
  minZoomMs: 30 * 60_000,
  maxZoomMs: 24 * 60 * 60_000
}

export function useTimeline(canvasRef: Ref<HTMLCanvasElement | null>) {
  const uiStore = useUIStore()
  const configStore = useConfigStore()
  const detectorStore = useDetectorStore()
  const viewStart = ref(Date.now() - 115 * 60_000)
  const viewEnd = ref(Date.now() + 5 * 60_000)
  const isDirty = ref(true)
  const selectedBlockId = ref<string | null>(null)
  const selectedSlotMs = ref<number | null>(null)
  let _userHasPanned = false
  let _initialFitDone = false

  const assignedBlocks = shallowRef<TimelineBlock[]>([])
  const unassignedBlocks = shallowRef<TimelineBlock[]>([])
  const productionBlocks = shallowRef<TimelineBlock[]>([])
  const eventMarkers = shallowRef<TimelineMarker[]>([])

  let animFrameId = 0
  let dpr = 1
  let W = 0
  let H = 0
  let leftM = 0
  let rightM = 0
  let drawW = 0
  const laneY = [0, 0, 0]
  const laneH = [0, 0, 0]
  let timeAxisY = 0

  function recalcLayout(): void {
    dpr = window.devicePixelRatio || 1
    const canvas = canvasRef.value
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    W = rect.width * dpr
    H = rect.height * dpr
    if (canvas.width !== W || canvas.height !== H) {
      canvas.width = W
      canvas.height = H
    }
    leftM = CFG.leftMargin * dpr
    rightM = CFG.rightMargin * dpr
    drawW = W - leftM - rightM
    const taxH = CFG.timeAxisHeight * dpr
    const drawH = H - taxH
    laneH[0] = Math.floor(drawH * CFG.laneProportion[0])
    laneH[1] = Math.floor(drawH * CFG.laneProportion[1])
    laneH[2] = drawH - laneH[0] - laneH[1]
    laneY[0] = 0
    laneY[1] = laneH[0]
    laneY[2] = laneH[0] + laneH[1]
    timeAxisY = H - taxH
  }

  function msToX(ms: number): number {
    return leftM + ((ms - viewStart.value) / (viewEnd.value - viewStart.value)) * drawW
  }

  function xToMs(x: number): number {
    return viewStart.value + ((x - leftM) / drawW) * (viewEnd.value - viewStart.value)
  }

  function drawBackground(ctx: CanvasRenderingContext2D): void {
    ctx.fillStyle = COLORS.background
    ctx.fillRect(0, 0, W, H)
    ctx.fillStyle = COLORS.background
    ctx.fillRect(leftM, laneY[0], drawW, laneH[0])
    ctx.fillStyle = COLORS.laneBgUnassigned
    ctx.fillRect(leftM, laneY[1], drawW, laneH[1])
    ctx.fillStyle = COLORS.laneBackground
    ctx.fillRect(leftM, laneY[2], drawW, laneH[2])

    ctx.fillStyle = COLORS.laneDivider
    ctx.fillRect(leftM, laneY[1] - 1, drawW, 2)
    ctx.fillRect(leftM, laneY[2] - 1, drawW, 2)
    ctx.fillRect(leftM - 1, 0, 1, timeAxisY)
    ctx.fillRect(leftM + drawW, 0, 1, timeAxisY)
  }

  function drawGrid(ctx: CanvasRenderingContext2D): void {
    const interval = CFG.gridIntervalMs
    const first = Math.ceil(viewStart.value / interval) * interval
    ctx.strokeStyle = COLORS.gridLine
    ctx.lineWidth = 1
    ctx.font = `${11 * dpr}px Inter, system-ui, sans-serif`
    ctx.fillStyle = COLORS.gridText
    ctx.textAlign = 'center'
    for (let t = first; t <= viewEnd.value; t += interval) {
      const x = msToX(t)
      if (x < leftM || x > leftM + drawW) continue
      ctx.beginPath()
      ctx.moveTo(x, 0)
      ctx.lineTo(x, timeAxisY)
      ctx.stroke()
      const d = new Date(t)
      ctx.fillText(
        `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`,
        x,
        timeAxisY + 22 * dpr
      )
    }
  }

  function drawNowLine(ctx: CanvasRenderingContext2D): void {
    const now = Date.now()
    if (now < viewStart.value || now > viewEnd.value) return
    const x = msToX(now)
    ctx.strokeStyle = COLORS.nowLine
    ctx.lineWidth = 2 * dpr
    ctx.setLineDash([6 * dpr, 4 * dpr])
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, timeAxisY)
    ctx.stroke()
    ctx.setLineDash([])
  }

  function drawLaneLabels(ctx: CanvasRenderingContext2D): void {
    ctx.font = `bold ${11 * dpr}px Inter, system-ui, sans-serif`
    ctx.fillStyle = COLORS.laneLabel
    ctx.textAlign = 'center'

    ctx.save()
    ctx.translate(leftM / 2, (laneH[0] + laneH[1]) / 2)
    ctx.rotate(-Math.PI / 2)
    ctx.fillText('Paradas', 0, 0)
    ctx.restore()

    ctx.save()
    ctx.translate(leftM / 2, laneY[2] + laneH[2] / 2)
    ctx.rotate(-Math.PI / 2)
    ctx.fillText('Produccion', 0, 0)
    ctx.restore()

    const rx = leftM + drawW + rightM / 2

    ctx.save()
    ctx.translate(rx, laneY[0] + laneH[0] / 2)
    ctx.rotate(Math.PI / 2)
    ctx.fillText('Asignado', 0, 0)
    ctx.restore()

    ctx.save()
    ctx.translate(rx, laneY[1] + laneH[1] / 2)
    ctx.rotate(Math.PI / 2)
    ctx.fillText('No asignado', 0, 0)
    ctx.restore()
  }

  function getSlotMs(): number {
    // Lee snapshot_interval_s del config del edge para que el slot del timeline
    // coincida con la ventana OEE textil real de 30 minutos.
    const oee = configStore.config.oee as Record<string, unknown> | undefined
    const intervalS = typeof oee?.snapshot_interval_s === 'number' ? oee.snapshot_interval_s : 1800
    return Math.max(intervalS, 60) * 1000
  }

  function blockAtTime(blocks: TimelineBlock[], ms: number, nowMs: number): TimelineBlock | null {
    // El bloque con start más tardío que cubra el punto gana (más específico).
    let best: TimelineBlock | null = null
    for (const b of blocks) {
      const end = b.end === 0 ? nowMs : b.end
      if (ms >= b.start && ms < end) {
        if (!best || b.start > best.start) best = b
      }
    }
    return best
  }

  function drawSlotLane(
    ctx: CanvasRenderingContext2D,
    lane: number,
    blocks: TimelineBlock[],
    emptyColor: string,
    showLabel: boolean,
    excludeBlocks?: TimelineBlock[],
    overlayBlocks?: TimelineBlock[]
  ): void {
    const y0 = laneY[lane]
    const h = laneH[lane]
    const pad = 3 * dpr
    const gap = 2 * dpr
    const r = 4 * dpr
    const cardY = y0 + pad
    const cardH = h - pad * 2
    const slotMs = getSlotMs()
    const nowMs = Date.now()
    const first = Math.floor(viewStart.value / slotMs) * slotMs

    ctx.font = `bold ${9 * dpr}px Inter, system-ui, sans-serif`
    ctx.textAlign = 'center'

    for (let t = first; t < viewEnd.value; t += slotMs) {
      const slotEnd = t + slotMs
      const x1Slot = Math.max(msToX(t) + gap, leftM + 1)
      const x2Slot = Math.min(msToX(slotEnd) - gap, leftM + drawW - 1)
      if (x2Slot - x1Slot < 1) continue

      // Si el midpoint cae en un bloque excluido, skip (sigue siendo slot-level)
      const mid = t + slotMs / 2
      const excluded = excludeBlocks ? blockAtTime(excludeBlocks, mid, nowMs) : null

      // Fondo vacío del slot
      ctx.fillStyle = emptyColor
      ctx.beginPath()
      ctx.roundRect(x1Slot, cardY, x2Slot - x1Slot, cardH, r)
      ctx.fill()

      if (excluded) {
        if (showLabel && x2Slot - x1Slot > 26 * dpr) {
          const d = new Date(t)
          const label = `${d.getHours()}:${d.getMinutes().toString().padStart(2, '0')}`
          ctx.fillStyle = COLORS.blockText
          ctx.globalAlpha = 0.35
          ctx.fillText(label, (x1Slot + x2Slot) / 2, cardY + cardH / 2 + 4 * dpr, x2Slot - x1Slot - 4 * dpr)
          ctx.globalAlpha = 1
        }
        continue
      }

      // Pintar cada bloque que solapa con este slot de forma proporcional
      let mainBlock: TimelineBlock | null = null
      let maxOverlap = 0

      for (const block of blocks) {
        const bEnd = block.end === 0 ? nowMs : block.end
        if (block.start >= slotEnd || bEnd <= t) continue  // sin solapamiento

        const overlapStart = Math.max(block.start, t)
        const overlapEnd = Math.min(bEnd, slotEnd)
        const overlapLen = overlapEnd - overlapStart

        // Calcular píxeles proporcionales al overlap real
        const bx1 = Math.max(msToX(overlapStart), x1Slot)
        const bx2 = Math.min(msToX(overlapEnd), x2Slot)
        const bw = Math.max(bx2 - bx1, 3 * dpr)  // mínimo 3px para micros

        ctx.fillStyle = block.color
        ctx.beginPath()
        ctx.roundRect(bx1, cardY, bw, cardH, r)
        ctx.fill()

        if (overlapLen > maxOverlap) {
          maxOverlap = overlapLen
          mainBlock = block
        }
      }

      // Resaltar selección / multi-select sobre el bloque principal
      if (mainBlock) {
        const bEnd = mainBlock.end === 0 ? nowMs : mainBlock.end
        const overlapStart = Math.max(mainBlock.start, t)
        const overlapEnd = Math.min(bEnd, slotEnd)
        const bx1 = Math.max(msToX(overlapStart), x1Slot)
        const bx2 = Math.min(msToX(overlapEnd), x2Slot)
        const bw = Math.max(bx2 - bx1, 3 * dpr)

        const slotSelected = mainBlock.id === selectedBlockId.value &&
          (lane !== 2 || selectedSlotMs.value === null || t === selectedSlotMs.value)
        if (slotSelected) {
          const pulse = 0.5 + 0.5 * Math.sin(Date.now() / 280)
          ctx.fillStyle = `rgba(255,255,255,${0.55 + 0.25 * pulse})`
          ctx.beginPath()
          ctx.roundRect(bx1, cardY, bw, cardH, r)
          ctx.fill()
          ctx.strokeStyle = '#ffffff'
          ctx.lineWidth = (6 + 2 * pulse) * dpr
          ctx.beginPath()
          ctx.roundRect(bx1 - 2 * dpr, cardY - 2 * dpr, bw + 4 * dpr, cardH + 4 * dpr, r + 2 * dpr)
          ctx.stroke()
          ctx.strokeStyle = '#00e5ff'
          ctx.lineWidth = 3 * dpr
          ctx.beginPath()
          ctx.roundRect(bx1, cardY, bw, cardH, r)
          ctx.stroke()
        }

        if (mainBlock.stopId && uiStore.multiSelectedIds.includes(mainBlock.stopId)) {
          ctx.strokeStyle = '#22d3ee'
          ctx.lineWidth = 2.5 * dpr
          ctx.beginPath()
          ctx.roundRect(bx1, cardY, bw, cardH, r)
          ctx.stroke()
          ctx.fillStyle = 'rgba(34,211,238,0.22)'
          ctx.fill()
        }
      }

      // Cortar porciones donde hay bloques asignados/microparadas:
      // la fila "No asignado" no debe mostrarlos — se deja vacío (emptyColor)
      // para que visualmente quede un corte proporcional al tiempo que ocupan.
      if (overlayBlocks) {
        for (const ob of overlayBlocks) {
          const obEnd = ob.end === 0 ? nowMs : ob.end
          if (ob.start >= slotEnd || obEnd <= t) continue
          const ox1 = ob.start <= t ? x1Slot : Math.max(msToX(ob.start), x1Slot)
          const ox2 = obEnd >= slotEnd ? x2Slot : Math.min(msToX(obEnd), x2Slot)
          if (ox2 <= ox1) continue
          // Sobrepinta con el color vacío del slot → corte limpio sin contenido
          ctx.fillStyle = emptyColor
          ctx.globalAlpha = 1
          ctx.fillRect(ox1, cardY, ox2 - ox1, cardH)
        }
      }

      if (showLabel && x2Slot - x1Slot > 26 * dpr) {
        const d = new Date(t)
        const label = `${d.getHours()}:${d.getMinutes().toString().padStart(2, '0')}`
        ctx.fillStyle = COLORS.blockText
        ctx.globalAlpha = mainBlock ? 1 : 0.35
        ctx.fillText(label, (x1Slot + x2Slot) / 2, cardY + cardH / 2 + 4 * dpr, x2Slot - x1Slot - 4 * dpr)
        ctx.globalAlpha = 1
      }
    }
  }

  function render(): void {
    const canvas = canvasRef.value
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    Object.assign(COLORS, document.documentElement.classList.contains('dark') ? DARK_COLORS : LIGHT_COLORS)
    recalcLayout()
    ctx.clearRect(0, 0, W, H)
    drawBackground(ctx)
    const synths = computeSynthBlocks()
    // Lane 0 (Asignado): slot-based, color vacío = fondo del lane (negro/oscuro)
    drawSlotLane(ctx, 0, assignedBlocks.value, COLORS.laneBackground, false)
    // Lane 1 (No asignado): slot-based para synths + stops no asignados
    // Los bloques asignados (microparadas incluidas) se superponen proporcionalmente
    const allUnassigned = [...synths, ...unassignedBlocks.value]
    drawSlotLane(ctx, 1, allUnassigned, COLORS.slotEmpty, true, undefined, assignedBlocks.value)
    drawSlotLane(ctx, 2, productionBlocks.value, COLORS.slotEmptyProd, false)
    drawCurrentSlotCameraIcon(ctx)
    drawGrid(ctx)
    drawLaneLabels(ctx)
    drawNowLine(ctx)
    isDirty.value = false
  }

  /**
   * Dibuja el ícono de cámara SOLO en el slot que contiene "ahora",
   * únicamente si no hubo ningún cut_detected en esa ventana.
   * Los slots históricos nunca muestran el ícono (no tenemos señal pasada fiable).
   */
  function drawCurrentSlotCameraIcon(ctx: CanvasRenderingContext2D): void {
    const lane = 2
    const y0 = laneY[lane]
    const h = laneH[lane]
    const pad = 3 * dpr
    const gap = 2 * dpr
    const cardY = y0 + pad
    const cardH = h - pad * 2

    const nowMs = Date.now()
    const slotMs = getSlotMs()
    const slotStart = Math.floor(nowMs / slotMs) * slotMs

    // Solo si el slot actual está visible
    if (slotStart >= viewEnd.value || slotStart + slotMs <= viewStart.value) return

    // Solo si hay un bloque de producción activo (la línea estaba corriendo)
    const block = blockAtTime(productionBlocks.value, slotStart + slotMs / 2, nowMs)
    if (!block) return

    // ¿Hubo algún cut_detected / CORTE en este slot?
    const hasCut = _cachedEvents.some((e) => {
      if (e.event_type !== 'cut_detected' && e.event_type !== 'CORTE') return false
      const t = new Date(e.timestamp).getTime()
      return t >= slotStart && t < slotStart + slotMs
    })
    if (hasCut) return  // hay conteos → no dibujar ícono

    const x1 = Math.max(msToX(slotStart) + gap, leftM + 1)
    const x2 = Math.min(msToX(slotStart + slotMs) - gap, leftM + drawW - 1)
    if (x2 - x1 < 12 * dpr) return

    const cx = (x1 + x2) / 2
    const cy = cardY + cardH / 2
    const s = Math.min(cardH / 2.5, 16 * dpr)
    const lw = Math.max(1.5 * dpr, 1)

    ctx.save()
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'

    // Cuerpo de la cámara
    const bw = s * 1.35, bh = s * 0.9
    const bx = cx - bw / 2, by = cy - bh / 2
    ctx.fillStyle = 'rgba(0,0,0,0.40)'
    ctx.strokeStyle = 'rgba(255,255,255,0.70)'
    ctx.lineWidth = lw
    ctx.beginPath()
    ctx.roundRect(bx, by, bw, bh, s * 0.14)
    ctx.fill()
    ctx.stroke()

    // Bolita visor
    ctx.fillStyle = 'rgba(255,255,255,0.60)'
    ctx.beginPath()
    ctx.arc(cx - bw * 0.28, by - s * 0.22, s * 0.16, 0, Math.PI * 2)
    ctx.fill()

    // Lente
    ctx.fillStyle = 'rgba(0,0,0,0.30)'
    ctx.strokeStyle = 'rgba(255,255,255,0.70)'
    ctx.lineWidth = lw
    ctx.beginPath()
    ctx.arc(cx, cy, s * 0.30, 0, Math.PI * 2)
    ctx.fill()
    ctx.stroke()

    // Diagonal roja de "apagado"
    ctx.strokeStyle = 'rgba(255, 60, 60, 0.92)'
    ctx.lineWidth = lw * 2.0
    const d = s * 0.62
    ctx.beginPath()
    ctx.moveTo(cx - d, cy + d * 0.75)
    ctx.lineTo(cx + d, cy - d * 0.75)
    ctx.stroke()

    ctx.restore()
  }

  function scheduleRender(): void {
    isDirty.value = true
  }

  let _lastOpenRender = 0
  function renderLoop(): void {
    const hasSynth = computeSynthBlocks().some((b) => b.end === 0)
    if (_hasOpenBlocks || hasSynth) {
      const now = Date.now()
      if (now - _lastOpenRender >= 1000) {
        _lastOpenRender = now
        isDirty.value = true
        // Live-follow: si el usuario no ha navegado al pasado y el fit inicial está hecho,
        // deslizar la ventana automáticamente para mantener "ahora" visible
        if (!_userHasPanned && _initialFitDone) {
          const span = viewEnd.value - viewStart.value
          const target = now + 5 * 60_000
          if (Math.abs(viewEnd.value - target) > 30_000) {
            viewEnd.value = target
            viewStart.value = target - span
          }
        }
      }
    }
    // Animar pulso de selección en cada frame
    if (selectedBlockId.value !== null) {
      isDirty.value = true
    }
    if (isDirty.value) render()
    animFrameId = requestAnimationFrame(renderLoop)
  }

  // end === 0 es centinela: bloque abierto, su extremo derecho es Date.now() en tiempo real
  let _hasOpenBlocks = false

  // Cache de últimos datos para re-evaluar bloque sintético en tiempo real
  let _cachedStops: Stop[] = []
  let _cachedRuns: ProductionRun[] = []
  let _cachedEvents: EdgeEvent[] = []
  let _cachedShiftStart: number = 0

  /** Umbral mínimo sin cortes para pintar bloque rojo (3 minutos) */
  const NO_DATA_THRESHOLD_MS = 3 * 60_000
  /** Duración mínima de un hueco para que valga la pena pintarlo (2 minutos) */
  const MIN_SYNTH_DURATION_MS = 2 * 60_000

  function _makeSynth(id: string, start: number, end: number): TimelineBlock {
    return {
      id,
      start,
      end,
      type: 'stop_unassigned' as const,
      label: 'Sin produccion',
      color: COLORS.paradaUnassigned
    }
  }

  /**
   * Calcula todos los bloques sintéticos "Sin producción" en tiempo real:
   * - Hueco pre-producción: desde el primer dato hasta el primer cut_detected
   * - Hueco actual: desde el último cut_detected hasta ahora (si > threshold)
   * - Sin datos en absoluto: bloque desde inicio inferido hasta ahora
   */
  function computeSynthBlocks(): TimelineBlock[] {
    const now = Date.now()
    const result: TimelineBlock[] = []

    const cutTimes = _cachedEvents
      .filter((e) => e.event_type === 'cut_detected' || e.event_type === 'CORTE')
      .map((e) => new Date(e.timestamp).getTime())
      .sort((a, b) => a - b)

    // Punto más antiguo conocido entre paradas, runs y eventos
    const allStarts = [
      ..._cachedStops.map((s) => new Date(s.started_at).getTime()),
      ..._cachedRuns.map((r) => new Date(r.started_at).getTime()),
      ...cutTimes,
    ]
    // Usar el inicio del turno como ancla si está disponible,
    // así el bloque sintético cubre todo el turno desde el inicio, no solo los últimos 3 min.
    const dataStart = _cachedShiftStart > 0
      ? _cachedShiftStart
      : allStarts.length > 0
        ? Math.min(...allStarts)
        : now - NO_DATA_THRESHOLD_MS

    // Info del detector JS para decidir si inyectar live stop
    const { trackerState, stopStartTime } = detectorStore
    const hasLiveStop =
      (trackerState === 'idle_wait' || trackerState === 'stop_open') &&
      stopStartTime !== null

    if (cutTimes.length === 0) {
      // Nunca hubo cortes
      if (hasLiveStop) {
        // El detector tiene parada activa → inyectar live stop en vez de bloque genérico
        const alreadyCovered = _cachedStops.some((s) => {
          const sStart = new Date(s.started_at).getTime()
          const sEnd = s.ended_at ? new Date(s.ended_at).getTime() : now + 60_000
          return sStart <= stopStartTime! && sEnd >= now
        })
        if (!alreadyCovered) {
          // Bloque genérico solo hasta donde empieza el live stop
          if (stopStartTime! - dataStart > MIN_SYNTH_DURATION_MS) {
            result.push(_makeSynth('__synth_full__', dataStart, stopStartTime!))
          }
          result.push({
            id: '__live_stop__',
            start: stopStartTime!,
            end: 0,
            type: 'stop_unassigned' as const,
            label: trackerState === 'stop_open' ? 'PARADA' : 'Micro parada',
            color: trackerState === 'stop_open'
              ? COLORS.paradaUnassigned
              : COLORS.microUnassigned,
          })
        } else {
          result.push(_makeSynth('__synth_full__', dataStart, 0))
        }
      } else if (trackerState === 'producing') {
        // Produciendo → solo bloque genérico si hay datos históricos sin cortes
        result.push(_makeSynth('__synth_full__', dataStart, 0))
      } else {
        result.push(_makeSynth('__synth_full__', dataStart, 0))
      }
      return result
    }

    const firstCut = cutTimes[0]
    const lastCut = cutTimes[cutTimes.length - 1]

    // Hueco pre-producción (inicio de datos → primer corte)
    if (firstCut - dataStart > MIN_SYNTH_DURATION_MS) {
      result.push(_makeSynth('__synth_pre__', dataStart, firstCut))
    }

    // ── Live stop del detector JS (reacción instantánea) ──
    // Sigue la misma lógica de ProduccionView:
    //   producing → nada | idle_wait → microparada (amarillo) | stop_open → parada (rojo)

    if (hasLiveStop) {
      // Solo inyectar si ningún stop de la BD ya cubre este rango
      const alreadyCovered = _cachedStops.some((s) => {
        const sStart = new Date(s.started_at).getTime()
        const sEnd = s.ended_at ? new Date(s.ended_at).getTime() : now + 60_000
        return sStart <= stopStartTime! && sEnd >= now
      })
      if (!alreadyCovered) {
        result.push({
          id: '__live_stop__',
          start: stopStartTime!,
          end: 0,
          type: 'stop_unassigned' as const,
          label: trackerState === 'stop_open' ? 'PARADA' : 'Micro parada',
          color: trackerState === 'stop_open'
            ? COLORS.paradaUnassigned
            : COLORS.microUnassigned,
        })
      }
    }

    // Hueco actual (último corte → ahora) — solo si el detector NO tiene live stop
    // (evita superponer el bloque sintético genérico con el live stop preciso)
    if (!hasLiveStop && now - lastCut > NO_DATA_THRESHOLD_MS) {
      result.push(_makeSynth('__synth_post__', lastCut, 0))
    }

    return result
  }

  function stopsToAssigned(stops: Stop[]): TimelineBlock[] {
    return stops
      .filter((s) => s.justified || s.stop_type === 'MICROPARADA')
      .map((s) => ({
        id: s.stop_id,
        start: new Date(s.started_at).getTime(),
        end: s.ended_at ? new Date(s.ended_at).getTime() : 0,
        type: 'stop_assigned' as const,
        label: s.stop_type.replace(/_/g, ' '),
        stopId: s.stop_id,
        color: s.stop_type === 'MICROPARADA' ? COLORS.micro : COLORS.parada
      }))
      .sort((a, b) => a.start - b.start)
  }

  function stopsToUnassigned(stops: Stop[]): TimelineBlock[] {
    return stops
      .filter((s) => !s.justified && s.stop_type !== 'MICROPARADA')
      .map((s) => ({
        id: s.stop_id,
        start: new Date(s.started_at).getTime(),
        end: s.ended_at ? new Date(s.ended_at).getTime() : 0,
        type: 'stop_unassigned' as const,
        label: s.stop_type.replace(/_/g, ' '),
        stopId: s.stop_id,
        color: s.stop_type === 'MICROPARADA' ? COLORS.microUnassigned : COLORS.paradaUnassigned
      }))
      .sort((a, b) => a.start - b.start)
  }

  function getProductColor(productId: string): string {
    if (productColorMap.has(productId)) return productColorMap.get(productId)!
    const color = PRODUCT_PALETTE[productColorMap.size % PRODUCT_PALETTE.length]
    productColorMap.set(productId, color)
    return color
  }

  function assignProduct(productId: string, label: string, startMs: number, endMs: number): void {
    const color = getProductColor(productId)
    const existing = productionBlocks.value.filter(
      (b) => b.id !== `prod-${startMs}` && (b.end <= startMs || b.start >= endMs)
    )
    existing.push({
      id: `prod-${productId}-${startMs}`,
      start: startMs,
      end: endMs,
      type: 'production',
      label,
      color
    })
    productionBlocks.value = existing.sort((a, b) => a.start - b.start)
    scheduleRender()
  }

  function runsToBlocks(runs: ProductionRun[]): TimelineBlock[] {
    return runs.map((run) => {
      const start = new Date(run.started_at).getTime()
      const end = run.ended_at ? new Date(run.ended_at).getTime() : 0
      const productId = run.sku ?? `run-${run.run_id}`
      const color = getProductColor(productId)
      return {
        id: `run-${run.run_id}`,
        start,
        end,
        type: 'production' as const,
        label: run.nombre ?? run.sku ?? 'Sin programacion',
        color
      }
    })
  }

  function eventsToMarkers(events: EdgeEvent[]): TimelineMarker[] {
    return events.map((e) => ({
      time: new Date(e.timestamp).getTime(),
      type: ((e.event_type === 'cut_detected' || e.event_type === 'CORTE') ? 'cut' : 'event') as TimelineMarker['type'],
      label: e.event_type
    }))
  }

  function updateData(stops: Stop[], runs: ProductionRun[], events: EdgeEvent[], shiftStartMs = 0): void {
    // Cachear para que computeSynthBlock() se evalúe en tiempo real
    _cachedStops = stops
    _cachedRuns = runs
    _cachedEvents = events
    _cachedShiftStart = shiftStartMs

    const now = Date.now()
    const assigned = stopsToAssigned(stops)
    const unassigned = stopsToUnassigned(stops)

    _hasOpenBlocks =
      assigned.some((b) => b.end === 0) ||
      unassigned.some((b) => b.end === 0)

    assignedBlocks.value = assigned
    unassignedBlocks.value = unassigned
    productionBlocks.value = runsToBlocks(runs)
    eventMarkers.value = eventsToMarkers(events)

    if (!_userHasPanned && !_initialFitDone) {
      const stopTimes = stops.map((s) => new Date(s.started_at).getTime())
      const runTimes = runs
        .map((r) => new Date(r.started_at).getTime())
        .filter((t) => t > 0 && t < now + 86_400_000)

      const anchorTimes = [...stopTimes, ...runTimes]
      if (anchorTimes.length > 0) {
        _initialFitDone = true
        const mostRecent = Math.max(...anchorTimes)
        const anyInView = anchorTimes.some(
          (t) => t >= viewStart.value && t <= viewEnd.value
        )
        if (!anyInView) {
          const spanMs = Math.min(2 * 3600_000, CFG.maxZoomMs)
          const end = Math.min(mostRecent + 30 * 60_000, now + 5 * 60_000)
          viewStart.value = end - spanMs
          viewEnd.value = end
        }
      }
    }

    scheduleRender()
  }

  function panBy(deltaMs: number): void {
    _userHasPanned = true
    viewStart.value += deltaMs
    viewEnd.value += deltaMs
    scheduleRender()
  }

  function zoomTo(centerMs: number, newSpanMs: number): void {
    _userHasPanned = true
    const span = Math.max(CFG.minZoomMs, Math.min(CFG.maxZoomMs, newSpanMs))
    viewStart.value = centerMs - span / 2
    viewEnd.value = centerMs + span / 2
    scheduleRender()
  }

  function scrollToNow(): void {
    _userHasPanned = false  // vuelve a modo live-follow automático
    const span = viewEnd.value - viewStart.value
    const target = Date.now() + 5 * 60_000
    viewEnd.value = target
    viewStart.value = target - span
    scheduleRender()
  }

  /** Ajusta la ventana desde el inicio del turno actual hasta ahora */
  function fitToMode(): void {
    viewEnd.value = Date.now() + 5 * 60_000
    viewStart.value = viewEnd.value - 4 * 3600_000
    scheduleRender()
  }

  function scrollToDate(ms: number): void {
    _userHasPanned = true
    const span = viewEnd.value - viewStart.value
    viewStart.value = ms
    viewEnd.value = ms + span
    scheduleRender()
  }

  function resetAutoFit(): void {
    _userHasPanned = false
    _initialFitDone = false
  }

  function setWindow(startMs: number, endMs: number): void {
    _userHasPanned = true
    viewStart.value = startMs
    viewEnd.value = endMs
    scheduleRender()
  }

  function hitTest(clientX: number, clientY: number): HitResult | null {
    const canvas = canvasRef.value
    if (!canvas) return null
    const rect = canvas.getBoundingClientRect()
    const x = (clientX - rect.left) * dpr
    const y = (clientY - rect.top) * dpr
    if (x < leftM || x > leftM + drawW || y >= timeAxisY) return null

    let lane: HitResult['lane']
    let blocks: TimelineBlock[]

    if (y < laneY[1]) {
      lane = 'assigned'
      blocks = assignedBlocks.value
    } else if (y < laneY[2]) {
      lane = 'unassigned'
      const synths = computeSynthBlocks()
      blocks = [
        ...unassignedBlocks.value.filter((b) => !b.id.startsWith('__synth_')),
        ...synths,
      ]
    } else {
      lane = 'production'
      blocks = productionBlocks.value
    }

    const ms = xToMs(x)
    const nowMs = Date.now()
    const hitBlock = blockAtTime(blocks, ms, nowMs)

    if (hitBlock) {
      // Parada real: incluir slotStart para que DashboardView pueda calcular el fragmento clickeado
      if (hitBlock.stopId) {
        const slotMs = getSlotMs()
        const slotStart = Math.floor(ms / slotMs) * slotMs
        return { block: hitBlock, lane, slotStart }
      }
      // Bloque de producción real: devolver con slotStart para resaltar solo ese slot
      if (lane === 'production' && !hitBlock.id.startsWith('__synth_')) {
        const slotMs = getSlotMs()
        const slotStart = Math.floor(ms / slotMs) * slotMs
        return { block: hitBlock, lane, slotStart }
      }
      // Bloques sintéticos de parada: caer al slot
    }

    // Lane 0 (Asignado): solo bloques reales — clic en vacío no abre modal
    if (lane === 'assigned') return null

    const slotMs = getSlotMs()
    const slotStart = Math.floor(ms / slotMs) * slotMs
    return {
      block: {
        id: lane === 'production' ? 'prod-click' : 'stop-click',
        start: slotStart,
        end: slotStart + slotMs,
        type: lane === 'unassigned' ? 'stop_unassigned' : 'production',
        color: lane === 'production' ? '#94a3b8' : '#dc2626'
      },
      lane
    }
  }

  function start(): void {
    isDirty.value = true
    renderLoop()
  }

  function stop(): void {
    if (animFrameId) {
      cancelAnimationFrame(animFrameId)
      animFrameId = 0
    }
  }

  onUnmounted(stop)

  return {
    viewStart,
    viewEnd,
    isDirty,
    selectedBlockId,
    selectedSlotMs,
    assignedBlocks,
    unassignedBlocks,
    productionBlocks,
    eventMarkers,
    msToX,
    xToMs,
    render,
    scheduleRender,
    updateData,
    assignProduct,
    panBy,
    zoomTo,
    scrollToNow,
    scrollToDate,
    setWindow,
    hitTest,
    start,
    stop,
    resetAutoFit,
    fitToMode
  }
}
