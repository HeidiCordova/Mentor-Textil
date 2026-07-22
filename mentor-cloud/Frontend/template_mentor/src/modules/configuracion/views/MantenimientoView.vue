<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import client from '@/api/client'

const healthData    = ref(null)
const serverMetrics = ref(null)
const lastChecked   = ref(null)
const isRefreshing  = ref(false)
const fetchError    = ref(null)
let interval = null

const GRAFANA_URL     = 'http://152.53.253.59:3000'
const GRAFANA_USER    = 'admin'
const GRAFANA_PASS    = 'Mentor2026'
const PROMETHEUS_URL  = 'http://152.53.253.59:9090'

const SERVICE_INFO = {
  gateway:   { label: 'Cloud Gateway',   desc: 'Enrutamiento JWT y proxy de servicios' },
  identity:  { label: 'Cloud Identity',  desc: 'Autenticación, usuarios y roles' },
  config:    { label: 'Cloud Config',    desc: 'Planta, líneas y dispositivos' },
  ingest:    { label: 'Cloud Ingest',    desc: 'Ingesta de datos desde Edge' },
  analytics: { label: 'Cloud Analytics', desc: 'Dashboard, análisis y reportes' },
}

const services = computed(() => {
  if (!healthData.value) return []
  const list = []
  const gw = healthData.value.gateway
  if (gw) list.push({ key: 'gateway', ...SERVICE_INFO.gateway, status: gw.status, latency: gw.latency_ms })
  const svcs = healthData.value.services || {}
  for (const key of ['identity', 'config', 'ingest', 'analytics']) {
    const s = svcs[key]
    if (s) list.push({ key, ...SERVICE_INFO[key], status: s.status, latency: s.latency_ms })
  }
  return list
})

const allOk = computed(() =>
  services.value.length > 0 && services.value.every(s => s.status === 'ok')
)

const pct  = key => serverMetrics.value ? Math.round(serverMetrics.value[key] ?? 0) : null
const val  = (key, dec = 1) => serverMetrics.value ? (serverMetrics.value[key] ?? 0).toFixed(dec) : '—'
const gaugeColor = p => p == null ? '#e5e7eb' : p < 60 ? '#10b981' : p < 85 ? '#f59e0b' : '#ef4444'
const cpuClass   = p => p == null ? '' : p < 60 ? 'val-ok' : p < 85 ? 'val-warn' : 'val-crit'

const fmtUptime = s => {
  if (!s) return '—'
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60)
  return d > 0 ? `${d}d ${h}h` : h > 0 ? `${h}h ${m}m` : `${m}m`
}
const fmtNet = bps => {
  if (bps == null) return '—'
  if (bps < 1024) return `${bps.toFixed(0)} B/s`
  if (bps < 1048576) return `${(bps / 1024).toFixed(1)} KB/s`
  return `${(bps / 1048576).toFixed(2)} MB/s`
}

const fetchAll = async () => {
  if (isRefreshing.value) return
  isRefreshing.value = true
  fetchError.value = null
  try {
    const [health, metrics] = await Promise.all([
      client.get('/health'),
      client.get('/server-metrics'),
    ])
    healthData.value    = health
    serverMetrics.value = metrics
    lastChecked.value   = new Date()
  } catch (e) {
    fetchError.value = e?.message || 'Error de conexión'
  } finally {
    isRefreshing.value = false
  }
}

onMounted(() => {
  fetchAll()
  interval = setInterval(fetchAll, 30000)
})

onUnmounted(() => clearInterval(interval))

