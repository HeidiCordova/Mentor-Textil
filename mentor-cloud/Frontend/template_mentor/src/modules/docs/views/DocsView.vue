<template>
  <div class="docs-view">

    <div class="docs-header">
      <div>
        <h1 class="docs-title">Documentacion de Usuario</h1>
        <p class="docs-subtitle">Guía paso a paso para configurar Mentor Textil desde cero</p>
      </div>
      <div class="docs-tabs">
        <button v-for="tab in tabs" :key="tab.id"
          :class="['tab-btn', { 'tab-btn-active': activeTab === tab.id }]"
          @click="activeTab = tab.id">
          <span v-html="tab.icon"></span>
          {{ tab.label }}
        </button>
      </div>
    </div>

    <!-- Tab: Vision General -->
    <div v-if="activeTab === 'overview'" class="tab-content">
      <div class="content-grid">

        <div class="doc-section">
          <h2 class="section-title">Arquitectura del sistema</h2>
          <p class="section-desc">Mentor Textil opera en dos capas: el edge (planta) y el cloud (servidor central). Ambas capas se comunican vía HTTPS usando identificadores únicos por línea de producción.</p>

          <div class="arch-diagram">
            <div class="arch-col">
              <div class="arch-box arch-edge">
                <div class="arch-box-header">
                  <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor"><rect x="2" y="3" width="20" height="14" rx="2" stroke-width="2"/><path d="M8 21h8M12 17v4" stroke-width="2" stroke-linecap="round"/></svg>
                  Jetson / Raspberry (Edge)
                </div>
                <ul class="arch-list">
                  <li>edge-gateway</li>
                  <li>enviador → cloud</li>
                  <li>vision-event-detector</li>
                  <li>ui-local :8080</li>
                  <li>PostgreSQL local</li>
                </ul>
              </div>
            </div>

            <div class="arch-arrow-col">
              <div class="arch-arrow-block">
                <svg width="60" height="24" viewBox="0 0 60 24" fill="none">
                  <path d="M0 12 H50" stroke="#3b82f6" stroke-width="2"/>
                  <path d="M44 6 L56 12 L44 18" stroke="#3b82f6" stroke-width="2" fill="none"/>
                </svg>
                <span class="arch-arrow-label">HTTPS<br/>X-Device-ID<br/>X-Linea-ID</span>
              </div>
            </div>

            <div class="arch-col">
              <div class="arch-box arch-cloud">
                <div class="arch-box-header">
                  <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path d="M17.5 19H9a7 7 0 110-14 5 5 0 019.08 1.57A4.5 4.5 0 1117.5 19z" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                  Cloud (mentormonitor-ai.com)
                </div>
                <ul class="arch-list">
                  <li>cloud-gateway :8081</li>
                  <li>cloud-ingest :8888</li>
                  <li>cloud-config :8082</li>
                  <li>cloud-identity :8083</li>
                  <li>PostgreSQL cloud</li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        <div class="doc-section">
          <h2 class="section-title">Identificadores clave</h2>
          <div class="id-cards">
            <div class="id-card">
              <div class="id-card-icon" style="background:#1e3a5f">
                <svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#60a5fa"><rect x="2" y="3" width="20" height="14" rx="2" stroke-width="2"/></svg>
              </div>
              <div>
                <p class="id-card-name">DEVICE_ID</p>
                <p class="id-card-desc">Identifica el hardware fisico (Jetson/Raspberry). Un dispositivo puede monitorear multiples lineas.</p>
              </div>
            </div>
            <div class="id-card">
              <div class="id-card-icon" style="background:#1a3a2a">
                <svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#34d399"><path d="M9 3H5a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2V9M9 3v6h12M9 3l12 6" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
              </div>
              <div>
                <p class="id-card-name">LINEA_ID</p>
                <p class="id-card-desc">Identifica la linea de produccion. Debe ser el mismo ID que tiene la linea en el cloud (config.lineas.id). Es el identificador primario para el enrutamiento de datos.</p>
              </div>
            </div>
            <div class="id-card">
              <div class="id-card-icon" style="background:#3a1a1a">
                <svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#f87171"><path d="M15 7a2 2 0 012 2m4 0a6 6 0 01-6 6 6 6 0 01-6-6 6 6 0 016-6 6 6 0 016 6z" stroke-width="2"/><path d="M12 16v5M8 21h8" stroke-width="2" stroke-linecap="round"/></svg>
              </div>
              <div>
                <p class="id-card-name">API Key</p>
                <p class="id-card-desc">Token de autenticacion del dispositivo edge. Se genera al aprovisionar y se guarda automaticamente en la BD del Jetson. No expira hasta ser revocada manualmente.</p>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>

    <!-- Tab: Configurar Cloud -->
    <div v-if="activeTab === 'cloud'" class="tab-content">
      <h2 class="section-title">Configurar una linea en el Cloud</h2>
      <p class="section-desc mb-6">Sigue estos pasos en orden. Cada elemento depende del anterior.</p>

      <div class="flow-container">

        <div v-for="(step, i) in cloudSteps" :key="i" class="flow-step-wrapper">
          <div :class="['flow-step', step.variant]">
            <div class="flow-step-num">{{ i + 1 }}</div>
            <div class="flow-step-body">
              <div class="flow-step-header">
                <span v-html="step.icon"></span>
                <h3 class="flow-step-title">{{ step.title }}</h3>
                <span class="flow-step-path">{{ step.path }}</span>
              </div>
              <p class="flow-step-desc">{{ step.desc }}</p>
              <div v-if="step.fields" class="flow-step-fields">
                <span v-for="f in step.fields" :key="f" class="field-tag">{{ f }}</span>
              </div>
              <div v-if="step.warning" class="flow-step-warning">
                <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>
                {{ step.warning }}
              </div>
            </div>
          </div>
          <div v-if="i < cloudSteps.length - 1" class="flow-connector">
            <svg width="2" height="32" viewBox="0 0 2 32"><line x1="1" y1="0" x2="1" y2="32" stroke="#334155" stroke-width="2" stroke-dasharray="4 3"/></svg>
            <svg width="12" height="8" viewBox="0 0 12 8" fill="none"><path d="M1 1L6 7L11 1" stroke="#334155" stroke-width="2" stroke-linecap="round"/></svg>
          </div>
        </div>

      </div>
    </div>

    <!-- Tab: Configurar Edge -->
    <div v-if="activeTab === 'edge'" class="tab-content">
      <h2 class="section-title">Configurar un dispositivo Edge</h2>
      <p class="section-desc mb-6">Pasos a ejecutar en el Jetson / Raspberry Pi despues de tener la linea creada en el cloud.</p>

      <div class="flow-container">

        <div v-for="(step, i) in edgeSteps" :key="i" class="flow-step-wrapper">
          <div :class="['flow-step', step.variant]">
            <div class="flow-step-num">{{ i + 1 }}</div>
            <div class="flow-step-body">
              <div class="flow-step-header">
                <span v-html="step.icon"></span>
                <h3 class="flow-step-title">{{ step.title }}</h3>
                <span v-if="step.path" class="flow-step-path">{{ step.path }}</span>
              </div>
              <p class="flow-step-desc">{{ step.desc }}</p>
              <div v-if="step.code" class="flow-step-code">
                <pre>{{ step.code }}</pre>
              </div>
              <div v-if="step.fields" class="flow-step-fields">
                <span v-for="f in step.fields" :key="f" class="field-tag">{{ f }}</span>
              </div>
              <div v-if="step.warning" class="flow-step-warning">
                <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>
                {{ step.warning }}
              </div>
            </div>
          </div>
          <div v-if="i < edgeSteps.length - 1" class="flow-connector">
            <svg width="2" height="32" viewBox="0 0 2 32"><line x1="1" y1="0" x2="1" y2="32" stroke="#334155" stroke-width="2" stroke-dasharray="4 3"/></svg>
            <svg width="12" height="8" viewBox="0 0 12 8" fill="none"><path d="M1 1L6 7L11 1" stroke="#334155" stroke-width="2" stroke-linecap="round"/></svg>
          </div>
        </div>

      </div>
    </div>

    <!-- Tab: Flujo completo -->
    <div v-if="activeTab === 'flow'" class="tab-content">
      <h2 class="section-title">Flujo de datos completo</h2>
      <p class="section-desc mb-6">Como fluyen los datos desde la camara/sensores en planta hasta los graficos en el cloud.</p>

      <div class="full-flow">
        <div v-for="(block, i) in flowBlocks" :key="i" class="full-flow-item">
          <div :class="['full-flow-node', block.color]">
            <span v-html="block.icon"></span>
            <div>
              <p class="full-flow-name">{{ block.name }}</p>
              <p class="full-flow-detail">{{ block.detail }}</p>
            </div>
          </div>
          <div v-if="i < flowBlocks.length - 1" class="full-flow-arrow">
            <svg width="32" height="16" fill="none" viewBox="0 0 32 16">
              <path d="M0 8 H24" :stroke="block.arrowColor || '#334155'" stroke-width="2"/>
              <path d="M18 2 L30 8 L18 14" :stroke="block.arrowColor || '#334155'" stroke-width="2" fill="none"/>
            </svg>
            <span class="full-flow-arrow-label">{{ block.arrowLabel }}</span>
          </div>
        </div>
      </div>

      <div class="flow-legend">
        <h3 class="legend-title">Cabeceras HTTP enviadas por el Jetson</h3>
        <div class="legend-grid">
          <div class="legend-item">
            <span class="legend-key">X-Device-ID</span>
            <span class="legend-val">ID del hardware fisico (ej: 3)</span>
          </div>
          <div class="legend-item">
            <span class="legend-key">X-Linea-ID</span>
            <span class="legend-val">ID de la linea en el cloud. Debe coincidir con config.lineas.id</span>
          </div>
          <div class="legend-item">
            <span class="legend-key">Authorization</span>
            <span class="legend-val">Bearer &lt;api_key&gt;  — generada al aprovisionar el dispositivo</span>
          </div>
        </div>
        <p class="legend-note">El cloud resuelve automaticamente planta_id y empresa_id a partir del LINEA_ID mediante la cadena: config.lineas → config.plantas → identity.empresas</p>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref } from 'vue'

