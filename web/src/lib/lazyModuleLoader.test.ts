import { describe, expect, it, vi } from 'vitest'
import { createLazyModuleLoader } from './lazyModuleLoader'

describe('lazy module loader', () => {
  it('deduplicates concurrent loads and keeps the resolved module cached', async () => {
    const module = { default: 'apps' }
    const loader = vi.fn(async () => module)
    const modules = createLazyModuleLoader({ '/apps': loader })

    const [first, second] = await Promise.all([modules.load('/apps'), modules.load('/apps')])
    const third = await modules.load('/apps')

    expect(first).toBe(module)
    expect(second).toBe(module)
    expect(third).toBe(module)
    expect(loader).toHaveBeenCalledTimes(1)
  })

  it('retries one transient failure before succeeding', async () => {
    vi.useFakeTimers()
    const module = { default: 'sites' }
    const loader = vi.fn()
      .mockRejectedValueOnce(new Error('temporary network failure'))
      .mockResolvedValueOnce(module)
    const modules = createLazyModuleLoader({ '/sites': loader }, { retryDelayMs: 20 })

    const result = modules.load('/sites')
    await vi.advanceTimersByTimeAsync(20)

    await expect(result).resolves.toBe(module)
    expect(loader).toHaveBeenCalledTimes(2)
    vi.useRealTimers()
  })

  it('allows a later attempt after both initial attempts fail', async () => {
    vi.useFakeTimers()
    const loader = vi.fn()
      .mockRejectedValueOnce(new Error('first failure'))
      .mockRejectedValueOnce(new Error('retry failure'))
      .mockResolvedValueOnce({ default: 'files' })
    const modules = createLazyModuleLoader({ '/files': loader }, { retryDelayMs: 20 })

    const first = modules.prefetch('/files')
    await vi.advanceTimersByTimeAsync(20)
    await expect(first).resolves.toBe(false)

    await expect(modules.load('/files')).resolves.toEqual({ default: 'files' })
    expect(loader).toHaveBeenCalledTimes(3)
    vi.useRealTimers()
  })
})
