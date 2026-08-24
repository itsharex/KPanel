// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  contextMenuBounds,
  contextMenuFocusOrigin,
  focusFirstContextMenuItem,
  moveContextMenuFocus,
  placeContextMenu,
  showContextMenuKeyboardFocus,
  showContextMenuPointerFocus,
} from './contextMenu'

function rect(left: number, top: number, width: number, height: number): DOMRect {
  return {
    x: left,
    y: top,
    left,
    top,
    right: left + width,
    bottom: top + height,
    width,
    height,
    toJSON: () => ({}),
  }
}

function setViewport(width: number, height: number): void {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: height })
  Object.defineProperty(window, 'visualViewport', { configurable: true, value: undefined })
  Object.defineProperty(document.documentElement, 'clientWidth', { configurable: true, value: width })
  Object.defineProperty(document.documentElement, 'clientHeight', { configurable: true, value: height })
}

describe('context menu placement', () => {
  beforeEach(() => {
    setViewport(1280, 800)
    document.body.replaceChildren()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('keeps a measured menu inside the visual viewport', () => {
    const menu = document.createElement('div')
    menu.getBoundingClientRect = () => rect(0, 0, 218, 430)

    const placed = placeContextMenu(menu, { x: 1200, y: 760 }, null)

    expect(placed).toMatchObject({ x: 1054, y: 362, maxWidth: 1264, maxHeight: 784 })
  })

  it('reserves the full layout size while an enter transition scales the menu', () => {
    setViewport(1280, 500)
    const menu = document.createElement('div')
    menu.getBoundingClientRect = () => rect(0, 0, 190.12, 298.76)
    Object.defineProperty(menu, 'offsetWidth', { configurable: true, value: 196 })
    Object.defineProperty(menu, 'offsetHeight', { configurable: true, value: 308 })

    const placed = placeContextMenu(menu, { x: 1250, y: 480 }, null, 10)

    expect(placed).toMatchObject({ x: 1074, y: 182, maxWidth: 1260, maxHeight: 480 })
    expect(placed.y + menu.offsetHeight).toBe(490)
  })

  it('uses the current desktop window body and visible taskbar as safe boundaries', () => {
    const desktop = document.createElement('div')
    desktop.className = 'desktop'
    desktop.getBoundingClientRect = () => rect(0, 0, 1280, 800)
    const body = document.createElement('div')
    body.className = 'desktop-window__body'
    body.getBoundingClientRect = () => rect(100, 80, 1080, 660)
    const anchor = document.createElement('button')
    body.append(anchor)
    const taskbar = document.createElement('div')
    taskbar.className = 'desktop__taskbar'
    taskbar.getBoundingClientRect = () => rect(8, 728, 1264, 56)
    desktop.append(body, taskbar)
    document.body.append(desktop)
    const menu = document.createElement('div')
    menu.getBoundingClientRect = () => rect(0, 0, 218, 430)

    const placed = placeContextMenu(menu, { x: 1200, y: 760 }, anchor)

    expect(placed).toMatchObject({ x: 954, y: 290, maxWidth: 1064, maxHeight: 632 })
    expect(placed.y + 430).toBeLessThanOrEqual(taskbar.getBoundingClientRect().top - 8)
    expect(menu.style.getPropertyValue('--context-menu-safe-bottom')).toBe('80px')
  })

  it('shrinks an oversized menu so every action remains reachable by scrolling', () => {
    setViewport(240, 180)
    const menu = document.createElement('div')
    menu.getBoundingClientRect = () => rect(0, 0, 300, 500)

    const placed = placeContextMenu(menu, { x: 220, y: 170 }, null)

    expect(placed).toMatchObject({ x: 8, y: 8, maxWidth: 224, maxHeight: 164 })
    expect(menu.style.getPropertyValue('--context-menu-max-width')).toBe('224px')
    expect(menu.style.getPropertyValue('--context-menu-max-height')).toBe('164px')
  })

  it('uses visualViewport offsets and ignores a hidden fullscreen taskbar', () => {
    Object.defineProperty(window, 'visualViewport', {
      configurable: true,
      value: { offsetLeft: 120, offsetTop: 80, width: 600, height: 400 },
    })
    const desktop = document.createElement('div')
    desktop.className = 'desktop'
    desktop.getBoundingClientRect = () => rect(0, 0, 1280, 800)
    const fullscreen = document.createElement('div')
    fullscreen.className = 'terminal-stage is-fullscreen'
    fullscreen.getBoundingClientRect = () => rect(120, 80, 600, 400)
    const anchor = document.createElement('div')
    fullscreen.append(anchor)
    const taskbar = document.createElement('div')
    taskbar.className = 'desktop__taskbar'
    taskbar.style.visibility = 'hidden'
    taskbar.getBoundingClientRect = () => rect(8, 440, 1264, 56)
    desktop.append(fullscreen, taskbar)
    document.body.append(desktop)
    const menu = document.createElement('div')
    menu.getBoundingClientRect = () => rect(0, 0, 200, 160)

    expect(contextMenuBounds(anchor)).toEqual({ left: 120, top: 80, right: 720, bottom: 480 })
    expect(placeContextMenu(menu, { x: 700, y: 470 }, anchor)).toMatchObject({ x: 512, y: 312 })
  })

  it('moves focus across enabled items and keeps the focused item in view', () => {
    const menu = document.createElement('div')
    const first = document.createElement('button')
    first.setAttribute('role', 'menuitem')
    const disabled = document.createElement('button')
    disabled.setAttribute('role', 'menuitem')
    disabled.disabled = true
    const last = document.createElement('button')
    last.setAttribute('role', 'menuitem')
    last.scrollIntoView = vi.fn()
    menu.append(first, disabled, last)
    document.body.append(menu)
    first.focus()
    const event = new KeyboardEvent('keydown', { key: 'ArrowDown', cancelable: true })

    expect(moveContextMenuFocus(menu, event)).toBe(true)
    expect(document.activeElement).toBe(last)
    expect(last.scrollIntoView).toHaveBeenCalledWith({ block: 'nearest' })
    expect(event.defaultPrevented).toBe(true)
  })

  it.each([
    { name: 'right click', button: 2, detail: 0, expected: 'pointer' },
    { name: 'mouse trigger click', button: 0, detail: 1, expected: 'pointer' },
    { name: 'keyboard context menu', button: 0, detail: 0, expected: 'keyboard' },
  ] as const)('identifies $name focus origin', ({ button, detail, expected }) => {
    expect(contextMenuFocusOrigin({ button, detail })).toBe(expected)
  })

  it('keeps pointer-open focus semantic but reveals it for keyboard input', () => {
    const menu = document.createElement('div')
    const disabled = document.createElement('button')
    disabled.disabled = true
    disabled.setAttribute('role', 'menuitem')
    const firstEnabled = document.createElement('button')
    firstEnabled.setAttribute('role', 'menuitem')
    menu.append(disabled, firstEnabled)
    document.body.append(menu)

    expect(focusFirstContextMenuItem(menu, 'pointer')).toBe(firstEnabled)
    expect(document.activeElement).toBe(firstEnabled)
    expect(menu.dataset.contextMenuFocus).toBe('pointer')
    expect(firstEnabled.hasAttribute('data-context-menu-active')).toBe(true)
    expect(firstEnabled.hasAttribute('aria-selected')).toBe(false)

    showContextMenuKeyboardFocus(menu)
    expect(menu.dataset.contextMenuFocus).toBe('keyboard')

    menu.addEventListener('pointermove', showContextMenuPointerFocus)
    menu.dispatchEvent(new Event('pointermove'))
    expect(menu.dataset.contextMenuFocus).toBe('pointer')
  })
})
