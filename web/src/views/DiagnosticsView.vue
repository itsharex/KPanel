<script setup lang="ts">
import { computed, inject, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from '@/i18n'
import { usePhraseCatalog } from '@/i18n/phrase'

usePhraseCatalog((locale) => locale === 'en-US'
  ? import('@/i18n/pages/DiagnosticsView/en-US').then((module) => module.default)
  : import('@/i18n/pages/DiagnosticsView/zh-TW').then((module) => module.default))
import {
  Activity,
  Cpu,
  ExternalLink,
  Gauge,
  Globe2,
  LoaderCircle,
  Menu,
  Network,
  PanelLeftClose,
  PanelLeftOpen,
  Play,
  Timer,
  TriangleAlert,
  X,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import AppInteractiveTerminal from '@/components/apps/AppInteractiveTerminal.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { desktopWindowActiveKey } from '@/lib/desktopRouteKeys'
import { formatDateTime } from '@/lib/format'
import { containWheelScroll } from '@/lib/scroll'
import { useToast } from '@/stores/toast'
import type {
  DiagnosticCatalog,
  DiagnosticCheck,
  DiagnosticJob,
  DiagnosticSummaryMetric,
} from '@/types/api'

const catalog = ref<DiagnosticCatalog>()
const jobs = ref<DiagnosticJob[]>([])
const selectedCheck = ref<DiagnosticCheck>()
const pendingCheck = ref<DiagnosticCheck>()
const activeJob = ref<DiagnosticJob>()
const runningJob = ref<DiagnosticJob>()
const loading = ref(true)
const starting = ref(false)
const error = ref('')
const viewMode = ref<'overview' | 'terminal'>('overview')
const commandsCollapsed = ref(false)
const mobileCommandsOpen = ref(false)
const pendingScore = ref(false)
const toast = useToast()
const i18n = useI18n()
const windowActive = inject(desktopWindowActiveKey, computed(() => true))
let controller: AbortController | undefined
let pollController: AbortController | undefined
let pollTimer: number | undefined
let pollFailures = 0
let pollGeneration = 0
let pollingJobID = ''
const activePollDelay = 2_000
const backgroundPollDelay = 15_000
const commandsCollapsedStorageKey = 'kpanel:diagnostics:commands-collapsed'

const categories = computed(() => catalog.value?.categories || [])
const groupedChecks = computed(() =>
  categories.value
    .map((category) => ({
      ...category,
      items: (catalog.value?.items || []).filter((item) => item.category === category.id),
    }))
    .filter((category) => category.items.length),
)
const testedCheckIDs = computed(() => new Set(
  jobs.value
    .filter((job) => job.status === 'succeeded' || job.status === 'failed')
    .map((job) => job.checkId),
))
const hasActiveJob = computed(
  () => [runningJob.value, activeJob.value].some((job) =>
    job?.status === 'queued' || job?.status === 'running',
  ),
)
const activeLog = computed(() => activeJob.value?.logs.join('\n') || '等待脚本输出…')
const scoreDimensions = [
  {
    id: 'performance',
    label: '性能',
    detail: 'CPU · 磁盘 · 系统吞吐',
    nativeID: 'native-cpu',
    category: 'hardware',
    keywords: ['yabs', 'cpu', '性能'],
    icon: Cpu,
    tone: 'performance',
  },
  {
    id: 'route',
    label: '路由',
    detail: '三网回程与线路质量',
    nativeID: 'native-route',
    category: 'network',
    keywords: ['besttrace', 'mtr', 'nxtrace', 'backtrace', '回程', '路由', '线路'],
    icon: Network,
    tone: 'route',
  },
  {
    id: 'latency',
    label: '延迟',
    detail: '往返时延 · 抖动 · 丢包',
    nativeID: 'native-latency',
    category: 'network',
    keywords: ['net-quality', 'tcp-quality', '网络质量', '延迟', '抖动', '丢包'],
    icon: Activity,
    tone: 'latency',
  },
  {
    id: 'speed',
    label: '网速',
    detail: '多节点上下行带宽',
    nativeID: 'native-speed',
    category: 'network',
    keywords: ['superspeed', 'speedtest', '测速', '带宽', '网速'],
    icon: Gauge,
    tone: 'speed',
  },
  {
    id: 'ip',
    label: 'IP 质量',
    detail: '风险 · 信誉 · 解锁能力',
    nativeID: 'native-ip-quality',
    category: 'access',
    keywords: ['ip-quality', 'IP 质量', '信誉'],
    icon: Globe2,
    tone: 'ip',
  },
] as const

type ScoreDimension = (typeof scoreDimensions)[number]

const scoreCheck = computed(() => {
  const items = catalog.value?.items || []
  const native = items.find((item) => item.id === 'native-comprehensive')
  if (native) return native
  const comprehensive = items.filter((item) => item.category === 'comprehensive')
  return comprehensive.find((item) => /NodeQuality|融合怪/i.test(item.name)) || comprehensive[0]
})
const scoreJob = computed(() => {
  const checkID = scoreCheck.value?.id
  if (!checkID) return undefined
  if (activeJob.value?.checkId === checkID) return activeJob.value
  return jobs.value.find((job) => job.checkId === checkID)
})
const scoreSummaryMetricCount = computed(() => scoreDimensions.reduce(
  (total, dimension) => total + summaryMetrics(dimension).length,
  0,
))
const scoreState = computed<'ready' | 'running' | 'completed' | 'failed' | 'busy' | 'unavailable'>(() => {
  if (!scoreCheck.value) return 'unavailable'
  if (scoreJob.value?.status === 'queued' || scoreJob.value?.status === 'running') return 'running'
  if (scoreJob.value?.status === 'succeeded') return 'completed'
  if (scoreJob.value?.status === 'failed') return 'failed'
  if (hasActiveJob.value) return 'busy'
  return 'ready'
})
const scoreProgress = computed(() => {
  if (scoreState.value === 'completed') return 100
  if (scoreState.value === 'running') return Math.max(0, Math.min(100, scoreJob.value?.progress || 0))
  return 0
})
const scoreRingOffset = computed(() => 351.86 * (1 - scoreProgress.value / 100))
const scoreRunLabel = computed(() => {
  switch (scoreState.value) {
    case 'unavailable': return '核心体检未就绪'
    case 'running': return '跑分进行中'
    case 'completed': return '再次开始跑分'
    case 'failed': return '重新开始跑分'
    case 'busy': return '终端测试进行中'
    default: return '开始一键跑分'
  }
})
const scoreStateLabel = computed(() => {
  switch (scoreState.value) {
    case 'unavailable': return '未就绪'
    case 'running': return '进行中'
    case 'completed': return '已完成'
    case 'failed': return '需要重试'
    case 'busy': return '单项测试中'
    default: return '待开始'
  }
})
const scoreRouteStatusLabel = computed(() => {
  switch (scoreState.value) {
    case 'unavailable': return '未就绪'
    case 'running': return '实时采集'
    case 'completed': return '已完成'
    case 'failed': return '需要重试'
    case 'busy': return '单项测试中'
    default: return '待开始'
  }
})
const scoreCenterLabel = computed(() => {
  switch (scoreState.value) {
    case 'running': return `${scoreProgress.value}%`
    case 'completed': return '完成'
    case 'failed': return '重试'
    case 'busy': return '—'
    default: return '—'
  }
})
const scoreStatusLabel = computed(() => {
  switch (scoreState.value) {
    case 'unavailable': return '等待 KPanel 核心体检能力就绪'
    case 'running': return scoreJob.value?.message || '正在采集综合体检数据'
    case 'completed': return scoreSummaryMetricCount.value > 0
      ? '真实测试结果已汇总，原始结果已保存在终端记录'
      : '综合测试完成，原始结果已保存在终端记录'
    case 'failed': return scoreJob.value?.message || '上次综合测试未完成'
    case 'busy': return '请先完成当前终端中的体检任务'
    default: return '从性能、路由、延迟、网速和 IP 质量开始检查'
  }
})

function categoryName(id: string): string {
  if (i18n.locale.value === 'en-US') {
    const labels: Record<string, string> = {
      access: 'IP & Access',
      network: 'Network',
      hardware: 'Hardware',
      benchmark: 'Benchmarks',
      core: 'KPanel Core',
    }
    return labels[id] || id
  }
  return categories.value.find((item) => item.id === id)?.name || id
}

function checkNameLabel(value: string): string {
  const labels: Record<string, string> = i18n.locale.value === 'en-US'
    ? {
        'ChatGPT 解锁检测': 'ChatGPT access check',
        'IP 质量体检': 'IP quality check',
        'SuperSpeed 三网测速': 'SuperSpeed network test',
        '网络质量体检': 'Network quality check',
        'YABS 性能测试': 'YABS benchmark',
        'NodeQuality 综合测评': 'Server health check',
        'KPanel 核心综合体检': 'KPanel core health check',
        'CPU 原生跑分': 'Native CPU benchmark',
        '内存原生跑分': 'Native memory benchmark',
        '硬盘原生跑分': 'Native disk benchmark',
        '出口路由基础检测': 'Native egress route check',
        '延迟原生检测': 'Native latency check',
        '网速原生测速': 'Native speed test',
        'IP 基础质量检测': 'Native IP quality check',
      }
    : {
      'NodeQuality 综合测评': '服务器综合体检',
      'KPanel 核心综合体检': 'KPanel 核心综合体检',
      }
  return labels[value] || value
}

function categoryIcon(id: string) {
  if (id === 'access') return Globe2
  if (id === 'network') return Network
  if (id === 'hardware') return Cpu
  return Gauge
}

function impactLabel(impact: DiagnosticCheck['impact']): string {
  if (impact === 'light') return '轻量检测'
  if (impact === 'network') return '消耗网络流量'
  return '高负载跑分'
}

function impactClass(impact: DiagnosticCheck['impact']): string {
  return `is-${impact}`
}

function sourceHost(value: string): string {
  try {
    return new URL(value).hostname
  } catch {
    return value
  }
}

function dimensionCheck(dimension: ScoreDimension): DiagnosticCheck | undefined {
  const items = catalog.value?.items || []
  const native = items.find((item) => item.id === dimension.nativeID)
  if (native) return native
  return items.find((item) =>
    item.category === dimension.category &&
    dimension.keywords.some((keyword) => `${item.id} ${item.name} ${item.description}`.toLowerCase().includes(keyword.toLowerCase())),
  )
}

function summaryMetrics(dimension: ScoreDimension): DiagnosticSummaryMetric[] {
  const sources = [dimensionJob(dimension), scoreJob.value]
  const metrics: DiagnosticSummaryMetric[] = []
  const seen = new Set<string>()
  for (const job of sources) {
    for (const metric of job?.summary?.dimensions?.[dimension.id]?.metrics || []) {
      if (seen.has(metric.key)) continue
      seen.add(metric.key)
      metrics.push(metric)
    }
  }
  const priority = dimension.id === 'ip'
    ? ['public_ip', 'quality', 'asn', 'country', 'location', 'reverse_dns', 'ipv4_ipv6', 'colo', 'unlock', 'host']
    : dimension.id === 'performance'
      ? ['cpu_score', 'memory_score', 'disk_read', 'disk_write', 'cpu_model', 'cpu_cores']
      : dimension.id === 'latency'
        ? ['average', 'jitter', 'loss', 'min', 'max']
        : dimension.id === 'speed'
          ? ['download', 'upload']
          : dimension.id === 'route'
            ? ['path', 'edge', 'location', 'average']
            : []
  metrics.sort((left, right) => {
    const leftIndex = priority.indexOf(left.key)
    const rightIndex = priority.indexOf(right.key)
    return (leftIndex < 0 ? priority.length : leftIndex) - (rightIndex < 0 ? priority.length : rightIndex)
  })
  return metrics
}

function summaryMetricLimit(dimension: ScoreDimension): number {
  return dimension.id === 'performance' ? 3 : 2
}

function summaryMetricLabel(key: string): string {
  const labels: Record<string, string> = {
    cpu_model: 'CPU',
    cpu_cores: '核心',
    memory: '内存',
    disk: '磁盘',
    disk_read: '磁盘读',
    disk_write: '磁盘写',
    disk_total: '磁盘总计',
    geekbench_single: '单核',
    geekbench_multi: '多核',
    cpu_score: 'CPU 分数',
    memory_score: '内存分数',
    memory_read: '内存读',
    memory_write: '内存写',
    upload: '上行',
    download: '下行',
    average: '平均延迟',
    jitter: '抖动',
    loss: '丢包',
    path: '线路',
    isp: '运营商',
    asn: 'ASN',
    host: '网络归属',
    country: '地区',
    location: '位置',
    ipv4_ipv6: 'IP 连通性',
    public_ip: '公网 IP',
    reverse_dns: '反向解析',
    colo: '边缘节点',
    edge: '探测节点',
    quality: 'IP 质量',
    unlock: '解锁',
    ct_average: '电信',
    cu_average: '联通',
    cm_average: '移动',
  }
  return labels[key] || key
}

function dimensionJob(dimension: ScoreDimension): DiagnosticJob | undefined {
  const check = dimensionCheck(dimension)
  if (!check) return undefined
  if (activeJob.value?.checkId === check.id) return activeJob.value
  return jobs.value.find((job) => job.checkId === check.id)
}

function summaryJobForOverview(): DiagnosticJob | undefined {
  if (scoreJob.value) return scoreJob.value
  for (const dimension of scoreDimensions) {
    const job = dimensionJob(dimension)
    if (job?.summary) return job
  }
  return undefined
}

const summaryReportURL = computed(() => {
  const jobsForSummary = [scoreJob.value, ...scoreDimensions.map((dimension) => dimensionJob(dimension))]
  return jobsForSummary.find((job) => job?.summary?.reportUrl)?.summary?.reportUrl || ''
})

function dimensionState(dimension: ScoreDimension): 'idle' | 'running' | 'completed' | 'failed' | 'covered' {
  const job = dimensionJob(dimension)
  if (job?.status === 'queued' || job?.status === 'running') return 'running'
  if (job?.status === 'succeeded') return 'completed'
  if (job?.status === 'failed') return 'failed'
  if (scoreState.value === 'running') return 'running'
  if (scoreState.value === 'completed') return 'completed'
  if (scoreState.value === 'failed') return 'failed'
  if (!dimensionCheck(dimension) && scoreCheck.value) return 'covered'
  return 'idle'
}

function dimensionStateLabel(dimension: ScoreDimension): string {
  const state = dimensionState(dimension)
  if (state === 'running') return '单项测试中'
  if (state === 'completed') return summaryMetrics(dimension).length > 0 ? '已汇总' : '单项已完成'
  if (state === 'failed') return '上次未完成'
  if (state === 'covered') return '综合入口覆盖'
  if (scoreState.value === 'running') return '综合测试中'
  if (scoreState.value === 'completed') return '已输出到终端'
  return dimensionCheck(dimension) ? '等待测试' : '脚本未提供'
}

function dimensionStateClass(dimension: ScoreDimension): string {
  return `is-${dimensionState(dimension)}`
}

function selectOverview(): void {
  viewMode.value = 'overview'
  mobileCommandsOpen.value = false
}

function openDimension(dimension: ScoreDimension): void {
  const check = dimensionCheck(dimension)
  if (check) selectCheck(check)
}

function openScoreTerminal(): void {
  const job = scoreJob.value
  if (!job) return
  openJob(job)
  viewMode.value = 'terminal'
}

function openSummaryTerminal(): void {
  const job = summaryJobForOverview()
  if (!job) return
  openJob(job)
  viewMode.value = 'terminal'
}

function stopPolling(): void {
  pollGeneration += 1
  pollingJobID = ''
  if (pollTimer) window.clearTimeout(pollTimer)
  pollTimer = undefined
  pollController?.abort()
  pollController = undefined
}

function schedulePoll(id: string, generation: number, delay: number): void {
  if (generation !== pollGeneration || pollingJobID !== id) return
  if (pollTimer) window.clearTimeout(pollTimer)
  pollTimer = window.setTimeout(() => {
    pollTimer = undefined
    void refreshJob(id, generation)
  }, delay)
}

async function refreshJob(id: string, generation = pollGeneration): Promise<void> {
  if (pollController) return
  const requestController = new AbortController()
  pollController = requestController
  try {
    const next = await api.diagnostics.job(id, requestController.signal)
    if (generation !== pollGeneration || pollController !== requestController) return
    pollFailures = 0
    const previous = jobs.value.find((item) => item.id === id)?.status
      || (activeJob.value?.id === id ? activeJob.value.status : undefined)
    if (runningJob.value?.id === id) runningJob.value = next
    if (activeJob.value?.id === id) activeJob.value = next
    const index = jobs.value.findIndex((item) => item.id === next.id)
    if (index >= 0) jobs.value.splice(index, 1, next)
    else jobs.value.unshift(next)
    if (next.status === 'succeeded' || next.status === 'failed') {
      stopPolling()
      if (previous === 'queued' || previous === 'running') {
        if (next.status === 'succeeded') toast.success(`${checkNameLabel(next.checkName)}已完成`)
        else toast.danger(`${checkNameLabel(next.checkName)}执行失败`, next.message)
      }
      if (runningJob.value?.id === id) runningJob.value = undefined
    }
  } catch (reason) {
    if (
      generation === pollGeneration &&
      pollController === requestController &&
      !(reason instanceof DOMException && reason.name === 'AbortError')
    ) {
      pollFailures += 1
      if (pollFailures >= 3) {
        toast.danger('体检进度刷新中断', '后台任务可能仍在运行，请稍后点击刷新重新连接。')
        stopPolling()
      }
    }
  } finally {
    if (pollController === requestController) pollController = undefined
    if (
      generation === pollGeneration &&
      pollingJobID === id &&
      hasActiveJob.value
    ) {
      schedulePoll(
        id,
        generation,
        windowActive.value ? activePollDelay : backgroundPollDelay,
      )
    }
  }
}

function startPolling(job: DiagnosticJob, immediate = windowActive.value): void {
  stopPolling()
  pollFailures = 0
  activeJob.value = job
  if (job.status === 'queued' || job.status === 'running') runningJob.value = job
  pollingJobID = job.id
  const generation = pollGeneration
  if (immediate) void refreshJob(job.id, generation)
  else schedulePoll(job.id, generation, backgroundPollDelay)
}

async function load(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    const [nextCatalog, history] = await Promise.all([
      api.diagnostics.catalog(controller.signal),
      api.diagnostics.jobs(controller.signal),
    ])
    catalog.value = nextCatalog
    jobs.value = history.items
    if (selectedCheck.value && !nextCatalog.items.some((item) => item.id === selectedCheck.value?.id)) {
      selectedCheck.value = undefined
    }
    const current = history.items.find((item) => item.id === activeJob.value?.id)
    if (current) activeJob.value = current
    const active = history.items.find((item) => item.status === 'queued' || item.status === 'running')
    if (active) {
      runningJob.value = active
      selectedCheck.value = nextCatalog.items.find((item) => item.id === active.checkId)
      viewMode.value = 'terminal'
      startPolling(active)
    } else {
      runningJob.value = undefined
      viewMode.value = 'overview'
      stopPolling()
    }
  } catch (reason) {
    if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
      error.value = reason instanceof ApiError ? reason.message : '无法读取体检项目，请检查 Agent 与 kejilion.sh 版本。'
    }
  } finally {
    loading.value = false
  }
}

