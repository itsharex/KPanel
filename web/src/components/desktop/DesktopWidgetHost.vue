<script setup lang="ts">
import type { DesktopWidgetDefinition } from '@/lib/desktopWidgets'

defineProps<{
  widget: DesktopWidgetDefinition
  componentProps?: Record<string, unknown>
}>()

const emit = defineEmits<{
  'drag-start': [event: PointerEvent]
  nudge: [direction: 'left' | 'right' | 'up' | 'down']
}>()

function onPointerDown(event: PointerEvent): void {
  const target = event.target instanceof Element ? event.target : undefined
  if (target?.closest('a, button, input, select, textarea, option, [contenteditable="true"], [data-widget-interactive]')) return
  emit('drag-start', event)
}

function onKeyDown(event: KeyboardEvent): void {
  if (!(event.ctrlKey || event.metaKey)) return
  const directions: Record<string, 'left' | 'right' | 'up' | 'down'> = {
    ArrowLeft: 'left', ArrowRight: 'right', ArrowUp: 'up', ArrowDown: 'down',
  }
  const direction = directions[event.key]
  if (!direction) return
  event.preventDefault()
  event.stopPropagation()
  emit('nudge', direction)
}
</script>

<template>
  <div
    class="desktop-widget-slot"
    :aria-label="widget.key"
    role="group"
    tabindex="0"
    @pointerdown="onPointerDown"
    @keydown="onKeyDown"
    @contextmenu.stop
  >
    <component
      :is="widget.component"
      v-bind="componentProps"
    />
  </div>
</template>
