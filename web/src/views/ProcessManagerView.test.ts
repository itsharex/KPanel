import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const source = readFileSync(new URL('./ProcessManagerView.vue', import.meta.url), 'utf8')

describe('ProcessManagerView performance contract', () => {
  it('keeps collection bounded and uses completion-based polling', () => {
    expect(source).toContain('const processLimit = 200')
    expect(source).toContain('const refreshIntervalMilliseconds = 2_000')
    expect(source).toContain('window.setTimeout(() => void load(true), refreshIntervalMilliseconds)')
    expect(source).not.toContain('setInterval(')
  })

  it('stops work when hidden and guards termination with process identity', () => {
    expect(source).toContain("document.visibilityState === 'visible'")
    expect(source).toContain('desktopWindowActive.value')
    expect(source).toContain('controller?.abort()')
    expect(source).toContain('startTimeTicks: process.startTimeTicks')
    expect(source).toContain("signal: pendingSignal.value")
  })
})
