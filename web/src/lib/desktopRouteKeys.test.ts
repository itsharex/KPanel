import { beforeEach, describe, expect, it, vi } from 'vitest'
import { desktopCloseGuardCoordinator } from './desktopRouteKeys'

describe('desktop close guard coordinator', () => {
  const cleanup: Array<() => void> = []

  beforeEach(() => {
    while (cleanup.length) cleanup.pop()?.()
  })

  it('stops mode changes when any registered view rejects closing', async () => {
    const first = vi.fn(() => true)
    const second = vi.fn(() => false)
    cleanup.push(desktopCloseGuardCoordinator.register('first', first))
    cleanup.push(desktopCloseGuardCoordinator.register('second', second))

    expect(await desktopCloseGuardCoordinator.checkAll()).toBe(false)
    expect(first).toHaveBeenCalledOnce()
    expect(second).toHaveBeenCalledOnce()
  })

  it('removes guards cleanly when a view unmounts', async () => {
    const guard = vi.fn(() => false)
    const unregister = desktopCloseGuardCoordinator.register('temporary', guard)
    unregister()

    expect(await desktopCloseGuardCoordinator.checkAll()).toBe(true)
    expect(guard).not.toHaveBeenCalled()
  })
})
