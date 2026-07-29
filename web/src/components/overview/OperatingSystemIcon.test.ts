import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'
import OperatingSystemIcon from './OperatingSystemIcon.vue'

async function render(distro: string, label: string): Promise<string> {
  return renderToString(createSSRApp(OperatingSystemIcon, { distro, label }))
}

describe('OperatingSystemIcon', () => {
  it('renders a lightweight brand-color vector mark', async () => {
    const html = await render('debian', 'Debian')

    expect(html).toContain('<svg')
    expect(html).toContain('aria-label="Debian"')
    expect(html).toContain('--os-accent:#A81D33')
    expect(html).not.toContain('<img')
    expect(html).not.toContain('.png')
  })

  it('uses the Linux mark for an unknown distribution', async () => {
    const html = await render('unknown', 'Custom Linux')

    expect(html).toContain('aria-label="Custom Linux"')
    expect(html).toContain('--os-accent:#FCC624')
    expect(html).toContain('--os-foreground:#141816')
  })

  it.each(['__proto__', 'constructor'])(
    'treats inherited object keys such as %s as an unknown distribution',
    async (distro) => {
      const html = await render(distro, 'Untrusted Linux')

      expect(html).toContain('aria-label="Untrusted Linux"')
      expect(html).toContain('--os-accent:#FCC624')
      expect(html).toContain('--os-foreground:#141816')
    },
  )
})
