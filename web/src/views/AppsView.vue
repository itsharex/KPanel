<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ArrowUpRight,
  Activity,
  CheckCircle2,
  ChevronRight,
  Download,
  Globe2,
  LoaderCircle,
  LockKeyhole,
  Network,
  PackageCheck,
  Play,
  RefreshCw,
  RotateCw,
  Search,
  ShieldCheck,
  Square,
  Store,
  Trash2,
  UnlockKeyhole,
  Wrench,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import DnsResolutionGuide from '@/components/common/DnsResolutionGuide.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import AppInteractiveTerminal from '@/components/apps/AppInteractiveTerminal.vue'
import { ApiError, api } from '@/lib/api'
import { appAccessURL, matchingAppProxySites } from '@/lib/appAccess'
import { useToast } from '@/stores/toast'
import type { AppInstallJob, AppMarketInventory, AppMarketItem, PublicNetworkSummary, Site } from '@/types/api'

type SourceFilter = 'all' | 'builtin' | 'thirdparty'
type StatusFilter = 'all' | 'installed' | 'running' | 'adapted'
type ConfirmAction = 'update' | 'uninstall' | undefined

const inventory = ref<AppMarketInventory>()
const sites = ref<Site[]>([])
const publicNetwork = ref<PublicNetworkSummary>()
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const search = ref('')
const category = ref('all')
const source = ref<SourceFilter>('all')
const status = ref<StatusFilter>('installed')
const selectedID = ref('')
const installOpen = ref(false)
const installPort = ref(0)
const installAccess = ref<'direct' | 'domain_only'>('direct')
const domain = ref('')
const domainError = ref('')
const domainWarning = ref('')
const sitesWarning = ref('')
const operation = ref('')
const confirmAction = ref<ConfirmAction>()
const checkedUpdates = ref<Record<string, 'available' | 'current'>>({})
const activeJob = ref<AppInstallJob>()
const jobDetailsOpen = ref(false)
const toast = useToast()
let controller: AbortController | undefined
let jobController: AbortController | undefined
let jobTimer: number | undefined
const activeJobStorageKey = 'kpanel:active-app-job'

const selected = computed(() => inventory.value?.items.find((item) => item.id === selectedID.value))
const selectedPort = computed(() => selected.value?.runtime.ports.find((port) => port.type === 'tcp' && port.publicPort))
const selectedDomains = computed(() =>
  selected.value ? matchingAppProxySites(selected.value, sites.value) : [],
)
const applicationTaskActive = computed(
  () => Boolean(activeJob.value) && isActiveJob(activeJob.value),
)

const filteredApps = computed(() => {
  const needle = search.value.trim().toLowerCase()
  return (inventory.value?.items || []).filter((item) => {
    if (category.value !== 'all' && item.cat !== category.value) return false
    if (source.value !== 'all' && item.source !== source.value) return false
    if (status.value === 'installed' && !item.runtime.installed) return false
    if (status.value === 'running' && item.runtime.state !== 'running') return false
    if (
      status.value === 'adapted' &&
      !item.capabilities.install?.enabled &&
      !item.capabilities.update?.enabled
    ) {
      return false
    }
    if (!needle) return true
    return [item.name_zh, item.name_en, item.desc_zh, item.token, item.runtime.containerName]
      .filter(Boolean)
      .some((value) => value!.toLowerCase().includes(needle))
  }).sort((left, right) => {
    if (left.runtime.installed !== right.runtime.installed) return left.runtime.installed ? -1 : 1
    return (left.num || 9999) - (right.num || 9999)
  })
})

const categoryCounts = computed(() => {
  const counts: Record<string, number> = { all: inventory.value?.items.length || 0 }
  for (const item of inventory.value?.items || []) counts[item.cat] = (counts[item.cat] || 0) + 1
  return counts
})

function capability(item: AppMarketItem, action: string): boolean {
  return item.capabilities[action]?.enabled === true
}

function categoryName(key: string): string {
  return inventory.value?.categories.find((item) => item.key === key)?.zh || key
}

function stateLabel(item: AppMarketItem): string {
  if (!item.runtime.installed) return '未安装'
  const labels: Record<string, string> = {
    running: '运行中',
    exited: '已停止',
    created: '待启动',
    restarting: '重启中',
    dead: '异常',
    unknown: '待核对',
  }
  return labels[item.runtime.state] || item.runtime.state
}

function updateLabel(item: AppMarketItem): string {
  if (checkedUpdates.value[item.id] === 'available') return '发现更新'
  if (checkedUpdates.value[item.id] === 'current') return '已是最新'
  const labels: Record<string, string> = {
    available: '发现更新',
    current: '已是最新',
    check_required: '可检查更新',
    unknown: '更新状态未知',
    not_installed: '未安装',
  }
  return labels[item.runtime.updateStatus] || '更新状态未知'
}

async function checkUpdate(): Promise<void> {
  const item = selected.value
  if (!item?.runtime.resourceVersion || !capability(item, 'check_update')) return
  operation.value = 'check_update'
  try {
    let result
    try {
      result = await api.apps.checkUpdate(item.id, item.runtime.resourceVersion)
    } catch (reason) {
      if (!(reason instanceof ApiError) || reason.code !== 'resource_conflict') throw reason
      const refreshedInventory = await api.apps.inventory()
      inventory.value = refreshedInventory
      const refreshed = refreshedInventory.items.find((candidate) => candidate.id === item.id)
      if (
        !refreshed?.runtime.resourceVersion ||
        !capability(refreshed, 'check_update')
      ) {
        throw reason
      }
      result = await api.apps.checkUpdate(refreshed.id, refreshed.runtime.resourceVersion)
    }
    checkedUpdates.value[item.id] = result.status
    toast.success(result.updateAvailable ? '发现可用更新' : '当前已是最新镜像')
  } catch (reason) {
    toast.danger('检查更新失败', reason instanceof ApiError ? reason.message : '镜像仓库暂时不可用。')
  } finally {
    operation.value = ''
  }
}

function openDetails(item: AppMarketItem): void {
  selectedID.value = item.id
  domain.value = ''
  domainError.value = ''
  domainWarning.value = ''
}

function openInstall(item: AppMarketItem): void {
  selectedID.value = item.id
  installPort.value = item.defaultPort || 0
  installAccess.value = 'direct'
  installOpen.value = true
}

function showAllApps(): void {
  status.value = 'all'
  category.value = 'all'
  source.value = 'all'
  search.value = ''
}

function isActiveJob(job?: AppInstallJob): boolean {
  return job?.status === 'queued' || job?.status === 'running'
}

function isBackgroundJob(result: unknown): result is AppInstallJob {
  return Boolean(
    result &&
      typeof result === 'object' &&
      'id' in result &&
      'appId' in result &&
      'progress' in result &&
      'stage' in result,
  )
}

function jobActionLabel(action?: AppInstallJob['action']): string {
  const labels: Record<AppInstallJob['action'], string> = {
    install: '安装',
    update: '更新',
    uninstall: '卸载',
    direct_access: '访问策略变更',
    manage: '脚本管理',
  }
  return action ? labels[action] : '操作'
}

function stopJobPolling(): void {
  if (jobTimer) window.clearInterval(jobTimer)
  jobTimer = undefined
  jobController?.abort()
  jobController = undefined
}

