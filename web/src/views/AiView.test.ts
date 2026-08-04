// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AiView from './AiView.vue'

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  messages: vi.fn(),
  providers: vi.fn(),
  models: vi.fn(),
  sessions: vi.fn(),
  update: vi.fn(),
}))

vi.mock('@/lib/aiApi', () => ({
  runEventURL: (id: string) => `/api/v1/ai/runs/${id}/events`,
  aiApi: {
    providers: { list: mocks.providers },
    models: mocks.models,
    sessions: {
      list: mocks.sessions,
      messages: mocks.messages,
      create: mocks.create, update: mocks.update, remove: vi.fn(), send: vi.fn(),
    },
    runs: { get: vi.fn(), decision: vi.fn(), cancel: vi.fn(), retry: vi.fn(), propose: vi.fn() },
    evolution: { memories: vi.fn(), procedures: vi.fn(), proposals: vi.fn() },
  },
}))

class MockEventSource {
  static urls: string[] = []
  static instances: MockEventSource[] = []
  listeners = new Map<string,((event:{data:string})=>void)[]>()
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  constructor(url: string) { MockEventSource.urls.push(url);MockEventSource.instances.push(this) }
  addEventListener(name:string,handler:(event:{data:string})=>void) { this.listeners.set(name,[...(this.listeners.get(name)||[]),handler]) }
  emit(name:string,value:unknown){for(const handler of this.listeners.get(name)||[])handler({data:JSON.stringify(value)})}
  close() {}
}

beforeEach(() => {
  vi.clearAllMocks()
  MockEventSource.urls = []
  MockEventSource.instances = []
  vi.stubGlobal('EventSource', MockEventSource)
  Element.prototype.scrollIntoView = vi.fn()
  mocks.providers.mockResolvedValue([
    { id: 'p1', name: 'Primary', enabled: true },
    { id: 'p2', name: 'Secondary', enabled: true },
  ])
  mocks.models.mockResolvedValue([
    { id: 'm1', providerId: 'p1', modelId: 'mock', displayName: 'Mock', contextWindow: 8192, enabled: true, isDefault: true, toolCalling: true },
    { id: 'm2', providerId: 'p2', modelId: 'next', displayName: 'Next', contextWindow: 32768, enabled: true, isDefault: false, toolCalling: true },
  ])
  mocks.sessions.mockResolvedValue([{
    id: 's1', title: 'Running', providerId: 'p1', modelId: 'm1', providerName: 'Mock', modelName: 'Mock',
    pinned: false, archived: false, modelAvailable: true, running: true, activeRunId: 'run-active',
    createdAt: '2026-08-04T00:00:00Z', updatedAt: '2026-08-04T00:00:00Z', lastMessageAt: '2026-08-04T00:00:00Z',
  }])
  mocks.messages.mockResolvedValue({ items: [], nextCursor: '' })
  mocks.update.mockImplementation(async (_id, body) => ({
    id: 's1', title: 'Running', providerId: body.providerId || 'p1', modelId: body.modelId || 'm1',
    providerName: body.providerId === 'p2' ? 'Secondary' : 'Primary', modelName: body.modelId === 'm2' ? 'Next' : 'Mock',
    modelAvailable: true, pinned: false, archived: false, running: true, activeRunId: 'run-active',
    createdAt: '2026-08-04T00:00:00Z', updatedAt: '2026-08-04T00:00:00Z', lastMessageAt: '2026-08-04T00:00:00Z',
  }))
})

function makeRouter(path='/ai/s/s1') {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/ai', component: AiView },
    { path: '/ai/s/:sessionId', component: AiView },
  ] })
  return router.push(path).then(()=>router.isReady()).then(()=>router)
}

describe('AI workspace reconnect', () => {
  it('reopens SSE for the active run after a route reload', async () => {
    const router = await makeRouter()
    const wrapper = mount(AiView, { global: { plugins: [router] } })
    await flushPromises()
    expect(mocks.messages).toHaveBeenCalledWith('s1')
    expect(MockEventSource.urls).toContain('/api/v1/ai/runs/run-active/events')
    wrapper.unmount()
  })

  it('creates a conversation with an explicitly selected provider and model', async () => {
    const router = await makeRouter()
    mocks.create.mockResolvedValue({ id: 's2', title: '巡检', providerId: 'p2', modelId: 'm2', providerName: 'Secondary', modelName: 'Next', modelAvailable: true, pinned: false, archived: false, running: false, createdAt: '', updatedAt: '', lastMessageAt: '' })
    const wrapper = mount(AiView, { global: { plugins: [router] } })
    await flushPromises()
    await wrapper.get('.ai-new-chat').trigger('click')
    await wrapper.get('select[aria-label="新会话模型"]').setValue('m2')
    await wrapper.get('.ai-new-session-form input').setValue('巡检')
    await wrapper.get('.ai-new-session-dialog .button--primary').trigger('click')
    await flushPromises()
    expect(mocks.create).toHaveBeenCalledWith('p2','m2','巡检')
    expect(router.currentRoute.value.fullPath).toBe('/ai/s/s2')
    wrapper.unmount()
  })

  it('switches provider and model during an active run for the next turn', async () => {
    const router = await makeRouter()
    const wrapper = mount(AiView, { global: { plugins: [router] } })
    await flushPromises()
    MockEventSource.instances[0]?.emit('run.snapshot',{run:{id:'run-active',sessionId:'s1',providerId:'p1',providerName:'Primary',modelId:'m1',modelName:'Mock',status:'running',step:1,usage:{inputTokens:0,outputTokens:0,totalTokens:0},createdAt:'',updatedAt:''},toolCalls:[],messages:[]})
    await flushPromises()
    const picker=wrapper.get('select[aria-label="选择模型"]')
    expect((picker.element as HTMLSelectElement).disabled).toBe(false)
    await picker.setValue('m2')
    await flushPromises()
    expect(mocks.update).toHaveBeenCalledWith('s1',{providerId:'p2',modelId:'m2'})
    expect(wrapper.text()).toContain('下一轮')
    wrapper.unmount()
  })

  it('loads archived conversations from the sidebar filter', async () => {
    const router = await makeRouter()
    const wrapper = mount(AiView, { global: { plugins: [router] } })
    await flushPromises()
    await wrapper.get('button[aria-label="查看已归档会话"]').trigger('click')
    await flushPromises()
    expect(mocks.sessions).toHaveBeenCalledWith('',true)
    wrapper.unmount()
  })
})
