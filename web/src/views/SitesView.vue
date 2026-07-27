<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  ArrowRight,
  Braces,
  ExternalLink,
  FileCode2,
  Flame,
  Globe2,
  KeyRound,
  LoaderCircle,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Trash2,
  TriangleAlert,
  Waypoints,
} from '@lucide/vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import AppInteractiveTerminal from '@/components/apps/AppInteractiveTerminal.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/feedback/StatusBadge.vue'
import { ApiError, api } from '@/lib/api'
import { formatDateTime, relativeTime, shortId } from '@/lib/format'
import { usePanelState } from '@/stores/panel'
import { useToast } from '@/stores/toast'
import type { Site, SiteInput, SiteInstallationProgress } from '@/types/api'

type Filter = 'all' | 'healthy' | 'drifted' | 'config-only'
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
const installProgress = ref<SiteInstallationProgress>()
const deleteOpen = ref(false)
const deletingSite = ref<Site>()
const deleteMode = ref<'configuration' | 'full'>('configuration')
const deleteError = ref('')
const deleting = ref(false)
const panel = usePanelState()
const toast = useToast()
let controller: AbortController | undefined

const installStageLabels: Record<string, string> = {
  submitting: '提交配置',
  queued: '等待执行',
  preflight: '环境校验',
  source: '获取源码',
  files: '准备文件',
  database: '创建数据库',
  configure: '写入配置',
  nginx_bootstrap: '临时入口',
  certificate: '签发证书',
  installing: '执行脚本',
  publish: '发布站点',
  activate: '激活服务',
  reconciling: '产物对账',
  reconcile: '产物对账',
  completed: '搭建完成',
  interrupted: '任务中断',
  script_unavailable: '脚本不可用',
  runner_unavailable: '后台执行器不可用',
  start_failed: '任务启动失败',
  reconcile_failed: '产物对账失败',
  failed: '搭建失败',
}

function installStageName(stage?: string): string {
  return installStageLabels[stage || ''] || '正在执行'
}

const installStageLabel = computed(() => installStageName(installProgress.value?.stage))

const serviceOptions = [
  {
    type: 'wordpress',
    title: 'WordPress',
    summary: '博客、企业官网与内容站一键成型',
    detail: '完整执行 kejilion.sh 同款建站流程',
    icon: Globe2,
    featured: true,
    badges: ['热门', '一键成品'],
  },
  {
    type: 'proxy',
    title: 'IP / 端口反代',
    summary: '代理本机、内网或 Docker 服务',
    detail: '例如 127.0.0.1:3000',
    icon: Server,
    featured: true,
    badges: ['热门'],
  },
  {
    type: 'static',
    title: '静态网站',
    summary: 'HTML、图片与前端构建产物',
    detail: '创建独立站点目录与默认首页',
    icon: FileCode2,
    featured: false,
    badges: [],
  },
  {
    type: 'php',
    title: 'PHP 网站',
    summary: '动态网站与自建 PHP 程序',
    detail: '使用 kejilion.sh 同款 PHP-FPM Socket',
    icon: Braces,
    featured: false,
    badges: [],
  },
  {
    type: 'proxy_domain',
    title: '域名反代',
    summary: '代理另一域名提供的 HTTPS 服务',
    detail: '自动配置上游 SNI',
    icon: Globe2,
    featured: false,
    badges: [],
  },
  {
    type: 'load_balance',
    title: '负载均衡',
    summary: '将请求分配到多个后端节点',
    detail: '支持 2–8 个 HTTP 上游',
    icon: Waypoints,
    featured: false,
    badges: [],
  },
  {
    type: 'redirect',
    title: '域名重定向',
    summary: '将访问跳转到另一个域名',
    detail: '支持 301、302、307、308',
    icon: ArrowRight,
    featured: false,
    badges: [],
  },
] as const satisfies ReadonlyArray<{
  type: SiteServiceType
  title: string
  summary: string
  detail: string
  icon: typeof FileCode2
  featured: boolean
  badges: readonly string[]
}>

