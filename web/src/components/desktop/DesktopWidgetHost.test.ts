// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { defineComponent, h, markRaw } from 'vue'
import { Clock3 } from '@lucide/vue'
import { mount } from '@vue/test-utils'
import DesktopWidgetHost from './DesktopWidgetHost.vue'
import type { DesktopWidgetDefinition } from '@/lib/desktopWidgets'

const widget: DesktopWidgetDefinition = {
  key: 'widget:test',
  component: markRaw(defineComponent({
    render: () => h('div', [
      h('header', { class: 'desktop-widget__drag-handle' }, 'Test widget'),
      h('button', { type: 'button' }, 'Open'),
    ]),
  })),
  icon: Clock3,
  titleKey: 'desktop.widgetClockTitle',
  descriptionKey: 'desktop.widgetClockDescription',
  tone: 'brand',
  columns: 2,
  rows: 1,
  defaultSlot: () => ({ column: 0, row: 0 }),
}

describe('DesktopWidgetHost', () => {
  it('starts pointer dragging from any non-interactive widget area', async () => {
    const wrapper = mount(DesktopWidgetHost, { props: { widget } })
    await wrapper.find('.desktop-widget-slot').trigger('pointerdown')
    expect(wrapper.emitted('drag-start')).toHaveLength(1)
  })

  it('does not steal pointer input from an interactive widget control', async () => {
    const wrapper = mount(DesktopWidgetHost, { props: { widget } })
    await wrapper.get('button').trigger('pointerdown')
    expect(wrapper.emitted('drag-start')).toBeUndefined()
  })

  it('emits a grid nudge for Ctrl/Cmd plus an arrow key', async () => {
    const wrapper = mount(DesktopWidgetHost, { props: { widget } })
    await wrapper.find('.desktop-widget-slot').trigger('keydown', { key: 'ArrowDown', ctrlKey: true })
    expect(wrapper.emitted('nudge')).toEqual([['down']])
  })
})