async function startCheck(check: DiagnosticCheck, keepOverview = false): Promise<void> {
  if (!check || starting.value || hasActiveJob.value) return
  starting.value = true
  try {
    const job = await api.diagnostics.start(check.id)
    jobs.value.unshift(job)
    selectedCheck.value = check
    if (!keepOverview) viewMode.value = 'terminal'
    startPolling(job)
    toast.success(
      keepOverview ? '综合跑分已开始' : `${checkNameLabel(check.name)}已开始`,
      check.provider === 'native'
        ? 'KPanel 原生探针正在后台运行，结果会实时回显并汇总。'
        : '任务已在后台运行；第三方脚本需要确认时可直接在终端输入。',
    )
  } catch (reason) {
    toast.danger(
      keepOverview ? '综合跑分启动失败' : '体检任务启动失败',
      reason instanceof ApiError ? reason.message : '请检查 Agent、systemd 和 kejilion.sh 是否正常并保持版本一致。',
    )
  } finally {
    starting.value = false
  }
}

async function confirmStart(): Promise<void> {
  const check = pendingScore.value ? scoreCheck.value : pendingCheck.value
  if (!check) return
  const keepOverview = pendingScore.value
  pendingScore.value = false
  pendingCheck.value = undefined
  await startCheck(check, keepOverview)
}

