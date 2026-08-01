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
const globalThemeSource = readFileSync(
  new URL('../styles/main.css', import.meta.url),
  'utf8',
)

describe('terminal and editor workspace theme', () => {
  it('keeps terminal surfaces neutral while reserving KPanel brand tokens for interaction', () => {
    expect(globalThemeSource).toContain('--terminal-shell-background: #0b1214')
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
  })

  it('keeps the editor dark while using KPanel brand and semantic tokens for controls', () => {
    expect(editorSource).toContain('--code-background: #071411')
    expect(editorSource).toContain('--code-caret: var(--brand, #35cba6)')
    expect(editorSource).toContain('color: var(--danger, #ef7a7a)')
    expect(editorSource).not.toContain('#409be8')
    expect(editorSource).not.toContain('#31415b')
  })
})
