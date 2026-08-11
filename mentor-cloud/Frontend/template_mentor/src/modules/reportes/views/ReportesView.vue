<template>
  <div class="reportes-view">
    <div class="page-header">
      <h1>Reportes de Producción</h1>
      <p>Genera y descarga reportes personalizados de producción</p>
    </div>

    <!-- Filtros -->
    <div class="filtros-section">
      <div class="filtros-grid">
        <div class="form-group">
          <label>Tipo de Reporte</label>
          <select v-model="filtros.tipoReporte" class="form-control">
            <option value="">Seleccionar...</option>
            <option value="oee">Reporte OEE</option>
            <option value="produccion">Reporte de Producción</option>
            <option value="paradas">Reporte de Paradas</option>
            <option value="defectos">Reporte de Defectos</option>
            <option value="rendimiento">Reporte de Rendimiento</option>
          </select>
        </div>

        <div class="form-group">
          <label>Compañía</label>
          <select v-model="filtros.compania" class="form-control">
            <option value="">Todas</option>
            <option value="1">Art Atlas</option>
            <option value="2">Indagua</option>
            <option value="3">Demo</option>
            <option value="4">Agrícola Cerro Prieto</option>
          </select>
        </div>

        <div class="form-group">
          <label>Planta</label>
          <select v-model="filtros.planta" class="form-control">
            <option value="">Todas</option>
            <option value="1">Planta Textil Lima</option>
            <option value="2">Planta Textil Norte</option>
            <option value="3">Indagua Principal</option>
            <option value="4">ACP Ica</option>
          </select>
        </div>

        <div class="form-group">
          <label>Línea</label>
          <select v-model="filtros.linea" class="form-control">
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
          <label>Fecha Inicio</label>
          <input type="date" v-model="filtros.fechaInicio" class="form-control">
        </div>

        <div class="form-group">
          <label>Fecha Fin</label>
          <input type="date" v-model="filtros.fechaFin" class="form-control">
        </div>

        <div class="form-group">
          <label>Turno</label>
          <select v-model="filtros.turno" class="form-control">
            <option value="">Todos</option>
            <option value="A">Turno A</option>
            <option value="B">Turno B</option>
            <option value="C">Turno C</option>
          </select>
        </div>

        <div class="form-group">
          <label>Formato</label>
          <select v-model="filtros.formato" class="form-control">
            <option value="excel">Excel (.xlsx)</option>
            <option value="pdf">PDF</option>
            <option value="csv">CSV</option>
          </select>
        </div>
      </div>

      <div class="filtros-actions">
        <button @click="limpiarFiltros" class="btn btn-secondary">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
          </svg>
          Limpiar
        </button>
        <button @click="generarReporte" class="btn btn-primary" :disabled="!filtros.tipoReporte">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/>
          </svg>
          Generar Reporte
        </button>
      </div>
    </div>

    <!-- Reportes Recientes -->
    <div class="reportes-recientes">
      <h2>Reportes Recientes</h2>
      <div class="tabla-wrapper">
        <table class="tabla-reportes">
          <thead>
            <tr>
              <th>Fecha Generación</th>
              <th>Tipo</th>
              <th>Periodo</th>
              <th>Compañía/Planta</th>
              <th>Formato</th>
              <th>Estado</th>
              <th>Acciones</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="reporte in reportesRecientes" :key="reporte.id">
              <td>{{ reporte.fecha }}</td>
              <td><span class="badge badge-info">{{ reporte.tipo }}</span></td>
              <td>{{ reporte.periodo }}</td>
              <td>{{ reporte.ubicacion }}</td>
              <td>{{ reporte.formato }}</td>
              <td><span :class="['badge', `badge-${reporte.estado}`]">{{ reporte.estadoTexto }}</span></td>
              <td>
                <div class="acciones">
                  <button @click="descargarReporte(reporte.id)" class="btn-icon" title="Descargar">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/>
                    </svg>
                  </button>
                  <button @click="eliminarReporte(reporte.id)" class="btn-icon btn-danger" title="Eliminar">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const filtros = ref({
  tipoReporte: '',
  compania: '',
  planta: '',
  linea: '',
  fechaInicio: '',
  fechaFin: '',
  turno: '',
  formato: 'excel'
})