const recipeOptions = [
  { recipe: 'discuz', title: 'Discuz 论坛', summary: '成熟中文社区论坛', detail: '脚本菜单 3', icon: Globe2 },
  { recipe: 'kodbox', title: '可道云 Kodbox', summary: '私有云盘与在线桌面', detail: '脚本菜单 4', icon: FileCode2 },
  { recipe: 'maccms', title: '苹果 CMS', summary: '影视内容管理系统', detail: '脚本菜单 5', icon: Globe2 },
  { recipe: 'dujiaoka', title: '独角数卡', summary: '数字商品发卡商城', detail: '脚本菜单 6', icon: Braces },
  { recipe: 'flarum', title: 'Flarum', summary: '现代轻量论坛', detail: '脚本菜单 7', icon: Globe2 },
  { recipe: 'typecho', title: 'Typecho', summary: '轻量博客系统', detail: '脚本菜单 8', icon: Braces },
  { recipe: 'linkstack', title: 'LinkStack', summary: '共享链接主页', detail: '脚本菜单 9', icon: Waypoints },
  { recipe: 'ai-prompt', title: 'AI 提示词生成器', summary: '脚本原生静态成品站', detail: '脚本菜单 27', icon: FileCode2 },
] as const satisfies ReadonlyArray<{
  recipe: NonNullable<SiteInput['recipe']>
  title: string
  summary: string
  detail: string
  icon: typeof FileCode2
}>

const form = reactive({
  primaryDomain: '',
  aliases: '',
  type: 'wordpress' as SiteServiceType,
  recipe: 'discuz' as NonNullable<SiteInput['recipe']>,
  upstream: '',
  upstreams: '',
  redirectTarget: '',
  redirectCode: 301 as RedirectCode,
  phpVersion: 'latest' as PHPVersion,
})

const siteWriteCapability = computed(() => capabilities.value.find((capability) => capability.id === 'sites.write'))
const wordPressCapability = computed(() =>
  capabilities.value.find((capability) => capability.id === 'sites.wordpress.install'),
)
const proxyCapability = computed(() =>
  capabilities.value.find((capability) => capability.id === 'sites.proxy.install'),
)
const recipeCapability = computed(() =>
  capabilities.value.find((capability) => capability.id === 'sites.recipes.install'),
)
const canCreate = computed(() => siteWriteCapability.value?.enabled === true)
const canInstallWordPress = computed(() => wordPressCapability.value?.enabled === true)
const canInstallProxy = computed(() => proxyCapability.value?.enabled === true)
const canInstallRecipes = computed(() => recipeCapability.value?.enabled === true)
const canCreateAny = computed(
  () => canCreate.value || canInstallWordPress.value || canInstallProxy.value || canInstallRecipes.value,
)
const wordPressReason = computed(
  () => wordPressCapability.value?.reason?.trim() || 'WordPress 一键搭建依赖尚未就绪。',
)
const siteWriteReason = computed(
  () =>
    siteWriteCapability.value?.reason?.trim() ||
    (siteWriteCapability.value
      ? 'Agent 当前缺少网站写入依赖。'
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
    if (filter.value === 'config-only') return !site.allowedActions?.includes('update')
    return true
  })
})

const counts = computed(() => ({
  all: sites.value.length,
  healthy: sites.value.filter((site) => site.health === 'healthy' && site.consistency === 'synced').length,
  drifted: sites.value.filter((site) => site.consistency !== 'synced').length,
  'config-only': sites.value.filter((site) => !site.allowedActions?.includes('update')).length,
}))

