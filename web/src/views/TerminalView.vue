<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Circle, Laptop, LoaderCircle, Plus, RefreshCw, Server, ShieldCheck, SquareTerminal, X } from '@lucide/vue'
import HostTerminal from '@/components/terminal/HostTerminal.vue'
import { api, ApiError } from '@/lib/api'
import type { ClusterHost, ClusterHostList } from '@/types/api'
import { usePhraseCatalog } from '@/i18n/phrase'
import { useI18n } from '@/i18n'

usePhraseCatalog(() => import('@/i18n/pages/TerminalView/en-US').then((module) => module.default))
const { t } = useI18n()

interface OpenTerminal {
  id: string
  hostId: string
  hostName: string
  offset: number
}

const inventory = ref<ClusterHostList>()
const sessions = ref<OpenTerminal[]>([])
const activeSessionId = ref('')
const loading = ref(true)
const openingHostId = ref('')
const errorMessage = ref('')
const search = ref('')
let controller: AbortController | undefined

const hosts = computed(() => {
  const needle = search.value.trim().toLowerCase()
  return (inventory.value?.items || []).filter((host) => !needle || `${host.name} ${host.origin}`.toLowerCase().includes(needle))
})

const activeSession = computed(() => sessions.value.find((item) => item.id === activeSessionId.value))

async function loadHosts(): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  errorMessage.value = ''
  try {
    inventory.value = await api.cluster.hosts(controller.signal)
  } catch {
    errorMessage.value = '连接列表加载失败，请检查 Agent 与集群状态。'
  } finally {
    loading.value = false
  }
}

async function openHost(host: ClusterHost): Promise<void> {
  const existing = sessions.value.find((item) => item.hostId === host.id)
  if (existing) {
    activeSessionId.value = existing.id
    return
  }
  if (!host.terminalAvailable || openingHostId.value) return
  openingHostId.value = host.id
  errorMessage.value = ''
  try {
    const opened = await api.terminals.open(host.id, 30, 120)
    const item = { id: opened.sessionId, hostId: host.id, hostName: host.name, offset: opened.offset }
    sessions.value.push(item)
    activeSessionId.value = item.id
  } catch (reason) {
    errorMessage.value = reason instanceof ApiError && reason.code === 'terminal_limit'
      ? '已达到终端会话上限，请先关闭不用的连接。'
      : '终端连接失败，请确认目标 KPanel 在线且双方均已更新。'
  } finally {
    openingHostId.value = ''
  }
}

function removeSession(id: string): void {
  const index = sessions.value.findIndex((item) => item.id === id)
  if (index < 0) return
  sessions.value.splice(index, 1)
  if (activeSessionId.value === id) activeSessionId.value = sessions.value[Math.max(0, index - 1)]?.id || ''
}

function hostStateLabel(host: ClusterHost): string {
  if (!host.terminalAvailable) return host.kind === 'light_node' ? '轻量监控节点' : '需要重新配对'
  if (host.isLocal) return '本机终端'
  return '加密直连'
}

onMounted(() => void loadHosts())
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div class="page terminal-page">
    <header class="page-heading terminal-heading">
      <div>
        <span class="eyebrow"><SquareTerminal :size="16" /> KPanel Terminal</span>
        <h1>多主机终端</h1>
        <p>本机与已配对 KPanel 使用同一登录态进入；远程流量沿集群加密通道传输，不开放新的 SSH 或公网端口。</p>
      </div>
      <button class="button button--secondary" type="button" :disabled="loading" @click="loadHosts"><RefreshCw :size="17" :class="{ spin: loading }" /> {{ t('terminal.refreshConnections') }}</button>
    </header>

    <div v-if="errorMessage" class="terminal-alert" role="alert">{{ errorMessage }}</div>

    <section class="terminal-workspace">
      <aside class="terminal-connections">
        <header>
          <div><strong>连接列表</strong><small>{{ hosts.length }} 台主机</small></div>
          <ShieldCheck :size="18" />
        </header>
        <label class="terminal-search"><input v-model="search" type="search" placeholder="搜索主机" /></label>
        <div class="terminal-connections__list">
          <div v-if="loading" class="terminal-connections__empty"><LoaderCircle class="spin" :size="22" /> {{ t('terminal.loadingHosts') }}</div>
          <div v-else-if="!hosts.length" class="terminal-connections__empty">暂无可显示主机</div>
          <button v-for="host in hosts" :key="host.id" class="terminal-host" :class="{ 'is-active': activeSession?.hostId === host.id }" type="button" :disabled="openingHostId === host.id" @click="openHost(host)">
            <span class="terminal-host__icon"><Laptop v-if="host.isLocal" :size="19" /><Server v-else :size="19" /></span>
            <span><strong>{{ host.name }}</strong><small>{{ host.origin || t('terminal.currentPanel') }}</small><em :class="{ 'is-ready': host.terminalAvailable }"><Circle :size="8" fill="currentColor" /> {{ hostStateLabel(host) }}</em></span>
            <LoaderCircle v-if="openingHostId === host.id" class="spin" :size="17" />
            <Plus v-else-if="host.terminalAvailable && !sessions.some((item) => item.hostId === host.id)" :size="17" />
          </button>
        </div>
      </aside>

      <main class="terminal-stage">
        <nav v-if="sessions.length" class="terminal-tabs" aria-label="已打开终端">
          <button v-for="item in sessions" :key="item.id" type="button" :class="{ 'is-active': item.id === activeSessionId }" @click="activeSessionId = item.id">
            <SquareTerminal :size="14" /><span>{{ item.hostName }}</span>
            <X :size="14" @click.stop="removeSession(item.id)" />
          </button>
        </nav>
        <div v-if="!sessions.length" class="terminal-empty"><span><SquareTerminal :size="34" /></span><h2>选择一台主机开始</h2><p>左侧会明确标记本机、可加密直连的 KPanel，以及仅提供监控的轻量节点。</p></div>
        <HostTerminal v-for="item in sessions" v-show="item.id === activeSessionId" :key="item.id" :session-id="item.id" :host-name="item.hostName" :initial-offset="item.offset" @close="removeSession(item.id)" />
      </main>
    </section>
  </div>
