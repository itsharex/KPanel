<script setup lang="ts">
import { computed } from 'vue'
import { LockKeyhole, PlugZap, ShieldAlert } from '@lucide/vue'
import { useI18n } from '@/i18n'
import { usePanelState } from '@/stores/panel'

const panel = usePanelState()
const i18n = useI18n()

const banner = computed(() => {
  const agent = panel.state.agent
  if (!agent) return undefined
  if (!agent.connected) {
    return {
      tone: 'danger',
      title: i18n.t('agent.bannerOfflineTitle'),
      detail: agent.reason || i18n.t('agent.bannerOfflineDetail'),
      icon: PlugZap,
    }
  }
  if (!agent.compatible) {
    return {
      tone: 'danger',
      title: i18n.t('agent.bannerIncompatibleTitle'),
      detail: agent.reason || i18n.t('agent.bannerIncompatibleDetail'),
      icon: ShieldAlert,
    }
  }
  if (agent.readOnly) {
    return {
      tone: 'warning',
      title: i18n.t('agent.bannerReadOnlyTitle'),
      detail: agent.reason || i18n.t('agent.bannerReadOnlyDetail'),
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
