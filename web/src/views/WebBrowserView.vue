<script setup lang="ts">
import { computed, inject, onBeforeUnmount, ref, watch } from 'vue'
import {
  ExternalLink,
  Globe2,
  Home,
  MoonStar,
  Plus,
  RefreshCw,
  ShieldCheck,
  TriangleAlert,
  X,
} from '@lucide/vue'
import { useRoute } from 'vue-router'
import {
  EMBEDDED_BROWSER_SLEEP_MS,
  MAX_EMBEDDED_BROWSER_TABS,
  MAX_LIVE_EMBEDDED_BROWSER_TABS,
  embeddedBrowserShortcutsKey,
  resolveEmbeddedBrowserInput,
  resolveEmbeddedBrowserTarget,
  type EmbeddedBrowserShortcut,
  type EmbeddedBrowserTarget,
} from '@/lib/embeddedBrowser'
import { desktopWindowActiveKey } from '@/lib/desktopRouteKeys'
import { useI18n } from '@/i18n'
import { usePhraseCatalog } from '@/i18n/phrase'

interface BrowserTab {
  id: string
  title: string
  target?: EmbeddedBrowserTarget
  shortcutID?: string
  iconURL?: string
  frameVersion: number
  lastActiveAt: number
}

interface PendingBrowserRequest {
  target?: EmbeddedBrowserTarget
  shortcut?: EmbeddedBrowserShortcut
}

const START_PAGE_SHORTCUT_LIMIT = 12
const route = useRoute()
const i18n = useI18n()
usePhraseCatalog(() => import('@/i18n/pages/WebBrowserView/en-US').then((module) => module.default))

const fallbackShortcuts = ref<EmbeddedBrowserShortcut[]>([])
const browserShortcuts = inject(embeddedBrowserShortcutsKey, fallbackShortcuts)
const fallbackWindowActive = ref(true)
const windowActive = inject(desktopWindowActiveKey, fallbackWindowActive)
const startPageShortcuts = computed(() => browserShortcuts.value.slice(0, START_PAGE_SHORTCUT_LIMIT))

const tabs = ref<BrowserTab[]>([])
const activeTabID = ref('')
const liveTabIDs = ref<Set<string>>(new Set())
const pendingRequest = ref<PendingBrowserRequest>()
const addressValue = ref('')
const addressInvalid = ref(false)
const sleepTimers = new Map<string, number>()
let tabSequence = 0

const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabID.value))
const liveTabs = computed(() => tabs.value.filter((tab) => liveTabIDs.value.has(tab.id)))
const oldestClosableTab = computed(() => tabs.value
  .filter((tab) => tab.id !== activeTabID.value)
  .sort((left, right) => left.lastActiveAt - right.lastActiveAt)[0])

function startPageTitle(): string {
  return i18n.t('desktop.browserNewTab')
}

function nextTabID(): string {
  tabSequence += 1
  return `browser-tab-${tabSequence}`
}

function replaceLiveTabIDs(update: (next: Set<string>) => void): void {
  const next = new Set(liveTabIDs.value)
  update(next)
  liveTabIDs.value = next
}

function clearSleepTimer(tabID: string): void {
  const timer = sleepTimers.get(tabID)
  if (timer !== undefined) window.clearTimeout(timer)
  sleepTimers.delete(tabID)
}

function sleepTab(tabID: string): void {
  clearSleepTimer(tabID)
  replaceLiveTabIDs((next) => next.delete(tabID))
}

function scheduleSleep(tabID: string): void {
  clearSleepTimer(tabID)
  sleepTimers.set(tabID, window.setTimeout(() => {
    if (tabID !== activeTabID.value || !windowActive.value) sleepTab(tabID)
  }, EMBEDDED_BROWSER_SLEEP_MS))
}

function enforceLiveLimit(): void {
  const live = tabs.value
    .filter((tab) => liveTabIDs.value.has(tab.id) && tab.id !== activeTabID.value)
    .sort((left, right) => left.lastActiveAt - right.lastActiveAt)
  while (liveTabIDs.value.size > MAX_LIVE_EMBEDDED_BROWSER_TABS && live.length) {
    const tab = live.shift()
    if (tab) sleepTab(tab.id)
  }
}

