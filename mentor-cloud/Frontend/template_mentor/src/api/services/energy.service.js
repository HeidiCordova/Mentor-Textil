import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'

export const energyService = {
  async getMeters(params = {}) {
    if (USE_MOCKS) return { data: [] }
    return client.get(API_ENDPOINTS.ENERGY.METERS, { params })
  },

  async getMeterById(id) {
    if (USE_MOCKS) return { data: {} }
    return client.get(`${API_ENDPOINTS.ENERGY.METERS}/${id}`)
  },

  async createMeter(data) {
    if (USE_MOCKS) return { data: {} }
    return client.post(API_ENDPOINTS.ENERGY.METERS, data)
  },

  async updateMeter(id, data) {
    if (USE_MOCKS) return { data: {} }
    return client.put(`${API_ENDPOINTS.ENERGY.METERS}/${id}`, data)
  },

  async deleteMeter(id) {
    if (USE_MOCKS) return { data: {} }
    return client.delete(`${API_ENDPOINTS.ENERGY.METERS}/${id}`)
  },

  async getSnapshots(params = {}) {
    if (USE_MOCKS) return { data: [], total: 0 }
    return client.get(API_ENDPOINTS.ENERGY.SNAPSHOTS, { params })
  },

  async enqueueEdgeConfig(payload) {
    if (USE_MOCKS) return { status: 'queued' }
    return client.post('/edge-control/energy/config', payload)
  },

  async enqueueMeterUpsert(payload) {
    if (USE_MOCKS) return { status: 'queued' }
    return client.post('/edge-control/energy/meters/upsert', payload)
  },

  async enqueueMeterDelete(payload) {
    if (USE_MOCKS) return { status: 'queued' }
    return client.post('/edge-control/energy/meters/delete', payload)
  },

  async enqueueMC60Command(payload) {
    if (USE_MOCKS) return { status: 'queued' }
    return client.post('/edge-control/energy/mc60/command', payload)
  },

  buildEdgeControlWsUrl(token) {
    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const t = encodeURIComponent(token || '')
    return `${proto}://${window.location.host}/api/edge-control/ws?token=${t}`
  }
}
