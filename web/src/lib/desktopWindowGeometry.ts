/**
 * Desktop window geometry helpers.
 *
 * Windows live in a viewport-sized coordinate space (CSS pixels). Geometry is
 * kept as plain data so it can be persisted to localStorage and unit-tested
 * without a DOM.
 */

export interface WindowGeometry {
  left: number
  top: number
  width: number
  height: number
}

export interface ViewportSize {
  width: number
  height: number
}

export const MIN_WINDOW_WIDTH = 420
export const MIN_WINDOW_HEIGHT = 280
export const DEFAULT_WINDOW_WIDTH = 880
export const DEFAULT_WINDOW_HEIGHT = 600
/** Taskbar + window chrome allowance so a window never opens fully offscreen. */
const TOP_MARGIN = 48
const SIDE_MARGIN = 24
const BOTTOM_MARGIN = 64

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

/**
 * Clamp a geometry so the title bar stays reachable and at least a slice of
 * the window remains on screen. Width and height are first bounded to the
 * viewport, then position is clamped so a visible, grabbable portion always
 * remains inside the viewport.
 */
export function clampToViewport(geometry: WindowGeometry, viewport: ViewportSize): WindowGeometry {
  const width = clamp(geometry.width, MIN_WINDOW_WIDTH, Math.max(viewport.width - SIDE_MARGIN * 2, MIN_WINDOW_WIDTH))
  const height = clamp(
    geometry.height,
    MIN_WINDOW_HEIGHT,
    Math.max(viewport.height - TOP_MARGIN - BOTTOM_MARGIN, MIN_WINDOW_HEIGHT),
  )
  const maxLeft = Math.max(viewport.width - SIDE_MARGIN - width, 0)
  const left = clamp(geometry.left, 0, maxLeft)
  const maxTop = Math.max(viewport.height - BOTTOM_MARGIN - height, 0)
  const top = clamp(geometry.top, 0, maxTop)
  return { left, top, width, height }
}

/**
 * Compute a cascade position for the next window of a given application so
 * repeated opens do not stack exactly on top of each other. Falls back to the
 * center when the viewport is too small to fit a default window.
 */
export function cascadePosition(
  index: number,
  viewport: ViewportSize,
  width = DEFAULT_WINDOW_WIDTH,
  height = DEFAULT_WINDOW_HEIGHT,
): WindowGeometry {
  const usableWidth = Math.max(viewport.width - SIDE_MARGIN * 2, MIN_WINDOW_WIDTH)
  const usableHeight = Math.max(viewport.height - TOP_MARGIN - BOTTOM_MARGIN, MIN_WINDOW_HEIGHT)
  const w = Math.min(width, usableWidth)
  const h = Math.min(height, usableHeight)
  const offset = (index % 6) * 28
  const left = clamp((viewport.width - w) / 2 + offset, SIDE_MARGIN, viewport.width - SIDE_MARGIN - w)
  const top = clamp((viewport.height - h) / 2 - offset / 2, TOP_MARGIN, viewport.height - TOP_MARGIN - h)
  return { left, top, width: w, height: h }
}

/** Normalize user-supplied geometry (e.g. from localStorage) against a viewport. */
export function normalizeGeometry(raw: Partial<WindowGeometry> | null | undefined, viewport: ViewportSize): WindowGeometry {
  if (!raw) return cascadePosition(0, viewport)
  const width = Number.isFinite(raw.width) ? raw.width! : DEFAULT_WINDOW_WIDTH
  const height = Number.isFinite(raw.height) ? raw.height! : DEFAULT_WINDOW_HEIGHT
  const left = Number.isFinite(raw.left) ? raw.left! : (viewport.width - width) / 2
  const top = Number.isFinite(raw.top) ? raw.top! : (viewport.height - height) / 2
  return clampToViewport({ left, top, width, height }, viewport)
}
