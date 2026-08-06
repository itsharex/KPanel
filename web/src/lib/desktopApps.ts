import type { Component } from 'vue'
import {
  Boxes,
  Bot,
  ClipboardList,
  Container,
  Folder,
  HeartPulse,
  LayoutDashboard,
  Network,
  Settings,
  SquareTerminal,
  Store,
} from '@lucide/vue'
import type { MessageKey } from '@/i18n/messages/zh-CN'

/**
 * Desktop application catalogue. Mirrors the left-navigation items in
 * AppShell.vue, one icon per route page. The terminal is a single-instance app
 * (opening it twice focuses the existing window).
 */

export interface DesktopApp {
  path: string
  labelKey: MessageKey
  icon: Component
  allowMultiple: boolean
}

export const desktopApps: DesktopApp[] = [
  { path: '/overview', labelKey: 'route.overview', icon: LayoutDashboard, allowMultiple: true },
  { path: '/ai', labelKey: 'route.ai', icon: Bot, allowMultiple: true },
  { path: '/sites', labelKey: 'route.sites', icon: Boxes, allowMultiple: true },
  { path: '/apps', labelKey: 'route.apps', icon: Store, allowMultiple: true },
  { path: '/docker', labelKey: 'route.docker', icon: Container, allowMultiple: true },
  { path: '/files', labelKey: 'route.files', icon: Folder, allowMultiple: true },
  { path: '/terminal', labelKey: 'route.terminal', icon: SquareTerminal, allowMultiple: false },
  { path: '/diagnostics', labelKey: 'route.diagnostics', icon: HeartPulse, allowMultiple: true },
  { path: '/cluster', labelKey: 'route.cluster', icon: Network, allowMultiple: true },
  { path: '/activity', labelKey: 'route.activity', icon: ClipboardList, allowMultiple: true },
  { path: '/settings', labelKey: 'route.settings', icon: Settings, allowMultiple: true },
]

export function findDesktopApp(path: string): DesktopApp | undefined {
  return desktopApps.find((app) => app.path === path)
}
