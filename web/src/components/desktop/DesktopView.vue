<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref } from 'vue'
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
  ListTree,
} from '@lucide/vue'
import DesktopWindow from '@/components/desktop/DesktopWindow.vue'
import DesktopEntryIcon from '@/components/desktop/DesktopEntryIcon.vue'
import DesktopClock from '@/components/desktop/DesktopClock.vue'
import DesktopMonitor from '@/components/desktop/DesktopMonitor.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import LogoMark from '@/components/common/LogoMark.vue'
import { DEFAULT_WINDOW_GRADIENT, desktopApps, findDesktopApp } from '@/lib/desktopApps'
import {
  getCachedDesktopEntries,
  loadDesktopEntries,
  type DesktopEntries,
  type DesktopEntry,
} from '@/lib/desktopEntries'
import { api, type SystemResourceSnapshot } from '@/lib/api'
import { prefetchNavigationRoute } from '@/lib/navigation'
import { embeddedBrowserSitesKey } from '@/lib/embeddedBrowser'
import {
  desktopCloseGuardCoordinator,
  desktopCloseGuardCoordinatorKey,
} from '@/lib/desktopRouteKeys'
import { useDesktopMode } from '@/stores/desktopMode'
import { useTheme } from '@/stores/theme'
import { useToast } from '@/stores/toast'
import { useI18n } from '@/i18n'
import type { AgentStatus } from '@/types/api'

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

function persistSiteNames(): void {
  try {
    window.localStorage.setItem(SITE_RENAMES_KEY, JSON.stringify(siteNames.value))
  } catch {
    // Browser storage is optional; the in-memory rename still works.
  }
}

const entries = ref<DesktopEntries | undefined>(applySiteNames(getCachedDesktopEntries()))
const browserSites = computed(() => (entries.value?.sites || []).flatMap((entry) => (
  entry.url
    ? [{ id: entry.id, name: entry.name, url: entry.url, iconURL: entry.iconURL }]
    : []
)))
provide(embeddedBrowserSitesKey, browserSites)
const systemResources = ref<SystemResourceSnapshot>()
const entriesLoading = ref(!entries.value)
let entriesAbort: AbortController | undefined
let entriesSequence = 0

// Context menu: `targetEntry` set when the menu is for an entry icon; cleared
// for the empty-desktop menu.
const contextMenu = ref<{ x: number; y: number; open: boolean }>({ x: 0, y: 0, open: false })
const contextMenuTarget = ref<'desktop' | 'taskbar'>('desktop')
const contextMenuElement = ref<HTMLElement>()
const menuEntry = ref<DesktopEntry>()
const detailEntry = ref<DesktopEntry>()
const renameEntry = ref<DesktopEntry>()
const renameValue = ref('')
let contextMenuOpener: HTMLElement | undefined

/** Icons currently playing their open-bounce animation. */
const bouncingIcon = ref<string>('')
const selectedIcon = ref<string>('')
let bounceTimer: number | undefined
let resizeFrame: number | undefined
let resizePersistTimer: number | undefined
let browserRequestSequence = 0

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
  if (entry.kind === 'site') {
    openWebsiteEntry(entry)
    return
  }
  openEntryExternally(entry)
}

function openEntryExternally(entry: DesktopEntry): void {
  if (!entry.url) return
  window.open(entry.url, '_blank', 'noopener,noreferrer')
}

