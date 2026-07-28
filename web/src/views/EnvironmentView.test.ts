import { createSSRApp, ssrContextKey, type Ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EnvironmentView from './EnvironmentView.vue'
import type { WebEnvironmentJob, WebEnvironmentSummary } from '@/types/api'

const mocks = vi.hoisted(() => ({
  start: vi.fn(),
  toastSuccess: vi.fn(),
  toastDanger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    webEnvironment: {
      summary: vi.fn(),
      catalog: vi.fn(),
      backups: vi.fn(),
      jobs: vi.fn(),
      job: vi.fn(),
      terminal: vi.fn(),
      terminalInput: vi.fn(),
      start: mocks.start,
      backupDownloadURL: vi.fn(),
    },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    danger: mocks.toastDanger,
  }),
}))

interface EnvironmentBindings {
  summary: Ref<WebEnvironmentSummary | undefined>
  jobs: Ref<WebEnvironmentJob[]>
  terminalOpen: Ref<boolean>
  terminalJob: Ref<WebEnvironmentJob | undefined>
  start: (input: Record<string, unknown>) => Promise<void>
}

function setupView(): EnvironmentBindings {
  const component = EnvironmentView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => EnvironmentBindings
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

function summary(): WebEnvironmentSummary {
  return {
    protocolVersion: '1',
    state: 'installed',
    profile: 'full',
    health: 'healthy',
    webRoot: '/home/web',
    diskBytes: 1,
    siteCount: 1,
    databaseCount: 1,
    certificateCount: 1,
    composeValid: true,
    nginxValid: true,
    resourceVersion: `sha256:${'a'.repeat(64)}`,
    scriptVersion: 'test',
    latestBackup: '',
    portConflicts: [],
    components: [],
    protection: { fail2ban: false, waf: false, cloudflare: false, ddos: false },
    optimization: { mode: 'custom', gzip: false, brotli: false, zstd: false },
    observedAt: '2026-07-28T00:00:00Z',
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('EnvironmentView background jobs', () => {
  it('always submits the current environment resource version and reopens the terminal', async () => {
    const job: WebEnvironmentJob = {
      id: 'b'.repeat(32),
      action: 'backup.create',
      target: 'web',
      status: 'running',
      stage: 'running',
      progress: 1,
      message: 'running',
      createdAt: '2026-07-28T00:00:00Z',
    }
    mocks.start.mockResolvedValueOnce(job)
    const view = setupView()
    view.summary.value = summary()

    await view.start({ action: 'backup.create' })

    expect(mocks.start).toHaveBeenCalledWith({
      action: 'backup.create',
      expectedResourceVersion: view.summary.value?.resourceVersion,
    })
    expect(view.jobs.value).toEqual([job])
    expect(view.terminalJob.value).toEqual(job)
    expect(view.terminalOpen.value).toBe(true)
    expect(mocks.toastSuccess).toHaveBeenCalledWith(
      '任务已转入后台',
      '关闭终端或刷新页面都不会中断执行。',
    )
  })
})
