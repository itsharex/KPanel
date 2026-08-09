import { describe, expect, it } from 'vitest'
import { resolveEmbeddedBrowserTarget } from './embeddedBrowser'

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
})
