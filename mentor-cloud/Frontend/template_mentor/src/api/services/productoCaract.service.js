import client from '../client'
import { API_ENDPOINTS } from '../endpoints'

export const productoCaractService = {
  // Obtiene columnas (variables) + valores por línea
  getCaracteristicas(params = {}) {
    return client.get(API_ENDPOINTS.PRODUCTO_CARACTERISTICAS, { params })
  },
  // Guarda valores de características
  saveCaracteristicas(body) {
    return client.put(API_ENDPOINTS.PRODUCTO_CARACTERISTICAS, body)
  },
  // Obtiene columnas configuradas para una línea
  getLineaVars(params = {}) {
    return client.get(API_ENDPOINTS.LINEA_PRODUCTO_VARS, { params })
  },
  // Guarda columnas configuradas para una línea
  saveLineaVars(body) {
    return client.put(API_ENDPOINTS.LINEA_PRODUCTO_VARS, body)
  },
  // CRUD productos master
  getAllProductos(params = {}) {
    return client.get(API_ENDPOINTS.PRODUCTOS, { params })
  },
  crearProducto(data) {
    return client.post(API_ENDPOINTS.PRODUCTOS_CREAR, data)
  },
  eliminarProducto(id) {
    return client.delete(`${API_ENDPOINTS.PRODUCTOS_CREAR}/${id}`)
  },
  // Catálogo de valores permitidos por línea+variable
  getCatalogo(params = {}) {
    return client.get(API_ENDPOINTS.LINEA_VAR_CATALOGO, { params })
  },
  getCatalogoAll(params = {}) {
    return client.get(API_ENDPOINTS.LINEA_VAR_CATALOGO + '/all', { params })
  },
  saveCatalogo(body) {
    return client.put(API_ENDPOINTS.LINEA_VAR_CATALOGO, body)
  },
  getVariablesLinea(params = {}) {
    return client.get('/variables-linea', { params })
  }
}
