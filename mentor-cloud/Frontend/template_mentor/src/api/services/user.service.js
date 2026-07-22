import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'
import { mockUsers } from '../mocks/usuarios.mock'

export const userService = {
  async getAll(params = {}) {
    if (USE_MOCKS) {
      return mockUsers.getAll(params)
    }
    return client.get(API_ENDPOINTS.USERS, { params })
  },

  async getById(id) {
    if (USE_MOCKS) {
      return mockUsers.getById(id)
    }
    return client.get(`${API_ENDPOINTS.USERS}/${id}`)
  },

  async create(data) {
    if (USE_MOCKS) {
      return mockUsers.create(data)
    }
    return client.post(API_ENDPOINTS.USERS, data)
  },

  async update(id, data) {
    if (USE_MOCKS) {
      return mockUsers.update(id, data)
    }
    return client.put(`${API_ENDPOINTS.USERS}/${id}`, data)
  },

  async delete(id) {
    if (USE_MOCKS) {
      return mockUsers.delete(id)
    }
    return client.delete(`${API_ENDPOINTS.USERS}/${id}`)
  }
}
