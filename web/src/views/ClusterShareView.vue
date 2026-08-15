<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  Activity,
  ArrowDown,
  ArrowUp,
  Clock3,
  Gauge,
  Globe2,
  HardDrive,
  MapPin,
  MemoryStick,
  RefreshCw,
  Server,
} from '@lucide/vue'
import { usePhraseCatalog } from '@/i18n/phrase'
import { ApiError, api } from '@/lib/api'
import {
  clampPercent,
  formatDateTime,
  formatDuration,
  formatPercent,
  formatRate,
  relativeTime,
} from '@/lib/format'
import type { PublicClusterShareHost, PublicClusterShareSnapshot } from '@/types/api'

usePhraseCatalog((locale) => locale === 'en-US'
  ? import('@/i18n/pages/ClusterShareView/en-US').then((module) => module.default)
  : import('@/i18n/pages/ClusterShareView/zh-TW').then((module) => module.default))

const route = useRoute()
const snapshot = ref<PublicClusterShareSnapshot>()
const loading = ref(true)
const refreshing = ref(false)
const errorMessage = ref('')
let controller: AbortController | undefined
let pollTimer: number | undefined

const token = computed(() => String(route.params.token || ''))
const tokenIsValid = computed(() => /^[a-f0-9]{64}$/.test(token.value))

function stateLabel(state: PublicClusterShareHost['state']): string {
  return {
    online: '在线',
    degraded: '需关注',
    offline: '离线',
    pending: '等待数据',
  }[state]
}

function locationLabel(host: PublicClusterShareHost): string {
  return [host.location.country, host.location.region, host.location.city]
    .filter((value, index, items) => value && items.indexOf(value) === index)
    .join(' · ') || '地区未公开'
}

function friendlyError(reason: unknown): string {
  if (reason instanceof ApiError && reason.status === 404) {
    return '分享链接无效、已关闭或已经重置。'
  }
  return '暂时无法读取集群状态，请稍后重试。'
}

