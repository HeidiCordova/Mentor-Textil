# 📐 Arquitectura del Frontend - Mentor Monitor

## 🎯 Visión General

Este documento describe la arquitectura completa del template frontend de Mentor Monitor, diseñado con Vue 3 y preparado para integrarse con un backend RESTful.

## 🏗️ Estructura de Carpetas

```
mentor-monitor/
├── public/                          # Archivos estáticos públicos
│   └── mentor logo.png             # Logo de la aplicación
│
├── src/                            # Código fuente principal
│   ├── main.js                     # Punto de entrada de la aplicación
│   ├── App.vue                     # Componente raíz
│   │
│   ├── api/                        # Capa de comunicación con backend
│   │   ├── client.js              # Cliente HTTP configurado (Axios)
│   │   ├── endpoints.js           # URLs de endpoints centralizadas
│   │   │
│   │   ├── mocks/                 # Datos de prueba (mock data)
│   │   │   ├── auth.mock.js      # Datos de autenticación
│   │   │   ├── dashboard.mock.js # Datos del dashboard
│   │   │   ├── plantas.mock.js   # Datos de plantas
│   │   │   ├── analisis.mock.js  # Datos de análisis
│   │   │   ├── empresas.mock.js  # Datos de empresas
│   │   │   └── usuarios.mock.js  # Datos de usuarios
│   │   │
│   │   └── services/              # Servicios de API por dominio
│   │       ├── auth.service.js   # Login, logout, refresh token
│   │       ├── dashboard.service.js # Estadísticas generales
│   │       ├── plant.service.js  # CRUD de plantas
│   │       ├── analysis.service.js # Análisis y reportes
│   │       ├── company.service.js # Gestión de empresas
│   │       └── user.service.js   # Gestión de usuarios
│   │
│   ├── assets/                    # Recursos estáticos
│   │   └── icons/                # Iconos SVG personalizados
│   │
│   ├── data/                      # Datos de configuración
│   │   └── arbol-paradas.json    # Árbol jerárquico de paradas
│   │
│   ├── modules/                   # Módulos funcionales de la app
│   │   │
│   │   ├── auth/                 # Módulo de autenticación
│   │   │   └── views/
│   │   │       └── LoginView.vue # Pantalla de login
│   │   │
│   │   ├── dashboard/            # Módulo principal
│   │   │   └── views/
│   │   │       └── DashboardView.vue # Dashboard con KPIs y gráficos
│   │   │
│   │   ├── administracion/       # Módulo de administración
│   │   │   └── views/
│   │   │       ├── CalendarizacionView.vue
│   │   │       ├── HabilitarLineaView.vue
│   │   │       ├── ProductosView.vue
│   │   │       ├── TipoDocumentoView.vue
│   │   │       └── TurnosView.vue
│   │   │
│   │   ├── alarmas/              # Módulo de alarmas
│   │   │   ├── router/
│   │   │   │   └── index.js
│   │   │   └── views/
│   │   │       └── AlarmasView.vue
│   │   │
│   │   ├── analisis/             # Análisis general
│   │   │   └── views/
│   │   │       ├── EnergiaView.vue
│   │   │       ├── GeneralView.vue
│   │   │       └── ProduccionView.vue
│   │   │
│   │   ├── analisis-avanzado/    # Análisis avanzado
│   │   │   └── views/
│   │   │       ├── GeneradorConsultasView.vue
│   │   │       └── KwhlView.vue
│   │   │
│   │   ├── analisis-energia/     # Análisis energético
│   │   │   └── views/
│   │   │       ├── ConsumoElectricoTarifarioView.vue
│   │   │       └── FactorCalificacionView.vue
│   │   │
│   │   ├── analisis-produccion/  # Análisis de producción
│   │   │   └── views/
│   │   │       ├── GraficaOEEView.vue
│   │   │       ├── GraficaParetoView.vue
│   │   │       ├── HistoriaLineaView.vue
│   │   │       ├── LineaTiempoView.vue
│   │   │       └── TiempoRealView.vue
│   │   │
│   │   ├── compromisos/          # Módulo de compromisos
│   │   │   ├── router/
│   │   │   └── views/
│   │   │       └── CompromisosView.vue
│   │   │
│   │   ├── configuracion/        # Configuración del sistema
│   │   │   └── views/
│   │   │       ├── ArbolParadasView.vue
│   │   │       ├── ArchivosView.vue
│   │   │       ├── CategoriaParadasView.vue
│   │   │       ├── EmpresaView.vue
│   │   │       ├── GestionParadasView.vue
│   │   │       └── ... (más vistas de configuración)
│   │   │
│   │   ├── datos-recibidos/      # Módulo de datos recibidos
│   │   │   ├── router/
│   │   │   └── views/
│   │   │
│   │   └── reportes/             # Módulo de reportes
│   │       ├── router/
│   │       └── views/
│   │
│   ├── router/                    # Configuración de rutas
│   │   └── index.js              # Router principal de Vue Router
│   │
│   ├── shared/                    # Código compartido
│   │   │
│   │   ├── components/           # Componentes reutilizables
│   │   │   ├── forms/           # Componentes de formularios
│   │   │   ├── layout/          # Componentes de layout
│   │   │   │   ├── AppSidebar.vue
│   │   │   │   ├── AppHeader.vue
│   │   │   │   └── AppLayout.vue
│   │   │   └── ui/              # Componentes UI básicos
│   │   │       ├── Button.vue
│   │   │       ├── Card.vue
│   │   │       ├── Loading.vue
│   │   │       ├── Modal.vue
│   │   │       └── ...
│   │   │
│   │   ├── composables/          # Composables de Vue
│   │   │   ├── useApi.js        # Manejo de llamadas API
│   │   │   ├── useAuth.js       # Lógica de autenticación
│   │   │   ├── useDateRange.js  # Manejo de rangos de fecha
│   │   │   ├── useModal.js      # Control de modales
│   │   │   ├── useTable.js      # Lógica de tablas
│   │   │   └── useToast.js      # Notificaciones toast
│   │   │
│   │   └── utils/                # Utilidades generales
│   │       ├── constants.js     # Constantes de la aplicación
│   │       ├── formatters.js    # Formateadores de datos
│   │       └── validators.js    # Validadores de formularios
│   │
│   ├── stores/                    # Estado global (Pinia)
│   │   ├── auth.js               # Store de autenticación
│   │   ├── filters.js            # Store de filtros
│   │   └── ui.js                 # Store de UI (sidebar, modales)
│   │
│   └── styles/                    # Estilos globales
│       ├── main.css              # Estilos principales
│       └── transitions.css       # Transiciones y animaciones
│
├── docs/                          # Documentación
│   ├── BACKEND_INTEGRATION.md    # Guía de integración backend
│   ├── ARQUITECTURA_FRONTEND.md  # Este documento
│   └── api-contracts/            # Contratos de API
│       ├── auth.json             # Endpoints de autenticación
│       ├── dashboard.json        # Endpoints de dashboard
│       ├── plantas.json          # Endpoints de plantas
│       └── analisis.json         # Endpoints de análisis
│
├── index.html                     # HTML principal
├── package.json                   # Dependencias del proyecto
├── vite.config.js                 # Configuración de Vite
├── tailwind.config.js            # Configuración de TailwindCSS
├── postcss.config.js             # Configuración de PostCSS
└── jsconfig.json                 # Configuración de JavaScript

```

