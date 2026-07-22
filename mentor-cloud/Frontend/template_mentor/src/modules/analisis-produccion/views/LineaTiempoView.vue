<template>
  <div class="linea-tiempo">
    <!-- Mensaje de selección -->
    <div class="mensaje-seleccion">
      <h3>Seleccione las variables:</h3>
    </div>

    <!-- Formulario de Filtros - 3 Columnas -->
    <div class="formulario-grid-nuevo">
      <!-- FECHAS -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">FECHAS</h3>
        <div class="campo-grupo-nuevo">
          <label>Inicio</label>
          <input 
            type="datetime-local" 
            v-model="fechaInicioInput"
            class="campo-entrada-nuevo"
          />
        </div>
        <div class="campo-grupo-nuevo">
          <label>Fin</label>
          <input 
            type="datetime-local" 
            v-model="fechaFinInput"
            class="campo-entrada-nuevo"
          />
        </div>
      </div>

      <!-- COMPAÑÍA / PLANTA / LÍNEA / LOCACIÓN -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">Compañía</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="companiaSeleccionada" class="campo-entrada-nuevo">
            <option v-for="comp in companias" :key="comp.id" :value="comp.id">
              {{ comp.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="subtitulo-seccion-nuevo">Planta</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="plantaSeleccionada" class="campo-entrada-nuevo">
            <option v-for="planta in plantasFiltradas" :key="planta.id" :value="planta.id">
              {{ planta.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="subtitulo-seccion-nuevo">Línea</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="lineaSeleccionada" class="campo-entrada-nuevo">
            <option v-for="linea in lineasFiltradas" :key="linea.id" :value="linea.id">
              {{ linea.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="subtitulo-seccion-nuevo">Locación</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="ubicacionSeleccionada" class="campo-entrada-nuevo">
            <option v-for="ubicacion in ubicaciones" :key="ubicacion.id" :value="ubicacion.id">
              {{ ubicacion.nombre }}
            </option>
          </select>
        </div>
      </div>

      <!-- DISPOSITIVO / VARIABLE con ADD -->
      <div class="seccion-box-nuevo">
        <h3 class="subtitulo-seccion-nuevo">Dispositivo</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="dispositivoSeleccionado" class="campo-entrada-nuevo">
            <option v-for="dispositivo in dispositivos" :key="dispositivo.id" :value="dispositivo.id">
              {{ dispositivo.nombre }}
            </option>
          </select>
        </div>
        
        <h3 class="subtitulo-seccion-nuevo">Variable</h3>
        <div class="campo-grupo-nuevo">
          <select v-model="variableSeleccionada" class="campo-entrada-nuevo">
            <option value="">Seleccione variable</option>
            <option v-for="variable in variables" :key="variable.id" :value="variable.id">
              {{ variable.nombre }}
            </option>
          </select>
        </div>
        
        <button @click="agregarVariable" class="btn-add-nuevo">ADD</button>
      </div>
    </div>

    <!-- Variables Agregadas -->
    <div v-if="variablesAgregadas.length > 0" class="variables-agregadas">
      <h4>Variables agregadas:</h4>
      <div class="lista-variables-agregadas">
        <div v-for="(varAgregada, index) in variablesAgregadas" :key="index" class="variable-tag-nuevo">
          <span>{{ varAgregada.dispositivo }} - {{ varAgregada.variable }}</span>
          <button @click="eliminarVariable(index)" class="btn-eliminar-nuevo">×</button>
        </div>
      </div>
    </div>

    <!-- Botón Buscar -->
    <div class="seccion-buscar">
      <button @click="buscar" class="btn-buscar-principal">
        BUSCAR
      </button>
    </div>

    <!-- Timeline Visual - Una fila por variable -->
    <div v-if="timelinesPorVariable.length > 0" class="contenedor-timeline">
      <div class="timeline-header">
        <h3>Timeline de Producción</h3>
        
        <!-- Sección de Filtros con Checkboxes -->
        <div class="seccion-filtros">
          <h4 class="titulo-filtros">Filtros de Estados</h4>
          <div class="filtros-grid">
            <label 
              v-for="filtro in filtrosDisponibles" 
              :key="filtro.id"
              class="filtro-checkbox"
            >
              <input 
                type="checkbox" 
                :value="filtro.id"
                v-model="filtrosActivos"
                @change="aplicarFiltros"
              />
              <span class="filtro-indicador" :style="{ backgroundColor: filtro.color }"></span>
              <span class="filtro-nombre">{{ filtro.nombre }}</span>
            </label>
          </div>
          
          <div class="acciones-filtros">
            <button @click="seleccionarTodosFiltros" class="btn-filtro-accion">
              Seleccionar Todos
            </button>
            <button @click="limpiarFiltros" class="btn-filtro-accion">
              Limpiar Filtros
            </button>
            <button @click="buscar" class="btn-buscar-timeline">
              <svg class="icono" viewBox="0 0 20 20" fill="currentColor" style="width: 16px; height: 16px;">
                <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
              </svg>
              APLICAR Y BUSCAR
            </button>
          </div>
        </div>
      </div>

      <!-- Timeline por cada variable -->
      <div class="timelines-container">
        <div v-for="(timeline, index) in timelinesPorVariable" :key="index" class="timeline-row">
          <div class="timeline-label">{{ timeline.variable }}</div>
          <div class="timeline-visual">
            <svg :viewBox="`0 0 ${anchoTimeline} 120`" preserveAspectRatio="none">
              <!-- Bloques de eventos -->
              <g v-for="(evento, eIndex) in timeline.eventos" :key="eIndex">
                <rect
                  :x="calcularPosicionX(evento.inicio)"
                  y="20"
                  :width="calcularAncho(evento.inicio, evento.fin)"
                  height="80"
                  :fill="evento.color"
                  :class="'timeline-bloque'"
                  @mouseenter="mostrarTooltipTimeline($event, evento, timeline.variable)"
                  @mouseleave="ocultarTooltipTimeline"
                />
                <!-- Etiqueta del evento si tiene espacio -->
                <text
                  v-if="calcularAncho(evento.inicio, evento.fin) > 100"
                  :x="calcularPosicionX(evento.inicio) + calcularAncho(evento.inicio, evento.fin) / 2"
                  y="65"
                  text-anchor="middle"
                  fill="white"
                  font-size="12"
                  font-weight="bold"
                  class="evento-texto"
                >
                  {{ evento.etiqueta }}
                </text>
              </g>
            </svg>
          </div>
        </div>
      </div>

      <!-- Eje de tiempo inferior -->
      <div class="timeline-eje">
        <div v-for="(marca, index) in marcasTiempo" :key="'marca-' + index" class="marca-tiempo">
          <div class="marca-linea"></div>
          <div class="marca-texto">{{ marca.texto }}</div>
        </div>
      </div>

      <!-- Resumen compacto por variable -->
      <div class="resumen-compacto">
        <div v-for="(resumen, index) in resumenPorVariable" :key="index" class="resumen-variable-row">
          <div class="resumen-nombre">{{ resumen.variable }}</div>
          <div class="resumen-barras">
            <div 
              v-for="estado in resumen.estados" 
              :key="estado.nombre"
              class="resumen-barra-item"
              :style="{ 
                width: estado.porcentaje + '%',
                background: estado.color
              }"
              :title="`${estado.nombre}: ${estado.porcentaje.toFixed(1)}%`"
            >
            </div>
          </div>
        </div>
      </div>

      <!-- Tabla de Eventos -->
      <div class="tabla-container">
        <div class="tabla-header">
          <h3>Detalle de Eventos</h3>
          <button @click="exportarExcel" class="btn-exportar">
            Exportar Excel
          </button>
        </div>
        <table class="tabla-datos">
          <thead>
            <tr>
              <th>Fecha/Hora Inicio</th>
              <th>Fecha/Hora Fin</th>
              <th>Duración (min)</th>
              <th>Estado</th>
              <th>Producto</th>
              <th>Cantidad Producida</th>
              <th>Motivo Parada</th>
              <th>Turno</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="evento in datosTimeline" :key="evento.id">
              <td>{{ formatearFechaHora(evento.inicio) }}</td>
              <td>{{ formatearFechaHora(evento.fin) }}</td>
              <td>{{ evento.duracion }}</td>
              <td>
                <span :class="['badge-estado', evento.estado]">
                  {{ evento.estado }}
                </span>
              </td>
              <td>{{ evento.producto }}</td>
              <td>{{ evento.cantidad.toLocaleString() }}</td>
              <td>{{ evento.motivoParada || '-' }}</td>
              <td>{{ evento.turno }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Cards de Resumen -->
      <div class="cards-resumen">
        <div class="card-metrica">
          <div class="card-icono produciendo">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
              <path d="M12 2L2 7l10 5 10-5-10-5z" stroke="currentColor" stroke-width="2"/>
              <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <div class="card-contenido">
            <div class="card-valor">{{ resumen.tiempoProduciendo }} min</div>
            <div class="card-label">Tiempo Produciendo</div>
          </div>
        </div>

        <div class="card-metrica">
          <div class="card-icono parada">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
              <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
              <rect x="8" y="8" width="8" height="8" fill="currentColor"/>
            </svg>
          </div>
          <div class="card-contenido">
            <div class="card-valor">{{ resumen.tiempoParada }} min</div>
            <div class="card-label">Tiempo Parada</div>
          </div>
        </div>

        <div class="card-metrica">
          <div class="card-icono microparada">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
              <path d="M12 8v4l3 3" stroke="currentColor" stroke-width="2"/>
              <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <div class="card-contenido">
            <div class="card-valor">{{ resumen.tiempoMicroparada }} min</div>
            <div class="card-label">Microparadas</div>
          </div>
        </div>

        <div class="card-metrica">
          <div class="card-icono produccion">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
              <path d="M20 7h-4V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v3H4a2 2 0 00-2 2v11a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2z" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <div class="card-contenido">
            <div class="card-valor">{{ resumen.totalProducido.toLocaleString() }}</div>
            <div class="card-label">Total Producido</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tooltip -->
    <Teleport to="body">
      <div
        v-if="tooltipVisible"
        class="tooltip-timeline"
        :style="{ left: tooltipX + 'px', top: tooltipY + 'px' }"
      >
        <div class="tooltip-titulo">{{ tooltipData.estado }}</div>
        <div class="tooltip-linea">
          <strong>Variable:</strong> {{ tooltipData.variableNombre }}
        </div>
        <div class="tooltip-linea">
          <strong>Inicio:</strong> {{ formatearFechaHora(tooltipData.inicio) }}
        </div>
        <div class="tooltip-linea">
          <strong>Fin:</strong> {{ formatearFechaHora(tooltipData.fin) }}
        </div>
        <div class="tooltip-linea">
          <strong>Duración:</strong> {{ calcularDuracionHoras(tooltipData.inicio, tooltipData.fin) }}
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

// Datos maestros
const companias = ref([
  { id: 1, nombre: 'Mentor Monitor S.A.' },
  { id: 2, nombre: 'Industrial Tech' }
])

const plantas = ref([
  { id: 1, nombre: 'Planta Norte', companiaId: 1 },
  { id: 2, nombre: 'Planta Sur', companiaId: 1 },
  { id: 3, nombre: 'Planta Central', companiaId: 2 }
])

const lineas = ref([
  { id: 1, nombre: 'Línea 1', plantaId: 1 },
  { id: 2, nombre: 'Línea 2', plantaId: 1 },
  { id: 3, nombre: 'Línea 3', plantaId: 2 }
])

const ubicaciones = ref([
  { id: 1, nombre: 'Linea 05 Locacion' },
  { id: 2, nombre: 'Zona A' },
  { id: 3, nombre: 'Zona B' }
])

const dispositivos = ref([
  { id: 1, nombre: 'Linea 05 Produccion' },
  { id: 2, nombre: 'Sensor Principal' },
  { id: 3, nombre: 'Medidor Central' }
])

const variables = ref([
  { id: 1, nombre: 'C_Unit_P (PLC-Ls-Conteo Unitario Principal)' },
  { id: 2, nombre: 'Temperatura' },
  { id: 3, nombre: 'Velocidad' },
  { id: 4, nombre: 'Presión' }
])

// Selecciones
const companiaSeleccionada = ref(1)
const plantaSeleccionada = ref(1)
const lineaSeleccionada = ref(1)
const ubicacionSeleccionada = ref(1)
const dispositivoSeleccionado = ref(1)
const variableSeleccionada = ref('')
const variablesAgregadas = ref([]) // Array de objetos con dispositivo y variable

// Fechas
const fechaInicio = ref('2025/11/19 00:00')
const fechaFin = ref('2025/11/19 23:59')

const fechaInicioInput = computed({
  get: () => fechaInicio.value.replace(' ', 'T').replace(/\//g, '-'),
  set: (val) => fechaInicio.value = val.replace('T', ' ').replace(/-/g, '/')
})

const fechaFinInput = computed({
  get: () => fechaFin.value.replace(' ', 'T').replace(/\//g, '-'),
  set: (val) => fechaFin.value = val.replace('T', ' ').replace(/-/g, '/')
})

// Filtros
const plantasFiltradas = computed(() => {
  return plantas.value.filter(p => p.companiaId === companiaSeleccionada.value)
})

const lineasFiltradas = computed(() => {
  return lineas.value.filter(l => l.plantaId === plantaSeleccionada.value)
})

// Datos del timeline
const datosTimeline = ref([])
const timelinesPorVariable = ref([])
const marcasTiempo = ref([])
const anchoTimeline = ref(1200)

// Filtros de estados
const filtrosDisponibles = ref([
  { id: 1, nombre: '360', color: '#3b82f6' },
  { id: 2, nombre: '350 ML', color: '#8b5cf6' },
  { id: 3, nombre: 'nacional', color: '#f97316' },
  { id: 4, nombre: '625 ml', color: '#78716c' },
  { id: 5, nombre: 'kola real', color: '#22c55e' },
  { id: 6, nombre: '300 ml', color: '#ec4899' },
  { id: 7, nombre: 'cielo', color: '#ef4444' }
])
const filtrosActivos = ref([])

// Tooltip
const tooltipVisible = ref(false)
const tooltipX = ref(0)
const tooltipY = ref(0)
const tooltipData = ref({})

// Computed para resumen
const resumenPorVariable = computed(() => {
  return timelinesPorVariable.value.map(timeline => {
    const totalDuracion = timeline.eventos.reduce((sum, e) => sum + (new Date(e.fin) - new Date(e.inicio)), 0)
    
    // Agrupar por estado y calcular porcentajes
    const estadosMap = {}
    timeline.eventos.forEach(evento => {
      const duracion = new Date(evento.fin) - new Date(evento.inicio)
      if (!estadosMap[evento.estado]) {
        estadosMap[evento.estado] = {
          nombre: evento.estado,
          duracion: 0,
          color: evento.color
        }
      }
      estadosMap[evento.estado].duracion += duracion
    })

    const estados = Object.values(estadosMap).map(estado => ({
      ...estado,
      porcentaje: (estado.duracion / totalDuracion) * 100
    })).sort((a, b) => b.porcentaje - a.porcentaje)

    return {
      variable: timeline.variable,
      estados: estados
    }
  })
})

// Resumen
const resumen = computed(() => {
  const tiempoProduciendo = datosTimeline.value
    .filter(e => e.estado === 'Produciendo')
    .reduce((sum, e) => sum + e.duracion, 0)
  
  const tiempoParada = datosTimeline.value
    .filter(e => e.estado === 'Parada')
    .reduce((sum, e) => sum + e.duracion, 0)
  
  const tiempoMicroparada = datosTimeline.value
    .filter(e => e.estado === 'Microparada')
    .reduce((sum, e) => sum + e.duracion, 0)
  
  const totalProducido = datosTimeline.value
    .reduce((sum, e) => sum + e.cantidad, 0)

  return {
    tiempoProduciendo,
    tiempoParada,
    tiempoMicroparada,
    totalProducido
  }
})

function agregarVariable() {
  if (!variableSeleccionada.value) {
    alert('Por favor seleccione una variable')
    return
  }

  const dispositivo = dispositivos.value.find(d => d.id === dispositivoSeleccionado.value)
  const variable = variables.value.find(v => v.id === variableSeleccionada.value)

  // Verificar si ya está agregada
  const yaExiste = variablesAgregadas.value.some(v => 
    v.dispositivoId === dispositivo.id && v.variableId === variable.id
  )

  if (yaExiste) {
    alert('Esta variable ya ha sido agregada')
    return
  }

  variablesAgregadas.value.push({
    dispositivoId: dispositivo.id,
    dispositivo: dispositivo.nombre,
    variableId: variable.id,
    variable: variable.nombre
  })

  // Limpiar selección
  variableSeleccionada.value = ''
}

function eliminarVariable(index) {
  variablesAgregadas.value.splice(index, 1)
  if (variablesAgregadas.value.length === 0) {
    timelinesPorVariable.value = []
    datosTimeline.value = []
  }
}

function buscar() {
  if (variablesAgregadas.value.length === 0) {
    alert('Por favor agregue al menos una variable')
    return
  }
  generarDatosTimeline()
}

function generarDatosTimeline() {
  if (variablesAgregadas.value.length === 0) {
    timelinesPorVariable.value = []
    datosTimeline.value = []
    return
  }

  const inicio = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T'))
  const fin = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T'))

  // Generar timeline para cada variable agregada
  const timelines = variablesAgregadas.value.map((varAgregada, varIndex) => {
    const eventos = []
    let horaActual = new Date(inicio)
    let eventoId = 1

    // Estados/valores posibles según el tipo de variable
    const valoresPosibles = [
      { etiqueta: '360', color: '#3b82f6' },
      { etiqueta: '350 ML', color: '#8b5cf6' },
      { etiqueta: 'nacional', color: '#f97316' },
      { etiqueta: '625 ml', color: '#78716c' },
      { etiqueta: 'kola real', color: '#22c55e' },
      { etiqueta: '300 ml', color: '#ec4899' },
      { etiqueta: 'cielo', color: '#ef4444' }
    ]

    // Generar eventos para esta variable
    while (horaActual < fin) {
      // Duración aleatoria entre 30 min y 4 horas
      const duracionHoras = Math.random() * 3.5 + 0.5
      const horaFinEvento = new Date(horaActual.getTime() + duracionHoras * 60 * 60000)
      
      // Seleccionar un valor aleatorio
      const valor = valoresPosibles[Math.floor(Math.random() * valoresPosibles.length)]

      eventos.push({
        id: `${varIndex}-${eventoId}`,
        inicio: new Date(horaActual),
        fin: horaFinEvento < fin ? horaFinEvento : fin,
        estado: valor.etiqueta,
        etiqueta: valor.etiqueta,
        color: valor.color,
        variable: varAgregada.variable,
        dispositivo: varAgregada.dispositivo
      })

      eventoId++
      horaActual = horaFinEvento
      if (horaActual >= fin) break
    }

    return {
      variable: varAgregada.variable,
      dispositivo: varAgregada.dispositivo,
      eventos: eventos
    }
  })

  timelinesPorVariable.value = timelines
  
  // Generar datos para la tabla (todos los eventos de todas las variables)
  const todosEventos = []
  timelines.forEach(timeline => {
    timeline.eventos.forEach(evento => {
      const duracionMs = new Date(evento.fin) - new Date(evento.inicio)
      const duracionMin = Math.round(duracionMs / (1000 * 60))
      
      todosEventos.push({
        id: evento.id,
        inicio: evento.inicio,
        fin: evento.fin,
        duracion: duracionMin,
        estado: evento.estado,
        variable: evento.variable,
        dispositivo: evento.dispositivo,
        producto: evento.etiqueta,
        cantidad: Math.floor(Math.random() * 1000) + 500,
        motivoParada: '',
        turno: duracionMin < 120 ? 'Mañana' : 'Tarde'
      })
    })
  })

  datosTimeline.value = todosEventos
  generarMarcasTiempo()
}

function calcularPosicionX(fecha) {
  const inicio = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T'))
  const fin = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T'))
  const fechaEvento = new Date(fecha)
  
  const totalDuracion = fin - inicio
  const duracionEvento = fechaEvento - inicio
  
  return (duracionEvento / totalDuracion) * anchoTimeline.value
}

function calcularAncho(fechaInicioEvento, fechaFinEvento) {
  const inicioEvento = new Date(fechaInicioEvento)
  const finEvento = new Date(fechaFinEvento)
  const inicioTotal = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T'))
  const finTotal = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T'))
  
  const totalDuracion = finTotal - inicioTotal
  const duracionEvento = finEvento - inicioEvento
  
  return (duracionEvento / totalDuracion) * anchoTimeline.value
}

function seleccionarTodosFiltros() {
  filtrosActivos.value = filtrosDisponibles.value.map(f => f.id)
  aplicarFiltros()
}

function limpiarFiltros() {
  filtrosActivos.value = []
  aplicarFiltros()
}

function aplicarFiltros() {
  if (filtrosActivos.value.length === 0) {
    // Mostrar todos
    generarDatosTimeline()
    return
  }

  // Filtrar eventos según filtros activos
  const nombresActivos = filtrosActivos.value.map(id => 
    filtrosDisponibles.value.find(f => f.id === id)?.nombre
  )

  timelinesPorVariable.value = timelinesPorVariable.value.map(timeline => ({
    ...timeline,
    eventos: timeline.eventos.filter(evento => 
      nombresActivos.includes(evento.estado)
    )
  }))
}

function generarMarcasTiempo() {
  const inicio = new Date(fechaInicio.value.replace(/\//g, '-').replace(' ', 'T'))
  const fin = new Date(fechaFin.value.replace(/\//g, '-').replace(' ', 'T'))
  const totalDias = Math.ceil((fin - inicio) / (1000 * 60 * 60 * 24))
  
  // Generar marcas cada 12 horas aproximadamente
  const numMarcas = Math.min(totalDias * 2 + 1, 20)
  const intervalo = (fin - inicio) / (numMarcas - 1)

  marcasTiempo.value = Array.from({ length: numMarcas }, (_, i) => {
    const fecha = new Date(inicio.getTime() + intervalo * i)
    return {
      texto: fecha.toLocaleDateString('es-ES', { 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      }),
      posicion: (i / (numMarcas - 1)) * 100
    }
  })
}

function getColorEstado(estado) {
  const colores = {
    'Produciendo': '#22c55e',
    'Parada': '#ef4444',
    'Microparada': '#f59e0b',
    'Sin Datos': '#6b7280'
  }
  return colores[estado] || '#6b7280'
}

function mostrarTooltipTimeline(event, evento, variableNombre) {
  tooltipData.value = {
    ...evento,
    variableNombre: variableNombre
  }
  tooltipX.value = event.clientX + 10
  tooltipY.value = event.clientY + 10
  tooltipVisible.value = true
}

function ocultarTooltipTimeline() {
  tooltipVisible.value = false
}

function formatearFechaHora(fecha) {
  if (!fecha) return '-'
  return new Date(fecha).toLocaleString('es-ES', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function calcularDuracionHoras(inicio, fin) {
  if (!inicio || !fin) return '-'
  const duracionMs = new Date(fin) - new Date(inicio)
  const horas = Math.floor(duracionMs / (1000 * 60 * 60))
  const minutos = Math.floor((duracionMs % (1000 * 60 * 60)) / (1000 * 60))
  
  if (horas > 0) {
    return `${horas}h ${minutos}min`
  }
  return `${minutos}min`
}

function exportarExcel() {
  if (datosTimeline.value.length === 0) {
    alert('No hay datos para exportar')
    return
  }

  const compania = companias.value.find(c => c.id === companiaSeleccionada.value)?.nombre || 'Empresa'
  const fechaReporte = new Date().toLocaleDateString('es-ES').replace(/\//g, '-')

  // Crear contenido Excel con XML
  let excelContent = `
<?xml version="1.0"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
  xmlns:o="urn:schemas-microsoft-com:office:office"
  xmlns:x="urn:schemas-microsoft-com:office:excel"
  xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"
  xmlns:html="http://www.w3.org/TR/REC-html40">
  <Styles>
    <Style ss:ID="header">
      <Font ss:Bold="1" ss:Color="#FFFFFF"/>
      <Interior ss:Color="#001f54" ss:Pattern="Solid"/>
      <Alignment ss:Horizontal="Center" ss:Vertical="Center"/>
    </Style>
  </Styles>
  <Worksheet ss:Name="Línea de Tiempo">
    <Table>
      <Row>
        <Cell ss:StyleID="header"><Data ss:Type="String">Fecha/Hora Inicio</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Fecha/Hora Fin</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Duración (min)</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Estado</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Variable</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Dispositivo</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Producto</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Cantidad</Data></Cell>
        <Cell ss:StyleID="header"><Data ss:Type="String">Turno</Data></Cell>
      </Row>`

  datosTimeline.value.forEach(evento => {
    excelContent += `
      <Row>
        <Cell><Data ss:Type="String">${formatearFechaHora(evento.inicio)}</Data></Cell>
        <Cell><Data ss:Type="String">${formatearFechaHora(evento.fin)}</Data></Cell>
        <Cell><Data ss:Type="Number">${evento.duracion}</Data></Cell>
        <Cell><Data ss:Type="String">${evento.estado}</Data></Cell>
        <Cell><Data ss:Type="String">${evento.variable || '-'}</Data></Cell>
        <Cell><Data ss:Type="String">${evento.dispositivo || '-'}</Data></Cell>
        <Cell><Data ss:Type="String">${evento.producto}</Data></Cell>
        <Cell><Data ss:Type="Number">${evento.cantidad}</Data></Cell>
        <Cell><Data ss:Type="String">${evento.turno}</Data></Cell>
      </Row>`
  })

  excelContent += `
    </Table>
  </Worksheet>
</Workbook>`

  // Crear y descargar el archivo
  const blob = new Blob([excelContent], { type: 'application/vnd.ms-excel' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${compania.replace(/\s+/g, '_')}_Linea_Tiempo_${fechaReporte}.xls`
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

onMounted(() => {
  // No generar datos automáticamente, esperar a que el usuario agregue variables
})
</script>

<style scoped>
.linea-tiempo {
  padding: 1.5rem;
  background: #f5f7fa;
  min-height: 100vh;
}

/* Mensaje de selección */
.mensaje-seleccion {
  background: white;
  border-left: 4px solid #001f54;
  padding: 1rem 1.5rem;
  margin-bottom: 1.5rem;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.mensaje-seleccion h3 {
  margin: 0;
  color: #001f54;
  font-size: 1.125rem;
  font-weight: 600;
}

/* Formulario Grid - 3 Columnas */
.formulario-grid-nuevo {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

.seccion-box-nuevo {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.subtitulo-seccion-nuevo {
  background: #001f54;
  color: white;
  padding: 0.625rem;
  text-align: center;
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0 0 1rem 0;
  border-radius: 4px;
  letter-spacing: 0.5px;
}

.campo-grupo-nuevo {
  margin-bottom: 1rem;
}

.campo-grupo-nuevo:last-child {
  margin-bottom: 0;
}

.campo-grupo-nuevo label {
  display: block;
  margin-bottom: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: #475569;
}

.campo-entrada-nuevo {
  width: 100%;
  padding: 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 4px;
  font-size: 0.875rem;
  transition: all 0.2s;
  background: white;
}

.campo-entrada-nuevo:focus {
  outline: none;
  border-color: #001f54;
  box-shadow: 0 0 0 3px rgba(0, 31, 84, 0.1);
}

/* Botón ADD */
.btn-add-nuevo {
  width: 100%;
  padding: 0.75rem;
  background: white;
  color: #001f54;
  border: 2px solid #001f54;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 1rem;
  transition: all 0.2s;
  font-size: 0.875rem;
}

.btn-add-nuevo:hover {
  background: #001f54;
  color: white;
}

/* Variables Agregadas */
.variables-agregadas {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1.25rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.variables-agregadas h4 {
  margin: 0 0 1rem 0;
  color: #475569;
  font-size: 0.9375rem;
  font-weight: 600;
}

.lista-variables-agregadas {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.variable-tag-nuevo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: #e0f2fe;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  border: 1px solid #bae6fd;
  font-size: 0.875rem;
  color: #0c4a6e;
}

.variable-tag-nuevo span {
  font-weight: 500;
}

.btn-eliminar-nuevo {
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 50%;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 1.125rem;
  font-weight: bold;
  line-height: 1;
  transition: background 0.2s;
  flex-shrink: 0;
}

.btn-eliminar-nuevo:hover {
  background: #dc2626;
}

/* Variables con Checkboxes - Ya no se usa pero mantengo por si acaso */
.variables-checkbox-container {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  padding: 0.5rem;
  background: #f8fafc;
}

.variable-checkbox-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  cursor: pointer;
  transition: background 0.2s;
  border-radius: 4px;
}

.variable-checkbox-item:hover {
  background: #e0f2fe;
}

.variable-checkbox-item input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: #001f54;
  flex-shrink: 0;
}

.variable-checkbox-item span {
  font-size: 0.8125rem;
  color: #334155;
  line-height: 1.3;
}

/* Botón Buscar */
.seccion-buscar {
  text-align: center;
  margin-bottom: 2rem;
}

.btn-buscar-principal {
  padding: 0.875rem 3rem;
  background: #001f54;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: 0 2px 4px rgba(0, 31, 84, 0.3);
  letter-spacing: 1px;
}

.btn-buscar-principal:hover {
  background: #003087;
  transform: translateY(-1px);
  box-shadow: 0 4px 6px rgba(0, 31, 84, 0.4);
}

/* Timeline Section */
.contenedor-timeline {
  margin-top: 2rem;
}

.timeline-header {
  margin-bottom: 1rem;
  padding: 1.5rem;
  background: white;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.timeline-header h3 {
  font-size: 1.25rem;
  color: #001f54;
  margin: 0 0 1.5rem 0;
}

/* Sección de Filtros */
.seccion-filtros {
  margin-top: 1rem;
}

.titulo-filtros {
  font-size: 1rem;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 1rem;
}

.filtros-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.filtro-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  border: 2px solid #e2e8f0;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  background: #f8fafc;
}

.filtro-checkbox:hover {
  border-color: #cbd5e1;
  background: white;
}

.filtro-checkbox input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
  accent-color: #001f54;
}

.filtro-indicador {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  border: 2px solid rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.filtro-nombre {
  font-size: 0.875rem;
  font-weight: 500;
  color: #475569;
  flex: 1;
}

.filtro-checkbox:has(input:checked) {
  border-color: #001f54;
  background: #e0f2fe;
}

.filtro-checkbox:has(input:checked) .filtro-nombre {
  color: #001f54;
  font-weight: 600;
}

.acciones-filtros {
  display: flex;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid #e2e8f0;
}

.btn-filtro-accion {
  padding: 0.625rem 1.25rem;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  background: white;
  color: #475569;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-filtro-accion:hover {
  background: #f1f5f9;
  border-color: #94a3b8;
}

.btn-buscar-timeline {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.5rem;
  background: #001f54;
  color: white;
  border: none;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.2s;
  margin-left: auto;
}

.btn-buscar-timeline:hover {
  background: #003087;
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 31, 84, 0.3);
}

.timelines-container {
  background: white;
  padding: 1.5rem 0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 1rem;
}

.timeline-row {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 1rem;
  margin-bottom: 0.5rem;
  align-items: center;
  border-bottom: 1px solid #e5e7eb;
  padding: 0.5rem 1.5rem;
}

.timeline-row:last-child {
  border-bottom: none;
}

.timeline-label {
  font-size: 0.9rem;
  font-weight: 500;
  color: #374151;
  writing-mode: horizontal-tb;
  text-align: left;
  padding: 0.5rem;
  background: #f9fafb;
  border-radius: 4px;
}

.timeline-visual {
  position: relative;
  width: 100%;
  height: 120px;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  background: white;
  overflow: hidden;
}

.timeline-visual svg {
  width: 100%;
  height: 100%;
}

.timeline-bloque {
  cursor: pointer;
  transition: opacity 0.2s;
  stroke: white;
  stroke-width: 1;
}

.timeline-bloque:hover {
  opacity: 0.8;
}

.evento-texto {
  pointer-events: none;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.3);
}

.timeline-eje {
  display: flex;
  justify-content: space-between;
  padding: 0 1.5rem;
  margin-bottom: 1rem;
  position: relative;
}

.marca-tiempo {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.marca-linea {
  width: 1px;
  height: 10px;
  background: #9ca3af;
  margin-bottom: 0.25rem;
}

.marca-texto {
  font-size: 0.75rem;
  color: #6b7280;
  white-space: nowrap;
}

.resumen-compacto {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.resumen-variable-row {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 1rem;
  align-items: center;
  margin-bottom: 0.75rem;
}

.resumen-variable-row:last-child {
  margin-bottom: 0;
}

.resumen-nombre {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.resumen-barras {
  display: flex;
  height: 24px;
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid #e5e7eb;
}

.resumen-barra-item {
  transition: opacity 0.2s;
  cursor: pointer;
}

.resumen-barra-item:hover {
  opacity: 0.8;
}

.tabla-container {
  background: white;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  margin-bottom: 2rem;
}

.tabla-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: #001f54;
  color: white;
}

.tabla-header h3 {
  margin: 0;
  font-size: 1.1rem;
}

.btn-exportar {
  padding: 0.5rem 1.5rem;
  background: white;
  color: #001f54;
  border: none;
  border-radius: 4px;
  font-weight: 600;
  cursor: pointer;
}

.btn-exportar:hover {
  background: #f3f4f6;
}

.tabla-datos {
  width: 100%;
  border-collapse: collapse;
}

.tabla-datos thead {
  background: #f9fafb;
  position: sticky;
  top: 0;
}

.tabla-datos th {
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  color: #374151;
  border-bottom: 2px solid #e5e7eb;
}

.tabla-datos td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e5e7eb;
  color: #6b7280;
}

.tabla-datos tbody tr:hover {
  background: #f9fafb;
}

.badge-estado {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge-estado.Produciendo {
  background: #dcfce7;
  color: #166534;
}

.badge-estado.Parada {
  background: #fee2e2;
  color: #991b1b;
}

.badge-estado.Microparada {
  background: #fef3c7;
  color: #92400e;
}

.cards-resumen {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
  gap: 1.5rem;
}

.card-metrica {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 1rem;
}

.card-icono {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-icono.produciendo {
  background: #dcfce7;
  color: #166534;
}

.card-icono.parada {
  background: #fee2e2;
  color: #991b1b;
}

.card-icono.microparada {
  background: #fef3c7;
  color: #92400e;
}

.card-icono.produccion {
  background: #dbeafe;
  color: #1e40af;
}

.card-contenido {
  flex: 1;
}

.card-valor {
  font-size: 1.5rem;
  font-weight: bold;
  color: #001f54;
  margin-bottom: 0.25rem;
}

.card-label {
  font-size: 0.875rem;
  color: #6b7280;
}

.tooltip-timeline {
  position: fixed;
  background: rgba(0, 0, 0, 0.9);
  color: white;
  padding: 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  pointer-events: none;
  z-index: 1000;
  backdrop-filter: blur(10px);
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  max-width: 300px;
}

.tooltip-titulo {
  font-weight: bold;
  margin-bottom: 0.5rem;
  font-size: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  padding-bottom: 0.5rem;
}

.tooltip-linea {
  margin: 0.25rem 0;
}

@media (max-width: 1024px) {
  .formulario-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .cards-resumen {
    grid-template-columns: 1fr;
  }
}
</style>
