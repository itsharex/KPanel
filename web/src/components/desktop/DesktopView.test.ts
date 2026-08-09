// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DesktopView from '@/components/desktop/DesktopView.vue'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'

function setupViewport(width: number, height: number): void {
  Object.defineProperty(window, 'innerWidth', { value: width, configurable: true })
  Object.defineProperty(window, 'innerHeight', { value: height, configurable: true })
}

describe('DesktopView', () => {
  beforeEach(() => {
    resetDesktopModeForTest()
    window.localStorage.clear()
    window.scrollTo = vi.fn()
    Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
      configurable: true,
      value: vi.fn(() => null),
    })
    setupViewport(1280, 800)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('renders the icon grid for all desktop apps', () => {
    const wrapper = mount(DesktopView)
    const icons = wrapper.findAll('.desktop__icon')
    expect(icons.length).toBeGreaterThanOrEqual(11)
    wrapper.unmount()
  })

  it('selects an icon on single click and opens it on double click', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    const wrapper = mount(DesktopView)
    const icon = wrapper.find('.desktop__icon')

    await icon.trigger('click')
    expect(icon.classes()).toContain('desktop__icon--selected')
    expect(desktop.windows.value).toHaveLength(0)

    await icon.trigger('dblclick')
    await nextTick()
    expect(desktop.windows.value).toHaveLength(1)
    wrapper.unmount()
  })

  it('renders a window when an app is opened', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/overview', 'route.overview', true)
    await nextTick()
    const wrapper = mount(DesktopView)
    await nextTick()
    expect(wrapper.find('.desktop-window').exists()).toBe(true)
    wrapper.unmount()
  })

  it('renders a taskbar item for each open window', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/overview', 'route.overview', true)
    desktop.openWindow('/files', 'route.files', true)
    await nextTick()
    const wrapper = mount(DesktopView)
    await nextTick()
    expect(wrapper.findAll('.desktop__taskbar-item').length).toBe(2)
    wrapper.unmount()
  })

  it('shows the window name on taskbar hover and closes it from the item context menu', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/overview', 'route.overview', false)
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()

    const taskbarItem = wrapper.find('.desktop__taskbar-item')
    expect(taskbarItem.attributes('title')).toBe(taskbarItem.find('.desktop__taskbar-label').text())

    await taskbarItem.trigger('contextmenu', { clientX: 620, clientY: 740 })
    await nextTick()
    const closeAction = wrapper.find('[data-context-action="close-window"]')
    expect(closeAction.exists()).toBe(true)
    expect(wrapper.find('[data-context-action="processes"]').exists()).toBe(false)

    await closeAction.trigger('click')
    vi.advanceTimersByTime(1)
    await nextTick()
    expect(desktop.windows.value).toHaveLength(0)
    wrapper.unmount()
  })

  it('shows the classic-mode switch button', () => {
    const wrapper = mount(DesktopView)
    expect(wrapper.find('.desktop__classic-button').exists()).toBe(true)
    wrapper.unmount()
  })

  it('mirrors the classic Agent status and version in the taskbar', () => {
    const wrapper = mount(DesktopView, {
      props: {
        agent: {
          connected: true,
          compatible: true,
          readOnly: false,
          version: '0.48.3',
          protocolVersion: '1',
        },
      },
    })
    expect(wrapper.find('.desktop__taskbar-agent-status').text()).toContain('Agent 在线')
    expect(wrapper.find('.desktop__taskbar-agent > small').text()).toBe('v0.48.3')
    wrapper.unmount()
  })

  it('replaces the taskbar version with the classic update action', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView, {
      props: {
        agent: {
          connected: true,
          compatible: true,
          readOnly: false,
          version: '0.48.3',
          protocolVersion: '1',
        },
        kpanelUpdateAvailable: true,
        kpanelUpdateDescription: '当前版本 v0.48.3，发现可用更新',
      },
    })
    const update = wrapper.find('.desktop__taskbar-agent-update')
    expect(update.text()).toContain('更新可用')
    expect(wrapper.find('.desktop__taskbar-agent > small').exists()).toBe(false)
    await update.trigger('click')
    expect(desktop.windows.value[0]?.path).toBe('/apps?app=kpanel&action=update')
    wrapper.unmount()
  })

  it('moves keyboard focus to the desktop when its blank surface is pressed', async () => {
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await wrapper.trigger('pointerdown')
    expect(document.activeElement).toBe(wrapper.element)
    wrapper.unmount()
  })

  it('makes closing immediate when reduced motion is requested', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/overview', 'route.overview', false)
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()

    await wrapper.find('.desktop-window__action--close').trigger('click')
    vi.advanceTimersByTime(1)
    await nextTick()

    expect(desktop.windows.value).toHaveLength(0)
    wrapper.unmount()
  })

  it('switches back to classic mode from the taskbar system area', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    const wrapper = mount(DesktopView)
    await wrapper.find('.desktop__classic-button').trigger('click')
    await nextTick()
    expect(desktop.mode.value).toBe('classic')
    wrapper.unmount()
  })

  it('opens and closes a context menu on right-click', async () => {
    const wrapper = mount(DesktopView)
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(false)
    await wrapper.trigger('contextmenu', { clientX: 200, clientY: 150 })
    await nextTick()
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(true)
    await wrapper.find('.desktop__context-menu button').trigger('click')
    await nextTick()
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(false)
    wrapper.unmount()
  })

  it('opens the process manager from the taskbar context menu', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/overview', 'route.overview', false)
    const wrapper = mount(DesktopView)

    await wrapper.find('.desktop__taskbar').trigger('contextmenu', { clientX: 500, clientY: 760 })
    await nextTick()
    const action = wrapper.find('[data-context-action="processes"]')
    expect(action.exists()).toBe(true)
    expect(action.text()).toContain('进程管理器')

    await action.trigger('click')
    await nextTick()
    expect(desktop.windows.value).toHaveLength(2)
    expect(desktop.windows.value.find((windowState) => windowState.path === '/processes')).toMatchObject({
      path: '/processes',
      titleKey: 'route.processes',
    })
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(false)
    wrapper.unmount()
  })

  it('keeps the context menu stable while the right mouse button is held', async () => {
    const wrapper = mount(DesktopView)
    await wrapper.trigger('contextmenu', { clientX: 240, clientY: 180 })
    await nextTick()
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(true)

    window.dispatchEvent(new MouseEvent('pointerdown', { button: 2, bubbles: true }))
    window.dispatchEvent(new MouseEvent('pointerdown', { button: 2, bubbles: true }))
    await nextTick()
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(true)

    window.dispatchEvent(new MouseEvent('pointerdown', { button: 0, bubbles: true }))
    await nextTick()
    expect(wrapper.find('.desktop__context-menu').exists()).toBe(false)
    wrapper.unmount()
  })

  it('supports arrow-key menu navigation and restores focus on Escape', async () => {
    const wrapper = mount(DesktopView, { attachTo: document.body })
    wrapper.element.focus()
    await wrapper.trigger('contextmenu', { clientX: 200, clientY: 150 })
    await nextTick()
    const items = wrapper.findAll('[role="menuitem"]')
    expect(document.activeElement).toBe(items[0]?.element)

    await items[0]!.trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[1]?.element)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()

    expect(wrapper.find('.desktop__context-menu').exists()).toBe(false)
    expect(document.activeElement).toBe(wrapper.element)
    wrapper.unmount()
  })

  it('renders a window icon even when the app is not in the catalogue', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    desktop.openWindow('/unknown-page', 'route.settings', true)
    await nextTick()
    const wrapper = mount(DesktopView)
    await nextTick()
    expect(wrapper.findAll('.desktop-window').length).toBe(1)
    wrapper.unmount()
  })
})
