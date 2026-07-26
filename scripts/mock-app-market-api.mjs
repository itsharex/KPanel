import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const catalog = JSON.parse(await readFile(join(root, 'internal', 'appmarket', 'catalog.json'), 'utf8'))
const legacy = JSON.parse(await readFile(join(root, 'internal', 'appmarket', 'legacy-apps.json'), 'utf8'))
const legacyByNumber = new Map(legacy.apps.map((item) => [item.num, item]))
const installed = new Map([
  ['speedtest', { state: 'running', direct: true }],
  ['it-tools', { state: 'running', direct: false }],
  ['openlist', { state: 'running', direct: true }],
  ['n8n', { state: 'exited', direct: false }],
])
const adapted = new Set(['speedtest', 'it-tools', 'dosgame'])

const items = catalog.apps.map((app) => {
  const mapping = legacyByNumber.get(app.num) || {}
  const runtime = installed.get(app.token)
  const isAdapted = adapted.has(app.token)
  const isRunning = runtime?.state === 'running'
  const port = mapping.defaultPort || 0
  return {
    ...app,
    defaultPort: port,
    runtime: runtime
      ? {
          installed: true,
          state: runtime.state,
          status: isRunning ? 'Up 3 hours' : 'Exited (0) 20 minutes ago',
          containerId: 'a'.repeat(64),
          containerName: mapping.container || app.slug,
          image: mapping.image || `${app.slug}:latest`,
          ports: port
            ? [
                {
                  privatePort: app.token === 'it-tools' ? 80 : 8080,
                  publicPort: port,
                  ip: runtime.direct ? '0.0.0.0' : '127.0.0.1',
                  type: 'tcp',
                },
              ]
            : [],
          accessMode: runtime.direct ? 'direct' : 'domain_only',
          updateStatus: 'check_required',
          resourceVersion: `sha256:${'b'.repeat(64)}`,
          detectedBy: ['docker', 'appno'],
        }
      : {
          installed: false,
          state: 'not_installed',
          ports: [],
          accessMode: 'not_applicable',
          updateStatus: 'not_installed',
          detectedBy: [],
        },
    capabilities: {
      install: {
        enabled: isAdapted && !runtime,
        reason: isAdapted ? (runtime ? '应用已安装' : '') : '该应用尚未完成 KPanel 声明式安全适配',
      },
      start: { enabled: Boolean(runtime && !isRunning), reason: '当前状态不允许启动' },
      stop: { enabled: Boolean(runtime && isRunning), reason: '当前状态不允许停止' },
      restart: { enabled: Boolean(runtime && isRunning), reason: '当前状态不允许重启' },
      check_update: { enabled: Boolean(runtime) },
      update: { enabled: Boolean(runtime && isAdapted), reason: '只允许更新配置完全匹配的声明式应用' },
      uninstall: { enabled: Boolean(runtime && isAdapted), reason: '现有应用由 kejilion.sh 管理' },
      add_domain: { enabled: Boolean(runtime && port) },
      direct_access: { enabled: Boolean(runtime && isAdapted), reason: '仅声明式适配应用支持安全切换' },
    },
  }
})

const inventory = {
  schemaVersion: 1,
  source: catalog.source,
  scriptSha256: legacy.scriptSha256,
  categories: catalog.categories,
  items,
  installed: installed.size,
  running: [...installed.values()].filter((item) => item.state === 'running').length,
  updateAvailable: 0,
  collectedAt: new Date().toISOString(),
}

