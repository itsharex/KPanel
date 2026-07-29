<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Check,
  Clock3,
  ExternalLink,
  KeyRound,
  LoaderCircle,
  Monitor,
  Moon,
  RefreshCw,
  Scale,
  Server,
  ShieldCheck,
  Sun,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api, resetApiSecurityState } from '@/lib/api'
import { formatDateTime, relativeTime } from '@/lib/format'
import { usePanelState } from '@/stores/panel'
import { useSession } from '@/stores/session'
import { useTheme, type ThemePreference } from '@/stores/theme'
import { useToast } from '@/stores/toast'

const router = useRouter()
const session = useSession()
const panel = usePanelState()
const theme = useTheme()
const toast = useToast()
const refreshing = ref(false)
const changingPassword = ref(false)
const passwordSubmitted = ref(false)
const passwordForm = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const capabilities = ref<Array<{ id: string; enabled: boolean; reason?: string }>>([])

const passwordChecks = computed(() => [
  { label: '至少 12 个字符', valid: passwordForm.newPassword.length >= 12 },
  {
    label: '包含字母和数字',
    valid: /[A-Za-z]/.test(passwordForm.newPassword) && /\d/.test(passwordForm.newPassword),
  },
])

const canChangePassword = computed(
  () =>
    passwordForm.currentPassword.length > 0 &&
    passwordChecks.value.every((item) => item.valid) &&
    passwordForm.newPassword === passwordForm.confirmPassword,
)

const agentState = computed(() => {
  const agent = panel.state.agent
  if (!agent?.connected) return { status: 'offline', label: '离线' }
  if (!agent.compatible) return { status: 'incompatible', label: '不兼容' }
  if (agent.readOnly) return { status: 'read_only', label: '写入依赖未就绪' }
  return { status: 'connected', label: '正常' }
})

const themes: Array<{ id: ThemePreference; label: string; description: string; icon: typeof Sun }> = [
  { id: 'light', label: '浅色', description: '始终使用明亮界面', icon: Sun },
  { id: 'dark', label: '深色', description: '始终使用低亮度界面', icon: Moon },
  { id: 'system', label: '跟随系统', description: '随设备设置自动切换', icon: Monitor },
]

async function refreshAgent(): Promise<void> {
  refreshing.value = true
  try {
    const [health, capabilityResult] = await Promise.all([api.agent.health(), api.agent.capabilities()])
    panel.setAgent(health)
    session.state.agent = health
    capabilities.value = capabilityResult
    toast.success('连接状态已更新')
  } catch (reason) {
    toast.danger('无法连接 Agent', reason instanceof ApiError ? reason.message : '请检查宿主机服务。')
  } finally {
    refreshing.value = false
  }
}

async function changePassword(): Promise<void> {
  passwordSubmitted.value = true
  if (!canChangePassword.value || changingPassword.value) return

  changingPassword.value = true
  try {
    await api.settings.changePassword(passwordForm.currentPassword, passwordForm.newPassword)
  } catch (reason) {
    toast.danger('密码修改失败', reason instanceof ApiError ? reason.message : '请确认当前密码后重试。')
    changingPassword.value = false
    return
  }

  passwordForm.currentPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  passwordSubmitted.value = false
  changingPassword.value = false

  resetApiSecurityState()
  session.state.authenticated = false
  session.state.user = undefined
  session.state.expiresAt = undefined
  session.state.agent = undefined

  toast.success('密码已修改', '请使用新密码重新登录。')
  await router.replace({ name: 'login' })
}

onMounted(async () => {
  try {
    capabilities.value = await api.agent.capabilities()
  } catch {
    capabilities.value = []
  }
})
</script>

