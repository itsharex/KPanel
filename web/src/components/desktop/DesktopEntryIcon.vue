<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Component } from 'vue'
import { Globe2 } from '@lucide/vue'
import type { DesktopEntry } from '@/lib/desktopEntries'

/**
 * One desktop icon tile. A static navigation app renders a lucide icon; an
 * app-market app renders its market icon image; a site renders its favicon or
 * a globe fallback. Windows-style interaction is intentional: single-click
 * selects, double-click (or Enter) opens. Context-menu behavior is handled by
 * the parent (DesktopView).
 */

const props = defineProps<{
  label: string
  navIcon?: Component
  entry?: DesktopEntry
  gradient: string
  active?: boolean
  selected?: boolean
  order?: number
}>()

const emit = defineEmits<{
  select: [event: MouseEvent]
  open: [event: MouseEvent | KeyboardEvent]
  context: [event: MouseEvent]
  warm: []
}>()

const imageFailed = ref(false)

watch(
  () => props.entry?.iconURL,
  () => {
    imageFailed.value = false
  },
)

const monogram = computed(() => props.label.trim().slice(0, 1).toLocaleUpperCase() || 'K')

function onSelect(event: MouseEvent): void {
  emit('select', event)
}

function onOpen(event: MouseEvent | KeyboardEvent): void {
  event.preventDefault()
  emit('open', event)
}

function onContext(event: MouseEvent): void {
  event.preventDefault()
  event.stopPropagation()
  emit('context', event)
}

function onImageError(): void {
  imageFailed.value = true
}
</script>

<template>
  <button
    class="desktop__icon"
    :class="{
      'desktop__icon--launching': active,
      'desktop__icon--selected': selected,
    }"
    :style="{ '--desktop-entry-order': String(order ?? 0) }"
    type="button"
    :aria-label="label"
    :title="label"
    @click="onSelect"
    @dblclick="onOpen"
    @keydown.enter="onOpen"
    @focus="emit('warm')"
    @pointerenter="emit('warm')"
    @contextmenu="onContext"
  >
    <span class="desktop__icon-glyph" :style="{ background: gradient }">
      <img
        v-if="entry?.iconURL && !imageFailed"
        class="desktop__icon-img"
        :src="entry.iconURL"
        alt=""
        draggable="false"
        loading="lazy"
        decoding="async"
        referrerpolicy="no-referrer"
        width="38"
        height="38"
        @error="onImageError"
      />
      <component v-else-if="navIcon" :is="navIcon" :size="38" :stroke-width="1.6" aria-hidden="true" />
      <Globe2 v-else-if="entry?.kind === 'site'" :size="34" aria-hidden="true" />
      <span v-else class="desktop__icon-monogram" aria-hidden="true">{{ monogram }}</span>
    </span>
    <span class="desktop__icon-label">{{ label }}</span>
  </button>
</template>
