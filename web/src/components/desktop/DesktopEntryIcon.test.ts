// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DesktopEntryIcon from './DesktopEntryIcon.vue'

function pointer(
  type: string,
  { x = 40, y = 50, id = 1, pointerType = 'touch' } = {},
): PointerEvent {
  const event = new Event(type, { bubbles: true, cancelable: true }) as PointerEvent
  Object.defineProperties(event, {
    button: { value: 0 },
    clientX: { value: x },
    clientY: { value: y },
    pointerId: { value: id },
    pointerType: { value: pointerType },
  })
  return event
}

function mountIcon() {
  return mount(DesktopEntryIcon, {
    attachTo: document.body,
    props: {
      label: '概览',
      gradient: 'linear-gradient(#0aa, #066)',
    },
  })
}

describe('DesktopEntryIcon touch interaction', () => {
  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ''
  })

  it('keeps Windows mouse selection and double-click activation', async () => {
    const wrapper = mountIcon()
    wrapper.element.dispatchEvent(pointer('pointerdown', { pointerType: 'mouse' }))

    await wrapper.trigger('click')
    expect(wrapper.emitted('select')).toHaveLength(1)
    expect(wrapper.emitted('open')).toBeUndefined()

    await wrapper.trigger('dblclick')
    expect(wrapper.emitted('open')).toHaveLength(1)
    wrapper.unmount()
  })

  it('opens once on a touch tap and ignores synthesized double-click events', async () => {
    const wrapper = mountIcon()
    wrapper.element.dispatchEvent(pointer('pointerdown'))
    window.dispatchEvent(pointer('pointerup'))

    await wrapper.trigger('click')
    await wrapper.trigger('click')
    await wrapper.trigger('dblclick')

    expect(wrapper.emitted('select')).toBeUndefined()
    expect(wrapper.emitted('open')).toHaveLength(1)
    wrapper.unmount()
  })

  it('opens the existing context menu on long press and consumes the following click', async () => {
    vi.useFakeTimers()
    const wrapper = mountIcon()
    wrapper.element.dispatchEvent(pointer('pointerdown', { x: 72, y: 84 }))

    vi.advanceTimersByTime(520)
    window.dispatchEvent(pointer('pointerup', { x: 72, y: 84 }))
    await wrapper.trigger('click')
    await wrapper.trigger('contextmenu')

    expect(wrapper.emitted('context')).toHaveLength(1)
    expect(wrapper.emitted('open')).toBeUndefined()
    wrapper.unmount()
  })

  it('cancels a long press when the finger moves or the pointer is cancelled', () => {
    vi.useFakeTimers()
    const moved = mountIcon()
    moved.element.dispatchEvent(pointer('pointerdown'))
    window.dispatchEvent(pointer('pointermove', { x: 56, y: 66 }))
    vi.advanceTimersByTime(600)
    expect(moved.emitted('context')).toBeUndefined()
    moved.unmount()

    const cancelled = mountIcon()
    cancelled.element.dispatchEvent(pointer('pointerdown', { id: 2 }))
    window.dispatchEvent(pointer('pointercancel', { id: 2 }))
    vi.advanceTimersByTime(600)
    expect(cancelled.emitted('context')).toBeUndefined()
    cancelled.unmount()
  })

  it('supports the keyboard context-menu key', async () => {
    const wrapper = mountIcon()
    await wrapper.trigger('keydown', { key: 'ContextMenu' })
    expect(wrapper.emitted('context')).toHaveLength(1)
    wrapper.unmount()
  })
})
