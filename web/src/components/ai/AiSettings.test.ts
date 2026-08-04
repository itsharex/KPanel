// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AiSettings from './AiSettings.vue'

const mocks=vi.hoisted(()=>({create:vi.fn(),update:vi.fn(),remove:vi.fn(),test:vi.fn(),sync:vi.fn(),addModel:vi.fn()}))

vi.mock('@/lib/aiApi', () => ({
  aiApi: {
    providers: mocks,
    evolution: {
      memories: vi.fn().mockResolvedValue([]), procedures: vi.fn().mockResolvedValue([]), proposals: vi.fn().mockResolvedValue([]),
      approve: vi.fn(), reject: vi.fn(), updateMemory: vi.fn(), removeMemory: vi.fn(), updateProcedure: vi.fn(), removeProcedure: vi.fn(),
    },
  },
}))

describe('AI provider API mode', () => {
  it('defaults OpenAI to Responses and keeps compatibility presets on Chat Completions', async () => {
    const wrapper = mount(AiSettings, { props: { providers: [], models: [] } })
    await flushPromises()
    expect(wrapper.get('select[aria-label="OpenAI API 模式"]').element).toHaveProperty('value', 'responses')

    await wrapper.get('select[aria-label="Provider 快速预设"]').setValue('openrouter')
    expect(wrapper.get('select[aria-label="OpenAI API 模式"]').element).toHaveProperty('value', 'chat_completions')

    const protocol = wrapper.findAll('select').find(select => select.find('option[value="anthropic"]').exists())
    await protocol!.setValue('anthropic')
    expect(wrapper.find('select[aria-label="OpenAI API 模式"]').exists()).toBe(false)
  })

  it('restores the saved mode when editing a provider', async () => {
    const wrapper = mount(AiSettings, { props: { providers: [{
      id: 'p1', name: 'Responses', protocol: 'openai_compatible', apiMode: 'responses', baseUrl: 'https://api.example.com/v1',
      endpointScope: 'public', enabled: true, apiKeySet: true, version: 1, createdAt: '', updatedAt: '',
    }], models: [] } })
    await wrapper.get('.ai-provider-card__main').trigger('click')
    expect(wrapper.get('select[aria-label="OpenAI API 模式"]').element).toHaveProperty('value', 'responses')
    expect(wrapper.text()).toContain('Responses')
  })

  it('saves, verifies and syncs a new API in one guided flow', async () => {
    mocks.create.mockResolvedValue({id:'p-new',name:'OpenAI',protocol:'openai_compatible',apiMode:'responses',baseUrl:'https://api.openai.com/v1',endpointScope:'public',enabled:true,apiKeySet:true,version:1,createdAt:'',updatedAt:''})
    mocks.test.mockResolvedValue({ok:true})
    mocks.sync.mockResolvedValue([{id:'m1'}])
    const wrapper=mount(AiSettings,{props:{providers:[],models:[]}})
    await wrapper.get('.ai-secret-input input').setValue('secret-for-test')
    await wrapper.get('.ai-provider-submit .button--primary').trigger('click')
    await flushPromises()
    expect(mocks.create).toHaveBeenCalled()
    expect(mocks.test).toHaveBeenCalledWith('p-new')
    expect(mocks.sync).toHaveBeenCalledWith('p-new')
    expect(wrapper.text()).toContain('连接成功，已同步 1 个模型')
  })
})