## 🎨 Patrones de Arquitectura

### 1. **Arquitectura Modular**
Cada módulo funcional es independiente y contiene sus propias vistas, rutas y lógica.

```javascript
modules/
  └── nombre-modulo/
      ├── router/           # Rutas específicas del módulo
      ├── views/            # Vistas del módulo
      ├── components/       # Componentes exclusivos (opcional)
      └── composables/      # Lógica específica (opcional)
```

### 2. **Separación de Capas**

#### **Capa de Presentación (UI)**
- Componentes Vue en `modules/*/views/`
- Componentes compartidos en `shared/components/`
- Estilos con TailwindCSS

#### **Capa de Lógica de Negocio**
- Composables en `shared/composables/`
- Stores de Pinia en `stores/`
- Utilidades en `shared/utils/`

#### **Capa de Datos**
- Servicios API en `api/services/`
- Cliente HTTP en `api/client.js`
- Mock data en `api/mocks/`

### 3. **Comunicación con Backend**

```
Vista (Vue Component)
  ↓ usa
Composable (useApi)
  ↓ llama
Servicio (*.service.js)
  ↓ usa
Cliente HTTP (api/client.js)
  ↓ hace request
Backend API
```

## 🔧 Tecnologías Principales

| Tecnología | Versión | Propósito |
|------------|---------|-----------|
| Vue.js | 3.4.0 | Framework principal |
| Vue Router | 4.2.5 | Navegación SPA |
| Pinia | 2.1.7 | Gestión de estado |
| Axios | 1.6.2 | Cliente HTTP |
| TailwindCSS | 3.4.0 | Framework CSS |
| ECharts | 5.4.3 | Gráficos interactivos |
| Vue-ECharts | 6.6.9 | Integración de ECharts |
| Vite | 5.0.8 | Build tool |

## 📦 Módulos Principales

### 1. **Auth (Autenticación)**
- Login con credenciales
- Manejo de JWT tokens
- Refresh automático de tokens
- Protección de rutas

**Archivos clave:**
- `src/modules/auth/views/LoginView.vue`
- `src/stores/auth.js`
- `src/api/services/auth.service.js`

### 2. **Dashboard**
- KPIs principales
- Gráficos de producción
- Medidor OEE
- Estadísticas en tiempo real

**Archivos clave:**
- `src/modules/dashboard/views/DashboardView.vue`
- `src/api/services/dashboard.service.js`

