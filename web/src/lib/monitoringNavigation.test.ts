import { describe, expect, it } from 'vitest'
import { monitoringTargetId, normalizeMonitoringMetric } from './monitoringNavigation'

describe('monitoring navigation', () => {
  it('accepts only supported metric query values', () => {
    expect(normalizeMonitoringMetric('cpu')).toBe('cpu')
    expect(normalizeMonitoringMetric(['network', 'disk'])).toBe('network')
    expect(normalizeMonitoringMetric('containers')).toBeUndefined()
    expect(normalizeMonitoringMetric(undefined)).toBeUndefined()
  })

  it('maps overview metrics to their history chart', () => {
    expect(monitoringTargetId('cpu')).toBe('host-cpu-load-history')
    expect(monitoringTargetId('load')).toBe('host-cpu-load-history')
    expect(monitoringTargetId('memory')).toBe('host-memory-disk-history')
    expect(monitoringTargetId('disk')).toBe('host-memory-disk-history')
    expect(monitoringTargetId('network')).toBe('host-network-history')
  })
})
