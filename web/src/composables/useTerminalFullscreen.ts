import { nextTick, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

const bodyClass = 'terminal-fullscreen-open'

export function useTerminalFullscreen(
  target: Ref<HTMLElement | undefined>,
  refreshLayout: () => void,
) {
  const fullscreen = ref(false)
  const fallbackFullscreen = ref(false)
  let nativeFullscreen = false
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

  async function enterFullscreen(): Promise<void> {
    const element = target.value
    if (!element || fullscreen.value) return
    if (typeof element.requestFullscreen === 'function') {
      try {
        await element.requestFullscreen()
        if (document.fullscreenElement === element) {
          nativeFullscreen = true
          fallbackFullscreen.value = false
          setFullscreenState(true)
          return
        }
      } catch {
        // iOS, embedded browsers and permission policies may reject native fullscreen.
      }
    }
    nativeFullscreen = false
    fallbackFullscreen.value = true
    setFullscreenState(true)
  }

  async function exitFullscreen(): Promise<void> {
    if (!fullscreen.value) return
    if (
      nativeFullscreen &&
      document.fullscreenElement === target.value &&
      typeof document.exitFullscreen === 'function'
    ) {
      try {
        await document.exitFullscreen()
      } catch {
        // Always restore the viewport fallback state even if the browser rejects exit.
      }
    }
    nativeFullscreen = false
    fallbackFullscreen.value = false
    setFullscreenState(false)
  }

  function toggleFullscreen(): void {
    void (fullscreen.value ? exitFullscreen() : enterFullscreen())
  }

  function handleFullscreenChange(): void {
    const ownsNativeFullscreen = document.fullscreenElement === target.value
    if (ownsNativeFullscreen) {
      nativeFullscreen = true
      fallbackFullscreen.value = false
      setFullscreenState(true)
    } else if (nativeFullscreen) {
      nativeFullscreen = false
      setFullscreenState(false)
    }
  }

  function handleEscape(event: KeyboardEvent): void {
    if (event.key !== 'Escape' || !fullscreen.value) return
    event.preventDefault()
    event.stopImmediatePropagation()
    void exitFullscreen()
  }

  onMounted(() => {
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    document.addEventListener('keydown', handleEscape, true)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('fullscreenchange', handleFullscreenChange)
    document.removeEventListener('keydown', handleEscape, true)
    if (nativeFullscreen && document.fullscreenElement === target.value) {
      void document.exitFullscreen?.().catch(() => undefined)
    }
    nativeFullscreen = false
    fallbackFullscreen.value = false
    if (bodyLocked) {
      document.documentElement.classList.remove(bodyClass)
      document.body.classList.remove(bodyClass)
    }
  })

  return { fullscreen, fallbackFullscreen, toggleFullscreen, exitFullscreen }
}
