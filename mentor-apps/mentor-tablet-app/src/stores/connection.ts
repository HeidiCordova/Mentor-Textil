import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ConnectionMode, AggregatedHealth, Operator } from '@/types'
import { setBaseURL, api, setApiMode, setCloudJWT, getCloudJWT, onCloudUnauthorized } from '@/services/api'
import { sse } from '@/services/sse'
import { probeDevice } from '@/services/discovery'

const PROBE_INTERVAL_MS = 10_000
const CLOUD_HEALTH_TIMEOUT_MS = 4_000

export const useConnectionStore = defineStore('connection', () => {
  const mode = ref<ConnectionMode>('OFFLINE')
  const edgeURL = ref('')
  const cloudURL = ref(localStorage.getItem('cloud_url') || '')
  const edgeReachable = ref(false)
  const cloudReachable = ref(false)
  // Solo sondear cloud si el usuario conectó explícitamente (evita spam de errores en modo edge puro)
  const cloudExplicitlyConnected = ref(false)
  const health = ref<AggregatedHealth | null>(null)
  const sseConnected = ref(false)
  const operatorId = ref(localStorage.getItem('operator_id') || '')
  const operator = ref<Operator | null>(restoreOperator())

  let probeTimer: ReturnType<typeof setInterval> | null = null

  const authenticated = computed(() => !!operator.value)

  const activeURL = computed(() => {
    if (edgeReachable.value) return edgeURL.value
    if (cloudReachable.value) return cloudURL.value
    return ''
  })

  function resolveMode(): void {
    if (edgeReachable.value && cloudReachable.value) {
      mode.value = 'HYBRID'
    } else if (edgeReachable.value) {
      mode.value = 'EDGE'
    } else if (cloudReachable.value) {
      mode.value = 'CLOUD'
    } else {
      mode.value = 'OFFLINE'
    }

    const url = activeURL.value
    if (url) setBaseURL(url)
    setApiMode(edgeReachable.value ? 'EDGE' : 'CLOUD')
  }

  async function probeCloud(): Promise<boolean> {
    if (!cloudURL.value) return false
    // Si la URL guardada es HTTP pero estamos en HTTPS, actualizarla para evitar Mixed Content
    if (window.location.protocol === 'https:' && cloudURL.value.startsWith('http://')) {
      const fixed = cloudURL.value.replace(/^http:\/\/[^/]+/, () => {
        const host = window.location.hostname
        const parts = host.split('.')
        const root = parts.length > 2 ? parts.slice(1).join('.') : host
        return `https://${root}`
      })
      cloudURL.value = fixed
      localStorage.setItem('cloud_url', fixed)
    }
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), CLOUD_HEALTH_TIMEOUT_MS)
    try {
      const res = await fetch(`${cloudURL.value}/health`, { signal: controller.signal })
      return res.ok
    } catch {
      return false
    } finally {
      clearTimeout(timer)
    }
  }

  async function probe(): Promise<void> {
    // No hacer probes HTTP desde HTTPS (Mixed Content)
    if (edgeURL.value && window.location.protocol !== 'https:') {
      edgeReachable.value = await probeDevice(edgeURL.value)
    }
    if (cloudURL.value && cloudExplicitlyConnected.value) {
      cloudReachable.value = await probeCloud()
    }
    resolveMode()

    if (edgeReachable.value) {
      try { health.value = await api.health() } catch { health.value = null }
    }
  }

  function connectToEdge(url: string): void {
    edgeURL.value = url
    localStorage.setItem('edge_url', url)
    setBaseURL(url)

    sse.connect(url)
    sseConnected.value = true
    sse.on('connected', () => { sseConnected.value = true })
    sse.on('disconnected', () => { sseConnected.value = false })
    sse.onError(() => { sseConnected.value = false })

    probe()
    startProbing()
  }

  function connectToCloud(url: string): void {
    cloudURL.value = url.replace(/\/$/, '')
    localStorage.setItem('cloud_url', cloudURL.value)
    setBaseURL(cloudURL.value)
    setApiMode('CLOUD')
    cloudReachable.value = true
    cloudExplicitlyConnected.value = true
    mode.value = 'CLOUD'
    startProbing()
  }

  function disconnect(): void {
    sse.disconnect()
    sseConnected.value = false
    edgeReachable.value = false
    cloudReachable.value = false
    health.value = null
    mode.value = 'OFFLINE'
    stopProbing()
  }

  function startProbing(): void {
    stopProbing()
    probeTimer = setInterval(probe, PROBE_INTERVAL_MS)
  }

  function stopProbing(): void {
    if (probeTimer) {
      clearInterval(probeTimer)
      probeTimer = null
    }
  }

  function setOperator(id: string): void {
    operatorId.value = id
    localStorage.setItem('operator_id', id)
  }

  async function login(username: string, password: string): Promise<boolean> {
    try {
      const res = await api.login(username, password)
      if (!res.ok) return false
      const op: Operator = { id: res.id, username: res.username, nombre: res.nombre, apellido: res.apellido, rol: res.rol, empresa_id: res.empresa_id }
      operator.value = op
      operatorId.value = op.nombre
      localStorage.setItem('operator_id', op.nombre)
      localStorage.setItem('operator', JSON.stringify(op))
      return true
    } catch {
      return false
    }
  }

  function logout(): void {
    operator.value = null
    operatorId.value = ''
    localStorage.removeItem('operator')
    localStorage.removeItem('operator_id')
    setCloudJWT('')
  }

  async function cloudLogin(username: string, password: string): Promise<boolean> {
    try {
      const res = await api.cloudLogin(username, password)
      if (!res.ok) return false
      const op: Operator = { id: res.id, username: res.username, nombre: res.nombre, apellido: res.apellido, rol: res.rol, empresa_id: res.empresa_id }
      operator.value = op
      operatorId.value = op.nombre
      localStorage.setItem('operator_id', op.nombre)
      localStorage.setItem('operator', JSON.stringify(op))
      // Conectar SSE al cloud stream para recibir cambios en tiempo real
      sse.connectCloud(cloudURL.value, res.token, res.empresa_id ?? undefined)
      sseConnected.value = true
      sse.onError(() => { sseConnected.value = false })
      _registerCloudUnauthorizedHandler()
      return true
    } catch {
      return false
    }
  }

  function _registerCloudUnauthorizedHandler(): void {
    onCloudUnauthorized(() => {
      // JWT expirado: limpiar sesión y volver al login
      operator.value = null
      operatorId.value = ''
      localStorage.removeItem('operator')
      localStorage.removeItem('operator_id')
      setCloudJWT('')
      sse.disconnect()
      sseConnected.value = false
      mode.value = 'OFFLINE'
      stopProbing()
      window.location.replace('/login')
    })
  }

  function restoreCloudSession(): boolean {
    const url = localStorage.getItem('cloud_url')
    const jwt = getCloudJWT()
    const op = restoreOperator()
    if (!url || !jwt || !op || !op.rol) return false
    cloudURL.value = url
    setBaseURL(url)
    setApiMode('CLOUD')
    setCloudJWT(jwt)
    operator.value = op
    operatorId.value = op.nombre
    cloudReachable.value = true
    cloudExplicitlyConnected.value = true
    mode.value = 'CLOUD'
    startProbing()
    // Reconectar SSE al cloud stream al restaurar sesión
    sse.connectCloud(url, jwt, op.empresa_id ?? undefined)
    sseConnected.value = true
    sse.onError(() => { sseConnected.value = false })
    _registerCloudUnauthorizedHandler()
    return true
  }

  const isCloudOnly = computed(() => mode.value === 'CLOUD')

  return {
    mode, edgeURL, cloudURL,
    edgeReachable, cloudReachable,
    health, sseConnected,
    operatorId, operator,
    authenticated, activeURL, isCloudOnly,
    connectToEdge, connectToCloud, disconnect,
    probe, probeCloud,
    setOperator, login, cloudLogin, logout,
    restoreCloudSession
  }
})

function restoreOperator(): Operator | null {
  try {
    const raw = localStorage.getItem('operator')
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}
