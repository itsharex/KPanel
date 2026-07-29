import { afterEach, describe, expect, it, vi } from 'vitest'
import { openTerminalURL } from './terminalLinks'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('terminal URL links', () => {
  it('opens HTTP and HTTPS links in an isolated browser tab', () => {
    const open = vi.fn()
    vi.stubGlobal('window', { open })

    expect(openTerminalURL('https://example.com/path?q=1')).toBe(true)
    expect(openTerminalURL('http://127.0.0.1:8080/')).toBe(true)
    expect(open).toHaveBeenNthCalledWith(
      1,
      'https://example.com/path?q=1',
      '_blank',
      'noopener,noreferrer',
    )
    expect(open).toHaveBeenNthCalledWith(
      2,
      'http://127.0.0.1:8080/',
      '_blank',
      'noopener,noreferrer',
    )
  })

  it('rejects malformed links and non-web protocols', () => {
    const open = vi.fn()
    vi.stubGlobal('window', { open })

    expect(openTerminalURL('not-a-url')).toBe(false)
    expect(openTerminalURL('javascript:alert(1)')).toBe(false)
    expect(openTerminalURL('file:///etc/passwd')).toBe(false)
    expect(openTerminalURL('data:text/html,unsafe')).toBe(false)
    expect(open).not.toHaveBeenCalled()
  })
})
