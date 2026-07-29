import { createSSRApp, ssrContextKey, type ComputedRef, type Ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ClusterView from './ClusterView.vue'
import type { ClusterHost, ClusterHostList } from '@/types/api'

const mocks = vi.hoisted(() => ({
  hosts: vi.fn(),
  host: vi.fn(),
  add: vi.fn(),
  rename: vi.fn(),
  remove: vi.fn(),
  refresh: vi.fn(),
  createPairingCode: vi.fn(),
  controllers: vi.fn(),
  revokeController: vi.fn(),
  open: vi.fn(),
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
    cluster: {
      hosts: mocks.hosts,
      host: mocks.host,
      add: mocks.add,
      rename: mocks.rename,
      remove: mocks.remove,
      refresh: mocks.refresh,
      createPairingCode: mocks.createPairingCode,
      controllers: mocks.controllers,
      revokeController: mocks.revokeController,
    },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    danger: mocks.toastDanger,
  }),
}))

interface ClusterBindings {
  inventory: Ref<ClusterHostList | undefined>
  filteredHosts: ComputedRef<ClusterHost[]>
  search: Ref<string>
  manageOpen: Ref<boolean>
  selected: Ref<ClusterHost | undefined>
  load: (silent?: boolean) => Promise<void>
  openManage: (host: ClusterHost) => void
  removeHost: () => Promise<void>
  openPanel: (host: ClusterHost) => void
}

function setupView(): ClusterBindings {
  const component = ClusterView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => ClusterBindings
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

function host(id: string, isLocal: boolean, origin: string): ClusterHost {
  return {
    id,
    isLocal,
    name: isLocal ? '当前 KPanel' : '香港节点',
    origin,
    remoteNodeId: isLocal ? 'local-node' : 'remote-node',
    federationProtocol: 'v1',
    panelVersion: '0.27.0',
    state: 'online',
    lastSnapshot: {
      telemetry: {
        agentVersion: '0.27.0',
        agentProtocolVersion: 'v1',
        hostname: isLocal ? 'center' : 'hk-01',
        os: 'Debian GNU/Linux 13',
        osId: 'debian',
        osLike: ['linux'],
        architecture: 'amd64',
        uptimeSeconds: 3600,
        load: { one: 0.1, five: 0.2, fifteen: 0.3 },
        cpu: { cores: 2, usagePercent: 12.5 },
        memory: {
          totalBytes: 8 * 1024 ** 3,
          availableBytes: 6 * 1024 ** 3,
          usedBytes: 2 * 1024 ** 3,
          usagePercent: 25,
        },
        disk: {
          totalBytes: 100 * 1024 ** 3,
          usedBytes: 20 * 1024 ** 3,
          usagePercent: 20,
        },
        network: {
          receivedBytes: 1000,
          sentBytes: 2000,
          tcpConnections: 10,
          udpConnections: 2,
        },
        publicNetwork: {
          ipv4: isLocal ? '203.0.113.10' : '198.51.100.20',
          country: isLocal ? 'CN' : 'HK',
          countryCode: isLocal ? 'CN' : 'HK',
          city: isLocal ? 'Shanghai' : 'Hong Kong',
          isp: 'Example Network',
        },
        collectedAt: '2026-07-29T10:00:00Z',
      },
      receivedAt: '2026-07-29T10:00:01Z',
      latencyMilliseconds: isLocal ? 0 : 42,
      receiveBytesPerSecond: 1024,
      transmitBytesPerSecond: 2048,
    },
    lastAttemptAt: '2026-07-29T10:00:01Z',
    lastSuccessAt: '2026-07-29T10:00:01Z',
    consecutiveFailures: 0,
    polling: false,
    resourceVersion: `${id}-version`,
    createdAt: '2026-07-29T09:00:00Z',
    updatedAt: '2026-07-29T10:00:01Z',
  }
}

function inventory(): ClusterHostList {
  return {
    items: [
      host('local', true, 'https://stored-local.invalid'),
      host('remote', false, 'https://hk.example.com'),
    ],
    total: 2,
    remoteTotal: 1,
    maxHosts: 100,
    pollIntervalSeconds: 30,
    nodeId: 'local-node',
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('window', {
    location: { origin: 'https://center.example.com' },
    open: mocks.open,
    confirm: vi.fn().mockReturnValue(true),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ClusterView inventory and navigation', () => {
  it('uses one cached host-list request and does not fan out into per-host requests', async () => {
    let resolveHosts: ((value: ClusterHostList) => void) | undefined
    mocks.hosts.mockReturnValueOnce(
      new Promise<ClusterHostList>((resolve) => {
        resolveHosts = resolve
      }),
    )
    const view = setupView()

    const first = view.load()
    const overlapping = view.load(true)

    expect(mocks.hosts).toHaveBeenCalledOnce()
    expect(mocks.host).not.toHaveBeenCalled()
    resolveHosts?.(inventory())
    await Promise.all([first, overlapping])
    expect(view.inventory.value?.total).toBe(2)
  })

  it('opens the exact panel origin in a new isolated browser context', () => {
    const view = setupView()
    const current = host('local', true, 'https://stored-local.invalid')
    const remote = host('remote', false, 'https://hk.example.com')

    view.openPanel(remote)
    view.openPanel(current)

    expect(mocks.open).toHaveBeenNthCalledWith(
      1,
      'https://hk.example.com',
      '_blank',
      'noopener,noreferrer',
    )
    expect(mocks.open).toHaveBeenNthCalledWith(
      2,
      'https://center.example.com',
      '_blank',
      'noopener,noreferrer',
    )
  })

  it('keeps the local node visibly marked and outside remote removal management', async () => {
    const view = setupView()
    const list = inventory()
    const local = list.items[0]
    if (!local) throw new Error('local fixture is missing')
    view.inventory.value = list

    view.search.value = '当前面板'
    expect(view.filteredHosts.value).toHaveLength(1)
    expect(view.filteredHosts.value[0]).toMatchObject({ id: 'local', isLocal: true })

    view.openManage(local)
    await view.removeHost()

    expect(view.manageOpen.value).toBe(false)
    expect(view.selected.value).toBeUndefined()
    expect(mocks.remove).not.toHaveBeenCalled()
    expect(view.inventory.value?.items.some((item) => item.isLocal)).toBe(true)
  })
})
