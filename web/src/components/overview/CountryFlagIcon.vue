<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps<{
  countryCode: string
  label: string
}>()

const flagModules = import.meta.glob('../../../node_modules/circle-flags/flags/*.svg', {
  import: 'default',
  query: '?url',
}) as Record<string, () => Promise<string>>

const flagLoaders = Object.fromEntries(
  Object.entries(flagModules).map(([path, loader]) => [
    path.match(/\/([^/]+)\.svg$/)?.[1] || '',
    loader,
  ]),
)

const source = ref('')
let loadSequence = 0

watch(
  () => props.countryCode.trim().toLowerCase(),
  async (code) => {
    const sequence = ++loadSequence
    source.value = ''
    const loader = flagLoaders[code]
    if (!loader) return
    try {
      const nextSource = await loader()
      if (sequence === loadSequence) source.value = nextSource
    } catch {
      // Unknown/missing flag assets render the caller's location fallback.
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  loadSequence += 1
})
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
