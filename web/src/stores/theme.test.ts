// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import {
  DEFAULT_THEME_COLORS,
  THEME_TOKEN_NAMES,
  deriveThemeTokens,
  serializeThemeColors,
  type ThemeColorIntent,
  type ThemeTokenMap,
} from '@/theme/colors'
import { initializeTheme, resetThemeForTest, useTheme } from './theme'

const MODE_STORAGE_KEY = 'kejilion-panel-theme'
const COLOR_STORAGE_KEY = 'kejilion-panel-colors'
const TRANSITION_CLASS = 'desktop-theme-transitioning'
const themeStoreSource = readFileSync(resolve(process.cwd(), 'src/stores/theme.ts'), 'utf8')

const CUSTOM_COLORS: ThemeColorIntent = {
  brand: '#315d7d',
  neutral: '#657483',
  signatureLinked: false,
  signature: '#b28c54',
}

function createMediaQuery(matches = false) {
  const listeners = new Set<(event: MediaQueryListEvent) => void>()
  const mediaQuery = {
    matches,
    media: '(prefers-color-scheme: dark)',
    onchange: null,
    addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
      listeners.add(listener)
    }),
    removeEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
      listeners.delete(listener)
    }),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  } as unknown as MediaQueryList

  return {
    mediaQuery,
    emit(next: boolean) {
      Object.defineProperty(mediaQuery, 'matches', { configurable: true, value: next })
      for (const listener of listeners) listener({ matches: next } as MediaQueryListEvent)
    },
  }
}

function installMediaQuery(mediaQuery: MediaQueryList): void {
  vi.stubGlobal('matchMedia', vi.fn(() => mediaQuery))
}

function dispatchStorage(key: string | null, newValue: string | null): void {
  window.dispatchEvent(new StorageEvent('storage', { key, newValue }))
}

function expectAppliedTokens(tokens: ThemeTokenMap): void {
  for (const token of THEME_TOKEN_NAMES) {
    expect(document.documentElement.style.getPropertyValue(token), token).toBe(tokens[token])
  }
}

function expectNoAppliedTokens(): void {
  for (const token of THEME_TOKEN_NAMES) {
    expect(document.documentElement.style.getPropertyValue(token), token).toBe('')
  }
}

beforeEach(() => {
  window.localStorage.clear()
  resetThemeForTest()
})

