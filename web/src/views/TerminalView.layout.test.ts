import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const terminalSource = readFileSync(new URL('./TerminalView.vue', import.meta.url), 'utf8')
const hostTerminalSource = readFileSync(new URL('../components/terminal/HostTerminal.vue', import.meta.url), 'utf8')

describe('multi-host terminal workspace layout', () => {
  it('keeps a large connection inventory in its own scroll region', () => {
    expect(terminalSource).toContain('class="terminal-connections__list"')
    expect(terminalSource).toMatch(
      /\.terminal-connections\s*\{[^}]*display:grid;[^}]*min-height:0;[^}]*overflow:hidden;/,
    )
    expect(terminalSource).toMatch(
      /\.terminal-connections__list\s*\{[^}]*min-height:0;[^}]*overflow-y:auto;/,
    )
  })

  it('reserves the remaining stage height for the terminal and composer', () => {
    expect(terminalSource).toMatch(
      /\.terminal-stage\s*\{[^}]*grid-template-rows:auto minmax\(0,1fr\);[^}]*min-height:0;/,
    )
  })

  it('contains wheel scrolling inside the host terminal viewport', () => {
    expect(hostTerminalSource).toContain('@wheel="containTerminalWheel"')
    expect(hostTerminalSource).toMatch(
      /\.host-terminal__screen :deep\(\.xterm-viewport\)\s*\{[^}]*overflow-y:scroll !important;[^}]*overscroll-behavior:contain;/,
    )
  })

  it('uses the terminal clipboard menu instead of the browser context menu', () => {
    expect(hostTerminalSource).toContain('@contextmenu="clipboardMenu?.open($event)"')
    expect(hostTerminalSource).toContain('@paste.capture="clipboardMenu?.handlePaste($event)"')
    expect(hostTerminalSource).toContain('terminal.attachCustomKeyEventHandler')
  })

  it('merges connection status into the session tabs without a duplicate terminal header', () => {
    expect(terminalSource).toContain('class="terminal-tab__status"')
    expect(terminalSource).toContain('@state-change="item.state = $event"')
    expect(hostTerminalSource).not.toContain('<header>')
    expect(hostTerminalSource).toContain('class="host-terminal__toolbar"')
  })

  it('moves fullscreen ownership into the shared terminal controls', () => {
    expect(terminalSource).not.toContain('terminal-fullscreen-toggle')
    expect(hostTerminalSource).toContain('<TerminalToolbar')
    expect(hostTerminalSource).toContain("'is-fullscreen': fullscreen")
    expect(hostTerminalSource).toMatch(
      /\.host-terminal\.is-fullscreen\s*\{[^}]*position:fixed;[^}]*inset:0;[^}]*height:100dvh;/,
    )
  })
})
