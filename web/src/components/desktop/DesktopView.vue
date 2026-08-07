<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Component } from 'vue'
import {
  ArrowLeft,
  RefreshCw,
  Sun,
  Moon,
  Info,
  AppWindow,
  ExternalLink,
} from '@lucide/vue'
import DesktopWindow from '@/components/desktop/DesktopWindow.vue'
import DesktopEntryIcon from '@/components/desktop/DesktopEntryIcon.vue'
import DesktopMonitor from '@/components/desktop/DesktopMonitor.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import { DEFAULT_WINDOW_GRADIENT, desktopApps, findDesktopApp } from '@/lib/desktopApps'
import { loadDesktopEntries, type DesktopEntries, type DesktopEntry } from '@/lib/desktopEntries'
import { useDesktopMode } from '@/stores/desktopMode'
import { useTheme } from '@/stores/theme'
import { useToast } from '@/stores/toast'
import { useI18n } from '@/i18n'

/**
 * Desktop overlay. Renders when desktop mode is active: a wallpaper layer, an
 * icon grid (static nav apps + installed app-market apps + configured sites),
 * a taskbar with open windows, and context menus. Switching back to classic
 * mode lives in the top-right corner.
 */

const desktop = useDesktopMode()
const theme = useTheme()
const toast = useToast()
const i18n = useI18n()

const openWindows = computed(() => desktop.windows.value)
const focusedWindow = computed(() =>
  desktop.windows.value.find((windowState) => windowState.id === desktop.focusedId.value),
)

// Dynamic entries: installed apps and configured sites surfaced as desktop
// icons that open their external URL.
const entries = ref<DesktopEntries>()
const entriesLoading = ref(true)
let entriesAbort: AbortController | undefined

// Context menu: `targetEntry` set when the menu is for an entry icon; cleared
// for the empty-desktop menu.
const contextMenu = ref<{ x: number; y: number; open: boolean }>({ x: 0, y: 0, open: false })
const menuEntry = ref<DesktopEntry>()
const detailEntry = ref<DesktopEntry>()

/** Icons currently playing their open-bounce animation. */
const bouncingIcon = ref<string>('')
let bounceTimer: number | undefined

function gradientFor(path: string): string {
  const gradient = findDesktopApp(path)?.gradient ?? DEFAULT_WINDOW_GRADIENT
  return `linear-gradient(145deg, ${gradient[0]} 0%, ${gradient[1]} 100%)`
}

const SITE_GRADIENT: [string, string] = ['#22d3ee', '#0e7490']

function entryGradient(entry: DesktopEntry): string {
  if (entry.kind === 'site') {
    return `linear-gradient(145deg, ${SITE_GRADIENT[0]} 0%, ${SITE_GRADIENT[1]} 100%)`
  }
  // App-market apps keep a neutral brand tile; the market icon image sits on it.
  return `linear-gradient(145deg, #5b7a72 0%, #243b36 100%)`
}

function openApp(path: string): void {
  const app = findDesktopApp(path)
  if (!app) return
  // Play a quick bounce on the icon before opening the window.
  if (bouncingIcon.value === path) return
  bouncingIcon.value = path
  window.setTimeout(() => {
    bouncingIcon.value = ''
    desktop.openWindow(app.path, app.labelKey, app.allowMultiple)
  }, 180)
}

function openEntry(entry: DesktopEntry): void {
  // Open the external URL in a new tab, never inside the desktop shell.
  window.open(entry.url, '_blank', 'noopener,noreferrer')
}

function openNavIcon(path: string): void {
  openApp(path)
}

function windowIcon(path: string): Component {
  return findDesktopApp(path)?.icon ?? AppWindow
}

function windowTitle(titleKey: string): string {
  return i18n.t(titleKey as Parameters<typeof i18n.t>[0])
}

function onContextMenu(event: MouseEvent): void {
  event.preventDefault()
  contextMenu.value = { x: event.clientX, y: event.clientY, open: true }
  menuEntry.value = undefined
}

function onEntryContext(event: MouseEvent, entry: DesktopEntry): void {
  event.preventDefault()
  contextMenu.value = { x: event.clientX, y: event.clientY, open: true }
  menuEntry.value = entry
}

function onEntryDoubleClick(_event: MouseEvent, entry: DesktopEntry): void {
  openEntry(entry)
}

function closeContextMenu(): void {
  contextMenu.value.open = false
  menuEntry.value = undefined
}

function onGlobalPointerDown(): void {
  closeContextMenu()
}

function onContextMenuAction(action: 'refresh' | 'theme' | 'classic' | 'about'): void {
  closeContextMenu()
  switch (action) {
    case 'refresh':
      window.location.reload()
      break
    case 'theme':
      theme.setTheme(theme.resolved.value === 'dark' ? 'light' : 'dark')
      break
    case 'classic':
      desktop.enterClassic()
      break
    case 'about':
      toast.success(i18n.t('desktop.aboutTitle'), i18n.t('desktop.aboutMessage'))
      break
  }
}

