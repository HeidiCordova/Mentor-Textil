import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const locacionService = {
  getAll(params = {}) {
    return client.get(API_ENDPOINTS.LOCACIONES, { params })
  },
  getById(id) {
    return client.get(`${API_ENDPOINTS.LOCACIONES}/${id}`)
  },
  create(data) {
    return client.post(API_ENDPOINTS.LOCACIONES, data)
  },
  update(id, data) {
    return client.put(`${API_ENDPOINTS.LOCACIONES}/${id}`, data)
  },
  delete(id) {
    return client.delete(`${API_ENDPOINTS.LOCACIONES}/${id}`)
  }
}
