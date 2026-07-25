<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Check, Clock3, KeyRound, Monitor, Moon, RefreshCw, Server, ShieldCheck, Sun } from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { formatDateTime, relativeTime } from '@/lib/format'
import { usePanelState } from '@/stores/panel'
import { useSession } from '@/stores/session'
import { useTheme, type ThemePreference } from '@/stores/theme'
import { useToast } from '@/stores/toast'

const session = useSession()
const panel = usePanelState()
const theme = useTheme()
const toast = useToast()
const refreshing = ref(false)
const capabilities = ref<Array<{ id: string; enabled: boolean; reason?: string }>>([])

const agentState = computed(() => {
  const agent = panel.state.agent
  if (!agent?.connected) return { status: 'offline', label: '离线' }
  if (!agent.compatible) return { status: 'incompatible', label: '不兼容' }
  if (agent.readOnly) return { status: 'read_only', label: '只读' }
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
      <p class="settings-note">密码修改与 TOTP 配置将在安全接口开放后显示；前端不会绕过能力白名单。</p>
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
  </div>
</template>
