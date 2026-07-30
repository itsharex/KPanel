import { createSSRApp, nextTick, ssrContextKey, type ComputedRef, type Ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SitesView from './SitesView.vue'
import type { Site, SiteInstallationProgress } from '@/types/api'

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  isTransientAgentError: () => false,
  api: {
    agent: { capabilities: vi.fn() },
    sites: {
      list: vi.fn(),
      installations: vi.fn(),
      installation: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
    },
    system: { publicNetwork: vi.fn() },
  },
}))

vi.mock('@/stores/panel', () => ({
  usePanelState: () => ({
    isReadOnly: { value: false },
  }),
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: vi.fn(),
    danger: vi.fn(),
  }),
}))

interface SitesBindings {
  sites: Ref<Site[]>
  filteredSites: ComputedRef<Site[]>
  search: Ref<string>
  filter: Ref<'all' | 'healthy' | 'drifted' | 'config-only'>
  showMoreTemplates: Ref<boolean>
  siteList: Ref<HTMLElement | undefined>
  installationPanel: Ref<HTMLElement | undefined>
  installationTaskView: ComputedRef<boolean>
  installationTaskFinished: ComputedRef<boolean>
  installProgress: Ref<SiteInstallationProgress | undefined>
  editorOpen: Ref<boolean>
  recentCreatedDomain: Ref<string>
  featuredServiceOptions: ComputedRef<Array<{ type: string }>>
  standardServiceOptions: ComputedRef<Array<{ type: string }>>
  recipeOptions: Array<{ recipe: string }>
  closeEditor: () => void
  focusInstallationPanel: (force?: boolean) => Promise<void>
  revealCreatedSite: (domain: string) => Promise<void>
}

function setupView(): SitesBindings {
  const component = SitesView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => SitesBindings
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

function site(domain: string): Site {
  return {
    id: `site-${domain}`,
    primaryDomain: domain,
    domains: [domain],
    type: 'static',
    enabled: true,
    health: 'healthy',
    consistency: 'synced',
    access: 'managed',
    source: 'panel',
    resourceVersion: `sha256:${'a'.repeat(64)}`,
  }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('SitesView creation experience', () => {
  it('keeps only the three primary website types visible and folds every recipe', () => {
    const view = setupView()

    expect(view.showMoreTemplates.value).toBe(false)
    expect(view.featuredServiceOptions.value.map((item) => item.type)).toEqual([
      'wordpress',
      'proxy',
      'static',
    ])
    expect(view.standardServiceOptions.value.map((item) => item.type)).toEqual([
      'php',
      'proxy_domain',
      'load_balance',
      'redirect',
    ])
    expect(view.recipeOptions).toHaveLength(10)
  })

  it('returns to the list and highlights the newly created website first', async () => {
    const view = setupView()
    const scrollIntoView = vi.fn()
    view.sites.value = [site('old.example.com'), site('new.example.com')]
    view.search.value = 'old'
    view.filter.value = 'healthy'
    view.siteList.value = { scrollIntoView } as unknown as HTMLElement

    await view.revealCreatedSite('new.example.com')

    expect(view.search.value).toBe('')
    expect(view.filter.value).toBe('all')
    expect(view.recentCreatedDomain.value).toBe('new.example.com')
    expect(view.filteredSites.value.map((item) => item.primaryDomain)).toEqual([
      'new.example.com',
      'old.example.com',
    ])
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })
  })

  it('switches to the focused terminal view for the entire job lifecycle', async () => {
    const view = setupView()
    const scrollIntoView = vi.fn()
    view.editorOpen.value = true
    view.installationPanel.value = { scrollIntoView } as unknown as HTMLElement

    view.installProgress.value = {
      id: 'site-job-1',
      domain: 'new.example.com',
      interactive: true,
      inputOpen: true,
      status: 'running',
      stage: 'installing',
      progress: 38,
      message: '正在执行脚本。',
    }
    await nextTick()

    expect(view.installationTaskView.value).toBe(true)
    expect(view.installationTaskFinished.value).toBe(false)
    await view.focusInstallationPanel()
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })

    view.installProgress.value = {
      ...view.installProgress.value,
      status: 'succeeded',
      stage: 'completed',
      progress: 100,
      message: '搭建完成。',
    }

    expect(view.installationTaskView.value).toBe(true)
    expect(view.installationTaskFinished.value).toBe(true)
  })

  it('keeps a completed job open until the user closes it, then highlights the site', async () => {
    const view = setupView()
    const scrollIntoView = vi.fn()
    view.sites.value = [site('new.example.com')]
    view.siteList.value = { scrollIntoView } as unknown as HTMLElement
    view.editorOpen.value = true
    view.installProgress.value = {
      id: 'site-job-2',
      domain: 'new.example.com',
      interactive: true,
      inputOpen: false,
      status: 'succeeded',
      stage: 'completed',
      progress: 100,
      message: '搭建完成。',
    }

    expect(view.editorOpen.value).toBe(true)
    view.closeEditor()
    await nextTick()

    expect(view.editorOpen.value).toBe(false)
    expect(view.recentCreatedDomain.value).toBe('new.example.com')
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'start' })
  })
})
