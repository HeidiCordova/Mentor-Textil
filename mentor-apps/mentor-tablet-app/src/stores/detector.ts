import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

const _KEY = 'mentor_detector_state_v1'

function _loadPersisted() {
  try {
    const raw = localStorage.getItem(_KEY)
    if (!raw) return null
    const p = JSON.parse(raw)
    // Descartar si la parada tiene más de 8 horas (turno distinto)
    if (p.stopStartTime && Date.now() - p.stopStartTime > 8 * 3600 * 1000) return null
    return p as { trackerState: string; stopStartTime: number | null }
  } catch { return null }
}

/**
 * Estado del detector compartido entre ProduccionView (escritura) y AppHeader (lectura).
 * Se persiste en localStorage para sobrevivir recargas de página.
 */
export const useDetectorStore = defineStore('detector', () => {
  const _saved = _loadPersisted()

  const trackerState       = ref<'producing' | 'idle_wait' | 'stop_open' | 'offline'>(
    (_saved?.trackerState as 'producing' | 'idle_wait' | 'stop_open' | 'offline') ?? 'producing'
  )
  const stopStartTime      = ref<number | null>(_saved?.stopStartTime ?? null)
  const isProduccionActive = ref(false)  // ProduccionView montado?
  const _now               = ref(Date.now())
  const microparadasCount  = ref(0)
  const paradasCount       = ref(0)
  const idleDurationS      = ref(0)  // synced from ProduccionView (Python source)

  // Segundos desde que comenzó la parada (se actualiza cada 1s via tick())
  const idleSecs = computed(() =>
    stopStartTime.value ? Math.round((_now.value - stopStartTime.value) / 1000) : 0
  )

  // Persistir automáticamente en cada cambio
  watch([trackerState, stopStartTime], () => {
    try {
      localStorage.setItem(_KEY, JSON.stringify({
        trackerState:  trackerState.value,
        stopStartTime: stopStartTime.value,
      }))
    } catch {}
  })

  function tick() { _now.value = Date.now() }

  // Corregir el contador al instante cuando el usuario vuelve a la pestaña
  // y descartar paradas de fondo creadas mientras la pestaña estaba oculta (throttling)
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        tick()
        if (_bgRunning && _bgPageHiddenAt && _bgLocalStopStart && _bgLocalStopStart >= _bgPageHiddenAt) {
          _bgLocalStopStart = null
          _bgMotionOffCount = 0
        }
        _bgPageHiddenAt = null
      } else {
        _bgPageHiddenAt = Date.now()
      }
    })
  }

  function setProducing() {
    trackerState.value  = 'producing'
    stopStartTime.value = null
  }

  function setIdle(startTs: number, isStopOpen: boolean) {
    stopStartTime.value = stopStartTime.value ?? startTs
    trackerState.value  = isStopOpen ? 'stop_open' : 'idle_wait'
  }

  // ── Detección de fondo (WS + canvas offscreen) ────────────────────────
  // Misma lógica pixel-diff que ProduccionView, corre cuando éste NO está montado.
  let _bgWs: WebSocket | null = null
  let _bgRetryTimer: ReturnType<typeof setTimeout> | null = null
  let _bgRetryDelay = 1000
  let _bgCanvas: HTMLCanvasElement | null = null
  let _bgPresenceBuffer: Uint8Array[] = []
  let _bgPresenceN = 0
  let _bgLastFrameAt = 0
  let _bgRunning = false

  // Config (se carga una vez al arrancar fondo)
  let _bgRoiPresencia: number[] | null = null
  let _bgRoi: number[] | null = null          // ROI de conteo (fallback si no hay presencia)
  let _bgThresholds: Record<string, any> = {}
  let _bgMicroStopMaxS = 120
  let _bgWarmup = true  // true hasta que el buffer de presencia se llene

  // Tracking de movimiento (espeja ProduccionView exactamente)
  let _bgMotionOffCount = 0
  let _bgLocalStopStart: number | null = null
  let _bgPageHiddenAt: number | null = null
  const _BG_MOTION_OFF_THR = 3
  const _BG_BRIEF_S = 3
  const _BG_FRAME_INTERVAL = 150 // ~7 fps máx

  async function _bgFetchConfig() {
    try {
      const cfg = await fetch('/edge/config').then(r => r.json())
      const rp = cfg.roi_presencia
      if (rp && typeof rp === 'object') {
        _bgRoiPresencia = Array.isArray(rp)
          ? (rp.length >= 4 ? rp : null)
          : (rp.width > 0 && rp.height > 0 ? [rp.x, rp.y, rp.width, rp.height] : null)
      } else {
        _bgRoiPresencia = null
      }
      // ROI de conteo como fallback (si no hay roi_presencia, usar roi para algo)
      const rc = cfg.roi
      if (rc && Array.isArray(rc) && rc.length >= 4) {
        _bgRoi = rc
      }
      _bgThresholds = cfg.thresholds ?? {}
      const ms = cfg.oee?.micro_stop_max_s
      if (typeof ms === 'number' && ms > 0) _bgMicroStopMaxS = ms
    } catch { /* non-fatal */ }
  }

  function _bgProcessMotion(rawMotion: boolean) {
    if (rawMotion) {
      _bgLocalStopStart = null
      _bgMotionOffCount = 0
      setProducing()
      return
    }
    // Sin movimiento
    _bgMotionOffCount++
    if (_bgMotionOffCount >= _BG_MOTION_OFF_THR && !_bgLocalStopStart) {
      // Heredar parada del store si existe (viene de ProduccionView), sino crear nueva
      _bgLocalStopStart = stopStartTime.value ?? Date.now()
    }
    if (!_bgLocalStopStart) {
      // Aún no pasó el threshold de confirmación, y no hay parada heredada → no tocar store
      return
    }
    const noMotionS = (Date.now() - _bgLocalStopStart) / 1000
    const stopOpenS = _bgMicroStopMaxS > 0 ? _bgMicroStopMaxS : 120
    if (noMotionS < _BG_BRIEF_S) {
      // Hueco breve entre prendas — no es parada todavía
      return
    }
    setIdle(_bgLocalStopStart, noMotionS >= stopOpenS)
  }

  function _bgAnalyze(canvas: HTMLCanvasElement) {
    // Determinar qué ROI usar: presencia preferido, conteo como fallback
    const roi = _bgRoiPresencia ?? _bgRoi
    if (!roi || roi.length < 4) return
    const [px, py, pw, ph] = roi
    if (pw < 8 || ph < 8) return
    const x0 = Math.max(0, px), y0 = Math.max(0, py)
    const x1 = Math.min(canvas.width, px + pw), y1 = Math.min(canvas.height, py + ph)
    const cw = x1 - x0, ch = y1 - y0
    if (cw < 8 || ch < 8) return
    try {
      const data = canvas.getContext('2d', { willReadFrequently: true })!
        .getImageData(x0, y0, cw, ch).data
      const n = cw * ch
      const gray = new Uint8Array(n)
      for (let i = 0; i < n; i++) gray[i] = (data[i * 4] + data[i * 4 + 1] + data[i * 4 + 2]) / 3
      if (_bgPresenceN !== n) { _bgPresenceBuffer = []; _bgPresenceN = n; _bgWarmup = true }
      const PIXEL_THR   = _bgThresholds.presencia_pixel_threshold ?? 8
      const MOTION_AREA = _bgThresholds.presencia_scale ?? 0.005
      const WIN = Math.max(5, Math.round((_bgThresholds.presencia_window ?? 375) * 0.08))
      _bgPresenceBuffer.push(gray)
      if (_bgPresenceBuffer.length > WIN) _bgPresenceBuffer.shift()
      // Warm-up: no evaluar movimiento hasta tener suficiente historial.
      // Esto evita falsos negativos que creen microparadas fantasma.
      if (_bgPresenceBuffer.length < WIN) return
      if (_bgWarmup) {
        _bgWarmup = false
        _bgMotionOffCount = 0   // resetear contador acumulado durante warm-up
        // NO borrar _bgLocalStopStart — si viene de ProduccionView con parada activa, preservarla.
        // Solo se cancela cuando se detecte movimiento real (rawMotion = true en _bgProcessMotion).
      }
      const ref = _bgPresenceBuffer[0]
      let motionPx = 0
      for (let i = 0; i < n; i++) {
        if (Math.abs(gray[i] - ref[i]) > PIXEL_THR) motionPx++
      }
      const rawMotion = (motionPx / n) >= MOTION_AREA
      _bgProcessMotion(rawMotion)
    } catch { /* ignore */ }
  }

  function _bgConnect() {
    if (_bgWs || isProduccionActive.value) return
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${proto}//${location.host}/vision/ws/stream`)
    ws.binaryType = 'arraybuffer'
    _bgWs = ws
    let rendering = false
    ws.onopen = () => { _bgRetryDelay = 1000 }
    ws.onmessage = (evt: MessageEvent) => {
      const now = Date.now()
      if (now - _bgLastFrameAt < _BG_FRAME_INTERVAL || rendering) return
      _bgLastFrameAt = now
      rendering = true
      const blob = new Blob([evt.data], { type: 'image/jpeg' })
      requestAnimationFrame(async () => {
        try {
          const bm = await createImageBitmap(blob)
          if (!_bgCanvas) _bgCanvas = document.createElement('canvas')
          if (_bgCanvas.width !== bm.width || _bgCanvas.height !== bm.height) {
            _bgCanvas.width = bm.width; _bgCanvas.height = bm.height
          }
          _bgCanvas.getContext('2d', { willReadFrequently: true })!.drawImage(bm, 0, 0)
          bm.close()
          _bgAnalyze(_bgCanvas)
        } catch { /* decode error */ }
        rendering = false
      })
    }
    ws.onclose = ws.onerror = () => {
      _bgWs = null
      if (_bgRunning && !isProduccionActive.value) {
        _bgRetryTimer = setTimeout(() => {
          _bgRetryDelay = Math.min(_bgRetryDelay * 1.5, 8000)
          _bgConnect()
        }, _bgRetryDelay)
      }
    }
  }

  function startBackground() {
    if (_bgRunning || isProduccionActive.value) return
    _bgRunning = true
    _bgLocalStopStart = stopStartTime.value
    _bgMotionOffCount = 0
    _bgPresenceBuffer = []
    _bgWarmup = true
    _bgFetchConfig().then(() => { if (_bgRunning) _bgConnect() })
  }

  function stopBackground() {
    _bgRunning = false
    if (_bgRetryTimer) { clearTimeout(_bgRetryTimer); _bgRetryTimer = null }
    if (_bgWs) { _bgWs.onclose = null; _bgWs.onerror = null; _bgWs.close(); _bgWs = null }
    _bgPresenceBuffer = []
    _bgCanvas = null
  }

  // Auto arrancar/parar cuando ProduccionView se monta/desmonta
  watch(isProduccionActive, (active) => {
    if (active) stopBackground()
    else startBackground()
  })

  return { trackerState, stopStartTime, idleSecs, idleDurationS, microparadasCount, paradasCount, isProduccionActive, tick, setProducing, setIdle, startBackground, stopBackground }
})
