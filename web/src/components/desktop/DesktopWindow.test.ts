// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { useRouter } from 'vue-router'
import DesktopWindow from './DesktopWindow.vue'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'
import WebBrowserView from '@/views/WebBrowserView.vue'
import { desktopBrowserHistoryKey } from '@/lib/desktopRouteKeys'
import type {
  DesktopBrowserHistory,
  DesktopBrowserHistoryPoint,
} from '@/lib/desktopBrowserHistory'

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

function createBrowserHistoryFixture() {
  const listeners = new Set<(point: DesktopBrowserHistoryPoint) => void>()
  const history: DesktopBrowserHistory = {
    navigate: vi.fn().mockResolvedValue(undefined),
    go: vi.fn(),
    subscribe(listener) {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
    dispose: vi.fn(),
  }
  return {
    history,
    emit(point: DesktopBrowserHistoryPoint) {
      for (const listener of [...listeners]) listener(point)
    },
  }
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
        icon: () => null,
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

  it('keeps lazily loaded page components out of Vue reactivity', async () => {
    routeMocks.resolveWindowComponent.mockResolvedValue({
      name: 'DesktopPageFixture',
      template: '<main data-testid="desktop-page-fixture" />',
    })
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/overview', 'route.overview', false)
    const windowState = desktop.windows.value.find((item) => item.id === id)!
    const wrapper = mount(DesktopWindow, {
      props: {
        windowState,
        icon: () => null,
      },
    })

    await flushPromises()
    expect(wrapper.find('[data-testid="desktop-page-fixture"]').exists()).toBe(true)
    expect(warning.mock.calls.flat().join(' ')).not.toContain('made a reactive object')
    wrapper.unmount()
    warning.mockRestore()
  })

  it('uses supplied application chrome and falls back when its icon fails', async () => {
    routeMocks.resolveWindowComponent.mockResolvedValue({
      name: 'ScriptPageFixture',
      template: '<main />',
    })
    const desktop = useDesktopMode()
    const id = desktop.openWindow('/app-script/openclaw', 'desktop.scriptWindowTitle', false)
    const windowState = desktop.windows.value.find((item) => item.id === id)!
    const wrapper = mount(DesktopWindow, {
      props: {
        windowState,
        icon: { template: '<svg data-testid="fallback-script-icon" />' },
        iconUrl: '/api/v1/apps/openclaw/icon',
        title: 'OpenClaw 的脚本终端',
      },
    })

    await flushPromises()
    expect(wrapper.attributes('aria-label')).toBe('OpenClaw 的脚本终端')
    expect(wrapper.find('.desktop-window__title').text()).toContain('OpenClaw 的脚本终端')
    const image = wrapper.find('.desktop-window__app-glyph img')
    expect(image.attributes('src')).toBe('/api/v1/apps/openclaw/icon')

    await image.trigger('error')
    expect(wrapper.find('[data-testid="fallback-script-icon"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('records page navigation for native Back and restores the matching window', async () => {
    routeMocks.resolveWindowComponent.mockResolvedValue({
      name: 'NavigableDesktopPageFixture',
      setup() {
        const router = useRouter()
        return {
          openMonitoring: () => router.push('/monitoring'),
          openProcesses: () => router.push('/processes'),
          goBack: () => router.back(),
        }
      },
      template: `
        <main>
          <button data-testid="open-monitoring" @click="openMonitoring">Monitoring</button>
          <button data-testid="open-processes" @click="openProcesses">Processes</button>
          <button data-testid="go-back" @click="goBack">Back</button>
        </main>
      `,
    })
    const desktop = useDesktopMode()
    const nativeHistory = createBrowserHistoryFixture()
    const id = desktop.openWindow('/overview', 'route.overview', false)
    const windowState = desktop.windows.value.find((item) => item.id === id)!
    const wrapper = mount(DesktopWindow, {
      props: {
        windowState,
        icon: () => null,
      },
      global: {
        provide: {
          [desktopBrowserHistoryKey as symbol]: nativeHistory.history,
        },
      },
    })

    await flushPromises()
    expect(wrapper.find('.desktop-window__back').exists()).toBe(false)

    await wrapper.get('[data-testid="open-monitoring"]').trigger('click')
    await flushPromises()
    expect(windowState.path).toBe('/monitoring')
    expect(nativeHistory.history.navigate).toHaveBeenLastCalledWith(
      { windowId: id, fullPath: '/overview' },
      { windowId: id, fullPath: '/monitoring' },
    )

    await wrapper.get('[data-testid="open-processes"]').trigger('click')
    await flushPromises()
    expect(windowState.path).toBe('/processes')
    expect(nativeHistory.history.navigate).toHaveBeenLastCalledWith(
      { windowId: id, fullPath: '/monitoring' },
      { windowId: id, fullPath: '/processes' },
    )

    await wrapper.get('[data-testid="go-back"]').trigger('click')
    expect(nativeHistory.history.go).toHaveBeenCalledWith(-1)
    expect(windowState.path).toBe('/processes')

    desktop.minimizeWindow(id)
    nativeHistory.emit({ windowId: id + 1, fullPath: '/overview' })
    expect(windowState.path).toBe('/processes')
    expect(windowState.minimized).toBe(true)

    nativeHistory.emit({ windowId: id, fullPath: '/monitoring' })
    await vi.waitFor(() => expect(windowState.path).toBe('/monitoring'))
    expect(desktop.windows.value).toHaveLength(1)
    expect(windowState.minimized).toBe(false)

    nativeHistory.emit({ windowId: id, fullPath: '/overview' })
    await vi.waitFor(() => expect(windowState.path).toBe('/overview'))
    expect(desktop.focusedId.value).toBe(id)
    wrapper.unmount()
  })

  it('hosts browser tabs in the titlebar without turning tab interaction into a window gesture', async () => {
    routeMocks.resolveWindowComponent.mockResolvedValue(WebBrowserView)
    const desktop = useDesktopMode()
    const id = desktop.openWindow(
      '/browser?url=https%3A%2F%2Fexample.com',
      'desktop.browser',
      false,
    )
    const windowState = desktop.windows.value.find((item) => item.id === id)!
    const wrapper = mount(DesktopWindow, {
      attachTo: document.body,
      props: {
        windowState,
        icon: () => null,
      },
    })

    await flushPromises()
    const titlebarTabs = wrapper.get('.desktop-window__title-extension .embedded-browser__tabs')
    expect(titlebarTabs.classes()).toContain('embedded-browser__tabs--titlebar')
    expect(wrapper.find('.desktop-window__body .embedded-browser__tabs').exists()).toBe(false)

    await titlebarTabs.get('[role="tab"]').trigger('dblclick')
    expect(windowState.maximized).toBe(false)

    await titlebarTabs.get('.embedded-browser__new-tab').trigger('click')
    expect(titlebarTabs.get('.embedded-browser__tab-count').text()).toBe('2/8')
    wrapper.unmount()
  })
})
