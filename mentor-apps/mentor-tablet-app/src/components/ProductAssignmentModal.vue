<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useCatalogStore } from '@/stores/catalog'
import { usePlantasLineasStore } from '@/stores/plantasLineas'

const props = defineProps<{ timeStart: number; timeEnd: number }>()
const emit = defineEmits<{
  assign: [
    productoId: number | null,
    sku: string | null,
    description: string | null,
    startMs: number,
    endMs: number
  ]
  cancel: []
}>()

const catalog = useCatalogStore()
const pl = usePlantasLineasStore()
const skuFilter = ref('')
const descFilter = ref('')
const selectedSku = ref<string | null>(null)
const sinProgramacion = ref(false)
const openEnd = ref(false)   // ∞ — duración indeterminada

const toDatetimeLocal = (ms: number): string => {
  const d = new Date(ms)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Fecha inicio: slot clickeado (o ahora si es 0)
const editStart = ref(toDatetimeLocal(props.timeStart > 0 ? props.timeStart : Date.now()))
// Fecha fin: slot fin, o +1h si es abierto
const editEnd = ref(toDatetimeLocal(props.timeEnd > 0 ? props.timeEnd : Date.now() + 3600_000))

const startMs = computed(() => new Date(editStart.value).getTime())
const endMs   = computed(() => new Date(editEnd.value).getTime())

onMounted(() => {
  catalog.loaded = false
  catalog.loadAll(pl.selectedLineaId ?? undefined)
})

const filtered = computed(() =>
  catalog.products.filter(
    (p) =>
      p.sku.toLowerCase().includes(skuFilter.value.toLowerCase()) &&
      p.description.toLowerCase().includes(descFilter.value.toLowerCase())
  )
)

const selectedProduct = computed(() =>
  catalog.products.find((p) => p.sku === selectedSku.value) ?? null
)

const canAccept = computed(() =>
  (sinProgramacion.value || selectedSku.value !== null) &&
  (openEnd.value || startMs.value < endMs.value)
)

function adjustEnd(deltaMs: number): void {
  const cur = new Date(editEnd.value).getTime()
  editEnd.value = toDatetimeLocal(cur + deltaMs)
}

function select(sku: string): void {
  selectedSku.value = sku
  sinProgramacion.value = false
}

function toggleSinProgramacion(): void {
  sinProgramacion.value = !sinProgramacion.value
  if (sinProgramacion.value) selectedSku.value = null
}

function accept(): void {
  if (!canAccept.value) return
  if (sinProgramacion.value) {
    emit('assign', null, null, null, startMs.value, openEnd.value ? 0 : endMs.value)
    return
  }
  const p = catalog.products.find((x) => x.sku === selectedSku.value)
  if (p) {
    emit(
      'assign',
      p.producto_id,
      p.sku,
      p.description,
      startMs.value,
      openEnd.value ? 0 : endMs.value
    )
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
    @click.self="emit('cancel')"
  >
    <div class="w-full max-w-2xl mx-4 bg-white rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">

      <!-- Encabezado -->
      <div class="px-5 py-3 bg-gray-50 border-b shrink-0">
        <h2 class="text-base font-bold text-gray-800">
          Asignación de Producto
          <span v-if="selectedProduct" class="text-indigo-600">
            — {{ selectedProduct.description }}
          </span>
          <span v-else-if="sinProgramacion" class="text-amber-600">— Sin programación</span>
        </h2>
        <p class="text-xs text-gray-500 mt-0.5">
          Seleccione el producto fabricado y ajuste el rango de tiempo
        </p>
      </div>

      <!-- Cuerpo scrollable -->
      <div class="px-5 py-4 space-y-4 overflow-y-auto flex-1">

        <!-- Fechas -->
        <div class="grid grid-cols-2 gap-4">
          <!-- Desde -->
          <div>
            <span class="text-[11px] font-semibold text-gray-500 uppercase tracking-wide block mb-1">Desde</span>
            <div class="flex items-center gap-1 border rounded-lg overflow-hidden focus-within:ring-2 focus-within:ring-teal-400">
              <span class="px-2 py-1.5 bg-teal-50 border-r text-teal-600">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </span>
              <input
                v-model="editStart"
                type="datetime-local"
                class="flex-1 px-2 py-1.5 text-sm text-gray-900 bg-white focus:outline-none"
                style="color-scheme: light"
              />
            </div>
          </div>

          <!-- Hasta -->
          <div>
            <span class="text-[11px] font-semibold text-gray-500 uppercase tracking-wide block mb-1">Hasta</span>
            <div class="flex items-center gap-1">
              <!-- − 15min -->
              <button
                class="shrink-0 w-7 h-8 flex items-center justify-center rounded-lg border bg-gray-50 hover:bg-gray-100 text-gray-600 font-bold text-base transition-colors"
                :disabled="openEnd"
                @click="adjustEnd(-15 * 60_000)"
              >−</button>

              <!-- input fecha fin o ∞ -->
              <div class="flex-1 flex items-center border rounded-lg overflow-hidden focus-within:ring-2 focus-within:ring-indigo-400"
                :class="openEnd ? 'bg-indigo-50' : 'bg-white'">
                <span class="px-2 py-1.5 border-r"
                  :class="openEnd ? 'bg-indigo-100 text-indigo-600' : 'bg-indigo-50 text-indigo-600'">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                </span>
                <div v-if="openEnd" class="flex-1 px-3 py-1.5 flex items-center gap-1.5">
                  <span class="text-indigo-600 font-bold text-lg leading-none">∞</span>
                  <span class="text-xs text-indigo-500 font-medium">Indeterminado</span>
                </div>
                <input
                  v-else
                  v-model="editEnd"
                  type="datetime-local"
                  class="flex-1 px-2 py-1.5 text-sm text-gray-900 bg-white focus:outline-none"
                  style="color-scheme: light"
                />
              </div>

              <!-- + 15min -->
              <button
                class="shrink-0 w-7 h-8 flex items-center justify-center rounded-lg border bg-gray-50 hover:bg-gray-100 text-gray-600 font-bold text-base transition-colors"
                :disabled="openEnd"
                @click="adjustEnd(15 * 60_000)"
              >+</button>

              <!-- ∞ toggle -->
              <button
                class="shrink-0 w-9 h-8 flex items-center justify-center rounded-lg border font-bold text-base transition-colors"
                :class="openEnd
                  ? 'bg-indigo-600 border-indigo-600 text-white shadow-inner'
                  : 'bg-gray-50 border-gray-200 text-gray-500 hover:bg-indigo-50 hover:border-indigo-300 hover:text-indigo-600'"
                title="Duración indeterminada"
                @click="openEnd = !openEnd"
              >∞</button>
            </div>
          </div>
        </div>

        <p v-if="!openEnd && startMs >= endMs" class="text-xs text-red-500">
          La hora de inicio debe ser anterior a la hora de fin
        </p>

        <!-- Sin programación -->
        <button
          class="w-full text-left px-3 py-2 rounded-lg border text-xs font-medium transition-colors"
          :class="sinProgramacion
            ? 'bg-amber-50 border-amber-400 text-amber-700'
            : 'bg-gray-50 border-gray-200 text-gray-600 hover:bg-gray-100'"
          @click="toggleSinProgramacion"
        >
          ⚙ Sin programación (setup / sin producto asignado)
        </button>

        <!-- Tabla productos -->
        <div v-if="!sinProgramacion" class="border rounded-lg overflow-hidden">
          <div class="grid grid-cols-[120px_1fr] bg-gray-100 border-b">
            <div class="px-3 py-2 border-r">
              <span class="text-xs font-semibold text-gray-600">SKU</span>
              <input
                v-model="skuFilter"
                class="w-full mt-1 px-2 py-0.5 text-xs border rounded bg-white focus:outline-none focus:ring-1 focus:ring-blue-400"
                placeholder="Filtrar..."
              />
            </div>
            <div class="px-3 py-2">
              <span class="text-xs font-semibold text-gray-600">Descripción</span>
              <input
                v-model="descFilter"
                class="w-full mt-1 px-2 py-0.5 text-xs border rounded bg-white focus:outline-none focus:ring-1 focus:ring-blue-400"
                placeholder="Filtrar..."
              />
            </div>
          </div>
          <div class="max-h-52 overflow-y-auto divide-y">
            <button
              v-for="p in filtered"
              :key="p.sku"
              class="grid grid-cols-[120px_1fr] w-full text-left transition-colors"
              :class="selectedSku === p.sku
                ? 'bg-blue-600 text-white'
                : 'hover:bg-gray-50 text-gray-700'"
              @click="select(p.sku)"
            >
              <span class="px-3 py-2.5 border-r font-mono text-xs font-semibold">{{ p.sku }}</span>
              <span class="px-3 py-2.5 text-xs">{{ p.description }}</span>
            </button>
            <div v-if="filtered.length === 0" class="px-4 py-6 text-center text-sm text-gray-400">
              No se encontraron productos
            </div>
          </div>
        </div>
      </div>

      <!-- Botones -->
      <div class="flex items-center justify-end gap-3 px-5 py-3 bg-gray-50 border-t shrink-0">
        <button
          class="px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg transition-colors flex items-center gap-1.5"
          @click="emit('cancel')"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
          Cancelar
        </button>
        <button
          class="px-4 py-2 text-sm font-semibold text-white rounded-lg transition-colors flex items-center gap-1.5"
          :class="canAccept ? 'bg-blue-600 hover:bg-blue-700' : 'bg-gray-300 cursor-not-allowed'"
          :disabled="!canAccept"
          @click="accept"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          Aceptar
        </button>
      </div>

    </div>
  </div>
</template>
