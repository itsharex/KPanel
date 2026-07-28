import { describe, expect, it } from 'vitest'
import { appAccessURL, matchingAppProxySites } from './appAccess'
import type { AppMarketItem, Site } from '@/types/api'

function app(accessMode: AppMarketItem['runtime']['accessMode'] = 'direct'): AppMarketItem {
  return {
    id: 'builtin-test',
    source: 'builtin',
    token: 'test',
    name_zh: '测试应用',
    name_en: 'Test App',
    desc_zh: '',
    desc_en: '',
    cat: 'test',
    icon: '',
    iconSha256: '',
    slug: 'test',
    installer: 'kejilion',
    runtime: {
      installed: true,
      state: 'running',
      ports: [{ privatePort: 3000, publicPort: 18080, type: 'tcp' }],
      accessMode,
      updateStatus: 'current',
      detectedBy: ['docker'],
    },
    capabilities: {},
  }
}

function site(overrides: Partial<Site> = {}): Site {
  return {
    id: 'app.example.com',
    primaryDomain: 'app.example.com',
    domains: ['app.example.com'],
    type: 'proxy',
    enabled: true,
    health: 'healthy',
    consistency: 'synced',
    access: 'managed',
    source: 'panel',
    upstream: 'http://127.0.0.1:18080',
    resourceVersion: 'version',
    ...overrides,
  }
}

describe('application access URL', () => {
  it('prefers an HTTPS domain with a usable certificate', () => {
    expect(
      appAccessURL(
        app(),
        [site({ certificate: { status: 'valid' } })],
        '192.0.2.10',
      ),
    ).toBe('https://app.example.com')
  })

  it('prefers an HTTP domain when no usable certificate is present', () => {
    expect(appAccessURL(app(), [site()], '192.0.2.10')).toBe('http://app.example.com')
  })

  it('keeps a disabled matching domain associated but falls back to the direct port', () => {
    const unavailable = site({ enabled: false })
    expect(matchingAppProxySites(app(), [unavailable])).toEqual([unavailable])
    expect(appAccessURL(app(), [unavailable], '192.0.2.10')).toBe('http://192.0.2.10:18080')
  })

  it('never combines the KPanel reverse-proxy hostname with an application port', () => {
    expect(appAccessURL(app(), [], 'kpanel.example.com')).toBe('')
  })

  it('formats a source IPv6 address for direct access', () => {
    expect(appAccessURL(app(), [], '2001:db8::10')).toBe('http://[2001:db8::10]:18080')
  })

  it.each([
    'http://127.0.0.1:18080/',
    'http://localhost:18080',
    'http://[::1]:18080',
    '127.0.0.1:18080',
  ])('recognizes the equivalent loopback upstream %s', (upstream) => {
    expect(matchingAppProxySites(app(), [site({ upstream })])).toHaveLength(1)
  })

  it('does not expose a direct URL in domain-only mode without a matching domain', () => {
    expect(appAccessURL(app('domain_only'), [], '192.0.2.10')).toBe('')
  })
})
