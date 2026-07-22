import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Dashboard from './views/Dashboard.vue'
import DevicesHome from './views/DevicesHome.vue'
import Config from './views/Config.vue'
import Health from './views/Health.vue'
import Hardware from './views/Hardware.vue'
import ProvisionLine from './views/ProvisionLine.vue'
import DeviceRegistry from './views/DeviceRegistry.vue'
import TiempoRealLocal from './views/TiempoRealLocal.vue'
import Usuarios from './views/Usuarios.vue'
import './style.css'

const routes = [
  { path: '/', component: Dashboard },
  { path: '/config', component: DevicesHome },
  { path: '/config/:lineaId', component: Config },
  { path: '/health', component: Health },
  { path: '/produccion/:lineaId?', redirect: to => `/config/${to.params.lineaId || ''}` },
  { path: '/hardware', component: Hardware },
  { path: '/provision', component: ProvisionLine },
  { path: '/registry', component: DeviceRegistry },
  { path: '/tiempo-real', component: TiempoRealLocal },
  { path: '/usuarios', component: Usuarios },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

createApp(App).use(router).mount('#app')