const activeTab = ref('overview')

const tabs = [
  {
    id: 'overview',
    label: 'Vision General',
    icon: '<svg width="15" height="15" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h10"/></svg>'
  },
  {
    id: 'cloud',
    label: 'Configurar Cloud',
    icon: '<svg width="15" height="15" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.5 19H9a7 7 0 110-14 5 5 0 019.08 1.57A4.5 4.5 0 1117.5 19z"/></svg>'
  },
  {
    id: 'edge',
    label: 'Configurar Edge',
    icon: '<svg width="15" height="15" fill="none" viewBox="0 0 24 24" stroke="currentColor"><rect x="2" y="3" width="20" height="14" rx="2" stroke-width="2"/><path d="M8 21h8M12 17v4" stroke-width="2" stroke-linecap="round"/></svg>'
  },
  {
    id: 'flow',
    label: 'Flujo de Datos',
    icon: '<svg width="15" height="15" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>'
  }
]

const ICON = {
  empresa: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0H5m14 0h2M5 21H3"/></svg>',
  planta: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12h18M3 6h18M3 18h18"/></svg>',
  linea: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3H5a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2V9M9 3v6h12M9 3l12 6"/></svg>',
  turno: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><circle cx="12" cy="12" r="10" stroke-width="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6l4 2"/></svg>',
  variable: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>',
  device: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><rect x="2" y="3" width="20" height="14" rx="2" stroke-width="2"/><path d="M8 21h8M12 17v4" stroke-width="2" stroke-linecap="round"/></svg>',
  key: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-6 6 6 6 0 01-6-6 6 6 0 016-6 6 6 0 016 6z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 16v5M8 21h8"/></svg>',
  check: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>',
  terminal: '<svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>',
}

