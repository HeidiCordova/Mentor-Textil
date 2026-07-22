<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import type { VelocidadNominalEntry } from '@/types'
import { api } from '@/services/api'
import { usePlantasLineasStore } from '@/stores/plantasLineas'
import { useProductionRunsStore } from '@/stores/productionRuns'
import { useConfigStore } from '@/stores/config'

const emit = defineEmits<{ close: [] }>()

const pl = usePlantasLineasStore()
const runsStore = useProductionRunsStore()
const configStore = useConfigStore()

const filas = ref<VelocidadNominalEntry[]>([])
const editadas = ref<Record<number, { velocidad_us: number; factor_conv: number }>>({})
const motivo = ref('')
const loading = ref(false)
const guardando = ref(false)
const guardadoOk = ref(false)
const errorMsg = ref('')

// Selector de unidad de velocidad — internamente siempre se guarda en u/s
const UNIDADES_VEL = [
  { key: 'us',  label: 'u/seg',  factor: 1,    step: '0.000001', decimals: 6 },
  { key: 'um',  label: 'u/min',  factor: 60,   step: '0.001',    decimals: 3 },
  { key: 'uh',  label: 'u/hora', factor: 3600, step: '0.01',     decimals: 2 },
] as const
type UnidadKey = typeof UNIDADES_VEL[number]['key']

// Leer unidad configurada en el OEE config (vel_unit: 'us'|'uh') como default
function defaultUnidad(): UnidadKey {
  const oee = configStore.config.oee as Record<string, unknown> | undefined
  const vu = oee?.vel_unit
  if (vu === 'uh') return 'uh'
  if (vu === 'um') return 'um'
  return 'us'
}
const unidadKey = ref<UnidadKey>(defaultUnidad())
const unidad = computed(() => UNIDADES_VEL.find(u => u.key === unidadKey.value)!)

function toDisplay(us: number): string {
  const v = us * unidad.value.factor
  return v === 0 ? '' : v.toFixed(unidad.value.decimals).replace(/\.?0+$/, '')
}
function fromDisplay(val: string, productoId: number): void {
  const num = parseFloat(val)
  if (!isNaN(num)) {
    editadas.value[productoId].velocidad_us = num / unidad.value.factor
  }
}

// Motivos cargados desde la API (los hardcodeados son el fallback)
const MOTIVOS_FALLBACK = [
  'Ajuste de velocidad de línea',
  'Cambio de formato / producto',
  'Optimización de rendimiento',
  'Corrección por calidad',
  'Instrucción de mantenimiento',
  'Orden de supervisión',
  'Calibración de equipo',
  'Otro',
]
const motivosApi = ref<string[]>([])
const MOTIVOS = computed(() => motivosApi.value.length ? motivosApi.value : MOTIVOS_FALLBACK)

async function cargarMotivos() {
  try {
    const res = await api.getMotivosVelocidad()
    const textos = (res.data ?? []).filter((m) => m.activo).map((m) => m.texto)
    if (textos.length) motivosApi.value = textos
  } catch {
    // silencioso — usa fallback
  }
}

// SKU del producto que está corriendo ahora mismo (misma lógica que AppHeader)
const skuActivo = computed(() => {
  const runs = runsStore.runs
  if (!runs.length) return null
  const open = runs.find((r) => !r.ended_at)
  const run = open ?? runs.reduce((a, b) =>
    new Date(a.started_at) > new Date(b.started_at) ? a : b
  )
  return run?.sku ?? null
})

// Filtro: por defecto muestra solo el producto activo cuando hay uno corriendo
const soloActivo = ref(true)

const filasVisibles = computed(() =>
  soloActivo.value && skuActivo.value
    ? filas.value.filter((f) => f.sku === skuActivo.value)
    : filas.value
)

watch(skuActivo, (val) => {
  if (!val) soloActivo.value = false
})

const hayCambios = computed(() =>
  filas.value.some((f) => {
    const d = editadas.value[f.producto_id]
    const velChanged = d && (Number(d.velocidad_us) !== f.velocidad_us || Number(d.factor_conv) !== f.factor_conv)
    return velChanged
  })
)

const sinVelocidad = computed(
  () => filas.value.filter((f) => (editadas.value[f.producto_id]?.velocidad_us ?? 0) === 0).length
)

