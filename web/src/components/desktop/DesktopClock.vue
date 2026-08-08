<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Clock3, Globe2 } from '@lucide/vue'
import CountryFlagIcon from '@/components/overview/CountryFlagIcon.vue'
import { useI18n } from '@/i18n'
import type { PublicNetworkSummary } from '@/types/api'

/**
 * Desktop clock. Local time stays prominent; the compact server section uses
 * the host-configured timezone and independent public-IP geolocation.
 */

const props = defineProps<{
  network?: PublicNetworkSummary
  systemTimezone?: string
}>()

const i18n = useI18n()
const { locale } = i18n

const now = ref(Date.now())
let timer: number | undefined

function tick(): void {
  now.value = Date.now()
}

function stopTimer(): void {
  if (timer === undefined) return
  window.clearInterval(timer)
  timer = undefined
}

function startTimer(): void {
  stopTimer()
  tick()
  timer = window.setInterval(tick, 1000)
}

function onVisibilityChange(): void {
  if (document.hidden) stopTimer()
  else startTimer()
}

onMounted(() => {
  if (!document.hidden) startTimer()
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onBeforeUnmount(() => {
  stopTimer()
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

function createFormatter(options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat | undefined {
  try {
    return new Intl.DateTimeFormat(locale.value, options)
  } catch {
    return undefined
  }
}

const localTimeFormatter = computed(() => createFormatter({
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
}))

const localDateFormatter = computed(() => createFormatter({
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  weekday: 'long',
}))

const localTime = computed(() => localTimeFormatter.value?.format(now.value) ?? '--:--:--')
const localDate = computed(() => localDateFormatter.value?.format(now.value) ?? '')

const serverTimezone = computed(() => props.systemTimezone?.trim() || '')

const serverTimeFormatter = computed(() => {
  if (!serverTimezone.value) return undefined
  return createFormatter({
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
    timeZone: serverTimezone.value,
  })
})

const serverTime = computed(() => serverTimeFormatter.value?.format(now.value) ?? '')

const serverLabel = computed(() => {
  const tz = serverTimezone.value
  if (!tz) return ''
  return tz.replace(/_/g, ' ')
})

const countryCode = computed(() => {
  const code = props.network?.countryCode?.trim().toUpperCase() || ''
  return /^[A-Z]{2}$/.test(code) ? code : ''
})

const serverLocation = computed(() => {
  const parts = [props.network?.country, props.network?.city || props.network?.region]
    .filter((part): part is string => Boolean(part))
  return [...new Set(parts)].join(' · ')
})
</script>

<template>
  <section class="desktop-clock">
    <header class="desktop-clock__header">
      <span><Clock3 :size="15" aria-hidden="true" />{{ i18n.t('desktop.localTime') }}</span>
      <i aria-hidden="true" />
    </header>
    <time class="desktop-clock__local" :datetime="new Date(now).toISOString()">
      <span class="desktop-clock__time">{{ localTime }}</span>
      <span class="desktop-clock__date">{{ localDate }}</span>
    </time>
    <div v-if="serverLocation || serverLabel" class="desktop-clock__server" :title="serverLabel">
      <span class="desktop-clock__server-icon"><Globe2 :size="15" aria-hidden="true" /></span>
      <span class="desktop-clock__server-copy">
        <span class="desktop-clock__server-main">
          <span class="desktop-clock__server-label">{{ i18n.t('desktop.serverTime') }}</span>
          <span class="desktop-clock__server-time">{{ serverTime || '—' }}</span>
        </span>
        <span class="desktop-clock__server-location">
          <CountryFlagIcon
            v-if="countryCode"
            :country-code="countryCode"
            :label="network?.country || countryCode"
          />
          <span class="desktop-clock__server-location-name">{{ serverLocation || i18n.t('desktop.hostLocationUnknown') }}</span>
          <span v-if="serverLabel" class="desktop-clock__server-timezone">{{ serverLabel }}</span>
        </span>
      </span>
    </div>
  </section>
</template>
