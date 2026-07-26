import { describe, expect, it } from 'vitest'
import {
  customPreset,
  detectDNSPreset,
  detectMirrorPreset,
  dnsPresets,
  parseDNSServers,
  timezonePresets,
} from './systemPresets'

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

  it('recognizes mirror products written by kejilion.sh and KPanel', () => {
    expect(detectMirrorPreset(['mirrors.aliyun.com'])).toBe('cn-default')
    expect(detectMirrorPreset(['mirrors.tuna.tsinghua.edu.cn'])).toBe('cn-edu')
    expect(detectMirrorPreset(['mirrors.xtom.hk'])).toBe('abroad')
    expect(detectMirrorPreset(['mirrors.huaweicloud.com'])).toBe('smart')
    expect(detectMirrorPreset(['deb.debian.org', 'security.debian.org'])).toBe('smart')
  })
})
