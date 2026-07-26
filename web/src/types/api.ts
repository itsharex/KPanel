export type HealthLevel = 'healthy' | 'warning' | 'critical' | 'unknown'
export type ConsistencyState =
  | 'synced'
  | 'drifted'
  | 'ambiguous'
  | 'conflicted'
  | 'unsupported'
  | 'read_only'
  | 'pending'
  | 'unknown'
export type ResourceAccess = 'managed' | 'read-only' | 'unmanaged'

export interface ApiList<T> {
  items: T[]
  total: number
  nextCursor?: string
}

export interface AgentStatus {
  connected: boolean
  readOnly: boolean
  compatible: boolean
  version?: string
  protocolVersion?: string
  lastSeenAt?: string
  reason?: string
}

export interface User {
  id: string
  username: string
  displayName?: string
  role?: string
  totpEnabled?: boolean
}

export interface AuthStatus {
  setupRequired: boolean
  authenticated: boolean
  user?: User
  csrfToken?: string
  expiresAt?: string
  agent?: AgentStatus
}

export interface SetupRequest {
  token: string
  username: string
  password: string
}

export interface LoginRequest {
  username: string
  password: string
  totpCode?: string
}

export interface AuthSession {
  user: User
  csrfToken?: string
  expiresAt?: string
}

export interface MetricValue {
  value: number
  total?: number
  unit?: string
  percent?: number
  change?: number
}

export interface ServiceStatus {
  id: string
  name: string
  state: 'running' | 'stopped' | 'degraded' | 'unknown'
  version?: string
  detail?: string
}

export interface NetworkSummary {
  receiveBytesPerSecond: number
  transmitBytesPerSecond: number
  totalReceivedBytes?: number
  totalTransmittedBytes?: number
  tcpConnections: number
  udpConnections: number
}

