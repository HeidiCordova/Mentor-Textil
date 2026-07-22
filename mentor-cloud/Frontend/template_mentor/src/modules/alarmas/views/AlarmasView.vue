<template>
  <div class="alarmas-view">
    <div class="page-header">
      <h1>Alarmas del Sistema</h1>
      <p>Monitoreo automático de servicios y dispositivos con notificaciones por email</p>
    </div>

    <!-- Tabs -->
    <div class="tabs">
      <button :class="['tab', { active: tab === 'alertas' }]" @click="tab = 'alertas'">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        Alertas del Sistema
      </button>
      <button :class="['tab', { active: tab === 'destinatarios' }]" @click="tab = 'destinatarios'; loadEmails()">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
          <polyline points="22,6 12,13 2,6"/>
        </svg>
        Destinatarios
      </button>
    </div>

    <!-- Tab: Alertas del sistema -->
    <div v-if="tab === 'alertas'">
      <div class="info-banner">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <span>Las siguientes alertas son monitoreadas automáticamente por Prometheus y Grafana. Recibirás un email cuando se disparen y otro cuando se resuelvan.</span>
      </div>

      <div class="alertas-grid">
        <div v-for="alerta in alertasCatalogo" :key="alerta.nombre" class="alerta-card" :class="`sev-${alerta.severidad}`">
          <div class="alerta-card-header">
            <div class="alerta-icon" :class="`icon-${alerta.severidad}`">
              <svg v-if="alerta.severidad === 'critical'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
            </div>
            <div>
              <div class="alerta-nombre">{{ alerta.nombre }}</div>
              <span :class="['sev-badge', `sev-badge-${alerta.severidad}`]">{{ alerta.severidad }}</span>
            </div>
          </div>
          <div class="alerta-detalles">
            <div class="detalle-row">
              <span class="detalle-label">Condición</span>
              <code class="detalle-code">{{ alerta.condicion }}</code>
            </div>
            <div class="detalle-row">
              <span class="detalle-label">Dispara tras</span>
              <span class="detalle-val">{{ alerta.para }}</span>
            </div>
            <div class="detalle-row">
              <span class="detalle-label">Qué significa</span>
              <span class="detalle-val">{{ alerta.descripcion }}</span>
            </div>
            <div class="detalle-row">
              <span class="detalle-label">Qué hacer</span>
              <span class="detalle-val accion">{{ alerta.accion }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="freq-nota">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
        </svg>
        Prometheus evalúa las reglas cada 30 segundos. Las alertas repetidas se agrupan y envían cada 4 horas.
      </div>
    </div>

    <!-- Tab: Destinatarios -->
    <div v-if="tab === 'destinatarios'" class="destinatarios-section">
      <div class="dest-header">
        <h2>Destinatarios de notificaciones</h2>
        <p>Todos los emails listados aquí recibirán alertas cuando se dispare cualquier alarma del sistema.</p>
      </div>

      <div v-if="loadingEmails" class="loading-state">
        <div class="spinner"></div>
        Cargando destinatarios...
      </div>

      <div v-else class="email-manager">
        <div class="email-list">
          <div v-if="emails.length === 0" class="empty-state">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#94a3b8" stroke-width="1.5">
              <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
              <polyline points="22,6 12,13 2,6"/>
            </svg>
            <span>No hay destinatarios configurados</span>
          </div>
          <div v-for="(email, idx) in emails" :key="idx" class="email-row">
            <div class="email-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
            </div>
            <span class="email-text">{{ email }}</span>
            <button class="btn-remove" @click="removeEmail(idx)" title="Eliminar">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
        </div>

        <div class="add-email-form">
          <input
            v-model="nuevoEmail"
            type="email"
            class="email-input"
            placeholder="nuevo@correo.com"
            @keyup.enter="addEmail"
          />
          <button class="btn btn-secondary" @click="addEmail" :disabled="!nuevoEmail">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14M5 12h14"/>
            </svg>
            Agregar
          </button>
        </div>

        <div v-if="saveError" class="error-msg">{{ saveError }}</div>

        <div class="save-row">
          <span class="save-hint">Los cambios se aplican en Grafana inmediatamente al guardar.</span>
          <button class="btn btn-primary" @click="saveEmails" :disabled="saving">
            <svg v-if="!saving" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"/>
              <polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/>
            </svg>
            <div v-else class="spinner-sm"></div>
            {{ saving ? 'Guardando...' : 'Guardar destinatarios' }}
          </button>
        </div>

        <div v-if="saveSuccess" class="success-msg">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          Destinatarios actualizados correctamente
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const tab = ref('alertas')
const emails = ref([])
const nuevoEmail = ref('')
const loadingEmails = ref(false)
const saving = ref(false)
const saveError = ref('')
const saveSuccess = ref(false)

const alertasCatalogo = [
  {
    nombre: 'EdgeDeviceCommandsStuck',
    severidad: 'warning',
    condicion: 'pg_pending_commands_oldest_seconds > 600',
    para: '2 minutos continuos',
    descripcion: 'Un dispositivo edge tiene comandos (ej: sincronización de configuración) pendientes sin entregar por más de 10 minutos. El dispositivo puede estar offline o sin conexión al cloud.',
    accion: 'Verificar conectividad del dispositivo. Revisar logs del edge-gateway en el Jetson.'
  },
  {
    nombre: 'EdgeDeviceCommandsBacklog',
    severidad: 'critical',
    condicion: 'pg_pending_commands_total > 10',
    para: '5 minutos continuos',
    descripcion: 'Se acumularon más de 10 comandos pendientes para un dispositivo. Indica que el dispositivo lleva mucho tiempo desconectado o hay un problema de sincronización masiva.',
    accion: 'Reiniciar el servicio edge-gateway en el Jetson. Si persiste, revisar la conexión de red.'
  },
  {
    nombre: 'CloudServiceDown',
    severidad: 'critical',
    condicion: 'up{job="cloud-ingest"} == 0',
    para: '1 minuto',
    descripcion: 'El servicio cloud-ingest (que recibe datos de producción desde los dispositivos edge) dejó de responder a Prometheus.',
    accion: 'Ejecutar: docker compose restart cloud-ingest en el servidor de producción.'
  },
  {
    nombre: 'HighCPU',
    severidad: 'warning',
    condicion: 'CPU uso > 90%',
    para: '5 minutos continuos',
    descripcion: 'El servidor cloud tiene uso de CPU mayor al 90% sostenido. Puede causar latencia en las APIs y lentitud en la plataforma.',
    accion: 'Identificar el proceso con alto consumo (htop). Evaluar si es necesario escalar el servidor.'
  },
  {
    nombre: 'HighMemory',
    severidad: 'warning',
    condicion: 'RAM uso > 90%',
    para: '5 minutos continuos',
    descripcion: 'La RAM del servidor supera el 90% de uso. Riesgo de que los servicios sean terminados por el sistema operativo (OOM killer).',
    accion: 'Revisar contenedores con consumo alto: docker stats. Reiniciar los servicios afectados.'
  },
  {
    nombre: 'DiskAlmostFull',
    severidad: 'critical',
    condicion: 'Disco "/" > 85%',
    para: '5 minutos continuos',
    descripcion: 'El disco raíz del servidor supera el 85% de uso. Si llega al 100% los servicios y la base de datos fallarán.',
    accion: 'Limpiar logs antiguos: docker system prune. Revisar el crecimiento de la base de datos.'
  }
]

async function loadEmails() {
  loadingEmails.value = true
  saveError.value = ''
  try {
    const resp = await fetch('/api/notifications', {
      headers: { Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}` }
    })
    const data = await resp.json()
    emails.value = data.emails || []
  } catch (e) {
    saveError.value = 'Error al cargar destinatarios'
  } finally {
    loadingEmails.value = false
  }
}

function addEmail() {
  const e = nuevoEmail.value.trim()
  if (!e || !e.includes('@')) return
  if (!emails.value.includes(e)) {
    emails.value.push(e)
  }
  nuevoEmail.value = ''
}

function removeEmail(idx) {
  emails.value.splice(idx, 1)
}

async function saveEmails() {
  saving.value = true
  saveError.value = ''
  saveSuccess.value = false
  try {
    const resp = await fetch('/api/notifications', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`
      },
      body: JSON.stringify({ emails: emails.value })
    })
    if (!resp.ok) {
      const err = await resp.json()
      saveError.value = err.error || 'Error al guardar'
      return
    }
    saveSuccess.value = true
    setTimeout(() => { saveSuccess.value = false }, 4000)
  } catch (e) {
    saveError.value = 'Error de conexión'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.alarmas-view { padding: 1.5rem; }

.page-header { margin-bottom: 1.5rem; }
.page-header h1 { font-size: 2rem; font-weight: 800; color: #1e293b; margin-bottom: 0.4rem; }
.page-header p { color: #64748b; font-size: 1rem; }

/* Tabs */
.tabs { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0; }
.tab {
  display: inline-flex; align-items: center; gap: 0.5rem;
  padding: 0.75rem 1.25rem; background: none; border: none;
  font-size: 0.9rem; font-weight: 600; color: #64748b;
  cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -2px;
  transition: all 0.2s;
}
.tab:hover { color: #3b82f6; }
.tab.active { color: #3b82f6; border-bottom-color: #3b82f6; }

/* Info banner */
.info-banner {
  display: flex; align-items: flex-start; gap: 0.75rem;
  background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 10px;
  padding: 1rem 1.25rem; margin-bottom: 1.5rem;
  font-size: 0.875rem; color: #1e40af;
}
.info-banner svg { flex-shrink: 0; margin-top: 1px; }

/* Alertas grid */
.alertas-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 380px), 1fr)); gap: 1.25rem; margin-bottom: 1.5rem; }

.alerta-card { background: white; border-radius: 12px; border: 1px solid #e2e8f0; overflow: hidden; }
.alerta-card.sev-critical { border-left: 4px solid #ef4444; }
.alerta-card.sev-warning  { border-left: 4px solid #f59e0b; }

.alerta-card-header { display: flex; align-items: center; gap: 1rem; padding: 1rem 1.25rem; background: #f8fafc; border-bottom: 1px solid #e2e8f0; }
.alerta-icon { padding: 0.5rem; border-radius: 8px; display: flex; }
.icon-critical { background: #fee2e2; color: #ef4444; }
.icon-warning  { background: #fef3c7; color: #d97706; }
.alerta-nombre { font-weight: 700; color: #1e293b; font-size: 0.95rem; margin-bottom: 0.25rem; }

.sev-badge { padding: 0.15rem 0.6rem; border-radius: 9999px; font-size: 0.7rem; font-weight: 700; text-transform: uppercase; }
.sev-badge-critical { background: #fee2e2; color: #991b1b; }
.sev-badge-warning  { background: #fef3c7; color: #92400e; }

.alerta-detalles { padding: 1rem 1.25rem; display: flex; flex-direction: column; gap: 0.75rem; }
.detalle-row { display: flex; flex-direction: column; gap: 0.25rem; }
.detalle-label { font-size: 0.75rem; font-weight: 600; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.05em; }
.detalle-code { font-family: monospace; font-size: 0.8rem; background: #f1f5f9; padding: 0.25rem 0.5rem; border-radius: 4px; color: #0f172a; width: fit-content; }
.detalle-val { font-size: 0.875rem; color: #334155; }
.detalle-val.accion { color: #3b82f6; font-weight: 500; }

.freq-nota { display: flex; align-items: center; gap: 0.5rem; font-size: 0.8rem; color: #94a3b8; padding: 0.75rem 1rem; background: #f8fafc; border-radius: 8px; }

/* Destinatarios */
.destinatarios-section { max-width: 680px; }
.dest-header { margin-bottom: 1.5rem; }
.dest-header h2 { font-size: 1.4rem; font-weight: 700; color: #1e293b; margin-bottom: 0.4rem; }
.dest-header p { color: #64748b; font-size: 0.9rem; }

.loading-state { display: flex; align-items: center; gap: 0.75rem; color: #64748b; padding: 2rem 0; }

.email-manager { display: flex; flex-direction: column; gap: 1.25rem; }

.email-list { background: white; border: 1px solid #e2e8f0; border-radius: 12px; overflow: hidden; min-height: 80px; }
.empty-state { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 2rem; color: #94a3b8; font-size: 0.875rem; }

.email-row { display: flex; align-items: center; gap: 0.75rem; padding: 0.875rem 1rem; border-bottom: 1px solid #f1f5f9; transition: background 0.15s; }
.email-row:last-child { border-bottom: none; }
.email-row:hover { background: #f8fafc; }
.email-icon { color: #94a3b8; flex-shrink: 0; }
.email-text { flex: 1; font-size: 0.9rem; color: #334155; }
.btn-remove { background: none; border: none; cursor: pointer; color: #94a3b8; padding: 0.25rem; border-radius: 4px; display: flex; transition: all 0.15s; }
.btn-remove:hover { background: #fee2e2; color: #ef4444; }

.add-email-form { display: flex; gap: 0.75rem; }
.email-input { flex: 1; padding: 0.75rem 1rem; border: 1px solid #e2e8f0; border-radius: 8px; font-size: 0.9rem; outline: none; transition: border-color 0.2s; }
.email-input:focus { border-color: #3b82f6; }

.save-row { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.save-hint { font-size: 0.8rem; color: #94a3b8; }

.btn { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.75rem 1.25rem; border-radius: 8px; font-weight: 600; font-size: 0.875rem; cursor: pointer; transition: all 0.2s; border: none; }
.btn:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary { background: #3b82f6; color: white; }
.btn-primary:not(:disabled):hover { background: #2563eb; box-shadow: 0 4px 12px rgba(59,130,246,0.3); }
.btn-secondary { background: #f1f5f9; color: #475569; border: 1px solid #e2e8f0; }
.btn-secondary:not(:disabled):hover { background: #e2e8f0; }

.error-msg { background: #fee2e2; color: #991b1b; border: 1px solid #fca5a5; border-radius: 8px; padding: 0.75rem 1rem; font-size: 0.875rem; }
.success-msg { display: flex; align-items: center; gap: 0.5rem; background: #dcfce7; color: #166534; border: 1px solid #86efac; border-radius: 8px; padding: 0.75rem 1rem; font-size: 0.875rem; }

.spinner { width: 20px; height: 20px; border: 2px solid #e2e8f0; border-top-color: #3b82f6; border-radius: 50%; animation: spin 0.8s linear infinite; }
.spinner-sm { width: 16px; height: 16px; border: 2px solid rgba(255,255,255,0.4); border-top-color: white; border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
