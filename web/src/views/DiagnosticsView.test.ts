import { createSSRApp, ssrContextKey } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DiagnosticsView from './DiagnosticsView.vue'
import type { DiagnosticJob } from '@/types/api'

const mocks = vi.hoisted(() => ({
  job: vi.fn(),
  toastDanger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    diagnostics: {
      catalog: vi.fn(),
      jobs: vi.fn(),
      job: mocks.job,
      start: vi.fn(),
    },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: vi.fn(),
    danger: mocks.toastDanger,
  }),
}))

interface DiagnosticBindings {
  refreshJob: (id: string) => Promise<void>
}

function setupView(): DiagnosticBindings {
  const component = DiagnosticsView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => DiagnosticBindings
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

function runningJob(): DiagnosticJob {
  return {
    id: 'a'.repeat(32),
    checkId: 'yabs',
    checkName: 'YABS 性能测试',
    category: 'hardware',
    sourceUrl: 'https://yabs.sh',
    estimatedMinutes: 30,
    impact: 'intensive',
    status: 'running',
    stage: 'running',
    progress: 10,
    logs: [],
    createdAt: '2026-07-27T12:00:00Z',
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('DiagnosticsView polling', () => {
  it('does not cancel or duplicate a slow in-flight refresh', async () => {
    let resolveJob: ((job: DiagnosticJob) => void) | undefined
    mocks.job.mockReturnValueOnce(
      new Promise<DiagnosticJob>((resolve) => {
        resolveJob = resolve
      }),
    )
    const view = setupView()

    const first = view.refreshJob('a'.repeat(32))
    const second = view.refreshJob('a'.repeat(32))

    expect(mocks.job).toHaveBeenCalledOnce()
    resolveJob?.(runningJob())
    await Promise.all([first, second])
  })

  it('tolerates two transient failures and reports the third', async () => {
    mocks.job.mockRejectedValue(new Error('temporary network failure'))
    const view = setupView()

    await view.refreshJob('a'.repeat(32))
    await view.refreshJob('a'.repeat(32))
    expect(mocks.toastDanger).not.toHaveBeenCalled()

    await view.refreshJob('a'.repeat(32))
    expect(mocks.toastDanger).toHaveBeenCalledWith(
      '体检进度刷新中断',
      '后台任务可能仍在运行，请稍后点击刷新重新连接。',
    )
  })
})
