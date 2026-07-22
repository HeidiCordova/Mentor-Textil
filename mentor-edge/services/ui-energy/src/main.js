import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Dashboard from './views/Dashboard.vue'
import Config from './views/Config.vue'
import MeterWrite from './views/MeterWrite.vue'
import Audit from './views/Audit.vue'
import Readings from './views/Readings.vue'
import Monitor from './views/Monitor.vue'
import Manual from './views/Manual.vue'
import './style.css'

const routes = [
  { path: '/',          component: Dashboard },
  { path: '/monitor',   component: Monitor },
  { path: '/lecturas',  component: Readings },
  { path: '/config',    component: Config },
  { path: '/programar', component: MeterWrite },
  { path: '/auditoria', component: Audit },
  { path: '/manual',    component: Manual },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

createApp(App).use(router).mount('#app')
