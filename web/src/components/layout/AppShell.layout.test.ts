import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const styles = readFileSync(new URL('../../styles/main.css', import.meta.url), 'utf8')
const sitesSource = readFileSync(new URL('../../views/SitesView.vue', import.meta.url), 'utf8')
const jobsSource = readFileSync(new URL('../../views/JobsView.vue', import.meta.url), 'utf8')

describe('responsive application shell comfort', () => {
  it('keeps the mobile navigation usable on short screens', () => {
    expect(styles).toMatch(
      /\.sidebar__nav\s*\{[^}]*min-height:\s*0;[^}]*flex:\s*1 1 auto;[^}]*overflow-x:\s*hidden;[^}]*overflow-y:\s*auto;[^}]*overscroll-behavior:\s*contain;/,
    )
    expect(styles).toMatch(/@media \(max-width: 920px\)[\s\S]*?\.sidebar\s*\{[^}]*height:\s*100dvh;/)
  })

  it('uses the available phone width without reserving a desktop scrollbar gutter', () => {
    expect(styles).toMatch(/@media \(max-width: 920px\)[\s\S]*?html\s*\{[^}]*scrollbar-gutter:\s*auto;/)
    expect(styles).toContain('env(safe-area-inset-bottom)')
  })

  it('keeps monitoring cards compact without forcing a single long column', () => {
    expect(styles).toMatch(
      /@media \(min-width: 360px\) and \(max-width: 680px\)[\s\S]*?\.metric-grid,[\s\S]*?grid-template-columns:\s*repeat\(2, minmax\(0, 1fr\)\);/,
    )
    expect(styles).toMatch(/\.metric-grid > :last-child:nth-child\(odd\)[\s\S]*?grid-column:\s*1 \/ -1;/)
  })

  it('keeps the first table column visible during horizontal scrolling', () => {
    expect(styles).toMatch(
      /@media \(max-width: 680px\)[\s\S]*?\.data-table th:first-child,[\s\S]*?\.data-table td:first-child\s*\{[^}]*position:\s*sticky;[^}]*left:\s*0;/,
    )
  })

  it('raises authentication content and balances bottom-sheet actions on phones', () => {
    expect(styles).toMatch(
      /@media \(max-width: 680px\)[\s\S]*?\.auth-layout__form\s*\{[^}]*place-items:\s*start center;[^}]*padding:\s*clamp\(96px, 15vh, 132px\)/,
    )
    expect(styles).toMatch(/\.modal-panel__footer \.button\s*\{[^}]*flex:\s*1 1 0;/)
  })

  it('keeps mobile search and filters in a compact two-row toolbar', () => {
    expect(sitesSource).toContain('class="toolbar-card toolbar-card--search-tabs"')
    expect(jobsSource).toContain('class="toolbar-card toolbar-card--search-tabs"')
    expect(styles).toMatch(
      /\.toolbar-card--search-tabs\s*\{[^}]*display:\s*grid;[^}]*grid-template-columns:\s*minmax\(0, 1fr\) auto;/,
    )
  })

  it('avoids persistent blur and broad transitions in the shared shell', () => {
    expect(styles).not.toMatch(/\.topbar\s*\{[^}]*backdrop-filter:/)
    expect(styles).not.toContain('transition: all')
    expect(styles).toContain('touch-action: manipulation')
  })
})
