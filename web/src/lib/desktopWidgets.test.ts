import { describe, expect, it } from 'vitest'
import { desktopWidgets } from './desktopWidgets'

describe('desktop widget registry', () => {
  it('gives built-in widgets enough grid width for complete metric values', () => {
    expect(desktopWidgets.map((widget) => widget.columns)).toEqual([4, 4, 4])
    expect(desktopWidgets.map((widget) => widget.rows)).toEqual([2, 3, 3])
    expect(desktopWidgets.map((widget) => widget.key)).toEqual([
      'widget:clock',
      'widget:monitor',
      'widget:services',
    ])
  })

  it('registers management metadata for every widget', () => {
    expect(desktopWidgets.every((widget) => (
      widget.icon && widget.titleKey && widget.descriptionKey && widget.tone
    ))).toBe(true)
  })
})
