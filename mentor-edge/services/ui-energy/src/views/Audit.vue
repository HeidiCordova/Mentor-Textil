<template>
  <div class="space-y-6">

    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Auditoria</h1>
        <p class="mt-0.5 text-sm text-gray-400">Historial de escrituras de parametros a los medidores</p>
      </div>
      <button @click="load" :disabled="loading"
        class="flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-lg
               bg-slate-700 text-gray-300 hover:bg-slate-600 transition-colors disabled:opacity-40">
        <svg :class="['w-3.5 h-3.5', loading ? 'animate-spin' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Actualizar
      </button>
    </div>

    <div class="card p-0 overflow-hidden">
      <div v-if="loading" class="py-16 flex justify-center">
        <svg class="w-5 h-5 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
        </svg>
      </div>

      <div v-else-if="error" class="py-12 text-center text-sm text-red-400">
        {{ error }}
      </div>

      <div v-else-if="entries.length === 0" class="py-16 text-center">
        <p class="text-sm text-gray-500">Sin operaciones registradas.</p>
        <p class="text-xs text-gray-600 mt-1">Las escrituras aparecen aqui despues de usar "Programar medidor".</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-xs">
          <thead>
            <tr class="border-b border-slate-700/60">
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Fecha/Hora (UTC)</th>
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Medidor</th>
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Unit ID</th>
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Comando</th>
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Parametros</th>
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Resultado</th>
              <th class="text-left px-4 py-3 text-gray-500 font-medium">Detalle</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-700/40">
            <tr v-for="e in entries" :key="e.id" class="hover:bg-slate-800/40 transition-colors">
              <td class="px-4 py-3 font-mono text-gray-400 whitespace-nowrap">{{ fmtTs(e.ts) }}</td>
              <td class="px-4 py-3 text-gray-300">{{ e.meter_id || '—' }}</td>
              <td class="px-4 py-3 text-gray-400">{{ e.unit_id ?? '—' }}</td>
              <td class="px-4 py-3">
                <span class="px-1.5 py-0.5 rounded font-mono bg-blue-900/30 text-blue-300 border border-blue-800/40">
                  {{ e.command }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-500 max-w-[200px] truncate">{{ fmtParams(e.params) }}</td>
              <td class="px-4 py-3">
                <span :class="['px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wide',
                  e.result === 'ok'
                    ? 'bg-green-900/40 text-green-400 border border-green-800/40'
                    : 'bg-red-900/40 text-red-400 border border-red-800/40']">
                  {{ e.result }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-500 max-w-[180px] truncate" :title="e.message">{{ e.message || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getAuditLog } from '../services/api.js'

const entries = ref([])
const loading = ref(false)
const error   = ref('')

function fmtTs(ts) {
  if (!ts) return '—'
  return ts.replace('T', ' ').replace(/\.\d+.*$/, '').replace('+00:00', '')
}

function fmtParams(raw) {
  if (!raw) return '—'
  try {
    const obj = typeof raw === 'string' ? JSON.parse(raw) : raw
    const skip = new Set(['meter_id'])
    return Object.entries(obj)
      .filter(([k]) => !skip.has(k))
      .map(([k, v]) => `${k}=${v}`)
      .join(', ') || '—'
  } catch {
    return String(raw)
  }
}

async function load() {
  loading.value = true
  error.value   = ''
  try {
    entries.value = await getAuditLog()
  } catch (e) {
    error.value = e.response?.data?.error ?? 'No se pudo cargar el historial'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
