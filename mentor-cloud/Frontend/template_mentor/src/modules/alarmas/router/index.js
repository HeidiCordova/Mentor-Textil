export default [
  {
    path: '/alarmas',
    name: 'Alarmas',
    component: () => import('../views/AlarmasView.vue'),
    meta: {
      title: 'Alarmas',
      icon: 'Bell'
    }
  }
]
