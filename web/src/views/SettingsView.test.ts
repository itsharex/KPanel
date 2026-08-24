import { readFileSync } from 'node:fs'
import { createSSRApp, ssrContextKey, type ComputedRef, type Ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ThemeColorIntent, ThemeColorKey, ThemeMode } from '@/theme/colors'
import SettingsView from './SettingsView.vue'

const settingsSource = readFileSync(new URL('./SettingsView.vue', import.meta.url), 'utf8')

const mocks = vi.hoisted(() => {
  class MockApiError extends Error {}

  return {
    MockApiError,
    changePassword: vi.fn(),
    changeUsername: vi.fn(),
    getSecurityEntrance: vi.fn(),
    updateSecurityEntrance: vi.fn(),
    getTOTPStatus: vi.fn(),
    startTOTPEnrollment: vi.fn(),
    confirmTOTPEnrollment: vi.fn(),
    rotateRecoveryCodes: vi.fn(),
    disableTOTP: vi.fn(),
    resetApiSecurityState: vi.fn(),
    replace: vi.fn(),
    toastSuccess: vi.fn(),
    toastDanger: vi.fn(),
    themeSetTheme: vi.fn(),
    themeSetColors: vi.fn(),
    themeResetColors: vi.fn(),
    themePreference: { value: 'system' },
    themeResolved: { value: 'light' },
    themeColors: {
      value: {
        brand: '#0c7a60',
        neutral: '#52645f',
        signatureLinked: true,
        signature: '#0c7a60',
      },
    },
    themeIsCustom: { value: false },
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
      changeUsername: mocks.changeUsername,
      securityEntrance: {
        get: mocks.getSecurityEntrance,
        update: mocks.updateSecurityEntrance,
      },
      totp: {
        status: mocks.getTOTPStatus,
        startEnrollment: mocks.startTOTPEnrollment,
        confirmEnrollment: mocks.confirmTOTPEnrollment,
        regenerateRecoveryCodes: mocks.rotateRecoveryCodes,
        disable: mocks.disableTOTP,
      },
    },
  },
  resetApiSecurityState: mocks.resetApiSecurityState,
}))

vi.mock('qrcode', () => ({ default: { toDataURL: vi.fn().mockResolvedValue('data:image/png;base64,test') } }))

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
    preference: mocks.themePreference,
    resolved: mocks.themeResolved,
    colors: mocks.themeColors,
    isCustom: mocks.themeIsCustom,
    setTheme: mocks.themeSetTheme,
    setColors: mocks.themeSetColors,
    resetColors: mocks.themeResetColors,
  }),
}))

vi.mock('@/stores/toast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    danger: mocks.toastDanger,
  }),
}))

