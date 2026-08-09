// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import WebBrowserView from './WebBrowserView.vue'

const mocks = vi.hoisted(() => ({
  query: {} as Record<string, string>,
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: mocks.query }),
}))

describe('WebBrowserView', () => {
  beforeEach(() => {
    mocks.query = {}
    window.open = vi.fn()
  })

  it('embeds a validated website in a sandbox and provides an external-browser action', async () => {
    mocks.query = { url: 'https://blog.example.com/path' }
    const wrapper = mount(WebBrowserView)

    const frame = wrapper.get('iframe.embedded-browser__frame')
    expect(frame.attributes('src')).toBe('https://blog.example.com/path')
    expect(frame.attributes('sandbox')).toContain('allow-scripts')
    expect(frame.attributes('sandbox')).not.toContain('allow-top-navigation')

    await wrapper.get('.embedded-browser__external').trigger('click')
    expect(window.open).toHaveBeenCalledWith(
      'https://blog.example.com/path',
      '_blank',
      'noopener,noreferrer',
    )
  })

  it('does not create a frame for an unsafe URL', () => {
    mocks.query = { url: 'javascript:alert(1)' }
    const wrapper = mount(WebBrowserView)

    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.get('.embedded-browser__external').attributes('disabled')).toBeDefined()
  })
})
