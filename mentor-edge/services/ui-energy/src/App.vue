<template>
  <div class="flex min-h-screen bg-slate-900">

    <!-- Sidebar -->
    <aside class="w-14 lg:w-52 flex flex-col bg-slate-800 border-r border-slate-700 shrink-0">

      <!-- Logo -->
      <div class="h-14 flex items-center gap-2.5 px-3.5 border-b border-slate-700 overflow-hidden">
        <svg class="w-7 h-7 shrink-0 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M13 10V3L4 14h7v7l9-11h-7z"/>
        </svg>
        <span class="hidden lg:block text-sm font-bold text-white whitespace-nowrap leading-tight">
          Mentor<br><span class="text-blue-400">Energy</span>
        </span>
      </div>

      <!-- Nav -->
      <nav class="flex-1 py-3 px-1.5 space-y-0.5">

        <router-link to="/" class="nav-item" :class="$route.path === '/' ? 'nav-item-active' : ''" title="Dashboard">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
          </svg>
          <span class="hidden lg:block">Dashboard</span>
        </router-link>

        <router-link to="/monitor" class="nav-item" :class="$route.path === '/monitor' ? 'nav-item-active' : ''" title="Monitor">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0
                 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0
                 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
          </svg>
          <span class="hidden lg:block">Monitor</span>
        </router-link>

        <router-link to="/lecturas" class="nav-item" :class="$route.path === '/lecturas' ? 'nav-item-active' : ''" title="Lecturas">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          <span class="hidden lg:block">Lecturas</span>
        </router-link>

        <router-link to="/config" class="nav-item" :class="$route.path === '/config' ? 'nav-item-active' : ''" title="Configuracion">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <span class="hidden lg:block">Configuracion</span>
        </router-link>

        <router-link to="/programar" class="nav-item" :class="$route.path === '/programar' ? 'nav-item-active' : ''" title="Programar medidor">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
          </svg>
          <span class="hidden lg:block">Programar</span>
        </router-link>

        <router-link to="/auditoria" class="nav-item" :class="$route.path === '/auditoria' ? 'nav-item-active' : ''" title="Auditoria">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"/>
          </svg>
          <span class="hidden lg:block">Auditoria</span>
        </router-link>

        <router-link to="/manual" class="nav-item" :class="$route.path === '/manual' ? 'nav-item-active' : ''" title="Manual">
          <svg class="nav-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
          </svg>
          <span class="hidden lg:block">Manual</span>
        </router-link>

      </nav>

      <!-- Device ID badge -->
      <div v-if="deviceId" class="px-2 py-3 border-t border-slate-700">
        <div class="hidden lg:block px-2 py-1.5 rounded bg-slate-700/50">
          <p class="text-[10px] text-gray-500 uppercase tracking-wider">Dispositivo</p>
          <p class="font-mono text-xs text-gray-300 truncate">{{ deviceId }}</p>
        </div>
      </div>

    </aside>

    <!-- Main content -->
    <main class="flex-1 min-w-0 p-6 overflow-auto">
      <router-view />
    </main>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getConfig } from './services/api.js'

const deviceId = ref('')

onMounted(async () => {
  try {
    const items = await getConfig()
    const entry = items.find(i => i.key === 'device_id')
    if (entry) deviceId.value = entry.value
  } catch {
    // no critical
  }
})
</script>
