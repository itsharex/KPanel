(() => {
  'use strict'

  const panelOrigin = document.documentElement.dataset.panelOrigin
  const status = document.getElementById('status')
  const viewport = document.getElementById('viewport')
  const maxDocumentBytes = 8 * 1024 * 1024
  const maxBinaryBytes = 16 * 1024 * 1024
  const maxImageBytes = 2 * 1024 * 1024
  const maxImageTotalBytes = 12 * 1024 * 1024
  const maxImages = 24
  const imageConcurrency = 4
  const allowedTags = new Set([
    'HTML', 'HEAD', 'TITLE', 'BODY', 'MAIN', 'SECTION', 'ARTICLE', 'HEADER', 'FOOTER', 'NAV',
    'ASIDE', 'DIV', 'SPAN', 'P', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'UL', 'OL', 'LI',
    'DL', 'DT', 'DD', 'TABLE', 'THEAD', 'TBODY', 'TFOOT', 'TR', 'TH', 'TD', 'CAPTION',
    'BLOCKQUOTE', 'PRE', 'CODE', 'EM', 'STRONG', 'B', 'I', 'U', 'S', 'SMALL', 'MARK',
    'BR', 'HR', 'A', 'IMG', 'FIGURE', 'FIGCAPTION', 'DETAILS', 'SUMMARY', 'TIME',
  ])

  let accessToken = ''
  let navigationController
  let blobURLs = []

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

  function decodeMetadata(value) {
    if (!value) return []
    const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized + '='.repeat((4 - normalized.length % 4) % 4)
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, character => character.charCodeAt(0))
    return JSON.parse(new TextDecoder().decode(bytes))
  }

  function encodeHeaders(pairs) {
    const bytes = new TextEncoder().encode(JSON.stringify(pairs))
    let binary = ''
    for (const byte of bytes) binary += String.fromCharCode(byte)
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
  }

  async function cancelBody(response) {
    try {
      await response.body?.cancel()
    } catch {
      // The request may already have been aborted or fully consumed.
    }
  }

  async function relayFetch(target, options = {}, signal) {
    const response = await fetch('/v1/fetch', {
      method: 'POST',
      signal,
      headers: {
        Authorization: `Bearer ${accessToken}`,
        'X-KPanel-Browser-Target-URL': target,
        'X-KPanel-Browser-Target-Method': options.method || 'GET',
        'X-KPanel-Browser-Target-Headers': encodeHeaders(options.headers || [
          ['Accept', '*/*'],
          ['Accept-Language', navigator.language || 'zh-CN'],
        ]),
      },
      body: options.body,
    })
    if (response.status === 401) {
      await cancelBody(response)
      send('session-expired')
      throw new Error('浏览器安全会话已过期')
    }
    if (!response.ok) {
      await cancelBody(response)
      throw new Error(`浏览器内核请求失败（${response.status}）`)
    }
    return {
      response,
      status: Number(response.headers.get('X-KPanel-Browser-Upstream-Status') || 502),
      headers: decodeMetadata(response.headers.get('X-KPanel-Browser-Upstream-Headers')),
    }
  }

  function headerValue(headers, name) {
    const pair = headers.find(([key]) => key.toLowerCase() === name.toLowerCase())
    return pair ? pair[1] : ''
  }

  function absoluteURL(value, base) {
    try {
      const target = new URL(value, base)
      return target.protocol === 'http:' || target.protocol === 'https:' ? target.href : ''
    } catch {
      return ''
    }
  }

  function revokeBlobs() {
    for (const value of blobURLs) URL.revokeObjectURL(value)
    blobURLs = []
  }

  async function readBounded(response, limit, oversizedMessage) {
    const reader = response.body?.getReader()
    if (!reader) throw new Error('当前浏览器不支持安全流式读取')
    const chunks = []
    let total = 0
    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        total += value.byteLength
        if (total > limit) {
          await reader.cancel()
          throw new Error(oversizedMessage)
        }
        chunks.push(value)
      }
    } finally {
      reader.releaseLock()
    }
    const bytes = new Uint8Array(total)
    let offset = 0
    for (const chunk of chunks) {
      bytes.set(chunk, offset)
      offset += chunk.byteLength
    }
    return bytes
  }

  function sanitizeHTML(source, baseURL) {
    const parsed = new DOMParser().parseFromString(source, 'text/html')
    const title = parsed.title.trim().slice(0, 160)
    for (const element of [...parsed.querySelectorAll('*')]) {
      if (!allowedTags.has(element.tagName)) {
        if (['SCRIPT', 'STYLE', 'LINK', 'IFRAME', 'OBJECT', 'EMBED', 'SVG', 'MATH'].includes(element.tagName)) {
          element.remove()
        } else {
          element.replaceWith(...element.childNodes)
        }
        continue
      }
      const anchorTarget = element.tagName === 'A' ? absoluteURL(element.getAttribute('href') || '', baseURL) : ''
      const imageTarget = element.tagName === 'IMG' ? absoluteURL(element.getAttribute('src') || '', baseURL) : ''
      for (const attribute of [...element.attributes]) {
        const name = attribute.name.toLowerCase()
        const keep = name === 'title' || name === 'alt' || name === 'lang' || name === 'dir' ||
          name === 'role' || name.startsWith('aria-') || name === 'colspan' || name === 'rowspan'
        if (!keep) element.removeAttribute(attribute.name)
      }
      if (anchorTarget) element.dataset.kpanelHref = anchorTarget
      if (imageTarget) element.dataset.kpanelImage = imageTarget
    }
    const readableStyle = parsed.createElement('style')
    readableStyle.textContent = `
      :root{color-scheme:light dark;font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:Canvas;color:CanvasText}
      body{max-width:1100px;margin:0 auto;padding:24px;line-height:1.65;overflow-wrap:anywhere}
      img{max-width:100%;height:auto}a[data-kpanel-href]{color:#0284c7;text-decoration:underline;cursor:pointer}
      table{max-width:100%;border-collapse:collapse}th,td{padding:6px 8px;border:1px solid color-mix(in srgb,CanvasText 18%,transparent)}
      pre{padding:12px;overflow:auto;background:color-mix(in srgb,CanvasText 7%,transparent);border-radius:8px}
    `
    parsed.head.append(readableStyle)
    return { document: parsed, title }
  }

  async function hydrateImages(documentRoot, signal) {
    const images = [...documentRoot.querySelectorAll('[data-kpanel-image]')].slice(0, maxImages)
    let nextImage = 0
    let totalBytes = 0
    async function worker() {
      while (nextImage < images.length && totalBytes < maxImageTotalBytes) {
        const image = images[nextImage]
        nextImage += 1
        const target = image.dataset.kpanelImage
        if (!target) continue
        try {
          const result = await relayFetch(target, { headers: [['Accept', 'image/*']] }, signal)
          const declared = Number(headerValue(result.headers, 'content-length') || 0)
          if (declared > maxImageBytes || declared + totalBytes > maxImageTotalBytes) {
            await cancelBody(result.response)
            continue
          }
          const bytes = await readBounded(result.response, maxImageBytes, '图片超过安全预算')
          if (bytes.byteLength + totalBytes > maxImageTotalBytes) continue
          totalBytes += bytes.byteLength
          const type = headerValue(result.headers, 'content-type') || 'application/octet-stream'
          const objectURL = URL.createObjectURL(new Blob([bytes], { type }))
          blobURLs.push(objectURL)
          image.src = objectURL
          delete image.dataset.kpanelImage
        } catch (error) {
          if (error?.name === 'AbortError') throw error
        }
      }
    }
    await Promise.all(Array.from(
      { length: Math.min(imageConcurrency, images.length) },
      () => worker(),
    ))
  }

  function connectDocument(documentRoot) {
    documentRoot.addEventListener('click', event => {
      const anchor = event.target.closest?.('[data-kpanel-href]')
      if (!anchor) return
      event.preventDefault()
      navigate(anchor.dataset.kpanelHref)
    })
  }

  async function renderHTML(result, target, signal) {
    const declared = Number(headerValue(result.headers, 'content-length') || 0)
    if (declared > maxDocumentBytes) {
      await cancelBody(result.response)
      throw new Error('网页文档超过 8 MiB 安全预算')
    }
    const bytes = await readBounded(result.response, maxDocumentBytes, '网页文档超过 8 MiB 安全预算')
    const source = new TextDecoder().decode(bytes)
    const sanitized = sanitizeHTML(source, target)
    viewport.srcdoc = '<!doctype html>\n' + sanitized.document.documentElement.outerHTML
    await new Promise(resolve => viewport.addEventListener('load', resolve, { once: true }))
    connectDocument(viewport.contentDocument)
    await hydrateImages(viewport.contentDocument, signal)
    send('title', { title: sanitized.title || new URL(target).hostname })
    hideStatus()
  }

  async function renderBinary(result, target) {
    const declared = Number(headerValue(result.headers, 'content-length') || 0)
    if (declared > maxBinaryBytes) {
      await cancelBody(result.response)
      throw new Error('此资源超过 16 MiB，请使用外部下载')
    }
    const bytes = await readBounded(result.response, maxBinaryBytes, '此资源超过 16 MiB，请使用外部下载')
    const type = headerValue(result.headers, 'content-type') || 'application/octet-stream'
    const objectURL = URL.createObjectURL(new Blob([bytes], { type }))
    blobURLs.push(objectURL)
    viewport.removeAttribute('srcdoc')
    viewport.src = objectURL
    hideStatus()
    send('title', { title: new URL(target).hostname })
  }

  async function navigate(rawTarget, redirectDepth = 0) {
    let target
    try {
      target = new URL(rawTarget).href
    } catch {
      showStatus('地址无效', '请输入完整的 HTTP 或 HTTPS 地址。', true)
      return
    }
    if (!accessToken) {
      showStatus('正在建立安全会话', '浏览器内核尚未收到 KPanel 会话。')
      return
    }
    navigationController?.abort()
    navigationController = new AbortController()
    const signal = navigationController.signal
    revokeBlobs()
    showStatus('正在安全加载', new URL(target).hostname)
    send('navigation', { url: target })
    try {
      const result = await relayFetch(target, {}, signal)
      if (result.status >= 300 && result.status < 400 && redirectDepth < 5) {
        const location = headerValue(result.headers, 'location')
        const redirected = absoluteURL(location, target)
        if (redirected) {
          await cancelBody(result.response)
          return navigate(redirected, redirectDepth + 1)
        }
      }
      const type = headerValue(result.headers, 'content-type').toLowerCase()
      if (type.includes('text/html') || type.includes('application/xhtml+xml')) {
        await renderHTML(result, target, signal)
      } else {
        await renderBinary(result, target)
      }
    } catch (error) {
      if (error?.name === 'AbortError') return
      showStatus('网页加载失败', error instanceof Error ? error.message : '未知错误', true)
      send('error', { message: error instanceof Error ? error.message : '网页加载失败' })
    }
  }

  window.addEventListener('message', event => {
    if (event.origin !== panelOrigin || event.source !== window.parent) return
    const message = event.data
    if (!message || message.type !== 'kpanel-browser:navigate' || typeof message.token !== 'string' ||
      typeof message.url !== 'string' || message.token.length > 2048 || message.url.length > 2048) return
    accessToken = message.token
    navigate(message.url)
  })

  window.addEventListener('beforeunload', revokeBlobs)
  send('ready')
})()
