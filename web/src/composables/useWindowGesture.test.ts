// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { useWindowGesture } from './useWindowGesture'
import type { WindowGeometry } from '@/lib/desktopWindowGeometry'

function pointer(type: string, x: number, y: number, id = 1): PointerEvent {
  return new PointerEvent(type, {
    bubbles: true,
    clientX: x,
    clientY: y,
    pointerId: id,
    button: 0,
  })
}

describe('useWindowGesture', () => {
  let getGeometry: () => WindowGeometry
  let updateGeometry: (geometry: WindowGeometry) => void
  let onMoveStart: () => void
  let onMoveEnd: () => void
  let target: HTMLElement

  beforeEach(() => {
    getGeometry = vi.fn(() => ({ left: 100, top: 100, width: 800, height: 600 }))
    updateGeometry = vi.fn()
    onMoveStart = vi.fn()
    onMoveEnd = vi.fn()
    target = document.createElement('div')
    document.body.appendChild(target)
  })

  it('moves the window by the pointer delta', async () => {
    const gesture = useWindowGesture(getGeometry, updateGeometry, onMoveStart, onMoveEnd)
    gesture.onPointerDown(pointer('pointerdown', 100, 100, 1), null)
    window.dispatchEvent(pointer('pointermove', 140, 130, 1))
    window.dispatchEvent(pointer('pointerup', 140, 130, 1))
    await nextTick()
    expect(updateGeometry).toHaveBeenLastCalledWith({ left: 140, top: 130, width: 800, height: 600 })
    expect(onMoveStart).toHaveBeenCalledTimes(1)
    expect(onMoveEnd).toHaveBeenCalledTimes(1)
  })

  it('resizes from the south-east edge', async () => {
    const gesture = useWindowGesture(getGeometry, updateGeometry)
    gesture.onPointerDown(pointer('pointerdown', 900, 700, 1), 'se')
    window.dispatchEvent(pointer('pointermove', 950, 740, 1))
    window.dispatchEvent(pointer('pointerup', 950, 740, 1))
    await nextTick()
    expect(updateGeometry).toHaveBeenLastCalledWith({ left: 100, top: 100, width: 850, height: 640 })
  })

  it('resizes from the west edge, moving the left edge and widening', async () => {
    const gesture = useWindowGesture(getGeometry, updateGeometry)
    gesture.onPointerDown(pointer('pointerdown', 100, 300, 1), 'w')
    window.dispatchEvent(pointer('pointermove', 60, 300, 1))
    window.dispatchEvent(pointer('pointerup', 60, 300, 1))
    await nextTick()
    expect(updateGeometry).toHaveBeenLastCalledWith({ left: 60, top: 100, width: 840, height: 600 })
  })

  it('resizes from the north edge, moving the top edge and growing', async () => {
    const gesture = useWindowGesture(getGeometry, updateGeometry)
    gesture.onPointerDown(pointer('pointerdown', 400, 100, 1), 'n')
    window.dispatchEvent(pointer('pointermove', 400, 70, 1))
    window.dispatchEvent(pointer('pointerup', 400, 70, 1))
    await nextTick()
    expect(updateGeometry).toHaveBeenLastCalledWith({ left: 100, top: 70, width: 800, height: 630 })
  })

  it('clamps resize to the minimum size', async () => {
    const gesture = useWindowGesture(getGeometry, updateGeometry)
    gesture.onPointerDown(pointer('pointerdown', 900, 700, 1), 'se')
    window.dispatchEvent(pointer('pointermove', 300, 200, 1))
    window.dispatchEvent(pointer('pointerup', 300, 200, 1))
    await nextTick()
    const called = (updateGeometry as ReturnType<typeof vi.fn>).mock.calls[0]?.[0] as { width: number; height: number }
    expect(called.width).toBeGreaterThanOrEqual(420)
    expect(called.height).toBeGreaterThanOrEqual(280)
  })

  it('ignores non-primary mouse buttons', async () => {
    const gesture = useWindowGesture(getGeometry, updateGeometry)
    const rightClick = new MouseEvent('pointerdown', { bubbles: true, button: 2, clientX: 100, clientY: 100 })
    gesture.onPointerDown(rightClick as PointerEvent, null)
    window.dispatchEvent(pointer('pointermove', 200, 200, 1))
    await nextTick()
    expect(updateGeometry).not.toHaveBeenCalled()
  })
})
