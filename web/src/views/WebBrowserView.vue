<script setup lang="ts">
import { computed, inject, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Globe2,
  Home,
  MoonStar,
  Plus,
  RefreshCw,
  Search,
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
  resolveEmbeddedBrowserStartInput,
  resolveEmbeddedBrowserTarget,
  type EmbeddedBrowserShortcut,
  type EmbeddedBrowserTarget,
} from '@/lib/embeddedBrowser'
import {
  createBrowserCoreCommandMessage,
  createBrowserCoreNavigateMessage,
  createBrowserCoreUpdateSessionMessage,
  isBrowserCoreEvent,
  resolveBrowserCoreLocation,
  type BrowserCoreEvent,
} from '@/lib/embeddedBrowserCore'
import { api } from '@/lib/api'
import { desktopWindowActiveKey, desktopWindowTitlebarTargetKey } from '@/lib/desktopRouteKeys'
import { useI18n } from '@/i18n'
import { usePhraseCatalog } from '@/i18n/phrase'
import type { BrowserCoreSession, BrowserReaderResponse } from '@/types/api'

interface BrowserTab {
  id: string
  title: string
  target?: EmbeddedBrowserTarget
  shortcutID?: string
  iconURL?: string
  frameVersion: number
  lastActiveAt: number
  error?: string
}

interface PendingBrowserRequest {
  target?: EmbeddedBrowserTarget
  shortcut?: EmbeddedBrowserShortcut
}

interface ReaderHistory {
  entries: EmbeddedBrowserTarget[]
  index: number
}

interface ReaderRuntime {
  port: MessagePort
  navigationID: string
  documentController?: AbortController
  resourceController: AbortController
  resourceActive: number
  resourceCount: number
  resourceBytes: number
  stylesheetCount: number
  stylesheetBytes: number
}

const START_PAGE_SHORTCUT_LIMIT = 12
const SEARCH_TAB_TITLE_LIMIT = 64
const route = useRoute()
const i18n = useI18n()
usePhraseCatalog(() => import('@/i18n/pages/WebBrowserView/en-US').then((module) => module.default))

const fallbackShortcuts = ref<EmbeddedBrowserShortcut[]>([])
const browserShortcuts = inject(embeddedBrowserShortcutsKey, fallbackShortcuts)
const fallbackWindowActive = ref(true)
const windowActive = inject(desktopWindowActiveKey, fallbackWindowActive)
const titlebarTarget = inject(desktopWindowTitlebarTargetKey, ref<HTMLElement>())
const startPageShortcuts = computed(() => browserShortcuts.value.slice(0, START_PAGE_SHORTCUT_LIMIT))
const searchEngineName = 'Bing'

const tabs = ref<BrowserTab[]>([])
const activeTabID = ref('')
const liveTabIDs = ref<Set<string>>(new Set())
const pendingRequest = ref<PendingBrowserRequest>()
const addressValue = ref('')
const addressInvalid = ref(false)
const addressEditing = ref(false)
const addressInput = ref<HTMLInputElement>()
const frameColorScheme = ref<'light' | 'dark'>('light')
const coreSession = ref<BrowserCoreSession>()
const coreStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
const coreError = ref('')
const coreLocation = computed(() => coreSession.value
  ? resolveBrowserCoreLocation(coreSession.value)
  : undefined)
const sleepTimers = new Map<string, number>()
const kernelFrames = new Map<string, HTMLIFrameElement>()
const readerRuntimes = new Map<string, ReaderRuntime>()
const readerHistories = new Map<string, ReaderHistory>()
const initializedKernelTabs = new Set<string>()
const activeNavigations = new Map<string, string>()
const readerSessionRetries = new Map<string, number>()
const readerNavigationDeadlines = new Map<string, number>()
const readerSessionRetrying = new Map<string, string>()
const readerDocumentRedirects = new Map<string, number>()
const coreAbortController = new AbortController()
let coreSessionRequest: Promise<BrowserCoreSession | undefined> | undefined
let themeObserver: MutationObserver | undefined
let tabSequence = 0
let navigationSequence = 0

const READER_IMAGE_LIMIT = 24
const READER_IMAGE_BYTES = 2 * 1024 * 1024
const READER_IMAGE_TOTAL_BYTES = 12 * 1024 * 1024
const READER_IMAGE_CONCURRENCY = 4
const READER_STYLESHEET_LIMIT = 16
const READER_STYLESHEET_BYTES = 512 * 1024
const READER_STYLESHEET_TOTAL_BYTES = 2 * 1024 * 1024
const READER_REDIRECT_LIMIT = 5
const READER_NAVIGATION_TIMEOUT_MS = 45_000

function apiErrorCode(error: unknown): string {
  if (!error || typeof error !== 'object' || !('code' in error)) return ''
  return typeof error.code === 'string' ? error.code : ''
}

const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabID.value))
const liveTabs = computed(() => tabs.value.filter((tab) => liveTabIDs.value.has(tab.id)))
const kernelTabs = computed(() => coreStatus.value === 'ready' && coreLocation.value
  ? liveTabs.value
  : [])
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

function closeReaderRuntime(tabID: string): void {
  const runtime = readerRuntimes.get(tabID)
  if (!runtime) return
  runtime.documentController?.abort()
  runtime.resourceController.abort()
  runtime.port.close()
  readerRuntimes.delete(tabID)
}

