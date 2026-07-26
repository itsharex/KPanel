<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type Component } from 'vue'
import {
  Activity,
  ArrowLeftRight,
  ArrowDownToLine,
  ArrowUpFromLine,
  Bolt,
  Box,
  Boxes,
  ChevronRight,
  CircleAlert,
  Clock3,
  Cpu,
  Database,
  Gauge,
  Globe2,
  HardDrive,
  KeyRound,
  MemoryStick,
  Network,
  Pencil,
  RefreshCw,
  RefreshCcw,
  Server,
  Settings2,
  ShieldCheck,
  Timer,
  Wrench,
} from '@lucide/vue'
import PageHeader from '@/components/common/PageHeader.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
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

interface ManagementTool {
  id: string
  title: string
  description: string
  value: string
  detail: string
  capability: string
  safety: string
  icon: Component
  tone?: 'blue' | 'violet' | 'amber' | 'danger'
}

const selectedTool = ref<ManagementTool>()

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

const basicSettings = computed<ManagementTool[]>(() => {
  if (!data.value) return []
  const management = data.value.management
  return [
    {
      id: 'hostname',
      title: '主机名',
      description: '对应 kejilion.sh 的“修改主机名”。',
      value: data.value.hostname || '未命名主机',
      detail: '同步识别系统当前 hostname',
      capability: 'system.hostname.write',
      safety: '写入前校验 Linux hostname 规则，并原子更新 hostname 与 hosts 映射。',
      icon: Pencil,
    },
    {
      id: 'ssh-port',
      title: 'SSH 端口',
      description: '对应 kejilion.sh 的“修改 SSH 端口”。',
      value: management.ssh.ports.length ? management.ssh.ports.join('、') : '待 Agent 升级',
      detail: management.ssh.source === 'default' ? 'OpenSSH 默认端口' : '来自 sshd 配置',
      capability: 'system.ssh-port.write',
      safety: '必须先开放并验证新端口，再保留旧会话和旧端口作为恢复通道。',
      icon: KeyRound,
      tone: 'blue',
    },
    {
      id: 'dns',
      title: 'DNS 地址',
      description: '对应 kejilion.sh 的“DNS 优化/修改 DNS”。',
      value: management.dns.servers.length ? management.dns.servers.join(' · ') : '未识别',
      detail: `解析器：${management.dns.manager || 'unknown'}`,
      capability: 'system.dns.write',
      safety: '按 systemd-resolved、resolvconf 或静态配置分别处理，不锁定 resolv.conf。',
      icon: Network,
      tone: 'violet',
    },
    {
      id: 'timezone',
      title: '系统时区',
      description: '对应 kejilion.sh 的“设置系统时区”。',
      value: management.timezone || '待 Agent 升级',
      detail: '显示宿主机实际时区',
      capability: 'system.timezone.write',
      safety: '仅接受 IANA 时区数据库中的有效名称，变更后立即回读验证。',
      icon: Timer,
      tone: 'amber',
    },
  ]
})

const systemTools = computed<ManagementTool[]>(() => {
  if (!data.value) return []
  const management = data.value.management
  const swapPercent = management.swap.totalBytes
    ? (management.swap.usedBytes / management.swap.totalBytes) * 100
    : 0
  return [
    {
      id: 'swap',
      title: '虚拟内存',
      description: '对应 kejilion.sh 的“设置虚拟内存”。',
      value: management.swap.totalBytes
        ? `${formatBytes(management.swap.totalBytes)} · 已用 ${formatPercent(swapPercent)}`
        : '未启用',
      detail: `${management.swap.activeDevices} 个活动 Swap`,
      capability: 'system.swap.write',
      safety: '仅管理 KPanel 专属 swapfile，不清除已有 Swap 分区或第三方 swapfile。',
      icon: MemoryStick,
    },
    {
      id: 'mirror',
      title: '系统镜像源',
      description: '对应 kejilion.sh 的“换系统更新源”。',
      value: management.packageSources[0] || '未识别',
      detail: management.packageManager ? `${management.packageManager.toUpperCase()} 软件源` : '等待 Agent 识别',
      capability: 'system.mirror.write',
      safety: '修改前备份源文件，执行语法与连通性测试；测试失败自动恢复。',
      icon: Globe2,
      tone: 'blue',
    },
    {
      id: 'ip-preference',
      title: 'V4 / V6 优先',
      description: '对应 kejilion.sh 的“设置 v4/v6 优先级”。',
      value: management.ipPreference === 'ipv4' ? 'IPv4 优先' : management.ipPreference === 'system_default' ? '系统默认' : '未识别',
      detail: management.ipPreference === 'ipv4' ? '已识别 gai.conf 规则' : '未发现 Kejilion IPv4 优先规则',
      capability: 'system.ip-preference.write',
      safety: '只维护一条带 KPanel 标识的 gai.conf 规则，撤销时不删除用户原配置。',
      icon: ArrowLeftRight,
      tone: 'violet',
    },
    {
      id: 'kernel',
      title: '内核优化',
      description: '对应 kejilion.sh 的“Linux 内核调优管理”。',
      value: management.kernelOptimization.enabled
        ? management.kernelOptimization.profile || '自定义'
        : '未启用',
      detail: management.kernelOptimization.source === 'kejilion' ? '已识别 Kejilion 产物' : '系统默认参数',
      capability: 'system.kernel-tuning.write',
      safety: '使用参数白名单和独立 sysctl 文件，应用前校验，失败时恢复上一版本。',
      icon: Settings2,
      tone: 'amber',
    },
    {
      id: 'bbr',
      title: 'BBR 加速',
      description: '对应 kejilion.sh 的“开启 BBR 加速”。',
      value: management.bbr.enabled ? '已启用' : management.bbr.supported ? '可启用' : '当前内核不支持',
      detail: [
        management.bbr.congestionControl ? `拥塞算法 ${management.bbr.congestionControl}` : '',
        management.bbr.defaultQDisc ? `队列 ${management.bbr.defaultQDisc}` : '',
      ]
        .filter(Boolean)
        .join(' · ') || '等待 Agent 识别',
      capability: 'system.bbr.write',
      safety: '先确认内核暴露 bbr 算法，再写入独立 sysctl 配置并回读生效状态。',
      icon: Bolt,
    },
  ]
})