### 3. **Configuración**
- Gestión de empresas
- Configuración de plantas
- Árbol de paradas
- Gestión de usuarios

### 4. **Análisis**
- Análisis de producción
- Análisis energético
- Reportes avanzados
- Generador de consultas

### 5. **Administración**
- Turnos
- Productos
- Calendarización
- Tipos de documento

## 🔐 Seguridad

### Autenticación JWT
```javascript
// Interceptor en api/client.js
apiClient.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  }
)
```

### Protección de Rutas
```javascript
// En router/index.js
router.beforeEach((to, from, next) => {
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth)
  const isAuthenticated = useAuthStore().isAuthenticated
  
  if (requiresAuth && !isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})
```

## 📊 Gestión de Estado

### Store de Autenticación
```javascript
// stores/auth.js
{
  user: null,
  token: null,
  isAuthenticated: false,
  login(credentials),
  logout(),
  refreshToken()
}
```

### Store de UI
```javascript
// stores/ui.js
{
  sidebarOpen: true,
  theme: 'light',
  toggleSidebar(),
  setTheme(theme)
}
```

### Store de Filtros
```javascript
// stores/filters.js
{
  dateRange: { start, end },
  selectedPlants: [],
  selectedLines: []
}
```

## 🌐 Rutas Principales

```javascript
/                           → Dashboard
/login                      → Login

/administracion/*          → Módulo de administración
/alarmas                   → Alarmas activas
/analisis/*                → Análisis general
/analisis-avanzado/*       → Análisis avanzado
/analisis-energia/*        → Análisis energético
/analisis-produccion/*     → Análisis de producción
/compromisos               → Compromisos
/configuracion/*           → Configuración del sistema
/datos-recibidos           → Datos recibidos
/reportes                  → Reportes
```

## 🎯 Convenciones de Código

### Nomenclatura de Archivos
- **Componentes:** PascalCase (ej: `AppSidebar.vue`)
- **Vistas:** PascalCase + "View" (ej: `DashboardView.vue`)
- **Servicios:** camelCase + ".service.js" (ej: `auth.service.js`)
- **Stores:** camelCase + ".js" (ej: `auth.js`)
- **Composables:** camelCase + "use" prefix (ej: `useApi.js`)

### Estructura de Componentes Vue
```vue
<script setup>
// 1. Imports
import { ref, computed, onMounted } from 'vue'

// 2. Props y Emits
const props = defineProps({...})
const emit = defineEmits([...])

// 3. Estado reactivo
const data = ref(null)

// 4. Computed
const computedValue = computed(() => {...})

// 5. Métodos
const fetchData = async () => {...}

// 6. Lifecycle hooks
onMounted(() => {...})
</script>

<template>
  <!-- HTML aquí -->
</template>

<style scoped>
/* Estilos aquí */
</style>
```

## 🔌 Integración con Backend

Ver documento detallado: `docs/BACKEND_INTEGRATION.md`

### Pasos para conectar con backend real:

1. **Actualizar endpoints** en `src/api/endpoints.js`
2. **Configurar CORS** en el backend
3. **Implementar endpoints** según contratos en `docs/api-contracts/`
4. **Reemplazar mock data** por llamadas reales
5. **Configurar variables de entorno** (`.env`)

## 🚀 Comandos de Desarrollo

```bash
# Instalar dependencias
npm install

# Servidor de desarrollo
npm run dev

# Build para producción
npm run build

# Preview de build
npm run preview

# Linter
npm run lint
```

## 📝 Variables de Entorno

Crear archivo `.env` en la raíz:

```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_APP_NAME=Mentor Monitor
VITE_APP_VERSION=2.0.0
```

Acceder en el código:
```javascript
const apiUrl = import.meta.env.VITE_API_BASE_URL
```

## 🎨 Temas y Estilos

### TailwindCSS
Configuración personalizada en `tailwind.config.js`:
- Colores de marca
- Espaciados personalizados
- Breakpoints responsive
- Plugins adicionales

### Gradientes Corporativos
```css
/* Primario - Púrpura/Azul */
background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);

/* Éxito - Verde */
background: linear-gradient(135deg, #10b981 0%, #059669 100%);

/* Advertencia - Naranja */
background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);

/* Peligro - Rojo */
background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
```

## 📈 Próximos Pasos

1. ✅ Integración con backend real
2. ✅ Implementar más gráficos ECharts
3. ✅ Sistema de notificaciones en tiempo real (WebSockets)
4. ✅ Exportación de reportes (PDF, Excel)
5. ✅ Modo oscuro completo
6. ✅ Internacionalización (i18n)
7. ✅ Tests unitarios e integración
8. ✅ PWA (Progressive Web App)

## 📞 Contacto

Para más información sobre la arquitectura o integración, contactar al equipo de desarrollo frontend.

---

**Última actualización:** Noviembre 2025  
**Versión del template:** 2.0
