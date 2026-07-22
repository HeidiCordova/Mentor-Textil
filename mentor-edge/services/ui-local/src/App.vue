<template>
  <div class="flex bg-slate-900" :class="isProduccionRoute ? 'h-screen overflow-hidden' : 'min-h-screen'">

    <!-- Sidebar -->
    <aside
      class="flex flex-col bg-slate-800 border-r border-slate-700 transition-all duration-300"
      :class="[sidebarOpen ? 'w-56' : 'w-14', isProduccionRoute ? 'h-screen overflow-hidden' : '']">

      <!-- Logo -->
      <div class="h-16 flex items-center gap-2.5 px-3.5 border-b border-slate-700 shrink-0 overflow-hidden">
        <img src="/mentor-logo.png" alt="Mentor Monitor" class="h-8 w-8 object-contain shrink-0" />
        <span v-if="sidebarOpen" class="text-base font-bold text-white whitespace-nowrap leading-tight">
          Mentor<br><span class="text-blue-400">Monitor</span>
        </span>
      </div>

      <!-- Nav items -->
      <nav class="flex-1 py-3 space-y-0.5 px-1.5 overflow-hidden">

        <router-link to="/" class="nav-item" :title="sidebarOpen ? '' : 'Inicio'">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>
          </svg>
          <span v-if="sidebarOpen" class="nav-label">Inicio</span>
        </router-link>

        <!-- Sección Configuración -->
        <div>
          <button @click="configOpen = !configOpen"
            class="nav-item w-full"
            :class="isConfigRoute ? 'nav-item-active' : ''"
            :title="sidebarOpen ? '' : 'Configuración'">
            <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            </svg>
            <span v-if="sidebarOpen" class="nav-label flex-1 text-left">Configuración</span>
            <svg v-if="sidebarOpen" class="w-3.5 h-3.5 text-gray-500 shrink-0 transition-transform"
              :class="configOpen ? 'rotate-90' : ''"
              fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
            </svg>
          </button>

          <!-- Sub-items: Dispositivos -->
          <div v-if="(sidebarOpen && configOpen) || isConfigRoute"
            class="mt-0.5 ml-3 pl-2 border-l border-slate-600 space-y-0.5">
            <router-link to="/config" class="nav-subitem">
              <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
              </svg>
              Dispositivos
            </router-link>
            <router-link to="/provision" class="nav-subitem">
              <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 4v16m8-8H4"/>
              </svg>
              Habilitar Línea
            </router-link>
            <router-link to="/registry" class="nav-subitem">
              <svg class="w-3.5 h-3.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/>
              </svg>
              Registro Dispositivo
            </router-link>
          </div>
        </div>

        <router-link to="/health" class="nav-item" :title="sidebarOpen ? '' : 'Estado'">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <span v-if="sidebarOpen" class="nav-label">Estado del Sistema</span>
        </router-link>

        <router-link to="/tiempo-real" class="nav-item" :title="sidebarOpen ? '' : 'Tiempo Real'">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
          </svg>
          <span v-if="sidebarOpen" class="nav-label">Tiempo Real</span>
        </router-link>

        <router-link to="/hardware" class="nav-item" :title="sidebarOpen ? '' : 'Hardware'">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
          </svg>
          <span v-if="sidebarOpen" class="nav-label">Hardware</span>
        </router-link>

        <router-link to="/usuarios" class="nav-item" :title="sidebarOpen ? '' : 'Usuarios'">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <span v-if="sidebarOpen" class="nav-label">Usuarios</span>
        </router-link>

      </nav>

      <!-- Footer: toggle + version -->
      <div class="border-t border-slate-700 p-2 shrink-0">
        <button @click="sidebarOpen = !sidebarOpen"
          class="w-full flex items-center justify-center p-1.5 rounded hover:bg-slate-700 text-gray-500 hover:text-white transition-colors"
          :title="sidebarOpen ? 'Colapsar menú' : 'Expandir menú'">
          <svg class="w-4 h-4 transition-transform" :class="sidebarOpen ? '' : 'rotate-180'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7"/>
          </svg>
        </button>
        <p v-if="sidebarOpen" class="text-center text-[11px] text-gray-600 mt-1 font-mono">edge v1</p>
      </div>

    </aside>

    <!-- Contenido principal -->
    <main class="flex-1 min-w-0"
      :class="isProduccionRoute ? 'overflow-hidden flex flex-col h-full' : 'overflow-auto p-6'">
      <router-view />
    </main>

  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const sidebarOpen = ref(true)
const configOpen = ref(false)

const isConfigRoute = computed(() =>
  route.path.startsWith('/config') || route.path.startsWith('/provision') || route.path.startsWith('/registry')
)
const isProduccionRoute = computed(() =>
  route.path.startsWith('/produccion') || route.path.startsWith('/tiempo-real')
)
const produccionRoute = computed(() => {
  const deviceId = route.params.deviceId
  return deviceId ? '/produccion/' + deviceId : '/produccion'
})
</script>

<style scoped>
.nav-item {
  @apply flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm text-gray-400 hover:bg-slate-700 hover:text-white transition-colors w-full;
}
.nav-item-active {
  @apply bg-slate-700/60 text-white;
}
.router-link-active.nav-item {
  @apply bg-blue-600/20 text-blue-400;
}
.nav-icon {
  @apply w-[18px] h-[18px] shrink-0;
}
.nav-label {
  @apply text-sm font-medium whitespace-nowrap;
}
.nav-subitem {
  @apply flex items-center gap-2 px-2 py-1.5 rounded text-xs text-gray-400 hover:bg-slate-700 hover:text-white transition-colors;
}
.router-link-active.nav-subitem {
  @apply bg-blue-600/20 text-blue-400;
}
</style>
