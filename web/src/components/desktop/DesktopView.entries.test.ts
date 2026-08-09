// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DesktopView from '@/components/desktop/DesktopView.vue'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'
import type { DesktopEntries } from '@/lib/desktopEntries'
import { api } from '@/lib/api'

vi.mock('@/lib/desktopEntries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/desktopEntries')>()
  return {
    ...actual,
    loadDesktopEntries: vi.fn(),
  }
})

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      sites: {
        ...actual.api.sites,
        appearance: vi.fn(),
      },
    },
  }
})

const mockedLoad = vi.mocked((await import('@/lib/desktopEntries')).loadDesktopEntries)
const mockedAppearance = vi.mocked(api.sites.appearance)

function makeEntries(): DesktopEntries {
  return {
    apps: [],
    sites: [],
    visible: [
      { key: 'app:nginx', kind: 'app', id: 'nginx', name: 'Nginx', launch: 'external', url: 'http://192.168.1.5:8080', iconURL: '/api/v1/apps/nginx/icon', app: undefined },
      { key: 'app:openclaw', kind: 'app', id: 'openclaw', name: 'OpenClaw', launch: 'script', iconURL: '/api/v1/apps/openclaw/icon', app: undefined },
      { key: 'site:blog', kind: 'site', id: 'blog', name: 'blog.example.com', launch: 'external', url: 'https://blog.example.com', iconURL: '/api/v1/sites/blog/icon', site: undefined },
    ],
    loadedAt: Date.now(),
  }
}

