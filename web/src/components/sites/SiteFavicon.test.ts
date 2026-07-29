import { createSSRApp, effectScope, nextTick, reactive, type Ref } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it, vi } from 'vitest'
import { createSiteFaviconFailureState } from '@/lib/siteFavicon'
import SiteFavicon from './SiteFavicon.vue'

interface SiteFaviconBindings {
  failed: Ref<boolean>
}

async function render(failed = false): Promise<string> {
  const component = SiteFavicon as unknown as {
    setup: (
      props: { siteId: string; domain: string; refreshKey: number },
      context: { expose: () => void },
    ) => SiteFaviconBindings
  }
  const renderable = {
    ...SiteFavicon,
    setup(
      props: { siteId: string; domain: string; refreshKey: number },
      context: { expose: () => void },
    ) {
      const bindings = component.setup(props, context)
      bindings.failed.value = failed
      return bindings
    },
  }
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
  try {
    return await renderToString(
      createSSRApp(renderable as unknown as typeof SiteFavicon, {
        siteId: 'a'.repeat(32),
        domain: 'example.com',
        refreshKey: 0,
      }),
    )
  } finally {
    warn.mockRestore()
  }
}

describe('SiteFavicon', () => {
  it('loads only the authenticated same-origin cache endpoint', async () => {
    const html = await render()

    expect(html).toContain(`/api/v1/sites/${'a'.repeat(32)}/icon`)
    expect(html).toContain('class="site-favicon"')
    expect(html).toContain('alt=""')
    expect(html).toContain('loading="lazy"')
    expect(html).toContain('decoding="async"')
    expect(html).toContain('fetchpriority="low"')
    expect(html).toContain('width="22"')
    expect(html).toContain('height="22"')
    expect(html).not.toContain('http://example.com')
    expect(html).not.toContain('https://example.com')
  })

  it('falls back to the existing globe after an image error', async () => {
    const html = await render(true)

    expect(html).toContain('<svg')
    expect(html).not.toContain('<img')
    expect(html).not.toContain('/api/v1/sites/')
  })

  it('allows one new attempt after a successful list refresh', async () => {
    const props = reactive({
      siteId: 'a'.repeat(32),
      domain: 'example.com',
      refreshKey: 1,
    })
    const scope = effectScope()
    const failed = scope.run(() =>
      createSiteFaviconFailureState(
        () => props.siteId,
        () => props.refreshKey,
      ),
    )
    if (!failed) throw new Error('favicon failure state was not created')

    failed.value = true
    props.refreshKey += 1
    await nextTick()

    expect(failed.value).toBe(false)
    scope.stop()
  })
})
