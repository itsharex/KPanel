import { readFileSync } from 'node:fs'
import { createSSRApp, ssrContextKey } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FilesView from './FilesView.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  action: vi.fn(),
  trash: vi.fn(),
  write: vi.fn(),
  thumbnailUrl: vi.fn(),
  success: vi.fn(),
  danger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    files: {
      list: mocks.list,
      action: mocks.action,
      trash: mocks.trash,
      contentUrl: vi.fn(),
      thumbnailUrl: mocks.thumbnailUrl,
      text: vi.fn(),
      write: mocks.write,
      upload: vi.fn(),
    },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.success,
    danger: mocks.danger,
  }),
}))

interface FileBindings {
  loadDirectory: (path?: string, append?: boolean) => Promise<void>
  savePreview: (content?: string) => Promise<void>
  submitDialog: () => Promise<void>
  cancelArchive: () => void
  openTrash: () => Promise<void>
  runTrashAction: (action: 'trash_restore' | 'trash_delete' | 'trash_empty') => Promise<void>
  pasteClipboard: (target?: string) => Promise<void>
  setClipboard: (mode: 'copy' | 'move', entry?: TestFileEntry) => void
  showContext: (event: MouseEvent, entry: TestFileEntry) => void
  showDirectoryContext: (event: MouseEvent) => void
  selectEntry: (event: MouseEvent, path: string) => void
  invertSelection: () => void
  preventNativeSelection: (event: Event) => void
  handleFileShortcut: (event: KeyboardEvent) => void
  openDialog: (action: 'mkdir' | 'rename' | 'chmod' | 'compress' | 'extract' | 'trash', entry?: TestFileEntry) => void
  setViewMode: (mode: 'list' | 'grid') => void
  restoreViewMode: () => void
  canShowThumbnail: (entry: TestFileEntry) => boolean
  thumbnailURL: (entry: TestFileEntry) => string
  markThumbnailFailed: (path: string) => void
  entryIconKind: (entry: TestFileEntry) => string
  currentPath: { value: string }
  directory: {
    value?: {
      path: string
      entries: TestFileEntry[]
    }
  }
  selected: { value: Set<string> }
  clipboard: {
    value?: {
      mode: 'copy' | 'move'
      entries: TestFileEntry[]
    }
  }
  contextMenu: { value?: { entry?: TestFileEntry; x: number; y: number } }
  dialogEntries: { value: TestFileEntry[] }
  dialogAction: { value?: 'mkdir' | 'rename' | 'chmod' | 'compress' | 'extract' | 'trash' }
  dialogValue: { value: string }
  dialogFormat: { value: 'tar.gz' | 'zip' | 'tar' }
  viewMode: { value: 'list' | 'grid' }
  trashEntries: { value: Array<{ id: string; resourceVersion: string; restorable: boolean }> }
  selectedTrash: { value: Set<string> }
  previewEntry: { value?: TestFileEntry }
  previewContent: { value: string }
  previewDirty: { value: boolean }
  codeEditorRef: {
    value?: {
      getValue: () => string
      markClean: () => void
      openSearch: () => void
      focus: () => void
    }
  }
}

interface TestFileEntry {
  name: string
  path: string
  kind: 'file'
  mime: string
  sizeBytes: number
  mode: string
  owner: string
  group: string
  modifiedAt: string
  resourceVersion: string
  editable: boolean
  previewable: boolean
}

function testEntry(name: string): TestFileEntry {
  return {
    name,
    path: `/${name}`,
    kind: 'file',
    mime: 'text/plain',
    sizeBytes: 4,
    mode: '-rw-r--r--',
    owner: 'root',
    group: 'root',
    modifiedAt: '2026-07-30T00:00:00Z',
    resourceVersion: `sha256:${name}`,
    editable: true,
    previewable: true,
  }
}

