import { readFileSync } from 'node:fs'
import { createSSRApp, nextTick, reactive, ssrContextKey } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FilesView from './FilesView.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  action: vi.fn(),
  trash: vi.fn(),
  write: vi.fn(),
  createDownloadTicket: vi.fn(),
  thumbnailUrl: vi.fn(),
  success: vi.fn(),
  danger: vi.fn(),
  route: { query: {} as Record<string, unknown> },
  push: vi.fn(),
}))

vi.mock('vue-router', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push }),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    files: {
      list: mocks.list,
      action: mocks.action,
      trash: mocks.trash,
      contentUrl: vi.fn(),
      createDownloadTicket: mocks.createDownloadTicket,
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

interface FileDirectoryResult {
  path: string
  entries: TestFileEntry[]
  offset: number
  total: number
  truncated: boolean
  readAt: string
}

function testDirectory(path: string): FileDirectoryResult {
  return {
    path,
    entries: [],
    offset: 0,
    total: 0,
    truncated: false,
    readAt: '2026-07-30T00:00:00Z',
  }
}

interface FileBindings {
  requestedFilePath: (value: unknown) => string | undefined
  loadDirectory: (path?: string, append?: boolean) => Promise<string | undefined>
  navigateDirectory: (path: string) => Promise<void>
  savePreview: (content?: string) => Promise<void>
  download: (entry: TestFileEntry) => Promise<void>
  downloadSelected: (entry?: TestFileEntry) => Promise<void>
  submitDialog: () => Promise<void>
  cancelArchive: () => void
  openTrash: () => Promise<void>
  toggleAllTrash: () => void
  runTrashAction: (action: 'trash_restore' | 'trash_delete' | 'trash_empty') => Promise<void>
  pasteClipboard: (target?: string) => Promise<void>
  setClipboard: (mode: 'copy' | 'move', entry?: TestFileEntry) => void
  showContext: (event: MouseEvent, entry: TestFileEntry) => void
  showDirectoryContext: (event: MouseEvent) => void
  handleEntryClick: (event: MouseEvent, entry: TestFileEntry) => void
  selectEntry: (event: MouseEvent, path: string) => void
  invertSelection: () => void
  preventNativeSelection: (event: Event) => void
  handleFileShortcut: (event: KeyboardEvent) => void
  filesPage: { value?: HTMLElement }
  openDialog: (action: 'mkdir' | 'rename' | 'chmod' | 'compress' | 'extract' | 'trash', entry?: TestFileEntry) => void
  setViewMode: (mode: 'list' | 'grid') => void
  restoreViewMode: () => void
  canShowThumbnail: (entry: TestFileEntry) => boolean
  thumbnailURL: (entry: TestFileEntry) => string
  markThumbnailFailed: (path: string) => void
  entryIconKind: (entry: TestFileEntry) => string
  currentPath: { value: string }
  search: { value: string }
  entries: { value: TestFileEntry[] }
  entrySearchCatalog: {
    value: Array<{ entry: TestFileEntry; searchName: string }>
  }
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
  mocks.route = reactive({ query: {} as Record<string, unknown> })
  mocks.push.mockImplementation(async (location: { query?: Record<string, unknown> }) => {
    mocks.route.query = location.query || {}
  })
  vi.stubGlobal('window', {
    innerWidth: 1280,
    innerHeight: 720,
    confirm: vi.fn(() => true),
    setTimeout: vi.fn(() => 1),
    clearTimeout: vi.fn(),
    localStorage: {
      getItem: vi.fn(),
      setItem: vi.fn(),
    },
  })
  vi.stubGlobal('document', { activeElement: null })
  mocks.list.mockResolvedValue(testDirectory('/web'))
  mocks.action.mockResolvedValue({ action: 'trash', succeeded: [], failed: [] })
  mocks.trash.mockResolvedValue({ entries: [], total: 0, readAt: '2026-07-30T00:00:00Z' })
  mocks.write.mockImplementation(async (_path: string, _content: string, _version: string) => ({
    entry: testEntry('saved.txt'),
  }))
  mocks.createDownloadTicket.mockResolvedValue({
    downloadUrl: '/api/v1/files/download/test-ticket',
    expiresAt: '2026-07-30T00:05:00Z',
  })
  mocks.thumbnailUrl.mockImplementation((path: string, version: string) => `/thumb?path=${path}&version=${version}`)
})

