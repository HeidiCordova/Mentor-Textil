import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const integrationService = {
  async listKeys(empresaId) {
    const params = empresaId ? { empresa_id: empresaId } : {}
    return client.get(API_ENDPOINTS.INTEGRATION.KEYS, { params })
  },

  async createKey(payload) {
    return client.post(API_ENDPOINTS.INTEGRATION.KEYS, payload)
  },

  async revokeKey(id, empresaId) {
    const params = empresaId ? { empresa_id: empresaId } : {}
    return client.delete(`${API_ENDPOINTS.INTEGRATION.KEYS}/${id}`, { params })
  }
}
