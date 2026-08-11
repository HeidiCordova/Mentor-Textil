import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/dashboard'
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/modules/auth/views/LoginView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/unauthorized',
      name: 'Unauthorized',
      component: () => import('@/shared/views/UnauthorizedView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      component: () => import('@/shared/components/layout/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/modules/dashboard/views/DashboardView.vue'),
          meta: { title: 'Dashboard' }
        },
        {
          path: 'configuracion',
          name: 'Configuracion',
          redirect: '/configuracion/empresa',
          children: [
            {
              path: 'empresa',
              name: 'Empresa',
              component: () => import('@/modules/configuracion/views/EmpresaView.vue'),
              meta: { title: 'Empresa' }
            },
            {
              path: 'usuario',
              name: 'Usuario',
              component: () => import('@/modules/configuracion/views/UsuarioView.vue'),
              meta: { title: 'Usuario' }
            },
            {
              path: 'roles',
              name: 'Roles',
              component: () => import('@/modules/configuracion/views/RolesView.vue'),
              meta: { title: 'Roles' }
            },
            {
              path: 'archivos',
              name: 'Archivos',
              component: () => import('@/modules/configuracion/views/ArchivosView.vue'),
              meta: { title: 'Archivos' }
            },
            {
              path: 'mantenimiento',
              name: 'Monitoreo',
              component: () => import('@/modules/configuracion/views/MantenimientoView.vue'),
              meta: { title: 'Monitoreo' }
            },
            {
              path: 'variables',
              name: 'Variables',
              component: () => import('@/modules/configuracion/views/VariablesView.vue'),
              meta: { title: 'Variables' }
            },
            {
              path: 'arbol-paradas',
              name: 'ArbolParadas',
              component: () => import('@/modules/configuracion/views/ArbolParadasView.vue'),
              meta: { title: 'Árbol de Paradas' }
            },

            {
              path: 'dispositivos',
              name: 'Dispositivos',
              component: () => import('@/modules/configuracion/views/DispositivosView.vue'),
              meta: { title: 'Dispositivos' }
            }
          ]
        },
        {
          path: 'administracion',
          name: 'Administracion',
          redirect: '/administracion/tipo-documento',
          children: [
            {
              path: 'tipo-documento',
              name: 'TipoDocumento',
              component: () => import('@/modules/administracion/views/TipoDocumentoView.vue'),
              meta: { title: 'Tipo de Documento' }
            },
            {
              path: 'calendarizacion',
              name: 'Calendarizacion',
              component: () => import('@/modules/administracion/views/CalendarizacionView.vue'),
              meta: { title: 'Calendarización' }
            },
            {
              path: 'turnos',
              name: 'Turnos',
              component: () => import('@/modules/administracion/views/TurnosView.vue'),
              meta: { title: 'Turnos' }
            },
            {
              path: 'productos',
              name: 'Productos',
              component: () => import('@/modules/administracion/views/ProductosView.vue'),
              meta: { title: 'Productos' }
            },
            {
              path: 'canvas-oee',
              name: 'CanvasOEE',
              component: () => import('@/modules/administracion/views/CanvasOEEView.vue'),
              meta: { title: 'Canvas OEE' }
            },
            {
              path: 'laboratorio-oee',
              name: 'LaboratorioOEE',
              component: () => import('@/modules/administracion/views/LaboratorioOEEView.vue'),
              meta: { title: 'Laboratorio OEE' }
            },
            {
              path: 'velocidad-nominal',
              name: 'VelocidadNominal',
              component: () => import('@/modules/administracion/views/VelocidadNominalView.vue'),
              meta: { title: 'Velocidad Nominal' }
            },
            {
              path: 'habilitar-linea',
              name: 'HabilitarLinea',
              component: () => import('@/modules/administracion/views/HabilitarLineaView.vue'),
              meta: { title: 'Habilitar Línea' }
            }
          ]
        },
        {
          path: 'analisis',
          name: 'Analisis',
          redirect: '/analisis/general',
          children: [
            {
              path: 'general',
              name: 'AnalisisGeneral',
              component: () => import('@/modules/analisis/views/GeneralView.vue'),
              meta: { title: 'Análisis General' }
            },
            {
              path: 'energia',
              name: 'AnalisisEnergia',
              component: () => import('@/modules/analisis/views/EnergiaView.vue'),
              meta: { title: 'Análisis Energía' }
            },
            {
              path: 'produccion',
              name: 'AnalisisProduccion',
              component: () => import('@/modules/analisis/views/ProduccionView.vue'),
              meta: { title: 'Análisis Producción' }
            }
          ]
        },
        {
          path: 'vista-rapida',
          name: 'VistaRapida',
          redirect: '/vista-rapida/factor-calificacion',
          children: [
            {
              path: 'factor-calificacion',
              name: 'FactorCalificacion',
              component: () => import('@/modules/vista-rapida/views/FactorCalificacionView.vue'),
              meta: { title: 'Factor de Calificación' }
            },
            {
              path: 'oee-turno',
              name: 'OeeTurno',
              component: () => import('@/modules/vista-rapida/views/OeeTurnoView.vue'),
              meta: { title: 'OEE por Turno' }
            }
          ]
        },
        {
          path: 'analisis-avanzado',
          name: 'AnalisisAvanzado',
          redirect: '/analisis-avanzado/generador-consultas',
          children: [
            {
              path: 'generador-consultas',
              name: 'GeneradorConsultas',
              component: () => import('@/modules/analisis-avanzado/views/GeneradorConsultasView.vue'),
              meta: { title: 'Generador de Consultas - R' }
            }
          ]
        },
        {
          path: 'analisis-energia',
          name: 'AnalisisEnergia',
          redirect: '/analisis-energia/medidores',
          children: [
            {
              path: 'control-edge',
              name: 'ControlEdgeEnergia',
              component: () => import('@/modules/energia/views/ControlEdgeEnergiaView.vue'),
              meta: { title: 'Control Edge Energia' }
            },
            {
              path: 'medidores',
              name: 'Medidores',
              component: () => import('@/modules/energia/views/MedidoresView.vue'),
              meta: { title: 'Medidores de Energia' }
            },
            {
              path: 'monitor',
              name: 'MonitorEnergia',
              component: () => import('@/modules/energia/views/MonitorEnergiaView.vue'),
              meta: { title: 'Monitor Energetico' }
            },
            {
              path: 'consumo-electrico-tarifario',
              name: 'ConsumoElectricoTarifario',
              component: () => import('@/modules/analisis-energia/views/ConsumoElectricoTarifarioView.vue'),
              meta: { title: 'Consumo Electrico Tarifario' }
            },
            {
              path: 'factor-calificacion',
              name: 'FactorCalificacionEnergia',
              component: () => import('@/modules/analisis-energia/views/FactorCalificacionView.vue'),
              meta: { title: 'Factor de Calificacion' }
            }
          ]
        },
        {
          path: 'analisis-produccion',
          name: 'AnalisisProduccion',
          redirect: '/analisis-produccion/linea-tiempo',
          children: [
            {
              path: 'linea-tiempo',
              name: 'LineaTiempo',
              component: () => import('@/modules/analisis-produccion/views/LineaTiempoView.vue'),
              meta: { title: 'Línea de Tiempo' }
            },
            {
              path: 'historia-linea',
              name: 'HistoriaLinea',
              component: () => import('@/modules/analisis-produccion/views/HistoriaLineaView.vue'),
              meta: { title: 'Historia de Línea' }
            },
            {
              path: 'grafica-oee',
              name: 'GraficaOEE',
              component: () => import('@/modules/analisis-produccion/views/GraficaOEEView.vue'),
              meta: { title: 'Gráfica de OEE' }
            },
            {
              path: 'grafica-pareto',
              name: 'GraficaPareto',
              component: () => import('@/modules/analisis-produccion/views/GraficaParetoView.vue'),
              meta: { title: 'Gráfica de Pareto' }
            },
            {
              path: 'tiempo-real',
              name: 'TiempoReal',
              component: () => import('@/modules/analisis-produccion/views/TiempoRealView.vue'),
              meta: { title: 'Tiempo Real' }
            },
            {
              path: 'monitor-oee',
              name: 'MonitorOEE',
              component: () => import('@/modules/analisis-produccion/views/MonitorOEEView.vue'),
              meta: { title: 'Monitor OEE' }
            }
          ]
        },
        {
          path: 'reportes',
          name: 'Reportes',
          component: () => import('@/modules/reportes/views/ReportesView.vue'),
          meta: { title: 'Reportes' }
        },
        {
          path: 'alarmas',
          name: 'Alarmas',
          component: () => import('@/modules/alarmas/views/AlarmasView.vue'),
          meta: { title: 'Alarmas' }
        },
        {
          path: 'datos-recibidos',
          name: 'DatosRecibidos',
          component: () => import('@/modules/datos-recibidos/views/DatosRecibidosView.vue'),
          meta: { title: 'Datos Recibidos' }
        },
        {
          path: 'compromisos',
          name: 'Compromisos',
          component: () => import('@/modules/compromisos/views/CompromisosView.vue'),
          meta: { title: 'Compromisos' }
        },
        {
          path: 'integracion',
          name: 'Integracion',
          component: () => import('@/modules/integracion/views/IntegracionView.vue'),
          meta: { title: 'API de Integración' }
        },
        {
          path: 'docs',
          name: 'Docs',
          component: () => import('@/modules/docs/views/DocsView.vue'),
          meta: { title: 'Documentacion' }
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'NotFound',
      component: () => import('@/shared/views/NotFoundView.vue'),
      meta: { requiresAuth: false }
    }
  ]
})

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth !== false)
  
  if (requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
