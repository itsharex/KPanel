<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  ArrowRight,
  Braces,
  FileCode2,
  Globe2,
  KeyRound,
  LoaderCircle,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  TriangleAlert,
  Waypoints,
} from '@lucide/vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { formatDateTime, relativeTime, shortId } from '@/lib/format'
import { usePanelState } from '@/stores/panel'
import { useToast } from '@/stores/toast'
import type { Site, SiteInput } from '@/types/api'

type Filter = 'all' | 'healthy' | 'drifted' | 'read-only'
type SiteServiceType = SiteInput['type']
type RedirectCode = NonNullable<SiteInput['redirectCode']>
type PHPVersion = NonNullable<SiteInput['phpVersion']>

const sites = ref<Site[]>([])
const capabilities = ref<Array<{ id: string; enabled: boolean; reason?: string; methods?: string[] }>>([])
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const search = ref('')
const filter = ref<Filter>('all')
const selectedSite = ref<Site>()
const editorOpen = ref(false)
const editingSite = ref<Site>()
const submitting = ref(false)
const formError = ref('')
const panel = usePanelState()
const toast = useToast()
let controller: AbortController | undefined

const serviceOptions = [
  {
    type: 'static',
    title: '静态网站',
    summary: 'HTML、图片与前端构建产物',
    detail: '创建独立站点目录与默认首页',
    icon: FileCode2,
  },
  {
    type: 'php',
    title: 'PHP 网站',
    summary: '动态网站与自建 PHP 程序',
    detail: '使用 kejilion.sh 同款 PHP-FPM Socket',
    icon: Braces,
  },
  {
    type: 'proxy',
    title: 'IP / 端口反代',
    summary: '代理本机、内网或 Docker 服务',
    detail: '例如 127.0.0.1:3000',
    icon: Server,
  },
  {
    type: 'proxy_domain',
    title: '域名反代',
    summary: '代理另一域名提供的 HTTPS 服务',
    detail: '自动配置上游 SNI',
    icon: Globe2,
  },
  {
    type: 'load_balance',
    title: '负载均衡',
    summary: '将请求分配到多个后端节点',
    detail: '支持 2–8 个 HTTP 上游',
    icon: Waypoints,
  },
  {
    type: 'redirect',
    title: '域名重定向',
    summary: '将访问跳转到另一个域名',
    detail: '支持 301、302、307、308',
    icon: ArrowRight,
  },
] as const satisfies ReadonlyArray<{
  type: SiteServiceType
  title: string
  summary: string
  detail: string
  icon: typeof FileCode2
}>

const form = reactive({
  primaryDomain: '',
  aliases: '',
  type: 'static' as SiteServiceType,
  upstream: '',
  upstreams: '',
  redirectTarget: '',
  redirectCode: 301 as RedirectCode,
  phpVersion: 'latest' as PHPVersion,
})

const siteWriteCapability = computed(() => capabilities.value.find((capability) => capability.id === 'sites.write'))
const canCreate = computed(() => siteWriteCapability.value?.enabled === true)
const siteWriteReason = computed(
  () =>
    siteWriteCapability.value?.reason?.trim() ||
    (siteWriteCapability.value
      ? 'Agent 当前未开放网站安全写入能力。'
      : '未从 Agent 获取网站写入能力状态，请检查 Agent 连接与版本。'),
)

const filteredSites = computed(() => {
  const query = search.value.trim().toLowerCase()
  return sites.value.filter((site) => {
    const matchesQuery =
      !query ||
      site.primaryDomain.toLowerCase().includes(query) ||
      site.domains.some((domain) => domain.toLowerCase().includes(query)) ||
      site.upstream?.toLowerCase().includes(query) ||
      site.rootPath?.toLowerCase().includes(query)
    if (!matchesQuery) return false
    if (filter.value === 'healthy') return site.health === 'healthy' && site.consistency === 'synced'
    if (filter.value === 'drifted') return site.consistency !== 'synced'
    if (filter.value === 'read-only') return site.access !== 'managed'
    return true
  })
})

