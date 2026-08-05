type NavigationScrollPosition = {
  behavior?: ScrollOptions['behavior']
  left?: number
  top?: number
}

export function resolveNavigationScroll(
  toPath: string,
  fromPath: string,
  savedPosition: NavigationScrollPosition | null,
): NavigationScrollPosition | false {
  if (savedPosition) return savedPosition
  if (toPath === fromPath) return false
  return { top: 0 }
}
