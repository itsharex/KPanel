<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Activity,
  Cpu,
  ExternalLink,
  Gauge,
  Globe2,
  History,
  LoaderCircle,
  Network,
  Play,
  RefreshCw,
  ShieldCheck,
  Timer,
  TriangleAlert,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { formatDateTime } from '@/lib/format'
import { useToast } from '@/stores/toast'
import type { DiagnosticCatalog, DiagnosticCheck, DiagnosticJob } from '@/types/api'

const catalog = ref<DiagnosticCatalog>()
const jobs = ref<DiagnosticJob[]>([])
const selectedCategory = ref('all')
const pendingCheck = ref<DiagnosticCheck>()
const activeJob = ref<DiagnosticJob>()
const loading = ref(true)
const refreshing = ref(false)
const starting = ref(false)
const error = ref('')
const toast = useToast()
let controller: AbortController | undefined
let pollController: AbortController | undefined
let pollTimer: number | undefined

const categories = computed(() => catalog.value?.categories || [])
const visibleChecks = computed(() =>
  (catalog.value?.items || []).filter(
    (item) => selectedCategory.value === 'all' || item.category === selectedCategory.value,
  ),
)
const recentJobs = computed(() => jobs.value.slice(0, 10))
const hasActiveJob = computed(
  () => activeJob.value?.status === 'queued' || activeJob.value?.status === 'running',
)
const activeLog = computed(() => activeJob.value?.logs.join('\n') || '等待脚本输出…')

function categoryName(id: string): string {
  return categories.value.find((item) => item.id === id)?.name || id
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

function stopPolling(): void {
  if (pollTimer) window.clearInterval(pollTimer)
  pollTimer = undefined
  pollController?.abort()
  pollController = undefined
}

async function refreshJob(id: string): Promise<void> {
  pollController?.abort()
  pollController = new AbortController()
  try {
    const next = await api.diagnostics.job(id, pollController.signal)
    const previous = activeJob.value?.status
    activeJob.value = next
    const index = jobs.value.findIndex((item) => item.id === next.id)
    if (index >= 0) jobs.value.splice(index, 1, next)
    else jobs.value.unshift(next)
    if (next.status === 'succeeded' || next.status === 'failed') {
      stopPolling()
      if (previous === 'queued' || previous === 'running') {
        if (next.status === 'succeeded') toast.success(`${next.checkName}已完成`)
        else toast.danger(`${next.checkName}执行失败`, next.message)
      }
    }
  } catch (reason) {
    if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
      stopPolling()
    }
  }
}

