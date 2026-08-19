// @vitest-environment node
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const css = readFileSync(new URL('./main.css', import.meta.url), 'utf8')

describe('AI workspace typography contract', () => {
  it('does not add semantic AI text below the 12px product floor', () => {
    const offenders: string[] = []
    const rules = css.replace(/\/\*[\s\S]*?\*\//g, '').matchAll(/([^{}]+)\{([^{}]*)\}/g)

    for (const match of rules) {
      const selector = match[1]!.trim()
      if (!selector.includes('.ai-')) continue
      for (const declaration of match[2]!.matchAll(/font-size\s*:\s*(\d+(?:\.\d+)?)px/g)) {
        if (Number(declaration[1]!) < 12) offenders.push(`${selector}: ${declaration[1]}px`)
      }
    }

    expect(offenders).toEqual([])
  })

  it('keeps the conversation body and composer on the 14px reading baseline', () => {
    expect(css).toMatch(/\.ai-markdown\s*\{[^}]*font-size:\s*14px/)
    expect(css).toMatch(/\.ai-composer textarea\s*\{[^}]*font-size:\s*14px/)
    expect(css).toMatch(/\.ai-choice__trigger\s*\{[^}]*font-size:\s*14px/)
  })
})
