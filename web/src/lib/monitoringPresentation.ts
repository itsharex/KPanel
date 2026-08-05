import type { MonitoringContainerSeries, MonitoringHistory } from '@/types/api'

export function normalizeTrendChartWidth(measured: number, fallback = 720): number {
  const width = Number.isFinite(measured) && measured > 0 ? measured : fallback
  return Math.max(320, Math.round(width))
}

export function trendLegendLabel(
  latestLabel: string | undefined,
  fallbackValue: number,
  formatter: (value: number) => string,
): string {
  return latestLabel ?? formatter(fallbackValue)
}

interface TimedSeries {
  points: Array<{ time: number }>
}

export function uniqueSeriesTimes(series: TimedSeries[]): number[] {
  return Array.from(new Set(series.flatMap((item) => item.points.map((point) => point.time))))
    .filter(Number.isFinite)
    .sort((left, right) => left - right)
}

export function nearestTimestamp(times: number[], target: number): number | undefined {
  if (!times.length) return undefined
  let low = 0
  let high = times.length - 1
  while (low < high) {
    const middle = Math.floor((low + high) / 2)
    if (times[middle]! < target) low = middle + 1
    else high = middle
  }
  const current = times[low]
  const previous = low > 0 ? times[low - 1] : undefined
  if (current === undefined) return previous
  if (previous === undefined) return current
  return Math.abs(previous - target) <= Math.abs(current - target) ? previous : current
}

export function svgClientXToViewBox(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
  viewBoxWidth: number,
): number | undefined {
  const matrix = svg.getScreenCTM?.()
  if (matrix && typeof svg.createSVGPoint === 'function') {
    try {
      const point = svg.createSVGPoint()
      point.x = clientX
      point.y = clientY
      return point.matrixTransform(matrix.inverse()).x
    } catch {
      // Fall back to the element box when the SVG matrix is temporarily non-invertible.
    }
  }
  const rect = svg.getBoundingClientRect()
  if (!rect.width) return undefined
  return ((clientX - rect.left) / rect.width) * viewBoxWidth
}

export function svgViewBoxXToClient(
  svg: SVGSVGElement,
  viewBoxX: number,
  viewBoxWidth: number,
): number | undefined {
  const matrix = svg.getScreenCTM?.()
  if (matrix && Number.isFinite(matrix.a) && Number.isFinite(matrix.e)) {
    return matrix.a * viewBoxX + matrix.e
  }
  const rect = svg.getBoundingClientRect()
  if (!rect.width || !viewBoxWidth) return undefined
  return rect.left + (viewBoxX / viewBoxWidth) * rect.width
}

export interface TrendTimeSelection {
  start: number
  end: number
}

export function trendSelectionFromViewBox(
  originX: number,
  currentX: number,
  plotLeft: number,
  plotRight: number,
  minimumTime: number,
  maximumTime: number,
  minimumPixels = 12,
): TrendTimeSelection | undefined {
  if (![originX, currentX, plotLeft, plotRight, minimumTime, maximumTime].every(Number.isFinite) ||
    plotRight <= plotLeft || maximumTime <= minimumTime) return undefined
  const clampedOrigin = Math.min(plotRight, Math.max(plotLeft, originX))
  const clampedCurrent = Math.min(plotRight, Math.max(plotLeft, currentX))
  if (Math.abs(clampedCurrent - clampedOrigin) < minimumPixels) return undefined
  const left = Math.min(clampedOrigin, clampedCurrent)
  const right = Math.max(clampedOrigin, clampedCurrent)
  const span = maximumTime - minimumTime
  const timeFor = (value: number) => minimumTime + ((value - plotLeft) / (plotRight - plotLeft)) * span
  return { start: timeFor(left), end: timeFor(right) }
}

function latestContainerTime(container: MonitoringContainerSeries): number | undefined {
  const value = Date.parse(container.points.at(-1)?.collectedAt || '')
  return Number.isFinite(value) ? value : undefined
}

export function newestContainerSampleTime(containers: MonitoringContainerSeries[]): number | undefined {
  let newest: number | undefined
  for (const container of containers) {
    const latest = latestContainerTime(container)
    if (latest !== undefined && (newest === undefined || latest > newest)) newest = latest
  }
  return newest
}

export function isHistoricalContainer(
  container: MonitoringContainerSeries,
  newestSampleTime: number | undefined,
): boolean {
  if (newestSampleTime === undefined) return false
  const latest = latestContainerTime(container)
  return latest === undefined || latest < newestSampleTime
}

export function sliceMonitoringHistory(
  history: MonitoringHistory,
  start: string,
  end: string,
): MonitoringHistory {
  const startTime = Date.parse(start)
  const endTime = Date.parse(end)
  if (!Number.isFinite(startTime) || !Number.isFinite(endTime) || startTime >= endTime) return history
  const inside = (collectedAt: string) => {
    const time = Date.parse(collectedAt)
    return Number.isFinite(time) && time >= startTime && time <= endTime
  }
  return {
    ...history,
    startedAt: start,
    endedAt: end,
    host: history.host.filter((point) => inside(point.collectedAt)),
    containers: history.containers.map((container) => ({
      ...container,
      points: container.points.filter((point) => inside(point.collectedAt)),
    })).filter((container) => container.points.length > 0),
    operatorLatency: history.operatorLatency?.map((series) => ({
      ...series,
      points: series.points.filter((point) => inside(point.collectedAt)),
    })),
  }
}
