import { createSSRApp, ssrContextKey, type Ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AppsView from './AppsView.vue'
import type { AppMarketInventory, Site } from '@/types/api'

const mocks = vi.hoisted(() => ({
  createSite: vi.fn(),
  inventory: vi.fn(),
  action: vi.fn(),
  toastSuccess: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    apps: {
      inventory: mocks.inventory,
      action: mocks.action,
      checkUpdate: vi.fn(),
      install: vi.fn(),
      job: vi.fn(),
      jobs: vi.fn(),
    },
    sites: {
      create: mocks.createSite,
      list: vi.fn(),
      remove: vi.fn(),
    },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    danger: vi.fn(),
    show: vi.fn(),
  }),
}))

interface AppsBindings {
  inventory: Ref<AppMarketInventory | undefined>
  sites: Ref<Site[]>
  selectedID: Ref<string>
  domain: Ref<string>
  domainError: Ref<string>
  domainWarning: Ref<string>
  addDomain: () => Promise<void>
}

function setupView(): AppsBindings {
  const component = AppsView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => AppsBindings
  }
  const app = createSSRApp({ render: () => null })
  app.provide(ssrContextKey, { modules: new Set<string>() })
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
  try {
    return app.runWithContext(() => component.setup({}, { expose: () => undefined }))
  } finally {
    warn.mockRestore()
  }
}

function inventory(resourceVersion: string): AppMarketInventory {
  return {
    schemaVersion: 1,
    source: 'test',
    scriptSha256: 'a'.repeat(64),
    catalogMode: 'embedded',
    categories: [],
    installed: 1,
    running: 1,
    updateAvailable: 0,
    collectedAt: '2026-07-28T00:00:00Z',
    items: [
      {
        id: 'builtin-13',
        num: 13,
        source: 'builtin',
        token: 'cloudreve',
        name_zh: 'Cloudreve',
        name_en: 'Cloudreve',
        desc_zh: '云盘',
        desc_en: 'Cloud storage',
        cat: 'storage',
        icon: '',
        iconSha256: 'b'.repeat(64),
        slug: 'cloudreve',
        installer: 'kejilion',
        runtime: {
          installed: true,
          state: 'running',
          ports: [{ privatePort: 5212, publicPort: 5212, ip: '0.0.0.0', type: 'tcp' }],
          accessMode: 'direct',
          updateStatus: 'check_required',
          resourceVersion,
          detectedBy: ['container'],
        },
        capabilities: {
          add_domain: { enabled: true },
          direct_access: { enabled: true },
        },
      },
    ],
  }
}

function proxySite(): Site {
  return {
    id: 'site-cloud',
    primaryDomain: 'cloud.example.com',
    domains: ['cloud.example.com'],
    type: 'proxy',
    enabled: true,
    health: 'healthy',
    consistency: 'synced',
    access: 'managed',
    source: 'panel',
    upstream: 'http://127.0.0.1:5212',
    resourceVersion: 'site-version',
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('AppsView domain binding', () => {
  it('keeps a successful domain visible and uses the refreshed app version for access policy', async () => {
    const created = proxySite()
    mocks.createSite.mockResolvedValueOnce(created)
    mocks.inventory.mockResolvedValueOnce(inventory('fresh-version'))
    mocks.action.mockRejectedValueOnce(new Error('active task'))
    const view = setupView()
    view.inventory.value = inventory('stale-version')
    view.selectedID.value = 'builtin-13'
    view.domain.value = created.primaryDomain

    await view.addDomain()

    expect(mocks.action).toHaveBeenCalledWith('builtin-13', 'direct_access', {
      resourceVersion: 'fresh-version',
      accessMode: 'domain_only',
    })
    expect(view.sites.value).toEqual([created])
    expect(view.domain.value).toBe('')
    expect(view.domainError.value).toBe('')
    expect(view.domainWarning.value).toContain('域名已绑定，但 IP + 端口访问策略未调整')
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      '域名已绑定',
      'cloud.example.com 已生效；直接访问策略可稍后单独调整。',
    )
  })
})
