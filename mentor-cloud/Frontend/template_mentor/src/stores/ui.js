import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  const sidebarCollapsed = ref(false)
  const isTablet = ref(window.innerWidth >= 768 && window.innerWidth < 1024)
  const isMobile = ref(window.innerWidth < 768)
  const sidebarOpen = ref(!isMobile.value)
  const loading = ref(false)
  const modalOpen = ref(false)

  function toggleSidebar() {
    if (isMobile.value) {
      sidebarOpen.value = !sidebarOpen.value
    } else {
      sidebarCollapsed.value = !sidebarCollapsed.value
    }
  }

  function closeSidebar() {
    if (isMobile.value) {
      sidebarOpen.value = false
    }
  }

  function openSidebar() {
    sidebarOpen.value = true
  }

  function setLoading(value) {
    loading.value = value
  }

  function openModal() {
    modalOpen.value = true
  }

  function closeModal() {
    modalOpen.value = false
  }

  function updateMobile() {
    const w = window.innerWidth
    isMobile.value = w < 768
    isTablet.value = w >= 768 && w < 1024
    if (!isMobile.value) {
      sidebarOpen.value = true
    }
    if (isTablet.value && !sidebarCollapsed.value) {
      sidebarCollapsed.value = true
    }
  }

  return {
    sidebarCollapsed,
    isTablet,
    isMobile,
    sidebarOpen,
    loading,
    modalOpen,
    toggleSidebar,
    closeSidebar,
    openSidebar,
    setLoading,
    openModal,
    closeModal,
    updateMobile
  }
})