async function load(silent = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  if (!silent || !snapshot.value) loading.value = true
  else refreshing.value = true
  if (!tokenIsValid.value) {
    snapshot.value = undefined
    errorMessage.value = '分享链接格式无效。'
    loading.value = false
    refreshing.value = false
    return
  }
  try {
    snapshot.value = await api.cluster.publicShare(token.value, controller.signal)
    errorMessage.value = ''
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return
    errorMessage.value = friendlyError(reason)
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function onVisibilityChange(): void {
  if (!document.hidden) void load(true)
}

watch(token, () => void load())

onMounted(() => {
  void load()
  pollTimer = window.setInterval(() => {
    if (!document.hidden) void load(true)
  }, 15_000)
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  controller?.abort()
  if (pollTimer) window.clearInterval(pollTimer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<template>
  <main class="share-page">
    <div class="share-page__glow share-page__glow--one" />
    <div class="share-page__glow share-page__glow--two" />

    <div class="share-shell">
      <header class="share-header">
        <a class="share-brand" href="https://github.com/kejilion/KPanel" target="_blank" rel="noopener noreferrer">
          <span><Server :size="18" /></span>
          <strong>KPanel</strong>
        </a>
        <button
          class="share-refresh"
          type="button"
          :disabled="loading || refreshing"
          aria-label="刷新公开状态"
          @click="load(true)"
        >
          <RefreshCw :size="16" :class="{ spin: refreshing }" />
          <span>{{ refreshing ? '正在刷新' : '刷新' }}</span>
        </button>
      </header>

      <section v-if="snapshot" class="share-hero">
        <div class="share-hero__copy">
          <span class="share-kicker"><Globe2 :size="14" /> PUBLIC FLEET</span>
          <h1>{{ snapshot.title }}</h1>
          <p>{{ snapshot.description || '这些是我正在运行的服务器。' }}</p>
          <small>数据生成于 {{ formatDateTime(snapshot.generatedAt) }} · {{ relativeTime(snapshot.generatedAt) }}</small>
        </div>
        <div class="share-stats" aria-label="集群状态概览">
          <div><strong>{{ snapshot.total }}</strong><span>全部机器</span></div>
          <div class="is-online"><strong>{{ snapshot.online }}</strong><span>在线</span></div>
          <div class="is-attention"><strong>{{ snapshot.attention }}</strong><span>需关注</span></div>
        </div>
      </section>

      <section v-if="loading && !snapshot" class="share-state" aria-live="polite">
        <RefreshCw class="spin" :size="24" />
        <strong>正在读取机器状态…</strong>
      </section>

      <section v-else-if="errorMessage && !snapshot" class="share-state share-state--error" role="alert">
        <Activity :size="25" />
        <strong>无法打开分享页</strong>
        <p>{{ errorMessage }}</p>
        <button type="button" @click="load()">重试</button>
      </section>

      <div v-else-if="errorMessage" class="share-warning" role="status">
        {{ errorMessage }} 当前保留上一次成功数据。
      </div>

      <section v-if="snapshot?.items.length" class="share-grid" aria-label="公开机器状态">
        <article v-for="host in snapshot.items" :key="host.id" class="share-card">
          <header class="share-card__header">
            <span class="share-card__icon"><Server :size="18" /></span>
            <div>
              <h2>{{ host.name }}</h2>
              <p><MapPin :size="12" /> {{ locationLabel(host) }}</p>
            </div>
            <span class="share-status" :class="`is-${host.state}`">
              <i /> {{ stateLabel(host.state) }}
            </span>
          </header>

          <div v-if="host.collectedAt" class="share-metrics">
            <div>
              <span><Gauge :size="14" /> CPU</span>
              <strong>{{ formatPercent(host.cpu.usagePercent) }}</strong>
              <i><b :style="{ width: `${clampPercent(host.cpu.usagePercent)}%` }" /></i>
              <small>{{ host.cpu.cores }} 核</small>
            </div>
            <div>
              <span><MemoryStick :size="14" /> 内存</span>
              <strong>{{ formatPercent(host.memory.usagePercent) }}</strong>
              <i><b :style="{ width: `${clampPercent(host.memory.usagePercent)}%` }" /></i>
              <small>{{ host.memory.totalBytes ? `${Math.round(host.memory.totalBytes / 1073741824)} GB` : '—' }}</small>
            </div>
            <div>
              <span><HardDrive :size="14" /> 磁盘</span>
              <strong>{{ formatPercent(host.disk.usagePercent) }}</strong>
              <i><b :style="{ width: `${clampPercent(host.disk.usagePercent)}%` }" /></i>
              <small>{{ host.disk.totalBytes ? `${Math.round(host.disk.totalBytes / 1073741824)} GB` : '—' }}</small>
            </div>
          </div>
          <div v-else class="share-card__empty">等待第一份状态数据</div>

          <dl class="share-details">
            <div>
              <dt>系统</dt>
              <dd>{{ host.os || '—' }}<small>{{ host.architecture || '' }}</small></dd>
            </div>
            <div>
              <dt><Clock3 :size="13" /> 运行时间</dt>
              <dd>{{ host.uptimeSeconds ? formatDuration(host.uptimeSeconds) : '—' }}</dd>
            </div>
            <div>
              <dt><ArrowDown :size="13" /> 下行</dt>
              <dd>{{ formatRate(host.network.receiveBytesPerSecond || 0) }}</dd>
            </div>
            <div>
              <dt><ArrowUp :size="13" /> 上行</dt>
              <dd>{{ formatRate(host.network.transmitBytesPerSecond || 0) }}</dd>
            </div>
          </dl>

          <footer>
            <span>{{ host.location.isp || '网络信息未公开' }}</span>
            <small>{{ host.collectedAt ? `采集于 ${relativeTime(host.collectedAt)}` : '尚无数据' }}</small>
          </footer>
        </article>
      </section>

      <section v-else-if="snapshot" class="share-state">
        <Server :size="26" />
        <strong>还没有可展示的机器</strong>
      </section>

      <footer class="share-footer">
        <span>Powered by <strong>KPanel</strong></span>
        <span>公开页不包含 IP、管理入口或访问凭据</span>
      </footer>
    </div>
  </main>
</template>

<style scoped>
.share-page {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  color: #e8eef8;
  background:
    radial-gradient(circle at 20% 0%, rgb(48 106 255 / 18%), transparent 35%),
    linear-gradient(155deg, #07101f 0%, #0a1220 48%, #091724 100%);
}

.share-page__glow {
  position: fixed;
  width: 440px;
  height: 440px;
  pointer-events: none;
  filter: blur(100px);
  opacity: 0.14;
  border-radius: 50%;
}

.share-page__glow--one { top: 10%; right: -180px; background: #31d7b7; }
.share-page__glow--two { bottom: -240px; left: -160px; background: #3978ff; }

.share-shell {
  position: relative;
  z-index: 1;
  width: min(1220px, calc(100% - 40px));
  margin: 0 auto;
  padding: 26px 0 34px;
}

.share-header,
.share-brand,
.share-refresh,
.share-card__header,
.share-status,
.share-details dt,
.share-footer {
  display: flex;
  align-items: center;
}

.share-header { justify-content: space-between; margin-bottom: 34px; }
.share-brand { gap: 10px; color: inherit; text-decoration: none; letter-spacing: 0.02em; }
.share-brand > span {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  color: #07101f;
  background: linear-gradient(135deg, #57e4ca, #8cf1c9);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgb(63 224 191 / 20%);
}

.share-refresh,
.share-state button {
  gap: 7px;
  padding: 9px 13px;
  color: #d7e3f4;
  background: rgb(255 255 255 / 5%);
  border: 1px solid rgb(255 255 255 / 10%);
  border-radius: 10px;
  cursor: pointer;
}

.share-refresh:disabled { cursor: wait; opacity: 0.58; }

.share-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 30px;
  padding: 34px;
  margin-bottom: 22px;
  background: linear-gradient(135deg, rgb(255 255 255 / 7%), rgb(255 255 255 / 3%));
  border: 1px solid rgb(255 255 255 / 9%);
  border-radius: 24px;
  box-shadow: 0 30px 80px rgb(0 0 0 / 24%);
  backdrop-filter: blur(18px);
}

.share-kicker { display: flex; align-items: center; gap: 7px; color: #66dfc3; font-size: 11px; font-weight: 800; letter-spacing: 0.16em; }
.share-hero h1 { margin: 12px 0 8px; font-size: clamp(28px, 4vw, 48px); line-height: 1.05; letter-spacing: -0.04em; }
.share-hero p { max-width: 670px; margin: 0 0 14px; color: #a9b9cf; font-size: 15px; line-height: 1.7; }
.share-hero small { color: #6f819b; }

.share-stats { display: grid; grid-template-columns: repeat(3, minmax(90px, 1fr)); }
.share-stats div { display: grid; gap: 4px; padding: 4px 20px; border-left: 1px solid rgb(255 255 255 / 9%); }
.share-stats strong { font-size: 30px; line-height: 1; }
.share-stats span { color: #8294ad; font-size: 11px; }
.share-stats .is-online strong { color: #5be0ad; }
.share-stats .is-attention strong { color: #ffbd6b; }

.share-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 15px; }
.share-card {
  min-width: 0;
  overflow: hidden;
  background: rgb(12 25 42 / 88%);
  border: 1px solid rgb(255 255 255 / 8%);
  border-radius: 18px;
  box-shadow: 0 18px 45px rgb(0 0 0 / 18%);
}

.share-card__header { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; gap: 11px; padding: 17px; }
.share-card__icon { display: grid; width: 35px; height: 35px; place-items: center; color: #5ee0c1; background: rgb(75 223 191 / 10%); border-radius: 10px; }
.share-card h2 { overflow: hidden; margin: 0 0 4px; font-size: 14px; text-overflow: ellipsis; white-space: nowrap; }
.share-card__header p { display: flex; align-items: center; gap: 4px; overflow: hidden; margin: 0; color: #7588a2; font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.share-status { gap: 5px; color: #8fa0b5; font-size: 10px; }
.share-status i { width: 7px; height: 7px; background: currentColor; border-radius: 50%; box-shadow: 0 0 10px currentColor; }
.share-status.is-online { color: #5be0ad; }
.share-status.is-degraded { color: #ffbd6b; }
.share-status.is-offline { color: #ff7786; }

.share-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); border-block: 1px solid rgb(255 255 255 / 7%); }
.share-metrics > div { display: grid; gap: 6px; padding: 14px; border-left: 1px solid rgb(255 255 255 / 7%); }
.share-metrics > div:first-child { border-left: 0; }
.share-metrics span { display: flex; align-items: center; gap: 5px; color: #7f91a9; font-size: 10px; }
.share-metrics strong { font-size: 16px; }
.share-metrics > div > i { height: 3px; overflow: hidden; background: rgb(255 255 255 / 8%); border-radius: 10px; }
.share-metrics b { display: block; height: 100%; background: linear-gradient(90deg, #397dff, #58dfbd); border-radius: inherit; }
.share-metrics small { color: #657892; font-size: 9px; }
.share-card__empty { padding: 29px; color: #71849d; text-align: center; border-block: 1px solid rgb(255 255 255 / 7%); }

.share-details { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px 18px; padding: 17px; margin: 0; }
.share-details div { min-width: 0; }
.share-details dt { gap: 5px; margin-bottom: 5px; color: #72849c; font-size: 10px; }
.share-details dd { overflow: hidden; margin: 0; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.share-details dd small { display: block; margin-top: 3px; color: #60728b; font-size: 9px; }
.share-card footer { display: flex; justify-content: space-between; gap: 10px; padding: 12px 17px; color: #697d96; background: rgb(255 255 255 / 2%); border-top: 1px solid rgb(255 255 255 / 6%); font-size: 9px; }

.share-state { display: grid; min-height: 280px; place-items: center; align-content: center; gap: 12px; color: #8799b1; text-align: center; }
.share-state p { margin: 0; }
.share-state--error svg { color: #ff7786; }
.share-warning { padding: 12px 15px; margin-bottom: 14px; color: #ffd08d; background: rgb(255 174 64 / 9%); border: 1px solid rgb(255 174 64 / 16%); border-radius: 12px; }
.share-footer { justify-content: space-between; gap: 20px; padding: 28px 4px 0; color: #536984; font-size: 10px; }
.share-footer strong { color: #7f94ae; }

@media (max-width: 1000px) {
  .share-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .share-hero { grid-template-columns: 1fr; align-items: start; }
  .share-stats div:first-child { border-left: 0; }
}

@media (max-width: 650px) {
  .share-shell { width: min(100% - 24px, 1220px); padding-top: 16px; }
  .share-header { margin-bottom: 18px; }
  .share-refresh span { display: none; }
  .share-hero { gap: 24px; padding: 24px 18px; border-radius: 18px; }
  .share-hero h1 { font-size: 31px; }
  .share-stats { width: 100%; }
  .share-stats div { padding: 2px 13px; }
  .share-stats div:first-child { border-left: 0; }
  .share-grid { grid-template-columns: minmax(0, 1fr); }
  .share-footer { align-items: flex-start; flex-direction: column; }
}

@media (prefers-reduced-motion: reduce) {
  .spin { animation: none; }
}
</style>
