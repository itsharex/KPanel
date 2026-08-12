import { describe, expect, it, vi } from 'vitest'
import { createTerminalTouchScroll } from './terminalTouchScroll'

function touchEvent(clientY: number): TouchEvent {
  return {
    touches: {
      length: 1,
      item: () => ({ clientY }),
    },
    preventDefault: vi.fn(),
    stopPropagation: vi.fn(),
  } as unknown as TouchEvent
}

describe('terminal touch scrolling', () => {
  it('converts a vertical swipe into terminal buffer rows', () => {
    const scrollLines = vi.fn()
    const touchScroll = createTerminalTouchScroll({
      getTerminal: () => ({ rows: 10, scrollLines }),
      getScreen: () => ({
        getBoundingClientRect: () => ({ height: 160 }),
      }) as HTMLElement,
    })
    const start = touchEvent(100)
    const move = touchEvent(68)

    touchScroll.start(start)
    touchScroll.move(move)

    expect(scrollLines).toHaveBeenCalledWith(2)
    expect(move.preventDefault).toHaveBeenCalledOnce()
    expect(move.stopPropagation).toHaveBeenCalledOnce()
  })

  it('clears the gesture after touch end', () => {
    const scrollLines = vi.fn()
    const touchScroll = createTerminalTouchScroll({
      getTerminal: () => ({ rows: 10, scrollLines }),
      getScreen: () => ({
        getBoundingClientRect: () => ({ height: 160 }),
      }) as HTMLElement,
    })

    touchScroll.start(touchEvent(100))
    touchScroll.end()
    touchScroll.move(touchEvent(60))

    expect(scrollLines).not.toHaveBeenCalled()
  })
})
