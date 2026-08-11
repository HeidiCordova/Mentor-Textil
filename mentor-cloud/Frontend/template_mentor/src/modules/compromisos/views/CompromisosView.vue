<template>
  <div class="compromisos-view">
    <div class="page-header">
      <h1>Compromisos de Producción</h1>
      <p>Gestión y seguimiento de compromisos y objetivos de producción</p>
    </div>

    <!-- Filtros y Acciones -->
    <div class="filtros-section">
      <div class="filtros-container">
        <select v-model="filtros.compania" class="form-control">
          <option value="">Todas las Compañías</option>
          <option value="1">Art Atlas</option>
          <option value="2">Indagua</option>
          <option value="3">Demo</option>
          <option value="4">Agrícola Cerro Prieto</option>
        </select>

        <select v-model="filtros.planta" class="form-control">
          <option value="">Todas las Plantas</option>
          <option value="1">Planta Textil Lima</option>
          <option value="2">Planta Textil Norte</option>
          <option value="3">Indagua Principal</option>
          <option value="4">ACP Ica</option>
        </select>

        <select v-model="filtros.periodo" class="form-control">
          <option value="mes">Este Mes</option>
          <option value="trimestre">Este Trimestre</option>
          <option value="semestre">Este Semestre</option>
          <option value="anio">Este Año</option>
        </select>

        <select v-model="filtros.estado" class="form-control">
          <option value="">Todos los Estados</option>
          <option value="cumplido">Cumplido</option>
          <option value="en-progreso">En Progreso</option>
          <option value="en-riesgo">En Riesgo</option>
          <option value="no-cumplido">No Cumplido</option>
        </select>
      </div>

      <button @click="mostrarNuevoCompromiso = true" class="btn btn-primary">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 5v14M5 12h14"/>
        </svg>
        Nuevo Compromiso
      </button>
    </div>

    <!-- Resumen de Compromisos -->
    <div class="resumen-grid">
      <div class="resumen-card cumplido">
        <div class="resumen-header">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 11-5.93-9.14"/>
            <path d="M22 4L12 14.01l-3-3"/>
          </svg>
          <h3>Cumplidos</h3>
        </div>
        <p class="resumen-numero">24</p>
        <p class="resumen-detalle">80% del total</p>
      </div>

      <div class="resumen-card en-progreso">
        <div class="resumen-header">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
          <h3>En Progreso</h3>
        </div>
        <p class="resumen-numero">4</p>
        <p class="resumen-detalle">13.3% del total</p>
      </div>

      <div class="resumen-card en-riesgo">
        <div class="resumen-header">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
            <line x1="12" y1="9" x2="12" y2="13"/>
            <line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          <h3>En Riesgo</h3>
        </div>
        <p class="resumen-numero">1</p>
        <p class="resumen-detalle">3.3% del total</p>
      </div>

      <div class="resumen-card no-cumplido">
        <div class="resumen-header">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
          <h3>No Cumplidos</h3>
        </div>
        <p class="resumen-numero">1</p>
        <p class="resumen-detalle">3.3% del total</p>
      </div>
    </div>

    <!-- Lista de Compromisos -->
    <div class="compromisos-lista">
      <h2>Compromisos Activos</h2>
      <div class="compromisos-container">
        <div v-for="compromiso in compromisos" :key="compromiso.id" class="compromiso-card" :class="`estado-${compromiso.estado}`">
          <div class="compromiso-header">
            <div class="compromiso-titulo">
              <h3>{{ compromiso.titulo }}</h3>
              <p>{{ compromiso.planta }} - {{ compromiso.linea }}</p>
            </div>
            <span :class="['badge-estado', `estado-${compromiso.estado}`]">{{ compromiso.estadoTexto }}</span>
          </div>

          <div class="compromiso-body">
            <div class="compromiso-meta">
              <div class="meta-item">
                <span class="meta-label">Objetivo:</span>
                <span class="meta-valor">{{ compromiso.objetivo }}</span>
              </div>
              <div class="meta-item">
                <span class="meta-label">Actual:</span>
                <span class="meta-valor destacado">{{ compromiso.actual }}</span>
              </div>
              <div class="meta-item">
                <span class="meta-label">Periodo:</span>
                <span class="meta-valor">{{ compromiso.periodo }}</span>
              </div>
            </div>

            <div class="progreso-container">
              <div class="progreso-header">
                <span>Progreso</span>
                <span class="progreso-porcentaje">{{ compromiso.progreso }}%</span>
              </div>
              <div class="progreso-barra">
                <div class="progreso-fill" :class="`fill-${compromiso.estado}`" :style="{ width: `${compromiso.progreso}%` }"></div>
              </div>
            </div>

            <div class="compromiso-info">
              <div class="info-item">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
                <span>{{ compromiso.responsable }}</span>
              </div>
              <div class="info-item">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                  <line x1="16" y1="2" x2="16" y2="6"/>
                  <line x1="8" y1="2" x2="8" y2="6"/>
                  <line x1="3" y1="10" x2="21" y2="10"/>
                </svg>
                <span>{{ compromiso.fechaVencimiento }}</span>
              </div>
            </div>
          </div>

          <div class="compromiso-footer">
            <button @click="verDetalle(compromiso.id)" class="btn-small btn-secondary">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
              Ver Detalle
            </button>
            <button @click="editarCompromiso(compromiso.id)" class="btn-small btn-primary">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
              Editar
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Nuevo Compromiso -->
    <Teleport to="body">
      <div v-if="mostrarNuevoCompromiso" class="modal-overlay" @click="mostrarNuevoCompromiso = false">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>Nuevo Compromiso</h3>
            <button @click="mostrarNuevoCompromiso = false" class="btn-close">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="form-grid">
              <div class="form-group full-width">
                <label>Título del Compromiso</label>
                <input type="text" v-model="nuevoCompromiso.titulo" class="form-control" placeholder="Ej: Aumentar OEE al 85%">
              </div>

              <div class="form-group">
                <label>Tipo de Meta</label>
                <select v-model="nuevoCompromiso.tipo" class="form-control">
                  <option value="">Seleccionar...</option>
                  <option value="oee">OEE</option>
                  <option value="produccion">Producción</option>
                  <option value="disponibilidad">Disponibilidad</option>
                  <option value="rendimiento">Rendimiento</option>
                  <option value="calidad">Calidad</option>
                  <option value="defectos">Reducción de Defectos</option>
                  <option value="paradas">Reducción de Paradas</option>
                </select>
              </div>

              <div class="form-group">
                <label>Valor Objetivo</label>
                <input type="number" v-model="nuevoCompromiso.objetivo" class="form-control" placeholder="85">
              </div>

              <div class="form-group">
                <label>Compañía</label>
                <select v-model="nuevoCompromiso.compania" class="form-control">
                  <option value="">Seleccionar...</option>
                  <option value="1">Art Atlas</option>
                  <option value="2">Indagua</option>
                  <option value="3">Demo</option>
                  <option value="4">Agrícola Cerro Prieto</option>
                </select>
              </div>

              <div class="form-group">
                <label>Planta</label>
                <select v-model="nuevoCompromiso.planta" class="form-control">
                  <option value="">Seleccionar...</option>
                  <option value="1">Planta Textil Lima</option>
                  <option value="2">Planta Textil Norte</option>
                  <option value="3">Indagua Principal</option>
                  <option value="4">ACP Ica</option>
                </select>
              </div>

              <div class="form-group">
                <label>Línea</label>
                <select v-model="nuevoCompromiso.linea" class="form-control">
                  <option value="">Todas</option>
                  <option value="1">Línea 01</option>
                  <option value="2">Línea 02</option>
                  <option value="3">Línea 03</option>
                  <option value="4">Línea 04</option>
                  <option value="5">Línea 05</option>
                  <option value="6">Línea 06</option>
                </select>
              </div>

              <div class="form-group">
                <label>Periodo</label>
                <select v-model="nuevoCompromiso.periodo" class="form-control">
                  <option value="mensual">Mensual</option>
                  <option value="trimestral">Trimestral</option>
                  <option value="semestral">Semestral</option>
                  <option value="anual">Anual</option>
                </select>
              </div>

              <div class="form-group">
                <label>Fecha de Vencimiento</label>
                <input type="date" v-model="nuevoCompromiso.fechaVencimiento" class="form-control">
              </div>

              <div class="form-group full-width">
                <label>Responsable</label>
                <input type="text" v-model="nuevoCompromiso.responsable" class="form-control" placeholder="Nombre del responsable">
              </div>

              <div class="form-group full-width">
                <label>Descripción / Observaciones</label>
                <textarea v-model="nuevoCompromiso.descripcion" class="form-control" rows="3" placeholder="Detalles adicionales del compromiso..."></textarea>
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button @click="mostrarNuevoCompromiso = false" class="btn btn-secondary">Cancelar</button>
            <button @click="crearCompromiso" class="btn btn-primary">Crear Compromiso</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const mostrarNuevoCompromiso = ref(false)

