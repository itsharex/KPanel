import { api } from '@/lib/api'
import { appAccessURL } from '@/lib/appAccess'
import type { AppMarketItem, PublicNetworkSummary, Site } from '@/types/api'

/**
 * Desktop dynamic entries: installed apps and configured sites surfaced as
 * desktop icons. Apps and sites that resolve to the same entry URL collapse to
 * a single app icon (the app entry wins), keeping the desktop uncluttered.
 */

export type DesktopEntryKind = 'app' | 'site'

export interface DesktopEntry {
  /** Stable key: `${kind}:${id}` so Vue can key icons reliably. */
  key: string
  kind: DesktopEntryKind
  id: string
  name: string
  /** External entry URL opened when the icon is activated. */
  url: string
  /** App market icon URL (apps) or site favicon endpoint (sites). */
  iconURL?: string
  /** Source item for the detail dialog. */
  app?: AppMarketItem
  site?: Site
}

export interface DesktopEntries {
  apps: DesktopEntry[]
  sites: DesktopEntry[]
  /** All entries with the URL-deduplicated set applied (app wins). */
  visible: DesktopEntry[]
  /** Reused by the desktop clock to avoid a duplicate public-network call. */
  publicNetwork?: PublicNetworkSummary
  loadedAt: number
}

function normalizeSiteURL(site: Site): string {
  const secure = site.certificate?.status === 'valid' || site.certificate?.status === 'expiring'
  return `${secure ? 'https' : 'http'}://${site.primaryDomain}`
}

function isConfiguredSite(site: Site): boolean {
  // Only healthy, enabled sites surface as desktop entries. Static and redirect
  // sites are reachable without an upstream; proxy-like types need one.
  if (!site.enabled || site.health !== 'healthy') return false
  if (!site.primaryDomain) return false
  if (site.type === 'static') return true
  if (site.type === 'redirect') return true
  return Boolean(site.upstream)
}

function appEntryName(item: AppMarketItem): string {
  return item.name_zh || item.name_en || item.id
}

function buildAppEntries(items: AppMarketItem[], sites: Site[], directHost: string): DesktopEntry[] {
  const entries: DesktopEntry[] = []
  for (const item of items) {
    if (!item.runtime.installed) continue
    const url = appAccessURL(item, sites, directHost)
    if (!url) continue
    entries.push({
      key: `app:${item.id}`,
      kind: 'app',
      id: item.id,
      name: appEntryName(item),
      url,
      iconURL: item.icon,
      app: item,
    })
  }
  return entries
}

function buildSiteEntries(sites: Site[]): DesktopEntry[] {
  return sites
    .filter(isConfiguredSite)
    .map((site) => ({
      key: `site:${site.id}`,
      kind: 'site' as const,
      id: site.id,
      name: site.primaryDomain,
      url: normalizeSiteURL(site),
      iconURL: api.sites.iconURL(site.id),
      site,
    }))
}

/**
 * Collapse entries that point at the same URL. Apps win: if an app and a site
 * resolve to the same URL, only the app entry is kept. Duplicate app entries
 * (same URL, different ids) also collapse to the first.
 */
function dedupeByURL(apps: DesktopEntry[], sites: DesktopEntry[]): DesktopEntry[] {
  const seen = new Set<string>()
  const visible: DesktopEntry[] = []
  for (const entry of [...apps, ...sites]) {
    const normalized = entry.url.replace(/\/+$/, '').toLowerCase()
    if (seen.has(normalized)) continue
    seen.add(normalized)
    visible.push(entry)
  }
  return visible
}

/**
 * Load installed apps and configured sites and compute the visible entry set.
 * Returns undefined when both sources fail, so the desktop can show an empty
 * state rather than stale data.
 *
 * `directHost` is the host used to build direct-access app URLs. Like the app
 * market view, the panel's public IP is preferred (so apps are reachable even
 * when the panel is accessed through a domain); tests pass a literal IP because
 * `appAccessURL` only emits direct URLs for IP hosts.
 */
export async function loadDesktopEntries(
  signal?: AbortSignal,
  directHost?: string,
): Promise<DesktopEntries> {
  const publicNetwork = directHost
    ? undefined
    : await api.system.publicNetwork(signal).catch(() => undefined)
  const host =
    directHost ||
    publicNetwork?.ipv4 ||
    publicNetwork?.ipv6 ||
    window.location.hostname

  const [inventory, sites] = await Promise.all([
    api.apps.inventory(signal).catch(() => undefined),
    api.sites.list(undefined, signal).catch(() => undefined),
  ])

  const apps = buildAppEntries(inventory?.items || [], sites?.items || [], host)
  const siteEntries = buildSiteEntries(sites?.items || [])
  const visible = dedupeByURL(apps, siteEntries)

  return { apps, sites: siteEntries, visible, publicNetwork, loadedAt: Date.now() }
}
