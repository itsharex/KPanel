// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import {
  createDesktopBrowserHistory,
  type DesktopBrowserHistoryPoint,
} from './desktopBrowserHistory'

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/overview', component: {} },
      { path: '/monitoring', component: {} },
      { path: '/files', component: {} },
    ],
  })
}

describe('desktop native browser history', () => {
  const histories: Array<ReturnType<typeof createDesktopBrowserHistory>> = []

  afterEach(() => {
    histories.splice(0).forEach((history) => history.dispose())
  })

  it('records a source baseline before the destination page', async () => {
    const router = createTestRouter()
    await router.replace('/overview')
    const history = createDesktopBrowserHistory(router)
    histories.push(history)

    await history.navigate(
      { windowId: 4, fullPath: '/overview' },
      { windowId: 4, fullPath: '/monitoring', monitoringZoomDepth: 1 },
    )

    expect(router.currentRoute.value.fullPath).toBe('/monitoring')
    expect(router.options.history.state).toMatchObject({
      __kpanelDesktopWindowId: 4,
      __kpanelDesktopFullPath: '/monitoring',
      monitoringZoomDepth: 1,
    })
  })

  it('serializes navigation across windows and emits native back/forward targets', async () => {
    const router = createTestRouter()
    await router.replace('/overview')
    const history = createDesktopBrowserHistory(router)
    histories.push(history)
    const listener = vi.fn<(point: DesktopBrowserHistoryPoint) => void>()
    history.subscribe(listener)

    await history.navigate(
      { windowId: 1, fullPath: '/overview' },
      { windowId: 1, fullPath: '/monitoring' },
    )
    await history.navigate(
      { windowId: 2, fullPath: '/files' },
      { windowId: 2, fullPath: '/files?path=%2Fvar%2Fwww' },
    )

    expect(router.currentRoute.value.fullPath).toBe('/files?path=/var/www')
    expect(listener).not.toHaveBeenCalled()

    window.dispatchEvent(new PopStateEvent('popstate', {
      state: {
        __kpanelDesktopWindowId: 2,
        __kpanelDesktopFullPath: '/files',
      },
    }))
    window.dispatchEvent(new PopStateEvent('popstate', {
      state: {
        __kpanelDesktopWindowId: 1,
        __kpanelDesktopFullPath: '/monitoring',
        monitoringZoomDepth: 2,
      },
    }))

    expect(listener).toHaveBeenNthCalledWith(1, { windowId: 2, fullPath: '/files' })
    expect(listener).toHaveBeenNthCalledWith(2, {
      windowId: 1,
      fullPath: '/monitoring',
      monitoringZoomDepth: 2,
    })
  })

  it('ignores untrusted native history metadata', () => {
    const router = createTestRouter()
    const history = createDesktopBrowserHistory(router)
    histories.push(history)
    const listener = vi.fn()
    history.subscribe(listener)

    window.dispatchEvent(new PopStateEvent('popstate', {
      state: {
        __kpanelDesktopWindowId: -1,
        __kpanelDesktopFullPath: 'https://example.com',
      },
    }))

    expect(listener).not.toHaveBeenCalled()
  })
})
