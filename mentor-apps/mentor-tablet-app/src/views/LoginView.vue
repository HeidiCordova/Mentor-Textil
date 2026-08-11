<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import SvgIcon from '@/components/SvgIcon.vue'
import { useConnectionStore } from '@/stores/connection'
import { useMachineStore } from '@/stores/machine'
import { useStopsStore } from '@/stores/stops'
import { useConfigStore } from '@/stores/config'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { probeDevice } from '@/services/discovery'

const router = useRouter()
const connection = useConnectionStore()
const machine = useMachineStore()
const stopsStore = useStopsStore()
const configStore = useConfigStore()
const plantasLineas = usePlantasLineasStore()

const loginMode = ref<'edge' | 'cloud'>('edge')
const username = ref('')
const password = ref('')
const loading = ref(true)
const loggingIn = ref(false)
const error = ref('')
const edgeOk = ref(false)

function resolveCloudUrl(): string {
  const saved = localStorage.getItem('cloud_url')
  if (saved) return saved
  // En HTTPS no podemos conectar a puertos HTTP — usamos el dominio raíz
  // que nginx enruta al gateway (port 8888) internamente
  if (window.location.protocol === 'https:') {
    const host = window.location.hostname
    // tablet.mentormonitor-ai.com → mentormonitor-ai.com (quita primer subdominio)
    const parts = host.split('.')
    const rootDomain = parts.length > 2 ? parts.slice(1).join('.') : host
    return `https://${rootDomain}`
  }
  return `http://${window.location.hostname}:8888`
}

async function autoConnect(): Promise<boolean> {
  if (connection.restoreCloudSession()) {
    loginMode.value = 'cloud'
    return true
  }
  // En HTTPS no podemos hacer probe por HTTP (Mixed Content) — saltamos edge
  if (window.location.protocol === 'https:') {
    loginMode.value = 'cloud'
    return false
  }
  // Si App.vue ya estableció la conexión edge, reutilizarla sin re-conectar
  if (connection.edgeURL) {
    return true
  }
  // Siempre conectar via el mismo origen (nginx proxea /edge/ internamente)
  const origin = window.location.origin
  const ok = await probeDevice(origin)
  if (ok) {
    localStorage.setItem('edge_url', origin)
    connection.connectToEdge(origin)
    return true
  }
  // Fallback: puerto directo
  const direct = `http://${window.location.hostname}:8005`
  const ok2 = await probeDevice(direct)
  if (ok2) { connection.connectToEdge(direct); return true }
  return false
}

function bindSSE(): void {
  machine.bindSSE()
  stopsStore.bindSSE()
  configStore.bindSSE()
}

async function handleEdgeLogin(): Promise<void> {
  if (!username.value.trim() || !password.value) return
  loggingIn.value = true
  error.value = ''
  const ok = await connection.login(username.value.trim(), password.value)
  if (ok) {
    bindSSE()
    plantasLineas.load()
    router.push('/dashboard').catch(() => { window.location.href = '/dashboard' })
  } else {
    error.value = 'Usuario o contraseña incorrectos'
  }
  loggingIn.value = false
}

async function handleCloudLogin(): Promise<void> {
  if (!username.value.trim() || !password.value) return
  loggingIn.value = true
  error.value = ''
  const url = resolveCloudUrl()

  connection.connectToCloud(url)
  const reachable = await connection.probeCloud()
  if (!reachable) {
    error.value = 'No se pudo conectar al servidor cloud'
    loggingIn.value = false
    return
  }

  const ok = await connection.cloudLogin(username.value.trim(), password.value)
  if (ok) {
    plantasLineas.load()
    router.push('/dashboard')
  } else {
    error.value = 'Credenciales incorrectas'
  }
  loggingIn.value = false
}

function handleSubmit(): void {
  if (loginMode.value === 'cloud') handleCloudLogin()
  else handleEdgeLogin()
}

onMounted(async () => {
  const connected = await autoConnect()
  if (connected && connection.authenticated) {
    plantasLineas.load()
    if (connection.mode !== 'CLOUD') bindSSE()
    router.push('/dashboard')
    return
  }
  if (connected && connection.mode !== 'CLOUD') {
    edgeOk.value = true
  }
  loading.value = false
})
</script>

