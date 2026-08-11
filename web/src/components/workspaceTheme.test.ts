import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const terminalSource = readFileSync(
  new URL('./apps/AppInteractiveTerminal.vue', import.meta.url),
  'utf8',
)
const editorSource = readFileSync(
  new URL('./files/CodeEditor.vue', import.meta.url),
  'utf8',
)
const diagnosticsSource = readFileSync(
  new URL('../views/DiagnosticsView.vue', import.meta.url),
  'utf8',
)
const terminalViewSource = readFileSync(
  new URL('../views/TerminalView.vue', import.meta.url),
  'utf8',
)
const hostTerminalSource = readFileSync(
  new URL('./terminal/HostTerminal.vue', import.meta.url),
  'utf8',
)
const filesSource = readFileSync(
  new URL('../views/FilesView.vue', import.meta.url),
  'utf8',
)
const dockerSource = readFileSync(
  new URL('../views/DockerView.vue', import.meta.url),
  'utf8',
)
const appsSource = readFileSync(
  new URL('../views/AppsView.vue', import.meta.url),
  'utf8',
)
const globalThemeSource = readFileSync(
  new URL('../styles/main.css', import.meta.url),
  'utf8',
)

describe('terminal and editor workspace theme', () => {
  it('keeps terminal surfaces neutral while reserving KPanel brand tokens for interaction', () => {
    expect(globalThemeSource).toContain('--terminal-shell-background: #0b1214')
    expect(globalThemeSource).toContain('--terminal-shell-radius: 12px')
    expect(terminalSource).toContain('--terminal-background: var(--terminal-shell-background, #0b1214)')
    expect(terminalSource).toContain('--terminal-accent: var(--brand, #35cba6)')
    expect(terminalSource).toContain("terminalThemeColor('--terminal-accent'")
    expect(terminalSource).toContain("terminalThemeColor('--terminal-ansi-green', '#91b56d')")
    expect(terminalSource).toContain("terminalThemeColor('--terminal-ansi-cyan', '#72aaa7')")
    expect(terminalSource).not.toContain("green: '#35cba6'")
    expect(terminalSource).not.toContain("cyan: '#5adaba'")
    expect(terminalSource).not.toContain('#6d5dfc')
    expect(terminalSource).not.toContain('#8b7cff')
  })

  it('uses the same terminal surface before and after an interactive diagnostic starts', () => {
    expect(diagnosticsSource).toContain('background: var(--terminal-shell-background, #0b1214)')
    expect(diagnosticsSource).toContain('background: var(--terminal-shell-panel, #111a1d)')
    expect(terminalViewSource).toContain('background:var(--terminal-shell-background,#0b1214)')
    expect(hostTerminalSource).toContain('background:var(--terminal-shell-background,#0b1214)')
    expect(terminalSource).toContain('--terminal-background: var(--terminal-shell-background, #0b1214)')
  })

  it('uses one classic-mode height for terminal and diagnostics workspaces', () => {
    expect(globalThemeSource).toContain('--terminal-workspace-height: clamp(620px, calc(100dvh - 190px), 760px)')
    expect(globalThemeSource).toContain('--terminal-workspace-min-height: 620px')
    expect(globalThemeSource).toContain('--terminal-workspace-radius: var(--radius-lg)')
    expect(terminalViewSource).toContain('height:var(--terminal-workspace-height)')
    expect(terminalViewSource).toContain('min-height:var(--terminal-workspace-min-height)')
    expect(terminalViewSource).toContain('border-radius:var(--terminal-workspace-radius)')
    expect(diagnosticsSource).toContain('height: var(--terminal-workspace-height)')
    expect(diagnosticsSource).toContain('min-height: var(--terminal-workspace-min-height)')
    expect(diagnosticsSource).toContain('border-radius: var(--terminal-workspace-radius)')
  })

  it('keeps the editor dark while using KPanel brand and semantic tokens for controls', () => {
    expect(editorSource).toContain('--code-background: var(--terminal-shell-background, #0b1214)')
    expect(editorSource).toContain('--code-caret: var(--brand, #35cba6)')
    expect(editorSource).toContain('color: var(--danger, #ef7a7a)')
    expect(editorSource).not.toContain('#409be8')
    expect(editorSource).not.toContain('#31415b')
  })

  it('uses the shared workspace palette for editor chrome and non-interactive logs', () => {
    for (const source of [filesSource, dockerSource, appsSource]) {
      expect(source).toContain('var(--terminal-shell-background, #0b1214)')
      expect(source).toContain('var(--terminal-shell-text, #d8dddc)')
    }
    expect(filesSource).not.toContain('background: #111a2c')
    expect(dockerSource).not.toContain('background: #0b1020')
    expect(appsSource).not.toContain('background: #111827')
  })

  it('uses one border radius and edge treatment across dark workspaces', () => {
    for (const source of [terminalSource, filesSource, dockerSource, appsSource]) {
      expect(source).toContain('var(--terminal-shell-radius, 12px)')
      expect(source).toContain('var(--terminal-shell-shadow, inset 0 1px 0 rgb(255 255 255 / 3%))')
    }
    expect(globalThemeSource).toContain('border-radius: var(--terminal-shell-radius)')
    expect(globalThemeSource).toContain('box-shadow: var(--terminal-shell-shadow)')
  })
})
