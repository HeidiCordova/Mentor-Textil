# 📡 Guía de Servicios API - Frontend

## 🎯 Descripción General

Este documento explica cómo funcionan los servicios API en el frontend y cómo el backend debe implementar los endpoints correspondientes.

## 🏗️ Arquitectura de Servicios

### Estructura de Carpetas
```
src/api/
├── client.js              # Cliente HTTP configurado
├── endpoints.js           # URLs centralizadas
├── services/              # Servicios por dominio
│   ├── auth.service.js
│   ├── dashboard.service.js
│   ├── plant.service.js
│   ├── analysis.service.js
│   ├── company.service.js
│   └── user.service.js
└── mocks/                 # Datos de prueba
    ├── auth.mock.js
    ├── dashboard.mock.js
    ├── plantas.mock.js
    ├── analisis.mock.js
    ├── empresas.mock.js
    └── usuarios.mock.js
```

## 🔧 Cliente HTTP (client.js)

### Configuración Base

```javascript
import axios from 'axios'

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json'
  }
})

// Interceptor de Request - Agrega JWT token
apiClient.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => Promise.reject(error)
)

// Interceptor de Response - Maneja errores globales
apiClient.interceptors.response.use(
  response => response,
  async error => {
    const originalRequest = error.config
    
    // Si el token expiró (401) y no es la ruta de login
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true
      
      try {
        // Intentar renovar el token
        const refreshToken = localStorage.getItem('refreshToken')
        const { data } = await axios.post(
          `${apiClient.defaults.baseURL}/auth/refresh`,
          { refreshToken }
        )
        
        localStorage.setItem('token', data.token)
        originalRequest.headers.Authorization = `Bearer ${data.token}`
        
        return apiClient(originalRequest)
      } catch (refreshError) {
        // Si falla el refresh, logout
        localStorage.clear()
        window.location.href = '/login'
        return Promise.reject(refreshError)
      }
    }
    
    return Promise.reject(error)
  }
)

export default apiClient
```

## 📋 Endpoints Centralizados (endpoints.js)

```javascript
export const endpoints = {
  // Autenticación
  auth: {
    login: '/auth/login',
    logout: '/auth/logout',
    refresh: '/auth/refresh',
    me: '/auth/me'
  },
  
  // Dashboard
  dashboard: {
    stats: '/dashboard/stats',
    production: '/dashboard/production',
    efficiency: '/dashboard/efficiency'
  },
  
  // Plantas
  plants: {
    list: '/plantas',
    detail: (id) => `/plantas/${id}`,
    create: '/plantas',
    update: (id) => `/plantas/${id}`,
    delete: (id) => `/plantas/${id}`,
    lines: (id) => `/plantas/${id}/lineas`
  },
  
  // Análisis
  analysis: {
    production: '/analisis/produccion',
    energy: '/analisis/energia',
    oee: '/analisis/oee',
    pareto: '/analisis/pareto'
  },
  
  // Empresas
  companies: {
    list: '/empresas',
    detail: (id) => `/empresas/${id}`,
    create: '/empresas',
    update: (id) => `/empresas/${id}`,
    delete: (id) => `/empresas/${id}`
  },
  
  // Usuarios
  users: {
    list: '/usuarios',
    detail: (id) => `/usuarios/${id}`,
    create: '/usuarios',
    update: (id) => `/usuarios/${id}`,
    delete: (id) => `/usuarios/${id}`,
    roles: '/usuarios/roles'
  }
}
```

## 📚 Servicios Implementados

### 1. Auth Service (auth.service.js)

```javascript
import apiClient from '../client'
import { endpoints } from '../endpoints'

export const authService = {
  /**
   * Login de usuario
   * @param {Object} credentials - { username, password }
   * @returns {Promise} { token, refreshToken, user }
   */
  async login(credentials) {
    const { data } = await apiClient.post(endpoints.auth.login, credentials)
    return data
  },

  /**
   * Logout de usuario
   * @returns {Promise}
   */
  async logout() {
    const { data } = await apiClient.post(endpoints.auth.logout)
    return data
  },

  /**
   * Renovar token
   * @param {string} refreshToken
   * @returns {Promise} { token, refreshToken }
   */
  async refreshToken(refreshToken) {
    const { data } = await apiClient.post(endpoints.auth.refresh, { refreshToken })
    return data
  },

  /**
   * Obtener usuario actual
   * @returns {Promise} User object
   */
  async getCurrentUser() {
    const { data } = await apiClient.get(endpoints.auth.me)
    return data
  }
}
```