const fmtTime = d =>
  d ? d.toLocaleTimeString('es-MX', { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '--:--:--'

const latencyClass = ms => {
  if (!ms || ms <= 0) return 'lat-zero'
  if (ms < 50)  return 'lat-fast'
  if (ms < 200) return 'lat-ok'
  return 'lat-slow'
}

const latencyBar = ms => Math.min((ms / 300) * 100, 100)
</script>

<template>
  <div class="monitoreo-view">
    <div class="view-header">
      <h1 class="view-title">MONITOREO</h1>
    </div>

    <div class="status-banner" :class="allOk ? 'banner-ok' : 'banner-warn'">
      <span class="banner-dot"></span>
      <span class="banner-text">
        {{ allOk ? 'Todos los servicios operativos' : 'Uno o más servicios con problemas' }}
      </span>
      <span class="banner-time">Última verificación: {{ fmtTime(lastChecked) }}</span>
      <button class="btn-refresh" @click="fetchHealth" :disabled="isRefreshing">
        <svg :class="{ spinning: isRefreshing }" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
        </svg>
        Actualizar
      </button>
    </div>

    <div v-if="fetchError && !healthData" class="error-box">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
        <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
      </svg>
      No se pudo conectar con el gateway: {{ fetchError }}
    </div>

    <!-- Skeleton servicios -->
    <div v-if="!healthData && !fetchError" class="skeleton-grid">
      <div v-for="n in 5" :key="n" class="skeleton-card"></div>
    </div>

    <!-- Tarjetas de servicios -->
    <div v-if="healthData" class="cards-grid">
      <div
        v-for="svc in services"
        :key="svc.key"
        class="svc-card"
        :class="svc.status === 'ok' ? 'card-ok' : 'card-err'"
      >
        <div class="card-top">
          <div class="card-icon" :class="svc.status === 'ok' ? 'icon-ok' : 'icon-err'">
            <svg v-if="svc.key === 'gateway'" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="16 3 21 3 21 8"/><line x1="4" y1="20" x2="21" y2="3"/>
              <polyline points="21 16 21 21 16 21"/><line x1="15" y1="15" x2="21" y2="21"/>
            </svg>
            <svg v-else-if="svc.key === 'identity'" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <svg v-else-if="svc.key === 'config'" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
            <svg v-else-if="svc.key === 'ingest'" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="20" x2="18" y2="10"/>
              <line x1="12" y1="20" x2="12" y2="4"/>
              <line x1="6"  y1="20" x2="6"  y2="14"/>
            </svg>
          </div>
          <div class="card-meta">
            <div class="card-title">{{ svc.label }}</div>
            <div class="card-desc">{{ svc.desc }}</div>
          </div>
          <div class="badge" :class="svc.status === 'ok' ? 'badge-ok' : 'badge-err'">
            {{ svc.status === 'ok' ? 'Operativo' : 'Error' }}
          </div>
        </div>
        <div class="card-bottom">
          <div class="lat-row">
            <span class="lat-label">Latencia</span>
            <span class="lat-ms" :class="latencyClass(svc.latency)">
              {{ svc.key === 'gateway' ? '—' : (svc.latency ?? 0) + ' ms' }}
            </span>
          </div>
          <div v-if="svc.key !== 'gateway'" class="lat-track">
            <div class="lat-fill" :class="latencyClass(svc.latency)" :style="{ width: latencyBar(svc.latency) + '%' }"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- Sección servidor -->
    <div class="section-title">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/>
        <line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/>
      </svg>
      Servidor
    </div>

    <div class="server-grid">
      <!-- CPU -->
      <div class="server-card">
        <div class="sc-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/>
            <line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/>
            <line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/>
            <line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/>
            <line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/>
          </svg>
          <span>CPU</span>
        </div>
        <div class="gauge-wrap">
          <svg viewBox="0 0 120 70" class="gauge-svg">
            <path d="M10 65 A50 50 0 0 1 110 65" fill="none" stroke="#e5e7eb" stroke-width="10" stroke-linecap="round"/>
            <path d="M10 65 A50 50 0 0 1 110 65" fill="none"
              :stroke="gaugeColor(pct('cpu_usage_pct'))"
              stroke-width="10" stroke-linecap="round"
              :stroke-dasharray="`${(pct('cpu_usage_pct') ?? 0) * 1.57} 157`"/>
            <text x="60" y="60" text-anchor="middle" class="gauge-val">{{ pct('cpu_usage_pct') != null ? pct('cpu_usage_pct') + '%' : '—' }}</text>
          </svg>
        </div>
        <div class="sc-footer">
          <span>Carga 1m</span>
          <span :class="cpuClass(pct('cpu_usage_pct'))">{{ val('load1', 2) }}</span>
        </div>
      </div>

      <!-- RAM -->
      <div class="server-card">
        <div class="sc-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 19v-3"/><path d="M10 19v-3"/><path d="M14 19v-3"/><path d="M18 19v-3"/>
            <rect x="2" y="6" width="20" height="10" rx="2"/>
          </svg>
          <span>RAM</span>
        </div>
        <div class="gauge-wrap">
          <svg viewBox="0 0 120 70" class="gauge-svg">
            <path d="M10 65 A50 50 0 0 1 110 65" fill="none" stroke="#e5e7eb" stroke-width="10" stroke-linecap="round"/>
            <path d="M10 65 A50 50 0 0 1 110 65" fill="none"
              :stroke="gaugeColor(pct('ram_usage_pct'))"
              stroke-width="10" stroke-linecap="round"
              :stroke-dasharray="`${(pct('ram_usage_pct') ?? 0) * 1.57} 157`"/>
            <text x="60" y="60" text-anchor="middle" class="gauge-val">{{ pct('ram_usage_pct') != null ? pct('ram_usage_pct') + '%' : '—' }}</text>
          </svg>
        </div>
        <div class="sc-footer">
          <span>{{ val('ram_used_gb') }} / {{ val('ram_total_gb') }} GB</span>
        </div>
      </div>

      <!-- Disco -->
      <div class="server-card">
        <div class="sc-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
            <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
          </svg>
          <span>Disco</span>
        </div>
        <div class="gauge-wrap">
          <svg viewBox="0 0 120 70" class="gauge-svg">
            <path d="M10 65 A50 50 0 0 1 110 65" fill="none" stroke="#e5e7eb" stroke-width="10" stroke-linecap="round"/>
            <path d="M10 65 A50 50 0 0 1 110 65" fill="none"
              :stroke="gaugeColor(pct('disk_usage_pct'))"
              stroke-width="10" stroke-linecap="round"
              :stroke-dasharray="`${(pct('disk_usage_pct') ?? 0) * 1.57} 157`"/>
            <text x="60" y="60" text-anchor="middle" class="gauge-val">{{ pct('disk_usage_pct') != null ? pct('disk_usage_pct') + '%' : '—' }}</text>
          </svg>
        </div>
        <div class="sc-footer">
          <span>{{ val('disk_used_gb', 0) }} / {{ val('disk_total_gb', 0) }} GB</span>
        </div>
      </div>

      <!-- Uptime -->
      <div class="server-card server-card-sm">
        <div class="sc-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
          </svg>
          <span>Uptime</span>
        </div>
        <div class="stat-big">{{ fmtUptime(serverMetrics?.uptime_seconds) }}</div>
      </div>

      <!-- Red RX -->
      <div class="server-card server-card-sm">
        <div class="sc-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="8 17 12 21 16 17"/><line x1="12" y1="21" x2="12" y2="3"/>
          </svg>
          <span>Red RX</span>
        </div>
        <div class="stat-big">{{ fmtNet(serverMetrics?.net_rx_bps) }}</div>
      </div>

      <!-- Red TX -->
      <div class="server-card server-card-sm">
        <div class="sc-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="16 7 12 3 8 7"/><line x1="12" y1="3" x2="12" y2="21"/>
          </svg>
          <span>Red TX</span>
        </div>
        <div class="stat-big">{{ fmtNet(serverMetrics?.net_tx_bps) }}</div>
      </div>
    </div>

    <!-- Panel herramientas -->
    <div class="section-title">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M3 3h7v7H3z"/><path d="M14 3h7v7h-7z"/><path d="M14 14h7v7h-7z"/><path d="M3 14h7v7H3z"/>
      </svg>
      Herramientas de observabilidad
    </div>

    <div class="tools-grid">
      <a :href="GRAFANA_URL" target="_blank" rel="noopener" class="tool-card">
        <div class="tool-icon tool-icon-grafana">
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="20" x2="18" y2="10"/>
            <line x1="12" y1="20" x2="12" y2="4"/>
            <line x1="6"  y1="20" x2="6"  y2="14"/>
          </svg>
        </div>
        <div class="tool-body">
          <div class="tool-name">Grafana</div>
          <div class="tool-url">{{ GRAFANA_URL }}</div>
          <div class="tool-creds">
            <span class="cred-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
              </svg>
              {{ GRAFANA_USER }}
            </span>
            <span class="cred-item">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
              {{ GRAFANA_PASS }}
            </span>
          </div>
        </div>
        <svg class="tool-arrow" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="7" y1="17" x2="17" y2="7"/><polyline points="7 7 17 7 17 17"/>
        </svg>
      </a>

      <!-- Prometheus -->
      <a :href="PROMETHEUS_URL" target="_blank" rel="noopener" class="tool-card">
        <div class="tool-icon tool-icon-prometheus">
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="9"/>
            <line x1="12" y1="3" x2="12" y2="7"/>
            <line x1="12" y1="17" x2="12" y2="21"/>
            <line x1="3" y1="12" x2="7" y2="12"/>
            <line x1="17" y1="12" x2="21" y2="12"/>
          </svg>
        </div>
        <div class="tool-body">
          <div class="tool-name">Prometheus</div>
          <div class="tool-url">{{ PROMETHEUS_URL }}</div>
          <div class="tool-creds">
            <span class="cred-item" style="color:#6b7280">Sin autenticación</span>
          </div>
        </div>
        <svg class="tool-arrow" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="7" y1="17" x2="17" y2="7"/><polyline points="7 7 17 7 17 17"/>
        </svg>
      </a>
    </div>

    <div class="legend">
      <span class="leg-item"><span class="leg-dot dot-fast"></span> &lt; 50 ms rápido</span>
      <span class="leg-item"><span class="leg-dot dot-ok"></span> 50–200 ms normal</span>
      <span class="leg-item"><span class="leg-dot dot-slow"></span> &gt; 200 ms lento</span>
      <span class="leg-note">Auto-actualización cada 30 s</span>
    </div>
  </div>
</template>

<style scoped>
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.monitoreo-view {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.view-header { margin-bottom: 0; }
.view-title {
  font-size: 1.875rem;
  font-weight: 700;
  color: #111827;
}

.status-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1.25rem;
  border-radius: 0.625rem;
  font-size: 0.9rem;
  font-weight: 500;
  flex-wrap: wrap;
}
.banner-ok   { background: #d1fae5; color: #065f46; }
.banner-warn { background: #fef3c7; color: #92400e; }
.banner-dot {
  width: 10px; height: 10px; border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}
.banner-text { flex: 1; }
.banner-time { font-size: 0.8rem; opacity: 0.8; }
.btn-refresh {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.375rem 0.875rem;
  background: white;
  border: 1px solid currentColor;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.8rem;
  font-weight: 600;
  color: inherit;
  transition: opacity 0.2s;
}
.btn-refresh:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-refresh:not(:disabled):hover { opacity: 0.75; }
.spinning { animation: spin 1s linear infinite; }

.error-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 1.25rem;
  background: #fee2e2;
  color: #7f1d1d;
  border-radius: 0.625rem;
  font-size: 0.9rem;
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}
.skeleton-card {
  height: 130px;
  background: linear-gradient(90deg, #e5e7eb 25%, #f3f4f6 50%, #e5e7eb 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 0.75rem;
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}
.svc-card {
  background: white;
  border-radius: 0.75rem;
  border: 2px solid transparent;
  padding: 1.25rem;
  box-shadow: 0 1px 4px rgba(0,0,0,.07);
  display: flex;
  flex-direction: column;
  gap: 1rem;
  transition: box-shadow 0.2s;
}
.svc-card:hover { box-shadow: 0 4px 12px rgba(0,0,0,.12); }
.card-ok  { border-color: #bbf7d0; }
.card-err { border-color: #fecaca; }

.card-top {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}
.card-icon {
  flex-shrink: 0;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}
.icon-ok  { background: #d1fae5; color: #065f46; }
.icon-err { background: #fee2e2; color: #991b1b; }

.card-meta  { flex: 1; min-width: 0; }
.card-title { font-size: 0.95rem; font-weight: 700; color: #111827; }
.card-desc  { font-size: 0.78rem; color: #6b7280; margin-top: 0.2rem; }
.badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.6rem;
  border-radius: 9999px;
  white-space: nowrap;
  flex-shrink: 0;
}
.badge-ok  { background: #d1fae5; color: #065f46; }
.badge-err { background: #fee2e2; color: #991b1b; }

.card-bottom { display: flex; flex-direction: column; gap: 0.4rem; }
.lat-row { display: flex; align-items: center; justify-content: space-between; }
.lat-label { font-size: 0.78rem; color: #6b7280; }
.lat-ms    { font-size: 0.875rem; font-weight: 700; }
.lat-zero  { color: #9ca3af; }
.lat-fast  { color: #059669; }
.lat-ok    { color: #2563eb; }
.lat-slow  { color: #dc2626; }

.lat-track {
  height: 6px;
  background: #e5e7eb;
  border-radius: 9999px;
  overflow: hidden;
}
.lat-fill {
  height: 100%;
  border-radius: 9999px;
  transition: width 0.5s ease;
}
.lat-fill.lat-fast { background: #10b981; }
.lat-fill.lat-ok   { background: #3b82f6; }
.lat-fill.lat-slow { background: #ef4444; }

.legend {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  padding: 0.75rem 1rem;
  background: #f9fafb;
  border-radius: 0.5rem;
  font-size: 0.78rem;
  color: #6b7280;
  flex-wrap: wrap;
}
.leg-item { display: flex; align-items: center; gap: 0.4rem; }
.leg-dot  { width: 8px; height: 8px; border-radius: 50%; }
.dot-fast { background: #10b981; }
.dot-ok   { background: #3b82f6; }
.dot-slow { background: #ef4444; }
.leg-note { margin-left: auto; font-style: italic; }

/* Sección títulos */
.section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  font-weight: 700;
  color: #374151;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-top: 0.5rem;
}

/* Grid servidor */
.server-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: 1rem;
}
@media (max-width: 640px) {
  .server-grid { grid-template-columns: 1fr; }
}

.server-card {
  background: white;
  border-radius: 0.75rem;
  border: 1px solid #e5e7eb;
  padding: 1rem 1rem 0.75rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.06);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.server-card-sm { grid-column: span 1; }

.sc-header {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.gauge-wrap { display: flex; justify-content: center; }
.gauge-svg  { width: 120px; height: 70px; overflow: visible; }
.gauge-val  { font-size: 16px; font-weight: 800; fill: #111827; }

.sc-footer {
  display: flex;
  justify-content: space-between;
  font-size: 0.78rem;
  color: #6b7280;
}
.val-ok   { color: #059669; font-weight: 700; }
.val-warn { color: #d97706; font-weight: 700; }
.val-crit { color: #dc2626; font-weight: 700; }

.stat-big {
  font-size: 1.5rem;
  font-weight: 800;
  color: #111827;
  text-align: center;
  padding: 0.5rem 0;
}

/* Herramientas */
.tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
  gap: 1rem;
}

.tool-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
  text-decoration: none;
  color: inherit;
  box-shadow: 0 1px 3px rgba(0,0,0,.06);
  transition: box-shadow 0.2s, border-color 0.2s;
}
.tool-card:hover {
  box-shadow: 0 4px 12px rgba(0,0,0,.1);
  border-color: #f97316;
}

.tool-icon {
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 0.625rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.tool-icon-grafana    { background: #fff4ed; color: #f97316; }
.tool-icon-prometheus { background: #fef3f2; color: #e35b26; }

.tool-body { flex: 1; min-width: 0; }
.tool-name { font-size: 0.95rem; font-weight: 700; color: #111827; }
.tool-url  { font-size: 0.78rem; color: #6b7280; margin-top: 0.1rem; word-break: break-all; }
.tool-creds {
  display: flex;
  gap: 1rem;
  margin-top: 0.5rem;
  flex-wrap: wrap;
}
.cred-item {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.78rem;
  background: #f3f4f6;
  border-radius: 0.375rem;
  padding: 0.2rem 0.5rem;
  color: #374151;
  font-family: monospace;
}
.tool-arrow { color: #9ca3af; flex-shrink: 0; }
</style>
