import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const stopsService = {
  list(params = {}) {
    return client.get(API_ENDPOINTS.STOPS, { params })
  }
}
