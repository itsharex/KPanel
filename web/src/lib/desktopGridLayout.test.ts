import { describe, expect, it } from 'vitest'
import {
  deriveDesktopGridLayout,
  desktopGridPlacementRect,
  dropDesktopGridItem,
  moveDesktopGridItemByKeyboard,
} from './desktopGridLayout'
import { desktopIconPositionForGridSlot, type DesktopIconBounds } from './desktopIconLayout'

const bounds: DesktopIconBounds = { width: 1000, height: 700 }
const items = [
  { key: 'icon:one' },
  { key: 'icon:two' },
  { key: 'widget:clock', columns: 3, rows: 2, defaultSlot: { column: 7, row: 0 } },
  { key: 'widget:monitor', columns: 3, rows: 3, defaultSlot: { column: 7, row: 2 } },
]

function assertNoOverlap(
  placements: ReturnType<typeof deriveDesktopGridLayout>['placements'],
): void {
  for (let left = 0; left < placements.length; left += 1) {
    const first = desktopGridPlacementRect(placements[left]!, bounds)
    for (let right = left + 1; right < placements.length; right += 1) {
      const second = desktopGridPlacementRect(placements[right]!, bounds)
      expect(
        first.left >= second.left + second.width
          || second.left >= first.left + first.width
          || first.top >= second.top + second.height
          || second.top >= first.top + first.height,
      ).toBe(true)
    }
  }
}

describe('desktop mixed grid layout', () => {
  it('places variable-size widgets in the same collision-free grid as icons', () => {
    const layout = deriveDesktopGridLayout(items, [], bounds, false)
    expect(layout.placements).toHaveLength(4)
    expect(layout.placements.find((item) => item.key === 'widget:clock')?.columns).toBe(3)
    expect(desktopGridPlacementRect(layout.placements.find((item) => item.key === 'widget:clock')!, bounds).width)
      .toBeGreaterThan(desktopGridPlacementRect(layout.placements[0]!, bounds).width)
    assertNoOverlap(layout.placements)
  })

  it('keeps a widget out of an occupied icon target instead of overlapping it', () => {
    const layout = deriveDesktopGridLayout(items, [], bounds, false)
    const moved = dropDesktopGridItem(
      layout.placements,
      items,
      'widget:clock',
      layout.placements.find((item) => item.key === 'icon:one')!.position,
      bounds,
    )
    assertNoOverlap(moved)
  })

  it('supports keyboard movement with the same collision rules', () => {
    const layout = deriveDesktopGridLayout(items, [], bounds, false)
    const current = layout.placements.find((item) => item.key === 'icon:one')!
    const moved = moveDesktopGridItemByKeyboard(layout.placements, items, 'icon:one', 'right', bounds)
    const next = moved.find((item) => item.key === 'icon:one')!
    expect(next.position).not.toEqual(current.position)
    assertNoOverlap(moved)
  })

  it('preserves explicit normalized widget positions across derivation', () => {
    const saved = desktopIconPositionForGridSlot({ column: 5, row: 1 }, bounds)
    const layout = deriveDesktopGridLayout(items, [{ key: 'widget:clock', position: saved }], bounds, false)
    const clock = layout.placements.find((item) => item.key === 'widget:clock')!
    expect(clock.position).toEqual(saved)
  })
})