function mountTab(tab: BrowserTab): void {
  if (!tab.target || tab.target.mixedContent) return
  clearSleepTimer(tab.id)
  replaceLiveTabIDs((next) => next.add(tab.id))
  enforceLiveLimit()
}

function activateTab(tabID: string): void {
  const nextTab = tabs.value.find((tab) => tab.id === tabID)
  if (!nextTab) return
  const previousID = activeTabID.value
  if (previousID && previousID !== tabID) scheduleSleep(previousID)
  activeTabID.value = tabID
  nextTab.lastActiveAt = Date.now()
  addressValue.value = nextTab.target?.href || ''
  addressInvalid.value = false
  if (windowActive.value) mountTab(nextTab)
}

function createStartTab(): BrowserTab {
  const existing = tabs.value.find((tab) => !tab.target)
  if (existing) {
    activateTab(existing.id)
    return existing
  }
  const tab: BrowserTab = {
    id: nextTabID(),
    title: startPageTitle(),
    frameVersion: 0,
    lastActiveAt: Date.now(),
  }
  tabs.value.push(tab)
  activateTab(tab.id)
  return tab
}

function shortcutForRequest(
  shortcutID: unknown,
  legacySiteID: unknown,
): EmbeddedBrowserShortcut | undefined {
  if (typeof shortcutID === 'string') {
    return browserShortcuts.value.find((shortcut) => shortcut.id === shortcutID)
  }
  if (typeof legacySiteID !== 'string') return undefined
  return browserShortcuts.value.find((shortcut) => (
    shortcut.id === legacySiteID || shortcut.id === `site:${legacySiteID}`
  ))
}

function applyTargetToTab(
  tab: BrowserTab,
  target: EmbeddedBrowserTarget,
  shortcut?: EmbeddedBrowserShortcut,
): void {
  sleepTab(tab.id)
  tab.target = target
  tab.title = shortcut?.name || target.hostname
  tab.shortcutID = shortcut?.id
  tab.iconURL = shortcut?.iconURL
  tab.frameVersion += 1
  activateTab(tab.id)
}

function openTarget(target: EmbeddedBrowserTarget, shortcut?: EmbeddedBrowserShortcut): boolean {
  const existing = tabs.value.find((tab) => tab.target?.href === target.href)
  if (existing) {
    if (shortcut) {
      existing.title = shortcut.name
      existing.shortcutID = shortcut.id
      existing.iconURL = shortcut.iconURL
    }
    pendingRequest.value = undefined
    activateTab(existing.id)
    return true
  }

  const startTab = tabs.value.find((tab) => !tab.target)
  if (startTab) {
    pendingRequest.value = undefined
    applyTargetToTab(startTab, target, shortcut)
    return true
  }

  if (tabs.value.length >= MAX_EMBEDDED_BROWSER_TABS) {
    pendingRequest.value = { target, shortcut }
    return false
  }

  const tab: BrowserTab = {
    id: nextTabID(),
    title: shortcut?.name || target.hostname,
    target,
    shortcutID: shortcut?.id,
    iconURL: shortcut?.iconURL,
    frameVersion: 0,
    lastActiveAt: Date.now(),
  }
  tabs.value.push(tab)
  pendingRequest.value = undefined
  activateTab(tab.id)
  return true
}

function requestNewTab(): void {
  const existing = tabs.value.find((tab) => !tab.target)
  if (existing) {
    activateTab(existing.id)
    return
  }
  if (tabs.value.length >= MAX_EMBEDDED_BROWSER_TABS) {
    pendingRequest.value = {}
    return
  }
  createStartTab()
}

function consumePendingRequest(): void {
  const request = pendingRequest.value
  pendingRequest.value = undefined
  if (!request) return
  if (request.target) openTarget(request.target, request.shortcut)
  else createStartTab()
}

function closeTab(tabID: string, consumePending = true): void {
  const index = tabs.value.findIndex((tab) => tab.id === tabID)
  if (index < 0) return
  const wasActive = tabID === activeTabID.value
  clearSleepTimer(tabID)
  replaceLiveTabIDs((next) => next.delete(tabID))
  tabs.value.splice(index, 1)

  if (consumePending && pendingRequest.value) {
    consumePendingRequest()
    return
  }
  if (!tabs.value.length) {
    createStartTab()
    return
  }
  if (wasActive) {
    const replacement = tabs.value[Math.min(index, tabs.value.length - 1)]
    if (replacement) activateTab(replacement.id)
  }
}

