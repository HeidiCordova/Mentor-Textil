import { ref, onUnmounted } from 'vue'

type ThrottledFn = (...args: unknown[]) => void

export function useThrottle(fn: ThrottledFn, intervalMs: number) {
  let lastCall = 0
  let pendingTimer: ReturnType<typeof setTimeout> | null = null
  const pending = ref(false)

  function throttled(...args: unknown[]): void {
    const now = Date.now()
    const elapsed = now - lastCall

    if (elapsed >= intervalMs) {
      lastCall = now
      pending.value = false
      fn(...args)
    } else if (!pendingTimer) {
      pending.value = true
      pendingTimer = setTimeout(() => {
        lastCall = Date.now()
        pending.value = false
        pendingTimer = null
        fn(...args)
      }, intervalMs - elapsed)
    }
  }

  function cancel(): void {
    if (pendingTimer) {
      clearTimeout(pendingTimer)
      pendingTimer = null
      pending.value = false
    }
  }

  onUnmounted(cancel)

  return { throttled, cancel, pending }
}
