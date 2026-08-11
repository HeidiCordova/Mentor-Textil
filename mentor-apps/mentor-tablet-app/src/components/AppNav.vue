<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SvgIcon from './SvgIcon.vue'
import { useStopsStore } from '@/stores/stops'
import { useConnectionStore } from '@/stores/connection'
import { useConfigStore } from '@/stores/config'
import { usePlantasLineasStore } from '@/stores/plantasLineas'

const route = useRoute()
const router = useRouter()
const stopsStore = useStopsStore()
const connection = useConnectionStore()
const configStore = useConfigStore()
const pl = usePlantasLineasStore()

const showEmpresaDropdown = ref(false)
const showPlantaDropdown = ref(false)
const showLineaDropdown  = ref(false)

interface NavItem {
  id: string
  icon: string
  label: string
  route: string
  badge?: number
}

const items = computed<NavItem[]>(() => {
  const all: NavItem[] = [
  { id: 'dashboard', icon: 'chart', label: 'Dashboard', route: '/dashboard' },
  {
    id: 'stops',
    icon: 'stop',
    label: 'Paradas',
    route: '/stops',
    badge: stopsStore.unjustifiedStops.length || undefined
  },
  { id: 'produccion', icon: 'chart', label: 'Producción', route: '/produccion' },
  { id: 'historial', icon: 'clock', label: 'Historial', route: '/historial' },
  { id: 'config', icon: 'settings', label: 'Configuración', route: '/config' },
  { id: 'status', icon: 'database', label: 'Estado', route: '/status' }
  ]
  // En modo cloud, ocultar secciones que son edge-only
  if (connection.isCloudOnly) {
    return all.filter(i => i.id !== 'config' && i.id !== 'status')
  }
  return all
})

const lineName = computed(() => {
  if (pl.lineaActual) return pl.lineaActual.nombre
  const cfg = configStore.config as Record<string, unknown>
  if (cfg?.oee && typeof cfg.oee === 'object') {
    const oee = cfg.oee as Record<string, unknown>
    if (oee.line_name && typeof oee.line_name === 'string') return oee.line_name
  }
  return 'Línea'
})

const plantaName = computed(() => pl.plantaActual?.nombre ?? 'Planta')

const lineCode = computed(() => {
  const n = lineName.value
  const words = n.trim().split(/\s+/)
  return words.length >= 2 ? (words[0][0] + words[1][0]).toUpperCase() : n.slice(0, 2).toUpperCase()
})

const lineDesc = computed(() => {
  if (pl.plantaActual) return pl.plantaActual.nombre
  const cfg = configStore.config as Record<string, unknown>
  const mode = (cfg?.mode as string) || 'textil'
  return `Telar Circular · ${mode.charAt(0).toUpperCase() + mode.slice(1)}`
})

const canChangePlanta = computed(() => pl.isAdmin)
// admin_planta puede cambiar linea dentro de su planta; admin puede cambiar todo
const canChangeLinea  = computed(() => pl.isAdmin || pl.isAdminPlanta)
// Mostrar sección empresa cuando haya datos (para todos los roles)
const showEmpresaSelector = computed(() => pl.empresasVisibles.length > 0 || pl.empresaActual !== null)
// Sólo admin puede cambiar empresa (dropdown interactivo)
const canInteractEmpresa  = computed(() => pl.isAdmin && pl.empresasVisibles.length > 1)
// Mostrar sección planta cuando haya datos
const showPlantaSelector  = computed(() => pl.plantasVisibles.length > 0 || pl.plantaActual !== null)
// Admin y admin_planta pueden cambiar la planta
const canInteractPlanta   = computed(() => canChangePlanta.value && pl.plantasVisibles.length > 0)

const empresaName = computed(() => pl.empresaActual?.nombre ?? 'Empresa')

const connectionLabel = computed(() => {
  const labels: Record<string, string> = {
    EDGE: 'Edge · Conectado',
    CLOUD: 'Cloud · Conectado',
    HYBRID: 'Híbrido · Conectado',
    OFFLINE: 'Sin conexión'
  }
  return labels[connection.mode] || connection.mode
})

const isConnected = computed(() => connection.mode !== 'OFFLINE')
const isCloudOnly = computed(() => connection.isCloudOnly)

function navigate(path: string): void {
  router.push(path)
}

function handleLogout(): void {
  connection.logout()
  pl.reset()
  router.push('/login')
}

onMounted(() => {
  pl.load()
})

// Recargar plantas/líneas cuando la conexión edge se establece
watch(
  () => connection.edgeReachable,
  (reachable) => {
    if (reachable) pl.load()
  }
)

// Recargar plantas/líneas cuando el modo cloud se establece (resuelve caso donde
// restoreCloudSession() aún no había corrido cuando AppNav se montó)
watch(
  () => connection.cloudReachable,
  (reachable) => {
    if (reachable && !pl.loaded) pl.load()
  }
)
</script>

