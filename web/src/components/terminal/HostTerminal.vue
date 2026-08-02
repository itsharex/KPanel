<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { X } from '@lucide/vue'
import { api, ApiError } from '@/lib/api'
import { openTerminalURL } from '@/lib/terminalLinks'
import { takeTerminalInputChunk, terminalInputShouldFlushImmediately, terminalLineSubmission } from '@/lib/terminalInput'
import { useI18n } from '@/i18n'

const { t } = useI18n()

const props = defineProps<{
  sessionId: string
  hostName: string
  initialOffset: number
}>()

const emit = defineEmits<{
  close: []
}>()

const host = ref<HTMLElement>()
const pendingLine = ref('')
const state = ref<'connecting' | 'connected' | 'reconnecting' | 'finished'>('connecting')
let terminal: Terminal | undefined
let fitAddon: FitAddon | undefined
let observer: ResizeObserver | undefined
let pollController: AbortController | undefined
let pollTimer: number | undefined
let inputTimer: number | undefined
let resizeTimer: number | undefined
let inputQueue = ''
let inputSending = false
let offset = props.initialOffset
let disposed = false
let closeSent = false
let lastRows = 0
let lastColumns = 0

function themeColor(name: string, fallback: string): string {
  if (!host.value) return fallback
  return window.getComputedStyle(host.value).getPropertyValue(name).trim() || fallback
}

function decodeBase64(value: string): Uint8Array {
  const decoded = window.atob(value)
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0))
}

function encodeBase64(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return window.btoa(binary).replace(/=+$/, '')
}

async function flushInput(): Promise<void> {
  if (inputSending || disposed || state.value === 'finished') return
  if (inputTimer) window.clearTimeout(inputTimer)
  inputTimer = undefined
  inputSending = true
  try {
    while (inputQueue && !disposed) {
      const { chunk, rest } = takeTerminalInputChunk(inputQueue)
      inputQueue = rest
      await api.terminals.input(props.sessionId, encodeBase64(chunk))
    }
  } catch {
    inputQueue = ''
    terminal?.write(`\r\n\x1b[31m[KPanel] ${t('terminal.inputFailed')}\x1b[0m\r\n`)
    state.value = 'reconnecting'
  } finally {
    inputSending = false
  }
}

function queueInput(data: string): void {
  if (disposed || state.value === 'finished') return
  inputQueue += data
  if (terminalInputShouldFlushImmediately(data) || new TextEncoder().encode(inputQueue).byteLength >= 2048) {
    void flushInput()
  } else if (!inputTimer) {
    inputTimer = window.setTimeout(() => void flushInput(), 12)
  }
}

function submitPendingLine(): void {
  if (!pendingLine.value || state.value === 'finished') return
  const value = terminalLineSubmission(pendingLine.value)
  pendingLine.value = ''
  queueInput(value)
}

async function poll(): Promise<void> {
  if (disposed || state.value === 'finished') return
  pollController?.abort()
  pollController = new AbortController()
  try {
    const chunk = await api.terminals.output(props.sessionId, offset, pollController.signal)
    if (chunk.truncated) terminal?.write(`\r\n\x1b[33m[KPanel] ${t('terminal.outputTruncated')}\x1b[0m\r\n`)
    if (chunk.data) terminal?.write(decodeBase64(chunk.data))
    offset = chunk.nextOffset
    state.value = chunk.closed || chunk.exitedAt ? 'finished' : 'connected'
    if (chunk.exitError) terminal?.write(`\r\n\x1b[31m[KPanel] ${chunk.exitError}\x1b[0m\r\n`)
    if (state.value !== 'finished') pollTimer = window.setTimeout(() => void poll(), 0)
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    if (reason instanceof ApiError && reason.code === 'terminal_not_found') {
      state.value = 'finished'
      return
    }
    state.value = 'reconnecting'
    pollTimer = window.setTimeout(() => void poll(), 750)
  }
}

function scheduleResize(): void {
  if (resizeTimer) window.clearTimeout(resizeTimer)
  resizeTimer = window.setTimeout(() => {
    fitAddon?.fit()
    const rows = terminal?.rows || 0
    const columns = terminal?.cols || 0
    if (!rows || !columns || (rows === lastRows && columns === lastColumns)) return
    lastRows = rows
    lastColumns = columns
    void api.terminals.resize(props.sessionId, rows, columns).catch(() => undefined)
  }, 100)
}

function closeSession(): void {
  if (!closeSent) {
    closeSent = true
    void api.terminals.close(props.sessionId).catch(() => undefined)
  }
  emit('close')
}

