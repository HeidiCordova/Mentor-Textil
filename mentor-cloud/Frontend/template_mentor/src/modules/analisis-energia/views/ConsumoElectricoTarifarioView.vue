<script setup>
import { ref, computed, onMounted } from 'vue'
import Card from '@/shared/components/ui/Card.vue'
import Button from '@/shared/components/ui/Button.vue'

const companias = ref([
  { id: 1, nombre: 'Industrias San Miguel' }
])

const plantas = ref([
  { id: 1, companiaId: 1, nombre: 'ISM Arequipa' }
])

const lineas = ref([
  { id: 1, nombre: 'LINEA 05' },
  { id: 2, nombre: 'Línea 1' }
])

const ubicaciones = ref([
  { id: 1, nombre: 'Toda la Línea' },
  { id: 2, nombre: 'Linea 05 Locacion' }
])

const dispositivos = ref([
  { id: 1, nombre: 'Medidor de Energía - Circutor' },
  { id: 2, nombre: 'Linea 05 Produccion' }
])

const agrupamientos = ref([
  { id: 1, nombre: '5 minutos', valor: '5min' },
  { id: 2, nombre: '15 minutos', valor: '15min' },
  { id: 3, nombre: '30 minutos', valor: '30min' },
  { id: 4, nombre: '1 hora', valor: '1h' }
])

const estructurasTarifarias = ref([
  { id: 1, nombre: 'MT3', valor: 'mt3' },
  { id: 2, nombre: 'MT4', valor: 'mt4' },
  { id: 3, nombre: 'BT3', valor: 'bt3' }
])

const variables = ref([
  { id: 1, nombre: '∫ EReal (kWh) (Energia Real (kWh))', valor: 'energia_real' },
  { id: 2, nombre: '∫ EReactiva (kVArh) (Energia Reactiva (kVArh))', valor: 'energia_reactiva' }
])

const companiaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const lineaSeleccionada = ref(1)
const ubicacionSeleccionada = ref(1)
const dispositivoSeleccionado = ref(1)
const agrupamientoSeleccionado = ref('5min')
const estructuraTarifariaSeleccionada = ref('mt3')
const variableSeleccionada = ref('energia_real')

const fechaInicio = ref('2025/11/01 00:00')
const fechaFin = ref('2025/11/19 23:59')

const fechaInicioInput = computed({
  get: () => {
    const [fecha, hora] = fechaInicio.value.split(' ')
    const [year, month, day] = fecha.split('/')
    return `${year}-${month}-${day}T${hora}`
  },
  set: (value) => {
    if (value) {
      const date = new Date(value)
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      fechaInicio.value = `${year}/${month}/${day} ${hours}:${minutes}`
    }
  }
})

const fechaFinInput = computed({
  get: () => {
    const [fecha, hora] = fechaFin.value.split(' ')
    const [year, month, day] = fecha.split('/')
    return `${year}-${month}-${day}T${hora}`
  },
  set: (value) => {
    if (value) {
      const date = new Date(value)
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      fechaFin.value = `${year}/${month}/${day} ${hours}:${minutes}`
    }
  }
})

const plantasFiltradas = computed(() => {
  if (!companiaSeleccionada.value) return []
  return plantas.value.filter(p => p.companiaId === parseInt(companiaSeleccionada.value))
})

const datosConsumo = ref([])
const mostrarResultados = ref(false)
const datosGrafico = ref({})

const tarifas = ['HP', 'HFP', 'HP PLUS', 'BASE']
const coloresTarifas = {
  'HP': '#ef4444',
  'HFP': '#f59e0b',
  'HP PLUS': '#8b5cf6',
  'BASE': '#3b82f6'
}

function buscar() {
  generarDatosConsumo()
  mostrarResultados.value = true
}

