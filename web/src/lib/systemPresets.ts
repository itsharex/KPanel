export const customPreset = '__custom__'
export type MirrorPreset = 'cn-default' | 'cn-edu' | 'abroad' | 'smart'

export const dnsPresets = [
  {
    value: 'cloudflare',
    label: 'Cloudflare（全球）',
    servers: ['1.1.1.1', '1.0.0.1'],
    ipv6Servers: ['2606:4700:4700::1111', '2606:4700:4700::1001'],
  },
  {
    value: 'google',
    label: 'Google Public DNS（全球）',
    servers: ['8.8.8.8', '8.8.4.4'],
    ipv6Servers: ['2001:4860:4860::8888', '2001:4860:4860::8844'],
  },
  {
    value: 'alidns',
    label: '阿里云公共 DNS（中国大陆）',
    servers: ['223.5.5.5', '223.6.6.6'],
    ipv6Servers: ['2400:3200::1', '2400:3200:baba::1'],
  },
  {
    value: 'dnspod',
    label: '腾讯 DNSPod（中国大陆）',
    servers: ['119.29.29.29', '182.254.116.116'],
    ipv6Servers: ['2402:4e00::', '2402:4e00:1::'],
  },
  {
    value: 'quad9',
    label: 'Quad9（安全拦截）',
    servers: ['9.9.9.9', '149.112.112.112'],
    ipv6Servers: ['2620:fe::fe', '2620:fe::9'],
  },
] as const

export type DNSPreset = (typeof dnsPresets)[number]

export const timezonePresets = [
  { value: 'Asia/Shanghai', label: '中国大陆 · 上海（Asia/Shanghai）' },
  { value: 'Asia/Hong_Kong', label: '中国香港（Asia/Hong_Kong）' },
  { value: 'Asia/Taipei', label: '中国台北（Asia/Taipei）' },
  { value: 'Asia/Singapore', label: '新加坡（Asia/Singapore）' },
  { value: 'Asia/Tokyo', label: '日本 · 东京（Asia/Tokyo）' },
  { value: 'Asia/Seoul', label: '韩国 · 首尔（Asia/Seoul）' },
  { value: 'Asia/Kolkata', label: '印度 · 加尔各答（Asia/Kolkata）' },
  { value: 'Asia/Dubai', label: '阿联酋 · 迪拜（Asia/Dubai）' },
  { value: 'Europe/London', label: '英国 · 伦敦（Europe/London）' },
  { value: 'Europe/Berlin', label: '德国 · 柏林（Europe/Berlin）' },
  { value: 'Europe/Paris', label: '法国 · 巴黎（Europe/Paris）' },
  { value: 'America/New_York', label: '美国东部 · 纽约（America/New_York）' },
  { value: 'America/Chicago', label: '美国中部 · 芝加哥（America/Chicago）' },
  { value: 'America/Denver', label: '美国山地 · 丹佛（America/Denver）' },
  { value: 'America/Los_Angeles', label: '美国西部 · 洛杉矶（America/Los_Angeles）' },
  { value: 'America/Toronto', label: '加拿大 · 多伦多（America/Toronto）' },
  { value: 'America/Sao_Paulo', label: '巴西 · 圣保罗（America/Sao_Paulo）' },
  { value: 'Australia/Sydney', label: '澳大利亚 · 悉尼（Australia/Sydney）' },
  { value: 'Pacific/Auckland', label: '新西兰 · 奥克兰（Pacific/Auckland）' },
  { value: 'Etc/UTC', label: '协调世界时（Etc/UTC）' },
] as const

export function parseDNSServers(value: string): string[] {
  return value
    .split(/[\s,，;；]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function detectDNSPreset(value: string): string {
  const servers = parseDNSServers(value)
  return dnsPresets.find((preset) =>
    [dnsServersForPreset(preset), dnsServersForPreset(preset, true)]
      .some((candidate) => candidate.join('\n') === servers.join('\n')),
  )?.value || customPreset
}

export function dnsServersForPreset(preset: DNSPreset, includeIPv6 = false): string[] {
  return includeIPv6 ? [...preset.servers, ...preset.ipv6Servers] : [...preset.servers]
}

const educationMirrorHosts = [
  'pku.edu.cn',
  'bjtu.edu.cn',
  'bfsu.edu.cn',
  'bupt.edu.cn',
  'cqu.edu.cn',
  'cqupt.edu.cn',
  'uestc.cn',
  'scau.edu.cn',
  'hust.edu.cn',
  'jlu.edu.cn',
  'nju.edu.cn',
  'njtech.edu.cn',
  'njupt.edu.cn',
  'sustech.edu.cn',
  'tuna.tsinghua.edu.cn',
  'sdu.edu.cn',
  'shanghaitech.edu.cn',
  'sjtu.edu.cn',
  'xjtu.edu.cn',
  'nwafu.edu.cn',
  'zju.edu.cn',
  'ustc.edu.cn',
]

const abroadMirrorHosts = [
  'xtom.',
  '.hk',
  '.sg',
  '.tw',
  '.jp',
  '.de',
  '.nl',
  '.ee',
  '.uk',
  '.au',
  'kernel.org',
  'osuosl.org',
  'princeton.edu',
  'nus.edu.sg',
]

export function detectMirrorPreset(sources: string[]): MirrorPreset {
  if (sources.some((source) => source.includes('huaweicloud.com'))) return 'smart'
  if (sources.some((source) => educationMirrorHosts.some((host) => source.includes(host)))) return 'cn-edu'
  if (
    sources.some((source) =>
      abroadMirrorHosts.some((host) => host.endsWith('.') ? source.includes(host) : source.endsWith(host)),
    )
  ) {
    return 'abroad'
  }
  if (
    sources.some((source) =>
      ['aliyun.com', 'tencent.com', '163.com', 'ctyun.cn', 'volces.com'].some((host) => source.includes(host)),
    )
  ) {
    return 'cn-default'
  }
  return 'smart'
}
