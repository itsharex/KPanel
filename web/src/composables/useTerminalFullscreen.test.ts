// @vitest-environment jsdom

import { defineComponent, nextTick } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useTerminalFullscreen } from './useTerminalFullscreen'

const Harness = defineComponent({
  setup() {
    const refreshed = vi.fn()
    const controls = useTerminalFullscreen(refreshed)
    return { refreshed, ...controls }
  },
  template: `
    <section :class="{ 'is-fullscreen': fullscreen }">
      <button type="button" @click="toggleFullscreen">toggle</button>
    </section>
  `,
})

describe('useTerminalFullscreen', () => {
  let wrapper: VueWrapper
  const requestFullscreen = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0)
      return 1
    })
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
      configurable: true,
      value: requestFullscreen,
    })
    wrapper = mount(Harness, { attachTo: document.body })
  })

  afterEach(() => {
    wrapper.unmount()
    document.documentElement.classList.remove('terminal-fullscreen-open')
    document.body.classList.remove('terminal-fullscreen-open')
    delete (HTMLElement.prototype as { requestFullscreen?: () => Promise<void> }).requestFullscreen
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('fills only the webpage and restores on the second action', async () => {
    await wrapper.get('button').trigger('click')
    await nextTick()
    expect(requestFullscreen).not.toHaveBeenCalled()
    expect(wrapper.get('section').classes()).toContain('is-fullscreen')
    expect(document.documentElement.classList.contains('terminal-fullscreen-open')).toBe(true)
    expect(document.body.classList.contains('terminal-fullscreen-open')).toBe(true)

    await wrapper.get('button').trigger('click')
    await nextTick()
    expect(wrapper.get('section').classes()).not.toContain('is-fullscreen')
    expect(document.documentElement.classList.contains('terminal-fullscreen-open')).toBe(false)
    expect(document.body.classList.contains('terminal-fullscreen-open')).toBe(false)
  })

  it('uses Escape to restore without propagating to an outer modal', async () => {
    await wrapper.get('button').trigger('click')
    await nextTick()
    const outerEscape = vi.fn()
    window.addEventListener('keydown', outerEscape)
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    await nextTick()

    expect(outerEscape).not.toHaveBeenCalled()
    expect(wrapper.get('section').classes()).not.toContain('is-fullscreen')
    window.removeEventListener('keydown', outerEscape)
  })
})
