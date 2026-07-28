import { createSSRApp, ssrContextKey, type Ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppsView from './AppsView.vue'
import { ApiError } from '@/lib/api'
import type { AppInstallJob, AppMarketInventory, Site } from '@/types/api'

const mocks = vi.hoisted(() => ({
  createSite: vi.fn(),
  listSites: vi.fn(),
  inventory: vi.fn(),
  action: vi.fn(),
  checkUpdate: vi.fn(),
  job: vi.fn(),
  publicNetwork: vi.fn(),
  toastSuccess: vi.fn(),
  toastDanger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {
    readonly status: number
    readonly code: string

    constructor(message: string, status = 0, code = 'request_failed') {
      super(message)
      this.status = status
      this.code = code
    }
  },
  api: {
    apps: {
      inventory: mocks.inventory,
      action: mocks.action,
      checkUpdate: mocks.checkUpdate,
      install: vi.fn(),
      job: mocks.job,
      jobs: vi.fn(),
    },
    sites: {
      create: mocks.createSite,
      list: mocks.listSites,
      remove: vi.fn(),
    },
    system: {
      publicNetwork: mocks.publicNetwork,
    },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    danger: mocks.toastDanger,
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
  sitesWarning: Ref<string>
  checkedUpdates: Ref<Record<string, 'available' | 'current'>>
  activeJob: Ref<AppInstallJob | undefined>
  jobDetailsOpen: Ref<boolean>
  load: (silent?: boolean) => Promise<void>
  checkUpdate: () => Promise<void>
  addDomain: () => Promise<void>
  openScriptManage: () => Promise<void>
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
          containerId: 'a'.repeat(64),
          detectedBy: ['container'],
        },
        capabilities: {
          add_domain: { enabled: true },
          check_update: { enabled: true },
          direct_access: { enabled: true },
          manage: { enabled: true },
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
  const storage = new Map<string, string>()
  vi.stubGlobal('window', {
    localStorage: {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    },
    setInterval: vi.fn(() => 1),
    clearInterval: vi.fn(),
    location: { hostname: 'localhost' },
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
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

  it('retries a transient site-list failure without hiding the bound domain', async () => {
    const created = proxySite()
    mocks.inventory.mockResolvedValueOnce(inventory('current-version'))
    mocks.listSites
      .mockRejectedValueOnce(new Error('temporary site read failure'))
      .mockResolvedValueOnce({ items: [created], total: 1 })
    mocks.publicNetwork.mockResolvedValueOnce(undefined)
    const view = setupView()

    await view.load(true)

    expect(mocks.listSites).toHaveBeenCalledTimes(2)
    expect(view.sites.value).toEqual([created])
    expect(view.sitesWarning.value).toBe('')
  })

  it('keeps the last successful domain snapshot when both site reads fail', async () => {
    const created = proxySite()
    mocks.inventory.mockResolvedValueOnce(inventory('current-version'))
    mocks.listSites.mockRejectedValue(new Error('site service unavailable'))
    mocks.publicNetwork.mockResolvedValueOnce(undefined)
    const view = setupView()
    view.sites.value = [created]

    await view.load(true)

    expect(mocks.listSites).toHaveBeenCalledTimes(2)
    expect(view.sites.value).toEqual([created])
    expect(view.sitesWarning.value).toContain('当前显示上次成功读取的结果')
  })
})

describe('AppsView update checks', () => {
  it('refreshes the inventory and retries a read-only resource conflict once', async () => {
    mocks.checkUpdate
      .mockRejectedValueOnce(new ApiError('container resourceVersion changed', 409, 'resource_conflict'))
      .mockResolvedValueOnce({
        containerId: 'a'.repeat(64),
        image: 'cloudreve/cloudreve:latest',
        status: 'current',
        updateAvailable: false,
        resourceVersion: 'fresh-version',
        checkedAt: '2026-07-28T00:00:00Z',
      })
    mocks.inventory.mockResolvedValueOnce(inventory('fresh-version'))
    const view = setupView()
    view.inventory.value = inventory('stale-version')
    view.selectedID.value = 'builtin-13'

    await view.checkUpdate()

    expect(mocks.checkUpdate).toHaveBeenNthCalledWith(1, 'builtin-13', 'stale-version')
    expect(mocks.checkUpdate).toHaveBeenNthCalledWith(2, 'builtin-13', 'fresh-version')
    expect(view.checkedUpdates.value['builtin-13']).toBe('current')
    expect(mocks.toastDanger).not.toHaveBeenCalled()
  })
})

describe('AppsView script management', () => {
  it('opens the fixed-selector interactive management job with the current resource version', async () => {
    const job: AppInstallJob = {
      id: '0123456789abcdef0123456789abcdef',
      appId: 'builtin-13',
      appName: 'Cloudreve',
      action: 'manage',
      interactive: true,
      inputOpen: true,
      status: 'running',
      stage: 'interactive',
      progress: 5,
      logs: [],
      createdAt: '2026-07-28T00:00:00Z',
    }
    mocks.action.mockResolvedValueOnce(job)
    mocks.job.mockResolvedValue(job)
    const view = setupView()
    view.inventory.value = inventory('fresh-version')
    view.selectedID.value = 'builtin-13'

    await view.openScriptManage()

    expect(mocks.action).toHaveBeenCalledWith('builtin-13', 'manage', {
      resourceVersion: 'fresh-version',
    })
    expect(view.activeJob.value).toEqual(job)
    expect(view.jobDetailsOpen.value).toBe(true)
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      '脚本管理终端已打开',
      'Cloudreve 正在使用固定应用编号进入 kejilion.sh 原生菜单。',
    )
  })
})
