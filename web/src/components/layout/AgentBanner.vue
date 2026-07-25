<script setup lang="ts">
import { computed } from 'vue'
import { LockKeyhole, PlugZap, ShieldAlert } from '@lucide/vue'
import { usePanelState } from '@/stores/panel'

const panel = usePanelState()

const banner = computed(() => {
  const agent = panel.state.agent
  if (!agent) return undefined
  if (!agent.connected) {
    return {
      tone: 'danger',
      title: '宿主机 Agent 已离线',
      detail: agent.reason || '当前仅展示最后一次观测结果，所有管理操作已禁用。',
      icon: PlugZap,
    }
  }
  if (!agent.compatible) {
    return {
      tone: 'danger',
      title: 'Agent 协议版本不兼容',
      detail: agent.reason || '为防止错误修改，面板已自动进入只读模式。',
      icon: ShieldAlert,
    }
  }
  if (agent.readOnly) {
    return {
      tone: 'warning',
      title: '面板当前为只读模式',
      detail: agent.reason || '可以安全查看资源，但暂时不能执行变更。',
      icon: LockKeyhole,
    }
  }
  return undefined
})
</script>

<template>
  <div v-if="banner" class="agent-banner" :class="`agent-banner--${banner.tone}`" role="status">
    <component :is="banner.icon" :size="18" aria-hidden="true" />
    <strong>{{ banner.title }}</strong>
    <span>{{ banner.detail }}</span>
  </div>
</template>