<template>
  <!-- Fondo con gradiente corporativo -->
  <div class="relative flex items-center justify-center min-h-screen overflow-hidden"
       style="background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%);">

    <!-- Círculos decorativos de fondo -->
    <div class="absolute top-[-120px] right-[-120px] w-96 h-96 rounded-full opacity-10"
         style="background: radial-gradient(circle, #6366f1, transparent)"></div>
    <div class="absolute bottom-[-100px] left-[-100px] w-80 h-80 rounded-full opacity-10"
         style="background: radial-gradient(circle, #3b82f6, transparent)"></div>

    <!-- Tarjeta principal -->
    <div class="relative w-full max-w-md mx-4">

      <!-- Header con logo -->
      <div class="text-center mb-8">
        <img src="/mentor-logo.png" alt="Mentor Textil" class="h-16 w-auto object-contain mx-auto mb-4 drop-shadow-lg" />
        <h1 class="text-3xl font-bold tracking-wide text-white">MENTOR <span style="color:#6366f1">TEXTIL</span></h1>
        <p class="text-sm text-slate-400 mt-1 tracking-widest uppercase">Tablet Industrial</p>
      </div>

      <!-- Card del formulario -->
      <div class="rounded-2xl border border-slate-700/50 p-8 shadow-2xl"
           style="background: rgba(15,23,42,0.85); backdrop-filter: blur(16px);">

        <div v-if="loading" class="flex items-center justify-center py-10 gap-3 text-slate-400 text-sm">
          <SvgIcon name="refresh" :size="20" class="animate-spin" />
          <span>Conectando...</span>
        </div>

        <template v-else>
          <!-- Toggle Edge / Cloud -->
          <div class="flex mb-6 rounded-xl p-1 gap-1" style="background:#1e293b; border: 1px solid rgba(100,116,139,0.3)">
            <button
              class="flex-1 py-2.5 text-sm font-semibold rounded-lg transition-all duration-200"
              :class="loginMode === 'edge'
                ? 'text-white shadow-md' : 'text-slate-500 hover:text-slate-300'"
              :style="loginMode === 'edge' ? 'background:linear-gradient(135deg,#334155,#475569)' : ''"
              @click="loginMode = 'edge'; error = ''"
            >Edge Local</button>
            <button
              class="flex-1 py-2.5 text-sm font-semibold rounded-lg transition-all duration-200"
              :class="loginMode === 'cloud'
                ? 'text-white shadow-md' : 'text-slate-500 hover:text-slate-300'"
              :style="loginMode === 'cloud' ? 'background:linear-gradient(135deg,#4f46e5,#6366f1)' : ''"
              @click="loginMode = 'cloud'; error = ''"
            >Cloud</button>
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-5">

            <!-- Sin dispositivo edge -->
            <div v-if="loginMode === 'edge' && !edgeOk"
                 class="text-center py-6 rounded-xl border border-red-500/20"
                 style="background:rgba(239,68,68,0.06)">
              <SvgIcon name="wifi-off" :size="32" class="mx-auto mb-3 text-red-400" />
              <p class="text-sm text-red-400 mb-4">No se encontró el dispositivo Edge</p>
              <button
                type="button"
                class="px-4 py-2 text-xs font-medium rounded-lg border border-slate-600 text-slate-300 hover:text-white hover:border-slate-400 transition-all"
                @click="router.push('/device')"
              >Configurar manualmente</button>
            </div>

            <template v-if="loginMode === 'cloud' || edgeOk">
              <!-- Campo usuario -->
              <div>
                <label class="block text-xs font-semibold text-slate-400 mb-2 uppercase tracking-wider">Usuario</label>
                <input
                  v-model="username"
                  type="text"
                  autocomplete="username"
                  autofocus
                  placeholder="Nombre de usuario"
                  class="w-full px-4 py-3.5 rounded-xl text-white placeholder-slate-500 text-sm transition-all outline-none"
                  style="background:#1e293b; border:1px solid rgba(100,116,139,0.4);"
                  onfocus="this.style.borderColor='rgba(99,102,241,0.7)'; this.style.boxShadow='0 0 0 3px rgba(99,102,241,0.15)'"
                  onblur="this.style.borderColor='rgba(100,116,139,0.4)'; this.style.boxShadow='none'"
                />
              </div>

              <!-- Campo contraseña -->
              <div>
                <label class="block text-xs font-semibold text-slate-400 mb-2 uppercase tracking-wider">Contraseña</label>
                <input
                  v-model="password"
                  type="password"
                  autocomplete="current-password"
                  placeholder="••••••••"
                  class="w-full px-4 py-3.5 rounded-xl text-white placeholder-slate-500 text-sm transition-all outline-none"
                  style="background:#1e293b; border:1px solid rgba(100,116,139,0.4);"
                  onfocus="this.style.borderColor='rgba(99,102,241,0.7)'; this.style.boxShadow='0 0 0 3px rgba(99,102,241,0.15)'"
                  onblur="this.style.borderColor='rgba(100,116,139,0.4)'; this.style.boxShadow='none'"
                />
              </div>

              <!-- Botón submit -->
              <button
                type="submit"
                :disabled="loggingIn || !username.trim() || !password"
                class="w-full flex items-center justify-center gap-2 px-4 py-3.5 rounded-xl font-bold text-sm transition-all duration-200 mt-2"
                :style="loggingIn || !username.trim() || !password
                  ? 'background:#1e293b; color:#475569; cursor:not-allowed;'
                  : 'background:linear-gradient(135deg,#4f46e5,#6366f1); color:white; box-shadow:0 4px 20px rgba(99,102,241,0.35);'"
              >
                <SvgIcon v-if="loggingIn" name="refresh" :size="16" class="animate-spin" />
                <span>{{ loggingIn ? 'Iniciando sesión...' : 'Iniciar Sesión →' }}</span>
              </button>
            </template>

            <!-- Error -->
            <div v-if="error"
                 class="flex items-center gap-2 px-4 py-3 rounded-xl text-sm text-red-300"
                 style="background:rgba(239,68,68,0.1); border:1px solid rgba(239,68,68,0.2)">
              <span>⚠</span> {{ error }}
            </div>

          </form>

          <!-- Link avanzado -->
          <div class="mt-6 pt-5 text-center border-t border-slate-700/40">
            <button
              class="text-xs text-slate-500 hover:text-slate-300 transition-colors tracking-wide"
              @click="router.push('/device')"
            >Selección avanzada de dispositivo</button>
          </div>
        </template>
      </div>

      <!-- Versión / footer -->
      <p class="text-center text-xs text-slate-600 mt-6 tracking-wider">MENTOR TEXTIL © 2026</p>
    </div>
  </div>
</template>
