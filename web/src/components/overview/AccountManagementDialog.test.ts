// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountManagementDialog from './AccountManagementDialog.vue'

const mocks = vi.hoisted(() => ({
  accounts: vi.fn(),
  accountAction: vi.fn(),
  success: vi.fn(),
  danger: vi.fn(),
}))

vi.mock('@/lib/api', () => ({
  ApiError: class MockApiError extends Error {},
  api: { system: { accounts: mocks.accounts, accountAction: mocks.accountAction } },
}))

vi.mock('@/stores/toast', () => ({ useToast: () => ({ success: mocks.success, danger: mocks.danger }) }))

const snapshot = {
  resourceVersion: 'a'.repeat(64), total: 3, truncated: false, observedAt: '2026-08-11T08:00:00Z',
  sshPolicy: { passwordAuthentication: true, publicKeyAuthentication: true, rootLogin: 'enabled' as const },
  accounts: [
    { username: 'root', uid: 0, gid: 0, home: '/root', shell: '/bin/bash', kind: 'root' as const, passwordStatus: 'enabled' as const, role: 'root' as const, groups: ['root'], sshKeys: [] },
    { username: 'daemon', uid: 1, gid: 1, home: '/usr/sbin', shell: '/usr/sbin/nologin', kind: 'system' as const, passwordStatus: 'locked' as const, role: 'standard' as const, groups: ['daemon'], sshKeys: [] },
    { username: 'operator', uid: 1000, gid: 1000, home: '/home/operator', shell: '/bin/bash', kind: 'human' as const, passwordStatus: 'locked' as const, role: 'passwordless-admin' as const, groups: ['operator', 'sudo'], sshKeys: [{ id: 'b'.repeat(64), type: 'ssh-ed25519', fingerprint: 'SHA256:test', comment: 'laptop' }] },
  ],
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.accounts.mockResolvedValue(snapshot)
  mocks.accountAction.mockResolvedValue({ action: 'set-ssh-policy', status: 'succeeded', changed: true, message: 'ok', resourceVersion: 'c'.repeat(64), appliedAt: '2026-08-11T08:01:00Z' })
  vi.stubGlobal('confirm', vi.fn(() => true))
})

describe('AccountManagementDialog', () => {
  it('shows human login accounts first and keeps system accounts optional', async () => {
    const wrapper = mount(AccountManagementDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('operator')
    expect(wrapper.text()).not.toContain('daemon')
    const checkbox = wrapper.find('input[type="checkbox"]')
    await checkbox.setValue(true)
    expect(wrapper.text()).toContain('daemon')
  })

  it('applies the clear key-login preset with an ordinary lockout warning', async () => {
    const wrapper = mount(AccountManagementDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
	await wrapper.findAll('button').find((button) => button.text() === 'SSH 登录策略')!.trigger('click')
	await wrapper.findAll('button').find((button) => button.text().includes('密钥登录模式'))!.trigger('click')
	await wrapper.findAll('button').find((button) => button.text().includes('应用登录策略'))!.trigger('click')
    await flushPromises()
    expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('至少有一把可用私钥'))
    expect(mocks.accountAction).toHaveBeenCalledWith({
      action: 'set-ssh-policy', expectedResourceVersion: 'a'.repeat(64),
      passwordAuthentication: false, rootLogin: 'key-only',
    })
  })

  it('does not read accounts when the adapter is unavailable', async () => {
    const wrapper = mount(AccountManagementDialog, {
      props: { open: true, readable: false, writable: false, unavailableReason: '需要升级脚本' },
      global: { stubs: { teleport: true } },
    })
    await flushPromises()
    expect(mocks.accounts).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('需要升级脚本')
  })

  it('validates password limits by UTF-8 bytes', async () => {
    const wrapper = mount(AccountManagementDialog, {
      props: { open: true, readable: true, writable: true }, global: { stubs: { teleport: true } },
    })
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text().includes('创建账户'))!.trigger('click')
    const username = wrapper.find('input[placeholder="operator"]')
    await username.setValue('unicodeuser')
    const password = wrapper.find('input[type="password"]')
    const submit = wrapper.find('form button[type="submit"]')
    await password.setValue('密'.repeat(86))
    expect(submit.attributes('disabled')).toBeDefined()
    await password.setValue('correct horse battery staple')
    expect((username.element as HTMLInputElement).value).toBe('unicodeuser')
    expect((password.element as HTMLInputElement).value).toBe('correct horse battery staple')
    expect(wrapper.find('form button[type="submit"]').attributes('disabled')).toBeUndefined()
  })
})
