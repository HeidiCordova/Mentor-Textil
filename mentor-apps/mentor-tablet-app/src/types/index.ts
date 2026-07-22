export type ConnectionMode = 'EDGE' | 'CLOUD' | 'HYBRID' | 'OFFLINE'

export type StopType =
  | 'MICROPARADA'
  | 'PARADA_NO_ASIGNADA'
  | 'NO_PROGRAMADA'
  | 'PROGRAMADA'
  | 'MECANICA'
  | 'ELECTRICA'
  | 'CAMBIO_FORMATO'
  | 'FALTA_MATERIAL'
  | 'CALIDAD'
  | 'REFRIGERIO'
  | 'CAPACITACION'
  | 'MANTENIMIENTO'
  | 'OTRA'

export type StopSource = 'detector' | 'operator' | 'cloud' | 'system'

export interface Stop {
  id: number
  stop_id: string
  device_id: string
  stop_type: StopType
  started_at: string
  ended_at: string | null
  duration_ms: number | null
  justified: boolean
  reason: string | null
  category: string | null
  categoria_id: number | null
  justified_by: string | null
  justified_at: string | null
  source: StopSource
  created_at: string
  updated_at: string
  synced: boolean
}

export interface CreateStopRequest {
  device_id?: string
  stop_type: StopType
  started_at: string
  ended_at?: string
  reason?: string
  category?: string
  categoria_id?: number
  source: StopSource
  linea_id?: number
  planta_id?: number
  empresa_id?: number
}

export interface JustifyStopRequest {
  stop_type?: StopType
  reason: string
  category: string
  categoria_id?: number
  justified_by?: string
}

export interface StopSummary {
  total_stops: number
  open_stops: number
  justified_stops: number
  unjustified_stops: number
  total_downtime_ms: number
  by_type: Record<string, number>
}

export interface EdgeEvent {
  id: number
  event_id: string
  device_id: string
  event_type: string
  timestamp: string
  payload: Record<string, unknown>
  synced: boolean
  dead: boolean
  created_at: string
}

export interface BufferSummary {
  total_count: number
  pending_count: number
  synced_count: number
  dead_count: number
  oldest_pending: string | null
  newest_pending: string | null
  disk_bytes: number
}

export interface AggregatedHealth {
  service: string
  status: 'ok' | 'degraded' | 'error'
  device_id: string
  uptime: number
  deps: Record<string, string>
}

export interface EdgeStatus {
  device_id: string
  buffer_pending: number
  cloud_connected: boolean
  config_version: number
  recent_errors: string[]
  uptime: number
}

export interface Command {
  command_id: string
  device_id: string
  command_type: string
  payload: Record<string, unknown>
  issued_by: string
  idempotency_key: string
  status: 'RECEIVED' | 'APPLIED' | 'FAILED'
  result: Record<string, unknown> | null
  error_message: string | null
  applied_at: string | null
}

export interface SSEMessage {
  type: string
  payload: unknown
  ts: number
}

export interface DeviceEntry {
  id: string
  name: string
  url: string
  lastSeen: number
}

export interface ShiftInfo {
  code: string
  label: string
  start: string
  end: string
}

export interface TimelineBlock {
  id: string
  start: number
  end: number
  type: 'production' | 'stop_assigned' | 'stop_unassigned'
  label?: string
  stopId?: string
  color: string
}

export interface TimelineMarker {
  time: number
  type: 'event' | 'cut'
  label?: string
}

export interface CategoryTreeNode {
  id: number
  nombre: string
  codigo?: string
  tipo_parada?: string
  children?: CategoryTreeNode[]
}

export interface ProductEntry {
  sku: string
  description: string
  active?: boolean
}

export interface VelocidadNominalEntry {
  producto_id: number
  sku: string
  descripcion: string
  velocidad_us: number
  factor_conv: number
}

export interface ProductionRun {
  id: number
  run_id: string
  device_id: string
  linea_id: number | null
  producto_id: number | null
  sku: string | null
  nombre: string | null
  started_at: string
  ended_at: string | null
  synced: boolean
  created_at: string
  updated_at: string
}

export interface UpsertProductionRunRequest {
  run_id?: string
  device_id?: string
  linea_id?: number
  producto_id?: number
  sku?: string | null
  nombre?: string | null
  started_at: string
  ended_at?: string | null
}

export interface Operator {
  id: number
  username: string
  nombre: string
  apellido?: string
  rol: string
  empresa_id?: number
}

export interface Turno {
  id: number
  nombre: string
  hora_inicio: string  // "HH:MM"
  hora_fin: string     // "HH:MM"
  planta_id: number | null
  activo: boolean
  synced_at: string
}
