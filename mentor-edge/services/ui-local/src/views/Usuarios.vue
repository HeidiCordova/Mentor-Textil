<template>
  <div>
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-white">Usuarios Sincronizados</h1>
        <p class="text-sm text-slate-400 mt-0.5">Operadores y admins disponibles en este dispositivo edge</p>
      </div>
      <button @click="load" :disabled="loading"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-700 hover:bg-slate-600 text-slate-200 disabled:opacity-50 transition-colors">
        <svg class="w-3.5 h-3.5" :class="loading ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Actualizar
      </button>
    </div>

    <!-- Error -->
    <div v-if="error" class="mb-4 px-4 py-3 rounded-lg bg-red-900/30 border border-red-700/40 text-sm text-red-300 flex items-center gap-2">
      <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
      </svg>
      {{ error }}
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading && !usuarios.length" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="i in 6" :key="i" class="h-24 rounded-xl bg-slate-800 animate-pulse"/>
    </div>

    <!-- Sin datos -->
    <div v-else-if="!loading && !usuarios.length && !error"
      class="flex flex-col items-center justify-center py-20 text-slate-500">
      <svg class="w-12 h-12 mb-3 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
      </svg>
      <p class="text-sm">No hay usuarios sincronizados</p>
      <p class="text-xs mt-1 text-slate-600">Espera el próximo ciclo de sync desde la nube</p>
    </div>

    <!-- Tabla -->
    <div v-else-if="usuarios.length" class="rounded-xl border border-slate-700/60 overflow-hidden">
      <!-- Header de tabla -->
      <div class="grid grid-cols-[1fr_1fr_1fr_auto] gap-3 px-4 py-2.5 bg-slate-800/80 border-b border-slate-700/60 text-xs font-semibold text-slate-400 uppercase tracking-wider">
        <span>Nombre</span>
        <span>Usuario</span>
        <span>Rol</span>
        <span class="text-right">Estado</span>
      </div>

      <!-- Filas -->
      <div v-for="u in usuarios" :key="u.id"
        class="grid grid-cols-[1fr_1fr_1fr_auto] gap-3 px-4 py-3 border-b border-slate-700/40 last:border-0 hover:bg-slate-800/40 transition-colors items-center">

        <!-- Nombre + apellido -->
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-8 h-8 rounded-full bg-blue-600/20 flex items-center justify-center shrink-0">
            <span class="text-xs font-bold text-blue-400 uppercase">{{ initials(u) }}</span>
          </div>
          <div class="min-w-0">
            <p class="text-sm font-medium text-white truncate">{{ u.nombre }}{{ u.apellido ? ' ' + u.apellido : '' }}</p>
            <p class="text-xs text-slate-500 truncate">ID {{ u.id }}</p>
          </div>
        </div>

        <!-- Username -->
        <span class="text-sm text-slate-300 font-mono truncate">{{ u.username }}</span>

        <!-- Rol -->
        <span class="inline-flex items-center">
          <span :class="rolBadgeClass(u.rol)"
            class="px-2 py-0.5 rounded-full text-xs font-medium">
            {{ u.rol || '—' }}
          </span>
        </span>

        <!-- Estado -->
        <div class="flex justify-end">
          <span v-if="u.activo" class="flex items-center gap-1 text-xs text-emerald-400">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block"/>
            Activo
          </span>
          <span v-else class="flex items-center gap-1 text-xs text-slate-500">
            <span class="w-1.5 h-1.5 rounded-full bg-slate-500 inline-block"/>
            Inactivo
          </span>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <p v-if="usuarios.length" class="mt-3 text-xs text-slate-600 text-right">
      {{ usuarios.length }} usuario{{ usuarios.length !== 1 ? 's' : '' }} disponible{{ usuarios.length !== 1 ? 's' : '' }} en este Jetson
    </p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL || '/api/gateway'

const usuarios = ref([])
const loading  = ref(false)
const error    = ref('')

async function load() {
  loading.value = true
  error.value   = ''
  try {
    const res = await axios.get(`${GATEWAY_URL}/edge/auth/operators`, { timeout: 5000 })
    usuarios.value = Array.isArray(res.data) ? res.data : []
  } catch (e) {
    error.value = 'No se pudo obtener la lista de usuarios: ' + (e.message || 'error desconocido')
  } finally {
    loading.value = false
  }
}

function initials(u) {
  const n = u.nombre?.[0] || ''
  const a = u.apellido?.[0] || ''
  return (n + a) || u.username?.[0] || '?'
}

function rolBadgeClass(rol) {
  const r = (rol || '').toLowerCase()
  if (r.includes('admin')) return 'bg-purple-500/20 text-purple-300'
  if (r.includes('supervisor')) return 'bg-amber-500/20 text-amber-300'
  if (r.includes('operador') || r.includes('operator')) return 'bg-blue-500/20 text-blue-300'
  return 'bg-slate-600/40 text-slate-300'
}

onMounted(load)
</script>
