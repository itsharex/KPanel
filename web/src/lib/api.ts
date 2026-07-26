import type {
  ApiList,
  AgentStatus,
  AppMarketInventory,
  AppImageUpdateResult,
  AppMutationResult,
  AuditEvent,
  AuthSession,
  AuthStatus,
  DockerInventory,
  DockerActionResult,
  Job,
  AppInstallJob,
  LoginRequest,
  PanelSettings,
  SetupRequest,
  Site,
  SiteInput,
  SystemActionInput,
  SystemActionResult,
  SystemOverview,
} from '@/types/api'

type QueryValue = string | number | boolean | undefined
interface ApiEnvelope<T> {
  data?: T
  csrfToken?: string
  error?: {
    code?: string
    message?: string
    details?: unknown
  }
  message?: string
}

interface ProblemPayload {
  title?: string
  status?: number
  code?: string
  detail?: string
  requestId?: string
  retryable?: boolean
  fieldErrors?: Record<string, string>
}

interface RawAgentHealth {
  status: string
  version?: string
  protocolVersion?: string
  readOnly?: boolean
  reasons?: string[]
  checkedAt?: string
}

interface RawSystemSummary {
  hostname: string
  os: string
  kernel?: string
  architecture?: string
  uptimeSeconds: number
  load: { one: number; five: number; fifteen: number }
  cpu: { model?: string; cores: number; frequencyMHz?: number; usagePercent: number }
  memory: {
    totalBytes: number
    availableBytes: number
    usedBytes: number
    usagePercent: number
    swapTotalBytes?: number
    swapUsedBytes?: number
  }
  disks: Array<{
    device: string
    mountPoint: string
    fileSystem: string
    totalBytes: number
    usedBytes: number
    usagePercent: number
  }>
  network: {
    receivedBytes: number
    sentBytes: number
    tcpConnections?: number
    udpConnections?: number
  }
  publicNetwork?: {
    ipv4?: string
    ipv6?: string
    isp?: string
    country?: string
    region?: string
    city?: string
    timezone?: string
    source?: string
    updatedAt?: string
  }
  management?: {
    ssh?: { ports?: number[]; source?: string }
    dns?: { servers?: string[]; manager?: string }
    timezone?: string
    swap?: {
      activeDevices?: number
      path?: string
      fileExists?: boolean
      fileActive?: boolean
      fileSizeBytes?: number
      fileUsedBytes?: number
      legacyExists?: boolean
      legacyActive?: boolean
      legacySizeBytes?: number
      otherActiveDevices?: number
      otherSwapTotalBytes?: number
      otherSwapUsedBytes?: number
    }
    packageManager?: string
    packageSources?: string[]
    maintenance?: {
      id?: string
      state?: string
      action?: string
      policy?: string
      stage?: string
      progress?: number
      message?: string
      startedAt?: string
      finishedAt?: string
      rebootRequired?: boolean
    }
    ipPreference?: string
    kernelOptimization?: { enabled?: boolean; profile?: string; source?: string }
    bbr?: {
      supported?: boolean
      enabled?: boolean
      congestionControl?: string
      defaultQDisc?: string
      available?: string[]
    }
  }
  collectedAt: string
}

interface RawSite {
  id: string
  primaryDomain: string
  domains?: string[]
  kind: string
  enabled: boolean
  health?: string
  tls?: {
    enabled: boolean
    status?: string
    expiresAt?: string
    source?: string
  }
  target?: string
  documentRoot?: string
  origin?: string
  consistency?: string
  resourceVersion?: string
  allowedActions?: string[]
  warnings?: string[]
  artifacts?: Array<{ kind: string; path: string; hash?: string }>
  reconciledAt?: string
}

interface RawDockerSummary {
  available: boolean
  serverVersion?: string
  containers: number
  running: number
  paused?: number
  stopped: number
  images: number
  collectedAt: string
}

