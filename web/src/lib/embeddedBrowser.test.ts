import { describe, expect, it } from 'vitest'
import {
  MAX_EMBEDDED_BROWSER_TABS,
  MAX_LIVE_EMBEDDED_BROWSER_TABS,
  resolveEmbeddedBrowserInput,
  resolveEmbeddedBrowserTarget,
} from './embeddedBrowser'

describe('embedded browser target', () => {
  it('normalizes ordinary configured website URLs', () => {
    expect(resolveEmbeddedBrowserTarget('https://blog.example.com/path', 'https:')).toEqual({
      href: 'https://blog.example.com/path',
      hostname: 'blog.example.com',
      mixedContent: false,
    })
  })

  it('marks HTTP targets as blocked inside an HTTPS panel', () => {
    expect(resolveEmbeddedBrowserTarget('http://blog.example.com', 'https:')?.mixedContent).toBe(true)
    expect(resolveEmbeddedBrowserTarget('http://blog.example.com', 'http:')?.mixedContent).toBe(false)
  })

  it('rejects active schemes, credentials, malformed values, and oversized input', () => {
    expect(resolveEmbeddedBrowserTarget('javascript:alert(1)')).toBeUndefined()
    expect(resolveEmbeddedBrowserTarget('data:text/html,test')).toBeUndefined()
    expect(resolveEmbeddedBrowserTarget('https://user:secret@example.com')).toBeUndefined()
    expect(resolveEmbeddedBrowserTarget('not a URL')).toBeUndefined()
    expect(resolveEmbeddedBrowserTarget(`https://example.com/${'a'.repeat(2_048)}`)).toBeUndefined()
  })

  it('turns a bare hostname into an HTTPS target without accepting active schemes', () => {
    expect(resolveEmbeddedBrowserInput('  example.com/path  ', 'https:')?.href).toBe(
      'https://example.com/path',
    )
    expect(resolveEmbeddedBrowserInput('http://example.com', 'https:')?.mixedContent).toBe(true)
    expect(resolveEmbeddedBrowserInput('javascript:alert(1)')).toBeUndefined()
  })

  it('keeps tab and live-frame resource limits small and explicit', () => {
    expect(MAX_EMBEDDED_BROWSER_TABS).toBe(8)
    expect(MAX_LIVE_EMBEDDED_BROWSER_TABS).toBeLessThan(MAX_EMBEDDED_BROWSER_TABS)
  })
})
