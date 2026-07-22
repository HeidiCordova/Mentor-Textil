import client from '../client'

export const turnoDiaService = {
  getAll(params = {}) {
    return client.get('/turno-dias', { params })
  },
  save(body) {
    return client.put('/turno-dias', body)
  }
}
