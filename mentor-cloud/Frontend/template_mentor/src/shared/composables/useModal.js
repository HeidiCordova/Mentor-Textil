import { ref } from 'vue'

export function useModal() {
  const isOpen = ref(false)
  const data = ref(null)

  function open(initialData = null) {
    data.value = initialData
    isOpen.value = true
  }

  function close() {
    isOpen.value = false
    setTimeout(() => {
      data.value = null
    }, 300)
  }

  function toggle() {
    if (isOpen.value) {
      close()
    } else {
      open()
    }
  }

  return {
    isOpen,
    data,
    open,
    close,
    toggle
  }
}
