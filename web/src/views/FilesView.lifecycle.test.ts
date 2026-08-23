// @vitest-environment jsdom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FilesView from './FilesView.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  remoteDownload: vi.fn(),
  createRemoteDownloadJob: vi.fn(),
  remoteDownloadJobs: vi.fn(),
  remoteDownloadJob: vi.fn(),
  cancelRemoteDownloadJob: vi.fn(),
  deleteRemoteDownloadJob: vi.fn(),
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
      createRemoteDownloadJob: mocks.createRemoteDownloadJob,
      remoteDownloadJobs: mocks.remoteDownloadJobs,
      remoteDownloadJob: mocks.remoteDownloadJob,
      cancelRemoteDownloadJob: mocks.cancelRemoteDownloadJob,
      deleteRemoteDownloadJob: mocks.deleteRemoteDownloadJob,
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
  mocks.remoteDownloadJobs.mockResolvedValue({ items: [] })
  mocks.hosts.mockResolvedValue({ nodeId: 'local-node', items: [] })
})

describe('FilesView remote download lifecycle', () => {
  it('backs off a failed active-task poll for ten seconds', async () => {
    vi.useFakeTimers()
    const activeJob = {
      id: 'a'.repeat(32),
      state: 'transferring',
      source: 'https://downloads.example.com',
      targetDirectory: '/',
      name: 'file.bin',
      loadedBytes: 1024,
      totalBytes: 4096,
      createdAt: '2026-08-23T00:00:00Z',
      updatedAt: '2026-08-23T00:00:01Z',
    }
    mocks.remoteDownloadJobs
      .mockResolvedValueOnce({ items: [activeJob] })
      .mockRejectedValueOnce(new Error('poll failed'))
      .mockResolvedValueOnce({ items: [activeJob] })
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
    try {
      await flushPromises()
      expect(mocks.remoteDownloadJobs).toHaveBeenCalledTimes(1)

      await vi.advanceTimersByTimeAsync(2_500)
      await flushPromises()
      expect(mocks.remoteDownloadJobs).toHaveBeenCalledTimes(2)

      await vi.advanceTimersByTimeAsync(2_500)
      await flushPromises()
      expect(mocks.remoteDownloadJobs).toHaveBeenCalledTimes(2)

      await vi.advanceTimersByTimeAsync(7_500)
      await flushPromises()
      expect(mocks.remoteDownloadJobs).toHaveBeenCalledTimes(3)
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
    }
  })

  it('restores persisted tasks on mount and does not cancel an active server task on unmount', async () => {
    mocks.remoteDownloadJobs.mockResolvedValueOnce({
      items: [{
        id: 'a'.repeat(32),
        state: 'transferring',
        source: 'https://downloads.example.com',
        targetDirectory: '/',
        name: 'file.bin',
        loadedBytes: 1024,
        totalBytes: 4096,
        createdAt: '2026-08-23T00:00:00Z',
        updatedAt: '2026-08-23T00:00:01Z',
      }],
    })
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

    expect(mocks.remoteDownloadJobs).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('file.bin')
    expect(wrapper.text()).toContain('关闭页面后仍会继续')
    const phase = wrapper.get('.remote-download-task__phase')
    expect(phase.attributes()).toMatchObject({
      role: 'status', 'aria-live': 'polite', 'aria-atomic': 'true',
    })
    expect(phase.text()).toBe('正在接收远程文件')
    expect(phase.text()).not.toContain('已接收')
    const bytes = wrapper.get('.remote-download-task__bytes')
    expect(bytes.text()).toContain('已接收')
    expect(bytes.attributes('role')).toBeUndefined()
    expect(bytes.attributes('aria-live')).toBeUndefined()

    wrapper.unmount()
    await flushPromises()

    expect(mocks.cancelRemoteDownloadJob).not.toHaveBeenCalled()
  })

  it('aborts only an in-flight task-list read when the page unmounts', async () => {
    let jobsSignal: AbortSignal | undefined
    mocks.remoteDownloadJobs.mockImplementationOnce((signal?: AbortSignal) => new Promise((_resolve, reject) => {
      jobsSignal = signal
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
    await vi.waitFor(() => expect(jobsSignal).toBeDefined())

    wrapper.unmount()
    await flushPromises()

    expect(jobsSignal?.aborted).toBe(true)
    expect(mocks.cancelRemoteDownloadJob).not.toHaveBeenCalled()
  })
})
