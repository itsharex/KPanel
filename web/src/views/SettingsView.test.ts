import { createSSRApp, ssrContextKey, type ComputedRef, type Ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsView from './SettingsView.vue'

const mocks = vi.hoisted(() => {
  class MockApiError extends Error {}

  return {
    MockApiError,
    changePassword: vi.fn(),
    resetApiSecurityState: vi.fn(),
    replace: vi.fn(),
    toastSuccess: vi.fn(),
    toastDanger: vi.fn(),
    sessionState: {
      authenticated: true,
      user: { id: 'admin', username: 'admin' } as { id: string; username: string } | undefined,
      expiresAt: '2026-07-26T00:00:00Z' as string | undefined,
      agent: { connected: true } as { connected: boolean } | undefined,
    },
  }
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: mocks.replace }),
}))

vi.mock('@/lib/api', () => ({
  ApiError: mocks.MockApiError,
  api: {
    agent: {
      capabilities: vi.fn().mockResolvedValue([]),
      health: vi.fn(),
    },
    settings: {
      changePassword: mocks.changePassword,
    },
  },
  resetApiSecurityState: mocks.resetApiSecurityState,
}))

vi.mock('@/stores/panel', () => ({
  usePanelState: () => ({
    state: { agent: undefined },
    setAgent: vi.fn(),
  }),
}))

vi.mock('@/stores/session', () => ({
  useSession: () => ({ state: mocks.sessionState }),
}))

vi.mock('@/stores/theme', () => ({
  useTheme: () => ({
    preference: { value: 'system' },
    setTheme: vi.fn(),
  }),
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    danger: mocks.toastDanger,
  }),
}))

interface SettingsBindings {
  passwordForm: {
    currentPassword: string
    newPassword: string
    confirmPassword: string
  }
  passwordChecks: ComputedRef<Array<{ label: string; valid: boolean }>>
  canChangePassword: ComputedRef<boolean>
  changingPassword: Ref<boolean>
  passwordSubmitted: Ref<boolean>
  changePassword: () => Promise<void>
}

function setupView(): SettingsBindings {
  const component = SettingsView as unknown as {
    setup: (props: Record<string, never>, context: { expose: () => void }) => SettingsBindings
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
  mocks.replace.mockResolvedValue(undefined)
  mocks.changePassword.mockResolvedValue(undefined)
  mocks.sessionState.authenticated = true
  mocks.sessionState.user = { id: 'admin', username: 'admin' }
  mocks.sessionState.expiresAt = '2026-07-26T00:00:00Z'
  mocks.sessionState.agent = { connected: true }
})

describe('SettingsView password change', () => {
  it('requires the existing password and a matching 12-character password with letters and digits', async () => {
    const view = setupView()

    view.passwordForm.currentPassword = 'old-password'
    view.passwordForm.newPassword = 'Short123'
    view.passwordForm.confirmPassword = 'Short123'
    expect(view.canChangePassword.value).toBe(false)

    view.passwordForm.newPassword = 'abcdefghijkl'
    view.passwordForm.confirmPassword = 'abcdefghijkl'
    expect(view.passwordChecks.value[1]?.valid).toBe(false)
    expect(view.canChangePassword.value).toBe(false)

    view.passwordForm.newPassword = 'StrongPassword123'
    view.passwordForm.confirmPassword = 'StrongPassword124'
    expect(view.canChangePassword.value).toBe(false)

    await view.changePassword()
    expect(view.passwordSubmitted.value).toBe(true)
    expect(mocks.changePassword).not.toHaveBeenCalled()
  })

  it('clears the local session and redirects to login after a successful change', async () => {
    const view = setupView()
    view.passwordForm.currentPassword = 'CurrentPassword123'
    view.passwordForm.newPassword = 'ReplacementPassword456'
    view.passwordForm.confirmPassword = 'ReplacementPassword456'

    await view.changePassword()

    expect(mocks.changePassword).toHaveBeenCalledWith('CurrentPassword123', 'ReplacementPassword456')
    expect(mocks.resetApiSecurityState).toHaveBeenCalledOnce()
    expect(mocks.sessionState).toMatchObject({
      authenticated: false,
      user: undefined,
      expiresAt: undefined,
      agent: undefined,
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('密码已修改', '请使用新密码重新登录。')
    expect(mocks.replace).toHaveBeenCalledWith({ name: 'login' })
    expect(view.passwordForm).toEqual({
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
    })
  })

  it('keeps the current session when the API rejects the current password', async () => {
    mocks.changePassword.mockRejectedValueOnce(new mocks.MockApiError('当前密码不正确'))
    const view = setupView()
    view.passwordForm.currentPassword = 'WrongPassword123'
    view.passwordForm.newPassword = 'ReplacementPassword456'
    view.passwordForm.confirmPassword = 'ReplacementPassword456'

    await view.changePassword()

    expect(mocks.toastDanger).toHaveBeenCalledWith('密码修改失败', '当前密码不正确')
    expect(mocks.resetApiSecurityState).not.toHaveBeenCalled()
    expect(mocks.replace).not.toHaveBeenCalled()
    expect(mocks.sessionState.authenticated).toBe(true)
    expect(view.changingPassword.value).toBe(false)
  })
})