const cloudSteps = [
  {
    variant: 'step-blue',
    icon: ICON.empresa,
    title: 'Crear la Empresa',
    path: 'Configuracion > Empresa > pestaña EMPRESA',
    desc: 'La empresa es la entidad raiz de toda la jerarquia. Todo lo demas (plantas, lineas, usuarios) pertenece a una empresa.',
    fields: ['Nombre', 'RUC / Codigo', 'Pais'],
  },
  {
    variant: 'step-blue',
    icon: ICON.planta,
    title: 'Crear la Planta',
    path: 'Configuracion > Empresa > pestaña PLANTA',
    desc: 'Una planta pertenece a una empresa. Representa una ubicacion fisica de produccion.',
    fields: ['Nombre', 'Empresa (FK)', 'Ubicacion'],
  },
  {
    variant: 'step-green',
    icon: ICON.linea,
    title: 'Crear la Linea de Produccion',
    path: 'Configuracion > Empresa > pestaña LINEA',
    desc: 'La linea es el elemento central. El ID generado aqui (config.lineas.id) es el LINEA_ID que deberas poner en el archivo .env del Jetson.',
    fields: ['Nombre', 'Planta (FK)', 'Codigo'],
    warning: 'Anota el ID generado. Este es el LINEA_ID del edge.'
  },
  {
    variant: 'step-blue',
    icon: ICON.turno,
    title: 'Configurar Turnos',
    path: 'Administracion > Turnos',
    desc: 'Define los turnos de trabajo de la linea (manana, tarde, noche). Los datos OEE se agrupan por turno.',
    fields: ['Nombre turno', 'Hora inicio', 'Hora fin', 'Linea (FK)'],
  },
  {
    variant: 'step-blue',
    icon: ICON.variable,
    title: 'Configurar Variables',
    path: 'Configuracion > Variables',
    desc: 'Las variables definen que campos del payload OEE se registran (piezas OK, piezas malas, tiempo ciclo, etc.).',
    fields: ['Nombre', 'Tipo', 'Unidad', 'Linea (FK)'],
  },
  {
    variant: 'step-orange',
    icon: ICON.device,
    title: 'Aprovisionar el Dispositivo Edge',
    path: 'Configuracion > Conexion Edge > Aprovisionar',
    desc: 'Registra el dispositivo fisico (Jetson). Se genera una API Key unica. Esta clave debe copiarse al archivo .env del Jetson como CLOUD_API_KEY.',
    fields: ['Device ID (unico)', 'Nombre descriptivo'],
    warning: 'La API Key solo se muestra una vez. Copiala inmediatamente.'
  },
  {
    variant: 'step-green',
    icon: ICON.check,
    title: 'Verificar conexion',
    path: 'Configuracion > Conexion Edge',
    desc: 'Una vez configurado el edge, el dispositivo aparece como "online" en la tabla dentro de los primeros 60 segundos.',
    fields: ['Estado: online', 'Linea ID correcto', 'Ultimo contacto reciente'],
  },
]

