import client from '../client'
import { API_ENDPOINTS } from '../endpoints'
import { USE_MOCKS } from '../endpoints'
import { mockAuth } from '../mocks/auth.mock'

export const authService = {
  async login(credentials) {
    if (USE_MOCKS) {
      return mockAuth.login(credentials)
    }
    return client.post(API_ENDPOINTS.AUTH.LOGIN, credentials)
  },

  async logout() {
    if (USE_MOCKS) {
      return mockAuth.logout()
    }
    return client.post(API_ENDPOINTS.AUTH.LOGOUT)
  },

  async refreshToken(refreshToken) {
    if (USE_MOCKS) {
      return mockAuth.refresh(refreshToken)
    }
    return client.post(API_ENDPOINTS.AUTH.REFRESH, { refreshToken })
  },

  async getCurrentUser() {
    if (USE_MOCKS) {
      return mockAuth.getCurrentUser()
    }
    return client.get(API_ENDPOINTS.AUTH.ME)
  }
}
