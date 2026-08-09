import type { HistoryState, RouteLocationRaw, Router } from 'vue-router'

const WINDOW_ID_KEY = '__kpanelDesktopWindowId'
const FULL_PATH_KEY = '__kpanelDesktopFullPath'
const MAX_FULL_PATH_LENGTH = 2_048

export interface DesktopBrowserHistoryPoint {
  windowId: number
  fullPath: string
  monitoringZoomDepth?: number
}

export interface DesktopBrowserHistory {
  navigate: (
    from: DesktopBrowserHistoryPoint,
    to: DesktopBrowserHistoryPoint,
  ) => Promise<void>
  go: (delta: number) => void
  subscribe: (listener: (point: DesktopBrowserHistoryPoint) => void) => () => void
  dispose: () => void
}

function validPoint(point: DesktopBrowserHistoryPoint): boolean {
  return Number.isSafeInteger(point.windowId)
    && point.windowId > 0
    && point.fullPath.startsWith('/')
    && point.fullPath.length <= MAX_FULL_PATH_LENGTH
}

function historyPoint(state: HistoryState): DesktopBrowserHistoryPoint | undefined {
  const windowId = state[WINDOW_ID_KEY]
  const fullPath = state[FULL_PATH_KEY]
  if (typeof windowId !== 'number' || typeof fullPath !== 'string') return undefined
  const depth = state.monitoringZoomDepth
  const point = {
    windowId,
    fullPath,
    ...(typeof depth === 'number' && Number.isSafeInteger(depth) && depth >= 0 && depth <= 32
      ? { monitoringZoomDepth: depth }
      : {}),
  }
  return validPoint(point) ? point : undefined
}

function stateFor(point: DesktopBrowserHistoryPoint): HistoryState {
  return {
    [WINDOW_ID_KEY]: point.windowId,
    [FULL_PATH_KEY]: point.fullPath,
    ...(typeof point.monitoringZoomDepth === 'number'
      ? { monitoringZoomDepth: point.monitoringZoomDepth }
      : {}),
  }
}

function locationFor(router: Router, point: DesktopBrowserHistoryPoint): RouteLocationRaw {
  const resolved = router.resolve(point.fullPath)
  return {
    path: resolved.path,
    query: resolved.query,
    hash: resolved.hash,
    state: stateFor({ ...point, fullPath: resolved.fullPath }),
  }
}

/**
 * Connect independent desktop-window routers to the document's native history.
 *
 * A navigation records both its owning window and full path. Browser Back / Forward
 * therefore restores the right window even when the user navigated in several
 * windows. iframe navigation is intentionally untouched and keeps participating in
 * the browser's native joint session history on its own.
 */
export function createDesktopBrowserHistory(router: Router): DesktopBrowserHistory {
  const listeners = new Set<(point: DesktopBrowserHistoryPoint) => void>()
  let navigationQueue = Promise.resolve()

  async function write(mode: 'push' | 'replace', point: DesktopBrowserHistoryPoint): Promise<void> {
    const resolved = router.resolve(point.fullPath)
    const normalized = { ...point, fullPath: resolved.fullPath }

    // Vue Router treats a same-URL navigation as a duplicate and would discard
    // custom state. Its history adapter can safely create/tag that entry while
    // preserving Vue Router's own position/scroll bookkeeping.
    if (router.currentRoute.value.fullPath === normalized.fullPath) {
      router.options.history[mode](normalized.fullPath, stateFor(normalized))
      return
    }

    await router[mode](locationFor(router, normalized))
  }

  function navigate(
    from: DesktopBrowserHistoryPoint,
    to: DesktopBrowserHistoryPoint,
  ): Promise<void> {
    if (!validPoint(from) || !validPoint(to) || (
      from.windowId === to.windowId && from.fullPath === to.fullPath
    )) {
      return Promise.resolve()
    }

    const run = async () => {
      const current = historyPoint(router.options.history.state)
      if (!current || current.windowId !== from.windowId || current.fullPath !== from.fullPath) {
        // The first desktop navigation converts the current document entry into
        // the source page. Switching from another window adds a source entry so
        // Back returns there before moving further through history.
        await write(current ? 'push' : 'replace', from)
      }
      await write('push', to)
    }

    navigationQueue = navigationQueue.then(run, run)
    return navigationQueue
  }

  const onPopState = (event: PopStateEvent) => {
    const point = historyPoint((event.state ?? {}) as HistoryState)
    if (!point) return
    for (const listener of [...listeners]) listener(point)
  }

  if (typeof window !== 'undefined') window.addEventListener('popstate', onPopState)

  return {
    navigate,
    go(delta) {
      if (!Number.isSafeInteger(delta) || delta === 0) return
      router.options.history.go(delta)
    },
    subscribe(listener) {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
    dispose() {
      listeners.clear()
      if (typeof window !== 'undefined') window.removeEventListener('popstate', onPopState)
    },
  }
}