const reportesRecientes = ref([
  { id: 1, fecha: '19/11/2025 10:30', tipo: 'OEE', periodo: '01/11 - 18/11', ubicacion: 'Planta Textil Lima', formato: 'Excel', estado: 'success', estadoTexto: 'Completado' },
  { id: 2, fecha: '18/11/2025 16:45', tipo: 'Producción', periodo: '18/11/2025', ubicacion: 'Todas las plantas', formato: 'PDF', estado: 'success', estadoTexto: 'Completado' },
  { id: 3, fecha: '18/11/2025 09:15', tipo: 'Paradas', periodo: '15/11 - 17/11', ubicacion: 'Planta Textil Norte / Línea 03', formato: 'Excel', estado: 'success', estadoTexto: 'Completado' },
  { id: 4, fecha: '17/11/2025 14:20', tipo: 'Defectos', periodo: '10/11 - 16/11', ubicacion: 'Indagua Principal', formato: 'CSV', estado: 'success', estadoTexto: 'Completado' },
  { id: 5, fecha: '17/11/2025 08:00', tipo: 'Rendimiento', periodo: 'Noviembre 2025', ubicacion: 'Planta Textil Lima', formato: 'PDF', estado: 'success', estadoTexto: 'Completado' }
])

function limpiarFiltros() {
  filtros.value = {
    tipoReporte: '',
    compania: '',
    planta: '',
    linea: '',
    fechaInicio: '',
    fechaFin: '',
    turno: '',
    formato: 'excel'
  }
}

function generarReporte() {
  // TODO: Conectar con backend
  console.log('Generando reporte con filtros:', filtros.value)
  alert('Función lista para conectar con backend')
}

function descargarReporte(id) {
  // TODO: Conectar con backend
  console.log('Descargando reporte:', id)
  alert('Función lista para conectar con backend')
}

function eliminarReporte(id) {
  // TODO: Conectar con backend
  if (confirm('¿Está seguro de eliminar este reporte?')) {
    console.log('Eliminando reporte:', id)
    alert('Función lista para conectar con backend')
  }
}
</script>

<style scoped>
.reportes-view {
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
}

.filtros-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
}

.form-group label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #475569;
  margin-bottom: 0.5rem;
}

.form-control {
  padding: 0.625rem;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  font-size: 0.875rem;
  transition: all 0.2s;
}

.form-control:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.filtros-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
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

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-primary:disabled {
  background: #cbd5e1;
  cursor: not-allowed;
}

.btn-secondary {
  background: #f1f5f9;
  color: #475569;
}

.btn-secondary:hover {
  background: #e2e8f0;
}

.reportes-recientes h2 {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1e293b;
  margin-bottom: 1rem;
}

.tabla-wrapper {
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.tabla-reportes {
  width: 100%;
  border-collapse: collapse;
}

.tabla-reportes thead {
  background: #f8fafc;
}

.tabla-reportes th {
  padding: 1rem;
  text-align: left;
  font-size: 0.75rem;
  font-weight: 700;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.tabla-reportes td {
  padding: 1rem;
  border-top: 1px solid #f1f5f9;
  font-size: 0.875rem;
  color: #334155;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-info {
  background: #dbeafe;
  color: #1e40af;
}

.badge-success {
  background: #dcfce7;
  color: #166534;
}

.acciones {
  display: flex;
  gap: 0.5rem;
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  background: white;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: #f8fafc;
  color: #3b82f6;
  border-color: #3b82f6;
}

.btn-icon.btn-danger:hover {
  background: #fef2f2;
  color: #dc2626;
  border-color: #dc2626;
}
</style>
