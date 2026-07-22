import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'
import { mockAnalysis } from '../mocks/analisis.mock'

export const analysisService = {
  async getGeneral(params = {}) {
    if (USE_MOCKS) {
      return mockAnalysis.getGeneral(params)
    }
    return client.get(API_ENDPOINTS.ANALYSIS.GENERAL, { params })
  },

  async getEnergy(params = {}) {
    if (USE_MOCKS) {
      return mockAnalysis.getEnergy(params)
    }
    return client.get(API_ENDPOINTS.ANALYSIS.ENERGY, { params })
  },

  async getProduction(params = {}) {
    if (USE_MOCKS) {
      return mockAnalysis.getProduction(params)
    }
    return client.get(API_ENDPOINTS.ANALYSIS.PRODUCTION, { params })
  },

  async getPareto(params = {}) {
    if (USE_MOCKS) {
      return mockAnalysis.getPareto(params)
    }
    return client.get(API_ENDPOINTS.ANALYSIS.PARETO, { params })
  }
}
