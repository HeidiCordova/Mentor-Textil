import axios from 'axios'

const http = axios.create({ baseURL: '/api', timeout: 10000 })

export async function getConfig() {
  const { data } = await http.get('/config')
  return data
}

export async function saveConfig(payload) {
  const { data } = await http.put('/config', payload)
  return data
}

export async function getMeters() {
  const { data } = await http.get('/meters')
  return data
}

export async function createMeter(meterData) {
  const { data } = await http.post('/meters', meterData)
  return data
}

export async function updateMeter(id, meterData) {
  const { data } = await http.patch(`/meters/${id}`, meterData)
  return data
}

export async function deleteMeter(id) {
  const { data } = await http.delete(`/meters/${id}`)
  return data
}

export async function getStats() {
  const { data } = await http.get('/stats')
  return data
}

export async function getSnapshots(filter = '') {
  const { data } = await http.get('/snapshots' + (filter ? `?filter=${filter}` : ''))
  return data
}

export async function getHealth() {
  const { data } = await axios.get('/health', { timeout: 5000 })
  return data
}

const writeHttp = axios.create({ baseURL: '/write-api', timeout: 15000 })

export async function getMeterDeviceConfig(unitId) {
  const { data } = await writeHttp.get(`/meter/${unitId}`)
  return data
}

export async function meterSetCT(unitId, meterId, primary, secondary) {
  const { data } = await writeHttp.post(`/meter/${unitId}/set-ct`, { meter_id: meterId, primary, secondary })
  return data
}

export async function meterSetSys(unitId, meterId, params) {
  const { data } = await writeHttp.post(`/meter/${unitId}/set-sys`, { meter_id: meterId, ...params })
  return data
}

export async function meterSetDir(unitId, meterId, l1, l2, l3) {
  const { data } = await writeHttp.post(`/meter/${unitId}/set-dir`, { meter_id: meterId, l1, l2, l3 })
  return data
}

export async function meterSetTime(unitId, meterId) {
  const { data } = await writeHttp.post(`/meter/${unitId}/set-time`, { meter_id: meterId })
  return data
}

export async function meterReset(unitId, meterId, type) {
  const { data } = await writeHttp.post(`/meter/${unitId}/reset`, { meter_id: meterId, type })
  return data
}

export async function scanMeters(start = 1, end = 100) {
  const { data } = await writeHttp.post('/scan', { start, end }, { timeout: 150000 })
  return data
}

export async function getLiveStatus() {
  const { data } = await writeHttp.get('/live', { timeout: 8000 })
  return data
}

export async function getLatestSnapshot(meterId) {
  const { data } = await http.get(`/latest?meter_id=${encodeURIComponent(meterId)}`)
  return data
}

export async function getMeterHistory(meterId, limit = 48) {
  const { data } = await writeHttp.get(
    `/history?meter_id=${encodeURIComponent(meterId)}&limit=${limit}`,
    { timeout: 8000 }
  )
  return data
}

export async function getAuditLog() {
  const { data } = await writeHttp.get('/audit')
  return data
}
