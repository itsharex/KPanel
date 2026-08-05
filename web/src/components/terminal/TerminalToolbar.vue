<script setup lang="ts">
import { ArrowUpToLine, Maximize2, Minimize2 } from '@lucide/vue'
import { useI18n } from '@/i18n'

defineProps<{
  fullscreen: boolean
}>()

const emit = defineEmits<{
  scrollTop: []
  toggleFullscreen: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="terminal-toolbar" role="toolbar" :aria-label="t('terminal.toolbar')" @contextmenu.prevent.stop>
    <button
      type="button"
      :title="t('terminal.scrollToTop')"
      :aria-label="t('terminal.scrollToTop')"
      @click.stop="emit('scrollTop')"
    >
      <ArrowUpToLine :size="17" />
    </button>
    <button
      type="button"
      :title="fullscreen ? t('terminal.restoreWindow') : t('terminal.fillPage')"
      :aria-label="fullscreen ? t('terminal.restoreWindow') : t('terminal.fillPage')"
      @click.stop="emit('toggleFullscreen')"
    >
      <Minimize2 v-if="fullscreen" :size="17" />
      <Maximize2 v-else :size="17" />
    </button>
  </div>
</template>

<style scoped>
.terminal-toolbar {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.terminal-toolbar button {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid var(--terminal-shell-border, var(--terminal-border, #29383a));
  border-radius: 8px;
  color: var(--terminal-shell-muted, var(--terminal-muted, #8a9695));
  background: var(--terminal-shell-panel, var(--terminal-panel, #111a1d));
  cursor: pointer;
}

.terminal-toolbar button:hover,
.terminal-toolbar button:focus-visible {
  color: var(--terminal-shell-text, var(--terminal-text, #d8dddc));
  border-color: var(--brand, var(--terminal-accent, #35cba6));
  outline: none;
}
</style>