function openJob(job: DiagnosticJob): void {
  activeJob.value = job
  viewMode.value = 'terminal'
  const check = catalog.value?.items.find((item) => item.id === job.checkId)
  if (check) {
    selectedCheck.value = check
  }
  if (job.status === 'queued' || job.status === 'running') {
    if (pollingJobID !== job.id) startPolling(job)
  } else if (pollingJobID === job.id) {
    stopPolling()
  }
}

function selectCheck(check: DiagnosticCheck): void {
  selectedCheck.value = check
  viewMode.value = 'terminal'
  mobileCommandsOpen.value = false
  const matchingJob = jobs.value.find((job) => job.checkId === check.id)
  if (matchingJob) {
    openJob(matchingJob)
  } else {
    // Keep polling the background task for status/lock purposes, but never
    // let it keep its terminal focus after the user selected another check.
    activeJob.value = undefined
  }
}

function requestCheck(check: DiagnosticCheck): void {
  selectCheck(check)
  pendingCheck.value = check
}

function requestScore(): void {
  if (!scoreCheck.value || hasActiveJob.value || starting.value) return
  pendingScore.value = true
}

function runCheckLabel(check: DiagnosticCheck): string {
  return `运行 ${checkNameLabel(check.name)}`
}

function containLogWheel(event: WheelEvent): void {
  containWheelScroll(event, event.currentTarget as HTMLElement)
}

function toggleCommands(): void {
  commandsCollapsed.value = !commandsCollapsed.value
  try {
    window.localStorage.setItem(
      commandsCollapsedStorageKey,
      commandsCollapsed.value ? '1' : '0',
    )
  } catch {
    // Collapsing remains available when storage is blocked.
  }
}

watch(windowActive, (active) => {
  const job = activeJob.value
  if (!job || (job.status !== 'queued' && job.status !== 'running')) return
  startPolling(job, active)
})

onMounted(() => {
  try {
    commandsCollapsed.value = window.localStorage.getItem(commandsCollapsedStorageKey) === '1'
  } catch {
    commandsCollapsed.value = false
  }
  void load()
})
onBeforeUnmount(() => {
  controller?.abort()
  stopPolling()
})
</script>