**Backend debe implementar:**
- `POST /api/auth/login` - Autenticar usuario
- `POST /api/auth/logout` - Cerrar sesión
- `POST /api/auth/refresh` - Renovar JWT token
- `GET /api/auth/me` - Obtener usuario autenticado

---

### 2. Dashboard Service (dashboard.service.js)

```javascript
import apiClient from '../client'
import { endpoints } from '../endpoints'

export const dashboardService = {
  /**
   * Obtener estadísticas generales
   * @returns {Promise} { totalPlantas, totalLineas, totalDispositivos, alarmasActivas, eficienciaPromedio, produccionHoy }
   */
  async getStats() {
    const { data } = await apiClient.get(endpoints.dashboard.stats)
    return data
  },

  /**
   * Obtener datos de producción
   * @param {Object} params - { startDate, endDate, plantId }
   * @returns {Promise} Array de datos de producción por hora
   */
  async getProductionData(params) {
    const { data } = await apiClient.get(endpoints.dashboard.production, { params })
    return data
  },

  /**
   * Obtener métricas de eficiencia
   * @param {Object} params - { startDate, endDate, plantId }
   * @returns {Promise} Datos de OEE (disponibilidad, rendimiento, calidad)
   */
  async getEfficiency(params) {
    const { data } = await apiClient.get(endpoints.dashboard.efficiency, { params })
    return data
  }
}
```

**Backend debe implementar:**
- `GET /api/dashboard/stats` - Estadísticas generales
- `GET /api/dashboard/production?startDate=&endDate=&plantId=` - Datos de producción
- `GET /api/dashboard/efficiency?startDate=&endDate=&plantId=` - Métricas OEE

---

### 3. Plant Service (plant.service.js)

```javascript
import apiClient from '../client'
import { endpoints } from '../endpoints'

export const plantService = {
  /**
   * Listar todas las plantas
   * @param {Object} params - { page, limit, search }
   * @returns {Promise} { data: [], total, page, limit }
   */
  async getAll(params = {}) {
    const { data } = await apiClient.get(endpoints.plants.list, { params })
    return data
  },

  /**
   * Obtener detalle de planta
   * @param {number} id - ID de la planta
   * @returns {Promise} Objeto planta
   */
  async getById(id) {
    const { data } = await apiClient.get(endpoints.plants.detail(id))
    return data
  },

  /**
   * Crear nueva planta
   * @param {Object} plant - Datos de la planta
   * @returns {Promise} Planta creada
   */
  async create(plant) {
    const { data } = await apiClient.post(endpoints.plants.create, plant)
    return data
  },

  /**
   * Actualizar planta
   * @param {number} id - ID de la planta
   * @param {Object} plant - Datos actualizados
   * @returns {Promise} Planta actualizada
   */
  async update(id, plant) {
    const { data } = await apiClient.put(endpoints.plants.update(id), plant)
    return data
  },

  /**
   * Eliminar planta
   * @param {number} id - ID de la planta
   * @returns {Promise}
   */
  async delete(id) {
    const { data } = await apiClient.delete(endpoints.plants.delete(id))
    return data
  },

  /**
   * Obtener líneas de una planta
   * @param {number} id - ID de la planta
   * @returns {Promise} Array de líneas
   */
  async getLines(id) {
    const { data } = await apiClient.get(endpoints.plants.lines(id))
    return data
  }
}
```

**Backend debe implementar:**
- `GET /api/plantas?page=&limit=&search=` - Listar plantas (paginado)
- `GET /api/plantas/:id` - Detalle de planta
- `POST /api/plantas` - Crear planta
- `PUT /api/plantas/:id` - Actualizar planta
- `DELETE /api/plantas/:id` - Eliminar planta
- `GET /api/plantas/:id/lineas` - Líneas de la planta

