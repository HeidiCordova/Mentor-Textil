<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { Stop, StopType, CategoryTreeNode } from '@/types'
import { useCatalogStore } from '@/stores/catalog'
import { usePlantasLineasStore } from '@/stores/plantasLineas'

const props = defineProps<{
  mode: 'create' | 'justify' | 'multi'
  stop?: Stop
  timeStart?: number
  timeEnd?: number
  multiCount?: number
  slotTimeStart?: number
  slotTimeEnd?: number
}>()

const emit = defineEmits<{
  create: [stopType: StopType, categoriaId: number, category: string, startMs: number, endMs: number]
  assign: [stopId: string, categoriaId: number, category: string, tipoParada: string]
  multiAssign: [categoriaId: number, category: string, tipoParada: string]
  cancel: []
}>()

const catalog = useCatalogStore()
const pl = usePlantasLineasStore()

const selectedType = ref<StopType>(props.stop?.stop_type || 'OTRA')

const toDatetimeLocal = (ms: number): string => {
  const d = new Date(ms)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const defaultStart = props.timeStart || Date.now() - 30 * 60_000
const defaultEnd = props.timeEnd || Date.now()

const editStart = ref(toDatetimeLocal(defaultStart))
const editEnd = ref(toDatetimeLocal(defaultEnd))

const startMs = computed(() => new Date(editStart.value).getTime())
const endMs = computed(() => new Date(editEnd.value).getTime())

const expanded = ref<Record<number, boolean>>({})
const selectedNode = ref<CategoryTreeNode | null>(null)

function findNodeById(
  nodes: CategoryTreeNode[],
  targetId: number,
  path: number[] = []
): { node: CategoryTreeNode; path: number[] } | null {
  for (const n of nodes) {
    if (n.id === targetId) return { node: n, path }
    if (n.children?.length) {
      const result = findNodeById(n.children, targetId, [...path, n.id])
      if (result) return result
    }
  }
  return null
}

function applyPreselect(): void {
  if (props.mode !== 'justify' || !props.stop?.categoria_id) return
  const result = findNodeById(catalog.stopCategories, props.stop.categoria_id)
  if (result) {
    selectedNode.value = result.node
    result.path.forEach((id) => { expanded.value[id] = true })
  }
}

onMounted(async () => {
  catalog.loaded = false
  await catalog.loadAll(pl.selectedLineaId ?? undefined)
  applyPreselect()
})

watch(() => catalog.stopCategories, applyPreselect, { once: true })

function toggle(id: number): void {
  expanded.value[id] = !expanded.value[id]
}

function isLeaf(node: CategoryTreeNode): boolean {
  return !node.children || node.children.length === 0
}

function select(node: CategoryTreeNode): void {
  if (isLeaf(node)) {
    selectedNode.value = node
  } else {
    toggle(node.id)
  }
}

const canAccept = computed(() => {
  if (props.mode === 'create') {
    return selectedNode.value !== null && startMs.value < endMs.value
  }
  return selectedNode.value !== null
})

const fmtTime = (iso: string) => {
  const d = new Date(iso)
  const h = d.getHours()
  const m = d.getMinutes().toString().padStart(2, '0')
  const s = d.getSeconds().toString().padStart(2, '0')
  const ampm = h >= 12 ? 'PM' : 'AM'
  return `${h % 12 || 12}:${m}:${s} ${ampm}`
}

const fromTime = computed(() => {
  if (props.slotTimeStart) return fmtTime(new Date(props.slotTimeStart).toISOString())
  return props.stop ? fmtTime(props.stop.started_at) : ''
})
const toTime = computed(() => {
  if (props.slotTimeEnd) return fmtTime(new Date(props.slotTimeEnd).toISOString())
  return props.stop?.ended_at ? fmtTime(props.stop.ended_at) : 'En curso'
})
const isAssigned = computed(() => props.stop?.justified ?? false)

function accept(): void {
  if (!canAccept.value || !selectedNode.value) return
  if (props.mode === 'create') {
    emit('create', selectedType.value, selectedNode.value.id, selectedNode.value.nombre, startMs.value, endMs.value)
  } else if (props.mode === 'multi') {
    emit('multiAssign', selectedNode.value.id, selectedNode.value.nombre, selectedNode.value.tipo_parada || '')
  } else if (props.stop) {
    emit('assign', props.stop.stop_id, selectedNode.value.id, selectedNode.value.nombre, selectedNode.value.tipo_parada || '')
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
    @click.self="emit('cancel')"
  >
    <div class="w-full max-w-2xl bg-white rounded-xl shadow-2xl flex flex-col" style="max-height: 90vh;">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3 border-b shrink-0">
        <h2 class="text-lg font-bold text-gray-800">
          {{ mode === 'create' ? 'Registrar Parada' : mode === 'multi' ? 'Asignación múltiple' : 'Asignación de Parada' }}
        </h2>
        <div class="flex items-center gap-2">
          <template v-if="mode === 'multi'">
            <span class="px-3 py-1 rounded-full bg-cyan-100 text-cyan-700 text-sm font-bold border border-cyan-300">
              📌 {{ multiCount }} parada{{ multiCount !== 1 ? 's' : '' }} seleccionada{{ multiCount !== 1 ? 's' : '' }}
            </span>
          </template>
          <template v-else-if="mode === 'justify'">
            <span class="text-sm font-medium text-gray-500">Estado</span>
            <span
              class="px-4 py-1.5 rounded-md text-sm font-semibold text-white"
              :class="isAssigned ? 'bg-green-500' : 'bg-orange-400'"
            >
              {{ isAssigned ? 'Asignada' : 'Reasignación' }}
            </span>
          </template>
        </div>
      </div>

      <!-- Banner: categoría actualmente asignada (solo en modo justify con stop ya asignado) -->
      <div
        v-if="mode === 'justify' && stop?.justified && stop?.category"
        class="px-5 py-2.5 bg-green-50 border-b border-green-200 flex items-center gap-2 shrink-0"
      >
        <svg class="w-4 h-4 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span class="text-xs text-green-700 font-medium">Categoría asignada:</span>
        <span class="text-xs text-green-800 font-bold">{{ stop.category }}</span>
      </div>

      <!-- Tree section — ocupa todo el espacio disponible -->
      <div class="flex flex-col flex-1 min-h-0">
        <!-- "Tipo de Parada" sub-header -->
        <div class="bg-gray-100 border-b px-5 py-2 shrink-0">
          <span class="text-sm font-semibold text-gray-600 tracking-wide">Tipo de Parada</span>
        </div>

        <!-- Árbol de categorías con scroll -->
        <div class="flex-1 overflow-y-auto">
          <template v-for="root in catalog.stopCategories" :key="root.id">
            <!-- Nivel 0 -->
            <button
              class="flex items-center gap-2 w-full px-4 py-2.5 text-sm text-gray-700 hover:bg-gray-50 transition-colors text-left border-b border-gray-100"
              :class="selectedNode?.id === root.id && isLeaf(root) ? 'bg-blue-50 text-blue-700 font-medium' : ''"
              @click="select(root)"
            >
              <svg
                class="w-4 h-4 text-gray-400 transition-transform flex-shrink-0"
                :class="expanded[root.id] ? 'rotate-90' : ''"
                fill="none" stroke="currentColor" viewBox="0 0 24 24"
              >
                <path v-if="root.children?.length" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
              <svg class="w-4 h-4 text-amber-500 flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
                <path d="M2 6a2 2 0 012-2h5l2 2h9a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
              </svg>
              <span class="truncate font-medium">{{ root.nombre }}</span>
              <svg v-if="selectedNode?.id === root.id" class="w-4 h-4 text-green-500 ml-auto flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
              </svg>
            </button>

            <template v-if="expanded[root.id] && root.children?.length">
              <template v-for="child in root.children" :key="child.id">
                <!-- Nivel 1 -->
                <button
                  class="flex items-center gap-2 w-full pl-10 pr-4 py-2.5 text-sm transition-colors text-left border-b border-gray-100"
                  :class="selectedNode?.id === child.id && isLeaf(child)
                    ? 'bg-blue-50 text-blue-700 font-medium'
                    : 'text-gray-700 hover:bg-gray-50'"
                  @click="select(child)"
                >
                  <svg
                    class="w-3.5 h-3.5 text-gray-400 transition-transform flex-shrink-0"
                    :class="expanded[child.id] ? 'rotate-90' : ''"
                    fill="none" stroke="currentColor" viewBox="0 0 24 24"
                  >
                    <path v-if="child.children?.length" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                  </svg>
                  <svg class="w-4 h-4 text-amber-400 flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M2 6a2 2 0 012-2h5l2 2h9a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                  </svg>
                  <span class="truncate">{{ child.nombre }}</span>
                  <svg v-if="selectedNode?.id === child.id" class="w-4 h-4 text-green-500 ml-auto flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
                  </svg>
                </button>

                <template v-if="expanded[child.id] && child.children?.length">
                  <template v-for="grand in child.children" :key="grand.id">
                    <!-- Nivel 2 -->
                    <button
                      class="flex items-center gap-2 w-full pl-16 pr-4 py-2.5 text-sm transition-colors text-left border-b border-gray-100"
                      :class="selectedNode?.id === grand.id && isLeaf(grand)
                        ? 'bg-blue-50 text-blue-700 font-medium'
                        : 'text-gray-700 hover:bg-gray-50'"
                      @click="select(grand)"
                    >
                      <svg
                        class="w-3 h-3 text-gray-400 transition-transform flex-shrink-0"
                        :class="expanded[grand.id] ? 'rotate-90' : ''"
                        fill="none" stroke="currentColor" viewBox="0 0 24 24"
                      >
                        <path v-if="grand.children?.length" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                      </svg>
                      <svg class="w-4 h-4 text-amber-300 flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M2 6a2 2 0 012-2h5l2 2h9a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
                      </svg>
                      <span class="truncate">{{ grand.nombre }}</span>
                      <svg v-if="selectedNode?.id === grand.id" class="w-4 h-4 text-green-500 ml-auto flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
                      </svg>
                    </button>

                    <template v-if="expanded[grand.id] && grand.children?.length">
                      <!-- Nivel 3 (hoja) -->
                      <button
                        v-for="leaf in grand.children"
                        :key="leaf.id"
                        class="flex items-center gap-2 w-full pl-[5.5rem] pr-4 py-2 text-sm transition-colors text-left border-b border-gray-100"
                        :class="selectedNode?.id === leaf.id
                          ? 'bg-blue-100 text-blue-700 font-semibold'
                          : 'text-gray-600 hover:bg-gray-50'"
                        @click="select(leaf)"
                      >
                        <span
                          class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                          :class="selectedNode?.id === leaf.id ? 'bg-blue-500' : 'bg-gray-300'"
                        />
                        <span class="truncate">{{ leaf.nombre }}</span>
                        <svg v-if="selectedNode?.id === leaf.id" class="w-4 h-4 text-green-500 ml-auto flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
                        </svg>
                      </button>
                    </template>
                  </template>
                </template>
              </template>
            </template>
          </template>
        </div>
      </div>

      <!-- Banda inferior: Asignación de Parada + Desde/Hasta -->
      <div class="shrink-0">
        <div class="bg-gradient-to-r from-sky-400 to-blue-500 px-5 py-2 text-center">
          <span class="text-sm font-semibold text-white tracking-wide">Asignación de Parada</span>
        </div>
        <!-- justify: muestra tiempos del stop -->
        <div v-if="mode === 'justify'" class="grid grid-cols-2">
          <div class="bg-teal-400 px-5 py-3 flex items-center justify-between">
            <span class="text-sm text-white/80 font-medium">Desde</span>
            <span class="text-base font-mono font-bold text-white">{{ fromTime }}</span>
          </div>
          <div class="bg-indigo-500 px-5 py-3 flex items-center justify-between">
            <span class="text-sm text-white/80 font-medium">Hasta</span>
            <span class="text-base font-mono font-bold text-white">{{ toTime }}</span>
          </div>
        </div>
        <!-- multi: banner de paradas afectadas -->
        <div v-else-if="mode === 'multi'" class="bg-gradient-to-r from-cyan-500 to-blue-600 px-5 py-3 text-center">
          <span class="text-sm font-semibold text-white">
            Se asignará la categoría seleccionada a {{ multiCount }} parada{{ multiCount !== 1 ? 's' : '' }}
          </span>
        </div>
        <!-- create: tiles de solo lectura -->
        <div v-else class="grid grid-cols-2">
          <div class="bg-teal-400 px-5 py-3 flex items-center justify-between">
            <span class="text-sm text-white/80 font-medium">Desde</span>
            <span class="text-base font-mono font-bold text-white">{{ fmtTime(editStart) }}</span>
          </div>
          <div class="bg-indigo-500 px-5 py-3 flex items-center justify-between">
            <span class="text-sm text-white/80 font-medium">Hasta</span>
            <span class="text-base font-mono font-bold text-white">{{ fmtTime(editEnd) }}</span>
          </div>
        </div>
        <p v-if="mode === 'create' && startMs >= endMs" class="text-xs text-red-500 px-4 py-1">
          La hora de inicio debe ser anterior a la hora de fin
        </p>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-end gap-3 px-5 py-3 bg-gray-50 border-t shrink-0">
        <button
          class="px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg transition-colors flex items-center gap-1"
          @click="emit('cancel')"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
          ✕ Cancelar
        </button>
        <button
          class="px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors flex items-center gap-1"
          :class="canAccept ? 'bg-blue-600 hover:bg-blue-700' : 'bg-gray-300 cursor-not-allowed'"
          :disabled="!canAccept"
          @click="accept"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          ✓ {{ mode === 'create' ? 'Registrar' : mode === 'multi' ? `Asignar ${multiCount} parada${multiCount !== 1 ? 's' : ''}` : 'Aceptar' }}
        </button>
      </div>
    </div>
  </div>
</template>
