import KPanelRelayTransport from '/kernel/runtime/v3/transport.mjs'

const panelOrigin = document.documentElement.dataset.panelOrigin
const status = document.getElementById('status')
const viewport = document.getElementById('viewport')
const navigationTimeoutMs = 45_000
const runtimeStartupTimeoutMs = 15_000
const runtimeWorkerPath = '/kernel/runtime/v3/sw.js'
const sessionChannel = new BroadcastChannel('kpanel-browser-session-v1')

let controller
let frame
let transport
let activeToken = ''
let activeNavigationID = ''
let navigationTimer
let stateTimer
let reportedURL = ''
let reportedTitle = ''
let runtimePromise

function send(type, detail = {}) {
  window.parent.postMessage({ type: `kpanel-browser:${type}`, ...detail }, panelOrigin)
}

function showStatus(title, message, error = false) {
  status.hidden = false
  status.toggleAttribute('data-error', error)
  status.querySelector('strong').textContent = title
  status.querySelector('small').textContent = message
  viewport.removeAttribute('data-ready')
}

function hideStatus() {
  status.hidden = true
  status.removeAttribute('data-error')
  viewport.dataset.ready = 'true'
}

function validTarget(raw) {
  try {
    const target = new URL(raw)
    return (target.protocol === 'http:' || target.protocol === 'https:') && target.href.length <= 2048
      ? target.href
      : ''
  } catch {
    return ''
  }
}

async function registerRuntimeWorker() {
  if (!('serviceWorker' in navigator)) throw new Error('当前浏览器不支持 Service Worker')
  const runtimeWorkerURL = new URL(runtimeWorkerPath, location.href).href
  for (const registration of await navigator.serviceWorker.getRegistrations()) {
    const workers = [registration.installing, registration.waiting, registration.active].filter(Boolean)
    if (workers.some(worker => worker.scriptURL !== runtimeWorkerURL)) {
      await registration.unregister()
    }
  }
  const registration = await navigator.serviceWorker.register(runtimeWorkerPath, {
    scope: '/',
    updateViaCache: 'none',
  })

  const activeWorker = () => registration.active?.scriptURL === runtimeWorkerURL &&
    registration.active.state === 'activated'
      ? registration.active
      : null
  if (activeWorker()) return registration

  await new Promise((resolve, reject) => {
    const timeout = window.setTimeout(() => {
      cleanup()
      reject(new Error('网页重写 Service Worker 激活超时'))
    }, runtimeStartupTimeoutMs)
    const onStateChange = () => {
      if (!activeWorker()) return
      cleanup()
      resolve()
    }
    const watch = worker => worker?.addEventListener('statechange', onStateChange)
    const onUpdateFound = () => watch(registration.installing)
    const cleanup = () => {
      window.clearTimeout(timeout)
      registration.removeEventListener('updatefound', onUpdateFound)
      for (const worker of [registration.installing, registration.waiting, registration.active]) {
        worker?.removeEventListener('statechange', onStateChange)
      }
    }
    registration.addEventListener('updatefound', onUpdateFound)
    for (const worker of [registration.installing, registration.waiting, registration.active]) watch(worker)
    onStateChange()
  })
  return registration
}

function decodedFrameURL() {
  try {
    const location = new URL(viewport.contentWindow.location.href)
    if (!location.pathname.startsWith(frame.prefix)) return ''
    return validTarget(decodeURIComponent(location.pathname.slice(frame.prefix.length)))
  } catch {
    return ''
  }
}

function syncFrameState() {
  if (!frame || !activeNavigationID) return
  const url = decodedFrameURL()
  if (url && url !== reportedURL) {
    reportedURL = url
    send('navigation', { url, navigationId: activeNavigationID })
  }
  try {
    const document = viewport.contentDocument
    const title = document?.title?.trim().slice(0, 160) || ''
    if (title && title !== reportedTitle) {
      reportedTitle = title
      send('title', { title, navigationId: activeNavigationID })
    }
    if (!status.hidden && url && document &&
      (document.readyState !== 'loading' || document.body?.childElementCount > 0)) {
      window.clearTimeout(navigationTimer)
      hideStatus()
    }
  } catch {
    // The rewritten frame can be in the middle of replacing its document.
  }
}

