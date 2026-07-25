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
}

export interface SystemOverview {
  hostname: string
  os: string
  kernel?: string
  architecture?: string
  uptimeSeconds: number
  observedAt: string
  cpu: MetricValue
  memory: MetricValue
  disk: MetricValue
  load: MetricValue
  network: NetworkSummary
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
  type: 'static' | 'proxy' | 'php' | 'redirect' | 'unknown'
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
  type: 'static' | 'proxy'
  upstream?: string
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

export interface JobAccepted {
  jobId: string
}
