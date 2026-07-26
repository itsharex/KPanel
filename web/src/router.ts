import { createRouter, createWebHistory } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
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
      component: () => import('@/views/SetupView.vue'),
      meta: { title: '初始化', public: true, guestOnly: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
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
          component: () => import('@/views/OverviewView.vue'),
          meta: { title: '概览' },
        },
        {
          path: 'sites',
          name: 'sites',
          component: () => import('@/views/SitesView.vue'),
          meta: { title: '网站' },
        },
        {
          path: 'apps',
          name: 'apps',
          component: () => import('@/views/AppsView.vue'),
          meta: { title: '应用市场' },
        },
        {
          path: 'docker',
          name: 'docker',
          component: () => import('@/views/DockerView.vue'),
          meta: { title: 'Docker' },
        },
        {
          path: 'activity',
          name: 'activity',
          component: () => import('@/views/ActivityView.vue'),
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
          component: () => import('@/views/SettingsView.vue'),
          meta: { title: '设置' },
        },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/overview' },
  ],
})

router.beforeEach(async (to) => {
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
