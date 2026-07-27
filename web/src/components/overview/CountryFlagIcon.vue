<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  countryCode: string
  label: string
}>()

const flagModules = import.meta.glob('../../../node_modules/circle-flags/flags/*.svg', {
  eager: true,
  import: 'default',
  query: '?url',
}) as Record<string, string>

const flagSources = Object.fromEntries(
  Object.entries(flagModules).map(([path, source]) => [
    path.match(/\/([^/]+)\.svg$/)?.[1] || '',
    source,
  ]),
)

const source = computed(() => flagSources[props.countryCode.trim().toLowerCase()] || '')
</script>

<template>
  <img
    v-if="source"
    class="country-flag"
    :src="source"
    :alt="`${props.label}国旗`"
    width="22"
    height="22"
  />
</template>