function sleepTab(tabID: string): void {
  clearSleepTimer(tabID)
  replaceLiveTabIDs((next) => next.delete(tabID))
  activeNavigations.delete(tabID)
  closeReaderRuntime(tabID)
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

async function ensureBrowserCore(force = false): Promise<BrowserCoreSession | undefined> {
  const current = coreSession.value
  if (!force && current && Date.parse(current.expiresAt) > Date.now() + 60_000) return current
  if (coreSessionRequest) return coreSessionRequest
  coreStatus.value = 'loading'
  coreError.value = ''
  const request = api.browser.createSession(coreAbortController.signal)
    .then((session) => {
      if (!resolveBrowserCoreLocation(session)) throw new Error('Invalid browser relay URL')
      coreSession.value = session
      coreStatus.value = 'ready'
      return session
    })
    .catch((error: unknown) => {
      if (coreAbortController.signal.aborted) return undefined
      coreSession.value = undefined
      coreStatus.value = 'error'
      coreError.value = apiErrorCode(error) === 'browser_disabled'
        ? i18n.t('desktop.browserDisabledMessage')
        : error instanceof Error
          ? error.message
          : i18n.t('desktop.browserCoreUnavailableMessage')
      return undefined
    })
    .finally(() => {
      if (coreSessionRequest === request) coreSessionRequest = undefined
    })
  coreSessionRequest = request
  return request
}

function nextNavigationID(tab: BrowserTab): string {
  navigationSequence += 1
  const navigationID = `${tab.id}:${navigationSequence}`
  activeNavigations.set(tab.id, navigationID)
  return navigationID
}

function readerFailure(tab: BrowserTab, navigationID: string, message: string): void {
  if (activeNavigations.get(tab.id) !== navigationID) return
  tab.error = message
  readerRuntimes.get(tab.id)?.port.postMessage({ type: 'error', navigationId: navigationID, message })
}

function refreshReaderSession(tab: BrowserTab, navigationID: string): void {
  if (readerSessionRetrying.get(tab.id) === navigationID) return
  const retries = readerSessionRetries.get(tab.id) || 0
  if (retries >= 1) {
    readerFailure(tab, navigationID, i18n.t('desktop.browserSessionExpired'))
    return
  }
  readerSessionRetries.set(tab.id, retries + 1)
  readerSessionRetrying.set(tab.id, navigationID)
  const deadline = readerNavigationDeadlines.get(tab.id) || Date.now() + READER_NAVIGATION_TIMEOUT_MS
  readerNavigationDeadlines.set(tab.id, deadline)
  const previousSession = coreSession.value
  const controller = new AbortController()
  const abortRefresh = () => controller.abort()
  coreAbortController.signal.addEventListener('abort', abortRefresh, { once: true })
  let timedOut = false
  const timeout = window.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, Math.max(1, deadline - Date.now()))
  void api.browser.createSession(controller.signal)
    .then((session) => {
      if (readerSessionRetrying.get(tab.id) !== navigationID ||
        activeNavigations.get(tab.id) !== navigationID || !tab.target) return
      if (!session || session.mode !== 'reader' || session.token === previousSession?.token) {
        throw new Error(i18n.t('desktop.browserSessionExpired'))
      }
      if (previousSession) Object.assign(previousSession, session)
      else coreSession.value = session
      coreStatus.value = 'ready'
      coreError.value = ''
      postNavigation(tab, true, true)
    })
    .catch((error: unknown) => {
      if (coreAbortController.signal.aborted || readerSessionRetrying.get(tab.id) !== navigationID ||
        activeNavigations.get(tab.id) !== navigationID) return
      const message = timedOut
        ? i18n.t('desktop.browserReaderTimeout')
        : error instanceof Error
          ? error.message
          : i18n.t('desktop.browserSessionExpired')
      readerFailure(tab, navigationID, message)
    })
    .finally(() => {
      window.clearTimeout(timeout)
      coreAbortController.signal.removeEventListener('abort', abortRefresh)
      if (readerSessionRetrying.get(tab.id) === navigationID) readerSessionRetrying.delete(tab.id)
    })
}

function readerHeaderValue(headers: Array<[string, string]>, name: string): string {
  return headers.find(([key]) => key.toLowerCase() === name.toLowerCase())?.[1] || ''
}

function readerHistory(tab: BrowserTab): ReaderHistory {
  let history = readerHistories.get(tab.id)
  if (!history) {
    history = { entries: [], index: -1 }
    readerHistories.set(tab.id, history)
  }
  return history
}

function recordReaderTarget(tab: BrowserTab, target: EmbeddedBrowserTarget): void {
  const history = readerHistory(tab)
  if (history.entries[history.index]?.href === target.href) return
  history.entries.splice(history.index + 1)
  history.entries.push(target)
  history.index = history.entries.length - 1
}

function replaceReaderTarget(tab: BrowserTab, target: EmbeddedBrowserTarget): void {
  const history = readerHistory(tab)
  if (history.index >= 0) history.entries[history.index] = target
}

