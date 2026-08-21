const encoder = new TextEncoder()

export const terminalInputFlushInterval = 12

export class TerminalInputQueue {
  private value = ''
  private size = 0

  get empty(): boolean {
    return this.value.length === 0
  }

  get byteLength(): number {
    return this.size
  }

  append(data: string): void {
    if (!data) return
    this.value += data
    this.size += encoder.encode(data).byteLength
  }

  take(maxBytes = 2048): string {
    const { chunk, rest } = takeTerminalInputChunk(this.value, maxBytes)
    this.value = rest
    this.size -= encoder.encode(chunk).byteLength
    return chunk
  }

  restore(chunk: string): void {
    if (!chunk) return
    this.value = chunk + this.value
    this.size += encoder.encode(chunk).byteLength
  }

  clear(): void {
    this.value = ''
    this.size = 0
  }
}

export async function drainTerminalInputQueue(
  queue: TerminalInputQueue,
  canContinue: () => boolean,
  send: (chunk: string) => Promise<void>,
): Promise<void> {
  while (!queue.empty && canContinue()) {
    const chunk = queue.take()
    try {
      await send(chunk)
    } catch (reason) {
      queue.restore(chunk)
      throw reason
    }
  }
}

export function terminalInputShouldFlushImmediately(data: string): boolean {
  for (const character of data) {
    const code = character.codePointAt(0) || 0
    if (code <= 0x1f || code === 0x7f) return true
  }
  return false
}

export function terminalEnterShouldSubmit(
  event: Pick<KeyboardEvent, 'isComposing' | 'keyCode'>,
): boolean {
  return !event.isComposing && event.keyCode !== 229
}

export function terminalLineSubmission(value: string): string {
  return `${value}\r`
}

export function takeTerminalInputChunk(
  value: string,
  maxBytes = 2048,
): { chunk: string; rest: string } {
  if (!value || maxBytes < 1) return { chunk: '', rest: value }
  let bytes = 0
  let characters = 0
  for (const character of value) {
    const size = encoder.encode(character).byteLength
    if (bytes + size > maxBytes) break
    bytes += size
    characters += character.length
  }
  if (characters === 0) {
    const firstCharacter = [...value][0]!
    return {
      chunk: firstCharacter,
      rest: value.slice(firstCharacter.length),
    }
  }
  return {
    chunk: value.slice(0, characters),
    rest: value.slice(characters),
  }
}