const counts = computed(() => ({
  all: sites.value.length,
  healthy: sites.value.filter((site) => site.health === 'healthy' && site.consistency === 'synced').length,
  drifted: sites.value.filter((site) => site.consistency !== 'synced').length,
  'read-only': sites.value.filter((site) => site.access !== 'managed').length,
}))

const selectedService = computed(() => serviceOptions.find((option) => option.type === form.type))

const formValid = computed(() => {
  const domain = form.primaryDomain.trim()
  if (!isDomain(domain)) return false
  if (form.type === 'proxy' || form.type === 'proxy_domain') return isOrigin(form.upstream)
  if (form.type === 'load_balance') {
    const upstreams = splitUpstreams(form.upstreams)
    return upstreams.length >= 2 && upstreams.length <= 8 && upstreams.every(isHTTPOrigin)
  }
  if (form.type === 'redirect') return isOrigin(form.redirectTarget)
  return true
})

function isDomain(value: string): boolean {
  return /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/i.test(
    value,
  )
}

function isOrigin(value: string): boolean {
  try {
    const parsed = new URL(value.trim())
    return (
      (parsed.protocol === 'http:' || parsed.protocol === 'https:') &&
      !parsed.username &&
      !parsed.password &&
      parsed.pathname === '/' &&
      !parsed.search &&
      !parsed.hash
    )
  } catch {
    return false
  }
}

function isHTTPOrigin(value: string): boolean {
  return isOrigin(value) && new URL(value.trim()).protocol === 'http:'
}

