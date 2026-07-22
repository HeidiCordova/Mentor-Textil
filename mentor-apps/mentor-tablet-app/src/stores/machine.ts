import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { EdgeEvent, EdgeStatus, BufferSummary } from '@/types'
import { api } from '@/services/api'
import { sse } from '@/services/sse'

const MAX_RECENT_EVENTS = 200

export const useMachineStore = defineStore('machine', () => {
  const status = ref<EdgeStatus | null>(null)
  const bufferSummary = ref<BufferSummary | null>(null)
  const recentEvents = ref<EdgeEvent[]>([])
  const loading = ref(false)

  const deviceId = computed(() => status.value?.device_id || '')
  const isCloudConnected = computed(() => status.value?.cloud_connected || false)

  let _lineaId: number | undefined

  async function fetchStatus(lineaId?: number): Promise<void> {
    if (lineaId !== undefined) _lineaId = lineaId
    try {
      status.value = await api.status(_lineaId)
    } catch {
      status.value = null
    }
  }

  async function fetchBuffer(lineaId?: number): Promise<void> {
    if (lineaId !== undefined) _lineaId = lineaId
    try {
      bufferSummary.value = await api.bufferSummary(_lineaId)
    } catch {
      bufferSummary.value = null
    }
  }

  async function fetchRecentEvents(limit = 50, since?: string, lineaId?: number): Promise<void> {
    if (lineaId !== undefined) _lineaId = lineaId
    try {
      recentEvents.value = await api.recentEvents(limit, since, _lineaId)
    } catch {
      // retain previous data
    }
  }

  function addEvent(event: EdgeEvent): void {
    recentEvents.value.unshift(event)
    if (recentEvents.value.length > MAX_RECENT_EVENTS) {
      recentEvents.value.pop()
    }
  }

  async function loadAll(): Promise<void> {
    loading.value = true
    await Promise.allSettled([fetchStatus(), fetchBuffer(), fetchRecentEvents()])
    loading.value = false
  }

  function bindSSE(): void {
    sse.on('event.created', (msg) => {
      const ev = msg.payload as EdgeEvent
      if (ev) addEvent(ev)
    })

    sse.on('stop.changed', () => {
      fetchStatus()
    })
  }

  return {
    status,
    bufferSummary,
    recentEvents,
    loading,
    deviceId,
    isCloudConnected,
    fetchStatus,
    fetchBuffer,
    fetchRecentEvents,
    loadAll,
    addEvent,
    bindSSE
  }
})