interface RawContainer {
  id: string
  name: string
  image: string
  state: string
  status?: string
  health?: string
  ports?: Array<{
    privatePort: number
    publicPort?: number
    ip?: string
    type?: string
    protocol?: string
  }>
  composeProject?: string
  ownership?: string
  resourceVersion?: string
  allowedActions?: string[]
}

interface RawJob {
  id: string
  action: string
  origin?: string
  state: string
  progress?: number
  stage?: string
  targetKind?: string
  targetId?: string
  targetLabel?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
  error?: ProblemPayload
}

interface RawAuditEvent {
  id: string
  occurredAt: string
  actorType?: string
  actorId?: string
  sourceIp?: string
  action: string
  targetKind?: string
  targetId?: string
  result?: string
  requestId?: string
}

interface RawWordPressInstallJob {
  id: string
  domain: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  message?: string
  site?: RawSite
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: unknown
  readonly requestId?: string

  constructor(message: string, status = 0, code = 'request_failed', details?: unknown, requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
    this.requestId = requestId
  }
}

let csrfToken = ''
let previousNetworkSample:
  | { receivedBytes: number; sentBytes: number; collectedAtMs: number }
  | undefined

function networkRates(system: RawSystemSummary): { receive: number; transmit: number } {
  const current = {
    receivedBytes: system.network.receivedBytes,
    sentBytes: system.network.sentBytes,
    collectedAtMs: Date.parse(system.collectedAt),
  }
  const previous = previousNetworkSample
  previousNetworkSample = current
  if (
    !previous ||
    !Number.isFinite(current.collectedAtMs) ||
    current.collectedAtMs <= previous.collectedAtMs ||
    current.receivedBytes < previous.receivedBytes ||
    current.sentBytes < previous.sentBytes
  ) {
    return { receive: 0, transmit: 0 }
  }
  const elapsedSeconds = (current.collectedAtMs - previous.collectedAtMs) / 1_000
  return {
    receive: (current.receivedBytes - previous.receivedBytes) / elapsedSeconds,
    transmit: (current.sentBytes - previous.sentBytes) / elapsedSeconds,
  }
}

function buildUrl(path: string, query?: Record<string, QueryValue>): string {
  const base = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/+$/, '')
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  const url = `${base}${normalizedPath}`
  if (!query) return url

  const params = new URLSearchParams()
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined) params.set(key, String(value))
  })
  const serialized = params.toString()
  return serialized ? `${url}?${serialized}` : url
}

function pickCsrfToken(headers: Headers, payload: unknown): void {
  const headerToken = headers.get('x-csrf-token')
  if (headerToken) {
    csrfToken = headerToken
    return
  }

  if (payload && typeof payload === 'object') {
    const envelope = payload as ApiEnvelope<unknown>
    if (typeof envelope.csrfToken === 'string') csrfToken = envelope.csrfToken
    const data = envelope.data
    if (data && typeof data === 'object' && 'csrfToken' in data) {
      const nestedToken = (data as { csrfToken?: unknown }).csrfToken
      if (typeof nestedToken === 'string') csrfToken = nestedToken
    }
  }
}

async function parsePayload(response: Response): Promise<unknown> {
  if (response.status === 204) return undefined
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('json')) {
    const text = await response.text()
    return text || undefined
  }
  return response.json()
}

