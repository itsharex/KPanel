interface OverviewViewport {
  scrollTop: number
  systemSectionOffset?: number
}

const desktopScrollPositions = new WeakMap<HTMLElement, OverviewViewport>()
let classicScrollPosition: OverviewViewport | undefined

function desktopWindowBody(page: HTMLElement | undefined): HTMLElement | undefined {
  return page?.closest<HTMLElement>('.desktop-window__body') ?? undefined
}

export function rememberOverviewViewport(page: HTMLElement | undefined): void {
  const body = desktopWindowBody(page)
  const systemSection = page?.querySelector<HTMLElement>('.overview-system-management')
  if (!body) {
    const systemSectionOffset = systemSection?.getBoundingClientRect().top
    classicScrollPosition = {
      scrollTop: window.scrollY,
      ...(Number.isFinite(systemSectionOffset) ? { systemSectionOffset } : {}),
    }
    return
  }
  const systemSectionOffset = systemSection
    ? systemSection.getBoundingClientRect().top - body.getBoundingClientRect().top
    : undefined
  desktopScrollPositions.set(body, {
    scrollTop: body.scrollTop,
    ...(Number.isFinite(systemSectionOffset) ? { systemSectionOffset } : {}),
  })
}

export function restoreOverviewViewport(page: HTMLElement | undefined): 'desktop' | 'classic' | false {
  const body = desktopWindowBody(page)
  if (!body) {
    const viewport = classicScrollPosition
    if (!viewport) return false
    classicScrollPosition = undefined
    window.scrollTo({ top: viewport.scrollTop })
    const systemSection = page?.querySelector<HTMLElement>('.overview-system-management')
    if (systemSection && viewport.systemSectionOffset !== undefined) {
      const restoredOffset = systemSection.getBoundingClientRect().top
      window.scrollTo({ top: Math.max(0, window.scrollY + restoredOffset - viewport.systemSectionOffset) })
    }
    return 'classic'
  }
  const viewport = desktopScrollPositions.get(body)
  if (!viewport) return false
  desktopScrollPositions.delete(body)
  body.scrollTop = viewport.scrollTop
  const systemSection = page?.querySelector<HTMLElement>('.overview-system-management')
  if (systemSection && viewport.systemSectionOffset !== undefined) {
    const restoredOffset = systemSection.getBoundingClientRect().top - body.getBoundingClientRect().top
    body.scrollTop = Math.max(0, body.scrollTop + restoredOffset - viewport.systemSectionOffset)
  }
  return 'desktop'
}
