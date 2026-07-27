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
  countryCode?: string
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
  osId?: string
  osLike: string[]
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
  apps?: {
    total: number
    installed: number
    running: number
    updateAvailable: number
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
  action: 'install' | 'update' | 'uninstall' | 'direct_access'
  interactive?: boolean
  inputOpen?: boolean
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  stage: string
  progress: number
  message?: string
  logs: string[]
  createdAt: string
  startedAt?: string
  finishedAt?: string
}

export interface AppTerminalChunk {
  dataBase64: string
  nextOffset: number
  inputOpen: boolean
  finished: boolean
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

export interface DiagnosticCategory {
  id: string
  name: string
}

export interface DiagnosticCheck {
  id: string
  category: string
  name: string
  description: string
  sourceUrl: string
  estimatedMinutes: number
  impact: 'light' | 'network' | 'intensive'
}

export interface DiagnosticCatalog {
  categories: DiagnosticCategory[]
  items: DiagnosticCheck[]
}

export interface DiagnosticJob {
  id: string
  checkId: string
  checkName: string
  category: string
  sourceUrl: string
  estimatedMinutes: number
  impact: 'light' | 'network' | 'intensive'
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  stage: string
  progress: number
  message?: string
  logs: string[]
  createdAt: string
  startedAt?: string
  finishedAt?: string
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
  type: 'wordpress' | 'recipe' | 'static' | 'php' | 'proxy' | 'proxy_domain' | 'load_balance' | 'redirect'
  recipe?: 'discuz' | 'kodbox' | 'maccms' | 'dujiaoka' | 'flarum' | 'typecho' | 'linkstack' | 'ai-prompt'
  upstream?: string
  upstreams?: string[]
  redirectTarget?: string
  redirectCode?: 301 | 302 | 307 | 308
  phpVersion?: 'latest' | '7.4'
  enabled?: boolean
  expectedResourceVersion?: string
}

export interface SiteInstallationProgress {
  id?: string
  interactive?: boolean
  inputOpen?: boolean
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  stage: string
  progress: number
  message: string
  events?: Array<{
    stage: string
    progress: number
    message: string
    at: string
  }>
}

export interface SiteDeleteResult {
  id: string
  primaryDomain: string
  status: 'deleted'
  mode: 'configuration' | 'full'
  resourceVersion: string
  removed: string[]
  databaseDropped: boolean
  warnings?: string[]
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
  networks: string[]
  mounts: Array<{
    type: string
    name?: string
    source?: string
    destination: string
  }>
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
  digests?: string[]
  sizeBytes: number
  createdAt?: string
  inUse: boolean
  resourceVersion?: string
}

export interface DockerNetwork {
  id: string
  name: string
  driver: string
  scope?: string
  containers?: number
  resourceVersion?: string
}

export interface DockerVolume {
  name: string
  driver: string
  mountpoint?: string
  inUse?: boolean
  resourceVersion?: string
}

export interface DockerInventory {
  available: boolean
  version?: string
  observedAt: string
  containers: DockerContainer[]
  images: DockerImage[]
  networks: DockerNetwork[]
  volumes: DockerVolume[]
  loading?: Partial<Record<'images' | 'networks' | 'volumes', boolean>>
  errors?: Partial<Record<'images' | 'networks' | 'volumes', string>>
}

export interface DockerContainerStats {
  containerId: string
  cpuPercent: number
  memoryBytes: number
  memoryLimitBytes: number
  memoryPercent: number
  networkRxBytes: number
  networkTxBytes: number
  blockReadBytes: number
  blockWriteBytes: number
  pids: number
  collectedAt: string
}

export interface DockerExecResult {
  containerId: string
  exitCode: number
  output: string
  truncated: boolean
  finishedAt: string
}

export interface DockerBackup {
  id: string
  sizeBytes: number
  createdAt: string
  format: 'kpanel-home-docker-v1'
}

export interface DockerEnvironment {
  available: boolean
  engineVersion?: string
  storageDriver?: string
  dataRoot?: string
  containers: number
  images: number
  mirrorPreset: 'cn' | 'official' | 'custom'
  registryMirrors: string[]
  ipv6Enabled: boolean
  ipv6Cidr?: string
  daemonConfig: 'missing' | 'valid' | 'invalid'
  daemonWarning?: string
  observedAt: string
}

export interface DockerContainerCreatePort {
  privatePort: number
  publicPort: number
  protocol?: 'tcp' | 'udp'
  hostIp?: string
}

export interface DockerContainerCreateMount {
  type?: 'volume' | 'bind'
  source: string
  target: string
  readOnly?: boolean
}

export interface DockerContainerCreateEnvironment {
  name: string
  value: string
}

export type DockerMaintenanceAction =
  | 'container_create'
  | 'container_access'
  | 'image_pull'
  | 'image_remove'
  | 'network_create'
  | 'network_remove'
  | 'network_connect'
  | 'network_disconnect'
  | 'volume_create'
  | 'volume_remove'
  | 'backup_create'
  | 'backup_restore'
  | 'backup_migrate'
  | 'daemon_mirror'
  | 'daemon_ipv6'
  | 'container_prune'
  | 'image_prune'
  | 'network_prune'
  | 'volume_prune'
  | 'prune'

export interface DockerMaintenanceInput {
  action: DockerMaintenanceAction
  image?: string
  target?: string
  name?: string
  driver?: string
  containerId?: string
  containerResourceVersion?: string
  expectedResourceVersion?: string
  preset?: 'cn' | 'official'
  enabled?: boolean
  ipv6Cidr?: string
  ports?: DockerContainerCreatePort[]
  mounts?: DockerContainerCreateMount[]
  environment?: DockerContainerCreateEnvironment[]
  command?: string[]
  network?: string
  restartPolicy?: 'no' | 'always' | 'unless-stopped' | 'on-failure'
  allowedIp?: string
  backupId?: string
  migrationHost?: string
  migrationUser?: string
  migrationPort?: number
}

export interface DockerMaintenanceJob {
  id: string
  action: DockerMaintenanceAction
  target?: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  stage: string
  progress: number
  message?: string
  resultPath?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
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