function closeOldestAndContinue(): void {
  const candidate = oldestClosableTab.value
  if (candidate) closeTab(candidate.id)
}

function cancelPendingRequest(): void {
  pendingRequest.value = undefined
}

function openPendingExternally(): void {
  const href = pendingRequest.value?.target?.href
  if (!href) return
  window.open(href, '_blank', 'noopener,noreferrer')
  pendingRequest.value = undefined
}

function submitAddress(): void {
  const target = resolveEmbeddedBrowserInput(addressValue.value)
  if (!target) {
    addressInvalid.value = true
    return
  }
  addressInvalid.value = false
  const tab = activeTab.value || createStartTab()
  applyTargetToTab(tab, target)
}

function openShortcut(shortcut: EmbeddedBrowserShortcut): void {
  const target = resolveEmbeddedBrowserTarget(shortcut.url)
  if (!target) return
  openTarget(target, shortcut)
}

function goHome(): void {
  const tab = activeTab.value
  if (!tab) {
    createStartTab()
    return
  }
  const otherStartTab = tabs.value.find((candidate) => candidate.id !== tab.id && !candidate.target)
  if (otherStartTab) {
    closeTab(tab.id, false)
    activateTab(otherStartTab.id)
    return
  }
  sleepTab(tab.id)
  tab.target = undefined
  tab.shortcutID = undefined
  tab.iconURL = undefined
  tab.title = startPageTitle()
  tab.frameVersion += 1
  addressValue.value = ''
  addressInvalid.value = false
}

function reload(): void {
  const tab = activeTab.value
  if (!tab?.target || tab.target.mixedContent) return
  tab.frameVersion += 1
  mountTab(tab)
}

function openExternal(): void {
  const href = activeTab.value?.target?.href
  if (href) window.open(href, '_blank', 'noopener,noreferrer')
}

function hideBrokenIcon(event: Event): void {
  if (event.currentTarget instanceof HTMLImageElement) event.currentTarget.hidden = true
}

watch(
  () => {
    const value = route.query.url
    const url = Array.isArray(value) ? value[0] : value
    const request = Array.isArray(route.query.request) ? route.query.request[0] : route.query.request
    const shortcut = Array.isArray(route.query.shortcut) ? route.query.shortcut[0] : route.query.shortcut
    const legacySite = Array.isArray(route.query.site) ? route.query.site[0] : route.query.site
    return { url, request, shortcut, legacySite }
  },
  ({ url, shortcut, legacySite }) => {
    if (!url) {
      if (!tabs.value.length) createStartTab()
      return
    }
    const target = resolveEmbeddedBrowserTarget(url)
    if (!target) {
      addressValue.value = String(url)
      addressInvalid.value = true
      if (!tabs.value.length) createStartTab()
      return
    }
    openTarget(target, shortcutForRequest(shortcut, legacySite))
  },
  { immediate: true },
)

watch(
  browserShortcuts,
  (shortcuts) => {
    for (const tab of tabs.value) {
      const shortcut = shortcuts.find((candidate) => candidate.id === tab.shortcutID)
      if (!shortcut) continue
      tab.title = shortcut.name
      tab.iconURL = shortcut.iconURL
    }
  },
  { deep: false },
)

watch(windowActive, (active) => {
  const tab = activeTab.value
  if (!tab?.target) return
  if (active) mountTab(tab)
  else scheduleSleep(tab.id)
})

onBeforeUnmount(() => {
  for (const timer of sleepTimers.values()) window.clearTimeout(timer)
  sleepTimers.clear()
})
</script>

