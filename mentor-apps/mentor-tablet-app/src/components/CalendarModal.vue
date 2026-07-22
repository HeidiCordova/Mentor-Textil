<script setup lang="ts">
import { ref, computed } from 'vue'

const emit = defineEmits<{
  select: [ms: number]
  cancel: []
}>()

// Estado interno del mini-calendario
const today = new Date()
const viewYear = ref(today.getFullYear())
const viewMonth = ref(today.getMonth()) // 0-based

const DAYS = ['L', 'M', 'X', 'J', 'V', 'S', 'D']
const MONTHS = [
  'enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio',
  'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'
]

const selectedDateStr = ref<string | null>(null) // 'YYYY-MM-DD'

const monthLabel = computed(() => `${MONTHS[viewMonth.value]} ${viewYear.value}`)

interface CalDay {
  day: number
  month: number // 0-based
  year: number
  isCurrentMonth: boolean
  isToday: boolean
  isFuture: boolean
  dateStr: string
}

const calendarDays = computed((): CalDay[] => {
  const days: CalDay[] = []
  const first = new Date(viewYear.value, viewMonth.value, 1)
  // Monday = 0 offset (ISO week)
  let offset = first.getDay() - 1
  if (offset < 0) offset = 6

  const todayStr = formatDate(today)
  const nowMs = Date.now()

  // Days from previous month
  for (let i = 0; i < offset; i++) {
    const d = new Date(viewYear.value, viewMonth.value, -offset + i + 1)
    const ds = formatDate(d)
    days.push({
      day: d.getDate(), month: d.getMonth(), year: d.getFullYear(),
      isCurrentMonth: false,
      isToday: ds === todayStr,
      isFuture: d.getTime() > nowMs,
      dateStr: ds
    })
  }

  // Days of current month
  const daysInMonth = new Date(viewYear.value, viewMonth.value + 1, 0).getDate()
  for (let d = 1; d <= daysInMonth; d++) {
    const date = new Date(viewYear.value, viewMonth.value, d)
    const ds = formatDate(date)
    days.push({
      day: d, month: viewMonth.value, year: viewYear.value,
      isCurrentMonth: true,
      isToday: ds === todayStr,
      isFuture: date.getTime() > nowMs,
      dateStr: ds
    })
  }

  // Fill up to 42 (6 rows)
  while (days.length < 42) {
    const last = days[days.length - 1]
    const next = new Date(last.year, last.month, last.day + 1)
    const ds = formatDate(next)
    days.push({
      day: next.getDate(), month: next.getMonth(), year: next.getFullYear(),
      isCurrentMonth: false,
      isToday: ds === todayStr,
      isFuture: next.getTime() > nowMs,
      dateStr: ds
    })
  }

  return days
})

function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${dd}`
}

function prevMonth(): void {
  if (viewMonth.value === 0) { viewMonth.value = 11; viewYear.value-- }
  else viewMonth.value--
}

function nextMonth(): void {
  if (viewMonth.value === 11) { viewMonth.value = 0; viewYear.value++ }
  else viewMonth.value++
}

function selectDay(d: CalDay): void {
  if (d.isFuture) return
  selectedDateStr.value = d.dateStr
}

function confirm(): void {
  if (!selectedDateStr.value) return
  const [y, m, dd] = selectedDateStr.value.split('-').map(Number)
  const ms = new Date(y, m - 1, dd, 0, 0, 0, 0).getTime()
  emit('select', ms)
}
</script>

<template>
  <Teleport to="body">
    <!-- Overlay -->
    <div
      class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center"
      @click.self="$emit('cancel')"
    >
      <div class="bg-white rounded-2xl shadow-2xl w-full max-w-[320px] mx-4 overflow-hidden">
        <!-- Header -->
        <div class="flex items-center justify-between px-5 py-4 border-b border-slate-100">
          <span class="font-semibold text-slate-800 text-sm">Seleccionar Fecha</span>
          <button
            class="w-7 h-7 rounded-full bg-red-500 text-white flex items-center justify-center text-sm font-bold hover:bg-red-400 transition-colors"
            @click="$emit('cancel')"
          >
            ✕
          </button>
        </div>

        <!-- Calendar -->
        <div class="px-5 py-4">
          <!-- Month navigation -->
          <div class="flex items-center justify-between mb-4">
            <button
              class="w-7 h-7 rounded-full hover:bg-slate-100 flex items-center justify-center text-slate-500 hover:text-slate-800 transition-colors"
              @click="prevMonth"
            >
              ‹
            </button>
            <span class="text-sm font-semibold text-slate-700 capitalize">{{ monthLabel }}</span>
            <button
              class="w-7 h-7 rounded-full hover:bg-slate-100 flex items-center justify-center text-slate-500 hover:text-slate-800 transition-colors"
              @click="nextMonth"
            >
              ›
            </button>
          </div>

          <!-- Day headers -->
          <div class="grid grid-cols-7 mb-2">
            <div
              v-for="d in DAYS"
              :key="d"
              class="text-center text-[11px] font-bold text-slate-400 py-1"
            >
              {{ d }}
            </div>
          </div>

          <!-- Day cells -->
          <div class="grid grid-cols-7 gap-y-1">
            <button
              v-for="(d, i) in calendarDays"
              :key="i"
              :disabled="d.isFuture"
              class="h-8 w-full rounded-lg text-[13px] font-medium transition-colors"
              :class="[
                d.isFuture
                  ? 'text-slate-200 cursor-not-allowed'
                  : d.dateStr === selectedDateStr
                    ? 'bg-blue-600 text-white font-bold'
                    : d.isToday
                      ? 'bg-blue-50 text-blue-600 font-bold ring-1 ring-blue-300'
                      : d.isCurrentMonth
                        ? 'text-slate-700 hover:bg-slate-100'
                        : 'text-slate-300 hover:bg-slate-50'
              ]"
              @click="selectDay(d)"
            >
              {{ d.day }}
            </button>
          </div>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-end gap-2 px-5 py-3 border-t border-slate-100 bg-slate-50">
          <button
            class="px-4 py-1.5 rounded-lg text-sm text-slate-600 hover:bg-slate-200 transition-colors font-medium"
            @click="$emit('cancel')"
          >
            Cancelar
          </button>
          <button
            :disabled="!selectedDateStr"
            class="px-5 py-1.5 rounded-lg text-sm font-semibold transition-colors"
            :class="selectedDateStr
              ? 'bg-blue-600 text-white hover:bg-blue-500'
              : 'bg-slate-200 text-slate-400 cursor-not-allowed'"
            @click="confirm"
          >
            Ver día
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