<template>
  <nav class="flex flex-col w-56 bg-edge-900 border-r border-edge-700/40 shrink-0 overflow-hidden">
    <!-- Brand -->
    <div class="px-4 pt-4 pb-3 border-b border-edge-700/30">
      <div class="flex items-center gap-2.5">
        <img src="/mentor-logo.png" alt="Mentor Textil" class="h-8 w-auto object-contain" />
        <div>
          <div class="text-[13px] font-bold text-edge-100 tracking-wide">MENTOR TEXTIL</div>
          <div class="flex items-center gap-1.5 mt-0.5">
            <span
              class="w-1.5 h-1.5 rounded-full"
              :class="isConnected ? 'bg-green-400 animate-pulse' : 'bg-red-400'"
            />
            <span
              class="text-[10px] font-medium"
              :class="isConnected ? 'text-green-400/80' : 'text-red-400/80'"
            >{{ connectionLabel }}</span>
          </div>
        </div>
      </div>
      <!-- Cloud mode indicator -->
      <div v-if="isCloudOnly" class="mt-2 px-2 py-1 rounded-md bg-blue-500/10 border border-blue-500/30">
        <span class="text-[9px] font-semibold text-blue-400 tracking-wide">☁ MODO CLOUD</span>
      </div>
    </div>

<!-- Empresa (siempre visible cuando hay datos; dropdown solo para ADMIN) -->
    <div v-if="showEmpresaSelector" class="px-3 pt-3 pb-0 relative">
      <span class="text-[10px] font-semibold text-edge-500 uppercase tracking-widest px-1">Empresa</span>
      <button
        :disabled="!canInteractEmpresa"
        @click="canInteractEmpresa && (showEmpresaDropdown = !showEmpresaDropdown); showPlantaDropdown = false; showLineaDropdown = false"
        class="mt-1.5 w-full flex items-center gap-2 px-3 py-2 rounded-lg bg-edge-800/70 border border-edge-700/40 transition-all"
        :class="canInteractEmpresa ? 'hover:border-purple-500/40 cursor-pointer' : 'cursor-default opacity-80'"
      >
        <span class="flex-1 text-left text-xs font-medium text-edge-100 truncate">{{ empresaName }}</span>
        <SvgIcon v-if="canInteractEmpresa" name="chevron-down" :size="12" :class="showEmpresaDropdown ? 'text-edge-500 shrink-0 rotate-180 transition-transform' : 'text-edge-500 shrink-0 transition-transform'" />
      </button>
      <!-- Dropdown empresas (solo ADMIN) -->
      <div v-if="showEmpresaDropdown && canInteractEmpresa"
        class="absolute left-3 right-3 top-full mt-1 z-50 bg-edge-800 border border-edge-700/50 rounded-lg shadow-xl overflow-hidden">
        <button
          v-for="e in pl.empresasVisibles" :key="e.id"
          @click="pl.selectEmpresa(e.id); showEmpresaDropdown = false"
          class="w-full text-left px-3 py-2 text-xs hover:bg-edge-700/60 transition-colors"
          :class="pl.selectedEmpresaId === e.id ? 'text-purple-400 font-semibold bg-purple-500/10' : 'text-edge-200'"
          >{{ e.nombre }}</button>
      </div>
    </div>

