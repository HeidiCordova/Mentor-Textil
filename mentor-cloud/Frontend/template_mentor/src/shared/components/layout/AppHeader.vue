<script setup>
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'

const uiStore = useUiStore()
const authStore = useAuthStore()
const themeStore = useThemeStore()
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <button class="header-toggle" @click="uiStore.toggleSidebar">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
          <path d="M3 12h18M3 6h18M3 18h18" stroke="currentColor" stroke-width="2"/>
        </svg>
      </button>
      <h1 class="header-title">{{ $route.meta.title || 'Mentor Monitor' }}</h1>
    </div>

    <div class="header-right">
      <button class="theme-toggle" @click="themeStore.toggle" :title="themeStore.isDark ? 'Modo claro' : 'Modo oscuro'">
        <svg v-if="themeStore.isDark" width="20" height="20" viewBox="0 0 24 24" fill="none">
          <circle cx="12" cy="12" r="5" stroke="currentColor" stroke-width="2"/>
          <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
        <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none">
          <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      <div class="header-user">
        <div class="user-avatar">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
            <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2" stroke="currentColor" stroke-width="2"/>
            <circle cx="12" cy="7" r="4" stroke="currentColor" stroke-width="2"/>
          </svg>
        </div>
        <div class="user-info">
          <span class="user-name">{{ authStore.user?.nombre || 'Usuario' }}</span>
          <span class="user-role">{{ authStore.user?.rol || 'ROL' }}</span>
        </div>
      </div>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  @apply flex items-center justify-between px-6 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700;
  height: var(--header-height);
}

.header-left {
  @apply flex items-center gap-4;
}

.header-toggle {
  @apply p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors;
  @apply text-gray-600 dark:text-gray-300;
}

.header-title {
  @apply text-xl font-semibold text-gray-900 dark:text-gray-100;
}

.header-right {
  @apply flex items-center gap-4;
}

.theme-toggle {
  @apply p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors;
  @apply text-gray-600 dark:text-gray-300;
}

.header-user {
  @apply flex items-center gap-3 px-3 py-2 rounded-lg;
  @apply hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors;
}

.user-avatar {
  @apply w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900 text-primary-600 dark:text-primary-300;
  @apply flex items-center justify-center;
}

.user-info {
  @apply flex flex-col;
}

.user-name {
  @apply text-sm font-medium text-gray-900 dark:text-gray-100;
}

.user-role {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

@media (max-width: 1024px) {
  .header-title {
    @apply text-lg;
  }
}

@media (max-width: 768px) {
  .header-title {
    @apply text-base;
  }

  .user-info {
    @apply hidden;
  }

  .app-header {
    @apply px-4;
  }
}
</style>
