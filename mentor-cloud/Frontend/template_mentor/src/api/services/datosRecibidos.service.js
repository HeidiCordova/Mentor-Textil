import client from '../client'

export const datosRecibidosService = {
  /**
   * Lista eventos crudos almacenados en ingest.raw_events (líneas OEE).
   * @param {Object} params - { device_id, event_type, from, to, limit, offset }
   */
  async listar(params = {}) {
    return client.get('/datos-recibidos', { params })
  },

  /**
   * Lista snapshots de energía para líneas de tipo Energía (medidores MC60).
   * @param {Object} params - { planta_id, from, to, limit, offset }
   */
  async listarEnergia(params = {}) {
    return client.get('/energy/snapshots', { params })
  },

  /**
   * OEE snapshots procesados con métricas calculadas.
   * @param {Object} params - { linea_id, empresa_id, device_id, from, to, limit }
   */
  async snapshots(params = {}) {
    return client.get('/oee/snapshots', { params })
  }
}
