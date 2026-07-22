import axios from 'axios'
import { API_BASE_URL, API_TIMEOUT } from './endpoints'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

const client = axios.create({
  baseURL: API_BASE_URL,
  timeout: API_TIMEOUT,
  headers: {
    'Content-Type': 'application/json'
  }
})

client.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore()
    const token = authStore.token
    
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

client.interceptors.response.use(
  (response) => {
    return response.data
  },
  async (error) => {
    const originalRequest = error.config
    
    if (error.response?.status === 401 && !originalRequest._retry) {
      // No reintentar rutas de autenticación para evitar bucle infinito
      const isAuthRoute = originalRequest.url?.includes('/auth/')
      if (isAuthRoute) {
        const authStore = useAuthStore()
        authStore.token = null
        authStore.refreshToken = null
        authStore.user = null
        localStorage.removeItem('token')
        localStorage.removeItem('refreshToken')
        localStorage.removeItem('user')
        window.location.replace('/login')
        return Promise.reject(error)
      }

      originalRequest._retry = true
      
      const authStore = useAuthStore()
      const refreshed = await authStore.doRefreshToken()
      
      if (refreshed) {
        return client(originalRequest)
      } else {
        authStore.token = null
        authStore.refreshToken = null
        authStore.user = null
        localStorage.removeItem('token')
        localStorage.removeItem('refreshToken')
        localStorage.removeItem('user')
        window.location.replace('/login')
      }
    }
    
    if (error.response?.status === 403) {
      router.push('/unauthorized')
    }
    
    return Promise.reject(error)
  }
)

export default client
