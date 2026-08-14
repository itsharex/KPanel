<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import type { Component } from 'vue'
import {
  ArrowLeft,
  CircleArrowUp,
  RefreshCw,
  Sun,
  Moon,
  Info,
  Pencil,
  SquareTerminal,
  AppWindow,
  ExternalLink,
  Globe2,
  ListTree,
  Grid2X2,
  MonitorCog,
  Plus,
  Trash2,
  EyeOff,
  X,
} from '@lucide/vue'
import DesktopWindow from '@/components/desktop/DesktopWindow.vue'
import DesktopEntryIcon from '@/components/desktop/DesktopEntryIcon.vue'
import DesktopClock from '@/components/desktop/DesktopClock.vue'
import DesktopMonitor from '@/components/desktop/DesktopMonitor.vue'
import DesktopIconManagerDialog from '@/components/desktop/DesktopIconManagerDialog.vue'
import DesktopShortcutDialog, {
  type DesktopShortcutDraft,
} from '@/components/desktop/DesktopShortcutDialog.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import LogoMark from '@/components/common/LogoMark.vue'
import { DEFAULT_WINDOW_GRADIENT, desktopApps, findDesktopApp } from '@/lib/desktopApps'
import {
  getCachedDesktopEntries,
  loadDesktopEntries,
  type DesktopEntries,
  type DesktopEntry,
} from '@/lib/desktopEntries'
import { api, ApiError, type SystemResourceSnapshot } from '@/lib/api'
import {
  autoArrangeDesktopIcons,
  deriveDesktopIconLayout,
  desktopIconPixelsToPosition,
  desktopIconPositionToPixels,
  dropDesktopIcon,
  MAX_DESKTOP_ICON_POSITIONS,
  moveDesktopIconByKeyboard,
  type DesktopIconBounds,
  type DesktopIconPlacement,
} from '@/lib/desktopIconLayout'
import { prefetchNavigationRoute } from '@/lib/navigation'
import {
  desktopCloseGuardCoordinator,
  desktopCloseGuardCoordinatorKey,
} from '@/lib/desktopRouteKeys'
import { useDesktopMode } from '@/stores/desktopMode'
import { useDesktopIcons } from '@/stores/desktopIcons'
import { useTheme } from '@/stores/theme'
import { useToast } from '@/stores/toast'
import { useI18n } from '@/i18n'
import type { AgentStatus, DesktopIconPosition, DesktopShortcut } from '@/types/api'

/**
 * Desktop overlay with Windows-style selection/open behavior, desktop-side
 * clock and resource widgets, and a persistent bottom taskbar.
 */

const props = defineProps<{
  agent?: AgentStatus
  kpanelUpdateAvailable?: boolean
  kpanelUpdateDescription?: string
}>()

const desktop = useDesktopMode()
const desktopIcons = useDesktopIcons()
const theme = useTheme()
const toast = useToast()
const i18n = useI18n()
provide(desktopCloseGuardCoordinatorKey, desktopCloseGuardCoordinator)

const openWindows = computed(() => desktop.windows.value)
const focusedWindow = computed(() =>
  desktop.windows.value.find((windowState) => windowState.id === desktop.focusedId.value),
)
const agentStatus = computed(() => {
  const agent = props.agent
  if (!agent?.connected) return { state: 'offline', label: i18n.t('agent.offline') }
  if (!agent.compatible) return { state: 'incompatible', label: i18n.t('agent.incompatible') }
  if (agent.readOnly) return { state: 'read-only', label: i18n.t('agent.readOnly') }
  return { state: 'online', label: i18n.t('agent.online') }
})

// Dynamic entries: installed apps and configured sites surfaced as desktop
// icons that open their external URL.
const SITE_RENAMES_KEY = 'kpanel:desktop-site-names:v1'
const MAX_SITE_NAME_LENGTH = 48

function readSiteNames(): Record<string, string> {
  try {
    const raw = window.localStorage.getItem(SITE_RENAMES_KEY)
    if (!raw || raw.length > 16_000) return {}
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    return Object.fromEntries(
      Object.entries(parsed)
        .filter(([id, name]) => id.length <= 128 && typeof name === 'string')
        .map(([id, name]) => [id, String(name).trim().slice(0, MAX_SITE_NAME_LENGTH)])
        .filter(([, name]) => Boolean(name)),
    )
  } catch {
    return {}
  }
}

const siteNames = ref<Record<string, string>>(readSiteNames())
const siteAppearanceNames = ref<Record<string, string>>({})

function siteDomainName(entry: DesktopEntry): string {
  return entry.site?.primaryDomain || entry.name
}

function defaultSiteName(entry: DesktopEntry): string {
  return siteAppearanceNames.value[entry.id] || siteDomainName(entry)
}

function applySiteNames(value?: DesktopEntries): DesktopEntries | undefined {
  if (!value) return undefined
  const apply = (entry: DesktopEntry): DesktopEntry => {
    if (entry.kind !== 'site') return entry
    return { ...entry, name: siteNames.value[entry.id] || defaultSiteName(entry) }
  }
  return {
    ...value,
    sites: value.sites.map(apply),
    visible: value.visible.map(apply),
  }
}

const entries = ref<DesktopEntries | undefined>(applySiteNames(getCachedDesktopEntries()))
const systemResources = ref<SystemResourceSnapshot>()
const entriesLoading = ref(!entries.value)
let entriesAbort: AbortController | undefined
let entriesSequence = 0

const workspace = computed(() => desktopIcons.workspace.value)
const hiddenEntryKeys = computed(() => new Set(workspace.value.hiddenEntryKeys))
const visibleDynamicEntries = computed(() =>
  (entries.value?.visible || []).filter((entry) => !hiddenEntryKeys.value.has(entry.key)),
)
const hiddenEntries = computed(() =>
  (entries.value?.visible || []).filter((entry) => hiddenEntryKeys.value.has(entry.key)),
)
const shortcuts = computed<DesktopShortcut[]>(() => workspace.value.shortcuts.map((shortcut) => ({
  ...shortcut,
  iconURL: shortcut.iconURL
    || (shortcut.iconVersion ? api.desktop.shortcutIconURL(shortcut.id, shortcut.iconVersion) : undefined),
})))
const shortcutEntries = computed<DesktopEntry[]>(() => shortcuts.value.map((shortcut) => ({
  key: `shortcut:${shortcut.id}`,
  kind: 'shortcut',
  id: shortcut.id,
  name: shortcut.name,
  description: shortcut.description,
  launch: 'external',
  url: shortcut.url,
  iconURL: shortcut.iconURL,
  shortcut,
})))

const iconsElement = ref<HTMLElement>()
const iconBounds = ref<DesktopIconBounds>({ width: 90, height: 96 })
const compactIconLayout = ref(window.innerWidth <= 760)
const localPositions = ref<Record<string, DesktopIconPosition>>({})
const dragPreview = ref<{ key: string; left: number; top: number }>()
const draggingIcon = ref('')
const arrangingIcons = ref(false)
const iconAnnouncement = ref('')
const iconManagerOpen = ref(false)
const shortcutDialogOpen = ref(false)
const editingShortcut = ref<DesktopShortcut>()
const deletingShortcut = ref<DesktopShortcut>()
const removingEntry = ref<DesktopEntry>()
const shortcutSaving = ref(false)
const shortcutError = ref('')
let pendingShortcutID = ''
let workspaceAbort: AbortController | undefined
let iconsResizeObserver: ResizeObserver | undefined

const allIconKeys = computed(() => [
  ...desktopApps.map((app) => `nav:${app.path}`),
  ...visibleDynamicEntries.value.map((entry) => entry.key),
  ...shortcutEntries.value.map((entry) => entry.key),
])

const savedPlacements = computed<DesktopIconPlacement[]>(() =>
  Object.entries(localPositions.value).map(([key, position]) => ({ key, position })),
)

const renderedIconLayout = computed(() => deriveDesktopIconLayout(
  allIconKeys.value,
  savedPlacements.value,
  iconBounds.value,
  compactIconLayout.value,
))

const renderedPositionByKey = computed(() => new Map(
  renderedIconLayout.value.placements.map((placement) => [placement.key, placement.position]),
))
const renderedOverflowIndexByKey = computed(() => new Map(
  renderedIconLayout.value.overflowKeys.map((key, index) => [key, index]),
))
const iconOverflowStartTop = computed(() => (
  renderedIconLayout.value.contentHeight
  + (renderedIconLayout.value.overflowKeys.length ? 44 : 0)
))
const iconScrollHeight = computed(() => {
  const layout = renderedIconLayout.value
  const overflowCount = layout.overflowKeys.length
  if (!overflowCount) return Math.ceil(layout.contentHeight)
  const overflowRows = Math.ceil(overflowCount / layout.grid.columns)
  return Math.ceil(
    iconOverflowStartTop.value
    + Math.max(0, overflowRows - 1) * layout.grid.stepY
    + layout.grid.metrics.height,
  )
})

