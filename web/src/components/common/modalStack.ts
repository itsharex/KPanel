const openModals: symbol[] = []

function syncBodyScrollLock(): void {
  if (typeof document === 'undefined') return
  document.body.classList.toggle('has-modal', openModals.length > 0)
}

export function activateModal(id: symbol): void {
  if (!openModals.includes(id)) openModals.push(id)
  syncBodyScrollLock()
}

export function deactivateModal(id: symbol): void {
  const index = openModals.indexOf(id)
  if (index >= 0) openModals.splice(index, 1)
  syncBodyScrollLock()
}

export function isTopModal(id: symbol): boolean {
  return openModals[openModals.length - 1] === id
}
