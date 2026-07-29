import { afterEach, describe, expect, it, vi } from 'vitest'
import { activateModal, deactivateModal, isTopModal } from './modalStack'

const registered: symbol[] = []

function modalID(label: string): symbol {
  const id = Symbol(label)
  registered.push(id)
  return id
}

function installDocumentStub(): Set<string> {
  const classes = new Set<string>()
  vi.stubGlobal('document', {
    body: {
      classList: {
        toggle(name: string, force: boolean) {
          if (force) classes.add(name)
          else classes.delete(name)
        },
      },
    },
  })
  return classes
}

afterEach(() => {
  for (const id of registered.splice(0)) deactivateModal(id)
  vi.unstubAllGlobals()
})

describe('modal stack', () => {
  it('keeps page scrolling locked until the last open modal closes', () => {
    const classes = installDocumentStub()
    const detail = modalID('detail')
    const confirmation = modalID('confirmation')

    activateModal(detail)
    activateModal(confirmation)
    deactivateModal(confirmation)

    expect(classes.has('has-modal')).toBe(true)
    expect(isTopModal(detail)).toBe(true)

    deactivateModal(detail)
    expect(classes.has('has-modal')).toBe(false)
  })

  it('identifies only the most recently opened modal as the Escape target', () => {
    installDocumentStub()
    const detail = modalID('detail')
    const progress = modalID('progress')

    activateModal(detail)
    activateModal(progress)

    expect(isTopModal(detail)).toBe(false)
    expect(isTopModal(progress)).toBe(true)

    deactivateModal(progress)
    expect(isTopModal(detail)).toBe(true)
  })
})
