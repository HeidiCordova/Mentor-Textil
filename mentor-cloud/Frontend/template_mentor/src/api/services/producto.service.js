import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const productoService = {
  getAll(params = {}) {
    return client.get(API_ENDPOINTS.PRODUCTOS, { params })
  },
  getById(id) {
    return client.get(`${API_ENDPOINTS.PRODUCTOS}/${id}`)
  },
  create(data) {
    return client.post(API_ENDPOINTS.PRODUCTOS, data)
  },
  update(id, data) {
    return client.put(`${API_ENDPOINTS.PRODUCTOS}/${id}`, data)
  },
  delete(id) {
    return client.delete(`${API_ENDPOINTS.PRODUCTOS}/${id}`)
  }
}
