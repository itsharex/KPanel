import { describe, expect, it, vi } from 'vitest'
import { readSidebarCollapsed, writeSidebarCollapsed } from './sidebarPreference'

describe('sidebar preference', () => {
  it('restores only the explicit collapsed value', () => {
    expect(readSidebarCollapsed({ getItem: () => '1' })).toBe(true)
    expect(readSidebarCollapsed({ getItem: () => '0' })).toBe(false)
    expect(readSidebarCollapsed({ getItem: () => null })).toBe(false)
  })

  it('persists expanded and collapsed states', () => {
    const setItem = vi.fn()
    const storage = { setItem }

    writeSidebarCollapsed(true, storage)
    writeSidebarCollapsed(false, storage)

    expect(setItem).toHaveBeenNthCalledWith(1, 'kejilion-panel-sidebar-collapsed', '1')
    expect(setItem).toHaveBeenNthCalledWith(2, 'kejilion-panel-sidebar-collapsed', '0')
  })

  it('falls back safely when browser storage is unavailable', () => {
    const unavailableReader = {
      getItem: () => {
        throw new Error('blocked')
      },
    }
    const unavailableWriter = {
      setItem: () => {
        throw new Error('blocked')
      },
    }

    expect(readSidebarCollapsed(unavailableReader)).toBe(false)
    expect(() => writeSidebarCollapsed(true, unavailableWriter)).not.toThrow()
  })
})
