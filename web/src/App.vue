<script setup lang="ts">
import { watchEffect } from 'vue'
import { RouterView } from 'vue-router'
import { useRoute } from 'vue-router'
import ToastHost from '@/components/feedback/ToastHost.vue'
import { useI18n } from '@/i18n'

const route = useRoute()
const i18n = useI18n()

watchEffect(() => {
  const title = route.meta.titleKey ? i18n.t(route.meta.titleKey) : ''
  document.title = title ? `${title} · KPanel` : 'KPanel'
})
</script>

<template>
  <RouterView />
  <ToastHost />
</template>
