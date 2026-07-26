import { describe, expect, it } from 'vitest'
import { customPreset, detectDNSPreset, dnsPresets, parseDNSServers, timezonePresets } from './systemPresets'

describe('system management presets', () => {
  it('parses DNS addresses from common separators', () => {
    expect(parseDNSServers('1.1.1.1, 1.0.0.1\n8.8.8.8；8.8.4.4')).toEqual([
      '1.1.1.1',
      '1.0.0.1',
      '8.8.8.8',
      '8.8.4.4',
    ])
  })

  it('recognizes an exact DNS preset without changing custom input', () => {
    expect(detectDNSPreset('223.5.5.5\n223.6.6.6')).toBe('alidns')
    expect(detectDNSPreset('10.0.0.2')).toBe(customPreset)
  })

  it('keeps preset identifiers and timezone values unique', () => {
    expect(new Set(dnsPresets.map((preset) => preset.value)).size).toBe(dnsPresets.length)
    expect(new Set(timezonePresets.map((preset) => preset.value)).size).toBe(timezonePresets.length)
  })
})
