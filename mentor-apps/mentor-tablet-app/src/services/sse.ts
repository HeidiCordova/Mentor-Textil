import type { SSEMessage } from '@/types'

type SSEHandler = (msg: SSEMessage) => void

const RECONNECT_BASE_MS = 1000
const RECONNECT_MAX_MS = 30000

export class SSEClient {
  private url = ''
  private source: EventSource | null = null
  private handlers = new Map<string, Set<SSEHandler>>()
  private globalHandlers = new Set<SSEHandler>()
  private errorHandlers = new Set<() => void>()
  private reconnectMs = RECONNECT_BASE_MS
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private active = false

  connect(baseURL: string): void {
    this.url = `${baseURL}/edge/stream`
    this.active = true
    this.reconnectMs = RECONNECT_BASE_MS
    this.open()
  }

  connectCloud(cloudURL: string, jwt: string, empresaId?: number): void {
    // empresa_id es requerido por el servidor — sin él devuelve 400 en loop.
    // Superadmins no tienen empresa_id en el JWT; esperar a que el usuario
    // seleccione una planta/línea (DashboardView watch lo provee).
    if (!empresaId) return
    const params = new URLSearchParams({ token: jwt, empresa_id: String(empresaId) })
    this.url = `${cloudURL}/api/v1/cloud/stream?${params.toString()}`
    this.active = true
    this.reconnectMs = RECONNECT_BASE_MS
    this.open()
  }

  disconnect(): void {
    this.active = false
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.source) {
      this.source.close()
      this.source = null
    }
  }

  on(eventType: string, handler: SSEHandler): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set())
    }
    this.handlers.get(eventType)!.add(handler)
    return () => this.handlers.get(eventType)?.delete(handler)
  }

  onAny(handler: SSEHandler): () => void {
    this.globalHandlers.add(handler)
    return () => this.globalHandlers.delete(handler)
  }

  onError(handler: () => void): () => void {
    this.errorHandlers.add(handler)
    return () => this.errorHandlers.delete(handler)
  }

  get connected(): boolean {
    return this.source?.readyState === EventSource.OPEN
  }

  private open(): void {
    if (!this.active) return
    if (this.source) {
      this.source.close()
    }

    this.source = new EventSource(this.url)

    this.source.onopen = () => {
      this.reconnectMs = RECONNECT_BASE_MS
    }

    this.source.onmessage = (ev: MessageEvent) => {
      try {
        const msg: SSEMessage = JSON.parse(ev.data)
        this.dispatch(msg)
      } catch {
        // malformed message
      }
    }

    this.source.addEventListener('connected', (ev: Event) => {
      const me = ev as MessageEvent
      try {
        const payload = JSON.parse(me.data)
        this.dispatch({ type: 'connected', payload, ts: Date.now() })
      } catch {
        // ignore
      }
    })

    this.source.onerror = () => {
      this.source?.close()
      this.source = null
      this.errorHandlers.forEach((h) => h())
      this.scheduleReconnect()
    }
  }

  private dispatch(msg: SSEMessage): void {
    const typed = this.handlers.get(msg.type)
    if (typed) {
      typed.forEach((h) => h(msg))
    }
    this.globalHandlers.forEach((h) => h(msg))
  }

  private scheduleReconnect(): void {
    if (!this.active) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.open()
    }, this.reconnectMs)
    this.reconnectMs = Math.min(this.reconnectMs * 2, RECONNECT_MAX_MS)
  }
}

export const sse = new SSEClient()