const filtros = ref({
  compania: '',
  planta: '',
  periodo: 'mes',
  estado: ''
})

const nuevoCompromiso = ref({
  titulo: '',
  tipo: '',
  objetivo: '',
  compania: '',
  planta: '',
  linea: '',
  periodo: 'mensual',
  fechaVencimiento: '',
  responsable: '',
  descripcion: ''
})

const compromisos = ref([
  {
    id: 1,
    titulo: 'OEE Mayor a 85%',
    planta: 'Planta Textil Lima',
    linea: 'Línea 01',
    objetivo: '85%',
    actual: '87%',
    periodo: 'Noviembre 2025',
    progreso: 102,
    estado: 'cumplido',
    estadoTexto: 'Cumplido',
    responsable: 'Juan Pérez',
    fechaVencimiento: '30/11/2025'
  },
  {
    id: 2,
    titulo: 'Producción 500,000 unidades',
    planta: 'Planta Textil Norte',
    linea: 'Línea 03',
    objetivo: '500,000',
    actual: '425,000',
    periodo: 'Noviembre 2025',
    progreso: 85,
    estado: 'en-progreso',
    estadoTexto: 'En Progreso',
    responsable: 'María García',
    fechaVencimiento: '30/11/2025'
  },
  {
    id: 3,
    titulo: 'Reducir Defectos a < 50',
    planta: 'Indagua Principal',
    linea: 'Todas',
    objetivo: '< 50',
    actual: '52',
    periodo: 'Noviembre 2025',
    progreso: 96,
    estado: 'en-riesgo',
    estadoTexto: 'En Riesgo',
    responsable: 'Carlos López',
    fechaVencimiento: '30/11/2025'
  },
  {
    id: 4,
    titulo: 'Disponibilidad > 90%',
    planta: 'Planta Textil Lima',
    linea: 'Línea 02',
    objetivo: '90%',
    actual: '92%',
    periodo: 'Noviembre 2025',
    progreso: 102,
    estado: 'cumplido',
    estadoTexto: 'Cumplido',
    responsable: 'Ana Martínez',
    fechaVencimiento: '30/11/2025'
  },
  {
    id: 5,
    titulo: 'Calidad 99%',
    planta: 'ACP Ica',
    linea: 'Línea 05',
    objetivo: '99%',
    actual: '98.5%',
    periodo: 'Noviembre 2025',
    progreso: 99,
    estado: 'en-progreso',
    estadoTexto: 'En Progreso',
    responsable: 'Luis Ramírez',
    fechaVencimiento: '30/11/2025'
  },
  {
    id: 6,
    titulo: 'Reducir Paradas a < 30 min',
    planta: 'Planta Textil Norte',
    linea: 'Línea 04',
    objetivo: '< 30 min',
    actual: '45 min',
    periodo: 'Noviembre 2025',
    progreso: 67,
    estado: 'no-cumplido',
    estadoTexto: 'No Cumplido',
    responsable: 'Pedro Sánchez',
    fechaVencimiento: '30/11/2025'
  }
])