function startPolling(job: DiagnosticJob): void {
  stopPolling()
  activeJob.value = job
  void refreshJob(job.id)
  pollTimer = window.setInterval(() => void refreshJob(job.id), 2_000)
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''
  try {
    const [nextCatalog, history] = await Promise.all([
      api.diagnostics.catalog(controller.signal),
      api.diagnostics.jobs(controller.signal),
    ])
    catalog.value = nextCatalog
    jobs.value = history.items
    const active = history.items.find((item) => item.status === 'queued' || item.status === 'running')
    if (active) startPolling(active)
  } catch (reason) {
    if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
      error.value = reason instanceof ApiError ? reason.message : '无法读取体检项目，请检查 Agent 与 kejilion.sh 版本。'
    }
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function confirmStart(): Promise<void> {
  const check = pendingCheck.value
  if (!check || starting.value || hasActiveJob.value) return
  starting.value = true
  try {
    const job = await api.diagnostics.start(check.id)
    jobs.value.unshift(job)
    pendingCheck.value = undefined
    startPolling(job)
    toast.success(`${check.name}已开始`, '页面会持续刷新第三方脚本输出。')
  } catch (reason) {
    toast.danger(
      '体检任务启动失败',
      reason instanceof ApiError ? reason.message : '请检查 Agent、systemd 与 kejilion.sh 体检协议。',
    )
  } finally {
    starting.value = false
  }
}

function openJob(job: DiagnosticJob): void {
  activeJob.value = job
  if (job.status === 'queued' || job.status === 'running') startPolling(job)
}

onMounted(() => void load())
onBeforeUnmount(() => {
  controller?.abort()
  stopPolling()
})
</script>

<template>
  <div class="diagnostics-page">
    <PageHeader
      title="体检"
      description="直接调用 kejilion.sh 的第三方测试合集，实时查看线路、IP 质量与性能跑分结果。"
    >
      <template #actions>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="17" :class="{ 'is-spinning': refreshing }" />
          刷新
        </button>
      </template>
    </PageHeader>

    <section class="diagnostic-notice">
      <ShieldCheck :size="21" />
      <div>
        <strong>命令与下载源由本机 kejilion.sh 提供</strong>
        <p>面板只提交固定体检编号，不接受自定义 Shell。第三方脚本可能安装依赖，并消耗 CPU、磁盘或网络流量。</p>
      </div>
    </section>

    <LoadingState v-if="loading" title="正在读取体检项目" description="正在校验本机脚本协议与第三方来源。" />
    <ErrorState v-else-if="error" title="体检功能暂不可用" :message="error" @retry="load()" />

    <template v-else-if="catalog">
      <nav class="diagnostic-tabs" aria-label="体检分类">
        <button
          type="button"
          :class="{ 'is-active': selectedCategory === 'all' }"
          @click="selectedCategory = 'all'"
        >
          全部 <span>{{ catalog.items.length }}</span>
        </button>
        <button
          v-for="item in categories"
          :key="item.id"
          type="button"
          :class="{ 'is-active': selectedCategory === item.id }"
          @click="selectedCategory = item.id"
        >
          {{ item.name }}
          <span>{{ catalog.items.filter((check) => check.category === item.id).length }}</span>
        </button>
      </nav>

      <section v-if="visibleChecks.length" class="diagnostic-grid">
        <article v-for="check in visibleChecks" :key="check.id" class="diagnostic-card">
          <header>
            <span class="diagnostic-card__icon">
              <component :is="categoryIcon(check.category)" :size="22" />
            </span>
            <div>
              <small>{{ categoryName(check.category) }}</small>
              <h2>{{ check.name }}</h2>
            </div>
          </header>
          <p>{{ check.description }}</p>
          <div class="diagnostic-card__meta">
            <span><Timer :size="14" /> 约 {{ check.estimatedMinutes }} 分钟</span>
            <span class="impact-pill" :class="impactClass(check.impact)">
              {{ impactLabel(check.impact) }}
            </span>
          </div>
          <a :href="check.sourceUrl" target="_blank" rel="noopener noreferrer" class="diagnostic-source">
            {{ sourceHost(check.sourceUrl) }} <ExternalLink :size="13" />
          </a>
          <button
            class="button button--primary diagnostic-card__action"
            type="button"
            :disabled="hasActiveJob || starting"
            @click="pendingCheck = check"
          >
            <Play :size="16" /> {{ hasActiveJob ? '已有任务运行中' : '开始体检' }}
          </button>
        </article>
      </section>
      <EmptyState v-else title="当前分类没有体检项目" description="切换其他分类后重试。" />

      <section v-if="activeJob" class="diagnostic-result">
        <header class="diagnostic-result__header">
          <div>
            <span class="eyebrow">当前结果</span>
            <h2>{{ activeJob.checkName }}</h2>
            <p>{{ activeJob.message }}</p>
          </div>
          <StatusBadge :status="activeJob.status" />
        </header>
        <div v-if="hasActiveJob" class="diagnostic-progress" aria-label="任务进度">
          <span :style="{ width: `${activeJob.progress}%` }" />
        </div>
        <pre class="diagnostic-log" aria-live="polite">{{ activeLog }}</pre>
        <footer>
          <span><Activity :size="14" /> {{ categoryName(activeJob.category) }}</span>
          <span><Timer :size="14" /> 开始于 {{ formatDateTime(activeJob.startedAt || activeJob.createdAt) }}</span>
          <a :href="activeJob.sourceUrl" target="_blank" rel="noopener noreferrer">
            查看来源 <ExternalLink :size="13" />
          </a>
        </footer>
      </section>

      <section class="diagnostic-history">
        <header>
          <div>
            <span class="eyebrow">历史记录</span>
            <h2><History :size="19" /> 最近体检</h2>
          </div>
        </header>
        <div v-if="recentJobs.length" class="diagnostic-history__list">
          <button v-for="job in recentJobs" :key="job.id" type="button" @click="openJob(job)">
            <span>
              <strong>{{ job.checkName }}</strong>
              <small>{{ formatDateTime(job.createdAt) }} · {{ categoryName(job.category) }}</small>
            </span>
            <StatusBadge :status="job.status" subtle />
          </button>
        </div>
        <EmptyState v-else title="还没有体检记录" description="选择上方项目开始第一次服务器体检。" />
      </section>
    </template>

    <ModalDialog
      :open="Boolean(pendingCheck)"
      title="确认运行第三方体检？"
      :description="pendingCheck ? `${pendingCheck.name} · 预计 ${pendingCheck.estimatedMinutes} 分钟` : ''"
      size="small"
      @close="pendingCheck = undefined"
    >
      <div v-if="pendingCheck" class="diagnostic-confirm">
        <TriangleAlert :size="24" />
        <div>
          <p>
            此操作将以 root 权限运行 kejilion.sh 中登记的第三方命令，可能安装检测依赖并产生较高资源占用。
          </p>
          <a :href="pendingCheck.sourceUrl" target="_blank" rel="noopener noreferrer">
            {{ pendingCheck.sourceUrl }} <ExternalLink :size="13" />
          </a>
        </div>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" :disabled="starting" @click="pendingCheck = undefined">
          取消
        </button>
        <button class="button button--primary" type="button" :disabled="starting" @click="confirmStart">
          <LoaderCircle v-if="starting" :size="16" class="is-spinning" />
          <Play v-else :size="16" />
          {{ starting ? '正在启动' : '确认开始' }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.diagnostics-page {
  display: grid;
  gap: 22px;
}

.diagnostic-notice,
.diagnostic-result,
.diagnostic-history {
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.diagnostic-notice {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  padding: 17px 19px;
  color: var(--text-secondary);
}

.diagnostic-notice > svg {
  flex: 0 0 auto;
  color: var(--success);
}

.diagnostic-notice strong {
  display: block;
  color: var(--text);
  margin-bottom: 4px;
}

.diagnostic-notice p,
.diagnostic-card p,
.diagnostic-result p {
  margin: 0;
}

.diagnostic-tabs {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.diagnostic-tabs button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  flex: 0 0 auto;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-secondary);
  padding: 9px 13px;
  cursor: pointer;
}

.diagnostic-tabs button span {
  color: var(--text-tertiary);
  font-size: 12px;
}

.diagnostic-tabs button.is-active {
  border-color: color-mix(in srgb, var(--primary) 42%, var(--border));
  background: color-mix(in srgb, var(--primary) 10%, var(--surface));
  color: var(--primary);
}

.diagnostic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 15px;
}

.diagnostic-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 255px;
  padding: 19px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-sm);
}

