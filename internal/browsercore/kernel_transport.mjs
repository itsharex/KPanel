const targetURLHeader = 'X-KPanel-Browser-Target-URL'
const targetMethodHeader = 'X-KPanel-Browser-Target-Method'
const targetHeadersHeader = 'X-KPanel-Browser-Target-Headers'
const upstreamStatusHeader = 'X-KPanel-Browser-Upstream-Status'
const upstreamHeadersHeader = 'X-KPanel-Browser-Upstream-Headers'
const sessionChannelName = 'kpanel-browser-session-v1'
const maxRelayedURLBytes = 16 * 1024
const maxRelayedBodyBytes = 16 * 1024 * 1024
const textEncoder = new TextEncoder()

const blockedRequestHeaders = new Set([
  'accept-encoding',
  'connection',
  'content-length',
  'host',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
])

function encodeBase64URL(value) {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function decodeBase64URL(value) {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized + '='.repeat((4 - normalized.length % 4) % 4)
  const binary = atob(padded)
  const bytes = Uint8Array.from(binary, character => character.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

function requestHeaderPairs(headers) {
  const pairs = []
  const entries = Array.isArray(headers) ? headers : Object.entries(headers || {})
  for (const entry of entries) {
    if (!Array.isArray(entry) || entry.length < 2) continue
    const [rawName, rawValue] = entry
    const name = String(rawName)
    if (blockedRequestHeaders.has(name.toLowerCase())) continue
    const values = Array.isArray(rawValue) ? rawValue : [rawValue]
    for (const value of values) pairs.push([name, String(value)])
  }
  if (!pairs.some(([name]) => name.toLowerCase() === 'user-agent')) {
    pairs.push(['User-Agent', navigator.userAgent])
  }
  if (!pairs.some(([name]) => name.toLowerCase() === 'accept-language')) {
    pairs.push(['Accept-Language', navigator.language || 'zh-CN'])
  }
  return pairs
}

function responseHeaderPairs(encoded) {
  const result = []
  const pairs = JSON.parse(decodeBase64URL(encoded || 'W10'))
  if (!Array.isArray(pairs)) throw new TypeError('invalid relay response headers')
  for (const pair of pairs) {
    if (!Array.isArray(pair) || pair.length !== 2) continue
    result.push([String(pair[0]), String(pair[1])])
  }
  return result
}

async function discard(response) {
  try {
    await response.body?.cancel()
  } catch {
    // The stream may already be closed by an abort or a network failure.
  }
}

function byteView(value) {
  if (value instanceof Uint8Array) return value
  if (value instanceof ArrayBuffer) return new Uint8Array(value)
  if (ArrayBuffer.isView(value)) {
    return new Uint8Array(value.buffer, value.byteOffset, value.byteLength)
  }
  throw new TypeError('browser request stream yielded a non-byte chunk')
}

function ensureBodyLimit(size) {
  if (size > maxRelayedBodyBytes) throw new RangeError('browser request body exceeds 16 MiB')
}

async function bufferRequestBody(body, signal) {
  if (body == null) return undefined
  signal?.throwIfAborted()
  if (body instanceof ArrayBuffer) {
    ensureBodyLimit(body.byteLength)
    return body
  }
  if (ArrayBuffer.isView(body)) {
    ensureBodyLimit(body.byteLength)
    return body
  }
  if (!(body instanceof ReadableStream)) return body

  const reader = body.getReader()
  const chunks = []
  let size = 0
  const abort = () => { void reader.cancel(signal?.reason).catch(() => {}) }
  signal?.addEventListener('abort', abort, { once: true })
  try {
    while (true) {
      signal?.throwIfAborted()
      const { value, done } = await reader.read()
      if (done) break
      const chunk = byteView(value)
      size += chunk.byteLength
      ensureBodyLimit(size)
      chunks.push(chunk)
    }
    signal?.throwIfAborted()
  } catch (error) {
    await reader.cancel(error).catch(() => {})
    throw error
  } finally {
    signal?.removeEventListener('abort', abort)
    reader.releaseLock()
  }

  if (size === 0) return undefined
  const buffered = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) {
    buffered.set(chunk, offset)
    offset += chunk.byteLength
  }
  return buffered
}

export default class KPanelRelayTransport {
  constructor(token) {
    if (typeof token !== 'string' || token.length === 0 || token.length > 2048) {
      throw new TypeError('invalid browser session token')
    }
    this.token = token
    this.ready = true
    this.sessionChannel = new BroadcastChannel(sessionChannelName)
  }

  async init() {}

  setToken(token) {
    if (typeof token !== 'string' || token.length === 0 || token.length > 2048) {
      throw new TypeError('invalid browser session token')
    }
    this.token = token
  }

  async request(remote, method, body, headers, signal) {
    const target = remote instanceof URL ? remote : new URL(remote)
    if ((target.protocol !== 'http:' && target.protocol !== 'https:') || textEncoder.encode(target.href).byteLength > maxRelayedURLBytes) {
      throw new TypeError('unsupported browser target')
    }
    const bufferedBody = await bufferRequestBody(body, signal)
    const init = {
      method: 'POST',
      signal,
      headers: {
        Authorization: `Bearer ${this.token}`,
        [targetURLHeader]: target.href,
        [targetMethodHeader]: String(method || 'GET').toUpperCase(),
        [targetHeadersHeader]: encodeBase64URL(JSON.stringify(requestHeaderPairs(headers))),
      },
      body: bufferedBody,
    }

    const response = await fetch('/v1/fetch', init)
    if (response.status === 401) {
      await discard(response)
      this.sessionChannel.postMessage({ type: 'session-expired' })
      throw new Error('browser session expired')
    }
    if (!response.ok) {
      await discard(response)
      throw new Error(`browser relay failed (${response.status})`)
    }
    const status = Number(response.headers.get(upstreamStatusHeader) || 502)
    if (!Number.isInteger(status) || status < 100 || status > 599) {
      await discard(response)
      throw new Error('browser relay returned an invalid status')
    }
    return {
      body: response.body || new ArrayBuffer(0),
      headers: responseHeaderPairs(response.headers.get(upstreamHeadersHeader) || ''),
      status,
      statusText: '',
    }
  }

  connect(_url, _protocols, _headers, _onopen, _onmessage, _onclose, onerror) {
    queueMicrotask(() => onerror('WebSocket is not available in this browser-core build'))
    return [() => {}, () => {}]
  }

  meta() {
    return {}
  }
}
