<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import AppNav from '@/components/AppNav.vue'
import { useConnectionStore } from '@/stores/connection'
import { useMachineStore } from '@/stores/machine'
import { useStopsStore } from '@/stores/stops'
import { useConfigStore } from '@/stores/config'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { probeDevice } from '@/services/discovery'

const route = useRoute()
const connection = useConnectionStore()
const machine = useMachineStore()
const stopsStore = useStopsStore()
const configStore = useConfigStore()
const plantasLineas = usePlantasLineasStore()

const darkMode = ref(false)
const navOpen = ref(false)

function applyTheme(dark: boolean): void {
  darkMode.value = dark
  document.documentElement.classList.toggle('dark', dark)
  localStorage.setItem('edge_theme', dark ? 'dark' : 'light')
}

function toggleDark(): void {
  applyTheme(!darkMode.value)
}

const isPublicPage = () =>
  route.path === '/login' || route.path === '/device' || route.path === '/'

onMounted(async () => {
  const saved = localStorage.getItem('edge_theme')
  applyTheme(saved === 'dark')

  if (connection.restoreCloudSession()) {
    plantasLineas.load()
    return
  }

  // En HTTPS no podemos hacer probes HTTP (Mixed Content) → sólo modo cloud
  if (window.location.protocol === 'https:') return

  // Probar mismo origen primero (nginx proxea /edge/ → funciona desde cualquier red)
  const candidates = [
    window.location.origin,
    `http://${window.location.hostname}:8005`,
  ]
  // Si ya hay una URL guardada del mismo host, usarla también (pero origin tiene prioridad)
  const savedURL = localStorage.getItem('edge_url')
  if (savedURL && savedURL !== window.location.origin) {
    candidates.push(savedURL)
  }
  for (const url of candidates) {
    const ok = await probeDevice(url)
    if (ok) {
      // Siempre guardar el origin como URL preferida para futuras sesiones
      if (url === window.location.origin) {
        localStorage.setItem('edge_url', url)
      }
      connection.connectToEdge(url)
      plantasLineas.load()
      if (connection.authenticated) {
        machine.bindSSE()
        stopsStore.bindSSE()
        configStore.bindSSE()
      }
      break  // no seguir probando más URLs
    }
  }
})
</script>

<template>
  <div class="flex flex-col h-screen bg-edge-950 text-edge-200">
    <AppHeader
      v-if="!isPublicPage()"
      :dark-mode="darkMode"
      @toggle-dark="toggleDark"
    />

    <div class="relative flex flex-1 min-h-0 overflow-hidden">
      <!-- Backdrop para cerrar el menú tocando fuera -->
      <Transition name="fade-bg">
        <div
          v-if="navOpen && !isPublicPage()"
          class="absolute inset-0 z-20 bg-black/50"
          @click="navOpen = false"
        />
      </Transition>

      <!-- Sidebar slide-in overlay -->
      <Transition name="slide-nav">
        <AppNav
          v-if="!isPublicPage() && navOpen"
          class="absolute left-0 top-0 h-full z-30 shadow-2xl shadow-black/50"
        />
      </Transition>

      <!-- Tab de toggle — siempre visible en el borde izquierdo -->
      <button
        v-if="!isPublicPage()"
        class="absolute top-1/2 -translate-y-1/2 z-40 flex items-center justify-center w-6 h-16 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 rounded-r-xl text-white shadow-lg transition-[left] duration-300 ease-in-out"
        :style="{ left: navOpen ? '224px' : '0px' }"
        @click="navOpen = !navOpen"
      >
        <!-- chevron apunta derecha cuando cerrado, izquierda cuando abierto -->
        <svg class="w-4 h-4 transition-transform duration-300" :class="navOpen ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" />
        </svg>
      </button>

      <main class="flex-1 min-h-0 overflow-hidden">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
/* Fondo oscuro al abrir el menú */
.fade-bg-enter-active,
.fade-bg-leave-active { transition: opacity 0.25s ease; }
.fade-bg-enter-from,
.fade-bg-leave-to { opacity: 0; }

/* Slide del sidebar desde la izquierda */
.slide-nav-enter-active,
.slide-nav-leave-active { transition: transform 0.3s ease; }
.slide-nav-enter-from,
.slide-nav-leave-to { transform: translateX(-100%); }
</style>
