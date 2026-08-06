// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { createWindowRouter, windowRouteRecords } from './desktopWindowRoute'

describe('desktop window route', () => {
  beforeEach(() => {
    document.head.innerHTML = ''
  })

  it('registers a single record for a plain page', () => {
    const records = windowRouteRecords('/files')
    expect(records).toHaveLength(1)
    expect(records[0]!.path).toBe('/files')
  })

  it('registers the ai session child route for the ai workspace', () => {
    const records = windowRouteRecords('/ai')
    const paths = records.map((record) => record.path)
    expect(paths).toContain('/ai')
    expect(paths).toContain('/ai/s/:sessionId')
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
})