---

### 4. Analysis Service (analysis.service.js)

```javascript
import apiClient from '../client'
import { endpoints } from '../endpoints'

export const analysisService = {
  /**
   * Análisis de producción
   * @param {Object} params - { startDate, endDate, plantId, lineId }
   * @returns {Promise} Datos de análisis de producción
   */
  async getProductionAnalysis(params) {
    const { data } = await apiClient.get(endpoints.analysis.production, { params })
    return data
  },

  /**
   * Análisis energético
   * @param {Object} params - { startDate, endDate, plantId }
   * @returns {Promise} Datos de consumo energético
   */
  async getEnergyAnalysis(params) {
    const { data } = await apiClient.get(endpoints.analysis.energy, { params })
    return data
  },

  /**
   * Análisis OEE
   * @param {Object} params - { startDate, endDate, plantId, lineId }
   * @returns {Promise} Métricas OEE detalladas
   */
  async getOEEAnalysis(params) {
    const { data } = await apiClient.get(endpoints.analysis.oee, { params })
    return data
  },

  /**
   * Gráfico de Pareto
   * @param {Object} params - { startDate, endDate, plantId, type }
   * @returns {Promise} Datos para gráfico de Pareto
   */
  async getParetoData(params) {
    const { data } = await apiClient.get(endpoints.analysis.pareto, { params })
    return data
  }
}
```

**Backend debe implementar:**
- `GET /api/analisis/produccion` - Análisis de producción
- `GET /api/analisis/energia` - Análisis energético
- `GET /api/analisis/oee` - Análisis OEE
- `GET /api/analisis/pareto` - Datos de Pareto

---

### 5. Company Service (company.service.js)

```javascript
import apiClient from '../client'
import { endpoints } from '../endpoints'

export const companyService = {
  /**
   * Listar todas las empresas
   * @returns {Promise} Array de empresas
   */
  async getAll() {
    const { data } = await apiClient.get(endpoints.companies.list)
    return data
  },

  /**
   * Obtener detalle de empresa
   * @param {number} id - ID de la empresa
   * @returns {Promise} Objeto empresa
   */
  async getById(id) {
    const { data } = await apiClient.get(endpoints.companies.detail(id))
    return data
  },

  /**
   * Crear nueva empresa
   * @param {Object} company - Datos de la empresa
   * @returns {Promise} Empresa creada
   */
  async create(company) {
    const { data } = await apiClient.post(endpoints.companies.create, company)
    return data
  },

  /**
   * Actualizar empresa
   * @param {number} id - ID de la empresa
   * @param {Object} company - Datos actualizados
   * @returns {Promise} Empresa actualizada
   */
  async update(id, company) {
    const { data } = await apiClient.put(endpoints.companies.update(id), company)
    return data
  },

  /**
   * Eliminar empresa
   * @param {number} id - ID de la empresa
   * @returns {Promise}
   */
  async delete(id) {
    const { data } = await apiClient.delete(endpoints.companies.delete(id))
    return data
  }
}
```

**Backend debe implementar:**
- `GET /api/empresas` - Listar empresas
- `GET /api/empresas/:id` - Detalle de empresa
- `POST /api/empresas` - Crear empresa
- `PUT /api/empresas/:id` - Actualizar empresa
- `DELETE /api/empresas/:id` - Eliminar empresa

---

### 6. User Service (user.service.js)

