import { describe, expect, it, vi } from 'vitest'
import type { MonitoringContainerSeries } from '@/types/api'
import {
  isHistoricalContainer,
  nearestTimestamp,
  newestContainerSampleTime,
  svgClientXToViewBox,
  uniqueSeriesTimes,
} from './monitoringPresentation'

function container(id: string, collectedAt?: string): MonitoringContainerSeries {
  return {
    containerId: id,
    name: id,
    image: 'image:latest',
    points: collectedAt ? [{
      collectedAt,
      cpuPercent: 0,
      memoryBytes: 0,
      memoryLimitBytes: 0,
      memoryPercent: 0,
      networkRxBytes: 0,
      networkTxBytes: 0,
      networkRxBytesPerSecond: 0,
      networkTxBytesPerSecond: 0,
      blockReadBytes: 0,
      blockWriteBytes: 0,
      pids: 0,
    }] : [],
  }
}

describe('monitoring presentation', () => {
  it('uses every series timestamp and reaches both edges', () => {
    const times = uniqueSeriesTimes([
      { points: [{ time: 20 }, { time: 30 }] },
      { points: [{ time: 10 }, { time: 30 }] },
    ])
    expect(times).toEqual([10, 20, 30])
    expect(nearestTimestamp(times, -100)).toBe(10)
    expect(nearestTimestamp(times, 100)).toBe(30)
    expect(nearestTimestamp(times, 24)).toBe(20)
  })

  it('uses the SVG screen matrix before the bounding box fallback', () => {
    const matrixTransform = vi.fn(() => ({ x: 64 }))
    const svg = {
      getScreenCTM: () => ({ inverse: () => ({}) }),
      createSVGPoint: () => ({ x: 0, y: 0, matrixTransform }),
      getBoundingClientRect: () => ({ left: 100, width: 900 }),
    } as unknown as SVGSVGElement

    expect(svgClientXToViewBox(svg, 180, 40, 720)).toBe(64)
    expect(matrixTransform).toHaveBeenCalledOnce()
  })

  it('falls back to linear box mapping when no SVG matrix is available', () => {
    const svg = {
      getScreenCTM: () => null,
      getBoundingClientRect: () => ({ left: 100, width: 900 }),
    } as unknown as SVGSVGElement
    expect(svgClientXToViewBox(svg, 550, 0, 720)).toBe(360)
  })

  it('marks only containers older than the newest sample as historical', () => {
    const current = container('current', '2026-07-31T12:10:00Z')
    const old = container('old', '2026-07-31T12:05:00Z')
    const empty = container('empty')
    const newest = newestContainerSampleTime([old, empty, current])

    expect(newest).toBe(Date.parse('2026-07-31T12:10:00Z'))
    expect(isHistoricalContainer(current, newest)).toBe(false)
    expect(isHistoricalContainer(old, newest)).toBe(true)
    expect(isHistoricalContainer(empty, newest)).toBe(true)
  })
})
