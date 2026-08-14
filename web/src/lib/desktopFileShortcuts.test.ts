// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { resetDesktopIconsForTest } from '@/stores/desktopIcons'
import type { DesktopWorkspace, DesktopWorkspaceUpdate } from '@/types/api'
import {
  addFileEntriesToDesktop,
  beginDesktopFileDrag,
  clearDesktopFileDrag,
  desktopFileDragEntries,
  DesktopShortcutLimitError,
  hasDesktopFileDrag,
} from './desktopFileShortcuts'

function workspace(overrides: Partial<DesktopWorkspace> = {}): DesktopWorkspace {
  return {
    schemaVersion: 2,
    resourceVersion: `sha256:${'1'.repeat(64)}`,
    available: true,
    hiddenEntryKeys: [],
    positions: {},
    labels: {},
    shortcuts: [],
    ...overrides,
  }
}

function installWorkspace(initial: DesktopWorkspace): void {
  vi.spyOn(api.desktop, 'workspace').mockResolvedValue(initial)
  vi.spyOn(api.desktop, 'updateWorkspace').mockImplementation(async (input: DesktopWorkspaceUpdate) => workspace({
    resourceVersion: `sha256:${'2'.repeat(64)}`,
    hiddenEntryKeys: input.hiddenEntryKeys,
    positions: input.positions,
    labels: input.labels,
    shortcuts: input.shortcuts.map((shortcut) => ({
      ...shortcut,
      createdAt: '2026-08-14T00:00:00Z',
      updatedAt: '2026-08-14T00:00:00Z',
    })),
  }))
}

function dragEvent() {
  const values = new Map<string, string>()
  const types: string[] = []
  const dataTransfer = {
    types,
    effectAllowed: 'none',
    setData(type: string, value: string) {
      if (!types.includes(type)) types.push(type)
      values.set(type, value)
    },
    getData(type: string) {
      return values.get(type) || ''
    },
  }
  return { dataTransfer } as unknown as DragEvent
}

describe('desktop file shortcuts', () => {
  beforeEach(() => {
    resetDesktopIconsForTest()
    clearDesktopFileDrag()
    vi.restoreAllMocks()
  })

  it('adds files and directories in one workspace write and ignores unsupported entries', async () => {
    installWorkspace(workspace())
    const result = await addFileEntriesToDesktop([
      { name: 'nginx.conf', path: '/etc/nginx/nginx.conf', kind: 'file' },
      { name: '网站目录', path: '/home/web', kind: 'directory' },
      { name: 'current', path: '/proc/self/fd/1', kind: 'symlink' },
    ])

    expect(result.added.map(({ name, targetType, path }) => ({ name, targetType, path }))).toEqual([
      { name: 'nginx.conf', targetType: 'file', path: '/etc/nginx/nginx.conf' },
      { name: '网站目录', targetType: 'directory', path: '/home/web' },
    ])
    expect(result.ignored).toHaveLength(1)
    expect(api.desktop.updateWorkspace).toHaveBeenCalledTimes(1)
  })

  it('deduplicates an existing target and enforces the bounded shortcut limit', async () => {
    const existing = workspace({
      shortcuts: [{
        id: 'a'.repeat(32), name: 'nginx.conf', description: '', targetType: 'file', path: '/etc/nginx.conf',
        createdAt: '2026-08-14T00:00:00Z', updatedAt: '2026-08-14T00:00:00Z',
      }],
    })
    installWorkspace(existing)
    const duplicate = await addFileEntriesToDesktop([{ name: 'nginx.conf', path: '/etc/nginx.conf', kind: 'file' }])
    expect(duplicate.added).toHaveLength(0)
    expect(duplicate.duplicates).toHaveLength(1)

    resetDesktopIconsForTest()
    const full = workspace({
      shortcuts: Array.from({ length: 64 }, (_, index) => ({
        id: index.toString(16).padStart(32, '0'), name: `Item ${index}`, description: '',
        targetType: 'file' as const, path: `/tmp/${index}`,
        createdAt: '2026-08-14T00:00:00Z', updatedAt: '2026-08-14T00:00:00Z',
      })),
    })
    vi.mocked(api.desktop.workspace).mockResolvedValue(full)
    await expect(addFileEntriesToDesktop([{ name: 'extra', path: '/tmp/extra', kind: 'file' }]))
      .rejects.toBeInstanceOf(DesktopShortcutLimitError)
  })

  it('accepts only the active in-memory drag token', () => {
    const event = dragEvent()
    expect(beginDesktopFileDrag(event, [{ name: 'etc', path: '/etc', kind: 'directory' }])).toBe(true)
    expect(hasDesktopFileDrag(event)).toBe(true)
    expect(desktopFileDragEntries(event)).toEqual([{ name: 'etc', path: '/etc', kind: 'directory' }])
    clearDesktopFileDrag()
    expect(desktopFileDragEntries(event)).toEqual([])
  })
})