</template>

<style scoped>
.terminal-page { min-height:calc(100vh - 100px); gap:18px; }
.terminal-heading { align-items:flex-end; }
.terminal-heading .eyebrow { display:flex; align-items:center; gap:7px; color:var(--brand); font-size:12px; font-weight:850; letter-spacing:.08em; text-transform:uppercase; }
.terminal-heading h1 { margin:7px 0 4px; }
.terminal-heading p { max-width:850px; margin:0; color:var(--text-muted); }
.terminal-alert { border:1px solid color-mix(in srgb,var(--danger) 34%,var(--border)); border-radius:10px; padding:11px 13px; color:var(--danger); background:color-mix(in srgb,var(--danger) 8%,var(--surface)); }
.terminal-workspace { display:grid; height:clamp(560px,calc(100vh - 220px),760px); min-height:560px; grid-template-columns:280px minmax(0,1fr); overflow:hidden; border:1px solid var(--border); border-radius:16px; background:var(--surface); box-shadow:var(--shadow-sm); }
.terminal-connections { display:grid; min-width:0; min-height:0; grid-template-rows:auto auto minmax(0,1fr); overflow:hidden; border-right:1px solid var(--border); background:color-mix(in srgb,var(--surface) 92%,var(--brand) 8%); }
.terminal-connections>header { display:flex; align-items:center; justify-content:space-between; padding:17px 16px 12px; color:var(--brand); }
.terminal-connections>header div { display:grid; gap:2px; color:var(--text); }
.terminal-connections>header small { color:var(--text-muted); font-weight:500; }
.terminal-search { display:block; padding:0 12px 10px; }
.terminal-search input { width:100%; border:1px solid var(--border); border-radius:9px; padding:9px 11px; color:var(--text); background:var(--surface); }
.terminal-connections__list { min-height:0; overflow-y:auto; overscroll-behavior:contain; padding-bottom:8px; scrollbar-gutter:stable; }
.terminal-host { display:grid; width:calc(100% - 16px); grid-template-columns:auto minmax(0,1fr) auto; align-items:center; gap:10px; margin:4px 8px; border:1px solid transparent; border-radius:11px; padding:11px; text-align:left; color:var(--text); background:transparent; }
.terminal-host:hover,.terminal-host.is-active { border-color:color-mix(in srgb,var(--brand) 45%,var(--border)); background:color-mix(in srgb,var(--brand) 10%,var(--surface)); }
.terminal-host__icon { display:grid; width:36px; height:36px; place-items:center; border-radius:10px; color:var(--brand); background:color-mix(in srgb,var(--brand) 12%,var(--surface)); }
.terminal-host>span:nth-child(2) { display:grid; min-width:0; gap:2px; }
.terminal-host strong,.terminal-host small { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.terminal-host small { color:var(--text-muted); font-size:11px; }
.terminal-host em { display:flex; align-items:center; gap:5px; color:var(--text-muted); font-size:10px; font-style:normal; }
.terminal-host em.is-ready { color:var(--success); }
.terminal-connections__empty { display:flex; align-items:center; justify-content:center; gap:8px; min-height:180px; padding:20px; color:var(--text-muted); text-align:center; }
.terminal-stage { display:grid; grid-template-rows:auto minmax(0,1fr); min-width:0; min-height:0; padding:12px; background:color-mix(in srgb,var(--terminal-shell-background,#0b1214) 96%,var(--surface)); }
.terminal-tabs { display:flex; gap:5px; overflow-x:auto; padding:0 0 9px; }
.terminal-tabs button { display:flex; flex:0 0 auto; align-items:center; gap:7px; border:1px solid var(--terminal-shell-border,#29383a); border-radius:8px; padding:7px 9px; color:var(--terminal-shell-muted,#8a9695); background:var(--terminal-shell-panel,#111a1d); }
.terminal-tabs button.is-active { color:var(--terminal-shell-text,#d8dddc); border-color:var(--brand); }
.terminal-empty { display:grid; place-content:center; justify-items:center; padding:36px; color:var(--terminal-shell-muted,#8a9695); text-align:center; }
.terminal-empty span { display:grid; width:72px; height:72px; place-items:center; border:1px solid var(--terminal-shell-border,#29383a); border-radius:20px; color:var(--brand); background:var(--terminal-shell-panel,#111a1d); }
.terminal-empty h2 { margin:18px 0 5px; color:var(--terminal-shell-text,#d8dddc); }
.terminal-empty p { max-width:480px; margin:0; }
.spin { animation:spin .8s linear infinite; }
@keyframes spin { to { transform:rotate(360deg); } }
@media (max-width: 900px) { .terminal-workspace { height:min(760px,calc(100dvh - 110px)); min-height:560px; grid-template-columns:1fr; grid-template-rows:minmax(140px,220px) minmax(0,1fr); } .terminal-connections { border-right:0; border-bottom:1px solid var(--border); } .terminal-stage { min-height:0; } }
</style>