const reinstallTool = computed<ManagementTool>(() => ({
  id: 'reinstall',
  title: '重装系统',
  description: '对应 kejilion.sh 的“重装系统”。此操作会清除系统并导致面板离线。',
  value: '高风险操作',
  detail: '当前版本保持锁定',
  capability: 'system.reinstall',
  safety: '必须依赖带外控制台、数据备份、一次性恢复凭证和二次确认；仅有 Web 会话时不开放。',
  icon: RefreshCcw,
  tone: 'danger',
}))

function capabilityState(id: string): { enabled: boolean; reason: string } {
  const capability = data.value?.management.capabilities[id]
  return {
    enabled: Boolean(capability?.enabled),
    reason: capability?.reason || '当前 Agent 仅提供状态读取',
  }
}

function openTool(tool: ManagementTool): void {
  selectedTool.value = tool
}

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
    <PageHeader
      title="系统监控与管理"
      description="实时资源状态与 kejilion.sh 系统工具统一入口；所有配置均以宿主机实际状态为准。"
    >
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

      <div class="management-layout">
        <section class="panel-card">
          <header class="panel-card__header">
            <div>
              <span class="panel-card__icon"><Wrench :size="18" /></span>
              <div>
                <h2>基础系统设置</h2>
                <p>与 kejilion.sh 当前系统配置双向识别</p>
              </div>
            </div>
            <span class="management-read-state"><ShieldCheck :size="14" /> 只读识别</span>
          </header>

          <div class="configuration-list">
            <button
              v-for="tool in basicSettings"
              :key="tool.id"
              class="configuration-row"
              type="button"
              @click="openTool(tool)"
            >
              <span class="configuration-row__icon" :class="tool.tone ? `is-${tool.tone}` : ''">
                <component :is="tool.icon" :size="18" />
              </span>
              <span class="configuration-row__body">
                <span>{{ tool.title }}</span>
                <strong>{{ tool.value }}</strong>
                <small>{{ tool.detail }}</small>
              </span>
              <span class="configuration-row__action">
                <span>{{ capabilityState(tool.capability).enabled ? '可配置' : '查看' }}</span>
                <ChevronRight :size="16" />
              </span>
            </button>
          </div>
        </section>

        <section class="panel-card">
          <header class="panel-card__header">
            <div>
              <span class="panel-card__icon panel-card__icon--blue"><Settings2 :size="18" /></span>
              <div>
                <h2>性能与网络工具</h2>
                <p>沿用脚本业务分组，显示真实生效状态</p>
              </div>
            </div>
          </header>

          <div class="system-tool-grid">
            <button
              v-for="tool in systemTools"
              :key="tool.id"
              class="system-tool"
              type="button"
              @click="openTool(tool)"
            >
              <span class="system-tool__top">
                <span class="system-tool__icon" :class="tool.tone ? `is-${tool.tone}` : ''">
                  <component :is="tool.icon" :size="19" />
                </span>
                <span class="system-tool__state">
                  {{ capabilityState(tool.capability).enabled ? '可配置' : '只读' }}
                </span>
              </span>
              <strong>{{ tool.title }}</strong>
              <span>{{ tool.value }}</span>
              <small>{{ tool.detail }}</small>
            </button>
          </div>
        </section>
      </div>

      <section class="danger-zone" aria-labelledby="reinstall-title">
        <span class="danger-zone__icon"><CircleAlert :size="21" /></span>
        <span class="danger-zone__body">
          <strong id="reinstall-title">重装系统</strong>
          <small>清除系统数据并导致 KPanel 离线。没有带外恢复通道时保持锁定。</small>
        </span>
        <button class="button button--danger button--small" type="button" @click="openTool(reinstallTool)">
          查看安全要求
        </button>
      </section>

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

    <ModalDialog
      :open="Boolean(selectedTool)"
      :title="selectedTool?.title || '系统工具'"
      :description="selectedTool?.description"
      @close="selectedTool = undefined"
    >
      <div v-if="selectedTool" class="management-dialog">
        <div class="management-dialog__current" :class="{ 'is-danger': selectedTool.tone === 'danger' }">
          <span>当前状态</span>
          <strong>{{ selectedTool.value }}</strong>
          <small>{{ selectedTool.detail }}</small>
        </div>

        <div class="management-dialog__section">
          <span class="management-dialog__section-icon"><ShieldCheck :size="17" /></span>
          <div>
            <strong>KPanel 安全执行规则</strong>
            <p>{{ selectedTool.safety }}</p>
          </div>
        </div>

        <div class="inline-alert inline-alert--warning">
          <CircleAlert :size="17" />
          <span>
            当前版本只读取并呈现宿主机真实状态，不发送变更命令。
            {{ capabilityState(selectedTool.capability).reason }}
          </span>
        </div>
      </div>
      <template #footer>
        <button class="button button--secondary" type="button" @click="selectedTool = undefined">关闭</button>
        <button class="button button--primary" type="button" disabled>
          <ShieldCheck :size="16" />
          安全执行器未启用
        </button>
      </template>
    </ModalDialog>
  </div>
</template>
