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
      { key: 'app:nginx', kind: 'app', id: 'nginx', name: 'Nginx', url: 'http://192.168.1.5:8080', iconURL: '/api/v1/apps/nginx/icon', app: undefined },
      { key: 'site:blog', kind: 'site', id: 'blog', name: 'blog.example.com', url: 'https://blog.example.com', iconURL: '/api/v1/sites/blog/icon', site: undefined },
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
    expect(srcs).toContain('/api/v1/sites/blog/icon')
    expect(wrapper.findAll('.desktop__icon-glyph--dynamic')).toHaveLength(2)
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
})