async function fetchReaderWithRedirects(
  session: BrowserCoreSession,
  initialTarget: EmbeddedBrowserTarget,
  kind: 'document' | 'image' | 'stylesheet',
  signal: AbortSignal,
): Promise<{ target: EmbeddedBrowserTarget; response: BrowserReaderResponse }> {
  let target = initialTarget
  for (let redirects = 0; redirects <= READER_REDIRECT_LIMIT; redirects += 1) {
    const response = await api.browser.fetchReader(session.token, target.href, kind, signal)
    if (response.status < 300 || response.status >= 400) return { target, response }
    const location = readerHeaderValue(response.headers, 'location')
    let redirected: EmbeddedBrowserTarget | undefined
    try {
      redirected = location
        ? resolveEmbeddedBrowserTarget(new URL(location, target.href).href)
        : undefined
    } catch {
      redirected = undefined
    }
    if (!redirected || redirects === READER_REDIRECT_LIMIT) {
      throw new Error(i18n.t('desktop.browserRedirectError'))
    }
    target = redirected
  }
  throw new Error(i18n.t('desktop.browserRedirectError'))
}

async function loadReaderDocument(
  tab: BrowserTab,
  initialTarget: EmbeddedBrowserTarget,
  navigationID: string,
): Promise<void> {
  const runtime = readerRuntimes.get(tab.id)
  const session = coreSession.value
  if (!runtime || session?.mode !== 'reader') return
  runtime.documentController?.abort()
  runtime.resourceController.abort()
  runtime.documentController = new AbortController()
  runtime.resourceController = new AbortController()
  runtime.navigationID = navigationID
  runtime.resourceActive = 0
  runtime.resourceCount = 0
  runtime.resourceBytes = 0
  runtime.stylesheetCount = 0
  runtime.stylesheetBytes = 0
  const controller = runtime.documentController
  const signal = controller.signal
  let timedOut = false
  const deadline = readerNavigationDeadlines.get(tab.id) || Date.now() + READER_NAVIGATION_TIMEOUT_MS
  readerNavigationDeadlines.set(tab.id, deadline)
  const timeout = window.setTimeout(() => {
    timedOut = true
    controller.abort()
  }, Math.max(1, deadline - Date.now()))
  runtime.port.postMessage({ type: 'loading', navigationId: navigationID, url: initialTarget.href })
  try {
    const { target, response } = await fetchReaderWithRedirects(session, initialTarget, 'document', signal)
    if (activeNavigations.get(tab.id) !== navigationID) return
    if (target.href !== initialTarget.href) {
      tab.target = target
      replaceReaderTarget(tab, target)
      if (tab.id === activeTabID.value && !addressEditing.value) addressValue.value = target.href
    }
    tab.error = undefined
    runtime.port.postMessage({
      type: 'render',
      navigationId: navigationID,
      url: target.href,
      status: response.status,
      headers: response.headers,
      body: response.body,
    }, [response.body])
    readerNavigationDeadlines.delete(tab.id)
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError' && !timedOut) return
    if (apiErrorCode(error) === 'browser_session_expired') {
      refreshReaderSession(tab, navigationID)
      return
    }
    if (activeNavigations.get(tab.id) !== navigationID) return
    const message = timedOut
      ? i18n.t('desktop.browserReaderTimeout')
      : error instanceof Error
        ? error.message
        : i18n.t('desktop.browserCoreUnavailableMessage')
    tab.error = message
    runtime.port.postMessage({ type: 'error', navigationId: navigationID, message })
    readerNavigationDeadlines.delete(tab.id)
  } finally {
    window.clearTimeout(timeout)
  }
}

function postNavigation(tab: BrowserTab, force = false, preserveReaderAttempt = false): void {
  const session = coreSession.value
  const location = coreLocation.value
  const frame = kernelFrames.get(tab.id)
  if (!session || !location || !frame?.contentWindow || !tab.target ||
    (!force && initializedKernelTabs.has(tab.id))) return
  if (session.mode === 'reader' && !readerRuntimes.has(tab.id)) return
  tab.error = undefined
  initializedKernelTabs.add(tab.id)
  const navigationID = nextNavigationID(tab)
  if (session.mode === 'reader') {
    if (!preserveReaderAttempt) {
      readerSessionRetries.set(tab.id, 0)
      readerNavigationDeadlines.set(tab.id, Date.now() + READER_NAVIGATION_TIMEOUT_MS)
    }
    recordReaderTarget(tab, tab.target)
    void loadReaderDocument(tab, tab.target, navigationID)
    return
  }
  frame.contentWindow.postMessage(
    createBrowserCoreNavigateMessage(session, tab.target.href, navigationID),
    location.targetOrigin,
  )
}

function setKernelFrame(tabID: string, value: unknown): void {
  if (value instanceof HTMLIFrameElement) {
    if (kernelFrames.get(tabID) !== value) initializedKernelTabs.delete(tabID)
    kernelFrames.set(tabID, value)
    return
  }
  closeReaderRuntime(tabID)
  kernelFrames.delete(tabID)
  initializedKernelTabs.delete(tabID)
  activeNavigations.delete(tabID)
}

function handleBrowserCoreEvent(tab: BrowserTab, message: BrowserCoreEvent | undefined): void {
  if (!message) return
  if (message.type === 'kpanel-browser:ready') {
    postNavigation(tab)
    return
  }
  if (message.type === 'kpanel-browser:session-expired') {
    void refreshBrowserCore()
    return
  }
  if (message.navigationId !== activeNavigations.get(tab.id)) return
  if (message.type === 'kpanel-browser:error') {
    tab.error = message.message
    return
  }
  if (message.type === 'kpanel-browser:title') {
    if (!tab.shortcutID && message.title.trim()) tab.title = message.title.trim()
    return
  }
  const target = resolveEmbeddedBrowserTarget(message.url)
  if (!target) return
  tab.target = target
  if (tab.id === activeTabID.value && !addressEditing.value) addressValue.value = target.href
}

