export const mockDashboard = {
  getStats: () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          totalPlantas: 12,
          totalLineas: 48,
          totalDispositivos: 156,
          alarmasActivas: 3,
          eficienciaPromedio: 87.5,
          produccionHoy: 12450
        })
      }, 300)
    })
  },

  getReports: () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          data: [
            { fecha: '2025-11-11', valor: 8500, tipo: 'Programada' },
            { fecha: '2025-11-12', valor: 7200, tipo: 'No Programada' },
            { fecha: '2025-11-13', valor: 9100, tipo: 'Programada' },
            { fecha: '2025-11-14', valor: 8800, tipo: 'No Programada' },
            { fecha: '2025-11-15', valor: 9500, tipo: 'Programada' },
            { fecha: '2025-11-16', valor: 8200, tipo: 'Programada' },
            { fecha: '2025-11-17', valor: 7800, tipo: 'No Programada' },
            { fecha: '2025-11-18', valor: 9200, tipo: 'Programada' }
          ]
        })
      }, 400)
    })
  },

  getCharts: () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          produccion: [
            { mes: 'Enero', valor: 45000 },
            { mes: 'Febrero', valor: 42000 },
            { mes: 'Marzo', valor: 48000 },
            { mes: 'Abril', valor: 46000 },
            { mes: 'Mayo', valor: 50000 },
            { mes: 'Junio', valor: 47000 }
          ],
          energia: [
            { hora: '00:00', consumo: 120 },
            { hora: '04:00', consumo: 95 },
            { hora: '08:00', consumo: 180 },
            { hora: '12:00', consumo: 210 },
            { hora: '16:00', consumo: 195 },
            { hora: '20:00', consumo: 140 }
          ]
        })
      }, 350)
    })
  }
}
