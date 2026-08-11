// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { rememberOverviewViewport, restoreOverviewViewport } from './overviewViewport'

describe('overview viewport restoration', () => {
  it('restores the saved scroll position inside the same desktop window', () => {
    const body = document.createElement('div')
    body.className = 'desktop-window__body'
    const page = document.createElement('main')
    body.appendChild(page)
    body.scrollTop = 914

    rememberOverviewViewport(page)
    body.scrollTop = 0

    expect(restoreOverviewViewport(page)).toBe('desktop')
    expect(body.scrollTop).toBe(914)
    expect(restoreOverviewViewport(page)).toBe(false)
  })

  it('keeps the system section at the same viewport offset when preceding content changes height', () => {
    const body = document.createElement('div')
    body.className = 'desktop-window__body'
    const page = document.createElement('main')
    const section = document.createElement('section')
    section.className = 'overview-system-management'
    page.appendChild(section)
    body.appendChild(page)
    body.getBoundingClientRect = () => ({ top: 100 }) as DOMRect
    section.getBoundingClientRect = () => ({ top: 340 }) as DOMRect
    body.scrollTop = 914

    rememberOverviewViewport(page)
    section.getBoundingClientRect = () => ({ top: 390 }) as DOMRect
    body.scrollTop = 0

    expect(restoreOverviewViewport(page)).toBe('desktop')
    expect(body.scrollTop).toBe(964)
  })

  it('restores classic mode after async content has rendered', () => {
    const page = document.createElement('main')
    const section = document.createElement('section')
    section.className = 'overview-system-management'
    page.appendChild(section)
    Object.defineProperty(window, 'scrollY', { value: 720, configurable: true })
    window.scrollTo = vi.fn()

    rememberOverviewViewport(page)

    expect(restoreOverviewViewport(page)).toBe('classic')
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 720 })
  })
})
