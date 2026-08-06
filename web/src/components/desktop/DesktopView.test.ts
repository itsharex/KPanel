// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
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
    setupViewport(1280, 800)
  })

  it('renders the icon grid for all desktop apps', () => {
    const wrapper = mount(DesktopView)
    const icons = wrapper.findAll('.desktop__icon')
    expect(icons.length).toBeGreaterThanOrEqual(11)
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

  it('shows the classic-mode switch button', () => {
    const wrapper = mount(DesktopView)
    expect(wrapper.find('.desktop__classic-button').exists()).toBe(true)
    wrapper.unmount()
  })

  it('switches back to classic mode from the top-right button', async () => {
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
