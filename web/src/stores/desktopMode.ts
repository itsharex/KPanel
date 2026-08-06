import { computed, reactive, readonly } from 'vue'
import {
  cascadePosition,
  clampToViewport,
  normalizeGeometry,
  type ViewportSize,
  type WindowGeometry,
} from '@/lib/desktopWindowGeometry'

/**
 * Desktop (Windows-style) mode state.
 *
 * The desktop is a pure front-end overlay: it reuses the existing lazy-loaded
 * route views as window contents and keeps all state (mode, open windows and
 * their geometry) in browser storage. It introduces no backend surface.
 */

export type DesktopMode = 'classic' | 'desktop'

export interface DesktopWindowState {
  id: number
  /** Navigation route path rendered inside this window. */
  path: string
  titleKey: string
  geometry: WindowGeometry
  minimized: boolean
  maximized: boolean
  z: number
  /** Focused window id (0 = none). */
  focusedId: number
}

interface DesktopState {
  mode: DesktopMode
  windows: DesktopWindowState[]
  focusedId: number
}

export interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}

const MODE_KEY = 'kejilion-panel-desktop-mode'
const WINDOWS_KEY = 'kejilion-panel-desktop-windows'
/** Bounded persistence: never grow browser storage without limit. */
const MAX_PERSISTED_WINDOWS = 12
const MAX_WINDOWS = 8

const state = reactive<DesktopState>({
  mode: 'classic',
  windows: [],
  focusedId: 0,
})

let nextWindowId = 1
let nextZ = 1

function isDesktopMode(value: unknown): value is DesktopMode {
  return value === 'classic' || value === 'desktop'
}

function isWindowRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function readPersistedMode(storage: StorageLike | undefined): DesktopMode {
  try {
    const raw = storage?.getItem(MODE_KEY) ?? null
    return isDesktopMode(raw) ? raw : 'classic'
  } catch {
    return 'classic'
  }
}

function readPersistedWindows(
  storage: StorageLike | undefined,
  viewport: ViewportSize,
): DesktopWindowState[] {
  try {
    const raw = storage?.getItem(WINDOWS_KEY) ?? null
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter(isWindowRecord)
      .slice(0, MAX_PERSISTED_WINDOWS)
      .map((record) => {
        const id = typeof record.id === 'number' ? record.id : nextWindowId++
        const path = typeof record.path === 'string' ? record.path : ''
        const titleKey = typeof record.titleKey === 'string' ? record.titleKey : ''
        if (!path || !titleKey) return null
        const geometry = normalizeGeometry(
          isWindowRecord(record.geometry) ? (record.geometry as Partial<WindowGeometry>) : undefined,
          viewport,
        )
        return {
          id,
          path,
          titleKey,
          geometry,
          minimized: Boolean(record.minimized),
          maximized: Boolean(record.maximized),
          z: 0,
          focusedId: id,
        }
      })
      .filter((entry): entry is DesktopWindowState => entry !== null)
  } catch {
    return []
  }
}

function persistWindows(storage: StorageLike | undefined): void {
  try {
    const snapshot = state.windows
      .slice(0, MAX_PERSISTED_WINDOWS)
      .map((windowState) => ({
        id: windowState.id,
        path: windowState.path,
        titleKey: windowState.titleKey,
        geometry: windowState.geometry,
        minimized: windowState.minimized,
        maximized: windowState.maximized,
      }))
    storage?.setItem(WINDOWS_KEY, JSON.stringify(snapshot))
  } catch {
    // Storage full/unavailable: desktop keeps working for the session.
  }
}

function persistMode(storage: StorageLike | undefined): void {
  try {
    storage?.setItem(MODE_KEY, state.mode)
  } catch {
    // Same as above: non-fatal.
  }
}

function bringToFront(id: number): void {
  const target = state.windows.find((windowState) => windowState.id === id)
  if (!target) return
  nextZ += 1
  target.z = nextZ
  state.focusedId = id
}

function resizeForViewport(viewport: ViewportSize): void {
  for (const windowState of state.windows) {
    windowState.geometry = clampToViewport(windowState.geometry, viewport)
  }
}

function defaultViewport(): ViewportSize {
  return { width: window.innerWidth, height: window.innerHeight }
}

function defaultStorage(): StorageLike | undefined {
  try {
    return window.localStorage
  } catch {
    return undefined
  }
}

