import type { Stop, EdgeEvent, StopType, StopSource } from '@/types'

let idSeq = 1
const rnd = (min: number, max: number) => Math.floor(Math.random() * (max - min + 1)) + min

function makeStop(
  offsetMs: number,
  durationMs: number,
  type: StopType,
  source: StopSource,
  justified: boolean,
  reason?: string
): Stop {
  const now = Date.now()
  const start = new Date(now + offsetMs)
  const end = new Date(now + offsetMs + durationMs)
  const sid = `demo-stop-${idSeq++}`
  return {
    id: idSeq,
    stop_id: sid,
    device_id: 'jetson-demo',
    stop_type: type,
    started_at: start.toISOString(),
    ended_at: end.toISOString(),
    duration_ms: durationMs,
    justified,
    reason: reason || null,
    category: justified ? type : null,
    categoria_id: null,
    justified_by: justified ? 'operator' : null,
    justified_at: justified ? end.toISOString() : null,
    source,
    created_at: start.toISOString(),
    updated_at: end.toISOString(),
    synced: false
  }
}

function makeEvent(offsetMs: number, eventType: string): EdgeEvent {
  const now = Date.now()
  const ts = new Date(now + offsetMs)
  const eid = `demo-evt-${idSeq++}`
  return {
    id: idSeq,
    event_id: eid,
    device_id: 'jetson-demo',
    event_type: eventType,
    timestamp: ts.toISOString(),
    payload: {},
    synced: false,
    dead: false,
    created_at: ts.toISOString()
  }
}

export function generateDemoData(): { stops: Stop[]; events: EdgeEvent[] } {
  idSeq = 1
  const H = 3600_000
  const M = 60_000

  const stops: Stop[] = [
    makeStop(-2 * H - 30 * M, 3 * M, 'MICROPARADA', 'detector', true, 'Microparada detectada'),
    makeStop(-2 * H - 10 * M, 2 * M, 'MICROPARADA', 'detector', true, 'Microparada detectada'),
    makeStop(-1 * H - 50 * M, 5 * M, 'MECANICA', 'detector', true, 'Falla Mecanica'),
    makeStop(-1 * H - 40 * M, 10 * M, 'PARADA_NO_ASIGNADA', 'detector', true, 'Falla linea'),
    makeStop(-1 * H - 25 * M, 4 * M, 'ELECTRICA', 'operator', true, 'Falla Electrica'),
    makeStop(-1 * H - 18 * M, 3 * M, 'PARADA_NO_ASIGNADA', 'detector', true, 'Parada asignada'),
    makeStop(-1 * H - 5 * M, rnd(1, 2) * M, 'MICROPARADA', 'detector', true, 'Microparada'),
    makeStop(-55 * M, 8 * M, 'CAMBIO_FORMATO', 'operator', true, 'Cambio de Formato'),
    makeStop(-40 * M, rnd(2, 3) * M, 'MICROPARADA', 'detector', true, 'Microparada detectada'),
    makeStop(-35 * M, rnd(2, 4) * M, 'MICROPARADA', 'detector', true, 'Micro tension hilo'),
    makeStop(-25 * M, 5 * M, 'PARADA_NO_ASIGNADA', 'detector', false),
    makeStop(-18 * M, 4 * M, 'PARADA_NO_ASIGNADA', 'detector', false),
    makeStop(-10 * M, rnd(1, 2) * M, 'MICROPARADA', 'detector', true, 'Microparada'),
    makeStop(-5 * M, rnd(1, 2) * M, 'MICROPARADA', 'detector', true, 'Microparada rapida')
  ]

  const events: EdgeEvent[] = []
  for (let i = -2 * H; i < 0; i += rnd(1, 4) * M) {
    events.push(makeEvent(i, rnd(0, 3) === 0 ? 'cut_detected' : 'frame_processed'))
  }

  return { stops, events }
}