async function cargar() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [vnData] = await Promise.all([
      api.velocidadNominal({ linea_id: pl.selectedLineaId ?? undefined }),
      runsStore.runs.length === 0
        ? runsStore.fetchRuns({ linea_id: pl.selectedLineaId ?? undefined, limit: 20 })
        : Promise.resolve(),
      cargarMotivos()
    ])
    filas.value = vnData
    editadas.value = {}
    filas.value.forEach((f) => {
      editadas.value[f.producto_id] = { velocidad_us: f.velocidad_us, factor_conv: f.factor_conv }
    })
  } catch (e: any) {
    errorMsg.value = e?.message ?? 'Error al cargar'
  } finally {
    loading.value = false
  }
}

async function guardar() {
  guardando.value = true
  guardadoOk.value = false
  errorMsg.value = ''
  try {
    const items = filas.value.map((f) => ({
      producto_id: f.producto_id,
      velocidad_us: Number(editadas.value[f.producto_id]?.velocidad_us ?? 0),
      factor_conv: Number(editadas.value[f.producto_id]?.factor_conv ?? 1),
      motivo: motivo.value || undefined
    }))
    await api.updateVelocidadNominal(items)
    // reflejar los nuevos valores en filas
    filas.value = filas.value.map((f) => ({
      ...f,
      velocidad_us: editadas.value[f.producto_id]?.velocidad_us ?? f.velocidad_us,
      factor_conv: editadas.value[f.producto_id]?.factor_conv ?? f.factor_conv
    }))
    motivo.value = ''
    guardadoOk.value = true
    setTimeout(() => { guardadoOk.value = false }, 2500)
  } catch (e: any) {
    errorMsg.value = e?.message ?? 'Error al guardar'
  } finally {
    guardando.value = false
  }
}

