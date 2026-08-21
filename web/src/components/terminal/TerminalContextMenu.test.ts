// @vitest-environment jsdom

import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Terminal } from '@xterm/xterm'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import TerminalContextMenu from './TerminalContextMenu.vue'

interface ExposedMenu {
  open: (event: MouseEvent) => void
  handlePaste: (event: ClipboardEvent) => void
  handleKeyEvent: (event: KeyboardEvent) => boolean
}

describe('TerminalContextMenu', () => {
  const terminal = {
    focus: vi.fn(),
    getSelection: vi.fn(() => 'selected output'),
    hasSelection: vi.fn(() => true),
    paste: vi.fn(),
    selectAll: vi.fn(),
  }
  const writeText = vi.fn(() => Promise.resolve())
  const readText = vi.fn(() => Promise.resolve('pasted command'))
  let wrapper: VueWrapper

  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText, readText },
    })
    wrapper = mount(TerminalContextMenu, {
      attachTo: document.body,
      props: {
        getTerminal: () => terminal as unknown as Terminal,
        canPaste: true,
      },
    })
  })

  afterEach(() => {
    wrapper.unmount()
  })

  async function openMenu(): Promise<HTMLElement> {
    const event = new MouseEvent('contextmenu', { clientX: 40, clientY: 50, cancelable: true })
    ;(wrapper.vm as unknown as ExposedMenu).open(event)
    await wrapper.vm.$nextTick()
    return document.body.querySelector<HTMLElement>('.terminal-context-menu')!
  }

  it('copies the current terminal selection', async () => {
    const menu = await openMenu()
    const copy = menu.querySelectorAll<HTMLButtonElement>('button')[0]!
    copy.click()
    await flushPromises()

    expect(writeText).toHaveBeenCalledWith('selected output')
    expect(terminal.focus).toHaveBeenCalled()
  })

  it('pastes clipboard text through xterm', async () => {
    const menu = await openMenu()
    const paste = menu.querySelectorAll<HTMLButtonElement>('button')[1]!
    paste.click()
    await flushPromises()

    expect(readText).toHaveBeenCalled()
    expect(terminal.paste).toHaveBeenCalledWith('pasted command')
  })

  it('captures native paste data without sending a duplicate browser event', () => {
    const preventDefault = vi.fn()
    const stopPropagation = vi.fn()
    ;(wrapper.vm as unknown as ExposedMenu).handlePaste({
      clipboardData: { getData: () => 'native paste' },
      preventDefault,
      stopPropagation,
    } as unknown as ClipboardEvent)

    expect(terminal.paste).toHaveBeenCalledWith('native paste')
    expect(preventDefault).toHaveBeenCalled()
    expect(stopPropagation).toHaveBeenCalled()
  })

  it('handles terminal copy shortcuts but leaves normal Ctrl+C available for SIGINT', async () => {
    const menu = wrapper.vm as unknown as ExposedMenu
    const copyEvent = new KeyboardEvent('keydown', {
      key: 'c',
      ctrlKey: true,
      shiftKey: true,
      cancelable: true,
    })
    expect(menu.handleKeyEvent(new KeyboardEvent('keydown', { key: 'c', ctrlKey: true }))).toBe(true)
    expect(menu.handleKeyEvent(copyEvent)).toBe(false)
    expect(menu.handleKeyEvent(new KeyboardEvent('keyup', { key: 'c', ctrlKey: true, shiftKey: true }))).toBe(true)
    await flushPromises()

    expect(writeText).toHaveBeenCalledWith('selected output')
    expect(writeText).toHaveBeenCalledTimes(1)
    expect(copyEvent.defaultPrevented).toBe(true)
  })

  it.each([
    { name: 'Ctrl+Shift+V', init: { key: 'v', ctrlKey: true, shiftKey: true } },
    { name: 'Shift+Insert', init: { key: 'Insert', shiftKey: true } },
  ])('pastes with $name while leaving native Ctrl+V to the paste event', async ({ init }) => {
    const menu = wrapper.vm as unknown as ExposedMenu
    const pasteEvent = new KeyboardEvent('keydown', { ...init, cancelable: true })

    expect(menu.handleKeyEvent(new KeyboardEvent('keydown', { key: 'v', ctrlKey: true }))).toBe(true)
    expect(menu.handleKeyEvent(pasteEvent)).toBe(false)
    await flushPromises()

    expect(readText).toHaveBeenCalledTimes(1)
    expect(terminal.paste).toHaveBeenCalledWith('pasted command')
    expect(pasteEvent.defaultPrevented).toBe(true)
  })

  it('does not intercept terminal paste shortcuts after input has closed', () => {
    wrapper.unmount()
    wrapper = mount(TerminalContextMenu, {
      attachTo: document.body,
      props: {
        getTerminal: () => terminal as unknown as Terminal,
        canPaste: false,
      },
    })

    const menu = wrapper.vm as unknown as ExposedMenu
    const pasteEvent = new KeyboardEvent('keydown', {
      key: 'v',
      ctrlKey: true,
      shiftKey: true,
      cancelable: true,
    })

    expect(menu.handleKeyEvent(pasteEvent)).toBe(true)
    expect(readText).not.toHaveBeenCalled()
    expect(pasteEvent.defaultPrevented).toBe(false)
  })

  it('closes on Escape without also closing a parent modal', async () => {
    await openMenu()
    const parentEscape = vi.fn()
    window.addEventListener('keydown', parentEscape)
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    await wrapper.vm.$nextTick()

    expect(parentEscape).not.toHaveBeenCalled()
    expect(document.body.querySelector('.terminal-context-menu')).toBeNull()
    window.removeEventListener('keydown', parentEscape)
  })

  it('selects the full terminal buffer from the menu', async () => {
    const menu = await openMenu()
    menu.querySelectorAll<HTMLButtonElement>('button')[2]!.click()

    expect(terminal.selectAll).toHaveBeenCalled()
  })
})
