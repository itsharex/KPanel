<script setup lang="ts">
import { CircleAlert, RotateCw } from '@lucide/vue'

withDefaults(
  defineProps<{
    title?: string
    message: string
    retryLabel?: string
  }>(),
  {
    title: '暂时无法获取数据',
    retryLabel: '重新加载',
  },
)

defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="error-state" role="alert">
    <div class="error-state__icon" aria-hidden="true">
      <CircleAlert :size="24" :stroke-width="1.8" />
    </div>
    <div>
      <h3>{{ title }}</h3>
      <p>{{ message }}</p>
      <button class="button button--secondary button--small" type="button" @click="$emit('retry')">
        <RotateCw :size="15" />
        {{ retryLabel }}
      </button>
    </div>
  </div>
</template>
