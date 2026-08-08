// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { initializeDesktopMode, resetDesktopModeForTest, useDesktopMode } from './desktopMode'

function makeStorage(): { store: Map<string, string>; getItem: (k: string) => string | null; setItem: (k: string, v: string) => void } {
  const store = new Map<string, string>()
  return {
    store,
    getItem: (key) => store.get(key) ?? null,
    setItem: (key, value) => {
      store.set(key, value)
    },
  }
}

function setupViewport(width: number, height: number): void {
  Object.defineProperty(window, 'innerWidth', { value: width, configurable: true })
  Object.defineProperty(window, 'innerHeight', { value: height, configurable: true })
}

describe('desktop mode', () => {
  beforeEach(() => {
    resetDesktopModeForTest()
  })

  it('starts in classic mode', () => {
    const desktop = useDesktopMode()
    expect(desktop.mode.value).toBe('classic')
    expect(desktop.windows.value).toHaveLength(0)
  })

  it('enters and leaves desktop mode', () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    expect(desktop.mode.value).toBe('desktop')
    desktop.enterClassic()
    expect(desktop.mode.value).toBe('classic')
  })

  it('opens a window with the requested route and title', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/overview', 'route.overview', true)
    const windowState = desktop.windows.value.find((item) => item.id === id)
    expect(windowState).toBeDefined()
    expect(windowState?.path).toBe('/overview')
    expect(windowState?.titleKey).toBe('route.overview')
    expect(windowState?.minimized).toBe(false)
    expect(desktop.focusedId.value).toBe(id)
  })

  it('focuses the existing window for single-instance paths instead of opening a new one', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const first = desktop.openWindow('/terminal', 'route.terminal', false)
    const second = desktop.openWindow('/terminal', 'route.terminal', false)
    expect(second).toBe(first)
    expect(desktop.windows.value.filter((item) => item.path === '/terminal')).toHaveLength(1)
  })

  it('allows multiple windows for multi-instance paths', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const first = desktop.openWindow('/files', 'route.files', true)
    const second = desktop.openWindow('/files', 'route.files', true)
    expect(second).not.toBe(first)
    expect(desktop.windows.value.filter((item) => item.path === '/files')).toHaveLength(2)
  })

  it('minimizes and restores a window, moving focus to the top sibling', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const first = desktop.openWindow('/overview', 'route.overview', true)
    const second = desktop.openWindow('/files', 'route.files', true)
    desktop.minimizeWindow(second)
    expect(desktop.windows.value.find((item) => item.id === second)?.minimized).toBe(true)
    expect(desktop.focusedId.value).toBe(first)
    desktop.restoreWindow(second)
    expect(desktop.windows.value.find((item) => item.id === second)?.minimized).toBe(false)
    expect(desktop.focusedId.value).toBe(second)
  })

  it('brings the focused window to the front', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const first = desktop.openWindow('/overview', 'route.overview', true)
    const second = desktop.openWindow('/files', 'route.files', true)
    const firstZ = desktop.windows.value.find((item) => item.id === first)!.z
    const secondZ = desktop.windows.value.find((item) => item.id === second)!.z
    expect(secondZ).toBeGreaterThan(firstZ)
    desktop.focusWindow(first)
    expect(desktop.windows.value.find((item) => item.id === first)!.z).toBeGreaterThan(secondZ)
  })

  it('closes a window and reassigns focus', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const first = desktop.openWindow('/overview', 'route.overview', true)
    desktop.openWindow('/files', 'route.files', true)
    desktop.closeWindow(first)
    expect(desktop.windows.value.some((item) => item.id === first)).toBe(false)
    expect(desktop.focusedId.value).not.toBe(first)
  })

  it('updates window geometry', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/overview', 'route.overview', true)
    desktop.updateGeometry(id, { left: 40, top: 40, width: 900, height: 600 })
    expect(desktop.windows.value.find((item) => item.id === id)?.geometry).toMatchObject({ left: 40, top: 40, width: 900, height: 600 })
  })

  it('caps the number of open windows to the fixed limit', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    for (let index = 0; index < 8; index += 1) {
      expect(desktop.openWindow('/ai', 'route.ai', true)).toBeGreaterThan(0)
    }
    expect(desktop.openWindow('/ai', 'route.ai', true)).toBe(0)
    expect(desktop.windows.value).toHaveLength(8)
  })

  it('navigates an existing single-instance window when a cross-app handoff supplies a full path', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const first = desktop.openWindow('/files?path=/home', 'route.files', false)
    const second = desktop.openWindow('/files?path=/home/web/site', 'route.files', false, true)

    expect(second).toBe(first)
    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]?.path).toBe('/files?path=/home/web/site')
  })

  it('restores persisted mode on initialize', () => {
    const storage = makeStorage()
    storage.store.set('kejilion-panel-desktop-mode', 'desktop')
    initializeDesktopMode(storage, { width: 1280, height: 800 })
    expect(useDesktopMode().mode.value).toBe('desktop')
    resetDesktopModeForTest()
  })

  it('restores persisted windows with normalized geometry', () => {
    const storage = makeStorage()
    storage.store.set(
      'kejilion-panel-desktop-windows',
      JSON.stringify([
        { id: 7, path: '/overview', titleKey: 'route.overview', geometry: { left: 30, top: 40, width: 800, height: 500 }, minimized: false, maximized: false },
        { id: 8, path: '/files', titleKey: 'route.files', geometry: { left: 9999, top: 9999, width: 900, height: 600 }, minimized: false, maximized: false },
      ]),
    )
    initializeDesktopMode(storage, { width: 1280, height: 800 })
    const desktop = useDesktopMode()
    expect(desktop.windows.value).toHaveLength(2)
    expect(desktop.windows.value[0]!.path).toBe('/overview')
    // Offscreen window is clamped back into the viewport.
    expect(desktop.windows.value[1]!.geometry.left + desktop.windows.value[1]!.geometry.width).toBeLessThanOrEqual(1280)
    expect(desktop.focusedId.value).toBe(8)
    resetDesktopModeForTest()
  })

  it('ignores malformed persisted windows', () => {
    const storage = makeStorage()
    storage.store.set('kejilion-panel-desktop-windows', '{not json')
    initializeDesktopMode(storage, { width: 1280, height: 800 })
    expect(useDesktopMode().windows.value).toHaveLength(0)
    resetDesktopModeForTest()
  })

  it('rejects unknown persisted routes and derives trusted titles from the app catalogue', () => {
    const storage = makeStorage()
    storage.store.set('kejilion-panel-desktop-windows', JSON.stringify([
      { id: 1, path: '/not-an-app', titleKey: 'route.settings', geometry: {}, minimized: false },
      { id: 2, path: '/files?path=/home', titleKey: '<script>', geometry: {}, minimized: false },
    ]))

    initializeDesktopMode(storage, { width: 1280, height: 800 })
    const windows = useDesktopMode().windows.value
    expect(windows).toHaveLength(1)
    expect(windows[0]?.path).toBe('/files?path=/home')
    expect(windows[0]?.titleKey).toBe('route.files')
  })

  it('rejects oversized persisted window payloads before parsing', () => {
    const storage = makeStorage()
    storage.store.set('kejilion-panel-desktop-windows', `[${' '.repeat(64_001)}]`)

    initializeDesktopMode(storage, { width: 1280, height: 800 })
    expect(useDesktopMode().windows.value).toHaveLength(0)
  })

  it('persists and restores a bounded full path including its query', () => {
    window.localStorage.clear()
    setupViewport(1280, 800)
    initializeDesktopMode(window.localStorage, { width: 1280, height: 800 })
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/files', 'route.files', false)
    desktop.updateWindowRoute(id, '/files?path=/home/web/site', 'route.files')
    expect(window.localStorage.getItem('kejilion-panel-desktop-windows')).toContain('/files?path=/home/web/site')

    resetDesktopModeForTest()
    initializeDesktopMode(window.localStorage, { width: 1280, height: 800 })
    expect(useDesktopMode().windows.value[0]?.path).toBe('/files?path=/home/web/site')
    window.localStorage.clear()
  })

  it('allocates unique ids when persisted ids are invalid or duplicated', () => {
    const storage = makeStorage()
    storage.store.set('kejilion-panel-desktop-windows', JSON.stringify([
      { id: 2, path: '/overview', geometry: {}, minimized: false },
      { id: 1, path: '/files', geometry: {}, minimized: false },
      { id: 1, path: '/terminal', geometry: {}, minimized: false },
    ]))

    initializeDesktopMode(storage, { width: 1280, height: 800 })
    const ids = useDesktopMode().windows.value.map((item) => item.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(ids.every((id) => id > 0)).toBe(true)
  })

  it('tolerates unavailable storage without throwing', () => {
    const brokenStorage = {
      getItem: () => {
        throw new Error('blocked')
      },
      setItem: () => {
        throw new Error('blocked')
      },
    }
    expect(() => initializeDesktopMode(brokenStorage, { width: 1280, height: 800 })).not.toThrow()
    const desktop = useDesktopMode()
    expect(() => desktop.openWindow('/overview', 'route.overview', true)).not.toThrow()
  })

  it('persists mode and windows to browser storage', () => {
    window.localStorage.clear()
    setupViewport(1280, 800)
    initializeDesktopMode(window.localStorage, { width: 1280, height: 800 })
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/overview', 'route.overview', true)
    expect(window.localStorage.getItem('kejilion-panel-desktop-mode')).toBe('desktop')
    expect(window.localStorage.getItem('kejilion-panel-desktop-windows')).toContain('/overview')
    window.localStorage.clear()
  })

  it('removes the focused id when the last window closes', () => {
    setupViewport(1280, 800)
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/overview', 'route.overview', true)
    desktop.closeWindow(id)
    expect(desktop.windows.value).toHaveLength(0)
    expect(desktop.focusedId.value).toBe(0)
  })
})
