import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

const bodyClass = 'terminal-fullscreen-open'

export function useTerminalFullscreen(
  refreshLayout: () => void,
) {
  const fullscreen = ref(false)
  let bodyLocked = false

  function refreshAfterLayout(): void {
    void nextTick(() => window.requestAnimationFrame(refreshLayout))
  }

  function setFullscreenState(enabled: boolean): void {
    fullscreen.value = enabled
    if (enabled && !bodyLocked) {
      document.documentElement.classList.add(bodyClass)
      document.body.classList.add(bodyClass)
      bodyLocked = true
    } else if (!enabled && bodyLocked) {
      document.documentElement.classList.remove(bodyClass)
      document.body.classList.remove(bodyClass)
      bodyLocked = false
    }
    refreshAfterLayout()
  }

  function enterFullscreen(): void {
    if (fullscreen.value) return
    // This intentionally fills only the webpage viewport. Never enter browser/OS fullscreen.
    setFullscreenState(true)
  }

  function exitFullscreen(): void {
    if (!fullscreen.value) return
    setFullscreenState(false)
  }

  function toggleFullscreen(): void {
    if (fullscreen.value) exitFullscreen()
    else enterFullscreen()
  }

  function handleEscape(event: KeyboardEvent): void {
    if (event.key !== 'Escape' || !fullscreen.value) return
    event.preventDefault()
    event.stopImmediatePropagation()
    exitFullscreen()
  }

  onMounted(() => {
    document.addEventListener('keydown', handleEscape, true)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', handleEscape, true)
    if (bodyLocked) {
      document.documentElement.classList.remove(bodyClass)
      document.body.classList.remove(bodyClass)
    }
  })

  return { fullscreen, toggleFullscreen, exitFullscreen }
}
