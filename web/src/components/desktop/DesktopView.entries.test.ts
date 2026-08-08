// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DesktopView from '@/components/desktop/DesktopView.vue'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'
import type { DesktopEntries } from '@/lib/desktopEntries'

vi.mock('@/lib/desktopEntries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/desktopEntries')>()
  return {
    ...actual,
    loadDesktopEntries: vi.fn(),
  }
})

const mockedLoad = vi.mocked((await import('@/lib/desktopEntries')).loadDesktopEntries)

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
    mockedLoad.mockResolvedValue(makeEntries())
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
    expect(desktop.windows.value[0]?.path).toBe('/apps?app=openclaw&action=manage')
    wrapper.unmount()
  })
})
