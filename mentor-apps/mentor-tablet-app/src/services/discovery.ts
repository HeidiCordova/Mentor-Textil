import type { DeviceEntry } from '@/types'

const STORAGE_KEY = 'mentor_devices'
const HEALTH_TIMEOUT_MS = 3000

export async function probeDevice(url: string): Promise<boolean> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), HEALTH_TIMEOUT_MS)

  try {
    const res = await fetch(`${url}/edge/health`, { signal: controller.signal })
    return res.ok
  } catch {
    return false
  } finally {
    clearTimeout(timer)
  }
}

export function loadSavedDevices(): DeviceEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

export function saveDevice(device: DeviceEntry): void {
  const devices = loadSavedDevices()
  const idx = devices.findIndex((d) => d.url === device.url)
  if (idx >= 0) {
    devices[idx] = device
  } else {
    devices.push(device)
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(devices))
}

export function removeDevice(url: string): void {
  const devices = loadSavedDevices().filter((d) => d.url !== url)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(devices))
}

export async function discoverDefaults(): Promise<DeviceEntry[]> {
  // Siempre probar primero el mismo origen desde el que cargó la app.
  // nginx proxea /edge/ → edge-gateway internamente, así funciona desde
  // cualquier red/router sin depender de IPs ni puertos hardcodeados.
  const origin = window.location.origin
  const directPort = `${window.location.protocol}//${window.location.hostname}:8005`
  const candidates = [
    { id: 'origin', name: 'Edge Local', url: origin },
    { id: 'direct', name: 'Edge Local (puerto directo)', url: directPort },
  ]

  const results: DeviceEntry[] = []
  const checks = candidates.map(async (c) => {
    const ok = await probeDevice(c.url)
    if (ok) {
      results.push({ ...c, lastSeen: Date.now() })
    }
  })

  await Promise.allSettled(checks)
  return results
}