function crearCompromiso() {
  // TODO: Conectar con backend
  console.log('Creando compromiso:', nuevoCompromiso.value)
  alert('Función lista para conectar con backend')
  mostrarNuevoCompromiso.value = false
}

function verDetalle(id) {
  // TODO: Conectar con backend
  console.log('Ver detalle compromiso:', id)
  alert('Función lista para conectar con backend')
}

function editarCompromiso(id) {
  // TODO: Conectar con backend
  console.log('Editar compromiso:', id)
  alert('Función lista para conectar con backend')
}
</script>

<style scoped>
.compromisos-view {
  padding: 1.5rem;
}

.page-header {
  margin-bottom: 2rem;
}

.page-header h1 {
  font-size: 2rem;
  font-weight: 800;
  color: #1e293b;
  margin-bottom: 0.5rem;
}

.page-header p {
  color: #64748b;
  font-size: 1rem;
}

.filtros-section {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
  display: flex;
  gap: 1rem;
  align-items: center;
}

.filtros-container {
  display: flex;
  gap: 1rem;
  flex: 1;
  flex-wrap: wrap;
}

.form-control {
  padding: 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  font-size: 0.875rem;
  transition: all 0.2s;
  min-width: 200px;
}

.form-control:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.resumen-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 250px), 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.resumen-card {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  border-left: 4px solid;
}

.resumen-card.cumplido {
  border-color: #22c55e;
}

.resumen-card.en-progreso {
  border-color: #3b82f6;
}

.resumen-card.en-riesgo {
  border-color: #f59e0b;
}

.resumen-card.no-cumplido {
  border-color: #ef4444;
}

.resumen-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.resumen-card.cumplido svg {
  color: #22c55e;
}

.resumen-card.en-progreso svg {
  color: #3b82f6;
}

