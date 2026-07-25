import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, normalizeList, resetApiSecurityState } from './api'

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  const headers = new Headers(init.headers)
  headers.set('content-type', headers.get('content-type') || 'application/json')
  return new Response(JSON.stringify(body), { ...init, headers })
}

afterEach(() => {
  vi.unstubAllGlobals()
  resetApiSecurityState()
})

describe('API client', () => {
  it('detects a server that still requires bootstrap', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(jsonResponse({ required: true }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.auth.status()).resolves.toMatchObject({
      setupRequired: true,
      authenticated: false,
    })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/auth/bootstrap',
      expect.objectContaining({ credentials: 'same-origin', cache: 'no-store' }),
    )
  })

  it('keeps the CSRF token in memory and sends it on mutations', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ required: false }))
      .mockResolvedValueOnce(
        jsonResponse({
          user: { id: 'user-1', username: 'admin', role: 'administrator' },
          csrfToken: 'csrf-secret',
          expiresAt: '2026-07-25T12:00:00Z',
        }),
      )
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.auth.status()).resolves.toMatchObject({
      setupRequired: false,
      authenticated: true,
      user: { username: 'admin' },
    })
    await api.auth.logout()

    const logoutInit = fetchMock.mock.calls[2]?.[1] as RequestInit
    expect(new Headers(logoutInit.headers).get('x-csrf-token')).toBe('csrf-secret')
    expect(logoutInit.method).toBe('POST')
  })

  it('parses the stable problem+json error contract', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValueOnce(
        jsonResponse(
          {
            title: '请求冲突',
            status: 409,
            code: 'resource_version_conflict',
            detail: '资源已被外部修改。',
            requestId: 'req-123',
          },
          { status: 409, headers: { 'content-type': 'application/problem+json' } },
        ),
      ),
    )

    const error = await api.sites.list().catch((reason: unknown) => reason)
    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({
      status: 409,
      code: 'resource_version_conflict',
      message: '资源已被外部修改。',
      requestId: 'req-123',
    })
  })

  it('normalizes list responses without a total field', () => {
    expect(normalizeList({ items: ['a', 'b'] } as { items: string[]; total: number })).toEqual({
      items: ['a', 'b'],
      total: 2,
    })
  })
})
