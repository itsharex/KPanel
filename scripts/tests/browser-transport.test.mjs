import assert from 'node:assert/strict'
import test from 'node:test'

import KPanelRelayTransport from '../../internal/browsercore/kernel_transport.mjs'

const upstreamHeaders = 'W10'

function transportWithoutChannel() {
  const transport = Object.create(KPanelRelayTransport.prototype)
  transport.token = 'test-token'
  return transport
}

async function withFetchCapture(callback) {
  const original = globalThis.fetch
  const calls = []
  globalThis.fetch = async (url, init) => {
    calls.push({ url, init })
    return new Response(new Uint8Array([111, 107]), {
      status: 200,
      headers: {
        'X-KPanel-Browser-Upstream-Status': '200',
        'X-KPanel-Browser-Upstream-Headers': upstreamHeaders,
      },
    })
  }
  try {
    await callback(calls)
  } finally {
    globalThis.fetch = original
  }
}

function targetAtUTF8ByteLimit(multibyte = false) {
  const prefix = multibyte ? 'https://example.com/%E8%B7%AF%E5%BE%84?q=' : 'https://example.com/?q='
  const remaining = 16 * 1024 - new TextEncoder().encode(prefix).byteLength
  assert.ok(remaining > 0)
  const target = new URL(prefix + 'a'.repeat(remaining))
  assert.equal(new TextEncoder().encode(target.href).byteLength, 16 * 1024)
  return target
}

test('buffers transferred request streams before the Relay fetch', async () => {
  await withFetchCapture(async (calls) => {
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new Uint8Array([1, 2]))
        controller.enqueue(new Uint8Array([3, 4]))
        controller.close()
      },
    })
    await transportWithoutChannel().request(
      new URL('https://example.com/post'),
      'POST',
      stream,
      [['Content-Type', 'application/octet-stream']],
    )

    assert.equal(calls.length, 1)
    assert.equal(calls[0].url, '/v1/fetch')
    assert.deepEqual([...calls[0].init.body], [1, 2, 3, 4])
    assert.equal('duplex' in calls[0].init, false)
  })
})

test('turns an empty transferred stream into a non-streaming Relay request', async () => {
  await withFetchCapture(async (calls) => {
    const stream = new ReadableStream({ start(controller) { controller.close() } })
    await transportWithoutChannel().request(new URL('https://example.com/post'), 'POST', stream, [])
    assert.equal(calls[0].init.body, undefined)
    assert.equal('duplex' in calls[0].init, false)
  })
})

test('rejects a transferred request body above the Relay hard limit before fetch', async () => {
  await withFetchCapture(async (calls) => {
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new Uint8Array(16 * 1024 * 1024 + 1))
        controller.close()
      },
    })
    await assert.rejects(
      transportWithoutChannel().request(new URL('https://example.com/upload'), 'POST', stream, []),
      /exceeds 16 MiB/,
    )
    assert.equal(calls.length, 0)
  })
})

test('accepts an exact-limit body and preserves typed-array view boundaries', async () => {
  await withFetchCapture(async (calls) => {
    const storage = new Uint8Array(16 * 1024 * 1024 + 2)
    storage[1] = 7
    storage[storage.length - 2] = 9
    const view = new Uint8Array(storage.buffer, 1, 16 * 1024 * 1024)

    await transportWithoutChannel().request(
      new URL('https://example.com/upload'),
      'POST',
      view,
      [],
    )

    assert.equal(calls.length, 1)
    assert.equal(calls[0].init.body.byteLength, 16 * 1024 * 1024)
    assert.equal(calls[0].init.body[0], 7)
    assert.equal(calls[0].init.body[calls[0].init.body.length - 1], 9)
    assert.equal('duplex' in calls[0].init, false)
  })
})

test('aborts while buffering without sending a partial Relay request', async () => {
  await withFetchCapture(async (calls) => {
    const controller = new AbortController()
    const stream = new ReadableStream({
      start(streamController) {
        streamController.enqueue(new Uint8Array([1, 2, 3]))
      },
      cancel() {},
    })
    controller.abort(new DOMException('cancelled', 'AbortError'))
    await assert.rejects(
      transportWithoutChannel().request(
        new URL('https://example.com/upload'),
        'POST',
        stream,
        [],
        controller.signal,
      ),
      error => error?.name === 'AbortError',
    )
    assert.equal(calls.length, 0)
  })
})

test('cancels a pending stream when the request is aborted during buffering', async () => {
  await withFetchCapture(async (calls) => {
    const controller = new AbortController()
    let cancelled = false
    const stream = new ReadableStream({
      start(streamController) {
        streamController.enqueue(new Uint8Array([1, 2, 3]))
      },
      cancel() { cancelled = true },
    })
    const request = transportWithoutChannel().request(
      new URL('https://example.com/upload'),
      'POST',
      stream,
      [],
      controller.signal,
    )
    queueMicrotask(() => controller.abort(new DOMException('cancelled', 'AbortError')))

    await assert.rejects(request, error => error?.name === 'AbortError')
    assert.equal(cancelled, true)
    assert.equal(calls.length, 0)
  })
})

test('accepts exact-limit ASCII and encoded multibyte target URLs', async () => {
  await withFetchCapture(async (calls) => {
    await transportWithoutChannel().request(targetAtUTF8ByteLimit(false), 'GET', undefined, [])
    await transportWithoutChannel().request(targetAtUTF8ByteLimit(true), 'GET', undefined, [])
    assert.equal(calls.length, 2)
  })
})

test('rejects target URLs above the UTF-8 byte limit before fetch', async () => {
  await withFetchCapture(async (calls) => {
    for (const multibyte of [false, true]) {
      const overLimit = new URL(targetAtUTF8ByteLimit(multibyte).href + 'a')
      await assert.rejects(
        transportWithoutChannel().request(overLimit, 'GET', undefined, []),
        /unsupported browser target/,
      )
    }
    assert.equal(calls.length, 0)
  })
})