function generarDatosConsumo() {
  const datos = []
  const fechaInicial = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T') + ':00')
  const fechaFinal = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T') + ':00')
  
  const dias = Math.ceil((fechaFinal - fechaInicial) / (1000 * 60 * 60 * 24))
  
  for (let i = 0; i < dias; i++) {
    const fecha = new Date(fechaInicial.getTime() + (i * 24 * 60 * 60 * 1000))
    
    const hp = 150 + Math.random() * 100
    const hfp = 200 + Math.random() * 120
    const hpPlus = 180 + Math.random() * 90
    const base = 120 + Math.random() * 80
    
    datos.push({
      fecha: fecha.toISOString().split('T')[0],
      HP: Math.round(hp * 100) / 100,
      HFP: Math.round(hfp * 100) / 100,
      'HP PLUS': Math.round(hpPlus * 100) / 100,
      BASE: Math.round(base * 100) / 100,
      total: Math.round((hp + hfp + hpPlus + base) * 100) / 100
    })
  }
  
  datosConsumo.value = datos
  
  // Generar datos para gráficos
  datosGrafico.value = {}
  tarifas.forEach(tarifa => {
    datosGrafico.value[tarifa] = datos.map(d => ({
      fecha: d.fecha,
      valor: d[tarifa]
    }))
  })
}

function generarPathTarifa(tarifa) {
  const datos = datosGrafico.value[tarifa]
  if (!datos || datos.length === 0) return ''
  
  const maxValor = Math.max(...Object.values(datosGrafico.value).flat().map(d => d.valor))
  const minValor = Math.min(...Object.values(datosGrafico.value).flat().map(d => d.valor))
  const rango = maxValor - minValor || 1
  
  let path = ''
  datos.forEach((punto, index) => {
    const x = (index / (datos.length - 1)) * 1200
    const y = 280 - (((punto.valor - minValor) / rango) * 260)
    
    if (index === 0) {
      path += `M ${x} ${y}`
    } else {
      path += ` L ${x} ${y}`
    }
  })
  
  return path
}

function formatearFecha(fecha) {
  const date = new Date(fecha)
  const dia = String(date.getDate()).padStart(2, '0')
  const mes = String(date.getMonth() + 1).padStart(2, '0')
  return `${dia}/${mes}`
}

const tooltipVisible = ref(false)
const tooltipData = ref({})
const tooltipStyle = ref({})

function mostrarTooltip(event, punto, tarifa) {
  tooltipData.value = { ...punto, tarifa }
  tooltipVisible.value = true
  
  const rect = event.target.getBoundingClientRect()
  tooltipStyle.value = {
    position: 'fixed',
    left: `${rect.left + rect.width / 2}px`,
    top: `${rect.top}px`,
    transform: 'translate(-50%, -100%)',
    pointerEvents: 'none',
    zIndex: 9999
  }
}

function ocultarTooltip() {
  tooltipVisible.value = false
}

const totales = computed(() => {
  if (datosConsumo.value.length === 0) return {}
  
  return {
    HP: datosConsumo.value.reduce((sum, d) => sum + d.HP, 0).toFixed(2),
    HFP: datosConsumo.value.reduce((sum, d) => sum + d.HFP, 0).toFixed(2),
    'HP PLUS': datosConsumo.value.reduce((sum, d) => sum + d['HP PLUS'], 0).toFixed(2),
    BASE: datosConsumo.value.reduce((sum, d) => sum + d.BASE, 0).toFixed(2),
    total: datosConsumo.value.reduce((sum, d) => sum + d.total, 0).toFixed(2)
  }
})

