import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const component = readFileSync(new URL('./PageHeader.vue', import.meta.url), 'utf8')

describe('compact page heading contract', () => {
  it('delegates the visible heading to the classic shell or desktop window', () => {
    expect(component).not.toContain('<h1')
    expect(component).not.toContain('<p')
    expect(component).not.toContain('<header')
    expect(component).toContain('desktop window title owns the page heading')
  })
})
