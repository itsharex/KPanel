// @vitest-environment jsdom

import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
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

const { createBrowserSession, fetchBrowserReader } = vi.hoisted(() => ({
  createBrowserSession: vi.fn(),
  fetchBrowserReader: vi.fn(),
}))

class ReaderTestPort {
  onmessage: ((event: { data: unknown }) => void) | null = null
  peer?: ReaderTestPort

  postMessage(data: unknown): void {
    this.peer?.onmessage?.({ data })
  }

  start(): void {}

  close(): void {
    this.onmessage = null
  }
}

class ReaderTestChannel {
  port1 = new ReaderTestPort()
  port2 = new ReaderTestPort()

  constructor() {
    this.port1.peer = this.port2
    this.port2.peer = this.port1
  }
}

vi.mock('@/lib/api', () => ({
  api: { browser: { createSession: createBrowserSession, fetchReader: fetchBrowserReader } },
}))

const coreSession = {
  mode: 'beta' as const,
  relayUrl: 'https://browser-relay.example.com',
  token: 'signed-browser-token',
  sessionId: 'browser-session-1',
  expiresAt: '2030-01-01T00:00:00Z',
}

const readerSession = {
  ...coreSession,
  mode: 'reader' as const,
  relayUrl: 'https://panel.example.com',
}

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
  await flushPromises()
  await nextTick()
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

async function postedNavigation(wrapper: VueWrapper): Promise<{ token: string; url: string; navigationId: string }> {
  const frame = wrapper.get('iframe.embedded-browser__frame')
  const postMessage = vi.fn()
  Object.defineProperty(frame.element, 'contentWindow', {
    configurable: true,
    value: { postMessage },
  })
  await frame.trigger('load')
  const message = postMessage.mock.calls.at(-1)?.[0]
  expect(postMessage).toHaveBeenLastCalledWith(message, 'https://browser-relay.example.com')
  return message as { token: string; url: string; navigationId: string }
}

