export interface LazyModuleLoader<Key extends string, Module> {
  load: (key: Key) => Promise<Module>
  prefetch: (key: Key) => Promise<boolean>
}

interface LazyModuleLoaderOptions {
  retryDelayMs?: number
}

export function createLazyModuleLoader<Key extends string, Module>(
  loaders: Record<Key, () => Promise<Module>>,
  options: LazyModuleLoaderOptions = {},
): LazyModuleLoader<Key, Module> {
  const cache = new Map<Key, Promise<Module>>()
  const retryDelayMs = options.retryDelayMs ?? 600

  async function loadWithRetry(loader: () => Promise<Module>): Promise<Module> {
    try {
      return await loader()
    } catch {
      await new Promise<void>((resolve) => globalThis.setTimeout(resolve, retryDelayMs))
      return loader()
    }
  }

  function load(key: Key): Promise<Module> {
    const cached = cache.get(key)
    if (cached) return cached

    const promise = loadWithRetry(loaders[key])
    cache.set(key, promise)
    void promise.catch(() => {
      if (cache.get(key) === promise) cache.delete(key)
    })
    return promise
  }

  async function prefetch(key: Key): Promise<boolean> {
    try {
      await load(key)
      return true
    } catch {
      return false
    }
  }

  return { load, prefetch }
}
