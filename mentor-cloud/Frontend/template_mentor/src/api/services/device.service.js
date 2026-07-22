import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'

export const deviceService = {
  async getAll(params = {}) {
    if (USE_MOCKS) return { data: [] }
    return client.get(API_ENDPOINTS.DEVICES, { params })
  },

  async getById(id) {
    if (USE_MOCKS) return { data: {} }
    return client.get(`${API_ENDPOINTS.DEVICES}/${id}`)
  },

  async create(data) {
    if (USE_MOCKS) return { data: {} }
    return client.post(API_ENDPOINTS.DEVICES, data)
  },

  async update(id, data) {
    if (USE_MOCKS) return { data: {} }
    return client.put(`${API_ENDPOINTS.DEVICES}/${id}`, data)
  },

  async revoke(id) {
    if (USE_MOCKS) return { data: {} }
    return client.delete(`${API_ENDPOINTS.DEVICES}/${id}`)
  },

  async regenerateKey(id) {
    if (USE_MOCKS) return { data: {} }
    return client.post(`${API_ENDPOINTS.DEVICES}/${id}/regenerate-key`)
  },

  async activate(id) {
    if (USE_MOCKS) return { data: {} }
    return client.post(`${API_ENDPOINTS.DEVICES}/${id}/activate`)
  },

  async deleteDevice(id) {
    if (USE_MOCKS) return { data: {} }
    return client.delete(`${API_ENDPOINTS.DEVICES}/${id}/permanent`)
  }
}
