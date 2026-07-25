<script setup lang="ts">
import type { Component } from 'vue'
import { computed } from 'vue'
import { clampPercent } from '@/lib/format'

const props = withDefaults(
  defineProps<{
    label: string
    value: string
    detail?: string
    percent?: number
    icon: Component
    tone?: 'brand' | 'blue' | 'violet' | 'amber'
  }>(),
  {
    detail: '',
    percent: undefined,
    tone: 'brand',
  },
)

const normalizedPercent = computed(() => clampPercent(props.percent))
</script>

<template>
  <article class="metric-card">
    <div class="metric-card__top">
      <span class="metric-card__icon" :class="`metric-card__icon--${tone}`">
        <component :is="icon" :size="19" :stroke-width="1.9" aria-hidden="true" />
      </span>
      <span>{{ label }}</span>
    </div>
    <strong>{{ value }}</strong>
    <div v-if="percent !== undefined" class="progress-track" :aria-label="`${label} ${value}`">
      <span :style="{ width: `${normalizedPercent}%` }" />
    </div>
    <small>{{ detail || '　' }}</small>
  </article>
</template>
