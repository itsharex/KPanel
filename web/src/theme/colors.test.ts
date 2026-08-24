import { describe, expect, it } from 'vitest'
import {
  DEFAULT_THEME_COLORS,
  THEME_COLOR_KEYS,
  THEME_TOKEN_NAMES,
  contrastRatio,
  deriveThemeTokens,
  normalizeHexColor,
  normalizeThemeColors,
  parseStoredThemeColors,
  serializeThemeColors,
  type ThemeColorIntent,
  type ThemeMode,
  type ThemeTokenMap,
} from './colors'

const HEX_TOKENS = [
  '--bg', '--surface', '--surface-subtle', '--surface-raised',
  '--text', '--text-soft', '--muted', '--border', '--border-strong', '--control-border',
  '--brand', '--brand-strong', '--brand-soft', '--brand-muted', '--theme-accent', '--on-brand',
  '--success', '--success-soft', '--blue', '--blue-soft', '--violet', '--violet-soft',
  '--amber', '--amber-soft', '--danger', '--danger-soft', '--on-danger', '--neutral-soft',
  '--sidebar', '--sidebar-text', '--sidebar-muted', '--sidebar-border', '--sidebar-hover',
  '--sidebar-active', '--sidebar-accent', '--desktop-label', '--scrollbar-track', '--scrollbar-thumb',
  '--scrollbar-thumb-hover', '--scrollbar-thumb-active',
] as const

const AA_PAIRS = [
  ['--text', '--surface'], ['--text-soft', '--surface'], ['--muted', '--surface'],
  ['--muted', '--surface-subtle'], ['--muted', '--neutral-soft'],
  ['--brand', '--surface'], ['--brand', '--brand-soft'], ['--brand-strong', '--surface'],
  ['--brand-strong', '--brand-soft'], ['--success', '--surface'], ['--success', '--success-soft'],
  ['--blue', '--surface'], ['--blue', '--blue-soft'], ['--violet', '--surface'],
  ['--violet', '--violet-soft'], ['--amber', '--surface'], ['--amber', '--amber-soft'],
  ['--danger', '--surface'], ['--danger', '--danger-soft'], ['--sidebar-text', '--sidebar'],
  ['--sidebar-muted', '--sidebar'], ['--theme-accent', '--sidebar'],
  ['--sidebar-accent', '--sidebar'],
  ['--on-brand', '--brand'], ['--on-brand', '--brand-strong'], ['--on-danger', '--danger'],
] as const

