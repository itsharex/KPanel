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

describe('terminal and editor workspace theme', () => {
  it('keeps the terminal dark and derives interactive accents from KPanel brand tokens', () => {
    expect(terminalSource).toContain('--terminal-background: #071411')
    expect(terminalSource).toContain('--terminal-accent: var(--brand, #35cba6)')
    expect(terminalSource).toContain("terminalThemeColor('--terminal-accent'")
    expect(terminalSource).not.toContain('#6d5dfc')
    expect(terminalSource).not.toContain('#8b7cff')
  })

  it('keeps the editor dark while using KPanel brand and semantic tokens for controls', () => {
    expect(editorSource).toContain('--code-background: #071411')
    expect(editorSource).toContain('--code-caret: var(--brand, #35cba6)')
    expect(editorSource).toContain('color: var(--danger, #ef7a7a)')
    expect(editorSource).not.toContain('#409be8')
    expect(editorSource).not.toContain('#31415b')
  })
})
