import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const variableService = {
  getAll(params = {}) {
    return client.get(API_ENDPOINTS.VARIABLES, { params })
  },
  getById(id) {
    return client.get(`${API_ENDPOINTS.VARIABLES}/${id}`)
  },
  create(data) {
    return client.post(API_ENDPOINTS.VARIABLES, data)
  },
  update(id, data) {
    return client.put(`${API_ENDPOINTS.VARIABLES}/${id}`, data)
  },
  delete(id) {
    return client.delete(`${API_ENDPOINTS.VARIABLES}/${id}`)
  }
}