onMounted(async () => {
  // Asegurar que el config está cargado (puede que no se haya llamado fetchConfig aún)
  const oee = configStore.config.oee as Record<string, unknown> | undefined
  if (!oee?.vel_unit) {
    await configStore.fetchConfig()
  }
  unidadKey.value = defaultUnidad()
  cargar()
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
    @click.self="emit('close')"
  >
    <div class="w-full max-w-3xl mx-4 bg-white rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">

      <!-- Encabezado -->
      <div class="px-5 py-3 bg-indigo-700 flex items-center justify-between shrink-0">
        <div>
          <h2 class="text-sm font-bold text-white">Velocidad Nominal por Producto</h2>
          <p class="text-xs text-indigo-200 mt-0.5">Edita y guarda para actualizar el OEE en tiempo real</p>
        </div>
        <button class="text-indigo-200 hover:text-white transition-colors" @click="emit('close')">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Barra de acciones -->
      <div class="px-5 py-2.5 bg-gray-50 border-b flex items-center justify-between gap-3 shrink-0">
        <div class="flex items-center gap-2 flex-wrap">              <!-- Selector de unidad de velocidad -->
              <div class="flex items-center gap-1">
                <span class="text-xs text-gray-500 font-medium">Unidad:</span>
                <div class="flex rounded-md border border-gray-300 overflow-hidden">
                  <button
                    v-for="u in UNIDADES_VEL" :key="u.key"
                    class="px-2.5 py-0.5 text-xs font-semibold transition-colors"
                    :class="unidadKey === u.key
                      ? 'bg-indigo-600 text-white'
                      : 'bg-white text-gray-600 hover:bg-gray-100'"
                    @click="unidadKey = u.key"
                  >{{ u.label }}</button>
                </div>
              </div>          <span
            v-if="skuActivo"
            class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-green-100 text-green-700 text-xs font-semibold"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span>
            Corriendo: {{ skuActivo }}
          </span>
          <button
            v-if="skuActivo"
            class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full border text-xs font-semibold transition-colors"
            :class="soloActivo
              ? 'bg-indigo-100 border-indigo-300 text-indigo-700'
              : 'bg-gray-100 border-gray-300 text-gray-500 hover:bg-gray-200'"
            @click="soloActivo = !soloActivo"
          >
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z"/>
            </svg>
            {{ soloActivo ? 'Solo activo' : 'Ver todos' }}
          </button>
          <span v-if="sinVelocidad > 0" class="text-xs text-amber-600 font-medium">
            ⚠ {{ sinVelocidad }} sin velocidad configurada
          </span>
        </div>
        <div class="flex items-center gap-2">
          <span v-if="guardadoOk" class="text-xs text-green-600 font-semibold">✓ Guardado</span>
          <span v-if="errorMsg" class="text-xs text-red-600">{{ errorMsg }}</span>
          <button
            class="px-4 py-1.5 rounded-lg bg-indigo-600 text-white text-xs font-bold hover:bg-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            :disabled="guardando || !hayCambios"
            @click="guardar"
          >{{ guardando ? 'Guardando...' : 'Guardar cambios' }}</button>
        </div>
      </div>

      <!-- Motivo del cambio -->
      <div class="px-5 py-2 bg-amber-50 border-b border-amber-200 flex items-center gap-3 shrink-0">
        <svg class="w-4 h-4 text-amber-600 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
        </svg>
        <label class="text-xs font-semibold text-amber-700 shrink-0 whitespace-nowrap">Motivo del cambio</label>
        <select
          v-model="motivo"
          class="flex-1 text-sm border border-amber-300 rounded-md px-2 py-1 bg-white text-gray-800 focus:outline-none focus:ring-2 focus:ring-amber-400"
        >
          <option value="" disabled>Seleccionar motivo...</option>
          <option v-for="m in MOTIVOS" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>

      <!-- Cuerpo -->
      <div class="overflow-y-auto flex-1">
        <!-- Estado vacío / cargando -->
        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-400 text-sm">
          Cargando productos...
        </div>
        <div v-else-if="filas.length === 0" class="flex flex-col items-center justify-center py-16 text-gray-400 text-sm gap-1">
          <span>No hay productos asignados a esta línea.</span>
          <span class="text-xs">Agrégalos desde Administración → Productos en el portal cloud.</span>
        </div>

        <!-- Tabla -->
        <table v-else class="w-full text-sm">
          <thead class="bg-gray-50 border-b">
            <tr>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide w-10">N°</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">SKU</th>
              <th class="px-4 py-2.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Descripción</th>
              <th class="px-4 py-2.5 text-right text-xs font-semibold text-gray-500 uppercase tracking-wide">Vel. nominal<br><span class="normal-case font-normal">({{ unidad.label }})</span></th>
              <th class="px-4 py-2.5 text-right text-xs font-semibold text-gray-500 uppercase tracking-wide w-24">Factor<br>conv.</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr
              v-for="(fila, idx) in filasVisibles"
              :key="fila.producto_id"
              :class="[
                'transition-colors',
                fila.sku === skuActivo ? 'bg-green-50 hover:bg-green-100/60' : 'hover:bg-gray-50',
                editadas[fila.producto_id]?.velocidad_us !== fila.velocidad_us ||
                editadas[fila.producto_id]?.factor_conv !== fila.factor_conv
                  ? 'ring-1 ring-inset ring-indigo-300' : ''
              ]"
            >
              <td class="px-4 py-2 text-gray-400 text-xs">{{ idx + 1 }}</td>
              <td class="px-4 py-2 font-mono text-xs font-semibold text-gray-700">
                <span class="flex items-center gap-1.5">
                  {{ fila.sku }}
                  <span
                    v-if="fila.sku === skuActivo"
                    class="w-1.5 h-1.5 rounded-full bg-green-500 shrink-0"
                  ></span>
                </span>
              </td>
              <td class="px-4 py-2 text-gray-600 text-xs">{{ fila.descripcion }}</td>
              <td class="px-4 py-2 text-right">
                <input
                  type="number"
                  min="0"
                  :step="unidad.step"
                  class="w-28 text-right px-2 py-1 border rounded-md text-sm text-gray-800 focus:outline-none focus:ring-2 focus:ring-indigo-400"
                  :class="(editadas[fila.producto_id]?.velocidad_us ?? 0) === 0 ? 'border-amber-300 bg-amber-50' : 'border-gray-300'"
                  :value="toDisplay(editadas[fila.producto_id]?.velocidad_us ?? 0)"
                  @change="fromDisplay(($event.target as HTMLInputElement).value, fila.producto_id)"
                  @keyup.enter="guardar"
                />
              </td>
              <td class="px-4 py-2 text-right">
                <input
                  type="number"
                  min="1"
                  step="1"
                  class="w-16 text-right px-2 py-1 border border-gray-300 rounded-md text-sm text-gray-800 focus:outline-none focus:ring-2 focus:ring-indigo-400"
                  v-model.number="editadas[fila.producto_id].factor_conv"
                  @keyup.enter="guardar"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Footer -->
      <div v-if="filas.length > 0" class="px-5 py-2.5 bg-gray-50 border-t text-xs text-gray-400 shrink-0">
        <span v-if="soloActivo && skuActivo">
          Mostrando producto activo · 
          <button class="underline text-indigo-400" @click="soloActivo = false">ver los {{ filas.length }}</button>
        </span>
        <span v-else>{{ filas.length }} producto{{ filas.length !== 1 ? 's' : '' }}</span>
        · Los cambios se aplican inmediatamente al OEE del edge y se sincronizan al cloud.
      </div>
    </div>
  </div>
</template>
