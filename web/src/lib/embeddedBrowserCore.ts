import type { BrowserCoreSession } from '@/types/api'

export const BROWSER_CORE_MESSAGE_PREFIX = 'kpanel-browser:'

export interface BrowserCoreNavigateMessage {
  type: 'kpanel-browser:navigate'
  token: string
  url: string
  navigationId: string
}

export interface BrowserCoreUpdateSessionMessage {
  type: 'kpanel-browser:update-session'
  token: string
}

export type BrowserCoreCommand = 'back' | 'forward' | 'reload'

export interface BrowserCoreCommandMessage {
  type: 'kpanel-browser:command'
  command: BrowserCoreCommand
  navigationId: string
}

export type BrowserCoreEvent =
  | { type: 'kpanel-browser:ready' }
  | { type: 'kpanel-browser:navigation'; url: string; navigationId: string }
  | { type: 'kpanel-browser:title'; title: string; navigationId: string }
  | { type: 'kpanel-browser:error'; message: string; navigationId: string }
  | { type: 'kpanel-browser:session-expired' }

export interface BrowserCoreLocation {
  frameURL: string
  origin: string
}

export function resolveBrowserCoreLocation(session: BrowserCoreSession): BrowserCoreLocation | undefined {
  if (session.mode !== 'beta') return undefined
  try {
    const relay = new URL(session.relayUrl)
    if ((relay.protocol !== 'http:' && relay.protocol !== 'https:') || relay.username || relay.password ||
      (relay.pathname !== '/' && relay.pathname !== '') || relay.search || relay.hash) return undefined
    return {
      frameURL: new URL('/kernel/', relay.origin).href,
      origin: relay.origin,
    }
  } catch {
    return undefined
  }
}

export function createBrowserCoreNavigateMessage(
  session: BrowserCoreSession,
  url: string,
  navigationId: string,
): BrowserCoreNavigateMessage {
  return {
    type: 'kpanel-browser:navigate',
    token: session.token,
    url,
    navigationId,
  }
}

export function createBrowserCoreUpdateSessionMessage(
  session: BrowserCoreSession,
): BrowserCoreUpdateSessionMessage {
  return {
    type: 'kpanel-browser:update-session',
    token: session.token,
  }
}

export function createBrowserCoreCommandMessage(
  command: BrowserCoreCommand,
  navigationId: string,
): BrowserCoreCommandMessage {
  return {
    type: 'kpanel-browser:command',
    command,
    navigationId,
  }
}

export function isBrowserCoreEvent(value: unknown): value is BrowserCoreEvent {
  if (!value || typeof value !== 'object') return false
  const message = value as Record<string, unknown>
  if (typeof message.type !== 'string' || !message.type.startsWith(BROWSER_CORE_MESSAGE_PREFIX)) {
    return false
  }
  switch (message.type) {
    case 'kpanel-browser:ready':
    case 'kpanel-browser:session-expired':
      return true
    case 'kpanel-browser:navigation':
      return validNavigationID(message.navigationId) &&
        typeof message.url === 'string' && message.url.length <= 2_048
    case 'kpanel-browser:title':
      return validNavigationID(message.navigationId) &&
        typeof message.title === 'string' && message.title.length <= 160
    case 'kpanel-browser:error':
      return validNavigationID(message.navigationId) &&
        typeof message.message === 'string' && message.message.length <= 512
    default:
      return false
  }
}

function validNavigationID(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0 && value.length <= 64
}