function onEntryMenuOpen(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry) openEntry(entry)
}

function onEntryMenuDetails(): void {
  const entry = menuEntry.value
  closeContextMenu()
  if (entry) detailEntry.value = entry
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

async function loadEntries(): Promise<void> {
  entriesAbort?.abort()
  entriesAbort = new AbortController()
  entriesLoading.value = true
  try {
    entries.value = await loadDesktopEntries(entriesAbort.signal)
  } catch {
    entries.value = undefined
  } finally {
    entriesLoading.value = false
  }
}

onMounted(() => {
  window.addEventListener('pointerdown', onGlobalPointerDown)
  window.addEventListener('resize', onViewportResize)
  void loadEntries()
})

onBeforeUnmount(() => {
  window.removeEventListener('pointerdown', onGlobalPointerDown)
  window.removeEventListener('resize', onViewportResize)
  entriesAbort?.abort()
  if (bounceTimer) window.clearTimeout(bounceTimer)
})

function onViewportResize(): void {
  desktop.resizeForViewport({ width: window.innerWidth, height: window.innerHeight })
}
</script>

<template>
  <div class="desktop" @contextmenu="onContextMenu">
    <div class="desktop__wallpaper" aria-hidden="true">
      <div class="desktop__aurora desktop__aurora--one" />
      <div class="desktop__aurora desktop__aurora--two" />
      <div class="desktop__aurora desktop__aurora--three" />
    </div>

    <div class="desktop__topbar">
      <div class="desktop__mode-switch">
        <button
          class="desktop__classic-button"
          type="button"
          :title="i18n.t('desktop.switchClassic')"
          :aria-label="i18n.t('desktop.switchClassic')"
          @click="desktop.enterClassic()"
        >
          <ArrowLeft :size="16" />
          <span>{{ i18n.t('desktop.switchClassic') }}</span>
        </button>
      </div>
    </div>

    <div class="desktop__icons" role="grid" :aria-label="i18n.t('desktop.gridLabel')">
      <!-- Static navigation apps -->
      <DesktopEntryIcon
        v-for="app in desktopApps"
        :key="app.path"
        :label="i18n.t(app.labelKey)"
        :nav-icon="app.icon"
        :gradient="gradientFor(app.path)"
        :active="bouncingIcon === app.path"
        @dblclick="openNavIcon(app.path)"
      />

      <!-- Dynamic entries: installed apps and sites -->
      <template v-if="entries">
        <DesktopEntryIcon
          v-for="entry in entries.visible"
          :key="entry.key"
          :label="entry.name"
          :entry="entry"
          :gradient="entryGradient(entry)"
          @dblclick="(event) => onEntryDoubleClick(event, entry)"
          @context="(event) => onEntryContext(event, entry)"
        />
      </template>
    </div>

    <DesktopMonitor />

    <DesktopWindow
      v-for="windowState in openWindows"
      :key="windowState.id"
      :window-state="windowState"
      :icon="windowIcon(windowState.path)"
    />

    <Transition name="desktop-menu">
      <div
        v-if="contextMenu.open"
        class="desktop__context-menu"
        :class="{ 'desktop__context-menu--entry': menuEntry }"
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        role="menu"
        @contextmenu.prevent
        @pointerdown.stop
      >
        <template v-if="menuEntry">
          <button type="button" role="menuitem" @click="onEntryMenuOpen">
            <ExternalLink :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.entryOpen') }}
          </button>
          <button type="button" role="menuitem" @click="onEntryMenuDetails">
            <Info :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.entryDetails') }}
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

    <footer class="desktop__taskbar" role="toolbar" :aria-label="i18n.t('desktop.taskbarLabel')">
      <button
        v-for="windowState in openWindows"
        :key="windowState.id"
        class="desktop__taskbar-item"
        :class="{
          'desktop__taskbar-item--active': windowState.id === focusedWindow?.id,
          'desktop__taskbar-item--minimized': windowState.minimized,
        }"
        type="button"
        :aria-label="windowTitle(windowState.titleKey)"
        :aria-pressed="windowState.id === focusedWindow?.id"
        @click="onTaskbarClick(windowState.id)"
      >
        <span
          class="desktop__taskbar-glyph"
          :style="{ background: gradientFor(windowState.path) }"
        >
          <component
            :is="findDesktopApp(windowState.path)?.icon"
            :size="15"
            :stroke-width="2"
            aria-hidden="true"
          />
        </span>
        <span>{{ windowTitle(windowState.titleKey) }}</span>
      </button>
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
  </div>
</template>
