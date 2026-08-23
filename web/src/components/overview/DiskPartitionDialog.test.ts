// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import DiskPartitionDialog from './DiskPartitionDialog.vue'
import englishSharedCatalog from '@/i18n/pages/shared/en-US'
import { registerPhraseCatalog, resetPhraseLocalizationForTest } from '@/i18n/phrase'

const mocks = vi.hoisted(() => ({ disks: vi.fn(), action: vi.fn(), success: vi.fn(), danger: vi.fn() }))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: { system: { disks: mocks.disks, diskAction: mocks.action } },
}))
vi.mock('@/stores/toast', () => ({ useToast: () => ({ success: mocks.success, danger: mocks.danger }) }))

const disabled = (reason = '设备正在使用') => ({ enabled: false, reason })
const enabled = { enabled: true }
const snapshot = {
  resourceVersion: 'a'.repeat(64),
  platform: { kind: 'wsl2' as const, label: 'WSL2', writable: true },
  observedAt: '2026-08-23T08:00:00Z',
  devices: [
    {
      id: 'disk-root', path: '/dev/sda', name: 'sda', type: 'disk', sizeBytes: 64 * 1024 ** 3,
      readOnly: false, removable: false, virtual: false, model: 'Virtual Disk', transport: 'scsi',
      mounts: [], protected: true, protectionReasons: ['承载系统根目录'],
      operations: { mount: disabled(), unmount: disabled(), format: disabled(), check: disabled(), repair: disabled() },
    },
    {
      id: 'part-root', parentId: 'disk-root', path: '/dev/sda1', name: 'sda1', type: 'part', sizeBytes: 64 * 1024 ** 3,
      readOnly: false, removable: false, virtual: false, filesystem: { type: 'ext4', uuid: 'root-uuid' },
      mounts: [{ path: '/', persistent: true, totalBytes: 60 * 1024 ** 3, usedBytes: 20 * 1024 ** 3, usagePercent: 33 }],
      protected: true, protectionReasons: ['承载系统根目录'],
      operations: { mount: disabled(), unmount: disabled(), format: disabled(), check: disabled(), repair: disabled() },
    },
    {
      id: 'loop-safe', path: '/dev/loop0', name: 'loop0', type: 'loop', sizeBytes: 768 * 1024 ** 2,
      readOnly: false, removable: false, virtual: true, filesystem: { type: 'ext4', label: 'test' }, mounts: [],
      protected: false, protectionReasons: [],
      operations: { mount: enabled, unmount: disabled('尚未挂载'), format: enabled, check: enabled, repair: enabled },
    },
  ],
}

beforeEach(() => {
  resetPhraseLocalizationForTest()
  vi.clearAllMocks()
  mocks.disks.mockResolvedValue(structuredClone(snapshot))
  mocks.action.mockResolvedValue({
    id: 'job-1', action: 'mount', deviceId: 'loop-safe', devicePath: '/dev/loop0', status: 'queued',
    stage: 'queued', progress: 2, message: '任务已进入队列', createdAt: '2026-08-23T08:01:00Z',
  })
})

afterEach(() => {
  document.body.innerHTML = ''
  resetPhraseLocalizationForTest()
})

describe('DiskPartitionDialog', () => {
  it('keeps the dialog open during load and renders protected topology', async () => {
    mocks.disks.mockReturnValueOnce(new Promise(() => undefined))
    const wrapper = mount(DiskPartitionDialog, {
      props: { open: true, readable: true, writable: true },
      global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('selects the first actionable leaf and submits a typed mount request', async () => {
    const wrapper = mount(DiskPartitionDialog, {
      props: { open: true, readable: true, writable: true },
      global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('/dev/sda1')
    expect(wrapper.text()).toContain('已保护')
    expect(wrapper.find('.disk-inspector').text()).toContain('/dev/loop0')
    const mountInput = wrapper.find('.disk-action-card input[type="text"]')
    const mountButton = () => wrapper.findAll('button').find((button) => button.text().includes('挂载文件系统'))!
    await mountInput.setValue('/home/review-disk')
    expect(wrapper.text()).toContain('系统保护范围')
    expect(mountButton().attributes('disabled')).toBeDefined()
    await mountInput.setValue('/mnt/review-disk')
    expect(mountButton().attributes('disabled')).toBeUndefined()
    await mountButton().trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('请核对目标和影响')
    const confirmButton = wrapper.findAll('button').find((button) => button.text().includes('确认执行'))!
    await confirmButton.trigger('click')
    await flushPromises()
    expect(mocks.action).toHaveBeenCalledWith({
      action: 'mount', deviceId: 'loop-safe', expectedResourceVersion: 'a'.repeat(64),
      mountPoint: '/mnt/review-disk', persist: true,
    })
  })

  it('requires confirmation before an irreversible format request', async () => {
    const wrapper = mount(DiskPartitionDialog, {
      props: { open: true, readable: true, writable: true },
      global: { stubs: { teleport: true } },
    })
    await flushPromises()
    const formatButton = wrapper.findAll('button').find((button) => button.text().includes('格式化设备'))!
    await formatButton.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('永久清除')
    const confirmButton = wrapper.findAll('button').find((button) => button.text().includes('确认执行'))!
    await confirmButton.trigger('click')
    await flushPromises()
    expect(mocks.action).toHaveBeenCalledWith({
      action: 'format', deviceId: 'loop-safe', expectedResourceVersion: 'a'.repeat(64), filesystem: 'ext4',
    })
  })

  it('restores an active job and disables conflicting actions', async () => {
    mocks.disks.mockResolvedValueOnce({
      ...structuredClone(snapshot),
      job: {
        id: 'job-running', action: 'check', deviceId: 'loop-safe', devicePath: '/dev/loop0',
        status: 'running', stage: 'checking', progress: 48, message: '正在执行只读检查', createdAt: '2026-08-23T08:01:00Z',
      },
    })
    const wrapper = mount(DiskPartitionDialog, {
      props: { open: true, readable: true, writable: true },
      global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('正在处理 /dev/loop0 · 48%')
    const formatButton = wrapper.findAll('button').find((button) => button.text().includes('格式化设备'))!
    expect(formatButton.attributes('disabled')).toBeDefined()
  })

  it('localizes content rendered through the teleported modal', async () => {
    registerPhraseCatalog(englishSharedCatalog)
    const wrapper = mount(DiskPartitionDialog, {
      attachTo: document.body,
      props: { open: true, readable: true, writable: true },
    })
    await flushPromises()

    expect(document.body.textContent).toContain('Disks and partitions')
    expect(document.body.textContent).toContain('Physical and virtual devices')
    expect(document.body.textContent).toContain('Mount filesystem')
    expect(document.body.textContent).not.toContain('查看真实块设备拓扑')

    const mountButton = [...document.querySelectorAll<HTMLButtonElement>('button')]
      .find((button) => button.textContent?.includes('Mount filesystem'))!
    mountButton.click()
    await flushPromises()
    expect(document.body.textContent).toContain(
      'Mount /dev/loop0 at /mnt/loop0 and add a boot-time mount entry. Continue?',
    )
    wrapper.unmount()
  })
})
