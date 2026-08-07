// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AppMarketItem, Site } from '@/types/api'
import { loadDesktopEntries } from './desktopEntries'
import { api } from '@/lib/api'

vi.mock('@/lib/api', () => ({
  api: {
    apps: {
      inventory: vi.fn(),
    },
    sites: {
      list: vi.fn(),
      iconURL: (id: string) => `/api/v1/sites/${id}/icon`,
    },
    system: {
      publicNetwork: vi.fn(),
    },
  },
}))

function makeSite(overrides: Partial<Site> & { id: string; primaryDomain: string }): Site {
  return {
    domains: [overrides.primaryDomain],
    type: 'proxy',
    enabled: true,
    health: 'healthy',
    consistency: 'synced',
    access: 'managed',
    source: 'panel',
    resourceVersion: 'v1',
    ...overrides,
  }
}

function inventory(items: AppMarketItem[] = []) {
  return {
    schemaVersion: 1,
    source: 'catalog',
    scriptSha256: '',
    catalogMode: 'cached' as const,
    collectedAt: '2026-01-01T00:00:00.000Z',
    items,
    installed: items.filter((i) => i.runtime.installed).length,
    running: 0,
    updateAvailable: 0,
    categories: [],
  }
}

function makeApp(overrides: Partial<AppMarketItem> & { id: string }): AppMarketItem {
  return {
    num: 1,
    source: 'builtin',
    token: 't',
    name_zh: overrides.id,
    name_en: overrides.id,
    desc_zh: '',
    desc_en: '',
    cat: 'dev',
    url: 'http://example.com',
    icon: `/api/v1/apps/${overrides.id}/icon`,
    iconSha256: 'sha',
    slug: overrides.id,
    defaultPort: 80,
    installPortConfigurable: false,
    installer: 'declarative',
    runtime: {
      installed: true,
      state: 'running',
      ports: [{ privatePort: 80, publicPort: 8080, type: 'tcp' }],
      accessMode: 'direct',
      updateStatus: 'current',
      detectedBy: [],
    },
    capabilities: {},
    ...overrides,
  }
}

