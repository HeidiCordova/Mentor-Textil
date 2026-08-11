import type {
  AggregatedHealth,
  BufferSummary,
  CreateStopRequest,
  EdgeEvent,
  EdgeStatus,
  JustifyStopRequest,
  Stop,
  StopSummary,
  Command,
  CategoryTreeNode,
  ProductEntry,
  VelocidadNominalEntry,
  Operator,
  ProductionRun,
  UpsertProductionRunRequest,
  Turno
} from '@/types'
import { getActivePinia } from 'pinia'

let baseURL = ''
let cloudJWT = ''
let apiMode: 'EDGE' | 'CLOUD' = 'EDGE'
let cloudLineaId = 0

/** Resuelve el linea_id para CLOUD mode: usa cloudLineaId si está seteado,
 *  sino intenta leerlo del plantasLineas store para no depender del orden de init. */
function resolveCloudLineaId(): number {
  if (cloudLineaId) return cloudLineaId
  try {
    const pinia = getActivePinia()
    if (pinia) {
      const pl = (pinia.state.value as Record<string, { selectedLineaId?: number | null }>)['plantasLineas']
      if (pl?.selectedLineaId) return pl.selectedLineaId
    }
  } catch { /* ignore */ }
  return 0
}

// Cloud session expiry callback — set by connection store to clear session + go to /login
let _onCloudUnauthorized: (() => void) | null = null
export function onCloudUnauthorized(cb: () => void): void {
  _onCloudUnauthorized = cb
}

export function setBaseURL(url: string): void {
  baseURL = url.replace(/\/$/, '')
}

export function getBaseURL(): string {
  return baseURL
}

export function setApiMode(mode: 'EDGE' | 'CLOUD'): void {
  apiMode = mode
}

export function getApiMode(): 'EDGE' | 'CLOUD' {
  return apiMode
}

export function setCloudLineaId(id: number): void {
  cloudLineaId = id
}

export function setCloudJWT(token: string): void {
  cloudJWT = token
  if (token) localStorage.setItem('cloud_jwt', token)
  else localStorage.removeItem('cloud_jwt')
}

export function getCloudJWT(): string {
  if (!cloudJWT) {
    cloudJWT = localStorage.getItem('cloud_jwt') || ''
  }
  return cloudJWT
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  headers?: Record<string, string>
): Promise<T> {
  const url = `${baseURL}${path}`
  const authHeaders: Record<string, string> = {}
  if (apiMode === 'CLOUD' && cloudJWT) {
    authHeaders['Authorization'] = `Bearer ${cloudJWT}`
  }
  const opts: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-Actor': localStorage.getItem('operator_id') || 'local:operator',
      ...authHeaders,
      ...headers
    }
  }

  if (body !== undefined) {
    opts.body = JSON.stringify(body)
  }

  const res = await fetch(url, opts)

  if (!res.ok) {
    if (res.status === 401 && apiMode === 'CLOUD') {
      // JWT expirado o inválido — limpiar sesión cloud y redirigir al login
      _onCloudUnauthorized?.()
    }
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }

  return res.json()
}