async function refreshJob(id: string): Promise<void> {
  jobController?.abort()
  jobController = new AbortController()
  try {
    const job = await api.apps.job(id, jobController.signal)
    const previousStatus = activeJob.value?.status
    activeJob.value = job
    if (isActiveJob(job)) return
    stopJobPolling()
    window.localStorage.removeItem(activeJobStorageKey)
    if (previousStatus === 'queued' || previousStatus === 'running') {
      if (job.status === 'succeeded') {
        toast.success(`后台${jobActionLabel(job.action)}完成`, `${job.appName} 已完成状态对账。`)
        if (job.action === 'uninstall' && selectedID.value === job.appId) selectedID.value = ''
        await load(true)
      } else {
        toast.danger(`后台${jobActionLabel(job.action)}失败`, job.message || '请查看任务日志后重试。')
      }
    }
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    if (reason instanceof ApiError && reason.status === 404) {
      stopJobPolling()
      activeJob.value = undefined
      window.localStorage.removeItem(activeJobStorageKey)
    }
  }
}

function beginJobPolling(id: string): void {
  stopJobPolling()
  window.localStorage.setItem(activeJobStorageKey, id)
  void refreshJob(id)
  jobTimer = window.setInterval(() => void refreshJob(id), 2_000)
}

function startJobPolling(job: AppInstallJob): void {
  activeJob.value = job
  beginJobPolling(job.id)
}

async function restoreBackgroundJob(): Promise<void> {
  const savedID = window.localStorage.getItem(activeJobStorageKey)
  if (savedID) {
    try {
      const job = await api.apps.job(savedID)
      activeJob.value = job
      if (isActiveJob(job)) startJobPolling(job)
      else window.localStorage.removeItem(activeJobStorageKey)
      return
    } catch (reason) {
      if (reason instanceof ApiError && reason.status === 404) {
        window.localStorage.removeItem(activeJobStorageKey)
      } else {
        beginJobPolling(savedID)
        return
      }
    }
  }
  try {
    const result = await api.apps.jobs()
    const running = result.items.find((job) => isActiveJob(job))
    if (running) startJobPolling(running)
  } catch {
    // The catalog remains usable when an older Agent has no job endpoint.
  }
}

function isAbortError(reason: unknown): boolean {
  return reason instanceof DOMException && reason.name === 'AbortError'
}

async function loadSites(signal: AbortSignal): Promise<Site[] | undefined> {
  try {
    return (await api.sites.list(undefined, signal)).items
  } catch (reason) {
    if (isAbortError(reason)) throw reason
    return undefined
  }
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  const requestController = new AbortController()
  controller = requestController
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''
  try {
    const sitesPromise = loadSites(requestController.signal)
    const publicNetworkPromise = api.system.publicNetwork(requestController.signal).catch(() => undefined)
    const nextInventory = await api.apps.inventory(requestController.signal)
    if (controller !== requestController) return
    inventory.value = nextInventory
    loading.value = false
    let nextSites = await sitesPromise
    if (controller !== requestController) return
    if (nextSites === undefined) nextSites = await loadSites(requestController.signal)
    if (controller !== requestController) return
    if (nextSites === undefined) {
      sitesWarning.value = sites.value.length
        ? '域名列表暂时无法刷新，当前显示上次成功读取的结果。'
        : '域名列表暂时无法读取，请刷新后重试；已有绑定不会因此被删除。'
    } else {
      sites.value = nextSites
      sitesWarning.value = ''
    }
    publicNetwork.value = await publicNetworkPromise
  } catch (reason) {
    if (isAbortError(reason)) return
    error.value = reason instanceof ApiError ? reason.message : '无法读取应用市场，请稍后重试。'
  } finally {
    if (controller === requestController) {
      loading.value = false
      refreshing.value = false
    }
  }
}

async function install(): Promise<void> {
  const item = selected.value
  if (!item || !capability(item, 'install')) return
  operation.value = 'install'
  try {
    const job = await api.apps.install(item.id, {
      hostPort: item.installer === 'kejilion' ? undefined : installPort.value || undefined,
      accessMode: item.installer === 'kejilion' ? undefined : installAccess.value,
    })
    installOpen.value = false
    selectedID.value = ''
    startJobPolling(job)
    jobDetailsOpen.value = true
    toast.success('已转入后台安装', `${item.name_zh} 安装期间可以继续使用面板。`)
  } catch (reason) {
    toast.danger('安装失败', reason instanceof ApiError ? reason.message : 'Agent 未能完成安装。')
  } finally {
    operation.value = ''
  }
}

async function lifecycle(action: 'start' | 'stop' | 'restart'): Promise<void> {
  const item = selected.value
  if (!item?.runtime.resourceVersion || !capability(item, action)) return
  operation.value = action
  try {
      await api.apps.action(item.id, action, { resourceVersion: item.runtime.resourceVersion })
      toast.success(action === 'start' ? '应用已启动' : action === 'stop' ? '应用已停止' : '应用已重启')
      if (action === 'stop' && status.value === 'running') status.value = 'installed'
      await load(true)
  } catch (reason) {
    toast.danger('操作失败', reason instanceof ApiError ? reason.message : '应用状态未能变更。')
  } finally {
    operation.value = ''
  }
}

async function confirmMutation(): Promise<void> {
  let item = selected.value
  const action = confirmAction.value
  if (!item?.runtime.resourceVersion || !action || !capability(item, action)) return
  operation.value = action
  try {
    let result
    try {
      result = await api.apps.action(item.id, action, { resourceVersion: item.runtime.resourceVersion })
    } catch (reason) {
      if (!(reason instanceof ApiError) || reason.code !== 'resource_conflict') throw reason
      const refreshedInventory = await api.apps.inventory()
      inventory.value = refreshedInventory
      const refreshed = refreshedInventory.items.find((candidate) => candidate.id === item?.id)
      if (!refreshed?.runtime.resourceVersion || !capability(refreshed, action)) throw reason
      item = refreshed
      result = await api.apps.action(refreshed.id, action, {
        resourceVersion: refreshed.runtime.resourceVersion,
      })
    }
    if (isBackgroundJob(result)) {
      confirmAction.value = undefined
      startJobPolling(result)
      jobDetailsOpen.value = true
      toast.success(`已转入后台${jobActionLabel(result.action)}`, `${item.name_zh} 处理期间可以继续使用面板。`)
      return
    }
    confirmAction.value = undefined
    toast.success(action === 'update' ? '应用更新完成' : '应用已卸载')
    if (action === 'uninstall') selectedID.value = ''
    await load(true)
  } catch (reason) {
    toast.danger(
      action === 'update' ? '更新失败' : '卸载失败',
      reason instanceof ApiError ? reason.message : 'Agent 拒绝了本次操作。',
    )
  } finally {
    operation.value = ''
  }
}

