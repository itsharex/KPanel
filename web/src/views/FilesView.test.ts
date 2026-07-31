import { createSSRApp, ssrContextKey } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FilesView from './FilesView.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  action: vi.fn(),
  success: vi.fn(),
  danger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: {
    files: {
      list: mocks.list,
      action: mocks.action,
      contentUrl: vi.fn(),
      text: vi.fn(),
      write: vi.fn(),
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
  submitDialog: () => Promise<void>
  pasteClipboard: (target?: string) => Promise<void>
  setClipboard: (mode: 'copy' | 'move', entry?: TestFileEntry) => void
  showContext: (event: MouseEvent, entry: TestFileEntry) => void
  showDirectoryContext: (event: MouseEvent) => void
  selectEntry: (event: MouseEvent, path: string) => void
  handleFileShortcut: (event: KeyboardEvent) => void
  openDialog: (action: 'mkdir' | 'rename' | 'chmod' | 'trash', entry?: TestFileEntry) => void
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
  dialogAction: { value?: 'trash' }
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
})

describe('FilesView directory loading', () => {
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
    })
    expect(mocks.danger).toHaveBeenCalledWith(
      '文件操作未完成',
      '0 项成功，1 项失败：文件状态已变化',
    )
    expect(mocks.list).toHaveBeenCalled()
    expect(view.dialogAction.value).toBeUndefined()
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

  it('clears a single selection when the selected row is clicked again', () => {
    const view = setupView()
    const entry = testEntry('toggle.txt')
    view.directory.value = { path: '/', entries: [entry] }

    view.selectEntry({} as MouseEvent, entry.path)
    view.selectEntry({} as MouseEvent, entry.path)

    expect([...view.selected.value]).toEqual([])
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
