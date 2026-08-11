// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SSHDefenseDialog from './SSHDefenseDialog.vue'

const mocks = vi.hoisted(() => ({
  sshDefense: vi.fn(),
  sshDefenseAction: vi.fn(),
  success: vi.fn(),
  danger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: { system: { sshDefense: mocks.sshDefense, sshDefenseAction: mocks.sshDefenseAction } },
}))

vi.mock('@/stores/toast', () => ({ useToast: () => ({ success: mocks.success, danger: mocks.danger }) }))

const snapshot = {
  resourceVersion: 'a'.repeat(64), installed: true, running: true, enabled: true, autostart: true,
  jail: 'sshd' as const, profile: 'standard' as const, banTimeSeconds: 3600, findTimeSeconds: 600, maxRetry: 5,
  currentFailed: 2, totalFailed: 18, currentBanned: 2, totalBanned: 5,
  bannedIps: ['198.51.100.8', '2001:db8::8'], bansTruncated: false,
  trustedAddresses: ['127.0.0.1/8'],
  recentEvents: [{ occurredAt: '2026-08-11 01:02:04,000', action: 'ban' as const, address: '198.51.100.8' }],
  maintenance: { state: 'idle' as const, progress: 0, rebootRequired: false }, observedAt: '2026-08-11T08:00:00Z',
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.sshDefense.mockResolvedValue(snapshot)
  mocks.sshDefenseAction.mockResolvedValue({ action: 'set-profile', status: 'succeeded', changed: true, message: 'ok', resourceVersion: 'b'.repeat(64), appliedAt: '2026-08-11T08:01:00Z' })
  vi.stubGlobal('confirm', vi.fn(() => true))
})

describe('SSHDefenseDialog', () => {
  it('stays open while the first snapshot is loading', async () => {
    mocks.sshDefense.mockReturnValueOnce(new Promise(() => undefined))
    const wrapper = mount(SSHDefenseDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('shows a compact status, presets, bans, trusted addresses, and recent events', async () => {
    const wrapper = mount(SSHDefenseDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('防御已开启')
    expect(wrapper.text()).toContain('标准')
    expect(wrapper.text()).toContain('198.51.100.8')
    expect(wrapper.text()).toContain('127.0.0.1/8')
    expect(wrapper.text()).toContain('已封禁')
  })

  it('keeps IP search and applies a fixed profile action', async () => {
    const wrapper = mount(SSHDefenseDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
    await wrapper.find('input[type="search"]').setValue('2001:')
    expect(wrapper.find('.ssh-defense-manager__list').text()).not.toContain('198.51.100.8')
    expect(wrapper.find('.ssh-defense-manager__list').text()).toContain('2001:db8::8')
    await wrapper.findAll('button').find((button) => button.text().includes('严格'))!.trigger('click')
    await flushPromises()
    expect(mocks.sshDefenseAction).toHaveBeenCalledWith({
      action: 'set-profile', expectedResourceVersion: 'a'.repeat(64), profile: 'strict',
    })
  })

  it('confirms and submits unban-all without arbitrary fields', async () => {
    const wrapper = mount(SSHDefenseDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text() === '全部解封')!.trigger('click')
    await flushPromises()
    expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('全部 SSH 封禁'))
    expect(mocks.sshDefenseAction).toHaveBeenCalledWith({ action: 'unban-all', expectedResourceVersion: 'a'.repeat(64) })
  })

  it('does not read the host when the adapter is unavailable', async () => {
    const wrapper = mount(SSHDefenseDialog, {
      props: { open: true, readable: false, writable: false, unavailableReason: '需要更新 kejilion.sh' },
      global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(mocks.sshDefense).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('需要更新 kejilion.sh')
  })
})
