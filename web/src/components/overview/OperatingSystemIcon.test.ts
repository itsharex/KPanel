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
    expect(html).toContain('title="Debian"')
    expect(html).toContain('aria-hidden="true"')
    expect(html).not.toContain('role="img"')
    expect(html).not.toContain('aria-label=')
    expect(html).toContain('--os-accent:#A81D33')
    expect(html).not.toContain('<img')
    expect(html).not.toContain('.png')
  })

  it('uses the Linux mark for an unknown distribution', async () => {
    const html = await render('unknown', 'Custom Linux')

    expect(html).toContain('title="Custom Linux"')
    expect(html).toContain('aria-hidden="true"')
    expect(html).toContain('--os-accent:#FCC624')
    expect(html).toContain('--os-foreground:#141816')
  })

  it.each(['__proto__', 'constructor'])(
    'treats inherited object keys such as %s as an unknown distribution',
    async (distro) => {
      const html = await render(distro, 'Untrusted Linux')

      expect(html).toContain('title="Untrusted Linux"')
      expect(html).toContain('aria-hidden="true"')
      expect(html).not.toContain('aria-label=')
      expect(html).toContain('--os-accent:#FCC624')
      expect(html).toContain('--os-foreground:#141816')
    },
  )

  it.each([
    ['rocky', 'Rocky Linux', '#10B981'],
    ['rhel', 'Red Hat Enterprise Linux', '#EE0000'],
    ['manjaro', 'Manjaro', '#35BFA4'],
    ['opensuse', 'openSUSE', '#73BA25'],
    ['suse', 'SUSE Linux Enterprise', '#0C322C'],
  ])('uses the correct %s brand mark', async (distro, label, accent) => {
    const html = await render(distro, label)

    expect(html).toContain(`title="${label}"`)
    expect(html).toContain('aria-hidden="true"')
    expect(html).toContain(`--os-accent:${accent}`)
    expect(html).toContain('<svg')
  })

  it('uses the bundled Oracle mark instead of another distribution brand', async () => {
    const html = await render('oracle', 'Oracle Linux')

    expect(html).toContain('title="Oracle Linux"')
    expect(html).toContain('aria-hidden="true"')
    expect(html).toContain('--os-accent:#C74634')
    expect(html).toContain('<img')
    expect(html).toContain('oracle.png')
  })
})
