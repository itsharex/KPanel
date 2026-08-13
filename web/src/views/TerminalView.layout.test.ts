import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const terminalSource = readFileSync(new URL('./TerminalView.vue', import.meta.url), 'utf8')
const hostTerminalSource = readFileSync(new URL('../components/terminal/HostTerminal.vue', import.meta.url), 'utf8')
const desktopStyles = readFileSync(new URL('../styles/desktop.css', import.meta.url), 'utf8')

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
      /\.terminal-stage\s*\{[^}]*grid-template-rows:auto auto minmax\(0,1fr\);[^}]*min-height:0;[^}]*overflow:hidden;/,
    )
  })

  it('uses an overlay drawer for host selection on mobile', () => {
    expect(terminalSource).toContain("'is-connections-drawer-open': mobileConnectionsOpen")
    expect(terminalSource).toContain('class="terminal-connections-overlay"')
    expect(terminalSource).toContain('aria-label="打开主机选择"')
    expect(terminalSource).toContain('aria-label="关闭主机选择"')
    expect(terminalSource).toContain('mobileConnectionsOpen.value = false')
    expect(terminalSource).toMatch(
      /@media \(max-width: 900px\)[\s\S]*?\.terminal-connections\s*\{[^}]*position:absolute;[^}]*transform:translateX\(-105%\);/,
    )
    expect(terminalSource).toMatch(
      /\.terminal-workspace\.is-connections-drawer-open \.terminal-connections\s*\{[^}]*transform:translateX\(0\);/,
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

  it('fits the terminal workspace to the desktop window instead of scrolling the outer page', () => {
    expect(desktopStyles).toMatch(
      /\.desktop-window__body:has\(> \.terminal-page\),[\s\S]*?overflow:\s*hidden;[\s\S]*?scrollbar-gutter:\s*auto;/,
    )
    expect(desktopStyles).toMatch(
      /\.desktop-window__body \.terminal-page\s*\{[^}]*height:\s*100% !important;[^}]*min-height:\s*0 !important;[^}]*overflow:\s*hidden;/,
    )
    expect(desktopStyles).toMatch(
      /\.desktop-window__body \.terminal-workspace\s*\{[^}]*height:\s*auto !important;[^}]*min-height:\s*0 !important;[^}]*flex:\s*1 1 0;/,
    )
  })

  it('does not reserve an inline host-list row in mobile desktop mode', () => {
    expect(desktopStyles).toMatch(
      /@media \(max-width: 900px\)[\s\S]*?\.desktop-window__body \.terminal-workspace,[\s\S]*?\.desktop-window__body \.terminal-workspace\.is-connections-collapsed\s*\{[^}]*grid-template-rows:\s*minmax\(0, 1fr\) !important;/,
    )
    expect(desktopStyles).toMatch(
      /@media \(max-width: 900px\)[\s\S]*?\.desktop-window__body \.terminal-connections\s*\{[^}]*border-right:\s*1px solid var\(--border\);[^}]*border-bottom:\s*0;/,
    )
  })

  it('contains wheel scrolling inside the host terminal viewport', () => {
    expect(hostTerminalSource).toContain('@wheel="containTerminalWheel"')
    expect(hostTerminalSource).toMatch(
      /\.host-terminal__screen :deep\(\.xterm-viewport\)\s*\{[^}]*overflow-y:scroll !important;[^}]*overscroll-behavior:contain;/,
    )
    expect(hostTerminalSource).toMatch(
      /\.host-terminal__screen :deep\(\.xterm-scrollable-element\)\s*\{[^}]*overscroll-behavior:contain;/,
    )
  })

  it('maps mobile vertical swipes to terminal scrollback without moving the desktop window', () => {
    expect(hostTerminalSource).toContain('@touchstart="terminalTouchScroll.start"')
    expect(hostTerminalSource).toContain('@touchmove="terminalTouchScroll.move"')
    expect(hostTerminalSource).toContain('@touchend="terminalTouchScroll.end"')
    expect(hostTerminalSource).toMatch(
      /\.host-terminal__screen :deep\(\.xterm\)\s*\{[^}]*touch-action:none;/,
    )
  })

  it('focuses the shell cursor by default and leaves the composer user-activated', () => {
    expect(hostTerminalSource).toContain('window.requestAnimationFrame(focusTerminal)')
    expect(hostTerminalSource).not.toContain('composerInput')
    expect(terminalSource).toContain('focusActiveTerminal()')
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
    expect(hostTerminalSource).toContain('defineExpose({ focusTerminal, scrollToTop, scheduleResize })')
  })
})
