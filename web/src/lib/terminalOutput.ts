const encoder = new TextEncoder()

const trueBlackBackground = encoder.encode('\x1b[48;2;0;0;0m')
const defaultBackground = encoder.encode('\x1b[49m')

export class TerminalOutputNormalizer {
  private matchLength = 0

  transform(data: string | Uint8Array): Uint8Array {
    const input = typeof data === 'string' ? encoder.encode(data) : data
    const output: number[] = []

    for (const byte of input) {
      if (byte === trueBlackBackground[this.matchLength]) {
        this.matchLength += 1
        if (this.matchLength === trueBlackBackground.length) {
          output.push(...defaultBackground)
          this.matchLength = 0
        }
        continue
      }

      if (this.matchLength > 0) {
        output.push(...trueBlackBackground.subarray(0, this.matchLength))
        this.matchLength = 0
        if (byte === trueBlackBackground[0]) {
          this.matchLength = 1
          continue
        }
      }

      output.push(byte)
    }

    return Uint8Array.from(output)
  }

  flush(): Uint8Array {
    const pending = trueBlackBackground.slice(0, this.matchLength)
    this.matchLength = 0
    return pending
  }

  reset(): void {
    this.matchLength = 0
  }
}