describe('WebBrowserView', () => {
  beforeEach(() => {
    vi.stubGlobal('MessageChannel', ReaderTestChannel)
    window.open = vi.fn()
    createBrowserSession.mockReset()
    fetchBrowserReader.mockReset()
    createBrowserSession.mockResolvedValue(coreSession)
    fetchBrowserReader.mockResolvedValue({
      status: 200,
      headers: [['Content-Type', 'text/html; charset=utf-8']],
      body: new TextEncoder().encode('<main>reader page</main>').buffer,
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    delete document.documentElement.dataset.theme
    vi.unstubAllGlobals()
  })

  it('opens on a lightweight start page and accepts a bare domain', async () => {
    const { wrapper } = await mountBrowser()

    expect(wrapper.get('.embedded-browser > .embedded-browser__tabs').classes())
      .not.toContain('embedded-browser__tabs--titlebar')
    expect(wrapper.get('.embedded-browser__content').classes())
      .toContain('embedded-browser__content--start')
    expect(wrapper.find('.embedded-browser__start').exists()).toBe(true)
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('Nginx')
    expect(wrapper.text()).toContain('我的博客')
    expect(wrapper.find('.embedded-browser__beta').exists()).toBe(false)

    await visitFromStart(wrapper, 'example.com/path')

    expect(wrapper.get('iframe.embedded-browser__frame').attributes('src')).toBe(
      'https://browser-relay.example.com/kernel/',
    )
    expect(wrapper.find('.embedded-browser__beta').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Beta')
    expect(await postedNavigation(wrapper)).toEqual({
      type: 'kpanel-browser:navigate',
      token: 'signed-browser-token',
      url: 'https://example.com/path',
      navigationId: expect.any(String),
    })
    expect(wrapper.get('.embedded-browser__content').classes())
      .not.toContain('embedded-browser__content--start')
    expect(wrapper.get('.embedded-browser__tab-count').text()).toBe('1/8')
    wrapper.unmount()
  })

  it('navigates and reloads an existing tab without replacing its kernel frame', async () => {
    const { wrapper } = await mountBrowser()
    await visitFromStart(wrapper, 'one.example.com')

    const frame = wrapper.get('iframe.embedded-browser__frame')
    const frameElement = frame.element
    const postMessage = vi.fn()
    Object.defineProperty(frameElement, 'contentWindow', {
      configurable: true,
      value: { postMessage },
    })
    await frame.trigger('load')
    const initialNavigationID = postMessage.mock.calls.at(-1)?.[0].navigationId

    const address = wrapper.get('.embedded-browser__address input')
    await address.trigger('focus')
    await address.setValue('two.example.com')
    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://browser-relay.example.com',
      source: (frameElement as HTMLIFrameElement).contentWindow,
      data: {
        type: 'kpanel-browser:navigation',
        url: 'https://one.example.com/',
        navigationId: initialNavigationID,
      },
    }))
    expect(address.element).toHaveProperty('value', 'two.example.com')
    await address.trigger('keydown.enter')
    await nextTick()

    expect(wrapper.get('iframe.embedded-browser__frame').element).toBe(frameElement)
    expect(postMessage.mock.calls.at(-1)?.[0]).toEqual({
      type: 'kpanel-browser:navigate',
      token: 'signed-browser-token',
      url: 'https://two.example.com/',
      navigationId: expect.any(String),
    })
    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://browser-relay.example.com',
      source: (frameElement as HTMLIFrameElement).contentWindow,
      data: {
        type: 'kpanel-browser:title',
        title: 'Stale first page',
        navigationId: initialNavigationID,
      },
    }))
    expect(wrapper.get('[role="tab"]').text()).not.toContain('Stale first page')

    await wrapper.get('[data-browser-action="reload"]').trigger('click')
    expect(postMessage.mock.calls.at(-1)?.[0]).toEqual({
      type: 'kpanel-browser:command',
      command: 'reload',
      navigationId: expect.any(String),
    })
    wrapper.unmount()
  })

  it('uses Bing global search and leaves regional routing to Bing', async () => {
    const domestic = await mountBrowser('/browser')
    expect(domestic.wrapper.text()).toContain('Bing')
    const domesticInput = domestic.wrapper.get('.embedded-browser__start-form input')
    await domesticInput.setValue('KPanel 部署教程')
    await domesticInput.trigger('keydown', { key: 'Enter' })
    await nextTick()
    const bingURL = new URL((await postedNavigation(domestic.wrapper)).url)
    expect(bingURL.hostname).toBe('www.bing.com')
    expect(bingURL.searchParams.get('q')).toBe('KPanel 部署教程')
    expect(domestic.wrapper.get('[role="tab"]').text()).toContain('搜索：KPanel 部署教程')
    domestic.wrapper.unmount()

    const international = await mountBrowser('/browser')
    expect(international.wrapper.text()).toContain('Bing')
    await visitFromStart(international.wrapper, 'KPanel documentation')
    const bingURLInternational = new URL((await postedNavigation(international.wrapper)).url)
    expect(bingURLInternational.hostname).toBe('www.bing.com')
    expect(bingURLInternational.searchParams.get('q')).toBe('KPanel documentation')
    international.wrapper.unmount()
  })

  it('opens an application shortcut through the same embedded browser channel', async () => {
    const { wrapper } = await mountBrowser(
      '/browser?shortcut=app%3Anginx&url=https%3A%2F%2Fnginx.example.com%2Fadmin&request=1',
    )

    expect(wrapper.get('iframe.embedded-browser__frame').attributes('src')).toBe(
      'https://browser-relay.example.com/kernel/',
    )
    expect((await postedNavigation(wrapper)).url).toBe('https://nginx.example.com/admin')
    expect(wrapper.get('[role="tab"]').text()).toContain('Nginx')
    expect(wrapper.get('.embedded-browser__tab-icon').attributes('src')).toBe(
      '/api/v1/apps/nginx/icon',
    )
    wrapper.unmount()
  })

  it('loads only the isolated browser kernel and provides an external-browser action', async () => {
    const { wrapper } = await mountBrowser(
      '/browser?site=blog&url=https%3A%2F%2Fblog.example.com%2Fpath&request=1',
    )

    const frame = wrapper.get('iframe.embedded-browser__frame')
    expect(frame.attributes('src')).toBe('https://browser-relay.example.com/kernel/')
    expect(frame.attributes('sandbox')).toContain('allow-scripts')
    expect(frame.attributes('sandbox')).toContain('allow-forms')
    expect(frame.attributes('sandbox')).toContain('allow-popups')
    expect(frame.attributes('sandbox')).not.toContain('allow-popups-to-escape-sandbox')
    expect(frame.attributes('allow')).toContain('autoplay')
    expect(wrapper.html()).not.toContain('src="https://blog.example.com/path"')
    expect(wrapper.get('[role="tab"]').text()).toContain('我的博客')

    await wrapper.get('.embedded-browser__external').trigger('click')
    expect(window.open).toHaveBeenCalledWith(
      'https://blog.example.com/path',
      '_blank',
      'noopener,noreferrer',
    )
    wrapper.unmount()
  })

  it('uses an opaque no-network sandbox for safe reader mode', async () => {
    createBrowserSession.mockResolvedValue(readerSession)
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fblog.example.com%2Fpath')

    const frame = wrapper.get('iframe.embedded-browser__frame')
    expect(frame.attributes('src')).toBe('https://panel.example.com/browser-reader/')
    expect(frame.attributes('sandbox')).toBe('allow-scripts')
    expect(frame.attributes('sandbox')).not.toContain('allow-same-origin')
    expect(frame.attributes('allow')).toBe('')
    expect(wrapper.find('.embedded-browser__beta').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('阅读模式')
    expect(wrapper.html()).not.toContain('src="https://blog.example.com/path"')

    const childMessages: unknown[] = []
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: {
        postMessage: vi.fn((_message: unknown, origin: string, ports: ReaderTestPort[]) => {
          if (origin !== '*') throw new Error('opaque reader connection requires a wildcard target origin')
          const connectedPort = ports[0]
          if (!connectedPort) throw new Error('reader port was not transferred')
          connectedPort.onmessage = (event) => childMessages.push(event.data)
          connectedPort.postMessage({ type: 'ready' })
        }),
      },
    })
    await frame.trigger('load')
    await flushPromises()

    expect(fetchBrowserReader).toHaveBeenCalledWith(
      'signed-browser-token',
      'https://blog.example.com/path',
      'document',
      expect.any(AbortSignal),
    )
    expect(childMessages).toEqual(expect.arrayContaining([
      expect.objectContaining({ type: 'loading', url: 'https://blog.example.com/path' }),
      expect.objectContaining({ type: 'render', status: 200 }),
    ]))
    wrapper.unmount()
  })

  it('follows bounded document redirects reported by the isolated reader', async () => {
    createBrowserSession.mockResolvedValue(readerSession)
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fkejilion.sh%2F')
    const frame = wrapper.get('iframe.embedded-browser__frame')
    const childMessages: Array<Record<string, unknown>> = []
    let childPort: ReaderTestPort | undefined
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: {
        postMessage: vi.fn((_message: unknown, _origin: string, ports: ReaderTestPort[]) => {
          childPort = ports[0]
          if (!childPort) throw new Error('reader port was not transferred')
          childPort.onmessage = (event) => childMessages.push(event.data as Record<string, unknown>)
          childPort.postMessage({ type: 'ready' })
        }),
      },
    })
    await frame.trigger('load')
    await flushPromises()

    for (let redirect = 1; redirect <= 5; redirect += 1) {
      const navigationID = childMessages.filter((message) => message.type === 'render').at(-1)?.navigationId
      childPort?.postMessage({
        type: 'redirect',
        navigationId: navigationID,
        url: `https://kejilion.sh/page-${redirect}`,
      })
      await flushPromises()
    }
    expect(fetchBrowserReader).toHaveBeenLastCalledWith(
      'signed-browser-token', 'https://kejilion.sh/page-5', 'document', expect.any(AbortSignal),
    )

    const finalNavigationID = childMessages.filter((message) => message.type === 'render').at(-1)?.navigationId
    childPort?.postMessage({
      type: 'redirect',
      navigationId: finalNavigationID,
      url: 'https://kejilion.sh/page-6',
    })
    await flushPromises()
    expect(fetchBrowserReader).toHaveBeenCalledTimes(6)
    expect(childMessages).toContainEqual(expect.objectContaining({
      type: 'error',
      message: expect.stringContaining('重定向过多'),
    }))
    wrapper.unmount()
  })

  it('follows reader image redirects and always completes the child resource request', async () => {
    createBrowserSession.mockResolvedValue(readerSession)
    fetchBrowserReader.mockImplementation(async (_token, url, kind) => {
      if (kind === 'document') {
        return {
          status: 200,
          headers: [['Content-Type', 'text/html; charset=utf-8']],
          body: new TextEncoder().encode('<main>reader page</main>').buffer,
        }
      }
      if (kind === 'stylesheet') {
        return {
          status: 200,
          headers: [['Content-Type', 'text/css; charset=utf-8']],
          body: new TextEncoder().encode('.layout{display:grid}').buffer,
        }
      }
      if (url === 'https://cdn.example.com/image') {
        return { status: 302, headers: [['Location', '/image.png']], body: new ArrayBuffer(0) }
      }
      return {
        status: 200,
        headers: [['Content-Type', 'image/png']],
        body: Uint8Array.from([1, 2, 3]).buffer,
      }
    })
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fblog.example.com%2Fpath')
    const frame = wrapper.get('iframe.embedded-browser__frame')
    const childMessages: Array<Record<string, unknown>> = []
    let childPort: ReaderTestPort | undefined
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: {
        postMessage: vi.fn((_message: unknown, origin: string, ports: ReaderTestPort[]) => {
          if (origin !== '*') throw new Error('opaque reader connection requires a wildcard target origin')
          const connectedPort = ports[0]
          if (!connectedPort) throw new Error('reader port was not transferred')
          childPort = connectedPort
          connectedPort.onmessage = (event) => childMessages.push(event.data as Record<string, unknown>)
          connectedPort.postMessage({ type: 'ready' })
        }),
      },
    })
    await frame.trigger('load')
    await flushPromises()
    const navigationID = childMessages.find((message) => message.type === 'render')?.navigationId
    childPort?.postMessage({
      type: 'resource',
      navigationId: navigationID,
      requestId: 'image-1',
      url: 'https://cdn.example.com/image',
    })
    await flushPromises()
    childPort?.postMessage({
      type: 'resource',
      kind: 'stylesheet',
      navigationId: navigationID,
      requestId: 'stylesheet-1',
      url: 'https://blog.example.com/site.css',
    })
    await flushPromises()

    expect(fetchBrowserReader).toHaveBeenCalledWith(
      'signed-browser-token', 'https://cdn.example.com/image.png', 'image', expect.any(AbortSignal),
    )
    expect(childMessages).toContainEqual(expect.objectContaining({
      type: 'resource-result',
      requestId: 'image-1',
      body: expect.any(ArrayBuffer),
    }))
    expect(fetchBrowserReader).toHaveBeenCalledWith(
      'signed-browser-token', 'https://blog.example.com/site.css', 'stylesheet', expect.any(AbortSignal),
    )
    expect(childMessages).toContainEqual(expect.objectContaining({
      type: 'resource-result',
      requestId: 'stylesheet-1',
      headers: [['Content-Type', 'text/css; charset=utf-8']],
    }))
    wrapper.unmount()
  })

  it('renews a reader session and stops after one failed retry', async () => {
    createBrowserSession
      .mockResolvedValueOnce(readerSession)
      .mockResolvedValueOnce({ ...readerSession, token: 'renewed-reader-token', sessionId: 'browser-session-2' })
    fetchBrowserReader.mockRejectedValue(Object.assign(new Error('expired'), { code: 'browser_session_expired' }))
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fblog.example.com%2Fpath')
    const frame = wrapper.get('iframe.embedded-browser__frame')
    const childMessages: Array<Record<string, unknown>> = []
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: {
        postMessage: vi.fn((_message: unknown, origin: string, ports: ReaderTestPort[]) => {
          if (origin !== '*') throw new Error('opaque reader connection requires a wildcard target origin')
          const connectedPort = ports[0]
          if (!connectedPort) throw new Error('reader port was not transferred')
          connectedPort.onmessage = (event) => childMessages.push(event.data as Record<string, unknown>)
          connectedPort.postMessage({ type: 'ready' })
        }),
      },
    })
    await frame.trigger('load')
    await flushPromises()
    await vi.waitFor(() => expect(fetchBrowserReader).toHaveBeenCalledTimes(2))

    expect(createBrowserSession).toHaveBeenCalledTimes(2)
    expect(fetchBrowserReader.mock.calls.at(-1)?.[0]).toBe('renewed-reader-token')
    expect(childMessages).toContainEqual(expect.objectContaining({
      type: 'error',
      message: expect.stringContaining('续期失败'),
    }))
    wrapper.unmount()
  })

  it('lets a new navigation replace an in-flight reader session renewal', async () => {
    let finishStaleRenewal: ((session: typeof readerSession) => void) | undefined
    const staleRenewal = new Promise<typeof readerSession>((resolve) => {
      finishStaleRenewal = resolve
    })
    createBrowserSession
      .mockResolvedValueOnce(readerSession)
      .mockReturnValueOnce(staleRenewal)
      .mockResolvedValueOnce({ ...readerSession, token: 'fresh-reader-token', sessionId: 'browser-session-3' })
    fetchBrowserReader.mockImplementation(async (token) => {
      if (token !== 'fresh-reader-token') {
        throw Object.assign(new Error('expired'), { code: 'browser_session_expired' })
      }
      return {
        status: 200,
        headers: [['Content-Type', 'text/html; charset=utf-8']],
        body: new TextEncoder().encode('<main>fresh page</main>').buffer,
      }
    })
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fblog.example.com%2Fold')
    const frame = wrapper.get('iframe.embedded-browser__frame')
    const childMessages: Array<Record<string, unknown>> = []
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: {
        postMessage: vi.fn((_message: unknown, origin: string, ports: ReaderTestPort[]) => {
          if (origin !== '*') throw new Error('opaque reader connection requires a wildcard target origin')
          const connectedPort = ports[0]
          if (!connectedPort) throw new Error('reader port was not transferred')
          connectedPort.onmessage = (event) => childMessages.push(event.data as Record<string, unknown>)
          connectedPort.postMessage({ type: 'ready' })
        }),
      },
    })
    await frame.trigger('load')
    await vi.waitFor(() => expect(createBrowserSession).toHaveBeenCalledTimes(2))

    const address = wrapper.get('.embedded-browser__address input')
    await address.setValue('https://blog.example.com/new')
    await address.trigger('keydown.enter')
    await vi.waitFor(() => expect(createBrowserSession).toHaveBeenCalledTimes(3))
    await vi.waitFor(() => expect(fetchBrowserReader).toHaveBeenCalledWith(
      'fresh-reader-token', 'https://blog.example.com/new', 'document', expect.any(AbortSignal),
    ))
    expect(childMessages).toContainEqual(expect.objectContaining({
      type: 'render',
      url: 'https://blog.example.com/new',
    }))

    finishStaleRenewal?.({ ...readerSession, token: 'stale-reader-token', sessionId: 'browser-session-2' })
    await flushPromises()
    expect(fetchBrowserReader.mock.calls.at(-1)?.[0]).toBe('fresh-reader-token')
    wrapper.unmount()
  })

  it('routes an HTTP target through the HTTPS browser kernel without direct mixed content', async () => {
    const { wrapper } = await mountBrowser('/browser?url=http%3A%2F%2Flegacy.example.com%2Fstatus')

    expect(wrapper.get('iframe.embedded-browser__frame').attributes('src')).toBe(
      'https://browser-relay.example.com/kernel/',
    )
    expect((await postedNavigation(wrapper)).url).toBe('http://legacy.example.com/status')
    expect(wrapper.text()).not.toContain('浏览器阻止了不安全的内嵌网页')
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

  it('keeps inactive live frames laid out so switching tabs preserves document state', async () => {
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fone.example.com')

    await openNewURL(wrapper, 'two.example.com')

    const frames = wrapper.findAll('iframe.embedded-browser__frame')
    expect(frames).toHaveLength(2)
    expect(frames.filter((frame) => frame.classes().includes('embedded-browser__frame--active')))
      .toHaveLength(1)
    const inactive = frames.find((frame) => !frame.classes().includes('embedded-browser__frame--active'))
    expect(inactive?.attributes('aria-hidden')).toBe('true')
    expect(inactive?.attributes('tabindex')).toBe('-1')
    expect(inactive?.attributes('style')).not.toContain('display: none')
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

  it('shows a recoverable state when the isolated browser core is unavailable', async () => {
    createBrowserSession.mockRejectedValueOnce(new Error('relay unavailable'))
    const { wrapper } = await mountBrowser('/browser?url=https%3A%2F%2Fexample.com')

    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toContain('relay unavailable')
    expect(wrapper.text()).toContain('内置浏览器暂不可用')
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