afterEach(() => {
  resetThemeForTest()
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('theme mode and custom color persistence', () => {
  it('restores the legacy mode key but defaults to CSS colors without inline tokens or color storage', () => {
    window.localStorage.setItem(MODE_STORAGE_KEY, 'dark')
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)

    initializeTheme()
    const theme = useTheme()

    expect(theme.preference.value).toBe('dark')
    expect(theme.resolved.value).toBe('dark')
    expect(theme.colors.value).toEqual(DEFAULT_THEME_COLORS)
    expect(theme.isCustom.value).toBe(false)
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.dataset.skin).toBeUndefined()
    expect(document.documentElement.style.colorScheme).toBe('dark')
    expect(window.localStorage.getItem(MODE_STORAGE_KEY)).toBe('dark')
    expect(window.localStorage.getItem(COLOR_STORAGE_KEY)).toBeNull()
    expectNoAppliedTokens()
  })

  it('restores a valid versioned color record and applies only the derived token allowlist', () => {
    const serialized = serializeThemeColors(CUSTOM_COLORS)
    window.localStorage.setItem(COLOR_STORAGE_KEY, serialized)
    window.localStorage.setItem(MODE_STORAGE_KEY, 'light')
    const media = createMediaQuery(true)
    installMediaQuery(media.mediaQuery)

    initializeTheme()
    const theme = useTheme()

    expect(theme.colors.value).toEqual(CUSTOM_COLORS)
    expect(theme.isCustom.value).toBe(true)
    expect(theme.resolved.value).toBe('light')
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'light'))
    expect(window.localStorage.getItem(COLOR_STORAGE_KEY)).toBe(serialized)
    expect(document.documentElement.dataset.skin).toBeUndefined()
  })

  it('cleans damaged color JSON and falls back to the stylesheet defaults', () => {
    window.localStorage.setItem(COLOR_STORAGE_KEY, '{"version":1,"brand":"var(--injected)"}')
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)

    initializeTheme()

    expect(useTheme().colors.value).toEqual(DEFAULT_THEME_COLORS)
    expect(useTheme().isCustom.value).toBe(false)
    expect(window.localStorage.getItem(COLOR_STORAGE_KEY)).toBeNull()
    expectNoAppliedTokens()
  })

  it('has no preset skin contract in the store, DOM projection, or storage access', () => {
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)
    const getItem = vi.spyOn(Storage.prototype, 'getItem')
    const setItem = vi.spyOn(Storage.prototype, 'setItem')
    const removeItem = vi.spyOn(Storage.prototype, 'removeItem')

    initializeTheme()
    const theme = useTheme()
    theme.setColors(CUSTOM_COLORS)
    theme.setTheme('dark')
    theme.resetColors()

    expect(themeStoreSource).not.toMatch(/skin|vip|graphite/i)
    for (const spy of [getItem, setItem, removeItem]) {
      expect(spy.mock.calls.some(([key]) => String(key).includes('skin'))).toBe(false)
    }
    expect(document.documentElement.dataset.skin).toBeUndefined()
    expectNoAppliedTokens()
  })

  it('persists mode and colors independently while keeping the color record atomic', () => {
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)
    initializeTheme()
    const theme = useTheme()

    theme.setColors(CUSTOM_COLORS)
    const storedColors = window.localStorage.getItem(COLOR_STORAGE_KEY)
    expect(JSON.parse(storedColors ?? '')).toEqual({ version: 1, ...CUSTOM_COLORS })
    expect(window.localStorage.getItem(MODE_STORAGE_KEY)).toBeNull()

    theme.setTheme('dark')
    expect(window.localStorage.getItem(MODE_STORAGE_KEY)).toBe('dark')
    expect(window.localStorage.getItem(COLOR_STORAGE_KEY)).toBe(storedColors)
    expect(theme.colors.value).toEqual(CUSTOM_COLORS)
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))
  })

  it('re-derives custom tokens when the followed system mode changes', () => {
    window.localStorage.setItem(COLOR_STORAGE_KEY, serializeThemeColors(CUSTOM_COLORS))
    const media = createMediaQuery(false)
    installMediaQuery(media.mediaQuery)
    initializeTheme()

    expect(useTheme().resolved.value).toBe('light')
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'light'))

    media.emit(true)

    expect(useTheme().resolved.value).toBe('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))
  })
})

