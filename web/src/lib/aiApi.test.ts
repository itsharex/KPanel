import { afterEach, describe, expect, it, vi } from 'vitest'
import { aiApi } from './aiApi'
import { resetApiSecurityState } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
  resetApiSecurityState()
})

describe('AI API client', () => {
  it('sends attachments as multipart files so WAF does not parse base64 JSON', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ runId: 'run-1' }), {
        status: 202,
        headers: { 'content-type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const file = new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], 'screen.png', { type: 'image/png' })

    await aiApi.sessions.send('session-1', '分析图片', [{
      name: file.name,
      mimeType: file.type,
      data: 'iVBORw==',
      file,
      size: file.size,
      kind: 'image',
    }])

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    const headers = new Headers(init.headers)
    expect(url).toBe('/api/v1/ai/sessions/session-1/messages')
    expect(headers.has('content-type')).toBe(false)
    expect(init.body).toBeInstanceOf(FormData)
    const form = init.body as FormData
    expect(form.get('content')).toBe('分析图片')
    expect((form.get('attachments') as File).name).toBe('screen.png')
  })
})