describe('FilesView downloads', () => {
  it('uses a short-lived download URL and reports ticket failures', async () => {
    const anchor = {
      href: '',
      download: '',
      rel: '',
      click: vi.fn(),
      remove: vi.fn(),
    }
    const appendChild = vi.fn()
    vi.stubGlobal('document', {
      activeElement: null,
      body: { appendChild },
      createElement: vi.fn(() => anchor),
    })
    const view = setupView()
    const entry = testEntry('hello.txt')

    await view.download(entry)

    expect(mocks.createDownloadTicket).toHaveBeenCalledWith('/hello.txt')
    expect(anchor.href).toBe('/api/v1/files/download/test-ticket')
    expect(anchor.download).toBe('hello.txt')
    expect(appendChild).toHaveBeenCalledWith(anchor)
    expect(anchor.click).toHaveBeenCalledOnce()
    expect(anchor.remove).toHaveBeenCalledOnce()

    mocks.createDownloadTicket.mockRejectedValueOnce(new Error('ticket failed'))
    await view.download(entry)
    expect(mocks.danger).toHaveBeenCalledWith('下载失败', 'ticket failed')
  })
})

describe('FilesView route path', () => {
  it('accepts bounded absolute paths and rejects traversal or relative paths', () => {
    const view = setupView()
    expect(view.requestedFilePath('/home/web/html/example.com')).toBe('/home/web/html/example.com')
    expect(view.requestedFilePath(['/root/project'])).toBe('/root/project')
    expect(view.requestedFilePath('../etc')).toBeUndefined()
    expect(view.requestedFilePath('/home/../etc')).toBeUndefined()
    expect(view.requestedFilePath('/home//example')).toBeUndefined()
    expect(view.requestedFilePath('/home/./example')).toBeUndefined()
    expect(view.requestedFilePath('/home\\example')).toBeUndefined()
    expect(view.requestedFilePath(`/home/${'x'.repeat(4096)}`)).toBeUndefined()
  })

  it('records successful directory navigation once using the Agent-confirmed path', async () => {
    mocks.route.query = { path: '/home' }
    mocks.list.mockResolvedValueOnce(testDirectory('/home/project'))
    const view = setupView()

    await view.navigateDirectory('/home/project/')
    await nextTick()

    expect(mocks.list).toHaveBeenCalledTimes(1)
    expect(mocks.push).toHaveBeenCalledOnce()
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'files',
      query: { path: '/home/project' },
    })
  })

  it('loads browser history changes including the bare files route', async () => {
    mocks.route.query = { path: '/home/project' }
    mocks.list.mockImplementation(async (path: string) => testDirectory(path))
    const view = setupView()
    view.currentPath.value = '/home/project'

    mocks.route.query = { path: '/home' }
    await vi.waitFor(() => expect(view.currentPath.value).toBe('/home'))
    mocks.route.query = {}
    await vi.waitFor(() => expect(view.currentPath.value).toBe('/'))

    expect(mocks.list.mock.calls.map(([path]) => path)).toEqual(['/home', '/'])
    expect(mocks.push).not.toHaveBeenCalled()
  })

  it('does not record failed or superseded directory navigation', async () => {
    let resolveSlow: ((value: FileDirectoryResult) => void) | undefined
    mocks.list.mockImplementation((path: string) => {
      if (path === '/slow') {
        return new Promise<FileDirectoryResult>((resolve) => {
          resolveSlow = resolve
        })
      }
      return Promise.resolve(testDirectory(path))
    })
    const view = setupView()

    const slow = view.navigateDirectory('/slow')
    await view.navigateDirectory('/fast')
    resolveSlow?.(testDirectory('/slow'))
    await slow

    expect(view.currentPath.value).toBe('/fast')
    expect(mocks.push).toHaveBeenCalledTimes(1)
    expect(mocks.push).toHaveBeenCalledWith({ name: 'files', query: { path: '/fast' } })

    mocks.push.mockClear()
    mocks.list.mockRejectedValueOnce(new Error('missing'))
    await view.navigateDirectory('/missing')
    expect(mocks.push).not.toHaveBeenCalled()
  })
})

