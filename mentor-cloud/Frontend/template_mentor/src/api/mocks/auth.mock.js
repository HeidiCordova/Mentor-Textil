export const mockAuth = {
  login: (credentials) => {
    return new Promise((resolve) => {
      setTimeout(() => {
        if (credentials.username === 'admin' && credentials.password === 'admin') {
          resolve({
            token: 'mock-jwt-token-12345',
            refreshToken: 'mock-refresh-token-67890',
            user: {
              id: 1,
              username: 'admin',
              email: 'admin@mentormonitor.com',
              nombre: 'Administrador',
              rol: 'ADMIN'
            }
          })
        } else {
          resolve({ error: 'Credenciales inválidas' })
        }
      }, 500)
    })
  },

  logout: () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({ success: true })
      }, 200)
    })
  },

  refresh: () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          token: 'mock-jwt-token-new-12345',
          refreshToken: 'mock-refresh-token-new-67890'
        })
      }, 300)
    })
  },

  getCurrentUser: () => {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          id: 1,
          username: 'admin',
          email: 'admin@mentormonitor.com',
          nombre: 'Administrador',
          rol: 'ADMIN'
        })
      }, 200)
    })
  }
}