function setupView(): FileBindings {
  const component = FilesView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => FileBindings
  }
  const app = createSSRApp({ render: () => null })
  app.provide(ssrContextKey, { modules: new Set<string>() })
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
  try {
    return app.runWithContext(() => component.setup({}, { expose: () => undefined }))
  } finally {
    warn.mockRestore()
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('window', {
    innerWidth: 1280,
    innerHeight: 720,
    confirm: vi.fn(() => true),
    localStorage: {
      getItem: vi.fn(),
      setItem: vi.fn(),
    },
  })
  mocks.list.mockResolvedValue({
    path: '/web',
    entries: [],
    offset: 0,
    total: 0,
    truncated: false,
    readAt: '2026-07-30T00:00:00Z',
  })
  mocks.action.mockResolvedValue({ action: 'trash', succeeded: [], failed: [] })
  mocks.trash.mockResolvedValue({ entries: [], total: 0, readAt: '2026-07-30T00:00:00Z' })
  mocks.write.mockImplementation(async (_path: string, _content: string, _version: string) => ({
    entry: testEntry('saved.txt'),
  }))
  mocks.thumbnailUrl.mockImplementation((path: string, version: string) => `/thumb?path=${path}&version=${version}`)
})

describe('FilesView large icon layout', () => {
  it('defaults to the list and persists the selected layout', () => {
    const view = setupView()

    expect(view.viewMode.value).toBe('list')
    view.setViewMode('grid')

    expect(view.viewMode.value).toBe('grid')
    expect(window.localStorage.setItem).toHaveBeenCalledWith('kpanel:files:view:v1', 'grid')
  })

  it('restores a valid saved layout preference', () => {
    vi.mocked(window.localStorage.getItem).mockReturnValue('grid')
    const view = setupView()

    view.restoreViewMode()

    expect(view.viewMode.value).toBe('grid')
  })

  it('only requests bounded safe raster thumbnails and falls back after an error', () => {
    const view = setupView()
    const image = { ...testEntry('photo.png'), mime: 'image/png', sizeBytes: 1024 }
    const svg = { ...testEntry('active.svg'), mime: 'image/svg+xml', sizeBytes: 1024 }
    const oversized = { ...testEntry('large.jpg'), mime: 'image/jpeg', sizeBytes: 13 * 1024 * 1024 }

    expect(view.canShowThumbnail(image)).toBe(true)
    expect(view.thumbnailURL(image)).toContain('/thumb?path=/photo.png')
    expect(view.canShowThumbnail(svg)).toBe(false)
    expect(view.canShowThumbnail(oversized)).toBe(false)

    view.markThumbnailFailed(image.path)
    expect(view.canShowThumbnail(image)).toBe(false)
  })

  it('lazy-loads thumbnails without making the original image draggable', () => {
    const source = readFileSync(new URL('./FilesView.vue', import.meta.url), 'utf8')

    expect(source).toContain('loading="lazy"')
    expect(source).toContain('decoding="async"')
    expect(source).toContain('draggable="false"')
    expect(source).toContain('markThumbnailFailed(entry.path)')
  })

  it('uses distinct icons for common file families', () => {
    const view = setupView()

    expect(view.entryIconKind({ ...testEntry('backup.tar.gz'), mime: 'application/gzip' })).toBe('archive')
    expect(view.entryIconKind({ ...testEntry('data.xlsx'), mime: 'application/octet-stream' })).toBe('spreadsheet')
    expect(view.entryIconKind({ ...testEntry('site.sql'), mime: 'text/plain' })).toBe('database')
    expect(view.entryIconKind({ ...testEntry('.env'), mime: 'text/plain' })).toBe('secret')
    expect(view.entryIconKind({ ...testEntry('main.go'), mime: 'text/plain' })).toBe('code')
  })
})

