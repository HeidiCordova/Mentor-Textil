export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'
export const API_TIMEOUT = import.meta.env.VITE_API_TIMEOUT || 30000
export const USE_MOCKS = import.meta.env.VITE_USE_MOCKS === 'true'

export const API_ENDPOINTS = {
  AUTH: {
    LOGIN: '/auth/login',
    LOGOUT: '/auth/logout',
    REFRESH: '/auth/refresh',
    ME: '/auth/me'
  },
  USERS: '/usuarios',
  COMPANIES: '/empresas',
  PLANTS: '/plantas',
  LINES: '/lineas',
  DEVICES: '/dispositivos',
  VARIABLES: '/variables',
  ROLES: '/roles',
  TURNOS: '/turnos',
  PRODUCTOS: '/productos',
  PRODUCTOS_CREAR: '/productos-crear',
  PRODUCTO_CARACTERISTICAS: '/producto-caracteristicas',
  LINEA_PRODUCTO_VARS: '/linea-producto-vars',
  LINEA_VAR_CATALOGO: '/linea-var-catalogo',
  VELOCIDAD_NOMINAL: '/velocidad-nominal',
  VELOCIDAD_NOMINAL_LOG: '/velocidad-nominal/log',
  MOTIVOS_VELOCIDAD: '/motivos-velocidad',
  CATEGORIA_PARADAS: '/categoria-paradas',
  ARBOL_PARADAS: '/arbol-paradas',
  LOCACIONES: '/locaciones',
  ANALYSIS: {
    GENERAL: '/analisis/general',
    ENERGY: '/analisis/energia',
    PRODUCTION: '/analisis/produccion',
    PARETO: '/analisis/pareto'
  },
  OEE: {
    SNAPSHOTS: '/oee/snapshots',
    SUMMARY: '/oee/summary',
    LATEST: '/oee/latest'
  },
  STOPS: '/stops',
  REPORTS: '/reportes',
  ALARMS: '/alarmas',
  PRODUCTION: '/produccion',
  DATA: '/datos-recibidos',
  COMMITMENTS: '/compromisos',
  EXCEL: {
    IMPORT: '/excel/import',
    EXPORT: '/excel/export'
  },
  DASHBOARD: {
    STATS: '/dashboard/stats',
    REPORTS: '/dashboard/reportes',
    CHARTS: '/dashboard/graficos'
  },
  INTEGRATION: {
    KEYS: '/integration/keys'
  },
  ENERGY: {
    METERS: '/energy/meters',
    SNAPSHOTS: '/energy/snapshots'
  }
}