watch(() => workspace.value.positions, (positions) => {
  if (draggingIcon.value) return
  localPositions.value = Object.fromEntries(
    Object.entries(positions).map(([key, position]) => [key, { ...position }]),
  )
}, { deep: true, immediate: true })

watch([() => workspace.value.labels, () => desktopIcons.loaded.value], ([labels, isLoaded]) => {
  if (!isLoaded) return
  siteNames.value = Object.fromEntries(
    Object.entries(labels)
      .filter(([key]) => key.startsWith('site:'))
      .map(([key, name]) => [key.slice('site:'.length), name]),
  )
  entries.value = applySiteNames(entries.value)
}, { deep: true })

// Context menu: `targetEntry` set when the menu is for an entry icon; cleared
// for the empty-desktop menu.
const contextMenu = ref<{ x: number; y: number; open: boolean }>({ x: 0, y: 0, open: false })
const contextMenuTarget = ref<'desktop' | 'taskbar' | 'taskbar-window'>('desktop')
const contextMenuElement = ref<HTMLElement>()
const menuEntry = ref<DesktopEntry>()
const menuNavPath = ref('')
const menuWindowId = ref<number>()
const detailEntry = ref<DesktopEntry>()
const externalOpenEntry = ref<DesktopEntry>()
const externalOpenImageFailed = ref(false)
const renameEntry = ref<DesktopEntry>()
const renameValue = ref('')
let contextMenuOpener: HTMLElement | undefined

interface DesktopWindowHandle {
  requestClose: () => Promise<void>
}

const desktopWindowRefs = new Map<number, DesktopWindowHandle>()

function setDesktopWindowRef(windowId: number, instance: unknown): void {
  if (!instance) {
    desktopWindowRefs.delete(windowId)
    return
  }
  const handle = instance as Partial<DesktopWindowHandle>
  if (typeof handle.requestClose === 'function') {
    desktopWindowRefs.set(windowId, handle as DesktopWindowHandle)
  }
}

/** Icons currently playing their open-bounce animation. */
const bouncingIcon = ref<string>('')
const selectedIcon = ref<string>('')
let bounceTimer: number | undefined
let resizeFrame: number | undefined
let resizePersistTimer: number | undefined

function motionDuration(duration: number): number {
  return window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ? 0 : duration
}

function gradientFor(path: string): string {
  const gradient = findDesktopApp(path)?.gradient ?? DEFAULT_WINDOW_GRADIENT
  return `linear-gradient(145deg, ${gradient[0]} 0%, ${gradient[1]} 100%)`
}

const SITE_GRADIENTS: Array<[string, string]> = [
  ['#22d3ee', '#0e7490'],
  ['#34d399', '#047857'],
  ['#60a5fa', '#1d4ed8'],
  ['#a78bfa', '#6d28d9'],
  ['#f472b6', '#be185d'],
  ['#fb923c', '#c2410c'],
  ['#facc15', '#a16207'],
  ['#2dd4bf', '#0f766e'],
]

function stableSiteColorIndex(entry: DesktopEntry): number {
  const key = entry.site?.primaryDomain || entry.url || entry.id
  let hash = 2_166_136_261
  for (let index = 0; index < key.length; index += 1) {
    hash ^= key.charCodeAt(index)
    hash = Math.imul(hash, 16_777_619)
  }
  return (hash >>> 0) % SITE_GRADIENTS.length
}

function entryGradient(entry: DesktopEntry): string {
  if (entry.kind === 'site') {
    const [start, end] = SITE_GRADIENTS[stableSiteColorIndex(entry)] ?? ['#22d3ee', '#0e7490']
    return `linear-gradient(145deg, ${start} 0%, ${end} 100%)`
  }
  if (entry.kind === 'shortcut') {
    return 'linear-gradient(145deg, #38bdf8 0%, #0369a1 100%)'
  }
  // App-market apps keep a neutral brand tile; the market icon image sits on it.
  return `linear-gradient(145deg, #5b7a72 0%, #243b36 100%)`
}

function openApp(path: string): void {
  const app = findDesktopApp(path)
  if (!app) return
  // Open immediately so the interface feels responsive, while the icon keeps
  // a short launch animation as visual acknowledgement.
  if (bounceTimer !== undefined) window.clearTimeout(bounceTimer)
  bouncingIcon.value = path
  const windowId = desktop.openWindow(app.path, app.labelKey, app.allowMultiple)
  if (windowId === 0) {
    toast.show(i18n.t('desktop.windowLimitTitle'), {
      message: i18n.t('desktop.windowLimitMessage'),
    })
  }
  bounceTimer = window.setTimeout(() => {
    bouncingIcon.value = ''
    bounceTimer = undefined
  }, motionDuration(460))
}

function openKPanelUpdate(): void {
  const app = findDesktopApp('/apps')
  if (!app) return
  const windowId = desktop.openWindow(
    '/apps?app=kpanel&action=update',
    app.labelKey,
    app.allowMultiple,
    true,
  )
  if (windowId === 0) {
    toast.show(i18n.t('desktop.windowLimitTitle'), {
      message: i18n.t('desktop.windowLimitMessage'),
    })
  }
}

function openEntry(entry: DesktopEntry): void {
  if (entry.kind === 'app' && entry.launch === 'script') {
    openAppScriptEntry(entry)
    return
  }
  if (entry.url) {
    requestExternalOpen(entry)
    return
  }
}

function requestExternalOpen(entry: DesktopEntry): void {
  if (!entry.url) return
  externalOpenImageFailed.value = false
  externalOpenEntry.value = entry
}

function closeExternalOpen(): void {
  externalOpenEntry.value = undefined
}

function confirmExternalOpen(): void {
  const entry = externalOpenEntry.value
  if (!entry?.url) return
  window.open(entry.url, '_blank', 'noopener,noreferrer')
  closeExternalOpen()
}

const externalOpenMonogram = computed(() =>
  externalOpenEntry.value?.name.trim().slice(0, 1).toLocaleUpperCase() || 'K',
)

function openAppScriptEntry(entry: DesktopEntry): void {
  const path = `/app-script/${encodeURIComponent(entry.id)}`
  const app = findDesktopApp(path)
  if (!app) return
  const windowId = desktop.openWindow(path, app.labelKey, app.allowMultiple, true)
  if (windowId === 0) {
    toast.show(i18n.t('desktop.windowLimitTitle'), {
      message: i18n.t('desktop.windowLimitMessage'),
    })
  }
}

function openAppMarketEntry(entry: DesktopEntry): void {
  const app = findDesktopApp('/apps')
  if (!app) return
  const query = new URLSearchParams({ app: entry.id })
  const windowId = desktop.openWindow(
    `/apps?${query.toString()}`,
    app.labelKey,
    app.allowMultiple,
    true,
  )
  if (windowId === 0) {
    toast.show(i18n.t('desktop.windowLimitTitle'), {
      message: i18n.t('desktop.windowLimitMessage'),
    })
  }
}

function openNavIcon(path: string): void {
  openApp(path)
}

function selectNavIcon(path: string): void {
  selectedIcon.value = `nav:${path}`
}

function selectEntry(entry: DesktopEntry): void {
  selectedIcon.value = entry.key
}

function warmNavIcon(path: string): void {
  void prefetchNavigationRoute(path)
}

function windowIcon(path: string): Component {
  return findDesktopApp(path)?.icon ?? AppWindow
}