function normalizeReaderCoreEvent(value: unknown): BrowserCoreEvent | undefined {
  if (!value || typeof value !== 'object') return undefined
  const message = value as Record<string, unknown>
  const type = typeof message.type === 'string' ? `kpanel-browser:${message.type}` : ''
  const normalized = { ...message, type }
  return isBrowserCoreEvent(normalized) ? normalized : undefined
}

function handleReaderResource(tab: BrowserTab, message: Record<string, unknown>): void {
  const runtime = readerRuntimes.get(tab.id)
  const session = coreSession.value
  const requestID = typeof message.requestId === 'string' ? message.requestId : ''
  const navigationID = typeof message.navigationId === 'string' ? message.navigationId : ''
  const target = typeof message.url === 'string' ? resolveEmbeddedBrowserTarget(message.url) : undefined
  const kind = message.kind === 'stylesheet' ? 'stylesheet' : 'image'
  if (!runtime || session?.mode !== 'reader' || !requestID || requestID.length > 64 ||
    navigationID !== runtime.navigationID || navigationID !== activeNavigations.get(tab.id)) return
  const overCount = kind === 'stylesheet'
    ? runtime.stylesheetCount >= READER_STYLESHEET_LIMIT
    : runtime.resourceCount >= READER_IMAGE_LIMIT
  if (!target || overCount || runtime.resourceActive >= READER_IMAGE_CONCURRENCY) {
    runtime.port.postMessage({ type: 'resource-result', navigationId: navigationID, requestId: requestID, headers: [] })
    return
  }
  if (kind === 'stylesheet') runtime.stylesheetCount += 1
  else runtime.resourceCount += 1
  runtime.resourceActive += 1
  const resourceController = new AbortController()
  const abortResource = () => resourceController.abort()
  const resourceSignal = runtime.resourceController.signal
  resourceSignal.addEventListener('abort', abortResource, { once: true })
  const timeout = window.setTimeout(abortResource, READER_NAVIGATION_TIMEOUT_MS)
  void fetchReaderWithRedirects(session, target, kind, resourceController.signal)
    .then(({ response }) => {
      const exceedsBudget = kind === 'stylesheet'
        ? response.body.byteLength > READER_STYLESHEET_BYTES ||
          runtime.stylesheetBytes + response.body.byteLength > READER_STYLESHEET_TOTAL_BYTES
        : response.body.byteLength > READER_IMAGE_BYTES ||
          runtime.resourceBytes + response.body.byteLength > READER_IMAGE_TOTAL_BYTES
      if (runtime.navigationID !== navigationID || activeNavigations.get(tab.id) !== navigationID ||
        response.status < 200 || response.status >= 300 || exceedsBudget) {
        runtime.port.postMessage({ type: 'resource-result', navigationId: navigationID, requestId: requestID, headers: [] })
        return
      }
      if (kind === 'stylesheet') runtime.stylesheetBytes += response.body.byteLength
      else runtime.resourceBytes += response.body.byteLength
      runtime.port.postMessage({
        type: 'resource-result',
        navigationId: navigationID,
        requestId: requestID,
        headers: response.headers,
        body: response.body,
      }, [response.body])
    })
    .catch((error: unknown) => {
      if (apiErrorCode(error) === 'browser_session_expired') refreshReaderSession(tab, navigationID)
      if (runtime.navigationID === navigationID && activeNavigations.get(tab.id) === navigationID) {
        runtime.port.postMessage({ type: 'resource-result', navigationId: navigationID, requestId: requestID, headers: [] })
      }
    })
    .finally(() => {
      window.clearTimeout(timeout)
      resourceSignal.removeEventListener('abort', abortResource)
      if (runtime.navigationID === navigationID) runtime.resourceActive = Math.max(0, runtime.resourceActive - 1)
    })
}

function handleReaderPortMessage(tab: BrowserTab, value: unknown): void {
  if (!value || typeof value !== 'object') return
  const message = value as Record<string, unknown>
  if (message.type === 'ready') {
    handleBrowserCoreEvent(tab, { type: 'kpanel-browser:ready' })
    return
  }
  if (message.type === 'open') {
    if (message.navigationId !== activeNavigations.get(tab.id) || typeof message.url !== 'string') return
    const target = resolveEmbeddedBrowserTarget(message.url)
    if (!target) return
    readerDocumentRedirects.delete(tab.id)
    tab.target = target
    tab.shortcutID = undefined
    tab.iconURL = undefined
    tab.title = target.hostname
    if (tab.id === activeTabID.value && !addressEditing.value) addressValue.value = target.href
    postNavigation(tab, true)
    return
  }
  if (message.type === 'redirect') {
    if (typeof message.navigationId !== 'string' || message.navigationId !== activeNavigations.get(tab.id) ||
      typeof message.url !== 'string') return
    const navigationID = message.navigationId
    const target = resolveEmbeddedBrowserTarget(message.url)
    if (!target || target.href === tab.target?.href) return
    const redirects = readerDocumentRedirects.get(tab.id) || 0
    if (redirects >= READER_REDIRECT_LIMIT) {
      readerFailure(tab, navigationID, i18n.t('desktop.browserRedirectError'))
      return
    }
    readerDocumentRedirects.set(tab.id, redirects + 1)
    tab.target = target
    tab.shortcutID = undefined
    tab.iconURL = undefined
    tab.title = target.hostname
    if (tab.id === activeTabID.value && !addressEditing.value) addressValue.value = target.href
    postNavigation(tab, true)
    return
  }
  if (message.type === 'resource') {
    handleReaderResource(tab, message)
    return
  }
  handleBrowserCoreEvent(tab, normalizeReaderCoreEvent(message))
}

