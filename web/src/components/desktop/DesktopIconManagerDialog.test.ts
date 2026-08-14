// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DesktopIconManagerDialog from './DesktopIconManagerDialog.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('DesktopIconManagerDialog', () => {
  it('emits autoArrange from the layout action when wide-layout editing is available', async () => {
    const wrapper = mount(DesktopIconManagerDialog, {
      attachTo: document.body,
      props: {
        open: true,
        hiddenEntries: [],
        shortcuts: [],
        canAutoArrange: true,
      },
    })
    await nextTick()

    const action = document.body.querySelector<HTMLButtonElement>('.desktop-icon-manager__layout-action')
    expect(action?.disabled).toBe(false)
    action?.click()
    await nextTick()

    expect(wrapper.emitted('autoArrange')).toHaveLength(1)
    wrapper.unmount()
  })

  it('disables autoArrange when the layout is compact or a workspace save is active', async () => {
    const wrapper = mount(DesktopIconManagerDialog, {
      attachTo: document.body,
      props: {
        open: true,
        hiddenEntries: [],
        shortcuts: [],
        canAutoArrange: false,
      },
    })
    await nextTick()

    const action = document.body.querySelector<HTMLButtonElement>('.desktop-icon-manager__layout-action')
    expect(action?.disabled).toBe(true)
    action?.click()
    expect(wrapper.emitted('autoArrange')).toBeUndefined()

    await wrapper.setProps({ canAutoArrange: true, busy: true })
    expect(action?.disabled).toBe(true)
    action?.click()
    expect(wrapper.emitted('autoArrange')).toBeUndefined()
    wrapper.unmount()
  })
})
