<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Activity, Clock, Cpu, Database, HardDrive, Network } from '@lucide/vue'
import { api } from '@/lib/api'
import { clampPercent, formatBytes, formatDuration, formatPercent, formatRate } from '@/lib/format'
import { useI18n } from '@/i18n'
import type { SystemOverview } from '@/types/api'

/**
 * Desktop monitor panel: a compact read-only readout pinned to the right side
 * of the desktop. Reuses the same overview endpoint as the Overview page so the
 * numbers always match the classic view. Refreshes on a bounded interval.
 */

const i18n = useI18n()
const overview = ref<SystemOverview>()
const loading = ref(true)
let refreshTimer: number | undefined
let controller: AbortController | undefined
let refreshActive = false

const cpuPercent = computed(() => overview.value?.cpu.percent)
const cpuCores = computed(() => overview.value?.cpu.cores)

const memoryUsed = computed(() => overview.value?.memory.value)
const memoryTotal = computed(() => overview.value?.memory.total)
const memoryPercent = computed(() => overview.value?.memory.percent)

const diskUsed = computed(() => overview.value?.disk.value)
const diskTotal = computed(() => overview.value?.disk.total)
const diskPercent = computed(() => overview.value?.disk.percent)

const load = computed(() => overview.value?.load)
const net = computed(() => overview.value?.network)
const uptime = computed(() => overview.value?.uptimeSeconds)

function cpuLabel(): string {
  const cores = cpuCores.value
  if (cores === undefined) return formatPercent(cpuPercent.value)
  return i18n.t('desktop.monitorCPUValue', {
    percent: formatPercent(cpuPercent.value),
    cores: String(cores),
  })
}

function memoryLabel(): string {
  if (memoryUsed.value === undefined || memoryTotal.value === undefined) return '—'
  return `${formatBytes(memoryUsed.value)} / ${formatBytes(memoryTotal.value)}`
}

function diskLabel(): string {
  if (diskUsed.value === undefined || diskTotal.value === undefined) return '—'
  return `${formatBytes(diskUsed.value)} / ${formatBytes(diskTotal.value)}`
}

async function refresh(silent = false): Promise<void> {
  if (refreshActive) return
  refreshActive = true
  if (!silent) loading.value = true
  controller?.abort()
  controller = new AbortController()
  try {
    overview.value = await api.overview.get(controller.signal)
  } catch {
    // Transient failures keep the last known values; the next tick retries.
  } finally {
    refreshActive = false
    loading.value = false
  }
}

onMounted(() => {
  void refresh()
  refreshTimer = window.setInterval(() => void refresh(true), 20_000)
})

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
  controller?.abort()
})
</script>

<template>
  <aside class="desktop-monitor" :aria-label="i18n.t('desktop.monitorLabel')">
    <header class="desktop-monitor__header">
      <Activity :size="15" aria-hidden="true" />
      <span>{{ i18n.t('desktop.monitorTitle') }}</span>
    </header>

    <div v-if="loading && !overview" class="desktop-monitor__loading">
      {{ i18n.t('desktop.entriesLoading') }}
    </div>

    <dl v-else class="desktop-monitor__list">
      <div class="desktop-monitor__row">
        <dt><Cpu :size="14" aria-hidden="true" /><span>{{ i18n.t('desktop.monitorCPU') }}</span></dt>
        <dd>{{ cpuLabel() }}</dd>
      </div>
      <div class="desktop-monitor__track" :aria-label="`${i18n.t('desktop.monitorCPU')} ${cpuLabel()}`">
        <span :style="{ width: `${clampPercent(cpuPercent)}%` }" />
      </div>

      <div class="desktop-monitor__row">
        <dt><Database :size="14" aria-hidden="true" /><span>{{ i18n.t('desktop.monitorMemory') }}</span></dt>
        <dd>{{ memoryLabel() }}</dd>
      </div>
      <div class="desktop-monitor__track" :aria-label="`${i18n.t('desktop.monitorMemory')} ${memoryLabel()}`">
        <span :style="{ width: `${clampPercent(memoryPercent)}%` }" />
      </div>

      <div class="desktop-monitor__row">
        <dt><HardDrive :size="14" aria-hidden="true" /><span>{{ i18n.t('desktop.monitorDisk') }}</span></dt>
        <dd>{{ diskLabel() }}</dd>
      </div>
      <div class="desktop-monitor__track" :aria-label="`${i18n.t('desktop.monitorDisk')} ${diskLabel()}`">
        <span :style="{ width: `${clampPercent(diskPercent)}%` }" />
      </div>

      <div class="desktop-monitor__row">
        <dt><Network :size="14" aria-hidden="true" /><span>{{ i18n.t('desktop.monitorNetwork') }}</span></dt>
        <dd class="desktop-monitor__network">
          <span>↓ {{ formatRate(net?.receiveBytesPerSecond) }}</span>
          <span>↑ {{ formatRate(net?.transmitBytesPerSecond) }}</span>
        </dd>
      </div>

      <div class="desktop-monitor__row">
        <dt><Activity :size="14" aria-hidden="true" /><span>{{ i18n.t('desktop.monitorLoad') }}</span></dt>
        <dd>
          {{ load?.one?.toFixed(2) ?? '—' }}
          {{ load?.five?.toFixed(2) ?? '' }}
          {{ load?.fifteen?.toFixed(2) ?? '' }}
        </dd>
      </div>

      <div class="desktop-monitor__row">
        <dt><Clock :size="14" aria-hidden="true" /><span>{{ i18n.t('desktop.monitorUptime') }}</span></dt>
        <dd>{{ formatDuration(uptime) }}</dd>
      </div>
    </dl>
  </aside>
</template>
