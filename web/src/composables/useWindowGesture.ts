import { onBeforeUnmount } from 'vue'
import type { WindowGeometry } from '@/lib/desktopWindowGeometry'
import { MIN_WINDOW_WIDTH, MIN_WINDOW_HEIGHT } from '@/lib/desktopWindowGeometry'

/**
 * Drag-to-move and edge-resize behaviour for a desktop window.
 *
 * Pointer events drive a move/resize gesture; the gesture updates the window
 * geometry through the store's `updateGeometry`, which clamps against the
 * viewport. The component remains responsive during the gesture because state
 * is applied on every pointermove (throttled by the browser frame rate) rather
 * than only on pointerup.
 */

export type ResizeEdge = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

type Gesture =
  | { kind: 'move'; startX: number; startY: number; startGeometry: WindowGeometry }
  | { kind: 'resize'; edge: ResizeEdge; startX: number; startY: number; startGeometry: WindowGeometry }

const RESIZE_MARGIN = 6

function edgeForTarget(target: HTMLElement): ResizeEdge | null {
  const edge = target.dataset.resizeEdge
  if (!edge) return null
  const valid: ResizeEdge[] = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']
  return valid.includes(edge as ResizeEdge) ? (edge as ResizeEdge) : null
}

function beginGesture(event: PointerEvent, startGeometry: WindowGeometry, edge: ResizeEdge | null): Gesture {
  if (edge) return { kind: 'resize', edge, startX: event.clientX, startY: event.clientY, startGeometry }
  return { kind: 'move', startX: event.clientX, startY: event.clientY, startGeometry }
}

export function useWindowGesture(
  getGeometry: () => WindowGeometry,
  updateGeometry: (geometry: WindowGeometry) => void,
  onMoveStart?: () => void,
  onMoveEnd?: () => void,
) {
  let gesture: Gesture | null = null
  let pointerId: number | undefined

  function onPointerDown(event: PointerEvent, edge: ResizeEdge | null): void {
    if (event.button !== 0) return
    const target = event.currentTarget as HTMLElement | null
    if (target?.hasAttribute('data-no-drag')) return
    gesture = beginGesture(event, getGeometry(), edge)
    pointerId = event.pointerId
    onMoveStart?.()
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    window.addEventListener('pointercancel', onPointerCancel)
    event.preventDefault()
  }

  function onPointerMove(event: PointerEvent): void {
    if (!gesture || pointerId !== event.pointerId) return
    const dx = event.clientX - gesture.startX
    const dy = event.clientY - gesture.startY
    const next: WindowGeometry = { ...gesture.startGeometry }
    if (gesture.kind === 'move') {
      next.left = gesture.startGeometry.left + dx
      next.top = gesture.startGeometry.top + dy
    } else {
      const { edge } = gesture
      if (edge.includes('e')) next.width = Math.max(MIN_WINDOW_WIDTH, gesture.startGeometry.width + dx)
      if (edge.includes('s')) next.height = Math.max(MIN_WINDOW_HEIGHT, gesture.startGeometry.height + dy)
      if (edge.includes('w')) {
        next.width = Math.max(MIN_WINDOW_WIDTH, gesture.startGeometry.width - dx)
        next.left = gesture.startGeometry.left + (gesture.startGeometry.width - next.width)
      }
      if (edge.includes('n')) {
        next.height = Math.max(MIN_WINDOW_HEIGHT, gesture.startGeometry.height - dy)
        next.top = gesture.startGeometry.top + (gesture.startGeometry.height - next.height)
      }
    }
    updateGeometry(next)
  }

  function onPointerUp(event: PointerEvent): void {
    if (pointerId !== event.pointerId) return
    finishGesture()
  }

  function onPointerCancel(event: PointerEvent): void {
    if (pointerId !== event.pointerId) return
    finishGesture()
  }

  function finishGesture(): void {
    gesture = null
    pointerId = undefined
    onMoveEnd?.()
    window.removeEventListener('pointermove', onPointerMove)
    window.removeEventListener('pointerup', onPointerUp)
    window.removeEventListener('pointercancel', onPointerCancel)
  }

  onBeforeUnmount(finishGesture)

  return {
    onPointerDown,
    edgeForTarget,
    RESIZE_MARGIN,
  }
}