const selectedService = computed(() => serviceOptions.find((option) => option.type === form.type))
const selectedRecipe = computed(() => recipeOptions.find((option) => option.recipe === form.recipe))
const featuredServiceOptions = computed(() => serviceOptions.filter((option) => option.featured))
const standardServiceOptions = computed(() => serviceOptions.filter((option) => !option.featured))
const canSubmit = computed(
  () =>
    formValid.value &&
    (form.type !== 'wordpress' || canInstallWordPress.value) &&
    (form.type !== 'proxy' || canInstallProxy.value) &&
    (form.type !== 'recipe' || canInstallRecipes.value) &&
    (form.type === 'wordpress' || form.type === 'proxy' || form.type === 'recipe' || canCreate.value),
)

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
    const capabilityPromise = api.agent.capabilities(controller.signal).catch(() => [])
    sites.value = (await api.sites.list(undefined, controller.signal)).items
    loading.value = false
    capabilities.value = await capabilityPromise
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
  form.type = canInstallWordPress.value
    ? 'wordpress'
    : canInstallProxy.value
      ? 'proxy'
      : canCreate.value
        ? 'static'
        : 'recipe'
  form.recipe = 'discuz'
  form.upstream = ''
  form.upstreams = ''
  form.redirectTarget = ''
  form.redirectCode = 301
  form.phpVersion = 'latest'
  formError.value = ''
  installProgress.value = undefined
  editorOpen.value = true
}

function openEdit(site: Site): void {
  if (!serviceOptions.some((option) => option.type === site.type)) return
  editingSite.value = site
  form.primaryDomain = site.primaryDomain
  form.aliases = site.domains.filter((domain) => domain !== site.primaryDomain).join('\n')
  form.type = site.type as SiteServiceType
  form.recipe = 'discuz'
  form.upstream = site.type === 'proxy' || site.type === 'proxy_domain' ? site.upstream || '' : ''
  form.upstreams = site.type === 'load_balance' ? (site.upstream || '').split(',').join('\n') : ''
  const redirectMatch = site.type === 'redirect' ? (site.upstream || '').match(/^(301|302|307|308)\s+(.+)$/) : null
  form.redirectCode = redirectMatch ? (Number(redirectMatch[1]) as RedirectCode) : 301
  form.redirectTarget = redirectMatch?.[2] || ''
  form.phpVersion = site.type === 'php' && site.upstream === 'php74' ? '7.4' : 'latest'
  formError.value = ''
  installProgress.value = undefined
  selectedSite.value = undefined
  editorOpen.value = true
}

async function submitSite(): Promise<void> {
  formError.value = ''
  if (!formValid.value) {
    formError.value = '请检查域名和当前服务所需的配置。'
    return
  }
  if (form.type === 'wordpress' && !canInstallWordPress.value) {
    formError.value = wordPressReason.value
    return
  }
  if (form.type === 'proxy' && !canInstallProxy.value) {
    formError.value = proxyCapability.value?.reason || '当前 Agent 无法调用 kejilion.sh IP+端口反向代理命令。'
    return
  }
  if (form.type === 'recipe' && !canInstallRecipes.value) {
    formError.value = recipeCapability.value?.reason || '当前 Agent 尚未启用 kejilion.sh 一键建站协议。'
    return
  }

  submitting.value = true
  installProgress.value = {
    status: 'running',
    stage: 'submitting',
    progress: 2,
    message: editingSite.value ? '正在提交网站设置。' : '正在提交建站配置并检查现有产物。',
  }
  const input: SiteInput = {
    primaryDomain: form.primaryDomain.trim().toLowerCase(),
    aliases: form.type === 'wordpress' || form.type === 'proxy' || form.type === 'recipe' ? [] : form.aliases
      .split(/[\n,]/)
      .map((value) => value.trim().toLowerCase())
      .filter(Boolean),
    type: form.type,
    recipe: form.type === 'recipe' ? form.recipe : undefined,
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
      : await api.sites.create(input, (progress) => {
          installProgress.value = progress
        })
    editorOpen.value = false
    toast.success(
      wasEditing
        ? '网站已更新'
        : form.type === 'wordpress' || form.type === 'proxy' || form.type === 'recipe'
          ? '一键建站已完成'
          : '网站已创建',
      form.type === 'wordpress' || form.type === 'proxy' || form.type === 'recipe'
        ? `${savedSite.primaryDomain} 的原生建站产物已与 kejilion.sh 完成对账。`
        : `${savedSite.primaryDomain} 已通过 nginx -t 校验并完成同步应用。`,
    )
    await load(true)
  } catch (reason) {
    const message = reason instanceof ApiError ? reason.message : '操作失败，请稍后重试。'
    formError.value = message
    const current = installProgress.value
    if (current && current.status !== 'failed') {
      installProgress.value = {
        ...current,
        status: 'failed',
        stage: 'failed',
        message,
        events: [
          ...(current.events || []),
          {
            stage: 'failed',
            progress: current.progress,
            message,
            at: new Date().toISOString(),
          },
        ],
      }
    }
  } finally {
    submitting.value = false
  }
}

