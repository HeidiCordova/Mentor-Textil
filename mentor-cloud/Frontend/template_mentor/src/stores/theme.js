import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(localStorage.getItem('theme') === 'dark')

  function applyTheme(dark) {
    if (dark) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  }

  function toggle() {
    isDark.value = !isDark.value
  }

  watch(isDark, (val) => applyTheme(val), { immediate: true })

  return { isDark, toggle }
})
