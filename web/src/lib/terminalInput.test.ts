import { describe, expect, it } from 'vitest'
import {
  drainTerminalInputQueue,
  TerminalInputQueue,
  takeTerminalInputChunk,
  terminalEnterShouldSubmit,
  terminalInputFlushInterval,
  terminalInputShouldFlushImmediately,
  terminalLineSubmission,
} from './terminalInput'

describe('terminal input transport', () => {
  it('flushes enter and control sequences immediately', () => {
    expect(terminalInputShouldFlushImmediately('\r')).toBe(true)
    expect(terminalInputShouldFlushImmediately('\u001b[A')).toBe(true)
    expect(terminalInputShouldFlushImmediately('\u0003')).toBe(true)
    expect(terminalInputShouldFlushImmediately('普通输入')).toBe(false)
  })

  it('submits a precomposed line as one payload terminated by Enter', () => {
    expect(terminalLineSubmission('echo ready')).toBe('echo ready\r')
    expect(terminalLineSubmission('中文输入')).toBe('中文输入\r')
    expect(terminalLineSubmission('')).toBe('\r')
  })

  it('does not submit the composer while Enter is confirming IME text', () => {
    expect(terminalEnterShouldSubmit({ isComposing: true, keyCode: 13 })).toBe(false)
    expect(terminalEnterShouldSubmit({ isComposing: false, keyCode: 229 })).toBe(false)
    expect(terminalEnterShouldSubmit({ isComposing: false, keyCode: 13 })).toBe(true)
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

  it('dequeues before an async send so newly typed input cannot be overwritten', () => {
    const queue = new TerminalInputQueue()
    queue.append('first')

    expect(queue.take()).toBe('first')
    queue.append('second')

    expect(queue.take()).toBe('second')
    expect(queue.empty).toBe(true)
  })

  it('drains input appended while an earlier request is still pending', async () => {
    const queue = new TerminalInputQueue()
    queue.append('first')
    let releaseFirst!: () => void
    const firstPending = new Promise<void>((resolve) => {
      releaseFirst = resolve
    })
    const sent: string[] = []

    const draining = drainTerminalInputQueue(queue, () => true, async (chunk) => {
      sent.push(chunk)
      if (sent.length === 1) await firstPending
    })
    await Promise.resolve()
    queue.append('second')
    releaseFirst()
    await draining

    expect(sent).toEqual(['first', 'second'])
    expect(queue.empty).toBe(true)
  })

  it('restores a failed chunk ahead of input typed while the request was pending', () => {
    const queue = new TerminalInputQueue()
    queue.append('first')
    const pending = queue.take()
    queue.append('second')
    queue.restore(pending)

    expect(queue.take()).toBe('firstsecond')
    expect(queue.empty).toBe(true)
  })

  it('restores a rejected in-flight chunk without losing later input', async () => {
    const queue = new TerminalInputQueue()
    queue.append('first')
    let rejectFirst!: (reason: Error) => void
    const firstPending = new Promise<void>((_resolve, reject) => {
      rejectFirst = reject
    })

    const draining = drainTerminalInputQueue(queue, () => true, () => firstPending)
    await Promise.resolve()
    queue.append('second')
    rejectFirst(new Error('offline'))

    await expect(draining).rejects.toThrow('offline')
    expect(queue.take()).toBe('firstsecond')
  })

  it('tracks UTF-8 queue size without rescanning the whole pending value', () => {
    const queue = new TerminalInputQueue()
    queue.append('a中文🙂')
    expect(queue.byteLength).toBe(new TextEncoder().encode('a中文🙂').byteLength)

    expect(queue.take(4)).toBe('a中')
    expect(queue.byteLength).toBe(new TextEncoder().encode('文🙂').byteLength)
    queue.clear()
    expect(queue.empty).toBe(true)
    expect(queue.byteLength).toBe(0)
  })

  it('uses a sub-frame batching delay for responsive terminal echo', () => {
    expect(terminalInputFlushInterval).toBeLessThan(16)
  })
})
