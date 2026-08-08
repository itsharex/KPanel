<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from '@/lib/api'

const props = defineProps<{
  siteId: string
  refreshKey: number
}>()

const name = ref('')
let controller: AbortController | undefined

async function load(): Promise<void> {
  controller?.abort()
  const request = new AbortController()
  controller = request
  name.value = ''
  try {
    const appearance = await api.sites.appearance(props.siteId, request.signal)
    if (controller === request && !request.signal.aborted) {
      name.value = appearance.name?.trim() || ''
    }
  } catch {
    // Website names are optional presentation metadata; keep the list usable
    // when a site is unavailable or does not expose a title.
  }
}

onMounted(() => void load())
watch(
  () => [props.siteId, props.refreshKey] as const,
  () => void load(),
)
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <small v-if="name" class="site-appearance-name" :title="name">{{ name }}</small>
</template>
