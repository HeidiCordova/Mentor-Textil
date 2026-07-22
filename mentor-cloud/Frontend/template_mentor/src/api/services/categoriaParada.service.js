import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const categoriaParadaService = {
  getAll(params = {}) {
    return client.get(API_ENDPOINTS.CATEGORIA_PARADAS, { params })
  },
  create(lineaId, tipo = 'programada', data) {
    return client.post(`${API_ENDPOINTS.CATEGORIA_PARADAS}?linea_id=${lineaId}&tipo=${tipo}`, data)
  },
  update(id, lineaId, tipo = 'programada', data) {
    return client.put(`${API_ENDPOINTS.CATEGORIA_PARADAS}/${id}?linea_id=${lineaId}&tipo=${tipo}`, data)
  },
  delete(id, lineaId, tipo = 'programada') {
    return client.delete(`${API_ENDPOINTS.CATEGORIA_PARADAS}/${id}?linea_id=${lineaId}&tipo=${tipo}`)
  }
}
