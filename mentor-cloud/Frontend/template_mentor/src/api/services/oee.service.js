import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const oeeService = {
  async getSnapshots(params = {}) {
    return client.get(API_ENDPOINTS.OEE.SNAPSHOTS, { params })
  },

  async getSummary(params = {}) {
    return client.get(API_ENDPOINTS.OEE.SUMMARY, { params })
  },

  async getLatest(lineaId, plantaId = null) {
    const params = { linea_id: lineaId }
    if (plantaId) params.planta_id = plantaId
    return client.get(API_ENDPOINTS.OEE.LATEST, { params })
  }
}
