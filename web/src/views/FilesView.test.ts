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
  loadDirectory: (path?: string) => Promise<void>
  submitDialog: () => Promise<void>
  currentPath: { value: string }
  directory: {
    value?: {
      path: string
      entries: Array<{
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
      }>
    }
  }
  selected: { value: Set<string> }
  dialogAction: { value?: 'trash' }
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
  mocks.list.mockResolvedValue({
    path: '/web',
    entries: [],
    truncated: false,
    readAt: '2026-07-30T00:00:00Z',
  })
  mocks.action.mockResolvedValue({ action: 'trash', succeeded: [], failed: [] })
})

describe('FilesView directory loading', () => {
  it('uses the Agent-confirmed path and clears stale errors', async () => {
    const view = setupView()
    await view.loadDirectory('/web')

    expect(mocks.list).toHaveBeenCalledWith('/web')
    expect(view.currentPath.value).toBe('/web')
    expect(view.directory.value?.entries).toEqual([])
    expect(mocks.danger).not.toHaveBeenCalled()
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
    view.directory.value = {
      path: '/',
      entries: [
        {
          name: 'keep.txt',
          path: '/keep.txt',
          kind: 'file',
          mime: 'text/plain',
          sizeBytes: 4,
          mode: '-rw-r--r--',
          owner: 'root',
          group: 'root',
          modifiedAt: '2026-07-30T00:00:00Z',
          resourceVersion: 'sha256:test',
          editable: true,
          previewable: true,
        },
      ],
    }
    view.selected.value = new Set(['/keep.txt'])
    view.dialogAction.value = 'trash'
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
})
