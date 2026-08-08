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
})
