import { createRouter, createWebHistory } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import {
  beginRouteNavigation,
  failRouteNavigation,
  finishRouteNavigation,
  loadNavigationRoute,
} from '@/lib/navigation'
import { useSession } from '@/stores/session'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    guestOnly?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    {
      path: '/setup',
      name: 'setup',
      component: () => loadNavigationRoute('/setup'),
      meta: { title: '初始化', public: true, guestOnly: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => loadNavigationRoute('/login'),
      meta: { title: '登录', public: true, guestOnly: true },
    },
    {
      path: '/',
      component: AppShell,
      children: [
        { path: '', redirect: '/overview' },
        {
          path: 'overview',
          name: 'overview',
          component: () => loadNavigationRoute('/overview'),
          meta: { title: '概览' },
        },
        {
          path: 'cluster',
          name: 'cluster',
          component: () => loadNavigationRoute('/cluster'),
          meta: { title: '集群' },
        },
        {
          path: 'sites',
          name: 'sites',
          component: () => loadNavigationRoute('/sites'),
          meta: { title: '网站' },
        },
        {
          path: 'sites/environment',
          name: 'sites-environment',
          component: () => loadNavigationRoute('/sites/environment'),
          meta: { title: '网站 · 环境管理' },
        },
        {
          path: 'apps',
          name: 'apps',
          component: () => loadNavigationRoute('/apps'),
          meta: { title: '应用市场' },
        },
        {
          path: 'files',
          name: 'files',
          component: () => loadNavigationRoute('/files'),
          meta: { title: '文件' },
        },
        {
          path: 'diagnostics',
          name: 'diagnostics',
          component: () => loadNavigationRoute('/diagnostics'),
          meta: { title: '体检' },
        },
        {
          path: 'docker',
          name: 'docker',
          component: () => loadNavigationRoute('/docker'),
          meta: { title: 'Docker' },
        },
        {
          path: 'activity',
          name: 'activity',
          component: () => loadNavigationRoute('/activity'),
          meta: { title: '活动记录' },
        },
        {
          path: 'jobs',
          redirect: { path: '/activity', query: { tab: 'jobs' } },
        },
        {
          path: 'audit',
          redirect: { path: '/activity', query: { tab: 'audit' } },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => loadNavigationRoute('/settings'),
          meta: { title: '设置' },
        },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/overview' },
  ],
})

router.beforeEach(async (to) => {
  beginRouteNavigation(to.path)
  const session = useSession()
  if (!session.state.checked) await session.refresh()

  if (session.state.setupRequired) {
    return to.name === 'setup' ? true : { name: 'setup' }
  }

  if (to.name === 'setup') {
    return session.state.authenticated ? { name: 'overview' } : { name: 'login' }
  }

  if (!to.meta.public && !session.state.authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly && session.state.authenticated) {
    return { name: 'overview' }
  }

  return true
})

router.afterEach((to) => {
  finishRouteNavigation(to.path)
})

router.onError((_error, to) => {
  failRouteNavigation(to.path)
})
