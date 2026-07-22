import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const roleService = {
  getAll() {
    return client.get(API_ENDPOINTS.ROLES)
  },
  create(data) {
    return client.post(API_ENDPOINTS.ROLES, data)
  },
  update(id, data) {
    return client.put(`${API_ENDPOINTS.ROLES}/${id}`, data)
  },
  delete(id) {
    return client.delete(`${API_ENDPOINTS.ROLES}/${id}`)
  }
}

