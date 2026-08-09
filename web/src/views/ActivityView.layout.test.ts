import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const source = readFileSync(new URL('./ActivityView.vue', import.meta.url), 'utf8')

describe('activity page layout', () => {
  it('does not stretch the tab row inside a desktop window', () => {
    expect(source).toMatch(/\.activity-page\s*\{[^}]*align-content:\s*start;/)
  })
})
