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

  it('collapses the host selector into a persistent narrow rail', () => {
    expect(terminalSource).toContain("'is-connections-collapsed': connectionsCollapsed")
    expect(terminalSource).toContain('aria-controls="terminal-connection-selector"')
    expect(terminalSource).toContain("'terminal.expandConnections'")
    expect(terminalSource).toContain("'terminal.collapseConnections'")
    expect(terminalSource).toContain('terminal-connections__toggle terminal-connections__refresh')
    expect(terminalSource).toContain('class="terminal-connections__rail"')
    expect(terminalSource).toContain('class="terminal-host-rail"')
    expect(terminalSource).toContain(':title="`${host.name} · ${hostStateLabel(host)}`"')
    expect(terminalSource).toMatch(
      /\.terminal-workspace\.is-connections-collapsed\s*\{[^}]*grid-template-columns:52px minmax\(0,1fr\);/,
    )
    expect(terminalSource).toMatch(
      /\.terminal-workspace\.is-connections-collapsed \.terminal-connections__heading,[^}]*\.terminal-connections__refresh\s*\{\s*display:none;/,
    )
  })

  it('reserves the remaining stage height for the terminal and composer', () => {
    expect(terminalSource).toMatch(
      /\.terminal-stage\s*\{[^}]*grid-template-rows:auto minmax\(0,1fr\);[^}]*min-height:0;/,
    )
  })

  it('keeps the mobile host terminal in two rows so the composer cannot overflow', () => {
    expect(hostTerminalSource).toMatch(
      /\.host-terminal\s*\{[^}]*grid-template-rows:minmax\(0,1fr\) auto;[^}]*min-height:0;[^}]*overflow:hidden;/,
    )
    expect(hostTerminalSource).not.toMatch(
      /@media \(max-width: 760px\)[\s\S]*?\.host-terminal\s*\{[^}]*grid-template-rows:auto minmax\(0,1fr\) auto;/,
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

  it('keeps session tabs and terminal actions in one dark toolbar row', () => {
    expect(terminalSource).toContain('class="terminal-tab__status"')
    expect(terminalSource).toContain('@state-change="item.state = $event"')
    expect(terminalSource).toContain('class="terminal-tabs-bar"')
    expect(terminalSource).toContain('<TerminalToolbar')
    expect(terminalSource).toMatch(
      /\.terminal-tabs-bar\s*\{[^}]*display:flex;[^}]*background:var\(--terminal-shell-panel/,
    )
  })

  it('fills the whole terminal stage so tabs remain switchable', () => {
    expect(terminalSource).not.toContain('terminal-fullscreen-toggle')
    expect(terminalSource).toContain("'is-fullscreen': workspaceFullscreen")
    expect(terminalSource).toContain('@click="selectSession(item.id)"')
    expect(terminalSource).toMatch(
      /\.terminal-stage\.is-fullscreen\s*\{[^}]*position:fixed;[^}]*inset:0;[^}]*height:100dvh;/,
    )
    expect(hostTerminalSource).toContain('defineExpose({ scrollToTop, scheduleResize })')
  })
})
