import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'
import DnsResolutionGuide from './DnsResolutionGuide.vue'

describe('DnsResolutionGuide', () => {
  it('renders copyable A and AAAA targets with official console links', async () => {
    const html = await renderToString(
      createSSRApp(DnsResolutionGuide, {
        ipv4: '203.0.113.10',
        ipv6: '2001:db8::10',
        compact: true,
      }),
    )

    expect(html).toContain('203.0.113.10')
    expect(html).toContain('2001:db8::10')
    expect(html).toContain('https://dash.cloudflare.com/')
    expect(html).toContain('https://dns.console.aliyun.com/')
    expect(html).toContain('https://console.cloud.tencent.com/cns')
    expect(html).toContain('https://console.huaweicloud.com/dns/')
  })

  it('explains how to recover when the public address is unavailable', async () => {
    const html = await renderToString(createSSRApp(DnsResolutionGuide))
    expect(html).toContain('暂未识别本机公网 IP')
  })
})
