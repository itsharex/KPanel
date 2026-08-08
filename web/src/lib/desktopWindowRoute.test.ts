// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createWindowRouter, resolveWindowComponent, windowRouteRecords } from './desktopWindowRoute'

describe('desktop window route', () => {
  beforeEach(() => {
    document.head.innerHTML = ''
    window.scrollTo = vi.fn()
  })

  it('registers every application route with the canonical names', () => {
    const records = windowRouteRecords()
    const paths = records.map((record) => record.path)
    const names = records.map((record) => record.name)

    expect(paths).toEqual(expect.arrayContaining([
      '/overview',
      '/monitoring',
      '/cluster',
      '/ai',
      '/ai/s/:sessionId',
      '/sites',
      '/sites/environment',
      '/apps',
      '/files',
      '/terminal',
      '/diagnostics',
      '/docker',
      '/activity',
      '/jobs',
      '/audit',
      '/settings',
    ]))
    expect(names).toEqual(expect.arrayContaining([
      'overview',
      'monitoring',
      'ai',
      'ai-session',
      'sites-environment',
      'files',
      'activity',
      'settings',
    ]))
  })

  it('resolves dynamic ai session paths to the ai workspace component', async () => {
    expect(await resolveWindowComponent('/ai/s/42')).toBe(await resolveWindowComponent('/ai'))
  })

  it('uses synchronous route placeholders so router navigation never owns page chunk loading', () => {
    const components = windowRouteRecords()
      .map((record) => record.component)
      .filter(Boolean)
    expect(components.length).toBeGreaterThan(0)
    expect(components.every((component) => typeof component !== 'function')).toBe(true)
  })

  it('navigates within the window router without touching the app router', async () => {
    const router = createWindowRouter('/ai')
    await router.push('/ai')
    expect(router.currentRoute.value.path).toBe('/ai')

    await router.push('/ai/s/42')
    expect(router.currentRoute.value.path).toBe('/ai/s/42')
    expect(router.currentRoute.value.params.sessionId).toBe('42')

    await router.replace('/ai')
    expect(router.currentRoute.value.path).toBe('/ai')
    expect(router.currentRoute.value.params.sessionId).toBeUndefined()
  })

  it('keeps independent navigation state across window routers', async () => {
    const first = createWindowRouter('/ai')
    const second = createWindowRouter('/ai')
    await first.push('/ai/s/7')
    await second.push('/ai/s/9')
    expect(first.currentRoute.value.params.sessionId).toBe('7')
    expect(second.currentRoute.value.params.sessionId).toBe('9')
  })

  it('resolves window-relative links against the window router', () => {
    const router = createWindowRouter('/ai')
    const resolved = router.resolve('/ai/s/3')
    expect(resolved.path).toBe('/ai/s/3')
    expect(resolved.params.sessionId).toBe('3')
  })

  it('resolves canonical named routes used by window page components', () => {
    const router = createWindowRouter('/overview')
    const files = router.resolve({ name: 'files', query: { path: '/home' } })
    const environment = router.resolve({ name: 'sites-environment' })

    expect(files.fullPath).toBe('/files?path=/home')
    expect(environment.path).toBe('/sites/environment')
  })

  it('redirects legacy activity paths inside the window', async () => {
    const router = createWindowRouter('/activity')
    await router.push('/jobs')
    expect(router.currentRoute.value.fullPath).toBe('/activity?tab=jobs')

    await router.push('/audit')
    expect(router.currentRoute.value.fullPath).toBe('/activity?tab=audit')
  })

  it('hands authentication routes back to the global application', async () => {
    const globalNavigation = vi.fn()
    const router = createWindowRouter('/settings', globalNavigation)
    await router.push('/settings')

    await router.replace({ name: 'login', query: { redirect: '/settings' } })

    expect(globalNavigation).toHaveBeenCalledWith('/login?redirect=/settings')
    expect(router.currentRoute.value.path).toBe('/settings')
  })

  it('hands cross-app links to the desktop while keeping same-app routes local', async () => {
    const desktopNavigation = vi.fn(() => true)
    const router = createWindowRouter('/sites', vi.fn(), desktopNavigation)
    await router.push('/sites')

    await router.push('/sites/environment')
    expect(router.currentRoute.value.path).toBe('/sites/environment')
    expect(desktopNavigation).not.toHaveBeenCalled()

    await router.push({ name: 'files', query: { path: '/home/web/site' } })
    expect(desktopNavigation).toHaveBeenCalledWith('/files?path=/home/web/site')
    expect(router.currentRoute.value.path).toBe('/sites/environment')
  })
})