export const api = {
  // ---------- edge-only endpoints (only work in EDGE/HYBRID) ----------
  health(): Promise<AggregatedHealth> {
    return request('GET', '/edge/health')
  },

  async status(lineaId?: number): Promise<EdgeStatus> {
    if (apiMode === 'CLOUD') {
      const lid = lineaId ?? resolveCloudLineaId()
      if (!lid) return { device_id: '', buffer_pending: 0, cloud_connected: true, config_version: 0, recent_errors: [], uptime: 0 }
      try {
        const snap = await request<{ device_id: string; hora: string }>('GET', `/api/oee/latest?linea_id=${lid}`)
        return { device_id: snap.device_id ?? '', buffer_pending: 0, cloud_connected: true, config_version: 0, recent_errors: [], uptime: 0 }
      } catch {
        return { device_id: '', buffer_pending: 0, cloud_connected: true, config_version: 0, recent_errors: [], uptime: 0 }
      }
    }
    const q = lineaId ? `?linea_id=${lineaId}` : ''
    return request('GET', `/edge/status${q}`)
  },

  async getConfig(): Promise<Record<string, unknown>> {
    const lineaId = resolveCloudLineaId()
    if (apiMode === 'CLOUD' && lineaId) {
      // En modo cloud, obtener mode/vel_unit desde cloud-config
      await request<{ mode: string; vel_unit: string; linea_id: number }>(
        'GET', `/api/linea-config?linea_id=${lineaId}`
      )
      // snapshot_interval_s: textil=1800s (30min)
      const snapshot_interval_s = 1800
      // Sintetizar la estructura que espera configStore (oee.vel_unit, oee.snapshot_interval_s)
      return {
        mode: 'textil',
        oee: {
          micro_stop_max_s: 120,
          stop_max_s: 86400,
          vel_unit: 'uh',
          vel_nominal_us: 0.008333333,
          snapshot_interval_s,
        },
      }
    }
    if (apiMode === 'CLOUD') return {}  // sin linea seleccionada, no intentar /edge/config
    return request('GET', '/edge/config')
  },

  async updateConfig(patch: Record<string, unknown>): Promise<Record<string, unknown>> {
    const lineaId = resolveCloudLineaId()
    if (apiMode === 'CLOUD' && lineaId && typeof patch.mode === 'string') {
      // En modo cloud, solo el campo mode se puede actualizar
      await request('PUT', '/api/linea-config', { linea_id: lineaId, mode: 'textil' })
      const snapshot_interval_s = 1800
      return {
        mode: 'textil',
        oee: {
          micro_stop_max_s: 120,
          stop_max_s: 86400,
          vel_unit: 'uh',
          vel_nominal_us: 0.008333333,
          snapshot_interval_s,
        },
      }
    }
    return request('PUT', '/edge/config', patch)
  },

  startCalibration(): Promise<{ status: string }> {
    return request('POST', '/edge/calibration/start')
  },

  async bufferSummary(lineaId?: number): Promise<BufferSummary> {
    if (apiMode === 'CLOUD') {
      // El buffer reside en el edge; en modo cloud devolvemos valores neutros
      return { total_count: 0, pending_count: 0, synced_count: 0, dead_count: 0, oldest_pending: null, newest_pending: null, disk_bytes: 0 }
    }
    const q = lineaId ? `?linea_id=${lineaId}` : ''
    return request('GET', `/edge/buffer/summary${q}`)
  },

  async recentEvents(limit = 50, since?: string, lineaId?: number): Promise<EdgeEvent[]> {
    if (apiMode === 'CLOUD') {
      const lid = lineaId ?? resolveCloudLineaId()
      if (!lid) return []
      const q = new URLSearchParams({ limit: String(limit) })
      q.set('linea_id', String(lid))
      if (since) q.set('from', since)
      type Snap = { id: number; device_id: string; hora: string }
      const res = await request<{ data: Snap[] }>('GET', `/api/oee/snapshots?${q.toString()}`)
      // Mapear cada snapshot a un EdgeEvent con event_type 'cut_detected'
      // para que la timeline pinte los slots de producción correctamente
      return (res.data ?? []).map((s, i) => ({
        id: s.id ?? i,
        event_id: `snap_${s.id ?? i}`,
        device_id: s.device_id ?? '',
        event_type: 'cut_detected',
        timestamp: s.hora,
        payload: {},
        synced: true,
        dead: false,
        created_at: s.hora
      }))
    }
    const q = new URLSearchParams({ limit: String(limit) })
    if (since) q.set('since', since)
    if (lineaId) q.set('linea_id', String(lineaId))
    return request('GET', `/edge/events/recent?${q.toString()}`)
  },

  pendingEvents(limit = 200): Promise<EdgeEvent[]> {
    if (apiMode === 'CLOUD') return Promise.resolve([])
    return request('GET', `/edge/events/pending?limit=${limit}`)
  },

  sendCommand(data: {
    command_type: string
    payload: Record<string, unknown>
    idempotency_key: string
    issued_by: string
  }): Promise<{ status: string; command_id: string }> {
    return request('POST', '/edge/commands', data)
  },

  getCommand(commandId: string): Promise<Command> {
    return request('GET', `/edge/commands/${commandId}`)
  },

  listCommands(limit = 50): Promise<Command[]> {
    return request('GET', `/edge/commands?limit=${limit}`)
  },

  // ---------- mode-aware endpoints (adapt between EDGE / CLOUD) ----------
  listStops(params?: {
    justified?: boolean
    open?: boolean
    stop_type?: string
    since?: string
    until?: string
    limit?: number
    linea_id?: number
    empresa_id?: number
    planta_id?: number
  }): Promise<Stop[]> {
    const q = new URLSearchParams()
    if (params) {
      if (params.justified !== undefined) q.set('justified', String(params.justified))
      if (params.open !== undefined) q.set('open', String(params.open))
      if (params.stop_type) q.set('stop_type', params.stop_type)
      if (params.since) q.set('since', params.since)
      if (params.until) q.set('until', params.until)
      if (params.limit) q.set('limit', String(params.limit))
      if (params.linea_id) q.set('linea_id', String(params.linea_id))
      if (params.empresa_id) q.set('empresa_id', String(params.empresa_id))
      if (params.planta_id) q.set('planta_id', String(params.planta_id))
    }
    const qs = q.toString()
    const prefix = apiMode === 'CLOUD' ? '/api/stops' : '/edge/stops'
    return request('GET', `${prefix}${qs ? '?' + qs : ''}`)
  },

  createStop(data: CreateStopRequest): Promise<Stop> {
    if (apiMode === 'CLOUD') {
      const p = new URLSearchParams()
      if (data.linea_id) p.set('linea_id', String(data.linea_id))
      if (data.planta_id) p.set('planta_id', String(data.planta_id))
      const q = p.toString() ? '?' + p.toString() : ''
      return request('POST', `/api/stops${q}`, data)
    }
    const q = data.linea_id ? `?linea_id=${data.linea_id}` : ''
    return request('POST', `/edge/stops${q}`, data)
  },

  getStop(stopId: string, lineaId?: number, plantaId?: number): Promise<Stop> {
    const p = new URLSearchParams()
    if (lineaId) p.set('linea_id', String(lineaId))
    if (plantaId) p.set('planta_id', String(plantaId))
    const q = p.toString() ? '?' + p.toString() : ''
    if (apiMode === 'CLOUD') return request('GET', `/api/stops/${stopId}${q}`)
    return request('GET', `/edge/stops/${stopId}${q}`)
  },

  justifyStop(stopId: string, data: JustifyStopRequest, lineaId?: number, plantaId?: number): Promise<Stop> {
    const p = new URLSearchParams()
    if (lineaId) p.set('linea_id', String(lineaId))
    if (plantaId) p.set('planta_id', String(plantaId))
    const q = p.toString() ? '?' + p.toString() : ''
    if (apiMode === 'CLOUD') {
      return request('POST', `/api/stops/${stopId}/justify${q}`, data)
    }
    return request('POST', `/edge/stops/${stopId}/justify${q}`, data)
  },

  closeStop(stopId: string, lineaId?: number, plantaId?: number): Promise<Stop> {
    const p = new URLSearchParams()
    if (lineaId) p.set('linea_id', String(lineaId))
    if (plantaId) p.set('planta_id', String(plantaId))
    const q = p.toString() ? '?' + p.toString() : ''
    if (apiMode === 'CLOUD') {
      return request('POST', `/api/stops/${stopId}/close${q}`)
    }
    return request('POST', `/edge/stops/${stopId}/close${q}`)
  },

  deleteStop(stopId: string, lineaId?: number, plantaId?: number): Promise<void> {
    const p = new URLSearchParams()
    if (lineaId) p.set('linea_id', String(lineaId))
    if (plantaId) p.set('planta_id', String(plantaId))
    const q = p.toString() ? '?' + p.toString() : ''
    if (apiMode === 'CLOUD') return request('DELETE', `/api/stops/${stopId}${q}`)
    return request('DELETE', `/edge/stops/${stopId}${q}`)
  },

  stopsSummary(params?: { hours?: number; linea_id?: number; empresa_id?: number; planta_id?: number }): Promise<StopSummary> {
    const q = new URLSearchParams()
    const hours = params?.hours ?? 24
    q.set('hours', String(hours))
    if (params?.linea_id) q.set('linea_id', String(params.linea_id))
    if (params?.empresa_id) q.set('empresa_id', String(params.empresa_id))
    if (params?.planta_id) q.set('planta_id', String(params.planta_id))
    const prefix = apiMode === 'CLOUD' ? '/api/stops/summary' : '/edge/stops/summary'
    return request('GET', `${prefix}?${q.toString()}`)
  },

  async stopCategories(params?: { linea_id?: number }): Promise<CategoryTreeNode[]> {
    if (apiMode === 'CLOUD') {
      if (!params?.linea_id) return []
      // El cloud devuelve un árbol real con IDs propios y hijos anidados (catNodoResp)
      type NodoResp = { id: number; nombre: string; hijos?: NodoResp[] }
      const res = await request<{ programadas: NodoResp[]; no_programadas: NodoResp[] }>(
        'GET', `/api/arbol-paradas?linea_id=${params.linea_id}`
      )
      const convertNodo = (n: NodoResp, tipoParada: string): CategoryTreeNode => {
        const node: CategoryTreeNode = { id: n.id, nombre: n.nombre, tipo_parada: tipoParada }
        if (n.hijos?.length) node.children = n.hijos.map(h => convertNodo(h, tipoParada))
        return node
      }
      const result: CategoryTreeNode[] = []
      const progNodes = (res.programadas ?? []).map(n => convertNodo(n, 'Programada'))
      const noProgNodes = (res.no_programadas ?? []).map(n => convertNodo(n, 'No Programada'))
      if (progNodes.length) result.push({ id: -1, nombre: 'Paradas Programadas', tipo_parada: 'Programada', children: progNodes })
      if (noProgNodes.length) result.push({ id: -2, nombre: 'Paradas No Programadas', tipo_parada: 'No Programada', children: noProgNodes })
      // Categorías de tiempo fijas — igual que en el edge (IDs negativos especiales)
      result.push(
        { id: -10, nombre: 'Refrigerio', tipo_parada: 'REFRIGERIO' },
        { id: -11, nombre: 'Capacitación Obligatoria', tipo_parada: 'CAPACITACION' },
        { id: -12, nombre: 'Mantenimiento Planificado', tipo_parada: 'MANTENIMIENTO' },
      )
      return result
    }
    const q = new URLSearchParams()
    if (params?.linea_id) q.set('linea_id', String(params.linea_id))
    const qs = q.toString()
    return request('GET', `/edge/catalogs/stop-categories${qs ? '?' + qs : ''}`)
  },

  async productCatalog(params?: { linea_id?: number }): Promise<ProductEntry[]> {
    if (apiMode === 'CLOUD') {
      type CloudProducto = { id: number; codigo: string; nombre: string; activo: boolean }
      const qs = params?.linea_id ? `?linea_id=${params.linea_id}` : ''
      const res = await request<{ data: CloudProducto[] }>('GET', `/api/productos${qs}`)
      return (res.data ?? []).filter(p => p.activo).map(p => ({
        producto_id: p.id,
        sku: p.codigo,
        description: p.nombre,
        active: p.activo
      }))
    }
    const qs = params?.linea_id ? `?linea_id=${params.linea_id}` : ''
    return request('GET', `/edge/catalogs/products${qs}`)
  },

  operators(): Promise<Operator[]> {
    if (apiMode === 'CLOUD') return Promise.resolve([])
    return request('GET', '/edge/auth/operators')
  },

  async velocidadNominal(params?: { linea_id?: number }): Promise<VelocidadNominalEntry[]> {
    if (apiMode === 'CLOUD') {
      const qs = params?.linea_id ? `?linea_id=${params.linea_id}` : ''
      const res = await request<{ data: VelocidadNominalEntry[] }>('GET', `/api/velocidad-nominal${qs}`)
      return res.data ?? []
    }
    const qs = params?.linea_id ? `?linea_id=${params.linea_id}` : ''
    const res = await request<{ data: VelocidadNominalEntry[] }>('GET', `/edge/catalogs/velocidad-nominal${qs}`)
    return res.data ?? []
  },

  updateVelocidadNominal(items: { producto_id: number; velocidad_us: number; factor_conv: number; motivo?: string }[]): Promise<{ ok: boolean; updated: number }> {
    if (apiMode === 'CLOUD') {
      return request('PUT', '/api/velocidad-nominal', items)
    }
    return request('PUT', '/edge/catalogs/velocidad-nominal', items)
  },

  getMotivosVelocidad(params?: { linea_id?: number }): Promise<{ data: { id: number; texto: string; activo: boolean; orden: number }[] }> {
    const qs = params?.linea_id ? `?linea_id=${params.linea_id}` : ''
    return request('GET', `/edge/catalogs/velocidad-nominal/motivos${qs}`)
  },

  // ---------- authentication ----------
  login(username: string, password: string): Promise<{ ok: boolean; id: number; username: string; nombre: string; apellido: string; rol: string; empresa_id: number; device_id: string }> {
    return request('POST', '/edge/auth/login', { username, password })
  },

  async cloudLogin(username: string, password: string): Promise<{ ok: boolean; id: number; username: string; nombre: string; apellido: string; rol: string; empresa_id: number | undefined; token: string }> {
    const res = await request<{
      token: string
      refreshToken: string
      user: { id: number; name: string; lastName: string; email: string; role: string; empresa_id: number | null }
    }>('POST', '/api/auth/login', { username, password })
    setCloudJWT(res.token)
    return {
      ok: true,
      id: res.user.id,
      username: res.user.email,
      nombre: res.user.name,
      apellido: res.user.lastName ?? '',
      rol: res.user.role,
      empresa_id: res.user.empresa_id ?? undefined,
      token: res.token
    }
  },

  async plantsLines(): Promise<{ plantas: { id: number; nombre: string; empresa_id: number; empresa_nombre: string }[]; lineas: { id: number; nombre: string; planta_id: number }[] }> {
    if (apiMode === 'CLOUD') {
      const [pRes, lRes] = await Promise.all([
        request<{ data: { id: number; nombre: string; empresa_id: number; compania: string }[] }>('GET', '/api/plantas'),
        request<{ data: { id: number; nombre: string; planta_id: number }[] }>('GET', '/api/lineas')
      ])
      const plantas = (pRes.data ?? []).map(p => ({
        id: p.id,
        nombre: p.nombre,
        empresa_id: p.empresa_id,
        empresa_nombre: p.compania ?? ''
      }))
      return { plantas, lineas: lRes.data ?? [] }
    }
    return request('GET', '/edge/catalogs/plants-lines')
  },

  listProductionRuns(params?: { since?: string; until?: string; linea_id?: number; planta_id?: number; limit?: number }): Promise<ProductionRun[]> {
    const q = new URLSearchParams()
    if (params?.since) q.set('since', params.since)
    if (params?.until) q.set('until', params.until)
    if (params?.linea_id) q.set('linea_id', String(params.linea_id))
    if (params?.planta_id) q.set('planta_id', String(params.planta_id))
    if (params?.limit) q.set('limit', String(params.limit))
    const qs = q.toString()
    const prefix = apiMode === 'CLOUD' ? '/api/production-runs' : '/edge/production-runs'
    return request('GET', `${prefix}${qs ? '?' + qs : ''}`)
  },

  upsertProductionRun(data: UpsertProductionRunRequest, params?: { linea_id?: number; planta_id?: number }): Promise<ProductionRun[]> {
    const base = apiMode === 'CLOUD' ? '/api/production-runs' : '/edge/production-runs'
    const q = new URLSearchParams()
    if (params?.linea_id) q.set('linea_id', String(params.linea_id))
    if (params?.planta_id) q.set('planta_id', String(params.planta_id))
    const qs = q.toString()
    return request('POST', `${base}${qs ? '?' + qs : ''}`, data)
  },

  deleteProductionRun(runId: string): Promise<ProductionRun[]> {
    const prefix = apiMode === 'CLOUD' ? '/api/production-runs' : '/edge/production-runs'
    return request('DELETE', `${prefix}/${runId}`)
  },

  listTurnos(params?: { planta_id?: number }): Promise<{ data: Turno[] }> {
    const q = new URLSearchParams()
    if (params?.planta_id) q.set('planta_id', String(params.planta_id))
    const qs = q.toString()
    const prefix = apiMode === 'CLOUD' ? '/api/turnos' : '/edge/catalogs/sync/turnos'
    return request('GET', `${prefix}${qs ? '?' + qs : ''}`)
  },

  async listVariables(params?: { clave?: string }): Promise<{ id: number; nombre: string; clave: string; valor: string; dispositivo_id?: number; empresa_id: number; activo: boolean }[]> {
    if (apiMode === 'CLOUD') {
      const q = new URLSearchParams()
      if (params?.clave) q.set('clave', params.clave)
      const qs = q.toString()
      const res = await request<{ data: { id: number; nombre: string; clave: string; valor: string; dispositivo_id?: number; empresa_id?: number; activo: boolean }[] }>('GET', `/api/variables${qs ? '?' + qs : ''}`)
      return (res.data ?? []).map(v => ({ ...v, empresa_id: v.empresa_id ?? 0 }))
    }
    const q = new URLSearchParams()
    if (params?.clave) q.set('clave', params.clave)
    const qs = q.toString()
    return request('GET', `/edge/variables${qs ? '?' + qs : ''}`)
  },

  setVariableValor(clave: string, valor: string, dispositivoId = 0): Promise<{ status: string; clave: string; valor: string }> {
    if (apiMode === 'CLOUD') {
      const lineaId = resolveCloudLineaId()
      return request('PATCH', '/api/variables/by-clave', { clave, valor, linea_id: lineaId })
    }
    return request('PATCH', `/edge/variables/${encodeURIComponent(clave)}`, {
      valor,
      dispositivo_id: dispositivoId
    })
  }
}
