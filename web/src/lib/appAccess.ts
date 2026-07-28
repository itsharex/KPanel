import type { AppMarketItem, Site } from '@/types/api'

function appPublicPort(item: AppMarketItem): number | undefined {
  return item.runtime.ports.find((entry) => entry.type === 'tcp' && entry.publicPort)?.publicPort
}

function proxyUpstreamPort(upstream: string): number | undefined {
  const value = upstream.trim()
  if (!value) return undefined

  try {
    const parsed = new URL(/^[a-z][a-z\d+.-]*:\/\//i.test(value) ? value : `http://${value}`)
    const hostname = parsed.hostname.toLowerCase().replace(/^\[|\]$/g, '')
    if (!['127.0.0.1', 'localhost', '::1'].includes(hostname)) return undefined
    if (parsed.pathname !== '/' || parsed.search || parsed.hash || parsed.username || parsed.password) {
      return undefined
    }
    const port = Number(parsed.port || (parsed.protocol === 'https:' ? 443 : 80))
    return Number.isInteger(port) && port > 0 && port <= 65535 ? port : undefined
  } catch {
    return undefined
  }
}

export function matchingAppProxySites(item: AppMarketItem, sites: Site[]): Site[] {
  const port = appPublicPort(item)
  if (!port) return []

  return sites.filter(
    (site) => site.type === 'proxy' && proxyUpstreamPort(site.upstream || '') === port,
  )
}

export function appAccessURL(item: AppMarketItem, sites: Site[], panelHostname: string): string {
  const domain = matchingAppProxySites(item, sites).find((site) => site.enabled)
  if (domain) {
    const secure = domain.certificate?.status === 'valid' || domain.certificate?.status === 'expiring'
    return `${secure ? 'https' : 'http'}://${domain.primaryDomain}`
  }

  const port = appPublicPort(item)
  if (!port || item.runtime.accessMode === 'domain_only') return ''
  return `http://${panelHostname}:${port}`
}
