// @vitest-environment jsdom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { moveRadioFocus } from './radioGroup'

function radioGroup(): HTMLButtonElement[] {
  document.body.innerHTML = `
    <div role="radiogroup">
      <button role="radio">First</button>
      <button role="radio">Second</button>
      <button role="radio">Third</button>
    </div>
  `
  const options = Array.from(document.querySelectorAll<HTMLButtonElement>('[role="radio"]'))
  for (const option of options) option.addEventListener('keydown', moveRadioFocus)
  return options
}

describe('radio group keyboard navigation', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('wraps arrow navigation and selects the newly focused option', () => {
    const options = radioGroup()
    const selected = vi.fn()
    options[2]?.addEventListener('click', selected)
    options[0]?.focus()

    const event = new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true, cancelable: true })
    options[0]?.dispatchEvent(event)

    expect(event.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(options[2])
    expect(selected).toHaveBeenCalledOnce()
  })

  it('supports Home and End without handling unrelated keys', () => {
    const options = radioGroup()
    const firstSelected = vi.fn()
    options[0]?.addEventListener('click', firstSelected)

    options[1]?.dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(options[2])

    options[2]?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(options[0])
    expect(firstSelected).toHaveBeenCalledOnce()

    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    options[0]?.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(false)
    expect(firstSelected).toHaveBeenCalledOnce()
  })
})