const edgeSteps = [
  {
    variant: 'step-blue',
    icon: ICON.terminal,
    title: 'Clonar el repositorio en el dispositivo',
    desc: 'Conectarse al Jetson/Raspberry por SSH y clonar el proyecto mentor-edge.',
    code: 'ssh orin@<IP_DISPOSITIVO>\ngit clone <REPO_URL> ~/mentor-edge',
  },
  {
    variant: 'step-orange',
    icon: ICON.key,
    title: 'Crear el archivo .env',
    path: '~/mentor-edge/infrastructure/docker/.env',
    desc: 'Crear el archivo de variables de entorno con los valores obtenidos del cloud. LINEA_ID debe coincidir exactamente con el ID de la linea en el cloud.',
    code: 'DEVICE_ID=<id_del_dispositivo>\nLINEA_ID=<id_linea_en_cloud>\nCLOUD_URL=https://mentormonitor-ai.com\nCLOUD_API_KEY=<api_key_del_aprovisionamiento>\nDB_PASSWORD=<password_postgres>',
    warning: 'LINEA_ID debe ser igual al ID de config.lineas en el cloud.'
  },
  {
    variant: 'step-blue',
    icon: ICON.linea,
    title: 'Habilitar la Linea (crear schema local)',
    path: 'UI Local :8080 > Configuracion > Habilitar Linea',
    desc: 'Desde la UI local del edge, seleccionar el schema de la linea y ejecutar la habilitacion. Esto crea el schema PostgreSQL local (linea_X) con todas las tablas necesarias.',
    fields: ['Schema: linea_<device_id>'],
  },
  {
    variant: 'step-blue',
    icon: ICON.terminal,
    title: 'Levantar los contenedores',
    desc: 'Iniciar todos los servicios edge con Docker Compose.',
    code: 'cd ~/mentor-edge/infrastructure/docker\ndocker compose -f docker-compose.jetson-orin.yml up -d',
  },
  {
    variant: 'step-green',
    icon: ICON.check,
    title: 'Verificar estado',
    path: 'UI Local :8080 > Configuracion > Registro Dispositivo',
    desc: 'Confirmar que todos los servicios estan healthy, el cloud_connected = true y el LINEA_ID correcto aparece en el panel de estado.',
    fields: ['cloud_connected: true', 'linea_id: <esperado>', 'buffer_pending: 0'],
  },
]

