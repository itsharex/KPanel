import { describe, expect, it } from 'vitest'
import type { MonitoringOperatorLatencySeries } from '@/types/api'
import {
  latestOperatorLatency,
  mergeOperatorLatencyVisibility,
} from './operatorLatencyPresentation'

const series: MonitoringOperatorLatencySeries[] = [
  {
    id: 'telecom-beijing', operator: 'telecom', region: 'beijing', address: '192.0.2.1',
    points: [{ collectedAt: '2026-08-03T00:00:00Z', latencyMilliseconds: 12.5 }],
  },
  {
    id: 'mobile-guangzhou', operator: 'mobile', region: 'guangzhou', address: '192.0.2.2',
    points: [{ collectedAt: '2026-08-03T00:00:00Z', latencyMilliseconds: null }],
  },
]

describe('operator latency presentation', () => {
  it('shows every newly discovered route by default while preserving user choices', () => {
    expect(mergeOperatorLatencyVisibility({}, series)).toEqual({
      'telecom-beijing': true,
      'mobile-guangzhou': true,
    })
    expect(mergeOperatorLatencyVisibility({ 'telecom-beijing': false }, series)).toEqual({
      'telecom-beijing': false,
      'mobile-guangzhou': true,
    })
  })

  it('preserves missing probes', () => {
    expect(latestOperatorLatency(series[0]!)).toBe(12.5)
    expect(latestOperatorLatency(series[1]!)).toBeNull()
  })
})