function splitUpstreams(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function sourceLabel(source: Site['source']): string {
  return {
    kejilion: 'kejilion.sh / 发现',
    panel: '面板创建',
    external: '外部配置',
    unknown: '未知来源',
  }[source]
}

function typeLabel(type: Site['type']): string {
  return {
    static: '静态站点',
    proxy: 'IP / 端口反代',
    proxy_domain: '域名反代',
    load_balance: '负载均衡',
    php: 'PHP 网站',
    wordpress: 'WordPress',
    redirect: '域名重定向',
    unknown: '未知类型',
  }[type]
}

function siteTargetLabel(site: Site): string {
  if (site.type === 'static' || site.type === 'php' || site.type === 'wordpress') return '站点目录'
  if (site.type === 'redirect') return '跳转目标'
  if (site.type === 'load_balance') return '上游节点'
  return '上游地址'
}

function siteTargetValue(site: Site): string {
  if (site.type === 'php' || site.type === 'wordpress') {
    const runtime = site.upstream === 'php74' ? 'PHP 7.4' : site.upstream === 'php' ? 'PHP 最新版' : ''
    return [site.rootPath, runtime].filter(Boolean).join(' · ') || '—'
  }
  return site.upstream || site.rootPath || '—'
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''

  try {
    const [siteResult, capabilityResult] = await Promise.allSettled([
      api.sites.list(undefined, controller.signal),
      api.agent.capabilities(controller.signal),
    ])
    capabilities.value = capabilityResult.status === 'fulfilled' ? capabilityResult.value : []
    if (siteResult.status === 'rejected') throw siteResult.reason
    sites.value = siteResult.value.items
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    error.value = reason instanceof ApiError ? reason.message : '无法读取网站列表。'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function openCreate(): void {
  editingSite.value = undefined
  form.primaryDomain = ''
  form.aliases = ''
  form.type = 'static'
  form.upstream = ''
  form.upstreams = ''
  form.redirectTarget = ''
  form.redirectCode = 301
  form.phpVersion = 'latest'
  formError.value = ''
  editorOpen.value = true
}

function openEdit(site: Site): void {
  if (!serviceOptions.some((option) => option.type === site.type)) return
  editingSite.value = site
  form.primaryDomain = site.primaryDomain
  form.aliases = site.domains.filter((domain) => domain !== site.primaryDomain).join('\n')
  form.type = site.type as SiteServiceType
  form.upstream = site.type === 'proxy' || site.type === 'proxy_domain' ? site.upstream || '' : ''
  form.upstreams = site.type === 'load_balance' ? (site.upstream || '').split(',').join('\n') : ''
  const redirectMatch = site.type === 'redirect' ? (site.upstream || '').match(/^(301|302|307|308)\s+(.+)$/) : null
  form.redirectCode = redirectMatch ? (Number(redirectMatch[1]) as RedirectCode) : 301
  form.redirectTarget = redirectMatch?.[2] || ''
  form.phpVersion = site.type === 'php' && site.upstream === 'php74' ? '7.4' : 'latest'
  formError.value = ''
  selectedSite.value = undefined
  editorOpen.value = true
}

async function submitSite(): Promise<void> {
  formError.value = ''
  if (!formValid.value) {
    formError.value = '请检查域名和当前服务所需的配置。'
    return
  }

  submitting.value = true
  const input: SiteInput = {
    primaryDomain: form.primaryDomain.trim().toLowerCase(),
    aliases: form.aliases
      .split(/[\n,]/)
      .map((value) => value.trim().toLowerCase())
      .filter(Boolean),
    type: form.type,
    upstream: form.type === 'proxy' || form.type === 'proxy_domain' ? form.upstream.trim() : undefined,
    upstreams: form.type === 'load_balance' ? splitUpstreams(form.upstreams) : undefined,
    redirectTarget: form.type === 'redirect' ? form.redirectTarget.trim() : undefined,
    redirectCode: form.type === 'redirect' ? form.redirectCode : undefined,
    phpVersion: form.type === 'php' ? form.phpVersion : undefined,
    enabled: true,
    expectedResourceVersion: editingSite.value?.resourceVersion,
  }

  try {
    const wasEditing = Boolean(editingSite.value)
    const savedSite = editingSite.value
      ? await api.sites.update(editingSite.value.id, input)
      : await api.sites.create(input)
    editorOpen.value = false
    toast.success(
      wasEditing ? '网站已安全更新' : '网站已安全创建',
      `${savedSite.primaryDomain} 已通过 nginx -t 校验并完成同步应用。`,
    )
    await load(true)
  } catch (reason) {
    formError.value = reason instanceof ApiError ? reason.message : '操作失败，请稍后重试。'
  } finally {
    submitting.value = false
  }
}

watch(editorOpen, (open) => {
  if (!open) formError.value = ''
})

onMounted(() => void load())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div class="page">
    <PageHeader
      title="网站管理"
      description="从实际产物发现网站；新建站点沿用 kejilion.sh 的 /home/web 架构，并通过安全事务提交。"
    >
      <template #actions>
        <button
          class="button button--primary"
          type="button"
          :disabled="!canCreate || panel.isReadOnly.value"
          :title="!canCreate ? siteWriteReason : ''"
          @click="openCreate"
        >
          <Plus :size="17" /> 新建网站
        </button>
      </template>
    </PageHeader>

    <div v-if="!canCreate && !loading" class="inline-alert inline-alert--info" role="status">
      <ShieldCheck :size="17" />
      <span><strong>网站写入当前不可用</strong><br />{{ siteWriteReason }}</span>
    </div>

    <section class="toolbar-card">
      <div class="search-field">
        <Search :size="17" aria-hidden="true" />
        <input v-model="search" type="search" placeholder="搜索域名、目录或上游地址" aria-label="搜索网站" />
      </div>
      <div class="filter-tabs" role="tablist" aria-label="网站筛选">
        <button
          v-for="item in [
            { key: 'all', label: '全部' },
            { key: 'healthy', label: '正常' },
            { key: 'drifted', label: '待核对' },
            { key: 'read-only', label: '只读' },
          ]"
          :key="item.key"
          type="button"
          role="tab"
          :aria-selected="filter === item.key"
          :class="{ 'is-active': filter === item.key }"
          @click="filter = item.key as Filter"
        >
          {{ item.label }} <span>{{ counts[item.key as Filter] }}</span>
        </button>
      </div>
      <button class="icon-button" type="button" aria-label="刷新网站列表" :disabled="refreshing" @click="load(true)">
        <RefreshCw :size="18" :class="{ spin: refreshing }" />
      </button>
    </section>

    <LoadingState v-if="loading" :rows="5" />
    <ErrorState v-else-if="error && !sites.length" :message="error" @retry="load()" />
    <EmptyState
      v-else-if="!filteredSites.length"
      :title="sites.length ? '没有符合条件的网站' : '尚未发现网站'"
      :description="sites.length ? '尝试更换搜索词或筛选条件。' : 'Agent 会安全扫描现有 Kejilion 网站产物。'"
    />

    <section v-else class="table-card">
      <div class="table-scroll">
        <table class="data-table">
          <thead>
            <tr>
              <th>网站</th>
              <th>类型 / 目标</th>
              <th>证书</th>
              <th>一致性</th>
              <th>来源</th>
              <th>观测时间</th>
              <th><span class="sr-only">操作</span></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="site in filteredSites" :key="site.id">
              <td>
                <button class="resource-name" type="button" @click="selectedSite = site">
                  <span class="resource-name__icon"><Globe2 :size="18" /></span>
                  <span>
                    <strong>{{ site.primaryDomain }}</strong>
                    <small>{{ site.enabled ? '已启用' : '已停用' }} · {{ site.domains.length }} 个域名</small>
                  </span>
                </button>
              </td>
              <td>
                <div class="table-stack">
                  <StatusBadge :status="site.type" :label="typeLabel(site.type)" subtle />
                  <small :title="siteTargetValue(site)">
                    {{ siteTargetValue(site) }}
                  </small>
                </div>
              </td>
              <td>
                <div class="table-stack">
                  <StatusBadge :status="site.certificate?.status || 'unknown'" subtle />
                  <small v-if="site.certificate?.expiresAt">{{ relativeTime(site.certificate.expiresAt) }}</small>
                </div>
              </td>
              <td>
                <div class="table-stack">
                  <StatusBadge :status="site.consistency" subtle />
                  <small v-if="site.reason" :title="site.reason">{{ site.reason }}</small>
                </div>
              </td>
              <td>
                <div class="table-stack">
                  <span>{{ sourceLabel(site.source) }}</span>
                  <small>{{ site.access === 'managed' ? '允许安全变更' : '仅查看' }}</small>
                </div>
              </td>
              <td>
                <span class="table-time">{{ relativeTime(site.observedAt) }}</span>
              </td>
              <td class="table-actions">
                <button class="button button--ghost button--small" type="button" @click="selectedSite = site">详情</button>
                <button
                  v-if="site.allowedActions?.includes('update')"
                  class="button button--secondary button--small"
                  type="button"
                  :disabled="panel.isReadOnly.value || !canCreate"
                  :title="!canCreate ? siteWriteReason : ''"
                  @click="openEdit(site)"
                >
                  设置
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <footer class="table-card__footer">已显示 {{ filteredSites.length }} / {{ sites.length }} 个网站</footer>
    </section>

    <ModalDialog
      :open="Boolean(selectedSite)"
      :title="selectedSite?.primaryDomain || '网站详情'"
      description="以下信息来自最近一次实际产物对账。"
      size="large"
      @close="selectedSite = undefined"
    >
      <template v-if="selectedSite">
        <div class="modal-status-row">
          <StatusBadge :status="selectedSite.health" />
          <StatusBadge :status="selectedSite.consistency" />
          <StatusBadge :status="selectedSite.access" />
        </div>

        <dl class="detail-list detail-list--grid">
          <div>
            <dt>类型</dt>
            <dd>{{ typeLabel(selectedSite.type) }}</dd>
          </div>
          <div>
            <dt>来源</dt>
            <dd>{{ sourceLabel(selectedSite.source) }}</dd>
          </div>
          <div>
            <dt>资源版本</dt>
            <dd><code>{{ shortId(selectedSite.resourceVersion, 20) }}</code></dd>
          </div>
          <div>
            <dt>最后对账</dt>
            <dd>{{ formatDateTime(selectedSite.observedAt) }}</dd>
          </div>
          <div class="detail-list__wide">
            <dt>{{ siteTargetLabel(selectedSite) }}</dt>
            <dd><code>{{ siteTargetValue(selectedSite) }}</code></dd>
          </div>
          <div class="detail-list__wide">
            <dt>绑定域名</dt>
            <dd>{{ selectedSite.domains.join('、') }}</dd>
          </div>
        </dl>

        <section class="detail-section">
          <h3><KeyRound :size="17" /> TLS 证书</h3>
          <div class="detail-section__line">
            <StatusBadge :status="selectedSite.certificate?.status || 'unknown'" />
            <span v-if="selectedSite.certificate?.expiresAt">
              到期时间 {{ formatDateTime(selectedSite.certificate.expiresAt) }}
            </span>
            <span v-else>未发现可用证书到期信息</span>
          </div>
        </section>

        <section v-if="selectedSite.artifacts?.length" class="detail-section">
          <h3><FileCode2 :size="17" /> 实际产物</h3>
          <ul class="artifact-list">
            <li v-for="artifact in selectedSite.artifacts" :key="`${artifact.kind}-${artifact.path}`">
              <Braces :size="15" />
              <span>{{ artifact.kind }}</span>
              <code>{{ artifact.path }}</code>
            </li>
          </ul>
        </section>

        <div v-if="selectedSite.warnings?.length" class="inline-alert inline-alert--warning">
          <TriangleAlert :size="17" />
          <span>{{ selectedSite.warnings.join('；') }}</span>
        </div>
      </template>
      <template #footer>
        <button class="button button--secondary" type="button" @click="selectedSite = undefined">关闭</button>
        <button
          v-if="selectedSite?.allowedActions?.includes('update')"
          class="button button--primary"
          type="button"
          :disabled="panel.isReadOnly.value || !canCreate"
          :title="!canCreate ? siteWriteReason : ''"
          @click="openEdit(selectedSite)"
        >
          编辑设置
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="editorOpen"
      :title="editingSite ? '编辑网站设置' : '新建网站'"
      description="按 kejilion.sh 的站点架构生成固定配置；先通过 nginx -t，成功后才 reload。"
      size="large"
      @close="editorOpen = false"
    >
      <form id="site-form" class="form-stack" @submit.prevent="submitSite">
        <div v-if="formError" class="inline-alert inline-alert--danger" role="alert">{{ formError }}</div>
        <label class="field">
          <span>主域名</span>
          <input
            v-model.trim="form.primaryDomain"
            placeholder="example.com"
            autocomplete="off"
            required
            :disabled="Boolean(editingSite)"
          />
          <small>{{ editingSite ? '首版更新不重命名主域名或移动网站目录。' : '不要包含协议、路径或端口。' }}</small>
        </label>

        <fieldset class="site-service-field">
          <legend>选择站点服务</legend>
          <div class="site-service-grid">
            <button
              v-for="option in serviceOptions"
              :key="option.type"
              class="site-service-card"
              :class="{ 'is-active': form.type === option.type }"
              type="button"
              :disabled="Boolean(editingSite)"
              :aria-pressed="form.type === option.type"
              @click="form.type = option.type"
            >
              <span class="site-service-card__icon"><component :is="option.icon" :size="20" /></span>
              <span class="site-service-card__content">
                <strong>{{ option.title }}</strong>
                <small>{{ option.summary }}</small>
                <em>{{ option.detail }}</em>
              </span>
            </button>
          </div>
          <small v-if="editingSite">服务类型保持不变，避免遗留目录或意外改变现有流量路径。</small>
        </fieldset>

        <div v-if="selectedService" class="site-service-summary">
          <component :is="selectedService.icon" :size="18" />
          <span><strong>{{ selectedService.title }}</strong>{{ selectedService.summary }}</span>
        </div>

        <fieldset v-if="form.type === 'php'" class="field site-inline-options">
          <legend>PHP 运行环境</legend>
          <div class="choice-pills">
            <button
              type="button"
              :class="{ 'is-active': form.phpVersion === 'latest' }"
              :aria-pressed="form.phpVersion === 'latest'"
              @click="form.phpVersion = 'latest'"
            >
              PHP 最新版
            </button>
            <button
              type="button"
              :class="{ 'is-active': form.phpVersion === '7.4' }"
              :aria-pressed="form.phpVersion === '7.4'"
              @click="form.phpVersion = '7.4'"
            >
              PHP 7.4
            </button>
          </div>
          <small>分别对应脚本架构中的 php 与 php74 PHP-FPM 服务。</small>
        </fieldset>

        <label v-if="form.type === 'proxy' || form.type === 'proxy_domain'" class="field">
          <span>上游地址</span>
          <input
            v-model.trim="form.upstream"
            type="url"
            :placeholder="form.type === 'proxy_domain' ? 'https://origin.example.com' : 'http://127.0.0.1:3000'"
            required
          />
          <small v-if="form.type === 'proxy'">仅允许本机、内网 IP 或 Docker 服务名，阻止意外代理公网地址。</small>
          <small v-else>填写完整域名源站，HTTPS 会自动启用上游 SNI；不接受路径、账号或查询参数。</small>
        </label>

        <label v-if="form.type === 'load_balance'" class="field">
          <span>后端节点</span>
          <textarea
            v-model="form.upstreams"
            rows="4"
            placeholder="http://10.0.0.11:8080&#10;http://10.0.0.12:8080"
            required
          />
          <small>每行一个 HTTP 源站，2–8 个；与 kejilion.sh 的 HTTP upstream 架构一致。</small>
        </label>

        <template v-if="form.type === 'redirect'">
          <label class="field">
            <span>跳转目标</span>
            <input v-model.trim="form.redirectTarget" type="url" placeholder="https://www.example.com" required />
            <small>访问路径与查询参数会原样追加到目标域名。</small>
          </label>
          <fieldset class="field site-inline-options">
            <legend>跳转方式</legend>
            <div class="choice-pills choice-pills--four">
              <button
                v-for="code in ([301, 302, 307, 308] as RedirectCode[])"
                :key="code"
                type="button"
                :class="{ 'is-active': form.redirectCode === code }"
                :aria-pressed="form.redirectCode === code"
                @click="form.redirectCode = code"
              >
                {{ code }}<small>{{ code === 301 || code === 308 ? '永久' : '临时' }}</small>
              </button>
            </div>
          </fieldset>
        </template>

        <label class="field">
          <span>附加域名（可选）</span>
          <textarea v-model="form.aliases" rows="3" placeholder="www.example.com&#10;api.example.com" />
          <small>每行一个域名，最多 20 个；主域名不要重复填写。</small>
        </label>
        <div class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          Agent 会写入 /home/web 对应产物并原子提交；脚本或人工配置保持只读。创建不会删除目录、数据库或证书，也不会为签发证书停止现有 Nginx。
        </div>
      </form>
      <template #footer>
        <button class="button button--secondary" type="button" @click="editorOpen = false">取消</button>
        <button class="button button--primary" type="submit" form="site-form" :disabled="submitting || !formValid">
          <LoaderCircle v-if="submitting" class="spin" :size="16" />
          {{ submitting ? '正在提交…' : editingSite ? '安全更新' : '创建网站' }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>
