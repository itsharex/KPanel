import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'
// jsdom is a pinned Vitest dependency; its standalone package does not ship TypeScript declarations.
// @ts-expect-error -- the runtime API is exercised directly in this test.
import { JSDOM } from 'jsdom'
import { describe, expect, it, vi } from 'vitest'

interface FakeReaderPort {
  onmessage?: (event: { data: unknown }) => void
  postMessage: ReturnType<typeof vi.fn>
  start: ReturnType<typeof vi.fn>
}

const readerScript = readFileSync(fileURLToPath(new URL('../../../internal/panel/browser_reader.js', import.meta.url)), 'utf8')

function readerHarness() {
  const dom = new JSDOM(`<!doctype html><html><body>
    <section id="reader-status"><strong></strong><small></small></section>
    <main id="reader-content" hidden></main>
  </body></html>`, {
    runScripts: 'outside-only',
    url: 'https://panel.example.com/browser-reader/',
  })
  Object.defineProperty(dom.window, 'TextDecoder', { configurable: true, value: TextDecoder })
  const port: FakeReaderPort = { postMessage: vi.fn(), start: vi.fn() }
  dom.window.eval(readerScript)
  dom.window.dispatchEvent(new dom.window.MessageEvent('message', {
    data: { type: 'kpanel-browser-reader:connect' },
    source: dom.window,
    ports: [port as unknown as MessagePort],
  }))
  expect(port.postMessage).toHaveBeenCalledWith({ type: 'ready' }, [])
  port.postMessage.mockClear()

  function realmBuffer(bytes: Uint8Array): ArrayBuffer {
    const buffer = new dom.window.ArrayBuffer(bytes.byteLength)
    new dom.window.Uint8Array(buffer).set(bytes)
    return buffer as ArrayBuffer
  }

  function render(body: Uint8Array, headers: Array<[string, string]>, url = 'https://example.com/page') {
    port.onmessage?.({
      data: {
        type: 'render',
        navigationId: 'navigation-1',
        url,
        status: 200,
        headers,
        body: realmBuffer(body),
      },
    })
  }

  return { dom, port, render }
}

describe('browser reader runtime', () => {
  it('rebuilds only allowed content and keeps target resources off the frame network', () => {
    const { dom, port, render } = readerHarness()
    const source = new TextEncoder().encode(`<!doctype html><title>Safe title</title>
      <main><h1 onclick="alert(1)">Safe heading</h1>
      <script>window.parent.postMessage('escaped', '*')</script>
      <form action="https://target.example/write"><input name="secret"></form>
      <img src="https://target.example/direct.png" onerror="alert(1)">
      <a href="/next" ping="https://target.example/ping">Next</a></main>`)
    render(source, [['Content-Type', 'text/html; charset=utf-8']])

    const content = dom.window.document.getElementById('reader-content')!
    expect(content.textContent).toContain('Safe heading')
    expect(content.querySelector('script, form, input')).toBeNull()
    expect(content.querySelector('h1')?.hasAttribute('onclick')).toBe(false)
    expect(content.querySelector('img')?.getAttribute('src')).toBeNull()
    expect(port.postMessage).toHaveBeenCalledWith(expect.objectContaining({
      type: 'resource',
      kind: 'image',
      url: 'https://target.example/direct.png',
    }), [])

    const anchor = content.querySelector('a')!
    expect(anchor.getAttribute('href')).toBeNull()
    anchor.dispatchEvent(new dom.window.MouseEvent('click', { bubbles: true, cancelable: true }))
    expect(port.postMessage).toHaveBeenCalledWith(expect.objectContaining({
      type: 'open',
      url: 'https://example.com/next',
    }), [])
    dom.window.close()
  })

  it('rejects node-flood markup before invoking DOMParser', () => {
    const { dom, port, render } = readerHarness()
    const parser = vi.spyOn(dom.window.DOMParser.prototype, 'parseFromString')
    render(new TextEncoder().encode(`<main>${'<br>'.repeat(12_001)}</main>`), [
      ['Content-Type', 'text/html; charset=utf-8'],
    ])
    expect(parser).not.toHaveBeenCalled()
    expect(port.postMessage).toHaveBeenCalledWith(expect.objectContaining({
      type: 'error',
      message: expect.stringContaining('结构超过安全呈现上限'),
    }), [])
    dom.window.close()
  })

  it('honors browser-supported legacy charset aliases', () => {
    const { dom, render } = readerHarness()
    render(Uint8Array.from([0xd6, 0xd0, 0xce, 0xc4]), [['Content-Type', 'text/plain; charset=gb2312']])
    expect(dom.window.document.getElementById('reader-content')?.textContent).toBe('中文')
    dom.window.close()
  })
})
