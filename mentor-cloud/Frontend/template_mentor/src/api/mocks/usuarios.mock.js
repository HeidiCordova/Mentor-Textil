const mockUsersData = [
  { id: 1, username: 'admin', email: 'admin@mentormonitor.com', nombre: 'Administrador', rol: 'ADMIN', activo: true },
  { id: 2, username: 'operador1', email: 'operador1@ism.com', nombre: 'Juan Pérez', rol: 'OPERATOR', activo: true },
  { id: 3, username: 'supervisor1', email: 'supervisor1@ism.com', nombre: 'María García', rol: 'SUPERVISOR', activo: true }
]

export const mockUsers = {
  getAll: (params) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          data: mockUsersData,
          total: mockUsersData.length,
          page: params.page || 1,
          perPage: params.perPage || 10
        })
      }, 300)
    })
  },

  getById: (id) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const user = mockUsersData.find(u => u.id === parseInt(id))
        resolve(user || null)
      }, 200)
    })
  },

  create: (data) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const newUser = {
          id: mockUsersData.length + 1,
          ...data,
          activo: true
        }
        mockUsersData.push(newUser)
        resolve(newUser)
      }, 400)
    })
  },

  update: (id, data) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const index = mockUsersData.findIndex(u => u.id === parseInt(id))
        if (index !== -1) {
          mockUsersData[index] = { ...mockUsersData[index], ...data }
          resolve(mockUsersData[index])
        } else {
          resolve(null)
        }
      }, 400)
    })
  },

  delete: (id) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        const index = mockUsersData.findIndex(u => u.id === parseInt(id))
        if (index !== -1) {
          mockUsersData.splice(index, 1)
          resolve({ success: true })
        } else {
          resolve({ success: false })
        }
      }, 300)
    })
  }
}
