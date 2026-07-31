<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Box, Cpu, Database, HardDrive, MemoryStick, Network, RefreshCw } from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import TrendChart, { type TrendSeries } from '@/components/monitoring/TrendChart.vue'
import { ApiError, api } from '@/lib/api'
import { formatBytes, formatDateTime, formatPercent, formatRate } from '@/lib/format'
import {
  monitoringTargetId,
  normalizeMonitoringMetric,
  type MonitoringMetric,
} from '@/lib/monitoringNavigation'
import { isHistoricalContainer, newestContainerSampleTime } from '@/lib/monitoringPresentation'
import type {
  MonitoringContainerSeries,
  MonitoringHistory,
  MonitoringHostPoint,
  MonitoringRange,
} from '@/types/api'

const ranges: Array<{ value: MonitoringRange; label: string }> = [
  { value: '1h', label: '1 小时' },
  { value: '6h', label: '6 小时' },
  { value: '24h', label: '24 小时' },
  { value: '7d', label: '7 天' },
]

const history = shallowRef<MonitoringHistory>()
const route = useRoute()
const selectedRange = ref<MonitoringRange>('24h')
const selectedContainerId = ref('')
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const diskChartMode = ref<'capacity' | 'io'>('capacity')
const networkChartMode = ref<'traffic' | 'connections'>('traffic')
let controller: AbortController | undefined

const latestHost = computed<MonitoringHostPoint | undefined>(() => history.value?.host.at(-1))
const selectedContainer = computed<MonitoringContainerSeries | undefined>(() => {
  const containers = history.value?.containers || []
  return containers.find((item) => item.containerId === selectedContainerId.value) || containers[0]
})
const latestContainer = computed(() => selectedContainer.value?.points.at(-1))
const newestContainerSample = computed(() => newestContainerSampleTime(history.value?.containers || []))
const selectedMetric = computed<MonitoringMetric | undefined>(() => normalizeMonitoringMetric(route.query.metric))

const hostCPU = computed<TrendSeries[]>(() => [
  {
    label: 'CPU',
    color: 'var(--brand)',
    points: (history.value?.host || []).map((point) => ({ at: point.collectedAt, value: point.cpuPercent })),
  },
  {
    label: '1 分钟负载占核',
    color: 'var(--violet)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.cpuCores > 0 ? (point.loadOne / point.cpuCores) * 100 : 0,
    })),
  },
])
const hostMemory = computed<TrendSeries[]>(() => {
  const series: TrendSeries[] = [{
    label: '内存',
    color: 'var(--blue)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: percent(point.memoryUsedBytes, point.memoryTotalBytes),
    })),
  }]
  if ((history.value?.host || []).some((point) => point.swapTotalBytes > 0)) {
    series.push({
      label: 'Swap',
      color: 'var(--violet)',
      points: (history.value?.host || []).map((point) => ({
        at: point.collectedAt,
        value: percent(point.swapUsedBytes, point.swapTotalBytes),
      })),
    })
  }
  return series
})
const hostDiskCapacity = computed<TrendSeries[]>(() => [{
  label: '系统盘使用率',
  color: 'var(--amber)',
  points: (history.value?.host || []).map((point) => ({ at: point.collectedAt, value: point.diskPercent })),
}])
const hostDiskIO = computed<TrendSeries[]>(() => [
  {
    label: '读取',
    color: 'var(--blue)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.diskReadBytesPerSecond || 0,
    })),
  },
  {
    label: '写入',
    color: 'var(--amber)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.diskWriteBytesPerSecond || 0,
    })),
  },
])
const activeHostDisk = computed(() => diskChartMode.value === 'io' ? hostDiskIO.value : hostDiskCapacity.value)
const hostNetworkTraffic = computed<TrendSeries[]>(() => [
  {
    label: '下载',
    color: 'var(--brand)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.networkRxBytesPerSecond,
    })),
  },
  {
    label: '上传',
    color: 'var(--blue)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.networkTxBytesPerSecond,
    })),
  },
])
const hostNetworkConnections = computed<TrendSeries[]>(() => [
  {
    label: 'TCP',
    color: 'var(--brand)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.tcpConnections,
    })),
  },
  {
    label: 'UDP',
    color: 'var(--violet)',
    points: (history.value?.host || []).map((point) => ({
      at: point.collectedAt,
      value: point.udpConnections,
    })),
  },
])
const activeHostNetwork = computed(() => networkChartMode.value === 'traffic'
  ? hostNetworkTraffic.value
  : hostNetworkConnections.value)