const flowBlocks = [
  {
    name: 'Camara / PLC / Sensor',
    detail: 'Captura evento en planta',
    icon: '<svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#94a3b8"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.069A1 1 0 0121 8.82v6.36a1 1 0 01-1.447.9L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>',
    color: 'node-gray',
    arrowLabel: 'local',
    arrowColor: '#475569'
  },
  {
    name: 'vision-event-detector',
    detail: 'Detecta y clasifica eventos',
    icon: '<svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#60a5fa"><circle cx="11" cy="11" r="8" stroke-width="2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-4.35-4.35"/></svg>',
    color: 'node-blue',
    arrowLabel: 'PostgreSQL\nlinea_X',
    arrowColor: '#3b82f6'
  },
  {
    name: 'edge-gateway',
    detail: 'Agrega y expone estado',
    icon: '<svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#a78bfa"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>',
    color: 'node-purple',
    arrowLabel: 'cola local',
    arrowColor: '#8b5cf6'
  },
  {
    name: 'enviador',
    detail: 'Sincroniza al cloud c/60s',
    icon: '<svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#34d399"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>',
    color: 'node-green',
    arrowLabel: 'HTTPS\nAPI Key',
    arrowColor: '#10b981'
  },
  {
    name: 'cloud-ingest',
    detail: 'Recibe y valida datos',
    icon: '<svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#fb923c"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/></svg>',
    color: 'node-orange',
    arrowLabel: 'PostgreSQL\ncloud',
    arrowColor: '#f97316'
  },
  {
    name: 'Dashboard / Analisis',
    detail: 'Visualizacion en tiempo real',
    icon: '<svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="#f472b6"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>',
    color: 'node-pink',
    arrowLabel: '',
    arrowColor: '#ec4899'
  },
]
</script>

<style scoped>
.docs-view {
  @apply p-6 max-w-5xl mx-auto;
}

.docs-header {
  @apply mb-8;
}

.docs-title {
  @apply text-2xl font-bold text-white mb-1;
}

.docs-subtitle {
  @apply text-sm text-gray-400 mb-5;
}

.docs-tabs {
  @apply flex gap-2 flex-wrap;
}

.tab-btn {
  @apply flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors;
  @apply bg-primary-800 border border-primary-700 text-gray-400 hover:text-white hover:border-primary-600;
}

.tab-btn-active {
  @apply bg-secondary-900/30 border-secondary-600 text-secondary-400;
}

.tab-content {
  @apply space-y-8;
}

.section-title {
  @apply text-lg font-bold text-white mb-2;
}

.section-desc {
  @apply text-sm text-gray-400;
}

.mb-6 {
  margin-bottom: 1.5rem;
}

/* Architecture diagram */
.content-grid {
  @apply space-y-8;
}

.doc-section {
  @apply bg-primary-800 rounded-xl border border-primary-700 p-6;
}

.arch-diagram {
  @apply flex items-center gap-4 mt-6 flex-wrap;
}

.arch-col {
  @apply flex-1 min-w-48;
}

.arch-box {
  @apply rounded-lg border p-4;
}

.arch-edge {
  @apply bg-blue-950/40 border-blue-800/50;
}

.arch-cloud {
  @apply bg-purple-950/40 border-purple-800/50;
}

.arch-box-header {
  @apply flex items-center gap-2 font-semibold text-sm text-white mb-3;
}

.arch-list {
  @apply space-y-1;
}

.arch-list li {
  @apply text-xs text-gray-400 font-mono;
}

