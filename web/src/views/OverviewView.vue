<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Activity,
  ArrowDownToLine,
  ArrowUpFromLine,
  Box,
  Boxes,
  Clock3,
  Cpu,
  Database,
  Gauge,
  Globe2,
  HardDrive,
  MemoryStick,
  RefreshCw,
  Server,
  ShieldCheck,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import MetricCard from '@/components/overview/MetricCard.vue'
import { ApiError, api } from '@/lib/api'
import { clampPercent, formatBytes, formatDateTime, formatDuration, formatPercent } from '@/lib/format'
import { usePanelState } from '@/stores/panel'
import type { SystemOverview } from '@/types/api'

const data = ref<SystemOverview>()
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const panel = usePanelState()
let controller: AbortController | undefined
let refreshTimer: number | undefined

const loadPercent = computed(() => {
  const cores = Number(data.value?.load.unit || 1)
  return clampPercent(((data.value?.load.value || 0) / Math.max(cores, 1)) * 100)
})

const agentLabel = computed(() => {
  const agent = data.value?.agent
  if (!agent?.connected) return { status: 'offline', label: 'Agent 离线' }
  if (!agent.compatible) return { status: 'incompatible', label: '版本不兼容' }
  if (agent.readOnly) return { status: 'read_only', label: '只读模式' }
  return { status: 'connected', label: '运行正常' }
})

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''

  try {
    data.value = await api.overview.get(controller.signal)
    panel.setAgent(data.value.agent)
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    error.value = reason instanceof ApiError ? reason.message : '无法读取主机状态。'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

onMounted(() => {
  void load()
  refreshTimer = window.setInterval(() => void load(true), 20_000)
})

onBeforeUnmount(() => {
  controller?.abort()
  if (refreshTimer) window.clearInterval(refreshTimer)
})
</script>

<template>
  <div class="page">
    <PageHeader title="主机概览" description="来自宿主机 Agent 的实时观测数据，不缓存为系统真相。">
      <template #actions>
        <span v-if="data" class="observed-at">
          <Clock3 :size="15" /> {{ formatDateTime(data.observedAt) }}
        </span>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="16" :class="{ spin: refreshing }" />
          刷新
        </button>
      </template>
    </PageHeader>

    <LoadingState v-if="loading" :rows="4" cards />
    <ErrorState v-else-if="error && !data" :message="error" @retry="load()" />

    <template v-else-if="data">
      <div v-if="error" class="inline-alert inline-alert--warning" role="status">
        自动刷新暂时失败，正在显示上一次观测结果。
      </div>

      <section class="metric-grid" aria-label="主机资源">
        <MetricCard
          label="CPU"
          :icon="Cpu"
          :value="formatPercent(data.cpu.percent)"
          :percent="data.cpu.percent"
          detail="当前总使用率"
        />
        <MetricCard
          label="内存"
          :icon="MemoryStick"
          tone="blue"
          :value="formatPercent(data.memory.percent)"
          :percent="data.memory.percent"
          :detail="`${formatBytes(data.memory.value)} / ${formatBytes(data.memory.total)}`"
        />
        <MetricCard
          label="系统盘"
          :icon="HardDrive"
          tone="violet"
          :value="formatPercent(data.disk.percent)"
          :percent="data.disk.percent"
          :detail="`${formatBytes(data.disk.value)} / ${formatBytes(data.disk.total)}`"
        />
        <MetricCard
          label="1 分钟负载"
          :icon="Gauge"
          tone="amber"
          :value="data.load.value.toFixed(2)"
          :percent="loadPercent"
          :detail="`${data.load.unit || '—'} 个 CPU 核心`"
        />
      </section>

      <div class="overview-grid">
        <section class="panel-card panel-card--system">
          <header class="panel-card__header">
            <div>
              <span class="panel-card__icon"><Server :size="18" /></span>
              <div>
                <h2>{{ data.hostname || '未命名主机' }}</h2>
                <p>系统信息</p>
              </div>
            </div>
            <StatusBadge :status="agentLabel.status" :label="agentLabel.label" />
          </header>

          <dl class="detail-list detail-list--grid">
            <div>
              <dt>操作系统</dt>
              <dd>{{ data.os || '—' }}</dd>
            </div>
            <div>
              <dt>内核</dt>
              <dd>{{ data.kernel || '—' }}</dd>
            </div>
            <div>
              <dt>架构</dt>
              <dd>{{ data.architecture || '—' }}</dd>
            </div>
            <div>
              <dt>运行时间</dt>
              <dd>{{ formatDuration(data.uptimeSeconds) }}</dd>
            </div>
          </dl>

          <div class="network-summary">
            <div>
              <span><ArrowDownToLine :size="16" /> 累计接收</span>
              <strong>{{ formatBytes(data.network.totalReceivedBytes) }}</strong>
            </div>
            <div>
              <span><ArrowUpFromLine :size="16" /> 累计发送</span>
              <strong>{{ formatBytes(data.network.totalTransmittedBytes) }}</strong>
            </div>
          </div>
        </section>

        <section class="panel-card">
          <header class="panel-card__header">
            <div>
              <span class="panel-card__icon panel-card__icon--blue"><Activity :size="18" /></span>
              <div>
                <h2>资源与一致性</h2>
                <p>当前发现结果</p>
              </div>
            </div>
          </header>

          <div class="resource-summary">
            <RouterLink to="/sites" class="resource-summary__item">
              <span class="resource-summary__icon resource-summary__icon--brand"><Globe2 :size="20" /></span>
              <span>
                <strong>{{ data.sites?.total ?? '—' }}</strong>
                <small>已发现网站</small>
              </span>
              <em v-if="data.sites">{{ data.sites.drifted }} 个待核对</em>
            </RouterLink>
            <RouterLink to="/docker" class="resource-summary__item">
              <span class="resource-summary__icon resource-summary__icon--blue"><Box :size="20" /></span>
              <span>
                <strong>{{ data.containers?.total ?? '—' }}</strong>
                <small>Docker 容器</small>
              </span>
              <em v-if="data.containers">{{ data.containers.running }} 个运行中</em>
            </RouterLink>
            <RouterLink to="/audit" class="resource-summary__item">
              <span class="resource-summary__icon resource-summary__icon--violet"><ShieldCheck :size="20" /></span>
              <span>
                <strong>完整</strong>
                <small>操作审计</small>
              </span>
              <em>查看记录</em>
            </RouterLink>
          </div>
        </section>
      </div>

      <section v-if="data.services.length" class="panel-card">
        <header class="panel-card__header">
          <div>
            <span class="panel-card__icon panel-card__icon--violet"><Database :size="18" /></span>
            <div>
              <h2>核心服务</h2>
              <p>固定白名单服务状态</p>
            </div>
          </div>
        </header>
        <div class="service-grid">
          <div v-for="service in data.services" :key="service.id" class="service-item">
            <span class="service-item__icon"><Boxes :size="17" /></span>
            <span>
              <strong>{{ service.name }}</strong>
              <small>{{ service.version || service.detail || '未报告版本' }}</small>
            </span>
            <StatusBadge :status="service.state" />
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
