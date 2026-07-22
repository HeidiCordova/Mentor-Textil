import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'
import { mockDashboard } from '../mocks/dashboard.mock'

export const dashboardService = {
  async getStats(params = {}) {
    if (USE_MOCKS) {
      return mockDashboard.getStats(params)
    }
    return client.get(API_ENDPOINTS.DASHBOARD.STATS, { params })
  },

  async getReports(params = {}) {
    if (USE_MOCKS) {
      return mockDashboard.getReports(params)
    }
    return client.get(API_ENDPOINTS.DASHBOARD.REPORTS, { params })
  },

  async getCharts(params = {}) {
    if (USE_MOCKS) {
      return mockDashboard.getCharts(params)
    }
    return client.get(API_ENDPOINTS.DASHBOARD.CHARTS, { params })
  }
}
