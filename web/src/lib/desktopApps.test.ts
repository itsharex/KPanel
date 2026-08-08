import { describe, expect, it } from 'vitest'
import { DEFAULT_WINDOW_GRADIENT, desktopApps, findDesktopApp } from './desktopApps'

describe('desktop app catalogue', () => {
  it('mirrors the classic navigation set', () => {
    const paths = desktopApps.map((app) => app.path)
    expect(paths).toEqual(
      expect.arrayContaining([
        '/overview',
        '/ai',
        '/sites',
        '/apps',
        '/docker',
        '/files',
        '/terminal',
        '/diagnostics',
        '/cluster',
        '/activity',
        '/settings',
      ]),
    )
    expect(desktopApps).toHaveLength(11)
  })

  it('gives every app a distinct gradient', () => {
    const gradients = desktopApps.map((app) => app.gradient.join('→'))
    const unique = new Set(gradients)
    expect(unique.size).toBe(desktopApps.length)
  })

  it('marks the terminal as single-instance', () => {
    expect(findDesktopApp('/terminal')?.allowMultiple).toBe(false)
    expect(findDesktopApp('/overview')?.allowMultiple).toBe(false)
  })

  it('maps safe dynamic script routes to a terminal window without adding a desktop launcher', () => {
    expect(findDesktopApp('/app-script/openclaw')?.labelKey).toBe('desktop.scriptWindowTitle')
    expect(findDesktopApp('/app-script/openclaw')?.allowMultiple).toBe(false)
    expect(findDesktopApp('/app-script/bad/path')).toBeUndefined()
    expect(desktopApps.map((app) => app.path)).not.toContain('/app-script')
  })

  it('returns undefined for unknown paths', () => {
    expect(findDesktopApp('/nope')).toBeUndefined()
  })

  it('provides a default gradient fallback', () => {
    expect(DEFAULT_WINDOW_GRADIENT).toHaveLength(2)
    expect(DEFAULT_WINDOW_GRADIENT[0]).toMatch(/^#/)
  })
})
