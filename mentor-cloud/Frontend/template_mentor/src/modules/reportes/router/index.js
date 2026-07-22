export default [
  {
    path: '/reportes',
    name: 'Reportes',
    component: () => import('../views/ReportesView.vue'),
    meta: {
      title: 'Reportes',
      icon: 'FileText'
    }
  }
]
