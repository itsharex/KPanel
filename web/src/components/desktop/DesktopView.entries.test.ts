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
    expect(labels).not.toContain('浏览器')
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

  it('confirms before opening a website in the system browser', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="blog.example.com"]').trigger('dblclick')
    await nextTick()

    expect(desktop.windows.value).toHaveLength(0)
    expect(document.body.querySelector('.desktop__external-confirm')?.textContent)
      .toContain('blog.example.com')
    expect(document.body.querySelector('.desktop__external-confirm')?.textContent)
      .toContain('https://blog.example.com')
    const confirmIcon = document.body.querySelector<HTMLImageElement>(
      '.desktop__external-confirm-icon-image',
    )
    expect(confirmIcon?.getAttribute('src')).toBe('/api/v1/sites/blog/icon')
    expect(window.open).not.toHaveBeenCalled()

    document.body.querySelector<HTMLButtonElement>('.modal-panel__footer .button--primary')?.click()
    await nextTick()
    expect(window.open).toHaveBeenCalledWith(
      'https://blog.example.com',
      '_blank',
      'noopener,noreferrer',
    )
    expect(document.body.querySelector('.desktop__external-confirm')).toBeNull()
    wrapper.unmount()
  })

  it('confirms before opening a URL-capable application in the system browser', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="Nginx"]').trigger('dblclick')
    await nextTick()

    expect(desktop.windows.value).toHaveLength(0)
    expect(document.body.querySelector('.desktop__external-confirm')?.textContent).toContain('Nginx')
    expect(document.body.querySelector('.desktop__external-confirm')?.textContent)
      .toContain('http://192.168.1.5:8080')
    expect(document.body.querySelector<HTMLImageElement>('.desktop__external-confirm-icon-image')
      ?.getAttribute('src')).toBe('/api/v1/apps/nginx/icon')
    expect(window.open).not.toHaveBeenCalled()

    document.body.querySelector<HTMLButtonElement>('.modal-panel__footer .button--primary')?.click()
    await nextTick()
    expect(window.open).toHaveBeenCalledWith(
      'http://192.168.1.5:8080',
      '_blank',
      'noopener,noreferrer',
    )
    wrapper.unmount()
  })

  it('shows a branded fallback in the confirmation when the website icon fails', async () => {
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="blog.example.com"]').trigger('dblclick')
    await nextTick()
    document.body.querySelector<HTMLImageElement>('.desktop__external-confirm-icon-image')
      ?.dispatchEvent(new Event('error'))
    await nextTick()

    expect(document.body.querySelector('.desktop__external-confirm .desktop__site-fallback-letter')
      ?.textContent).toBe('B')
    expect(document.body.querySelector('.desktop__external-confirm .desktop__site-fallback-badge'))
      .not.toBeNull()
    wrapper.unmount()
  })

  it('routes the website context-menu action through the same confirmation', async () => {
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="blog.example.com"]').trigger('contextmenu', { clientX: 120, clientY: 80 })
    await nextTick()
    const siteItems = wrapper.findAll('.desktop__context-menu [role="menuitem"]')
    expect(siteItems[0]?.text()).toContain('使用系统浏览器打开')
    await siteItems[0]?.trigger('click')
    expect(window.open).not.toHaveBeenCalled()
    expect(document.body.querySelector('.desktop__external-confirm')).not.toBeNull()
    wrapper.unmount()
  })

  it('routes the website-detail action through the same confirmation', async () => {
    const wrapper = mount(DesktopView, { attachTo: document.body })
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="blog.example.com"]').trigger('contextmenu', { clientX: 120, clientY: 80 })
    await nextTick()
    await wrapper.findAll('.desktop__context-menu [role="menuitem"]')[1]?.trigger('click')
    await nextTick()

    expect(document.body.querySelector('.desktop__detail')).not.toBeNull()
    document.body.querySelector<HTMLButtonElement>('.modal-panel__footer .button--primary')?.click()
    await nextTick()

    expect(document.body.querySelector('.desktop__detail')).toBeNull()
    expect(document.body.querySelector('.desktop__external-confirm')?.textContent)
      .toContain('https://blog.example.com')
    expect(window.open).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps the application-detail action for URL applications', async () => {
    const desktop = useDesktopMode()
    const wrapper = mount(DesktopView)
    await nextTick()
    await nextTick()

    await wrapper.find('button[title="Nginx"]').trigger('contextmenu', { clientX: 80, clientY: 80 })
    await nextTick()
    const appItems = wrapper.findAll('.desktop__context-menu [role="menuitem"]')
    expect(appItems).toHaveLength(2)
    await appItems[1]?.trigger('click')
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
    await wrapper.findAll('.desktop__context-menu [role="menuitem"]')[1]?.trigger('click')

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
    expect(wrapper.findAll('.desktop__context-menu [role="menuitem"]')).toHaveLength(2)

    await wrapper.find('button[title="blog.example.com"]').trigger('contextmenu', { clientX: 120, clientY: 80 })
    await nextTick()
    const siteItems = wrapper.findAll('.desktop__context-menu [role="menuitem"]')
    expect(siteItems).toHaveLength(3)
    await siteItems[2]?.trigger('click')
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
