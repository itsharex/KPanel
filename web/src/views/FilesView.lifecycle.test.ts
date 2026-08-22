// @vitest-environment jsdom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FilesView from './FilesView.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  remoteDownload: vi.fn(),
  hosts: vi.fn(),
  route: { query: {} as Record<string, unknown> },
  push: vi.fn(),
}))

vi.mock('vue-router', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push }),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {
    readonly status = 0
    readonly code = 'request_failed'
  },
  api: {
    files: {
      list: mocks.list,
      remoteDownload: mocks.remoteDownload,
      entry: vi.fn(),
      text: vi.fn(),
      write: vi.fn(),
      action: vi.fn(),
      transferFromPanel: vi.fn(),
      trash: vi.fn(),
      upload: vi.fn(),
      contentUrl: vi.fn(() => ''),
      archiveUrl: vi.fn(() => ''),
      createDownloadTicket: vi.fn(),
      createArchiveDownloadTicket: vi.fn(),
      thumbnailUrl: vi.fn(() => ''),
    },
    cluster: { hosts: mocks.hosts },
  },
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({ success: vi.fn(), danger: vi.fn(), show: vi.fn() }),
}))

function directory(path: string) {
  return {
    path,
    entries: [],
    offset: 0,
    total: 0,
    totalKnown: true,
    truncated: false,
    scanTruncated: false,
    readAt: '2026-08-22T00:00:00Z',
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.route.query = {}
  mocks.list.mockResolvedValue(directory('/'))
  mocks.hosts.mockResolvedValue({ nodeId: 'local-node', items: [] })
})

describe('FilesView remote download lifecycle', () => {
  it('does not start a directory read after unmount aborts a download', async () => {
    let downloadSignal: AbortSignal | undefined
    mocks.remoteDownload.mockImplementation((
      _input: unknown,
      _onEvent: (event: unknown) => void,
      signal?: AbortSignal,
    ) => new Promise((_resolve, reject) => {
      downloadSignal = signal
      signal?.addEventListener(
        'abort',
        () => reject(new DOMException('Aborted', 'AbortError')),
        { once: true },
      )
    }))
    const wrapper = shallowMount(FilesView, {
      attachTo: document.body,
      global: {
        stubs: {
          ModalDialog: {
            props: ['open'],
            template: '<div v-if="open"><slot /></div>',
          },
        },
      },
    })
    await flushPromises()
    mocks.list.mockClear()

    const remoteButton = wrapper
      .findAll('.file-command-bar__actions button')
      .find((button) => button.text().includes('远程下载'))
    expect(remoteButton).toBeDefined()
    await remoteButton!.trigger('click')
    await wrapper.find('.remote-download-form input[type="url"]').setValue('https://downloads.example.com/file.bin')
    await wrapper.find('.remote-download-form').trigger('submit')
    await vi.waitFor(() => expect(mocks.remoteDownload).toHaveBeenCalledOnce())

    wrapper.unmount()
    await flushPromises()

    expect(downloadSignal?.aborted).toBe(true)
    expect(mocks.list).not.toHaveBeenCalled()
  })
})
