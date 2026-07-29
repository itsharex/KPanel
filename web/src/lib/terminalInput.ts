const encoder = new TextEncoder()

export function terminalInputShouldFlushImmediately(data: string): boolean {
  for (const character of data) {
    const code = character.codePointAt(0) || 0
    if (code <= 0x1f || code === 0x7f) return true
  }
  return false
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