async function openScriptManage(): Promise<void> {
  const item = selected.value
  if (!item?.runtime.resourceVersion || !capability(item, 'manage')) return
  operation.value = 'manage'
  try {
    const result = await api.apps.action(item.id, 'manage', {
      resourceVersion: item.runtime.resourceVersion,
    })
    if (!isBackgroundJob(result)) {
      throw new Error('Agent 未返回交互管理任务')
    }
    startJobPolling(result)
    jobDetailsOpen.value = true
    toast.success('脚本管理终端已打开', `${item.name_zh} 正在使用固定应用编号进入 kejilion.sh 原生菜单。`)
  } catch (reason) {
    toast.danger(
      '脚本管理启动失败',
      reason instanceof ApiError ? reason.message : 'Agent 未能打开该应用的原生管理终端。',
    )
  } finally {
    operation.value = ''
  }
}

async function toggleAccess(): Promise<void> {
  let item = selected.value
  if (
    applicationTaskActive.value ||
    !item?.runtime.resourceVersion ||
    !capability(item, 'direct_access')
  ) return
  const next = item.runtime.accessMode === 'domain_only' ? 'direct' : 'domain_only'
  operation.value = 'direct_access'
  try {
    let result
    try {
      result = await api.apps.action(item.id, 'direct_access', {
        resourceVersion: item.runtime.resourceVersion,
        accessMode: next,
      })
    } catch (reason) {
      if (!(reason instanceof ApiError) || reason.code !== 'resource_conflict') throw reason
      const refreshedInventory = await api.apps.inventory()
      inventory.value = refreshedInventory
      const refreshed = refreshedInventory.items.find((candidate) => candidate.id === item?.id)
      if (
        !refreshed?.runtime.resourceVersion ||
        !capability(refreshed, 'direct_access')
      ) {
        throw reason
      }
      item = refreshed
      result = await api.apps.action(refreshed.id, 'direct_access', {
        resourceVersion: refreshed.runtime.resourceVersion,
        accessMode: next,
      })
    }
    if (isBackgroundJob(result)) {
      startJobPolling(result)
      jobDetailsOpen.value = true
      toast.success('访问策略已转入后台', `${item.name_zh} 正在调用 kejilion.sh 原生防火墙规则。`)
      return
    }
    toast.success(next === 'domain_only' ? '已阻止 IP + 端口访问' : '已放行 IP + 端口访问')
    await load(true)
  } catch (reason) {
    toast.danger('访问策略变更失败', reason instanceof ApiError ? reason.message : '容器端口绑定未能完成切换。')
  } finally {
    operation.value = ''
  }
}

async function addDomain(): Promise<void> {
  const item = selected.value
  const port = selectedPort.value?.publicPort
  if (applicationTaskActive.value || !item || !port || !domain.value.trim()) return
  domainError.value = ''
  domainWarning.value = ''
  operation.value = 'add_domain'
  const hostname = domain.value.trim().toLowerCase()
  try {
    const createdSite = await api.sites.create({
      primaryDomain: hostname,
      aliases: [],
      type: 'proxy',
      upstream: `http://127.0.0.1:${port}`,
      enabled: true,
    })
    sites.value = [createdSite, ...sites.value.filter((site) => site.id !== createdSite.id)]
    domain.value = ''
  } catch (reason) {
    domainError.value = reason instanceof ApiError ? reason.message : '域名绑定失败，请检查网站与 Nginx 状态。'
    operation.value = ''
    return
  }

  try {
    const refreshedInventory = await api.apps.inventory()
    inventory.value = refreshedInventory
    const refreshedItem = refreshedInventory.items.find((candidate) => candidate.id === item.id)
    if (!refreshedItem) throw new Error('应用详情暂时无法重新读取')
    if (
      capability(refreshedItem, 'direct_access') &&
      refreshedItem.runtime.accessMode !== 'domain_only' &&
      refreshedItem.runtime.resourceVersion
    ) {
      const result = await api.apps.action(refreshedItem.id, 'direct_access', {
        resourceVersion: refreshedItem.runtime.resourceVersion,
        accessMode: 'domain_only',
      })
      if (isBackgroundJob(result)) {
        startJobPolling(result)
        toast.success('域名已绑定', `${hostname} 已生效，IP + 端口阻止规则正在后台应用。`)
        return
      }
      toast.success('域名已绑定', `${hostname} 已生效，并已阻止 IP + 端口直接访问。`)
      await load(true)
      return
    }
    toast.success('域名已绑定', `${hostname} 已反向代理到 ${refreshedItem.name_zh}。`)
  } catch (reason) {
    const detail = reason instanceof ApiError ? reason.message : '应用状态暂时无法刷新'
    domainWarning.value = `域名已绑定，但 IP + 端口访问策略未调整：${detail}`
    toast.success('域名已绑定', `${hostname} 已生效；直接访问策略可稍后单独调整。`)
  } finally {
    operation.value = ''
  }
}

async function removeDomain(site: Site): Promise<void> {
  if (applicationTaskActive.value) return
  operation.value = `remove_domain:${site.id}`
  domainError.value = ''
  domainWarning.value = ''
  try {
    let removed = false
    try {
      await api.sites.remove(site.id, site.resourceVersion, 'configuration')
      removed = true
    } catch (reason) {
      if (!(reason instanceof ApiError) || reason.code !== 'resource_conflict') throw reason
      const refreshed = await api.sites.list()
      sites.value = refreshed.items
      const current = refreshed.items.find(
        (candidate) =>
          candidate.id === site.id ||
          candidate.primaryDomain === site.primaryDomain,
      )
      if (current) {
        await api.sites.remove(current.id, current.resourceVersion, 'configuration')
      }
      removed = true
    }
    if (!removed) return
    toast.success('域名已解绑', `${site.primaryDomain} 的反向代理已移除。`)
    await load(true)
  } catch (reason) {
    domainError.value = reason instanceof ApiError ? reason.message : '域名解绑失败，原配置保持不变。'
  } finally {
    operation.value = ''
  }
}

function openURL(item: AppMarketItem): string {
  const directHost =
    publicNetwork.value?.ipv4 ||
    publicNetwork.value?.ipv6 ||
    window.location.hostname
  return appAccessURL(item, sites.value, directHost)
}

onMounted(() => {
  void load()
  void restoreBackgroundJob()
})
onBeforeUnmount(() => {
  controller?.abort()
  stopJobPolling()
})
</script>

