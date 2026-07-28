import { describe, expect, it } from 'vitest'
import {
  takeTerminalInputChunk,
  terminalInputShouldFlushImmediately,
} from './terminalInput'

describe('terminal input transport', () => {
  it('flushes enter and control sequences immediately', () => {
    expect(terminalInputShouldFlushImmediately('\r')).toBe(true)
    expect(terminalInputShouldFlushImmediately('\u001b[A')).toBe(true)
    expect(terminalInputShouldFlushImmediately('\u0003')).toBe(true)
    expect(terminalInputShouldFlushImmediately('普通输入')).toBe(false)
  })

  it('chunks UTF-8 input without splitting characters', () => {
    const value = `${'a'.repeat(2046)}中文rest`
    const first = takeTerminalInputChunk(value)
    expect(new TextEncoder().encode(first.chunk).byteLength).toBeLessThanOrEqual(2048)
    expect(first.chunk.endsWith('中')).toBe(false)
    expect(`${first.chunk}${first.rest}`).toBe(value)
  })

  it('keeps large pasted input below the FIFO atomic write target', () => {
    let rest = '输入🙂'.repeat(4000)
    let chunks = 0
    while (rest) {
      const next = takeTerminalInputChunk(rest)
      expect(next.chunk).not.toBe('')
      expect(new TextEncoder().encode(next.chunk).byteLength).toBeLessThanOrEqual(2048)
      rest = next.rest
      chunks += 1
    }
    expect(chunks).toBeGreaterThan(1)
  })
})