<template>
  <div class="diagnostics-page">
    <PageHeader
      title="体检"
      description="KPanel 原生探针负责核心跑分，第三方脚本作为可选增值服务补充线路与解锁测试。"
    />

    <LoadingState v-if="loading" title="正在读取体检项目" description="正在准备 KPanel 原生探针与可选脚本目录。" />
    <ErrorState v-else-if="error" title="体检功能暂不可用" :message="error" @retry="load()" />

    <template v-else-if="catalog">
      <section
        class="diagnostic-workbench"
        :class="{
          'is-command-panel-collapsed': commandsCollapsed,
          'is-command-drawer-open': mobileCommandsOpen,
        }"
      >
        <button
          v-if="mobileCommandsOpen"
          class="diagnostic-command-overlay"
          type="button"
          aria-label="关闭体检项目选择"
          @click="mobileCommandsOpen = false"
        />
        <aside id="diagnostic-command-drawer" class="diagnostic-command-panel">
          <div class="diagnostic-command-panel__toolbar">
            <button
              class="diagnostic-command-panel__toggle diagnostic-command-panel__desktop-toggle"
              type="button"
              aria-controls="diagnostic-command-selector"
              :aria-expanded="!commandsCollapsed"
              :title="commandsCollapsed ? '展开体检列表' : '收起体检列表'"
              :aria-label="commandsCollapsed ? '展开体检列表' : '收起体检列表'"
              @click="toggleCommands"
            >
              <PanelLeftOpen v-if="commandsCollapsed" :size="17" />
              <PanelLeftClose v-else :size="17" />
            </button>
            <button
              class="diagnostic-command-panel__toggle diagnostic-command-panel__mobile-close"
              type="button"
              aria-label="关闭体检项目选择"
              @click="mobileCommandsOpen = false"
            >
              <X :size="17" />
            </button>
          </div>
          <button
            class="diagnostic-command-overview"
            :class="{ 'is-active': viewMode === 'overview' }"
            type="button"
            :aria-current="viewMode === 'overview' ? 'page' : undefined"
            @click="selectOverview"
          >
            <span class="diagnostic-command-overview__icon"><Gauge :size="17" /></span>
            <span class="diagnostic-command-overview__copy">
              <strong>综合跑分</strong>
              <small>默认首页</small>
            </span>
            <i class="diagnostic-command-overview__dot" :class="`is-${scoreState}`" />
          </button>
          <div id="diagnostic-command-selector" v-if="groupedChecks.length" v-show="!commandsCollapsed || mobileCommandsOpen" class="diagnostic-command-list">
            <section
              v-for="group in groupedChecks"
              :key="group.id"
              class="diagnostic-command-group"
              :class="`is-category-${group.id}`"
            >
              <header>
                <span>{{ categoryName(group.id) }}</span>
                <small>{{ group.items.length }}</small>
              </header>
              <div
                v-for="check in group.items"
                :key="check.id"
                class="diagnostic-command-row"
                :class="[`is-category-${check.category}`, { 'is-active': viewMode === 'terminal' && selectedCheck?.id === check.id }]"
              >
                <button class="diagnostic-command-select" type="button" @click="selectCheck(check)">
                  <span class="diagnostic-card__icon"><component :is="categoryIcon(check.category)" :size="17" /></span>
                  <strong>{{ checkNameLabel(check.name) }}</strong>
                  <span class="diagnostic-command-badges">
                    <small v-if="check.provider === 'native'" class="diagnostic-command-native">KPanel</small>
                    <small v-if="testedCheckIDs.has(check.id)" class="diagnostic-command-tested">已测</small>
                  </span>
                </button>
                <button
                  class="diagnostic-command-run"
                  type="button"
                  :disabled="hasActiveJob || starting"
                  :title="runCheckLabel(check)"
                  :aria-label="runCheckLabel(check)"
                  @click="requestCheck(check)"
                >
                  <LoaderCircle v-if="starting && pendingCheck?.id === check.id" :size="15" class="is-spinning" />
                  <Play v-else :size="15" />
                </button>
              </div>
            </section>
          </div>
          <div v-if="commandsCollapsed && !mobileCommandsOpen" class="diagnostic-command-rail" aria-label="收起的体检命令列表">
            <button
              class="diagnostic-command-rail__item diagnostic-command-rail__overview"
              :class="{ 'is-active': viewMode === 'overview' }"
              type="button"
              title="综合跑分"
              aria-label="综合跑分"
              @click="selectOverview"
            >
              <Gauge :size="17" />
            </button>
            <button
              v-for="check in catalog.items"
              :key="check.id"
              class="diagnostic-command-rail__item"
              :class="[`is-category-${check.category}`, { 'is-active': viewMode === 'terminal' && selectedCheck?.id === check.id }]"
              type="button"
              :title="checkNameLabel(check.name)"
              :aria-label="checkNameLabel(check.name)"
              @click="selectCheck(check)"
            >
              <component :is="categoryIcon(check.category)" :size="17" />
            </button>
          </div>
          <EmptyState v-if="(!commandsCollapsed || mobileCommandsOpen) && !groupedChecks.length" title="暂无体检项目" description="请刷新后重试。" />
        </aside>

        <section class="diagnostic-result" :class="{ 'is-overview': viewMode === 'overview' }">
          <div class="diagnostic-mobile-selector">
            <button
              type="button"
              aria-controls="diagnostic-command-drawer"
              :aria-expanded="mobileCommandsOpen"
              aria-label="打开体检项目选择"
              @click="mobileCommandsOpen = true"
            >
              <Menu :size="18" />
              <span>{{ viewMode === 'overview' ? '综合跑分' : (selectedCheck ? checkNameLabel(selectedCheck.name) : '选择体检项目') }}</span>
            </button>
            <small>{{ catalog.items.length }} 个项目</small>
          </div>
          <template v-if="viewMode === 'overview'">
            <div class="diagnostic-overview">
              <header class="diagnostic-overview__header">
                <div>
                  <div class="diagnostic-overview__eyebrow"><span><Gauge :size="15" /></span> 服务器体检工作台</div>
                  <h2>一键综合跑分</h2>
                  <p>先由 KPanel 原生探针完成核心检测，再保留第三方脚本作为可选增值服务。</p>
                </div>
                <span class="diagnostic-overview__status" :class="`is-${scoreState}`">
                  <i /> {{ scoreStateLabel }}
                </span>
              </header>

              <section class="diagnostic-score-hero" aria-labelledby="diagnostic-score-title">
                <div class="diagnostic-score-ring" :class="`is-${scoreState}`" aria-label="综合跑分进度">
                  <svg viewBox="0 0 140 140" aria-hidden="true">
                    <circle class="diagnostic-score-ring__track" cx="70" cy="70" r="56" />
                    <circle
                      class="diagnostic-score-ring__progress"
                      cx="70"
                      cy="70"
                      r="56"
                      :style="{ strokeDashoffset: scoreRingOffset }"
                    />
                  </svg>
                  <div class="diagnostic-score-ring__center">
                    <strong>{{ scoreCenterLabel }}</strong>
                    <span>{{ scoreState === 'running' ? '实时进度' : '综合状态' }}</span>
                  </div>
                </div>

                <div class="diagnostic-score-hero__copy">
                  <div class="diagnostic-score-hero__label">
                    综合跑分 <span>POWERED BY KPanel</span>
                    <em v-if="scoreSummaryMetricCount">真实结果已汇总</em>
                  </div>
                  <h3 id="diagnostic-score-title">{{ scoreCheck ? checkNameLabel(scoreCheck.name) : '等待综合评测脚本' }}</h3>
                  <p>{{ scoreStatusLabel }}</p>
                  <div class="diagnostic-score-meta">
                    <span><Timer :size="15" /> 约 {{ scoreCheck?.estimatedMinutes || '—' }} 分钟</span>
                    <span><Activity :size="15" /> {{ scoreCheck?.provider === 'native' ? '原生结果实时回显' : (scoreCheck ? '实时终端可追踪' : '请更新体检目录') }}</span>
                  </div>
                  <div class="diagnostic-score-actions">
                    <button
                      class="button button--primary"
                      type="button"
                      :disabled="!scoreCheck || hasActiveJob || starting"
                      @click="requestScore"
                    >
                      <LoaderCircle v-if="starting && pendingScore" :size="16" class="is-spinning" />
                      <Play v-else :size="16" />
                      {{ scoreRunLabel }}
                    </button>
                    <button
                      v-if="scoreJob"
                      class="button button--secondary"
                      type="button"
                      @click="openScoreTerminal"
                    >
                      查看终端
                    </button>
                  </div>
                </div>

                <div class="diagnostic-score-route" aria-label="综合跑分路径">
                  <div class="diagnostic-score-route__header">
                    <strong>五维体检路径</strong>
                    <span>{{ scoreRouteStatusLabel }}</span>
                  </div>
                  <div v-for="(dimension, index) in scoreDimensions" :key="dimension.id" class="diagnostic-score-route__item">
                    <span class="diagnostic-score-route__index">0{{ index + 1 }}</span>
                    <span>{{ dimension.label }}</span>
                    <i :class="dimensionStateClass(dimension)" />
                  </div>
                </div>
              </section>

              <section class="diagnostic-score-dimensions" aria-labelledby="diagnostic-score-dimensions-title">
                <header>
                  <div>
                    <h3 id="diagnostic-score-dimensions-title">体检维度</h3>
                    <p>综合入口统一启动原生检测；需要深挖或补充线路、解锁信息时，再点选第三方脚本。</p>
                  </div>
                  <span>{{ catalog.items.length }} 个检测项目可用</span>
                </header>
                <div class="diagnostic-score-dimension-grid">
                  <button
                    v-for="dimension in scoreDimensions"
                    :key="dimension.id"
                    class="diagnostic-score-dimension"
                    :class="[`is-tone-${dimension.tone}`, dimensionStateClass(dimension)]"
                    type="button"
                    :disabled="!dimensionCheck(dimension)"
                    :aria-label="dimensionCheck(dimension) ? `查看${dimension.label}测试` : `${dimension.label}脚本未提供`"
                    @click="openDimension(dimension)"
                  >
                    <span class="diagnostic-score-dimension__icon"><component :is="dimension.icon" :size="19" /></span>
                    <span class="diagnostic-score-dimension__copy">
                      <strong>{{ dimension.label }}</strong>
                      <span v-if="!summaryMetrics(dimension).length">{{ dimension.detail }}</span>
                      <span v-else class="is-summary">
                        <template v-for="(metric, metricIndex) in summaryMetrics(dimension).slice(0, summaryMetricLimit(dimension))" :key="metric.key">
                          <b>{{ summaryMetricLabel(metric.key) }}</b> {{ metric.value }}<i v-if="metricIndex < Math.min(summaryMetrics(dimension).length, summaryMetricLimit(dimension)) - 1"> · </i>
                        </template>
                      </span>
                    </span>
                    <span class="diagnostic-score-dimension__state"><i />{{ dimensionStateLabel(dimension) }}</span>
                  </button>
                </div>
              </section>

              <section class="diagnostic-score-note">
                <div class="diagnostic-score-note__icon"><Activity :size="17" /></div>
                <div>
                  <strong>{{ scoreSummaryMetricCount ? '真实结果已汇总' : '结果透明，原始输出不丢失' }}</strong>
                  <p>{{ scoreSummaryMetricCount ? (scoreCheck?.provider === 'native' ? '以下指标来自 KPanel 原生探针的实际回显；完整过程仍保留在任务日志中。' : '以下指标来自脚本原始输出；未识别项目仍保留在终端中。') : '页面只呈现真实任务状态与进度，不替脚本猜测分数；详细结果和第三方提示都可在终端中查看。' }}</p>
                </div>
                <a
                  v-if="summaryReportURL"
                  class="diagnostic-score-note__link"
                  :href="summaryReportURL"
                  target="_blank"
                  rel="noreferrer"
                >
                  查看完整报告 <ExternalLink :size="14" />
                </a>
                <button v-else-if="summaryJobForOverview()" class="diagnostic-score-note__link" type="button" @click="openSummaryTerminal">
                  打开最近结果 <ExternalLink :size="14" />
                </button>
              </section>
            </div>
          </template>
          <template v-else>
            <div v-if="hasActiveJob" class="diagnostic-progress" aria-label="任务进度">
              <span :style="{ width: `${activeJob?.progress || 0}%` }" />
            </div>
            <div v-if="!activeJob?.interactive" class="diagnostic-terminal-bar">
              <span><i :class="{ 'is-live': hasActiveJob }" /> {{ hasActiveJob ? '实时输出' : '终端输出' }}</span>
              <StatusBadge v-if="activeJob" :status="activeJob.status" />
            </div>
            <AppInteractiveTerminal
              v-if="activeJob?.interactive && windowActive"
              class="diagnostic-interactive-terminal"
              :job-id="activeJob.id"
              :input-open="activeJob.inputOpen"
              kind="diagnostic"
            />
            <p v-else-if="!activeJob" class="diagnostic-log diagnostic-log-empty" @wheel="containLogWheel">选择左侧体检命令，点击“开始体检”后在这里查看实时输出。</p>
            <pre v-else class="diagnostic-log" aria-live="polite" data-i18n-ignore @wheel="containLogWheel">{{ activeLog }}</pre>
            <footer v-if="activeJob">
              <span><Activity :size="14" /> {{ activeJob.message }}</span>
              <span><Timer :size="14" /> {{ formatDateTime(activeJob.startedAt || activeJob.createdAt) }}</span>
              <a v-if="activeJob.sourceUrl" :href="activeJob.sourceUrl" target="_blank" rel="noopener noreferrer">
                查看来源 <ExternalLink :size="13" />
              </a>
              <span v-else class="diagnostic-source">KPanel 原生探针</span>
            </footer>
            <footer v-else-if="selectedCheck">
              <span><Timer :size="14" /> 约 {{ selectedCheck.estimatedMinutes }} 分钟</span>
              <span class="impact-pill" :class="impactClass(selectedCheck.impact)">{{ impactLabel(selectedCheck.impact) }}</span>
              <a v-if="selectedCheck.sourceUrl" :href="selectedCheck.sourceUrl" target="_blank" rel="noopener noreferrer">
                {{ sourceHost(selectedCheck.sourceUrl) }} <ExternalLink :size="13" />
              </a>
              <span v-else class="diagnostic-source">KPanel 原生探针</span>
            </footer>
          </template>
        </section>
      </section>

    </template>

    <ModalDialog
      :open="Boolean(pendingCheck || pendingScore)"
      :title="pendingScore ? '确认开始一键跑分？' : (pendingCheck?.provider === 'native' ? '确认运行 KPanel 原生检测？' : '确认运行第三方体检？')"
      :description="pendingScore ? `${scoreCheck ? checkNameLabel(scoreCheck.name) : '综合评测'} · 预计 ${scoreCheck?.estimatedMinutes || '—'} 分钟` : (pendingCheck ? `${checkNameLabel(pendingCheck.name)} · 预计 ${pendingCheck.estimatedMinutes} 分钟` : '')"
      size="small"
      @close="pendingCheck = undefined; pendingScore = false"
    >
      <div v-if="pendingScore" class="diagnostic-confirm diagnostic-score-confirm">
        <Gauge :size="24" />
        <div>
          <p>
            {{ scoreCheck?.provider === 'native' ? '将使用 KPanel 原生探针完成 CPU、内存、硬盘、路由、延迟、测速和 IP 基础质量检测。测试期间会产生受控的 CPU、磁盘和网络开销。' : '将调用脚本目录中的综合评测入口，以真实终端输出完成一次多维度体检。测试期间可能消耗较多网络、CPU 或磁盘资源。' }}
          </p>
          <div class="diagnostic-score-confirm__list">
            <span v-for="dimension in scoreDimensions" :key="dimension.id"><i /> {{ dimension.label }}</span>
          </div>
        </div>
      </div>
      <div v-else-if="pendingCheck" class="diagnostic-confirm">
        <TriangleAlert :size="24" />
        <div>
          <p>
            {{ pendingCheck.provider === 'native' ? '此操作将运行 KPanel 原生探针，不安装第三方测试工具；硬盘和测速项目会产生受控的 I/O 或网络流量。' : '此操作将以 root 权限运行 kejilion.sh 中登记的第三方命令，可能安装测试工具并占用较多网络、CPU 或磁盘资源。' }}
          </p>
          <a v-if="pendingCheck.sourceUrl" :href="pendingCheck.sourceUrl" target="_blank" rel="noopener noreferrer">
            {{ pendingCheck.sourceUrl }} <ExternalLink :size="13" />
          </a>
          <span v-else class="diagnostic-source">KPanel 原生探针 · 不依赖第三方脚本</span>
        </div>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" :disabled="starting" @click="pendingCheck = undefined; pendingScore = false">
          取消
        </button>
        <button class="button button--primary" type="button" :disabled="starting" @click="confirmStart">
          <LoaderCircle v-if="starting" :size="16" class="is-spinning" />
          <Play v-else :size="16" />
          {{ starting ? '正在启动' : (pendingScore ? '确认开始跑分' : '确认开始') }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.diagnostics-page {
  display: grid;
  gap: 16px;
  min-height: 0;
}

.diagnostic-card p,
.diagnostic-result p {
  margin: 0;
}

.diagnostic-workbench {
  display: grid;
  grid-template-columns: minmax(270px, 310px) minmax(0, 1fr);
  height: var(--terminal-workspace-height);
  min-height: var(--terminal-workspace-min-height);
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--terminal-workspace-radius);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
  transition: grid-template-columns 180ms ease;
}

