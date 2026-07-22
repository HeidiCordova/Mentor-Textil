export const ROLES = {
  ADMIN: 'ADMIN',
  SUPERVISOR: 'SUPERVISOR',
  OPERATOR: 'OPERATOR',
  VIEWER: 'VIEWER'
}

export const CHART_TYPES = {
  LINE: 'line',
  BAR: 'bar',
  AREA: 'area',
  HEATMAP: 'heatmap',
  GAUGE: 'gauge',
  PARETO: 'pareto'
}

export const AGGREGATION_OPTIONS = [
  { value: '1 minuto', label: '1 minuto' },
  { value: '5 minutos', label: '5 minutos' },
  { value: '15 minutos', label: '15 minutos' },
  { value: '30 minutos', label: '30 minutos' },
  { value: '1 hora', label: '1 hora' },
  { value: '1 día', label: '1 día' }
]

export const DATE_PRESETS = [
  { value: 'today', label: 'Hoy' },
  { value: 'yesterday', label: 'Ayer' },
  { value: 'last7days', label: 'Últimos 7 días' },
  { value: 'last30days', label: 'Últimos 30 días' },
  { value: 'thisWeek', label: 'Esta semana' },
  { value: 'thisMonth', label: 'Este mes' },
  { value: 'custom', label: 'Personalizado' }
]

export const FILE_TYPES = {
  EXCEL: ['xlsx', 'xls', 'csv'],
  PDF: ['pdf'],
  IMAGE: ['jpg', 'jpeg', 'png', 'svg']
}

export const MAX_FILE_SIZE = 10 * 1024 * 1024

export const TOAST_DURATION = {
  SHORT: 2000,
  MEDIUM: 3000,
  LONG: 5000
}
