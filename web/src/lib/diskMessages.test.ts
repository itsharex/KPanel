import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import englishSharedCatalog from '@/i18n/pages/shared/en-US'
import { registerPhraseCatalog, resetPhraseLocalizationForTest, translatePhrase } from '@/i18n/phrase'
import {
  diskProblemMessage,
  diskRuntimeMessageSources,
  localizeDiskRuntimeMessage,
} from './diskMessages'

beforeEach(() => {
  resetPhraseLocalizationForTest()
  registerPhraseCatalog(englishSharedCatalog)
})

afterEach(() => resetPhraseLocalizationForTest())

describe('disk runtime messages', () => {
  it('keeps the complete privileged v1 vocabulary localized in English', () => {
    for (const source of diskRuntimeMessageSources) {
      const sample = source.replaceAll('{0}', 'lsblk')
      expect(translatePhrase(sample), source).not.toMatch(/[\u3400-\u9fff]/)
    }
  })

  it('uses stable problem copy and safely degrades an unknown Chinese outcome', () => {
    expect(diskProblemMessage('disk_partition_conflict')).toContain('磁盘状态已变化')
    expect(localizeDiskRuntimeMessage('未来脚本新增的未知结果', 'en-US', translatePhrase))
      .toBe('The disk task returned an unrecognized status. Refresh and review it.')
  })
})