function openDelete(site: Site): void {
  deletingSite.value = site
  deleteMode.value = 'configuration'
  deleteError.value = ''
  selectedSite.value = undefined
  deleteOpen.value = true
}

async function deleteSite(): Promise<void> {
  const site = deletingSite.value
  if (!site || deleting.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    let resourceVersion = site.resourceVersion
    if (!/^sha256:[a-f0-9]{64}$/.test(resourceVersion)) {
      const refreshed = await api.sites.list()
      const current = refreshed.items.find((item) => item.id === site.id)
      resourceVersion = current?.resourceVersion || ''
    }
    if (!/^sha256:[a-f0-9]{64}$/.test(resourceVersion)) {
      throw new ApiError('无法读取站点当前版本，请刷新页面后重试。', 422, 'site_version_unavailable')
    }
    const result = await api.sites.remove(
      site.id,
      resourceVersion,
      deleteMode.value,
    )
    deleteOpen.value = false
    deletingSite.value = undefined
    toast.success(
      deleteMode.value === 'full' ? '站点数据已删除' : '网站配置已移除',
      (deleteMode.value === 'full'
        ? `${result.primaryDomain} 已按 k web 业务清理。`
        : `${result.primaryDomain} 的 Nginx 访问配置已移除，网站目录、证书和数据库均已保留。`) +
        (result.warnings?.length ? ` ${result.warnings.join('；')}` : ''),
    )
    await load(true)
  } catch (reason) {
    deleteError.value = reason instanceof ApiError ? reason.message : '删除失败，原网站产物已尽可能恢复。'
  } finally {
    deleting.value = false
  }
}

watch(editorOpen, (open) => {
  if (!open) {
    formError.value = ''
    installProgress.value = undefined
  }
})

function sitePublicURL(site: Site): string {
  const certificateStatus = site.certificate?.status
  const protocol = certificateStatus && !['missing', 'unknown'].includes(certificateStatus) ? 'https' : 'http'
  return `${protocol}://${site.primaryDomain}`
}