describe('DesktopView dynamic entries', () => {
  beforeEach(() => {
    resetDesktopModeForTest()
    window.localStorage.clear()
    window.scrollTo = vi.fn()
    window.open = vi.fn()
    mockedLoad.mockResolvedValue(makeEntries())
    mockedAppearance.mockReset()
    mockedAppearance.mockResolvedValue({})
  })

  it('renders dynamic app and site icons alongside static nav icons', async () => {
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()
    const labels = wrapper.findAll('.desktop__icon-label').map((el) => el.text())
    expect(labels).toContain('Nginx')
    expect(labels).toContain('OpenClaw')
    expect(labels).toContain('blog.example.com')
    // Static nav icons still present.
    expect(labels).toContain('概览')
    expect(labels).toContain('浏览器')
    wrapper.unmount()
  })

  it('opens the shared browser launcher on its lightweight start page', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="浏览器"]').trigger('dblclick')
    await nextTick()

    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]?.path).toBe('/browser')
    expect(wrapper.find('.desktop-window__title').text()).toContain('浏览器')
    wrapper.unmount()
  })

  it('renders external URLs as img sources for dynamic entries', async () => {
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()
    const imgs = wrapper.findAll('.desktop__icon-img')
    const srcs = imgs.map((img) => img.attributes('src'))
    expect(srcs).toContain('/api/v1/apps/nginx/icon')
    expect(srcs).toContain('/api/v1/apps/openclaw/icon')
    expect(srcs).toContain('/api/v1/sites/blog/icon')
    expect(wrapper.findAll('.desktop__icon-glyph--dynamic')).toHaveLength(3)
    wrapper.unmount()
  })

  it('renders a branded website fallback when its favicon is unavailable', async () => {
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    const siteIcon = wrapper.find('button[title="blog.example.com"]')
    await siteIcon.find('.desktop__icon-img').trigger('error')

    expect(siteIcon.find('.desktop__site-fallback-letter').text()).toBe('B')
    expect(siteIcon.find('.desktop__site-fallback-badge').exists()).toBe(true)
    wrapper.unmount()
  })

  it('opens websites in the reusable desktop browser by default', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="blog.example.com"]').trigger('dblclick')
    await nextTick()

    expect(desktop.windows.value).toHaveLength(1)
    const path = desktop.windows.value[0]?.path || ''
    expect(path).toContain('/browser?')
    const firstQuery = new URLSearchParams(path.split('?')[1])
    expect(firstQuery.get('url')).toBe('https://blog.example.com')
    expect(window.open).not.toHaveBeenCalled()

    await wrapper.find('button[title="blog.example.com"]').trigger('dblclick')
    await nextTick()
    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]?.path).not.toBe(path)
    expect(new URLSearchParams(desktop.windows.value[0]?.path.split('?')[1]).get('request')).not.toBe(
      firstQuery.get('request'),
    )
    wrapper.unmount()
  })

  it('opens URL-capable applications in the reusable desktop browser by default', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="Nginx"]').trigger('dblclick')
    await nextTick()

    expect(desktop.windows.value).toHaveLength(1)
    const path = desktop.windows.value[0]?.path || ''
    expect(path).toContain('/browser?')
    const query = new URLSearchParams(path.split('?')[1])
    expect(query.get('shortcut')).toBe('app:nginx')
    expect(query.get('url')).toBe('http://192.168.1.5:8080')
    expect(window.open).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps an explicit system-browser action in the website context menu', async () => {
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="blog.example.com"]').trigger('contextmenu', { clientX: 120, clientY: 80 })
    await nextTick()
    const siteItems = wrapper.findAll('.desktop__context-menu [role="menuitem"]')
    expect(siteItems[1]?.text()).toContain('用系统浏览器打开')
    await siteItems[1]?.trigger('click')

    expect(window.open).toHaveBeenCalledWith(
      'https://blog.example.com',
      '_blank',
      'noopener,noreferrer',
    )
    wrapper.unmount()
  })

  it('keeps explicit external-open and application-detail actions for URL applications', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="Nginx"]').trigger('contextmenu', { clientX: 80, clientY: 80 })
    await nextTick()
    const appItems = wrapper.findAll('.desktop__context-menu [role="menuitem"]')
    expect(appItems).toHaveLength(3)
    await appItems[1]?.trigger('click')
    expect(window.open).toHaveBeenCalledWith(
      'http://192.168.1.5:8080',
      '_blank',
      'noopener,noreferrer',
    )

    await wrapper.find('button[title="Nginx"]').trigger('contextmenu', { clientX: 80, clientY: 80 })
    await nextTick()
    await wrapper.findAll('.desktop__context-menu [role="menuitem"]')[2]?.trigger('click')
    expect(desktop.windows.value[0]?.path).toBe('/apps?app=nginx')
    wrapper.unmount()
  })

  it('opens the matching application-market detail from an app context menu', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="Nginx"]').trigger('contextmenu', { clientX: 80, clientY: 80 })
    await nextTick()
    await wrapper.findAll('.desktop__context-menu [role="menuitem"]')[2]?.trigger('click')

    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]?.path).toBe('/apps?app=nginx')
    expect(wrapper.find('.desktop__detail').exists()).toBe(false)
    wrapper.unmount()
  })

  it('offers persistent rename only for website icons', async () => {
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="Nginx"]').trigger('contextmenu', { clientX: 80, clientY: 80 })
    await nextTick()
    expect(wrapper.findAll('.desktop__context-menu [role="menuitem"]')).toHaveLength(3)

    await wrapper.find('button[title="blog.example.com"]').trigger('contextmenu', { clientX: 120, clientY: 80 })
    await nextTick()
    const siteItems = wrapper.findAll('.desktop__context-menu [role="menuitem"]')
    expect(siteItems).toHaveLength(4)
    await siteItems[3]?.trigger('click')
    await nextTick()

    const input = document.body.querySelector<HTMLInputElement>('.desktop__rename-form input')
    expect(input).not.toBeNull()
    if (!input) throw new Error('rename input was not rendered')
    input.value = '我的博客'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    document.body.querySelector<HTMLButtonElement>('.modal-panel__footer .button--primary')?.click()
    await nextTick()
    await nextTick()

    expect(window.localStorage.getItem('kpanel:desktop-site-names:v1')).toContain('我的博客')
    expect(wrapper.findAll('.desktop__icon-label').map((label) => label.text())).toContain('我的博客')
    wrapper.unmount()
  })

  it('prioritizes a local rename, then the website name, then the domain', async () => {
    mockedAppearance.mockResolvedValue({ name: 'Example Blog' })
    let wrapper = mount(DesktopView)
    await flushPromises()
    expect(wrapper.findAll('.desktop__icon-label').map((label) => label.text())).toContain('Example Blog')
    wrapper.unmount()

    window.localStorage.setItem(
      'kpanel:desktop-site-names:v1',
      JSON.stringify({ blog: 'My renamed blog' }),
    )
    wrapper = mount(DesktopView)
    await flushPromises()
    expect(wrapper.findAll('.desktop__icon-label').map((label) => label.text())).toContain(
      'My renamed blog',
    )
    wrapper.unmount()

    window.localStorage.clear()
    mockedAppearance.mockReset()
    mockedAppearance.mockRejectedValue(new Error('appearance unavailable'))
    wrapper = mount(DesktopView)
    await flushPromises()
    expect(wrapper.findAll('.desktop__icon-label').map((label) => label.text())).toContain(
      'blog.example.com',
    )
    wrapper.unmount()
  })

  it('launches a script-managed app directly into its management intent', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    const icon = wrapper.find('button[title="OpenClaw"]')
    await icon.trigger('contextmenu', { clientX: 100, clientY: 100 })
    await nextTick()
    expect(wrapper.find('.desktop__context-menu [role="menuitem"]').text()).toContain('脚本管理')
    await icon.trigger('dblclick')
    await nextTick()

    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]?.path).toBe('/app-script/openclaw')
    expect(desktop.windows.value[0]?.titleKey).toBe('desktop.scriptWindowTitle')
    expect(wrapper.find('.desktop-window__title').text()).toContain('OpenClaw 的脚本终端')
    expect(wrapper.find('.desktop-window__app-glyph img').attributes('src')).toBe('/api/v1/apps/openclaw/icon')
    expect(wrapper.find('.app-script-page__header').exists()).toBe(false)
    wrapper.unmount()
  })
})
