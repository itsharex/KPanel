export interface ContextMenuPoint {
  x: number
  y: number
}

export interface ContextMenuBounds {
  left: number
  top: number
  right: number
  bottom: number
}

export interface ContextMenuPlacement extends ContextMenuPoint {
  bounds: ContextMenuBounds
  maxWidth: number
  maxHeight: number
}

export type ContextMenuFocusOrigin = 'pointer' | 'keyboard'

const fullscreenBoundarySelector = '.terminal-stage.is-fullscreen, .interactive-terminal.is-fullscreen'
const enabledMenuItemSelector = '[role="menuitem"]:not(:disabled)'
const activeMenuItemAttribute = 'data-context-menu-active'

function focusContextMenuItem(menu: HTMLElement, target: HTMLButtonElement): void {
  menu.querySelectorAll<HTMLElement>(`[${activeMenuItemAttribute}]`).forEach((item) => {
    item.removeAttribute(activeMenuItemAttribute)
  })
  target.setAttribute(activeMenuItemAttribute, '')
  target.focus({ preventScroll: true })
}

function elementBounds(element: Element | null | undefined): ContextMenuBounds | undefined {
  if (!element) return undefined
  const rect = element.getBoundingClientRect()
  if (rect.width <= 0 || rect.height <= 0) return undefined
  return { left: rect.left, top: rect.top, right: rect.right, bottom: rect.bottom }
}

function intersectBounds(first: ContextMenuBounds, second: ContextMenuBounds): ContextMenuBounds {
  const left = Math.max(first.left, second.left)
  const top = Math.max(first.top, second.top)
  return {
    left,
    top,
    right: Math.max(left, Math.min(first.right, second.right)),
    bottom: Math.max(top, Math.min(first.bottom, second.bottom)),
  }
}

function viewportBounds(): ContextMenuBounds {
  const viewport = window.visualViewport
  const left = viewport?.offsetLeft ?? 0
  const top = viewport?.offsetTop ?? 0
  const width = viewport?.width || document.documentElement.clientWidth || window.innerWidth
  const height = viewport?.height || document.documentElement.clientHeight || window.innerHeight
  return { left, top, right: left + width, bottom: top + height }
}

function taskbarIsVisible(taskbar: HTMLElement): boolean {
  const style = window.getComputedStyle(taskbar)
  return style.display !== 'none' && style.visibility !== 'hidden'
}

export function contextMenuBounds(anchor: Element | null | undefined): ContextMenuBounds {
  let bounds = viewportBounds()
  const desktop = anchor?.closest('.desktop')
  const desktopBounds = elementBounds(desktop)
  if (desktopBounds) bounds = intersectBounds(bounds, desktopBounds)

  const localBoundary = anchor?.closest(fullscreenBoundarySelector)
    || anchor?.closest('.desktop-window__body')
  const localBounds = elementBounds(localBoundary)
  if (localBounds) bounds = intersectBounds(bounds, localBounds)

  const taskbar = desktop?.querySelector<HTMLElement>('.desktop__taskbar')
  const taskbarBounds = taskbar && taskbarIsVisible(taskbar) ? elementBounds(taskbar) : undefined
  if (taskbarBounds) bounds.bottom = Math.max(bounds.top, Math.min(bounds.bottom, taskbarBounds.top))
  return bounds
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), Math.max(minimum, maximum))
}

export function placeContextMenu(
  menu: HTMLElement,
  point: ContextMenuPoint,
  anchor: Element | null | undefined,
  margin = 8,
): ContextMenuPlacement {
  const bounds = contextMenuBounds(anchor)
  const maxWidth = Math.max(0, bounds.right - bounds.left - margin * 2)
  const maxHeight = Math.max(0, bounds.bottom - bounds.top - margin * 2)

  menu.style.setProperty('--context-menu-max-width', `${maxWidth}px`)
  menu.style.setProperty('--context-menu-max-height', `${maxHeight}px`)
  menu.style.setProperty('--context-menu-safe-left', `${Math.max(0, bounds.left + margin)}px`)
  menu.style.setProperty('--context-menu-safe-right', `${Math.max(0, window.innerWidth - bounds.right + margin)}px`)
  menu.style.setProperty('--context-menu-safe-bottom', `${Math.max(0, window.innerHeight - bounds.bottom + margin)}px`)

  const menuRect = menu.getBoundingClientRect()
  // Enter transitions can temporarily scale getBoundingClientRect(). Keep the
  // final, untransformed layout size in the placement calculation as well.
  const width = Math.min(Math.max(menuRect.width, menu.offsetWidth), maxWidth)
  const height = Math.min(Math.max(menuRect.height, menu.offsetHeight), maxHeight)
  const minimumX = bounds.left + margin
  const minimumY = bounds.top + margin
  return {
    bounds,
    maxWidth,
    maxHeight,
    x: clamp(point.x, minimumX, bounds.right - margin - width),
    y: clamp(point.y, minimumY, bounds.bottom - margin - height),
  }
}

export function contextMenuFocusOrigin(
  event: Pick<MouseEvent, 'button' | 'detail'>,
): ContextMenuFocusOrigin {
  return event.button === 2 || event.detail > 0 ? 'pointer' : 'keyboard'
}

export function focusFirstContextMenuItem(
  menu: HTMLElement,
  origin: ContextMenuFocusOrigin,
): HTMLButtonElement | undefined {
  menu.dataset.contextMenuFocus = origin
  const target = menu.querySelector<HTMLButtonElement>(enabledMenuItemSelector) || undefined
  if (target) focusContextMenuItem(menu, target)
  return target
}

export function showContextMenuKeyboardFocus(menu: HTMLElement): void {
  menu.dataset.contextMenuFocus = 'keyboard'
}

export function showContextMenuPointerFocus(event: Event): void {
  const menu = event.currentTarget
  if (!(menu instanceof HTMLElement)) return
  menu.dataset.contextMenuFocus = 'pointer'
}

export function moveContextMenuFocus(menu: HTMLElement, event: KeyboardEvent): boolean {
  const items = [...menu.querySelectorAll<HTMLButtonElement>(enabledMenuItemSelector)]
  if (!items.length) return false
  const current = document.activeElement instanceof HTMLButtonElement
    ? items.indexOf(document.activeElement)
    : -1
  let target: HTMLButtonElement | undefined
  if (event.key === 'ArrowDown') target = items[(current + 1 + items.length) % items.length]
  else if (event.key === 'ArrowUp') target = items[(current - 1 + items.length) % items.length]
  else if (event.key === 'Home') target = items[0]
  else if (event.key === 'End') target = items[items.length - 1]
  else return false

  event.preventDefault()
  if (target) focusContextMenuItem(menu, target)
  target?.scrollIntoView?.({ block: 'nearest' })
  return true
}