<template>
  <div class="page app-market">
    <PageHeader
      title="应用市场"
      description="已安装应用优先呈现；脚本内置与第三方应用使用 kejilion.sh 原生交互流程在后台安装。"
    >
      <template #actions>
        <a class="button button--secondary" href="https://app.kejilion.sh" target="_blank" rel="noopener noreferrer">
          <ArrowUpRight :size="16" />
          官方目录
        </a>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="16" :class="{ spin: refreshing }" />
          刷新状态
        </button>
      </template>
    </PageHeader>

    <section v-if="inventory" class="market-hero">
      <div class="market-hero__copy">
        <span class="market-hero__eyebrow"><Store :size="15" /> KPanel App Center</span>
        <h2>从发现到运行，一站式管理</h2>
        <p>应用、容器、域名和访问策略保持在同一个工作流中，且兼容脚本的应用编号与端口产物。</p>
      </div>
      <div class="market-stats">
        <div><strong>{{ inventory.items.length }}</strong><span>全部应用</span></div>
        <div><strong>{{ inventory.installed }}</strong><span>已安装</span></div>
        <div><strong>{{ inventory.running }}</strong><span>运行中</span></div>
        <div><strong>{{ inventory.items.filter((item) => capability(item, 'install') || capability(item, 'update')).length }}</strong><span>可直接安装</span></div>
      </div>
    </section>

    <div v-if="inventory?.catalogWarning" class="inline-alert inline-alert--warning">
      {{ inventory.catalogWarning }}
    </div>

    <section v-if="activeJob" class="app-job-banner" :class="`is-${activeJob.status}`">
      <span class="app-job-banner__icon">
        <LoaderCircle v-if="isActiveJob(activeJob)" class="spin" :size="20" />
        <CheckCircle2 v-else-if="activeJob.status === 'succeeded'" :size="20" />
        <Activity v-else :size="20" />
      </span>
      <div class="app-job-banner__body">
        <span>
          <strong>{{ activeJob.appName }}</strong>
          <StatusBadge :status="activeJob.status" subtle />
        </span>
        <small>{{ activeJob.message || '正在准备后台任务…' }}</small>
        <i class="app-job-banner__progress">
          <b :style="{ width: `${activeJob.progress || 0}%` }" />
        </i>
      </div>
      <strong class="app-job-banner__percent">
        {{ activeJob.interactive && isActiveJob(activeJob) ? '交互中' : `${activeJob.progress || 0}%` }}
      </strong>
      <button class="button button--secondary button--small" type="button" @click="jobDetailsOpen = true">
        查看进度 <ChevronRight :size="15" />
      </button>
    </section>

    <section v-if="inventory" class="market-toolbar">
      <label class="market-search">
        <Search :size="18" />
        <input v-model="search" type="search" placeholder="搜索应用名称、功能或容器…" />
      </label>
      <div class="market-segment" aria-label="来源筛选">
        <button
          v-for="item in [
            { key: 'all', label: '全部来源' },
            { key: 'builtin', label: '脚本内置' },
            { key: 'thirdparty', label: '第三方' },
          ]"
          :key="item.key"
          type="button"
          :class="{ 'is-active': source === item.key }"
          @click="source = item.key as SourceFilter"
        >
          {{ item.label }}
        </button>
      </div>
      <div class="market-segment" aria-label="状态筛选">
        <button
          v-for="item in [
            { key: 'installed', label: '已安装' },
            { key: 'all', label: '全部应用' },
            { key: 'running', label: '运行中' },
            { key: 'adapted', label: '可直接安装' },
          ]"
          :key="item.key"
          type="button"
          :class="{ 'is-active': status === item.key }"
          @click="status = item.key as StatusFilter"
        >
          {{ item.label }}
        </button>
      </div>
    </section>

    <nav v-if="inventory" class="market-categories" aria-label="应用分类">
      <button :class="{ 'is-active': category === 'all' }" type="button" @click="category = 'all'">
        全部 <span>{{ categoryCounts.all }}</span>
      </button>
      <button
        v-for="item in inventory.categories"
        :key="item.key"
        :class="{ 'is-active': category === item.key }"
        type="button"
        @click="category = item.key"
      >
        {{ item.zh }} <span>{{ categoryCounts[item.key] || 0 }}</span>
      </button>
    </nav>

    <LoadingState v-if="loading" title="正在读取应用市场…" />
    <ErrorState v-else-if="error && !inventory" :message="error" @retry="load()" />
    <EmptyState
      v-else-if="!filteredApps.length"
      title="没有符合条件的应用"
      description="尝试清除搜索词或切换分类与状态筛选。"
    />

    <section v-else class="app-grid" aria-live="polite">
      <article
        v-for="item in filteredApps"
        :key="item.id"
        class="app-card"
        :class="{ 'is-installed': item.runtime.installed }"
      >
        <button class="app-card__main" type="button" @click="openDetails(item)">
          <span class="app-card__icon">
            <img :src="item.icon" :alt="`${item.name_zh} 图标`" width="128" height="128" loading="lazy" />
          </span>
          <span class="app-card__body">
            <span class="app-card__title">
              <strong>{{ item.name_zh }}</strong>
              <StatusBadge
                v-if="item.runtime.installed"
                :status="item.runtime.state"
                :label="stateLabel(item)"
                subtle
              />
            </span>
            <span class="app-card__meta">
              <em>{{ categoryName(item.cat) }}</em>
              <em>{{ item.source === 'builtin' ? `内置 #${item.num}` : '第三方' }}</em>
              <em v-if="capability(item, 'install') || capability(item, 'update')" class="is-adapted">
                <ShieldCheck :size="12" /> 可直接安装
              </em>
            </span>
            <span class="app-card__description">{{ item.desc_zh }}</span>
          </span>
        </button>
        <footer class="app-card__footer">
          <span v-if="item.runtime.installed" class="app-card__runtime">
            <span :class="['runtime-dot', `is-${item.runtime.state}`]" />
            {{ item.runtime.containerName || '已标记安装' }}
          </span>
          <span v-else class="app-card__runtime">{{ item.name_en }}</span>
          <button
            v-if="!item.runtime.installed && capability(item, 'install')"
            class="button button--primary button--small"
            type="button"
            @click="openInstall(item)"
          >
            <Download :size="14" /> 安装
          </button>
          <button v-else class="button button--ghost button--small" type="button" @click="openDetails(item)">
            {{ item.runtime.installed ? '管理' : '了解详情' }}
          </button>
        </footer>
      </article>
    </section>

    <section v-if="inventory && status === 'installed'" class="install-more-card">
      <span><Store :size="22" /></span>
      <div>
        <strong>{{ inventory.installed ? '还想安装更多应用？' : '还没有安装应用' }}</strong>
        <p>前往完整应用列表，选择支持后台安装的应用；安装期间可以继续使用面板。</p>
      </div>
      <button class="button button--primary" type="button" @click="showAllApps">
        浏览全部应用 <ChevronRight :size="16" />
      </button>
    </section>

    <footer v-if="inventory && filteredApps.length" class="market-result">
      已显示 {{ filteredApps.length }} / {{ inventory.items.length }} 个应用
      <span>
        目录来源 app.kejilion.sh ·
        {{ inventory.catalogMode === 'live' ? '动态同步' : inventory.catalogMode === 'cached' ? '安全缓存' : '内置快照' }}
        · 状态来源宿主机
      </span>
    </footer>

    <ModalDialog
      :open="Boolean(selected) && !installOpen && !confirmAction"
      :title="selected?.name_zh || '应用详情'"
      :description="selected?.desc_zh"
      size="wide"
      @close="selectedID = ''"
    >
      <template v-if="selected">
        <div class="app-detail-head">
          <span class="app-detail-head__icon">
            <img :src="selected.icon" alt="" width="128" height="128" />
          </span>
          <div>
            <span class="app-detail-head__badges">
              <StatusBadge
                :status="selected.runtime.installed ? selected.runtime.state : 'unknown'"
                :label="stateLabel(selected)"
              />
              <span class="source-pill">{{ selected.source === 'builtin' ? `脚本内置 #${selected.num}` : '第三方应用' }}</span>
              <span class="source-pill">{{ categoryName(selected.cat) }}</span>
            </span>
            <strong>{{ selected.name_en }}</strong>
            <small><code>k app {{ selected.token }}</code></small>
          </div>
          <a
            v-if="selected.url"
            class="icon-button"
            :href="selected.url"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="打开应用官网"
          >
            <ArrowUpRight :size="18" />
          </a>
        </div>

        <div v-if="selected.runtime.warning" class="inline-alert inline-alert--warning">
          <Wrench :size="17" /> {{ selected.runtime.warning }}
        </div>

        <section v-if="selected.runtime.installed" class="app-control-panel">
          <div class="app-control-panel__status">
            <div>
              <span>运行状态</span>
              <strong>{{ stateLabel(selected) }}</strong>
              <small>{{ selected.runtime.status || selected.runtime.image || '已由脚本标记安装' }}</small>
            </div>
            <div>
              <span>更新状态</span>
              <strong>{{ updateLabel(selected) }}</strong>
              <small>{{ capability(selected, 'update') ? '可安全拉取并回滚' : '保留原管理方式' }}</small>
            </div>
            <div>
              <span>访问策略</span>
              <strong>{{ selected.runtime.accessMode === 'domain_only' ? '仅域名访问' : selected.runtime.accessMode === 'direct' ? 'IP + 端口' : '未识别' }}</strong>
              <small>{{ selectedPort ? `${selectedPort.ip || '0.0.0.0'}:${selectedPort.publicPort}` : '没有可用 HTTP 端口' }}</small>
            </div>
          </div>
          <div class="app-control-panel__actions">
            <button
              v-if="capability(selected, 'manage')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation) || applicationTaskActive"
              title="打开该应用对应的 kejilion.sh 原生交互菜单"
              @click="openScriptManage"
            >
              <LoaderCircle v-if="operation === 'manage'" class="spin" :size="15" />
              <Wrench v-else :size="15" /> 脚本管理
            </button>
            <button
              v-if="capability(selected, 'start')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation) || applicationTaskActive"
              @click="lifecycle('start')"
            >
              <Play :size="15" /> 启动
            </button>
            <button
              v-if="capability(selected, 'stop')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation) || applicationTaskActive"
              @click="lifecycle('stop')"
            >
              <Square :size="14" /> 停止
            </button>
            <button
              v-if="capability(selected, 'restart')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation) || applicationTaskActive"
              @click="lifecycle('restart')"
            >
              <RotateCw :size="15" /> 重启
            </button>
            <button
              v-if="capability(selected, 'check_update')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation) || applicationTaskActive"
              @click="checkUpdate"
            >
              <LoaderCircle v-if="operation === 'check_update'" class="spin" :size="15" />
              <RefreshCw v-else :size="15" /> 检查更新
            </button>
            <a
              v-if="openURL(selected)"
              class="button button--primary"
              :href="openURL(selected)"
              target="_blank"
              rel="noopener noreferrer"
            >
              <ArrowUpRight :size="15" /> 打开应用
            </a>
          </div>
        </section>

        <div v-if="!selected.runtime.installed" class="app-install-state">
          <PackageCheck :size="25" />
          <div>
            <strong>当前未安装</strong>
            <p v-if="capability(selected, 'install') && selected.installer === 'kejilion'">
              此应用会在后台打开 kejilion.sh 原生交互终端；专属安装向导、端口、域名和凭据输入均与 SSH 端一致。
            </p>
            <p v-else-if="capability(selected, 'install')">
              此应用已有固定镜像、端口和回滚策略，可以由 KPanel 在后台安全安装。
            </p>
            <p v-else>{{ selected.capabilities.install?.reason || '等待专属安装适配器。' }}</p>
          </div>
          <button
            v-if="capability(selected, 'install')"
            class="button button--primary"
            type="button"
            @click="openInstall(selected)"
          >
            <Download :size="16" /> 开始安装
          </button>
        </div>

        <div v-if="selected.runtime.installed" class="app-detail-grid">
          <section class="app-detail-section app-detail-section--domain">
            <header><Globe2 :size="18" /><div><strong>域名访问</strong><small>复用 KPanel 网站反向代理</small></div></header>
            <div v-if="selectedDomains.length" class="domain-list">
              <div
                v-for="site in selectedDomains"
                :key="site.id"
                class="domain-list__item"
              >
                <a :href="`http://${site.primaryDomain}`" target="_blank" rel="noopener noreferrer">
                  <CheckCircle2 :size="15" /> {{ site.primaryDomain }} <ArrowUpRight :size="13" />
                </a>
                <button
                  class="icon-button icon-button--small icon-button--danger"
                  type="button"
                  :disabled="Boolean(operation) || applicationTaskActive"
                  aria-label="解绑域名"
                  @click="removeDomain(site)"
                >
                  <LoaderCircle v-if="operation === `remove_domain:${site.id}`" class="spin" :size="14" />
                  <Trash2 v-else :size="14" />
                </button>
              </div>
            </div>
            <form v-if="capability(selected, 'add_domain')" class="domain-form" @submit.prevent="addDomain">
              <label class="field">
                <span>添加域名</span>
                <input v-model.trim="domain" placeholder="app.example.com" autocomplete="off" required />
              </label>
              <button
                class="button button--secondary"
                type="submit"
                :disabled="operation === 'add_domain' || applicationTaskActive || !domain"
              >
                <LoaderCircle v-if="operation === 'add_domain'" class="spin" :size="15" />
                <Globe2 v-else :size="15" /> 绑定
              </button>
            </form>
            <p v-if="sitesWarning" class="field-warning">{{ sitesWarning }}</p>
            <p v-if="domainError" class="field-error">{{ domainError }}</p>
            <p v-if="domainWarning" class="field-warning">{{ domainWarning }}</p>
            <p v-if="!capability(selected, 'add_domain')" class="muted-note">
              {{ selected.capabilities.add_domain?.reason }}
            </p>
            <DnsResolutionGuide
              :ipv4="publicNetwork?.ipv4"
              :ipv6="publicNetwork?.ipv6"
              compact
            />
          </section>

          <section class="app-detail-section app-detail-section--access">
            <header><Network :size="18" /><div><strong>IP + 端口访问</strong><small>通过容器监听地址切换，不写入全局防火墙</small></div></header>
            <div class="access-card">
              <span :class="selected.runtime.accessMode === 'domain_only' ? 'is-locked' : 'is-open'">
                <LockKeyhole v-if="selected.runtime.accessMode === 'domain_only'" :size="19" />
                <UnlockKeyhole v-else :size="19" />
              </span>
              <div>
                <strong>{{ selected.runtime.accessMode === 'domain_only' ? '已阻止直接访问' : '允许直接访问' }}</strong>
                <small>域名反向代理不受影响</small>
              </div>
              <button
                class="button button--secondary button--small"
                type="button"
                :disabled="!capability(selected, 'direct_access') || Boolean(operation) || applicationTaskActive"
                :title="selected.capabilities.direct_access?.reason"
                @click="toggleAccess"
              >
                {{ selected.runtime.accessMode === 'domain_only' ? '放行' : '阻止' }}
              </button>
            </div>
          </section>
        </div>

        <section v-if="selected.runtime.installed" class="danger-zone">
          <div>
            <strong>维护与卸载</strong>
            <small>更新失败会自动恢复旧容器；卸载只删除已核验的容器与兼容标记，不清理共享镜像。</small>
          </div>
          <button
            class="button button--secondary"
            type="button"
            :disabled="!capability(selected, 'update') || Boolean(operation) || applicationTaskActive"
            :title="selected.capabilities.update?.reason"
            @click="confirmAction = 'update'"
          >
            <RefreshCw :size="15" /> 更新
          </button>
          <button
            class="button button--danger"
            type="button"
            :disabled="!capability(selected, 'uninstall') || Boolean(operation) || applicationTaskActive"
            :title="selected.capabilities.uninstall?.reason"
            @click="confirmAction = 'uninstall'"
          >
            <Trash2 :size="15" /> 卸载
          </button>
        </section>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="installOpen && Boolean(selected)"
      :title="`安装 ${selected?.name_zh || ''}`"
      description="任务提交后会在宿主机后台运行；关闭窗口或切换页面不会中断安装。"
      size="small"
      @close="installOpen = false"
    >
      <form id="app-install-form" class="form-stack" @submit.prevent="install">
        <label v-if="selected?.installer !== 'kejilion'" class="field">
          <span>访问端口</span>
          <input
            v-model.number="installPort"
            type="number"
            min="1"
            max="65535"
            :placeholder="selected?.defaultPort ? String(selected.defaultPort) : '留空使用脚本默认端口'"
          />
          <small>留空时沿用 kejilion.sh 默认端口；低位系统端口和已有服务发生冲突时请换用其他端口。</small>
        </label>
        <fieldset v-if="selected?.installer === 'declarative'" class="access-options">
          <legend>初始访问方式</legend>
          <button
            type="button"
            :class="{ 'is-active': installAccess === 'direct' }"
            @click="installAccess = 'direct'"
          >
            <UnlockKeyhole :size="18" /><span><strong>IP + 端口</strong><small>安装后立即可访问</small></span>
          </button>
          <button
            type="button"
            :class="{ 'is-active': installAccess === 'domain_only' }"
            @click="installAccess = 'domain_only'"
          >
            <LockKeyhole :size="18" /><span><strong>仅域名访问</strong><small>绑定到 127.0.0.1</small></span>
          </button>
        </fieldset>
        <div class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          {{
            selected?.installer === 'kejilion'
              ? '提交后直接打开 kejilion.sh 网页终端；端口、域名、账号和密码均按原脚本提示输入。'
              : '使用固定声明式模板；容器创建失败不会写入脚本安装标记。'
          }}
        </div>
      </form>
      <template #footer>
        <button class="button button--secondary" type="button" @click="installOpen = false">取消</button>
        <button class="button button--primary" type="submit" form="app-install-form" :disabled="Boolean(operation)">
          <LoaderCircle v-if="operation === 'install'" class="spin" :size="16" />
          <Download v-else :size="16" /> {{ operation === 'install' ? '正在提交…' : '后台安装' }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="jobDetailsOpen && Boolean(activeJob)"
      :title="`${activeJob?.appName || ''} ${jobActionLabel(activeJob?.action)}进度`"
      description="任务由宿主机后台执行，离开本页面不会中断。"
      size="large"
      @close="jobDetailsOpen = false"
    >
      <template v-if="activeJob">
        <div class="job-detail-summary">
          <span class="app-job-banner__icon">
            <LoaderCircle v-if="isActiveJob(activeJob)" class="spin" :size="21" />
            <CheckCircle2 v-else-if="activeJob.status === 'succeeded'" :size="21" />
            <Activity v-else :size="21" />
          </span>
          <div>
            <strong>{{ activeJob.message || `正在执行${jobActionLabel(activeJob.action)}任务` }}</strong>
            <small>阶段：{{ activeJob.stage }} · 任务 {{ activeJob.id }}</small>
          </div>
          <StatusBadge :status="activeJob.status" />
        </div>
        <div v-if="!activeJob.interactive" class="job-detail-progress">
          <i><b :style="{ width: `${activeJob.progress || 0}%` }" /></i>
          <strong>{{ activeJob.progress || 0 }}%</strong>
        </div>
        <AppInteractiveTerminal
          v-if="activeJob.interactive"
          :key="activeJob.id"
          :job-id="activeJob.id"
          :input-open="activeJob.inputOpen"
        />
        <section v-else class="job-log">
          <header>
            <strong>实时日志</strong>
            <small>显示最近 {{ activeJob.logs.length }} 行</small>
          </header>
          <pre v-if="activeJob.logs.length">{{ activeJob.logs.join('\n') }}</pre>
          <p v-else>任务已进入队列，正在等待首批输出…</p>
        </section>
      </template>
      <template #footer>
        <button class="button button--secondary" type="button" @click="jobDetailsOpen = false">
          后台运行
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(confirmAction)"
      :title="confirmAction === 'uninstall' ? '确认卸载应用？' : '确认更新应用？'"
      :description="
        confirmAction === 'uninstall'
          ? selected?.installer === 'kejilion'
            ? '后台调用 kejilion.sh 原生卸载函数；仅在主容器 ID 与安装标记同时匹配时执行。'
            : '容器会停止并删除；共享镜像缓存不会删除。'
          : selected?.installer === 'kejilion'
            ? '后台调用 kejilion.sh 原生更新函数，并在更新后恢复原访问策略。'
            : 'KPanel 会先拉取新镜像，失败时恢复原容器。'
      "
      size="small"
      @close="confirmAction = undefined"
    >
      <div class="confirm-app">
        <img v-if="selected" :src="selected.icon" alt="" />
        <div><strong>{{ selected?.name_zh }}</strong><small>{{ selected?.runtime.containerName }}</small></div>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" @click="confirmAction = undefined">取消</button>
        <button
          class="button"
          :class="confirmAction === 'uninstall' ? 'button--danger' : 'button--primary'"
          type="button"
          :disabled="Boolean(operation)"
          @click="confirmMutation"
        >
          <LoaderCircle v-if="operation" class="spin" :size="16" />
          {{ confirmAction === 'uninstall' ? '确认卸载' : '开始更新' }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.app-market {
  --market-accent: #6d5dfc;
  --market-accent-soft: color-mix(in srgb, var(--market-accent) 12%, transparent);
}

.market-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(420px, 0.9fr);
  gap: 28px;
  overflow: hidden;
  padding: 28px;
  border: 1px solid color-mix(in srgb, var(--market-accent) 22%, var(--border));
  border-radius: 22px;
  background:
    radial-gradient(circle at 6% 0%, color-mix(in srgb, var(--market-accent) 22%, transparent), transparent 36%),
    linear-gradient(135deg, color-mix(in srgb, var(--surface) 92%, var(--market-accent)), var(--surface));
  box-shadow: var(--shadow-sm);
}

.market-hero::after {
  position: absolute;
  right: -90px;
  bottom: -130px;
  width: 290px;
  height: 290px;
  border: 52px solid color-mix(in srgb, var(--market-accent) 7%, transparent);
  border-radius: 50%;
  content: '';
}

.market-hero__copy,
.market-stats {
  position: relative;
  z-index: 1;
}

.market-hero__eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 12px;
  color: var(--market-accent);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.market-hero h2 {
  margin: 0 0 9px;
  font-size: clamp(24px, 3vw, 34px);
  letter-spacing: -0.04em;
}

.market-hero p {
  max-width: 660px;
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.75;
}

.market-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  align-self: center;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--border) 72%, transparent);
  border-radius: 16px;
  background: color-mix(in srgb, var(--surface) 80%, transparent);
  backdrop-filter: blur(12px);
}

