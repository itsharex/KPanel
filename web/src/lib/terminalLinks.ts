const allowedProtocols = new Set(['http:', 'https:'])

export function openTerminalURL(uri: string): boolean {
  let url: URL
  try {
    url = new URL(uri)
  } catch {
    return false
  }
  if (!allowedProtocols.has(url.protocol)) return false

  window.open(url.href, '_blank', 'noopener,noreferrer')
  return true
}
