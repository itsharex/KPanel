const numberFormatter = new Intl.NumberFormat('zh-CN', {
  maximumFractionDigits: 1,
})

const dateFormatter = new Intl.DateTimeFormat('zh-CN', {
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
})

export function formatPercent(value?: number): string {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return `${numberFormatter.format(Math.max(0, value))}%`
}

export function formatBytes(value?: number, decimals = 1): string {
  if (value === undefined || !Number.isFinite(value)) return '—'
  if (value === 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const base = 1024
  const index = Math.min(Math.floor(Math.log(Math.abs(value)) / Math.log(base)), units.length - 1)
  const scaled = value / Math.pow(base, index)

  return `${scaled.toFixed(index === 0 ? 0 : decimals)} ${units[index]}`
}

export function formatRate(value?: number): string {
  const formatted = formatBytes(value)
  return formatted === '—' ? formatted : `${formatted}/s`
}

export function formatDuration(totalSeconds?: number): string {
  if (totalSeconds === undefined || !Number.isFinite(totalSeconds) || totalSeconds < 0) return '—'

  const seconds = Math.floor(totalSeconds)
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
  if (minutes > 0) return `${minutes} 分钟`
  return `${seconds} 秒`
}

export function formatDateTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : dateFormatter.format(date)
}

export function relativeTime(value?: string, now = Date.now()): string {
  if (!value) return '从未'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知'

  const deltaSeconds = Math.round((date.getTime() - now) / 1000)
  const absolute = Math.abs(deltaSeconds)
  const formatter = new Intl.RelativeTimeFormat('zh-CN', { numeric: 'auto' })

  if (absolute < 60) return formatter.format(deltaSeconds, 'second')
  if (absolute < 3600) return formatter.format(Math.round(deltaSeconds / 60), 'minute')
  if (absolute < 86400) return formatter.format(Math.round(deltaSeconds / 3600), 'hour')
  return formatter.format(Math.round(deltaSeconds / 86400), 'day')
}

export function clampPercent(value?: number): number {
  if (value === undefined || !Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

export function shortId(value?: string, length = 12): string {
  if (!value) return '—'
  return value.length <= length ? value : value.slice(0, length)
}
