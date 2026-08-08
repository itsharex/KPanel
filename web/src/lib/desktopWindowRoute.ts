import {
  createMemoryHistory,
  createRouter,
  type RouteLocationNormalizedLoaded,
  type Router,
  type RouteRecordRaw,
} from 'vue-router'
import type { Component } from 'vue'
import { loadNavigationRoute } from '@/lib/navigation'
import { canonicalDesktopAppPath } from '@/lib/desktopApps'

/**
 * Window-scoped router.
 *
 * Each desktop window gets its own router over an in-memory history. Views that
 * call `useRoute()` / `useRouter()` / `RouterLink` resolve against this window
 * router, so `AiView` can `push('/ai/s/123')` without leaving the desktop and
 * multiple windows of the same app never share navigation state.
 */

const ROUTE_KEYS: readonly (keyof RouteLocationNormalizedLoaded)[] = [
  'name',
  'meta',
  'path',
  'hash',
  'query',
  'params',
  'fullPath',
  'matched',
  'redirectedFrom',
]

/**
 * Build the route object vue-router injects as `routeLocationKey`: a plain
 * object whose properties read through to the router's current route, so
 * `useRoute()` in a window stays reactive to window-internal navigation.
 */
export function reactiveRouteFor(router: Router): RouteLocationNormalizedLoaded {
  const reactiveRoute = {} as Record<string, unknown>
  for (const key of ROUTE_KEYS) {
    Object.defineProperty(reactiveRoute, key, {
      get: () => router.currentRoute.value[key],
      enumerable: true,
    })
  }
  return reactiveRoute as unknown as RouteLocationNormalizedLoaded
}

type NavigationPath = Parameters<typeof loadNavigationRoute>[0]

export type WindowGlobalNavigation = (fullPath: string) => void
export type WindowDesktopNavigation = (fullPath: string) => boolean

const GLOBAL_HANDOFF_ROUTE_NAMES = new Set(['setup', 'login'])

/** AiView is not part of the shared lazy route registry; load it directly. */
function aiComponentLoader() {
  return () => import('@/views/AiView.vue').then((module) => module.default)
}

function appScriptComponentLoader() {
  return () => import('@/views/AppScriptView.vue').then((module) => module.default)
}

// DesktopWindow renders the resolved view directly and never mounts a nested
// RouterView. Route records therefore use a synchronous placeholder: this
// keeps navigation deterministic and leaves lazy loading, error UI and retry
// ownership exclusively with DesktopWindow.loadComponent().
const WINDOW_ROUTE_PLACEHOLDER: Component = {}

/** Resolve the page component for a window route path. */
export function resolveWindowComponent(path: string): Promise<Component> {
  if (path === '/ai' || path.startsWith('/ai/s/')) return aiComponentLoader()()
  if (/^\/app-script\/[A-Za-z0-9_-]{1,128}$/.test(path)) return appScriptComponentLoader()()
  return loadNavigationRoute(path as NavigationPath).then((module) => module.default)
}

/** Build the complete route table used by every desktop window. */
export function windowRouteRecords(): RouteRecordRaw[] {
  return [
    { path: '/', redirect: '/overview' },
    {
      path: '/setup',
      name: 'setup',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/login',
      name: 'login',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/overview',
      name: 'overview',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/monitoring',
      name: 'monitoring',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/processes',
      name: 'processes',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/cluster',
      name: 'cluster',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/ai',
      name: 'ai',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/ai/s/:sessionId',
      name: 'ai-session',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/sites',
      name: 'sites',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/sites/environment',
      name: 'sites-environment',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/apps',
      name: 'apps',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/app-script/:appId',
      name: 'app-script',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/files',
      name: 'files',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/terminal',
      name: 'terminal',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/diagnostics',
      name: 'diagnostics',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/docker',
      name: 'docker',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/activity',
      name: 'activity',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    {
      path: '/jobs',
      redirect: { path: '/activity', query: { tab: 'jobs' } },
    },
    {
      path: '/audit',
      redirect: { path: '/activity', query: { tab: 'audit' } },
    },
    {
      path: '/settings',
      name: 'settings',
      component: WINDOW_ROUTE_PLACEHOLDER,
    },
    { path: '/:pathMatch(.*)*', redirect: '/overview' },
  ]
}

function navigateGlobal(fullPath: string): void {
  window.location.assign(fullPath)
}

/** Create an independent router instance for a desktop window. */
export function createWindowRouter(
  initialPath: string,
  globalNavigation: WindowGlobalNavigation = navigateGlobal,
  desktopNavigation?: WindowDesktopNavigation,
): Router {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: windowRouteRecords(),
    scrollBehavior: () => ({ top: 0 }),
  })

  // Authentication pages belong to the application router. In particular,
  // Settings signs out through its injected router; hand that navigation back
  // to the top-level document instead of rendering a login page in a window.
  router.beforeEach((to, from) => {
    if (typeof to.name === 'string' && GLOBAL_HANDOFF_ROUTE_NAMES.has(to.name)) {
      globalNavigation(to.fullPath)
      return false
    }
    if (
      from.path !== '/'
      && canonicalDesktopAppPath(to.path) !== canonicalDesktopAppPath(from.path)
      && desktopNavigation?.(to.fullPath)
    ) {
      return false
    }
    return true
  })

  // Memory history starts at "/"; navigate to the window's page so the window
  // RouterView renders its component immediately.
  void router.push(initialPath).catch(() => router.replace('/overview'))
  return router
}
