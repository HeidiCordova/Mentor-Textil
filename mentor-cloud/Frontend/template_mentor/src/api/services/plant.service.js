import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'
import { mockPlants } from '../mocks/plantas.mock'

export const plantService = {
  async getAll(params = {}) {
    if (USE_MOCKS) {
      return mockPlants.getAll(params)
    }
    return client.get(API_ENDPOINTS.PLANTS, { params })
  },

  async getById(id) {
    if (USE_MOCKS) {
      return mockPlants.getById(id)
    }
    return client.get(`${API_ENDPOINTS.PLANTS}/${id}`)
  },

  async create(data) {
    if (USE_MOCKS) {
      return mockPlants.create(data)
    }
    return client.post(API_ENDPOINTS.PLANTS, data)
  },

  async update(id, data) {
    if (USE_MOCKS) {
      return mockPlants.update(id, data)
    }
    return client.put(`${API_ENDPOINTS.PLANTS}/${id}`, data)
  },

  async delete(id) {
    if (USE_MOCKS) {
      return mockPlants.delete(id)
    }
    return client.delete(`${API_ENDPOINTS.PLANTS}/${id}`)
  },

  async provision(id) {
    return client.post(`/admin/plantas/${id}/provision`)
  }
}
