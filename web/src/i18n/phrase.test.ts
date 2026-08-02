import { beforeEach, describe, expect, it } from 'vitest'
import {
  registerPhraseCatalog,
  resetPhraseLocalizationForTest,
  translatePhrase,
} from './phrase'

describe('route phrase localization', () => {
  beforeEach(() => {
    resetPhraseLocalizationForTest()
  })

  it('translates exact and parameterized phrases', () => {
    const unregister = registerPhraseCatalog([
      ['应用市场', 'App marketplace'],
      ['已选择 {0} 项', '{0} selected'],
    ])
    expect(translatePhrase('应用市场')).toBe('App marketplace')
    expect(translatePhrase('  已选择 3 项 ')).toBe('  3 selected ')
    unregister()
    expect(translatePhrase('应用市场')).toBe('应用市场')
  })

  it('keeps unknown business and third-party text unchanged', () => {
    expect(translatePhrase('container exited: 权限不足')).toBe('container exited: 权限不足')
  })

  it('keeps command examples unchanged even when a catalog documents them', () => {
    registerPhraseCatalog([
      ['k typecho <域名>', 'k typecho <域名>'],
    ])
    expect(translatePhrase('k typecho <域名>')).toBe('k typecho <域名>')
  })
})