const containerCPU = computed<TrendSeries[]>(() => [{
  label: 'CPU',
  color: 'var(--brand)',
  points: (selectedContainer.value?.points || []).map((point) => ({
    at: point.collectedAt,
    value: point.cpuPercent,
  })),
}])
const containerMemory = computed<TrendSeries[]>(() => [{
  label: '内存',
  color: 'var(--violet)',
  points: (selectedContainer.value?.points || []).map((point) => ({
    at: point.collectedAt,
    value: point.memoryPercent,
  })),
}])
const containerNetwork = computed<TrendSeries[]>(() => [
  {
    label: '接收',
    color: 'var(--brand)',
    points: (selectedContainer.value?.points || []).map((point) => ({
      at: point.collectedAt,
      value: point.networkRxBytesPerSecond,
    })),
  },
  {
    label: '发送',
    color: 'var(--blue)',
    points: (selectedContainer.value?.points || []).map((point) => ({
      at: point.collectedAt,
      value: point.networkTxBytesPerSecond,
    })),
  },
])
const containerBlock = computed<TrendSeries[]>(() => [
  {
    label: '块读取',
    color: 'var(--brand)',
    points: (selectedContainer.value?.points || []).map((point) => ({
      at: point.collectedAt,
      value: point.blockReadBytesPerSecond || 0,
    })),
  },
  {
    label: '块写入',
    color: 'var(--amber)',
    points: (selectedContainer.value?.points || []).map((point) => ({
      at: point.collectedAt,
      value: point.blockWriteBytesPerSecond || 0,
    })),
  },
])

function percent(used?: number, total?: number): number {
  if (!used || !total) return 0
  return Math.min(100, Math.max(0, (used / total) * 100))
}

function selectContainer(container: MonitoringContainerSeries): void {
  selectedContainerId.value = container.containerId
}

