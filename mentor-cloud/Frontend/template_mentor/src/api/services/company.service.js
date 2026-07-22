import client from '../client'
import { API_ENDPOINTS, USE_MOCKS } from '../endpoints'
import companyMock from '../mocks/empresas.mock'

export default {
  getAll() {
    if (USE_MOCKS) {
      return companyMock.getAll()
    }
    return client.get(API_ENDPOINTS.COMPANIES)
  },

  getById(id) {
    if (USE_MOCKS) {
      return companyMock.getById(id)
    }
    return client.get(`${API_ENDPOINTS.COMPANIES}/${id}`)
  },

  create(data) {
    if (USE_MOCKS) {
      return companyMock.create(data)
    }
    return client.post(API_ENDPOINTS.COMPANIES, data)
  },

  update(id, data) {
    if (USE_MOCKS) {
      return companyMock.update(id, data)
    }
    return client.put(`${API_ENDPOINTS.COMPANIES}/${id}`, data)
  },

  delete(id) {
    if (USE_MOCKS) {
      return companyMock.delete(id)
    }
    return client.delete(`${API_ENDPOINTS.COMPANIES}/${id}`)
  }
}
