// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DesktopWindow from './DesktopWindow.vue'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'

const routeMocks = vi.hoisted(() => ({
  resolveWindowComponent: vi.fn(),
}))

vi.mock('@/lib/desktopWindowRoute', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/lib/desktopWindowRoute')>(),
  resolveWindowComponent: routeMocks.resolveWindowComponent,
}))

function setupViewport(): void {
  Object.defineProperty(window, 'innerWidth', { value: 1280, configurable: true })
  Object.defineProperty(window, 'innerHeight', { value: 800, configurable: true })
}

describe('DesktopWindow lazy view loading', () => {
  beforeEach(() => {
    resetDesktopModeForTest()
    window.localStorage.clear()
    window.scrollTo = vi.fn()
    setupViewport()
    routeMocks.resolveWindowComponent.mockReset()
  })

  it('shows a retryable error instead of hanging when a page chunk rejects', async () => {
    routeMocks.resolveWindowComponent.mockRejectedValue(new Error('chunk unavailable'))
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/overview', 'route.overview', false)
    const windowState = desktop.windows.value.find((item) => item.id === id)!
    const wrapper = mount(DesktopWindow, {
      props: {
        windowState,
        icon: { template: '<span />' },
      },
    })

    await flushPromises()
    expect(wrapper.find('.desktop-window__loading').exists()).toBe(false)
    expect(wrapper.find('.desktop-window__load-error').exists()).toBe(true)

    await wrapper.find('.desktop-window__load-error button').trigger('click')
    await flushPromises()
    expect(routeMocks.resolveWindowComponent).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.desktop-window__load-error').exists()).toBe(true)
    wrapper.unmount()
  })
})