async function ensureRuntime(token) {
  if (runtimePromise) return runtimePromise
  runtimePromise = (async () => {
    const Controller = globalThis.$scramjetController?.Controller
    if (!Controller || !globalThis.$scramjet) throw new Error('网页重写组件未能加载')
    const registration = await registerRuntimeWorker()
    transport = new KPanelRelayTransport(token)
    controller = new Controller({
      serviceworker: registration.active,
      transport,
    })
    await Promise.race([
      controller.wait(),
      new Promise((_, reject) => window.setTimeout(() => {
        reject(new Error('网页重写控制器握手超时'))
      }, runtimeStartupTimeoutMs)),
    ])
    frame = controller.createFrame(viewport)
    viewport.addEventListener('load', () => {
      window.clearTimeout(navigationTimer)
      syncFrameState()
      if (activeNavigationID) hideStatus()
    })
    stateTimer = window.setInterval(syncFrameState, 500)
  })().catch(error => {
    runtimePromise = undefined
    throw error
  })
  return runtimePromise
}

async function useToken(token) {
  if (token === activeToken) return
  transport.setToken(token)
  activeToken = token
}

async function refreshSession(token) {
  await ensureRuntime(token)
  await useToken(token)
  if (frame && activeNavigationID) frame.reload()
}

async function navigate(rawTarget, token, navigationID) {
  const target = validTarget(rawTarget)
  if (!target) {
    showStatus('地址无效', '请输入完整的 HTTP 或 HTTPS 地址。', true)
    send('error', { message: '地址无效', navigationId: navigationID })
    return
  }
  activeNavigationID = navigationID
  showStatus('正在加载网页', new URL(target).hostname)
  try {
    await ensureRuntime(token)
    await useToken(token)
    send('navigation', { url: target, navigationId: navigationID })
    window.clearTimeout(navigationTimer)
    navigationTimer = window.setTimeout(() => {
      showStatus('网页仍在加载', '目标站点响应较慢，可继续等待、刷新或使用系统浏览器打开。', true)
      send('error', { message: '网页加载超时', navigationId: navigationID })
    }, navigationTimeoutMs)
    frame.go(target)
  } catch (error) {
    window.clearTimeout(navigationTimer)
    const message = error instanceof Error ? error.message : '浏览器内核启动失败'
    showStatus('网页加载失败', message, true)
    send('error', { message, navigationId: navigationID })
  }
}

function runCommand(command, navigationID) {
  if (!frame) return
  activeNavigationID = navigationID
  if (command === 'back') frame.back()
  else if (command === 'forward') frame.forward()
  else if (command === 'reload') {
    showStatus('正在刷新网页', '正在重新加载当前页面。')
    frame.reload()
  }
}

window.addEventListener('message', event => {
  if (event.origin !== panelOrigin || event.source !== window.parent) return
  const message = event.data
  if (!message || typeof message !== 'object') return
  if (message.type === 'kpanel-browser:command') {
    if (!['back', 'forward', 'reload'].includes(message.command) ||
      typeof message.navigationId !== 'string' || message.navigationId.length === 0 ||
      message.navigationId.length > 64) return
    runCommand(message.command, message.navigationId)
    return
  }
  if (typeof message.token !== 'string' || message.token.length === 0 || message.token.length > 2048) return
  if (message.type === 'kpanel-browser:update-session') {
    if (runtimePromise) void refreshSession(message.token).catch(() => {})
    return
  }
  if (message.type !== 'kpanel-browser:navigate' || typeof message.url !== 'string' ||
    typeof message.navigationId !== 'string' || message.url.length > 2048 ||
    message.navigationId.length === 0 || message.navigationId.length > 64) return
  void navigate(message.url, message.token, message.navigationId)
})

sessionChannel.addEventListener('message', event => {
  if (event.data?.type === 'session-expired') send('session-expired')
})

window.addEventListener('beforeunload', () => {
  window.clearTimeout(navigationTimer)
  window.clearInterval(stateTimer)
  sessionChannel.close()
})

send('ready')
