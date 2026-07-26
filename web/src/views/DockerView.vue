<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Box,
  Boxes,
  BrushCleaning,
  CircleStop,
  Container,
  Download,
  FileText,
  HardDrive,
  LoaderCircle,
  Network,
  Play,
  Plus,
  RefreshCw,
  RotateCw,
  Search,
  ShieldCheck,
  Trash2,
  Waypoints,
  Wrench,
} from '@lucide/vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { formatBytes, formatDateTime, relativeTime, shortId } from '@/lib/format'
import { usePanelState } from '@/stores/panel'
import { useToast } from '@/stores/toast'
import type {
  DockerContainer,
  DockerInventory,
  DockerMaintenanceInput,
  DockerMaintenanceJob,
} from '@/types/api'

type DockerTab = 'containers' | 'images' | 'networks' | 'volumes'
type ContainerAction = 'start' | 'stop' | 'restart' | 'remove'

const data = ref<DockerInventory>()
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const search = ref('')
const activeTab = ref<DockerTab>('containers')
const selectedContainer = ref<DockerContainer>()
const pendingAction = ref<ContainerAction>()
const actionRunning = ref(false)
const logsOpen = ref(false)
const logsLoading = ref(false)
const logLines = ref<string[]>([])
const logError = ref('')
const toolsOpen = ref(false)
const taskRunning = ref(false)
const activeJob = ref<DockerMaintenanceJob>()
const imageReference = ref('')
const networkName = ref('')
const volumeName = ref('')
const membershipContainerID = ref('')
const membershipNetworkID = ref('')
const mirrorPreset = ref<'cn' | 'official'>('cn')
const ipv6Enabled = ref(false)
const ipv6CIDR = ref('fd42:6b50:616e:656c::/64')
const pruneConfirmation = ref('')
const pendingMaintenance = ref<{
  title: string
  description: string
  input: DockerMaintenanceInput
}>()
const panel = usePanelState()
const toast = useToast()
let controller: AbortController | undefined
let logController: AbortController | undefined
let jobTimer: number | undefined
const activeDockerJobKey = 'kpanel.active-docker-job'

const tabs = computed(() => [
  { id: 'containers' as const, label: '容器', count: data.value?.containers.length || 0, icon: Container },
  { id: 'images' as const, label: '镜像', count: data.value?.images.length || 0, icon: Box },
  { id: 'networks' as const, label: '网络', count: data.value?.networks.length || 0, icon: Network },
  { id: 'volumes' as const, label: '存储卷', count: data.value?.volumes.length || 0, icon: HardDrive },
])

const runningCount = computed(() => data.value?.containers.filter((item) => item.state === 'running').length || 0)
const managedCount = computed(() => data.value?.containers.filter((item) => item.access === 'managed').length || 0)
const membershipContainers = computed(() =>
  (data.value?.containers || []).filter(
    (item) => item.access === 'managed' && item.name !== 'kejilion-panel' && item.resourceVersion,
  ),
)
const membershipNetworks = computed(() =>
  (data.value?.networks || []).filter(
    (item) => item.resourceVersion && !['bridge', 'host', 'none', 'kejilion-panel-network'].includes(item.name),
  ),
)

const filteredContainers = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return data.value?.containers || []
  return (data.value?.containers || []).filter(
    (item) =>
      item.name.toLowerCase().includes(query) ||
      item.image.toLowerCase().includes(query) ||
      item.project?.toLowerCase().includes(query),
  )
})

const filteredImages = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return data.value?.images || []
  return (data.value?.images || []).filter(
    (item) => item.id.toLowerCase().includes(query) || item.tags.some((tag) => tag.toLowerCase().includes(query)),
  )
})

const filteredNetworks = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return data.value?.networks || []
  return (data.value?.networks || []).filter(
    (item) => item.name.toLowerCase().includes(query) || item.driver.toLowerCase().includes(query),
  )
})

const filteredVolumes = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return data.value?.volumes || []
  return (data.value?.volumes || []).filter(
    (item) => item.name.toLowerCase().includes(query) || item.driver.toLowerCase().includes(query),
  )
})

function formatPorts(container: DockerContainer): string {
  if (!container.ports.length) return '无公开端口'
  return container.ports
    .map((port) => `${port.publicPort ? `${port.ip || '0.0.0.0'}:${port.publicPort} → ` : ''}${port.privatePort}/${port.protocol}`)
    .join('，')
}