<!-- Planta (siempre visible cuando hay datos; dropdown para admin/admin_planta) -->
    <div v-if="showPlantaSelector" class="px-3 pt-3 pb-0 relative">
      <span class="text-[10px] font-semibold text-edge-500 uppercase tracking-widest px-1">Planta</span>
      <button
        :disabled="!canInteractPlanta"
        @click="canInteractPlanta && (showPlantaDropdown = !showPlantaDropdown); showEmpresaDropdown = false; showLineaDropdown = false"
        class="mt-1.5 w-full flex items-center gap-2 px-3 py-2 rounded-lg bg-edge-800/70 border border-edge-700/40 transition-all"
        :class="canInteractPlanta ? 'hover:border-blue-500/40 cursor-pointer' : 'cursor-default opacity-80'"
      >
        <span class="flex-1 text-left text-xs font-medium text-edge-100 truncate">{{ plantaName }}</span>
        <SvgIcon v-if="canInteractPlanta" name="chevron-down" :size="12" :class="showPlantaDropdown ? 'text-edge-500 shrink-0 rotate-180 transition-transform' : 'text-edge-500 shrink-0 transition-transform'" />
      </button>
      <!-- Dropdown plantas (admin/admin_planta) -->
      <div v-if="showPlantaDropdown && canInteractPlanta"
        class="absolute left-3 right-3 top-full mt-1 z-50 bg-edge-800 border border-edge-700/50 rounded-lg shadow-xl overflow-hidden">
        <button
          v-for="p in pl.plantasVisibles" :key="p.id"
          @click="pl.selectPlanta(p.id); showPlantaDropdown = false"
          class="w-full text-left px-3 py-2 text-xs hover:bg-edge-700/60 transition-colors"
          :class="pl.selectedPlantaId === p.id ? 'text-blue-400 font-semibold bg-blue-500/10' : 'text-edge-200'"
        >{{ p.nombre }}</button>
        <div v-if="pl.plantasVisibles.length === 0" class="px-3 py-2 text-xs text-edge-500 italic">Sin plantas disponibles</div>
      </div>
    </div>

    <!-- Line selector -->
    <div class="px-3 py-3 border-b border-edge-700/30 relative">
      <span class="text-[10px] font-semibold text-edge-500 uppercase tracking-widest px-1">Línea</span>
      <button
        :disabled="!canChangeLinea"
        @click="canChangeLinea && (showLineaDropdown = !showLineaDropdown); showEmpresaDropdown = false; showPlantaDropdown = false"
        class="mt-2 w-full flex items-center gap-3 p-2.5 rounded-xl bg-edge-800/70 border border-edge-700/40 transition-all group"
        :class="canChangeLinea ? 'hover:border-production-active/40 hover:bg-edge-800 cursor-pointer' : 'cursor-default opacity-80'"
      >
        <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-production-active/15 text-production-active font-bold text-xs tracking-wider group-hover:bg-production-active/25 transition-colors">
          {{ lineCode }}
        </div>
        <div class="flex-1 text-left min-w-0">
          <div class="text-xs font-semibold text-edge-100 truncate">{{ lineName }}</div>
          <div class="text-[10px] text-edge-400 truncate">{{ lineDesc }}</div>
        </div>
        <SvgIcon v-if="canChangeLinea" name="chevron-down" :size="14" :class="showLineaDropdown ? 'text-edge-500 shrink-0 group-hover:text-edge-300 rotate-180 transition-transform' : 'text-edge-500 shrink-0 group-hover:text-edge-300 transition-transform'" />
      </button>
      <!-- Dropdown líneas -->
      <div v-if="showLineaDropdown && canChangeLinea"
        class="absolute left-3 right-3 top-full mt-1 z-50 bg-edge-800 border border-edge-700/50 rounded-lg shadow-xl overflow-hidden max-h-52 overflow-y-auto">
        <button
          v-for="l in pl.lineasVisibles" :key="l.id"
          @click="pl.selectLinea(l.id); showLineaDropdown = false"
          class="w-full text-left px-3 py-2 text-xs hover:bg-edge-700/60 transition-colors"
          :class="pl.selectedLineaId === l.id ? 'text-production-active font-semibold bg-production-active/10' : 'text-edge-200'"
        >{{ l.nombre }}</button>
        <div v-if="pl.lineasVisibles.length === 0" class="px-3 py-2 text-xs text-edge-500 italic">Sin líneas disponibles</div>
      </div>
    </div>

    <!-- Navigation -->
    <div class="flex-1 px-3 py-3 overflow-y-auto">
      <span class="text-[10px] font-semibold text-edge-500 uppercase tracking-widest px-1 mb-2 block">Navegación</span>
      <div class="space-y-1">
        <button
          v-for="item in items"
          :key="item.id"
          class="relative w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-150"
          :class="[
            route.path.startsWith(item.route)
              ? 'bg-production-active/15 text-production-active shadow-sm shadow-production-active/10'
              : 'text-edge-400 hover:text-edge-200 hover:bg-edge-800/60'
          ]"
          @click="navigate(item.route)"
        >
          <SvgIcon :name="item.icon" :size="18" />
          <span class="text-xs font-medium flex-1 text-left">{{ item.label }}</span>
          <span
            v-if="item.badge"
            class="flex items-center justify-center min-w-[20px] h-5 px-1.5 text-[10px] font-bold rounded-full bg-stop-assigned text-white"
          >
            {{ item.badge > 99 ? '99+' : item.badge }}
          </span>
        </button>
      </div>
    </div>

    <!-- Bottom -->
    <div class="px-3 py-3 border-t border-edge-700/30 space-y-1">
      <button
        class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-edge-500 hover:text-edge-300 hover:bg-edge-800/60 transition-colors"
        @click="navigate('/device')"
      >
        <SvgIcon name="wifi" :size="16" />
        <span class="text-[10px] font-medium">Dispositivo</span>
      </button>
      <button
        class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-red-500/70 hover:text-red-400 hover:bg-red-500/10 transition-colors"
        @click="handleLogout"
      >
        <SvgIcon name="close" :size="16" />
        <span class="text-[10px] font-medium">Cerrar sesion</span>
      </button>
    </div>
  </nav>
</template>