<template>
  <section class="embedded-browser">
    <nav class="embedded-browser__tabs" role="tablist" :aria-label="i18n.t('desktop.browserTabsLabel')">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="embedded-browser__tab"
        :class="{ 'embedded-browser__tab--active': tab.id === activeTabID }"
      >
        <button
          class="embedded-browser__tab-main"
          type="button"
          role="tab"
          :aria-selected="tab.id === activeTabID"
          :title="tab.title"
          @click="activateTab(tab.id)"
        >
          <img
            v-if="tab.iconURL"
            class="embedded-browser__tab-icon"
            :src="tab.iconURL"
            alt=""
            @error="hideBrokenIcon"
          >
          <Globe2 v-else :size="14" aria-hidden="true" />
          <span>{{ tab.title }}</span>
          <MoonStar
            v-if="tab.target && !tab.target.mixedContent && !liveTabIDs.has(tab.id)"
            class="embedded-browser__tab-sleep"
            :size="12"
            :aria-label="i18n.t('desktop.browserTabSleeping')"
          />
        </button>
        <button
          class="embedded-browser__tab-close"
          type="button"
          :title="i18n.t('desktop.browserCloseTab')"
          :aria-label="i18n.t('desktop.browserCloseNamedTab', { name: tab.title })"
          @click="closeTab(tab.id)"
        >
          <X :size="13" aria-hidden="true" />
        </button>
      </div>
      <button
        class="embedded-browser__new-tab"
        type="button"
        :title="i18n.t('desktop.browserNewTab')"
        :aria-label="i18n.t('desktop.browserNewTab')"
        @click="requestNewTab"
      >
        <Plus :size="16" aria-hidden="true" />
      </button>
      <span class="embedded-browser__tab-count">{{ tabs.length }}/{{ MAX_EMBEDDED_BROWSER_TABS }}</span>
    </nav>

    <form class="embedded-browser__toolbar" @submit.prevent="submitAddress">
      <button
        class="embedded-browser__tool"
        type="button"
        :title="i18n.t('desktop.browserHome')"
        :aria-label="i18n.t('desktop.browserHome')"
        @click="goHome"
      >
        <Home :size="16" aria-hidden="true" />
      </button>
      <label class="embedded-browser__address" :class="{ 'embedded-browser__address--invalid': addressInvalid }">
        <ShieldCheck :size="14" aria-hidden="true" />
        <span class="embedded-browser__sr-only">{{ i18n.t('desktop.browserAddressLabel') }}</span>
        <input
          v-model="addressValue"
          type="text"
          inputmode="url"
          maxlength="2048"
          autocomplete="off"
          autocapitalize="off"
          spellcheck="false"
          :placeholder="i18n.t('desktop.browserAddressPlaceholder')"
          :aria-invalid="addressInvalid"
          @input="addressInvalid = false"
        >
      </label>
      <button
        class="embedded-browser__tool"
        type="button"
        :disabled="!activeTab?.target || activeTab.target.mixedContent"
        :title="i18n.t('desktop.browserReload')"
        :aria-label="i18n.t('desktop.browserReload')"
        @click="reload"
      >
        <RefreshCw :size="16" aria-hidden="true" />
      </button>
      <button
        class="button button--primary button--small embedded-browser__external"
        type="button"
        :disabled="!activeTab?.target"
        @click="openExternal"
      >
        <ExternalLink :size="14" aria-hidden="true" />
        <span>{{ i18n.t('desktop.browserOpenExternal') }}</span>
      </button>
    </form>

    <div v-if="pendingRequest" class="embedded-browser__limit" role="status" aria-live="polite">
      <TriangleAlert :size="18" aria-hidden="true" />
      <div>
        <strong>{{ i18n.t('desktop.browserTabLimitTitle', { count: MAX_EMBEDDED_BROWSER_TABS }) }}</strong>
        <span>{{ i18n.t('desktop.browserTabLimitMessage', {
          name: pendingRequest.shortcut?.name || pendingRequest.target?.hostname || i18n.t('desktop.browserNewTab'),
        }) }}</span>
      </div>
      <button
        v-if="oldestClosableTab"
        class="button button--secondary button--small"
        type="button"
        @click="closeOldestAndContinue"
      >
        {{ i18n.t('desktop.browserCloseOldest', { name: oldestClosableTab.title }) }}
      </button>
      <button
        v-if="pendingRequest.target"
        class="button button--secondary button--small"
        type="button"
        @click="openPendingExternally"
      >
        {{ i18n.t('desktop.browserOpenExternal') }}
      </button>
      <button class="embedded-browser__limit-cancel" type="button" @click="cancelPendingRequest">
        {{ i18n.t('common.cancel') }}
      </button>
    </div>

    <main class="embedded-browser__content">
      <div v-if="!activeTab?.target" class="embedded-browser__start">
        <div class="embedded-browser__start-mark" aria-hidden="true"><Globe2 :size="30" /></div>
        <h1>{{ i18n.t('desktop.browserStartTitle') }}</h1>
        <p>{{ i18n.t('desktop.browserStartDescription') }}</p>
        <form class="embedded-browser__start-form" @submit.prevent="submitAddress">
          <Globe2 :size="19" aria-hidden="true" />
          <input
            v-model="addressValue"
            type="text"
            inputmode="url"
            maxlength="2048"
            autocomplete="off"
            autocapitalize="off"
            spellcheck="false"
            :placeholder="i18n.t('desktop.browserAddressPlaceholder')"
            :aria-label="i18n.t('desktop.browserAddressLabel')"
            :aria-invalid="addressInvalid"
            autofocus
            @input="addressInvalid = false"
          >
          <button type="submit">{{ i18n.t('desktop.browserVisit') }}</button>
        </form>
        <span v-if="addressInvalid" class="embedded-browser__input-error" role="alert">
          {{ i18n.t('desktop.browserInvalidURL') }}
        </span>
        <section v-if="startPageShortcuts.length" class="embedded-browser__shortcuts">
          <h2>{{ i18n.t('desktop.browserConfiguredShortcuts') }}</h2>
          <div>
            <button
              v-for="shortcut in startPageShortcuts"
              :key="shortcut.id"
              type="button"
              @click="openShortcut(shortcut)"
            >
              <span>
                <img v-if="shortcut.iconURL" :src="shortcut.iconURL" alt="" @error="hideBrokenIcon">
                <Globe2 :size="18" aria-hidden="true" />
              </span>
              <strong>{{ shortcut.name }}</strong>
            </button>
          </div>
        </section>
      </div>

      <div v-else-if="activeTab.target.mixedContent" class="embedded-browser__state" role="alert">
        <span><ShieldCheck :size="24" aria-hidden="true" /></span>
        <strong>{{ i18n.t('desktop.browserMixedContentTitle') }}</strong>
        <p>{{ i18n.t('desktop.browserMixedContentMessage') }}</p>
        <button class="button button--primary" type="button" @click="openExternal">
          <ExternalLink :size="15" aria-hidden="true" />
          {{ i18n.t('desktop.browserOpenExternal') }}
        </button>
      </div>

      <iframe
        v-for="tab in liveTabs"
        v-show="tab.id === activeTabID"
        :key="`${tab.id}:${tab.frameVersion}`"
        class="embedded-browser__frame"
        :src="tab.target?.href"
        :title="tab.title"
        sandbox="allow-downloads allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts"
        referrerpolicy="no-referrer"
        allow="fullscreen"
      />
    </main>

    <footer v-if="activeTab?.target && !activeTab.target.mixedContent" class="embedded-browser__hint">
      <ShieldCheck :size="13" aria-hidden="true" />
      <span>{{ i18n.t('desktop.browserEmbedHint') }}</span>
    </footer>
  </section>
