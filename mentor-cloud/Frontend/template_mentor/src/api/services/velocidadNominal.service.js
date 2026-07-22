import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const velocidadNominalService = {
  /** GET /velocidad-nominal?linea_id=X → { data: [...] } */
  getByLinea(params = {}) {
    return client.get(API_ENDPOINTS.VELOCIDAD_NOMINAL, { params })
  },
  /** PUT /velocidad-nominal → { ok, updated } */
  save(items) {
    return client.put(API_ENDPOINTS.VELOCIDAD_NOMINAL, items)
  },
  /** GET /velocidad-nominal/log?linea_id=X&limit=N → { data: [...] } */
  getLog(params = {}) {
    return client.get(API_ENDPOINTS.VELOCIDAD_NOMINAL_LOG, { params })
  },
  /** GET /motivos-velocidad?linea_id=X → { data: [...] } */
  getMotivos(params = {}) {
    return client.get(API_ENDPOINTS.MOTIVOS_VELOCIDAD, { params })
  },
  /** POST /motivos-velocidad?linea_id=X */
  createMotivo(lineaID, body) {
    return client.post(API_ENDPOINTS.MOTIVOS_VELOCIDAD + '?linea_id=' + lineaID, body)
  },
  /** PUT /motivos-velocidad/:id?linea_id=X */
  updateMotivo(lineaID, id, body) {
    return client.put(API_ENDPOINTS.MOTIVOS_VELOCIDAD + '/' + id + '?linea_id=' + lineaID, body)
  },
  /** DELETE /motivos-velocidad/:id?linea_id=X */
  deleteMotivo(lineaID, id) {
    return client.delete(API_ENDPOINTS.MOTIVOS_VELOCIDAD + '/' + id + '?linea_id=' + lineaID)
  }
}