function exportarExcel() {
  if (datosConsumo.value.length === 0) return
  
  // Construir HTML del Excel basado en los datos actuales
  let html = `
    <html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">
    <head>
      <meta charset="UTF-8">
      <xml>
        <x:ExcelWorkbook>
          <x:ExcelWorksheets>
            <x:ExcelWorksheet>
              <x:Name>Consumo Eléctrico</x:Name>
              <x:WorksheetOptions><x:DisplayGridlines/></x:WorksheetOptions>
            </x:ExcelWorksheet>
          </x:ExcelWorksheets>
        </x:ExcelWorkbook>
      </xml>
    </head>
    <body>
      <table>
        <thead>
          <tr>
            <th>Fecha</th>
            <th>HP (kWh)</th>
            <th>HFP (kWh)</th>
            <th>HP PLUS (kWh)</th>
            <th>BASE (kWh)</th>
            <th>Total (kWh)</th>
          </tr>
        </thead>
        <tbody>
  `
  
  datosConsumo.value.forEach(dato => {
    html += `
      <tr>
        <td>${formatearFecha(dato.fecha)}</td>
        <td>${dato.HP.toFixed(2)}</td>
        <td>${dato.HFP.toFixed(2)}</td>
        <td>${dato['HP PLUS'].toFixed(2)}</td>
        <td>${dato.BASE.toFixed(2)}</td>
        <td>${dato.total.toFixed(2)}</td>
      </tr>
    `
  })
  
  html += `
          <tr style="font-weight: bold; background-color: #f0f9ff;">
            <td>TOTAL</td>
            <td>${totales.value.HP}</td>
            <td>${totales.value.HFP}</td>
            <td>${totales.value['HP PLUS']}</td>
            <td>${totales.value.BASE}</td>
            <td>${totales.value.total}</td>
          </tr>
        </tbody>
      </table>
    </body>
    </html>
  `
  
  const blob = new Blob([html], { type: 'application/vnd.ms-excel' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  
  // Nombre del archivo con fecha del reporte
  const compania = companias.value.find(c => c.id === companiaSeleccionada.value)?.nombre || 'Consumo'
  const fechaReporte = new Date().toISOString().split('T')[0]
  link.download = `${compania.replace(/\s+/g, '_')}_Consumo_Electrico_${fechaReporte}.xls`
  
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  
})
</script>

<template>
  <div class="consumo-container">
    <h1 class="titulo-principal">CONSUMO ELÉCTRICO TARIFARIO</h1>
    
    <div class="subtitulo-seccion">Seleccione la variable de Energía Real a monitorear:</div>

    <div class="formulario-grid">
      <!-- FECHAS -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">FECHAS</h3>
        <div class="campo-grupo">
          <label>Inicio</label>
          <input type="datetime-local" v-model="fechaInicioInput" />
        </div>
        <div class="campo-grupo">
          <label>Fin</label>
          <input type="datetime-local" v-model="fechaFinInput" />
        </div>
      </div>

      <!-- AGRUPAMIENTO -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">AGRUPAMIENTO</h3>
        <div class="campo-grupo">
          <label>Agrupamiento</label>
          <select v-model="agrupamientoSeleccionado">
            <option v-for="agrupamiento in agrupamientos" :key="agrupamiento.id" :value="agrupamiento.valor">
              {{ agrupamiento.nombre }}
            </option>
          </select>
        </div>
        <div class="campo-grupo">
          <label>Estructura Tarifaria</label>
          <select v-model="estructuraTarifariaSeleccionada">
            <option v-for="estructura in estructurasTarifarias" :key="estructura.id" :value="estructura.valor">
              {{ estructura.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- COMPAÑÍA/PLANTA/LÍNEA -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">Compañía</h3>
        <div class="campo-grupo">
          <select v-model="companiaSeleccionada">
            <option v-for="compania in companias" :key="compania.id" :value="compania.id">
              {{ compania.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Planta</h3>
        <div class="campo-grupo">
          <select v-model="plantaSeleccionada">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Línea</h3>
        <div class="campo-grupo">
          <select v-model="lineaSeleccionada">
            <option v-for="linea in lineas" :key="linea.id" :value="linea.id">
              {{ linea.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- LOCACIÓN/DISPOSITIVO/VARIABLE -->
      <div class="seccion-box">
        <h3 class="seccion-titulo">Locación</h3>
        <div class="campo-grupo">
          <select v-model="ubicacionSeleccionada">
            <option v-for="ubicacion in ubicaciones" :key="ubicacion.id" :value="ubicacion.id">
              {{ ubicacion.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Dispositivo</h3>
        <div class="campo-grupo">
          <select v-model="dispositivoSeleccionado">
            <option v-for="dispositivo in dispositivos" :key="dispositivo.id" :value="dispositivo.id">
              {{ dispositivo.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="seccion-titulo mt-3">Variable</h3>
        <div class="campo-grupo">
          <select v-model="variableSeleccionada">
            <option v-for="variable in variables" :key="variable.id" :value="variable.valor">
              {{ variable.nombre }}
            </option>
          </select>
        </div>
      </div>
    </div>

    <div class="buscar-container">
      <Button @click="buscar" variant="primary" class="btn-buscar">
        BUSCAR
      </Button>
    </div>

    <!-- RESULTADOS -->
    <div v-if="mostrarResultados && datosConsumo.length > 0">
      <!-- TARJETAS DE TOTALES -->
      <div class="totales-grid">
        <div class="total-card" v-for="tarifa in tarifas" :key="tarifa" :style="{ borderTopColor: coloresTarifas[tarifa] }">
          <div class="total-label">{{ tarifa }}</div>
          <div class="total-valor">{{ totales[tarifa] }} kWh</div>
        </div>
        <div class="total-card total-general">
          <div class="total-label">TOTAL</div>
          <div class="total-valor">{{ totales.total }} kWh</div>
        </div>
      </div>

      <!-- GRÁFICO -->
      <div class="grafico-container">
        <div class="grafico-header">
          <h3>Consumo Eléctrico por Tarifa</h3>
          <div class="leyenda-tarifas">
            <div v-for="tarifa in tarifas" :key="tarifa" class="leyenda-item">
              <span class="leyenda-color" :style="{ backgroundColor: coloresTarifas[tarifa] }"></span>
              <span>{{ tarifa }}</span>
            </div>
          </div>
        </div>
        
        <svg viewBox="0 0 1200 300" preserveAspectRatio="none" class="grafico-svg">
          <defs>
            <filter id="shadowConsumo">
              <feGaussianBlur in="SourceAlpha" stdDeviation="2"/>
              <feOffset dx="0" dy="2" result="offsetblur"/>
              <feComponentTransfer>
                <feFuncA type="linear" slope="0.3"/>
              </feComponentTransfer>
              <feMerge>
                <feMergeNode/>
                <feMergeNode in="SourceGraphic"/>
              </feMerge>
            </filter>
          </defs>

          <path 
            v-for="tarifa in tarifas"
            :key="tarifa"
            :d="generarPathTarifa(tarifa)" 
            fill="none" 
            :stroke="coloresTarifas[tarifa]"
            stroke-width="2.5"
            filter="url(#shadowConsumo)"
          />
          
          <template v-for="tarifa in tarifas" :key="tarifa">
            <circle 
              v-for="(punto, index) in datosGrafico[tarifa]" 
              :key="`${tarifa}-${index}`"
              v-show="index % Math.max(1, Math.floor(datosGrafico[tarifa].length / 20)) === 0"
              :cx="(index / (datosGrafico[tarifa].length - 1)) * 1200"
              :cy="280 - (((punto.valor - Math.min(...Object.values(datosGrafico).flat().map(d => d.valor))) / (Math.max(...Object.values(datosGrafico).flat().map(d => d.valor)) - Math.min(...Object.values(datosGrafico).flat().map(d => d.valor)))) * 260)"
              r="4"
              :fill="coloresTarifas[tarifa]"
              class="punto-interactivo"
              @mouseenter="mostrarTooltip($event, punto, tarifa)"
              @mouseleave="ocultarTooltip"
            />
          </template>
        </svg>

        <Teleport to="body">
          <div v-if="tooltipVisible" :style="tooltipStyle" class="tooltip-grafico">
            <div class="tooltip-contenido">
              <div class="tooltip-titulo">{{ tooltipData.tarifa }}</div>
              <div class="tooltip-fecha">{{ formatearFecha(tooltipData.fecha) }}</div>
              <div class="tooltip-valor">{{ tooltipData.valor ? tooltipData.valor.toFixed(2) : '0.00' }} kWh</div>
            </div>
          </div>
        </Teleport>
      </div>

      <!-- TABLA -->
      <div class="tabla-container">
        <div class="tabla-header">
          <h3>Detalle de Consumo Diario</h3>
          <Button @click="exportarExcel" variant="secondary" class="btn-exportar">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" style="margin-right: 0.5rem;">
              <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4m4-5l5 5m0 0l5-5m-5 5V3" stroke="currentColor" stroke-width="2"/>
            </svg>
            Exportar Excel
          </Button>
        </div>
        <div class="tabla-scroll">
          <table class="tabla-consumo">
            <thead>
              <tr>
                <th>Fecha</th>
                <th>HP (kWh)</th>
                <th>HFP (kWh)</th>
                <th>HP PLUS (kWh)</th>
                <th>BASE (kWh)</th>
                <th>Total (kWh)</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(dato, index) in datosConsumo" :key="index">
                <td>{{ formatearFecha(dato.fecha) }}</td>
                <td>{{ dato.HP.toFixed(2) }}</td>
                <td>{{ dato.HFP.toFixed(2) }}</td>
                <td>{{ dato['HP PLUS'].toFixed(2) }}</td>
                <td>{{ dato.BASE.toFixed(2) }}</td>
                <td class="total-row">{{ dato.total.toFixed(2) }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="totales-row">
                <td><strong>TOTAL</strong></td>
                <td><strong>{{ totales.HP }}</strong></td>
                <td><strong>{{ totales.HFP }}</strong></td>
                <td><strong>{{ totales['HP PLUS'] }}</strong></td>
                <td><strong>{{ totales.BASE }}</strong></td>
                <td><strong>{{ totales.total }}</strong></td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.consumo-container {
  padding: 1.5rem;
}

.titulo-principal {
  font-size: 1.6rem;
  font-weight: 700;
  color: white;
  background: #001f54;
  padding: 1rem 1.5rem;
  margin: -1.5rem -1.5rem 1.5rem -1.5rem;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.subtitulo-seccion {
  font-size: 1.1rem;
  font-weight: 600;
  color: white;
  background: #001f54;
  padding: 0.8rem 1.5rem;
  margin: -0.5rem -1.5rem 1.5rem -1.5rem;
}

.formulario-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.seccion-box {
  background-color: #f5f5f5;
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 1rem;
}

.seccion-titulo {
  font-size: 0.9rem;
  font-weight: 700;
  color: #001f54;
  margin-bottom: 0.6rem;
  text-transform: none;
  letter-spacing: 0;
}

.mt-3 {
  margin-top: 1rem;
}

.campo-grupo {
  margin-bottom: 0.6rem;
}

.campo-grupo:last-child {
  margin-bottom: 0;
}

.campo-grupo label {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  color: #374151;
  margin-bottom: 0.3rem;
}

.campo-grupo input,
.campo-grupo select {
  width: 100%;
  padding: 0.5rem 0.7rem;
  font-size: 0.875rem;
  border: 1px solid #ccc;
  border-radius: 4px;
  background-color: white;
  transition: all 0.2s;
}

.campo-grupo input:focus,
.campo-grupo select:focus {
  outline: none;
  border-color: #001f54;
  box-shadow: 0 0 0 2px rgba(0, 31, 84, 0.1);
}

.buscar-container {
  display: flex;
  justify-content: center;
  margin: 2rem 0;
}

.btn-buscar {
  padding: 0.7rem 4rem;
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: 1px;
  text-transform: uppercase;
  background: #4a4a4a;
  border: none;
  color: white;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-buscar:hover {
  background: #333;
}

.totales-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 180px), 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.total-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-top: 4px solid;
  border-radius: 8px;
  padding: 1.2rem;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.total-general {
  border-top-color: #001f54 !important;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
}

.total-label {
  font-size: 0.8rem;
  font-weight: 700;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 0.5rem;
}

.total-valor {
  font-size: 1.4rem;
  font-weight: 700;
  color: #001f54;
}

.grafico-container {
  margin: 2rem 0;
  background-color: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.grafico-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.grafico-header h3 {
  font-size: 1.1rem;
  font-weight: 700;
  color: #001f54;
}

.leyenda-tarifas {
  display: flex;
  gap: 1.5rem;
}

.leyenda-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: #374151;
}

.leyenda-color {
  width: 16px;
  height: 16px;
  border-radius: 3px;
}

.grafico-svg {
  width: 100%;
  height: 300px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: linear-gradient(to bottom, #f9fafb 0%, white 100%);
}

.punto-interactivo {
  cursor: pointer;
  transition: all 0.2s;
}

.punto-interactivo:hover {
  r: 6;
  filter: drop-shadow(0 0 4px currentColor);
}

.tooltip-grafico {
  animation: fadeIn 0.2s;
  margin-bottom: 10px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translate(-50%, -100%) scale(0.9);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -100%) scale(1);
  }
}

.tooltip-contenido {
  background: rgba(30, 41, 59, 0.95);
  backdrop-filter: blur(8px);
  color: white;
  padding: 0.6rem 0.9rem;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  white-space: nowrap;
}

.tooltip-titulo {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #94a3b8;
  margin-bottom: 0.2rem;
}

.tooltip-fecha {
  font-size: 0.75rem;
  color: #cbd5e1;
  margin-bottom: 0.3rem;
}

.tooltip-valor {
  font-size: 0.95rem;
  font-weight: 700;
  color: #60a5fa;
}

.tabla-container {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.tabla-header {
  background: #001f54;
  color: white;
  padding: 1rem 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.tabla-header h3 {
  font-size: 1rem;
  font-weight: 700;
  margin: 0;
}

.btn-exportar {
  display: flex;
  align-items: center;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  background: white;
  color: #001f54;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s;
}

.btn-exportar:hover {
  background: #f0f9ff;
  transform: translateY(-1px);
}

.tabla-scroll {
  overflow-x: auto;
  max-height: 500px;
  overflow-y: auto;
}

.tabla-consumo {
  width: 100%;
  border-collapse: collapse;
}

.tabla-consumo thead {
  background: #f9fafb;
  position: sticky;
  top: 0;
  z-index: 10;
}

.tabla-consumo th {
  padding: 0.8rem 1rem;
  text-align: left;
  font-size: 0.85rem;
  font-weight: 700;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
}

.tabla-consumo tbody tr {
  border-bottom: 1px solid #f3f4f6;
  transition: background-color 0.15s;
}

.tabla-consumo tbody tr:hover {
  background-color: #f9fafb;
}

.tabla-consumo td {
  padding: 0.7rem 1rem;
  font-size: 0.85rem;
  color: #374151;
}

.tabla-consumo .total-row {
  font-weight: 600;
  color: #001f54;
}

.tabla-consumo tfoot {
  background: #f0f9ff;
  position: sticky;
  bottom: 0;
  z-index: 10;
}

.tabla-consumo tfoot td {
  padding: 1rem;
  font-size: 0.9rem;
  border-top: 2px solid #001f54;
  color: #001f54;
}

.totales-row td {
  font-weight: 700 !important;
}

@media (max-width: 768px) {
  .formulario-grid {
    grid-template-columns: 1fr;
  }

  .totales-grid {
    grid-template-columns: 1fr 1fr;
  }

  .titulo-principal {
    font-size: 1.2rem;
  }

  .subtitulo-seccion {
    font-size: 0.95rem;
  }
}
</style>
