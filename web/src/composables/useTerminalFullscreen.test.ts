// @vitest-environment jsdom

import { defineComponent, nextTick, ref } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useTerminalFullscreen } from './useTerminalFullscreen'

const Harness = defineComponent({
  setup() {
    const target = ref<HTMLElement>()
    const refreshed = vi.fn()
    const controls = useTerminalFullscreen(target, refreshed)
    return { target, refreshed, ...controls }
  },
  template: `
    <section ref="target" :class="{ 'is-fullscreen': fullscreen }">
      <button type="button" @click="toggleFullscreen">toggle</button>
    </section>
  `,
})

describe('useTerminalFullscreen', () => {
  let wrapper: VueWrapper
  let fullscreenElement: Element | null
  const requestFullscreen = vi.fn()
  const exitFullscreen = vi.fn()

  beforeEach(() => {
    fullscreenElement = null
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0)
      return 1
    })
    Object.defineProperty(document, 'fullscreenElement', {
      configurable: true,
      get: () => fullscreenElement,
    })
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
      configurable: true,
      value: requestFullscreen.mockImplementation(function (this: HTMLElement) {
        fullscreenElement = this
        document.dispatchEvent(new Event('fullscreenchange'))
        return Promise.resolve()
      }),
    })
    Object.defineProperty(document, 'exitFullscreen', {
      configurable: true,
      value: exitFullscreen.mockImplementation(() => {
        fullscreenElement = null
        document.dispatchEvent(new Event('fullscreenchange'))
        return Promise.resolve()
      }),
    })
    wrapper = mount(Harness, { attachTo: document.body })
  })

  afterEach(() => {
    wrapper.unmount()
    document.body.classList.remove('terminal-fullscreen-open')
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('uses native fullscreen and restores on the second action', async () => {
    await wrapper.get('button').trigger('click')
    await nextTick()
    expect(requestFullscreen).toHaveBeenCalledTimes(1)
    expect(wrapper.get('section').classes()).toContain('is-fullscreen')
    expect(document.body.classList.contains('terminal-fullscreen-open')).toBe(true)

    await wrapper.get('button').trigger('click')
    await nextTick()
    expect(exitFullscreen).toHaveBeenCalledTimes(1)
    expect(wrapper.get('section').classes()).not.toContain('is-fullscreen')
  })

  it('falls back to viewport fullscreen when native fullscreen is rejected', async () => {
    requestFullscreen.mockRejectedValueOnce(new Error('blocked'))
    await wrapper.get('button').trigger('click')
    await nextTick()

    expect(wrapper.get('section').classes()).toContain('is-fullscreen')
    expect(document.body.classList.contains('terminal-fullscreen-open')).toBe(true)
    expect((wrapper.vm as unknown as { fallbackFullscreen: boolean }).fallbackFullscreen).toBe(true)
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
