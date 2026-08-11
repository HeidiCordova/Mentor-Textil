<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const uiStore = useUiStore()
const authStore = useAuthStore()

const menuItems = ref([
  {
    id: 'inicio',
    label: 'Inicio',
    icon: 'home',
    path: '/dashboard'
  },
  {
    id: 'configuracion',
    label: 'Configuración',
    icon: 'settings',
    children: [
      { id: 'empresa', label: 'Empresa', path: '/configuracion/empresa' },
      { id: 'usuario', label: 'Usuario', path: '/configuracion/usuario' },
      { id: 'roles', label: 'Roles', path: '/configuracion/roles' },
      { id: 'mantenimiento', label: 'Monitoreo', path: '/configuracion/mantenimiento' },
      { id: 'variables', label: 'Variables', path: '/configuracion/variables' },
      { id: 'arbol-paradas', label: 'Árbol de Paradas', path: '/configuracion/arbol-paradas' },
      { id: 'dispositivos', label: 'Conexión Edge', path: '/configuracion/dispositivos' }
    ]
  },
  {
    id: 'administracion',
    label: 'Administración',
    icon: 'settings',
    children: [
      { id: 'tipo-documento', label: 'Tipo de Documento', path: '/administracion/tipo-documento' },
      { id: 'calendarizacion', label: 'Calendarización', path: '/administracion/calendarizacion', wip: true },
      { id: 'turnos', label: 'Turnos', path: '/administracion/turnos' },
      { id: 'productos', label: 'Productos', path: '/administracion/productos' },
      { id: 'velocidad-nominal', label: 'Velocidad Nominal', path: '/administracion/velocidad-nominal' },
      { id: 'canvas-oee', label: 'Canvas OEE', path: '/administracion/canvas-oee' },
      { id: 'laboratorio-oee', label: 'Laboratorio OEE', path: '/administracion/laboratorio-oee' },
      { id: 'habilitar-linea', label: 'Habilitar Línea', path: '/administracion/habilitar-linea' }
    ]
  },
  {
    id: 'integracion',
    label: 'API Integración',
    icon: 'link',
    path: '/integracion'
  },
  {
    id: 'vista-rapida',
    label: 'Vista Rápida',
    icon: 'chart',
    wip: true,
    children: [
      { id: 'factor-calificacion', label: 'Factor de Calificación', path: '/vista-rapida/factor-calificacion' },
      { id: 'oee-turno', label: 'OEE por Turno', path: '/vista-rapida/oee-turno' }
    ]
  },
  {
    id: 'analisis',
    label: 'Análisis',
    icon: 'chart',
    children: [
      { id: 'generador-consultas', label: 'Generador de Consultas - R', path: '/analisis-avanzado/generador-consultas' }
    ]
  },
  {
    id: 'analisis-energia',
    label: 'Energia',
    icon: 'bolt',
    children: [
      { id: 'control-edge-energia', label: 'Control Edge', path: '/analisis-energia/control-edge' },
      { id: 'medidores', label: 'Medidores', path: '/analisis-energia/medidores' },
      { id: 'monitor-energia', label: 'Monitor Energetico', path: '/analisis-energia/monitor' },
      { id: 'consumo-electrico-tarifario', label: 'Consumo Electrico Tarifario', path: '/analisis-energia/consumo-electrico-tarifario', wip: true },
      { id: 'factor-calificacion-energia', label: 'Factor de Calificacion', path: '/analisis-energia/factor-calificacion', wip: true }
    ]
  },
  {
    id: 'configuracion-energia',
    label: 'Configuración de Energía',
    icon: 'bolt',
    children: [
      { id: 'energia-dashboard', label: 'Dashboard', path: '/analisis-energia/monitor' },
      { id: 'energia-monitor', label: 'Monitor', path: '/analisis-energia/monitor' },
      { id: 'energia-lecturas', label: 'Lecturas', path: '/analisis-energia/lecturas', wip: true },
      { id: 'energia-configuracion', label: 'Configuración', path: '/analisis-energia/control-edge' },
      { id: 'energia-programar', label: 'Programar Medidor', path: '/analisis-energia/programar', wip: true },
      { id: 'energia-auditoria', label: 'Auditoría', path: '/analisis-energia/auditoria', wip: true },
      { id: 'energia-manual', label: 'Manual', path: '/analisis-energia/manual', wip: true }
    ]
  },
  {
    id: 'analisis-produccion',
    label: 'Análisis Producción',
    icon: 'factory',
    children: [
      { id: 'linea-tiempo', label: 'Línea de Tiempo', path: '/analisis-produccion/linea-tiempo', wip: true },
      { id: 'historia-linea', label: 'Historia de Línea', path: '/analisis-produccion/historia-linea', wip: true },
      { id: 'grafica-oee', label: 'Gráfica de OEE', path: '/analisis-produccion/grafica-oee' },
      { id: 'grafica-pareto', label: 'Gráfica de Pareto', path: '/analisis-produccion/grafica-pareto' },
      { id: 'tiempo-real', label: 'Tiempo Real', path: '/analisis-produccion/tiempo-real' },
      { id: 'monitor-oee', label: 'Monitor OEE', path: '/analisis-produccion/monitor-oee' }
    ]
  },
  {
    id: 'reportes',
    label: 'Reportes',
    icon: 'document',
    path: '/reportes',
    wip: true
  },
  {
    id: 'alarmas',
    label: 'Alarmas',
    icon: 'alert',
    path: '/alarmas'
  },
  {
    id: 'datos-recibidos',
    label: 'Datos Recibidos',
    icon: 'download',
    path: '/datos-recibidos'
  },
  {
    id: 'compromisos',
    label: 'Compromisos',
    icon: 'check',
    path: '/compromisos',
    wip: true
  },
  {
    id: 'docs',
    label: 'Documentacion',
    icon: 'document',
    path: '/docs'
  }
])