function scriptWindowEntry(path: string): DesktopEntry | undefined {
  const match = path.match(/^\/app-script\/([A-Za-z0-9_-]{1,128})(?:[?#]|$)/)
  if (!match) return undefined
  const appID = match[1]
  return entries.value?.apps.find((entry) => entry.id === appID)
    ?? entries.value?.visible.find((entry) => entry.kind === 'app' && entry.id === appID)
}

function windowIconURL(path: string): string | undefined {
  return scriptWindowEntry(path)?.iconURL
}

function windowTitle(titleKey: string, path?: string): string {
  const scriptEntry = path ? scriptWindowEntry(path) : undefined
  if (scriptEntry) return i18n.t('desktop.namedScriptWindowTitle', { name: scriptEntry.name })
  return i18n.t(titleKey as Parameters<typeof i18n.t>[0])
}

async function showContextMenu(
  event: MouseEvent,
  entry?: DesktopEntry,
  target: 'desktop' | 'taskbar' | 'taskbar-window' = 'desktop',
  windowId?: number,
  navPath = '',
): Promise<void> {
  event.preventDefault()
  contextMenuOpener = event.currentTarget instanceof HTMLElement
    ? event.currentTarget
    : document.activeElement instanceof HTMLElement
      ? document.activeElement
      : undefined
  contextMenu.value = { x: event.clientX, y: event.clientY, open: true }
  contextMenuTarget.value = target
  menuEntry.value = entry
  menuNavPath.value = navPath
  menuWindowId.value = windowId
  await nextTick()

  const menu = contextMenuElement.value
  if (!menu) return
  const rect = menu.getBoundingClientRect()
  const margin = 10
  contextMenu.value = {
    open: true,
    x: Math.min(Math.max(margin, event.clientX), Math.max(margin, window.innerWidth - rect.width - margin)),
    y: Math.min(Math.max(margin, event.clientY), Math.max(margin, window.innerHeight - rect.height - margin)),
  }
  menu.querySelector<HTMLButtonElement>('button')?.focus({ preventScroll: true })
}

function onContextMenu(event: MouseEvent): void {
  void showContextMenu(event)
}

function onEntryContext(event: MouseEvent, entry: DesktopEntry): void {
  selectEntry(entry)
  void showContextMenu(event, entry)
}

function onNavContext(event: MouseEvent, path: string): void {
  selectNavIcon(path)
  void showContextMenu(event, undefined, 'desktop', undefined, path)
}

function onTaskbarContext(event: MouseEvent): void {
  void showContextMenu(event, undefined, 'taskbar')
}

function onTaskbarItemContext(event: MouseEvent, windowId: number): void {
  void showContextMenu(event, undefined, 'taskbar-window', windowId)
}

function onEntryOpen(_event: MouseEvent | KeyboardEvent, entry: DesktopEntry): void {
  openEntry(entry)
}

function closeContextMenu(restoreFocus = true): void {
  contextMenu.value.open = false
  menuEntry.value = undefined
  menuNavPath.value = ''
  menuWindowId.value = undefined
  const opener = contextMenuOpener
  contextMenuOpener = undefined
  if (restoreFocus && opener?.isConnected) {
    void nextTick(() => opener.focus({ preventScroll: true }))
  }
}

function measureIconWorkArea(): void {
  const rect = iconsElement.value?.getBoundingClientRect()
  const fallbackRight = window.innerWidth > 900 ? 342 : 16
  iconBounds.value = {
    width: Math.max(90, rect?.width || window.innerWidth - fallbackRight),
    height: Math.max(96, rect?.height || window.innerHeight - 88),
  }
  compactIconLayout.value = window.innerWidth <= 760
}

function iconSlotStyle(key: string): Record<string, string> {
  if (dragPreview.value?.key === key) {
    return { left: `${dragPreview.value.left}px`, top: `${dragPreview.value.top}px` }
  }
  const position = renderedPositionByKey.value.get(key)
  if (position) {
    const pixels = desktopIconPositionToPixels(position, iconBounds.value)
    return { left: `${pixels.left}px`, top: `${pixels.top}px` }
  }
  const overflowIndex = renderedOverflowIndexByKey.value.get(key)
  if (overflowIndex === undefined) return { display: 'none' }
  const grid = renderedIconLayout.value.grid
  return {
    left: `${(overflowIndex % grid.columns) * grid.stepX}px`,
    top: `${iconOverflowStartTop.value + Math.floor(overflowIndex / grid.columns) * grid.stepY}px`,
  }
}

function workspaceErrorMessage(error: unknown): string {
  if (error instanceof ApiError && error.status === 409) return i18n.t('desktop.workspaceConflict')
  if (error instanceof ApiError) return error.message
  return i18n.t('desktop.workspaceSaveFailed')
}

async function persistPositions(next: Record<string, DesktopIconPosition>): Promise<void> {
  if (Object.keys(next).length > MAX_DESKTOP_ICON_POSITIONS) {
    const message = i18n.t('desktop.iconLayoutLimitMessage', { count: MAX_DESKTOP_ICON_POSITIONS })
    toast.danger(i18n.t('desktop.iconLayoutLimitTitle'), message)
    throw new Error(message)
  }
  localPositions.value = next
  try {
    await desktopIcons.mutate((draft) => {
      draft.positions = Object.fromEntries(
        Object.entries(next).map(([key, position]) => [key, { ...position }]),
      )
    })
  } catch (error) {
    localPositions.value = Object.fromEntries(
      Object.entries(workspace.value.positions).map(([key, position]) => [key, { ...position }]),
    )
    toast.danger(i18n.t('desktop.workspaceSaveErrorTitle'), workspaceErrorMessage(error))
    throw error
  }
}

function placementsToPositions(
  placements: DesktopIconPlacement[],
  base = localPositions.value,
): Record<string, DesktopIconPosition> {
  const next = Object.fromEntries(
    Object.entries(base).map(([key, position]) => [key, { ...position }]),
  )
  for (const placement of placements) next[placement.key] = { ...placement.position }
  return next
}

interface IconDragState {
  key: string
  pointerId: number
  pointerType: string
  captureTarget?: HTMLElement
  pointerCaptured: boolean
  startX: number
  startY: number
  lastX: number
  lastY: number
  startScrollTop: number
  originLeft: number
  originTop: number
  moved: boolean
}

let iconDrag: IconDragState | undefined
let iconAutoScrollFrame: number | undefined
const suppressActivationAfterDrag = new Set<string>()

function removeIconDragListeners(): void {
  window.removeEventListener('pointermove', onIconDragMove)
  window.removeEventListener('pointerup', onIconDragEnd)
  window.removeEventListener('pointercancel', onIconDragCancel)
  window.removeEventListener('blur', cancelIconDrag)
  iconsElement.value?.removeEventListener('scroll', onIconDragScroll)
}

function stopIconAutoScroll(): void {
  if (iconAutoScrollFrame === undefined) return
  window.cancelAnimationFrame(iconAutoScrollFrame)
  iconAutoScrollFrame = undefined
}

function updateIconDragPreview(drag: IconDragState): void {
  const scrollDelta = (iconsElement.value?.scrollTop || 0) - drag.startScrollTop
  const normalized = desktopIconPixelsToPosition(
    {
      left: drag.originLeft + drag.lastX - drag.startX,
      top: drag.originTop + drag.lastY - drag.startY + scrollDelta,
    },
    iconBounds.value,
  )
  const pixels = desktopIconPositionToPixels(normalized, iconBounds.value)
  dragPreview.value = { key: drag.key, ...pixels }
}

function iconAutoScrollVelocity(clientY: number): number {
  const element = iconsElement.value
  const rect = element?.getBoundingClientRect()
  if (!element || !rect || rect.height <= 0 || element.scrollHeight <= element.clientHeight) return 0
  const edge = Math.min(56, Math.max(32, rect.height * 0.12))
  if (clientY < rect.top + edge) {
    return -Math.ceil(Math.min(1, (rect.top + edge - clientY) / edge) * 18)
  }
  if (clientY > rect.bottom - edge) {
    return Math.ceil(Math.min(1, (clientY - (rect.bottom - edge)) / edge) * 18)
  }
  return 0
}

function scheduleIconAutoScroll(): void {
  if (iconAutoScrollFrame !== undefined) return
  const drag = iconDrag
  if (!drag?.moved || !iconAutoScrollVelocity(drag.lastY)) return
  iconAutoScrollFrame = window.requestAnimationFrame(() => {
    iconAutoScrollFrame = undefined
    const active = iconDrag
    const element = iconsElement.value
    if (!active?.moved || !element) return
    const velocity = iconAutoScrollVelocity(active.lastY)
    if (!velocity) return
    const before = element.scrollTop
    element.scrollTop = Math.max(
      0,
      Math.min(element.scrollHeight - element.clientHeight, before + velocity),
    )
    if (element.scrollTop === before) return
    updateIconDragPreview(active)
    scheduleIconAutoScroll()
  })
}

function onIconDragScroll(): void {
  const drag = iconDrag
  if (drag?.moved) updateIconDragPreview(drag)
}

function releaseIconDragPointer(drag: IconDragState): void {
  const target = drag.captureTarget
  if (!target || !drag.pointerCaptured) return
  target.removeEventListener('lostpointercapture', onIconDragLostPointerCapture)
  try {
    if (
      typeof target.releasePointerCapture === 'function'
      && (typeof target.hasPointerCapture !== 'function' || target.hasPointerCapture(drag.pointerId))
    ) {
      target.releasePointerCapture(drag.pointerId)
    }
  } catch {
    // Pointer capture may already have been released by the browser.
  }
  drag.pointerCaptured = false
}

function captureIconDragPointer(drag: IconDragState): void {
  const target = drag.captureTarget
  if (drag.pointerCaptured || !target || typeof target.setPointerCapture !== 'function') return
  target.addEventListener('lostpointercapture', onIconDragLostPointerCapture)
  try {
    target.setPointerCapture(drag.pointerId)
    drag.pointerCaptured = true
  } catch {
    target.removeEventListener('lostpointercapture', onIconDragLostPointerCapture)
    // Window listeners keep dragging functional when capture is unavailable.
  }
}

function beginIconDrag(event: PointerEvent, key: string): void {
  if (event.button === 0 && event.isPrimary !== false) suppressActivationAfterDrag.delete(key)
  if (compactIconLayout.value || event.button !== 0 || event.isPrimary === false || iconDrag) return
  const pointerType = event.pointerType || 'mouse'
  if ((pointerType === 'touch' || pointerType === 'pen') && !arrangingIcons.value) return
  const position = renderedPositionByKey.value.get(key)
  if (!position) return
  const origin = desktopIconPositionToPixels(position, iconBounds.value)
  iconDrag = {
    key,
    pointerId: event.pointerId,
    pointerType,
    captureTarget: event.currentTarget instanceof HTMLElement ? event.currentTarget : undefined,
    pointerCaptured: false,
    startX: event.clientX,
    startY: event.clientY,
    lastX: event.clientX,
    lastY: event.clientY,
    startScrollTop: iconsElement.value?.scrollTop || 0,
    originLeft: origin.left,
    originTop: origin.top,
    moved: false,
  }
  window.addEventListener('pointermove', onIconDragMove, { passive: false })
  window.addEventListener('pointerup', onIconDragEnd)
  window.addEventListener('pointercancel', onIconDragCancel)
  window.addEventListener('blur', cancelIconDrag)
  iconsElement.value?.addEventListener('scroll', onIconDragScroll, { passive: true })
}

function onIconDragMove(event: PointerEvent): void {
  const drag = iconDrag
  if (!drag || event.pointerId !== drag.pointerId) return
  const deltaX = event.clientX - drag.startX
  const deltaY = event.clientY - drag.startY
  drag.lastX = event.clientX
  drag.lastY = event.clientY
  const threshold = drag.pointerType === 'mouse' ? 6 : 12
  if (!drag.moved && Math.hypot(deltaX, deltaY) < threshold) return
  if (!drag.moved) {
    captureIconDragPointer(drag)
    drag.moved = true
    draggingIcon.value = drag.key
    selectedIcon.value = drag.key
    closeContextMenu(false)
    document.body.classList.add('desktop-icon-dragging')
  }
  event.preventDefault()
  updateIconDragPreview(drag)
  scheduleIconAutoScroll()
}

function finishIconDrag(): IconDragState | undefined {
  const drag = iconDrag
  iconDrag = undefined
  stopIconAutoScroll()
  removeIconDragListeners()
  if (drag) releaseIconDragPointer(drag)
  document.body.classList.remove('desktop-icon-dragging')
  draggingIcon.value = ''
  return drag
}

function onIconDragEnd(event: PointerEvent): void {
  const drag = iconDrag
  if (!drag || event.pointerId !== drag.pointerId) return
  const preview = dragPreview.value
  finishIconDrag()
  dragPreview.value = undefined
  if (!drag.moved || !preview) return
  suppressActivationAfterDrag.add(drag.key)
  const destination = desktopIconPixelsToPosition(preview, iconBounds.value)
  const placements = dropDesktopIcon(
    renderedIconLayout.value.placements,
    drag.key,
    destination,
    iconBounds.value,
  )
  const next = placementsToPositions(placements)
  iconAnnouncement.value = i18n.t('desktop.iconMoved', {
    name: iconLabel(drag.key),
  })
  void persistPositions(next).catch(() => undefined)
}

function cancelIconDrag(): void {
  const drag = finishIconDrag()
  dragPreview.value = undefined
  if (drag?.moved) suppressActivationAfterDrag.add(drag.key)
}

function onIconDragCancel(event: PointerEvent): void {
  if (iconDrag && event.pointerId === iconDrag.pointerId) cancelIconDrag()
}

function onIconDragLostPointerCapture(event: PointerEvent): void {
  if (iconDrag && event.pointerId === iconDrag.pointerId) cancelIconDrag()
}

function suppressDraggedActivation(event: Event, key: string): void {
  if (!suppressActivationAfterDrag.has(key)) return
  event.preventDefault()
  event.stopImmediatePropagation()
}

function clearDraggedActivationSuppression(key: string): void {
  suppressActivationAfterDrag.delete(key)
}

function iconLabel(key: string): string {
  if (key.startsWith('nav:')) {
    const app = desktopApps.find((candidate) => `nav:${candidate.path}` === key)
    return app ? i18n.t(app.labelKey) : key
  }
  return [...visibleDynamicEntries.value, ...shortcutEntries.value]
    .find((entry) => entry.key === key)?.name || key
}

function nudgeIcon(key: string, deltaX: number, deltaY: number): void {
  if (compactIconLayout.value) return
  if (!renderedPositionByKey.value.has(key)) {
    const message = i18n.t('desktop.iconLayoutLimitMessage', { count: MAX_DESKTOP_ICON_POSITIONS })
    iconAnnouncement.value = message
    toast.danger(i18n.t('desktop.iconLayoutLimitTitle'), message)
    return
  }
  const direction = deltaX < 0 ? 'left' : deltaX > 0 ? 'right' : deltaY < 0 ? 'up' : 'down'
  const placements = moveDesktopIconByKeyboard(
    renderedIconLayout.value.placements,
    key,
    direction,
    iconBounds.value,
  )
  selectedIcon.value = key
  iconAnnouncement.value = i18n.t('desktop.iconMoved', { name: iconLabel(key) })
  void persistPositions(placementsToPositions(placements)).catch(() => undefined)
}

async function autoArrangeIcons(): Promise<void> {
  if (compactIconLayout.value) return
  closeContextMenu()
  const arranged = autoArrangeDesktopIcons(allIconKeys.value, iconBounds.value)
  if (arranged.overflowKeys.length) {
    const message = i18n.t('desktop.iconLayoutLimitMessage', { count: MAX_DESKTOP_ICON_POSITIONS })
    iconAnnouncement.value = message
    toast.danger(i18n.t('desktop.iconLayoutLimitTitle'), message)
    return
  }
  try {
    await persistPositions(placementsToPositions(arranged.placements))
    iconAnnouncement.value = i18n.t('desktop.iconsArranged')
    toast.success(i18n.t('desktop.iconsArranged'))
  } catch {
    // persistPositions already surfaced a specific failure.
  }
}

function onGlobalPointerDown(event: PointerEvent): void {
  if (iconDrag && event.pointerId !== iconDrag.pointerId) cancelIconDrag()
  if (!contextMenu.value.open) return
  // A right-button press may be followed by one or more contextmenu events
  // while the button is held. Keep the existing menu mounted and let the
  // contextmenu handler reposition it, instead of starting close/open
  // transitions in the same pointer cycle.
  if (event.button === 2) return
  const target = event.target
  if (target instanceof Node && contextMenuElement.value?.contains(target)) return
  closeContextMenu(false)
}

function onGlobalKeyDown(event: KeyboardEvent): void {
  if (event.key !== 'Escape') return
  if (iconDrag) cancelIconDrag()
  else if (contextMenu.value.open) closeContextMenu()
  else selectedIcon.value = ''
}

function onContextMenuKeyDown(event: KeyboardEvent): void {
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const items = Array.from(contextMenuElement.value?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') || [])
  if (!items.length) return
  event.preventDefault()
  const current = items.indexOf(document.activeElement as HTMLButtonElement)
  const index = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? items.length - 1
      : event.key === 'ArrowDown'
        ? (current + 1 + items.length) % items.length
        : (current - 1 + items.length) % items.length
  items[index]?.focus({ preventScroll: true })
}

function onDesktopPointerDown(event: PointerEvent): void {
  const target = event.target as HTMLElement
  if (target.closest('.desktop-window, .desktop__widgets, .desktop__taskbar, .desktop__icon')) return
  selectedIcon.value = ''
  ;(event.currentTarget as HTMLElement).focus({ preventScroll: true })
}

function onContextMenuAction(
  action: 'refresh' | 'theme' | 'classic' | 'about' | 'processes' | 'add-shortcut'
    | 'auto-arrange' | 'manage-icons' | 'arrange-mode',
): void {
  closeContextMenu()
  switch (action) {
    case 'refresh':
      void refreshDesktop()
      break
    case 'theme':
      theme.setTheme(theme.resolved.value === 'dark' ? 'light' : 'dark')
      break
    case 'classic':
      void enterClassicSafely()
      break
    case 'about':
      toast.success(i18n.t('desktop.aboutTitle'), i18n.t('desktop.aboutMessage'))
      break
    case 'add-shortcut':
      openShortcutDialog()
      break
    case 'auto-arrange':
      void autoArrangeIcons()
      break
    case 'manage-icons':
      iconManagerOpen.value = true
      break
    case 'arrange-mode':
      arrangingIcons.value = !arrangingIcons.value
      toast.show(i18n.t(arrangingIcons.value
        ? 'desktop.arrangeModeEnabled'
        : 'desktop.arrangeModeDisabled'))
      break
    case 'processes': {
      const windowId = desktop.openWindow('/processes', 'route.processes', false)
      if (windowId === 0) {
        toast.show(i18n.t('desktop.windowLimitTitle'), {
          message: i18n.t('desktop.windowLimitMessage'),
        })
      }
      break
    }
  }
}

function onNavMenuOpen(): void {
  const path = menuNavPath.value
  closeContextMenu()
  if (path) openNavIcon(path)
}

async function enterClassicSafely(): Promise<void> {
  if (await desktopCloseGuardCoordinator.checkAll()) desktop.enterClassic()
}

function onEntryMenuOpen(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry) openEntry(entry)
}

function onEntryMenuDetails(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (!entry) return
  if (entry.kind === 'app') {
    openAppMarketEntry(entry)
    return
  }
  detailEntry.value = entry
}

function onDetailEntryOpen(): void {
  const entry = detailEntry.value
  detailEntry.value = undefined
  if (entry) openEntry(entry)
}

function onEntryMenuRename(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry?.kind !== 'site') return
  renameEntry.value = entry
  renameValue.value = entry.name
}

function closeRename(): void {
  renameEntry.value = undefined
  renameValue.value = ''
}

async function saveRename(): Promise<void> {
  const entry = renameEntry.value
  const name = renameValue.value.trim().slice(0, MAX_SITE_NAME_LENGTH)
  if (entry?.kind !== 'site' || !name) return
  const defaultName = defaultSiteName(entry)
  try {
    await desktopIcons.mutate((draft) => {
      if (name === defaultName) delete draft.labels[entry.key]
      else draft.labels[entry.key] = name
    })
    const next = { ...siteNames.value }
    if (name === defaultName) delete next[entry.id]
    else next[entry.id] = name
    siteNames.value = next
    window.localStorage.removeItem(SITE_RENAMES_KEY)
    entries.value = applySiteNames(entries.value)
    closeRename()
  } catch (error) {
    toast.danger(i18n.t('desktop.workspaceSaveErrorTitle'), workspaceErrorMessage(error))
  }
}

async function resetRename(): Promise<void> {
  const entry = renameEntry.value
  if (entry?.kind !== 'site') return
  try {
    await desktopIcons.mutate((draft) => {
      delete draft.labels[entry.key]
    })
    const next = { ...siteNames.value }
    delete next[entry.id]
    siteNames.value = next
    window.localStorage.removeItem(SITE_RENAMES_KEY)
    entries.value = applySiteNames(entries.value)
    closeRename()
  } catch (error) {
    toast.danger(i18n.t('desktop.workspaceSaveErrorTitle'), workspaceErrorMessage(error))
  }
}

function requestRemoveEntry(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry?.kind === 'app' || entry?.kind === 'site') removingEntry.value = entry
}

async function confirmRemoveEntry(): Promise<void> {
  const entry = removingEntry.value
  if (!entry || (entry.kind !== 'app' && entry.kind !== 'site')) return
  try {
    await desktopIcons.mutate((draft) => {
      if (!draft.hiddenEntryKeys.includes(entry.key)) draft.hiddenEntryKeys.push(entry.key)
    })
    selectedIcon.value = ''
    removingEntry.value = undefined
    toast.success(i18n.t('desktop.removedFromDesktopTitle'), entry.name)
  } catch (error) {
    toast.danger(i18n.t('desktop.workspaceSaveErrorTitle'), workspaceErrorMessage(error))
  }
}

async function restoreEntry(entry: DesktopEntry): Promise<void> {
  try {
    await desktopIcons.mutate((draft) => {
      draft.hiddenEntryKeys = draft.hiddenEntryKeys.filter((key) => key !== entry.key)
    })
    toast.success(i18n.t('desktop.restoredToDesktopTitle'), entry.name)
  } catch (error) {
    toast.danger(i18n.t('desktop.workspaceSaveErrorTitle'), workspaceErrorMessage(error))
  }
}

function openShortcutDialog(shortcut?: DesktopShortcut): void {
  pendingShortcutID = ''
  editingShortcut.value = shortcut
  shortcutError.value = ''
  shortcutDialogOpen.value = true
  iconManagerOpen.value = false
  closeContextMenu(false)
}

function closeShortcutDialog(): void {
  if (shortcutSaving.value) return
  shortcutDialogOpen.value = false
  editingShortcut.value = undefined
  shortcutError.value = ''
  pendingShortcutID = ''
}

async function saveShortcut(
  draft: DesktopShortcutDraft,
  icon: File | undefined,
  removeIcon: boolean,
): Promise<void> {
  shortcutSaving.value = true
  shortcutError.value = ''
  const id = draft.id || pendingShortcutID || desktopIcons.generateShortcutID()
  try {
    await desktopIcons.mutate((workspaceDraft) => {
      const next = { id, name: draft.name, description: draft.description, url: draft.url }
      const index = workspaceDraft.shortcuts.findIndex((shortcut) => shortcut.id === id)
      if (index >= 0) workspaceDraft.shortcuts.splice(index, 1, next)
      else workspaceDraft.shortcuts.push(next)
    })
    if (!draft.id) pendingShortcutID = id
    if (removeIcon) await api.desktop.removeShortcutIcon(id)
    if (icon) await api.desktop.uploadShortcutIcon(id, icon)
    if (removeIcon || icon) await desktopIcons.load()
    shortcutDialogOpen.value = false
    editingShortcut.value = undefined
    pendingShortcutID = ''
    toast.success(i18n.t(draft.id ? 'desktop.shortcutUpdated' : 'desktop.shortcutCreated'), draft.name)
  } catch (error) {
    shortcutError.value = workspaceErrorMessage(error)
  } finally {
    shortcutSaving.value = false
  }
}

function requestDeleteShortcut(shortcut?: DesktopShortcut): void {
  const target = shortcut || menuEntry.value?.shortcut
  closeContextMenu()
  if (target) deletingShortcut.value = target
}

async function confirmDeleteShortcut(): Promise<void> {
  const shortcut = deletingShortcut.value
  if (!shortcut) return
  try {
    await desktopIcons.mutate((draft) => {
      draft.shortcuts = draft.shortcuts.filter((item) => item.id !== shortcut.id)
      delete draft.positions[`shortcut:${shortcut.id}`]
    })
    deletingShortcut.value = undefined
    selectedIcon.value = ''
    toast.success(i18n.t('desktop.shortcutDeleted'), shortcut.name)
  } catch (error) {
    toast.danger(i18n.t('desktop.workspaceSaveErrorTitle'), workspaceErrorMessage(error))
  }
}

function editMenuShortcut(): void {
  const shortcut = menuEntry.value?.shortcut
  closeContextMenu()
  if (shortcut) openShortcutDialog(shortcut)
}

function onTaskbarClick(windowId: number): void {
  const target = desktop.windows.value.find((windowState) => windowState.id === windowId)
  if (!target) return
  if (target.minimized || desktop.focusedId.value !== windowId) {
    desktop.restoreWindow(windowId)
  } else {
    desktop.minimizeWindow(windowId)
  }
}

async function closeTaskbarWindow(): Promise<void> {
  const windowId = menuWindowId.value
  if (windowId === undefined) return
  const windowHandle = desktopWindowRefs.get(windowId)
  closeContextMenu()
  await windowHandle?.requestClose()
}

async function loadEntries(force = false): Promise<void> {
  entriesAbort?.abort()
  entriesAbort = new AbortController()
  const sequence = ++entriesSequence
  entriesLoading.value = true
  try {
    const nextEntries = await loadDesktopEntries(entriesAbort.signal, undefined, force)
    if (sequence === entriesSequence) {
      entries.value = applySiteNames(nextEntries)
      void loadSiteAppearanceNames(nextEntries, entriesAbort.signal, sequence)
    }
  } catch {
    if (sequence === entriesSequence) entries.value = undefined
  } finally {
    if (sequence === entriesSequence) entriesLoading.value = false
  }
}

async function loadWorkspace(): Promise<void> {
  workspaceAbort?.abort()
  workspaceAbort = new AbortController()
  const legacyNames = readSiteNames()
  try {
    const value = await desktopIcons.load(workspaceAbort.signal)
    const persistedNames = Object.fromEntries(
      Object.entries(value.labels)
        .filter(([key]) => key.startsWith('site:'))
        .map(([key, name]) => [key.slice('site:'.length), name]),
    )
    siteNames.value = { ...legacyNames, ...persistedNames }
    entries.value = applySiteNames(entries.value)
    if (!value.available) {
      toast.danger(
        i18n.t('desktop.workspaceUnavailableTitle'),
        i18n.t('desktop.workspaceUnavailableMessage'),
      )
      return
    }

    const legacyEntries = Object.entries(legacyNames)
      .filter(([id]) => !Object.hasOwn(value.labels, `site:${id}`))
    if (legacyEntries.length) {
      await desktopIcons.mutate((draft) => {
        for (const [id, name] of legacyEntries) draft.labels[`site:${id}`] = name
      })
      window.localStorage.removeItem(SITE_RENAMES_KEY)
    }
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') return
    toast.danger(i18n.t('desktop.workspaceLoadErrorTitle'), workspaceErrorMessage(error))
  }
}

async function refreshDesktop(): Promise<void> {
  await Promise.allSettled([loadEntries(true), loadWorkspace()])
}

function appearanceEntries(value: DesktopEntries): DesktopEntry[] {
  const unique = new Map<string, DesktopEntry>()
  for (const entry of [...value.sites, ...value.visible]) {
    if (entry.kind === 'site') unique.set(entry.id, entry)
  }
  return [...unique.values()]
}

async function loadSiteAppearanceNames(
  value: DesktopEntries,
  signal: AbortSignal,
  sequence: number,
): Promise<void> {
  const queue = appearanceEntries(value)
  const names: Record<string, string> = {}
  let cursor = 0

  async function worker(): Promise<void> {
    while (!signal.aborted && cursor < queue.length) {
      const entry = queue[cursor]
      cursor += 1
      if (!entry) return
      try {
        const appearance = await api.sites.appearance(entry.id, signal)
        const name = appearance.name?.trim().slice(0, MAX_SITE_NAME_LENGTH)
        if (name) names[entry.id] = name
      } catch {
        // Appearance is optional. The primary domain remains the safe fallback.
      }
    }
  }

  await Promise.all(Array.from({ length: Math.min(4, queue.length) }, () => worker()))
  if (signal.aborted || sequence !== entriesSequence) return
  siteAppearanceNames.value = names
  entries.value = applySiteNames(entries.value)
}

onMounted(() => {
  document.documentElement.classList.add('desktop-mode-open')
  document.body.classList.add('desktop-mode-open')
  window.addEventListener('pointerdown', onGlobalPointerDown)
  window.addEventListener('keydown', onGlobalKeyDown)
  window.addEventListener('resize', onViewportResize)
  void loadEntries()
  void loadWorkspace()
  void nextTick(() => {
    measureIconWorkArea()
    if (typeof ResizeObserver !== 'undefined' && iconsElement.value) {
      iconsResizeObserver = new ResizeObserver(measureIconWorkArea)
      iconsResizeObserver.observe(iconsElement.value)
    }
  })
})

onBeforeUnmount(() => {
  entriesSequence += 1
  document.documentElement.classList.remove('desktop-mode-open')
  document.body.classList.remove('desktop-mode-open')
  window.removeEventListener('pointerdown', onGlobalPointerDown)
  window.removeEventListener('keydown', onGlobalKeyDown)
  window.removeEventListener('resize', onViewportResize)
  entriesAbort?.abort()
  workspaceAbort?.abort()
  iconsResizeObserver?.disconnect()
  cancelIconDrag()
  suppressActivationAfterDrag.clear()
  if (bounceTimer !== undefined) window.clearTimeout(bounceTimer)
  if (resizeFrame !== undefined) window.cancelAnimationFrame(resizeFrame)
  if (resizePersistTimer !== undefined) {
    window.clearTimeout(resizePersistTimer)
    desktop.resizeForViewport({ width: window.innerWidth, height: window.innerHeight })
  }
})

function onViewportResize(): void {
  if (resizeFrame !== undefined) window.cancelAnimationFrame(resizeFrame)
  resizeFrame = window.requestAnimationFrame(() => {
    resizeFrame = undefined
    measureIconWorkArea()
    desktop.resizeForViewport({ width: window.innerWidth, height: window.innerHeight }, false)
  })
  if (resizePersistTimer !== undefined) window.clearTimeout(resizePersistTimer)
  resizePersistTimer = window.setTimeout(() => {
    resizePersistTimer = undefined
    desktop.resizeForViewport({ width: window.innerWidth, height: window.innerHeight })
  }, 180)
}
</script>

<template>
  <div class="desktop" tabindex="-1" @pointerdown="onDesktopPointerDown" @contextmenu="onContextMenu">
    <div class="desktop__wallpaper" aria-hidden="true">
      <div class="desktop__aurora desktop__aurora--one" />
      <div class="desktop__aurora desktop__aurora--two" />
      <div class="desktop__aurora desktop__aurora--three" />
    </div>

    <aside class="desktop__widgets" :aria-label="i18n.t('desktop.toolbarLabel')" @contextmenu.stop>
      <DesktopClock
        :network="entries?.publicNetwork"
        :system-timezone="systemResources?.timezone"
      />
      <DesktopMonitor @snapshot="systemResources = $event" />
    </aside>

    <nav
      ref="iconsElement"
      class="desktop__icons"
      :class="{ 'desktop__icons--arranging': arrangingIcons }"
      :aria-label="i18n.t('desktop.gridLabel')"
      :aria-busy="entriesLoading"
    >
      <div
        class="desktop__icons-scroll-space"
        :style="{ height: `${iconScrollHeight}px` }"
        aria-hidden="true"
      />
      <p
        v-if="renderedIconLayout.overflowKeys.length"
        class="desktop__icons-overflow-note"
        :style="{ top: `${renderedIconLayout.contentHeight + 8}px` }"
        role="status"
      >
        {{ i18n.t('desktop.iconOverflowNotice', {
          count: renderedIconLayout.overflowKeys.length,
          limit: MAX_DESKTOP_ICON_POSITIONS,
        }) }}
      </p>
      <!-- Static navigation apps -->
      <div
        v-for="(app, index) in desktopApps"
        :key="app.path"
        class="desktop__icon-slot"
        :class="{ 'desktop__icon-slot--dragging': draggingIcon === `nav:${app.path}` }"
        :style="iconSlotStyle(`nav:${app.path}`)"
        :data-icon-key="`nav:${app.path}`"
        @pointerdown="beginIconDrag($event, `nav:${app.path}`)"
        @keydown.capture="clearDraggedActivationSuppression(`nav:${app.path}`)"
        @click.capture="suppressDraggedActivation($event, `nav:${app.path}`)"
        @dblclick.capture="suppressDraggedActivation($event, `nav:${app.path}`)"
      >
        <DesktopEntryIcon
          :label="i18n.t(app.labelKey)"
          :nav-icon="app.icon"
          :gradient="gradientFor(app.path)"
          :active="bouncingIcon === app.path"
          :selected="selectedIcon === `nav:${app.path}`"
          :order="index"
          :arranging="arrangingIcons"
          :dragging="draggingIcon === `nav:${app.path}`"
          @select="selectNavIcon(app.path)"
          @open="openNavIcon(app.path)"
          @context="(event) => onNavContext(event, app.path)"
          @warm="warmNavIcon(app.path)"
          @nudge="(x, y) => nudgeIcon(`nav:${app.path}`, x, y)"
        />
      </div>

      <!-- Dynamic entries: installed apps and sites -->
      <template v-if="entries">
        <div
          v-for="(entry, index) in visibleDynamicEntries"
          :key="entry.key"
          class="desktop__icon-slot"
          :class="{ 'desktop__icon-slot--dragging': draggingIcon === entry.key }"
          :style="iconSlotStyle(entry.key)"
          :data-icon-key="entry.key"
          @pointerdown="beginIconDrag($event, entry.key)"
          @keydown.capture="clearDraggedActivationSuppression(entry.key)"
          @click.capture="suppressDraggedActivation($event, entry.key)"
          @dblclick.capture="suppressDraggedActivation($event, entry.key)"
        >
          <DesktopEntryIcon
            :label="entry.name"
            :entry="entry"
            :gradient="entryGradient(entry)"
            :selected="selectedIcon === entry.key"
            :order="desktopApps.length + index"
            :arranging="arrangingIcons"
            :dragging="draggingIcon === entry.key"
            @select="selectEntry(entry)"
            @open="(event) => onEntryOpen(event, entry)"
            @context="(event) => onEntryContext(event, entry)"
            @nudge="(x, y) => nudgeIcon(entry.key, x, y)"
          />
        </div>
      </template>
      <div
        v-for="(entry, index) in shortcutEntries"
        :key="entry.key"
        class="desktop__icon-slot"
        :class="{ 'desktop__icon-slot--dragging': draggingIcon === entry.key }"
        :style="iconSlotStyle(entry.key)"
        :data-icon-key="entry.key"
        @pointerdown="beginIconDrag($event, entry.key)"
        @keydown.capture="clearDraggedActivationSuppression(entry.key)"
        @click.capture="suppressDraggedActivation($event, entry.key)"
        @dblclick.capture="suppressDraggedActivation($event, entry.key)"
      >
        <DesktopEntryIcon
          :label="entry.name"
          :entry="entry"
          :gradient="entryGradient(entry)"
          :selected="selectedIcon === entry.key"
          :order="desktopApps.length + visibleDynamicEntries.length + index"
          :arranging="arrangingIcons"
          :dragging="draggingIcon === entry.key"
          @select="selectEntry(entry)"
          @open="(event) => onEntryOpen(event, entry)"
          @context="(event) => onEntryContext(event, entry)"
          @nudge="(x, y) => nudgeIcon(entry.key, x, y)"
        />
      </div>
      <span v-if="entriesLoading" class="desktop__sr-only" aria-live="polite">
        {{ i18n.t('desktop.entriesLoading') }}
      </span>
      <span class="desktop__sr-only" aria-live="polite">{{ iconAnnouncement }}</span>
    </nav>

    <DesktopWindow
      v-for="windowState in openWindows"
      :key="windowState.id"
      :ref="(instance) => setDesktopWindowRef(windowState.id, instance)"
      :window-state="windowState"
      :icon="windowIcon(windowState.path)"
      :icon-url="windowIconURL(windowState.path)"
      :title="windowTitle(windowState.titleKey, windowState.path)"
    />

    <Transition name="desktop-menu">
      <div
        v-if="contextMenu.open"
        ref="contextMenuElement"
        class="desktop__context-menu"
        :class="{ 'desktop__context-menu--entry': menuEntry || menuNavPath }"
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        role="menu"
        @contextmenu.prevent.stop
        @pointerdown.stop
        @keydown="onContextMenuKeyDown"
      >
        <template v-if="menuEntry">
          <button type="button" role="menuitem" @click="onEntryMenuOpen">
            <SquareTerminal v-if="menuEntry.launch === 'script'" :size="15" aria-hidden="true" />
            <AppWindow v-else-if="menuEntry.url" :size="15" aria-hidden="true" />
            <ExternalLink v-else :size="15" aria-hidden="true" />
            {{ menuEntry.launch === 'script'
              ? i18n.t('desktop.entryScriptManage')
              : menuEntry.url
                ? i18n.t('desktop.systemBrowserOpen')
                : i18n.t('desktop.entryOpen') }}
          </button>
          <button type="button" role="menuitem" @click="onEntryMenuDetails">
            <Info :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.entryDetails') }}
          </button>
          <button
            v-if="menuEntry.kind === 'site'"
            type="button"
            role="menuitem"
            @click="onEntryMenuRename"
          >
            <Pencil :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.entryRename') }}
          </button>
          <button
            v-if="menuEntry.kind === 'shortcut'"
            type="button"
            role="menuitem"
            @click="editMenuShortcut"
          >
            <Pencil :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.shortcutEdit') }}
          </button>
          <div class="desktop__context-separator" role="separator" />
          <button
            v-if="menuEntry.kind === 'app' || menuEntry.kind === 'site'"
            type="button"
            role="menuitem"
            class="desktop__context-danger"
            :disabled="!workspace.available || desktopIcons.saving.value"
            @click="requestRemoveEntry"
          >
            <EyeOff :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.removeFromDesktop') }}
          </button>
          <button
            v-else
            type="button"
            role="menuitem"
            class="desktop__context-danger"
            :disabled="!workspace.available || desktopIcons.saving.value"
            @click="requestDeleteShortcut()"
          >
            <Trash2 :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.shortcutDelete') }}
          </button>
        </template>
        <template v-else-if="menuNavPath">
          <button type="button" role="menuitem" @click="onNavMenuOpen">
            <AppWindow :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.entryOpen') }}
          </button>
        </template>
        <template v-else-if="contextMenuTarget === 'taskbar'">
          <button
            type="button"
            role="menuitem"
            data-context-action="processes"
            @click="onContextMenuAction('processes')"
          >
            <ListTree :size="15" aria-hidden="true" />
            {{ i18n.t('route.processes') }}
          </button>
        </template>
        <template v-else-if="contextMenuTarget === 'taskbar-window'">
          <button
            type="button"
            role="menuitem"
            data-context-action="close-window"
            @click="closeTaskbarWindow"
          >
            <X :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.closeWindow') }}
          </button>
        </template>
        <template v-else>
          <button
            type="button"
            role="menuitem"
            :disabled="!workspace.available || desktopIcons.saving.value"
            @click="onContextMenuAction('add-shortcut')"
          >
            <Plus :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.shortcutAdd') }}
          </button>
          <button
            type="button"
            role="menuitem"
            :disabled="!workspace.available"
            @click="onContextMenuAction('manage-icons')"
          >
            <MonitorCog :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.iconManagerTitle') }}
          </button>
          <button
            type="button"
            role="menuitem"
            :disabled="compactIconLayout || !workspace.available || desktopIcons.saving.value"
            @click="onContextMenuAction('auto-arrange')"
          >
            <Grid2X2 :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.autoArrange') }}
          </button>
          <button
            type="button"
            role="menuitem"
            :disabled="compactIconLayout"
            @click="onContextMenuAction('arrange-mode')"
          >
            <Grid2X2 :size="15" aria-hidden="true" />
            {{ arrangingIcons
              ? i18n.t('desktop.exitArrangeMode')
              : i18n.t('desktop.enterArrangeMode') }}
          </button>
          <div class="desktop__context-separator" role="separator" />
          <button type="button" role="menuitem" @click="onContextMenuAction('refresh')">
            <RefreshCw :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.menuRefresh') }}
          </button>
          <button type="button" role="menuitem" @click="onContextMenuAction('theme')">
            <Sun v-if="theme.resolved.value === 'dark'" :size="15" aria-hidden="true" />
            <Moon v-else :size="15" aria-hidden="true" />
            {{ theme.resolved.value === 'dark' ? i18n.t('desktop.menuLight') : i18n.t('desktop.menuDark') }}
          </button>
          <button type="button" role="menuitem" @click="onContextMenuAction('about')">
            <Info :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.menuAbout') }}
          </button>
          <div class="desktop__context-separator" role="separator" />
          <button type="button" role="menuitem" @click="onContextMenuAction('classic')">
            <ArrowLeft :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.switchClassic') }}
          </button>
        </template>
      </div>
    </Transition>

    <footer
      class="desktop__taskbar"
      role="toolbar"
      :aria-label="i18n.t('desktop.taskbarLabel')"
      @contextmenu.prevent.stop="onTaskbarContext"
    >
      <div class="desktop__taskbar-brand" aria-label="KPanel">
        <LogoMark compact />
        <span>KPanel</span>
        <div v-if="props.agent" class="desktop__taskbar-agent">
          <span class="desktop__taskbar-agent-status" :class="`desktop__taskbar-agent-status--${agentStatus.state}`">
            <i aria-hidden="true" />
            <span>{{ agentStatus.label }}</span>
          </span>
          <button
            v-if="props.kpanelUpdateAvailable"
            class="desktop__taskbar-agent-update"
            type="button"
            :aria-label="props.kpanelUpdateDescription"
            :title="props.kpanelUpdateDescription"
            @click="openKPanelUpdate"
          >
            <CircleArrowUp :size="13" aria-hidden="true" />
            <span>{{ i18n.t('nav.updateAvailable') }}</span>
          </button>
          <small v-else-if="props.agent.version">v{{ props.agent.version }}</small>
        </div>
      </div>
      <div class="desktop__taskbar-apps">
        <button
          v-for="windowState in openWindows"
          :key="windowState.id"
          class="desktop__taskbar-item"
          :class="{
            'desktop__taskbar-item--active': windowState.id === focusedWindow?.id,
            'desktop__taskbar-item--minimized': windowState.minimized,
          }"
          type="button"
          :data-window-id="windowState.id"
          :aria-label="windowTitle(windowState.titleKey, windowState.path)"
          :title="windowTitle(windowState.titleKey, windowState.path)"
          :aria-pressed="windowState.id === focusedWindow?.id"
          @click="onTaskbarClick(windowState.id)"
          @contextmenu.stop="onTaskbarItemContext($event, windowState.id)"
        >
          <span
            class="desktop__taskbar-glyph"
            :style="{ background: gradientFor(windowState.path) }"
          >
            <component
              :is="findDesktopApp(windowState.path)?.icon || AppWindow"
              :size="19"
              :stroke-width="1.9"
              aria-hidden="true"
            />
          </span>
          <span class="desktop__taskbar-label">{{ windowTitle(windowState.titleKey, windowState.path) }}</span>
          <i aria-hidden="true" />
        </button>
      </div>
      <div class="desktop__system-tray">
        <button
          class="desktop__tray-button"
          type="button"
          :title="theme.resolved.value === 'dark' ? i18n.t('desktop.menuLight') : i18n.t('desktop.menuDark')"
          :aria-label="theme.resolved.value === 'dark' ? i18n.t('desktop.menuLight') : i18n.t('desktop.menuDark')"
          @click="theme.setTheme(theme.resolved.value === 'dark' ? 'light' : 'dark')"
        >
          <Sun v-if="theme.resolved.value === 'dark'" :size="16" aria-hidden="true" />
          <Moon v-else :size="16" aria-hidden="true" />
        </button>
        <button
          class="desktop__classic-button"
          type="button"
          :title="i18n.t('desktop.switchClassic')"
          :aria-label="i18n.t('desktop.switchClassic')"
          @click="enterClassicSafely"
        >
          <ArrowLeft :size="15" aria-hidden="true" />
          <span>{{ i18n.t('desktop.switchClassic') }}</span>
        </button>
      </div>
    </footer>

    <ModalDialog
      :open="Boolean(externalOpenEntry)"
      :title="i18n.t('desktop.externalOpenConfirmTitle')"
      size="compact"
      @close="closeExternalOpen"
    >
      <div v-if="externalOpenEntry" class="desktop__external-confirm">
        <div class="desktop__external-confirm-entry">
          <span
            class="desktop__external-confirm-icon"
            :style="{ background: entryGradient(externalOpenEntry) }"
            aria-hidden="true"
          >
            <img
              v-if="externalOpenEntry.iconURL && !externalOpenImageFailed"
              class="desktop__external-confirm-icon-image"
              :src="externalOpenEntry.iconURL"
              alt=""
              decoding="async"
              referrerpolicy="no-referrer"
              width="64"
              height="64"
              @error="externalOpenImageFailed = true"
            />
            <span v-else-if="externalOpenEntry.kind === 'site'" class="desktop__site-fallback">
              <span class="desktop__site-fallback-letter">{{ externalOpenMonogram }}</span>
              <span class="desktop__site-fallback-badge">
                <Globe2 :size="11" :stroke-width="2.2" />
              </span>
            </span>
            <span v-else class="desktop__icon-monogram">{{ externalOpenMonogram }}</span>
          </span>
          <div class="desktop__external-confirm-identity">
            <strong>{{ externalOpenEntry.name }}</strong>
            <code>{{ externalOpenEntry.url }}</code>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="button button--ghost" type="button" @click="closeExternalOpen">
          {{ i18n.t('common.cancel') }}
        </button>
        <button class="button button--primary" type="button" @click="confirmExternalOpen">
          <ExternalLink :size="15" aria-hidden="true" />
          {{ i18n.t('desktop.systemBrowserOpen') }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(detailEntry)"
      :title="detailEntry?.name || ''"
      size="small"
      @close="detailEntry = undefined"
    >
      <dl v-if="detailEntry" class="desktop__detail">
        <template v-if="detailEntry.kind === 'app'">
          <dt>{{ i18n.t('desktop.detailType') }}</dt>
          <dd>{{ i18n.t('desktop.detailApp') }}</dd>
          <dt>{{ i18n.t('desktop.detailStatus') }}</dt>
          <dd>{{ detailEntry.app?.runtime.state || i18n.t('desktop.detailUnknown') }}</dd>
          <dt>{{ i18n.t('desktop.detailURL') }}</dt>
          <dd class="desktop__detail-url">{{ detailEntry.url }}</dd>
        </template>
        <template v-else-if="detailEntry.kind === 'site'">
          <dt>{{ i18n.t('desktop.detailType') }}</dt>
          <dd>{{ i18n.t('desktop.detailSite') }}</dd>
          <dt>{{ i18n.t('desktop.detailDomain') }}</dt>
          <dd>{{ detailEntry.site?.primaryDomain }}</dd>
          <dt>{{ i18n.t('desktop.detailType2') }}</dt>
          <dd>{{ detailEntry.site?.type }}</dd>
          <dt>{{ i18n.t('desktop.detailURL') }}</dt>
          <dd class="desktop__detail-url">{{ detailEntry.url }}</dd>
        </template>
        <template v-else>
          <dt>{{ i18n.t('desktop.detailType') }}</dt>
          <dd>{{ i18n.t('desktop.detailShortcut') }}</dd>
          <dt>{{ i18n.t('desktop.detailDescription') }}</dt>
          <dd>{{ detailEntry.description || i18n.t('desktop.detailNoDescription') }}</dd>
          <dt>{{ i18n.t('desktop.detailURL') }}</dt>
          <dd class="desktop__detail-url">{{ detailEntry.url }}</dd>
        </template>
      </dl>
      <template #footer>
        <button class="button button--primary" type="button" @click="onDetailEntryOpen">
          <ExternalLink :size="15" aria-hidden="true" />
          {{ i18n.t('desktop.entryOpen') }}
        </button>
        <button class="button button--ghost" type="button" @click="detailEntry = undefined">
          {{ i18n.t('common.closeDialog') }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(renameEntry)"
      :title="i18n.t('desktop.renameTitle')"
      size="small"
      @close="closeRename"
    >
      <form class="desktop__rename-form" @submit.prevent="saveRename">
        <label>
          <span>{{ i18n.t('desktop.renameLabel') }}</span>
          <input
            v-model="renameValue"
            :maxlength="MAX_SITE_NAME_LENGTH"
            :placeholder="renameEntry ? defaultSiteName(renameEntry) : ''"
            autocomplete="off"
          />
        </label>
        <small>{{ renameEntry?.site?.primaryDomain }}</small>
      </form>
      <template #footer>
        <button
          v-if="renameEntry && siteNames[renameEntry.id]"
          class="button button--ghost"
          type="button"
          @click="resetRename"
        >
          {{ i18n.t('desktop.renameReset') }}
        </button>
        <button class="button button--ghost" type="button" @click="closeRename">
          {{ i18n.t('common.cancel') }}
        </button>
        <button class="button button--primary" type="button" :disabled="!renameValue.trim()" @click="saveRename">
          {{ i18n.t('desktop.renameSave') }}
        </button>
      </template>
    </ModalDialog>

    <DesktopIconManagerDialog
      :open="iconManagerOpen"
      :hidden-entries="hiddenEntries"
      :shortcuts="shortcuts"
      :busy="desktopIcons.saving.value"
      @close="iconManagerOpen = false"
      @add="openShortcutDialog()"
      @edit="openShortcutDialog"
      @remove="requestDeleteShortcut"
      @restore="restoreEntry"
    />

    <DesktopShortcutDialog
      :open="shortcutDialogOpen"
      :shortcut="editingShortcut"
      :saving="shortcutSaving"
      :error-message="shortcutError"
      @close="closeShortcutDialog"
      @save="saveShortcut"
    />

    <ModalDialog
      :open="Boolean(removingEntry)"
      :title="i18n.t('desktop.removeFromDesktopTitle')"
      size="compact"
      @close="removingEntry = undefined"
    >
      <div v-if="removingEntry" class="desktop__confirm-copy">
        <strong>{{ removingEntry.name }}</strong>
        <p>{{ removingEntry.kind === 'app'
          ? i18n.t('desktop.removeAppFromDesktopMessage')
          : i18n.t('desktop.removeSiteFromDesktopMessage') }}</p>
      </div>
      <template #footer>
        <button class="button button--ghost" type="button" @click="removingEntry = undefined">
          {{ i18n.t('common.cancel') }}
        </button>
        <button
          class="button button--primary"
          type="button"
          :disabled="desktopIcons.saving.value"
          @click="confirmRemoveEntry"
        >
          {{ i18n.t('desktop.removeFromDesktop') }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(deletingShortcut)"
      :title="i18n.t('desktop.shortcutDeleteTitle')"
      size="compact"
      @close="deletingShortcut = undefined"
    >
      <div v-if="deletingShortcut" class="desktop__confirm-copy">
        <strong>{{ deletingShortcut.name }}</strong>
        <p>{{ i18n.t('desktop.shortcutDeleteMessage') }}</p>
      </div>
      <template #footer>
        <button class="button button--ghost" type="button" @click="deletingShortcut = undefined">
          {{ i18n.t('common.cancel') }}
        </button>
        <button
          class="button button--danger"
          type="button"
          :disabled="desktopIcons.saving.value"
          @click="confirmDeleteShortcut"
        >
          {{ i18n.t('desktop.shortcutDelete') }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>