.market-stats div {
  display: grid;
  gap: 4px;
  padding: 18px 14px;
  text-align: center;
}

.market-stats div + div {
  border-left: 1px solid var(--border);
}

.market-stats strong {
  font-size: 23px;
  letter-spacing: -0.04em;
}

.market-stats span {
  color: var(--text-tertiary);
  font-size: 12px;
}

.market-toolbar {
  display: grid;
  grid-template-columns: minmax(240px, 1fr) auto auto;
  gap: 12px;
  align-items: center;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--surface);
  box-shadow: var(--shadow-xs);
}

.market-search {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 0 13px;
  border: 1px solid var(--border);
  border-radius: 12px;
  color: var(--text-tertiary);
  background: var(--surface-muted);
}

.market-search:focus-within {
  border-color: color-mix(in srgb, var(--market-accent) 58%, var(--border));
  box-shadow: 0 0 0 3px var(--market-accent-soft);
}

.market-search input {
  width: 100%;
  height: 42px;
  border: 0;
  outline: 0;
  color: var(--text-primary);
  background: transparent;
}

.market-segment {
  display: flex;
  gap: 4px;
  padding: 4px;
  border-radius: 12px;
  background: var(--surface-muted);
}

.market-segment button,
.market-categories button {
  border: 0;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
}

