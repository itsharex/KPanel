<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import {
  ArrowUpRight,
  Check,
  Copy,
  Gauge,
  KeyRound,
  LoaderCircle,
  MemoryStick,
  Network,
  Pencil,
  Plus,
  RefreshCw,
  Server,
  ShieldCheck,
  Trash2,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import CountryFlagIcon from '@/components/overview/CountryFlagIcon.vue'
import OperatingSystemIcon from '@/components/overview/OperatingSystemIcon.vue'
import { ApiError, api } from '@/lib/api'
import {
  clampPercent,
  formatDateTime,
  formatDuration,
  formatPercent,
  formatRate,
  relativeTime,
} from '@/lib/format'
import { useToast } from '@/stores/toast'
import type {
  ClusterController,
  ClusterHost,
  ClusterHostList,
  ClusterPairingCode,
} from '@/types/api'

const toast = useToast()
const inventory = ref<ClusterHostList>()
const loading = ref(true)
const refreshing = ref(false)
const refreshWarning = ref('')
const loadError = ref('')
const search = ref('')
const addOpen = ref(false)
const accessOpen = ref(false)
const manageOpen = ref(false)
const adding = ref(false)
const saving = ref(false)
const deleting = ref(false)
const generatingCode = ref(false)
const controllersLoading = ref(false)
const pairingCode = ref<ClusterPairingCode>()
const controllers = ref<ClusterController[]>([])
const selected = ref<ClusterHost>()
const addOriginInput = ref<HTMLInputElement>()
const addForm = reactive({ name: '', origin: '', pairingCode: '' })
const editName = ref('')
let loadInFlight = false
let loadController: AbortController | undefined
let pollTimer: number | undefined
const delayedRefreshes = new Set<number>()

const filteredHosts = computed(() => {
  const term = search.value.trim().toLocaleLowerCase()
  if (!term) return inventory.value?.items || []
  return (inventory.value?.items || []).filter((host) => {
    const telemetry = host.lastSnapshot?.telemetry
    return [
      host.name,
      host.origin,
      host.isLocal ? '本机 当前面板' : '',
      telemetry?.hostname,
      telemetry?.os,
      telemetry?.publicNetwork?.country,
      telemetry?.publicNetwork?.city,
      telemetry?.publicNetwork?.isp,
    ]
      .filter(Boolean)
      .some((value) => String(value).toLocaleLowerCase().includes(term))
  })
})

const onlineCount = computed(
  () => inventory.value?.items.filter((item) => item.state === 'online').length || 0,
)
const attentionCount = computed(
  () =>
    inventory.value?.items.filter((item) => !['online', 'unknown'].includes(item.state)).length ||
    0,
)

function friendlyError(reason: unknown, fallback: string): string {
  if (!(reason instanceof ApiError)) return fallback
  const messages: Record<string, string> = {
    cluster_origin_invalid: '主机 URL 必须是没有路径和参数的 HTTPS 地址。',
    cluster_origin_blocked: '该地址被网络安全策略拒绝；私网地址需由部署管理员加入 CIDR 白名单。',
    cluster_pairing_failed: '授权码无效、已过期或已被使用，请在目标 KPanel 重新生成。',
    cluster_duplicate: '该 KPanel 已经添加到主机列表。',
    cluster_host_limit: '已达到 100 台主机上限。',
    cluster_remote_tls_error: '目标 KPanel 的 HTTPS 证书校验失败。',
    cluster_remote_unreachable: '暂时无法连接目标 KPanel，请检查域名、证书和网络。',
    cluster_resource_changed: '主机信息已变化，请刷新后重试。',
  }
  return messages[reason.code] || reason.message || fallback
}

async function load(silent = false): Promise<void> {
  if (loadInFlight) return
  loadInFlight = true
  if (!silent && !inventory.value) loading.value = true
  else refreshing.value = true
  loadController = new AbortController()
  try {
    inventory.value = await api.cluster.hosts(loadController.signal)
    loadError.value = ''
    refreshWarning.value = ''
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    const message = friendlyError(reason, '无法读取集群主机，请稍后重试。')
    if (inventory.value) refreshWarning.value = `${message} 当前保留上次成功数据。`
    else loadError.value = message
  } finally {
    loading.value = false
    refreshing.value = false
    loadInFlight = false
  }
}

function openAdd(): void {
  addOpen.value = true
  void nextTick(() => addOriginInput.value?.focus())
}

function closeAdd(): void {
  if (adding.value) return
  addOpen.value = false
  addForm.name = ''
  addForm.origin = ''
  addForm.pairingCode = ''
}

async function addHost(): Promise<void> {
  if (adding.value || !addForm.origin.trim() || !addForm.pairingCode.trim()) return
  adding.value = true
  try {
    const host = await api.cluster.add({
      name: addForm.name.trim() || undefined,
      origin: addForm.origin.trim(),
      pairingCode: addForm.pairingCode.trim(),
    })
    adding.value = false
    closeAdd()
    toast.success('主机已加入集群', `${host.name} 已完成只读配对。`)
    await load(true)
  } catch (reason) {
    toast.danger('添加主机失败', friendlyError(reason, '请检查目标 KPanel 和授权码后重试。'))
  } finally {
    adding.value = false
  }
}

async function openAccess(): Promise<void> {
  accessOpen.value = true
  await loadControllers()
}

function closeAccess(): void {
  accessOpen.value = false
  pairingCode.value = undefined
}

async function loadControllers(): Promise<void> {
  controllersLoading.value = true
  try {
    controllers.value = (await api.cluster.controllers()).items
  } catch (reason) {
    toast.danger('无法读取授权列表', friendlyError(reason, '请稍后重试。'))
  } finally {
    controllersLoading.value = false
  }
}

async function createPairingCode(): Promise<void> {
  generatingCode.value = true
  try {
    pairingCode.value = await api.cluster.createPairingCode()
  } catch (reason) {
    toast.danger('授权码生成失败', friendlyError(reason, '请稍后重试。'))
  } finally {
    generatingCode.value = false
  }
}

async function copyPairingCode(): Promise<void> {
  if (!pairingCode.value) return
  try {
    await navigator.clipboard.writeText(pairingCode.value.code)
    toast.success('授权码已复制')
  } catch {
    toast.danger('复制失败', '请手动选择授权码复制。')
  }
}

async function revokeController(controller: ClusterController): Promise<void> {
  if (!window.confirm(`撤销 ${controller.name || controller.fingerprint} 的只读访问授权？`)) return
  try {
    await api.cluster.revokeController(controller.id)
    controllers.value = controllers.value.filter((item) => item.id !== controller.id)
    toast.success('控制端授权已撤销')
  } catch (reason) {
    toast.danger('撤销失败', friendlyError(reason, '请刷新后重试。'))
  }
}

function openManage(host: ClusterHost): void {
  if (host.isLocal) return
  selected.value = host
  editName.value = host.name
  manageOpen.value = true
}

function closeManage(): void {
  if (saving.value || deleting.value) return
  manageOpen.value = false
  selected.value = undefined
}

async function saveName(): Promise<void> {
  const host = selected.value
  if (!host || saving.value || !editName.value.trim()) return
  saving.value = true
  try {
    const updated = await api.cluster.rename(host.id, {
      name: editName.value.trim(),
      expectedResourceVersion: host.resourceVersion,
    })
    upsertHost(updated)
    selected.value = updated
    toast.success('主机名称已更新')
  } catch (reason) {
    toast.danger('保存失败', friendlyError(reason, '请刷新后重试。'))
  } finally {
    saving.value = false
  }
}

async function removeHost(): Promise<void> {
  const host = selected.value
  if (!host || deleting.value) return
  if (!window.confirm(`从当前 KPanel 移除 ${host.name}？目标主机业务不会受到影响。`)) return
  deleting.value = true
  try {
    const result = await api.cluster.remove(host.id, host.resourceVersion)
    if (inventory.value) {
      inventory.value.items = inventory.value.items.filter((item) => item.id !== host.id)
      inventory.value.total = inventory.value.items.length
    }
    deleting.value = false
    closeManage()
    if (result.credentialRemoved === false) {
      toast.danger('主机已移除，但凭据清理失败', '请检查 KPanel 数据目录权限；服务重启时会再次清理孤立凭据。')
    } else {
      toast.success(
        '主机已移除',
        result.remoteRevoked ? '远端授权也已撤销。' : '远端不可达；可稍后在目标 KPanel 撤销残留授权。',
      )
    }
  } catch (reason) {
    toast.danger('移除失败', friendlyError(reason, '请刷新后重试。'))
  } finally {
    deleting.value = false
  }
}

async function refreshHost(host: ClusterHost): Promise<void> {
  if (host.polling) return
  try {
    const queued = await api.cluster.refresh(host.id)
    upsertHost(queued)
    const timer = window.setTimeout(async () => {
      delayedRefreshes.delete(timer)
      await load(true)
    }, 1_200)
    delayedRefreshes.add(timer)
  } catch (reason) {
    toast.danger('刷新失败', friendlyError(reason, '请稍后重试。'))
  }
}

function upsertHost(host: ClusterHost): void {
  if (!inventory.value) return
  const index = inventory.value.items.findIndex((item) => item.id === host.id)
  if (index >= 0) inventory.value.items[index] = host
  else inventory.value.items.unshift(host)
  inventory.value.total = inventory.value.items.length
}

function openPanel(host: ClusterHost): void {
  window.open(displayOrigin(host), '_blank', 'noopener,noreferrer')
}

function displayOrigin(host: ClusterHost): string {
  return host.isLocal ? window.location.origin : host.origin
}

function onVisibilityChange(): void {
  if (!document.hidden) void load(true)
}

onMounted(() => {
  void load()
  pollTimer = window.setInterval(() => {
    if (!document.hidden) void load(true)
  }, 15_000)
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  loadController?.abort()
  if (pollTimer) window.clearInterval(pollTimer)
  delayedRefreshes.forEach((timer) => window.clearTimeout(timer))
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<template>
  <div class="page cluster-page">
    <PageHeader
      title="集群监控"
      description="每台 KPanel 都能同时作为中心端与被控端；集中查看只读主机概要，管理操作仍在目标面板独立登录完成。"
    >
      <template #actions>
        <button class="button button--secondary" type="button" @click="openAccess">
          <KeyRound :size="16" /> 接入授权
        </button>
        <button class="button button--secondary" type="button" :disabled="refreshing" @click="load(true)">
          <RefreshCw :size="16" :class="{ spin: refreshing }" /> 刷新
        </button>
        <button class="button button--primary" type="button" @click="openAdd">
          <Plus :size="16" /> 添加主机
        </button>
      </template>
    </PageHeader>

    <section class="cluster-hero">
      <div>
        <span class="cluster-hero__eyebrow"><Network :size="15" /> KPanel Federation</span>
        <h2>一处观察，独立管理</h2>
        <p>浏览器只读取当前中心端缓存；远端轮询由后端按固定协议完成，不共享登录态和 Agent 凭据。</p>
      </div>
      <div class="cluster-stats">
        <div><strong>{{ inventory?.total || 0 }}</strong><span>全部节点</span></div>
        <div><strong>{{ onlineCount }}</strong><span>在线</span></div>
        <div><strong>{{ attentionCount }}</strong><span>需关注</span></div>
        <div><strong>{{ inventory?.maxHosts || 100 }}</strong><span>远程上限</span></div>
      </div>
    </section>

    <div v-if="refreshWarning" class="inline-alert inline-alert--warning" role="status">
      {{ refreshWarning }}
    </div>

    <label v-if="inventory?.items.length" class="cluster-search">
      <Server :size="17" />
      <input
        v-model="search"
        type="search"
        aria-label="搜索集群主机"
        placeholder="搜索名称、系统、地区或运营商…"
      />
    </label>

    <LoadingState v-if="loading" title="正在读取集群主机…" />
    <ErrorState v-else-if="loadError && !inventory" :message="loadError" @retry="load()" />
    <EmptyState
      v-else-if="!inventory?.items.length"
      title="尚未添加主机"
      description="先在目标 KPanel 生成一次性授权码，再将它添加到当前面板。"
    >
      <button class="button button--primary" type="button" @click="openAdd">
        <Plus :size="16" /> 添加第一台主机
      </button>
    </EmptyState>
    <EmptyState
      v-else-if="!filteredHosts.length"
      title="没有匹配的主机"
      description="请清除搜索词后重试。"
    />

    <section v-else class="cluster-grid" :aria-busy="refreshing">
      <article v-for="host in filteredHosts" :key="host.id" class="cluster-card">
        <header class="cluster-card__header">
          <OperatingSystemIcon
            :distro="host.lastSnapshot?.telemetry.osId || 'linux'"
            :label="host.lastSnapshot?.telemetry.os || 'Linux'"
          />
          <div>
            <span>
              <strong>{{ host.name }}</strong>
              <em v-if="host.isLocal" class="cluster-card__local">本机</em>
              <StatusBadge :status="host.state" subtle />
            </span>
            <a :href="displayOrigin(host)" target="_blank" rel="noopener noreferrer">
              {{ displayOrigin(host) }} <ArrowUpRight :size="12" />
            </a>
          </div>
          <button
            class="icon-button icon-button--small"
            type="button"
            :disabled="host.polling"
            :aria-label="`刷新 ${host.name}`"
            @click="refreshHost(host)"
          >
            <LoaderCircle v-if="host.polling" class="spin" :size="15" />
            <RefreshCw v-else :size="15" />
          </button>
        </header>

        <div v-if="host.lastSnapshot" class="cluster-card__metrics">
          <div>
            <span><Gauge :size="14" /> CPU</span>
            <strong>{{ formatPercent(host.lastSnapshot.telemetry.cpu.usagePercent) }}</strong>
            <i
              role="progressbar"
              aria-label="CPU 使用率"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-valuenow="clampPercent(host.lastSnapshot.telemetry.cpu.usagePercent)"
            >
              <b :style="{ width: `${clampPercent(host.lastSnapshot.telemetry.cpu.usagePercent)}%` }" />
            </i>
          </div>
          <div>
            <span><MemoryStick :size="14" /> 内存</span>
            <strong>{{ formatPercent(host.lastSnapshot.telemetry.memory.usagePercent) }}</strong>
            <i
              role="progressbar"
              aria-label="内存使用率"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-valuenow="clampPercent(host.lastSnapshot.telemetry.memory.usagePercent)"
            >
              <b :style="{ width: `${clampPercent(host.lastSnapshot.telemetry.memory.usagePercent)}%` }" />
            </i>
          </div>
          <div>
            <span><Server :size="14" /> 磁盘</span>
            <strong>{{ formatPercent(host.lastSnapshot.telemetry.disk.usagePercent) }}</strong>
            <i
              role="progressbar"
              aria-label="磁盘使用率"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-valuenow="clampPercent(host.lastSnapshot.telemetry.disk.usagePercent)"
            >
              <b :style="{ width: `${clampPercent(host.lastSnapshot.telemetry.disk.usagePercent)}%` }" />
            </i>
          </div>
        </div>

        <div v-if="host.lastSnapshot" class="cluster-card__details">
          <div>
            <span>系统</span>
            <strong>{{ host.lastSnapshot.telemetry.os }}</strong>
            <small>{{ host.lastSnapshot.telemetry.architecture }} · {{ host.lastSnapshot.telemetry.kernel }}</small>
          </div>
          <div>
            <span>地区</span>
            <strong>
              <CountryFlagIcon
                v-if="host.lastSnapshot.telemetry.publicNetwork.countryCode"
                :country-code="host.lastSnapshot.telemetry.publicNetwork.countryCode"
                :label="host.lastSnapshot.telemetry.publicNetwork.country || '地区'"
              />
              {{
                [
                  host.lastSnapshot.telemetry.publicNetwork.country,
                  host.lastSnapshot.telemetry.publicNetwork.city,
                ].filter(Boolean).join(' · ') || '未获取'
              }}
            </strong>
            <small>{{ host.lastSnapshot.telemetry.publicNetwork.isp || '运营商未知' }}</small>
          </div>
          <div>
            <span>网络</span>
            <strong>↓ {{ formatRate(host.lastSnapshot.receiveBytesPerSecond) }}</strong>
            <small>↑ {{ formatRate(host.lastSnapshot.transmitBytesPerSecond) }}</small>
          </div>
          <div>
            <span>运行时间</span>
            <strong>{{ formatDuration(host.lastSnapshot.telemetry.uptimeSeconds) }}</strong>
            <small>延迟 {{ host.lastSnapshot.latencyMilliseconds }} ms</small>
          </div>
        </div>
        <div v-else class="cluster-card__empty">
          <ShieldCheck :size="20" />
          <div>
            <strong>等待首次主机摘要</strong>
            <small>{{ host.lastError || '配对已完成，后端正在安全轮询。' }}</small>
          </div>
        </div>

        <div v-if="host.lastError" class="cluster-card__warning" role="status">
          {{ host.lastError }}
        </div>

        <footer class="cluster-card__footer">
          <span>
            最近在线 {{ relativeTime(host.lastSuccessAt) }}
            <small v-if="host.lastSnapshot">采集于 {{ formatDateTime(host.lastSnapshot.receivedAt) }}</small>
          </span>
          <div>
            <button
              v-if="!host.isLocal"
              class="button button--ghost button--small"
              type="button"
              @click="openManage(host)"
            >
              <Pencil :size="14" /> 管理
            </button>
            <button class="button button--primary button--small" type="button" @click="openPanel(host)">
              {{ host.isLocal ? '当前面板' : '打开面板' }} <ArrowUpRight :size="14" />
            </button>
          </div>
        </footer>
      </article>
    </section>

    <ModalDialog
      :open="addOpen"
      title="添加 KPanel 主机"
      description="在目标 KPanel 的“集群 → 接入授权”生成一次性授权码。"
      size="small"
      @close="closeAdd"
    >
      <form id="cluster-add-form" class="form-stack" @submit.prevent="addHost">
        <label class="field">
          主机名称（可选）
          <input v-model="addForm.name" maxlength="80" placeholder="例如：香港生产机" autocomplete="off" />
          <small>留空时使用目标主机名。</small>
        </label>
        <label class="field">
          主机 URL
          <input
            ref="addOriginInput"
            v-model="addForm.origin"
            type="url"
            required
            maxlength="512"
            placeholder="https://panel.example.com"
            autocomplete="url"
          />
          <small>必须是可验证证书的 HTTPS 根地址，不能包含路径或参数。</small>
        </label>
        <label class="field">
          一次性授权码
          <input
            v-model="addForm.pairingCode"
            type="password"
            required
            maxlength="81"
            autocomplete="off"
            placeholder="粘贴目标 KPanel 生成的授权码"
          />
          <small>仅用于本次配对，不会保存在浏览器或审计日志中。</small>
        </label>
      </form>
      <template #footer>
        <button class="button button--secondary" type="button" :disabled="adding" @click="closeAdd">取消</button>
        <button
          class="button button--primary"
          type="submit"
          form="cluster-add-form"
          :disabled="adding || !addForm.origin.trim() || !addForm.pairingCode.trim()"
        >
          <LoaderCircle v-if="adding" class="spin" :size="16" />
          <Plus v-else :size="16" />
          {{ adding ? '正在安全配对…' : '添加主机' }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="accessOpen"
      title="本机接入授权"
      description="本机可以同时被其他 KPanel 只读监控；授权可随时撤销。"
      size="medium"
      @close="closeAccess"
    >
      <div class="cluster-access">
        <section class="cluster-access__code">
          <div>
            <KeyRound :size="20" />
            <span>
              <strong>一次性授权码</strong>
              <small>有效期 5 分钟、只能使用一次、权限固定为只读主机摘要。</small>
            </span>
          </div>
          <button
            v-if="!pairingCode"
            class="button button--primary"
            type="button"
            :disabled="generatingCode"
            @click="createPairingCode"
          >
            <LoaderCircle v-if="generatingCode" class="spin" :size="16" />
            <KeyRound v-else :size="16" /> 生成授权码
          </button>
          <div v-else class="cluster-access__token">
            <code>{{ pairingCode.code }}</code>
            <button class="icon-button icon-button--small" type="button" aria-label="复制授权码" @click="copyPairingCode">
              <Copy :size="15" />
            </button>
            <small>到期时间：{{ formatDateTime(pairingCode.expiresAt) }}</small>
          </div>
        </section>

        <section class="cluster-access__controllers">
          <header>
            <div>
              <strong>已授权控制端</strong>
              <small>这里只列出可读取本机概要的 KPanel，不包含任何远程管理权限。</small>
            </div>
            <button
              class="icon-button icon-button--small"
              type="button"
              aria-label="刷新已授权控制端"
              :disabled="controllersLoading"
              @click="loadControllers"
            >
              <RefreshCw :size="14" :class="{ spin: controllersLoading }" />
            </button>
          </header>
          <p v-if="controllersLoading && !controllers.length">正在读取授权列表…</p>
          <p v-else-if="!controllers.length">暂无已授权控制端。</p>
          <template v-else>
            <article v-for="controller in controllers" :key="controller.id">
              <span>
                <strong>{{ controller.name || '未命名 KPanel' }}</strong>
                <code>{{ controller.fingerprint }}</code>
                <small>
                  授权于 {{ formatDateTime(controller.createdAt) }} · 最近访问
                  {{ relativeTime(controller.lastSeenAt) }}
                </small>
              </span>
              <button
                class="icon-button icon-button--small icon-button--danger"
                type="button"
                :aria-label="`撤销 ${controller.name || '控制端'} 授权`"
                @click="revokeController(controller)"
              >
                <Trash2 :size="14" />
              </button>
            </article>
          </template>
        </section>
      </div>
    </ModalDialog>

    <ModalDialog
      :open="manageOpen"
      :title="selected ? `管理 ${selected.name}` : '管理主机'"
      description="修改仅影响当前中心端显示；移除不会停止目标主机业务。"
      size="small"
      @close="closeManage"
    >
      <div v-if="selected" class="form-stack">
        <label class="field">
          显示名称
          <input v-model="editName" maxlength="80" autocomplete="off" />
        </label>
        <div class="cluster-manage__identity">
          <span>目标地址</span><code>{{ selected.origin }}</code>
          <span>节点 ID</span><code>{{ selected.remoteNodeId }}</code>
          <span>Panel / Agent</span>
          <code>{{ selected.panelVersion || '未知' }} / {{ selected.lastSnapshot?.telemetry.agentVersion || '未知' }}</code>
        </div>
      </div>
      <template #footer>
        <button class="button button--danger" type="button" :disabled="saving || deleting" @click="removeHost">
          <LoaderCircle v-if="deleting" class="spin" :size="16" />
          <Trash2 v-else :size="16" /> 移除主机
        </button>
        <button class="button button--secondary" type="button" :disabled="saving || deleting" @click="closeManage">关闭</button>
        <button
          class="button button--primary"
          type="button"
          :disabled="saving || deleting || !editName.trim()"
          @click="saveName"
        >
          <LoaderCircle v-if="saving" class="spin" :size="16" />
          <Check v-else :size="16" /> 保存名称
        </button>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.cluster-page {
  gap: 18px;
}

.cluster-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(420px, 0.72fr);
  align-items: center;
  gap: 28px;
  padding: 28px 30px;
  overflow: hidden;
  background:
    radial-gradient(circle at 95% 10%, color-mix(in srgb, var(--brand) 13%, transparent), transparent 33%),
    linear-gradient(135deg, color-mix(in srgb, var(--brand-soft) 58%, var(--surface)), var(--surface));
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.cluster-hero__eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--brand);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.cluster-hero h2 {
  margin: 8px 0 6px;
  font-size: clamp(24px, 2.6vw, 34px);
  letter-spacing: -0.045em;
}

.cluster-hero p {
  max-width: 650px;
  margin: 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.7;
}

.cluster-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: hidden;
  background: color-mix(in srgb, var(--surface) 88%, transparent);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.cluster-stats div {
  display: grid;
  min-height: 74px;
  place-content: center;
  text-align: center;
  border-left: 1px solid var(--border);
}

.cluster-stats div:first-child {
  border-left: 0;
}

.cluster-stats strong {
  font-size: 22px;
}

.cluster-stats span {
  margin-top: 2px;
  color: var(--muted);
  font-size: 11px;
}

.cluster-search {
  display: flex;
  width: min(520px, 100%);
  height: 42px;
  align-items: center;
  gap: 9px;
  padding: 0 13px;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.cluster-search:focus-within {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand) 12%, transparent);
}

.cluster-search input {
  width: 100%;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: 0;
}

.cluster-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.cluster-card {
  display: grid;
  min-width: 0;
  overflow: hidden;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
}

.cluster-card__header {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--border);
}

.cluster-card__header > div {
  min-width: 0;
}

.cluster-card__header > div > span {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.cluster-card__header strong {
  overflow: hidden;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cluster-card__local {
  flex: 0 0 auto;
  padding: 2px 7px;
  color: var(--brand);
  font-size: 10px;
  font-style: normal;
  font-weight: 800;
  background: var(--brand-soft);
  border: 1px solid color-mix(in srgb, var(--brand) 22%, var(--border));
  border-radius: 999px;
}

.cluster-card__header a {
  display: inline-flex;
  max-width: 100%;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  overflow: hidden;
  color: var(--muted);
  font-size: 11px;
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cluster-card__metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  background: var(--border);
  border-bottom: 1px solid var(--border);
}

.cluster-card__metrics > div {
  display: grid;
  gap: 5px;
  padding: 12px;
  background: var(--surface);
}

.cluster-card__metrics span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 10px;
}

.cluster-card__metrics strong {
  font-size: 17px;
}

.cluster-card__metrics i {
  display: block;
  height: 3px;
  overflow: hidden;
  background: var(--surface-muted);
  border-radius: 99px;
}

.cluster-card__metrics b {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--brand), #3bbfa3);
  border-radius: inherit;
}

.cluster-card__details {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
  padding: 15px 16px;
}

.cluster-card__details > div {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.cluster-card__details span {
  color: var(--muted);
  font-size: 10px;
}

.cluster-card__details strong {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  overflow-wrap: anywhere;
  font-size: 12px;
}

.cluster-card__details small {
  overflow: hidden;
  color: var(--muted);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cluster-card__empty {
  display: flex;
  min-height: 138px;
  align-items: center;
  justify-content: center;
  gap: 11px;
  padding: 20px;
  color: var(--muted);
}

.cluster-card__empty div {
  display: grid;
  gap: 4px;
}

.cluster-card__empty small {
  max-width: 280px;
  line-height: 1.5;
}

.cluster-card__warning {
  padding: 9px 16px;
  color: var(--danger);
  background: var(--danger-soft);
  border-top: 1px solid color-mix(in srgb, var(--danger) 18%, transparent);
  font-size: 11px;
}

.cluster-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  margin-top: auto;
  border-top: 1px solid var(--border);
}

.cluster-card__footer > span {
  display: grid;
  color: var(--text-soft);
  font-size: 10px;
}

.cluster-card__footer small {
  color: var(--muted);
}

.cluster-card__footer > div {
  display: flex;
  flex: 0 0 auto;
  gap: 6px;
}

.cluster-access {
  display: grid;
  gap: 16px;
}

.cluster-access__code,
.cluster-access__controllers {
  padding: 16px;
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.cluster-access__code > div:first-child,
.cluster-access__controllers header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.cluster-access__code > div:first-child {
  justify-content: flex-start;
  margin-bottom: 14px;
}

.cluster-access__code span,
.cluster-access__controllers header div,
.cluster-access__controllers article span {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.cluster-access small {
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
}

.cluster-access__token {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
}

.cluster-access__token code {
  padding: 10px;
  overflow: auto hidden;
  color: var(--brand);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 11px;
  user-select: all;
}

.cluster-access__token small {
  grid-column: 1 / -1;
}

.cluster-access__controllers header {
  margin-bottom: 10px;
}

.cluster-access__controllers > p {
  margin: 12px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.cluster-access__controllers article {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 11px 0;
  border-top: 1px solid var(--border);
}

.cluster-access__controllers code {
  overflow: hidden;
  color: var(--muted);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cluster-manage__identity {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 8px 12px;
  padding: 13px;
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.cluster-manage__identity span {
  color: var(--muted);
  font-size: 11px;
}

.cluster-manage__identity code {
  overflow-wrap: anywhere;
  font-size: 11px;
}

@media (max-width: 1240px) {
  .cluster-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .cluster-hero {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 680px) {
  .cluster-grid {
    grid-template-columns: 1fr;
  }

  .cluster-hero {
    padding: 21px;
  }

  .cluster-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .cluster-stats div:nth-child(3) {
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .cluster-stats div:nth-child(4) {
    border-top: 1px solid var(--border);
  }

  .cluster-card__footer {
    align-items: flex-start;
    flex-direction: column;
  }

  .cluster-card__footer > div,
  .cluster-card__footer .button {
    width: 100%;
  }

  .cluster-card__footer .button {
    flex: 1 1 0;
  }
}
</style>
