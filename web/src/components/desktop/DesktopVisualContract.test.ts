import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const styles = readFileSync(new URL('../../styles/desktop.css', import.meta.url), 'utf8')
const windowSource = readFileSync(new URL('./DesktopWindow.vue', import.meta.url), 'utf8')
const browserSource = readFileSync(new URL('../../views/WebBrowserView.vue', import.meta.url), 'utf8')

describe('desktop visual and interaction contract', () => {
  it('keeps desktop chrome, windows, fullscreen views and teleports in a stable layer order', () => {
    expect(styles).toMatch(/\.desktop\s*\{[^}]*z-index:\s*1000;/)
    expect(styles).toContain('z-index: 1200;')
    expect(styles).toContain('z-index: 2800 !important;')
    expect(styles).toContain('z-index: 5000 !important;')
    expect(styles).toContain('z-index: 5200 !important;')
  })

  it('removes the root scrollbar gutter while desktop mode owns the viewport', () => {
    expect(styles).toMatch(/html\.desktop-mode-open\s*\{[^}]*scrollbar-gutter:\s*auto;[^}]*scrollbar-width:\s*none;/)
    expect(styles).toContain('html.desktop-mode-open::-webkit-scrollbar')
    expect(styles).toMatch(/\.desktop\s*\{[^}]*width:\s*100vw;[^}]*height:\s*100vh;[^}]*height:\s*100dvh;/)
  })

  it('lets every supported in-window fullscreen surface escape window clipping', () => {
    expect(styles).toContain('.desktop-window:has(.terminal-stage.is-fullscreen)')
    expect(styles).toContain('.desktop-window:has(.interactive-terminal.is-fullscreen)')
    expect(styles).toContain('.desktop-window:has(.diagnostic-workbench.is-fullscreen)')
    expect(styles).toContain('.desktop-window__body:has(.interactive-terminal.is-fullscreen)')
  })

  it('keeps mobile CSS geometry aligned with the TypeScript work area', () => {
    expect(styles).toContain('min-width: min(320px, calc(100vw - 48px));')
    expect(styles).toContain('min-height: min(220px, calc(100vh - 88px));')
  })

  it('uses Windows-style desktop selection, controls and bottom taskbar', () => {
    expect(styles).toContain('.desktop__icon--selected')
    expect(styles).toContain('.desktop-window__action--close:hover')
    expect(styles).toContain('grid-template-columns: minmax(150px, 1fr) auto minmax(150px, 1fr);')
    expect(styles).toMatch(/\.desktop__taskbar-brand \.brand__mark\s*\{[^}]*width:\s*36px;[^}]*height:\s*36px;/)
    expect(styles).toContain('.desktop__taskbar-brand > span,')
    expect(windowSource.indexOf('desktop-window__action--minimize')).toBeLessThan(
      windowSource.indexOf('desktop-window__action--close'),
    )
  })

  it('keeps desktop icons and labels crisp in both color themes', () => {
    expect(styles).toMatch(/\.desktop__icon-glyph--dynamic::before\s*\{[^}]*display:\s*none;/)
    expect(styles).toMatch(/\.desktop__icon-label\s*\{[^}]*font-size:\s*11px;/)
    expect(styles).toMatch(/:root:not\(\[data-theme='dark'\]\) \.desktop__icon-label\s*\{[^}]*text-shadow:\s*none;/)
  })

  it('preserves wallpaper depth in light mode without changing the dark treatment', () => {
    expect(styles).toContain('linear-gradient(145deg, rgb(226 242 239 / 18%), rgb(190 218 224 / 7%))')
    expect(styles).toMatch(/:root\[data-theme='dark'\] \.desktop__wallpaper::after\s*\{/)
    expect(styles).toMatch(/:root\[data-theme='dark'\] \.desktop__aurora\s*\{[^}]*opacity:\s*\.2;/)
  })

  it('uses the shared window surface, border and shadow tokens in the browser', () => {
    expect(browserSource).toMatch(/\.embedded-browser\s*\{[^}]*background:\s*var\(--bg\);/)
    expect(browserSource).toMatch(/\.embedded-browser__shortcuts button\s*\{[^}]*border:\s*1px solid var\(--border\);[^}]*box-shadow:\s*var\(--shadow-sm\);/)
    expect(browserSource).toMatch(/\.embedded-browser__start-form\s*\{[^}]*box-shadow:\s*var\(--shadow-sm\);/)
  })

  it('does not reserve a light outer scrollbar gutter around script terminals', () => {
    expect(styles).toMatch(/\.desktop-window__body:has\(> \.app-script-page\)\s*\{[^}]*overflow:\s*hidden;[^}]*background:\s*var\(--terminal-shell-background\);[^}]*scrollbar-gutter:\s*auto;/)
  })

  it('keeps focused, minimized and closing window states keyboard-safe', () => {
    expect(windowSource).toContain('tabindex="-1"')
    expect(windowSource).toContain(':inert="windowState.minimized || closing || undefined"')
    expect(windowSource).toContain(':aria-hidden="windowState.minimized || closing"')
    expect(windowSource).toContain('element.focus({ preventScroll: true })')
  })
})