.market-segment button {
  padding: 8px 11px;
  border-radius: 9px;
  font-size: 12px;
  white-space: nowrap;
}

.market-segment button.is-active,
.market-categories button.is-active {
  color: var(--market-accent);
  background: var(--surface);
  box-shadow: var(--shadow-xs);
}

.market-categories {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 3px;
  scrollbar-width: none;
}

.market-categories button {
  display: inline-flex;
  gap: 7px;
  align-items: center;
  flex: 0 0 auto;
  padding: 9px 13px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  font-size: 13px;
}

.market-categories button span {
  min-width: 20px;
  padding: 1px 6px;
  border-radius: 999px;
  color: var(--text-tertiary);
  background: var(--surface-muted);
  font-size: 11px;
}

.app-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 15px;
}

.app-card {
  display: flex;
  min-width: 0;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--surface);
  box-shadow: var(--shadow-xs);
  transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
}

.app-card:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--market-accent) 30%, var(--border));
  box-shadow: var(--shadow-sm);
}

.app-card.is-installed {
  border-color: color-mix(in srgb, var(--success) 26%, var(--border));
}

.app-card__main {
  display: flex;
  gap: 14px;
  flex: 1;
  padding: 17px;
  border: 0;
  color: inherit;
  text-align: left;
  background: transparent;
  cursor: pointer;
}