```javascript
import apiClient from '../client'
import { endpoints } from '../endpoints'

export const userService = {
  /**
   * Listar usuarios
   * @param {Object} params - { page, limit, role, search }
   * @returns {Promise} { data: [], total, page, limit }
   */
  async getAll(params = {}) {
    const { data } = await apiClient.get(endpoints.users.list, { params })
    return data
  },

  /**
   * Obtener detalle de usuario
   * @param {number} id - ID del usuario
   * @returns {Promise} Objeto usuario
   */
  async getById(id) {
    const { data } = await apiClient.get(endpoints.users.detail(id))
    return data
  },

  /**
   * Crear nuevo usuario
   * @param {Object} user - Datos del usuario
   * @returns {Promise} Usuario creado
   */
  async create(user) {
    const { data } = await apiClient.post(endpoints.users.create, user)
    return data
  },

  /**
   * Actualizar usuario
   * @param {number} id - ID del usuario
   * @param {Object} user - Datos actualizados
   * @returns {Promise} Usuario actualizado
   */
  async update(id, user) {
    const { data } = await apiClient.put(endpoints.users.update(id), user)
    return data
  },

  /**
   * Eliminar usuario
   * @param {number} id - ID del usuario
   * @returns {Promise}
   */
  async delete(id) {
    const { data } = await apiClient.delete(endpoints.users.delete(id))
    return data
  },

  /**
   * Obtener roles disponibles
   * @returns {Promise} Array de roles
   */
  async getRoles() {
    const { data } = await apiClient.get(endpoints.users.roles)
    return data
  }
}
```

**Backend debe implementar:**
- `GET /api/usuarios?page=&limit=&role=&search=` - Listar usuarios
- `GET /api/usuarios/:id` - Detalle de usuario
- `POST /api/usuarios` - Crear usuario
- `PUT /api/usuarios/:id` - Actualizar usuario
- `DELETE /api/usuarios/:id` - Eliminar usuario
- `GET /api/usuarios/roles` - Listar roles

---

## 🎨 Uso en Componentes Vue

### Ejemplo con Composable

```vue
<script setup>
import { ref, onMounted } from 'vue'
import { useApi } from '@/shared/composables/useApi'
import { plantService } from '@/api/services/plant.service'

const { loading, error, execute } = useApi()
const plants = ref([])

const fetchPlants = async () => {
  const { data } = await execute(() => plantService.getAll())
  if (data) {
    plants.value = data.data
  }
}

onMounted(() => {
  fetchPlants()
})
</script>

<template>
  <div>
    <Loading v-if="loading" />
    <ErrorMessage v-else-if="error" :message="error" />
    <PlantList v-else :plants="plants" />
  </div>
</template>
```

### Ejemplo Directo

```vue
<script setup>
import { ref } from 'vue'
import { dashboardService } from '@/api/services/dashboard.service'

const stats = ref(null)
const loading = ref(false)
const error = ref(null)

const fetchStats = async () => {
  try {
    loading.value = true
    const response = await dashboardService.getStats()
    stats.value = response
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}
</script>
```

## 🔄 Manejo de Errores

### Estructura de Error del Backend

El backend debe retornar errores en este formato:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Los datos proporcionados no son válidos",
    "details": [
      {
        "field": "email",
        "message": "El email es requerido"
      }
    ]
  }
}
```

### Códigos de Error Esperados

| Código HTTP | Significado | Acción Frontend |
|-------------|-------------|-----------------|
| 200 | OK | Procesar datos |
| 201 | Created | Mostrar éxito, actualizar lista |
| 400 | Bad Request | Mostrar errores de validación |
| 401 | Unauthorized | Redirigir a login |
| 403 | Forbidden | Mostrar mensaje de permisos |
| 404 | Not Found | Mostrar "no encontrado" |
| 500 | Server Error | Mostrar error genérico |

## 📦 Mock Data

Los archivos en `api/mocks/` contienen datos de ejemplo para desarrollo sin backend:

```javascript
// api/mocks/dashboard.mock.js
export const dashboardMockData = {
  stats: {
    totalPlantas: 12,
    totalLineas: 48,
    totalDispositivos: 156,
    alarmasActivas: 3,
    eficienciaPromedio: 87.5,
    produccionHoy: 15240
  }
}
```

## 🚀 Migración de Mock a Real

Para cambiar de mock data a endpoints reales:

1. Actualizar `VITE_API_BASE_URL` en `.env`
2. Comentar/eliminar imports de mocks
3. Descomentar llamadas a servicios reales
4. Verificar que el backend retorne el formato esperado

---

**Última actualización:** Noviembre 2025  
**Versión:** 2.0