function handleKernelLoad(tab: BrowserTab): void {
  const frame = kernelFrames.get(tab.id)
  const session = coreSession.value
  const location = coreLocation.value
  if (!frame?.contentWindow || !session || !location) return
  if (session.mode !== 'reader') {
    postNavigation(tab)
    return
  }
  closeReaderRuntime(tab.id)
  initializedKernelTabs.delete(tab.id)
  const channel = new MessageChannel()
  const runtime: ReaderRuntime = {
    port: channel.port1,
    navigationID: '',
    resourceController: new AbortController(),
    resourceActive: 0,
    resourceCount: 0,
    resourceBytes: 0,
    stylesheetCount: 0,
    stylesheetBytes: 0,
  }
  readerRuntimes.set(tab.id, runtime)
  channel.port1.onmessage = (event) => handleReaderPortMessage(tab, event.data)
  channel.port1.start()
  // The reader intentionally has an opaque origin because its sandbox excludes
  // allow-same-origin. The transferred port is bound to this exact iframe window;
  // subsequent content and credentials travel only over the private channel.
  frame.contentWindow.postMessage({ type: 'kpanel-browser-reader:connect' }, '*', [channel.port2])
}

async function refreshBrowserCore(): Promise<void> {
  const session = await ensureBrowserCore(true)
  const location = coreLocation.value
  if (!session || !location) return
  if (session.mode === 'reader') {
    readerSessionRetries.clear()
    for (const tab of tabs.value) {
      if (readerRuntimes.has(tab.id) && tab.target) postNavigation(tab, true)
    }
    return
  }
  for (const frame of kernelFrames.values()) {
    frame.contentWindow?.postMessage(createBrowserCoreUpdateSessionMessage(session), location.targetOrigin)
  }
}

function handleKernelMessage(event: MessageEvent): void {
  const location = coreLocation.value
  if (coreSession.value?.mode !== 'beta' || !location || event.origin !== location.origin || !isBrowserCoreEvent(event.data)) return
  const tab = tabs.value.find((candidate) => kernelFrames.get(candidate.id)?.contentWindow === event.source)
  if (!tab) return
  handleBrowserCoreEvent(tab, event.data)
}

function mountTab(tab: BrowserTab): void {
  if (!tab.target) return
  clearSleepTimer(tab.id)
  replaceLiveTabIDs((next) => next.add(tab.id))
  enforceLiveLimit()
  void ensureBrowserCore()
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
  readerDocumentRedirects.delete(tab.id)
  tab.target = target
  tab.title = shortcut?.name || target.hostname
  tab.shortcutID = shortcut?.id
  tab.iconURL = shortcut?.iconURL
  activateTab(tab.id)
  postNavigation(tab, true)
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
  closeReaderRuntime(tabID)
  readerHistories.delete(tabID)
  readerSessionRetries.delete(tabID)
  readerNavigationDeadlines.delete(tabID)
  readerSessionRetrying.delete(tabID)
  readerDocumentRedirects.delete(tabID)
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
  addressEditing.value = false
  addressInput.value?.blur()
  addressInvalid.value = false
  const tab = activeTab.value || createStartTab()
  applyTargetToTab(tab, target)
}

function submitStartInput(): void {
  const resolution = resolveEmbeddedBrowserStartInput(addressValue.value)
  if (!resolution) {
    addressInvalid.value = true
    return
  }
  addressInvalid.value = false
  const tab = activeTab.value || createStartTab()
  applyTargetToTab(tab, resolution.target)
  if (resolution.kind === 'search' && resolution.query) {
    tab.title = i18n.t('desktop.browserSearchTabTitle', {
      query: resolution.query.slice(0, SEARCH_TAB_TITLE_LIMIT),
    })
  }
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
  readerHistories.delete(tab.id)
  readerDocumentRedirects.delete(tab.id)
  tab.target = undefined
  tab.shortcutID = undefined
  tab.iconURL = undefined
  tab.title = startPageTitle()
  tab.frameVersion += 1
  addressValue.value = ''
  addressInvalid.value = false
}