function openWebsiteEntry(entry: DesktopEntry): void {
  if (!entry.url) return
  const app = findDesktopApp('/browser')
  if (!app) return
  browserRequestSequence += 1
  const query = new URLSearchParams({
    site: entry.id,
    url: entry.url,
    request: String(browserRequestSequence),
  })
  const windowId = desktop.openWindow(
    `/browser?${query.toString()}`,
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
  selectedIcon.value = `app:${path}`
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
  target: 'desktop' | 'taskbar' = 'desktop',
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
  void showContextMenu(event)
}

function onTaskbarContext(event: MouseEvent): void {
  void showContextMenu(event, undefined, 'taskbar')
}

function onEntryOpen(_event: MouseEvent | KeyboardEvent, entry: DesktopEntry): void {
  openEntry(entry)
}

function closeContextMenu(restoreFocus = true): void {
  contextMenu.value.open = false
  menuEntry.value = undefined
  const opener = contextMenuOpener
  contextMenuOpener = undefined
  if (restoreFocus && opener?.isConnected) {
    void nextTick(() => opener.focus({ preventScroll: true }))
  }
}

function onGlobalPointerDown(event: PointerEvent): void {
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
  if (contextMenu.value.open) closeContextMenu()
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

function onContextMenuAction(action: 'refresh' | 'theme' | 'classic' | 'about' | 'processes'): void {
  closeContextMenu()
  switch (action) {
    case 'refresh':
      void loadEntries(true)
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

async function enterClassicSafely(): Promise<void> {
  if (await desktopCloseGuardCoordinator.checkAll()) desktop.enterClassic()
}

function onEntryMenuOpen(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry) openEntry(entry)
}

function onEntryMenuExternal(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry?.kind === 'site') openEntryExternally(entry)
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

function saveRename(): void {
  const entry = renameEntry.value
  const name = renameValue.value.trim().slice(0, MAX_SITE_NAME_LENGTH)
  if (entry?.kind !== 'site' || !name) return
  const defaultName = defaultSiteName(entry)
  const next = { ...siteNames.value }
  if (name === defaultName) delete next[entry.id]
  else next[entry.id] = name
  siteNames.value = next
  persistSiteNames()
  entries.value = applySiteNames(entries.value)
  closeRename()
}

function resetRename(): void {
  const entry = renameEntry.value
  if (entry?.kind !== 'site') return
  const next = { ...siteNames.value }
  delete next[entry.id]
  siteNames.value = next
  persistSiteNames()
  entries.value = applySiteNames(entries.value)
  closeRename()
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
})

onBeforeUnmount(() => {
  entriesSequence += 1
  document.documentElement.classList.remove('desktop-mode-open')
  document.body.classList.remove('desktop-mode-open')
  window.removeEventListener('pointerdown', onGlobalPointerDown)
  window.removeEventListener('keydown', onGlobalKeyDown)
  window.removeEventListener('resize', onViewportResize)
  entriesAbort?.abort()
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
      class="desktop__icons"
      :aria-label="i18n.t('desktop.gridLabel')"
      :aria-busy="entriesLoading"
    >
      <!-- Static navigation apps -->
      <DesktopEntryIcon
        v-for="(app, index) in desktopApps"
        :key="app.path"
        :label="i18n.t(app.labelKey)"
        :nav-icon="app.icon"
        :gradient="gradientFor(app.path)"
        :active="bouncingIcon === app.path"
        :selected="selectedIcon === `app:${app.path}`"
        :order="index"
        @select="selectNavIcon(app.path)"
        @open="openNavIcon(app.path)"
        @context="(event) => onNavContext(event, app.path)"
        @warm="warmNavIcon(app.path)"
      />

      <!-- Dynamic entries: installed apps and sites -->
      <template v-if="entries">
        <DesktopEntryIcon
          v-for="(entry, index) in entries.visible"
          :key="entry.key"
          :label="entry.name"
          :entry="entry"
          :gradient="entryGradient(entry)"
          :selected="selectedIcon === entry.key"
          :order="desktopApps.length + index"
          @select="selectEntry(entry)"
          @open="(event) => onEntryOpen(event, entry)"
          @context="(event) => onEntryContext(event, entry)"
        />
      </template>
      <span v-if="entriesLoading" class="desktop__sr-only" aria-live="polite">
        {{ i18n.t('desktop.entriesLoading') }}
      </span>
    </nav>

    <DesktopWindow
      v-for="windowState in openWindows"
      :key="windowState.id"
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
        :class="{ 'desktop__context-menu--entry': menuEntry }"
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        role="menu"
        @contextmenu.prevent.stop
        @pointerdown.stop
        @keydown="onContextMenuKeyDown"
      >
        <template v-if="menuEntry">
          <button type="button" role="menuitem" @click="onEntryMenuOpen">
            <SquareTerminal v-if="menuEntry.launch === 'script'" :size="15" aria-hidden="true" />
            <AppWindow v-else-if="menuEntry.kind === 'site'" :size="15" aria-hidden="true" />
            <ExternalLink v-else :size="15" aria-hidden="true" />
            {{ menuEntry.launch === 'script'
              ? i18n.t('desktop.entryScriptManage')
              : menuEntry.kind === 'site'
                ? i18n.t('desktop.browserOpenEmbedded')
                : i18n.t('desktop.entryOpen') }}
          </button>
          <button
            v-if="menuEntry.kind === 'site'"
            type="button"
            role="menuitem"
            @click="onEntryMenuExternal"
          >
            <ExternalLink :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.browserOpenExternal') }}
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
        <template v-else>
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
          :aria-pressed="windowState.id === focusedWindow?.id"
          @click="onTaskbarClick(windowState.id)"
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
          <dd class="desktop__detail-url">
            <a :href="detailEntry.url" target="_blank" rel="noopener noreferrer">
              {{ detailEntry.url }}
            </a>
          </dd>
        </template>
        <template v-else>
          <dt>{{ i18n.t('desktop.detailType') }}</dt>
          <dd>{{ i18n.t('desktop.detailSite') }}</dd>
          <dt>{{ i18n.t('desktop.detailDomain') }}</dt>
          <dd>{{ detailEntry.site?.primaryDomain }}</dd>
          <dt>{{ i18n.t('desktop.detailType2') }}</dt>
          <dd>{{ detailEntry.site?.type }}</dd>
          <dt>{{ i18n.t('desktop.detailURL') }}</dt>
          <dd class="desktop__detail-url">
            <a :href="detailEntry.url" target="_blank" rel="noopener noreferrer">
              {{ detailEntry.url }}
            </a>
          </dd>
        </template>
      </dl>
      <template #footer>
        <button class="button button--primary" type="button" @click="detailEntry ? openEntry(detailEntry) : undefined">
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
  </div>
</template>