const expandedItems = ref([])

function toggleExpand(itemId) {
  const index = expandedItems.value.indexOf(itemId)
  if (index > -1) {
    expandedItems.value.splice(index, 1)
  } else {
    expandedItems.value.push(itemId)
  }
}

function isExpanded(itemId) {
  return expandedItems.value.includes(itemId)
}

function isActive(path) {
  if (!path) return false
  return route.path === path || route.path.startsWith(path + '/')
}

function navigate(path, wip) {
  if (wip || !path) return
  router.push(path)
  if (uiStore.isMobile) {
    uiStore.closeSidebar()
  }
}
</script>

<template>
  <aside
    :class="[
      'sidebar',
      {
        'sidebar-collapsed': uiStore.sidebarCollapsed && !uiStore.isMobile,
        'sidebar-mobile-open': uiStore.sidebarOpen && uiStore.isMobile,
        'sidebar-mobile-closed': !uiStore.sidebarOpen && uiStore.isMobile
      }
    ]"
  >
    <div class="sidebar-header">
      <div class="logos-container">
        <img src="/mentor logo.png" alt="Mentor Textil" class="logo-mentor">
      </div>
      <div class="user-info">
        <div class="user-avatar">
          <span>{{ authStore.user?.name?.charAt(0) || 'U' }}</span>
        </div>
        <div class="user-details">
          <p class="user-name">{{ authStore.user?.name || 'Usuario' }}</p>
          <p class="user-role">{{ authStore.user?.role || 'Administrador' }}</p>
        </div>
      </div>
    </div>

    <nav class="sidebar-nav">
      <template v-for="item in menuItems" :key="item.id">
        <div v-if="item.children" class="menu-group">
          <button
            :class="['menu-item', 'menu-item-parent', { 'menu-item-expanded': isExpanded(item.id) && !item.wip, 'menu-item-wip': item.wip }]"
            @click="!item.wip && toggleExpand(item.id)"
          >
            <span class="menu-item-icon" v-html="getIcon(item.icon)"></span>
            <span class="menu-item-label">{{ item.label }}</span>
            <span v-if="item.wip" class="wip-badge">En desarrollo</span>
            <span v-else class="menu-item-arrow" v-html="getIcon('chevron-down')"></span>
          </button>
          <Transition name="expand">
            <div v-show="isExpanded(item.id)" class="menu-children">
              <button
                v-for="child in item.children"
                :key="child.id"
                :class="['menu-item', 'menu-item-child', { 'menu-item-active': isActive(child.path), 'menu-item-wip': child.wip }]"
                @click="navigate(child.path, child.wip)"
              >
                <span class="menu-item-label">{{ child.label }}</span>
                <span v-if="child.wip" class="wip-badge">En desarrollo</span>
              </button>
            </div>
          </Transition>
        </div>

        <button
          v-else
          :class="['menu-item', { 'menu-item-active': isActive(item.path), 'menu-item-wip': item.wip }]"
          @click="navigate(item.path, item.wip)"
        >
          <span class="menu-item-icon" v-html="getIcon(item.icon)"></span>
          <span class="menu-item-label">{{ item.label }}</span>
          <span v-if="item.wip" class="wip-badge">En desarrollo</span>
        </button>
      </template>
    </nav>

    <div class="sidebar-footer">
      <button class="menu-item" @click="authStore.logout">
        <span class="menu-item-icon" v-html="getIcon('logout')"></span>
        <span class="menu-item-label">Salir</span>
      </button>
    </div>
  </aside>
</template>

