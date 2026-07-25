<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  Braces,
  FileCode2,
  Globe2,
  KeyRound,
  LoaderCircle,
  Plus,
  RefreshCw,
  Search,
  ShieldCheck,
  TriangleAlert,
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

const form = reactive({
  primaryDomain: '',
  aliases: '',
  type: 'static' as 'static' | 'proxy',
  upstream: '',
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

const formValid = computed(() => {
  const domain = form.primaryDomain.trim()
  const validDomain = domain.includes('.') && !domain.includes('/') && !domain.includes(' ')
  const validUpstream = form.type === 'static' || /^https?:\/\/[^ ]+$/i.test(form.upstream.trim())
  return validDomain && validUpstream
})

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
    proxy: '反向代理',
    php: 'PHP',
    redirect: '重定向',
    unknown: '未知类型',
  }[type]
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (silent) refreshing.value = true
  else loading.value = true
  error.value = ''

  try {
    const [siteResult, capabilityResult] = await Promise.all([
      api.sites.list(undefined, controller.signal),
      api.agent.capabilities(controller.signal).catch(() => []),
    ])
    sites.value = siteResult.items
    capabilities.value = capabilityResult
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
  formError.value = ''
  editorOpen.value = true
}

function openEdit(site: Site): void {
  editingSite.value = site
  form.primaryDomain = site.primaryDomain
  form.aliases = site.domains.filter((domain) => domain !== site.primaryDomain).join('\n')
  form.type = site.type === 'proxy' ? 'proxy' : 'static'
  form.upstream = site.upstream || ''
  formError.value = ''
  selectedSite.value = undefined
  editorOpen.value = true
}

async function submitSite(): Promise<void> {
  formError.value = ''
  if (!formValid.value) {
    formError.value = '请检查域名与上游地址。'
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
    upstream: form.type === 'proxy' ? form.upstream.trim() : undefined,
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
      description="从 Nginx 配置、站点目录与证书实际产物中发现；安全写入同步执行，不接管未知资源。"
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
                  <small :title="site.upstream || site.rootPath">
                    {{ site.upstream || site.rootPath || '未识别目标' }}
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
            <dt>{{ selectedSite.type === 'proxy' ? '上游地址' : '站点目录' }}</dt>
            <dd><code>{{ selectedSite.upstream || selectedSite.rootPath || '—' }}</code></dd>
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
      description="提交后同步执行安全事务：校验资源版本与 nginx -t，失败自动回滚，成功后 reload。"
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
        <label class="field">
          <span>网站类型</span>
          <select v-model="form.type" :disabled="Boolean(editingSite)">
            <option value="static">静态网站</option>
            <option value="proxy">反向代理</option>
          </select>
          <small v-if="editingSite">首版不在静态站与反向代理之间转换，避免遗留或删除业务文件。</small>
        </label>
        <label v-if="form.type === 'proxy'" class="field">
          <span>上游地址</span>
          <input v-model.trim="form.upstream" type="url" placeholder="http://127.0.0.1:3000" required />
          <small>仅允许 HTTP 或 HTTPS 上游，Agent 会继续执行 SSRF 与端口校验。</small>
        </label>
        <label class="field">
          <span>附加域名（可选）</span>
          <textarea v-model="form.aliases" rows="3" placeholder="www.example.com&#10;api.example.com" />
        </label>
        <div class="inline-alert inline-alert--info">
          <ShieldCheck :size="17" />
          Agent 会锁定资源并原子写入；仅更新由 Panel 固定模板创建且未被外部修改的网站。脚本或人工配置保持只读，首版不删除网站、目录、数据库或证书。
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
