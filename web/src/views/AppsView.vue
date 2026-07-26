<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ArrowUpRight,
  CheckCircle2,
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
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { useToast } from '@/stores/toast'
import type { AppMarketInventory, AppMarketItem, Site } from '@/types/api'

type SourceFilter = 'all' | 'builtin' | 'thirdparty'
type StatusFilter = 'all' | 'installed' | 'running' | 'adapted'
type ConfirmAction = 'update' | 'uninstall' | undefined

const inventory = ref<AppMarketInventory>()
const sites = ref<Site[]>([])
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const search = ref('')
const category = ref('all')
const source = ref<SourceFilter>('all')
const status = ref<StatusFilter>('all')
const selectedID = ref('')
const installOpen = ref(false)
const installPort = ref(0)
const installAccess = ref<'direct' | 'domain_only'>('direct')
const domain = ref('')
const domainError = ref('')
const operation = ref('')
const confirmAction = ref<ConfirmAction>()
const checkedUpdates = ref<Record<string, 'available' | 'current'>>({})
const toast = useToast()
let controller: AbortController | undefined

const selected = computed(() => inventory.value?.items.find((item) => item.id === selectedID.value))
const selectedPort = computed(() => selected.value?.runtime.ports.find((port) => port.type === 'tcp' && port.publicPort))
const selectedDomains = computed(() => {
  const port = selectedPort.value?.publicPort
  if (!port) return []
  const targets = new Set([`http://127.0.0.1:${port}`, `http://localhost:${port}`])
  return sites.value.filter((site) => site.type === 'proxy' && targets.has(site.upstream || ''))
})

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
    const result = await api.apps.checkUpdate(item.id, item.runtime.resourceVersion)
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
}