describe('desktop entries', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads installed apps as entries with entry URLs', async () => {
    const app = makeApp({ id: 'nginx', runtime: { installed: true, state: 'running', ports: [{ privatePort: 80, publicPort: 8080, type: 'tcp' }], accessMode: 'direct', updateStatus: 'current', detectedBy: [] } })
    vi.mocked(api.apps.inventory).mockResolvedValue(inventory([app]))
    vi.mocked(api.sites.list).mockResolvedValue({ items: [], total: 0 })

    const entries = await loadDesktopEntries(undefined, '192.168.1.5')
    expect(entries.apps).toHaveLength(1)
    expect(entries.apps[0]!.kind).toBe('app')
    expect(entries.apps[0]!.url).toContain('8080')
    expect(entries.visible).toHaveLength(1)
  })

  it('skips installed apps without a reachable URL', async () => {
    const app = makeApp({ id: 'noaccess', runtime: { installed: true, state: 'stopped', ports: [], accessMode: 'domain_only', updateStatus: 'current', detectedBy: [] } })
    vi.mocked(api.apps.inventory).mockResolvedValue(inventory([app]))
    vi.mocked(api.sites.list).mockResolvedValue({ items: [], total: 0 })

    const entries = await loadDesktopEntries(undefined, '192.168.1.5')
    expect(entries.apps).toHaveLength(0)
    expect(entries.visible).toHaveLength(0)
  })

  it('loads configured sites as site entries', async () => {
    const site = makeSite({ id: 's1', primaryDomain: 'example.com', type: 'proxy', upstream: 'http://127.0.0.1:8081', enabled: true, health: 'healthy' })
    vi.mocked(api.apps.inventory).mockResolvedValue(inventory([]))
    vi.mocked(api.sites.list).mockResolvedValue({ items: [site], total: 1 })

    const entries = await loadDesktopEntries(undefined, '192.168.1.5')
    expect(entries.sites).toHaveLength(1)
    expect(entries.sites[0]!.kind).toBe('site')
    expect(entries.sites[0]!.url).toBe('http://example.com')
    expect(entries.sites[0]!.iconURL).toContain('s1/icon')
  })

  it('excludes disabled, unhealthy, or proxy sites without upstream', async () => {
    const disabled = makeSite({ id: 'd', primaryDomain: 'disabled.com', enabled: false, health: 'healthy', upstream: 'http://127.0.0.1:1' })
    const unhealthy = makeSite({ id: 'u', primaryDomain: 'unhealthy.com', enabled: true, health: 'warning', upstream: 'http://127.0.0.1:2' })
    const noUpstreamProxy = makeSite({ id: 'n', primaryDomain: 'noup.com', enabled: true, health: 'healthy', upstream: undefined })
    // Static sites are reachable without an upstream.
    const staticSite = makeSite({ id: 'st', primaryDomain: 'static.com', type: 'static', enabled: true, health: 'healthy', upstream: undefined })
    vi.mocked(api.apps.inventory).mockResolvedValue(inventory([]))
    vi.mocked(api.sites.list).mockResolvedValue({ items: [disabled, unhealthy, noUpstreamProxy, staticSite], total: 4 })

    const entries = await loadDesktopEntries(undefined, '192.168.1.5')
    expect(entries.sites).toHaveLength(1)
    expect(entries.sites[0]!.name).toBe('static.com')
  })

  it('dedupes an app and a site pointing at the same URL, keeping the app', async () => {
    // The app listens on public port 8080 and a proxy site forwards to it, so
    // appAccessURL resolves to https://apps.example.com. The site entry for the
    // same domain resolves to the same URL, so only the app entry survives.
    const app = makeApp({
      id: 'panel',
      runtime: { installed: true, state: 'running', ports: [{ privatePort: 80, publicPort: 8080, type: 'tcp' }], accessMode: 'direct', updateStatus: 'current', detectedBy: [] },
    })
    const proxySite = makeSite({
      id: 'p',
      primaryDomain: 'apps.example.com',
      type: 'proxy',
      upstream: 'http://127.0.0.1:8080',
      enabled: true,
      health: 'healthy',
      certificate: { status: 'valid' },
    })
    // The same domain as a site entry, matching the app's resolved URL.
    const site = makeSite({
      id: 's',
      primaryDomain: 'apps.example.com',
      type: 'proxy',
      upstream: 'http://127.0.0.1:9999',
      enabled: true,
      health: 'healthy',
      certificate: { status: 'valid' },
    })
    vi.mocked(api.apps.inventory).mockResolvedValue(inventory([app]))
    vi.mocked(api.sites.list).mockResolvedValue({ items: [proxySite, site], total: 2 })

    const entries = await loadDesktopEntries(undefined, '192.168.1.5')
    // App resolves through its proxy site to https://apps.example.com.
    expect(entries.apps[0]!.url).toBe('https://apps.example.com')
    // Two configured sites on the same domain.
    expect(entries.sites).toHaveLength(2)
    expect(entries.sites[0]!.url).toBe('https://apps.example.com')
    // The app and both sites share one URL → only the app entry is visible.
    expect(entries.visible).toHaveLength(1)
    expect(entries.visible[0]!.kind).toBe('app')
    expect(entries.visible[0]!.id).toBe('panel')
  })

  it('returns empty visible set when both sources fail', async () => {
    vi.mocked(api.apps.inventory).mockRejectedValue(new Error('offline'))
    vi.mocked(api.sites.list).mockRejectedValue(new Error('offline'))

    const entries = await loadDesktopEntries(undefined, '192.168.1.5')
    expect(entries.apps).toHaveLength(0)
    expect(entries.sites).toHaveLength(0)
    expect(entries.visible).toHaveLength(0)
  })
})