describe('FilesView large icon layout', () => {
  it('skips layout and paint work for offscreen directory entries', () => {
    const source = readFileSync(new URL('./FilesView.vue', import.meta.url), 'utf8')

    expect(source).toMatch(/\.file-row--entry\s*\{[^}]*content-visibility:\s*auto;/)
    expect(source).toMatch(/\.file-grid-card\s*\{[^}]*content-visibility:\s*auto;/)
  })

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
  it('reuses the sorted directory catalog while the search query changes', () => {
    const view = setupView()
    view.directory.value = {
      path: '/',
      entries: [testEntry('zeta.log'), testEntry('alpha.txt'), testEntry('beta.txt')],
    }

    view.search.value = 'txt'
    const catalog = view.entrySearchCatalog.value
    view.search.value = ''
    expect(view.entries.value.map((entry) => entry.name)).toEqual([
      'alpha.txt',
      'beta.txt',
      'zeta.log',
    ])

    view.search.value = 'txt'
    expect(view.entries.value.map((entry) => entry.name)).toEqual(['alpha.txt', 'beta.txt'])
    expect(view.entrySearchCatalog.value).toBe(catalog)
  })

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
    const first = testEntry('website.txt')
    const second = testEntry('assets.txt')
    view.currentPath.value = '/web'
    view.directory.value = { path: '/web', entries: [first, second] }
    view.selected.value = new Set([first.path, second.path])
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
      sources: [first.path, second.path],
      target: '/web',
      name: 'release.zip',
      format: 'zip',
      expectedResourceVersions: {
        [first.path]: first.resourceVersion,
        [second.path]: second.resourceVersion,
      },
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

  it('selects and permanently deletes every visible recycle-bin entry', async () => {
    const view = setupView()
    const entries = [
      { id: 'trash-first', resourceVersion: 'sha256:first', restorable: true },
      { id: 'trash-second', resourceVersion: 'sha256:second', restorable: true },
    ].map((entry, index) => ({
      ...entry,
      name: `${index}.txt`,
      originalPath: `/${index}.txt`,
      kind: 'file' as const,
      sizeBytes: 4,
      mode: '-rw-r--r--',
      owner: 'root',
      group: 'root',
      deletedAt: '2026-07-30T00:00:00Z',
    }))
    mocks.trash.mockResolvedValueOnce({
      entries,
      total: entries.length,
      readAt: '2026-07-30T00:00:00Z',
    })
    mocks.action.mockResolvedValueOnce({
      action: 'trash_delete',
      succeeded: entries.map((entry) => ({ path: entry.id })),
      failed: [],
    })

    await view.openTrash()
    view.toggleAllTrash()
    await view.runTrashAction('trash_delete')

    expect(window.confirm).toHaveBeenCalledWith('彻底删除选中的 2 项？此操作不可恢复。')
    expect(mocks.action).toHaveBeenCalledWith({
      action: 'trash_delete',
      trashIds: ['trash-first', 'trash-second'],
      expectedResourceVersions: {
        'trash-first': 'sha256:first',
        'trash-second': 'sha256:second',
      },
    })
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

  it('keeps the full selection for every batch-capable context action', () => {
    const view = setupView()
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    view.directory.value = { path: '/', entries: [first, second] }
    view.selected.value = new Set([first.path, second.path])

    view.openDialog('compress', second)
    expect(view.dialogEntries.value.map((entry) => entry.path)).toEqual([first.path, second.path])

    view.openDialog('chmod', second)
    expect(view.dialogEntries.value.map((entry) => entry.path)).toEqual([first.path, second.path])

    view.setClipboard('copy', second)
    expect(view.clipboard.value?.entries.map((entry) => entry.path)).toEqual([first.path, second.path])

    view.selected.value = new Set([first.path, second.path])
    view.setClipboard('move', second)
    expect(view.clipboard.value?.mode).toBe('move')
    expect(view.clipboard.value?.entries.map((entry) => entry.path)).toEqual([first.path, second.path])
  })

  it('keeps rename and extract scoped to the context-menu entry', () => {
    const view = setupView()
    const first = testEntry('first.txt')
    const archive = { ...testEntry('second.zip'), mime: 'application/zip' }
    view.directory.value = { path: '/', entries: [first, archive] }
    view.selected.value = new Set([first.path, archive.path])

    view.openDialog('rename', archive)
    expect(view.dialogEntries.value.map((entry) => entry.path)).toEqual([archive.path])

    view.openDialog('extract', archive)
    expect(view.dialogEntries.value.map((entry) => entry.path)).toEqual([archive.path])
  })

  it('trashes every selected entry from a selected entry context menu', async () => {
    const view = setupView()
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    view.directory.value = { path: '/', entries: [first, second] }
    view.selected.value = new Set([first.path, second.path])
    mocks.action.mockResolvedValueOnce({
      action: 'trash',
      succeeded: [{ path: first.path }, { path: second.path }],
      failed: [],
    })

    view.openDialog('trash', second)
    await view.submitDialog()

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'trash',
      sources: [first.path, second.path],
      expectedResourceVersions: {
        [first.path]: first.resourceVersion,
        [second.path]: second.resourceVersion,
      },
    })
  })

  it('submits every selected entry and resource version for batch chmod', async () => {
    const view = setupView()
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    view.directory.value = { path: '/', entries: [first, second] }
    view.selected.value = new Set([first.path, second.path])
    mocks.action.mockResolvedValueOnce({
      action: 'chmod',
      succeeded: [{ path: first.path }, { path: second.path }],
      failed: [],
    })

    view.openDialog('chmod', second)
    view.dialogValue.value = '640'
    await view.submitDialog()

    expect(mocks.action).toHaveBeenCalledWith({
      action: 'chmod',
      sources: [first.path, second.path],
      mode: '640',
      expectedResourceVersions: {
        [first.path]: first.resourceVersion,
        [second.path]: second.resourceVersion,
      },
    })
  })

  it('downloads every selected file sequentially from a selected entry context menu', async () => {
    const anchors: Array<{ click: ReturnType<typeof vi.fn>; remove: ReturnType<typeof vi.fn> }> = []
    vi.stubGlobal('document', {
      activeElement: null,
      body: { appendChild: vi.fn() },
      createElement: vi.fn(() => {
        const anchor = { href: '', download: '', rel: '', click: vi.fn(), remove: vi.fn() }
        anchors.push(anchor)
        return anchor
      }),
    })
    const view = setupView()
    const first = testEntry('first.txt')
    const second = testEntry('second.txt')
    view.directory.value = { path: '/', entries: [first, second] }
    view.selected.value = new Set([first.path, second.path])
    mocks.createDownloadTicket.mockImplementation(async (path: string) => ({
      downloadUrl: `/download${path}`,
      expiresAt: '2026-07-30T00:05:00Z',
    }))

    await view.downloadSelected(second)

    expect(mocks.createDownloadTicket.mock.calls.map(([path]) => path)).toEqual([first.path, second.path])
    expect(anchors.map((anchor) => anchor.click.mock.calls.length)).toEqual([1, 1])
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

  it('opens entries with one tap on phone widths without changing desktop selection behavior', () => {
    const view = setupView()
    const entry = testEntry('mobile.txt')
    view.directory.value = { path: '/', entries: [entry] }
    const mobileEvent = {
      preventDefault: vi.fn(),
      stopPropagation: vi.fn(),
    } as unknown as MouseEvent

    window.innerWidth = 390
    view.handleEntryClick(mobileEvent, entry)
    expect(view.previewEntry.value?.path).toBe(entry.path)
    expect(view.selected.value.size).toBe(0)
    expect(mobileEvent.preventDefault).toHaveBeenCalledOnce()

    window.innerWidth = 1280
    view.previewEntry.value = undefined
    view.handleEntryClick({ shiftKey: false, ctrlKey: false, metaKey: false } as MouseEvent, entry)
    expect(view.previewEntry.value).toBeUndefined()
    expect(view.selected.value).toEqual(new Set([entry.path]))
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
    const target = { matches: vi.fn(() => false) } as unknown as HTMLElement
    view.filesPage.value = {
      contains: (node: Node | null) => node === target,
    } as HTMLElement

    view.handleFileShortcut({
      key: 'a',
      ctrlKey: true,
      preventDefault,
      target,
    } as unknown as KeyboardEvent)

    expect(preventDefault).toHaveBeenCalled()
    expect([...view.selected.value]).toEqual([first.path, second.path])
  })

  it('ignores destructive shortcuts when focus is outside the files page', () => {
    const view = setupView()
    const entry = testEntry('outside.txt')
    view.directory.value = { path: '/', entries: [entry] }
    view.selected.value = new Set([entry.path])
    view.filesPage.value = { contains: () => false } as unknown as HTMLElement
    const preventDefault = vi.fn()

    view.handleFileShortcut({
      key: 'Delete',
      preventDefault,
      target: { matches: vi.fn(() => false) },
    } as unknown as KeyboardEvent)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(view.dialogAction.value).toBeUndefined()
  })

  it('ignores file shortcuts while a toolbar or menu button has focus', () => {
    const view = setupView()
    const entry = testEntry('button-focus.txt')
    const target = { matches: vi.fn(() => true) } as unknown as HTMLElement
    view.directory.value = { path: '/', entries: [entry] }
    view.selected.value = new Set([entry.path])
    view.filesPage.value = { contains: (node: Node | null) => node === target } as HTMLElement
    const preventDefault = vi.fn()

    view.handleFileShortcut({ key: 'Delete', preventDefault, target } as unknown as KeyboardEvent)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(view.dialogAction.value).toBeUndefined()
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
