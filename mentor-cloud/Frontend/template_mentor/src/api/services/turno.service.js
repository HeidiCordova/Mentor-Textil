import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const turnoService = {
  getAll(params = {}) {
    return client.get(API_ENDPOINTS.TURNOS, { params })
  },
  getById(id) {
    return client.get(`${API_ENDPOINTS.TURNOS}/${id}`)
  },
  create(data) {
    return client.post(API_ENDPOINTS.TURNOS, data)
  },
  update(id, data) {
    return client.put(`${API_ENDPOINTS.TURNOS}/${id}`, data)
  },
  delete(id) {
    return client.delete(`${API_ENDPOINTS.TURNOS}/${id}`)
  }
}
