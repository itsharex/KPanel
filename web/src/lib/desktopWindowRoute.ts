import {
  createMemoryHistory,
  createRouter,
  type RouteLocationNormalizedLoaded,
  type Router,
  type RouteRecordRaw,
} from 'vue-router'
import type { Component } from 'vue'
import { loadNavigationRoute } from '@/lib/navigation'

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

/** AiView is not part of the shared lazy route registry; load it directly. */
function aiComponentLoader() {
  return () => import('@/views/AiView.vue').then((module) => module.default)
}

/** Resolve the page component for a window route path. */
export function resolveWindowComponent(path: string): Promise<Component> {
  if (path === '/ai') return aiComponentLoader()()
  return loadNavigationRoute(path as NavigationPath).then((module) => module.default)
}

function routeComponentLoader(path: string) {
  return () => resolveWindowComponent(path)
}

/**
 * Build the minimal route table for a window whose initial page is `path`.
 * The Ai workspace additionally registers its `/ai/s/:sessionId` child route so
 * session navigation stays inside the window.
 */
export function windowRouteRecords(initialPath: string): RouteRecordRaw[] {
  const records: RouteRecordRaw[] = [
    {
      path: initialPath,
      name: `desktop-window-${initialPath}`,
      component: routeComponentLoader(initialPath),
    },
  ]
  if (initialPath === '/ai') {
    records.push({
      path: '/ai/s/:sessionId',
      name: 'desktop-window-ai-session',
      component: aiComponentLoader(),
    })
  }
  return records
}

/** Create an independent router instance for a desktop window. */
export function createWindowRouter(initialPath: string): Router {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: windowRouteRecords(initialPath),
    scrollBehavior: () => ({ top: 0 }),
  })
  // Memory history starts at "/"; navigate to the window's page so the window
  // RouterView renders its component immediately.
  void router.push(initialPath)
  return router
}
