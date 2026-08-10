import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const source = readFileSync(new URL('./OverviewView.vue', import.meta.url), 'utf8')
const styles = readFileSync(new URL('../styles/main.css', import.meta.url), 'utf8')

describe('OverviewView service status layout', () => {
  it('uses an explicit details wrapper instead of styling every service item span', () => {
    expect(source).toContain('<span class="service-item__details">')
    expect(styles).toMatch(/\.service-item__details\s*\{[^}]*display:\s*grid;/)
    expect(styles).not.toContain('.service-item > span:not(.service-item__icon)')
  })

  it('places the process manager beside monitoring history without adding sidebar navigation', () => {
    expect(source).toContain('class="realtime-monitoring__actions"')
    expect(source).toContain('to="/processes"')
    expect(source.indexOf('to="/processes"')).toBeLessThan(source.indexOf('to="/monitoring"'))
    expect(styles).toMatch(/\.realtime-monitoring__actions\s*\{[^}]*display:\s*flex;/)
  })

  it('keeps resource dialogs on the overview route and preserves the read-only DNS fact', () => {
    expect(source).toContain('<h2>网络工具</h2>')
    expect(source).not.toContain(['性能', '与网络工具'].join(''))
    expect(source).toContain('<dt>DNS 地址</dt>')
    expect(source).toContain('<HostsManagerDialog')
    expect(source).toContain('<CronManagerDialog')
    expect(source).toContain('<NetworkInterfacesDialog')
    expect(source).toContain('<FirewallManagerDialog')
  })
})
