import axios from 'axios'

// Todas las URLs usan rutas relativas que nginx proxea a los contenedores internos.
// Esto permite acceder desde cualquier navegador sin importar la IP del Jetson.
const CONFIG_SERVICE_URL = import.meta.env.VITE_CONFIG_SERVICE_URL || '/api/config'
const RESILIENCIA_URL   = import.meta.env.VITE_RESILIENCIA_URL    || '/api/resiliencia'
const ENVIADOR_URL      = import.meta.env.VITE_ENVIADOR_URL       || '/api/enviador'
const DETECTOR_URL      = import.meta.env.VITE_DETECTOR_URL       || '/api/detector'
const GATEWAY_URL       = import.meta.env.VITE_GATEWAY_URL        || '/api/gateway'

export const configService = {
  /** Lista de linea_id registradas (excluye _system) */
  async getLines() {
    const response = await axios.get(`${CONFIG_SERVICE_URL}/config/lines`)
    return response.data
  },

  /** Config completa de una línea por linea_id (int) */
  async getConfig(lineaId) {
    const response = await axios.get(`${CONFIG_SERVICE_URL}/config`, {
      params: { linea_id: lineaId }
    })
    return response.data
  },

  /** Guarda/crea config de una línea por linea_id (int) */
  async updateConfig(config, lineaId) {
    const response = await axios.put(`${CONFIG_SERVICE_URL}/config`, config, {
      params: { linea_id: lineaId }
    })
    return response.data
  },

  /** Fuerza re-aprendizaje del fondo del detector de presencia (sin pasar por config store) */
  async resetPresenceBackground() {
    const response = await axios.post(`${DETECTOR_URL}/reset-presence`)
    return response.data
  },

  async getConfigVersion(lineaId) {
    const response = await axios.get(`${CONFIG_SERVICE_URL}/config/version`, {
      params: { linea_id: lineaId }
    })
    return response.data
  },

  async deleteLine(lineaId) {
    const response = await axios.delete(`${CONFIG_SERVICE_URL}/config/lines`, {
      params: { linea_id: lineaId }
    })
    return response.data
  },

  /** device_id del Jetson (identifica la placa física) */
  async getDeviceId() {
    const response = await axios.get(`${CONFIG_SERVICE_URL}/config/device-id`)
    return response.data
  },

  async setDeviceId(deviceId) {
    const response = await axios.put(`${CONFIG_SERVICE_URL}/config/device-id`, { device_id: deviceId })
    return response.data
  },

  /** Defaults globales del sistema (linea_id=0) */
  async getSystemDefaults() {
    try {
      const response = await axios.get(`${CONFIG_SERVICE_URL}/config/system`)
      return response.data
    } catch {
      return {}
    }
  },

  async setSystemDefaults(data) {
    const response = await axios.put(`${CONFIG_SERVICE_URL}/config/system`, data)
    return response.data
  },

}


// ── Edge Gateway service ────────────────────────────────────────────────────
// Centraliza todas las llamadas al edge-gateway local (puerto 8005 vía nginx).
export const gatewayService = {
  /** Estado compacto del dispositivo: uptime, buffer, cloud_connected, config_version */
  async getStatus() {
    const response = await axios.get(`${GATEWAY_URL}/edge/status`, { timeout: 5000 })
    return response.data
  },

  /** Health agregado de todos los sub-servicios edge */
  async getAggregatedHealth() {
    const response = await axios.get(`${GATEWAY_URL}/edge/health`, { timeout: 7000 })
    return response.data
  },

  /**
   * Conteo de registros sincronizados por recurso.
   * (solo los 3 recursos con endpoint dedicado; el resto se infiere de commands)
   */
  async getSyncCounts() {
    const [cats, prods, turnos] = await Promise.allSettled([
      axios.get(`${GATEWAY_URL}/edge/catalogs/sync/categorias`, { timeout: 5000 }),
      axios.get(`${GATEWAY_URL}/edge/catalogs/sync/productos`,  { timeout: 5000 }),
      axios.get(`${GATEWAY_URL}/edge/catalogs/sync/turnos`,     { timeout: 5000 }),
    ])
    return {
      categorias: cats.status  === 'fulfilled' ? (cats.value.data.data?.length  ?? 0) : null,
      productos:  prods.status === 'fulfilled' ? (prods.value.data.data?.length ?? 0) : null,
      turnos:     turnos.status=== 'fulfilled' ? (turnos.value.data.data?.length?? 0) : null,
    }
  },

  /** Últimos N comandos recibidos (incluye SYNCs, calibración, etc.) */
  async getRecentCommands(limit = 20) {
    const response = await axios.get(`${GATEWAY_URL}/edge/commands?limit=${limit}`, { timeout: 5000 })
    return Array.isArray(response.data) ? response.data : []
  },

  /** Dispara calibración automática en el detector vía gateway */
  async startCalibration() {
    const response = await axios.post(`${GATEWAY_URL}/edge/calibration/start`, null, { timeout: 15000 })
    return response.data
  },

  /** Métricas de hardware del Jetson: CPU, RAM, disco, GPU, temperaturas */
  async getSystemMetrics() {
    const response = await axios.get(`${GATEWAY_URL}/edge/system`, { timeout: 4000 })
    return response.data
  },
}

export const resilienciaService = {
  async getBufferHealth() {
    const response = await axios.get(`${RESILIENCIA_URL}/health/buffer`)
    return response.data
  },

  async getHealth() {
    const response = await axios.get(`${RESILIENCIA_URL}/health`)
    return response.data
  }
}

export const detectorService = {
  async getHealth() {
    const response = await axios.get(`${DETECTOR_URL}/health`)
    return response.data
  }
}

export const enviadorService = {
  async getHealth() {
    const response = await axios.get(`${ENVIADOR_URL}/health`)
    return response.data
  }
}

export const cameraService = {
  /**
   * Verifica salud de la cámara RTSP via edge-gateway.
   * @param {string} [url] URL RTSP explícita; si se omite usa la configurada en el gateway
   * @returns {Promise<CameraHealthResult>}
   */
  async checkHealth(url = null) {
    const params = url ? { url } : {}
    const response = await axios.get(`${GATEWAY_URL}/edge/camera/health`, {
      params,
      timeout: 20000  // 20 s: el probe puede tardar
    })
    return response.data
  }
}

