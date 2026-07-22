import client from '../client'

export const canvasOeeService = {
  get(params = {}) {
    return client.get('/canvas-oee', { params })
  },

  save(body) {
    return client.put('/canvas-oee', body)
  },

  resetDefault(body = {}) {
    return client.post('/canvas-oee/reset-default', body)
  }
}