.app-card__icon,
.app-detail-head__icon {
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--border) 74%, transparent);
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 5px 14px rgb(16 24 40 / 8%);
}

.app-card__icon {
  width: 50px;
  height: 50px;
}

.app-card__icon img,
.app-detail-head__icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.app-card__body,
.app-card__title {
  min-width: 0;
}

.app-card__body {
  display: grid;
  gap: 7px;
}

.app-card__title {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.app-card__title strong {
  overflow: hidden;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-card__meta {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}

.app-card__meta em,
.source-pill {
  display: inline-flex;
  gap: 3px;
  align-items: center;
  padding: 3px 7px;
  border-radius: 999px;
  color: var(--text-tertiary);
  background: var(--surface-muted);
  font-size: 10px;
  font-style: normal;
}

.app-card__meta em.is-adapted {
  color: var(--success);
  background: color-mix(in srgb, var(--success) 10%, transparent);
}

.app-card__description {
  display: -webkit-box;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.65;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.app-card__footer {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  padding: 11px 14px;
  border-top: 1px solid var(--border);
}

.app-card__runtime {
  display: flex;
  min-width: 0;
  gap: 7px;
  align-items: center;
  overflow: hidden;
  color: var(--text-tertiary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--text-tertiary);
}

.runtime-dot.is-running {
  background: var(--success);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--success) 12%, transparent);
}

.market-result {
  display: flex;
  justify-content: space-between;
  color: var(--text-tertiary);
  font-size: 12px;
}

.app-detail-head {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 12px;
  align-items: center;
  margin-bottom: 10px;
}

.app-detail-head__icon {
  width: 56px;
  height: 56px;
  border-radius: 15px;
}

.app-detail-head > div {
  display: grid;
  gap: 5px;
}

.app-detail-head__badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.app-detail-head > div > strong {
  font-size: 18px;
}

.app-detail-head small {
  color: var(--text-tertiary);
}

.app-control-panel {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  overflow: hidden;
  margin-bottom: 10px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface-muted);
}

.app-control-panel__status {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
}

.app-control-panel__status > div {
  display: grid;
  gap: 5px;
  padding: 10px 12px;
}

.app-control-panel__status > div + div {
  border-left: 1px solid var(--border);
}

.app-control-panel__status span,
.app-control-panel__status small {
  color: var(--text-tertiary);
  font-size: 11px;
}

.app-control-panel__actions {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  padding: 9px 10px;
  border-left: 1px solid var(--border);
  background: var(--surface);
}

.app-control-panel__actions .button {
  min-height: 34px;
  padding: 7px 10px;
}

.app-install-state {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 15px;
  align-items: center;
  padding: 20px;
  border: 1px dashed color-mix(in srgb, var(--market-accent) 38%, var(--border));
  border-radius: 16px;
  color: var(--market-accent);
  background: var(--market-accent-soft);
}

.app-install-state p {
  margin: 4px 0 0;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.app-detail-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 9px;
}

.app-detail-section {
  padding: 11px 12px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
}

.app-detail-section > header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 8px;
  color: var(--market-accent);
}

.app-detail-section > header div {
  display: grid;
  gap: 2px;
}

.app-detail-section > header strong {
  color: var(--text-primary);
  font-size: 13px;
}

.app-detail-section > header small {
  color: var(--text-tertiary);
  font-size: 11px;
}

.app-detail-section--domain {
  display: grid;
  grid-template-columns: minmax(150px, 0.36fr) minmax(0, 1fr);
  gap: 8px 12px;
  align-items: end;
}

.app-detail-section--domain > header {
  align-self: center;
  margin-bottom: 0;
}

.app-detail-section--domain > .domain-list,
.app-detail-section--domain > .domain-form {
  grid-column: 2;
}

.app-detail-section--domain > .domain-list {
  margin-bottom: 0;
}

.app-detail-section--domain > :deep(.dns-guide),
.app-detail-section--domain > .field-warning,
.app-detail-section--domain > .field-error,
.app-detail-section--domain > .muted-note {
  grid-column: 1 / -1;
}

.app-detail-section--access {
  display: grid;
  grid-template-columns: minmax(220px, 0.55fr) minmax(0, 1fr);
  gap: 12px;
  align-items: center;
}

.app-detail-section--access > header {
  margin-bottom: 0;
}

