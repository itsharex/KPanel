import type { Component } from 'vue'
import { Activity, Clock3, LayoutGrid } from '@lucide/vue'
import DesktopClock from '@/components/desktop/DesktopClock.vue'
import DesktopMonitor from '@/components/desktop/DesktopMonitor.vue'
import DesktopServiceStatus from '@/components/desktop/DesktopServiceStatus.vue'
import type { MessageKey } from '@/i18n/messages/zh-CN'
import type { DesktopGridItem } from '@/lib/desktopGridLayout'
import type { DesktopIconGrid, DesktopIconGridSlot } from '@/lib/desktopIconLayout'

/** Stable keys are persisted in the desktop workspace and must never be based on UI order. */
export interface DesktopWidgetDefinition extends Omit<DesktopGridItem, 'defaultSlot'> {
  component: Component
  icon: Component
  titleKey: MessageKey
  descriptionKey: MessageKey
  tone: 'brand' | 'blue' | 'violet'
  defaultSlot: (grid: DesktopIconGrid) => DesktopIconGridSlot
}

const rightColumn = (width: number, row: number) => (grid: DesktopIconGrid): DesktopIconGridSlot => ({
  column: Math.max(0, grid.columns - width),
  row,
})

/** Built-ins are intentionally data-driven so future widgets only register here. */
export const desktopWidgets: readonly DesktopWidgetDefinition[] = Object.freeze([
  {
    key: 'widget:clock',
    component: DesktopClock,
    icon: Clock3,
    titleKey: 'desktop.widgetClockTitle',
    descriptionKey: 'desktop.widgetClockDescription',
    tone: 'brand',
    columns: 4,
    rows: 2,
    defaultSlot: rightColumn(4, 0),
  },
  {
    key: 'widget:monitor',
    component: DesktopMonitor,
    icon: Activity,
    titleKey: 'desktop.widgetMonitorTitle',
    descriptionKey: 'desktop.widgetMonitorDescription',
    tone: 'blue',
    columns: 4,
    rows: 3,
    defaultSlot: rightColumn(4, 2),
  },
  {
    key: 'widget:services',
    component: DesktopServiceStatus,
    icon: LayoutGrid,
    titleKey: 'desktop.widgetServicesTitle',
    descriptionKey: 'desktop.widgetServicesDescription',
    tone: 'violet',
    columns: 4,
    rows: 3,
    defaultSlot: rightColumn(4, 5),
  },
])

export const desktopWidgetKeys = new Set(desktopWidgets.map((widget) => widget.key))
