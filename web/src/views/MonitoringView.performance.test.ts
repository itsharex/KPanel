import { createSSRApp, ssrContextKey, type Ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MonitoringHistory, MonitoringRange } from '@/types/api'
import MonitoringView from './MonitoringView.vue'

const mocks = vi.hoisted(() => ({
  history: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({
    back: vi.fn(),
    push: vi.fn(),
    replace: vi.fn(),
  }),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: { monitoring: { history: mocks.history } },
}))

interface MonitoringBindings {
  selectedRange: Ref<MonitoringRange>
  load: () => Promise<void>
}

function setupView(): MonitoringBindings {
  const component = MonitoringView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => MonitoringBindings
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

const emptyHistory = {
  range: '6h',
  startedAt: '2026-08-05T08:00:00Z',
  endedAt: '2026-08-05T14:00:00Z',
  bucketSeconds: 60,
  host: [],
  containers: [],
  operatorLatency: [],
  storage: {},
  scannedBytes: 0,
  skippedLines: 0,
  truncatedSeries: 0,
} as unknown as MonitoringHistory

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
  })
  mocks.history.mockResolvedValue(emptyHistory)
})

describe('MonitoringView performance defaults', () => {
  it('loads the focused six-hour window first', async () => {
    const view = setupView()

    expect(view.selectedRange.value).toBe('6h')
    await view.load()
    expect(mocks.history).toHaveBeenCalledWith('6h', undefined, expect.any(AbortSignal))
  })
})
