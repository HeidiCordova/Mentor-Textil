<script setup lang="ts">
import { onMounted, computed } from 'vue'
import SvgIcon from '@/components/SvgIcon.vue'
import StatusIndicator from '@/components/StatusIndicator.vue'
import { useConnectionStore } from '@/stores/connection'
import { useMachineStore } from '@/stores/machine'

const connection = useConnectionStore()
const machine = useMachineStore()

const healthServices = computed(() => {
  if (!connection.health) return []
  const deps = connection.health.deps || {}
  return [
    { name: 'Vision Detector', status: deps['detector'] || 'unknown', port: 8001 },
    { name: 'Resiliencia', status: deps['resiliencia'] || 'unknown', port: 8002 },
    { name: 'Enviador', status: deps['enviador'] || 'unknown', port: 8003 },
    { name: 'Config Service', status: deps['config'] || 'unknown', port: 8004 },
    { name: 'Edge Gateway', status: connection.health.status, port: 8005 }
  ]
})

const bufferStats = computed(() => {
  const b = machine.bufferSummary
  if (!b) return []
  return [
    { label: 'Total', value: b.total_count, color: 'blue' as const },
    { label: 'Pendientes', value: b.pending_count, color: b.pending_count > 100 ? ('yellow' as const) : ('green' as const) },
    { label: 'Enviados', value: b.synced_count, color: 'green' as const },
    { label: 'Fallidos', value: b.dead_count, color: b.dead_count > 0 ? ('red' as const) : ('green' as const) }
  ]
})

function statusColor(status: string): string {
  const map: Record<string, string> = {
    ok: 'text-green-400',
    degraded: 'text-yellow-400',
    error: 'text-red-400'
  }
  return map[status] || 'text-edge-500'
}

function statusBg(status: string): string {
  const map: Record<string, string> = {
    ok: 'bg-green-500/10',
    degraded: 'bg-yellow-500/10',
    error: 'bg-red-500/10'
  }
  return map[status] || 'bg-edge-800'
}

onMounted(() => {
  connection.probe()
  if (!connection.isCloudOnly) machine.loadAll()
})
</script>

