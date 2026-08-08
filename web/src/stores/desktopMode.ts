import { computed, reactive, readonly } from 'vue'
import {
  cascadePosition,
  clampToViewport,
  normalizeGeometry,
  type ViewportSize,
  type WindowGeometry,
} from '@/lib/desktopWindowGeometry'
import { canonicalDesktopAppPath, desktopRoutePath, findDesktopApp } from '@/lib/desktopApps'

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
const MAX_WINDOWS = 8
const BASE_WINDOW_Z = 100
const MAX_WINDOW_Z = 900

const state = reactive<DesktopState>({
  mode: 'classic',
  windows: [],
  focusedId: 0,
})

let nextWindowId = 1
let nextZ = BASE_WINDOW_Z

function isDesktopMode(value: unknown): value is DesktopMode {
  return value === 'classic' || value === 'desktop'
}

function isWindowRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function titleKeyForWindowPath(path: string): string | undefined {
  path = desktopRoutePath(path)
  if (path === '/monitoring') return 'route.monitoring'
  if (path === '/processes') return 'route.processes'
  if (path === '/sites/environment') return 'route.environment'
  if (path.startsWith('/ai/s/') && !/^\/ai\/s\/[A-Za-z0-9_-]{1,128}$/.test(path)) return undefined
  return findDesktopApp(path)?.labelKey
}

function isSafeWindowFullPath(fullPath: string): boolean {
  return (
    fullPath.startsWith('/')
    && !fullPath.startsWith('//')
    && fullPath.length <= 768
    && !/[\u0000-\u001f\\]/.test(fullPath)
    && Boolean(titleKeyForWindowPath(fullPath))
  )
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
    if (raw.length > 64_000) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    const restored: DesktopWindowState[] = []
    const usedIds = new Set<number>()
    const allocateId = (): number => {
      while (usedIds.has(nextWindowId)) nextWindowId += 1
      const id = nextWindowId
      nextWindowId += 1
      return id
    }
    for (const value of parsed.filter(isWindowRecord).slice(0, MAX_WINDOWS)) {
      const path = typeof value.path === 'string' ? value.path : ''
      const titleKey = titleKeyForWindowPath(path)
      if (!titleKey || !isSafeWindowFullPath(path)) continue

      const persistedId = value.id
      const canReuseId = Number.isSafeInteger(persistedId) && Number(persistedId) > 0 && !usedIds.has(Number(persistedId))
      const id = canReuseId ? Number(persistedId) : allocateId()
      usedIds.add(id)
      restored.push({
        id,
        path,
        titleKey,
        geometry: normalizeGeometry(
          isWindowRecord(value.geometry) ? (value.geometry as Partial<WindowGeometry>) : undefined,
          viewport,
        ),
        minimized: Boolean(value.minimized),
        maximized: Boolean(value.maximized),
        z: BASE_WINDOW_Z,
        focusedId: id,
      })
    }
    return restored
  } catch {
    return []
  }
}

function persistWindows(storage: StorageLike | undefined): void {
  try {
    const snapshot = state.windows
      .slice(0, MAX_WINDOWS)
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
  if (nextZ >= MAX_WINDOW_Z) normalizeWindowStack()
  nextZ += 1
  target.z = nextZ
  state.focusedId = id
}

function normalizeWindowStack(): void {
  const ordered = [...state.windows].sort((a, b) => a.z - b.z)
  let z = BASE_WINDOW_Z
  for (const windowState of ordered) {
    windowState.z = z
    z += 1
  }
  nextZ = z
}

function resizeForViewport(viewport: ViewportSize, persist = true): void {
  for (const windowState of state.windows) {
    windowState.geometry = clampToViewport(windowState.geometry, viewport)
  }
  if (persist) persistWindows(defaultStorage())
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
    let z = BASE_WINDOW_Z
    for (const windowState of state.windows) {
      windowState.z = z
      windowState.focusedId = windowState.id
      z += 1
    }
    nextZ = z
    nextWindowId = Math.max(nextWindowId, restored.reduce((max, w) => Math.max(max, w.id), 0) + 1)
    state.focusedId = [...state.windows]
      .filter((windowState) => !windowState.minimized)
      .sort((a, b) => b.z - a.z)[0]?.id ?? 0
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
   * the existing window instead. Returns 0 when the bounded window limit is
   * reached; callers surface that state rather than silently closing user work.
   */
  function openWindow(path: string, titleKey: string, allowMultiple: boolean, navigateExisting = false): number {
    if (!allowMultiple) {
      const appPath = canonicalDesktopAppPath(path)
      const existing = state.windows.find(
        (windowState) => canonicalDesktopAppPath(windowState.path) === appPath,
      )
      if (existing) {
        const safeTitleKey = titleKeyForWindowPath(path)
        if (navigateExisting && safeTitleKey && isSafeWindowFullPath(path)) {
          existing.path = path
          existing.titleKey = safeTitleKey
        }
        existing.minimized = false
        bringToFront(existing.id)
        persistWindows(defaultStorage())
        return existing.id
      }
    }
    if (state.windows.length >= MAX_WINDOWS) return 0
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
      const top = [...state.windows]
        .filter((windowState) => !windowState.minimized)
        .sort((a, b) => b.z - a.z)[0]
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
    if (state.focusedId === id) return
    bringToFront(id)
  }

  function updateGeometry(id: number, geometry: WindowGeometry, persist = true): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    if (!target) return
    target.geometry = clampToViewport(geometry, defaultViewport())
    if (persist) persistWindows(defaultStorage())
  }

  function commitGeometry(id: number): void {
    if (!state.windows.some((windowState) => windowState.id === id)) return
    persistWindows(defaultStorage())
  }

  function updateWindowRoute(id: number, path: string, titleKey: string): void {
    const target = state.windows.find((windowState) => windowState.id === id)
    const safeTitleKey = titleKeyForWindowPath(path)
    if (!target || !isSafeWindowFullPath(path) || !safeTitleKey || safeTitleKey !== titleKey) return
    if (target.path === path && target.titleKey === safeTitleKey) return
    target.path = path
    target.titleKey = safeTitleKey
    persistWindows(defaultStorage())
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
    commitGeometry,
    updateWindowRoute,
    resizeForViewport,
  }
}

export function resetDesktopModeForTest(): void {
  state.mode = 'classic'
  state.windows.splice(0, state.windows.length)
  state.focusedId = 0
  nextWindowId = 1
  nextZ = BASE_WINDOW_Z
}
