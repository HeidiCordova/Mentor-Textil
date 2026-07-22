import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const lineService = {
  getAll(params = {}) {
    return client.get(API_ENDPOINTS.LINES, { params })
  },
  getById(id) {
    return client.get(`${API_ENDPOINTS.LINES}/${id}`)
  },
  create(data) {
    return client.post(API_ENDPOINTS.LINES, data)
  },
  update(id, data) {
    return client.put(`${API_ENDPOINTS.LINES}/${id}`, data)
  },
  delete(id) {
    return client.delete(`${API_ENDPOINTS.LINES}/${id}`)
  },
  provisionLinea(plantaId, lineaId) {
    return client.post(`/admin/plantas/${plantaId}/lineas/${lineaId}/provision`)
  }
}