describe('cross-tab synchronization and resilience', () => {
  it('synchronizes versioned colors and mode, then treats a null color value as a reset', () => {
    const media = createMediaQuery(false)
    installMediaQuery(media.mediaQuery)
    initializeTheme()

    dispatchStorage(COLOR_STORAGE_KEY, serializeThemeColors(CUSTOM_COLORS))
    dispatchStorage(MODE_STORAGE_KEY, 'dark')

    expect(useTheme().colors.value).toEqual(CUSTOM_COLORS)
    expect(useTheme().isCustom.value).toBe(true)
    expect(useTheme().preference.value).toBe('dark')
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))

    dispatchStorage(COLOR_STORAGE_KEY, null)

    expect(useTheme().colors.value).toEqual(DEFAULT_THEME_COLORS)
    expect(useTheme().isCustom.value).toBe(false)
    expect(useTheme().preference.value).toBe('dark')
    expectNoAppliedTokens()
  })

  it('resets both independent dimensions when another tab clears storage', () => {
    window.localStorage.setItem(MODE_STORAGE_KEY, 'dark')
    window.localStorage.setItem(COLOR_STORAGE_KEY, serializeThemeColors(CUSTOM_COLORS))
    const media = createMediaQuery(false)
    installMediaQuery(media.mediaQuery)
    initializeTheme()

    dispatchStorage(null, null)

    expect(useTheme().preference.value).toBe('system')
    expect(useTheme().resolved.value).toBe('light')
    expect(useTheme().colors.value).toEqual(DEFAULT_THEME_COLORS)
    expect(useTheme().isCustom.value).toBe(false)
    expect(document.documentElement.dataset.theme).toBe('light')
    expectNoAppliedTokens()
  })

  it('keeps storage synchronization active when matchMedia is unavailable', () => {
    vi.stubGlobal('matchMedia', undefined)

    initializeTheme()
    dispatchStorage(COLOR_STORAGE_KEY, serializeThemeColors(CUSTOM_COLORS))
    dispatchStorage(MODE_STORAGE_KEY, 'dark')

    expect(useTheme().isCustom.value).toBe(true)
    expect(useTheme().preference.value).toBe('dark')
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))
  })

  it('keeps in-memory theming usable when browser storage is unavailable', () => {
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('blocked')
    })
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new DOMException('blocked')
    })

    expect(() => initializeTheme()).not.toThrow()
    const theme = useTheme()
    expect(() => theme.setColors(CUSTOM_COLORS)).not.toThrow()
    expect(() => theme.setTheme('dark')).not.toThrow()

    expect(theme.colors.value).toEqual(CUSTOM_COLORS)
    expect(theme.isCustom.value).toBe(true)
    expect(theme.preference.value).toBe('dark')
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))

    expect(() => theme.resetColors()).not.toThrow()
    expect(theme.isCustom.value).toBe(false)
    expectNoAppliedTokens()
  })

  it('reinitializes without leaving duplicate media or storage listeners', () => {
    const first = createMediaQuery()
    const second = createMediaQuery(true)
    const matchMedia = vi.fn()
      .mockReturnValueOnce(first.mediaQuery)
      .mockReturnValueOnce(second.mediaQuery)
    vi.stubGlobal('matchMedia', matchMedia)
    const addWindowListener = vi.spyOn(window, 'addEventListener')

    initializeTheme()
    initializeTheme()

    expect(first.mediaQuery.removeEventListener).toHaveBeenCalledOnce()
    expect(second.mediaQuery.addEventListener).toHaveBeenCalledOnce()
    expect(addWindowListener.mock.calls.filter(([type]) => type === 'storage')).toHaveLength(1)
    expect(document.documentElement.dataset.theme).toBe('dark')

    first.emit(false)
    expect(document.documentElement.dataset.theme).toBe('dark')
    second.emit(false)
    expect(document.documentElement.dataset.theme).toBe('light')
  })
})

describe('application lifecycle', () => {
  it('does not animate initialization but gives color and mode changes one shared desktop transition', () => {
    vi.useFakeTimers()
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)

    initializeTheme()
    const theme = useTheme()
    expect(document.documentElement.classList.contains(TRANSITION_CLASS)).toBe(false)

    theme.setColors(CUSTOM_COLORS)
    expect(document.documentElement.classList.contains(TRANSITION_CLASS)).toBe(true)
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'light'))

    vi.advanceTimersByTime(300)
    theme.setTheme('dark')
    expect(document.documentElement.classList.contains(TRANSITION_CLASS)).toBe(true)
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))

    vi.advanceTimersByTime(459)
    expect(document.documentElement.classList.contains(TRANSITION_CLASS)).toBe(true)
    vi.advanceTimersByTime(1)
    expect(document.documentElement.classList.contains(TRANSITION_CLASS)).toBe(false)
  })

  it('resetColors removes every runtime token and the versioned color record without changing mode', () => {
    const media = createMediaQuery()
    installMediaQuery(media.mediaQuery)
    initializeTheme()
    const theme = useTheme()

    theme.setTheme('dark')
    theme.setColors(CUSTOM_COLORS)
    expectAppliedTokens(deriveThemeTokens(CUSTOM_COLORS, 'dark'))
    expect(window.localStorage.getItem(COLOR_STORAGE_KEY)).not.toBeNull()

    theme.resetColors()

    expect(theme.preference.value).toBe('dark')
    expect(theme.colors.value).toEqual(DEFAULT_THEME_COLORS)
    expect(theme.isCustom.value).toBe(false)
    expect(window.localStorage.getItem(COLOR_STORAGE_KEY)).toBeNull()
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.dataset.skin).toBeUndefined()
    expectNoAppliedTokens()
  })
})
