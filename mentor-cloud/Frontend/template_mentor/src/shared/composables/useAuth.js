import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'

export function useAuth() {
  const authStore = useAuthStore()

  const isAuthenticated = computed(() => authStore.isAuthenticated)
  const user = computed(() => authStore.user)
  const userRole = computed(() => authStore.userRole)

  async function login(credentials) {
    return await authStore.login(credentials)
  }

  async function logout() {
    await authStore.logout()
  }

  function hasRole(roles) {
    return authStore.hasRole(roles)
  }

  function hasPermission(permission) {
    return true
  }

  return {
    isAuthenticated,
    user,
    userRole,
    login,
    logout,
    hasRole,
    hasPermission
  }
}
