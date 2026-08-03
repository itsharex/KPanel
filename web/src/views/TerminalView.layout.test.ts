import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const terminalSource = readFileSync(new URL('./TerminalView.vue', import.meta.url), 'utf8')

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
})