onMounted(() => void load())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div class="page">
    <PageHeader
      title="网站管理"
      description="从实际产物发现并管理网站；新建站点沿用 kejilion.sh 的 /home/web 架构，并通过原子事务提交。"
    >
      <template #actions>
        <button
          class="button button--primary"
          type="button"
          :disabled="!canCreateAny || panel.isReadOnly.value"
          :title="!canCreateAny ? siteWriteReason : ''"
          @click="openCreate"
        >
          <Plus :size="17" /> 新建网站
        </button>
      </template>
    </PageHeader>

    <div v-if="!canCreateAny && !loading" class="inline-alert inline-alert--info" role="status">
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
            { key: 'config-only', label: '仅配置操作' },
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
      :description="sites.length ? '尝试更换搜索词或筛选条件。' : 'Agent 会扫描现有 Kejilion 网站产物。'"
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
                <a
                  class="resource-name resource-name--link"
                  :href="sitePublicURL(site)"
                  target="_blank"
                  rel="noopener noreferrer"
                  :title="`访问 ${site.primaryDomain}`"
                >
                  <span class="resource-name__icon"><Globe2 :size="18" /></span>
                  <span>
                    <strong>{{ site.primaryDomain }} <ExternalLink :size="11" /></strong>
                    <small>{{ site.enabled ? '已启用' : '已停用' }} · {{ site.domains.length }} 个域名</small>
                  </span>
                </a>
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
                  <small>{{ site.allowedActions?.length ? '可直接管理' : '仅展示产物' }}</small>
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
                <button
                  v-if="site.allowedActions?.includes('delete')"
                  class="button button--ghost button--small button--danger-text"
                  type="button"
                  :disabled="panel.isReadOnly.value || !canCreate"
                  :title="!canCreate ? siteWriteReason : ''"
                  @click="openDelete(site)"
                >
                  删除
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
        <a
          v-if="selectedSite"
          class="button button--secondary"
          :href="sitePublicURL(selectedSite)"
          target="_blank"
          rel="noopener noreferrer"
        >
          <ExternalLink :size="16" /> 访问网站
        </a>
        <button
          v-if="selectedSite?.allowedActions?.includes('delete')"
          class="button button--ghost button--danger-text"
          type="button"
          :disabled="panel.isReadOnly.value || !canCreate"
          :title="!canCreate ? siteWriteReason : ''"
          @click="openDelete(selectedSite)"
        >
          <Trash2 :size="16" /> 删除站点
        </button>
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
      :open="deleteOpen && Boolean(deletingSite)"
      :title="`删除 ${deletingSite?.primaryDomain || ''}`"
      description="按当前实际配置文件执行；提交后先撤下配置，通过 nginx -t 后再 reload，失败自动恢复。"
      size="small"
      @close="!deleting && (deleteOpen = false)"
    >
      <form id="site-delete-form" class="form-stack" @submit.prevent="deleteSite">
        <div v-if="deleteError" class="inline-alert inline-alert--danger" role="alert">{{ deleteError }}</div>

        <fieldset class="delete-mode-grid">
          <legend>删除范围</legend>
          <button
            type="button"
            :class="{ 'is-active': deleteMode === 'configuration' }"
            @click="deleteMode = 'configuration'"
          >
            <strong>仅移除网站配置</strong>
            <small>删除 Nginx 入口；保留网站目录、证书和数据库，便于重新绑定。</small>
          </button>
          <button
            type="button"
            :class="{ 'is-active': deleteMode === 'full' }"
            @click="deleteMode = 'full'"
          >
            <strong>按 k web 完整删除</strong>
            <small>同时删除该域名的网站目录、证书和同名数据库（若存在），与 k web 删除产物一致。</small>
          </button>
        </fieldset>

        <div v-if="deleteMode === 'full'" class="inline-alert inline-alert--danger">
          <TriangleAlert :size="17" />
          <span>完整删除不可从面板撤销。请先确认网站数据已有独立备份。</span>
        </div>

      </form>
      <template #footer>
        <button class="button button--secondary" type="button" :disabled="deleting" @click="deleteOpen = false">
          取消
        </button>
        <button
          class="button button--danger"
          type="submit"
          form="site-delete-form"
          :disabled="deleting"
        >
          <LoaderCircle v-if="deleting" class="spin" :size="16" />
          <Trash2 v-else :size="16" />
          {{ deleting ? '正在删除…' : deleteMode === 'full' ? '完整删除站点' : '移除网站配置' }}
        </button>
      </template>
    </ModalDialog>

    <ModalDialog
      :open="editorOpen"
      :title="editingSite ? '编辑网站设置' : '新建网站'"
      description="按 kejilion.sh 的站点架构生成固定配置；先通过 nginx -t，成功后才 reload。"
      size="large"
      @close="!submitting && (editorOpen = false)"
    >
      <form id="site-form" class="form-stack" @submit.prevent="submitSite">
        <div v-if="formError" class="inline-alert inline-alert--danger" role="alert">{{ formError }}</div>
        <div
          v-if="installProgress && (submitting || installProgress.status === 'failed')"
          class="site-install-progress"
          :class="{ 'is-failed': installProgress.status === 'failed' }"
          role="status"
          aria-live="polite"
        >
          <div class="site-install-progress__heading">
            <span>
              <LoaderCircle v-if="submitting" class="spin" :size="17" />
              <TriangleAlert v-else :size="17" />
              {{ installStageLabel }}
            </span>
            <strong>{{ installProgress.progress }}%</strong>
          </div>
          <div
            class="site-install-progress__track"
            role="progressbar"
            :aria-valuenow="installProgress.progress"
            aria-valuemin="0"
            aria-valuemax="100"
          >
            <span :style="{ width: `${installProgress.progress}%` }"></span>
          </div>
          <p>{{ installProgress.message }}</p>
          <ol v-if="installProgress.events?.length" class="site-install-progress__events">
            <li
              v-for="(event, index) in installProgress.events"
              :key="`${event.at}-${index}`"
              :class="{ 'is-current': index === installProgress.events.length - 1 }"
            >
              <span>{{ event.progress }}%</span>
              <div>
                <strong>{{ installStageName(event.stage) }}</strong>
                <small>{{ event.message }}</small>
              </div>
            </li>
          </ol>
          <small class="site-install-progress__privacy">
            上方时间线仅展示安全进度事件；原生脚本输出和后续交互请查看下方受登录保护的终端。
          </small>
          <AppInteractiveTerminal
            v-if="installProgress.interactive && installProgress.id"
            :job-id="installProgress.id"
            :input-open="installProgress.inputOpen"
            kind="site"
          />
        </div>
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

        <fieldset v-if="!editingSite" class="site-service-field site-service-field--featured">
          <legend><Flame :size="16" /> 热门搭建</legend>
          <div class="site-service-grid">
            <button
              v-for="option in featuredServiceOptions"
              :key="option.type"
              class="site-service-card"
              :class="{ 'is-active': form.type === option.type }"
              type="button"
              :disabled="
                (option.type === 'wordpress' && !canInstallWordPress) ||
                (option.type === 'proxy' && !canInstallProxy)
              "
              :aria-pressed="form.type === option.type"
              :title="
                option.type === 'wordpress' && !canInstallWordPress
                  ? wordPressReason
                  : option.type === 'proxy' && !canInstallProxy
                    ? proxyCapability?.reason
                    : ''
              "
              @click="form.type = option.type"
            >
              <span class="site-service-card__icon"><component :is="option.icon" :size="20" /></span>
              <span class="site-service-card__content">
                <span class="site-service-card__heading">
                  <strong>{{ option.title }}</strong>
                  <span
                    v-for="badge in option.badges"
                    :key="badge"
                    class="site-service-card__badge"
                    :class="{ 'is-hot': badge === '热门' }"
                  >
                    {{ badge }}
                  </span>
                </span>
                <small>{{ option.summary }}</small>
                <em>{{ option.detail }}</em>
              </span>
            </button>
          </div>
          <small>WordPress 对齐脚本完整产物；IP + 端口反代适合 Docker 与本机服务，是最常用的两个入口。</small>
        </fieldset>

        <fieldset v-if="!editingSite" class="site-service-field">
          <legend><Globe2 :size="16" /> kejilion.sh 一键成品站</legend>
          <div class="site-service-grid">
            <button
              v-for="option in recipeOptions"
              :key="option.recipe"
              class="site-service-card"
              :class="{ 'is-active': form.type === 'recipe' && form.recipe === option.recipe }"
              type="button"
              :disabled="!canInstallRecipes"
              :aria-pressed="form.type === 'recipe' && form.recipe === option.recipe"
              :title="!canInstallRecipes ? recipeCapability?.reason : ''"
              @click="form.type = 'recipe'; form.recipe = option.recipe"
            >
              <span class="site-service-card__icon"><component :is="option.icon" :size="20" /></span>
              <span class="site-service-card__content">
                <span class="site-service-card__heading">
                  <strong>{{ option.title }}</strong>
                  <span class="site-service-card__badge">一键成品</span>
                </span>
                <small>{{ option.summary }}</small>
                <em>{{ option.detail }}</em>
              </span>
            </button>
          </div>
          <small>直接调用 kejilion.sh 固定编号的原生业务流程，Web 不复制另一套建站逻辑，因此脚本与面板共用同一份产物。</small>
        </fieldset>

        <fieldset class="site-service-field">
          <legend>{{ editingSite ? '站点服务' : '更多建站方式' }}</legend>
          <div class="site-service-grid">
            <button
              v-for="option in editingSite ? serviceOptions : standardServiceOptions"
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
        <div v-if="form.type === 'recipe' && selectedRecipe" class="site-service-summary">
          <component :is="selectedRecipe.icon" :size="18" />
          <span><strong>{{ selectedRecipe.title }}</strong>{{ selectedRecipe.summary }}</span>
        </div>

        <div v-if="form.type === 'wordpress'" class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          <span>
            后台执行 kejilion.sh 的 WordPress 固定非交互分支，由脚本原生流程完成 LDNMP、数据库、
            证书、官方 Nginx 模板和 WordPress 源码部署；KPanel 只负责显示进度并对账最终产物。
          </span>
        </div>
        <div v-if="form.type === 'proxy' && !editingSite" class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          <span>
            后台执行 kejilion.sh 的 IP+端口反向代理固定非交互分支，官方反向代理模板、证书、
            Nginx 重载及端口访问控制均沿用脚本原生流程。
          </span>
        </div>
        <div v-if="form.type === 'recipe'" class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          <span>
            任务由宿主机上的 <code>kejilion.sh</code> 原生菜单流程在后台执行，沿用其 LDNMP、数据库、证书、
            Nginx 模板和目录结构。面板不会记录脚本输出中的数据库密码，也不会覆盖同域名现有产物。
          </span>
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
          <small v-if="form.type === 'proxy'">支持本机、内网、公网 IP、域名或 Docker 服务名，与 k web 的 IP+端口反代一致。</small>
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

        <label v-if="form.type !== 'wordpress' && form.type !== 'proxy' && form.type !== 'recipe'" class="field">
          <span>附加域名（可选）</span>
          <textarea v-model="form.aliases" rows="3" placeholder="www.example.com&#10;api.example.com" />
          <small>每行一个域名，最多 20 个；主域名不要重复填写。</small>
        </label>
        <div class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          Agent 直接对账 /home/web 实际产物；kejilion.sh、KPanel 或手工调整后的站点都在同一份配置上继续管理。
          无法解析的网站仍可移除 Nginx 配置入口。
        </div>
      </form>
      <template #footer>
        <button class="button button--secondary" type="button" :disabled="submitting" @click="editorOpen = false">取消</button>
        <button class="button button--primary" type="submit" form="site-form" :disabled="submitting || !canSubmit">
          <LoaderCircle v-if="submitting" class="spin" :size="16" />
          {{
            submitting
              ? form.type === 'wordpress'
                ? '正在搭建 WordPress…'
                : form.type === 'proxy' || form.type === 'recipe'
                  ? 'kejilion.sh 正在后台搭建…'
                : '正在提交…'
              : editingSite
                ? '更新设置'
                : form.type === 'wordpress'
                  ? '一键搭建 WordPress'
                  : form.type === 'proxy'
                    ? '一键创建反向代理'
                  : form.type === 'recipe'
                    ? `一键搭建 ${selectedRecipe?.title || '成品站'}`
                  : '创建网站'
          }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>
