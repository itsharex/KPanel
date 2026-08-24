const RADIO_KEYS = new Set(['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'])

export function moveRadioFocus(event: KeyboardEvent): void {
  if (!RADIO_KEYS.has(event.key)) return
  const current = event.currentTarget as HTMLButtonElement | null
  const group = current?.closest<HTMLElement>('[role="radiogroup"]')
  if (!current || !group) return
  const options = Array.from(group.querySelectorAll<HTMLButtonElement>('[role="radio"]'))
  const index = options.indexOf(current)
  if (index < 0 || options.length === 0) return

  event.preventDefault()
  const nextIndex = event.key === 'Home'
    ? 0
    : event.key === 'End'
      ? options.length - 1
      : (index + (event.key === 'ArrowLeft' || event.key === 'ArrowUp' ? -1 : 1) + options.length) % options.length
  const nextOption = options[nextIndex]
  nextOption?.focus()
  nextOption?.click()
}