</template>

<style scoped>
.embedded-browser {
  display: flex;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  flex-direction: column;
  color: var(--text);
  background: var(--surface-subtle);
}

.embedded-browser__tabs {
  display: flex;
  min-width: 0;
  min-height: 38px;
  align-items: flex-end;
  gap: 3px;
  padding: 5px 8px 0;
  overflow-x: auto;
  scrollbar-width: none;
  background: color-mix(in srgb, var(--surface-raised) 92%, var(--bg));
  border-bottom: 1px solid var(--border);
}

.embedded-browser__tabs::-webkit-scrollbar { display: none; }

.embedded-browser__tab {
  display: flex;
  width: clamp(118px, 18vw, 190px);
  min-width: 96px;
  height: 32px;
  align-items: center;
  flex: 0 1 190px;
  color: var(--text-soft);
  background: color-mix(in srgb, var(--surface) 76%, transparent);
  border: 1px solid transparent;
  border-bottom: 0;
  border-radius: 9px 9px 0 0;
}

.embedded-browser__tab--active {
  color: var(--text);
  background: var(--surface);
  border-color: var(--border);
}

.embedded-browser__tab-main {
  display: flex;
  min-width: 0;
  height: 100%;
  align-items: center;
  gap: 6px;
  flex: 1 1 auto;
  padding: 0 5px 0 9px;
  color: inherit;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.embedded-browser__tab-main > svg:first-child { flex: 0 0 auto; color: var(--brand); }
.embedded-browser__tab-main > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.embedded-browser__tab-icon { width: 15px; height: 15px; flex: 0 0 auto; object-fit: cover; border-radius: 4px; }
.embedded-browser__tab-sleep { flex: 0 0 auto; margin-left: auto; color: var(--muted); }

.embedded-browser__tab-close,
.embedded-browser__new-tab,
.embedded-browser__tool,
.embedded-browser__limit-cancel {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  padding: 0;
  color: var(--text-soft);
  background: transparent;
  border: 0;
  cursor: pointer;
}

.embedded-browser__tab-close { width: 24px; height: 24px; margin-right: 3px; border-radius: 6px; }
.embedded-browser__tab-close:hover { color: var(--text); background: var(--surface-hover); }
.embedded-browser__new-tab { width: 30px; height: 30px; align-self: center; border-radius: 8px; }
.embedded-browser__new-tab:hover { color: var(--text); background: var(--surface-hover); }
.embedded-browser__tab-count { align-self: center; margin: 0 2px 0 auto; color: var(--muted); font-size: 10px; white-space: nowrap; }

.embedded-browser__toolbar {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
  background: color-mix(in srgb, var(--surface-raised) 94%, var(--bg));
}

.embedded-browser__tool {
  width: 32px;
  height: 32px;
  color: var(--text-soft);
  background: var(--surface-subtle);
  border: 1px solid var(--border);
  border-radius: 9px;
}

.embedded-browser__tool:not(:disabled):hover { color: var(--text); background: var(--surface); }
.embedded-browser__tool:disabled { opacity: .42; cursor: default; }

.embedded-browser__address {
  display: flex;
  min-width: 0;
  height: 34px;
  flex: 1 1 auto;
  align-items: center;
  gap: 7px;
  padding: 0 11px;
  color: var(--text-soft);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  transition: border-color .16s ease, box-shadow .16s ease;
}

.embedded-browser__address:focus-within {
  border-color: color-mix(in srgb, var(--brand) 58%, var(--border));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand) 12%, transparent);
}