.diagnostic-workbench.is-command-panel-collapsed {
  grid-template-columns: 52px minmax(0, 1fr);
}

.diagnostic-workbench.is-command-panel-collapsed .diagnostic-command-overview {
  display: none;
}

.diagnostic-command-panel {
  position: relative;
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  min-width: 0;
  min-height: 0;
  border-right: 1px solid var(--border);
  background: color-mix(in srgb, var(--surface-subtle) 38%, var(--surface));
}

.diagnostic-command-panel__toolbar {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 8px 0;
}

.diagnostic-command-panel__toggle {
  position: static;
  z-index: 3;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-muted);
  background: var(--surface);
  cursor: pointer;
}

.diagnostic-command-panel__toggle:hover,
.diagnostic-command-panel__toggle:focus-visible {
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 50%, var(--border));
  outline: none;
}

.diagnostic-command-panel__mobile-close,
.diagnostic-command-overlay,
.diagnostic-mobile-selector {
  display: none;
}

.diagnostic-command-overview {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) auto;
  gap: 9px;
  align-items: center;
  min-width: 0;
  min-height: 58px;
  padding: 8px 12px;
  margin: 8px;
  border: 1px solid color-mix(in srgb, var(--brand) 24%, var(--border));
  border-radius: var(--radius);
  color: var(--text);
  background: color-mix(in srgb, var(--brand-soft) 42%, var(--surface));
  text-align: left;
  cursor: pointer;
  transition: border-color 160ms ease, background 160ms ease, transform 160ms ease;
}

.diagnostic-command-overview:hover,
.diagnostic-command-overview:focus-visible,
.diagnostic-command-overview.is-active {
  border-color: var(--brand);
  background: color-mix(in srgb, var(--brand-soft) 72%, var(--surface));
  outline: none;
}

.diagnostic-command-overview:active {
  transform: translateY(1px);
}

.diagnostic-command-overview__icon {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  color: var(--brand-strong);
  background: var(--brand-soft);
  border: 1px solid var(--brand-muted);
  border-radius: 10px;
}

.diagnostic-command-overview__copy {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.diagnostic-command-overview__copy strong {
  overflow: hidden;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.diagnostic-command-overview__copy small {
  color: var(--muted);
  font-size: 12px;
}

.diagnostic-command-overview__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--muted);
}

.diagnostic-command-overview__dot.is-running {
  background: var(--brand);
  box-shadow: 0 0 0 4px var(--brand-soft);
}

.diagnostic-command-overview__dot.is-completed {
  background: var(--brand-strong);
}

.diagnostic-command-overview__dot.is-failed {
  background: var(--danger);
}

.diagnostic-command-overview__dot.is-busy {
  background: var(--amber);
}

.diagnostic-command-overview__dot.is-unavailable {
  background: var(--border-strong);
}

.diagnostic-command-list {
  display: grid;
  align-content: start;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.diagnostic-command-rail {
  display: grid;
  min-height: 0;
  align-content: start;
  justify-items: center;
  gap: 7px;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 8px 7px 10px;
  scrollbar-width: none;
}

.diagnostic-command-rail::-webkit-scrollbar {
  display: none;
}

.diagnostic-command-rail__item {
  --diagnostic-category: var(--primary);

  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--diagnostic-category) 26%, transparent);
  border-radius: 9px;
  color: var(--diagnostic-category);
  background: color-mix(in srgb, var(--diagnostic-category) 10%, var(--surface));
  cursor: pointer;
}

.diagnostic-command-rail__overview {
  --diagnostic-category: var(--brand-strong);
  margin-bottom: 3px;
}

.diagnostic-command-rail__item:hover,
.diagnostic-command-rail__item:focus-visible,
.diagnostic-command-rail__item.is-active {
  border-color: var(--diagnostic-category);
  background: color-mix(in srgb, var(--diagnostic-category) 18%, var(--surface));
  outline: none;
}

.diagnostic-command-group {
  --diagnostic-category: var(--primary);
}

.diagnostic-command-group + .diagnostic-command-group {
  padding-top: 8px;
  margin-top: 8px;
  border-top: 1px dashed color-mix(in srgb, var(--diagnostic-category) 28%, var(--border));
}

.diagnostic-command-group > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 9px 5px;
  color: var(--diagnostic-category);
  font-size: 12px;
  font-weight: 700;
}

.diagnostic-command-group > header small {
  color: var(--text-tertiary);
  font-size: 12px;
}

.diagnostic-command-row {
  --diagnostic-category: var(--primary);

  display: grid;
  grid-template-columns: minmax(0, 1fr) 34px;
  gap: 4px;
  align-items: center;
  width: 100%;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
}