onMounted(() => {
  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    convertEol: false,
    fontFamily: '"Cascadia Code", "SFMono-Regular", Consolas, monospace',
    fontSize: 13,
    lineHeight: 1.25,
    scrollback: 5000,
    theme: {
      background: themeColor('--terminal-shell-background', '#0b1214'),
      foreground: themeColor('--terminal-shell-text', '#d8dddc'),
      cursor: themeColor('--brand', '#35cba6'),
      cursorAccent: themeColor('--terminal-shell-background', '#0b1214'),
      selectionBackground: 'rgb(53 203 166 / 20%)',
      black: '#1d2426', red: '#d86f74', green: '#91b56d', yellow: '#d5ae62',
      blue: '#76a4c7', magenta: '#ad8bb8', cyan: '#72aaa7', white: '#c9cecd',
      brightBlack: '#687376', brightRed: '#e68589', brightGreen: '#a7c982', brightYellow: '#e3c27b',
      brightBlue: '#8bb9dc', brightMagenta: '#c19bcb', brightCyan: '#8cc2be', brightWhite: '#f0f2f1',
    },
  })
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon((_event, uri) => void openTerminalURL(uri)))
  terminal.parser.registerOscHandler(8, () => true)
  terminal.parser.registerOscHandler(52, () => true)
  terminal.onData(queueInput)
  if (host.value) {
    terminal.open(host.value)
    observer = new ResizeObserver(scheduleResize)
    observer.observe(host.value)
    scheduleResize()
    terminal.focus()
  }
  void poll()
})

onBeforeUnmount(() => {
  disposed = true
  pollController?.abort()
  if (pollTimer) window.clearTimeout(pollTimer)
  if (inputTimer) window.clearTimeout(inputTimer)
  if (resizeTimer) window.clearTimeout(resizeTimer)
  observer?.disconnect()
  terminal?.dispose()
  if (!closeSent) void api.terminals.close(props.sessionId).catch(() => undefined)
})
</script>

<template>
  <section class="host-terminal">
    <header>
      <div>
        <strong>{{ hostName }}</strong>
        <small>{{ state === 'connected' ? t('terminal.connected') : state === 'finished' ? t('terminal.finished') : state === 'reconnecting' ? t('terminal.reconnecting') : t('terminal.connecting') }}</small>
      </div>
      <button type="button" :title="t('terminal.close')" :aria-label="t('terminal.close')" @click="closeSession"><X :size="17" /></button>
    </header>
    <div ref="host" class="host-terminal__screen terminal-screen" @click="terminal?.focus()" />
    <form class="host-terminal__composer" @submit.prevent="submitPendingLine">
      <input v-model="pendingLine" type="text" autocomplete="off" autocapitalize="off" spellcheck="false" maxlength="8192" :placeholder="t('terminal.inputPlaceholder')" :disabled="state === 'finished'" />
      <button type="submit" :disabled="state === 'finished'">{{ t('terminal.send') }}</button>
    </form>
  </section>
</template>

<style scoped>
.host-terminal { display:grid; grid-template-rows:auto minmax(420px,1fr) auto; min-height:0; overflow:hidden; border:1px solid var(--terminal-shell-border,#29383a); border-radius:var(--terminal-shell-radius,12px); background:var(--terminal-shell-background,#0b1214); box-shadow:var(--terminal-shell-shadow); }
.host-terminal header { display:flex; align-items:center; justify-content:space-between; gap:16px; padding:11px 14px; border-bottom:1px solid var(--terminal-shell-border,#29383a); background:var(--terminal-shell-panel,#111a1d); }
.host-terminal header div { display:grid; gap:2px; }
.host-terminal header strong { color:#f2faf7; font-size:13px; }
.host-terminal header small { color:var(--terminal-shell-muted,#8a9695); font-size:11px; }
.host-terminal header button { display:grid; width:34px; height:34px; place-items:center; border:1px solid var(--terminal-shell-border,#29383a); border-radius:9px; color:var(--terminal-shell-muted,#8a9695); background:transparent; }
.host-terminal__screen { min-width:0; min-height:420px; padding:10px 7px; }
.host-terminal__composer { display:grid; grid-template-columns:1fr auto; gap:8px; padding:9px 10px; border-top:1px solid var(--terminal-shell-border,#29383a); background:var(--terminal-shell-panel,#111a1d); }
.host-terminal__composer input { min-width:0; border:1px solid var(--terminal-shell-border,#29383a); border-radius:8px; padding:9px 11px; color:var(--terminal-shell-text,#d8dddc); background:var(--terminal-shell-background,#0b1214); font:12px ui-monospace,SFMono-Regular,Menlo,Consolas,monospace; }
.host-terminal__composer button { border:0; border-radius:8px; padding:0 16px; color:#05251c; background:var(--brand,#35cba6); font-weight:800; }
.host-terminal :deep(.xterm-viewport) { scrollbar-color:var(--terminal-shell-scrollbar,#35474a) var(--terminal-shell-background,#0b1214); }
@media (max-width: 760px) { .host-terminal { grid-template-rows:auto minmax(360px,60vh) auto; } .host-terminal__screen { min-height:360px; } }
</style>
