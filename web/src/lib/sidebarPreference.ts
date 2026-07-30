const SIDEBAR_COLLAPSED_KEY = 'kejilion-panel-sidebar-collapsed'

interface SidebarPreferenceReader {
  getItem: (key: string) => string | null
}

interface SidebarPreferenceWriter {
  setItem: (key: string, value: string) => void
}

export function readSidebarCollapsed(storage: SidebarPreferenceReader = localStorage): boolean {
  try {
    return storage.getItem(SIDEBAR_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

export function writeSidebarCollapsed(
  collapsed: boolean,
  storage: SidebarPreferenceWriter = localStorage,
): void {
  try {
    storage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // Storage can be unavailable in hardened browser contexts; the in-memory state still works.
  }
}
