export function formatDate(date, format = 'dd/MM/yyyy') {
  if (!date) return ''
  
  const d = new Date(date)
  const day = String(d.getDate()).padStart(2, '0')
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const year = d.getFullYear()
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  const seconds = String(d.getSeconds()).padStart(2, '0')

  return format
    .replace('dd', day)
    .replace('MM', month)
    .replace('yyyy', year)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
}

export function formatDateTime(date) {
  return formatDate(date, 'dd/MM/yyyy HH:mm')
}

export function formatNumber(number, decimals = 0) {
  if (number === null || number === undefined) return ''
  return Number(number).toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

export function formatCurrency(amount, currency = 'PEN') {
  if (amount === null || amount === undefined) return ''
  return new Intl.NumberFormat('es-PE', {
    style: 'currency',
    currency
  }).format(amount)
}

export function formatPercentage(value, decimals = 1) {
  if (value === null || value === undefined) return ''
  return `${Number(value).toFixed(decimals)}%`
}

export function truncate(text, maxLength = 50) {
  if (!text) return ''
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

export function capitalize(text) {
  if (!text) return ''
  return text.charAt(0).toUpperCase() + text.slice(1).toLowerCase()
}

export function formatFileSize(bytes) {
  if (bytes === 0) return '0 Bytes'
  
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}
