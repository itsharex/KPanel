// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/lib/api'
import { resetDesktopIconsForTest } from '@/stores/desktopIcons'
import type { DesktopWorkspace, DesktopWorkspaceUpdate } from '@/types/api'
import {
  addFileEntriesToDesktop,
  beginDesktopFileDrag,
  clearDesktopFileDrag,
  crossPanelFileDragEntries,
  crossPanelFileDragEntry,
  desktopFileDragOrigin,
  desktopFileDragEntries,
  DesktopShortcutLimitError,
  hasCrossPanelFileDrag,
  hasDesktopFileDrag,
  peekDesktopFileDragEntries,
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
    expect(beginDesktopFileDrag(event, [{
      name: 'etc', path: '/etc', kind: 'directory', resourceVersion: 'sha256:etc',
    }])).toBe(true)
    expect(event.dataTransfer?.effectAllowed).toBe('all')
    expect(hasDesktopFileDrag(event)).toBe(true)
    expect(peekDesktopFileDragEntries(event)).toEqual([{
      name: 'etc', path: '/etc', kind: 'directory', resourceVersion: 'sha256:etc',
    }])
    expect(desktopFileDragEntries(event)).toEqual([{
      name: 'etc', path: '/etc', kind: 'directory', resourceVersion: 'sha256:etc',
    }])
    expect(desktopFileDragOrigin(event)).toBe('file-manager')
    clearDesktopFileDrag()
    expect(desktopFileDragEntries(event)).toEqual([])
  })

  it('distinguishes a native desktop shortcut drag from a Files drag', () => {
    const event = dragEvent()
    expect(beginDesktopFileDrag(event, [{
      name: 'app', path: '/app', kind: 'directory', resourceVersion: 'sha256:app',
    }], 'd'.repeat(32), 'desktop-shortcut')).toBe(true)
    expect(desktopFileDragOrigin(event)).toBe('desktop-shortcut')
  })

  it('serializes one versioned cross-panel descriptor without an authorization secret', () => {
    const event = dragEvent()
    expect(beginDesktopFileDrag(event, [{
      name: 'app', path: '/app', kind: 'directory', resourceVersion: 'sha256:app-version',
    }], 'a'.repeat(32))).toBe(true)
    clearDesktopFileDrag()

    expect(hasDesktopFileDrag(event)).toBe(false)
    expect(hasCrossPanelFileDrag(event)).toBe(true)
    expect(crossPanelFileDragEntry(event)).toEqual({
      version: 1,
      sourceNodeId: 'a'.repeat(32),
      name: 'app',
      path: '/app',
      kind: 'directory',
      resourceVersion: 'sha256:app-version',
    })
    expect(event.dataTransfer?.getData('application/x-kpanel-cross-panel-file-v1')).not.toContain('token')
  })

  it('serializes a bounded multi-item cross-panel descriptor without partial selection', () => {
    const event = dragEvent()
    beginDesktopFileDrag(event, [
      { name: 'a', path: '/a', kind: 'file', resourceVersion: 'sha256:a' },
      { name: 'b', path: '/b', kind: 'file', resourceVersion: 'sha256:b' },
    ], 'b'.repeat(32))
    clearDesktopFileDrag()
    expect(hasCrossPanelFileDrag(event)).toBe(true)
    expect(crossPanelFileDragEntry(event)).toBeUndefined()
    expect(crossPanelFileDragEntries(event)).toEqual({
      sourceNodeId: 'b'.repeat(32),
      entries: [
        { name: 'a', path: '/a', kind: 'file', resourceVersion: 'sha256:a' },
        { name: 'b', path: '/b', kind: 'file', resourceVersion: 'sha256:b' },
      ],
    })
    expect(event.dataTransfer?.getData('application/x-kpanel-cross-panel-file-v1')).toBe('')
  })

  it('reads the non-secret text fallback when a cross-origin browser strips custom MIME types', () => {
    const source = dragEvent()
    beginDesktopFileDrag(source, [{
      name: 'app', path: '/app', kind: 'directory', resourceVersion: 'sha256:app',
    }], 'e'.repeat(32), 'desktop-shortcut')
    const textPayload = source.dataTransfer!.getData('text/plain')
    const target = {
      dataTransfer: {
        types: ['text/plain'],
        getData: (type: string) => type === 'text/plain' ? textPayload : '',
      },
    } as unknown as DragEvent

    expect(hasCrossPanelFileDrag(target)).toBe(true)
    expect(crossPanelFileDragEntries(target)).toEqual({
      sourceNodeId: 'e'.repeat(32),
      entries: [{ name: 'app', path: '/app', kind: 'directory', resourceVersion: 'sha256:app' }],
    })
  })

  it('advertises an over-limit drag so the target can reject it explicitly', () => {
    const event = dragEvent()
    beginDesktopFileDrag(event, Array.from({ length: 65 }, (_, index) => ({
      name: `${index}.txt`, path: `/${index}.txt`, kind: 'file' as const,
      resourceVersion: `sha256:${index}`,
    })), 'c'.repeat(32))
    clearDesktopFileDrag()

    expect(hasCrossPanelFileDrag(event)).toBe(true)
    expect(crossPanelFileDragEntries(event)).toBeUndefined()
  })
})