<script>
function getIcon(name) {
  const icons = {
    home: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z" stroke="currentColor" stroke-width="2"/><path d="M9 22V12h6v10" stroke="currentColor" stroke-width="2"/></svg>',
    settings: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2"/><path d="M12 1v6m0 6v6m-9-9h6m6 0h6" stroke="currentColor" stroke-width="2"/></svg>',
    chart: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M18 20V10M12 20V4M6 20v-6" stroke="currentColor" stroke-width="2"/></svg>',
    document: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" stroke="currentColor" stroke-width="2"/></svg>',
    alert: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" stroke="currentColor" stroke-width="2"/></svg>',
    download: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4m4-5l5 5m0 0l5-5m-5 5V3" stroke="currentColor" stroke-width="2"/></svg>',
    check: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M9 11l3 3L22 4" stroke="currentColor" stroke-width="2"/></svg>',
    link: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71" stroke="currentColor" stroke-width="2" stroke-linecap="round"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>',
    bolt: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M13 2L3 14h8l-1 8 10-12h-8l1-8z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    factory: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M3 21h18M5 21V9l5 4V9l5 4V7a2 2 0 012-2h4v16" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><path d="M9 21v-4h6v4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    device: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><rect x="4" y="2" width="16" height="20" rx="2" stroke="currentColor" stroke-width="2"/><line x1="8" y1="6" x2="16" y2="6" stroke="currentColor" stroke-width="2"/><line x1="8" y1="10" x2="16" y2="10" stroke="currentColor" stroke-width="2"/><circle cx="12" cy="17" r="1.5" stroke="currentColor" stroke-width="2"/></svg>',
    logout: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none"><path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4m7 14l5-5-5-5" stroke="currentColor" stroke-width="2"/></svg>',
    'chevron-down': '<svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2"/></svg>'
  }
  return icons[name] || ''
}

export { getIcon }
</script>

<style scoped>
.sidebar {
  @apply fixed top-0 left-0 h-full bg-primary-800 text-white;
  @apply flex flex-col z-50;
  width: var(--sidebar-width);
  transition: transform 0.3s ease, width 0.3s ease;
}

.sidebar-collapsed {
  width: 64px;
}

.sidebar-mobile-closed {
  transform: translateX(-100%);
}

.sidebar-mobile-open {
  transform: translateX(0);
}

.sidebar-header {
  @apply p-4 border-b border-primary-700;
}

.logos-container {
  @apply flex items-center justify-center mb-3;
}

.logo-mentor {
  @apply h-12 w-auto;
}

.logo-planta-container {
  @apply flex items-center justify-center mb-4;
}

.logo-planta {
  @apply h-20 w-auto rounded-xl shadow-lg;
}

.user-info {
  @apply flex items-center gap-3 p-3 bg-primary-900 rounded-lg;
}

.user-avatar {
  @apply w-10 h-10 rounded-full bg-primary-600 flex items-center justify-center;
  @apply text-white font-bold text-sm flex-shrink-0;
}

.user-details {
  @apply flex-1 min-w-0;
}

.user-name {
  @apply text-sm font-semibold text-white truncate;
}

.user-role {
  @apply text-xs text-gray-400 truncate;
}

.sidebar-collapsed .logos-container {
  @apply flex-col gap-2;
}

.sidebar-collapsed .logo-mentor,
.sidebar-collapsed .logo-planta {
  @apply h-8 w-8;
}

.sidebar-collapsed .user-info {
  @apply flex-col p-2;
}

.sidebar-collapsed .user-details {
  @apply hidden;
}

.sidebar-nav {
  @apply flex-1 overflow-y-auto py-4;
  /* Ocultar scrollbar */
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none; /* IE and Edge */
}

/* Ocultar scrollbar para Chrome, Safari y Opera */
.sidebar-nav::-webkit-scrollbar {
  display: none;
}

.menu-item {
  @apply w-full flex items-center gap-3 px-6 py-3 text-left;
  @apply text-gray-300 hover:bg-primary-700 hover:text-white;
  @apply transition-colors duration-200;
}

.menu-item-active {
  @apply bg-primary-700 text-white;
}

.menu-item-icon {
  @apply flex-shrink-0 flex items-center justify-center;
}

.menu-item-label {
  @apply flex-1 truncate;
}

.sidebar-collapsed .menu-item-label {
  @apply hidden;
}

.menu-item-arrow {
  @apply flex-shrink-0 transition-transform duration-200;
}

.menu-item-expanded .menu-item-arrow {
  @apply rotate-180;
}

.sidebar-collapsed .menu-item-arrow {
  @apply hidden;
}

.menu-item-parent {
  @apply font-medium;
}

.menu-children {
  @apply bg-primary-900;
}

.menu-item-child {
  @apply pl-14 py-2 text-sm;
}

.menu-item-wip {
  @apply opacity-50 cursor-not-allowed;
  pointer-events: none;
}

.wip-badge {
  @apply text-xs font-medium px-1.5 py-0.5 rounded flex-shrink-0;
  background: rgba(234, 179, 8, 0.15);
  color: #ca8a04;
  border: 1px solid rgba(234, 179, 8, 0.3);
  letter-spacing: 0.02em;
}

.sidebar-footer {
  @apply border-t border-primary-700 p-4;
}

.expand-enter-active,
.expand-leave-active {
  @apply transition-all duration-300;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  @apply opacity-0;
  max-height: 0;
}

.expand-enter-to,
.expand-leave-from {
  @apply opacity-100;
  max-height: 500px;
}

@media (max-width: 1024px) {
  .sidebar:not(.sidebar-collapsed) {
    width: var(--sidebar-width);
  }
}

@media (max-width: 768px) {
  .sidebar {
    width: 260px;
  }
}
</style>