function postBrowserCommand(command: 'back' | 'forward' | 'reload'): void {
  const tab = activeTab.value
  const session = coreSession.value
  const location = coreLocation.value
  const frame = tab ? kernelFrames.get(tab.id) : undefined
  if (!tab?.target || !session || !location || !frame?.contentWindow) return
  tab.error = undefined
  if (session.mode === 'reader') {
    readerDocumentRedirects.delete(tab.id)
    const history = readerHistory(tab)
    if (command === 'back') {
      if (history.index <= 0) return
      history.index -= 1
    } else if (command === 'forward') {
      if (history.index >= history.entries.length - 1) return
      history.index += 1
    }
    const target = command === 'reload' ? tab.target : history.entries[history.index]
    if (!target) return
    readerSessionRetries.set(tab.id, 0)
    readerNavigationDeadlines.set(tab.id, Date.now() + READER_NAVIGATION_TIMEOUT_MS)
    const navigationID = nextNavigationID(tab)
    tab.target = target
    if (tab.id === activeTabID.value && !addressEditing.value) addressValue.value = target.href
    void loadReaderDocument(tab, target, navigationID)
    return
  }
  const navigationID = nextNavigationID(tab)
  frame.contentWindow.postMessage(
    createBrowserCoreCommandMessage(command, navigationID),
    location.targetOrigin,
  )
}

function goBack(): void {
  postBrowserCommand('back')
}

function goForward(): void {
  postBrowserCommand('forward')
}

function reload(): void {
  postBrowserCommand('reload')
}

function openExternal(): void {
  const href = activeTab.value?.target?.href
  if (href) window.open(href, '_blank', 'noopener,noreferrer')
}

function hideBrokenIcon(event: Event): void {
  if (event.currentTarget instanceof HTMLImageElement) event.currentTarget.hidden = true
}

function syncFrameColorScheme(): void {
  if (typeof document === 'undefined') return
  frameColorScheme.value = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
}

syncFrameColorScheme()

onMounted(() => {
  window.addEventListener('message', handleKernelMessage)
  themeObserver = new MutationObserver(syncFrameColorScheme)
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme'],
  })
})

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
  coreAbortController.abort()
  window.removeEventListener('message', handleKernelMessage)
  themeObserver?.disconnect()
  for (const tabID of readerRuntimes.keys()) closeReaderRuntime(tabID)
  kernelFrames.clear()
  initializedKernelTabs.clear()
  activeNavigations.clear()
  readerHistories.clear()
  readerSessionRetries.clear()
  readerNavigationDeadlines.clear()
  readerSessionRetrying.clear()
  for (const timer of sleepTimers.values()) window.clearTimeout(timer)
  sleepTimers.clear()
})
</script>

<template>
  <section class="embedded-browser">
    <Teleport :to="titlebarTarget || 'body'" :disabled="!titlebarTarget">
      <nav
        class="embedded-browser__tabs"
        :class="{ 'embedded-browser__tabs--titlebar': titlebarTarget }"
        role="tablist"
        :aria-label="i18n.t('desktop.browserTabsLabel')"
      >
        <div class="embedded-browser__tab-track">
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
              @dblclick.stop
            >
              <img
                v-if="tab.iconURL"
                class="embedded-browser__tab-icon"
                :src="tab.iconURL"
                alt=""
                @error="hideBrokenIcon"
              >
              <Globe2 v-else :size="15" aria-hidden="true" />
              <span>{{ tab.title }}</span>
              <MoonStar
                v-if="tab.target && !liveTabIDs.has(tab.id)"
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
              @dblclick.stop
            >
              <X :size="13" aria-hidden="true" />
            </button>
          </div>
        </div>
        <button
          class="embedded-browser__new-tab"
          type="button"
          :title="i18n.t('desktop.browserNewTab')"
          :aria-label="i18n.t('desktop.browserNewTab')"
          @click="requestNewTab"
          @dblclick.stop
        >
          <Plus :size="16" aria-hidden="true" />
        </button>
        <span class="embedded-browser__tab-count">{{ tabs.length }}/{{ MAX_EMBEDDED_BROWSER_TABS }}</span>
      </nav>
    </Teleport>

    <form class="embedded-browser__toolbar" @submit.prevent="submitAddress">
      <button
        class="embedded-browser__tool"
        type="button"
        data-browser-action="back"
        :disabled="!activeTab?.target"
        :title="i18n.t('desktop.browserBack')"
        :aria-label="i18n.t('desktop.browserBack')"
        @click="goBack"
      >
        <ChevronLeft :size="17" aria-hidden="true" />
      </button>
      <button
        class="embedded-browser__tool"
        type="button"
        data-browser-action="forward"
        :disabled="!activeTab?.target"
        :title="i18n.t('desktop.browserForward')"
        :aria-label="i18n.t('desktop.browserForward')"
        @click="goForward"
      >
        <ChevronRight :size="17" aria-hidden="true" />
      </button>
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
          ref="addressInput"
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
          @focus="addressEditing = true"
          @blur="addressEditing = false"
          @keydown.enter.prevent="submitAddress"
        >
      </label>
      <button
        class="embedded-browser__tool"
        type="button"
        data-browser-action="reload"
        :disabled="!activeTab?.target"
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

    <main
      class="embedded-browser__content"
      :class="{ 'embedded-browser__content--start': !activeTab?.target }"
    >
      <div v-if="!activeTab?.target" class="embedded-browser__start">
        <div class="embedded-browser__start-mark" aria-hidden="true"><Globe2 :size="30" /></div>
        <h1>{{ i18n.t('desktop.browserStartTitle') }}</h1>
        <p>{{ i18n.t('desktop.browserStartDescription', { engine: searchEngineName }) }}</p>
        <form class="embedded-browser__start-form" @submit.prevent="submitStartInput">
          <Search :size="19" aria-hidden="true" />
          <input
            v-model="addressValue"
            type="text"
            inputmode="url"
            maxlength="2048"
            autocomplete="off"
            autocapitalize="off"
            spellcheck="false"
            :placeholder="i18n.t('desktop.browserStartPlaceholder')"
            :aria-label="i18n.t('desktop.browserStartInputLabel')"
            :aria-invalid="addressInvalid"
            autofocus
            @input="addressInvalid = false"
            @keydown.enter.prevent="submitStartInput"
          >
          <button type="submit">{{ i18n.t('desktop.browserGo') }}</button>
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

      <div v-else-if="coreStatus === 'loading' || coreStatus === 'idle'" class="embedded-browser__state" role="status">
        <span><ShieldCheck :size="24" aria-hidden="true" /></span>
        <strong>{{ i18n.t('desktop.browserCoreLoadingTitle') }}</strong>
        <p>{{ i18n.t('desktop.browserCoreLoadingMessage') }}</p>
      </div>

      <div v-else-if="coreStatus === 'error'" class="embedded-browser__state" role="alert">
        <span><TriangleAlert :size="24" aria-hidden="true" /></span>
        <strong>{{ i18n.t('desktop.browserCoreUnavailableTitle') }}</strong>
        <p>{{ coreError || i18n.t('desktop.browserCoreUnavailableMessage') }}</p>
        <div class="embedded-browser__state-actions">
          <button class="button button--primary" type="button" @click="refreshBrowserCore">
            <RefreshCw :size="15" aria-hidden="true" />
            {{ i18n.t('common.retry') }}
          </button>
          <button class="button button--secondary" type="button" @click="openExternal">
            <ExternalLink :size="15" aria-hidden="true" />
            {{ i18n.t('desktop.browserOpenExternal') }}
          </button>
        </div>
      </div>

      <iframe
        v-for="tab in kernelTabs"
        :key="`${tab.id}:${tab.frameVersion}`"
        :ref="(value) => setKernelFrame(tab.id, value)"
        class="embedded-browser__frame"
        :class="{ 'embedded-browser__frame--active': tab.id === activeTabID }"
        :style="{ colorScheme: frameColorScheme }"
        :src="coreLocation?.frameURL"
        :title="tab.title"
        :aria-hidden="tab.id !== activeTabID"
        :tabindex="tab.id === activeTabID ? 0 : -1"
        :sandbox="coreLocation?.sandbox"
        :allow="coreSession?.mode === 'beta' ? 'autoplay; fullscreen; picture-in-picture' : ''"
        :allowfullscreen="coreSession?.mode === 'beta'"
        referrerpolicy="no-referrer"
        @load="handleKernelLoad(tab)"
      />
    </main>

    <footer
      v-if="activeTab?.target && activeTab.error"
      class="embedded-browser__hint embedded-browser__hint--error"
    >
      <TriangleAlert :size="13" aria-hidden="true" />
      <span>{{ activeTab.error }}</span>
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
  background: var(--bg);
  font-size: 13px;
}