interface SettingsBindings {
  usernameForm: { newUsername: string; currentPassword: string }
  usernameValid: ComputedRef<boolean>
  canChangeUsername: ComputedRef<boolean>
  changingUsername: Ref<boolean>
  usernameSubmitted: Ref<boolean>
  changeUsername: () => Promise<void>
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
  colorDraft: ThemeColorIntent
  colorInputs: Record<ThemeColorKey, string>
  colorErrors: Record<ThemeColorKey, string>
  colorPreviewMode: Ref<ThemeMode>
  colorPreviewTokens: ComputedRef<Record<string, string>>
  hasColorErrors: ComputedRef<boolean>
  colorDraftDirty: ComputedRef<boolean>
  updateThemeColor: (key: ThemeColorKey, event: Event) => void
  applyThemeColors: () => void
  cancelThemeColorChanges: () => void
  resetThemeColors: () => void
  securityEntry: Ref<{ enabled: boolean; path?: string; resourceVersion: string } | undefined>
  securityEntryPath: Ref<string>
  saveSecurityEntry: (enabled: boolean, regenerate?: boolean) => Promise<void>
  totpForm: { currentPassword: string; code: string; secondFactor: string }
  totpAction: Ref<'idle' | 'enroll' | 'verify' | 'recovery' | 'rotate' | 'disable'>
  recoveryCodes: Ref<string[]>
  startTOTPEnrollment: () => Promise<void>
  confirmTOTPEnrollment: () => Promise<void>
  finishTOTPFlow: () => Promise<void>
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

function themeColorInput(value: string, type = 'text'): Event {
  return { currentTarget: { value, type } } as unknown as Event
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.replace.mockResolvedValue(undefined)
  mocks.changePassword.mockResolvedValue(undefined)
  mocks.changeUsername.mockResolvedValue(undefined)
  mocks.getSecurityEntrance.mockResolvedValue({ enabled: false, resourceVersion: 'sha256:initial' })
  mocks.updateSecurityEntrance.mockResolvedValue({
    enabled: true,
    path: 'panel-generated',
    resourceVersion: 'sha256:updated',
  })
  mocks.getTOTPStatus.mockResolvedValue({ enabled: false, recoveryCodesRemaining: 0 })
  mocks.startTOTPEnrollment.mockResolvedValue({
    id: 'enrollment-1', secret: 'JBSWY3DPEHPK3PXP', otpauthUri: 'otpauth://totp/KPanel:admin', expiresAt: '2026-08-01T00:10:00Z',
  })
  mocks.confirmTOTPEnrollment.mockResolvedValue({ recoveryCodes: ['AAAAA-BBBBB-CCCCC', 'DDDDD-EEEEE-FFFFF'] })
  mocks.sessionState.authenticated = true
  mocks.sessionState.user = { id: 'admin', username: 'admin' }
  mocks.sessionState.expiresAt = '2026-07-26T00:00:00Z'
  mocks.sessionState.agent = { connected: true }
  mocks.themePreference.value = 'system'
  mocks.themeResolved.value = 'light'
  mocks.themeColors.value = {
    brand: '#0c7a60',
    neutral: '#52645f',
    signatureLinked: true,
    signature: '#0c7a60',
  }
  mocks.themeIsCustom.value = false
  mocks.themeSetColors.mockImplementation((colors: ThemeColorIntent) => {
    mocks.themeColors.value = { ...colors }
    mocks.themeIsCustom.value = true
  })
  mocks.themeResetColors.mockImplementation(() => {
    mocks.themeColors.value = {
      brand: '#0c7a60',
      neutral: '#52645f',
      signatureLinked: true,
      signature: '#0c7a60',
    }
    mocks.themeIsCustom.value = false
  })
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

describe('SettingsView username change', () => {
  it('keeps the current password empty until user interaction while allowing explicit browser fill', () => {
    expect(settingsSource).toContain('name="username-current-password"')
    expect(settingsSource).toMatch(/name="username-current-password"[\s\S]*?autocomplete="current-password"/)
    expect(settingsSource).toContain(':readonly="!usernamePasswordUnlocked"')
    expect(settingsSource).toContain('@focus="unlockUsernamePassword"')
  })

  it('keeps two-step verification immediately below the security entrance', () => {
    expect(settingsSource.indexOf('<h2>登录安全入口</h2>')).toBeLessThan(settingsSource.indexOf('<h2>两步验证</h2>'))
  })

  it('keeps password change immediately below username change', () => {
    const usernameIndex = settingsSource.indexOf('<h2>修改用户名</h2>')
    const passwordIndex = settingsSource.indexOf('<h2>修改密码</h2>')
    const securityEntryIndex = settingsSource.indexOf('<h2>登录安全入口</h2>')

    expect(usernameIndex).toBeGreaterThanOrEqual(0)
    expect(passwordIndex).toBeGreaterThan(usernameIndex)
    expect(passwordIndex).toBeLessThan(securityEntryIndex)
  })
})

describe('SettingsView appearance', () => {
  it('provides accessible color inputs, linked accents, a local preview, and explicit actions', () => {
    expect(settingsSource).toContain('v-if="field.key !== \'signature\' || !colorDraft.signatureLinked"')
    expect(settingsSource).toContain('class="theme-color-field__picker"')
    expect(settingsSource).toContain('type="color"')
    expect(settingsSource).toContain(':value="colorInputs[field.key]"')
    expect(settingsSource).toContain(':aria-invalid="Boolean(colorErrors[field.key])"')
    expect(settingsSource).toContain('v-model="colorDraft.signatureLinked"')
    expect(settingsSource).toContain('role="radiogroup" aria-label="配色预览模式"')
    expect(settingsSource).toContain('@click="colorPreviewMode = option.id"')
    expect(settingsSource).toContain(':style="colorPreviewStyle"')
    expect(settingsSource).toContain('@click="resetThemeColors"')
    expect(settingsSource).toContain('@click="cancelThemeColorChanges"')
    expect(settingsSource).toContain('@click="applyThemeColors"')
    expect(settingsSource).not.toContain('theme.setSkin')
    expect(settingsSource).not.toContain('KPanel VIP')
    expect(settingsSource).not.toContain('KPanel 经典')
  })

  it('keeps the preview mode and interface mode as separate accessible radio groups', () => {
    expect(settingsSource).toContain('role="radiogroup" aria-label="配色预览模式"')
    expect(settingsSource).toContain(':tabindex="colorPreviewMode === option.id ? 0 : -1"')
    expect(settingsSource).toContain(':aria-checked="colorPreviewMode === option.id"')
    expect(settingsSource).toContain('role="radiogroup" aria-label="明暗模式"')
    expect(settingsSource).toContain(':tabindex="theme.preference.value === option.id ? 0 : -1"')
    expect(settingsSource).toContain(':aria-checked="theme.preference.value === option.id"')
    expect(settingsSource).toContain('@click="theme.setTheme(option.id)"')
    expect(settingsSource.match(/@keydown="moveRadioFocus"/g)).toHaveLength(3)
  })

  it('validates and normalizes Hex input before applying a complete color intent', () => {
    const view = setupView()

    view.updateThemeColor('brand', themeColorInput('#not-a-color'))
    expect(view.colorErrors.brand).toBe('请输入 3 或 6 位 Hex 颜色，例如 #315d7d')
    expect(view.hasColorErrors.value).toBe(true)
    view.applyThemeColors()
    expect(mocks.themeSetColors).not.toHaveBeenCalled()

    view.updateThemeColor('brand', themeColorInput('#357'))
    view.updateThemeColor('neutral', themeColorInput('#65717D'))
    view.colorDraft.signatureLinked = false
    view.updateThemeColor('signature', themeColorInput('#B28C54'))
    expect(view.colorErrors.brand).toBe('')
    expect(view.colorInputs).toMatchObject({ brand: '#357', neutral: '#65717D', signature: '#B28C54' })

    view.applyThemeColors()

    expect(mocks.themeSetColors).toHaveBeenCalledWith({
      brand: '#335577',
      neutral: '#65717d',
      signatureLinked: false,
      signature: '#b28c54',
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('配色已应用', '浅色和深色层级已自动生成。')
    expect(view.colorDraft).toEqual(mocks.themeColors.value)
    expect(view.colorInputs).toEqual({ brand: '#335577', neutral: '#65717d', signature: '#b28c54' })
  })

  it('links the preview accent to the theme color and previews light and dark without applying', () => {
    const view = setupView()
    view.updateThemeColor('brand', themeColorInput('#315d7d'))
    view.updateThemeColor('signature', themeColorInput('#b28c54'))

    const linkedAccent = view.colorPreviewTokens.value['--theme-accent']
    const lightSurface = view.colorPreviewTokens.value['--surface']
    view.colorDraft.signatureLinked = false
    const independentAccent = view.colorPreviewTokens.value['--theme-accent']
    view.colorPreviewMode.value = 'dark'

    expect(independentAccent).not.toBe(linkedAccent)
    expect(view.colorPreviewTokens.value['--surface']).not.toBe(lightSurface)
    expect(mocks.themeSetColors).not.toHaveBeenCalled()
    expect(mocks.themeSetTheme).not.toHaveBeenCalled()
  })

  it('cancels drafts and restores the default color intent explicitly', () => {
    mocks.themeColors.value = {
      brand: '#315d7d',
      neutral: '#65717d',
      signatureLinked: false,
      signature: '#b28c54',
    }
    mocks.themeIsCustom.value = true
    const view = setupView()

    view.updateThemeColor('brand', themeColorInput('#a13f49'))
    view.updateThemeColor('neutral', themeColorInput('invalid'))
    view.cancelThemeColorChanges()
    expect(view.colorDraft).toEqual(mocks.themeColors.value)
    expect(view.colorInputs).toEqual({ brand: '#315d7d', neutral: '#65717d', signature: '#b28c54' })
    expect(view.colorErrors).toEqual({ brand: '', neutral: '', signature: '' })
    expect(view.colorDraftDirty.value).toBe(false)

    view.resetThemeColors()
    expect(mocks.themeResetColors).toHaveBeenCalledOnce()
    expect(view.colorDraft).toEqual({
      brand: '#0c7a60',
      neutral: '#52645f',
      signatureLinked: true,
      signature: '#0c7a60',
    })
    expect(mocks.toastSuccess).toHaveBeenCalledWith('已恢复默认配色')
  })
})

describe('SettingsView username submission', () => {
  it('validates the new username and requires the current password', async () => {
    const view = setupView()
    view.usernameForm.newUsername = 'bad name'
    view.usernameForm.currentPassword = 'CurrentPassword123'
    expect(view.usernameValid.value).toBe(false)
    expect(view.canChangeUsername.value).toBe(false)

    view.usernameForm.newUsername = 'admin'
    expect(view.canChangeUsername.value).toBe(false)

    view.usernameForm.newUsername = 'operator-01'
    view.usernameForm.currentPassword = ''
    await view.changeUsername()
    expect(view.usernameSubmitted.value).toBe(true)
    expect(mocks.changeUsername).not.toHaveBeenCalled()
  })

  it('clears the local session and redirects after a successful username change', async () => {
    const view = setupView()
    view.usernameForm.newUsername = 'operator-01'
    view.usernameForm.currentPassword = 'CurrentPassword123'

    await view.changeUsername()

    expect(mocks.changeUsername).toHaveBeenCalledWith('CurrentPassword123', 'operator-01')
    expect(mocks.resetApiSecurityState).toHaveBeenCalledOnce()
    expect(mocks.sessionState.authenticated).toBe(false)
    expect(mocks.toastSuccess).toHaveBeenCalledWith('用户名已修改', '请使用新用户名重新登录。')
    expect(mocks.replace).toHaveBeenCalledWith({ name: 'login' })
  })
})

describe('SettingsView security entrance', () => {
  it('updates from the current resource version and reflects the generated path', async () => {
    const view = setupView()
    view.securityEntry.value = { enabled: false, resourceVersion: 'sha256:initial' }
    view.securityEntryPath.value = ''

    await view.saveSecurityEntry(true, true)

    expect(mocks.updateSecurityEntrance).toHaveBeenCalledWith({
      enabled: true,
      path: '',
      regenerate: true,
      expectedResourceVersion: 'sha256:initial',
    })
    expect(view.securityEntry.value).toEqual({
      enabled: true,
      path: 'panel-generated',
      resourceVersion: 'sha256:updated',
    })
    expect(view.securityEntryPath.value).toBe('panel-generated')
  })
})

describe('SettingsView two-factor authentication', () => {
  it('requires the current password, verifies TOTP, and shows recovery codes before logout', async () => {
    const view = setupView()
    view.totpAction.value = 'enroll'
    view.totpForm.currentPassword = 'CurrentPassword123'

    await view.startTOTPEnrollment()
    expect(mocks.startTOTPEnrollment).toHaveBeenCalledWith('CurrentPassword123')
    expect(view.totpAction.value).toBe('verify')

    view.totpForm.code = '123456'
    await view.confirmTOTPEnrollment()
    expect(mocks.confirmTOTPEnrollment).toHaveBeenCalledWith('enrollment-1', '123456')
    expect(view.totpAction.value).toBe('recovery')
    expect(view.recoveryCodes.value).toEqual(['AAAAA-BBBBB-CCCCC', 'DDDDD-EEEEE-FFFFF'])
    expect(mocks.replace).not.toHaveBeenCalled()

    await view.finishTOTPFlow()
    expect(mocks.resetApiSecurityState).toHaveBeenCalled()
    expect(mocks.replace).toHaveBeenCalledWith({ name: 'login' })
  })
})