export interface PublicNetworkSummary {
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

export interface CapabilityState {
  enabled: boolean
  reason?: string
  methods?: string[]
}

export interface SystemManagement {
  ssh: {
    ports: number[]
    source: 'configured' | 'default' | 'unknown'
  }
  dns: {
    servers: string[]
    manager: string
  }
  timezone?: string
  swap: {
    totalBytes: number
    usedBytes: number
    activeDevices: number
    path: string
    fileExists: boolean
    fileActive: boolean
    fileSizeBytes: number
    fileUsedBytes: number
    legacyExists: boolean
    legacyActive: boolean
    legacySizeBytes: number
    otherActiveDevices: number
    otherSwapTotalBytes: number
    otherSwapUsedBytes: number
  }
  packageManager?: string
  packageSources: string[]
  maintenance: {
    id?: string
    state: 'idle' | 'running' | 'succeeded' | 'failed'
    action?: 'update' | 'cleanup'
    policy?: 'full' | 'cache' | 'standard'
    stage?: string
    progress: number
    message?: string
    startedAt?: string
    finishedAt?: string
    rebootRequired: boolean
  }
  ipPreference: 'ipv4' | 'system_default' | 'unknown'
  kernelOptimization: {
    enabled: boolean
    profile?: string
    source?: string
  }
  bbr: {
    supported: boolean
    enabled: boolean
    congestionControl?: string
    defaultQDisc?: string
    available: string[]
  }
  capabilities: Record<string, CapabilityState>
}

export interface SystemOverview {
  hostname: string
  os: string
  kernel?: string
  architecture?: string
  uptimeSeconds: number
  observedAt: string
  cpu: MetricValue & {
    model?: string
    cores: number
    frequencyMHz?: number
  }
  memory: MetricValue
  disk: MetricValue
  load: MetricValue & {
    one: number
    five: number
    fifteen: number
  }
  network: NetworkSummary
  publicNetwork: PublicNetworkSummary
  management: SystemManagement
  services: ServiceStatus[]
  agent: AgentStatus
  sites?: {
    total: number
    healthy: number
    drifted: number
  }
  containers?: {
    total: number
    running: number
    stopped: number
  }
}

export interface SystemActionInput {
  action:
    | 'hostname'
    | 'ssh-port'
    | 'dns'
    | 'timezone'
    | 'swap'
    | 'mirror'
    | 'ip-preference'
    | 'kernel-tuning'
    | 'bbr'
    | 'update'
    | 'cleanup'
    | 'reboot'
  hostname?: string
  port?: number
  servers?: string[]
  timezone?: string
  swapSizeMiB?: number
  mirrorPreset?: 'cn-default' | 'cn-edu' | 'abroad' | 'smart'
  preference?: 'ipv4' | 'system_default'
  profile?: 'high' | 'balanced' | 'web' | 'stream' | 'game' | 'off'
  maintenancePolicy?: 'full' | 'cache' | 'standard'
  confirmation?: 'REBOOT'
  enabled?: boolean
}

export interface SystemActionResult {
  action: string
  status: string
  changed: boolean
  message: string
  backupPath?: string
  appliedAt: string
}

export interface AppMarketCategory {
  key: string
  zh: string
  en: string
}

export interface AppActionCapability {
  enabled: boolean
  reason?: string
}

export interface AppMarketRuntime {
  installed: boolean
  state: string
  status?: string
  containerId?: string
  containerName?: string
  image?: string
  ports: Array<{
    privatePort: number
    publicPort?: number
    ip?: string
    type: string
  }>
  accessMode: 'direct' | 'domain_only' | 'unknown' | 'not_applicable'
  updateStatus: 'available' | 'current' | 'check_required' | 'unknown' | 'not_installed'
  resourceVersion?: string
  detectedBy: string[]
  warning?: string
}

export interface AppMarketItem {
  id: string
  num?: number
  source: 'builtin' | 'thirdparty'
  token: string
  name_zh: string
  name_en: string
  desc_zh: string
  desc_en: string
  cat: string
  url?: string
  icon: string
  iconSha256: string
  slug: string
  defaultPort?: number
  installer: 'declarative' | 'kejilion' | 'guided'
  runtime: AppMarketRuntime
  capabilities: Record<string, AppActionCapability>
}

export interface AppMarketInventory {
  schemaVersion: number
  source: string
  scriptSha256: string
  catalogMode: 'live' | 'cached' | 'embedded'
  catalogWarning?: string
  catalogRefreshedAt?: string
  categories: AppMarketCategory[]
  items: AppMarketItem[]
  installed: number
  running: number
  updateAvailable: number
  collectedAt: string
}

export interface AppMutationResult {
  containerId?: string
  action: string
  status: string
  resourceVersion?: string
}

export interface AppInstallJob {
  id: string
  appId: string
  appName: string
  action: 'install'
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  stage: string
  progress: number
  message?: string
  logs: string[]
  createdAt: string
  startedAt?: string
  finishedAt?: string
}

export interface AppImageUpdateResult {
  containerId: string
  image: string
  status: 'available' | 'current'
  updateAvailable: boolean
  localDigest?: string
  remoteDigest?: string
  resourceVersion: string
  checkedAt: string
}

export interface CertificateSummary {
  status: 'valid' | 'expiring' | 'expired' | 'missing' | 'unknown'
  issuer?: string
  expiresAt?: string
  daysRemaining?: number
}

export interface Site {
  id: string
  primaryDomain: string
  domains: string[]
  type: 'static' | 'proxy' | 'proxy_domain' | 'load_balance' | 'php' | 'wordpress' | 'redirect' | 'unknown'
  enabled: boolean
  health: HealthLevel
  consistency: ConsistencyState
  access: ResourceAccess
  source: 'kejilion' | 'panel' | 'external' | 'unknown'
  rootPath?: string
  upstream?: string
  certificate?: CertificateSummary
  resourceVersion: string
  observedAt?: string
  reason?: string
  allowedActions?: string[]
  warnings?: string[]
  artifacts?: Array<{
    kind: string
    path: string
    hash?: string
  }>
}

export interface SiteInput {
  primaryDomain: string
  aliases?: string[]
  type: 'wordpress' | 'static' | 'php' | 'proxy' | 'proxy_domain' | 'load_balance' | 'redirect'
  upstream?: string
  upstreams?: string[]
  redirectTarget?: string
  redirectCode?: 301 | 302 | 307 | 308
  phpVersion?: 'latest' | '7.4'
  enabled?: boolean
  expectedResourceVersion?: string
}

export interface DockerPort {
  privatePort: number
  publicPort?: number
  protocol: 'tcp' | 'udp' | 'sctp'
  ip?: string
}

export interface DockerContainer {
  id: string
  name: string
  image: string
  state: 'running' | 'paused' | 'restarting' | 'exited' | 'dead' | 'created' | 'unknown'
  health?: HealthLevel
  access: ResourceAccess
  consistency: ConsistencyState
  project?: string
  ports: DockerPort[]
  createdAt?: string
  startedAt?: string
  cpuPercent?: number
  memoryBytes?: number
  memoryLimitBytes?: number
  reason?: string
  allowedActions?: string[]
  resourceVersion?: string
  statusText?: string
}

export interface DockerImage {
  id: string
  tags: string[]
  sizeBytes: number
  createdAt?: string
  inUse: boolean
}

export interface DockerNetwork {
  id: string
  name: string
  driver: string
  scope?: string
  containers?: number
}

export interface DockerVolume {
  name: string
  driver: string
  mountpoint?: string
  inUse?: boolean
}

export interface DockerInventory {
  available: boolean
  version?: string
  observedAt: string
  containers: DockerContainer[]
  images: DockerImage[]
  networks: DockerNetwork[]
  volumes: DockerVolume[]
}

export type JobStatus =
  | 'queued'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'failed_rolled_back'
  | 'failed_needs_attention'
  | 'interrupted'
  | 'cancelled'

export interface JobStage {
  name: string
  status: JobStatus
  startedAt?: string
  finishedAt?: string
  message?: string
}

export interface Job {
  id: string
  action: string
  resourceType?: string
  resourceName?: string
  status: JobStatus
  progress?: number
  actor?: string
  source?: 'web' | 'cli' | 'reconcile' | 'system'
  createdAt: string
  startedAt?: string
  finishedAt?: string
  errorCode?: string
  errorMessage?: string
  stages?: JobStage[]
}

export interface AuditEvent {
  id: string
  occurredAt: string
  actor: string
  source: 'web' | 'cli' | 'reconcile' | 'system' | 'external'
  action: string
  resourceType?: string
  resourceName?: string
  outcome: 'success' | 'failure' | 'denied' | 'observed'
  requestId?: string
  summary?: string
  remoteAddress?: string
}

export interface PanelSettings {
  panelVersion?: string
  agent: AgentStatus
  serverTime?: string
  sessionExpiresAt?: string
  publicUrl?: string
  reconcileIntervalSeconds?: number
  telemetryEnabled?: boolean
}

export interface DockerActionResult {
  jobId?: string
  containerId?: string
  action?: string
  status?: string
  resourceVersion?: string
}