function containerIsHistorical(container: MonitoringContainerSeries): boolean {
  return isHistoricalContainer(container, newestContainerSample.value)
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''
  try {
    const result = await api.monitoring.history(selectedRange.value, controller.signal)
    history.value = result
    if (!result.containers.some((item) => item.containerId === selectedContainerId.value)) {
      selectedContainerId.value = result.containers[0]?.containerId || ''
    }
  } catch (reason) {
    if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
      error.value = reason instanceof ApiError
        ? reason.message
        : '无法读取历史监控数据，请检查 Agent 状态后重试。'
    }
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function changeRange(range: MonitoringRange): void {
  if (range === selectedRange.value) return
  selectedRange.value = range
  void load()
}

function chartIsSelected(...metrics: MonitoringMetric[]): boolean {
  return selectedMetric.value !== undefined && metrics.includes(selectedMetric.value)
}

async function focusSelectedMetric(): Promise<void> {
  const metric = selectedMetric.value
  if (!metric || !history.value?.host.length) return
  await nextTick()
  document.getElementById(monitoringTargetId(metric))?.scrollIntoView({
    behavior: 'smooth',
    block: 'center',
  })
}

watch([selectedMetric, () => history.value?.host.length], () => void focusSelectedMetric(), { flush: 'post' })

onMounted(() => void load())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <section class="monitoring-page">
    <PageHeader title="历史监控" description="轻量沉淀主机与容器资源趋势，数据仅保存在当前服务器。">
      <template #actions>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="16" :class="{ 'is-spinning': refreshing }" />
          刷新
        </button>
      </template>
    </PageHeader>

    <div class="monitoring-toolbar" aria-label="监控时间范围">
      <button
        v-for="range in ranges"
        :key="range.value"
        class="range-button"
        :class="{ 'range-button--active': selectedRange === range.value }"
        type="button"
        @click="changeRange(range.value)"
      >
        {{ range.label }}
      </button>
      <span v-if="history?.storage.lastSampleAt" class="monitoring-toolbar__meta">
        最近采样 {{ formatDateTime(history.storage.lastSampleAt) }}
      </span>
    </div>

    <LoadingState v-if="loading" :rows="4" cards label="正在读取历史监控数据" />
    <ErrorState v-else-if="error" title="历史监控读取失败" :message="error" @retry="load()" />
    <template v-else-if="history">
      <div class="summary-grid">
        <article class="summary-card">
          <span class="summary-card__icon"><Cpu :size="19" /></span>
          <div><span>CPU</span><strong>{{ formatPercent(latestHost?.cpuPercent) }}</strong></div>
          <small>{{ latestHost?.cpuCores || 0 }} 核 · 负载 {{ latestHost?.loadOne.toFixed(2) || '0.00' }}</small>
        </article>
        <article class="summary-card">
          <span class="summary-card__icon is-blue"><MemoryStick :size="19" /></span>
          <div><span>内存</span><strong>{{ formatPercent(percent(latestHost?.memoryUsedBytes, latestHost?.memoryTotalBytes)) }}</strong></div>
          <small>{{ formatBytes(latestHost?.memoryUsedBytes) }} / {{ formatBytes(latestHost?.memoryTotalBytes) }}</small>
        </article>
        <article class="summary-card">
          <span class="summary-card__icon is-amber"><HardDrive :size="19" /></span>
          <div><span>系统盘</span><strong>{{ formatPercent(latestHost?.diskPercent) }}</strong></div>
          <small>{{ formatBytes(latestHost?.diskUsedBytes) }} / {{ formatBytes(latestHost?.diskTotalBytes) }}</small>
        </article>
        <article class="summary-card">
          <span class="summary-card__icon is-violet"><Database :size="19" /></span>
          <div><span>历史数据</span><strong>{{ formatBytes(history.storage.storageBytes) }}</strong></div>
          <small>保留 {{ history.storage.retentionDays }} 天 · 上限 {{ formatBytes(history.storage.maxStorageBytes) }}</small>
        </article>
      </div>

      <div v-if="history.storage.lastError || history.storage.storageLimitReached" class="monitoring-warning">
        {{ history.storage.lastError || '历史数据已达到固定存储上限，系统将优先保留最新数据。' }}
      </div>

      <div v-if="history.host.length" class="chart-grid">
        <article
          id="host-cpu-load-history"
          class="chart-card"
          :class="{ 'chart-card--selected': chartIsSelected('cpu', 'load') }"
        >
          <header><div><Cpu :size="18" /><strong>CPU 与负载</strong></div><span>{{ history.host.length }} 个点</span></header>
          <TrendChart :series="hostCPU" :formatter="formatPercent" :max-value="100" />
        </article>
        <article
          id="host-memory-history"
          class="chart-card"
          :class="{ 'chart-card--selected': chartIsSelected('memory') }"
        >
          <header><div><MemoryStick :size="18" /><strong>内存</strong></div><span>内存 / Swap</span></header>
          <TrendChart :series="hostMemory" :formatter="formatPercent" :max-value="100" />
        </article>
        <article
          id="host-disk-history"
          class="chart-card"
          :class="{ 'chart-card--selected': chartIsSelected('disk') }"
        >
          <header>
            <div><HardDrive :size="18" /><strong>磁盘</strong></div>
            <div class="chart-switch" aria-label="磁盘指标">
              <button type="button" :class="{ 'is-active': diskChartMode === 'capacity' }" @click="diskChartMode = 'capacity'">容量</button>
              <button type="button" :class="{ 'is-active': diskChartMode === 'io' }" @click="diskChartMode = 'io'">读写 I/O</button>
            </div>
          </header>
          <TrendChart
            :series="activeHostDisk"
            :formatter="diskChartMode === 'io' ? formatRate : formatPercent"
            :max-value="diskChartMode === 'capacity' ? 100 : undefined"
          />
        </article>
        <article
          id="host-network-history"
          class="chart-card"
          :class="{ 'chart-card--selected': chartIsSelected('network') }"
        >
          <header>
            <div><Network :size="18" /><strong>网络与连接</strong></div>
            <div class="chart-switch" aria-label="网络指标">
              <button type="button" :class="{ 'is-active': networkChartMode === 'traffic' }" @click="networkChartMode = 'traffic'">流量</button>
              <button type="button" :class="{ 'is-active': networkChartMode === 'connections' }" @click="networkChartMode = 'connections'">连接数</button>
            </div>
          </header>
          <TrendChart :series="activeHostNetwork" :formatter="networkChartMode === 'traffic' ? formatRate : (value) => value.toFixed(0)" />
        </article>
      </div>
      <EmptyState v-else title="正在积累历史数据" description="功能启用后约 1 分钟生成首个主机采样点，刷新页面即可查看。" />

      <section class="container-section">
        <header class="section-heading">
          <div>
            <span class="section-heading__icon"><Box :size="18" /></span>
            <div><h2>容器监控</h2><p>运行中容器每 5 分钟采样一次，最多记录 32 个。</p></div>
          </div>
          <span>{{ history.containers.length }} 个容器有历史数据</span>
        </header>

        <div v-if="history.containers.length" class="container-layout">
          <div class="container-list">
            <button
              v-for="container in history.containers"
              :key="container.containerId"
              class="container-row"
              :class="{
                'container-row--active': selectedContainer?.containerId === container.containerId,
                'container-row--historical': containerIsHistorical(container),
              }"
              type="button"
              @click="selectContainer(container)"
            >
              <span>
                <span class="container-row__title">
                  <strong>{{ container.name }}</strong>
                  <em v-if="containerIsHistorical(container)">历史</em>
                </span>
                <small>{{ container.image }}</small>
              </span>
              <span>
                <strong>{{ formatPercent(container.points.at(-1)?.cpuPercent) }}</strong>
                <small>{{ formatBytes(container.points.at(-1)?.memoryBytes) }}</small>
              </span>
            </button>
          </div>
          <div class="container-detail">
            <header>
              <div><h3>{{ selectedContainer?.name }}</h3><p>{{ selectedContainer?.image }}</p></div>
              <div class="container-detail__latest">
                <span>CPU <strong>{{ formatPercent(latestContainer?.cpuPercent) }}</strong></span>
                <span>内存 <strong>{{ formatBytes(latestContainer?.memoryBytes) }}</strong></span>
                <span>进程 <strong>{{ latestContainer?.pids || 0 }}</strong></span>
              </div>
            </header>
            <div class="container-charts">
              <article><strong>CPU</strong><TrendChart :series="containerCPU" :formatter="formatPercent" :max-value="100" /></article>
              <article><strong>内存</strong><TrendChart :series="containerMemory" :formatter="formatPercent" :max-value="100" /></article>
              <article><strong>磁盘 I/O</strong><TrendChart :series="containerBlock" :formatter="formatRate" /></article>
              <article><strong>网络</strong><TrendChart :series="containerNetwork" :formatter="formatRate" /></article>
            </div>
          </div>
        </div>
        <EmptyState v-else title="暂无容器历史数据" description="没有运行中的 Docker 容器，或首轮容器采样尚未完成。" />
      </section>

      <footer class="monitoring-footnote">
        采样间隔：主机 {{ history.storage.hostIntervalSeconds }} 秒，容器
        {{ history.storage.containerIntervalSeconds }} 秒。查询读取
        {{ formatBytes(history.scannedBytes) }}，跳过 {{ history.skippedLines }} 条异常记录。
      </footer>
    </template>
  </section>