.embedded-browser__address--invalid { border-color: var(--danger); }
.embedded-browser__address svg { flex: 0 0 auto; color: var(--success); }
.embedded-browser__address input { width: 100%; min-width: 0; color: var(--text); background: transparent; border: 0; outline: none; font-size: 11px; }
.embedded-browser__external { flex: 0 0 auto; white-space: nowrap; }

.embedded-browser__limit {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 11px;
  color: var(--text-soft);
  background: color-mix(in srgb, var(--warning-soft) 82%, var(--surface));
  border-bottom: 1px solid color-mix(in srgb, var(--warning) 25%, var(--border));
  font-size: 11px;
}

.embedded-browser__limit > svg { flex: 0 0 auto; color: var(--warning); }
.embedded-browser__limit > div { display: grid; min-width: 0; flex: 1 1 auto; gap: 1px; }
.embedded-browser__limit strong { color: var(--text); }
.embedded-browser__limit span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.embedded-browser__limit-cancel { min-width: 44px; height: 28px; padding: 0 8px; border-radius: 7px; }
.embedded-browser__limit-cancel:hover { color: var(--text); background: var(--surface-hover); }

.embedded-browser__content {
  position: relative;
  display: grid;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.embedded-browser__frame { width: 100%; height: 100%; min-height: 0; background: #fff; border: 0; }

.embedded-browser__start {
  display: flex;
  width: min(680px, calc(100% - 40px));
  min-height: 100%;
  align-items: center;
  justify-self: center;
  flex-direction: column;
  padding: clamp(34px, 9vh, 82px) 0 36px;
  overflow-y: auto;
  text-align: center;
}

.embedded-browser__start-mark {
  display: grid;
  width: 64px;
  height: 64px;
  place-items: center;
  color: #e0f2fe;
  background: linear-gradient(145deg, #38bdf8, #0369a1);
  border: 1px solid color-mix(in srgb, #7dd3fc 50%, transparent);
  border-radius: 20px;
  box-shadow: 0 14px 32px rgb(3 105 161 / 22%);
}

.embedded-browser__start h1 { margin: 16px 0 4px; color: var(--text); font-size: clamp(20px, 3vw, 28px); }
.embedded-browser__start > p { margin: 0 0 20px; color: var(--muted); font-size: 12px; }

.embedded-browser__start-form {
  display: flex;
  width: 100%;
  min-height: 50px;
  align-items: center;
  gap: 10px;
  padding: 5px 6px 5px 16px;
  background: var(--surface);
  border: 1px solid color-mix(in srgb, var(--brand) 25%, var(--border));
  border-radius: 16px;
  box-shadow: 0 14px 38px rgb(15 23 42 / 10%);
}

.embedded-browser__start-form > svg { flex: 0 0 auto; color: var(--brand); }
.embedded-browser__start-form input { min-width: 0; flex: 1 1 auto; color: var(--text); background: transparent; border: 0; outline: none; font-size: 14px; }
.embedded-browser__start-form button { height: 38px; padding: 0 17px; color: #fff; background: var(--brand); border: 0; border-radius: 11px; cursor: pointer; white-space: nowrap; }
.embedded-browser__start-form:focus-within { border-color: var(--brand); box-shadow: 0 0 0 4px color-mix(in srgb, var(--brand) 11%, transparent), 0 14px 38px rgb(15 23 42 / 10%); }
.embedded-browser__input-error { margin-top: 8px; color: var(--danger); font-size: 11px; }

.embedded-browser__shortcuts { width: 100%; margin-top: 28px; text-align: left; }
.embedded-browser__shortcuts h2 { margin: 0 0 10px; color: var(--muted); font-size: 11px; font-weight: 650; }
.embedded-browser__shortcuts > div { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; }
.embedded-browser__shortcuts button { display: flex; min-width: 0; align-items: center; gap: 8px; padding: 8px; color: var(--text); background: color-mix(in srgb, var(--surface) 82%, transparent); border: 1px solid var(--border); border-radius: 11px; cursor: pointer; }
.embedded-browser__shortcuts button:hover { border-color: color-mix(in srgb, var(--brand) 34%, var(--border)); background: var(--surface); }
.embedded-browser__shortcuts button > span { position: relative; display: grid; width: 28px; height: 28px; flex: 0 0 auto; place-items: center; color: var(--brand); background: var(--brand-soft); border-radius: 8px; overflow: hidden; }
.embedded-browser__shortcuts img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.embedded-browser__shortcuts strong { overflow: hidden; font-size: 10px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }

.embedded-browser__hint { display: flex; min-width: 0; align-items: center; gap: 6px; padding: 5px 11px; overflow: hidden; color: var(--muted); background: var(--surface); border-top: 1px solid var(--border); font-size: 10px; white-space: nowrap; }
.embedded-browser__hint svg { flex: 0 0 auto; color: var(--success); }
.embedded-browser__hint span { overflow: hidden; text-overflow: ellipsis; }

.embedded-browser__state { display: flex; min-height: 260px; align-items: center; align-self: stretch; justify-content: center; flex-direction: column; gap: 10px; padding: 28px; color: var(--muted); text-align: center; }
.embedded-browser__state > span { display: grid; width: 48px; height: 48px; place-items: center; color: var(--warning); background: var(--warning-soft); border-radius: 14px; }
.embedded-browser__state strong { color: var(--text); font-size: 15px; }
.embedded-browser__state p { max-width: 480px; margin: 0; font-size: 12px; line-height: 1.7; }
.embedded-browser__sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

@container desktop-window (max-width: 820px) {
  .embedded-browser__shortcuts > div { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .embedded-browser__limit { align-items: flex-start; flex-wrap: wrap; }
  .embedded-browser__limit > div { flex-basis: calc(100% - 30px); }
}

@container desktop-window (max-width: 580px) {
  .embedded-browser__tabs { padding-inline: 5px; }
  .embedded-browser__tab { width: 118px; }
  .embedded-browser__external span { display: none; }
  .embedded-browser__external { width: 34px; padding: 0; }
  .embedded-browser__toolbar { gap: 5px; padding-inline: 6px; }
  .embedded-browser__shortcuts > div { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .embedded-browser__start { width: calc(100% - 24px); }
  .embedded-browser__start-form { padding-left: 12px; }
}

@media (prefers-reduced-motion: reduce) {
  .embedded-browser__address,
  .embedded-browser__start-form { transition: none; }
}
</style>
