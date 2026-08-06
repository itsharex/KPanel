import type { InjectionKey } from 'vue'
import type { Router, RouteLocationNormalizedLoaded } from 'vue-router'
import { routeLocationKey, routerKey, routerViewLocationKey } from 'vue-router'

/**
 * Typed handles to vue-router's internal injection keys.
 *
 * These keys are imported by name from the same module specifier the rest of
 * the app uses, so they are the same module instance in dev and prod. Desktop
 * windows provide overrides under these keys so a window's page resolves
 * `useRoute()` / `useRouter()` / `RouterView` against the window's own
 * in-memory router instead of the app router.
 */

export const windowRouterKey = routerKey as InjectionKey<Router>
export const windowRouteKey = routeLocationKey as InjectionKey<RouteLocationNormalizedLoaded>
export const windowRouterViewKey = routerViewLocationKey as InjectionKey<unknown>
