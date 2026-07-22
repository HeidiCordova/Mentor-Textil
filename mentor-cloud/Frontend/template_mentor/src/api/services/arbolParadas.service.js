import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const arbolParadasService = {
  get(lineaId) {
    return client.get(API_ENDPOINTS.ARBOL_PARADAS, { params: { linea_id: lineaId } })
  },
  importar(data) {
    return client.post(`${API_ENDPOINTS.ARBOL_PARADAS}/importar`, data)
  },
  exportar(lineaId) {
    return client.get(`${API_ENDPOINTS.ARBOL_PARADAS}/exportar`, { params: { linea_id: lineaId } })
  }
}
