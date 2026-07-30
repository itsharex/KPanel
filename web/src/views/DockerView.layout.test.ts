import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const dockerSource = readFileSync(new URL('./DockerView.vue', import.meta.url), 'utf8')

describe('Docker resource toolbar layout', () => {
  it('keeps resource actions aligned to the right on desktop', () => {
    expect(dockerSource).toContain('.workspace-card > header:not(.resource-section__header)')
    expect(dockerSource).toMatch(/\.resource-section__header\s*\{[^}]*align-items:\s*center;/)
    expect(dockerSource).toMatch(/\.resource-section__header > \.card-actions\s*\{[^}]*margin-left:\s*auto;[^}]*justify-content:\s*flex-end;/)
  })

  it('lets resource actions wrap below the title on narrower screens', () => {
    expect(dockerSource).toMatch(/@media \(max-width: 1000px\)[\s\S]*?\.resource-section__header > \.card-actions\s*\{[^}]*width:\s*100%;[^}]*margin-left:\s*0;[^}]*flex-wrap:\s*wrap;/)
  })

  it('exposes right-click menus for every daily Docker resource', () => {
    expect(dockerSource).toContain('@contextmenu="showContainerContext($event, container)"')
    expect(dockerSource).toContain('@contextmenu="showImageContext($event, image)"')
    expect(dockerSource).toContain('@contextmenu="showNetworkContext($event, network)"')
    expect(dockerSource).toContain('@contextmenu="showVolumeContext($event, volume)"')
    expect(dockerSource).toContain('class="docker-context-menu"')
    expect(dockerSource).toContain('role="menu"')
  })
})