.embedded-browser__tabs {
  display: flex;
  min-width: 0;
  min-height: 38px;
  align-items: flex-end;
  gap: 3px;
  padding: 5px 8px 0;
  overflow: hidden;
  background: color-mix(in srgb, var(--surface-raised) 92%, var(--bg));
  border-bottom: 1px solid var(--border);
}

.embedded-browser__tab-track {
  display: flex;
  min-width: 0;
  align-items: flex-end;
  gap: 3px;
  flex: 1 1 auto;
  overflow-x: auto;
  scrollbar-width: none;
}

.embedded-browser__tab-track::-webkit-scrollbar { display: none; }

.embedded-browser__tabs--titlebar {
  width: 100%;
  height: 42px;
  min-height: 42px;
  align-items: center;
  padding: 0 3px 0 5px;
  background: transparent;
  border: 0;
}

.embedded-browser__tabs--titlebar .embedded-browser__tab-track {
  height: 100%;
  align-items: center;
}

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
  font-size: 12px;
  font-weight: 600;
}

.embedded-browser__tab--active {
  color: var(--text);
  background: var(--surface);
  border-color: var(--border);
}

.embedded-browser__tabs--titlebar .embedded-browser__tab {
  height: 30px;
  background: color-mix(in srgb, var(--surface) 72%, transparent);
  border-bottom: 1px solid transparent;
  border-radius: 8px;
}

.embedded-browser__tabs--titlebar .embedded-browser__tab--active {
  background: var(--surface);
  border-color: var(--border);
  box-shadow: 0 1px 2px rgb(0 0 0 / 8%);
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
.embedded-browser__tab-main > svg:first-child,
.embedded-browser__tool > svg,
.embedded-browser__address > svg { stroke-width: 2.15; }
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
.embedded-browser__address input { width: 100%; min-width: 0; color: var(--text); background: transparent; border: 0; outline: none; font-size: 12px; font-weight: 500; }
.embedded-browser__address input::placeholder { color: var(--muted); opacity: 1; }
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

.embedded-browser__content--start {
  overflow-x: hidden;
  overflow-y: auto;
  scrollbar-color: color-mix(in srgb, var(--text) 24%, transparent) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.embedded-browser__content--start::-webkit-scrollbar { width: 9px; }
.embedded-browser__content--start::-webkit-scrollbar-track { background: transparent; }
.embedded-browser__content--start::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--text) 22%, transparent);
  background-clip: padding-box;
  border: 2px solid transparent;
  border-radius: 999px;
}

