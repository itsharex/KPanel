import { createSSRApp, ssrContextKey, type Ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OverviewView from './OverviewView.vue'
import type { SystemOverview } from '@/types/api'

const mocks = vi.hoisted(() => ({
  overviewGet: vi.fn(),
  setAgent: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    overview: { get: mocks.overviewGet },
    system: { action: vi.fn(), maintenance: vi.fn() },
  },
}))

vi.mock('@/stores/panel', () => ({
  usePanelState: () => ({ setAgent: mocks.setAgent }),
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({ success: vi.fn(), danger: vi.fn() }),
}))

interface OverviewBindings {
  data: Ref<SystemOverview | undefined>
  actionForm: { timezone: string; timezonePreset: string }
  openTool: (tool: { id: string }) => void
  load: (silent?: boolean) => Promise<void>
}

function setupView(): OverviewBindings {
  const component = OverviewView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => OverviewBindings
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

function overview(id: string): SystemOverview {
  return {
    agent: { version: id },
    publicNetwork: {},
    management: {
      ssh: { ports: [], defense: { enabled: false } },
      dns: { servers: [] },
      swap: {},
      packageSources: [],
      kernelOptimization: { enabled: false },
      bbr: { enabled: false },
      bbrv3: { installed: false },
      maintenance: { state: 'idle', progress: 0 },
    },
  } as unknown as SystemOverview
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('OverviewView refresh stability', () => {
  it('streams the first load but applies later refreshes atomically', async () => {
    const initialPartial = overview('initial-partial')
    const initialComplete = overview('initial-complete')
    const refreshedComplete = overview('refreshed-complete')
    let finishRefresh: ((value: SystemOverview) => void) | undefined
    mocks.overviewGet
      .mockImplementationOnce(async (_signal: AbortSignal, onUpdate?: (value: SystemOverview) => void) => {
        expect(onUpdate).toEqual(expect.any(Function))
        onUpdate?.(initialPartial)
        return initialComplete
      })
      .mockImplementationOnce((_signal: AbortSignal, onUpdate?: (value: SystemOverview) => void) => {
        expect(onUpdate).toBeUndefined()
        return new Promise<SystemOverview>((resolve) => {
          finishRefresh = resolve
        })
      })

    const view = setupView()
    await view.load()
    expect(view.data.value).toStrictEqual(initialComplete)

    const refresh = view.load(true)
    expect(view.data.value).toStrictEqual(initialComplete)
    finishRefresh?.(refreshedComplete)
    await refresh
    expect(view.data.value).toStrictEqual(refreshedComplete)
    expect(mocks.overviewGet).toHaveBeenCalledTimes(2)
  })

  it('does not present Shanghai when the Agent cannot identify the host timezone', () => {
    const view = setupView()
    view.data.value = overview('unknown-timezone')

    view.openTool({ id: 'timezone' })

    expect(view.actionForm.timezone).toBe('')
    expect(view.actionForm.timezonePreset).toBe('__custom__')
  })
})
