<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Component } from 'vue'
import { ArrowLeft, RefreshCw, Sun, Moon, Info, AppWindow } from '@lucide/vue'
import DesktopWindow from '@/components/desktop/DesktopWindow.vue'
import { DEFAULT_WINDOW_GRADIENT, desktopApps, findDesktopApp } from '@/lib/desktopApps'
import { useDesktopMode } from '@/stores/desktopMode'
import { useTheme } from '@/stores/theme'
import { useToast } from '@/stores/toast'
import { useI18n } from '@/i18n'

/**
 * Desktop overlay. Renders when desktop mode is active: a wallpaper layer, an
 * icon grid (double-click to open a window), a taskbar with open windows, and
 * a context menu. Switching back to classic mode lives in the top-right corner.
 */

const desktop = useDesktopMode()
const theme = useTheme()
const toast = useToast()
const i18n = useI18n()

const openWindows = computed(() => desktop.windows.value)
const focusedWindow = computed(() =>
  desktop.windows.value.find((windowState) => windowState.id === desktop.focusedId.value),
)

const contextMenu = ref<{ x: number; y: number; open: boolean }>({ x: 0, y: 0, open: false })
/** Icons currently playing their open-bounce animation. */
const bouncingIcon = ref<string>('')
let bounceTimer: number | undefined

function gradientFor(path: string): string {
  const gradient = findDesktopApp(path)?.gradient ?? DEFAULT_WINDOW_GRADIENT
  return `linear-gradient(145deg, ${gradient[0]} 0%, ${gradient[1]} 100%)`
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

function windowIcon(path: string): Component {
  return findDesktopApp(path)?.icon ?? AppWindow
}

function windowTitle(titleKey: string): string {
  return i18n.t(titleKey as Parameters<typeof i18n.t>[0])
}

function onContextMenu(event: MouseEvent): void {
  event.preventDefault()
  contextMenu.value = { x: event.clientX, y: event.clientY, open: true }
}

function closeContextMenu(): void {
  contextMenu.value.open = false
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

function onTaskbarClick(windowId: number): void {
  const target = desktop.windows.value.find((windowState) => windowState.id === windowId)
  if (!target) return
  if (target.minimized || desktop.focusedId.value !== windowId) {
    desktop.restoreWindow(windowId)
  } else {
    desktop.minimizeWindow(windowId)
  }
}

onMounted(() => {
  window.addEventListener('pointerdown', onGlobalPointerDown)
  window.addEventListener('resize', onViewportResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('pointerdown', onGlobalPointerDown)
  window.removeEventListener('resize', onViewportResize)
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
      <button
        v-for="app in desktopApps"
        :key="app.path"
        class="desktop__icon"
        :class="{ 'desktop__icon--bouncing': bouncingIcon === app.path }"
        type="button"
        role="gridcell"
        :aria-label="i18n.t(app.labelKey)"
        :title="i18n.t(app.labelKey)"
        @dblclick="openApp(app.path)"
      >
        <span
          class="desktop__icon-glyph"
          :style="{ background: gradientFor(app.path) }"
        >
          <component :is="app.icon" :size="30" :stroke-width="1.6" aria-hidden="true" />
        </span>
        <span class="desktop__icon-label">{{ i18n.t(app.labelKey) }}</span>
      </button>
    </div>

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
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        role="menu"
        @contextmenu.prevent
        @pointerdown.stop
      >
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
  </div>
</template>