<template>
  <div class="flex flex-col h-full p-4 overflow-y-auto">
    <h2 class="text-base font-semibold text-edge-100 mb-4">Estado del Sistema</h2>

    <!-- Panel informativo en modo CLOUD -->
    <div v-if="connection.isCloudOnly" class="mb-6 p-4 rounded-xl bg-blue-500/10 border border-blue-500/20">
      <div class="flex items-center gap-2 mb-3">
        <span class="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
        <span class="text-xs font-semibold text-blue-400">☁ Conectado al Cloud</span>
      </div>
      <div class="space-y-2">
        <div class="flex justify-between text-xs">
          <span class="text-edge-500">URL Cloud</span>
          <span class="text-edge-300 font-mono text-[11px] truncate max-w-[60%]">{{ connection.cloudURL }}</span>
        </div>
        <div class="flex justify-between text-xs">
          <span class="text-edge-500">SSE Browser</span>
          <span :class="connection.sseConnected ? 'text-green-400' : 'text-yellow-400'">
            {{ connection.sseConnected ? 'Activo' : 'Reconectando...' }}
          </span>
        </div>
        <div class="flex justify-between text-xs">
          <span class="text-edge-500">Operador</span>
          <span class="text-edge-300">{{ connection.operatorId }}</span>
        </div>
        <div class="flex justify-between text-xs">
          <span class="text-edge-500">Modo</span>
          <span class="text-blue-400 font-semibold">CLOUD · Solo lectura</span>
        </div>
      </div>
      <p class="mt-3 text-[11px] text-edge-500 leading-relaxed">
        En modo Cloud no hay acceso a los servicios del dispositivo Jetson (health, buffer, eventos).
        El diagnóstico del hardware debe realizarse con conexión directa al edge.
      </p>
    </div>

    <template v-if="!connection.isCloudOnly">
    <div class="grid grid-cols-2 gap-4 mb-6">
      <div class="p-4 rounded-xl bg-edge-800 border border-edge-700/50">
        <h3 class="text-sm font-semibold text-edge-200 mb-3">Conexion</h3>
        <div class="space-y-2">
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Modo</span>
            <span class="text-edge-200 font-medium">{{ connection.mode }}</span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Edge URL</span>
            <span class="text-edge-300 font-mono text-[11px]">{{ connection.edgeURL || '--' }}</span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Edge</span>
            <span :class="connection.edgeReachable ? 'text-green-400' : 'text-red-400'">
              {{ connection.edgeReachable ? 'Conectado' : 'Desconectado' }}
            </span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">SSE</span>
            <span :class="connection.sseConnected ? 'text-green-400' : 'text-red-400'">
              {{ connection.sseConnected ? 'Activo' : 'Inactivo' }}
            </span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Operador</span>
            <span class="text-edge-300">{{ connection.operatorId }}</span>
          </div>
        </div>
      </div>

      <div class="p-4 rounded-xl bg-edge-800 border border-edge-700/50">
        <h3 class="text-sm font-semibold text-edge-200 mb-3">Dispositivo</h3>
        <div v-if="machine.status" class="space-y-2">
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Device ID</span>
            <span class="text-edge-200 font-mono text-[11px]">{{ machine.status.device_id }}</span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Uptime</span>
            <span class="text-edge-300">{{ machine.status.uptime ? Math.round(machine.status.uptime / 3600) + 'h' : '--' }}</span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Cloud</span>
            <span :class="machine.isCloudConnected ? 'text-green-400' : 'text-yellow-400'">
              {{ machine.isCloudConnected ? 'Sincronizado' : 'Local' }}
            </span>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-edge-500">Config Version</span>
            <span class="text-edge-300 tabular-nums">{{ machine.status.config_version ?? '--' }}</span>
          </div>
        </div>
        <div v-else class="text-xs text-edge-500 py-4 text-center">Sin datos del dispositivo</div>
      </div>
    </div>

    <div class="mb-6">
      <h3 class="text-sm font-semibold text-edge-200 mb-3">Servicios</h3>
      <div class="grid grid-cols-5 gap-2">
        <div
          v-for="svc in healthServices"
          :key="svc.name"
          class="flex flex-col items-center p-3 rounded-lg border border-edge-700/50"
          :class="statusBg(svc.status)"
        >
          <div class="flex items-center justify-center w-8 h-8 rounded-full mb-2" :class="statusBg(svc.status)">
            <span class="w-2.5 h-2.5 rounded-full" :class="statusColor(svc.status).replace('text-', 'bg-')" />
          </div>
          <span class="text-[11px] font-medium text-edge-200 text-center">{{ svc.name }}</span>
          <span class="text-[10px] text-edge-500">:{{ svc.port }}</span>
          <span class="text-[10px] mt-1" :class="statusColor(svc.status)">{{ svc.status }}</span>
        </div>
      </div>
    </div>

    <div v-if="bufferStats.length > 0">
      <h3 class="text-sm font-semibold text-edge-200 mb-3">Buffer de Eventos</h3>
      <div class="grid grid-cols-4 gap-3">
        <StatusIndicator
          v-for="stat in bufferStats"
          :key="stat.label"
          :label="stat.label"
          :value="stat.value"
          :color="stat.color"
        />
      </div>
    </div>

    <div class="mt-6 flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-edge-700/50 text-edge-200 hover:bg-edge-600/50 transition-colors"
        @click="connection.probe(); machine.loadAll()"
      >
        <SvgIcon name="refresh" :size="14" />
        Actualizar
      </button>
    </div>
    </template>

    <div class="mt-4 flex items-center gap-2">
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20 transition-colors"
        @click="connection.disconnect()"
      >
        <SvgIcon name="wifi-off" :size="14" />
        Cerrar sesión
      </button>
    </div>
  </div>
</template>
