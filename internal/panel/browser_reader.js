(() => {
  'use strict'

  const status = document.getElementById('reader-status')
  const content = document.getElementById('reader-content')
  const maxNodes = 12_000
  const maxDepth = 128
  const maxHTMLCharacters = 4 * 1024 * 1024
  const maxImages = 24
  const maxImageBytes = 2 * 1024 * 1024
  const maxImageTotalBytes = 12 * 1024 * 1024
  const imageConcurrency = 4
  const allowedImageTypes = new Set(['image/avif', 'image/gif', 'image/jpeg', 'image/png', 'image/webp'])
  const allowedTags = new Set([
    'main', 'section', 'article', 'header', 'footer', 'nav', 'aside', 'div', 'span', 'p',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'dl', 'dt', 'dd',
    'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'caption', 'blockquote',
    'pre', 'code', 'em', 'strong', 'b', 'i', 'u', 's', 'small', 'mark', 'br', 'hr',
    'a', 'img', 'figure', 'figcaption', 'details', 'summary', 'time', 'abbr', 'kbd', 'samp', 'var',
  ])
  const discardedTags = new Set([
    'script', 'style', 'link', 'base', 'meta', 'iframe', 'frame', 'frameset', 'object', 'embed',
    'applet', 'svg', 'math', 'form', 'input', 'button', 'select', 'option', 'textarea', 'label',
    'audio', 'video', 'source', 'track', 'canvas', 'portal', 'noscript', 'template', 'xmp',
    'plaintext', 'noembed', 'noframes',
  ])

  let port
  let currentNavigationID = ''
  let imageSequence = 0
  let imageQueue = []
  let imageInFlight = 0
  let imageTotalBytes = 0
  const pendingImages = new Map()
  let blobURLs = []

  function send(message, transfer = []) {
    if (port) port.postMessage(message, transfer)
  }

  function showStatus(title, message, error = false) {
    status.hidden = false
    status.toggleAttribute('data-error', error)
    status.querySelector('strong').textContent = title
    status.querySelector('small').textContent = message
    content.hidden = true
  }

  function showContent() {
    status.hidden = true
    status.removeAttribute('data-error')
    content.hidden = false
  }

  function revokeBlobs() {
    for (const value of blobURLs) URL.revokeObjectURL(value)
    blobURLs = []
    pendingImages.clear()
    imageQueue = []
    imageInFlight = 0
    imageTotalBytes = 0
  }

  function headerValue(headers, name) {
    if (!Array.isArray(headers)) return ''
    const pair = headers.find(value => Array.isArray(value) && value.length === 2 &&
      typeof value[0] === 'string' && value[0].toLowerCase() === name.toLowerCase())
    return pair && typeof pair[1] === 'string' ? pair[1] : ''
  }

  function absoluteURL(value, base) {
    try {
      const target = new URL(value, base)
      if ((target.protocol !== 'http:' && target.protocol !== 'https:') || target.username || target.password ||
        target.href.length > 2048) return ''
      return target.href
    } catch {
      return ''
    }
  }

  function forEachChild(source, callback) {
    let child = source.firstChild
    while (child) {
      callback(child)
      child = child.nextSibling
    }
  }

  function appendSanitized(source, destination, baseURL, state, depth = 0) {
    if (state.nodes >= maxNodes) return
    if (source.nodeType === Node.TEXT_NODE) {
      state.nodes += 1
      const value = source.nodeValue || ''
      if (value.trim()) state.hasVisibleContent = true
      destination.append(document.createTextNode(value))
      return
    }
    if (depth >= maxDepth || source.nodeType !== Node.ELEMENT_NODE ||
      source.namespaceURI !== 'http://www.w3.org/1999/xhtml') return
    const tag = source.localName.toLowerCase()
    if (discardedTags.has(tag)) return
    if (!allowedTags.has(tag)) {
      forEachChild(source, child => appendSanitized(child, destination, baseURL, state, depth + 1))
      return
    }
    state.nodes += 1
    if (state.nodes > maxNodes) return
    const element = document.createElement(tag)
    for (const name of ['title', 'alt', 'lang', 'dir', 'role', 'aria-label', 'aria-hidden', 'colspan', 'rowspan', 'datetime']) {
      const value = source.getAttribute(name)
      if (value === null || value.length > 512 || /[\u0000\r\n]/.test(value)) continue
      if ((name === 'colspan' || name === 'rowspan') && !/^(?:[1-9]|[1-9][0-9]|100)$/.test(value)) continue
      if (name === 'dir' && !/^(?:ltr|rtl|auto)$/i.test(value)) continue
      element.setAttribute(name, value)
    }
    if (tag === 'a') {
      const href = absoluteURL(source.getAttribute('href') || '', baseURL)
      if (href) element.dataset.kpanelHref = href
    } else if (tag === 'img' && state.images.length < maxImages) {
      const src = absoluteURL(source.getAttribute('src') || '', baseURL)
      if (src) {
        state.images.push({ element, url: src })
        state.hasVisibleContent = true
      }
    }
    forEachChild(source, child => appendSanitized(child, element, baseURL, state, depth + 1))
    destination.append(element)
  }

  function sniffCharset(bytes) {
    let sample = ''
    for (const byte of bytes.subarray(0, 1024)) sample += String.fromCharCode(byte)
    const direct = /<meta\b[^>]*\bcharset\s*=\s*["']?\s*([a-z0-9._:-]{1,40})/i.exec(sample)
    if (direct) return direct[1]
    const legacy = /<meta\b[^>]*\bcontent\s*=\s*["'][^"']*charset\s*=\s*([a-z0-9._:-]{1,40})/i.exec(sample)
    return legacy?.[1] || ''
  }

  function decoderFor(headers, bytes) {
    const type = headerValue(headers, 'content-type')
    const match = /(?:^|;)\s*charset\s*=\s*["']?([^;"'\s]+)/i.exec(type)
    let label = ''
    if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) label = 'utf-8'
    else if (bytes.length >= 2 && bytes[0] === 0xff && bytes[1] === 0xfe) label = 'utf-16le'
    else if (bytes.length >= 2 && bytes[0] === 0xfe && bytes[1] === 0xff) label = 'utf-16be'
    else label = (match?.[1] || sniffCharset(bytes) || 'utf-8').toLowerCase()
    try {
      if (!/^[a-z0-9._:-]{1,40}$/i.test(label)) throw new RangeError('invalid charset')
      return new TextDecoder(label)
    } catch {
      return new TextDecoder()
    }
  }

  function enforceHTMLParseBudget(source) {
    if (source.length > maxHTMLCharacters) throw new Error('网页 HTML 超过安全呈现上限，请使用系统浏览器打开')
    let markers = 0
    for (let index = source.indexOf('<'); index !== -1; index = source.indexOf('<', index + 1)) {
      markers += 1
      if (markers > maxNodes) throw new Error('网页结构超过安全呈现上限，请使用系统浏览器打开')
    }
  }

  function looksLikeHTML(bytes) {
    const sample = new TextDecoder().decode(bytes.subarray(0, 4096)).replace(/^\uFEFF/, '').trimStart()
    return /^(?:<!doctype\s+html\b|<html\b|<head\b|<body\b|<[a-z][^>]*>)/i.test(sample)
  }

  function looksLikeText(bytes) {
    const sample = bytes.subarray(0, 4096)
    let controls = 0
    for (const byte of sample) {
      if (byte === 0) return false
      if (byte < 0x20 && byte !== 0x09 && byte !== 0x0a && byte !== 0x0d) controls += 1
    }
    return controls <= Math.max(2, Math.floor(sample.length * 0.01))
  }

  function pumpImages() {
    while (imageInFlight < imageConcurrency && imageQueue.length && imageTotalBytes < maxImageTotalBytes) {
      const image = imageQueue.shift()
      imageSequence += 1
      const requestID = `image-${imageSequence}`
      pendingImages.set(requestID, image.element)
      imageInFlight += 1
      send({ type: 'resource', kind: 'image', requestId: requestID, navigationId: currentNavigationID, url: image.url })
    }
  }

  function metaRefreshTarget(parsed, target) {
    for (const meta of parsed.querySelectorAll('meta[http-equiv][content]')) {
      if ((meta.getAttribute('http-equiv') || '').trim().toLowerCase() !== 'refresh') continue
      const match = /^\s*(\d+(?:\.\d+)?)\s*;\s*url\s*=\s*(.*?)\s*$/i.exec(meta.getAttribute('content') || '')
      if (!match || Number(match[1]) > 1) continue
      let candidate = match[2].trim()
      if ((candidate.startsWith('"') && candidate.endsWith('"')) ||
        (candidate.startsWith("'") && candidate.endsWith("'"))) candidate = candidate.slice(1, -1).trim()
      const resolved = absoluteURL(candidate, target)
      if (resolved && resolved !== target) return resolved
    }
    return ''
  }

  function renderHTML(bytes, headers, target) {
    const source = decoderFor(headers, bytes).decode(bytes)
    enforceHTMLParseBudget(source)
    const parsed = new DOMParser().parseFromString(source, 'text/html')
    const redirected = metaRefreshTarget(parsed, target)
    if (redirected) {
      send({ type: 'redirect', navigationId: currentNavigationID, url: redirected })
      return
    }
    const fragment = document.createDocumentFragment()
    const state = { nodes: 0, images: [], hasVisibleContent: false }
    forEachChild(parsed.body, child => appendSanitized(child, fragment, target, state))
    if (!state.hasVisibleContent) {
      throw new Error('该网站依赖完整 JavaScript、Cookie 或浏览器验证，请使用系统浏览器打开')
    }
    content.replaceChildren(fragment)
    imageQueue = state.images
    const title = parsed.title.trim().slice(0, 160) || new URL(target).hostname
    showContent()
    send({ type: 'navigation', navigationId: currentNavigationID, url: target })
    send({ type: 'title', navigationId: currentNavigationID, title })
    pumpImages()
  }

  function renderText(bytes, headers, target) {
    const pre = document.createElement('pre')
    pre.textContent = decoderFor(headers, bytes).decode(bytes)
    content.replaceChildren(pre)
    showContent()
    send({ type: 'navigation', navigationId: currentNavigationID, url: target })
    send({ type: 'title', navigationId: currentNavigationID, title: new URL(target).hostname })
  }

  function renderImage(bytes, headers, target) {
    if (bytes.byteLength > maxImageBytes) throw new Error('图片过大，请使用系统浏览器打开')
    const type = headerValue(headers, 'content-type').split(';', 1)[0].trim().toLowerCase()
    if (!allowedImageTypes.has(type)) throw new Error('内置浏览器不支持此资源类型，请使用系统浏览器打开')
    const objectURL = URL.createObjectURL(new Blob([bytes], { type }))
    blobURLs.push(objectURL)
    const image = document.createElement('img')
    image.src = objectURL
    image.alt = new URL(target).hostname
    content.replaceChildren(image)
    showContent()
    send({ type: 'navigation', navigationId: currentNavigationID, url: target })
    send({ type: 'title', navigationId: currentNavigationID, title: new URL(target).hostname })
  }

  function handleRender(message) {
    if (typeof message.navigationId !== 'string' || !message.navigationId || message.navigationId.length > 64 ||
      typeof message.url !== 'string' || message.url.length > 2048 || !(message.body instanceof ArrayBuffer) ||
      !Array.isArray(message.headers) || typeof message.status !== 'number') return
    const target = absoluteURL(message.url, message.url)
    if (!target) return
    currentNavigationID = message.navigationId
    revokeBlobs()
    showStatus('正在安全呈现', new URL(target).hostname)
    try {
      const bytes = new Uint8Array(message.body)
      const type = headerValue(message.headers, 'content-type').toLowerCase()
      if (allowedImageTypes.has(type.split(';', 1)[0].trim())) {
        renderImage(bytes, message.headers, target)
      } else if (type.includes('html') || looksLikeHTML(bytes)) {
        renderHTML(bytes, message.headers, target)
      } else if (type.startsWith('text/') || type.includes('json') || looksLikeText(bytes)) {
        renderText(bytes, message.headers, target)
      } else {
        throw new Error('内置浏览器不支持此资源类型，请使用系统浏览器打开')
      }
    } catch (error) {
      const messageText = error instanceof Error ? error.message : '网页内容呈现失败'
      showStatus('网页呈现失败', messageText, true)
      send({ type: 'error', navigationId: currentNavigationID, message: messageText.slice(0, 512) })
    }
  }

  function handleResource(message) {
    if (message.navigationId !== currentNavigationID || typeof message.requestId !== 'string') return
    const image = pendingImages.get(message.requestId)
    if (!image) return
    pendingImages.delete(message.requestId)
    imageInFlight = Math.max(0, imageInFlight - 1)
    if (message.body instanceof ArrayBuffer && Array.isArray(message.headers) && message.body.byteLength <= maxImageBytes &&
      imageTotalBytes + message.body.byteLength <= maxImageTotalBytes) {
      const type = headerValue(message.headers, 'content-type').split(';', 1)[0].trim().toLowerCase()
      if (allowedImageTypes.has(type)) {
        imageTotalBytes += message.body.byteLength
        const objectURL = URL.createObjectURL(new Blob([message.body], { type }))
        blobURLs.push(objectURL)
        image.src = objectURL
      }
    }
    pumpImages()
  }

  function handlePortMessage(event) {
    const message = event.data
    if (!message || typeof message !== 'object') return
    if (message.type === 'loading' && typeof message.navigationId === 'string' && message.navigationId &&
      message.navigationId.length <= 64 && typeof message.url === 'string' && message.url.length <= 2048) {
      const target = absoluteURL(message.url, message.url)
      if (!target) return
      currentNavigationID = message.navigationId
      revokeBlobs()
      showStatus('正在加载', new URL(target).hostname)
    } else if (message.type === 'render') {
      handleRender(message)
    } else if (message.type === 'resource-result') {
      handleResource(message)
    } else if (message.type === 'error' && message.navigationId === currentNavigationID && typeof message.message === 'string') {
      showStatus('网页加载失败', message.message.slice(0, 512), true)
      send({ type: 'error', navigationId: currentNavigationID, message: message.message.slice(0, 512) })
    }
  }

  content.addEventListener('click', event => {
    const anchor = event.target.closest?.('[data-kpanel-href]')
    if (!anchor) return
    event.preventDefault()
    const target = absoluteURL(anchor.dataset.kpanelHref || '', anchor.dataset.kpanelHref || '')
    if (target) send({ type: 'open', navigationId: currentNavigationID, url: target })
  })

  window.addEventListener('message', event => {
    if (port || event.source !== window.parent || !event.data || event.data.type !== 'kpanel-browser-reader:connect' ||
      event.ports.length !== 1) return
    port = event.ports[0]
    port.onmessage = handlePortMessage
    port.start()
    send({ type: 'ready' })
  })

  window.addEventListener('beforeunload', revokeBlobs)
})()
