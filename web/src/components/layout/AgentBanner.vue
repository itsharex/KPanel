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
      detail: agent.reason || '当前 Panel 无法调用该 Agent 的写入协议，请同步升级两端版本。',
      icon: ShieldAlert,
    }
  }
  if (agent.readOnly) {
    return {
      tone: 'warning',
      title: 'Agent 写入依赖未就绪',
      detail: agent.reason || '可以查看实时资源；请按提示补齐宿主机依赖后执行变更。',
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
