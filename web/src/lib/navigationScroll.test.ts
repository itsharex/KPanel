import { describe, expect, it } from 'vitest'
import { resolveNavigationScroll } from './navigationScroll'

describe('resolveNavigationScroll', () => {
  it('restores the browser position during back and forward navigation', () => {
    const savedPosition = { left: 0, top: 864 }

    expect(resolveNavigationScroll('/monitoring', '/monitoring', savedPosition)).toBe(savedPosition)
  })

  it('keeps the current position for same-page query navigation', () => {
    expect(resolveNavigationScroll('/monitoring', '/monitoring', null)).toBe(false)
  })

  it('scrolls to the top when entering another page', () => {
    expect(resolveNavigationScroll('/monitoring', '/overview', null)).toEqual({ top: 0 })
  })
})
