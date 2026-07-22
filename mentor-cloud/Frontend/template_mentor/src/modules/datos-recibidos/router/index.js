export default [
  {
    path: '/datos-recibidos',
    name: 'DatosRecibidos',
    component: () => import('../views/DatosRecibidosView.vue'),
    meta: {
      title: 'Datos Recibidos',
      icon: 'Database'
    }
  }
]