.diagnostic-command-group.is-category-access,
.diagnostic-command-row.is-category-access,
.diagnostic-command-rail__item.is-category-access { --diagnostic-category: #087a72; }
.diagnostic-command-group.is-category-network,
.diagnostic-command-row.is-category-network,
.diagnostic-command-rail__item.is-category-network { --diagnostic-category: #2563c4; }
.diagnostic-command-group.is-category-hardware,
.diagnostic-command-row.is-category-hardware,
.diagnostic-command-rail__item.is-category-hardware { --diagnostic-category: #965900; }
.diagnostic-command-group.is-category-benchmark,
.diagnostic-command-row.is-category-benchmark,
.diagnostic-command-rail__item.is-category-benchmark { --diagnostic-category: #7546c8; }
.diagnostic-command-group.is-category-comprehensive,
.diagnostic-command-row.is-category-comprehensive,
.diagnostic-command-rail__item.is-category-comprehensive { --diagnostic-category: #7546c8; }
.diagnostic-command-group.is-category-core,
.diagnostic-command-row.is-category-core,
.diagnostic-command-rail__item.is-category-core { --diagnostic-category: #15b8a6; }

:global(:root[data-theme='dark'] .diagnostic-command-group.is-category-access),
:global(:root[data-theme='dark'] .diagnostic-command-row.is-category-access),
:global(:root[data-theme='dark'] .diagnostic-command-rail__item.is-category-access) { --diagnostic-category: #4ecdc4; }
:global(:root[data-theme='dark'] .diagnostic-command-group.is-category-network),
:global(:root[data-theme='dark'] .diagnostic-command-row.is-category-network),
:global(:root[data-theme='dark'] .diagnostic-command-rail__item.is-category-network) { --diagnostic-category: #6ea8fe; }
:global(:root[data-theme='dark'] .diagnostic-command-group.is-category-hardware),
:global(:root[data-theme='dark'] .diagnostic-command-row.is-category-hardware),
:global(:root[data-theme='dark'] .diagnostic-command-rail__item.is-category-hardware) { --diagnostic-category: #f5b942; }
:global(:root[data-theme='dark'] .diagnostic-command-group.is-category-benchmark),
:global(:root[data-theme='dark'] .diagnostic-command-row.is-category-benchmark),
:global(:root[data-theme='dark'] .diagnostic-command-rail__item.is-category-benchmark),
:global(:root[data-theme='dark'] .diagnostic-command-group.is-category-comprehensive),
:global(:root[data-theme='dark'] .diagnostic-command-row.is-category-comprehensive),
:global(:root[data-theme='dark'] .diagnostic-command-rail__item.is-category-comprehensive) { --diagnostic-category: #b58cff; }
:global(:root[data-theme='dark'] .diagnostic-command-group.is-category-core),
:global(:root[data-theme='dark'] .diagnostic-command-row.is-category-core),
:global(:root[data-theme='dark'] .diagnostic-command-rail__item.is-category-core) { --diagnostic-category: #49d5c1; }

.diagnostic-command-select {
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr) auto;
  gap: 9px;
  align-items: center;
  min-width: 0;
  padding: 7px 6px 7px 8px;
  border: 0;
  color: var(--text);
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.diagnostic-command-row:hover {
  background: var(--surface);
}

.diagnostic-command-row.is-active {
  border-color: color-mix(in srgb, var(--diagnostic-category) 42%, var(--border));
  background: color-mix(in srgb, var(--diagnostic-category) 8%, var(--surface));
}

.diagnostic-command-select strong {
  overflow: hidden;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.diagnostic-command-badges {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  justify-content: flex-end;
  min-width: 0;
}

.diagnostic-command-native,
.diagnostic-command-tested {
  border-radius: 999px;
  padding: 2px 6px;
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.diagnostic-command-native {
  color: var(--diagnostic-category);
  background: color-mix(in srgb, var(--diagnostic-category) 14%, var(--surface));
}

.diagnostic-command-tested {
  border-radius: 999px;
  padding: 2px 6px;
  color: var(--diagnostic-category);
  background: color-mix(in srgb, var(--diagnostic-category) 12%, var(--surface));
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.diagnostic-card__icon {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  border-radius: 9px;
  color: var(--diagnostic-category);
  background: color-mix(in srgb, var(--diagnostic-category) 12%, var(--surface));
}

.diagnostic-command-run {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--diagnostic-category) 38%, var(--border));
  border-radius: 8px;
  color: var(--diagnostic-category);
  background: color-mix(in srgb, var(--diagnostic-category) 8%, var(--surface));
  cursor: pointer;
}

.diagnostic-command-run:hover:not(:disabled) {
  color: var(--surface);
  background: var(--diagnostic-category);
}

.diagnostic-command-run:disabled {
  cursor: not-allowed;
  opacity: .42;
}

.diagnostic-result.is-overview {
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: var(--border-strong) transparent;
}

.diagnostic-result.is-overview::-webkit-scrollbar {
  width: 8px;
}

.diagnostic-result.is-overview::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: var(--border-strong);
  background-clip: padding-box;
}

.diagnostic-overview {
  display: grid;
  align-content: start;
  gap: 18px;
  min-height: 100%;
  padding: 22px;
  background: color-mix(in srgb, var(--surface-subtle) 30%, var(--surface));
}

.diagnostic-overview__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
}

.diagnostic-overview__eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 8px;
  color: var(--brand-strong);
  font-size: 13px;
  font-weight: 700;
}

.diagnostic-overview__eyebrow > span {
  display: grid;
  width: 25px;
  height: 25px;
  place-items: center;
  color: var(--brand-strong);
  background: var(--brand-soft);
  border: 1px solid var(--brand-muted);
  border-radius: 8px;
}

.diagnostic-overview__header h2 {
  margin: 0;
  color: var(--text);
  font-size: clamp(20px, 2.4vw, 27px);
  line-height: 1.2;
}

.diagnostic-overview__header p {
  max-width: 600px;
  margin: 7px 0 0;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.6;
}

.diagnostic-overview__status {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 10px;
  color: var(--text-soft);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 13px;
  font-weight: 700;
}

.diagnostic-overview__status i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--muted);
}

.diagnostic-overview__status.is-running {
  color: var(--brand-strong);
  border-color: var(--brand-muted);
}

.diagnostic-overview__status.is-running i {
  background: var(--brand);
  box-shadow: 0 0 0 4px var(--brand-soft);
}

.diagnostic-overview__status.is-completed {
  color: var(--brand-strong);
  background: var(--brand-soft);
  border-color: var(--brand-muted);
}

.diagnostic-overview__status.is-completed i {
  background: var(--brand-strong);
}

.diagnostic-overview__status.is-failed {
  color: var(--danger);
  background: var(--danger-soft);
  border-color: color-mix(in srgb, var(--danger) 28%, var(--border));
}

.diagnostic-overview__status.is-failed i {
  background: var(--danger);
}

.diagnostic-overview__status.is-busy {
  color: var(--amber);
  background: var(--amber-soft);
  border-color: color-mix(in srgb, var(--amber) 28%, var(--border));
}

.diagnostic-overview__status.is-busy i {
  background: var(--amber);
}

.diagnostic-score-hero {
  display: grid;
  grid-template-columns: minmax(145px, .4fr) minmax(0, 1fr);
  gap: 20px;
  align-items: center;
  padding: 18px;
  background: color-mix(in srgb, var(--brand-soft) 38%, var(--surface));
  border: 1px solid color-mix(in srgb, var(--brand) 18%, var(--border));
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.diagnostic-score-ring {
  position: relative;
  display: grid;
  width: 148px;
  height: 148px;
  place-items: center;
  justify-self: center;
}

.diagnostic-score-ring svg {
  width: 148px;
  height: 148px;
  overflow: visible;
  transform: rotate(-90deg);
}

.diagnostic-score-ring circle {
  fill: none;
  stroke-width: 7;
}

.diagnostic-score-ring__track {
  stroke: color-mix(in srgb, var(--brand) 13%, var(--border));
}

.diagnostic-score-ring__progress {
  stroke: var(--brand);
  stroke-linecap: round;
  stroke-dasharray: 351.86;
  transition: stroke-dashoffset 300ms ease, stroke 160ms ease;
}

.diagnostic-score-ring.is-failed .diagnostic-score-ring__progress {
  stroke: var(--danger);
}

.diagnostic-score-ring.is-busy .diagnostic-score-ring__progress {
  stroke: var(--amber);
}

.diagnostic-score-ring.is-unavailable .diagnostic-score-ring__progress {
  stroke: var(--border-strong);
}

.diagnostic-score-ring__center {
  position: absolute;
  display: grid;
  place-items: center;
  gap: 4px;
}

.diagnostic-score-ring__center strong {
  color: var(--text);
  font-size: 24px;
  font-weight: 750;
  line-height: 1.1;
}

.diagnostic-score-ring__center span {
  color: var(--muted);
  font-size: 12px;
}

.diagnostic-score-hero__copy {
  min-width: 0;
}

.diagnostic-score-hero__label {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--brand-strong);
  font-size: 13px;
  font-weight: 750;
}

.diagnostic-score-hero__label span {
  padding: 3px 7px;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 650;
  letter-spacing: .04em;
}

.diagnostic-score-hero__label em {
  padding: 3px 7px;
  color: var(--brand-strong);
  background: var(--brand-soft);
  border: 1px solid var(--brand-muted);
  border-radius: 999px;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
  letter-spacing: 0;
}

.diagnostic-score-hero__copy h3 {
  margin: 9px 0 0;
  color: var(--text);
  font-size: 21px;
  line-height: 1.3;
}

.diagnostic-score-hero__copy p {
  max-width: 560px;
  margin: 7px 0 0;
  color: var(--text-soft);
  font-size: 14px;
  line-height: 1.6;
}

.diagnostic-score-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 18px;
  margin-top: 15px;
  color: var(--muted);
  font-size: 13px;
}

.diagnostic-score-meta span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.diagnostic-score-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 9px;
  margin-top: 18px;
}

.diagnostic-score-actions .button {
  min-height: 40px;
}

.diagnostic-score-route {
  display: grid;
  grid-column: 1 / -1;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  align-content: start;
  gap: 1px;
  padding: 11px 12px;
  background: color-mix(in srgb, var(--surface) 74%, transparent);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.diagnostic-score-route__header {
  display: flex;
  grid-column: 1 / -1;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-bottom: 6px;
  margin-bottom: 2px;
  border-bottom: 1px solid var(--border);
}

.diagnostic-score-route__header strong {
  font-size: 14px;
}

.diagnostic-score-route__header span {
  color: var(--brand-strong);
  font-size: 12px;
  font-weight: 700;
}

.diagnostic-score-route__item {
  display: grid;
  grid-template-columns: 25px minmax(0, 1fr) 8px;
  gap: 7px;
  align-items: center;
  min-height: 26px;
  color: var(--text-soft);
  font-size: 12px;
}

.diagnostic-score-route__index {
  color: var(--muted);
  font: 650 12px/1 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.diagnostic-score-route__item i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--border-strong);
}

.diagnostic-score-route__item i.is-running {
  background: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.diagnostic-score-route__item i.is-completed {
  background: var(--brand-strong);
}

.diagnostic-score-route__item i.is-failed {
  background: var(--danger);
}

.diagnostic-score-route__item i.is-covered {
  background: var(--brand-muted);
}

.diagnostic-score-dimensions {
  display: grid;
  gap: 12px;
}

.diagnostic-score-dimensions > header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}

.diagnostic-score-dimensions h3 {
  margin: 0;
  color: var(--text);
  font-size: 17px;
}

.diagnostic-score-dimensions p {
  margin: 5px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

.diagnostic-score-dimensions > header > span {
  flex: 0 0 auto;
  color: var(--muted);
  font-size: 13px;
}

.diagnostic-score-dimension-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.diagnostic-score-dimension {
  --dimension-color: var(--brand-strong);

  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  gap: 11px;
  align-items: center;
  min-width: 0;
  min-height: 86px;
  padding: 13px;
  color: var(--text);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  text-align: left;
  cursor: pointer;
  transition: border-color 160ms ease, background 160ms ease, transform 160ms ease;
}

.diagnostic-score-dimension:hover:not(:disabled),
.diagnostic-score-dimension:focus-visible {
  border-color: color-mix(in srgb, var(--dimension-color) 62%, var(--border));
  background: color-mix(in srgb, var(--dimension-color) 7%, var(--surface));
  outline: none;
  transform: translateY(-1px);
}

.diagnostic-score-dimension:disabled {
  cursor: not-allowed;
  opacity: .68;
}

.diagnostic-score-dimension.is-covered:disabled {
  border-style: dashed;
  opacity: .84;
}

.diagnostic-score-dimension.is-tone-performance { --dimension-color: #965900; }
.diagnostic-score-dimension.is-tone-route { --dimension-color: #2563c4; }
.diagnostic-score-dimension.is-tone-latency { --dimension-color: #087a72; }
.diagnostic-score-dimension.is-tone-speed { --dimension-color: #7546c8; }
.diagnostic-score-dimension.is-tone-ip { --dimension-color: #0c9b78; }

.diagnostic-score-dimension__icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  color: var(--dimension-color);
  background: color-mix(in srgb, var(--dimension-color) 10%, var(--surface));
  border: 1px solid color-mix(in srgb, var(--dimension-color) 22%, var(--border));
  border-radius: 12px;
}

.diagnostic-score-dimension__copy {
  display: grid;
  min-width: 0;
  gap: 4px;
}

.diagnostic-score-dimension__copy strong {
  font-size: 15px;
}

.diagnostic-score-dimension__copy > span {
  overflow: hidden;
  color: var(--muted);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.diagnostic-score-dimension__copy > span.is-summary {
  display: -webkit-box;
  overflow: hidden;
  color: var(--text-soft);
  font-weight: 650;
  white-space: normal;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  line-height: 1.35;
}

.diagnostic-score-dimension__state {
  display: inline-flex;
  grid-column: 2;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 12px;
}

.diagnostic-score-dimension__state i {
  width: 7px;
  height: 7px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--border-strong);
}

.diagnostic-score-dimension.is-running .diagnostic-score-dimension__state,
.diagnostic-score-dimension.is-running .diagnostic-score-dimension__state i {
  color: var(--brand-strong);
}

.diagnostic-score-dimension.is-running .diagnostic-score-dimension__state i {
  background: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.diagnostic-score-dimension.is-completed .diagnostic-score-dimension__state,
.diagnostic-score-dimension.is-completed .diagnostic-score-dimension__state i {
  color: var(--brand-strong);
}

.diagnostic-score-dimension.is-completed .diagnostic-score-dimension__state i {
  background: var(--brand-strong);
}

.diagnostic-score-dimension.is-failed .diagnostic-score-dimension__state,
.diagnostic-score-dimension.is-failed .diagnostic-score-dimension__state i {
  color: var(--danger);
}

.diagnostic-score-dimension.is-failed .diagnostic-score-dimension__state i {
  background: var(--danger);
}

.diagnostic-score-dimension.is-covered .diagnostic-score-dimension__state,
.diagnostic-score-dimension.is-covered .diagnostic-score-dimension__state i {
  color: var(--brand-strong);
}

.diagnostic-score-dimension.is-covered .diagnostic-score-dimension__state i {
  background: var(--brand-muted);
}

.diagnostic-score-note {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 13px 14px;
  background: color-mix(in srgb, var(--surface) 65%, var(--surface-subtle));
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.diagnostic-score-note__icon {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--brand-strong);
  background: var(--brand-soft);
  border-radius: 9px;
}

.diagnostic-score-note strong {
  display: block;
  color: var(--text-soft);
  font-size: 14px;
}

.diagnostic-score-note p {
  max-width: 680px;
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.55;
}

.diagnostic-score-note__link {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  margin-left: auto;
  padding: 5px 0;
  color: var(--brand-strong);
  background: transparent;
  border: 0;
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.diagnostic-score-note__link:hover,
.diagnostic-score-note__link:focus-visible {
  color: var(--brand);
  outline: none;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.diagnostic-result h2 {
  margin: 2px 0 0;
  font-size: 17px;
}

.diagnostic-result footer span,
.diagnostic-result footer a,
.diagnostic-source {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.impact-pill {
  padding: 4px 7px;
  border-radius: 999px;
  background: var(--surface-muted);
}

.impact-pill.is-network {
  color: var(--warning);
}

.impact-pill.is-intensive {
  color: var(--danger);
}

.diagnostic-result {
  overflow: hidden;
}

.diagnostic-result {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--surface);
  container: diagnostic-result / inline-size;
}

.diagnostic-progress {
  flex: 0 0 auto;
  height: 4px;
  background: var(--surface-muted);
}

.diagnostic-progress span {
  display: block;
  height: 100%;
  min-width: 3%;
  border-radius: 999px;
  background: var(--primary);
  transition: width 220ms ease;
}

.diagnostic-terminal-bar {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  min-height: 42px;
  padding: 8px 16px;
  border-bottom: 1px solid color-mix(in srgb, var(--terminal-shell-border, #29383a) 78%, var(--terminal-shell-text, #d8dddc));
  background: var(--terminal-shell-panel, #111a1d);
  color: color-mix(in srgb, var(--terminal-shell-text, #d8dddc) 78%, var(--terminal-shell-muted, #8a9695));
  font-size: 12px;
}

.diagnostic-terminal-bar > span:first-child {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.diagnostic-terminal-bar i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #667085;
}

.diagnostic-terminal-bar i.is-live {
  background: #36d399;
  box-shadow: 0 0 0 4px rgb(54 211 153 / 12%);
}

.diagnostic-log {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
  overflow: auto;
  overscroll-behavior: contain;
  margin: 0;
  padding: 18px 20px;
  background: var(--terminal-shell-background, #0b1214);
  color: var(--terminal-shell-text, #d8dddc);
  font: 12.5px/1.65 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.diagnostic-interactive-terminal {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
  border: 0;
  border-radius: 0;
}

.diagnostic-interactive-terminal :deep(.interactive-terminal__screen) {
  height: auto;
  min-height: 0;
}

.diagnostic-result footer {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  gap: 10px 18px;
  box-sizing: border-box;
  min-height: 42px;
  padding: 12px 20px;
  color: var(--text-tertiary);
  font-size: 13px;
}

.diagnostic-result footer a {
  color: var(--primary);
}

.diagnostic-confirm {
  display: flex;
  gap: 14px;
  color: var(--warning);
}

.diagnostic-confirm > svg {
  flex: 0 0 auto;
}

.diagnostic-confirm p {
  margin: 0 0 10px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.diagnostic-confirm a {
  color: var(--primary);
  font-size: 13px;
  overflow-wrap: anywhere;
}

.diagnostic-score-confirm > svg {
  color: var(--brand-strong);
}

.diagnostic-score-confirm__list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px 12px;
  color: var(--text-soft);
  font-size: 13px;
}

.diagnostic-score-confirm__list span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.diagnostic-score-confirm__list i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--brand);
}

.is-spinning {
  animation: diagnostic-spin 900ms linear infinite;
}

@keyframes diagnostic-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1120px) {
  .diagnostic-score-hero {
    grid-template-columns: minmax(145px, .4fr) minmax(0, 1fr);
  }

  .diagnostic-score-route {
    gap: 8px;
  }

  .diagnostic-score-route__header {
    grid-column: 1 / -1;
  }

  .diagnostic-score-route__item {
    grid-template-columns: 25px minmax(0, 1fr) 8px;
  }
}

@media (max-width: 900px) {
  .diagnostic-score-dimension-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@container diagnostic-result (max-width: 760px) {
  .diagnostic-score-route {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .diagnostic-score-dimension-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@container diagnostic-result (max-width: 560px) {
  .diagnostic-score-route {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .diagnostic-score-dimension-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@container diagnostic-result (min-width: 860px) {
  .diagnostic-score-dimension-grid {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }

  .diagnostic-score-dimension {
    grid-template-columns: 34px minmax(0, 1fr);
    gap: 8px;
    min-height: 96px;
    padding: 11px;
  }

  .diagnostic-score-dimension__icon {
    width: 34px;
    height: 34px;
    border-radius: 10px;
  }

  .diagnostic-score-dimension__copy strong {
    font-size: 14px;
  }

  .diagnostic-score-dimension__copy > span {
    display: -webkit-box;
    overflow: hidden;
    white-space: normal;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    line-height: 1.35;
  }

  .diagnostic-score-dimension__copy > span.is-summary {
    -webkit-line-clamp: 3;
  }
}

/* A desktop window can be narrower than the browser viewport. Keep the
   diagnostic drawer responsive to that bounded surface, not only to the
   outer viewport media query. */
@container desktop-window (max-width: 820px) {
  .diagnostic-workbench,
  .diagnostic-workbench.is-command-panel-collapsed {
    position: relative;
    grid-template-columns: minmax(0, 1fr) !important;
    grid-template-rows: minmax(0, 1fr);
    height: 100% !important;
    min-height: 0 !important;
  }

  .diagnostic-command-panel {
    position: absolute;
    z-index: 22;
    inset: 0 auto 0 0;
    width: min(320px, calc(100% - 48px));
    border-right: 1px solid var(--border);
    border-bottom: 0;
    box-shadow: var(--shadow-md);
    transform: translateX(-105%);
    transition: transform .2s ease;
  }

  .diagnostic-workbench.is-command-drawer-open .diagnostic-command-panel {
    transform: translateX(0);
  }

  .diagnostic-command-overlay {
    position: absolute;
    z-index: 21;
    inset: 0;
    display: block;
    border: 0;
    background: rgb(5 16 13 / 42%);
  }

  .diagnostic-command-panel__desktop-toggle {
    display: none;
  }

  .diagnostic-command-panel__mobile-close {
    display: grid;
  }

  .diagnostic-command-list {
    padding-top: 8px;
  }

  .diagnostic-command-overview {
    margin-top: 8px;
  }

  .diagnostic-mobile-selector {
    display: flex;
    min-width: 0;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-height: 50px;
    padding: 8px 12px;
    border-bottom: 1px solid color-mix(in srgb, var(--terminal-shell-border, #29383a) 78%, var(--terminal-shell-text, #d8dddc));
    background: var(--terminal-shell-panel, #111a1d);
  }

  .diagnostic-mobile-selector button {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 8px;
    border: 0;
    padding: 7px 8px;
    color: var(--terminal-shell-text, #d8dddc);
    background: transparent;
    font-weight: 700;
    cursor: pointer;
  }

  .diagnostic-mobile-selector button span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .diagnostic-mobile-selector small {
    flex: 0 0 auto;
    color: var(--terminal-shell-muted, #8a9695);
  }

  .diagnostic-overview {
    gap: 15px;
    padding: 15px;
  }

  .diagnostic-overview__header {
    flex-direction: column;
    gap: 10px;
  }

  .diagnostic-overview__status {
    align-self: flex-start;
  }

  .diagnostic-score-hero {
    grid-template-columns: minmax(0, 1fr);
    gap: 18px;
    padding: 16px;
  }

  .diagnostic-score-ring {
    width: 144px;
    height: 144px;
  }

  .diagnostic-score-ring svg {
    width: 144px;
    height: 144px;
  }

  .diagnostic-score-route {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }

  .diagnostic-score-route__header {
    grid-column: 1 / -1;
  }

  .diagnostic-score-dimension-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .diagnostic-score-note {
    flex-wrap: wrap;
  }

  .diagnostic-score-note__link {
    width: 100%;
    margin-left: 41px;
  }

  .diagnostic-log,
  .diagnostic-interactive-terminal :deep(.interactive-terminal__screen) {
    min-height: min(400px, 48dvh);
  }

  .diagnostic-terminal-bar,
  .diagnostic-result footer {
    padding-right: 14px;
    padding-left: 14px;
  }
}

@media (max-width: 680px) {
  .diagnostic-workbench {
    position: relative;
    grid-template-columns: minmax(0, 1fr);
    height: auto;
    min-height: min(580px, calc(100dvh - 110px));
    border-radius: 14px;
  }

  .diagnostic-workbench.is-command-panel-collapsed {
    grid-template-columns: minmax(0, 1fr);
  }

  .diagnostic-command-panel {
    position: absolute;
    z-index: 22;
    inset: 0 auto 0 0;
    width: min(320px, calc(100% - 48px));
    border-right: 1px solid var(--border);
    border-bottom: 0;
    box-shadow: var(--shadow-md);
    transform: translateX(-105%);
    transition: transform .2s ease;
  }

  .diagnostic-workbench.is-command-drawer-open .diagnostic-command-panel {
    transform: translateX(0);
  }

  .diagnostic-command-overlay {
    position: absolute;
    z-index: 21;
    inset: 0;
    display: block;
    border: 0;
    background: rgb(5 16 13 / 42%);
  }

  .diagnostic-command-panel__desktop-toggle {
    display: none;
  }

  .diagnostic-command-panel__mobile-close {
    display: grid;
  }

  .diagnostic-command-list {
    max-height: none;
    padding-top: 8px;
  }

  .diagnostic-command-overview {
    margin-top: 8px;
  }

  .diagnostic-overview {
    gap: 15px;
    padding: 15px;
  }

  .diagnostic-overview__header {
    flex-direction: column;
    gap: 10px;
  }

  .diagnostic-overview__status {
    align-self: flex-start;
  }

  .diagnostic-score-hero {
    grid-template-columns: minmax(0, 1fr);
    gap: 18px;
    padding: 16px;
  }

  .diagnostic-score-ring {
    width: 144px;
    height: 144px;
  }

  .diagnostic-score-ring svg {
    width: 144px;
    height: 144px;
  }

  .diagnostic-score-route {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }

  .diagnostic-score-route__header {
    grid-column: 1 / -1;
  }

  .diagnostic-score-dimension-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .diagnostic-score-note {
    flex-wrap: wrap;
  }

  .diagnostic-score-note__link {
    width: 100%;
    margin-left: 41px;
  }

  .diagnostic-mobile-selector {
    display: flex;
    min-width: 0;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-height: 50px;
    padding: 8px 12px;
    border-bottom: 1px solid color-mix(in srgb, var(--terminal-shell-border, #29383a) 78%, var(--terminal-shell-text, #d8dddc));
    background: var(--terminal-shell-panel, #111a1d);
  }

  .diagnostic-mobile-selector button {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 8px;
    border: 0;
    padding: 7px 8px;
    color: var(--terminal-shell-text, #d8dddc);
    background: transparent;
    font-weight: 700;
    cursor: pointer;
  }

  .diagnostic-mobile-selector button span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .diagnostic-mobile-selector small {
    flex: 0 0 auto;
    color: var(--terminal-shell-muted, #8a9695);
  }

  .diagnostic-log,
  .diagnostic-interactive-terminal :deep(.interactive-terminal__screen) {
    min-height: min(400px, 48dvh);
  }

  .diagnostic-terminal-bar,
  .diagnostic-result footer {
    padding-right: 14px;
    padding-left: 14px;
  }

}

@media (prefers-reduced-motion: reduce) {
  .diagnostic-command-panel {
    transition: none;
  }
}
</style>
