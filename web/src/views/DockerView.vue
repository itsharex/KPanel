<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Box,
  Boxes,
  CircleStop,
  Container,
  FileText,
  HardDrive,
  LoaderCircle,
  Network,
  Play,
  RefreshCw,
  RotateCw,
  Search,
  ShieldCheck,
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
import type { DockerContainer, DockerInventory } from '@/types/api'

type DockerTab = 'containers' | 'images' | 'networks' | 'volumes'
type ContainerAction = 'start' | 'stop' | 'restart'

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
const panel = usePanelState()
const toast = useToast()
let controller: AbortController | undefined
let logController: AbortController | undefined

const tabs = computed(() => [
  { id: 'containers' as const, label: '容器', count: data.value?.containers.length || 0, icon: Container },
  { id: 'images' as const, label: '镜像', count: data.value?.images.length || 0, icon: Box },
  { id: 'networks' as const, label: '网络', count: data.value?.networks.length || 0, icon: Network },
  { id: 'volumes' as const, label: '存储卷', count: data.value?.volumes.length || 0, icon: HardDrive },
])

const runningCount = computed(() => data.value?.containers.filter((item) => item.state === 'running').length || 0)
const managedCount = computed(() => data.value?.containers.filter((item) => item.access === 'managed').length || 0)

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
      result.jobId ? `${selectedContainer.value.name} · ${shortId(result.jobId)}` : selectedContainer.value.name,
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

onMounted(() => void load())
onBeforeUnmount(() => {
  controller?.abort()
  logController?.abort()
})
</script>

<template>
  <div class="page">
    <PageHeader title="Docker 管理" description="所有资源均可查看；只有通过归属验证的容器允许生命周期操作。">
      <template #actions>
        <span v-if="data" class="observed-at">{{ data.version ? `Docker ${data.version}` : 'Docker Engine' }}</span>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="16" :class="{ spin: refreshing }" /> 刷新
        </button>
      </template>
    </PageHeader>

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
              <thead><tr><th>镜像标签</th><th>镜像 ID</th><th>大小</th><th>创建时间</th><th>使用状态</th></tr></thead>
              <tbody>
                <tr v-for="image in filteredImages" :key="image.id">
                  <td><strong>{{ image.tags.join(', ') || '&lt;none&gt;' }}</strong></td>
                  <td><code>{{ shortId(image.id) }}</code></td>
                  <td>{{ formatBytes(image.sizeBytes) }}</td>
                  <td>{{ relativeTime(image.createdAt) }}</td>
                  <td><StatusBadge :status="image.inUse ? 'managed' : 'unknown'" :label="image.inUse ? '使用中' : '未使用'" subtle /></td>
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
              <thead><tr><th>网络名称</th><th>驱动</th><th>作用域</th><th>已连接容器</th><th>网络 ID</th></tr></thead>
              <tbody>
                <tr v-for="network in filteredNetworks" :key="network.id">
                  <td><strong>{{ network.name }}</strong></td>
                  <td><code>{{ network.driver }}</code></td>
                  <td>{{ network.scope || 'local' }}</td>
                  <td>{{ network.containers ?? '—' }}</td>
                  <td><code>{{ shortId(network.id) }}</code></td>
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
              <thead><tr><th>卷名称</th><th>驱动</th><th>挂载点</th><th>使用状态</th></tr></thead>
              <tbody>
                <tr v-for="volume in filteredVolumes" :key="volume.name">
                  <td><strong>{{ volume.name }}</strong></td>
                  <td><code>{{ volume.driver }}</code></td>
                  <td><span class="table-code" :title="volume.mountpoint">{{ volume.mountpoint || '—' }}</span></td>
                  <td><StatusBadge :status="volume.inUse ? 'managed' : 'unknown'" :label="volume.inUse ? '使用中' : '未使用'" subtle /></td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <footer class="table-card__footer">
          最后观测于 {{ formatDateTime(data.observedAt) }} · 不提供 Exec、Prune、Pull 或任意 Compose 编辑
        </footer>
      </section>
    </template>

    <ModalDialog
      :open="Boolean(pendingAction)"
      :title="pendingAction === 'stop' ? '确认停止容器' : pendingAction === 'restart' ? '确认重启容器' : '确认启动容器'"
      :description="selectedContainer ? `${selectedContainer.name} · ${selectedContainer.image}` : ''"
      size="small"
      @close="pendingAction = undefined; selectedContainer = undefined"
    >
      <div class="confirm-content">
        <span class="confirm-content__icon" :class="{ 'is-danger': pendingAction === 'stop' }">
          <CircleStop v-if="pendingAction === 'stop'" :size="23" />
          <RotateCw v-else-if="pendingAction === 'restart'" :size="23" />
          <Play v-else :size="23" />
        </span>
        <p>
          Agent 会再次验证容器身份与允许动作。{{ pendingAction === 'stop' ? '停止可能造成对应服务短暂不可用。' : '' }}
        </p>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" @click="pendingAction = undefined; selectedContainer = undefined">
          取消
        </button>
        <button
          class="button"
          :class="pendingAction === 'stop' ? 'button--danger' : 'button--primary'"
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
