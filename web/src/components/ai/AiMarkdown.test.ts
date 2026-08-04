// @vitest-environment jsdom
import { describe,expect,it } from 'vitest'
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
})