function permits(container: DockerContainer, action: ContainerAction): boolean {
  return Boolean(
    container.resourceVersion &&
    container.allowedActions?.some(
      (allowed) => allowed === action || allowed.endsWith(`.${action}`) || allowed.endsWith(`/${action}`),
    ),
  )
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''

  try {
    data.value = await api.docker.inventory(controller.signal, (partial) => {
      data.value = partial
      loading.value = false
    })
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    error.value = reason instanceof ApiError ? reason.message : '无法读取 Docker 资源。'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function askAction(container: DockerContainer, action: ContainerAction): void {
  selectedContainer.value = container
  pendingAction.value = action
}

async function runAction(): Promise<void> {
  if (!selectedContainer.value || !pendingAction.value || !selectedContainer.value.resourceVersion) return
  actionRunning.value = true

  try {
    const action = pendingAction.value
    const result = await api.docker.action(
      selectedContainer.value.id,
      action,
      selectedContainer.value.resourceVersion,
    )
    toast.success(
      result.jobId ? '容器操作已进入任务队列' : '容器操作已完成',
      result.jobId
        ? `${selectedContainer.value.name} · ${shortId(result.jobId)}`
        : pendingAction.value === 'remove'
          ? `${selectedContainer.value.name} 已删除，存储卷与镜像保留`
          : selectedContainer.value.name,
    )
    selectedContainer.value = undefined
    pendingAction.value = undefined
    await load(true)
  } catch (reason) {
    toast.danger('容器操作失败', reason instanceof ApiError ? reason.message : '请稍后重试。')
  } finally {
    actionRunning.value = false
  }
}

async function showLogs(container: DockerContainer): Promise<void> {
  selectedContainer.value = container
  logsOpen.value = true
  logsLoading.value = true
  logLines.value = []
  logError.value = ''
  logController?.abort()
  logController = new AbortController()

  try {
    const result = await api.docker.logs(container.id, 200, logController.signal)
    logLines.value = result.lines || []
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    logError.value = reason instanceof ApiError ? reason.message : '无法读取容器日志。'
  } finally {
    logsLoading.value = false
  }
}

function closeLogs(): void {
  logController?.abort()
  logsOpen.value = false
  selectedContainer.value = undefined
}

function stopJobPolling(): void {
  if (jobTimer) window.clearInterval(jobTimer)
  jobTimer = undefined
}

async function refreshJob(id: string): Promise<void> {
  try {
    const job = await api.docker.job(id)
    activeJob.value = job
    if (job.status === 'queued' || job.status === 'running') return
    stopJobPolling()
    window.localStorage.removeItem(activeDockerJobKey)
    if (job.status === 'succeeded') {
      toast.success(
        'Docker 后台任务完成',
        job.resultPath ? `备份文件：${job.resultPath}` : job.message || job.target || '资源状态已更新。',
      )
      await load(true)
    } else {
      toast.danger('Docker 后台任务失败', job.message || '请刷新资源后重试。')
    }
  } catch {
    stopJobPolling()
    window.localStorage.removeItem(activeDockerJobKey)
  }
}

function startJobPolling(job: DockerMaintenanceJob): void {
  stopJobPolling()
  activeJob.value = job
  window.localStorage.setItem(activeDockerJobKey, job.id)
  void refreshJob(job.id)
  jobTimer = window.setInterval(() => void refreshJob(job.id), 1_500)
}

async function restoreBackgroundJob(): Promise<void> {
  const saved = window.localStorage.getItem(activeDockerJobKey)
  if (saved) {
    try {
      const job = await api.docker.job(saved)
      activeJob.value = job
      if (job.status === 'queued' || job.status === 'running') startJobPolling(job)
      else window.localStorage.removeItem(activeDockerJobKey)
      return
    } catch {
      window.localStorage.removeItem(activeDockerJobKey)
    }
  }
  try {
    const jobs = await api.docker.jobs()
    const running = jobs.items.find((job) => job.status === 'queued' || job.status === 'running')
    if (running) startJobPolling(running)
  } catch {
    // Older Agents remain usable without the background task endpoints.
  }
}

async function submitTask(input: DockerMaintenanceInput): Promise<void> {
  if (taskRunning.value || panel.isReadOnly.value) return
  taskRunning.value = true
  try {
    const job = await api.docker.task(input)
    pendingMaintenance.value = undefined
    startJobPolling(job)
    toast.success('已转入后台执行', '可以离开 Docker 页面，任务会继续运行。')
  } catch (reason) {
    toast.danger('Docker 任务提交失败', reason instanceof ApiError ? reason.message : 'Agent 拒绝了本次操作。')
  } finally {
    taskRunning.value = false
  }
}

function pullImage(): void {
  const image = imageReference.value.trim()
  if (!image) return
  void submitTask({ action: 'image_pull', image })
  imageReference.value = ''
}

function createNetwork(): void {
  const name = networkName.value.trim()
  if (!name) return
  void submitTask({ action: 'network_create', name, driver: 'bridge' })
  networkName.value = ''
}

function createVolume(): void {
  const name = volumeName.value.trim()
  if (!name) return
  void submitTask({ action: 'volume_create', name, driver: 'local' })
  volumeName.value = ''
}

function updateNetworkMembership(action: 'network_connect' | 'network_disconnect'): void {
  const container = membershipContainers.value.find((item) => item.id === membershipContainerID.value)
  const network = membershipNetworks.value.find((item) => item.id === membershipNetworkID.value)
  if (!container?.resourceVersion || !network?.resourceVersion) return
  void submitTask({
    action,
    target: network.id,
    expectedResourceVersion: network.resourceVersion,
    containerId: container.id,
    containerResourceVersion: container.resourceVersion,
  })
}

function askImageRemoval(image: DockerInventory['images'][number]): void {
  if (!image.resourceVersion || image.inUse) return
  const target = image.id
  pendingMaintenance.value = {
    title: '确认删除镜像',
    description: image.tags.join(', ') || shortId(image.id),
    input: {
      action: 'image_remove',
      target,
      expectedResourceVersion: image.resourceVersion,
    },
  }
}

function askNetworkRemoval(network: DockerInventory['networks'][number]): void {
  if (!network.resourceVersion || Number(network.containers || 0) > 0) return
  pendingMaintenance.value = {
    title: '确认删除网络',
    description: network.name,
    input: {
      action: 'network_remove',
      target: network.id,
      expectedResourceVersion: network.resourceVersion,
    },
  }
}

function askVolumeRemoval(volume: DockerInventory['volumes'][number]): void {
  if (!volume.resourceVersion || volume.inUse) return
  pendingMaintenance.value = {
    title: '确认删除存储卷',
    description: volume.name,
    input: {
      action: 'volume_remove',
      target: volume.name,
      expectedResourceVersion: volume.resourceVersion,
    },
  }
}

function askPrune(): void {
  if (pruneConfirmation.value !== 'PRUNE') return
  pendingMaintenance.value = {
    title: '确认清理未使用资源',
    description: '容器、镜像、网络、存储卷与构建缓存',
    input: { action: 'prune', confirmation: 'PRUNE' },
  }
}

function createBackup(): void {
  void submitTask({ action: 'backup_create' })
}

function updateMirror(): void {
  pendingMaintenance.value = {
    title: '确认切换 Docker 镜像源',
    description:
      mirrorPreset.value === 'cn'
        ? '启用与 kejilion.sh 一致的中国大陆镜像源列表'
        : '删除 registry-mirrors，恢复 Docker 官方默认线路',
    input: { action: 'daemon_mirror', preset: mirrorPreset.value },
  }
}

function updateIPv6(): void {
  pendingMaintenance.value = {
    title: `确认${ipv6Enabled.value ? '开启' : '关闭'} Docker IPv6`,
    description: ipv6Enabled.value
      ? `固定 IPv6 网段：${ipv6CIDR.value.trim()}`
      : '移除 fixed-cidr-v6 并关闭 Docker IPv6',
    input: {
      action: 'daemon_ipv6',
      enabled: ipv6Enabled.value,
      ipv6Cidr: ipv6Enabled.value ? ipv6CIDR.value.trim() : undefined,
    },
  }
}

onMounted(() => {
  void load()
  void restoreBackgroundJob()
})
onBeforeUnmount(() => {
  controller?.abort()
  logController?.abort()
  stopJobPolling()
})
</script>

<template>
  <div class="page">
    <PageHeader title="Docker 管理" description="所有资源均可查看；只有通过归属验证的容器允许生命周期操作。">
      <template #actions>
        <span v-if="data" class="observed-at">{{ data.version ? `Docker ${data.version}` : 'Docker Engine' }}</span>
        <button class="button button--primary" type="button" :disabled="panel.isReadOnly.value" @click="toolsOpen = true">
          <Wrench :size="16" /> Docker 工具箱
        </button>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="16" :class="{ spin: refreshing }" /> 刷新
        </button>
      </template>
    </PageHeader>

    <div
      v-if="activeJob && (activeJob.status === 'queued' || activeJob.status === 'running')"
      class="inline-alert inline-alert--info docker-job"
      role="status"
    >
      <LoaderCircle class="spin" :size="17" />
      <span>
        <strong>{{ activeJob.message || 'Docker 后台任务正在执行' }}</strong>
        <small>{{ activeJob.target || activeJob.action }} · {{ activeJob.progress }}%</small>
      </span>
      <progress :value="activeJob.progress" max="100">{{ activeJob.progress }}%</progress>
    </div>

    <LoadingState v-if="loading" :rows="4" cards />
    <ErrorState v-else-if="error && !data" :message="error" @retry="load()" />

    <template v-else-if="data">
      <div v-if="!data.available" class="inline-alert inline-alert--warning" role="status">
        Docker Engine 当前不可用，以下可能是最后一次成功观测的数据。
      </div>

      <section class="summary-strip" aria-label="Docker 摘要">
        <div>
          <span class="summary-strip__icon"><Container :size="20" /></span>
          <span><strong>{{ data.containers.length }}</strong><small>容器总数</small></span>
        </div>
        <div>
          <span class="summary-strip__icon summary-strip__icon--success"><Play :size="20" /></span>
          <span><strong>{{ runningCount }}</strong><small>运行中</small></span>
        </div>
        <div>
          <span class="summary-strip__icon summary-strip__icon--blue"><ShieldCheck :size="20" /></span>
          <span><strong>{{ managedCount }}</strong><small>可安全管理</small></span>
        </div>
        <div>
          <span class="summary-strip__icon summary-strip__icon--violet"><Boxes :size="20" /></span>
          <span><strong>{{ data.images.length }}</strong><small>本地镜像</small></span>
        </div>
      </section>

      <section class="resource-browser">
        <header class="resource-browser__toolbar">
          <div class="tab-bar" role="tablist" aria-label="Docker 资源类型">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.id"
              :class="{ 'is-active': activeTab === tab.id }"
              @click="activeTab = tab.id"
            >
              <component :is="tab.icon" :size="16" />
              {{ tab.label }}
              <span>{{ tab.count }}</span>
            </button>
          </div>
          <div class="search-field search-field--small">
            <Search :size="16" />
            <input v-model="search" type="search" placeholder="搜索当前资源" aria-label="搜索 Docker 资源" />
          </div>
        </header>

        <div v-if="error" class="inline-alert inline-alert--warning">刷新失败，正在显示上次观测结果。</div>

        <template v-if="activeTab === 'containers'">
          <EmptyState
            v-if="!filteredContainers.length"
            title="没有符合条件的容器"
            description="Docker Engine 未返回容器，或搜索条件没有匹配项。"
          />
          <div v-else class="table-scroll">
            <table class="data-table">
              <thead>
                <tr>
                  <th>容器</th>
                  <th>状态</th>
                  <th>端口</th>
                  <th>归属</th>
                  <th>项目</th>
                  <th><span class="sr-only">操作</span></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="container in filteredContainers" :key="container.id">
                  <td>
                    <div class="resource-name">
                      <span class="resource-name__icon resource-name__icon--docker"><Container :size="18" /></span>
                      <span>
                        <strong>{{ container.name }}</strong>
                        <small :title="container.image">{{ container.image }}</small>
                      </span>
                    </div>
                  </td>
                  <td>
                    <div class="table-stack">
                      <StatusBadge :status="container.state" />
                      <small v-if="container.statusText">{{ container.statusText }}</small>
                    </div>
                  </td>
                  <td><span class="table-code" :title="formatPorts(container)">{{ formatPorts(container) }}</span></td>
                  <td>
                    <div class="table-stack">
                      <StatusBadge :status="container.access" subtle />
                      <small>{{ container.access === 'managed' ? '已验证' : '禁止变更' }}</small>
                    </div>
                  </td>
                  <td>{{ container.project || '—' }}</td>
                  <td class="table-actions">
                    <button class="icon-button icon-button--small" type="button" aria-label="查看日志" @click="showLogs(container)">
                      <FileText :size="16" />
                    </button>
                    <button
                      v-if="container.state !== 'running' && permits(container, 'start')"
                      class="icon-button icon-button--small icon-button--success"
                      type="button"
                      aria-label="启动容器"
                      :disabled="panel.isReadOnly.value"
                      @click="askAction(container, 'start')"
                    >
                      <Play :size="16" />
                    </button>
                    <button
                      v-if="container.state === 'running' && permits(container, 'restart')"
                      class="icon-button icon-button--small"
                      type="button"
                      aria-label="重启容器"
                      :disabled="panel.isReadOnly.value"
                      @click="askAction(container, 'restart')"
                    >
                      <RotateCw :size="16" />
                    </button>
                    <button
                      v-if="container.state === 'running' && permits(container, 'stop')"
                      class="icon-button icon-button--small icon-button--danger"
                      type="button"
                      aria-label="停止容器"
                      :disabled="panel.isReadOnly.value"
                      @click="askAction(container, 'stop')"
                    >
                      <CircleStop :size="16" />
                    </button>
                    <button
                      v-if="container.state !== 'running' && permits(container, 'remove')"
                      class="icon-button icon-button--small icon-button--danger"
                      type="button"
                      aria-label="删除容器"
                      :disabled="panel.isReadOnly.value"
                      @click="askAction(container, 'remove')"
                    >
                      <Trash2 :size="16" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <template v-else-if="activeTab === 'images'">
          <LoadingState v-if="data.loading?.images" :rows="3" />
          <div v-else-if="data.errors?.images" class="inline-alert inline-alert--warning">
            镜像列表加载失败：{{ data.errors.images }}
          </div>
          <EmptyState v-else-if="!filteredImages.length" title="没有符合条件的镜像" />
          <div v-else class="table-scroll">
            <table class="data-table">
              <thead><tr><th>镜像标签</th><th>镜像 ID</th><th>大小</th><th>创建时间</th><th>使用状态</th><th><span class="sr-only">操作</span></th></tr></thead>
              <tbody>
                <tr v-for="image in filteredImages" :key="image.id">
                  <td><strong>{{ image.tags.join(', ') || '&lt;none&gt;' }}</strong></td>
                  <td><code>{{ shortId(image.id) }}</code></td>
                  <td>{{ formatBytes(image.sizeBytes) }}</td>
                  <td>{{ relativeTime(image.createdAt) }}</td>
                  <td><StatusBadge :status="image.inUse ? 'managed' : 'unknown'" :label="image.inUse ? '使用中' : '未使用'" subtle /></td>
                  <td class="table-actions">
                    <button
                      v-if="image.resourceVersion && !image.inUse"
                      class="icon-button icon-button--small icon-button--danger"
                      type="button"
                      aria-label="删除镜像"
                      :disabled="panel.isReadOnly.value"
                      @click="askImageRemoval(image)"
                    >
                      <Trash2 :size="16" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <template v-else-if="activeTab === 'networks'">
          <LoadingState v-if="data.loading?.networks" :rows="3" />
          <div v-else-if="data.errors?.networks" class="inline-alert inline-alert--warning">
            网络列表加载失败：{{ data.errors.networks }}
          </div>
          <EmptyState v-else-if="!filteredNetworks.length" title="没有符合条件的网络" />
          <div v-else class="table-scroll">
            <table class="data-table">
              <thead><tr><th>网络名称</th><th>驱动</th><th>作用域</th><th>已连接容器</th><th>网络 ID</th><th><span class="sr-only">操作</span></th></tr></thead>
              <tbody>
                <tr v-for="network in filteredNetworks" :key="network.id">
                  <td><strong>{{ network.name }}</strong></td>
                  <td><code>{{ network.driver }}</code></td>
                  <td>{{ network.scope || 'local' }}</td>
                  <td>{{ network.containers ?? '—' }}</td>
                  <td><code>{{ shortId(network.id) }}</code></td>
                  <td class="table-actions">
                    <button
                      v-if="network.resourceVersion && !network.containers && !['bridge', 'host', 'none', 'kejilion-panel-network'].includes(network.name)"
                      class="icon-button icon-button--small icon-button--danger"
                      type="button"
                      aria-label="删除网络"
                      :disabled="panel.isReadOnly.value"
                      @click="askNetworkRemoval(network)"
                    >
                      <Trash2 :size="16" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <template v-else>
          <LoadingState v-if="data.loading?.volumes" :rows="3" />
          <div v-else-if="data.errors?.volumes" class="inline-alert inline-alert--warning">
            存储卷加载失败：{{ data.errors.volumes }}
          </div>
          <EmptyState v-else-if="!filteredVolumes.length" title="没有符合条件的存储卷" />
          <div v-else class="table-scroll">
            <table class="data-table">
              <thead><tr><th>卷名称</th><th>驱动</th><th>挂载点</th><th>使用状态</th><th><span class="sr-only">操作</span></th></tr></thead>
              <tbody>
                <tr v-for="volume in filteredVolumes" :key="volume.name">
                  <td><strong>{{ volume.name }}</strong></td>
                  <td><code>{{ volume.driver }}</code></td>
                  <td><span class="table-code" :title="volume.mountpoint">{{ volume.mountpoint || '—' }}</span></td>
                  <td><StatusBadge :status="volume.inUse ? 'managed' : 'unknown'" :label="volume.inUse ? '使用中' : '未使用'" subtle /></td>
                  <td class="table-actions">
                    <button
                      v-if="volume.resourceVersion && !volume.inUse && !volume.name.includes('kpanel')"
                      class="icon-button icon-button--small icon-button--danger"
                      type="button"
                      aria-label="删除存储卷"
                      :disabled="panel.isReadOnly.value"
                      @click="askVolumeRemoval(volume)"
                    >
                      <Trash2 :size="16" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <footer class="table-card__footer">
          最后观测于 {{ formatDateTime(data.observedAt) }} · 固定动作通过后台任务执行；不开放 Exec 或任意 Compose/命令编辑
        </footer>
      </section>
    </template>

    <ModalDialog
      :open="toolsOpen"
      title="Docker 工具箱"
      description="对齐 kejilion.sh 的常用镜像、网络、存储卷与安全清理流程；长任务在后台执行。"
      size="large"
      @close="toolsOpen = false"
    >
      <div class="docker-tools">
        <section class="docker-tool">
          <span><Download :size="20" /></span>
          <div>
            <strong>拉取或更新镜像</strong>
            <small>输入完整镜像名，例如 nginx:alpine</small>
            <div class="docker-tool__form">
              <input v-model="imageReference" class="text-input" type="text" maxlength="255" placeholder="镜像:标签" />
              <button class="button button--primary button--small" type="button" :disabled="!imageReference.trim() || taskRunning" @click="pullImage">
                拉取
              </button>
            </div>
          </div>
        </section>
        <section class="docker-tool">
          <span><Network :size="20" /></span>
          <div>
            <strong>创建 bridge 网络</strong>
            <small>仅创建带 KPanel 归属标签的本地网络</small>
            <div class="docker-tool__form">
              <input v-model="networkName" class="text-input" type="text" maxlength="128" placeholder="网络名称" />
              <button class="button button--secondary button--small" type="button" :disabled="!networkName.trim() || taskRunning" @click="createNetwork">
                <Plus :size="15" /> 创建
              </button>
            </div>
          </div>
        </section>
        <section class="docker-tool">
          <span><Waypoints :size="20" /></span>
          <div>
            <strong>连接 / 断开容器网络</strong>
            <small>仅允许已验证的 Kejilion 容器和非系统网络；执行前再次核验两端资源版本。</small>
            <div class="docker-tool__membership">
              <select v-model="membershipContainerID" class="text-input" aria-label="选择容器">
                <option value="">选择容器</option>
                <option v-for="container in membershipContainers" :key="container.id" :value="container.id">
                  {{ container.name }}
                </option>
              </select>
              <select v-model="membershipNetworkID" class="text-input" aria-label="选择网络">
                <option value="">选择网络</option>
                <option v-for="network in membershipNetworks" :key="network.id" :value="network.id">
                  {{ network.name }}
                </option>
              </select>
              <div class="docker-tool__form">
                <button
                  class="button button--secondary button--small"
                  type="button"
                  :disabled="!membershipContainerID || !membershipNetworkID || taskRunning"
                  @click="updateNetworkMembership('network_connect')"
                >
                  连接
                </button>
                <button
                  class="button button--secondary button--small"
                  type="button"
                  :disabled="!membershipContainerID || !membershipNetworkID || taskRunning"
                  @click="updateNetworkMembership('network_disconnect')"
                >
                  断开
                </button>
              </div>
            </div>
          </div>
        </section>
        <section class="docker-tool">
          <span><HardDrive :size="20" /></span>
          <div>
            <strong>创建 local 存储卷</strong>
            <small>删除时仍会回读版本并检查使用状态</small>
            <div class="docker-tool__form">
              <input v-model="volumeName" class="text-input" type="text" maxlength="128" placeholder="存储卷名称" />
              <button class="button button--secondary button--small" type="button" :disabled="!volumeName.trim() || taskRunning" @click="createVolume">
                <Plus :size="15" /> 创建
              </button>
            </div>
          </div>
        </section>
        <section class="docker-tool">
          <span><Boxes :size="20" /></span>
          <div>
            <strong>备份 Docker 应用数据</strong>
            <small>归档 /home/docker 下的应用目录，自动排除 KPanel 自身；文件以 0600 权限保存到 Agent 备份目录。</small>
            <div class="docker-tool__form">
              <button class="button button--secondary button--small" type="button" :disabled="taskRunning" @click="createBackup">
                创建后台备份
              </button>
            </div>
          </div>
        </section>
        <section class="docker-tool">
          <span><RefreshCw :size="20" /></span>
          <div>
            <strong>Docker 镜像源</strong>
            <small>保留 daemon.json 的其他配置；失败时自动回滚并再次启动 Docker。</small>
            <div class="docker-tool__form">
              <select v-model="mirrorPreset" class="text-input" aria-label="Docker 镜像源">
                <option value="cn">kejilion.sh 中国大陆镜像组</option>
                <option value="official">Docker 官方默认线路</option>
              </select>
              <button class="button button--secondary button--small" type="button" :disabled="taskRunning" @click="updateMirror">
                应用并重启
              </button>
            </div>
          </div>
        </section>
        <section class="docker-tool">
          <span><Network :size="20" /></span>
          <div>
            <strong>Docker IPv6</strong>
            <small>使用真实可路由或 ULA 的 /64 网段；不再采用脚本中的文档示例地址 2001:db8::/32。</small>
            <label class="docker-tool__check">
              <input v-model="ipv6Enabled" type="checkbox" />
              开启 Docker IPv6
            </label>
            <div class="docker-tool__form">
              <input
                v-model="ipv6CIDR"
                class="text-input"
                type="text"
                maxlength="64"
                placeholder="fd42:6b50:616e:656c::/64"
                :disabled="!ipv6Enabled"
              />
              <button
                class="button button--secondary button--small"
                type="button"
                :disabled="taskRunning || (ipv6Enabled && !ipv6CIDR.trim())"
                @click="updateIPv6"
              >
                应用并重启
              </button>
            </div>
          </div>
        </section>
        <section class="docker-tool docker-tool--danger">
          <span><BrushCleaning :size="20" /></span>
          <div>
            <strong>清理未使用资源</strong>
            <small>对应 docker system prune：清理停止容器、未使用镜像/网络/卷和构建缓存，不触碰运行中的 KPanel。</small>
            <div class="docker-tool__form">
              <input v-model="pruneConfirmation" class="text-input" type="text" placeholder="输入 PRUNE 确认" />
              <button class="button button--danger button--small" type="button" :disabled="pruneConfirmation !== 'PRUNE' || taskRunning" @click="askPrune">
                清理
              </button>
            </div>
          </div>
        </section>
      </div>
      <div class="inline-alert inline-alert--info">
        备份完成后会显示宿主机文件路径，可用于下载或迁移。系统更新会同时维护发行版提供的 Docker 软件包。
        覆盖式还原与远程 SSH 迁移需要目标路径、冲突策略和远端凭证，当前仅开放经过校验的备份；卸载 Docker 会直接终止 KPanel，
        因此必须回到 kejilion.sh 或 SSH 执行。
      </div>
      <template #footer>
        <span class="modal-footer-note">任务状态会保存在 Agent 数据目录</span>
        <button class="button button--secondary" type="button" @click="toolsOpen = false">关闭</button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(pendingMaintenance)"
      :title="pendingMaintenance?.title || '确认 Docker 操作'"
      :description="pendingMaintenance?.description"
      size="small"
      @close="pendingMaintenance = undefined"
    >
      <div class="confirm-content">
        <span class="confirm-content__icon is-danger"><Trash2 :size="23" /></span>
        <p>Agent 会再次核验资源版本和 KPanel 保护规则。操作完成后可能无法恢复。</p>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" @click="pendingMaintenance = undefined">取消</button>
        <button
          class="button button--danger"
          type="button"
          :disabled="taskRunning || !pendingMaintenance"
          @click="pendingMaintenance && submitTask(pendingMaintenance.input)"
        >
          <LoaderCircle v-if="taskRunning" class="spin" :size="16" />
          {{ taskRunning ? '正在提交…' : '确认执行' }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(pendingAction)"
      :title="pendingAction === 'stop' ? '确认停止容器' : pendingAction === 'restart' ? '确认重启容器' : pendingAction === 'remove' ? '确认删除容器' : '确认启动容器'"
      :description="selectedContainer ? `${selectedContainer.name} · ${selectedContainer.image}` : ''"
      size="small"
      @close="pendingAction = undefined; selectedContainer = undefined"
    >
      <div class="confirm-content">
        <span class="confirm-content__icon" :class="{ 'is-danger': pendingAction === 'stop' || pendingAction === 'remove' }">
          <Trash2 v-if="pendingAction === 'remove'" :size="23" />
          <CircleStop v-else-if="pendingAction === 'stop'" :size="23" />
          <RotateCw v-else-if="pendingAction === 'restart'" :size="23" />
          <Play v-else :size="23" />
        </span>
        <p>
          Agent 会再次验证容器身份与允许动作。{{
            pendingAction === 'stop'
              ? '停止可能造成对应服务短暂不可用。'
              : pendingAction === 'remove'
                ? '只删除已停止容器，不删除镜像或存储卷。'
                : ''
          }}
        </p>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" @click="pendingAction = undefined; selectedContainer = undefined">
          取消
        </button>
        <button
          class="button"
          :class="pendingAction === 'stop' || pendingAction === 'remove' ? 'button--danger' : 'button--primary'"
          type="button"
          :disabled="actionRunning"
          @click="runAction"
        >
          <LoaderCircle v-if="actionRunning" class="spin" :size="16" />
          {{ actionRunning ? '正在提交…' : '确认执行' }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="logsOpen"
      :title="`${selectedContainer?.name || '容器'} 日志`"
      description="只显示最近 200 行；应用日志可能包含敏感信息，请谨慎查看和复制。"
      size="large"
      @close="closeLogs"
    >
      <LoadingState v-if="logsLoading" :rows="3" />
      <ErrorState v-else-if="logError" :message="logError" retry-label="重新读取" @retry="selectedContainer && showLogs(selectedContainer)" />
      <EmptyState v-else-if="!logLines.length" title="容器没有返回日志" />
      <pre v-else class="log-view" tabindex="0">{{ logLines.join('\n') }}</pre>
      <template #footer>
        <span class="modal-footer-note">日志按纯文本安全呈现</span>
        <button class="button button--secondary" type="button" @click="closeLogs">关闭</button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.docker-job {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) minmax(120px, 220px);
  align-items: center;
  gap: 12px;
}

.docker-job span {
  display: grid;
  gap: 2px;
}

.docker-job small {
  color: var(--muted);
}

.docker-job progress {
  width: 100%;
}

.docker-tools {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 16px;
}

.docker-tool {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 16px;
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.docker-tool > span {
  display: grid;
  width: 38px;
  height: 38px;
  color: var(--brand);
  background: var(--brand-soft);
  border-radius: 11px;
  place-items: center;
}

.docker-tool > div {
  display: grid;
  gap: 6px;
}

.docker-tool small {
  min-height: 32px;
  color: var(--muted);
  line-height: 1.45;
}

.docker-tool--danger > span {
  color: var(--danger);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
}

.docker-tool__form {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.docker-tool__membership {
  display: grid;
  gap: 8px;
  margin-top: 4px;
}

.docker-tool__check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--text);
  font-size: 0.9rem;
}

.text-input {
  width: 100%;
  min-width: 0;
  height: 38px;
  padding: 0 11px;
  color: var(--text);
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
}

.text-input:focus {
  border-color: var(--brand);
  outline: 3px solid color-mix(in srgb, var(--brand) 12%, transparent);
}

@media (max-width: 760px) {
  .docker-job,
  .docker-tools {
    grid-template-columns: 1fr;
  }
}
</style>
