// @vitest-environment jsdom
import { describe,expect,it,vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AiMarkdown from './AiMarkdown.vue'

describe('AiMarkdown',()=>{
  it('renders markdown and removes executable HTML',()=>{
    const wrapper=mount(AiMarkdown,{props:{content:'**安全** <img src=x onerror="alert(1)"><script>alert(2)</script>'}})
    expect(wrapper.html()).toContain('<strong>安全</strong>')
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.html()).toContain('&lt;img')
  })

  it('copies fenced code from the top-right action',async()=>{
    const writeText=vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator,'clipboard',{value:{writeText},configurable:true})
    const wrapper=mount(AiMarkdown,{props:{content:'```sh\necho safe\n```'}})
    const button=wrapper.get('button[aria-label="复制代码"]')
    await button.trigger('click')
    expect(writeText).toHaveBeenCalledWith('echo safe\n')
    expect(button.text()).toBe('已复制')
  })
})
