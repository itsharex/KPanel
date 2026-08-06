import { describe, expect, it } from 'vitest'
import {
  clampToViewport,
  cascadePosition,
  normalizeGeometry,
  MIN_WINDOW_WIDTH,
  MIN_WINDOW_HEIGHT,
} from './desktopWindowGeometry'

describe('desktop window geometry', () => {
  it('clamps a window to minimum size', () => {
    const result = clampToViewport({ left: 0, top: 0, width: 100, height: 100 }, { width: 1280, height: 800 })
    expect(result.width).toBe(MIN_WINDOW_WIDTH)
    expect(result.height).toBe(MIN_WINDOW_HEIGHT)
  })

  it('clamps a window that exceeds the viewport', () => {
    const result = clampToViewport(
      { left: -1000, top: -1000, width: 5000, height: 5000 },
      { width: 1280, height: 800 },
    )
    expect(result.width).toBeLessThanOrEqual(1280 - 24 * 2)
    expect(result.height).toBeLessThanOrEqual(800 - 48 - 64)
    expect(result.left).toBeGreaterThanOrEqual(0)
    expect(result.top).toBeGreaterThanOrEqual(0)
  })

  it('keeps the title bar reachable when a window is far off the top edge', () => {
    const result = clampToViewport({ left: 40, top: -2000, width: 900, height: 600 }, { width: 1280, height: 800 })
    expect(result.top).toBeGreaterThanOrEqual(0)
  })

  it('clamps left so a visible slice remains on screen', () => {
    const result = clampToViewport({ left: 2000, top: 50, width: 900, height: 600 }, { width: 1280, height: 800 })
    expect(result.left).toBeLessThanOrEqual(1280 - 24)
    expect(result.left + result.width).toBeGreaterThan(24)
  })

  it('cascades repeated windows without stacking exactly', () => {
    const viewport = { width: 1280, height: 800 }
    const first = cascadePosition(0, viewport)
    const second = cascadePosition(1, viewport)
    const third = cascadePosition(2, viewport)
    expect(second.left).not.toBe(first.left)
    expect(third.left).not.toBe(second.left)
    expect(third.top).not.toBe(second.top)
  })

  it('keeps every cascade position fully inside the viewport', () => {
    const viewport = { width: 1280, height: 800 }
    for (let index = 0; index < 12; index += 1) {
      const position = cascadePosition(index, viewport)
      expect(position.left).toBeGreaterThanOrEqual(0)
      expect(position.top).toBeGreaterThanOrEqual(0)
      expect(position.left + position.width).toBeLessThanOrEqual(viewport.width)
      expect(position.top + position.height).toBeLessThanOrEqual(viewport.height)
    }
  })

  it('falls back to a centered position on a small viewport', () => {
    const result = cascadePosition(0, { width: 500, height: 400 })
    expect(result.width).toBeLessThanOrEqual(500 - 24 * 2)
    expect(result.left).toBeGreaterThanOrEqual(0)
  })

  it('normalizes missing geometry to a cascade position', () => {
    const geometry = normalizeGeometry(null, { width: 1280, height: 800 })
    expect(geometry.width).toBeGreaterThan(0)
    expect(geometry.height).toBeGreaterThan(0)
  })

  it('normalizes invalid persisted geometry against the viewport', () => {
    const geometry = normalizeGeometry(
      { left: -5000, top: -5000, width: 20000, height: 20 },
      { width: 1280, height: 800 },
    )
    expect(geometry.width).toBeLessThanOrEqual(1280 - 24 * 2)
    expect(geometry.height).toBeGreaterThanOrEqual(MIN_WINDOW_HEIGHT)
    expect(geometry.left).toBeGreaterThanOrEqual(0)
    expect(geometry.top).toBeGreaterThanOrEqual(0)
  })
})