.embedded-browser__content--start::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--text) 34%, transparent);
  background-clip: padding-box;
}

.embedded-browser__frame {
  grid-area: 1 / 1;
  width: 100%;
  height: 100%;
  min-height: 0;
  visibility: hidden;
  background: Canvas;
  border: 0;
  pointer-events: none;
}

.embedded-browser__frame--active {
  visibility: visible;
  pointer-events: auto;
}

.embedded-browser__start {
  display: flex;
  width: min(680px, calc(100% - 40px));
  min-height: 100%;
  align-items: center;
  justify-self: center;
  flex-direction: column;
  padding: clamp(34px, 9vh, 82px) 0 36px;
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

.embedded-browser__start-mark > svg { stroke-width: 2.2; }
.embedded-browser__start h1 { margin: 16px 0 5px; color: var(--text); font-size: clamp(22px, 3vw, 29px); font-weight: 700; letter-spacing: -.025em; line-height: 1.2; }
.embedded-browser__start > p { margin: 0 0 20px; color: var(--text-soft); font-size: 13px; font-weight: 500; line-height: 1.5; }

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
  box-shadow: var(--shadow-sm);
}

.embedded-browser__start-form > svg { flex: 0 0 auto; color: var(--brand); stroke-width: 2.2; }
.embedded-browser__start-form input { min-width: 0; flex: 1 1 auto; color: var(--text); background: transparent; border: 0; outline: none; font-size: 14px; font-weight: 500; }
.embedded-browser__start-form input::placeholder { color: var(--muted); opacity: 1; }
.embedded-browser__start-form button { height: 38px; padding: 0 17px; color: #fff; background: var(--brand); border: 0; border-radius: 11px; cursor: pointer; font-size: 13px; font-weight: 700; white-space: nowrap; }
.embedded-browser__start-form:focus-within { border-color: var(--brand); box-shadow: 0 0 0 4px color-mix(in srgb, var(--brand) 11%, transparent), var(--shadow-sm); }
.embedded-browser__input-error { margin-top: 8px; color: var(--danger); font-size: 11px; }

.embedded-browser__shortcuts { width: 100%; margin-top: 28px; text-align: left; }
.embedded-browser__shortcuts h2 { margin: 0 0 10px; color: var(--text-soft); font-size: 12px; font-weight: 700; }
.embedded-browser__shortcuts > div { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; }
.embedded-browser__shortcuts button { display: flex; min-width: 0; min-height: 50px; align-items: center; gap: 9px; padding: 8px 9px; color: var(--text); background: var(--surface); border: 1px solid var(--border); border-radius: 11px; box-shadow: var(--shadow-sm); cursor: pointer; transition: border-color .16s ease, background-color .16s ease, transform .16s ease; }
.embedded-browser__shortcuts button:hover { border-color: color-mix(in srgb, var(--brand) 34%, var(--border)); background: var(--surface-raised); transform: translateY(-1px); }
.embedded-browser__shortcuts button > span { position: relative; display: grid; width: 32px; height: 32px; flex: 0 0 auto; place-items: center; color: var(--brand); background: var(--brand-soft); border: 1px solid color-mix(in srgb, var(--brand) 18%, transparent); border-radius: 9px; overflow: hidden; }
.embedded-browser__shortcuts button > span > svg { stroke-width: 2.2; }
.embedded-browser__shortcuts img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.embedded-browser__shortcuts strong { overflow: hidden; font-size: 12px; font-weight: 600; line-height: 1.25; text-overflow: ellipsis; white-space: nowrap; }

.embedded-browser__hint { display: flex; min-width: 0; align-items: center; gap: 6px; padding: 5px 11px; overflow: hidden; color: var(--muted); background: var(--surface); border-top: 1px solid var(--border); font-size: 10px; white-space: nowrap; }
.embedded-browser__hint svg { flex: 0 0 auto; color: var(--success); }
.embedded-browser__hint span { overflow: hidden; text-overflow: ellipsis; }
.embedded-browser__hint--error { color: var(--danger); }
.embedded-browser__hint--error svg { color: var(--danger); }

.embedded-browser__state { display: flex; min-height: 260px; align-items: center; align-self: stretch; justify-content: center; flex-direction: column; gap: 10px; padding: 28px; color: var(--muted); text-align: center; }
.embedded-browser__state > span { display: grid; width: 48px; height: 48px; place-items: center; color: var(--warning); background: var(--warning-soft); border-radius: 14px; }
.embedded-browser__state strong { color: var(--text); font-size: 15px; }
.embedded-browser__state p { max-width: 480px; margin: 0; font-size: 12px; line-height: 1.7; }
.embedded-browser__state-actions { display: flex; align-items: center; justify-content: center; gap: 8px; flex-wrap: wrap; }
.embedded-browser__sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

@container desktop-window (max-width: 820px) {
  .embedded-browser__shortcuts > div { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .embedded-browser__limit { align-items: flex-start; flex-wrap: wrap; }
  .embedded-browser__limit > div { flex-basis: calc(100% - 30px); }
}

@container desktop-window (max-width: 580px) {
  .embedded-browser__tabs:not(.embedded-browser__tabs--titlebar) { padding-inline: 5px; }
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