async function request<T>(
  path: string,
  options: {
    method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
    body?: unknown
    query?: Record<string, QueryValue>
    signal?: AbortSignal
  } = {},
): Promise<T> {
  const method = options.method || 'GET'
  const headers = new Headers({ Accept: 'application/json' })
  if (options.body !== undefined) headers.set('Content-Type', 'application/json')
  if (method !== 'GET' && csrfToken) headers.set('X-CSRF-Token', csrfToken)

  let response: Response
  try {
    response = await fetch(buildUrl(path, options.query), {
      method,
      credentials: 'same-origin',
      cache: 'no-store',
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: options.signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new ApiError('无法连接到面板服务，请检查服务状态后重试。', 0, 'network_error', error)
  }

  const payload = await parsePayload(response)
  if (path === '/auth/bootstrap' || path === '/auth/login' || path === '/auth/session') {
    pickCsrfToken(response.headers, payload)
  }

  if (!response.ok) {
    const envelope = payload && typeof payload === 'object' ? (payload as ApiEnvelope<unknown>) : undefined
    const problem = payload && typeof payload === 'object' ? (payload as ProblemPayload) : undefined
    const message =
      envelope?.error?.message ||
      envelope?.message ||
      problem?.detail ||
      problem?.title ||
      (typeof payload === 'string' ? payload : '') ||
      `请求失败（HTTP ${response.status}）`
    throw new ApiError(
      message,
      response.status,
      envelope?.error?.code || problem?.code || `http_${response.status}`,
      envelope?.error?.details || problem?.fieldErrors,
      problem?.requestId || response.headers.get('x-request-id') || undefined,
    )
  }

  if (payload && typeof payload === 'object' && 'data' in payload) {
    return (payload as ApiEnvelope<T>).data as T
  }
  return payload as T
}

export function normalizeList<T>(value: ApiList<T> | T[] | undefined): ApiList<T> {
  if (Array.isArray(value)) return { items: value, total: value.length }
  if (!value) return { items: [], total: 0 }
  return { ...value, total: Number.isFinite(value.total) ? value.total : value.items.length }
}

function normalizeAgent(raw: RawAgentHealth): AgentStatus {
  const status = raw.status?.toLowerCase() || 'unknown'
  const connected = !['offline', 'unavailable', 'unreachable', 'error'].includes(status)
  const compatible = !['incompatible', 'protocol_mismatch'].includes(status)
  return {
    connected,
    compatible,
    readOnly: Boolean(raw.readOnly || !connected || !compatible),
    version: raw.version,
    protocolVersion: raw.protocolVersion,
    lastSeenAt: raw.checkedAt,
    reason: raw.reasons?.join('；'),
  }
}

async function createSite(
  body: SiteInput,
  onProgress?: (status: string, message: string) => void,
): Promise<Site> {
  const result = await request<RawSite | RawWordPressInstallJob>('/sites', { method: 'POST', body })
  if (body.type !== 'wordpress' || !('status' in result) || !('id' in result)) {
    return normalizeSite(result as RawSite)
  }
  let job = result as RawWordPressInstallJob
  for (let attempt = 0; attempt <= 900; attempt += 1) {
    onProgress?.(job.status, job.message || 'WordPress 安装任务正在执行。')
    if (job.status === 'succeeded') {
      if (!job.site) throw new ApiError('WordPress 已完成，但网站对账结果缺失。', 503, 'wordpress_result_missing')
      return normalizeSite(job.site)
    }
    if (job.status === 'failed') {
      throw new ApiError(
        job.message || 'WordPress 安装失败，已尝试回滚本次新建产物。',
        422,
        'wordpress_install_failed',
      )
    }
    if (attempt === 900) break
    await new Promise((resolve) => setTimeout(resolve, 2_000))
    job = await request<RawWordPressInstallJob>(
      `/site-installations/${encodeURIComponent(job.id)}`,
    )
  }
  throw new ApiError('WordPress 安装状态等待超时，请在网站列表中核对实际产物。', 504, 'wordpress_install_timeout')
}

function normalizeSite(raw: RawSite): Site {
  const kindMap: Record<string, Site['type']> = {
    static: 'static',
    reverse_proxy: 'proxy',
    proxy: 'proxy',
    domain_proxy: 'proxy_domain',
    load_balance: 'load_balance',
    php: 'php',
    wordpress: 'wordpress',
    redirect: 'redirect',
  }
  const consistencyMap: Record<string, Site['consistency']> = {
    in_sync: 'synced',
    drifted: 'drifted',
    ambiguous: 'ambiguous',
    conflicted: 'conflicted',
    unsupported: 'unsupported',
    read_only: 'read_only',
  }
  const sourceMap: Record<string, Site['source']> = {
    web: 'panel',
    panel: 'panel',
    cli: 'kejilion',
    discovered: 'kejilion',
    external: 'external',
  }
  const actions = raw.allowedActions || []
  const certificateStatus = raw.tls?.enabled ? raw.tls.status || 'unknown' : 'missing'
  return {
    id: raw.id,
    primaryDomain: raw.primaryDomain,
    domains: raw.domains || [raw.primaryDomain],
    type: kindMap[raw.kind] || 'unknown',
    enabled: raw.enabled,
    health: (['healthy', 'warning', 'critical'].includes(raw.health || '')
      ? raw.health
      : 'unknown') as Site['health'],
    consistency: consistencyMap[raw.consistency || ''] || 'unknown',
    access:
      actions.length > 0 || (raw.origin === 'web' && raw.consistency === 'in_sync')
        ? 'managed'
        : raw.origin === 'external'
          ? 'unmanaged'
          : 'read-only',
    source: sourceMap[raw.origin || ''] || 'unknown',
    rootPath: raw.documentRoot,
    upstream: raw.target,
    certificate: {
      status: (['valid', 'expiring', 'expired', 'missing'].includes(certificateStatus)
        ? certificateStatus
        : 'unknown') as NonNullable<Site['certificate']>['status'],
      issuer: raw.tls?.source,
      expiresAt: raw.tls?.expiresAt,
    },
    resourceVersion: raw.resourceVersion || '',
    observedAt: raw.reconciledAt,
    reason: raw.warnings?.[0],
    warnings: raw.warnings,
    allowedActions: actions,
    artifacts: raw.artifacts,
  }
}

function normalizeContainer(raw: RawContainer): DockerInventory['containers'][number] {
  const actions = raw.allowedActions || []
  const state = ['running', 'paused', 'restarting', 'exited', 'dead', 'created'].includes(raw.state)
    ? raw.state
    : 'unknown'
  return {
    id: raw.id,
    name: raw.name,
    image: raw.image,
    state: state as DockerInventory['containers'][number]['state'],
    health: (['healthy', 'warning', 'critical'].includes(raw.health || '') ? raw.health : 'unknown') as
      | 'healthy'
      | 'warning'
      | 'critical'
      | 'unknown',
    access: actions.length > 0 ? 'managed' : raw.ownership === 'external' ? 'unmanaged' : 'read-only',
    consistency: raw.ownership === 'ambiguous' ? 'ambiguous' : 'synced',
    project: raw.composeProject,
    ports: (raw.ports || []).map((port) => ({
      privatePort: port.privatePort,
      publicPort: port.publicPort,
      ip: port.ip,
      protocol: (port.protocol || port.type || 'tcp') as 'tcp' | 'udp' | 'sctp',
    })),
    allowedActions: actions,
    resourceVersion: raw.resourceVersion,
    statusText: raw.status,
  }
}

function normalizeJob(raw: RawJob): Job {
  const knownStates: Job['status'][] = [
    'queued',
    'running',
    'succeeded',
    'failed_rolled_back',
    'failed_needs_attention',
    'interrupted',
    'cancelled',
  ]
  return {
    id: raw.id,
    action: raw.action,
    resourceType: raw.targetKind,
    resourceName: raw.targetLabel || raw.targetId,
    status: knownStates.includes(raw.state as Job['status']) ? (raw.state as Job['status']) : 'failed',
    progress: raw.progress,
    actor: raw.origin,
    source: (['web', 'cli', 'reconcile', 'system'].includes(raw.origin || '') ? raw.origin : 'system') as Job['source'],
    createdAt: raw.createdAt,
    startedAt: raw.startedAt,
    finishedAt: raw.finishedAt,
    errorCode: raw.error?.code,
    errorMessage: raw.error?.detail || raw.error?.title,
    stages: raw.stage
      ? [
          {
            name: raw.stage,
            status: raw.state === 'running' ? 'running' : raw.state === 'succeeded' ? 'succeeded' : 'failed',
          },
        ]
      : undefined,
  }
}

export const api = {
  auth: {
    status: async (signal?: AbortSignal): Promise<AuthStatus> => {
      const bootstrap = await request<{ required: boolean }>('/auth/bootstrap', { signal })
      if (bootstrap.required) return { setupRequired: true, authenticated: false }
      try {
        const session = await request<AuthSession>('/auth/session', { signal })
        return {
          setupRequired: false,
          authenticated: true,
          user: session.user,
          csrfToken: session.csrfToken,
          expiresAt: session.expiresAt,
        }
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          return { setupRequired: false, authenticated: false }
        }
        throw error
      }
    },
    setup: async (body: SetupRequest): Promise<AuthStatus> => {
      const session = await request<AuthSession>('/auth/bootstrap', { method: 'POST', body })
      return {
        setupRequired: false,
        authenticated: true,
        user: session.user,
        csrfToken: session.csrfToken,
        expiresAt: session.expiresAt,
      }
    },
    login: async (body: LoginRequest): Promise<AuthStatus> => {
      const session = await request<AuthSession>('/auth/login', { method: 'POST', body })
      return {
        setupRequired: false,
        authenticated: true,
        user: session.user,
        csrfToken: session.csrfToken,
        expiresAt: session.expiresAt,
      }
    },
    logout: () => request<void>('/auth/logout', { method: 'POST' }),
  },
  agent: {
    health: async (signal?: AbortSignal) => normalizeAgent(await request<RawAgentHealth>('/agent/health', { signal })),
    capabilities: async (signal?: AbortSignal) => {
      type Capability = { id: string; enabled: boolean; reason?: string; methods?: string[] }
      const result = await request<ApiList<Capability> | Capability[]>('/capabilities', { signal })
      return normalizeList(result).items
    },
  },
  overview: {
    get: async (signal?: AbortSignal): Promise<SystemOverview> => {
      type Capability = { id: string; enabled: boolean; reason?: string; methods?: string[] }
      const [system, agent, capabilitiesResult, sitesResult, dockerResult, containersResult] = await Promise.all([
        request<RawSystemSummary>('/system/summary', { signal }),
        request<RawAgentHealth>('/agent/health', { signal }),
        request<ApiList<Capability> | Capability[]>('/capabilities', { signal }).catch(() => []),
        request<ApiList<RawSite> | RawSite[]>('/sites', { signal }).catch(() => []),
        request<RawDockerSummary>('/docker/summary', { signal }).catch(() => undefined),
        request<ApiList<RawContainer> | RawContainer[]>('/docker/containers', { signal }).catch(() => []),
      ])
      const rootDisk = system.disks.find((disk) => disk.mountPoint === '/') || system.disks[0]
      const sites = normalizeList(sitesResult).items.map(normalizeSite)
      const containers = normalizeList(containersResult).items
      const capabilities = Object.fromEntries(
        normalizeList(capabilitiesResult).items.map((capability) => [
          capability.id,
          {
            enabled: capability.enabled,
            reason: capability.reason,
            methods: capability.methods,
          },
        ]),
      )
      const rates = networkRates(system)
      const knownServices = [
        { id: 'nginx', name: 'Nginx' },
        { id: 'mysql', name: 'MySQL' },
        { id: 'php', name: 'PHP' },
        { id: 'php74', name: 'PHP 7.4' },
        { id: 'redis', name: 'Redis' },
      ]
      const services: SystemOverview['services'] = dockerResult
        ? [
            {
              id: 'docker',
              name: 'Docker Engine',
              state: dockerResult.available ? 'running' : 'stopped',
              version: dockerResult.serverVersion,
            },
            ...knownServices.flatMap((known) => {
              const container = containers.find((item) => item.name.replace(/^\/+/, '') === known.id)
              if (!container) return []
              const state: SystemOverview['services'][number]['state'] =
                container.state === 'running'
                  ? 'running'
                  : container.state === 'paused' || container.state === 'restarting'
                    ? 'degraded'
                    : ['exited', 'dead', 'created'].includes(container.state)
                      ? 'stopped'
                      : 'unknown'
              return [{ id: known.id, name: known.name, state, detail: container.image }]
            }),
          ]
        : []
      return {
        hostname: system.hostname,
        os: system.os,
        kernel: system.kernel,
        architecture: system.architecture,
        uptimeSeconds: system.uptimeSeconds,
        observedAt: system.collectedAt,
        cpu: {
          value: system.cpu.usagePercent,
          percent: system.cpu.usagePercent,
          unit: '%',
          model: system.cpu.model,
          cores: system.cpu.cores,
          frequencyMHz: system.cpu.frequencyMHz,
        },
        memory: {
          value: system.memory.usedBytes,
          total: system.memory.totalBytes,
          percent: system.memory.usagePercent,
          unit: 'bytes',
        },
        disk: {
          value: rootDisk?.usedBytes || 0,
          total: rootDisk?.totalBytes,
          percent: rootDisk?.usagePercent,
          unit: 'bytes',
        },
        load: {
          value: system.load.one,
          unit: String(system.cpu.cores),
          one: system.load.one,
          five: system.load.five,
          fifteen: system.load.fifteen,
        },
        network: {
          receiveBytesPerSecond: rates.receive,
          transmitBytesPerSecond: rates.transmit,
          totalReceivedBytes: system.network.receivedBytes,
          totalTransmittedBytes: system.network.sentBytes,
          tcpConnections: system.network.tcpConnections || 0,
          udpConnections: system.network.udpConnections || 0,
        },
        publicNetwork: {
          ipv4: system.publicNetwork?.ipv4,
          ipv6: system.publicNetwork?.ipv6,
          isp: system.publicNetwork?.isp,
          country: system.publicNetwork?.country,
          region: system.publicNetwork?.region,
          city: system.publicNetwork?.city,
          timezone: system.publicNetwork?.timezone,
          source: system.publicNetwork?.source,
          updatedAt: system.publicNetwork?.updatedAt,
        },
        management: {
          ssh: {
            ports: system.management?.ssh?.ports || [],
            source:
              system.management?.ssh?.source === 'configured' || system.management?.ssh?.source === 'default'
                ? system.management.ssh.source
                : 'unknown',
          },
          dns: {
            servers: system.management?.dns?.servers || [],
            manager: system.management?.dns?.manager || 'unknown',
          },
          timezone: system.management?.timezone,
          swap: {
            totalBytes: system.memory.swapTotalBytes || 0,
            usedBytes: system.memory.swapUsedBytes || 0,
            activeDevices: system.management?.swap?.activeDevices || 0,
            path: system.management?.swap?.path || '/swapfile',
            fileExists: Boolean(system.management?.swap?.fileExists),
            fileActive: Boolean(system.management?.swap?.fileActive),
            fileSizeBytes: system.management?.swap?.fileSizeBytes || 0,
            fileUsedBytes: system.management?.swap?.fileUsedBytes || 0,
            legacyExists: Boolean(system.management?.swap?.legacyExists),
            legacyActive: Boolean(system.management?.swap?.legacyActive),
            legacySizeBytes: system.management?.swap?.legacySizeBytes || 0,
            otherActiveDevices: system.management?.swap?.otherActiveDevices || 0,
            otherSwapTotalBytes: system.management?.swap?.otherSwapTotalBytes || 0,
            otherSwapUsedBytes: system.management?.swap?.otherSwapUsedBytes || 0,
          },
          packageManager: system.management?.packageManager,
          packageSources: system.management?.packageSources || [],
          maintenance: {
            id: system.management?.maintenance?.id,
            state: ['running', 'succeeded', 'failed'].includes(system.management?.maintenance?.state || '')
              ? (system.management?.maintenance?.state as 'running' | 'succeeded' | 'failed')
              : 'idle',
            action:
              system.management?.maintenance?.action === 'update' ||
              system.management?.maintenance?.action === 'cleanup'
                ? system.management.maintenance.action
                : undefined,
            policy:
              system.management?.maintenance?.policy === 'full' ||
              system.management?.maintenance?.policy === 'cache' ||
              system.management?.maintenance?.policy === 'standard'
                ? system.management.maintenance.policy
                : undefined,
            stage: system.management?.maintenance?.stage,
            progress: system.management?.maintenance?.progress || 0,
            message: system.management?.maintenance?.message,
            startedAt: system.management?.maintenance?.startedAt,
            finishedAt: system.management?.maintenance?.finishedAt,
            rebootRequired: Boolean(system.management?.maintenance?.rebootRequired),
          },
          ipPreference:
            system.management?.ipPreference === 'ipv4' || system.management?.ipPreference === 'system_default'
              ? system.management.ipPreference
              : 'unknown',
          kernelOptimization: {
            enabled: Boolean(system.management?.kernelOptimization?.enabled),
            profile: system.management?.kernelOptimization?.profile,
            source: system.management?.kernelOptimization?.source,
          },
          bbr: {
            supported: Boolean(system.management?.bbr?.supported),
            enabled: Boolean(system.management?.bbr?.enabled),
            congestionControl: system.management?.bbr?.congestionControl,
            defaultQDisc: system.management?.bbr?.defaultQDisc,
            available: system.management?.bbr?.available || [],
          },
          capabilities,
        },
        services,
        agent: normalizeAgent(agent),
        sites: {
          total: sites.length,
          healthy: sites.filter((site) => site.health === 'healthy').length,
          drifted: sites.filter((site) => site.consistency !== 'synced').length,
        },
        containers: dockerResult
          ? { total: dockerResult.containers, running: dockerResult.running, stopped: dockerResult.stopped }
          : undefined,
      }
    },
  },
  system: {
    action: (body: SystemActionInput): Promise<SystemActionResult> =>
      request<SystemActionResult>('/system/actions', { method: 'POST', body }),
  },
  sites: {
    list: async (query?: { search?: string; cursor?: string }, signal?: AbortSignal): Promise<ApiList<Site>> => {
      const result = normalizeList(await request<ApiList<RawSite> | RawSite[]>('/sites', { query, signal }))
      return { ...result, items: result.items.map(normalizeSite) }
    },
    create: createSite,
    update: async (id: string, body: SiteInput): Promise<Site> =>
      normalizeSite(await request<RawSite>(`/sites/${encodeURIComponent(id)}`, { method: 'PATCH', body })),
    remove: (id: string, expectedResourceVersion: string) =>
      request<{ id: string; primaryDomain: string; status: string; resourceVersion: string }>(
        `/sites/${encodeURIComponent(id)}`,
        { method: 'DELETE', body: { expectedResourceVersion } },
      ),
  },
  apps: {
    inventory: (signal?: AbortSignal): Promise<AppMarketInventory> =>
      request<AppMarketInventory>('/apps', { signal }),
    install: (
      id: string,
      body: { hostPort?: number; accessMode?: 'direct' | 'domain_only' },
    ): Promise<AppInstallJob> =>
      request<AppInstallJob>(`/apps/${encodeURIComponent(id)}/install`, { method: 'POST', body }),
    job: (id: string, signal?: AbortSignal): Promise<AppInstallJob> =>
      request<AppInstallJob>(`/app-jobs/${encodeURIComponent(id)}`, { signal }),
    jobs: async (signal?: AbortSignal): Promise<ApiList<AppInstallJob>> =>
      normalizeList(await request<ApiList<AppInstallJob> | AppInstallJob[]>('/app-jobs', { signal })),
    action: (
      id: string,
      action: 'start' | 'stop' | 'restart' | 'update' | 'uninstall' | 'direct_access',
      body: { resourceVersion: string; accessMode?: 'direct' | 'domain_only' },
    ): Promise<AppMutationResult> =>
      request<AppMutationResult>(`/apps/${encodeURIComponent(id)}/${action}`, { method: 'POST', body }),
    checkUpdate: (id: string, resourceVersion: string): Promise<AppImageUpdateResult> =>
      request<AppImageUpdateResult>(`/apps/${encodeURIComponent(id)}/check_update`, {
        method: 'POST',
        body: { resourceVersion },
      }),
  },
  docker: {
    inventory: async (signal?: AbortSignal): Promise<DockerInventory> => {
      const [summary, containersResult, imagesResult, networksResult, volumesResult] = await Promise.all([
        request<RawDockerSummary>('/docker/summary', { signal }),
        request<ApiList<RawContainer> | RawContainer[]>('/docker/containers', { signal }),
        request<ApiList<DockerInventory['images'][number]> | DockerInventory['images']>('/docker/images', { signal }),
        request<ApiList<DockerInventory['networks'][number]> | DockerInventory['networks']>('/docker/networks', { signal }),
        request<ApiList<DockerInventory['volumes'][number]> | DockerInventory['volumes']>('/docker/volumes', { signal }),
      ])
      return {
        available: summary.available,
        version: summary.serverVersion,
        observedAt: summary.collectedAt,
        containers: normalizeList(containersResult).items.map(normalizeContainer),
        images: normalizeList(imagesResult).items,
        networks: normalizeList(networksResult).items,
        volumes: normalizeList(volumesResult).items,
      }
    },
    action: (id: string, action: 'start' | 'stop' | 'restart', resourceVersion: string) =>
      request<DockerActionResult>(`/docker/containers/${encodeURIComponent(id)}/${action}`, {
        method: 'POST',
        body: { resourceVersion },
      }),
    logs: (id: string, tail = 200, signal?: AbortSignal) =>
      request<{ lines: string[]; truncated?: boolean }>(`/docker/containers/${encodeURIComponent(id)}/logs`, {
        query: { tail },
        signal,
      }),
  },
  jobs: {
    list: async (query?: { limit?: number }, signal?: AbortSignal): Promise<ApiList<Job>> => {
      const result = normalizeList(await request<ApiList<RawJob> | RawJob[]>('/jobs', { query, signal }))
      return { ...result, items: result.items.map(normalizeJob) }
    },
  },
  audit: {
    list: async (
      query?: { source?: string; outcome?: string; cursor?: string },
      signal?: AbortSignal,
    ): Promise<ApiList<AuditEvent>> => {
      const result = normalizeList(await request<ApiList<RawAuditEvent> | RawAuditEvent[]>('/audit', { query, signal }))
      return {
        ...result,
        items: result.items.map((item) => ({
          id: item.id,
          occurredAt: item.occurredAt,
          actor: item.actorId || item.actorType || 'system',
          source: (['web', 'cli', 'reconcile', 'system', 'external'].includes(item.actorType || '')
            ? item.actorType
            : 'system') as AuditEvent['source'],
          action: item.action,
          resourceType: item.targetKind,
          resourceName: item.targetId,
          outcome: (['success', 'failure', 'denied', 'observed'].includes(item.result || '')
            ? item.result
            : item.result === 'failed'
              ? 'failure'
              : 'observed') as AuditEvent['outcome'],
          requestId: item.requestId,
          remoteAddress: item.sourceIp,
        })),
      }
    },
  },
  settings: {
    get: (signal?: AbortSignal) => request<PanelSettings>('/settings', { signal }),
    changePassword: (currentPassword: string, newPassword: string) =>
      request<void>('/settings/password', {
        method: 'PUT',
        body: { currentPassword, newPassword },
      }),
  },
}

export function resetApiSecurityState(): void {
  csrfToken = ''
  previousNetworkSample = undefined
}