function expectAccessible(tokens: ThemeTokenMap): void {
  expect(Object.keys(tokens)).toEqual([...THEME_TOKEN_NAMES])
  for (const token of HEX_TOKENS) expect(tokens[token]).toMatch(/^#[0-9a-f]{6}$/)
  for (const [foreground, background] of AA_PAIRS) {
    expect(
      contrastRatio(tokens[foreground], tokens[background]),
      `${foreground}/${background}`,
    ).toBeGreaterThanOrEqual(4.5)
  }
  expect(contrastRatio(tokens['--control-border'], tokens['--surface'])).toBeGreaterThanOrEqual(3)
  expect(contrastRatio(tokens['--control-border'], tokens['--surface-raised'])).toBeGreaterThanOrEqual(3)
  expect(contrastRatio(tokens['--theme-accent'], tokens['--surface'])).toBeGreaterThanOrEqual(3)
  expect(contrastRatio(tokens['--theme-accent'], tokens['--surface-raised'])).toBeGreaterThanOrEqual(3)
  for (const value of Object.values(tokens)) {
    expect(value).not.toMatch(/[;{}]|url\(|var\(|NaN|undefined/i)
  }
}

describe('theme color input contract', () => {
  it('publishes a fixed three-intent color key list', () => {
    expect(THEME_COLOR_KEYS).toEqual(['brand', 'neutral', 'signature'])
    expect(DEFAULT_THEME_COLORS).toEqual({
      brand: '#0c7a60',
      neutral: '#52645f',
      signatureLinked: true,
      signature: '#0c7a60',
    })
  })

  it('normalizes safe short and long hexadecimal colors only', () => {
    expect(normalizeHexColor('#ABC')).toBe('#aabbcc')
    expect(normalizeHexColor('  #Aa10Ff  ')).toBe('#aa10ff')
    for (const invalid of [null, 42, '', 'fff', '#ffff', '#ffffffff', 'red', 'var(--brand)', 'url(x)', '#12;bad']) {
      expect(normalizeHexColor(invalid)).toBeNull()
    }
  })

  it('fills partial or invalid live input from the safe defaults', () => {
    expect(normalizeThemeColors({ brand: '#123', neutral: 'bad', signatureLinked: false })).toEqual({
      brand: '#112233',
      neutral: DEFAULT_THEME_COLORS.neutral,
      signatureLinked: false,
      signature: DEFAULT_THEME_COLORS.signature,
    })
    expect(normalizeThemeColors(null)).toEqual(DEFAULT_THEME_COLORS)
  })

  it('round-trips the explicit version-one storage shape', () => {
    const colors: ThemeColorIntent = {
      brand: '#123456',
      neutral: '#657483',
      signatureLinked: false,
      signature: '#d1a35f',
    }
    const serialized = serializeThemeColors(colors)
    expect(JSON.parse(serialized)).toEqual({ version: 1, ...colors })
    expect(parseStoredThemeColors(serialized)).toEqual(colors)
  })

  it('rejects malformed, oversized, future, incomplete, and injected storage', () => {
    const invalid = [
      null,
      '',
      'null',
      '[]',
      '{',
      JSON.stringify({ version: 2, ...DEFAULT_THEME_COLORS }),
      JSON.stringify({ version: 1, brand: '#fff', neutral: '#000', signatureLinked: 'yes', signature: '#fff' }),
      JSON.stringify({ version: 1, brand: 'var(--x)', neutral: '#000', signatureLinked: true, signature: '#fff' }),
      JSON.stringify({ version: 1, ...DEFAULT_THEME_COLORS, extra: '#ffffff' }),
      ' '.repeat(513),
    ]
    for (const value of invalid) expect(parseStoredThemeColors(value)).toBeNull()
  })

  it('calculates canonical WCAG sRGB contrast and rejects non-colors', () => {
    expect(contrastRatio('#000', '#fff')).toBe(21)
    expect(contrastRatio('#777777', '#ffffff')).toBeCloseTo(4.478, 3)
    expect(() => contrastRatio('red', '#fff')).toThrow(TypeError)
  })
})

describe('derived custom theme', () => {
  it.each(['light', 'dark'] as const)('derives an exact, safe token allowlist in %s mode', (mode) => {
    expectAccessible(deriveThemeTokens(DEFAULT_THEME_COLORS, mode))
  })

  it('uses one neutral intent for visibly different light and dark foundations', () => {
    const light = deriveThemeTokens(DEFAULT_THEME_COLORS, 'light')
    const dark = deriveThemeTokens(DEFAULT_THEME_COLORS, 'dark')
    expect(light['--bg']).not.toBe(dark['--bg'])
    expect(light['--surface']).not.toBe(dark['--surface'])
    expect(light['--sidebar']).not.toBe(dark['--sidebar'])
    expect(light['--on-brand']).toBe('#ffffff')
    expect(dark['--on-brand']).toBe('#0b111b')
  })

  it('derives sidebar hover feedback from brand intent instead of neutral text', () => {
    const blue = deriveThemeTokens({ ...DEFAULT_THEME_COLORS, brand: '#315dbe' }, 'dark')
    const coral = deriveThemeTokens({ ...DEFAULT_THEME_COLORS, brand: '#b64f3f' }, 'dark')
    expect(blue['--sidebar']).toBe(coral['--sidebar'])
    expect(blue['--sidebar-text']).toBe(coral['--sidebar-text'])
    expect(blue['--sidebar-hover']).not.toBe(coral['--sidebar-hover'])
  })

  it('links signature to brand without allowing the dormant signature value to leak', () => {
    const first = deriveThemeTokens({
      ...DEFAULT_THEME_COLORS,
      brand: '#315d7d',
      signatureLinked: true,
      signature: '#ff0000',
    }, 'light')
    const second = deriveThemeTokens({
      ...DEFAULT_THEME_COLORS,
      brand: '#315d7d',
      signatureLinked: true,
      signature: '#00ffff',
    }, 'light')
    expect(first).toEqual(second)
    expect(first['--theme-accent']).toBe(first['--sidebar-accent'])
  })

  it('uses an independent signature only for signature surfaces', () => {
    const linked = deriveThemeTokens({
      ...DEFAULT_THEME_COLORS,
      brand: '#315d7d',
      signatureLinked: true,
      signature: '#d0aa65',
    }, 'dark')
    const independent = deriveThemeTokens({
      ...DEFAULT_THEME_COLORS,
      brand: '#315d7d',
      signatureLinked: false,
      signature: '#d0aa65',
    }, 'dark')
    expect(independent['--brand']).toBe(linked['--brand'])
    expect(independent['--theme-accent']).not.toBe(linked['--theme-accent'])
    expect(independent['--theme-accent']).toBe(independent['--sidebar-accent'])
  })

  it('searches accent lightness in both directions for sidebar text and surface marks', () => {
    const light = deriveThemeTokens({
      ...DEFAULT_THEME_COLORS,
      signatureLinked: false,
      signature: '#ffffff',
    }, 'light')
    const dark = deriveThemeTokens({
      ...DEFAULT_THEME_COLORS,
      signatureLinked: false,
      signature: '#000000',
    }, 'dark')

    expect(light['--theme-accent']).not.toBe('#ffffff')
    expect(dark['--theme-accent']).not.toBe('#000000')
    for (const tokens of [light, dark]) {
      expect(contrastRatio(tokens['--theme-accent'], tokens['--sidebar'])).toBeGreaterThanOrEqual(4.5)
      expect(contrastRatio(tokens['--theme-accent'], tokens['--surface'])).toBeGreaterThanOrEqual(3)
      expect(contrastRatio(tokens['--theme-accent'], tokens['--surface-raised'])).toBeGreaterThanOrEqual(3)
    }
  })

  const extremes: ThemeColorIntent[] = [
    { brand: '#000000', neutral: '#000000', signatureLinked: true, signature: '#000000' },
    { brand: '#ffffff', neutral: '#ffffff', signatureLinked: true, signature: '#ffffff' },
    { brand: '#000000', neutral: '#ffffff', signatureLinked: false, signature: '#ffffff' },
    { brand: '#ffffff', neutral: '#000000', signatureLinked: false, signature: '#000000' },
    { brand: '#ff0000', neutral: '#00ff00', signatureLinked: false, signature: '#0000ff' },
    { brand: '#00ffff', neutral: '#ff00ff', signatureLinked: false, signature: '#ffff00' },
    { brand: '#808080', neutral: '#808080', signatureLinked: true, signature: '#808080' },
  ]

  it.each(extremes.flatMap((colors) => [
    { colors, mode: 'light' as const },
    { colors, mode: 'dark' as const },
  ]))('keeps extreme input accessible in $mode mode: $colors', ({ colors, mode }) => {
    expectAccessible(deriveThemeTokens(colors, mode))
  })

  it('stays deterministic and accessible across a seeded random color sample', () => {
    let state = 0x6d2b79f5
    const randomHex = () => {
      state = (Math.imul(state ^ (state >>> 15), 1 | state) + 0x6d2b79f5) | 0
      const value = (state ^ (state >>> 14)) >>> 0
      return `#${(value & 0xffffff).toString(16).padStart(6, '0')}`
    }
    for (let index = 0; index < 96; index += 1) {
      const colors: ThemeColorIntent = {
        brand: randomHex(),
        neutral: randomHex(),
        signatureLinked: index % 2 === 0,
        signature: randomHex(),
      }
      for (const mode of ['light', 'dark'] as const satisfies readonly ThemeMode[]) {
        const first = deriveThemeTokens(colors, mode)
        expect(first).toEqual(deriveThemeTokens(colors, mode))
        expectAccessible(first)
      }
    }
  })
})
