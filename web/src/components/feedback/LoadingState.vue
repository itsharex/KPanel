<script setup lang="ts">
withDefaults(
  defineProps<{
    rows?: number
    cards?: boolean
    label?: string
  }>(),
  {
    rows: 4,
    cards: false,
    label: '正在加载',
  },
)
</script>

<template>
  <div class="loading-state" :class="{ 'loading-state--cards': cards }" role="status" :aria-label="label">
    <div v-for="index in rows" :key="index" class="skeleton-block">
      <span class="skeleton-line skeleton-line--short" />
      <span class="skeleton-line" />
      <span class="skeleton-line skeleton-line--medium" />
    </div>
  </div>
</template>