describe('FilesView directory loading', () => {
  it('keeps the collapsed sidebar offset scoped through a custom property', () => {
    const source = readFileSync(new URL('./FilesView.vue', import.meta.url), 'utf8')

    expect(source).toContain('var(--app-shell-inline-offset)')
    expect(source).not.toContain(':global(.app-shell__main--sidebar-collapsed)')
  })

  it('uses the Agent-confirmed path and clears stale errors', async () => {
    const view = setupView()
    await view.loadDirectory('/web')

    expect(mocks.list).toHaveBeenCalledWith(
      '/web',
      { offset: 0, search: undefined },
      expect.any(AbortSignal),
    )
    expect(view.currentPath.value).toBe('/web')
    expect(view.directory.value?.entries).toEqual([])
    expect(mocks.danger).not.toHaveBeenCalled()
  })

  it('appends a subsequent directory page without duplicating entries', async () => {
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    mocks.list
      .mockResolvedValueOnce({
        path: '/web',
        entries: [first],
        offset: 0,
        nextOffset: 1,
        total: 2,
        truncated: true,
        readAt: '2026-07-30T00:00:00Z',
      })
      .mockResolvedValueOnce({
        path: '/web',
        entries: [first, second],
        offset: 1,
        total: 2,
        truncated: false,
        readAt: '2026-07-30T00:00:01Z',
      })
    const view = setupView()

    await view.loadDirectory('/web')
    await view.loadDirectory('/web', true)

    expect(mocks.list).toHaveBeenLastCalledWith(
      '/web',
      { offset: 1, search: undefined },
      expect.any(AbortSignal),
    )
    expect(view.directory.value?.entries.map((entry) => entry.name)).toEqual([
      'first.txt',
      'second.txt',
    ])
  })

  it('keeps the current directory when refresh fails', async () => {
    mocks.list.mockRejectedValueOnce(new Error('offline'))
    const view = setupView()
    view.currentPath.value = '/docker'

    await view.loadDirectory('/web')

    expect(view.currentPath.value).toBe('/docker')
    expect(mocks.danger).toHaveBeenCalledWith('目录读取失败', 'offline')
  })

  it('reports partial batch results and refreshes the real directory state', async () => {
    const view = setupView()
    const entry = testEntry('keep.txt')
    view.directory.value = {
      path: '/',
      entries: [entry],
    }
    view.selected.value = new Set(['/keep.txt'])
    view.openDialog('trash')
    mocks.action.mockResolvedValueOnce({
      action: 'trash',
      succeeded: [],
      failed: [{ path: '/keep.txt', detail: '文件状态已变化' }],
    })

    await view.submitDialog()

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'trash',
      sources: ['/keep.txt'],
      expectedResourceVersions: { '/keep.txt': 'sha256:keep.txt' },
    })
    expect(mocks.danger).toHaveBeenCalledWith(
      '文件操作未完成',
      '0 项成功，1 项失败：文件状态已变化',
    )
    expect(mocks.list).toHaveBeenCalled()
    expect(view.dialogAction.value).toBeUndefined()
  })

  it('compresses selected entries with the chosen fixed archive format', async () => {
    const view = setupView()
    const entry = testEntry('website.txt')
    view.currentPath.value = '/web'
    view.directory.value = { path: '/web', entries: [entry] }
    view.selected.value = new Set([entry.path])
    view.openDialog('compress')
    view.dialogValue.value = 'release'
    view.dialogFormat.value = 'zip'
    mocks.action.mockResolvedValueOnce({
      action: 'compress',
      succeeded: [{ path: '/web/release.zip' }],
      failed: [],
    })

    await view.submitDialog()

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'compress',
      sources: [entry.path],
      target: '/web',
      name: 'release.zip',
      format: 'zip',
      expectedResourceVersions: { [entry.path]: entry.resourceVersion },
    }, expect.any(AbortSignal))
    expect(mocks.success).toHaveBeenCalledWith('压缩完成', '1 项已处理')
  })

  it('extracts a supported archive into a new non-overwriting directory', async () => {
    const view = setupView()
    const entry = testEntry('backup.tar.gz')
    view.currentPath.value = '/backups'
    view.directory.value = { path: '/backups', entries: [entry] }
    mocks.action.mockResolvedValueOnce({
      action: 'extract',
      succeeded: [{ path: entry.path, destination: '/backups/backup' }],
      failed: [],
    })

    view.openDialog('extract', entry)
    await view.submitDialog()

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'extract',
      sources: [entry.path],
      target: '/backups',
      name: 'backup',
      format: 'tar.gz',
      expectedResourceVersion: entry.resourceVersion,
    }, expect.any(AbortSignal))
    expect(mocks.success).toHaveBeenCalledWith('解压完成', '1 项已处理')
  })

  it('aborts an active archive request and reports cleanup without a false failure', async () => {
    const view = setupView()
    const entry = testEntry('large.log')
    view.currentPath.value = '/logs'
    view.directory.value = { path: '/logs', entries: [entry] }
    view.selected.value = new Set([entry.path])
    view.openDialog('compress')
    mocks.action.mockImplementationOnce((_input: unknown, signal?: AbortSignal) =>
      new Promise((_resolve, reject) => {
        signal?.addEventListener('abort', () => reject(Object.assign(new Error('aborted'), { name: 'AbortError' })))
      }),
    )

    const pending = view.submitDialog()
    view.cancelArchive()
    await pending

    expect(mocks.success).toHaveBeenCalledWith('操作已停止', '未完成的临时文件已清理。')
    expect(mocks.danger).not.toHaveBeenCalledWith('文件操作失败', expect.any(String))
    expect(view.dialogAction.value).toBeUndefined()
  })

  it('restores a recycle-bin entry with resource-version protection', async () => {
    const view = setupView()
    mocks.trash.mockResolvedValueOnce({
      entries: [{
        id: 'trash-id',
        name: 'config.json',
        originalPath: '/etc/config.json',
        kind: 'file',
        sizeBytes: 4,
        mode: '-rw-r--r--',
        owner: 'root',
        group: 'root',
        deletedAt: '2026-07-30T00:00:00Z',
        resourceVersion: 'sha256:trash',
        restorable: true,
      }],
      total: 1,
      readAt: '2026-07-30T00:00:00Z',
    })
    mocks.action.mockResolvedValueOnce({
      action: 'trash_restore',
      succeeded: [{ path: 'trash-id', destination: '/etc/config.json' }],
      failed: [],
    })

    await view.openTrash()
    view.selectedTrash.value = new Set(['trash-id'])
    await view.runTrashAction('trash_restore')

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'trash_restore',
      trashIds: ['trash-id'],
      expectedResourceVersions: { 'trash-id': 'sha256:trash' },
    })
    expect(mocks.success).toHaveBeenCalledWith('恢复完成', '1 项已处理')
  })

  it('saves the live editor value without copying the document on every keystroke', async () => {
    const view = setupView()
    const entry = testEntry('config.json')
    const markClean = vi.fn()
    view.previewEntry.value = entry
    view.previewContent.value = 'stale content'
    view.previewDirty.value = true
    view.codeEditorRef.value = {
      getValue: () => 'latest editor content',
      markClean,
      openSearch: vi.fn(),
      focus: vi.fn(),
    }
    mocks.write.mockResolvedValueOnce({ entry: { ...entry, resourceVersion: 'sha256:saved' } })

    await view.savePreview()

    expect(mocks.write).toHaveBeenCalledWith(
      entry.path,
      'latest editor content',
      entry.resourceVersion,
    )
    expect(view.previewContent.value).toBe('latest editor content')
    expect(view.previewDirty.value).toBe(false)
    expect(markClean).toHaveBeenCalledOnce()
  })

  it('selects an unchecked entry when opening its context menu', () => {
    const view = setupView()
    const checked = testEntry('checked.txt')
    const clicked = testEntry('clicked.txt')
    view.directory.value = { path: '/', entries: [checked, clicked] }
    view.selected.value = new Set([checked.path])

    view.showContext(
      {
        preventDefault: vi.fn(),
        clientX: 400,
        clientY: 300,
      } as unknown as MouseEvent,
      clicked,
    )

    expect([...view.selected.value]).toEqual([clicked.path])
    expect(view.contextMenu.value?.entry?.path).toBe(clicked.path)
  })

  it('preserves a multi-selection when opening a selected entry context menu', () => {
    const view = setupView()
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    view.directory.value = { path: '/', entries: [first, second] }
    view.selected.value = new Set([first.path, second.path])

    view.showContext(
      {
        preventDefault: vi.fn(),
        clientX: 400,
        clientY: 300,
      } as unknown as MouseEvent,
      second,
    )

    expect([...view.selected.value]).toEqual([first.path, second.path])
    expect(view.contextMenu.value?.entry?.path).toBe(second.path)
  })

  it('uses Windows-style click, control-click, and shift-click selection', () => {
    const view = setupView()
    const first = testEntry('a.txt')
    const second = testEntry('b.txt')
    const third = testEntry('c.txt')
    view.directory.value = { path: '/', entries: [first, second, third] }

    view.selectEntry({} as MouseEvent, first.path)
    expect([...view.selected.value]).toEqual([first.path])

    view.selectEntry({ ctrlKey: true } as MouseEvent, third.path)
    expect([...view.selected.value]).toEqual([first.path, third.path])

    view.selectEntry({ shiftKey: true } as MouseEvent, second.path)
    expect([...view.selected.value]).toEqual([second.path, third.path])
  })

  it('prevents browser text selection inside the file list', () => {
    const view = setupView()
    const preventDefault = vi.fn()

    view.preventNativeSelection({ preventDefault } as unknown as Event)

    expect(preventDefault).toHaveBeenCalledOnce()
  })

  it('clears a single selection when the selected row is clicked again', () => {
    const view = setupView()
    const entry = testEntry('toggle.txt')
    view.directory.value = { path: '/', entries: [entry] }

    view.selectEntry({} as MouseEvent, entry.path)
    view.selectEntry({} as MouseEvent, entry.path)

    expect([...view.selected.value]).toEqual([])
  })

  it('inverts the selection across the visible entries', () => {
    const view = setupView()
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    const third = testEntry('third.txt')
    view.directory.value = { path: '/', entries: [first, second, third] }
    view.selected.value = new Set([first.path, third.path])

    view.invertSelection()

    expect([...view.selected.value]).toEqual([second.path])
  })

  it('selects every visible entry with control-a', () => {
    const view = setupView()
    const first = testEntry('a.txt')
    const second = testEntry('b.txt')
    view.directory.value = { path: '/', entries: [first, second] }
    const preventDefault = vi.fn()

    view.handleFileShortcut({
      key: 'a',
      ctrlKey: true,
      preventDefault,
    } as unknown as KeyboardEvent)

    expect(preventDefault).toHaveBeenCalled()
    expect([...view.selected.value]).toEqual([first.path, second.path])
  })

  it('copies to the page clipboard, clears selection, and does not execute a file action', () => {
    const view = setupView()
    const checked = testEntry('checked.txt')
    const clicked = testEntry('clicked.txt')
    view.directory.value = { path: '/', entries: [checked, clicked] }
    view.selected.value = new Set([checked.path])

    view.setClipboard('copy', clicked)

    expect(view.clipboard.value?.mode).toBe('copy')
    expect(view.clipboard.value?.entries.map((entry) => entry.path)).toEqual([clicked.path])
    expect([...view.selected.value]).toEqual([])
    expect(mocks.action).not.toHaveBeenCalled()
  })

  it('opens a current-directory context menu from blank space', () => {
    const view = setupView()
    const preventDefault = vi.fn()

    view.showDirectoryContext({
      preventDefault,
      clientX: 400,
      clientY: 300,
      target: { closest: vi.fn(() => null) },
    } as unknown as MouseEvent)

    expect(preventDefault).toHaveBeenCalled()
    expect(view.contextMenu.value).toEqual({ x: 400, y: 300 })
  })

  it('pastes copied entries into the current directory and keeps the clipboard', async () => {
    const view = setupView()
    const entry = testEntry('source.txt')
    view.currentPath.value = '/target'
    view.clipboard.value = { mode: 'copy', entries: [entry] }
    mocks.action.mockResolvedValueOnce({
      action: 'copy',
      succeeded: [{ path: entry.path, destination: '/target/source.txt' }],
      failed: [],
    })

    await view.pasteClipboard()

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'copy',
      sources: [entry.path],
      target: '/target',
    })
    expect(view.clipboard.value?.entries).toEqual([entry])
    expect(mocks.list).toHaveBeenCalled()
  })

  it('keeps only failed entries after a partial cut paste', async () => {
    const view = setupView()
    const moved = testEntry('moved.txt')
    const failed = testEntry('failed.txt')
    view.clipboard.value = { mode: 'move', entries: [moved, failed] }
    mocks.action.mockResolvedValueOnce({
      action: 'move',
      succeeded: [{ path: moved.path, destination: `/target/${moved.name}` }],
      failed: [{ path: failed.path, detail: '目标已存在' }],
    })

    await view.pasteClipboard('/target')

    expect(view.clipboard.value?.mode).toBe('move')
    expect(view.clipboard.value?.entries.map((entry) => entry.path)).toEqual([failed.path])
    expect(mocks.danger).toHaveBeenCalledWith(
      '部分文件未粘贴',
      '1 项成功，1 项失败：目标已存在',
    )
  })
})
