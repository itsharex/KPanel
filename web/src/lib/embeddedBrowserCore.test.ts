import { describe, expect, it } from 'vitest'
import {
  createBrowserCoreCommandMessage,
  createBrowserCoreNavigateMessage,
  createBrowserCoreUpdateSessionMessage,
  isBrowserCoreEvent,
  resolveBrowserCoreLocation,
} from './embeddedBrowserCore'

const session = {
  mode: 'beta' as const,
  relayUrl: 'https://browser.example.com',
  token: 'signed-token',
  sessionId: 'session-1',
  expiresAt: '2030-01-01T00:00:00Z',
}

describe('embeddedBrowserCore', () => {
  it('always resolves the isolated kernel page instead of a target URL', () => {
    expect(resolveBrowserCoreLocation(session)).toEqual({
      frameURL: 'https://browser.example.com/kernel/',
      origin: 'https://browser.example.com',
      targetOrigin: 'https://browser.example.com',
      sandbox: 'allow-same-origin allow-scripts allow-forms allow-modals allow-popups allow-downloads',
    })
  })

  it('resolves reader mode to a same-origin opaque sandbox', () => {
    expect(resolveBrowserCoreLocation({ ...session, mode: 'reader' })).toEqual({
      frameURL: 'https://browser.example.com/browser-reader/',
      origin: 'null',
      targetOrigin: '*',
      sandbox: 'allow-scripts',
    })
  })

  it('rejects a relay URL that is not an isolated origin', () => {
    expect(resolveBrowserCoreLocation({ ...session, relayUrl: 'https://browser.example.com/base' }))
      .toBeUndefined()
    expect(resolveBrowserCoreLocation({ ...session, relayUrl: 'https://user@browser.example.com' }))
      .toBeUndefined()
  })

  it('fails closed when the API returns an unknown mode', () => {
    expect(resolveBrowserCoreLocation({ ...session, mode: 'disabled' as 'reader' })).toBeUndefined()
  })

  it('keeps the target and token in a postMessage payload', () => {
    expect(createBrowserCoreNavigateMessage(session, 'https://target.example/path', 'tab-1:7')).toEqual({
      type: 'kpanel-browser:navigate',
      token: 'signed-token',
      url: 'https://target.example/path',
      navigationId: 'tab-1:7',
    })
  })

  it('refreshes the relay token without forcing a page navigation', () => {
    expect(createBrowserCoreUpdateSessionMessage(session)).toEqual({
      type: 'kpanel-browser:update-session',
      token: 'signed-token',
    })
  })

  it('creates bounded browser history commands', () => {
    expect(createBrowserCoreCommandMessage('back', 'tab-1:8')).toEqual({
      type: 'kpanel-browser:command',
      command: 'back',
      navigationId: 'tab-1:8',
    })
  })

  it('accepts only bounded known kernel events', () => {
    expect(isBrowserCoreEvent({ type: 'kpanel-browser:ready' })).toBe(true)
    expect(isBrowserCoreEvent({
      type: 'kpanel-browser:navigation',
      url: 'https://example.com',
      navigationId: 'tab-1:7',
    })).toBe(true)
    expect(isBrowserCoreEvent({
      type: 'kpanel-browser:navigation',
      url: 'https://example.com',
    })).toBe(false)
    expect(isBrowserCoreEvent({
      type: 'kpanel-browser:navigation',
      url: 1,
      navigationId: 'tab-1:7',
    })).toBe(false)
    expect(isBrowserCoreEvent({ type: 'kpanel-browser:unknown' })).toBe(false)
  })
})
