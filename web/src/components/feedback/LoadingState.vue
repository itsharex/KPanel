<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@/i18n'

const props = withDefaults(
  defineProps<{
    rows?: number
    cards?: boolean
    label?: string
  }>(),
  {
    rows: 4,
    cards: false,
    label: undefined,
  },
)
const i18n = useI18n()
const resolvedLabel = computed(() => props.label || i18n.t('common.loading'))
</script>

<template>
  <div class="loading-state" :class="{ 'loading-state--cards': cards }" role="status" :aria-label="resolvedLabel">
    <div v-for="index in rows" :key="index" class="skeleton-block">
      <span class="skeleton-line skeleton-line--short" />
      <span class="skeleton-line" />
      <span class="skeleton-line skeleton-line--medium" />
    </div>
  </div>
</template>