function openInstall(item: AppMarketItem): void {
  selectedID.value = item.id
  installPort.value = item.defaultPort || 0
  installAccess.value = 'direct'
  installOpen.value = true
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''
  try {
    const [appsResult, sitesResult] = await Promise.allSettled([
      api.apps.inventory(controller.signal),
      api.sites.list(undefined, controller.signal),
    ])
    if (appsResult.status === 'rejected') throw appsResult.reason
    inventory.value = appsResult.value
    sites.value = sitesResult.status === 'fulfilled' ? sitesResult.value.items : []
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    error.value = reason instanceof ApiError ? reason.message : '无法读取应用市场，请稍后重试。'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function install(): Promise<void> {
  const item = selected.value
  if (!item || !capability(item, 'install')) return
  operation.value = 'install'
  try {
    await api.apps.install(item.id, {
      hostPort: installPort.value || undefined,
      accessMode: installAccess.value,
    })
    installOpen.value = false
    toast.success('应用安装完成', `${item.name_zh} 已按声明式安全模板启动。`)
    await load(true)
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
    await load(true)
  } catch (reason) {
    toast.danger('操作失败', reason instanceof ApiError ? reason.message : '应用状态未能变更。')
  } finally {
    operation.value = ''
  }
}

async function confirmMutation(): Promise<void> {
  const item = selected.value
  const action = confirmAction.value
  if (!item?.runtime.resourceVersion || !action || !capability(item, action)) return
  operation.value = action
  try {
    await api.apps.action(item.id, action, { resourceVersion: item.runtime.resourceVersion })
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

async function toggleAccess(): Promise<void> {
  const item = selected.value
  if (!item?.runtime.resourceVersion || !capability(item, 'direct_access')) return
  const next = item.runtime.accessMode === 'domain_only' ? 'direct' : 'domain_only'
  operation.value = 'direct_access'
  try {
    await api.apps.action(item.id, 'direct_access', {
      resourceVersion: item.runtime.resourceVersion,
      accessMode: next,
    })
    toast.success(next === 'domain_only' ? '已阻止 IP + 端口访问' : '已放行 IP + 端口访问')
    await load(true)
  } catch (reason) {
    toast.danger('访问策略变更失败', reason instanceof ApiError ? reason.message : '容器端口绑定未能安全切换。')
  } finally {
    operation.value = ''
  }
}

async function addDomain(): Promise<void> {
  const item = selected.value
  const port = selectedPort.value?.publicPort
  if (!item || !port || !domain.value.trim()) return
  domainError.value = ''
  operation.value = 'add_domain'
  try {
    const hostname = domain.value.trim().toLowerCase()
    await api.sites.create({
      primaryDomain: hostname,
      aliases: [],
      type: 'proxy',
      upstream: `http://127.0.0.1:${port}`,
      enabled: true,
    })
    if (capability(item, 'direct_access') && item.runtime.accessMode === 'direct' && item.runtime.resourceVersion) {
      await api.apps.action(item.id, 'direct_access', {
        resourceVersion: item.runtime.resourceVersion,
        accessMode: 'domain_only',
      })
    }
    domain.value = ''
    toast.success('域名已绑定', `${hostname} 已反向代理到 ${item.name_zh}。`)
    await load(true)
  } catch (reason) {
    domainError.value = reason instanceof ApiError ? reason.message : '域名绑定失败，请检查网站与 Nginx 状态。'
  } finally {
    operation.value = ''
  }
}

async function removeDomain(site: Site): Promise<void> {
  operation.value = `remove_domain:${site.id}`
  domainError.value = ''
  try {
    await api.sites.remove(site.id, site.resourceVersion)
    toast.success('域名已解绑', `${site.primaryDomain} 的反向代理已安全移除。`)
    await load(true)
  } catch (reason) {
    domainError.value = reason instanceof ApiError ? reason.message : '域名解绑失败，原配置保持不变。'
  } finally {
    operation.value = ''
  }
}

function directURL(item: AppMarketItem): string {
  const port = item.runtime.ports.find((entry) => entry.type === 'tcp' && entry.publicPort)?.publicPort
  if (!port || item.runtime.accessMode === 'domain_only') return ''
  return `http://${window.location.hostname}:${port}`
}

onMounted(() => void load())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div class="page app-market">
    <PageHeader
      title="应用市场"
      description="完整呈现 kejilion.sh 应用目录；运行状态来自宿主机 Docker，写操作只通过已审计的安全适配器执行。"
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
        <div><strong>{{ inventory.items.filter((item) => capability(item, 'install') || capability(item, 'update')).length }}</strong><span>安全适配</span></div>
      </div>
    </section>

    <div v-if="inventory?.catalogWarning" class="inline-alert inline-alert--warning">
      {{ inventory.catalogWarning }}
    </div>

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
            { key: 'all', label: '全部状态' },
            { key: 'installed', label: '已安装' },
            { key: 'running', label: '运行中' },
            { key: 'adapted', label: '可安全操作' },
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
          <span class="app-card__icon"><img :src="item.icon" :alt="`${item.name_zh} 图标`" loading="lazy" /></span>
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
                <ShieldCheck :size="12" /> 安全适配
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
            {{ item.runtime.installed ? '管理' : '查看' }}
          </button>
        </footer>
      </article>
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
      size="large"
      @close="selectedID = ''"
    >
      <template v-if="selected">
        <div class="app-detail-head">
          <span class="app-detail-head__icon"><img :src="selected.icon" alt="" /></span>
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
              v-if="capability(selected, 'start')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation)"
              @click="lifecycle('start')"
            >
              <Play :size="15" /> 启动
            </button>
            <button
              v-if="capability(selected, 'stop')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation)"
              @click="lifecycle('stop')"
            >
              <Square :size="14" /> 停止
            </button>
            <button
              v-if="capability(selected, 'restart')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation)"
              @click="lifecycle('restart')"
            >
              <RotateCw :size="15" /> 重启
            </button>
            <button
              v-if="capability(selected, 'check_update')"
              class="button button--secondary"
              type="button"
              :disabled="Boolean(operation)"
              @click="checkUpdate"
            >
              <LoaderCircle v-if="operation === 'check_update'" class="spin" :size="15" />
              <RefreshCw v-else :size="15" /> 检查更新
            </button>
            <a
              v-if="directURL(selected)"
              class="button button--primary"
              :href="directURL(selected)"
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
            <p v-if="capability(selected, 'install')">此应用已有固定镜像、端口和回滚策略，可以由 KPanel 安全安装。</p>
            <p v-else>{{ selected.capabilities.install?.reason || '等待安全适配。' }}</p>
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
          <section class="app-detail-section">
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
                  :disabled="Boolean(operation)"
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
              <button class="button button--secondary" type="submit" :disabled="operation === 'add_domain' || !domain">
                <LoaderCircle v-if="operation === 'add_domain'" class="spin" :size="15" />
                <Globe2 v-else :size="15" /> 绑定
              </button>
            </form>
            <p v-if="domainError" class="field-error">{{ domainError }}</p>
            <p v-if="!capability(selected, 'add_domain')" class="muted-note">
              {{ selected.capabilities.add_domain?.reason }}
            </p>
          </section>

          <section class="app-detail-section">
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
                :disabled="!capability(selected, 'direct_access') || Boolean(operation)"
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
            :disabled="!capability(selected, 'update') || Boolean(operation)"
            :title="selected.capabilities.update?.reason"
            @click="confirmAction = 'update'"
          >
            <RefreshCw :size="15" /> 更新
          </button>
          <button
            class="button button--danger"
            type="button"
            :disabled="!capability(selected, 'uninstall') || Boolean(operation)"
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
      description="使用固定镜像与端口模板；安装后 kejilion.sh 的应用编号和端口文件也会同步生成。"
      size="small"
      @close="installOpen = false"
    >
      <form id="app-install-form" class="form-stack" @submit.prevent="install">
        <label class="field">
          <span>访问端口</span>
          <input v-model.number="installPort" type="number" min="1024" max="65535" required />
          <small>默认沿用 kejilion.sh 的端口；发生冲突时请换用其他端口。</small>
        </label>
        <fieldset class="access-options">
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
          不执行远程 Shell，不加载第三方 Compose；容器创建失败不会写入脚本安装标记。
        </div>
      </form>
      <template #footer>
        <button class="button button--secondary" type="button" @click="installOpen = false">取消</button>
        <button class="button button--primary" type="submit" form="app-install-form" :disabled="Boolean(operation)">
          <LoaderCircle v-if="operation === 'install'" class="spin" :size="16" />
          <Download v-else :size="16" /> {{ operation === 'install' ? '正在安装…' : '确认安装' }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="Boolean(confirmAction)"
      :title="confirmAction === 'uninstall' ? '确认卸载应用？' : '确认更新应用？'"
      :description="
        confirmAction === 'uninstall'
          ? '容器会停止并删除；共享镜像缓存不会删除。'
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
  gap: 15px;
  align-items: center;
  margin-bottom: 18px;
}

.app-detail-head__icon {
  width: 66px;
  height: 66px;
  border-radius: 18px;
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
  overflow: hidden;
  margin-bottom: 18px;
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
  padding: 16px;
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
  gap: 8px;
  justify-content: flex-end;
  padding: 11px;
  border-top: 1px solid var(--border);
  background: var(--surface);
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
  grid-template-columns: 1.2fr 0.8fr;
  gap: 14px;
}

.app-detail-section {
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: var(--surface);
}

.app-detail-section > header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
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

.domain-list {
  display: grid;
  gap: 6px;
  margin-bottom: 10px;
}

.domain-list__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 9px 10px;
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
  padding: 12px;
  border-radius: 12px;
  background: var(--surface-muted);
}

.access-card > span {
  display: grid;
  width: 38px;
  height: 38px;
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
  margin-top: 14px;
  padding-top: 14px;
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
}
</style>
