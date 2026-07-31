export const monitoringMetrics = ['cpu', 'memory', 'disk', 'load', 'network'] as const

export type MonitoringMetric = (typeof monitoringMetrics)[number]

export function normalizeMonitoringMetric(value: unknown): MonitoringMetric | undefined {
  const candidate = Array.isArray(value) ? value[0] : value
  if (typeof candidate !== 'string') return undefined
  return monitoringMetrics.find((metric) => metric === candidate)
}

export function monitoringTargetId(metric: MonitoringMetric): string {
  if (metric === 'cpu' || metric === 'load') return 'host-cpu-load-history'
  if (metric === 'memory' || metric === 'disk') return 'host-memory-disk-history'
  return 'host-network-history'
}
