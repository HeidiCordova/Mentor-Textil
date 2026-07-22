const mockPlantsData = [
  { id: 1, nombre: 'ISM Arequipa', compania: 'Industrias San Miguel', lineas: 5, activo: true },
  { id: 2, nombre: 'Histórico Embotelladora San Miguel', compania: 'Industrias San Miguel', lineas: 8, activo: true },
  { id: 3, nombre: 'PLC MODICON M221', compania: 'Industrias San Miguel', lineas: 3, activo: true }
]

export const mockPlants = {
  getAll: (params) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          data: mockPlantsData,
          total: mockPlantsData.length,
          page: params.page || 1,
          perPage: params.perPage || 10
        })
      }, 300)
    })
  },

  getById: (id) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const plant = mockPlantsData.find(p => p.id === parseInt(id))
        resolve(plant || null)
      }, 200)
    })
  },

  create: (data) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const newPlant = {
          id: mockPlantsData.length + 1,
          ...data,
          activo: true
        }
        mockPlantsData.push(newPlant)
        resolve(newPlant)
      }, 400)
    })
  },

  update: (id, data) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const index = mockPlantsData.findIndex(p => p.id === parseInt(id))
        if (index !== -1) {
          mockPlantsData[index] = { ...mockPlantsData[index], ...data }
          resolve(mockPlantsData[index])
        } else {
          resolve(null)
        }
      }, 400)
    })
  },

  delete: (id) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const index = mockPlantsData.findIndex(p => p.id === parseInt(id))
        if (index !== -1) {
          mockPlantsData.splice(index, 1)
          resolve({ success: true })
        } else {
          resolve({ success: false })
        }
      }, 300)
    })
  }
}
