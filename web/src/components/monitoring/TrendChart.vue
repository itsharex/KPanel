<script setup lang="ts">
import { computed } from 'vue'

export interface TrendSeries {
  label: string
  color: string
  points: Array<{ at: string; value: number }>
}

const props = withDefaults(defineProps<{
  series: TrendSeries[]
  formatter?: (value: number) => string
  zeroBased?: boolean
}>(), {
  formatter: (value: number) => value.toFixed(1),
  zeroBased: true,
})

const width = 720
const height = 190
const padding = { top: 18, right: 12, bottom: 26, left: 12 }

const allPoints = computed(() => props.series.flatMap((item) => item.points))
const times = computed(() => allPoints.value.map((item) => new Date(item.at).getTime()).filter(Number.isFinite))
const values = computed(() => allPoints.value.map((item) => item.value).filter(Number.isFinite))
const minimumTime = computed(() => Math.min(...times.value))
const maximumTime = computed(() => Math.max(...times.value))
const minimumValue = computed(() => props.zeroBased ? 0 : Math.min(...values.value))
const maximumValue = computed(() => {
  const maximum = Math.max(...values.value)
  return maximum <= minimumValue.value ? minimumValue.value + 1 : maximum
})
const hasData = computed(() => times.value.length > 0 && values.value.length > 0)

function linePath(points: TrendSeries['points']): string {
  if (!hasData.value) return ''
  const timeSpan = Math.max(1, maximumTime.value - minimumTime.value)
  const valueSpan = Math.max(1, maximumValue.value - minimumValue.value)
  return points
    .filter((point) => Number.isFinite(point.value) && Number.isFinite(new Date(point.at).getTime()))
    .map((point, index) => {
      const x = padding.left +
        ((new Date(point.at).getTime() - minimumTime.value) / timeSpan) *
        (width - padding.left - padding.right)
      const y = padding.top +
        (1 - (point.value - minimumValue.value) / valueSpan) *
        (height - padding.top - padding.bottom)
      return `${index === 0 ? 'M' : 'L'}${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
}

function lastValue(series: TrendSeries): number {
  return series.points.at(-1)?.value ?? 0
}

function timeLabel(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <div class="trend-chart">
    <div class="trend-chart__legend">
      <span v-for="item in series" :key="item.label">
        <i :style="{ backgroundColor: item.color }" />
        {{ item.label }}
        <strong>{{ formatter(lastValue(item)) }}</strong>
      </span>
    </div>
    <div v-if="hasData" class="trend-chart__canvas">
      <svg :viewBox="`0 0 ${width} ${height}`" role="img" aria-label="资源历史趋势">
        <line
          v-for="line in [0, 1, 2, 3]"
          :key="line"
          :x1="padding.left"
          :x2="width - padding.right"
          :y1="padding.top + line * ((height - padding.top - padding.bottom) / 3)"
          :y2="padding.top + line * ((height - padding.top - padding.bottom) / 3)"
          class="trend-chart__grid"
        />
        <path
          v-for="item in series"
          :key="item.label"
          :d="linePath(item.points)"
          :stroke="item.color"
          class="trend-chart__line"
        />
      </svg>
      <div class="trend-chart__axis">
        <span>{{ timeLabel(minimumTime) }}</span>
        <span>{{ timeLabel(maximumTime) }}</span>
      </div>
    </div>
    <div v-else class="trend-chart__empty">等待采样数据</div>
  </div>
</template>

<style scoped>
.trend-chart {
  min-width: 0;
}

.trend-chart__legend {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 18px;
  min-height: 28px;
  color: var(--text-muted);
  font-size: .8rem;
}

.trend-chart__legend span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.trend-chart__legend i {
  width: 8px;
  height: 8px;
  border-radius: 999px;
}

.trend-chart__legend strong {
  color: var(--text);
  font-size: .86rem;
}

.trend-chart__canvas {
  overflow: hidden;
  margin-top: 4px;
}

.trend-chart svg {
  display: block;
  width: 100%;
  height: 170px;
}

.trend-chart__grid {
  stroke: var(--border);
  stroke-width: 1;
  stroke-dasharray: 4 6;
}

.trend-chart__line {
  fill: none;
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}

.trend-chart__axis {
  display: flex;
  justify-content: space-between;
  color: var(--text-subtle);
  font-size: .72rem;
}

.trend-chart__empty {
  min-height: 190px;
  display: grid;
  place-items: center;
  color: var(--text-muted);
  font-size: .86rem;
}
</style>