.resumen-card.en-riesgo svg {
  color: #f59e0b;
}

.resumen-card.no-cumplido svg {
  color: #ef4444;
}

.resumen-header h3 {
  font-size: 1rem;
  font-weight: 600;
  color: #64748b;
}

.resumen-numero {
  font-size: 2.5rem;
  font-weight: 800;
  color: #1e293b;
  margin-bottom: 0.25rem;
}

.resumen-detalle {
  font-size: 0.875rem;
  color: #64748b;
}

.compromisos-lista h2 {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 1.5rem;
}

.compromisos-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 450px), 1fr));
  gap: 1.5rem;
}

.compromiso-card {
  background: white;
  border-radius: 12px;
  border: 2px solid #e2e8f0;
  overflow: hidden;
  transition: all 0.3s;
}

.compromiso-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
}

.compromiso-card.estado-cumplido {
  border-left: 4px solid #22c55e;
}

.compromiso-card.estado-en-progreso {
  border-left: 4px solid #3b82f6;
}

.compromiso-card.estado-en-riesgo {
  border-left: 4px solid #f59e0b;
}

.compromiso-card.estado-no-cumplido {
  border-left: 4px solid #ef4444;
}

.compromiso-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.25rem;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}

.compromiso-titulo h3 {
  font-size: 1.125rem;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 0.25rem;
}

.compromiso-titulo p {
  font-size: 0.875rem;
  color: #64748b;
}

.badge-estado {
  padding: 0.35rem 0.85rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-estado.estado-cumplido {
  background: #dcfce7;
  color: #166534;
}

.badge-estado.estado-en-progreso {
  background: #dbeafe;
  color: #1e40af;
}

.badge-estado.estado-en-riesgo {
  background: #fef3c7;
  color: #92400e;
}

.badge-estado.estado-no-cumplido {
  background: #fee2e2;
  color: #991b1b;
}

.compromiso-body {
  padding: 1.25rem;
}

.compromiso-meta {
  display: flex;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.meta-label {
  font-size: 0.75rem;
  color: #64748b;
  font-weight: 500;
}

.meta-valor {
  font-size: 1rem;
  color: #1e293b;
  font-weight: 600;
}

.meta-valor.destacado {
  font-size: 1.25rem;
  color: #3b82f6;
  font-weight: 700;
}

.progreso-container {
  margin-bottom: 1.25rem;
}

.progreso-header {
  display: flex;
  justify-content: space-between;
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
  color: #64748b;
  font-weight: 500;
}

.progreso-porcentaje {
  color: #1e293b;
  font-weight: 700;
}

.progreso-barra {
  height: 10px;
  background: #e2e8f0;
  border-radius: 9999px;
  overflow: hidden;
}

.progreso-fill {
  height: 100%;
  border-radius: 9999px;
  transition: width 0.5s;
}

.progreso-fill.fill-cumplido {
  background: #22c55e;
}

.progreso-fill.fill-en-progreso {
  background: #3b82f6;
}

.progreso-fill.fill-en-riesgo {
  background: #f59e0b;
}

.progreso-fill.fill-no-cumplido {
  background: #ef4444;
}

.compromiso-info {
  display: flex;
  gap: 1.5rem;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: #64748b;
}

.info-item svg {
  color: #94a3b8;
}

.compromiso-footer {
  display: flex;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  background: #f8fafc;
  border-top: 1px solid #e2e8f0;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover {
  background: #2563eb;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-secondary {
  background: #f1f5f9;
  color: #475569;
}

.btn-secondary:hover {
  background: #e2e8f0;
}

.btn-small {
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  flex: 1;
  justify-content: center;
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 9999;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: white;
  border-radius: 16px;
  width: 90%;
  max-width: min(800px, 92vw);
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.2);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e2e8f0;
}

.modal-header h3 {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1e293b;
}

.btn-close {
  background: none;
  border: none;
  color: #64748b;
  cursor: pointer;
  transition: color 0.2s;
}

.btn-close:hover {
  color: #ef4444;
}

.modal-body {
  padding: 1.5rem;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
  gap: 1.25rem;
}

.form-group {
  display: flex;
  flex-direction: column;
}

.form-group.full-width {
  grid-column: 1 / -1;
}

.form-group label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #475569;
  margin-bottom: 0.5rem;
}

textarea.form-control {
  resize: vertical;
  font-family: inherit;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid #e2e8f0;
}

@media (max-width: 1024px) {
  .compromisos-container {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .filtros-section {
    flex-direction: column;
  }

  .filtros-container {
    width: 100%;
  }

  .form-control {
    width: 100%;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