/** Initialize from persistence. Called once on app boot. */
export function initializeDesktopMode(
  storage: StorageLike | undefined = defaultStorage(),
  viewport: ViewportSize = defaultViewport(),
): void {
  state.mode = readPersistedMode(storage)
  const restored = readPersistedWindows(storage, viewport)
  if (restored.length) {
    state.windows.splice(0, state.windows.length, ...restored)
    // Reassign z values in insertion order so the last restored window is on top.
    let z = 1
    for (const windowState of state.windows) {
      windowState.z = z
      windowState.focusedId = windowState.id
      z += 1
    }
    nextZ = z
    nextWindowId = Math.max(nextWindowId, restored.reduce((max, w) => Math.max(max, w.id), 0) + 1)
    state.focusedId = state.windows[state.windows.length - 1]?.id ?? 0
    persistWindows(storage)
  }
}

export function useDesktopMode() {
  const windows = computed(() => state.windows)

  function enterDesktop(): void {
    if (state.mode === 'desktop') return
    state.mode = 'desktop'
    resizeForViewport(defaultViewport())
    persistMode(defaultStorage())
  }

  function enterClassic(): void {
    if (state.mode === 'classic') return
    state.mode = 'classic'
    persistMode(defaultStorage())
  }

  function toggleMode(): void {
    if (state.mode === 'desktop') enterClassic()
    else enterDesktop()
  }

  /**
   * Open a window for a route path. `allowMultiple` governs whether the same
   * path may spawn more than one window; single-instance paths (terminal) focus
   * the existing window instead.
   */
  function openWindow(path: string, titleKey: string, allowMultiple: boolean): number {
    if (!allowMultiple) {
      const existing = state.windows.find((windowState) => windowState.path === path)
      if (existing) {
        existing.minimized = false
        bringToFront(existing.id)
        persistWindows(defaultStorage())
        return existing.id
      }
    }
    const id = nextWindowId++
    nextZ += 1
    const geometry = cascadeGeometry(id)
    state.windows.push({
      id,
      path,
      titleKey,
      geometry,
      minimized: false,
      maximized: false,
      z: nextZ,
      focusedId: id,
    })
    state.focusedId = id
    trimToWindowLimit()
    persistWindows(defaultStorage())
    return id
  }

  function cascadeGeometry(id: number): WindowGeometry {
    const index = state.windows.filter((windowState) => windowState.id !== id).length
    return cascadePosition(index, defaultViewport())
  }

  function closeWindow(id: number): void {
    const index = state.windows.findIndex((windowState) => windowState.id === id)
    if (index === -1) return
    state.windows.splice(index, 1)
    if (state.focusedId === id) {
      const top = [...state.windows].sort((a, b) => b.z - a.z)[0]
      state.focusedId = top?.id ?? 0
    }
    persistWindows(defaultStorage())
  }

  function minimizeWindow(id: number): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target) return
    target.minimized = true
    if (state.focusedId === id) {
      const top = [...state.windows]
        .filter((windowState) => windowState.id !== id && !windowState.minimized)
        .sort((a, b) => b.z - a.z)[0]
      state.focusedId = top?.id ?? 0
    }
    persistWindows(defaultStorage())
  }

  function restoreWindow(id: number): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target) return
    target.minimized = false
    bringToFront(id)
    persistWindows(defaultStorage())
  }

  function toggleMinimized(id: number): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target) return
    if (target.minimized) restoreWindow(id)
    else minimizeWindow(id)
  }

  function toggleMaximize(id: number): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target) return
    target.maximized = !target.maximized
    bringToFront(id)
    persistWindows(defaultStorage())
  }

  function focusWindow(id: number): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target || target.minimized) return
    bringToFront(id)
    persistWindows(defaultStorage())
  }

  function updateGeometry(id: number, geometry: WindowGeometry): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target) return
    target.geometry = clampToViewport(geometry, defaultViewport())
    persistWindows(defaultStorage())
  }

  function trimToWindowLimit(): void {
    if (state.windows.length <= MAX_WINDOWS) return
    const ordered = [...state.windows].sort((a, b) => b.z - a.z)
    const keep = ordered.slice(0, MAX_WINDOWS).map((windowState) => windowState.id)
    for (const windowState of [...state.windows]) {
      if (!keep.includes(windowState.id)) closeWindow(windowState.id)
    }
  }

  return {
    state: readonly(state),
    windows,
    focusedId: computed(() => state.focusedId),
    mode: computed(() => state.mode),
    enterDesktop,
    enterClassic,
    toggleMode,
    openWindow,
    closeWindow,
    minimizeWindow,
    restoreWindow,
    toggleMinimized,
    toggleMaximize,
    focusWindow,
    updateGeometry,
    resizeForViewport,
  }
}

export function resetDesktopModeForTest(): void {
  state.mode = 'classic'
  state.windows.splice(0, state.windows.length)
  state.focusedId = 0
  nextWindowId = 1
  nextZ = 1
}
