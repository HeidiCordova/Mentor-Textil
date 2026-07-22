let companies = [
  {
    id: 1,
    nombre: 'Coca-Cola',
    ruc: '20123456789',
    direccion: 'Av. Industrial Norte 897',
    telefono: '997712345',
    email: 'contacto@cocacola.com',
    responsable: 'Carlos Ramírez',
    estado: true,
    creado: '2025-01-15T10:30:00'
  },
  {
    id: 2,
    nombre: 'Alicorp',
    ruc: '20987654321',
    direccion: 'Jr. Lima 456, Callao',
    telefono: '998823456',
    email: 'info@alicorp.com',
    responsable: 'María González',
    estado: true,
    creado: '2025-02-20T14:20:00'
  },
  {
    id: 3,
    nombre: 'Gloria S.A.',
    ruc: '20456789123',
    direccion: 'Av. República de Panamá 2461',
    telefono: '999934567',
    email: 'contacto@gloria.com.pe',
    responsable: 'Juan Pérez',
    estado: true,
    creado: '2025-03-10T09:15:00'
  }
]

let nextId = 4

export default {
  getAll() {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({ data: companies })
      }, 300)
    })
  },

  getById(id) {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        const company = companies.find(c => c.id === parseInt(id))
        if (company) {
          resolve({ data: company })
        } else {
          reject({ message: 'Empresa no encontrada' })
        }
      }, 200)
    })
  },

  create(data) {
    return new Promise((resolve) => {
      setTimeout(() => {
        const newCompany = {
          id: nextId++,
          ...data,
          creado: new Date().toISOString()
        }
        companies.push(newCompany)
        resolve({ data: newCompany })
      }, 400)
    })
  },

  update(id, data) {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        const index = companies.findIndex(c => c.id === parseInt(id))
        if (index !== -1) {
          companies[index] = { ...companies[index], ...data }
          resolve({ data: companies[index] })
        } else {
          reject({ message: 'Empresa no encontrada' })
        }
      }, 400)
    })
  },

  delete(id) {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        const index = companies.findIndex(c => c.id === parseInt(id))
        if (index !== -1) {
          companies.splice(index, 1)
          resolve({ data: { message: 'Empresa eliminada correctamente' } })
        } else {
          reject({ message: 'Empresa no encontrada' })
        }
      }, 300)
    })
  }
}