.diagnostic-card header {
  display: flex;
  gap: 12px;
  align-items: center;
}

.diagnostic-card__icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  border-radius: 13px;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 11%, var(--surface));
}

.diagnostic-card small,
.eyebrow {
  color: var(--text-tertiary);
  font-size: 12px;
}

.diagnostic-card h2,
.diagnostic-result h2,
.diagnostic-history h2 {
  margin: 2px 0 0;
  font-size: 17px;
}

.diagnostic-card > p {
  min-height: 43px;
  color: var(--text-secondary);
  line-height: 1.55;
}

.diagnostic-card__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--text-tertiary);
}

.diagnostic-card__meta > span:first-child,
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

.diagnostic-source {
  width: fit-content;
  max-width: 100%;
  overflow: hidden;
  color: var(--primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.diagnostic-card__action {
  width: 100%;
  margin-top: auto;
}

.diagnostic-result,
.diagnostic-history {
  overflow: hidden;
}

.diagnostic-result__header,
.diagnostic-history > header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border);
}

.diagnostic-result__header p {
  margin-top: 5px;
  color: var(--text-secondary);
}

.diagnostic-progress {
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

.diagnostic-log {
  min-height: 250px;
  max-height: 560px;
  overflow: auto;
  margin: 0;
  padding: 18px 20px;
  background: #10151f;
  color: #dce5f3;
  font: 12.5px/1.65 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.diagnostic-result footer {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 18px;
  padding: 12px 20px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.diagnostic-result footer a {
  color: var(--primary);
}

.diagnostic-history h2 {
  display: flex;
  align-items: center;
  gap: 7px;
}

.diagnostic-history__list {
  display: grid;
}

.diagnostic-history__list button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
  padding: 14px 20px;
  border: 0;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  text-align: left;
  cursor: pointer;
}

.diagnostic-history__list button:last-child {
  border-bottom: 0;
}

.diagnostic-history__list button:hover {
  background: var(--surface-muted);
}

.diagnostic-history__list strong,
.diagnostic-history__list small {
  display: block;
}

.diagnostic-history__list small {
  margin-top: 4px;
  color: var(--text-tertiary);
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
  font-size: 12px;
  overflow-wrap: anywhere;
}

.is-spinning {
  animation: diagnostic-spin 900ms linear infinite;
}

@keyframes diagnostic-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 680px) {
  .diagnostic-grid {
    grid-template-columns: 1fr;
  }

  .diagnostic-card {
    min-height: auto;
  }

  .diagnostic-result__header {
    align-items: center;
  }
}
</style>
