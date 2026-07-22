export const mockAnalysis = {
  getGeneral: (params) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const data = []
        const startDate = new Date('2025-11-13')
        
        for (let i = 0; i < 100; i++) {
          const date = new Date(startDate)
          date.setHours(date.getHours() + i)
          
          data.push({
            timestamp: date.toISOString(),
            value: Math.random() * 5000 + 2000,
            status: Math.random() > 0.8 ? 'alarm' : 'normal'
          })
        }
        
        resolve({ data })
      }, 500)
    })
  },

  getEnergy: (params) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const data = []
        const startDate = new Date('2025-11-13')
        
        for (let i = 0; i < 50; i++) {
          const date = new Date(startDate)
          date.setHours(date.getHours() + i)
          
          data.push({
            timestamp: date.toISOString(),
            consumption: Math.random() * 300 + 100,
            power: Math.random() * 500 + 200
          })
        }
        
        resolve({ data })
      }, 500)
    })
  },

  getProduction: (params) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const data = []
        const startDate = new Date('2025-11-13')
        
        for (let i = 0; i < 80; i++) {
          const date = new Date(startDate)
          date.setHours(date.getHours() + i)
          
          data.push({
            timestamp: date.toISOString(),
            produced: Math.floor(Math.random() * 1000 + 500),
            target: 800,
            efficiency: Math.random() * 30 + 70
          })
        }
        
        resolve({ data })
      }, 500)
    })
  },

  getPareto: (params) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          data: [
            { categoria: 'Falla Mecánica', frecuencia: 45, porcentaje: 35 },
            { categoria: 'Falta de Material', frecuencia: 32, porcentaje: 25 },
            { categoria: 'Mantenimiento', frecuencia: 28, porcentaje: 22 },
            { categoria: 'Cambio de Producto', frecuencia: 15, porcentaje: 12 },
            { categoria: 'Otros', frecuencia: 8, porcentaje: 6 }
          ]
        })
      }, 400)
    })
  }
}
