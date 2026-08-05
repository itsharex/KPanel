<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ClipboardPaste, Copy, TextSelect } from '@lucide/vue'
import type { Terminal } from '@xterm/xterm'
import { useI18n } from '@/i18n'

const props = defineProps<{
  getTerminal: () => Terminal | undefined
  canPaste: boolean
  contained?: boolean
}>()

const { t } = useI18n()
const menu = ref<HTMLElement>()
const visible = ref(false)
const left = ref(0)
const top = ref(0)
const selection = ref('')
const feedback = ref<'copyFailed' | 'pasteBlocked'>()

function close(focusTerminal = false): void {
  visible.value = false
  feedback.value = undefined
  if (focusTerminal) props.getTerminal()?.focus()
}

async function positionMenu(): Promise<void> {
  await nextTick()
  if (!menu.value) return
  const padding = 8
  const bounds = menu.value.getBoundingClientRect()
  left.value = Math.max(padding, Math.min(left.value, window.innerWidth - bounds.width - padding))
  top.value = Math.max(padding, Math.min(top.value, window.innerHeight - bounds.height - padding))
  menu.value.querySelector<HTMLButtonElement>('button:not(:disabled)')?.focus()
}

function open(event: MouseEvent): void {
  event.preventDefault()
  event.stopPropagation()
  selection.value = props.getTerminal()?.getSelection() || ''
  feedback.value = undefined
  left.value = event.clientX
  top.value = event.clientY
  visible.value = true
  void positionMenu()
}

function fallbackCopy(value: string): boolean {
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.left = '-10000px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    return document.execCommand('copy')
  } catch {
    return false
  } finally {
    textarea.remove()
  }
}

async function writeClipboard(value: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      // HTTP access and hardened browsers can reject the async Clipboard API.
    }
  }
  return fallbackCopy(value)
}

async function copySelection(): Promise<void> {
  const value = props.getTerminal()?.getSelection() || selection.value
  if (!value) return
  if (await writeClipboard(value)) {
    close(true)
    return
  }
  feedback.value = 'copyFailed'
}

async function pasteClipboard(): Promise<void> {
  if (!props.canPaste) return
  if (navigator.clipboard?.readText) {
    try {
      const value = await navigator.clipboard.readText()
      if (value) props.getTerminal()?.paste(value)
      close(true)
      return
    } catch {
      // Keep the menu open with a keyboard fallback when clipboard read is denied.
    }
  }
  feedback.value = 'pasteBlocked'
  props.getTerminal()?.focus()
}

function selectAll(): void {
  props.getTerminal()?.selectAll()
  close(true)
}

function handlePaste(event: ClipboardEvent): void {
  if (!props.canPaste) {
    event.preventDefault()
    return
  }
  const value = event.clipboardData?.getData('text/plain') || ''
  if (!value) return
  event.preventDefault()
  event.stopPropagation()
  props.getTerminal()?.paste(value)
  close()
}

function handleKeyEvent(event: KeyboardEvent): boolean {
  if (event.type !== 'keydown') return true
  const isMacCopy = event.metaKey && !event.ctrlKey && event.key.toLowerCase() === 'c'
  const isTerminalCopy = event.ctrlKey && event.shiftKey && event.key.toLowerCase() === 'c'
  const isInsertCopy = event.ctrlKey && event.key === 'Insert'
  if ((isMacCopy || isTerminalCopy || isInsertCopy) && props.getTerminal()?.hasSelection()) {
    event.preventDefault()
    event.stopPropagation()
    void copySelection()
    return false
  }
  return true
}

function handleDocumentPointer(event: PointerEvent): void {
  if (visible.value && !menu.value?.contains(event.target as Node)) close()
}

function handleEscape(event: KeyboardEvent): void {
  if (event.key !== 'Escape' || !visible.value) return
  event.preventDefault()
  event.stopImmediatePropagation()
  close(true)
}

function handleViewportChange(): void {
  close()
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointer, true)
  document.addEventListener('keydown', handleEscape, true)
  window.addEventListener('resize', handleViewportChange)
  window.addEventListener('blur', handleViewportChange)
  document.addEventListener('scroll', handleViewportChange, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointer, true)
  document.removeEventListener('keydown', handleEscape, true)
  window.removeEventListener('resize', handleViewportChange)
  window.removeEventListener('blur', handleViewportChange)
  document.removeEventListener('scroll', handleViewportChange, true)
})

defineExpose({ open, handlePaste, handleKeyEvent })
</script>

<template>
  <Teleport to="body" :disabled="contained">
    <div
      v-if="visible"
      ref="menu"
      class="terminal-context-menu"
      role="menu"
      :aria-label="t('terminal.contextMenu')"
      :style="{ left: `${left}px`, top: `${top}px` }"
      @contextmenu.prevent
    >
      <button type="button" role="menuitem" :disabled="!selection" @click="copySelection">
        <Copy :size="15" />
        <span>{{ t('terminal.copy') }}</span>
        <kbd>{{ t('terminal.copyShortcut') }}</kbd>
      </button>
      <button type="button" role="menuitem" :disabled="!canPaste" @click="pasteClipboard">
        <ClipboardPaste :size="15" />
        <span>{{ t('terminal.paste') }}</span>
        <kbd>{{ t('terminal.pasteShortcut') }}</kbd>
      </button>
      <hr />
      <button type="button" role="menuitem" @click="selectAll">
        <TextSelect :size="15" />
        <span>{{ t('terminal.selectAll') }}</span>
      </button>
      <p v-if="feedback" role="status">
        {{ t(feedback === 'copyFailed' ? 'terminal.copyFailed' : 'terminal.pasteBlocked') }}
      </p>
    </div>
  </Teleport>
</template>

<style scoped>
.terminal-context-menu {
  position: fixed;
  z-index: 7000;
  display: grid;
  width: 228px;
  padding: 6px;
  border: 1px solid var(--border-strong, #3a4b4e);
  border-radius: 10px;
  color: var(--text, #e8eeee);
  background: var(--surface, #152023);
  box-shadow: 0 14px 36px rgb(0 0 0 / 32%);
}

.terminal-context-menu button {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  width: 100%;
  border: 0;
  border-radius: 7px;
  padding: 8px 9px;
  color: inherit;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.terminal-context-menu button:hover:not(:disabled),
.terminal-context-menu button:focus-visible {
  outline: none;
  background: var(--surface-subtle, #203034);
}

.terminal-context-menu button:disabled {
  opacity: .42;
  cursor: not-allowed;
}

.terminal-context-menu kbd {
  color: var(--muted, #91a09e);
  font: 10px/1.2 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.terminal-context-menu hr {
  width: 100%;
  height: 1px;
  margin: 5px 0;
  border: 0;
  background: var(--border, #29383a);
}

.terminal-context-menu p {
  margin: 5px 7px 3px;
  color: var(--warning, #d5ae62);
  font-size: 11px;
  line-height: 1.45;
}
</style>
