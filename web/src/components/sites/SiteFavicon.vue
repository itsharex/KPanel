<script setup lang="ts">
import { Globe2 } from '@lucide/vue'
import { api } from '@/lib/api'
import { createSiteFaviconFailureState } from '@/lib/siteFavicon'

const props = defineProps<{
  siteId: string
  domain: string
  refreshKey: number
}>()

const failed = createSiteFaviconFailureState(
  () => props.siteId,
  () => props.refreshKey,
)
</script>

<template>
  <span class="resource-name__icon">
    <img
      v-if="!failed"
      class="site-favicon"
      :src="api.sites.iconURL(props.siteId)"
      :title="`${props.domain} 网站图标`"
      alt=""
      width="22"
      height="22"
      loading="lazy"
      decoding="async"
      fetchpriority="low"
      @error="failed = true"
    />
    <Globe2 v-else :size="18" aria-hidden="true" />
  </span>
</template>