const systemSummary = {
  hostname: 'kpanel-demo',
  os: 'Debian GNU/Linux 13',
  kernel: '6.12.0-amd64',
  architecture: 'amd64',
  uptimeSeconds: 864000,
  load: { one: 0.38, five: 0.42, fifteen: 0.36 },
  cpu: { model: 'AMD EPYC', cores: 4, frequencyMHz: 2445, usagePercent: 18.6 },
  memory: {
    totalBytes: 8 * 1024 ** 3,
    availableBytes: 5.1 * 1024 ** 3,
    usedBytes: 2.9 * 1024 ** 3,
    usagePercent: 36.25,
    swapTotalBytes: 2 * 1024 ** 3,
    swapUsedBytes: 128 * 1024 ** 2,
  },
  disks: [
    {
      device: '/dev/vda1',
      mountPoint: '/',
      fileSystem: 'ext4',
      totalBytes: 80 * 1024 ** 3,
      usedBytes: 27 * 1024 ** 3,
      usagePercent: 33.75,
    },
  ],
  network: {
    receivedBytes: 24 * 1024 ** 3,
    sentBytes: 11 * 1024 ** 3,
    tcpConnections: 96,
    udpConnections: 18,
  },
  publicNetwork: {
    ipv4: '203.0.113.10',
    ipv6: '2001:db8::10',
    isp: 'KPanel Visual Test',
    country: 'China',
    region: 'Shanghai',
    city: 'Shanghai',
    timezone: 'Asia/Shanghai',
  },
  management: {
    ssh: { ports: [22, 2222], source: 'configured' },
    dns: { servers: ['1.1.1.1', '2606:4700:4700::1111'], manager: 'systemd-resolved' },
    timezone: 'Asia/Shanghai',
    swap: {
      activeDevices: 1,
      path: '/swapfile',
      fileExists: true,
      fileActive: true,
      fileSizeBytes: 2 * 1024 ** 3,
      fileUsedBytes: 128 * 1024 ** 2,
      legacyExists: false,
      legacyActive: false,
      legacySizeBytes: 0,
      otherActiveDevices: 0,
      otherSwapTotalBytes: 0,
      otherSwapUsedBytes: 0,
    },
    packageManager: 'apt',
    packageSources: ['https://deb.debian.org/debian'],
    maintenance: { state: 'idle', progress: 0, rebootRequired: true },
    ipPreference: 'ipv4',
    kernelOptimization: { enabled: true, profile: '均衡优化模式', source: 'kejilion' },
    bbr: {
      supported: true,
      enabled: true,
      congestionControl: 'bbr',
      defaultQDisc: 'fq',
      available: ['reno', 'cubic', 'bbr'],
    },
  },
  collectedAt: new Date().toISOString(),
}

const systemCapabilities = [
  'hostname',
  'ssh-port',
  'dns',
  'timezone',
  'swap',
  'mirror',
  'ip-preference',
  'kernel-tuning',
  'bbr',
  'update',
  'cleanup',
  'reboot',
].map((action) => ({ id: `system.${action}.write`, enabled: true, methods: ['POST'] }))

function send(response, status, body) {
  const data = JSON.stringify(body)
  response.writeHead(status, {
    'Content-Type': 'application/json; charset=utf-8',
    'Content-Length': Buffer.byteLength(data),
    'Cache-Control': 'no-store',
  })
  response.end(data)
}

createServer((request, response) => {
  const url = new URL(request.url, 'http://127.0.0.1:8080')
  if (request.method === 'GET' && url.pathname === '/api/v1/auth/bootstrap') {
    send(response, 200, { required: false })
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/auth/session') {
    send(response, 200, {
      user: { id: 'visual-test', username: 'admin', displayName: 'Admin', role: 'owner' },
      csrfToken: 'visual-test-csrf',
      expiresAt: new Date(Date.now() + 3_600_000).toISOString(),
    })
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/agent/health') {
    send(response, 200, {
      status: 'ok',
      version: '0.12.0',
      protocolVersion: 'v1',
      readOnly: false,
      checkedAt: new Date().toISOString(),
    })
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/apps') {
    send(response, 200, inventory)
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/system/summary') {
    send(response, 200, systemSummary)
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/sites') {
    send(response, 200, {
      items: [
        {
          id: 'c'.repeat(32),
          primaryDomain: 'tools.example.com',
          domains: ['tools.example.com'],
          kind: 'reverse_proxy',
          enabled: true,
          health: 'healthy',
          consistency: 'in_sync',
          origin: 'web',
          target: 'http://127.0.0.1:8064',
          resourceVersion: `sha256:${'d'.repeat(64)}`,
          allowedActions: ['update'],
        },
      ],
    })
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/capabilities') {
    send(response, 200, {
      items: [
        { id: 'apps.install', enabled: true, methods: ['POST'] },
        ...systemCapabilities,
        { id: 'system.reinstall', enabled: false, reason: '需要带外控制台' },
      ],
    })
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/docker/summary') {
    send(response, 200, {
      available: true,
      serverVersion: '28.3.2',
      containers: 5,
      running: 4,
      paused: 0,
      stopped: 1,
      images: 12,
      collectedAt: new Date().toISOString(),
    })
    return
  }
  if (request.method === 'GET' && url.pathname === '/api/v1/docker/containers') {
    send(response, 200, { items: [] })
    return
  }
  if (request.method === 'POST' && url.pathname === '/api/v1/system/actions') {
    send(response, 200, {
      action: 'reboot',
      status: 'accepted',
      changed: true,
      message: '视觉测试：重启任务已模拟排队',
      appliedAt: new Date().toISOString(),
    })
    return
  }
  if (request.method === 'POST' || request.method === 'DELETE') {
    send(response, 200, {
      action: url.pathname.split('/').at(-1),
      status: 'completed',
      resourceVersion: `sha256:${'e'.repeat(64)}`,
    })
    return
  }
  send(response, 404, { title: 'Not found', status: 404, code: 'not_found' })
}).listen(8080, '127.0.0.1', () => {
  process.stdout.write('KPanel visual mock API: http://127.0.0.1:8080\n')
})
