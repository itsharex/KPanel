// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { useRouter } from 'vue-router'
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

  it('navigates back through files, monitoring, and process pages inside one window', async () => {
    routeMocks.resolveWindowComponent.mockResolvedValue({
      name: 'NavigableDesktopPageFixture',
      setup() {
        const router = useRouter()
        return {
          openMonitoring: () => router.push('/monitoring'),
          openProcesses: () => router.push('/processes'),
        }
      },
      template: `
        <main>
          <button data-testid="open-monitoring" @click="openMonitoring">Monitoring</button>
          <button data-testid="open-processes" @click="openProcesses">Processes</button>
        </main>
      `,
    })
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
    const back = wrapper.get('.desktop-window__back')
    expect(back.attributes('disabled')).toBeDefined()

    await wrapper.get('[data-testid="open-monitoring"]').trigger('click')
    await flushPromises()
    expect(windowState.path).toBe('/monitoring')
    expect(back.attributes('disabled')).toBeUndefined()

    await wrapper.get('[data-testid="open-processes"]').trigger('click')
    await flushPromises()
    expect(windowState.path).toBe('/processes')
    expect(back.attributes('disabled')).toBeUndefined()

    await back.trigger('click')
    await vi.waitFor(() => expect(windowState.path).toBe('/monitoring'))
    expect(desktop.windows.value).toHaveLength(1)

    await wrapper.trigger('keydown', { altKey: true, key: 'ArrowLeft' })
    await vi.waitFor(() => expect(windowState.path).toBe('/overview'))
    expect(back.attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
