import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, normalizeList, resetApiSecurityState } from './api'

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  const headers = new Headers(init.headers)
  headers.set('content-type', headers.get('content-type') || 'application/json')
  return new Response(JSON.stringify(body), { ...init, headers })
}

afterEach(() => {
  vi.unstubAllGlobals()
  resetApiSecurityState()
})

describe('API client', () => {
  it('detects a server that still requires bootstrap', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(jsonResponse({ required: true }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.auth.status()).resolves.toMatchObject({
      setupRequired: true,
      authenticated: false,
    })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/auth/bootstrap',
      expect.objectContaining({ credentials: 'same-origin', cache: 'no-store' }),
    )
  })

  it('keeps the CSRF token in memory and sends it on mutations', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ required: false }))
      .mockResolvedValueOnce(
        jsonResponse({
          user: { id: 'user-1', username: 'admin', role: 'administrator' },
          csrfToken: 'csrf-secret',
          expiresAt: '2026-07-25T12:00:00Z',
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.auth.status()).resolves.toMatchObject({
      setupRequired: false,
      authenticated: true,
      user: { username: 'admin' },
    })
    await api.auth.logout()

    const logoutInit = fetchMock.mock.calls[2]?.[1] as RequestInit
    expect(new Headers(logoutInit.headers).get('x-csrf-token')).toBe('csrf-secret')
    expect(logoutInit.method).toBe('POST')
  })

  it('parses the stable problem+json error contract', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValueOnce(
        jsonResponse(
          {
            title: '请求冲突',
            status: 409,
            code: 'resource_version_conflict',
            detail: '资源已被外部修改。',
            requestId: 'req-123',
          },
          { status: 409, headers: { 'content-type': 'application/problem+json' } },
        ),
      ),
    )

    const error = await api.sites.list().catch((reason: unknown) => reason)
    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({
      status: 409,
      code: 'resource_version_conflict',
      message: '资源已被外部修改。',
      requestId: 'req-123',
    })
  })

  it('normalizes site create and update responses from the Agent contract', async () => {
    const createdRaw = {
      id: 'a'.repeat(32),
      primaryDomain: 'example.com',
      domains: ['example.com', 'www.example.com'],
      kind: 'reverse_proxy',
      enabled: true,
      health: 'healthy',
      tls: {
        enabled: true,
        status: 'valid',
        expiresAt: '2026-12-31T00:00:00Z',
        source: 'acme',
      },
      target: 'http://127.0.0.1:3000',
      origin: 'web',
      consistency: 'in_sync',
      resourceVersion: `sha256:${'b'.repeat(64)}`,
      allowedActions: ['update'],
      artifacts: [{ kind: 'nginx', path: '/etc/nginx/conf.d/example.com.conf', hash: 'abc' }],
      warnings: [],
      reconciledAt: '2026-07-25T10:00:00Z',
    }
    const updatedRaw = {
      ...createdRaw,
      domains: ['example.com'],
      target: 'http://127.0.0.1:4000',
      resourceVersion: `sha256:${'c'.repeat(64)}`,
      reconciledAt: '2026-07-25T10:05:00Z',
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(createdRaw))
      .mockResolvedValueOnce(jsonResponse(updatedRaw))
    vi.stubGlobal('fetch', fetchMock)

    const created = await api.sites.create({
      primaryDomain: 'example.com',
      aliases: ['www.example.com'],
      type: 'proxy',
      upstream: 'http://127.0.0.1:3000',
      enabled: true,
    })
    const updated = await api.sites.update(createdRaw.id, {
      primaryDomain: 'example.com',
      aliases: [],
      type: 'proxy',
      upstream: 'http://127.0.0.1:4000',
      enabled: true,
      expectedResourceVersion: createdRaw.resourceVersion,
    })

    expect(created).toMatchObject({
      id: createdRaw.id,
      type: 'proxy',
      upstream: createdRaw.target,
      consistency: 'synced',
      access: 'managed',
      source: 'panel',
      certificate: {
        status: 'valid',
        issuer: 'acme',
        expiresAt: '2026-12-31T00:00:00Z',
      },
    })
    expect(updated).toMatchObject({
      domains: ['example.com'],
      upstream: updatedRaw.target,
      resourceVersion: updatedRaw.resourceVersion,
      observedAt: updatedRaw.reconciledAt,
    })

    const createInit = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/sites')
    expect(createInit.method).toBe('POST')
    expect(JSON.parse(String(createInit.body))).not.toHaveProperty('expectedResourceVersion')

    const updateInit = fetchMock.mock.calls[1]?.[1] as RequestInit
    expect(fetchMock.mock.calls[1]?.[0]).toBe(`/api/v1/sites/${createdRaw.id}`)
    expect(updateInit.method).toBe('PATCH')
    expect(JSON.parse(String(updateInit.body))).toMatchObject({
      upstream: 'http://127.0.0.1:4000',
      expectedResourceVersion: createdRaw.resourceVersion,
    })
  })

  it('sends only the typed system action payload through the protected mutation path', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      jsonResponse({
        action: 'dns',
        status: 'succeeded',
        changed: true,
        message: 'DNS 已更新',
        appliedAt: '2026-07-26T03:00:00Z',
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.system.action({ action: 'dns', servers: ['1.1.1.1', '8.8.8.8'] })).resolves.toMatchObject({
      action: 'dns',
      changed: true,
    })
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/system/actions')
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({
      action: 'dns',
      servers: ['1.1.1.1', '8.8.8.8'],
    })
  })

  it('submits only the selected system cleanup policy', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      jsonResponse({
        action: 'cleanup',
        status: 'succeeded',
        changed: true,
        message: 'queued',
        appliedAt: '2026-07-26T03:00:00Z',
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await api.system.action({ action: 'cleanup', maintenancePolicy: 'standard' })

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      action: 'cleanup',
      maintenancePolicy: 'standard',
    })
  })

  it('keeps the overview compatible with an older Agent without management fields', async () => {
    const collectedAt = '2026-07-25T10:00:00Z'
    const system = {
      hostname: 'legacy-host',
      os: 'Debian 12',
      kernel: '6.1.0',
      architecture: 'amd64',
      uptimeSeconds: 120,
      load: { one: 0.2, five: 0.1, fifteen: 0.1 },
      cpu: { cores: 2, usagePercent: 5 },
      memory: {
        totalBytes: 1024,
        availableBytes: 512,
        usedBytes: 512,
        usagePercent: 50,
        swapTotalBytes: 256,
        swapUsedBytes: 64,
      },
      disks: [],
      network: { receivedBytes: 100, sentBytes: 200 },
      collectedAt,
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(system))
      .mockResolvedValueOnce(
        jsonResponse({
          status: 'ok',
          version: '0.1.3',
          protocolVersion: 'v1',
          readOnly: false,
          checkedAt: collectedAt,
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: 'system.read', enabled: true }] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ available: false, containers: 0, running: 0, stopped: 0, images: 0, collectedAt }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
    vi.stubGlobal('fetch', fetchMock)

    const overview = await api.overview.get()

    expect(overview.hostname).toBe('legacy-host')
    expect(overview.management).toMatchObject({
      ssh: { ports: [], source: 'unknown' },
      dns: { servers: [], manager: 'unknown' },
      swap: {
        totalBytes: 256,
        usedBytes: 64,
        activeDevices: 0,
        path: '/swapfile',
        fileExists: false,
        fileActive: false,
        legacyExists: false,
        legacyActive: false,
        otherActiveDevices: 0,
      },
      packageSources: [],
      maintenance: { state: 'idle', progress: 0, rebootRequired: false },
      ipPreference: 'unknown',
      bbr: { supported: false, enabled: false, available: [] },
      capabilities: { 'system.read': { enabled: true } },
    })
  })

  it('normalizes kejilion.sh and legacy swap artifacts separately', async () => {
    const collectedAt = '2026-07-26T05:00:00Z'
    const system = {
      hostname: 'swap-host',
      os: 'Debian 13',
      architecture: 'amd64',
      uptimeSeconds: 120,
      load: { one: 0.2, five: 0.1, fifteen: 0.1 },
      cpu: { cores: 2, usagePercent: 5 },
      memory: {
        totalBytes: 8 * 1024 ** 3,
        availableBytes: 6 * 1024 ** 3,
        usedBytes: 2 * 1024 ** 3,
        usagePercent: 25,
        swapTotalBytes: 3 * 1024 ** 3,
        swapUsedBytes: 128 * 1024,
      },
      disks: [],
      network: { receivedBytes: 100, sentBytes: 200 },
      management: {
        swap: {
          activeDevices: 3,
          path: '/swapfile',
          fileExists: true,
          fileActive: true,
          fileSizeBytes: 1024 ** 3,
          fileUsedBytes: 128 * 1024,
          legacyExists: true,
          legacyActive: true,
          legacySizeBytes: 2 * 1024 ** 3,
          otherActiveDevices: 1,
          otherSwapTotalBytes: 512 * 1024 ** 2,
          otherSwapUsedBytes: 0,
        },
      },
      collectedAt,
    }
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(system))
      .mockResolvedValueOnce(
        jsonResponse({
          status: 'ok',
          version: '0.5.1',
          protocolVersion: 'v1alpha1',
          readOnly: false,
          checkedAt: collectedAt,
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: 'system.swap.write', enabled: true }] }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
      .mockResolvedValueOnce(jsonResponse({ available: false, containers: 0, running: 0, stopped: 0, images: 0, collectedAt }))
      .mockResolvedValueOnce(jsonResponse({ items: [] }))
    vi.stubGlobal('fetch', fetchMock)

    const overview = await api.overview.get()

    expect(overview.management.swap).toMatchObject({
      totalBytes: 3 * 1024 ** 3,
      activeDevices: 3,
      path: '/swapfile',
      fileExists: true,
      fileActive: true,
      fileSizeBytes: 1024 ** 3,
      legacyExists: true,
      legacyActive: true,
      legacySizeBytes: 2 * 1024 ** 3,
      otherActiveDevices: 1,
      otherSwapTotalBytes: 512 * 1024 ** 2,
    })
  })

  it('uses only the supported jobs limit query and normalizes job records', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      jsonResponse({
        items: [
          {
            id: 'req-123',
            action: 'docker.restart',
            origin: 'web',
            state: 'succeeded',
            progress: 100,
            stage: 'completed',
            targetKind: 'container',
            targetId: 'abc123',
            targetLabel: 'nginx',
            createdAt: '2026-07-25T10:00:00Z',
            startedAt: '2026-07-25T10:00:01Z',
            finishedAt: '2026-07-25T10:00:02Z',
          },
        ],
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await api.jobs.list({ limit: 3 })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/jobs?limit=3')
    expect(result).toMatchObject({
      total: 1,
      items: [
        {
          id: 'req-123',
          action: 'docker.restart',
          resourceType: 'container',
          resourceName: 'nginx',
          status: 'succeeded',
          progress: 100,
          actor: 'web',
          source: 'web',
          stages: [{ name: 'completed', status: 'succeeded' }],
        },
      ],
    })
  })

  it('sends the observed resource version with Docker lifecycle actions', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ required: false }))
      .mockResolvedValueOnce(
        jsonResponse({
          user: { id: 'user-1', username: 'admin', role: 'admin' },
          csrfToken: 'csrf-secret',
          expiresAt: '2026-07-25T12:00:00Z',
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ containerId: 'abc123def456', action: 'restart', status: 'completed' }))
    vi.stubGlobal('fetch', fetchMock)

    await api.auth.status()
    await api.docker.action('abc123def456', 'restart', `sha256:${'a'.repeat(64)}`)

    const actionInit = fetchMock.mock.calls[2]?.[1] as RequestInit
    expect(actionInit.method).toBe('POST')
    expect(new Headers(actionInit.headers).get('x-csrf-token')).toBe('csrf-secret')
    expect(JSON.parse(String(actionInit.body))).toEqual({
      resourceVersion: `sha256:${'a'.repeat(64)}`,
    })
  })

  it('normalizes list responses without a total field', () => {
    expect(normalizeList({ items: ['a', 'b'] } as { items: string[]; total: number })).toEqual({
      items: ['a', 'b'],
      total: 2,
    })
  })
})
