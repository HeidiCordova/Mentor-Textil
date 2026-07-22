<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterView } from 'vue-router'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import { useUiStore } from '@/stores/ui'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

const uiStore = useUiStore()

onMounted(() => {
  window.addEventListener('resize', uiStore.updateMobile)
  uiStore.updateMobile()
})

onUnmounted(() => {
  window.removeEventListener('resize', uiStore.updateMobile)
})
</script>

<template>
  <div class="app-layout">
    <AppSidebar />
    <div class="main-container" :class="{ 'sidebar-collapsed': uiStore.sidebarCollapsed && !uiStore.isMobile }">
      <AppHeader />
      <main class="main-content">
        <RouterView />
      </main>
    </div>
    <div
      v-if="uiStore.sidebarOpen && uiStore.isMobile"
      class="sidebar-overlay"
      @click="uiStore.closeSidebar"
    ></div>
  </div>
</template>

<style scoped>
.app-layout {
  @apply flex min-h-screen bg-gray-100 dark:bg-gray-950;
}

.main-container {
  @apply flex-1 flex flex-col;
  margin-left: var(--sidebar-width);
  transition: margin-left 0.3s ease;
}

.main-container.sidebar-collapsed {
  margin-left: var(--sidebar-collapsed-width);
}

.main-content {
  @apply flex-1 overflow-auto bg-gray-100 dark:bg-gray-950;
  padding: clamp(1rem, 2vw, 1.5rem);
}

.sidebar-overlay {
  @apply fixed inset-0 bg-black bg-opacity-50 z-40;
}

@media (max-width: 768px) {
  .main-container {
    margin-left: 0;
  }

  .main-container.sidebar-collapsed {
    margin-left: 0;
  }
}
</style>
