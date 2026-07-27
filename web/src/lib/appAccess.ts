import type { AppMarketItem, Site } from '@/types/api'

function appPublicPort(item: AppMarketItem): number | undefined {
  return item.runtime.ports.find((entry) => entry.type === 'tcp' && entry.publicPort)?.publicPort
}

export function matchingAppProxySites(item: AppMarketItem, sites: Site[]): Site[] {
  const port = appPublicPort(item)
  if (!port) return []

  const targets = new Set([`http://127.0.0.1:${port}`, `http://localhost:${port}`])
  return sites.filter(
    (site) => site.enabled && site.type === 'proxy' && targets.has(site.upstream || ''),
  )
}

export function appAccessURL(item: AppMarketItem, sites: Site[], panelHostname: string): string {
  const domain = matchingAppProxySites(item, sites)[0]
  if (domain) {
    const secure = domain.certificate?.status === 'valid' || domain.certificate?.status === 'expiring'
    return `${secure ? 'https' : 'http'}://${domain.primaryDomain}`
  }

  const port = appPublicPort(item)
  if (!port || item.runtime.accessMode === 'domain_only') return ''
  return `http://${panelHostname}:${port}`
}