.arch-arrow-col {
  @apply flex flex-col items-center justify-center;
}

.arch-arrow-block {
  @apply flex flex-col items-center gap-1;
}

.arch-arrow-label {
  @apply text-[10px] text-blue-400 text-center leading-tight;
}

/* ID Cards */
.id-cards {
  @apply space-y-3 mt-4;
}

.id-card {
  @apply flex items-start gap-3 p-3 rounded-lg bg-primary-900/50;
}

.id-card-icon {
  @apply w-10 h-10 rounded-lg flex items-center justify-center shrink-0;
}

.id-card-name {
  @apply text-sm font-semibold text-white font-mono;
}

.id-card-desc {
  @apply text-xs text-gray-400 mt-0.5;
}

/* Flow steps */
.flow-container {
  @apply space-y-0;
}

.flow-step-wrapper {
  @apply flex flex-col items-start;
}

.flow-step {
  @apply flex items-start gap-4 p-4 rounded-xl border w-full;
}

.step-blue { @apply bg-blue-950/30 border-blue-800/40; }
.step-green { @apply bg-green-950/30 border-green-800/40; }
.step-orange { @apply bg-orange-950/30 border-orange-800/40; }

.flow-step-num {
  @apply w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0;
  @apply bg-primary-700 text-white;
}

.flow-step-body {
  @apply flex-1;
}

.flow-step-header {
  @apply flex items-center gap-2 flex-wrap mb-1;
}

.flow-step-title {
  @apply text-sm font-semibold text-white;
}

.flow-step-path {
  @apply text-[10px] text-gray-500 font-mono bg-primary-900/60 px-2 py-0.5 rounded;
}

.flow-step-desc {
  @apply text-xs text-gray-400 leading-relaxed;
}

.flow-step-fields {
  @apply flex flex-wrap gap-1.5 mt-2;
}

.field-tag {
  @apply text-[10px] px-2 py-0.5 rounded bg-primary-700/60 text-gray-300 font-mono;
}

.flow-step-code {
  @apply mt-2 bg-gray-950 rounded-lg p-3 overflow-x-auto;
}

.flow-step-code pre {
  @apply text-xs text-green-400 font-mono whitespace-pre;
}

.flow-step-warning {
  @apply flex items-center gap-1.5 mt-2 text-xs text-yellow-400 bg-yellow-950/30 border border-yellow-800/40 rounded px-3 py-1.5;
}

.flow-connector {
  @apply flex flex-col items-start ml-8 py-0.5;
}

/* Full flow */
.full-flow {
  @apply flex flex-wrap items-center gap-2;
}

.full-flow-item {
  @apply flex items-center gap-2;
}

.full-flow-node {
  @apply flex items-center gap-2 px-3 py-2 rounded-lg border text-sm;
}

.node-gray  { @apply bg-slate-800/60 border-slate-700; }
.node-blue  { @apply bg-blue-950/40 border-blue-800/50; }
.node-purple { @apply bg-purple-950/40 border-purple-800/50; }
.node-green { @apply bg-green-950/40 border-green-800/50; }
.node-orange { @apply bg-orange-950/40 border-orange-800/50; }
.node-pink  { @apply bg-pink-950/40 border-pink-800/50; }

.full-flow-name {
  @apply text-xs font-semibold text-white;
}

.full-flow-detail {
  @apply text-[10px] text-gray-400;
}

.full-flow-arrow {
  @apply flex flex-col items-center gap-0.5;
}

.full-flow-arrow-label {
  @apply text-[9px] text-gray-500 text-center whitespace-pre;
}

/* Legend */
.flow-legend {
  @apply bg-primary-800 border border-primary-700 rounded-xl p-5 mt-6;
}

.legend-title {
  @apply text-sm font-semibold text-white mb-3;
}

.legend-grid {
  @apply space-y-2 mb-3;
}

.legend-item {
  @apply flex gap-3 text-xs items-baseline;
}

.legend-key {
  @apply font-mono bg-primary-900/60 px-2 py-0.5 rounded text-blue-300 shrink-0;
}

.legend-val {
  @apply text-gray-400;
}

.legend-note {
  @apply text-xs text-gray-500 border-t border-primary-700 pt-3;
}
</style>
