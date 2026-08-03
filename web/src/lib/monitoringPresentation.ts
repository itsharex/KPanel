import type { MonitoringContainerSeries } from '@/types/api'

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