</template>

<style scoped>
.monitoring-page { display: grid; gap: 18px; }
.monitoring-toolbar {
  display: flex; align-items: center; gap: 8px; padding: 6px; border: 1px solid var(--border);
  border-radius: 12px; background: var(--surface); box-shadow: var(--shadow-sm);
}
.range-button {
  min-height: 34px; padding: 0 14px; border: 0; border-radius: 8px;
  color: var(--text-soft); background: transparent; cursor: pointer;
}
.range-button:hover, .range-button--active { color: var(--brand-strong); background: var(--brand-soft); }
.monitoring-toolbar__meta { margin-left: auto; padding-right: 8px; color: var(--muted); font-size: .78rem; }
.summary-grid, .chart-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.summary-card, .chart-card, .container-section {
  border: 1px solid var(--border); border-radius: 14px; background: var(--surface); box-shadow: var(--shadow-sm);
}
.summary-card { display: grid; grid-template-columns: auto 1fr; gap: 10px 12px; padding: 16px; }
.summary-card__icon, .section-heading__icon {
  display: grid; width: 38px; height: 38px; place-items: center; border-radius: 10px;
  background: var(--brand-soft); color: var(--brand);
}
.summary-card__icon.is-blue { color: var(--blue); background: var(--blue-soft); }
.summary-card__icon.is-violet { color: var(--violet); background: var(--violet-soft); }
.summary-card__icon.is-amber { color: var(--amber); background: var(--amber-soft); }
.summary-card div { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
.summary-card span, .summary-card small { color: var(--muted); font-size: .78rem; }
.summary-card strong { color: var(--text); font-size: 1.25rem; }
.summary-card small { grid-column: 1 / -1; }
.monitoring-warning {
  padding: 11px 14px; border: 1px solid color-mix(in srgb, var(--amber) 35%, var(--border));
  border-radius: 10px; color: var(--amber); background: var(--amber-soft); font-size: .82rem;
}
.chart-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.chart-card { min-width: 0; padding: 16px; }
.chart-card--selected {
  border-color: color-mix(in srgb, var(--brand) 62%, var(--border));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand) 14%, transparent), var(--shadow-sm);
}
.chart-card--wide { grid-column: 1 / -1; }
.chart-card > header, .section-heading, .container-detail > header {
  display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 10px;
}
.chart-card > header div, .section-heading > div { display: flex; align-items: center; gap: 8px; }
.chart-card > header span, .section-heading > span { color: var(--muted); font-size: .76rem; }
.chart-switch { display: inline-flex !important; gap: 2px !important; padding: 3px; border: 1px solid var(--border); border-radius: 9px; background: var(--surface-subtle); }
.chart-switch button { min-height: 26px; padding: 0 9px; border: 0; border-radius: 6px; color: var(--muted); background: transparent; font-size: .72rem; cursor: pointer; }
.chart-switch button:hover, .chart-switch button.is-active { color: var(--brand-strong); background: var(--brand-soft); }
.container-section { padding: 18px; }
.section-heading h2, .container-detail h3 { margin: 0; font-size: 1rem; }
.section-heading p, .container-detail p { margin: 3px 0 0; color: var(--muted); font-size: .76rem; }
.container-layout { display: grid; grid-template-columns: minmax(220px, 300px) minmax(0, 1fr); gap: 14px; }
.container-list { max-height: 430px; overflow: auto; padding-right: 4px; }
.container-row {
  display: flex; width: 100%; align-items: center; justify-content: space-between; gap: 12px;
  padding: 11px 12px; border: 1px solid transparent; border-radius: 10px;
  color: var(--text); background: transparent; text-align: left; cursor: pointer;
}
.container-row + .container-row { margin-top: 4px; }
.container-row:hover, .container-row--active {
  border-color: color-mix(in srgb, var(--brand) 40%, var(--border)); background: var(--brand-soft);
}
.container-row--historical { color: var(--muted); background: var(--surface-subtle); opacity: .66; }
.container-row--historical:hover, .container-row--historical.container-row--active { opacity: 1; }
.container-row > span { min-width: 0; }
.container-row > span:last-child { flex: 0 0 auto; text-align: right; }
.container-row strong, .container-row small { display: block; }
.container-row__title { display: flex; min-width: 0; align-items: center; gap: 7px; }
.container-row__title strong { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.container-row__title em {
  flex: 0 0 auto; padding: 1px 5px; border-radius: 999px; color: var(--muted);
  background: color-mix(in srgb, var(--muted) 12%, transparent); font-size: .62rem; font-style: normal;
}
.container-row small {
  max-width: 180px; margin-top: 3px; overflow: hidden; color: var(--muted);
  font-size: .7rem; text-overflow: ellipsis; white-space: nowrap;
}
.container-detail {
  min-width: 0; padding: 14px; border: 1px solid var(--border);
  border-radius: 12px; background: var(--surface-subtle);
}
.container-detail__latest { display: flex; gap: 14px; color: var(--muted); font-size: .74rem; }
.container-detail__latest strong { color: var(--text); }
.container-charts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.container-charts > article { min-width: 0; padding: 12px; border: 1px solid var(--border); border-radius: 11px; background: var(--surface); }
.container-charts > article > strong { display: block; margin-bottom: 4px; font-size: .78rem; }
.monitoring-footnote { color: var(--muted); font-size: .74rem; text-align: center; }
@media (max-width: 1180px) {
  .summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .container-charts { grid-template-columns: 1fr; }
}
@media (max-width: 780px) {
  .monitoring-toolbar { flex-wrap: wrap; }
  .monitoring-toolbar__meta { width: 100%; margin-left: 0; padding: 4px 8px; }
  .summary-grid, .chart-grid, .container-layout { grid-template-columns: 1fr; }
  .chart-card--wide { grid-column: auto; }
  .container-list { max-height: 240px; }
  .container-detail > header { align-items: flex-start; flex-direction: column; }
}
</style>
