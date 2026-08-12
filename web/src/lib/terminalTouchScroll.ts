interface TerminalScrollTarget {
  rows: number
  scrollLines(lines: number): void
}

interface TerminalTouchScrollOptions {
  getTerminal: () => TerminalScrollTarget | undefined
  getScreen: () => HTMLElement | undefined
}

export function createTerminalTouchScroll(options: TerminalTouchScrollOptions) {
  let lastY: number | undefined
  let remainder = 0

  function touchY(event: TouchEvent): number | undefined {
    if (event.touches.length !== 1) return undefined
    return event.touches.item(0)?.clientY
  }

  function start(event: TouchEvent): void {
    lastY = touchY(event)
    remainder = 0
  }

  function move(event: TouchEvent): void {
    const currentY = touchY(event)
    const target = options.getTerminal()
    if (currentY === undefined || lastY === undefined || !target) {
      lastY = undefined
      remainder = 0
      return
    }

    remainder += lastY - currentY
    lastY = currentY

    const screenHeight = options.getScreen()?.getBoundingClientRect().height || 0
    const cellHeight = target.rows > 0 && screenHeight > 0 ? screenHeight / target.rows : 16
    const lines = Math.trunc(remainder / Math.max(1, cellHeight))
    if (lines !== 0) {
      target.scrollLines(lines)
      remainder -= lines * cellHeight
    }

    event.preventDefault()
    event.stopPropagation()
  }

  function end(): void {
    lastY = undefined
    remainder = 0
  }

  return { start, move, end }
}
