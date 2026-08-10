import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const diagnosticsSource = readFileSync(new URL('./DiagnosticsView.vue', import.meta.url), 'utf8')
const desktopStyles = readFileSync(new URL('../styles/desktop.css', import.meta.url), 'utf8')

describe('diagnostics workspace layout', () => {
  it('keeps duplicate refresh and fullscreen controls out of the terminal bar', () => {
    expect(diagnosticsSource).not.toContain('aria-label="刷新体检命令"')
    expect(diagnosticsSource).not.toContain('class="diagnostic-fullscreen-toggle"')
    expect(diagnosticsSource).not.toContain('diagnostic-fullscreen-open')
  })

  it('contains scroll chaining inside the diagnostic log', () => {
    expect(diagnosticsSource).toContain('@wheel="containLogWheel"')
    expect(diagnosticsSource).toMatch(
      /\.diagnostic-log\s*\{[^}]*overflow: auto;[^}]*overscroll-behavior: contain;/,
    )
  })

  it('keeps the command list compact with a per-command run action and category color', () => {
    expect(diagnosticsSource).not.toContain('class="diagnostic-tabs"')
    expect(diagnosticsSource).toContain('class="diagnostic-command-group"')
    expect(diagnosticsSource).toContain('class="diagnostic-command-row"')
    expect(diagnosticsSource).toContain('class="diagnostic-command-run"')
    expect(diagnosticsSource).toContain('@click="requestCheck(check)"')
    expect(diagnosticsSource).not.toContain('{{ categoryName(check.category) }} · 约')
    expect(diagnosticsSource).toMatch(/\.diagnostic-command-row\.is-category-access,[^}]*--diagnostic-category:/)
    expect(diagnosticsSource).toMatch(/\.diagnostic-command-row\.is-category-network,[^}]*--diagnostic-category:/)
    expect(diagnosticsSource).toMatch(/\.diagnostic-command-row\.is-category-hardware,[^}]*--diagnostic-category:/)
    expect(diagnosticsSource).toMatch(/\.diagnostic-command-row\.is-category-comprehensive,[^}]*--diagnostic-category:/)
    expect(diagnosticsSource).toMatch(
      /\.diagnostic-command-group \+ \.diagnostic-command-group\s*\{[^}]*border-top: 1px dashed/,
    )
  })

  it('collapses commands into a persistent icon rail', () => {
    expect(diagnosticsSource).toContain("'is-command-panel-collapsed': commandsCollapsed")
    expect(diagnosticsSource).toContain('aria-controls="diagnostic-command-selector"')
    expect(diagnosticsSource).toContain('class="diagnostic-command-rail"')
    expect(diagnosticsSource).toContain('class="diagnostic-command-rail__item"')
    expect(diagnosticsSource).toContain(':title="checkNameLabel(check.name)"')
    expect(diagnosticsSource).toMatch(
      /\.diagnostic-workbench\.is-command-panel-collapsed\s*\{[^}]*grid-template-columns:\s*52px minmax\(0, 1fr\);/,
    )
  })

  it('uses darker category accents in light mode and brighter accents in dark mode', () => {
    expect(diagnosticsSource).toContain('--diagnostic-category: #087a72;')
    expect(diagnosticsSource).toContain('--diagnostic-category: #2563c4;')
    expect(diagnosticsSource).toContain('--diagnostic-category: #965900;')
    expect(diagnosticsSource).toContain('--diagnostic-category: #7546c8;')
    expect(diagnosticsSource).toContain(":global(:root[data-theme='dark'] .diagnostic-command-group.is-category-access)")
    expect(diagnosticsSource).toContain('--diagnostic-category: #4ecdc4;')
  })

  it('does not stretch the history row away from the workbench inside a desktop window', () => {
    expect(desktopStyles).toMatch(
      /\.desktop-window__body > \.diagnostics-page\s*\{[^}]*height:\s*100%;[^}]*min-height:\s*0;[^}]*overflow-y:\s*auto;[^}]*align-content:\s*start;/,
    )
  })

  it('keeps the whole diagnostic terminal visible before the history scroll region', () => {
    expect(desktopStyles).toMatch(
      /\.desktop-window__body:has\(> \.terminal-page\),[\s\S]*?\.desktop-window__body:has\(> \.diagnostics-page\)\s*\{[^}]*overflow:\s*hidden;/,
    )
    expect(desktopStyles).toMatch(
      /\.desktop-window__body \.diagnostic-workbench\s*\{[^}]*height:\s*calc\(100% - 86px\) !important;[^}]*min-height:\s*0 !important;/,
    )
  })
})
