<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  label?: string
  subtle?: boolean
}>()

const statusMap: Record<string, { label: string; tone: string }> = {
  healthy: { label: '健康', tone: 'success' },
  running: { label: '运行中', tone: 'success' },
  valid: { label: '有效', tone: 'success' },
  succeeded: { label: '已完成', tone: 'success' },
  success: { label: '成功', tone: 'success' },
  synced: { label: '已同步', tone: 'success' },
  managed: { label: '可管理', tone: 'success' },
  connected: { label: '已连接', tone: 'success' },
  online: { label: '在线', tone: 'success' },
  warning: { label: '需关注', tone: 'warning' },
  degraded: { label: '异常', tone: 'warning' },
  stale: { label: '数据过期', tone: 'warning' },
  expiring: { label: '即将过期', tone: 'warning' },
  drifted: { label: '存在漂移', tone: 'warning' },
  ambiguous: { label: '待确认', tone: 'warning' },
  conflicted: { label: '存在冲突', tone: 'danger' },
  unsupported: { label: '暂不支持', tone: 'neutral' },
  restarting: { label: '重启中', tone: 'warning' },
  queued: { label: '排队中', tone: 'info' },
  pending: { label: '处理中', tone: 'info' },
  running_job: { label: '执行中', tone: 'info' },
  observed: { label: '已观测', tone: 'info' },
  static: { label: '静态站点', tone: 'info' },
  proxy: { label: '反向代理', tone: 'info' },
  php: { label: 'PHP 站点', tone: 'info' },
  stopped: { label: '已停止', tone: 'neutral' },
  exited: { label: '已退出', tone: 'neutral' },
  created: { label: '已创建', tone: 'neutral' },
  read_only: { label: '结构待适配', tone: 'neutral' },
  'read-only': { label: '结构待适配', tone: 'neutral' },
  unmanaged: { label: '外部资源', tone: 'neutral' },
  unknown: { label: '未知', tone: 'neutral' },
  critical: { label: '严重', tone: 'danger' },
  failed: { label: '失败', tone: 'danger' },
  failed_rolled_back: { label: '失败，已回滚', tone: 'warning' },
  failed_needs_attention: { label: '失败，需处理', tone: 'danger' },
  interrupted: { label: '已中断', tone: 'danger' },
  cancelled: { label: '已取消', tone: 'neutral' },
  failure: { label: '失败', tone: 'danger' },
  denied: { label: '已拒绝', tone: 'danger' },
  expired: { label: '已过期', tone: 'danger' },
  missing: { label: '缺失', tone: 'danger' },
  dead: { label: '异常退出', tone: 'danger' },
  offline: { label: '离线', tone: 'danger' },
  auth_failed: { label: '授权失效', tone: 'danger' },
  tls_error: { label: '证书异常', tone: 'danger' },
  incompatible: { label: '版本不兼容', tone: 'danger' },
}

const normalizedStatus = computed(() => props.status.toLowerCase().replace(/\s+/g, '_'))
const config = computed(() => statusMap[normalizedStatus.value] || { label: props.status || '未知', tone: 'neutral' })
</script>

<template>
  <span class="status-badge" :class="[`is-${config.tone}`, { 'is-subtle': subtle }]">
    <span class="status-badge__dot" aria-hidden="true" />
    {{ label || config.label }}
  </span>
</template>
