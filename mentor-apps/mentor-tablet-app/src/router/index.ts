import { createRouter, createWebHistory } from 'vue-router'
import { useConnectionStore } from '@/stores/connection'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/login'
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue')
    },
    {
      path: '/device',
      name: 'device',
      component: () => import('@/views/DeviceSelectView.vue')
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/stops',
      name: 'stops',
      component: () => import('@/views/StopsView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/config',
      name: 'config',
      component: () => import('@/views/ConfigView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/status',
      name: 'status',
      component: () => import('@/views/StatusView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/produccion',
      name: 'produccion',
      component: () => import('@/views/ProduccionView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/historial',
      name: 'historial',
      component: () => import('@/views/HistorialView.vue'),
      meta: { requiresAuth: true }
    }
  ]
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth) {
    const connection = useConnectionStore()
    if (!connection.authenticated) {
      return { name: 'login' }
    }
  }
})

export default router