.domain-list {
  display: grid;
  gap: 6px;
  margin-bottom: 6px;
}

.domain-list__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 9px;
  border-radius: 10px;
  background: var(--surface-muted);
}

.domain-list__item a {
  display: flex;
  gap: 7px;
  align-items: center;
  color: var(--text-primary);
  font-size: 12px;
  text-decoration: none;
}

.domain-form {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  align-items: end;
}

.field-error {
  margin: 7px 0 0;
  color: var(--danger);
  font-size: 11px;
}

.field-warning {
  margin: 7px 0 0;
  color: var(--amber);
  font-size: 11px;
  line-height: 1.5;
}

.muted-note {
  margin: 0;
  color: var(--text-tertiary);
  font-size: 12px;
  line-height: 1.6;
}

.access-card {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 10px;
  align-items: center;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--surface-muted);
}

.access-card > span {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 11px;
}

.access-card > span.is-locked {
  color: var(--success);
  background: color-mix(in srgb, var(--success) 12%, transparent);
}

.access-card > span.is-open {
  color: var(--warning);
  background: color-mix(in srgb, var(--warning) 12%, transparent);
}

.access-card > div {
  display: grid;
  gap: 3px;
}

.access-card small {
  color: var(--text-tertiary);
  font-size: 11px;
}

.danger-zone {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 9px;
  align-items: center;
  margin-top: 9px;
  padding-top: 9px;
  border-top: 1px solid var(--border);
}

.danger-zone > div {
  display: grid;
  gap: 4px;
}

.danger-zone small {
  color: var(--text-tertiary);
  font-size: 11px;
}

.access-options {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 9px;
  padding: 0;
  border: 0;
}

.access-options legend {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 700;
}

.access-options button {
  display: flex;
  gap: 9px;
  align-items: center;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 12px;
  color: var(--text-secondary);
  text-align: left;
  background: var(--surface);
  cursor: pointer;
}

.access-options button.is-active {
  border-color: var(--market-accent);
  color: var(--market-accent);
  background: var(--market-accent-soft);
}

.access-options button span {
  display: grid;
  gap: 2px;
}

.access-options button small {
  color: var(--text-tertiary);
}

.confirm-app {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 14px;
  border-radius: 14px;
  background: var(--surface-muted);
}

.confirm-app img {
  width: 46px;
  height: 46px;
  border-radius: 12px;
}

.confirm-app div {
  display: grid;
  gap: 4px;
}

.confirm-app small {
  color: var(--text-tertiary);
}

.app-job-banner {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  gap: 14px;
  align-items: center;
  padding: 15px 17px;
  border: 1px solid color-mix(in srgb, var(--market-accent) 28%, var(--border));
  border-radius: 16px;
  background: linear-gradient(120deg, var(--market-accent-soft), var(--surface));
  box-shadow: var(--shadow-xs);
}

.app-job-banner.is-failed {
  border-color: color-mix(in srgb, var(--danger) 35%, var(--border));
  background: color-mix(in srgb, var(--danger) 7%, var(--surface));
}

.app-job-banner__icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 12px;
  color: var(--market-accent);
  background: var(--surface);
  box-shadow: var(--shadow-xs);
}

.app-job-banner__body {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.app-job-banner__body > span {
  display: flex;
  gap: 8px;
  align-items: center;
}

.app-job-banner__body small {
  overflow: hidden;
  color: var(--text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-job-banner__progress,
.job-detail-progress i {
  display: block;
  overflow: hidden;
  height: 6px;
  border-radius: 99px;
  background: color-mix(in srgb, var(--border) 70%, transparent);
}

.app-job-banner__progress b,
.job-detail-progress b {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--market-accent), #8e83ff);
  transition: width 0.35s ease;
}

.app-job-banner__percent {
  color: var(--market-accent);
  font-variant-numeric: tabular-nums;
}

.install-more-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 16px;
  align-items: center;
  padding: 20px;
  border: 1px dashed color-mix(in srgb, var(--market-accent) 38%, var(--border));
  border-radius: 18px;
  background: color-mix(in srgb, var(--market-accent) 5%, var(--surface));
}

.install-more-card > span {
  display: grid;
  width: 46px;
  height: 46px;
  place-items: center;
  border-radius: 14px;
  color: var(--market-accent);
  background: var(--market-accent-soft);
}

.install-more-card div {
  display: grid;
  gap: 4px;
}

.install-more-card p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 13px;
}

.job-detail-summary {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 13px;
  align-items: center;
}

.job-detail-summary > div {
  display: grid;
  gap: 4px;
}

.job-detail-summary small {
  color: var(--text-tertiary);
  font-size: 11px;
}

.job-detail-progress {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  align-items: center;
  margin-top: 18px;
}

.job-detail-progress i {
  height: 9px;
}

.job-detail-progress strong {
  min-width: 44px;
  color: var(--market-accent);
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.job-log {
  overflow: hidden;
  margin-top: 18px;
  border: 1px solid var(--border);
  border-radius: 14px;
}

.job-log header {
  display: flex;
  justify-content: space-between;
  padding: 11px 13px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-muted);
}

.job-log header small {
  color: var(--text-tertiary);
}

.job-log pre,
.job-log p {
  overflow: auto;
  max-height: 340px;
  margin: 0;
  padding: 14px;
  color: #dce4ff;
  font: 12px/1.65 var(--font-mono);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: #111827;
}

@media (max-width: 1280px) {
  .market-hero {
    grid-template-columns: 1fr;
  }

  .app-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .market-toolbar {
    grid-template-columns: 1fr;
  }

  .market-segment {
    overflow-x: auto;
  }

  .app-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .app-detail-grid {
    grid-template-columns: 1fr;
  }

  .app-control-panel {
    grid-template-columns: 1fr;
  }

  .app-control-panel__actions {
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .app-detail-section--domain,
  .app-detail-section--access {
    grid-template-columns: 1fr;
  }

  .app-detail-section--domain > header,
  .app-detail-section--access > header {
    margin-bottom: 8px;
  }

  .app-detail-section--domain > .domain-list,
  .app-detail-section--domain > .domain-form,
  .app-detail-section--domain > :deep(.dns-guide),
  .app-detail-section--domain > .field-warning,
  .app-detail-section--domain > .field-error,
  .app-detail-section--domain > .muted-note {
    grid-column: 1;
  }
}

@media (max-width: 640px) {
  .market-hero {
    padding: 21px;
  }

  .market-stats {
    grid-template-columns: repeat(2, 1fr);
  }

  .market-stats div:nth-child(3) {
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .market-stats div:nth-child(4) {
    border-top: 1px solid var(--border);
  }

  .app-grid {
    grid-template-columns: 1fr;
  }

  .market-result,
  .app-control-panel__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .app-control-panel__status {
    grid-template-columns: 1fr;
  }

  .app-control-panel__status > div + div {
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .app-install-state,
  .danger-zone {
    grid-template-columns: 1fr;
  }

  .access-options {
    grid-template-columns: 1fr;
  }

  .app-job-banner,
  .install-more-card,
  .job-detail-summary {
    grid-template-columns: 1fr;
  }

  .app-job-banner__percent {
    display: none;
  }
}
</style>
