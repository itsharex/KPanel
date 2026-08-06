<script setup lang="ts">
import { computed, onBeforeUnmount, provide, ref } from 'vue'
import type { Component } from 'vue'
import { Minimize, Square, Copy, X } from '@lucide/vue'
import { createWindowRouter, reactiveRouteFor, resolveWindowComponent } from '@/lib/desktopWindowRoute'
import { windowRouteKey, windowRouterKey } from '@/lib/desktopRouteKeys'
import { useDesktopMode } from '@/stores/desktopMode'
import { useI18n } from '@/i18n'
import { useWindowGesture } from '@/composables/useWindowGesture'
import type { DesktopWindowState } from '@/stores/desktopMode'

/**
 * One desktop window. The window renders the page component for its route
 * directly (no nested RouterView) and provides a window-scoped router + route,
 * so views that call `useRoute()` / `useRouter()` / `RouterLink` resolve against
 * the window. `AiView` can navigate `/ai/s/:id` inside the window without
 * leaving the desktop, and multiple windows of the same app stay independent.
 */

const props = defineProps<{
  windowState: DesktopWindowState
  icon: Component
}>()

const desktop = useDesktopMode()
const i18n = useI18n()
const router = createWindowRouter(props.windowState.path)
const component = ref<Component>()
const open = ref(false)
const closing = ref(false)

// Provide the window router + route to this window's subtree so views resolve
// navigation against the window instead of the app.
provide(windowRouterKey, router)
provide(windowRouteKey, reactiveRouteFor(router))

const isFocused = computed(() => desktop.focusedId.value === props.windowState.id)

const title = computed(() => i18n.t(props.windowState.titleKey as Parameters<typeof i18n.t>[0]))

function isMaximized(): boolean {
  return props.windowState.maximized
}

function onMinimize(): void {
  desktop.minimizeWindow(props.windowState.id)
}

function onToggleMaximize(): void {
  desktop.toggleMaximize(props.windowState.id)
}

function onClose(): void {
  closing.value = true
  window.setTimeout(() => desktop.closeWindow(props.windowState.id), 160)
}

function onPointerDownWindow(): void {
  desktop.focusWindow(props.windowState.id)
}

// Load the page component for this window's route lazily.
const loadComponent = () => {
  void resolveWindowComponent(props.windowState.path).then((comp) => {
    component.value = comp
  })
}
void router.push(props.windowState.path)
loadComponent()

// Drag + resize gesture updates geometry through the store.
const gesture = useWindowGesture(
  () => props.windowState.geometry,
  (geometry) => desktop.updateGeometry(props.windowState.id, geometry),
  () => desktop.focusWindow(props.windowState.id),
)

onBeforeUnmount(() => {
  void router
})

// Open animation: first frame registers the open state.
requestAnimationFrame(() => {
  open.value = true
})

const handleEdges = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw'] as const
</script>

<template>
  <section
    class="desktop-window"
    :class="{
      'desktop-window--focused': isFocused,
      'desktop-window--maximized': isMaximized(),
      'desktop-window--minimized': windowState.minimized,
      'desktop-window--open': open,
      'desktop-window--closing': closing,
    }"
    :style="{
      left: `${windowState.geometry.left}px`,
      top: `${windowState.geometry.top}px`,
      width: `${windowState.geometry.width}px`,
      height: `${windowState.geometry.height}px`,
      zIndex: windowState.z,
    }"
    role="dialog"
    :aria-label="title"
    :aria-hidden="windowState.minimized"
    @pointerdown="onPointerDownWindow"
  >
    <header
      class="desktop-window__titlebar"
      @pointerdown="(event) => gesture.onPointerDown(event, null)"
      @dblclick="onToggleMaximize"
    >
      <div class="desktop-window__title">
        <component :is="icon" :size="15" :stroke-width="2" aria-hidden="true" />
        <span>{{ title }}</span>
      </div>
      <div class="desktop-window__actions">
        <button
          class="desktop-window__action"
          type="button"
          :aria-label="i18n.t('desktop.minimize')"
          @click="onMinimize"
        >
          <Minimize :size="13" />
        </button>
        <button
          class="desktop-window__action"
          type="button"
          :aria-label="i18n.t('desktop.maximize')"
          @click="onToggleMaximize"
        >
          <Square v-if="!isMaximized()" :size="12" />
          <Copy v-else :size="12" />
        </button>
        <button
          class="desktop-window__action desktop-window__action--close"
          type="button"
          :aria-label="i18n.t('desktop.close')"
          @click="onClose"
        >
          <X :size="14" />
        </button>
      </div>
    </header>

    <div class="desktop-window__body">
      <component :is="component" v-if="component" />
    </div>

    <div
      v-for="edge in handleEdges"
      :key="edge"
      class="desktop-window__resize"
      :class="`desktop-window__resize--${edge}`"
      :data-resize-edge="edge"
      :aria-hidden="true"
      @pointerdown="(event) => gesture.onPointerDown(event, gesture.edgeForTarget(event.target as HTMLElement))"
    />
  </section>
</template>
