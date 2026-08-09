// @vitest-environment jsdom

import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import WebBrowserView from './WebBrowserView.vue'
import {
  EMBEDDED_BROWSER_SLEEP_MS,
  MAX_EMBEDDED_BROWSER_TABS,
  embeddedBrowserShortcutsKey,
  type EmbeddedBrowserShortcut,
} from '@/lib/embeddedBrowser'
import { desktopWindowActiveKey } from '@/lib/desktopRouteKeys'

const configuredShortcuts: EmbeddedBrowserShortcut[] = [
  {
    id: 'app:nginx',
    kind: 'app',
    name: 'Nginx',
    url: 'https://nginx.example.com',
    iconURL: '/api/v1/apps/nginx/icon',
  },
  {
    id: 'site:blog',
    kind: 'site',
    name: '我的博客',
    url: 'https://blog.example.com',
    iconURL: '/api/v1/sites/blog/icon',
  },
]

async function mountBrowser(
  initialPath = '/browser',
  active = ref(true),
): Promise<{ wrapper: VueWrapper; router: Router }> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/browser', component: {} }],
  })
  await router.push(initialPath)
  await router.isReady()
  const wrapper = mount(WebBrowserView, {
    global: {
      plugins: [router],
      provide: {
        [embeddedBrowserShortcutsKey as symbol]: ref(configuredShortcuts),
        [desktopWindowActiveKey as symbol]: active,
      },
    },
  })
  return { wrapper, router }
}

async function visitFromStart(wrapper: VueWrapper, value: string): Promise<void> {
  await wrapper.get('.embedded-browser__start-form input').setValue(value)
  await wrapper.get('.embedded-browser__start-form').trigger('submit')
  await nextTick()
}

async function openNewURL(wrapper: VueWrapper, value: string): Promise<void> {
  await wrapper.get('.embedded-browser__new-tab').trigger('click')
  await nextTick()
  await visitFromStart(wrapper, value)
}

describe('WebBrowserView', () => {
  beforeEach(() => {
    window.open = vi.fn()
  })

  afterEach(() => {
    vi.useRealTimers()
    delete document.documentElement.dataset.theme
  })

  it('opens on a lightweight start page and accepts a bare domain', async () => {
    const { wrapper } = await mountBrowser()

    expect(wrapper.find('.embedded-browser__start').exists()).toBe(true)
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('Nginx')
    expect(wrapper.text()).toContain('我的博客')

    await visitFromStart(wrapper, 'example.com/path')

    expect(wrapper.get('iframe.embedded-browser__frame').attributes('src')).toBe(
      'https://example.com/path',
    )
    expect(wrapper.get('.embedded-browser__tab-count').text()).toBe('1/8')
    wrapper.unmount()
  })

  it('opens an application shortcut through the same embedded browser channel', async () => {
    const { wrapper } = await mountBrowser(
      '/browser?shortcut=app%3Anginx&url=https%3A%2F%2Fnginx.example.com%2Fadmin&request=1',
    )

    expect(wrapper.get('iframe.embedded-browser__frame').attributes('src')).toBe(
      'https://nginx.example.com/admin',
    )
    expect(wrapper.get('[role="tab"]').text()).toContain('Nginx')
    expect(wrapper.get('.embedded-browser__tab-icon').attributes('src')).toBe(
      '/api/v1/apps/nginx/icon',
    )
    wrapper.unmount()
  })

  it('embeds a configured website in a sandbox and provides an external-browser action', async () => {
    const { wrapper } = await mountBrowser(
      '/browser?site=blog&url=https%3A%2F%2Fblog.example.com%2Fpath&request=1',
    )

    const frame = wrapper.get('iframe.embedded-browser__frame')
    expect(frame.attributes('src')).toBe('https://blog.example.com/path')
    expect(frame.attributes('sandbox')).toContain('allow-scripts')
    expect(frame.attributes('sandbox')).not.toContain('allow-top-navigation')
    expect(wrapper.get('[role="tab"]').text()).toContain('我的博客')

    await wrapper.get('.embedded-browser__external').trigger('click')
    expect(window.open).toHaveBeenCalledWith(
      'https://blog.example.com/path',
      '_blank',
      'noopener,noreferrer',
    )
    wrapper.unmount()
  })

  it('propagates the dark panel color scheme to embedded browser UI', async () => {
    document.documentElement.dataset.theme = 'dark'
    const { wrapper } = await mountBrowser(
      '/browser?url=https%3A%2F%2Fblog.example.com%2Fpath',
    )

    const frame = wrapper.get('iframe.embedded-browser__frame')
    expect(window.getComputedStyle(frame.element).colorScheme).toBe('dark')

    wrapper.unmount()
  })

  it('keeps at most two iframe documents alive while retaining more tabs', async () => {
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fone.example.com')

    await openNewURL(wrapper, 'two.example.com')
    await openNewURL(wrapper, 'three.example.com')

    expect(wrapper.findAll('[role="tab"]')).toHaveLength(3)
    expect(wrapper.findAll('iframe.embedded-browser__frame')).toHaveLength(2)
    wrapper.unmount()
  })

  it('sleeps live frames after the browser window stays inactive', async () => {
    vi.useFakeTimers()
    const active = ref(true)
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fone.example.com', active)
    expect(wrapper.findAll('iframe')).toHaveLength(1)

    active.value = false
    await nextTick()
    vi.advanceTimersByTime(EMBEDDED_BROWSER_SLEEP_MS)
    await nextTick()

    expect(wrapper.findAll('iframe')).toHaveLength(0)
    active.value = true
    await nextTick()
    expect(wrapper.findAll('iframe')).toHaveLength(1)
    wrapper.unmount()
  })

  it('keeps the requested website pending at the tab limit instead of replacing a tab', async () => {
    const { wrapper, router } = await mountBrowser('/browser?url=https%3A%2F%2Fsite-1.example.com')

    for (let index = 2; index <= MAX_EMBEDDED_BROWSER_TABS; index += 1) {
      await openNewURL(wrapper, `site-${index}.example.com`)
    }
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(MAX_EMBEDDED_BROWSER_TABS)

    await router.replace(
      '/browser?url=https%3A%2F%2Fsite-9.example.com&request=9',
    )
    await nextTick()

    expect(wrapper.find('.embedded-browser__limit').exists()).toBe(true)
    expect(wrapper.text()).toContain('标签页已满')
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(MAX_EMBEDDED_BROWSER_TABS)

    await wrapper.get('.embedded-browser__limit .button--secondary').trigger('click')
    await nextTick()
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(MAX_EMBEDDED_BROWSER_TABS)
    expect(wrapper.get('[role="tab"][aria-selected="true"]').text()).toContain('site-9.example.com')
    wrapper.unmount()
  })

  it('rejects unsafe custom input without creating a frame', async () => {
    const { wrapper } = await mountBrowser()
    await visitFromStart(wrapper, 'javascript:alert(1)')

    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('reacts to every desktop website request and focuses an existing matching tab', async () => {
    const { wrapper, router } = await mountBrowser(
      '/browser?url=https%3A%2F%2Fone.example.com&request=1',
    )
    await router.replace('/browser?url=https%3A%2F%2Ftwo.example.com&request=2')
    await nextTick()
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(2)

    await router.replace('/browser?url=https%3A%2F%2Fone.example.com&request=3')
    await nextTick()
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(2)
    expect(wrapper.get('[role="tab"][aria-selected="true"]').text()).toContain('one.example.com')
    wrapper.unmount()
  })
})
