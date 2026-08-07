<script setup lang="ts">
import type { Component } from 'vue'
import { Globe2 } from '@lucide/vue'
import type { DesktopEntry } from '@/lib/desktopEntries'

/**
 * One desktop icon tile. A static navigation app renders a lucide icon; an
 * app-market app renders its market icon image; a site renders its favicon or
 * a globe fallback. Double-click and context-menu behavior is handled by the
 * parent (DesktopView). Entry icons stop contextmenu propagation so the entry
 * menu opens; static nav icons let the event bubble to the desktop menu.
 */

const props = defineProps<{
  label: string
  navIcon?: Component
  entry?: DesktopEntry
  gradient: string
  active?: boolean
}>()

const emit = defineEmits<{
  dblclick: [event: MouseEvent]
  context: [event: MouseEvent]
}>()

function onDblClick(event: MouseEvent): void {
  emit('dblclick', event)
}

function onContext(event: MouseEvent): void {
  // Static nav icons have no entry: let the event reach the desktop menu.
  if (!props.entry) return
  // Entry icons: stop the native contextmenu so only the entry menu opens.
  event.preventDefault()
  event.stopPropagation()
  emit('context', event)
}
</script>

<template>
  <button
    class="desktop__icon"
    :class="{ 'desktop__icon--active': active }"
    type="button"
    :aria-label="label"
    :title="label"
    @dblclick="onDblClick"
    @contextmenu="onContext"
  >
    <span class="desktop__icon-glyph" :style="{ background: gradient }">
      <img
        v-if="entry?.iconURL"
        class="desktop__icon-img"
        :src="entry.iconURL"
        :alt="label"
        loading="lazy"
        decoding="async"
        width="30"
        height="30"
      />
      <component v-else-if="navIcon" :is="navIcon" :size="30" :stroke-width="1.6" aria-hidden="true" />
      <Globe2 v-else :size="28" aria-hidden="true" />
    </span>
    <span class="desktop__icon-label">{{ label }}</span>
  </button>
</template>