<template>
  <div class="page page--narrow">
    <PageHeader title="设置" description="账户与设备偏好。宿主机安全策略由 Agent 配置控制，不能在浏览器中放宽。">
      <template #actions>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="refreshAgent">
          <RefreshCw :size="16" :class="{ spin: refreshing }" /> 检查连接
        </button>
      </template>
    </PageHeader>

    <section class="settings-section panel-card">
      <header class="settings-section__header">
        <span><ShieldCheck :size="19" /></span>
        <div><h2>管理账户</h2><p>当前登录身份与会话信息</p></div>
      </header>
      <div class="account-card">
        <span class="avatar avatar--large">{{ session.state.user?.username?.slice(0, 1).toUpperCase() || 'A' }}</span>
        <div>
          <strong>{{ session.state.user?.displayName || session.state.user?.username || '管理员' }}</strong>
          <small>{{ session.state.user?.role || 'administrator' }}</small>
        </div>
        <StatusBadge status="connected" label="当前会话" subtle />
      </div>
      <dl class="settings-list">
        <div>
          <dt><Clock3 :size="17" /> Session 到期时间</dt>
          <dd>{{ formatDateTime(session.state.expiresAt) }}</dd>
        </div>
        <div>
          <dt><KeyRound :size="17" /> 身份验证</dt>
          <dd>{{ session.state.user?.totpEnabled ? '已启用 TOTP' : '密码登录' }}</dd>
        </div>
      </dl>
      <p class="settings-note">TOTP 配置将在账户管理接口实现后显示；宿主机能力按真实适配器状态呈现。</p>
    </section>

    <section class="settings-section panel-card">
      <header class="settings-section__header">
        <span><KeyRound :size="19" /></span>
        <div><h2>修改密码</h2><p>更新当前管理员账户的登录凭据</p></div>
      </header>
      <form class="form-stack password-form" novalidate @submit.prevent="changePassword">
        <label class="field">
          <span>当前密码</span>
          <input
            v-model="passwordForm.currentPassword"
            type="password"
            name="current-password"
            autocomplete="current-password"
            :aria-invalid="passwordSubmitted && passwordForm.currentPassword.length === 0"
            required
          />
          <small v-if="passwordSubmitted && passwordForm.currentPassword.length === 0">请输入当前密码。</small>
        </label>

        <label class="field">
          <span>新密码</span>
          <input
            v-model="passwordForm.newPassword"
            type="password"
            name="new-password"
            autocomplete="new-password"
            minlength="12"
            :aria-invalid="passwordSubmitted && !passwordChecks.every((item) => item.valid)"
            required
          />
        </label>

        <div class="password-checks" aria-label="新密码要求">
          <span v-for="check in passwordChecks" :key="check.label" :class="{ 'is-valid': check.valid }">
            <i aria-hidden="true" /> {{ check.label }}
          </span>
        </div>

        <label class="field">
          <span>确认新密码</span>
          <input
            v-model="passwordForm.confirmPassword"
            type="password"
            name="confirm-password"
            autocomplete="new-password"
            minlength="12"
            :aria-invalid="passwordSubmitted && passwordForm.newPassword !== passwordForm.confirmPassword"
            required
          />
          <small v-if="passwordSubmitted && passwordForm.newPassword !== passwordForm.confirmPassword">
            两次输入的密码不一致。
          </small>
        </label>

        <button class="button button--primary" type="submit" :disabled="changingPassword">
          <LoaderCircle v-if="changingPassword" class="spin" :size="17" />
          {{ changingPassword ? '正在修改…' : '修改密码' }}
        </button>
      </form>
      <p class="settings-note">修改成功后当前会话将立即失效，需要使用新密码重新登录。</p>
    </section>

    <section class="settings-section panel-card">
      <header class="settings-section__header">
        <span><Sun :size="19" /></span>
        <div><h2>界面主题</h2><p>仅保存在当前浏览器，不上传服务器</p></div>
      </header>
      <div class="theme-options">
        <button
          v-for="option in themes"
          :key="option.id"
          type="button"
          :class="{ 'is-active': theme.preference.value === option.id }"
          @click="theme.setTheme(option.id)"
        >
          <span><component :is="option.icon" :size="19" /></span>
          <strong>{{ option.label }}</strong>
          <small>{{ option.description }}</small>
          <Check v-if="theme.preference.value === option.id" class="theme-options__check" :size="17" />
        </button>
      </div>
    </section>

    <section class="settings-section panel-card">
      <header class="settings-section__header">
        <span><Server :size="19" /></span>
        <div><h2>宿主机 Agent</h2><p>面板唯一的特权操作边界</p></div>
        <StatusBadge :status="agentState.status" :label="agentState.label" />
      </header>
      <dl class="settings-list settings-list--agent">
        <div>
          <dt>Agent 版本</dt>
          <dd>{{ panel.state.agent?.version || '—' }}</dd>
        </div>
        <div>
          <dt>协议版本</dt>
          <dd>{{ panel.state.agent?.protocolVersion || '—' }}</dd>
        </div>
        <div>
          <dt>最后检查</dt>
          <dd>{{ relativeTime(panel.state.agent?.lastSeenAt) }}</dd>
        </div>
        <div>
          <dt>已开放能力</dt>
          <dd>{{ capabilities.filter((item) => item.enabled).length }} / {{ capabilities.length }}</dd>
        </div>
      </dl>
      <div v-if="panel.state.agent?.reason" class="inline-alert inline-alert--warning">{{ panel.state.agent.reason }}</div>
      <p class="settings-note">
        Web 容器不挂载 Docker Socket 或宿主机根目录；Agent 只通过本地 Unix Socket 接收类型化动作。
      </p>
    </section>

    <section class="settings-section panel-card">
      <header class="settings-section__header">
        <span><Scale :size="19" /></span>
        <div><h2>开源许可</h2><p>GNU AGPL v3.0 only</p></div>
      </header>
      <p class="settings-note">
        KPanel 源代码采用 AGPL-3.0-only；第三方组件继续使用各自的原始许可。
      </p>
      <div class="license-actions">
        <a
          class="button button--ghost"
          href="https://github.com/kejilion/KPanel"
          target="_blank"
          rel="noopener noreferrer"
        >
          查看源码 <ExternalLink :size="15" />
        </a>
        <a
          class="button button--ghost"
          href="https://github.com/kejilion/KPanel/blob/main/LICENSE"
          target="_blank"
          rel="noopener noreferrer"
        >
          查看许可协议 <ExternalLink :size="15" />
        </a>
      </div>
    </section>
  </div>
</template>

<style scoped>
.password-form {
  max-width: 560px;
  padding: 18px;
}

.password-form > .button {
  justify-self: start;
  min-width: 132px;
}

.license-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
}

@media (max-width: 640px) {
  .password-form {
    max-width: none;
  }

  .password-form > .button {
    width: 100%;
  }

  .license-actions .button {
    width: 100%;
  }
}
</style>
